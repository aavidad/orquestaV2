//go:build linux

package firecrackerattestor

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type LinuxPublisher struct {
	EvidencePath string
	ReceiptPath  string
}

func (publisher LinuxPublisher) Publish(
	evidence Evidence,
	receipt Receipt,
) (Publication, error) {
	return publisher.publish(evidence, receipt, defaultPublisherOperations())
}

type publisherOperations struct {
	effectiveUID           func() int
	outputUID              uint32
	outputGID              uint32
	verifyTrustedAncestors func(string) error
	random                 io.Reader
	syncFile               func(*os.File) error
	closeStagedWriter      func(*os.File) error
	renameNoReplace        func(int, string, int, string) error
	syncDirectory          func(*os.File) error
}

func defaultPublisherOperations() publisherOperations {
	return publisherOperations{
		effectiveUID:           os.Geteuid,
		outputUID:              0,
		outputGID:              0,
		verifyTrustedAncestors: VerifyTrustedAncestors,
		random:                 rand.Reader,
		syncFile:               (*os.File).Sync,
		closeStagedWriter:      (*os.File).Close,
		renameNoReplace: func(oldDirectory int, oldName string, newDirectory int, newName string) error {
			return unix.Renameat2(
				oldDirectory, oldName, newDirectory, newName, unix.RENAME_NOREPLACE,
			)
		},
		syncDirectory: func(directory *os.File) error {
			return unix.Fsync(int(directory.Fd()))
		},
	}
}

func (publisher LinuxPublisher) publish(
	evidence Evidence,
	receipt Receipt,
	operations publisherOperations,
) (Publication, error) {
	if !operations.valid() || operations.effectiveUID() != 0 ||
		publisher.EvidencePath == publisher.ReceiptPath ||
		!canonicalOutputPath(publisher.EvidencePath) ||
		!canonicalOutputPath(publisher.ReceiptPath) ||
		evidence.Schema != EvidenceSchema || evidence.Suite != Suite ||
		evidence.Status != "passed" {
		return Publication{}, ErrInvalid
	}
	content, err := json.Marshal(evidence)
	if err != nil {
		return Publication{}, errors.New("firecracker_attestor_e2e.evidence_invalid")
	}
	content = append(content, '\n')
	sum := sha256.Sum256(content)
	evidenceDigest := hex.EncodeToString(sum[:])
	receipt.EvidenceSHA256 = evidenceDigest
	receiptContent, err := marshalReceipt(receipt)
	if err != nil {
		return Publication{}, err
	}
	if err := publishPairNoClobber(
		publisher.EvidencePath, content,
		publisher.ReceiptPath, receiptContent,
		operations,
	); err != nil {
		return Publication{}, err
	}
	return Publication{
		EvidenceSHA256: evidenceDigest,
		EvidencePath:   publisher.EvidencePath,
		ReceiptPath:    publisher.ReceiptPath,
	}, nil
}

func (operations publisherOperations) valid() bool {
	return operations.effectiveUID != nil &&
		operations.verifyTrustedAncestors != nil &&
		operations.random != nil &&
		operations.syncFile != nil &&
		operations.closeStagedWriter != nil &&
		operations.renameNoReplace != nil &&
		operations.syncDirectory != nil
}

func publishPairNoClobber(
	evidencePath string,
	evidenceContent []byte,
	receiptPath string,
	receiptContent []byte,
	operations publisherOperations,
) (resultErr error) {
	evidence, err := stagePublicationFile(
		evidencePath, evidenceContent, operations,
	)
	if err != nil {
		return err
	}
	defer func() {
		resultErr = errors.Join(resultErr, evidence.release())
	}()
	receipt, err := stagePublicationFile(
		receiptPath, receiptContent, operations,
	)
	if err != nil {
		return err
	}
	defer func() {
		resultErr = errors.Join(resultErr, receipt.release())
	}()

	if err := evidence.publish(); err != nil {
		return errors.Join(err, evidence.rollback())
	}
	if err := operations.syncDirectory(evidence.parent); err != nil {
		return errors.Join(
			errors.New("firecracker_attestor_e2e.output_parent_sync_failed"),
			evidence.rollback(),
		)
	}
	if err := receipt.publish(); err != nil {
		return errors.Join(err, receipt.rollback(), evidence.rollback())
	}
	if err := operations.syncDirectory(receipt.parent); err != nil {
		return errors.Join(
			errors.New("firecracker_attestor_e2e.output_parent_sync_failed"),
			receipt.rollback(),
			evidence.rollback(),
		)
	}
	return nil
}

