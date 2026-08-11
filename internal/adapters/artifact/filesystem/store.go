package filesystem

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"strings"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const artifactRefPrefix = "artifact:sha256:"

type Store struct {
	root              *os.Root
	projectDigest     string
	namespace         string
	syncDirectoryFn   func(*os.Root, string) error
	readVerifyHook    func()
	readBytesHook     func(int)
	tempCreateHook    func(string)
	beforePublishHook func(string)
}

func Open(rootPath string) (*Store, error) {
	return openStore(rootPath, (*os.File).Sync)
}

// OpenForProject binds one store to an opaque project namespace while keeping
// Open available for local single-project installations. The raw project ref
// never becomes a filesystem path.
func OpenForProject(rootPath string, projectRef goal.ProjectRef) (*Store, error) {
	if projectRef.String() == "" {
		return nil, artifactError(ports.ArtifactErrorStoreUnavailable, nil)
	}
	projectDigest := sha256.Sum256([]byte(projectRef.String()))
	projectDigestText := hex.EncodeToString(projectDigest[:])
	store, err := openStore(rootPath, (*os.File).Sync)
	if err != nil {
		return nil, err
	}
	store.projectDigest = projectDigestText
	store.namespace = path.Join("projects", projectDigestText)
	return store, nil
}

func openStore(rootPath string, rootSyncFn func(*os.File) error) (*Store, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, artifactError(ports.ArtifactErrorRootRequired, nil)
	}
	if rootSyncFn == nil {
		rootSyncFn = (*os.File).Sync
	}
	root, err := openPrivateRoot(rootPath, rootSyncFn)
	if err != nil {
		return nil, err
	}
	return &Store{root: root, syncDirectoryFn: syncDirectory}, nil
}

func (store *Store) Close() error {
	if store == nil || store.root == nil {
		return nil
	}
	if err := store.root.Close(); err != nil {
		return artifactError(ports.ArtifactErrorIO, nil)
	}
	return nil
}

func (store *Store) Put(ctx context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	if store == nil || store.root == nil {
		return ports.StoredArtifact{}, artifactError(ports.ArtifactErrorStoreUnavailable, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.StoredArtifact{}, artifactError(ports.ArtifactErrorIO, err)
	}
	if err := ports.ValidateArtifactMediaType(request.MediaType); err != nil {
		return ports.StoredArtifact{}, err
	}

	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef(artifactRefPrefix + digestText)
	if err != nil {
		return ports.StoredArtifact{}, artifactError(ports.ArtifactErrorIO, nil)
	}
	stored := ports.StoredArtifact{
		Ref: ref, Digest: digestText, MediaType: request.MediaType,
		Size: int64(len(request.Content)),
	}
	finalPath := store.blobPath(digestText)
	if err := store.ensureExistingOrWrite(ctx, finalPath, request.Content, digestText); err != nil {
		return ports.StoredArtifact{}, err
	}
	if store.projectDigest != "" {
		metadata := artifactMetadata{
			Schema: metadataSchema, ProjectDigest: store.projectDigest, ArtifactDigest: digestText,
			MediaType: request.MediaType, Size: stored.Size,
		}
		if err := store.ensureArtifactMetadata(ctx, digestText, metadata); err != nil {
			return ports.StoredArtifact{}, err
		}
	}
	return stored, nil
}

func (store *Store) Get(ctx context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	if store == nil || store.root == nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorStoreUnavailable, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorIO, err)
	}
	digest, err := digestFromRef(ref)
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	if expectedSize < 0 {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorExpectedSizeInvalid, nil)
	}
	filePath := store.blobPath(digest)
	mediaType := ""
	if store.projectDigest != "" {
		metadata, metadataErr := store.readArtifactMetadata(digest, expectedSize)
		if errors.Is(metadataErr, os.ErrNotExist) {
			if _, blobErr := store.readVerifiedBlob(filePath, digest, expectedSize); errors.Is(blobErr, os.ErrNotExist) {
				return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorNotFound, nil)
			} else if blobErr != nil {
				return ports.ArtifactContent{}, blobErr
			}
			return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorFileChanged, nil)
		}
		if metadataErr != nil {
			return ports.ArtifactContent{}, metadataErr
		}
		mediaType = metadata.MediaType
	}
	content, err := store.readVerifiedBlob(filePath, digest, expectedSize)
	if errors.Is(err, os.ErrNotExist) {
		return ports.ArtifactContent{}, artifactError(ports.ArtifactErrorNotFound, nil)
	}
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	return ports.ArtifactContent{
		Ref: ref, Digest: digest, MediaType: mediaType, Size: int64(len(content)), Content: content,
	}, nil
}

