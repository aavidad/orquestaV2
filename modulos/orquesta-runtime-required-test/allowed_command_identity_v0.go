package orquestaruntimerequiredtest

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	allowedCommandMemfdCloexecV0      = 0x0001
	allowedCommandMemfdAllowSealingV0 = 0x0002
	allowedCommandFAddSealsV0         = 0x0409
	allowedCommandFGetSealsV0         = 0x040a
	allowedCommandFSealSealV0         = 0x0001
	allowedCommandFSealShrinkV0       = 0x0002
	allowedCommandFSealGrowV0         = 0x0004
	allowedCommandFSealWriteV0        = 0x0008
	allowedCommandRequiredSealsV0     = allowedCommandFSealSealV0 | allowedCommandFSealShrinkV0 | allowedCommandFSealGrowV0 | allowedCommandFSealWriteV0
)

// allowedCommandIdentityRegistryV0 owns the descriptors admitted at
// configuration time. A command is launched only through a duplicate of one
// of these descriptors, never by reopening its configured path.
type allowedCommandIdentityRegistryV0 struct {
	mu          sync.Mutex
	commands    map[string]*allowedCommandIdentityV0
	resources   []*os.File
	commandDirs []string
	closed      bool
}

type allowedCommandIdentityV0 struct {
	alias, path string
	// file is the immutable executable master. source is retained only to
	// revalidate the admitted pathname/inode/content before every launch.
	file, source *os.File
	sha          [sha256.Size]byte
	dev, ino     uint64
}

func newAllowedCommandIdentityRegistryV0(allowed map[string]string) (*allowedCommandIdentityRegistryV0, error) {
	r := &allowedCommandIdentityRegistryV0{commands: make(map[string]*allowedCommandIdentityV0, len(allowed))}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("required_test_allowed_commands_required")
	}
	for alias, path := range allowed {
		normalizedAlias := strings.TrimSpace(alias)
		if commandIsShellV0(normalizedAlias) || commandIsShellV0(strings.TrimSpace(path)) {
			_ = r.Close()
			return nil, fmt.Errorf("required_test_command_shell_prohibited")
		}
		if _, duplicate := r.commands[normalizedAlias]; duplicate {
			_ = r.Close()
			return nil, fmt.Errorf("required_test_command_alias_duplicate: %s", normalizedAlias)
		}
		identity, err := captureAllowedCommandIdentityV0(alias, path)
		if err != nil {
			_ = r.Close()
			return nil, err
		}
		r.commands[identity.alias] = identity
		r.resources = append(r.resources, identity.file, identity.source)
		r.commandDirs = append(r.commandDirs, filepath.Dir(identity.path))
	}
	return r, nil
}

func captureAllowedCommandIdentityV0(alias, path string) (*allowedCommandIdentityV0, error) {
	alias, path = strings.TrimSpace(alias), strings.TrimSpace(path)
	if alias == "" || strings.ContainsAny(alias, `/\\`) || commandIsShellV0(alias) || !filepath.IsAbs(path) || pathHasCredentialMarkerV0(path) || commandIsShellV0(path) {
		return nil, fmt.Errorf("required_test_command_identity_invalid: %s", alias)
	}
	before, err := lstatAllowedCommandV0(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("required_test_command_identity_invalid: %s", alias)
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("required_test_command_identity_invalid: %s", alias)
	}
	source := os.NewFile(uintptr(fd), path)
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || uint64(stat.Dev) != before.dev || uint64(stat.Ino) != before.ino || stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Mode&0o111 == 0 {
		_ = source.Close()
		return nil, fmt.Errorf("required_test_command_identity_changed: %s", alias)
	}
	snapshot, digest, err := sealedAllowedCommandSnapshotV0(fd)
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("required_test_command_identity_invalid: %s", alias)
	}
	sourceDigest, err := sha256FileDescriptorV0(fd)
	if err != nil || sourceDigest != digest {
		_ = snapshot.Close()
		_ = source.Close()
		return nil, fmt.Errorf("required_test_command_identity_changed: %s", alias)
	}
	return &allowedCommandIdentityV0{alias: alias, path: path, file: snapshot, source: source, sha: digest, dev: before.dev, ino: before.ino}, nil
}

