// Package effectivefile persists an already-redacted effective configuration
// document. It owns filesystem policy but knows nothing about config values or
// secret resolution.
package effectivefile

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	documentType           = "orquesta.effective_config"
	afterTempSyncFailpoint = "after_temp_sync"
)

var (
	ErrInvalidOptions          = errors.New("effectivefile.invalid_options")
	ErrUnsafeFilesystem        = errors.New("effectivefile.unsafe_filesystem")
	ErrDestinationUnrecognized = errors.New("effectivefile.destination_unrecognized")
	ErrDocumentInvalid         = errors.New("effectivefile.document_invalid")
)

// Options contains one atomic replace request. Content must already be
// redacted and carry the Orquesta effective-config ownership marker.
type Options struct {
	Path             string
	Content          []byte
	MaxExistingBytes int64
	Failpoint        func(string) error
}

// Write validates ownership and atomically replaces Path. The durability
// order is temp write+sync, optional after_temp_sync failpoint, chmod 0400,
// second sync, rename and parent-directory fsync.
func Write(ctx context.Context, options Options) error {
	if ctx == nil || strings.TrimSpace(options.Path) == "" || options.Path != strings.TrimSpace(options.Path) || options.MaxExistingBytes <= 0 ||
		len(options.Content) == 0 || int64(len(options.Content)) > options.MaxExistingBytes {
		return ErrInvalidOptions
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateDocument(options.Content); err != nil {
		return err
	}
	cleanPath := filepath.Clean(options.Path)
	if filepath.Base(cleanPath) == "." || filepath.Base(cleanPath) == string(filepath.Separator) {
		return ErrInvalidOptions
	}
	directoryPath, err := ensurePrivateDirectory(ctx, filepath.Dir(cleanPath))
	if err != nil {
		return err
	}
	directory, err := openStableDirectory(directoryPath)
	if err != nil {
		return err
	}
	defer directory.Close()
	targetName := filepath.Base(cleanPath)
	return directory.withLock(ctx, targetName+".lock", func() error {
		return writeLocked(ctx, directory, targetName, options)
	})
}

func writeLocked(ctx context.Context, directory *stableDirectory, targetName string, options Options) error {
	if err := validateDestination(ctx, directory.root, targetName, options.MaxExistingBytes); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	temporary, temporaryName, err := directory.createTemporary()
	if err != nil {
		return err
	}
	renamed := false
	defer func() {
		_ = temporary.Close()
		if !renamed {
			_ = directory.root.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(options.Content); err != nil {
		return fmt.Errorf("%w: write temporary: %v", ErrUnsafeFilesystem, err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("%w: sync temporary: %v", ErrUnsafeFilesystem, err)
	}
	if options.Failpoint != nil {
		if err := options.Failpoint(afterTempSyncFailpoint); err != nil {
			return fmt.Errorf("effectivefile.failpoint.%s: %w", afterTempSyncFailpoint, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := temporary.Chmod(0o400); err != nil {
		return fmt.Errorf("%w: chmod temporary: %v", ErrUnsafeFilesystem, err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("%w: sync permissions: %v", ErrUnsafeFilesystem, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("%w: close temporary: %v", ErrUnsafeFilesystem, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := directory.ensureStillNamed(); err != nil {
		return err
	}
	if err := directory.root.Rename(temporaryName, targetName); err != nil {
		return fmt.Errorf("%w: rename: %v", ErrUnsafeFilesystem, err)
	}
	renamed = true
	return directory.sync()
}

// ReservedPaths reports adapter-owned sidecars next to one effective output.
func ReservedPaths(path string) []string {
	if strings.TrimSpace(path) == "" {
		return []string{}
	}
	return []string{path + ".lock"}
}

type effectiveEnvelope struct {
	DocumentType     string            `json:"document_type"`
	SchemaVersion    int               `json:"schema_version"`
	RegistryRevision string            `json:"registry_revision"`
	SnapshotHash     string            `json:"snapshot_hash"`
	Entries          []json.RawMessage `json:"entries"`
}

func validateDocument(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var document effectiveEnvelope
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("%w: decode", ErrDocumentInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing data", ErrDocumentInvalid)
	}
	encodedHash := strings.TrimPrefix(document.SnapshotHash, "sha256:")
	if document.DocumentType != documentType || document.SchemaVersion <= 0 ||
		strings.TrimSpace(document.RegistryRevision) == "" || len(encodedHash) != 64 || len(document.Entries) == 0 {
		return fmt.Errorf("%w: ownership marker", ErrDocumentInvalid)
	}
	if _, err := hex.DecodeString(encodedHash); err != nil {
		return fmt.Errorf("%w: snapshot hash", ErrDocumentInvalid)
	}
	return nil
}

func ensurePrivateDirectory(ctx context.Context, raw string) (string, error) {
	directory, err := filepath.Abs(filepath.Clean(raw))
	if err != nil {
		return "", fmt.Errorf("%w: directory path", ErrUnsafeFilesystem)
	}
	if err := rejectSymlinkComponents(ctx, directory); err != nil {
		return "", err
	}
	missing := make([]string, 0)
	for current := directory; ; current = filepath.Dir(current) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		info, statErr := os.Lstat(current)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return "", fmt.Errorf("%w: directory component", ErrUnsafeFilesystem)
			}
			break
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("%w: inspect directory: %v", ErrUnsafeFilesystem, statErr)
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("%w: directory root", ErrUnsafeFilesystem)
		}
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		current := missing[index]
		if err := os.Mkdir(current, 0o700); err != nil {
			return "", fmt.Errorf("%w: create directory: %v", ErrUnsafeFilesystem, err)
		}
		if err := syncDirectory(filepath.Dir(current)); err != nil {
			return "", fmt.Errorf("%w: sync directory creation: %v", ErrUnsafeFilesystem, err)
		}
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return "", fmt.Errorf("%w: inspect target directory: %v", ErrUnsafeFilesystem, err)
	}
	identity, err := identityFromInfo(info)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		!info.IsDir() || info.Mode().Perm() != 0o700 ||
		identity.uid != uint32(os.Geteuid()) {
		return "", fmt.Errorf("%w: target directory permissions mode=%o uid=%d expected_uid=%d identity_error=%v",
			ErrUnsafeFilesystem, info.Mode().Perm(), identity.uid, os.Geteuid(), err)
	}
	return directory, nil
}

func rejectSymlinkComponents(ctx context.Context, absolute string) error {
	volume := filepath.VolumeName(absolute)
	current := volume + string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, current)
	for _, component := range strings.Split(remainder, string(filepath.Separator)) {
		if component == "" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink component", ErrUnsafeFilesystem)
		}
	}
	return nil
}

func validateDestination(ctx context.Context, root *os.Root, name string, maxBytes int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	before, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || before.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: inspect destination", ErrUnsafeFilesystem)
	}
	if err := validateDestinationIdentity(before, maxBytes); err != nil {
		return err
	}
	opened, err := root.Open(name)
	if err != nil {
		return fmt.Errorf("%w: open destination", ErrUnsafeFilesystem)
	}
	defer opened.Close()
	after, err := opened.Stat()
	if err != nil || !os.SameFile(before, after) {
		return fmt.Errorf("%w: destination changed", ErrUnsafeFilesystem)
	}
	if err := validateDestinationIdentity(after, maxBytes); err != nil {
		return err
	}
	content, err := io.ReadAll(io.LimitReader(opened, maxBytes))
	if err != nil || int64(len(content)) != after.Size() {
		return fmt.Errorf("%w: read destination", ErrUnsafeFilesystem)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateDocument(content); err != nil {
		return fmt.Errorf("%w: ownership marker", ErrDestinationUnrecognized)
	}
	return nil
}

func validateDestinationIdentity(info os.FileInfo, maxBytes int64) error {
	identity, err := identityFromInfo(info)
	if err != nil {
		return ErrDestinationUnrecognized
	}
	return validateDestinationProperties(info.Mode(), info.Size(), identity, maxBytes)
}

func validateDestinationProperties(mode os.FileMode, size int64, identity fileIdentity, maxBytes int64) error {
	if !mode.IsRegular() || mode.Perm() != 0o400 || mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		identity.uid != uint32(os.Geteuid()) ||
		identity.links != 1 || size <= 0 || size > maxBytes {
		return ErrDestinationUnrecognized
	}
	return nil
}

type fileIdentity struct {
	uid   uint32
	links uint64
}

func identityFromInfo(info os.FileInfo) (fileIdentity, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return fileIdentity{}, errors.New("effectivefile.stat_identity_unavailable")
	}
	return fileIdentity{uid: stat.Uid, links: uint64(stat.Nlink)}, nil
}

func syncDirectory(directory string) error {
	opened, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer opened.Close()
	return opened.Sync()
}
