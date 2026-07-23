//go:build linux

package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func artifactTestNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenRejectsUnsafePreexistingRootAndAncestorWithoutRepair(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		base := privateTestDirectory(t)
		root := filepath.Join(base, "artifacts")
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatal(err)
		}
		assertOpenRejectedWithoutPath(t, root, ports.ArtifactErrorRootPermissions)
		assertMode(t, root, 0o755)
	})
	t.Run("ancestor", func(t *testing.T) {
		base := privateTestDirectory(t)
		ancestor := filepath.Join(base, "shared")
		if err := os.Mkdir(ancestor, 0o770); err != nil {
			t.Fatal(err)
		}
		// Mkdir applies the process umask. Force the unsafe fixture mode so this
		// security regression test is deterministic under both 0002 and 0022.
		if err := os.Chmod(ancestor, 0o770); err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(ancestor, "artifacts")
		assertOpenRejectedWithoutPath(t, root, ports.ArtifactErrorRootPermissions)
		assertMode(t, ancestor, 0o770)
		if _, err := os.Lstat(root); !os.IsNotExist(err) {
			t.Fatalf("unsafe ancestor child created: %v", err)
		}
	})
}

func TestStoreRejectsUnsafePreexistingShardWithoutRepair(t *testing.T) {
	for _, target := range []string{"sha256", "digest_shard"} {
		t.Run(target, func(t *testing.T) {
			root := privateTestDirectory(t)
			store, err := Open(root)
			artifactTestNoError(t, err)
			t.Cleanup(func() { _ = store.Close() })
			content := []byte("unsafe shard")
			digest := sha256.Sum256(content)
			digestText := hex.EncodeToString(digest[:])
			shard := filepath.Join(root, "sha256")
			if err := os.Mkdir(shard, 0o700); err != nil {
				t.Fatal(err)
			}
			if target == "digest_shard" {
				shard = filepath.Join(shard, digestText[:2])
				if err := os.Mkdir(shard, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(shard, 0o755); err != nil {
				t.Fatal(err)
			}
			_, err = store.Put(context.Background(), ports.PutArtifactRequest{MediaType: "text/plain", Content: content})
			assertArtifactCode(t, err, ports.ArtifactErrorDirectoryInvalid)
			assertMode(t, shard, 0o755)
		})
	}
}

func TestStoreRejectsBlobHardlinkModeAndFIFO(t *testing.T) {
	for name, mutate := range map[string]func(string, string) error{
		"hardlink": func(root, blob string) error { return os.Link(blob, filepath.Join(root, "second-name")) },
		"mode":     func(_, blob string) error { return os.Chmod(blob, 0o640) },
	} {
		t.Run(name, func(t *testing.T) {
			store, root, stored := storedSecurityFixture(t, []byte(name+" blob"))
			blob := filepath.Join(root, filepath.FromSlash(blobPath(stored.Digest)))
			if err := mutate(root, blob); err != nil {
				t.Fatal(err)
			}
			_, err := store.Get(context.Background(), stored.Ref, stored.Size)
			assertArtifactCode(t, err, ports.ArtifactErrorFileInvalid)
			if name == "mode" {
				assertMode(t, blob, 0o640)
			}
		})
	}
	t.Run("fifo", func(t *testing.T) {
		root := privateTestDirectory(t)
		store, err := Open(root)
		artifactTestNoError(t, err)
		t.Cleanup(func() { _ = store.Close() })
		content := []byte("fifo target")
		digest := sha256.Sum256(content)
		digestText := hex.EncodeToString(digest[:])
		final := filepath.Join(root, filepath.FromSlash(blobPath(digestText)))
		if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(final, 0o600); err != nil {
			t.Skipf("fifo unsupported: %v", err)
		}
		_, err = store.Put(context.Background(), ports.PutArtifactRequest{MediaType: "text/plain", Content: content})
		assertArtifactCode(t, err, ports.ArtifactErrorFileInvalid)
	})
}

func TestStoreRejectsUnsafeTempAndReadTOCTOU(t *testing.T) {
	t.Run("temp_mode", func(t *testing.T) {
		root := privateTestDirectory(t)
		store, err := Open(root)
		artifactTestNoError(t, err)
		t.Cleanup(func() { _ = store.Close() })
		store.tempCreateHook = func(name string) {
			_ = os.Chmod(filepath.Join(root, filepath.FromSlash(name)), 0o640)
		}
		_, err = store.Put(context.Background(), ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("temp")})
		assertArtifactCode(t, err, ports.ArtifactErrorFileInvalid)
	})
	for name, mutate := range map[string]func(string){
		"read_mode_drift": func(blob string) { _ = os.Chmod(blob, 0o640) },
		"read_replacement": func(blob string) {
			_ = os.Rename(blob, blob+".old")
			_ = os.WriteFile(blob, []byte("read race"), 0o600)
		},
		"read_in_place_same_size": func(blob string) { _ = os.WriteFile(blob, []byte("evil race"), 0o600) },
	} {
		t.Run(name, func(t *testing.T) {
			store, root, stored := storedSecurityFixture(t, []byte("read race"))
			blob := filepath.Join(root, filepath.FromSlash(blobPath(stored.Digest)))
			store.readVerifyHook = func() { mutate(blob) }
			_, err := store.Get(context.Background(), stored.Ref, stored.Size)
			assertArtifactCode(t, err, ports.ArtifactErrorFileChanged)
		})
	}
}

func TestStoreRejectsTempSwapImmediatelyBeforePublish(t *testing.T) {
	root := privateTestDirectory(t)
	store, err := Open(root)
	artifactTestNoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	content, replacement := []byte("expected"), []byte("tampered")
	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	evidence := ""
	store.beforePublishHook = func(name string) {
		temporary := filepath.Join(root, filepath.FromSlash(name))
		evidence = temporary + ".validated"
		if err := os.Rename(temporary, evidence); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(temporary, replacement, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: content}
	stored, err := store.Put(context.Background(), request)
	assertArtifactCode(t, err, ports.ArtifactErrorDigestMismatch)
	if stored.Ref.String() != "" {
		t.Fatalf("failed Put returned ref %q", stored.Ref)
	}
	final := filepath.Join(root, filepath.FromSlash(blobPath(digestText)))
	assertFileContent(t, evidence, content)
	assertFileContent(t, final, replacement)

	store.beforePublishHook = nil
	retried, err := store.Put(context.Background(), request)
	assertArtifactCode(t, err, ports.ArtifactErrorDigestMismatch)
	if retried.Ref.String() != "" {
		t.Fatalf("failed retry returned ref %q", retried.Ref)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root)
	artifactTestNoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	ref, err := goal.NewArtifactRef(artifactRefPrefix + digestText)
	artifactTestNoError(t, err)
	_, err = reopened.Get(context.Background(), ref, int64(len(content)))
	assertArtifactCode(t, err, ports.ArtifactErrorDigestMismatch)
	assertFileContent(t, evidence, content)
	assertFileContent(t, final, replacement)
}

func TestPrivateMetadataOwnerSeamRejectsForeignEUID(t *testing.T) {
	root := privateTestDirectory(t)
	directoryInfo, err := os.Lstat(root)
	artifactTestNoError(t, err)
	directoryStat := *directoryInfo.Sys().(*syscall.Stat_t)
	directoryStat.Uid++
	foreignDirectory := fileInfoWithStat{FileInfo: directoryInfo, stat: &directoryStat}
	if privateDirectoryValid(foreignDirectory, os.Geteuid()) {
		t.Fatal("foreign-owned directory accepted")
	}

	file := filepath.Join(root, "blob")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(file)
	artifactTestNoError(t, err)
	stat := *info.Sys().(*syscall.Stat_t)
	stat.Uid++
	foreign := fileInfoWithStat{FileInfo: info, stat: &stat}
	if privateFileValid(foreign, 1, os.Geteuid()) {
		t.Fatal("foreign-owned file accepted")
	}
}

func TestStoreConcurrentPutPreservesSinglePrivateBlob(t *testing.T) {
	store := openTestStore(t)
	stores := make([]*Store, 24)
	for index := range stores {
		stores[index] = store
	}
	assertConcurrentPut(t, stores, []byte("concurrent artifact"))
}

func TestConcurrentStoreHandlesPublishOnePrivateBlob(t *testing.T) {
	root := privateTestDirectory(t)
	stores := make([]*Store, 12)
	for index := range stores {
		store, err := Open(root)
		artifactTestNoError(t, err)
		stores[index] = store
		t.Cleanup(func() { _ = store.Close() })
	}
	results := assertConcurrentPut(t, stores, []byte("cross-store CAS"))
	blob := filepath.Join(root, filepath.FromSlash(blobPath(results[0].Digest)))
	info, err := os.Lstat(blob)
	artifactTestNoError(t, err)
	stat := info.Sys().(*syscall.Stat_t)
	if info.Mode() != 0o600 || stat.Nlink != 1 {
		t.Fatalf("blob mode=%v nlink=%d", info.Mode(), stat.Nlink)
	}
}

func assertConcurrentPut(t *testing.T, stores []*Store, content []byte) []ports.StoredArtifact {
	t.Helper()
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: content}
	results := make([]ports.StoredArtifact, len(stores))
	errorsFound := make([]error, len(stores))
	var group sync.WaitGroup
	for index, store := range stores {
		group.Add(1)
		go func() {
			defer group.Done()
			results[index], errorsFound[index] = store.Put(context.Background(), request)
		}()
	}
	group.Wait()
	for index, err := range errorsFound {
		if err != nil || results[index] != results[0] {
			t.Fatalf("Put[%d] result=%+v err=%v", index, results[index], err)
		}
	}
	return results
}

type fileInfoWithStat struct {
	os.FileInfo
	stat *syscall.Stat_t
}

func (info fileInfoWithStat) Sys() any { return info.stat }

func privateTestDirectory(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func storedSecurityFixture(t *testing.T, content []byte) (*Store, string, ports.StoredArtifact) {
	t.Helper()
	root := privateTestDirectory(t)
	store, err := Open(root)
	artifactTestNoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	stored, err := store.Put(context.Background(), ports.PutArtifactRequest{MediaType: "text/plain", Content: content})
	artifactTestNoError(t, err)
	return store, root, stored
}

func assertOpenRejectedWithoutPath(t *testing.T, root, code string) {
	t.Helper()
	store, err := Open(root)
	if store != nil {
		_ = store.Close()
		t.Fatal("unsafe root opened")
	}
	assertArtifactCode(t, err, code)
	if strings.Contains(err.Error(), root) {
		t.Fatalf("public error leaked root: %q", err)
	}
}

func assertArtifactCode(t *testing.T, err error, code string) {
	t.Helper()
	if got := ports.ArtifactContractErrorCode(err); got != code {
		t.Fatalf("error=%v code=%q want=%q", err, got, code)
	}
}

func assertMode(t *testing.T, name string, mode os.FileMode) {
	t.Helper()
	info, err := os.Lstat(name)
	artifactTestNoError(t, err)
	if info.Mode().Perm() != mode {
		t.Fatalf("mode=%o want=%o", info.Mode().Perm(), mode)
	}
}

func assertFileContent(t *testing.T, name string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(name)
	artifactTestNoError(t, err)
	if string(got) != string(want) {
		t.Fatalf("content at %s = %q, want %q", name, got, want)
	}
}
