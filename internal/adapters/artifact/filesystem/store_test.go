package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestStorePutGetIsContentAddressedAndIdempotent(t *testing.T) {
	store := openTestStore(t)
	request := ports.PutArtifactRequest{MediaType: "text/markdown", Content: []byte("artifact")}

	first, err := store.Put(context.Background(), request)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	second, err := store.Put(context.Background(), request)
	if err != nil {
		t.Fatalf("Put() second error = %v", err)
	}
	if first != second {
		t.Fatalf("idempotent Put changed metadata: first=%+v second=%+v", first, second)
	}
	got, err := store.Get(context.Background(), first.Ref, first.Size)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got.Content) != "artifact" || got.Digest != first.Digest || got.Size != first.Size {
		t.Fatalf("Get() = %+v", got)
	}
}

func TestStoreUsesPrivateFilesAndRejectsCorruption(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	stored, err := store.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain",
		Content:   []byte("original"),
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	filePath := filepath.Join(root, filepath.FromSlash(blobPath(stored.Digest)))
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("artifact mode = %o", got)
	}
	if err := os.WriteFile(filePath, []byte("changed!"), 0o600); err != nil {
		t.Fatalf("corrupt fixture: %v", err)
	}
	if _, err := store.Get(context.Background(), stored.Ref, stored.Size); err == nil || err.Error() != "artifact.digest_mismatch" {
		t.Fatalf("Get() corruption error = %v", err)
	}
}

func TestStoreRejectsMalformedRefAndSymlinkEscape(t *testing.T) {
	store := openTestStore(t)
	badRef, _ := goal.NewArtifactRef("artifact:sha256:../outside")
	if _, err := store.Get(context.Background(), badRef, 0); err == nil {
		t.Fatal("malformed ref accepted")
	}

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "sha256")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	escapeStore, err := Open(root)
	if err != nil {
		t.Fatalf("Open() symlink fixture error = %v", err)
	}
	t.Cleanup(func() { _ = escapeStore.Close() })
	if _, err := escapeStore.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain",
		Content:   []byte("must stay inside"),
	}); err == nil {
		t.Fatal("symlink escape accepted")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatalf("ReadDir(outside) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("outside directory mutated: %v", entries)
	}
}

func TestStorePutRejectsSymlinkAtFinalBlobPath(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	content := []byte("correct content behind a symlink")
	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	finalPath := filepath.Join(root, filepath.FromSlash(blobPath(digestText)))
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(blob shard) error = %v", err)
	}
	targetPath := filepath.Join(root, "symlink-target.blob")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	relativeTarget, err := filepath.Rel(filepath.Dir(finalPath), targetPath)
	if err != nil {
		t.Fatalf("Rel(target) error = %v", err)
	}
	if err := os.Symlink(relativeTarget, finalPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if _, err := store.Put(context.Background(), ports.PutArtifactRequest{
		MediaType: "text/plain", Content: content,
	}); err == nil || !strings.Contains(err.Error(), "artifact.file_invalid") {
		t.Fatalf("Put() final symlink error = %v", err)
	}
}

