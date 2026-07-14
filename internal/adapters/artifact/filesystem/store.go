package filesystem

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const artifactRefPrefix = "artifact:sha256:"

type Store struct {
	root            *os.Root
	syncDirectoryFn func(*os.Root, string) error
}

func Open(rootPath string) (*Store, error) {
	return openStore(rootPath, syncDirectory)
}

func openStore(rootPath string, syncFn func(*os.Root, string) error) (*Store, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, errors.New("artifact.root_required")
	}
	if syncFn == nil {
		syncFn = syncDirectory
	}
	root, err := openPrivateRoot(rootPath, syncFn)
	if err != nil {
		return nil, err
	}
	return &Store{root: root, syncDirectoryFn: syncFn}, nil
}

func openPrivateRoot(configuredPath string, syncFn func(*os.Root, string) error) (*os.Root, error) {
	absolutePath, err := filepath.Abs(configuredPath)
	if err != nil {
		return nil, fmt.Errorf("artifact.root_stat: %w", err)
	}
	anchor := filepath.VolumeName(absolutePath) + string(filepath.Separator)
	relativePath, err := filepath.Rel(anchor, absolutePath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return nil, errors.New("artifact.root_invalid")
	}
	current, err := os.OpenRoot(anchor)
	if err != nil {
		return nil, fmt.Errorf("artifact.root_open: %w", err)
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = current.Close()
		}
	}()

	if relativePath != "." {
		for _, component := range strings.Split(relativePath, string(filepath.Separator)) {
			if component == "" || component == "." || component == ".." {
				return nil, errors.New("artifact.root_invalid")
			}
			info, statErr := current.Lstat(component)
			created := false
			if errors.Is(statErr, fs.ErrNotExist) {
				if mkdirErr := current.Mkdir(component, 0o700); mkdirErr != nil {
					return nil, fmt.Errorf("artifact.root_create: %w", mkdirErr)
				}
				created = true
				info, statErr = current.Lstat(component)
			}
			if statErr != nil {
				return nil, fmt.Errorf("artifact.root_stat: %w", statErr)
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return nil, errors.New("artifact.root_invalid")
			}
			if created {
				if info.Mode().Perm()&0o077 != 0 {
					return nil, errors.New("artifact.root_permissions")
				}
				if syncErr := syncFn(current, "."); syncErr != nil {
					return nil, syncErr
				}
			}
			next, openErr := current.OpenRoot(component)
			if openErr != nil {
				return nil, fmt.Errorf("artifact.root_open: %w", openErr)
			}
			_ = current.Close()
			current = next
		}
	}
	info, err := current.Stat(".")
	if err != nil {
		return nil, fmt.Errorf("artifact.root_stat: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("artifact.root_invalid")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("artifact.root_permissions")
	}
	closeOnError = false
	return current, nil
}

func (store *Store) Close() error {
	if store == nil || store.root == nil {
		return nil
	}
	return store.root.Close()
}

