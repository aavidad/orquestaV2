package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryLocalStateIdentityIsStableAndBackupCopyDiffers(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "source", "state.sqlite")
	source := openLocalIdentityTestRepository(t, sourcePath)
	first, supported, err := source.LocalStateIdentity()
	if err != nil {
		t.Fatalf("source identity: %v", err)
	}
	second, secondSupported, err := source.LocalStateIdentity()
	if err != nil || secondSupported != supported || second != first {
		t.Fatalf("identity changed in one opening: first=%q second=%q supported=%v/%v err=%v",
			first, second, supported, secondSupported, err)
	}
	if err := source.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}
	if _, _, err := source.LocalStateIdentity(); err == nil {
		t.Fatal("closed repository retained a readable local identity handle")
	}
	reopened := openLocalIdentityTestRepository(t, sourcePath)
	reopenedIdentity, reopenedSupported, err := reopened.LocalStateIdentity()
	if err != nil || reopenedSupported != supported || reopenedIdentity != first {
		t.Fatalf("identity changed after repository restart: first=%q reopened=%q supported=%v/%v err=%v",
			first, reopenedIdentity, supported, reopenedSupported, err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatalf("close reopened source: %v", err)
	}
	if !supported {
		t.Skip("host does not expose reliable local file identity")
	}

	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source database: %v", err)
	}
	restoredDirectory := filepath.Join(root, "restored")
	if err := os.Mkdir(restoredDirectory, 0o700); err != nil {
		t.Fatalf("create restored directory: %v", err)
	}
	restoredPath := filepath.Join(restoredDirectory, "state.sqlite")
	if err := os.WriteFile(restoredPath, payload, 0o600); err != nil {
		t.Fatalf("copy database: %v", err)
	}
	restored := openLocalIdentityTestRepository(t, restoredPath)
	t.Cleanup(func() { _ = restored.Close() })
	restoredIdentity, restoredSupported, err := restored.LocalStateIdentity()
	if err != nil {
		t.Fatalf("restored identity: %v", err)
	}
	if !restoredSupported {
		t.Skip("restored filesystem does not expose reliable local file identity")
	}
	if restoredIdentity == "" || restoredIdentity == first {
		t.Fatalf("backup copy inherited source identity: source=%q restored=%q", first, restoredIdentity)
	}
}

func TestRepositoryLocalStateIdentityDetectsPathReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "active", "state.sqlite")
	repository := openLocalIdentityTestRepository(t, path)
	t.Cleanup(func() { _ = repository.Close() })
	identity, supported, err := repository.LocalStateIdentity()
	if err != nil {
		t.Fatalf("initial identity: %v", err)
	}

	activePath := path + ".active"
	if err := os.Rename(path, activePath); err != nil {
		t.Fatalf("move active database: %v", err)
	}
	replacementActive := true
	t.Cleanup(func() {
		if replacementActive {
			_ = os.Remove(path)
			_ = os.Rename(activePath, path)
		}
	})
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatalf("write replacement: %v", err)
	}
	if _, _, err := repository.LocalStateIdentity(); err == nil {
		t.Fatal("path replacement was accepted for retained repository handle")
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove replacement: %v", err)
	}
	if err := os.Rename(activePath, path); err != nil {
		t.Fatalf("restore active database: %v", err)
	}
	replacementActive = false
	revalidated, revalidatedSupported, err := repository.LocalStateIdentity()
	if err != nil || revalidated != identity || revalidatedSupported != supported {
		t.Fatalf("restored active handle identity=%q supported=%v err=%v", revalidated, revalidatedSupported, err)
	}
}

func openLocalIdentityTestRepository(t *testing.T, path string) *Repository {
	t.Helper()
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: time.Second, MaxOpenConnections: 2,
	})
	if err != nil {
		t.Fatalf("open repository: %v: %v", err, errors.Unwrap(err))
	}
	return repository
}
