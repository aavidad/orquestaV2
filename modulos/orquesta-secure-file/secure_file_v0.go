//go:build linux

package orquestasecurefile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	linuxOTmpfileV0        = 0x410000
	linuxATEmptyPathV0     = 0x1000
	linuxATSymlinkFollowV0 = 0x400
	linuxATFDCWDV0         = -100
)

func OpenDirectoryV0(path string, options DirectoryOptionsV0) (*os.File, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.VolumeName(path) != "" {
		return nil, ErrInvalidPathV0
	}
	createMode, err := validateDirectoryOptionsV0(options)
	if err != nil {
		return nil, err
	}
	fd, err := syscall.Open(string(filepath.Separator), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, ErrUnsafeDirectoryV0
	}
	var rootStat syscall.Stat_t
	if err := syscall.Fstat(fd, &rootStat); err != nil || !secureDirectoryStatV0(rootStat, path == string(filepath.Separator), options.FinalMode) {
		_ = syscall.Close(fd)
		return nil, ErrUnsafeDirectoryV0
	}
	current := string(filepath.Separator)
	parts := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	for index, part := range parts {
		if part == "" {
			continue
		}
		next, openErr := syscall.Openat(fd, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if errors.Is(openErr, syscall.ENOENT) && options.Create {
			if mkdirErr := syscall.Mkdirat(fd, part, createMode); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = syscall.Close(fd)
				return nil, ErrUnsafeDirectoryV0
			}
			if syncErr := syscall.Fsync(fd); syncErr != nil {
				_ = syscall.Close(fd)
				return nil, ErrUnsafeDirectoryV0
			}
			next, openErr = syscall.Openat(fd, part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		}
		if openErr != nil {
			_ = syscall.Close(fd)
			if errors.Is(openErr, syscall.ENOENT) {
				return nil, os.ErrNotExist
			}
			return nil, ErrUnsafeDirectoryV0
		}
		var stat syscall.Stat_t
		final := index == len(parts)-1
		if err := syscall.Fstat(next, &stat); err != nil || !secureDirectoryStatV0(stat, final, options.FinalMode) {
			_ = syscall.Close(next)
			_ = syscall.Close(fd)
			return nil, ErrUnsafeDirectoryV0
		}
		_ = syscall.Close(fd)
		fd = next
		current = filepath.Join(current, part)
	}
	return os.NewFile(uintptr(fd), current), nil
}

func validateDirectoryOptionsV0(options DirectoryOptionsV0) (uint32, error) {
	if options.CreateMode&^uint32(0o777) != 0 || options.FinalMode&^uint32(0o777) != 0 {
		return 0, ErrInvalidPathV0
	}
	createMode := options.CreateMode
	if createMode == 0 {
		createMode = 0o700
	}
	if createMode&0o022 != 0 || options.FinalMode&0o022 != 0 {
		return 0, ErrInvalidPathV0
	}
	if options.Create && options.FinalMode != 0 && createMode != options.FinalMode {
		return 0, ErrInvalidPathV0
	}
	return createMode, nil
}

func secureDirectoryStatV0(stat syscall.Stat_t, final bool, finalMode uint32) bool {
	if stat.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		return false
	}
	uid := stat.Uid
	if uid != 0 && uid != uint32(os.Geteuid()) {
		return false
	}
	mode := stat.Mode & 0o777
	special := stat.Mode & secureSpecialModeBitsV0()
	rootStickyAncestor := !final && uid == 0 && special == uint32(syscall.S_ISVTX)
	if special != 0 && !rootStickyAncestor {
		return false
	}
	if mode&0o022 != 0 && !rootStickyAncestor {
		return false
	}
	if final && finalMode != 0 && (uid != uint32(os.Geteuid()) || mode != finalMode&0o777) {
		return false
	}
	return true
}

