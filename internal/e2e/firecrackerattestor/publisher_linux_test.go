//go:build linux

package firecrackerattestor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

const publisherTestDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestPublisherStagesBothFilesAndPublishesReceiptLast(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "evidence.json")
	receiptPath := filepath.Join(root, "receipt.txt")
	operations := testPublisherOperations()
	var events []string
	syncFile := operations.syncFile
	operations.syncFile = func(file *os.File) error {
		events = append(events, "file-sync:"+filepath.Base(strings.TrimSuffix(
			file.Name(), " (staged writer)",
		)))
		return syncFile(file)
	}
	closeWriter := operations.closeStagedWriter
	operations.closeStagedWriter = func(file *os.File) error {
		events = append(events, "file-close:"+filepath.Base(strings.TrimSuffix(
			file.Name(), " (staged writer)",
		)))
		return closeWriter(file)
	}
	renameNoReplace := operations.renameNoReplace
	operations.renameNoReplace = func(
		oldDirectory int,
		oldName string,
		newDirectory int,
		newName string,
	) error {
		if newName == filepath.Base(receiptPath) {
			if _, err := os.Stat(evidencePath); err != nil {
				t.Fatalf("receipt attempted before evidence publication: %v", err)
			}
			if _, err := os.Stat(receiptPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("receipt existed before commit rename: %v", err)
			}
		}
		events = append(events, "rename:"+newName)
		return renameNoReplace(oldDirectory, oldName, newDirectory, newName)
	}
	syncDirectory := operations.syncDirectory
	operations.syncDirectory = func(directory *os.File) error {
		events = append(events, "directory-sync")
		return syncDirectory(directory)
	}

	publication, err := (LinuxPublisher{
		EvidencePath: evidencePath,
		ReceiptPath:  receiptPath,
	}).publish(validPublisherEvidence(), validPublisherReceipt(), operations)
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{
		"file-sync:evidence.json",
		"file-close:evidence.json",
		"file-sync:receipt.txt",
		"file-close:receipt.txt",
		"rename:evidence.json",
		"directory-sync",
		"rename:receipt.txt",
		"directory-sync",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("publication order mismatch:\n got: %v\nwant: %v", events, wantEvents)
	}
	if publication.EvidencePath != evidencePath ||
		publication.ReceiptPath != receiptPath ||
		!validDigest(publication.EvidenceSHA256) {
		t.Fatalf("invalid publication: %+v", publication)
	}
	assertPublishedMetadata(t, evidencePath, operations.outputUID, operations.outputGID)
	assertPublishedMetadata(t, receiptPath, operations.outputUID, operations.outputGID)
	receiptContent, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(
		receiptContent,
		[]byte("evidence_sha256="+publication.EvidenceSHA256+"\n"),
	) {
		t.Fatal("receipt does not bind published evidence")
	}
	assertNoStagingFiles(t, root)
}

func TestPublisherExistingTargetNeverClobbered(t *testing.T) {
	tests := []struct {
		name         string
		existingName string
		absentName   string
	}{
		{name: "evidence", existingName: "evidence.json", absentName: "receipt.txt"},
		{name: "receipt", existingName: "receipt.txt", absentName: "evidence.json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			existingPath := filepath.Join(root, test.existingName)
			original := []byte("foreign-existing-content")
			if err := os.WriteFile(existingPath, original, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := (LinuxPublisher{
				EvidencePath: filepath.Join(root, "evidence.json"),
				ReceiptPath:  filepath.Join(root, "receipt.txt"),
			}).publish(
				validPublisherEvidence(),
				validPublisherReceipt(),
				testPublisherOperations(),
			)
			if err == nil || !strings.Contains(err.Error(), "output_exists") {
				t.Fatalf("existing target accepted: %v", err)
			}
			got, readErr := os.ReadFile(existingPath)
			if readErr != nil || !bytes.Equal(got, original) {
				t.Fatalf("existing target changed: content=%q err=%v", got, readErr)
			}
			if _, statErr := os.Stat(filepath.Join(root, test.absentName)); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("counterpart unexpectedly published: %v", statErr)
			}
			assertNoStagingFiles(t, root)
		})
	}
}

