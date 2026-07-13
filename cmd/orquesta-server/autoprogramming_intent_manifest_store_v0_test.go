package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestAutoprogrammingIntentManifestStoreV0CreateIfAbsentRaceConflictAndSymlink(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatalf("manifest issues=%+v", issues)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(store.RootDir, request.RequestRef+".json"))
	if err != nil || !bytes.Equal(raw, manifest.RequestJSON) {
		t.Fatalf("stored raw mismatch: err=%v raw=%q", err, raw)
	}
	entries, err := os.ReadDir(store.RootDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files leaked: entries=%v err=%v", entries, err)
	}
	conflictRequest := request
	conflictRequest.BranchRef = "branch-ref-intent-manifest-conflict"
	conflict, conflictIssues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(conflictRequest)
	if len(conflictIssues) != 0 {
		t.Fatalf("conflict issues=%+v", conflictIssues)
	}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), conflict); !errors.Is(err, errAutoprogrammingIntentManifestConflictV0) {
		t.Fatalf("conflict err=%v", err)
	}
	symlinkRequest := request
	symlinkRequest.RequestRef = "request-ref-intent-manifest-symlink"
	symlinkManifest, symlinkIssues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(symlinkRequest)
	if len(symlinkIssues) != 0 {
		t.Fatalf("symlink issues=%+v", symlinkIssues)
	}
	if err := os.Symlink(filepath.Join(store.RootDir, "other"), filepath.Join(store.RootDir, symlinkRequest.RequestRef+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), symlinkManifest); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("symlink err=%v", err)
	}
}

func TestOpenPrivateIntentManifestDirV0RejectsEmptyPathV0(t *testing.T) {
	if dir, err := openPrivateIntentManifestDirV0("  ", false); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) || dir != nil {
		t.Fatalf("empty path opened current directory: dir=%v err=%v", dir, err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0RejectsBeforePublishingInvalidBytesV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-intent-manifest-prepublish"
	valid, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	invalid := valid
	invalid.RequestJSON = append(append([]byte(nil), valid.RequestJSON...), ' ')
	sum := sha256.Sum256(invalid.RequestJSON)
	invalid.RequestSHA256 = hex.EncodeToString(sum[:])
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), invalid); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("invalid create err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(store.RootDir, request.RequestRef+".json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid final published: %v", err)
	}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), valid); err != nil {
		t.Fatalf("valid create after rejection: %v", err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0RejectsNonRegularAndOversizedWithoutBlockingV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-intent-manifest-obstacle"
	root := newIntentManifestStoreRootForTestV0(t)
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: root}
	path := filepath.Join(root, request.RequestRef+".json")
	if err := syscall.Mkfifo(path, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := store.LoadAutoprogrammingIntentManifestV0(context.Background(), request.RequestRef)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
			t.Fatalf("FIFO err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO blocked manifest load")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, maxAutoprogrammingIntentManifestBytesV0+1), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadAutoprogrammingIntentManifestV0(context.Background(), request.RequestRef); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("oversized err=%v", err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0RejectsWorldWritableAncestorV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-intent-manifest-ancestor"
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	unsafeParent := filepath.Join(t.TempDir(), "unsafe")
	if err := os.Mkdir(unsafeParent, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafeParent, 0o777); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(unsafeParent, "manifests")
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: root}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("unsafe ancestor err=%v", err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created under unsafe ancestor: %v", err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0ForcesReadOnlyModeDespiteUmaskV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-intent-manifest-umask"
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	if err := os.Mkdir(store.RootDir, 0o700); err != nil {
		t.Fatal(err)
	}
	previous := syscall.Umask(0o777)
	_, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest)
	syscall.Umask(previous)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.RootDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	info, err := entries[0].Info()
	if err != nil || info.Mode().Perm() != 0o400 {
		t.Fatalf("mode=%v err=%v", info.Mode(), err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0ConcurrentDivergentNeverPublishesPartialV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-intent-manifest-divergent-race"
	left, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	request.BranchRef = "branch-ref-intent-manifest-divergent-race"
	right, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, candidate := range []orquestaautoprogramming.AutoprogrammingIntentManifestV0{left, right} {
		candidate := candidate
		go func() {
			<-start
			_, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), candidate)
			errs <- err
		}()
	}
	close(start)
	var accepted, conflicts int
	for range 2 {
		err := <-errs
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, errAutoprogrammingIntentManifestConflictV0):
			conflicts++
		default:
			t.Fatalf("unexpected race error: %v", err)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("accepted=%d conflicts=%d", accepted, conflicts)
	}
	stored, err := store.LoadAutoprogrammingIntentManifestV0(context.Background(), request.RequestRef)
	if err != nil || (!bytes.Equal(stored.RequestJSON, left.RequestJSON) && !bytes.Equal(stored.RequestJSON, right.RequestJSON)) {
		t.Fatalf("partial or foreign bytes stored: manifest=%+v err=%v", stored, err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0RejectsUnsafeRootPermissionsAndPartialV0(t *testing.T) {
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-intent-manifest-unsafe"
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}

	t.Run("root symlink", func(t *testing.T) {
		parent, outside := t.TempDir(), t.TempDir()
		root := filepath.Join(parent, "manifests")
		if err := os.Symlink(outside, root); err != nil {
			t.Fatal(err)
		}
		store := serverAutoprogrammingIntentManifestStoreV0{RootDir: root}
		if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
			t.Fatalf("root symlink err=%v", err)
		}
		if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
			t.Fatalf("wrote outside root: entries=%v err=%v", entries, err)
		}
	})

	t.Run("expanded final permissions", func(t *testing.T) {
		store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
		if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(store.RootDir, manifest.RequestRef+".json")
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.LoadAutoprogrammingIntentManifestV0(context.Background(), manifest.RequestRef); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
			t.Fatalf("expanded permissions accepted: %v", err)
		}
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("evidence permissions repaired: %v", info.Mode())
		}
	})

	t.Run("partial final", func(t *testing.T) {
		root := newIntentManifestStoreRootForTestV0(t)
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, manifest.RequestRef+".json")
		if err := os.WriteFile(path, []byte("{"), 0o400); err != nil {
			t.Fatal(err)
		}
		store := serverAutoprogrammingIntentManifestStoreV0{RootDir: root}
		if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); !errors.Is(err, errAutoprogrammingIntentManifestConflictV0) {
			t.Fatalf("partial final err=%v", err)
		}
		raw, _ := os.ReadFile(path)
		if string(raw) != "{" {
			t.Fatalf("partial evidence overwritten: %q", raw)
		}
	})

	if _, err := (serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}).LoadAutoprogrammingIntentManifestV0(context.Background(), "../escape"); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("traversal err=%v", err)
	}
}

func TestAutoprogrammingIntentManifestStoreV0RejectsHardlinkV0(t *testing.T) {
	root := filepath.Join(t.TempDir(), "intent-manifests")
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: root}
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-manifest-hardlink-001"
	manifest, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(request)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if _, err := store.CreateAutoprogrammingIntentManifestIfAbsentV0(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	path, _, err := store.pathV0(manifest.RequestRef)
	if err != nil {
		t.Fatal(err)
	}
	link := path + ".hardlink"
	if err := os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadAutoprogrammingIntentManifestV0(context.Background(), manifest.RequestRef); !errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("hardlink accepted: %v", err)
	}
}

func newIntentManifestStoreRootForTestV0(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "manifests")
}
