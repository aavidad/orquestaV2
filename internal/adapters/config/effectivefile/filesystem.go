package effectivefile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const lockRetryDelay = 5 * time.Millisecond

type stableDirectory struct {
	path   string
	handle *os.File
	root   *os.Root
}

func openStableDirectory(path string) (*stableDirectory, error) {
	before, err := os.Lstat(path)
	if err != nil || !validPrivateDirectory(before) {
		return nil, fmt.Errorf("%w: open directory identity", ErrUnsafeFilesystem)
	}
	handle, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: open directory handle", ErrUnsafeFilesystem)
	}
	closeHandle := true
	defer func() {
		if closeHandle {
			_ = handle.Close()
		}
	}()
	opened, err := handle.Stat()
	if err != nil || !validPrivateDirectory(opened) || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%w: directory changed while opening", ErrUnsafeFilesystem)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("%w: open rooted directory", ErrUnsafeFilesystem)
	}
	rooted, err := root.Stat(".")
	if err != nil || !validPrivateDirectory(rooted) || !os.SameFile(opened, rooted) {
		_ = root.Close()
		return nil, fmt.Errorf("%w: rooted directory changed", ErrUnsafeFilesystem)
	}
	closeHandle = false
	return &stableDirectory{path: path, handle: handle, root: root}, nil
}

func (directory *stableDirectory) Close() {
	if directory == nil {
		return
	}
	_ = directory.root.Close()
	_ = directory.handle.Close()
}

func (directory *stableDirectory) ensureStillNamed() error {
	current, err := os.Lstat(directory.path)
	opened, openedErr := directory.handle.Stat()
	if err != nil || openedErr != nil || !validPrivateDirectory(current) || !validPrivateDirectory(opened) ||
		!os.SameFile(current, opened) {
		return fmt.Errorf("%w: directory identity changed", ErrUnsafeFilesystem)
	}
	return nil
}

func (directory *stableDirectory) sync() error {
	if err := directory.handle.Sync(); err != nil {
		return fmt.Errorf("%w: sync directory: %v", ErrUnsafeFilesystem, err)
	}
	return nil
}

func (directory *stableDirectory) withLock(ctx context.Context, name string, operation func() error) error {
	descriptor, err := syscall.Openat(int(directory.handle.Fd()), name,
		syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("%w: open lock: %v", ErrUnsafeFilesystem, err)
	}
	lock := os.NewFile(uintptr(descriptor), filepath.Join(directory.path, name))
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil {
		return fmt.Errorf("%w: inspect lock", ErrUnsafeFilesystem)
	}
	identity, identityErr := identityFromInfo(info)
	if identityErr != nil || !info.Mode().IsRegular() || info.Mode() != 0o600 || info.Size() != 0 ||
		identity.uid != uint32(os.Geteuid()) || identity.links != 1 {
		return fmt.Errorf("%w: invalid lock", ErrUnsafeFilesystem)
	}
	for {
		err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return fmt.Errorf("%w: lock: %v", ErrUnsafeFilesystem, err)
		}
		timer := time.NewTimer(lockRetryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if err := directory.ensureStillNamed(); err != nil {
		return err
	}
	return operation()
}

func (directory *stableDirectory) createTemporary() (*os.File, string, error) {
	for attempt := 0; attempt < 32; attempt++ {
		random := make([]byte, 16)
		if _, err := rand.Read(random); err != nil {
			return nil, "", fmt.Errorf("%w: random temporary name", ErrUnsafeFilesystem)
		}
		name := ".effective-config-" + hex.EncodeToString(random)
		file, err := directory.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", fmt.Errorf("%w: create temporary: %v", ErrUnsafeFilesystem, err)
		}
	}
	return nil, "", fmt.Errorf("%w: temporary name exhausted", ErrUnsafeFilesystem)
}

func validPrivateDirectory(info os.FileInfo) bool {
	if info == nil || info.Mode() != os.ModeDir|0o700 {
		return false
	}
	identity, err := identityFromInfo(info)
	return err == nil && identity.uid == uint32(os.Geteuid())
}