func TestPublisherRenameRaceNeverClobbersLateTarget(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "evidence.json")
	receiptPath := filepath.Join(root, "receipt.txt")
	foreign := []byte("late foreign target")
	operations := testPublisherOperations()
	renameNoReplace := operations.renameNoReplace
	operations.renameNoReplace = func(
		oldDirectory int,
		oldName string,
		newDirectory int,
		newName string,
	) error {
		if newName == filepath.Base(evidencePath) {
			if err := os.WriteFile(evidencePath, foreign, 0o400); err != nil {
				t.Fatal(err)
			}
		}
		return renameNoReplace(oldDirectory, oldName, newDirectory, newName)
	}

	_, err := (LinuxPublisher{
		EvidencePath: evidencePath,
		ReceiptPath:  receiptPath,
	}).publish(
		validPublisherEvidence(),
		validPublisherReceipt(),
		operations,
	)
	if err == nil || !errors.Is(err, unix.EEXIST) {
		t.Fatalf("late target did not win no-clobber race: %v", err)
	}
	got, readErr := os.ReadFile(evidencePath)
	if readErr != nil || !bytes.Equal(got, foreign) {
		t.Fatalf("late target changed: content=%q err=%v", got, readErr)
	}
	if _, statErr := os.Stat(receiptPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("receipt unexpectedly committed: %v", statErr)
	}
	assertNoStagingFiles(t, root)
}

func TestPublisherSecondRenameFailureRemovesOnlyOwnedEvidence(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "evidence.json")
	receiptPath := filepath.Join(root, "receipt.txt")
	operations := testPublisherOperations()
	renameNoReplace := operations.renameNoReplace
	renameCount := 0
	operations.renameNoReplace = func(
		oldDirectory int,
		oldName string,
		newDirectory int,
		newName string,
	) error {
		renameCount++
		if renameCount == 2 {
			return unix.EIO
		}
		return renameNoReplace(oldDirectory, oldName, newDirectory, newName)
	}

	if _, err := (LinuxPublisher{
		EvidencePath: evidencePath,
		ReceiptPath:  receiptPath,
	}).publish(
		validPublisherEvidence(),
		validPublisherReceipt(),
		operations,
	); err == nil || !errors.Is(err, unix.EIO) {
		t.Fatalf("second rename failure lost: %v", err)
	}
	if renameCount != 2 {
		t.Fatalf("unexpected rename count: %d", renameCount)
	}
	for _, path := range []string{evidencePath, receiptPath} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("partial publication remains at %s: %v", path, err)
		}
	}
	assertNoStagingFiles(t, root)
}

func TestPublisherCleanupRefusesForeignReplacement(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "evidence.json")
	movedEvidencePath := filepath.Join(root, "owned-evidence-moved")
	receiptPath := filepath.Join(root, "receipt.txt")
	foreign := []byte("foreign replacement")
	operations := testPublisherOperations()
	renameNoReplace := operations.renameNoReplace
	renameCount := 0
	operations.renameNoReplace = func(
		oldDirectory int,
		oldName string,
		newDirectory int,
		newName string,
	) error {
		renameCount++
		if renameCount == 1 {
			return renameNoReplace(oldDirectory, oldName, newDirectory, newName)
		}
		if err := os.Rename(evidencePath, movedEvidencePath); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(evidencePath, foreign, 0o400); err != nil {
			t.Fatal(err)
		}
		return unix.EIO
	}

	_, err := (LinuxPublisher{
		EvidencePath: evidencePath,
		ReceiptPath:  receiptPath,
	}).publish(
		validPublisherEvidence(),
		validPublisherReceipt(),
		operations,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "output_cleanup_identity_mismatch") {
		t.Fatalf("foreign replacement was not detected: %v", err)
	}
	got, readErr := os.ReadFile(evidencePath)
	if readErr != nil || !bytes.Equal(got, foreign) {
		t.Fatalf("foreign replacement removed: content=%q err=%v", got, readErr)
	}
	if _, statErr := os.Stat(movedEvidencePath); statErr != nil {
		t.Fatalf("owned inode unexpectedly removed through foreign name: %v", statErr)
	}
	if _, statErr := os.Stat(receiptPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("receipt unexpectedly committed: %v", statErr)
	}
}

func TestPublisherRejectsSamePathBeforeStaging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "publication")
	_, err := (LinuxPublisher{
		EvidencePath: path,
		ReceiptPath:  path,
	}).publish(
		validPublisherEvidence(),
		validPublisherReceipt(),
		testPublisherOperations(),
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("same path accepted: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("same-path publication created output: %v", statErr)
	}
}

