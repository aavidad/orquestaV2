package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var errUnsafeFile = errors.New("credentials.local.unsafe_file")

type fileSystem struct {
	path, name, next string
	directory        *os.File
	root             *os.Root
	owner            uint32
	maximum          int64
	syncFile         func(*os.File) error
	gate             chan struct{}
}

type readableFile interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

// ReservedPaths reports the fixed transactional namespace adjacent to path.
// Composition roots use it to keep unrelated runtime surfaces away from the
// recovery document owned by this adapter.
func ReservedPaths(source string) []string {
	if strings.TrimSpace(source) == "" {
		return []string{}
	}
	return []string{filepath.Clean(source) + ".next"}
}

func openFileSystem(path string, owner int, maximum int64, syncFile func(*os.File) error) (*fileSystem, error) {
	if path == "" || path != strings.TrimSpace(path) || strings.IndexByte(path, 0) >= 0 || owner < 0 || maximum <= 0 {
		return nil, errUnsafeFile
	}
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil || filepath.Base(absolute) == "." || filepath.Base(absolute) == string(filepath.Separator) {
		return nil, errUnsafeFile
	}
	directoryPath := filepath.Dir(absolute)
	if err := rejectSymlinkParents(directoryPath); err != nil {
		return nil, err
	}
	before, err := os.Lstat(directoryPath)
	if err != nil || !privateDirectory(before, uint32(owner)) {
		return nil, errUnsafeFile
	}
	directory, err := os.Open(directoryPath)
	if err != nil {
		return nil, errUnsafeFile
	}
	opened, statErr := directory.Stat()
	if statErr != nil || !privateDirectory(opened, uint32(owner)) || !os.SameFile(before, opened) {
		directory.Close()
		return nil, errUnsafeFile
	}
	root, err := os.OpenRoot(directoryPath)
	if err != nil {
		directory.Close()
		return nil, errUnsafeFile
	}
	rootInfo, err := root.Stat(".")
	if err != nil || !privateDirectory(rootInfo, uint32(owner)) || !os.SameFile(opened, rootInfo) {
		root.Close()
		directory.Close()
		return nil, errUnsafeFile
	}
	if syncFile == nil {
		syncFile = func(file *os.File) error { return file.Sync() }
	}
	fs := &fileSystem{
		path: absolute, name: filepath.Base(absolute), next: filepath.Base(absolute) + ".next",
		directory: directory, root: root, owner: uint32(owner), maximum: maximum, syncFile: syncFile,
		gate: make(chan struct{}, 1),
	}
	fs.gate <- struct{}{}
	return fs, nil
}

func (fs *fileSystem) close() {
	if fs == nil {
		return
	}
	_ = fs.root.Close()
	_ = fs.directory.Close()
}

func (fs *fileSystem) withLock(ctx context.Context, operation func() error) error {
	if ctx == nil || operation == nil {
		return errUnsafeFile
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-fs.gate:
	}
	defer func() { fs.gate <- struct{}{} }()
	retry := time.NewTicker(5 * time.Millisecond)
	defer retry.Stop()
	for {
		err := syscall.Flock(int(fs.directory.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return errUnsafeFile
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-retry.C:
		}
	}
	defer syscall.Flock(int(fs.directory.Fd()), syscall.LOCK_UN)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := fs.ensureDirectory(); err != nil {
		return err
	}
	return operation()
}

func (fs *fileSystem) ensureDirectory() error {
	current, err := os.Lstat(filepath.Dir(fs.path))
	opened, openedErr := fs.directory.Stat()
	if err != nil || openedErr != nil || !privateDirectory(current, fs.owner) ||
		!privateDirectory(opened, fs.owner) || !os.SameFile(current, opened) {
		return errUnsafeFile
	}
	return nil
}

func (fs *fileSystem) read(name string) ([]byte, bool, error) {
	before, err := fs.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil || !privateRegular(before, fs.owner, fs.maximum) {
		return nil, false, errUnsafeFile
	}
	file, err := fs.root.Open(name)
	if err != nil {
		return nil, false, errUnsafeFile
	}
	defer file.Close()
	return fs.readOpened(file, before)
}

func (fs *fileSystem) readOpened(file readableFile, before os.FileInfo) ([]byte, bool, error) {
	after, err := file.Stat()
	if err != nil || !privateRegular(after, fs.owner, fs.maximum) || !os.SameFile(before, after) {
		return nil, false, errUnsafeFile
	}
	content, err := io.ReadAll(io.LimitReader(file, fs.maximum+1))
	if err != nil || int64(len(content)) > fs.maximum || int64(len(content)) != after.Size() {
		return unsafeRead(content)
	}
	final, err := file.Stat()
	if err != nil || !privateRegular(final, fs.owner, fs.maximum) || !os.SameFile(after, final) ||
		final.Size() != after.Size() || !final.ModTime().Equal(after.ModTime()) {
		return unsafeRead(content)
	}
	return content, true, nil
}

func unsafeRead(content []byte) ([]byte, bool, error) {
	clear(content)
	return nil, false, errUnsafeFile
}

func (fs *fileSystem) replace(content []byte, hit func(string) error) error {
	if len(content) == 0 || int64(len(content)) > fs.maximum {
		return errUnsafeFile
	}
	if _, found, err := fs.read(fs.next); err != nil || found {
		return errUnsafeFile
	}
	next, err := fs.root.OpenFile(fs.next, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return errUnsafeFile
	}
	removeNext := true
	defer func() {
		_ = next.Close()
		if removeNext {
			_ = fs.root.Remove(fs.next)
		}
	}()
	written, err := next.Write(content)
	if err != nil || written != len(content) || fs.syncFile(next) != nil {
		return errUnsafeFile
	}
	info, err := next.Stat()
	if err != nil || !privateRegular(info, fs.owner, fs.maximum) || info.Size() != int64(len(content)) {
		return errUnsafeFile
	}
	removeNext = false
	if err := hit(FailpointAfterReplacementSync); err != nil {
		return err
	}
	if err := next.Close(); err != nil || fs.ensureDirectory() != nil || fs.root.Rename(fs.next, fs.name) != nil {
		return errUnsafeFile
	}
	if err := hit(FailpointAfterStoreRename); err != nil {
		return err
	}
	if err := fs.syncFile(fs.directory); err != nil {
		return errUnsafeFile
	}
	if err := hit(FailpointAfterStoreDirectorySync); err != nil {
		return err
	}
	return nil
}

func (fs *fileSystem) installNext() error {
	if err := fs.ensureDirectory(); err != nil {
		return err
	}
	if err := fs.root.Rename(fs.next, fs.name); err != nil || fs.syncFile(fs.directory) != nil {
		return errUnsafeFile
	}
	return nil
}

func rejectSymlinkParents(absolute string) error {
	current := string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, current)
	for _, component := range strings.Split(remainder, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errUnsafeFile
		}
	}
	return nil
}

func privateDirectory(info os.FileInfo, owner uint32) bool {
	identity, ok := fileIdentity(info)
	return ok && info.Mode() == os.ModeDir|0o700 && identity.Uid == owner
}

func privateRegular(info os.FileInfo, owner uint32, maximum int64) bool {
	identity, ok := fileIdentity(info)
	return ok && info.Mode() == 0o600 && info.Size() > 0 &&
		info.Size() <= maximum && identity.Uid == owner && identity.Nlink == 1
}

func fileIdentity(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok && stat != nil
}
