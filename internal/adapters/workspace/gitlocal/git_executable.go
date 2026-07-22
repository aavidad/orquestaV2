package gitlocal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	pinnedGitChildPath     = "/proc/self/fd/3"
	snapshotSourceRef      = "snapshot-source:gitlocal-object-authoritative:v2"
	snapshotSourceContract = "orquesta.snapshot-source/gitlocal-object-authoritative/v2\n" +
		"stream=ORQ-SNAPSHOT-2\nobjects=git-cat-file-batch-rehash\n" +
		"identity=commit,parent,tree,diff,write-set\ntraversal=iterative-bounded\n" +
		"executable=sealed-memfd-sha256\n"
	pinnedGitSeals      = unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_EXEC | unix.F_SEAL_SEAL
	gitCommandWaitDelay = 500 * time.Millisecond
)

type gitPinPolicy struct {
	ownerUID    uint32
	afterHash   func()
	mountFlags  func(int) (int64, error)
	memfdCreate func(string, int) (int, error)
}

func pinGitExecutable(path string, policy gitPinPolicy) (*os.File, string, error) {
	invalid := func(cause error) (*os.File, string, error) {
		return nil, "", &Error{Code: CodeConfigInvalid, Cause: cause}
	}
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return invalid(nil)
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK),
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return invalid(err)
	}
	source := os.NewFile(uintptr(fd), "orquesta-git-source")
	if source == nil {
		_ = unix.Close(fd)
		return invalid(nil)
	}
	fail := func(cause error) (*os.File, string, error) {
		_ = source.Close()
		return invalid(cause)
	}
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil || !safeGitExecutableStat(&opened, policy.ownerUID) {
		return fail(err)
	}
	if err := validateExecutableMount(fd, policy); err != nil {
		return fail(err)
	}
	sealed, digest, err := copySealedGitExecutable(source, opened.Size, policy)
	if err != nil {
		return fail(err)
	}
	if policy.afterHash != nil {
		policy.afterHash()
	}
	var after, bound unix.Stat_t
	if err := unix.Fstat(fd, &after); err != nil || unix.Lstat(path, &bound) != nil ||
		!safeGitExecutableStat(&after, policy.ownerUID) || !safeGitExecutableStat(&bound, policy.ownerUID) ||
		!sameExecutableCapture(&opened, &after) || !sameExecutableCapture(&opened, &bound) {
		_ = sealed.Close()
		return fail(errors.New("git executable changed during capture"))
	}
	if err := source.Close(); err != nil {
		_ = sealed.Close()
		return invalid(err)
	}
	return sealed, digest, nil
}

func validateExecutableMount(fd int, policy gitPinPolicy) error {
	flags := policy.mountFlags
	if flags == nil {
		flags = func(fd int) (int64, error) {
			var stat unix.Statfs_t
			err := unix.Fstatfs(fd, &stat)
			return int64(stat.Flags), err
		}
	}
	value, err := flags(fd)
	if err != nil || value&unix.MS_NOEXEC != 0 {
		return errors.New("git executable mount is not executable")
	}
	return nil
}

func copySealedGitExecutable(source *os.File, size int64, policy gitPinPolicy) (*os.File, string, error) {
	create := policy.memfdCreate
	if create == nil {
		create = unix.MemfdCreate
	}
	fd, err := create("orquesta-pinned-git", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING|unix.MFD_EXEC)
	if err != nil {
		return nil, "", err
	}
	file := os.NewFile(uintptr(fd), "orquesta-pinned-git")
	fail := func(cause error) (*os.File, string, error) {
		if file != nil {
			_ = file.Close()
		} else {
			_ = unix.Close(fd)
		}
		return nil, "", cause
	}
	if file == nil {
		return fail(errors.New("invalid git memfd"))
	}
	digest := sha256.New()
	if written, err := io.Copy(io.MultiWriter(file, digest), io.NewSectionReader(source, 0, size)); err != nil || written != size {
		return fail(errors.New("git executable changed during copy"))
	}
	if err := sealMemfd(file, 0o500, pinnedGitSeals); err != nil {
		return fail(err)
	}
	return file, hex.EncodeToString(digest.Sum(nil)), nil
}

func sealMemfd(file *os.File, mode os.FileMode, required int) error {
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, required); err != nil {
		return err
	}
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&required != required {
		return errors.New("memfd seals unavailable")
	}
	return nil
}

func safeGitExecutableStat(stat *unix.Stat_t, owner uint32) bool {
	return stat != nil && stat.Mode&unix.S_IFMT == unix.S_IFREG && stat.Mode&0o111 != 0 && stat.Mode&0o022 == 0 &&
		stat.Mode&(unix.S_ISUID|unix.S_ISGID|unix.S_ISVTX) == 0 && stat.Uid == owner && stat.Size > 0
}

func sameExecutableCapture(left, right *unix.Stat_t) bool {
	return left != nil && right != nil && left.Dev == right.Dev && left.Ino == right.Ino && left.Uid == right.Uid &&
		left.Mode == right.Mode && left.Size == right.Size && left.Mtim == right.Mtim && left.Ctim == right.Ctim
}

func snapshotSourceIdentity(contract, gitDigest string) (string, string) {
	digest := sha256.Sum256([]byte(contract + "\x00" + gitDigest))
	return snapshotSourceRef, hex.EncodeToString(digest[:])
}

func (adapter *Adapter) pinnedGitCommand(ctx context.Context, extraFiles []*os.File, args ...string) (*exec.Cmd, func(), error) {
	if adapter == nil {
		return nil, func() {}, &Error{Code: CodeUnavailable}
	}
	adapter.gitMu.RLock()
	if adapter.closed || adapter.gitFile == nil {
		adapter.gitMu.RUnlock()
		return nil, func() {}, &Error{Code: CodeUnavailable}
	}
	git, err := duplicateFile(adapter.gitFile, "orquesta-git-command")
	adapter.gitMu.RUnlock()
	if err != nil {
		return nil, func() {}, &Error{Code: CodeUnavailable, Cause: err}
	}
	files := []*os.File{git}
	for _, extra := range extraFiles {
		file, err := duplicateFile(extra, "orquesta-git-extra")
		if err != nil {
			closeFiles(files)
			return nil, func() {}, &Error{Code: CodeUnavailable, Cause: err}
		}
		files = append(files, file)
	}
	command := exec.CommandContext(ctx, pinnedGitChildPath, args...)
	configurePinnedCommand(command)
	command.ExtraFiles = files
	return command, func() { closeFiles(files) }, nil
}

func duplicateFile(source *os.File, name string) (*os.File, error) {
	if source == nil {
		return nil, errors.New("missing pinned descriptor")
	}
	fd, err := unix.FcntlInt(source.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func configurePinnedCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	command.WaitDelay = gitCommandWaitDelay
	command.Cancel = func() error { return killPinnedProcessGroup(command) }
}

func runPinnedCommand(command *exec.Cmd) error  { return finishPinnedCommand(command, command.Run()) }
func waitPinnedCommand(command *exec.Cmd) error { return finishPinnedCommand(command, command.Wait()) }

func finishPinnedCommand(command *exec.Cmd, err error) error {
	if cleanupErr := killPinnedProcessGroup(command); !errors.Is(cleanupErr, os.ErrProcessDone) {
		err = errors.Join(err, cleanupErr)
	}
	return err
}

func killPinnedProcessGroup(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return os.ErrProcessDone
	}
	if err := unix.Kill(-command.Process.Pid, unix.SIGKILL); !errors.Is(err, unix.ESRCH) {
		return err
	}
	return os.ErrProcessDone
}

func closeFiles(files []*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}