func (store *Store) ensureExistingOrWrite(ctx context.Context, finalPath string, content []byte, digest string) error {
	directory := path.Dir(finalPath)
	if err := store.ensureDurableDirectory(directory); err != nil {
		return err
	}
	if _, err := store.readVerifiedBlob(finalPath, digest, int64(len(content))); err == nil {
		return store.syncDirectory(directory)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tempPath, err := store.tempPath(directory)
	if err != nil {
		return err
	}
	file, err := store.root.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return artifactError(ports.ArtifactErrorIO, nil)
	}
	defer func() {
		_ = file.Close()
		_ = store.root.Remove(tempPath)
	}()
	if store.tempCreateHook != nil {
		store.tempCreateHook(tempPath)
	}
	if err := validateOpenPrivateFile(store.root, tempPath, file, 0); err != nil {
		return err
	}

	if written, err := file.Write(content); err != nil || written != len(content) {
		return artifactError(ports.ArtifactErrorIO, nil)
	}
	if err := ctx.Err(); err != nil {
		return artifactError(ports.ArtifactErrorIO, err)
	}
	if err := file.Sync(); err != nil {
		return artifactError(ports.ArtifactErrorIO, nil)
	}
	if err := validateOpenPrivateFile(store.root, tempPath, file, int64(len(content))); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return artifactError(ports.ArtifactErrorIO, nil)
	}
	published, err := store.publishNoReplace(tempPath, finalPath)
	if err != nil {
		return err
	}
	if !published {
		if _, err := store.readVerifiedBlob(finalPath, digest, int64(len(content))); err != nil {
			return err
		}
		if err := store.root.Remove(tempPath); err != nil {
			return artifactError(ports.ArtifactErrorIO, nil)
		}
	}
	if _, err := store.readVerifiedBlob(finalPath, digest, int64(len(content))); err != nil {
		return err
	}
	return store.syncDirectory(directory)
}

func (store *Store) syncDirectory(directory string) error {
	syncFn := store.syncDirectoryFn
	if syncFn == nil {
		syncFn = syncDirectory
	}
	err := syncFn(store.root, directory)
	if err == nil || ports.ArtifactContractErrorCode(err) != "" {
		return err
	}
	return artifactError(ports.ArtifactErrorIO, err)
}

func (store *Store) tempPath(directory string) (string, error) {
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", artifactError(ports.ArtifactErrorIO, nil)
	}
	return path.Join(directory, ".artifact-"+hex.EncodeToString(suffix[:])+".tmp"), nil
}

func syncDirectory(root *os.Root, directory string) error {
	handle, before, err := openPrivateDirectory(root, directory)
	if err != nil {
		return err
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return artifactError(ports.ArtifactErrorIO, err)
	}
	return verifyOpenPrivateDirectory(root, directory, handle, before)
}

func digestFromRef(ref goal.ArtifactRef) (string, error) {
	digest, found := strings.CutPrefix(ref.String(), artifactRefPrefix)
	if !found || len(digest) != sha256.Size*2 {
		return "", artifactError(ports.ArtifactErrorRefInvalid, nil)
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return "", artifactError(ports.ArtifactErrorRefInvalid, nil)
	}
	return digest, nil
}

func blobPath(digest string) string {
	return path.Join("sha256", digest[:2], digest+".blob")
}

func (store *Store) blobPath(digest string) string {
	return path.Join(store.namespace, blobPath(digest))
}

var _ application.ArtifactStore = (*Store)(nil)

func artifactError(code string, cause error) error {
	return ports.NewArtifactContractError(code, cause)
}
