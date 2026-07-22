//go:build linux

package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"orquesta/internal/ports"
)

func openPrivateRoot(configuredPath string, syncFn func(*os.File) error) (_ *os.Root, resultErr error) {
	absolutePath, err := filepath.Abs(configuredPath)
	if err != nil {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	anchor := filepath.VolumeName(absolutePath) + string(filepath.Separator)
	relativePath, err := filepath.Rel(anchor, absolutePath)
	if err != nil || relativePath == "." || !filepath.IsLocal(relativePath) {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	current, err := os.OpenRoot(anchor)
	if err != nil {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	defer func() {
		if resultErr != nil {
			_ = current.Close()
		}
	}()
	anchorInfo, err := current.Stat(".")
	if err != nil {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	anchorStat, ownerOK := anchorInfo.Sys().(*syscall.Stat_t)
	if !ownerOK || !validSystemAncestor(anchorInfo, os.Geteuid(), anchorStat.Uid) {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	trustedSystemOwner := anchorStat.Uid
	components := strings.Split(relativePath, string(filepath.Separator))
	for index, component := range components {
		next, openErr := openRootComponent(current, component, index == len(components)-1, trustedSystemOwner, syncFn)
		if openErr != nil {
			return nil, openErr
		}
		_ = current.Close()
		current = next
	}
	return current, nil
}

func openRootComponent(parent *os.Root, component string, leaf bool, trustedSystemOwner uint32, syncFn func(*os.File) error) (*os.Root, error) {
	if component == "" || component == "." || component == ".." {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	before, err := parent.Lstat(component)
	created := false
	if errors.Is(err, fs.ErrNotExist) {
		if err := parent.Mkdir(component, 0o700); err != nil {
			return nil, artifactError(ports.ArtifactErrorIO, nil)
		}
		created = true
		before, err = parent.Lstat(component)
	}
	if err != nil || !validRootComponent(before, leaf || created, os.Geteuid(), trustedSystemOwner) {
		if before != nil && before.IsDir() {
			return nil, artifactError(ports.ArtifactErrorRootPermissions, nil)
		}
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	if created {
		if err := syncValidatedRootParent(parent, trustedSystemOwner, syncFn); err != nil {
			return nil, err
		}
	}
	private := leaf || created
	valid := func(info os.FileInfo) bool {
		return validRootComponent(info, private, os.Geteuid(), trustedSystemOwner)
	}
	next, err := parent.OpenRoot(component)
	if err != nil {
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	rooted, rootErr := next.Stat(".")
	current, currentErr := parent.Lstat(component)
	if rootErr != nil || currentErr != nil || !os.SameFile(before, rooted) || !os.SameFile(rooted, current) ||
		!valid(rooted) || !valid(current) {
		_ = next.Close()
		return nil, artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	return next, nil
}

func syncValidatedRootParent(parent *os.Root, trustedSystemOwner uint32, syncFn func(*os.File) error) error {
	before, err := parent.Lstat(".")
	if err != nil || !validSystemAncestor(before, os.Geteuid(), trustedSystemOwner) {
		return artifactError(ports.ArtifactErrorRootInvalid, nil)
	}
	valid := func(info os.FileInfo) bool { return validSystemAncestor(info, os.Geteuid(), trustedSystemOwner) }
	handle, err := openCheckedDirectory(parent, ".", before, valid, ports.ArtifactErrorRootInvalid)
	if err != nil {
		return err
	}
	defer handle.Close()
	if err := syncFn(handle); err != nil {
		return artifactError(ports.ArtifactErrorIO, err)
	}
	return verifyCheckedDirectory(parent, ".", handle, before, valid, ports.ArtifactErrorRootInvalid)
}

func validRootComponent(info os.FileInfo, private bool, euid int, trustedSystemOwner uint32) bool {
	return private && privateDirectoryValid(info, euid) ||
		!private && validSystemAncestor(info, euid, trustedSystemOwner)
}

func validSystemAncestor(info os.FileInfo, euid int, trustedSystemOwner uint32) bool {
	stat, ok := privateStat(info)
	if !ok || !info.IsDir() {
		return false
	}
	if stat.Uid == trustedSystemOwner {
		return info.Mode().Perm()&0o022 == 0 || info.Mode()&os.ModeSticky != 0
	}
	return stat.Uid == uint32(euid) && info.Mode().Perm()&0o022 == 0
}

func (store *Store) ensureDurableDirectory(directory string) error {
	cleaned := path.Clean(directory)
	if cleaned == "." {
		return nil
	}
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return artifactError(ports.ArtifactErrorDirectoryInvalid, nil)
	}
	current := ""
	parent := "."
	for _, component := range strings.Split(cleaned, "/") {
		current = path.Join(current, component)
		if err := store.root.Mkdir(current, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return artifactError(ports.ArtifactErrorIO, nil)
		}
		if err := validatePrivateDirectoryAt(store.root, current); err != nil {
			return err
		}
		if err := store.syncDirectory(parent); err != nil {
			return err
		}
		parent = current
	}
	return nil
}

func validatePrivateDirectoryAt(root *os.Root, name string) error {
	handle, before, err := openPrivateDirectory(root, name)
	if err != nil {
		return err
	}
	defer handle.Close()
	return verifyOpenPrivateDirectory(root, name, handle, before)
}

func openPrivateDirectory(root *os.Root, name string) (*os.File, os.FileInfo, error) {
	before, err := root.Lstat(name)
	if err != nil || !privateDirectoryValid(before, os.Geteuid()) {
		return nil, nil, artifactError(ports.ArtifactErrorDirectoryInvalid, nil)
	}
	handle, err := openCheckedDirectory(root, name, before, isPrivateDirectory, ports.ArtifactErrorDirectoryInvalid)
	if err != nil {
		return nil, nil, err
	}
	return handle, before, nil
}

func verifyOpenPrivateDirectory(root *os.Root, name string, handle *os.File, before os.FileInfo) error {
	return verifyCheckedDirectory(root, name, handle, before, isPrivateDirectory, ports.ArtifactErrorDirectoryInvalid)
}

func isPrivateDirectory(info os.FileInfo) bool {
	return privateDirectoryValid(info, os.Geteuid())
}

func openCheckedDirectory(root *os.Root, name string, before os.FileInfo, valid func(os.FileInfo) bool, errorCode string) (*os.File, error) {
	handle, err := root.Open(name)
	if err != nil {
		return nil, artifactError(errorCode, nil)
	}
	opened, err := handle.Stat()
	if err == nil && valid(opened) && os.SameFile(before, opened) {
		return handle, nil
	}
	_ = handle.Close()
	return nil, artifactError(errorCode, nil)
}

func verifyCheckedDirectory(root *os.Root, name string, handle *os.File, before os.FileInfo, valid func(os.FileInfo) bool, errorCode string) error {
	opened, openErr := handle.Stat()
	current, currentErr := root.Lstat(name)
	if openErr != nil || currentErr != nil || !valid(opened) || !valid(current) || !os.SameFile(before, opened) || !os.SameFile(opened, current) {
		return artifactError(errorCode, nil)
	}
	return nil
}

func validateOpenPrivateFile(root *os.Root, name string, file *os.File, size int64) error {
	before, err := root.Lstat(name)
	if err != nil || !privateFileValid(before, size, os.Geteuid()) {
		return artifactError(ports.ArtifactErrorFileInvalid, nil)
	}
	if !privateFileIsCurrent(root, name, file, before, size) {
		return artifactError(ports.ArtifactErrorFileChanged, nil)
	}
	return nil
}

func (store *Store) readVerifiedBlob(filePath, digest string, expectedSize int64) ([]byte, error) {
	before, err := store.root.Lstat(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		return nil, artifactError(ports.ArtifactErrorIO, nil)
	}
	if !privateFileValid(before, expectedSize, os.Geteuid()) {
		return nil, artifactError(ports.ArtifactErrorFileInvalid, nil)
	}
	file, err := store.root.OpenFile(filePath, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, artifactError(ports.ArtifactErrorFileChanged, nil)
	}
	defer file.Close()
	if !privateFileIsCurrent(store.root, filePath, file, before, expectedSize) {
		return nil, artifactError(ports.ArtifactErrorFileChanged, nil)
	}
	limit := expectedSize
	if limit < math.MaxInt64 {
		limit++
	}
	content, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, artifactError(ports.ArtifactErrorIO, nil)
	}
	if int64(len(content)) != expectedSize {
		return nil, artifactError(ports.ArtifactErrorSizeMismatch, nil)
	}
	contentDigest := sha256.Sum256(content)
	if hex.EncodeToString(contentDigest[:]) != digest {
		return nil, artifactError(ports.ArtifactErrorDigestMismatch, nil)
	}
	if store.readVerifyHook != nil {
		store.readVerifyHook()
	}
	if !privateFileIsCurrent(store.root, filePath, file, before, expectedSize) {
		return nil, artifactError(ports.ArtifactErrorFileChanged, nil)
	}
	return content, nil
}

func privateFileIsCurrent(root *os.Root, name string, file *os.File, before os.FileInfo, size int64) bool {
	opened, openErr := file.Stat()
	current, currentErr := root.Lstat(name)
	return openErr == nil && currentErr == nil && privateFileValid(opened, size, os.Geteuid()) &&
		privateFileValid(current, size, os.Geteuid()) && samePrivateMetadata(before, opened) &&
		samePrivateMetadata(opened, current)
}

func (store *Store) publishNoReplace(temporary, final string) (bool, error) {
	directory := path.Dir(final)
	if path.Dir(temporary) != directory || path.Base(temporary) == "." || path.Base(final) == "." {
		return false, artifactError(ports.ArtifactErrorFileInvalid, nil)
	}
	handle, before, err := openPrivateDirectory(store.root, directory)
	if err != nil {
		return false, err
	}
	defer handle.Close()
	if store.beforePublishHook != nil {
		store.beforePublishHook(temporary)
	}
	err = unix.Renameat2(int(handle.Fd()), path.Base(temporary), int(handle.Fd()), path.Base(final), unix.RENAME_NOREPLACE)
	if errors.Is(err, unix.EEXIST) {
		return false, nil
	}
	if err != nil {
		return false, artifactError(ports.ArtifactErrorIO, nil)
	}
	if err := verifyOpenPrivateDirectory(store.root, directory, handle, before); err != nil {
		return false, err
	}
	return true, nil
}