type allowedCommandStatV0 struct {
	mode     os.FileMode
	dev, ino uint64
}

func (s allowedCommandStatV0) Mode() os.FileMode { return s.mode }
func lstatAllowedCommandV0(path string) (allowedCommandStatV0, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return allowedCommandStatV0{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return allowedCommandStatV0{}, fmt.Errorf("required_test_command_stat_unavailable")
	}
	return allowedCommandStatV0{info.Mode(), uint64(stat.Dev), uint64(stat.Ino)}, nil
}

func sealedAllowedCommandSnapshotV0(sourceFD int) (*os.File, [sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	fd, err := memfdCreateAllowedCommandV0()
	if err != nil {
		return nil, zero, err
	}
	snapshot := os.NewFile(uintptr(fd), "orquesta-allowed-command-snapshot-v0")
	failed := true
	defer func() {
		if failed {
			_ = snapshot.Close()
		}
	}()
	if err := copyFileDescriptorV0(sourceFD, fd); err != nil {
		return nil, zero, err
	}
	if err := syscall.Fchmod(fd, 0o500); err != nil {
		return nil, zero, err
	}
	digest, err := sha256FileDescriptorV0(fd)
	if err != nil {
		return nil, zero, err
	}
	if err := addAndVerifyAllowedCommandSealsV0(fd); err != nil {
		return nil, zero, err
	}
	failed = false
	return snapshot, digest, nil
}

func memfdCreateAllowedCommandV0() (int, error) {
	number := memfdCreateSyscallNumberV0()
	if number == 0 {
		return -1, fmt.Errorf("required_test_command_memfd_unavailable")
	}
	name, err := syscall.BytePtrFromString("orquesta-allowed-command-v0")
	if err != nil {
		return -1, err
	}
	fd, _, errno := syscall.Syscall(number, uintptr(unsafe.Pointer(name)), uintptr(allowedCommandMemfdCloexecV0|allowedCommandMemfdAllowSealingV0), 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func memfdCreateSyscallNumberV0() uintptr {
	return memfdCreateSyscallNumberForGOARCHV0(runtime.GOARCH)
}

func memfdCreateSyscallNumberForGOARCHV0(goarch string) uintptr {
	switch goarch {
	case "amd64":
		return 319
	case "386":
		return 356
	case "arm64", "riscv64", "loong64":
		return 279
	case "arm":
		return 385
	case "ppc64", "ppc64le":
		return 360
	case "s390x":
		return 350
	default:
		return 0
	}
}

func copyFileDescriptorV0(sourceFD, destinationFD int) error {
	buffer := make([]byte, 32*1024)
	for offset := int64(0); ; {
		n, readErr := syscall.Pread(sourceFD, buffer, offset)
		if readErr == syscall.EINTR {
			continue
		}
		if n > 0 {
			written := 0
			for written < n {
				count, writeErr := syscall.Pwrite(destinationFD, buffer[written:n], offset+int64(written))
				if writeErr == syscall.EINTR {
					continue
				}
				if writeErr != nil || count <= 0 {
					if writeErr == nil {
						writeErr = io.ErrShortWrite
					}
					return writeErr
				}
				written += count
			}
			offset += int64(n)
		}
		if readErr == io.EOF || n == 0 {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func addAndVerifyAllowedCommandSealsV0(fd int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(allowedCommandFAddSealsV0), uintptr(allowedCommandRequiredSealsV0))
	if errno != 0 {
		return errno
	}
	seals, err := allowedCommandSealsV0(fd)
	if err != nil || seals&allowedCommandRequiredSealsV0 != allowedCommandRequiredSealsV0 {
		if err != nil {
			return err
		}
		return fmt.Errorf("required_test_command_memfd_seals_unavailable")
	}
	return nil
}

func allowedCommandSealsV0(fd int) (int, error) {
	seals, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(allowedCommandFGetSealsV0), 0)
	if errno != 0 {
		return 0, errno
	}
	return int(seals), nil
}
func sha256FileDescriptorV0(fd int) ([sha256.Size]byte, error) {
	var out [sha256.Size]byte
	h := sha256.New()
	buf := make([]byte, 32*1024)
	for off := int64(0); ; {
		n, err := syscall.Pread(fd, buf, off)
		if n > 0 {
			_, _ = h.Write(buf[:n])
			off += int64(n)
		}
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return out, err
		}
	}
	copy(out[:], h.Sum(nil))
	return out, nil
}

func (r *allowedCommandIdentityRegistryV0) resolve(command string) (commandAllowlistResolutionV0, error) {
	tokens, err := splitCommandV0(command)
	if err != nil {
		return commandAllowlistResolutionV0{}, &commandAllowlistResolutionErrorV0{Failure: commandAllowlistSyntaxFailureV0, Syntax: err}
	}
	return r.resolveTokens(tokens)
}

func (r *allowedCommandIdentityRegistryV0) resolveAlias(alias string, args ...string) (commandAllowlistResolutionV0, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" || strings.ContainsAny(alias, `/\\`) {
		return commandAllowlistResolutionV0{}, &commandAllowlistResolutionErrorV0{Failure: commandAllowlistSyntaxFailureV0, Syntax: fmt.Errorf("required_test_command_name_invalid")}
	}
	return r.resolveTokens(append([]string{alias}, args...))
}

func (r *allowedCommandIdentityRegistryV0) resolveTokens(tokens []string) (commandAllowlistResolutionV0, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return commandAllowlistResolutionV0{}, fmt.Errorf("required_test_command_identity_changed")
	}
	id, ok := r.commands[tokens[0]]
	if !ok {
		return commandAllowlistResolutionV0{}, &commandAllowlistResolutionErrorV0{Failure: commandAllowlistNotAllowedFailureV0, Command: tokens[0]}
	}
	if commandIsShellV0(tokens[0]) {
		return commandAllowlistResolutionV0{}, &commandAllowlistResolutionErrorV0{Failure: commandAllowlistShellFailureV0, Command: tokens[0]}
	}
	current, err := lstatAllowedCommandV0(id.path)
	if err != nil || current.dev != id.dev || current.ino != id.ino || !current.mode.IsRegular() || current.mode&os.ModeSymlink != 0 {
		return commandAllowlistResolutionV0{}, fmt.Errorf("required_test_command_identity_changed: %s", id.alias)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(id.source.Fd()), &stat); err != nil || uint64(stat.Dev) != id.dev || uint64(stat.Ino) != id.ino || stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Mode&0o111 == 0 {
		return commandAllowlistResolutionV0{}, fmt.Errorf("required_test_command_identity_changed: %s", id.alias)
	}
	digest, err := sha256FileDescriptorV0(int(id.source.Fd()))
	if err != nil || digest != id.sha {
		return commandAllowlistResolutionV0{}, fmt.Errorf("required_test_command_identity_changed: %s", id.alias)
	}
	seals, err := allowedCommandSealsV0(int(id.file.Fd()))
	if err != nil || seals&allowedCommandRequiredSealsV0 != allowedCommandRequiredSealsV0 {
		return commandAllowlistResolutionV0{}, fmt.Errorf("required_test_command_identity_changed: %s", id.alias)
	}
	duplicate, err := duplicateAllowedCommandFileV0(id.file)
	if err != nil {
		return commandAllowlistResolutionV0{}, fmt.Errorf("required_test_command_identity_changed: %s", id.alias)
	}
	return commandAllowlistResolutionV0{Tokens: tokens, CommandPath: id.path, identity: id, executionFile: duplicate}, nil
}
func duplicateAllowedCommandFileV0(file *os.File) (*os.File, error) {
	fd, _, errno := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), uintptr(syscall.F_DUPFD_CLOEXEC), 3)
	if errno != 0 {
		return nil, errno
	}
	return os.NewFile(fd, file.Name()), nil
}
func (r *allowedCommandIdentityRegistryV0) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	var first error
	for _, resource := range r.resources {
		first = errors.Join(first, resource.Close())
	}
	return first
}

func (r *allowedCommandIdentityRegistryV0) commandDirectoriesV0() []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	return append([]string(nil), r.commandDirs...)
}