type stagedPublicationFile struct {
	operations publisherOperations
	parent     *os.File
	targetName string
	stagedName string
	descriptor *os.File
	identity   unix.Stat_t
	published  bool
	removed    bool
}

func stagePublicationFile(
	path string,
	content []byte,
	operations publisherOperations,
) (stagedResult *stagedPublicationFile, resultErr error) {
	if err := operations.verifyTrustedAncestors(path); err != nil {
		return nil, err
	}
	parentPath, name := filepath.Dir(path), filepath.Base(path)
	parent, err := openDirectoryNoLinks("/", strings.TrimPrefix(parentPath, "/"))
	if err != nil {
		return nil, errors.New("firecracker_attestor_e2e.output_parent_unsafe")
	}
	var target unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &target, unix.AT_SYMLINK_NOFOLLOW); !errors.Is(err, unix.ENOENT) {
		parent.Close()
		return nil, errors.New("firecracker_attestor_e2e.output_exists")
	}
	suffix := make([]byte, 16)
	if _, err := io.ReadFull(operations.random, suffix); err != nil {
		parent.Close()
		return nil, errors.New("firecracker_attestor_e2e.random_failed")
	}
	temporary := "." + name + ".new-" + hex.EncodeToString(suffix)
	fd, err := unix.Openat(
		int(parent.Fd()), temporary,
		unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0o400,
	)
	if err != nil {
		parent.Close()
		return nil, errors.New("firecracker_attestor_e2e.output_create_failed")
	}
	file := os.NewFile(uintptr(fd), path+" (staged writer)")
	staged := &stagedPublicationFile{
		operations: operations,
		parent:     parent,
		targetName: name,
		stagedName: temporary,
		descriptor: file,
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, staged.release())
		}
	}()
	if unix.Fstat(fd, &staged.identity) != nil {
		return nil, errors.New("firecracker_attestor_e2e.output_metadata_failed")
	}
	if unix.Fchown(fd, int(operations.outputUID), int(operations.outputGID)) != nil ||
		unix.Fchmod(fd, 0o400) != nil {
		return nil, errors.New("firecracker_attestor_e2e.output_metadata_failed")
	}
	for len(content) > 0 {
		written, err := file.Write(content)
		if err != nil || written <= 0 {
			return nil, errors.New("firecracker_attestor_e2e.output_write_failed")
		}
		content = content[written:]
	}
	if operations.syncFile(file) != nil {
		return nil, errors.New("firecracker_attestor_e2e.output_sync_failed")
	}
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		!validPublicationStat(&stat, operations.outputUID, operations.outputGID) {
		return nil, errors.New("firecracker_attestor_e2e.output_metadata_failed")
	}
	descriptorFD, err := unix.Openat(
		int(parent.Fd()), temporary,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return nil, errors.New("firecracker_attestor_e2e.output_metadata_failed")
	}
	descriptor := os.NewFile(uintptr(descriptorFD), path+" (staged identity)")
	var descriptorStat unix.Stat_t
	if unix.Fstat(descriptorFD, &descriptorStat) != nil ||
		!sameFileIdentity(&stat, &descriptorStat) ||
		!validPublicationStat(
			&descriptorStat, operations.outputUID, operations.outputGID,
		) {
		descriptor.Close()
		return nil, errors.New("firecracker_attestor_e2e.output_metadata_failed")
	}
	staged.descriptor = descriptor
	if err := operations.closeStagedWriter(file); err != nil {
		return nil, errors.New("firecracker_attestor_e2e.output_close_failed")
	}
	staged.identity = descriptorStat
	return staged, nil
}