func TestStorePutSynchronizesDirectoryPublicationCausallyAndSurvivesReopen(t *testing.T) {
	rootPath := t.TempDir()
	if err := os.Chmod(rootPath, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	store, err := Open(rootPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	realSync := store.syncDirectoryFn
	var synchronized []string
	store.syncDirectoryFn = func(root *os.Root, directory string) error {
		synchronized = append(synchronized, directory)
		return realSync(root, directory)
	}
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("durable artifact")}
	stored, err := store.Put(context.Background(), request)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	shardDirectory := path.Join("sha256", stored.Digest[:2])
	if want := []string{".", "sha256", shardDirectory}; !reflect.DeepEqual(synchronized, want) {
		t.Fatalf("directory sync order = %#v, want %#v", synchronized, want)
	}
	for _, directory := range []string{"sha256", shardDirectory} {
		info, err := os.Stat(filepath.Join(rootPath, filepath.FromSlash(directory)))
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", directory, err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("directory %s mode = %v", directory, info.Mode())
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := Open(rootPath)
	if err != nil {
		t.Fatalf("Open(restarted) error = %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	got, err := reopened.Get(context.Background(), stored.Ref, stored.Size)
	if err != nil {
		t.Fatalf("Get(restarted) error = %v", err)
	}
	if string(got.Content) != string(request.Content) || got.Digest != stored.Digest {
		t.Fatalf("Get(restarted) = %+v", got)
	}
}

func TestStorePutRetriesPublicationAfterDirectorySyncFailure(t *testing.T) {
	store := openTestStore(t)
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("retry durable publication")}
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	shardDirectory := path.Dir(blobPath(digestText))
	realSync := store.syncDirectoryFn
	injected := errors.New("injected directory sync failure")
	failOnce := true
	store.syncDirectoryFn = func(root *os.Root, directory string) error {
		if directory == shardDirectory && failOnce {
			failOnce = false
			return injected
		}
		return realSync(root, directory)
	}
	if stored, err := store.Put(context.Background(), request); !errors.Is(err, injected) || stored.Ref.String() != "" {
		t.Fatalf("Put(fault) stored=%+v error=%v", stored, err)
	}

	var retrySyncs []string
	store.syncDirectoryFn = func(root *os.Root, directory string) error {
		retrySyncs = append(retrySyncs, directory)
		return realSync(root, directory)
	}
	stored, err := store.Put(context.Background(), request)
	if err != nil {
		t.Fatalf("Put(retry) error = %v", err)
	}
	if want := []string{".", "sha256", shardDirectory}; !reflect.DeepEqual(retrySyncs, want) {
		t.Fatalf("retry sync order = %#v, want %#v", retrySyncs, want)
	}
	got, err := store.Get(context.Background(), stored.Ref, stored.Size)
	if err != nil {
		t.Fatalf("Get(retry) error = %v", err)
	}
	if string(got.Content) != string(request.Content) {
		t.Fatalf("Get(retry) content = %q", got.Content)
	}
}

func TestOpenCreatesRootCausallyAndStoreSurvivesReopen(t *testing.T) {
	base := t.TempDir()
	rootPath := filepath.Join(base, "level-one", "level-two", "artifacts")
	var synchronizedParents []string
	store, err := openStore(rootPath, func(root *os.Root, directory string) error {
		if directory != "." {
			return errors.New("unexpected root sync path")
		}
		synchronizedParents = append(synchronizedParents, filepath.Clean(root.Name()))
		return syncDirectory(root, directory)
	})
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	wantParents := []string{
		filepath.Clean(base),
		filepath.Join(base, "level-one"),
		filepath.Join(base, "level-one", "level-two"),
	}
	if !reflect.DeepEqual(synchronizedParents, wantParents) {
		t.Fatalf("root sync order = %#v, want %#v", synchronizedParents, wantParents)
	}
	store.syncDirectoryFn = syncDirectory
	for _, directory := range []string{
		filepath.Join(base, "level-one"),
		filepath.Join(base, "level-one", "level-two"),
		rootPath,
	} {
		info, err := os.Lstat(directory)
		if err != nil {
			t.Fatalf("Lstat(%s) error = %v", directory, err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("created root component %s mode = %v", directory, info.Mode())
		}
	}
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("root durability")}
	stored, err := store.Put(context.Background(), request)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := Open(rootPath)
	if err != nil {
		t.Fatalf("Open(restarted) error = %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	got, err := reopened.Get(context.Background(), stored.Ref, stored.Size)
	if err != nil {
		t.Fatalf("Get(restarted) error = %v", err)
	}
	if string(got.Content) != string(request.Content) {
		t.Fatalf("Get(restarted) content = %q", got.Content)
	}
}

func TestOpenRejectsIntermediateSymlinkWithoutCreatingTargetRoot(t *testing.T) {
	base := t.TempDir()
	target := t.TempDir()
	linkPath := filepath.Join(base, "linked-parent")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	store, err := Open(filepath.Join(linkPath, "artifacts"))
	if store != nil {
		_ = store.Close()
		t.Fatalf("Open() followed intermediate symlink")
	}
	if err == nil || err.Error() != "artifact.root_invalid" {
		t.Fatalf("Open() symlink error = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(target, "artifacts")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("symlink target was mutated: %v", statErr)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("Chmod(root) error = %v", err)
	}
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}
