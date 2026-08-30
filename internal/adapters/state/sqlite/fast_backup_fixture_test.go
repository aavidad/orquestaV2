package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
)

// cachedV09RestoreBackup holds only an immutable, already-published canonical
// backup used as input by restore-only tests. Tests about CreateBackup itself
// must not use this fixture.
type cachedV09RestoreBackup struct {
	sync.Once
	receipt  application.BackupReceipt
	payload  []byte
	manifest []byte
	err      error
}

func seedCachedV09RestoreBackup(
	t *testing.T,
	destinationRoot string,
	state *cachedV09RestoreBackup,
	build func(*testing.T) (application.BackupReceipt, string),
) application.BackupReceipt {
	t.Helper()
	state.Do(func() {
		receipt, sourceRoot := build(t)
		name := strings.TrimPrefix(receipt.Ref.String(), backupRefPrefix)
		if name == receipt.Ref.String() || name == "" || filepath.Base(name) != name {
			state.err = errors.New("invalid cached V09 backup ref")
			return
		}
		state.receipt = receipt
		state.payload, state.err = readPrivateV09FixtureFile(filepath.Join(sourceRoot, name, recoveryPayloadName))
		if state.err != nil {
			return
		}
		state.manifest, state.err = readPrivateV09FixtureFile(filepath.Join(sourceRoot, name, recoveryManifestName))
	})
	if state.err != nil || len(state.payload) == 0 || len(state.manifest) == 0 {
		t.Fatalf("build cached V09 restore backup: %v", state.err)
	}

	name := strings.TrimPrefix(state.receipt.Ref.String(), backupRefPrefix)
	directory := filepath.Join(destinationRoot, name)
	if err := createPrivateSQLiteDirectoryChain(directory); err != nil {
		t.Fatalf("create cached V09 backup directory: %v", err)
	}
	for fileName, content := range map[string][]byte{
		recoveryPayloadName:  state.payload,
		recoveryManifestName: state.manifest,
	} {
		if err := writePrivateTestDatabaseSeed(filepath.Join(directory, fileName), content); err != nil {
			t.Fatalf("copy cached V09 backup %s: %v", fileName, err)
		}
	}
	return state.receipt
}

func readPrivateV09FixtureFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("cached V09 backup file is not private regular data")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, errors.New("cached V09 backup file is empty")
	}
	return content, nil
}

func TestCachedV09RestoreBackupCreatesIndependentPublishedCopies(t *testing.T) {
	var state cachedV09RestoreBackup
	builds := 0
	build := func(t *testing.T) (application.BackupReceipt, string) {
		builds++
		repository, _ := openTestRepository(t)
		recovery, backupRoot, _ := newV09TestRecovery(t, repository, time.Unix(1_720_000_000, 0).UTC(), nil)
		receipt, err := recovery.CreateBackup(context.Background())
		if err != nil {
			t.Fatalf("create canonical cached backup: %v", err)
		}
		if err := recovery.Close(); err != nil {
			t.Fatalf("close canonical cached backup recovery: %v", err)
		}
		return receipt, backupRoot
	}
	firstRoot := filepath.Join(t.TempDir(), "backups")
	first := seedCachedV09RestoreBackup(t, firstRoot, &state, build)
	secondRoot := filepath.Join(t.TempDir(), "backups")
	second := seedCachedV09RestoreBackup(t, secondRoot, &state, build)
	if builds != 1 || first != second {
		t.Fatalf("cached backup builds=%d first=%+v second=%+v", builds, first, second)
	}
	name := strings.TrimPrefix(first.Ref.String(), backupRefPrefix)
	firstInfo, err := os.Stat(filepath.Join(firstRoot, name, recoveryPayloadName))
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(filepath.Join(secondRoot, name, recoveryPayloadName))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(firstInfo, secondInfo) || firstInfo.Mode().Perm() != 0o600 || secondInfo.Mode().Perm() != 0o600 {
		t.Fatalf("cached backup copies are not independent private files: first=%v second=%v", firstInfo, secondInfo)
	}
}
