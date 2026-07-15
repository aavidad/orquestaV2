//go:build linux

package sqlite

import (
	"errors"
	"os"
	"sort"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

type lockedRecoveryRoot struct {
	path   string
	file   *os.File
	root   *os.Root
	device uint64
	inode  uint64
	uid    uint32
	mode   uint32
}

type recoveryRootLocks struct {
	roots map[string]*lockedRecoveryRoot
}

func recoveryRootLockSupportError() error { return nil }

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
	root, err := os.OpenRoot("/proc/self/fd/" + strconv.Itoa(descriptor))
	if err != nil {
		return nil, errors.New("sqlite.recovery_root_handle_unavailable")
	}
	rootInfo, err := root.Stat(".")
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok || uint64(rootStat.Dev) != uint64(opened.Dev) || rootStat.Ino != opened.Ino ||
		uint32(rootStat.Uid) != opened.Uid || uint32(rootStat.Mode) != opened.Mode {
		_ = root.Close()
		return nil, errors.New("sqlite.recovery_root_handle_invalid")
	}
	closeDescriptor = false
	return &lockedRecoveryRoot{
		path: path, file: os.NewFile(uintptr(descriptor), path), root: root, device: uint64(opened.Dev),
		inode: opened.Ino, uid: opened.Uid, mode: opened.Mode,
	}, nil
}

func (locks *recoveryRootLocks) verify(paths ...string) error {
	if locks == nil {
		return errors.New("sqlite.recovery_root_lock_closed")
	}
	for _, path := range paths {
		root := locks.roots[path]
		if root == nil || root.file == nil || root.root == nil {
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
	rootInfo, err := root.root.Stat(".")
	if err != nil {
		return errors.New("sqlite.recovery_locked_root_changed")
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok || uint64(rootStat.Dev) != root.device || rootStat.Ino != root.inode ||
		uint32(rootStat.Uid) != root.uid || uint32(rootStat.Mode) != root.mode {
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

// openedRoot returns the persistent os.Root bound to the locked inode. It does
// not resolve rootPath again, so a same-UID pathname replacement cannot redirect
// later filesystem operations.
func (locks *recoveryRootLocks) openedRoot(rootPath string) (*os.Root, error) {
	if locks == nil {
		return nil, errors.New("sqlite.recovery_root_lock_closed")
	}
	root := locks.roots[rootPath]
	if root == nil || root.file == nil || root.root == nil {
		return nil, errors.New("sqlite.recovery_root_lock_missing")
	}
	return root.root, nil
}

func (locks *recoveryRootLocks) sync(path string) error {
	if err := locks.verify(path); err != nil {
		return err
	}
	if err := locks.roots[path].file.Sync(); err != nil {
		return err
	}
	return locks.verify(path)
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
		if root.root != nil {
			if err := root.root.Close(); err != nil && result == nil {
				result = err
			}
			root.root = nil
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

func recoveryDescriptorPath(file *os.File) (string, error) {
	if file == nil {
		return "", errors.New("sqlite.recovery_file_handle_required")
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	path := "/proc/self/fd/" + strconv.FormatUint(uint64(file.Fd()), 10)
	resolved, err := os.Stat(path)
	if err != nil || !os.SameFile(info, resolved) {
		return "", errors.New("sqlite.recovery_file_handle_unavailable")
	}
	return path, nil
}
