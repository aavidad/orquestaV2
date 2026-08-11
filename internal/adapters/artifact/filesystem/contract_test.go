//go:build linux

package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"orquesta/internal/adapters/artifact/contracttest"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestProjectStorePassesSharedArtifactContract(t *testing.T) {
	backend := &filesystemContractBackend{root: privateTestDirectory(t)}
	contracttest.Run(t, contracttest.Backend{
		Open:                   backend.open,
		LogicalObjectCount:     backend.logicalObjectCount,
		TamperContent:          backend.tamperContent,
		TamperMetadata:         backend.tamperMetadata,
		TamperOversizedContent: backend.tamperContent,
		ResetReadCount:         func() { backend.readCount.Store(0) },
		ReadCount:              backend.readCount.Load,
		AssertProjectRefOpaque: backend.assertProjectRefOpaque,
	})
}

func TestProjectStoreRepairsVerifiedBlobWithoutMetadata(t *testing.T) {
	root := privateTestDirectory(t)
	project, err := goal.NewProjectRef("project:repair-frontier")
	artifactTestNoError(t, err)
	store, err := OpenForProject(root, project)
	artifactTestNoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("verified blob frontier")}
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	if err := store.ensureExistingOrWrite(context.Background(), store.blobPath(digestText), request.Content, digestText); err != nil {
		t.Fatalf("create blob-only frontier: %v", err)
	}
	metadataName := filepath.Join(root, filepath.FromSlash(store.metadataPath(digestText)))
	if _, err := os.Lstat(metadataName); !os.IsNotExist(err) {
		t.Fatalf("metadata exists before repair: %v", err)
	}
	ref, err := goal.NewArtifactRef(artifactRefPrefix + digestText)
	artifactTestNoError(t, err)
	if _, err := store.Get(context.Background(), ref, int64(len(request.Content))); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorFileChanged {
		t.Fatalf("blob-only frontier read error=%v", err)
	}

	stored, err := store.Put(context.Background(), request)
	artifactTestNoError(t, err)
	content, err := store.Get(context.Background(), stored.Ref, stored.Size)
	artifactTestNoError(t, err)
	if content.MediaType != request.MediaType || string(content.Content) != string(request.Content) {
		t.Fatalf("repaired content=%+v", content)
	}
	for _, name := range []string{
		filepath.Join(root, filepath.FromSlash(store.blobPath(digestText))), metadataName,
	} {
		info, err := os.Lstat(name)
		artifactTestNoError(t, err)
		if info.Mode() != 0o600 {
			t.Fatalf("private file %s mode=%v", filepath.Base(name), info.Mode())
		}
	}
	encoded, err := os.ReadFile(metadataName)
	artifactTestNoError(t, err)
	metadata, err := decodeArtifactMetadata(encoded)
	artifactTestNoError(t, err)
	if metadata.MediaType != request.MediaType || metadata.ArtifactDigest != stored.Digest || metadata.Size != stored.Size {
		t.Fatalf("repaired metadata=%+v", metadata)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenForProject(root, project)
	artifactTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	restartedContent, err := restarted.Get(context.Background(), stored.Ref, stored.Size)
	artifactTestNoError(t, err)
	if restartedContent.MediaType != request.MediaType || string(restartedContent.Content) != string(request.Content) {
		t.Fatalf("restarted repaired content=%+v", restartedContent)
	}
}

func TestProjectStoreRetriesMetadataDirectorySyncFrontier(t *testing.T) {
	root := privateTestDirectory(t)
	project, err := goal.NewProjectRef("project:metadata-sync-frontier")
	artifactTestNoError(t, err)
	store, err := OpenForProject(root, project)
	artifactTestNoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("metadata directory sync")}
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	shard := filepath.ToSlash(filepath.Dir(store.blobPath(digestText)))
	realSync := store.syncDirectoryFn
	injected := errors.New("injected metadata directory sync failure")
	shardSyncs := 0
	store.syncDirectoryFn = func(root *os.Root, directory string) error {
		if directory == shard {
			shardSyncs++
			if shardSyncs == 2 {
				return injected
			}
		}
		return realSync(root, directory)
	}
	if stored, err := store.Put(context.Background(), request); !errors.Is(err, injected) || stored.Ref.String() != "" {
		t.Fatalf("metadata sync frontier stored=%+v error=%v", stored, err)
	}
	store.syncDirectoryFn = realSync
	stored, err := store.Put(context.Background(), request)
	artifactTestNoError(t, err)
	content, err := store.Get(context.Background(), stored.Ref, stored.Size)
	artifactTestNoError(t, err)
	if content.MediaType != request.MediaType || string(content.Content) != string(request.Content) {
		t.Fatalf("metadata sync replay content=%+v", content)
	}
}