func (store *Store) Put(ctx context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	if store == nil || store.root == nil {
		return ports.StoredArtifact{}, errors.New("artifact.store_unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ports.StoredArtifact{}, err
	}
	if strings.TrimSpace(request.MediaType) == "" {
		return ports.StoredArtifact{}, errors.New("artifact.media_type_required")
	}

	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef(artifactRefPrefix + digestText)
	if err != nil {
		return ports.StoredArtifact{}, fmt.Errorf("artifact.ref_create: %w", err)
	}
	stored := ports.StoredArtifact{
		Ref:       ref,
		Digest:    digestText,
		MediaType: request.MediaType,
		Size:      int64(len(request.Content)),
	}
	finalPath := blobPath(digestText)
	if err := store.ensureExistingOrWrite(ctx, finalPath, request.Content, digestText); err != nil {
		return ports.StoredArtifact{}, err
	}
	return stored, nil
}

func (store *Store) Get(ctx context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	if store == nil || store.root == nil {
		return ports.ArtifactContent{}, errors.New("artifact.store_unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ports.ArtifactContent{}, err
	}
	digest, err := digestFromRef(ref)
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	if expectedSize < 0 {
		return ports.ArtifactContent{}, errors.New("artifact.expected_size_invalid")
	}
	filePath := blobPath(digest)
	content, err := store.readVerifiedBlob(filePath, digest, expectedSize)
	if errors.Is(err, fs.ErrNotExist) {
		return ports.ArtifactContent{}, errors.New("artifact.not_found")
	}
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	return ports.ArtifactContent{
		Ref:     ref,
		Digest:  digest,
		Size:    int64(len(content)),
		Content: content,
	}, nil
}

func (store *Store) ensureExistingOrWrite(ctx context.Context, finalPath string, content []byte, digest string) error {
	directory := path.Dir(finalPath)
	if err := store.ensureDurableDirectory(directory); err != nil {
		return err
	}
	if _, err := store.readVerifiedBlob(finalPath, digest, int64(len(content))); err == nil {
		return store.syncDirectory(directory)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("artifact.existing_invalid: %w", err)
	}

	tempPath, err := store.tempPath(directory)
	if err != nil {
		return err
	}
	file, err := store.root.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("artifact.temp_create: %w", err)
	}
	removeTemp := true
	defer func() {
		_ = file.Close()
		if removeTemp {
			_ = store.root.Remove(tempPath)
		}
	}()

	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("artifact.temp_write: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("artifact.temp_sync: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("artifact.temp_close: %w", err)
	}

	if err := store.root.Link(tempPath, finalPath); err != nil {
		if _, readErr := store.readVerifiedBlob(finalPath, digest, int64(len(content))); readErr != nil {
			return fmt.Errorf("artifact.publish: %w", err)
		}
	}
	if err := store.root.Remove(tempPath); err != nil {
		return fmt.Errorf("artifact.temp_remove: %w", err)
	}
	removeTemp = false
	return store.syncDirectory(directory)
}

func (store *Store) readVerifiedBlob(filePath, digest string, expectedSize int64) ([]byte, error) {
	info, err := store.root.Lstat(filePath)
	if err != nil {
		return nil, fmt.Errorf("artifact.stat: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("artifact.file_invalid")
	}
	if info.Size() != expectedSize {
		return nil, errors.New("artifact.size_mismatch")
	}
	file, err := store.root.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("artifact.open: %w", err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || openedInfo.Size() != expectedSize || !os.SameFile(info, openedInfo) {
		return nil, errors.New("artifact.file_changed")
	}
	limit := expectedSize
	if limit < math.MaxInt64 {
		limit++
	}
	content, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, fmt.Errorf("artifact.read: %w", err)
	}
	if int64(len(content)) != expectedSize {
		return nil, errors.New("artifact.size_mismatch")
	}
	if err := verifyContent(content, digest); err != nil {
		return nil, err
	}
	currentInfo, err := store.root.Lstat(filePath)
	if err != nil || currentInfo.Mode()&os.ModeSymlink != 0 || !currentInfo.Mode().IsRegular() || !os.SameFile(openedInfo, currentInfo) {
		return nil, errors.New("artifact.file_changed")
	}
	return content, nil
}

func (store *Store) ensureDurableDirectory(directory string) error {
	cleaned := path.Clean(directory)
	if cleaned == "." {
		return nil
	}
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return errors.New("artifact.directory_invalid")
	}
	current := ""
	parent := "."
	for _, component := range strings.Split(cleaned, "/") {
		current = path.Join(current, component)
		if err := store.root.Mkdir(current, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("artifact.directory_create: %w", err)
		}
		info, err := store.root.Lstat(current)
		if err != nil {
			return fmt.Errorf("artifact.directory_stat: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("artifact.directory_invalid")
		}
		if err := store.syncDirectory(parent); err != nil {
			return err
		}
		parent = current
	}
	return nil
}

func (store *Store) syncDirectory(directory string) error {
	syncFn := store.syncDirectoryFn
	if syncFn == nil {
		syncFn = syncDirectory
	}
	return syncFn(store.root, directory)
}

func (store *Store) tempPath(directory string) (string, error) {
	var suffix [12]byte
	if _, err := io.ReadFull(rand.Reader, suffix[:]); err != nil {
		return "", fmt.Errorf("artifact.temp_random: %w", err)
	}
	return path.Join(directory, ".artifact-"+hex.EncodeToString(suffix[:])+".tmp"), nil
}

func syncDirectory(root *os.Root, directory string) error {
	handle, err := root.Open(directory)
	if err != nil {
		return fmt.Errorf("artifact.directory_open: %w", err)
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("artifact.directory_sync: %w", err)
	}
	return nil
}

func verifyContent(content []byte, expectedDigest string) error {
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != expectedDigest {
		return errors.New("artifact.digest_mismatch")
	}
	return nil
}

func digestFromRef(ref goal.ArtifactRef) (string, error) {
	value := ref.String()
	if !strings.HasPrefix(value, artifactRefPrefix) {
		return "", errors.New("artifact.ref_invalid")
	}
	digest := strings.TrimPrefix(value, artifactRefPrefix)
	if len(digest) != sha256.Size*2 {
		return "", errors.New("artifact.ref_invalid")
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return "", errors.New("artifact.ref_invalid")
	}
	return digest, nil
}

func blobPath(digest string) string {
	return path.Join("sha256", digest[:2], digest+".blob")
}

var _ application.ArtifactStore = (*Store)(nil)