func TestPublisherStageSyncAndCloseFailuresPublishNothing(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*publisherOperations)
	}{
		{
			name: "receipt_sync",
			mutate: func(operations *publisherOperations) {
				syncFile := operations.syncFile
				calls := 0
				operations.syncFile = func(file *os.File) error {
					calls++
					if err := syncFile(file); err != nil {
						return err
					}
					if calls == 2 {
						return unix.EIO
					}
					return nil
				}
			},
		},
		{
			name: "receipt_close",
			mutate: func(operations *publisherOperations) {
				closeWriter := operations.closeStagedWriter
				calls := 0
				operations.closeStagedWriter = func(file *os.File) error {
					calls++
					if err := closeWriter(file); err != nil {
						return err
					}
					if calls == 2 {
						return unix.EIO
					}
					return nil
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			operations := testPublisherOperations()
			test.mutate(&operations)
			evidencePath := filepath.Join(root, "evidence.json")
			receiptPath := filepath.Join(root, "receipt.txt")
			if _, err := (LinuxPublisher{
				EvidencePath: evidencePath,
				ReceiptPath:  receiptPath,
			}).publish(
				validPublisherEvidence(),
				validPublisherReceipt(),
				operations,
			); err == nil {
				t.Fatal("stage failure accepted")
			}
			for _, path := range []string{evidencePath, receiptPath} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("stage failure published %s: %v", path, err)
				}
			}
			assertNoStagingFiles(t, root)
		})
	}
}

func TestDefaultPublisherRequiresRootMetadataAndTrustedAncestors(t *testing.T) {
	operations := defaultPublisherOperations()
	if operations.outputUID != 0 || operations.outputGID != 0 {
		t.Fatalf(
			"default output owner is not root:root: %d:%d",
			operations.outputUID,
			operations.outputGID,
		)
	}
	unsafeRoot := t.TempDir()
	if err := os.Chmod(unsafeRoot, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := operations.verifyTrustedAncestors(
		filepath.Join(unsafeRoot, "publication"),
	); err == nil {
		t.Fatal("unsafe writable ancestry accepted")
	}
	rootStat := unix.Stat_t{
		Mode:  unix.S_IFREG | 0o400,
		Uid:   0,
		Gid:   0,
		Nlink: 1,
	}
	if !validPublicationStat(&rootStat, 0, 0) {
		t.Fatal("root:root 0400 regular publication rejected")
	}
	rootStat.Mode = unix.S_IFREG | 0o600
	if validPublicationStat(&rootStat, 0, 0) {
		t.Fatal("writable publication metadata accepted")
	}
}

func testPublisherOperations() publisherOperations {
	operations := defaultPublisherOperations()
	operations.effectiveUID = func() int { return 0 }
	operations.outputUID = uint32(os.Geteuid())
	operations.outputGID = uint32(os.Getegid())
	operations.verifyTrustedAncestors = func(string) error { return nil }
	operations.random = bytes.NewReader(make([]byte, 32))
	return operations
}

func validPublisherEvidence() Evidence {
	return Evidence{
		Schema: EvidenceSchema,
		Suite:  Suite,
		Status: "passed",
	}
}

func validPublisherReceipt() Receipt {
	return Receipt{
		Schema:               ReceiptSchema,
		Status:               "passed",
		ConfigSHA256:         publisherTestDigest,
		UnitSHA256:           publisherTestDigest,
		PrimitivesUnitSHA256: publisherTestDigest,
		LauncherSHA256:       publisherTestDigest,
		AssetDigest:          publisherTestDigest,
		PolicyDigest:         publisherTestDigest,
		E2ESuite:             Suite,
		MaxConcurrentRuns:    ExpectedConcurrentRuns,
		PhysicalMicroVMCount: ExpectedConcurrentRuns,
		ConcurrentHighWater:  ExpectedConcurrentRuns,
		AllAttestationsValid: true,
		ZeroResidualRuns:     true,
		NetworkAbsent:        true,
		APIAbsent:            true,
		VsockAbsent:          true,
		SerialAbsent:         true,
		MemorySwapMaxZero:    true,
	}
}

func assertPublishedMetadata(
	t *testing.T,
	path string,
	uid uint32,
	gid uint32,
) {
	t.Helper()
	var stat unix.Stat_t
	if err := unix.Lstat(path, &stat); err != nil {
		t.Fatal(err)
	}
	if !validPublicationStat(&stat, uid, gid) {
		t.Fatalf(
			"unsafe publication metadata: mode=%#o uid=%d gid=%d nlink=%d",
			stat.Mode,
			stat.Uid,
			stat.Gid,
			stat.Nlink,
		)
	}
}

func assertNoStagingFiles(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".new-") {
			t.Fatalf("staging file leaked: %s", entry.Name())
		}
	}
}