func TestProjectStoreDoesNotRepairCorruptBlobOrMetadata(t *testing.T) {
	for name, mutate := range map[string]func(*Store, string, string) error{
		"blob": func(store *Store, root, digest string) error {
			return os.WriteFile(filepath.Join(root, filepath.FromSlash(store.blobPath(digest))), []byte("CORRUPT FRONTIER"), 0o600)
		},
		"metadata": func(store *Store, root, digest string) error {
			metadataName := filepath.Join(root, filepath.FromSlash(store.metadataPath(digest)))
			encoded, err := os.ReadFile(metadataName)
			if err != nil {
				return err
			}
			metadata, err := decodeArtifactMetadata(encoded)
			if err != nil {
				return err
			}
			metadata.ProjectDigest = strings.Repeat("0", sha256.Size*2)
			return os.WriteFile(metadataName, encodeArtifactMetadata(metadata), 0o600)
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := privateTestDirectory(t)
			project, err := goal.NewProjectRef("project:corrupt-" + name)
			artifactTestNoError(t, err)
			store, err := OpenForProject(root, project)
			artifactTestNoError(t, err)
			t.Cleanup(func() { _ = store.Close() })
			request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("corrupt frontier")}
			stored, err := store.Put(context.Background(), request)
			artifactTestNoError(t, err)
			artifactTestNoError(t, mutate(store, root, stored.Digest))
			if replay, err := store.Put(context.Background(), request); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorDigestMismatch || replay.Ref.String() != "" {
				t.Fatalf("corrupt replay stored=%+v error=%v", replay, err)
			}
		})
	}
}

func TestProjectStoreRejectsUnsafeMetadataSidecarWithoutRepair(t *testing.T) {
	for name, mutate := range map[string]func(string, string) error{
		"mode": func(_, metadataName string) error { return os.Chmod(metadataName, 0o640) },
		"hardlink": func(root, metadataName string) error {
			return os.Link(metadataName, filepath.Join(root, "metadata-second-name"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := privateTestDirectory(t)
			project, err := goal.NewProjectRef("project:unsafe-metadata-" + name)
			artifactTestNoError(t, err)
			store, err := OpenForProject(root, project)
			artifactTestNoError(t, err)
			t.Cleanup(func() { _ = store.Close() })
			request := ports.PutArtifactRequest{MediaType: "text/plain", Content: []byte("unsafe metadata sidecar")}
			stored, err := store.Put(context.Background(), request)
			artifactTestNoError(t, err)
			metadataName := filepath.Join(root, filepath.FromSlash(store.metadataPath(stored.Digest)))
			artifactTestNoError(t, mutate(root, metadataName))

			if _, err := store.Get(context.Background(), stored.Ref, stored.Size); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorFileInvalid {
				t.Fatalf("unsafe metadata Get error=%v", err)
			}
			if replay, err := store.Put(context.Background(), request); ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorFileInvalid || replay.Ref.String() != "" {
				t.Fatalf("unsafe metadata replay=%+v error=%v", replay, err)
			}
		})
	}
}

func TestOpenForProjectRejectsEmptyProjectBeforeCreatingRoot(t *testing.T) {
	root := filepath.Join(privateTestDirectory(t), "not-created")
	store, err := OpenForProject(root, goal.ProjectRef{})
	if store != nil || ports.ArtifactContractErrorCode(err) != ports.ArtifactErrorStoreUnavailable {
		t.Fatalf("OpenForProject(empty) store=%v error=%v", store, err)
	}
	if _, statErr := os.Lstat(root); !os.IsNotExist(statErr) {
		t.Fatalf("invalid project created root: %v", statErr)
	}
}

type filesystemContractBackend struct {
	root      string
	readCount atomic.Int64
}

func (backend *filesystemContractBackend) open(t testing.TB, project goal.ProjectRef) contracttest.Handle {
	t.Helper()
	store, err := OpenForProject(backend.root, project)
	if err != nil {
		t.Fatal(err)
	}
	store.readBytesHook = func(count int) { backend.readCount.Add(int64(count)) }
	var once sync.Once
	var closeErr error
	closeFn := func() error {
		once.Do(func() { closeErr = store.Close() })
		return closeErr
	}
	t.Cleanup(func() { _ = closeFn() })
	return contracttest.Handle{Store: store, Close: closeFn}
}

func (backend *filesystemContractBackend) logicalObjectCount(t testing.TB) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(backend.root, func(name string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(name, ".blob") {
			count++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func (backend *filesystemContractBackend) tamperContent(t testing.TB, handle contracttest.Handle, stored ports.StoredArtifact, content []byte) {
	t.Helper()
	store := handle.Store.(*Store)
	if err := os.WriteFile(filepath.Join(backend.root, filepath.FromSlash(store.blobPath(stored.Digest))), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (backend *filesystemContractBackend) tamperMetadata(t testing.TB, handle contracttest.Handle, stored ports.StoredArtifact) {
	t.Helper()
	store := handle.Store.(*Store)
	name := filepath.Join(backend.root, filepath.FromSlash(store.metadataPath(stored.Digest)))
	encoded, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := decodeArtifactMetadata(encoded)
	if err != nil {
		t.Fatal(err)
	}
	metadata.ProjectDigest = strings.Repeat("f", sha256.Size*2)
	if err := os.WriteFile(name, encodeArtifactMetadata(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (backend *filesystemContractBackend) assertProjectRefOpaque(t testing.TB, project goal.ProjectRef) {
	t.Helper()
	err := filepath.WalkDir(backend.root, func(name string, _ os.DirEntry, err error) error {
		if err == nil && strings.Contains(name, project.String()) {
			return &projectRefLeakError{name: name}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

type projectRefLeakError struct{ name string }

func (err *projectRefLeakError) Error() string { return "project ref leaked in path: " + err.name }
