//go:build linux

package sqlite

import (
	"errors"
	"os"
	"sort"
	"syscall"

	"golang.org/x/sys/unix"
)

type lockedRecoveryRoot struct {
	path   string
	file   *os.File
	device uint64
	inode  uint64
	uid    uint32
	mode   uint32
}

type recoveryRootLocks struct {
	roots map[string]*lockedRecoveryRoot
}

func acquireRecoveryRootLocks(paths ...string) (*recoveryRootLocks, error) {
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	locks := &recoveryRootLocks{roots: make(map[string]*lockedRecoveryRoot, len(ordered))}
	for _, path := range ordered {
		root, err := lockPrivateRecoveryRoot(path)
		if err != nil {
			_ = locks.Close()
			return nil, err
		}
		locks.roots[path] = root
	}
	return locks, nil
}

func lockPrivateRecoveryRoot(path string) (*lockedRecoveryRoot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	closeDescriptor := true
	defer func() {
		if closeDescriptor {
			_ = unix.Close(descriptor)
		}
	}()
	var opened unix.Stat_t
	if err := unix.Fstat(descriptor, &opened); err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Dev) != uint64(opened.Dev) || stat.Ino != opened.Ino ||
		opened.Uid != uint32(os.Geteuid()) || opened.Mode&unix.S_IFMT != unix.S_IFDIR || opened.Mode&0o7777 != 0o700 {
		return nil, errors.New("sqlite.recovery_lock_root_invalid")
	}
	if err := unix.Flock(descriptor, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, err
	}
	closeDescriptor = false
	return &lockedRecoveryRoot{
		path: path, file: os.NewFile(uintptr(descriptor), path), device: uint64(opened.Dev),
		inode: opened.Ino, uid: opened.Uid, mode: opened.Mode,
	}, nil
}

func (locks *recoveryRootLocks) verify(paths ...string) error {
	if locks == nil {
		return errors.New("sqlite.recovery_root_lock_closed")
	}
	for _, path := range paths {
		root := locks.roots[path]
		if root == nil || root.file == nil {
			return errors.New("sqlite.recovery_root_lock_missing")
		}
		if err := root.verify(); err != nil {
			return err
		}
	}
	return nil
}

func (root *lockedRecoveryRoot) verify() error {
	var opened unix.Stat_t
	if err := unix.Fstat(int(root.file.Fd()), &opened); err != nil {
		return err
	}
	if uint64(opened.Dev) != root.device || opened.Ino != root.inode || opened.Uid != root.uid ||
		opened.Mode != root.mode {
		return errors.New("sqlite.recovery_locked_root_changed")
	}
	info, err := os.Lstat(root.path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("sqlite.recovery_root_path_replaced")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Dev) != root.device || stat.Ino != root.inode || uint32(stat.Uid) != root.uid ||
		uint32(stat.Mode) != root.mode {
		return errors.New("sqlite.recovery_root_path_replaced")
	}
	return nil
}

func (locks *recoveryRootLocks) sync(path string) error {
	if err := locks.verify(path); err != nil {
		return err
	}
	return locks.roots[path].file.Sync()
}

func (locks *recoveryRootLocks) Close() error {
	if locks == nil {
		return nil
	}
	var result error
	for path, root := range locks.roots {
		if root == nil || root.file == nil {
			continue
		}
		if err := unix.Flock(int(root.file.Fd()), unix.LOCK_UN); err != nil && result == nil {
			result = err
		}
		if err := root.file.Close(); err != nil && result == nil {
			result = err
		}
		root.file = nil
		delete(locks.roots, path)
	}
	return result
}