func ReadFileAtV0(dir *os.File, name string, options FileOptionsV0) ([]byte, error) {
	if dir == nil || !validBaseNameV0(name) {
		return nil, ErrInvalidPathV0
	}
	if err := validateFileOptionsV0(options, false); err != nil {
		return nil, err
	}
	dirFD, err := secureDirectoryFDV0(dir)
	if err != nil {
		return nil, err
	}
	fd, err := syscall.Openat(dirFD, name, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || !secureFileStatV0(stat, options) {
		_ = syscall.Close(fd)
		return nil, ErrUnsafeFileV0
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, ErrUnsafeFileV0
	}
	data, readErr := io.ReadAll(io.LimitReader(file, options.MaxBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > options.MaxBytes {
		return nil, ErrUnsafeFileV0
	}
	return data, nil
}

func secureFileStatV0(stat syscall.Stat_t, options FileOptionsV0) bool {
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Nlink != 1 || stat.Uid != uint32(os.Geteuid()) || stat.Size < 0 || stat.Size > options.MaxBytes {
		return false
	}
	mode := stat.Mode & (uint32(0o777) | secureSpecialModeBitsV0())
	if mode&secureSpecialModeBitsV0() != 0 {
		return false
	}
	if options.ExactMode != 0 {
		return mode == options.ExactMode
	}
	return mode&0o022 == 0
}

func ReadFileBeneathV0(rootDir, rel string, options FileOptionsV0) ([]byte, error) {
	if err := validateFileOptionsV0(options, false); err != nil {
		return nil, err
	}
	if strings.TrimSpace(rootDir) != rootDir || strings.TrimSpace(rel) != rel {
		return nil, ErrInvalidPathV0
	}
	if rootDir == "" || rootDir == "." || !filepath.IsAbs(rootDir) || filepath.Clean(rootDir) != rootDir || rel == "" || rel == "." || filepath.Clean(rel) != rel || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, ErrInvalidPathV0
	}
	dir, err := OpenDirectoryV0(rootDir, DirectoryOptionsV0{})
	if err != nil {
		return nil, err
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts[:len(parts)-1] {
		if !validBaseNameV0(part) {
			dir.Close()
			return nil, ErrInvalidPathV0
		}
		nextFD, openErr := syscall.Openat(int(dir.Fd()), part, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if openErr != nil {
			dir.Close()
			return nil, openErr
		}
		var stat syscall.Stat_t
		if err := syscall.Fstat(nextFD, &stat); err != nil || !secureDirectoryStatV0(stat, false, 0) {
			_ = syscall.Close(nextFD)
			dir.Close()
			return nil, ErrUnsafeDirectoryV0
		}
		dir.Close()
		dir = os.NewFile(uintptr(nextFD), part)
		if dir == nil {
			_ = syscall.Close(nextFD)
			return nil, ErrUnsafeDirectoryV0
		}
	}
	defer dir.Close()
	return ReadFileAtV0(dir, parts[len(parts)-1], options)
}

// CreateFileIfAbsentAtV0 publishes an anonymous inode with linkat: concurrent
// creators never expose partial bytes or leave named temporary files.
func CreateFileIfAbsentAtV0(dir *os.File, name string, data []byte, options FileOptionsV0) ([]byte, bool, error) {
	return createFileIfAbsentAtV0(dir, name, data, options, nil)
}

func createFileIfAbsentAtV0(dir *os.File, name string, data []byte, options FileOptionsV0, beforePublish func() error) ([]byte, bool, error) {
	if dir == nil || !validBaseNameV0(name) || len(data) == 0 {
		return nil, false, ErrInvalidPathV0
	}
	if err := validateFileOptionsV0(options, true); err != nil {
		return nil, false, err
	}
	if int64(len(data)) > options.MaxBytes {
		return nil, false, ErrInvalidPathV0
	}
	dirFD, err := secureDirectoryFDV0(dir)
	if err != nil {
		return nil, false, err
	}
	fd, err := syscall.Openat(dirFD, ".", syscall.O_WRONLY|syscall.O_CLOEXEC|linuxOTmpfileV0, options.ExactMode&0o777)
	if err != nil {
		if anonymousFileUnsupportedV0(err) {
			return nil, false, ErrAnonymousFileUnsupportedV0
		}
		return nil, false, ErrUnsafeFileV0
	}
	file := os.NewFile(uintptr(fd), "anonymous-secure-file")
	if file == nil {
		_ = syscall.Close(fd)
		return nil, false, ErrUnsafeFileV0
	}
	open := true
	defer func() {
		if open {
			_ = file.Close()
		}
	}()
	if err := syscall.Fchmod(fd, options.ExactMode&0o777); err != nil {
		return nil, false, ErrUnsafeFileV0
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || !secureAnonymousFileStatV0(stat, options, 0) {
		return nil, false, ErrUnsafeFileV0
	}
	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if writeErr != nil {
		return nil, false, ErrUnsafeFileV0
	}
	if err := syscall.Fstat(fd, &stat); err != nil || !secureAnonymousFileStatV0(stat, options, int64(len(data))) {
		return nil, false, ErrUnsafeFileV0
	}
	if beforePublish != nil {
		if err := beforePublish(); err != nil {
			return nil, false, err
		}
	}
	created := true
	if err := linkAnonymousFileAtV0(fd, dirFD, name); err != nil {
		if !errors.Is(err, syscall.EEXIST) {
			if anonymousFileUnsupportedV0(err) || errors.Is(err, ErrAnonymousFileUnsupportedV0) {
				return nil, false, ErrAnonymousFileUnsupportedV0
			}
			return nil, false, ErrUnsafeFileV0
		}
		created = false
	}
	if err := file.Close(); err != nil {
		return nil, false, ErrUnsafeFileV0
	}
	open = false
	if created {
		if err := dir.Sync(); err != nil {
			return nil, false, ErrUnsafeFileV0
		}
	}
	stored, err := ReadFileAtV0(dir, name, options)
	if err != nil {
		return nil, false, err
	}
	return stored, created, nil
}

func secureAnonymousFileStatV0(stat syscall.Stat_t, options FileOptionsV0, size int64) bool {
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Nlink != 0 || stat.Uid != uint32(os.Geteuid()) || stat.Size != size || stat.Size < 0 || stat.Size > options.MaxBytes {
		return false
	}
	mode := stat.Mode & (uint32(0o777) | secureSpecialModeBitsV0())
	return mode == options.ExactMode
}

func validateFileOptionsV0(options FileOptionsV0, requireExact bool) error {
	if options.MaxBytes <= 0 || options.ExactMode&^uint32(0o777) != 0 || options.ExactMode&0o022 != 0 || requireExact && options.ExactMode == 0 {
		return ErrInvalidPathV0
	}
	return nil
}

func secureSpecialModeBitsV0() uint32 {
	return uint32(syscall.S_ISUID | syscall.S_ISGID | syscall.S_ISVTX)
}

func anonymousFileUnsupportedV0(err error) bool {
	return errors.Is(err, syscall.EOPNOTSUPP) || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EISDIR) || errors.Is(err, syscall.ENOSYS)
}

func linkAnonymousFileAtV0(fileFD, dirFD int, name string) error {
	err := linkAnonymousFileEmptyPathAtV0(fileFD, dirFD, name)
	if !errors.Is(err, syscall.EPERM) {
		return err
	}
	// AT_EMPTY_PATH may require CAP_DAC_READ_SEARCH on older kernels. /proc/self/fd
	// publishes the same anonymous inode without introducing a named temporary.
	err = linkAnonymousFileProcFDAtV0(fileFD, dirFD, name)
	if err != nil && !errors.Is(err, syscall.EEXIST) {
		return ErrAnonymousFileUnsupportedV0
	}
	return err
}

func linkAnonymousFileEmptyPathAtV0(fileFD, dirFD int, name string) error {
	emptyPtr, err := syscall.BytePtrFromString("")
	if err != nil {
		return err
	}
	namePtr, err := syscall.BytePtrFromString(name)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_LINKAT,
		uintptr(fileFD), uintptr(unsafe.Pointer(emptyPtr)),
		uintptr(dirFD), uintptr(unsafe.Pointer(namePtr)),
		linuxATEmptyPathV0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func linkAnonymousFileProcFDAtV0(fileFD, dirFD int, name string) error {
	oldPtr, err := syscall.BytePtrFromString("/proc/self/fd/" + strconv.Itoa(fileFD))
	if err != nil {
		return err
	}
	namePtr, err := syscall.BytePtrFromString(name)
	if err != nil {
		return err
	}
	atFDCWD := linuxATFDCWDV0
	_, _, errno := syscall.Syscall6(
		syscall.SYS_LINKAT,
		uintptr(atFDCWD), uintptr(unsafe.Pointer(oldPtr)),
		uintptr(dirFD), uintptr(unsafe.Pointer(namePtr)),
		linuxATSymlinkFollowV0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func secureDirectoryFDV0(dir *os.File) (int, error) {
	fd := int(dir.Fd())
	if fd < 0 {
		return -1, ErrUnsafeDirectoryV0
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || !secureDirectoryStatV0(stat, true, 0) {
		return -1, ErrUnsafeDirectoryV0
	}
	return fd, nil
}

func validBaseNameV0(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, `/\\`)
}