func (staged *stagedPublicationFile) publish() error {
	if staged == nil || staged.parent == nil || staged.descriptor == nil ||
		staged.published || staged.removed {
		return ErrInvalid
	}
	if err := staged.operations.renameNoReplace(
		int(staged.parent.Fd()), staged.stagedName,
		int(staged.parent.Fd()), staged.targetName,
	); err != nil {
		return fmt.Errorf("firecracker_attestor_e2e.output_publish_failed: %w", err)
	}
	staged.published = true
	if !staged.nameStillOwned(staged.targetName, true) {
		return errors.New("firecracker_attestor_e2e.output_identity_changed")
	}
	return nil
}

func (staged *stagedPublicationFile) rollback() error {
	if staged == nil || !staged.published || staged.removed {
		return nil
	}
	if !staged.nameStillOwned(staged.targetName, true) {
		return errors.New("firecracker_attestor_e2e.output_cleanup_identity_mismatch")
	}
	if err := unix.Unlinkat(int(staged.parent.Fd()), staged.targetName, 0); err != nil {
		return errors.New("firecracker_attestor_e2e.output_cleanup_failed")
	}
	var descriptorStat unix.Stat_t
	if unix.Fstat(int(staged.descriptor.Fd()), &descriptorStat) != nil ||
		!sameFileIdentity(&staged.identity, &descriptorStat) ||
		descriptorStat.Nlink != 0 {
		return errors.New("firecracker_attestor_e2e.output_cleanup_identity_mismatch")
	}
	staged.removed = true
	if err := staged.operations.syncDirectory(staged.parent); err != nil {
		return errors.New("firecracker_attestor_e2e.output_parent_sync_failed")
	}
	return nil
}

func (staged *stagedPublicationFile) release() error {
	if staged == nil {
		return nil
	}
	var cleanupErr error
	if !staged.published && !staged.removed &&
		staged.parent != nil && staged.descriptor != nil {
		if !staged.nameStillOwned(staged.stagedName, false) {
			cleanupErr = errors.New(
				"firecracker_attestor_e2e.output_cleanup_identity_mismatch",
			)
		} else if err := unix.Unlinkat(
			int(staged.parent.Fd()), staged.stagedName, 0,
		); err != nil && !errors.Is(err, unix.ENOENT) {
			cleanupErr = errors.New(
				"firecracker_attestor_e2e.output_cleanup_failed",
			)
		} else {
			staged.removed = true
		}
	}
	if staged.descriptor != nil {
		_ = staged.descriptor.Close()
	}
	if staged.parent != nil {
		_ = staged.parent.Close()
	}
	return cleanupErr
}

func (staged *stagedPublicationFile) nameStillOwned(
	name string,
	requireMetadata bool,
) bool {
	if staged == nil || staged.parent == nil || staged.descriptor == nil {
		return false
	}
	var descriptorStat, namedStat unix.Stat_t
	owned := unix.Fstat(int(staged.descriptor.Fd()), &descriptorStat) == nil &&
		unix.Fstatat(
			int(staged.parent.Fd()), name, &namedStat, unix.AT_SYMLINK_NOFOLLOW,
		) == nil &&
		sameFileIdentity(&staged.identity, &descriptorStat) &&
		sameFileIdentity(&descriptorStat, &namedStat) &&
		descriptorStat.Mode&unix.S_IFMT == unix.S_IFREG &&
		namedStat.Mode&unix.S_IFMT == unix.S_IFREG
	if !owned || !requireMetadata {
		return owned
	}
	return validPublicationStat(
		&descriptorStat,
		staged.operations.outputUID,
		staged.operations.outputGID,
	) && validPublicationStat(
		&namedStat,
		staged.operations.outputUID,
		staged.operations.outputGID,
	)
}

func sameFileIdentity(left, right *unix.Stat_t) bool {
	return left != nil && right != nil &&
		left.Dev == right.Dev && left.Ino == right.Ino
}

func validPublicationStat(stat *unix.Stat_t, uid, gid uint32) bool {
	return stat != nil &&
		stat.Mode&unix.S_IFMT == unix.S_IFREG &&
		stat.Mode&0o777 == 0o400 &&
		stat.Uid == uid && stat.Gid == gid &&
		stat.Nlink == 1
}

func canonicalOutputPath(path string) bool {
	return path != "" && filepath.IsAbs(path) && filepath.Clean(path) == path &&
		filepath.Base(path) != "." && filepath.Base(path) != "/" &&
		!strings.ContainsRune(path, 0)
}
