//go:build linux

package bubblewrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type cgroupRoot struct {
	mu       sync.Mutex
	file     *os.File
	identity string
	period   int64
	next     uint64
	closed   bool
}
type cgroupLeaf struct {
	root, dir int
	name      string
	closed    bool
}
type cgroupController interface {
	newLeaf(Limits) (cgroupSession, error)
	Close() error
}
type cgroupSession interface {
	apply(*exec.Cmd) error
	kill() error
	exceeded() (bool, error)
	close(context.Context) error
}

func openCgroupRoot(name string) (*cgroupRoot, error) {
	if name == "" || !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return nil, resourceError()
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, name, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if err != nil {
		return nil, resourceError()
	}
	root := &cgroupRoot{file: os.NewFile(uintptr(fd), "orquesta-cgroup")}
	var fs unix.Statfs_t
	var stat unix.Stat_t
	cpuMax, readErr := readControl(fd, "cpu.max")
	period, periodErr := cpuPeriod(cpuMax)
	if unix.Fstatfs(fd, &fs) != nil || fs.Type != unix.CGROUP2_SUPER_MAGIC || unix.Fstat(fd, &stat) != nil || stat.Uid != uint32(os.Geteuid()) || stat.Mode&0o077 != 0 || readErr != nil || periodErr != nil {
		_ = root.file.Close()
		return nil, resourceError()
	}
	identity := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%o:%d:%d:%s", stat.Dev, stat.Ino, stat.Mode, stat.Uid, stat.Gid, cpuMax)))
	root.identity, root.period = hex.EncodeToString(identity[:]), period
	return root, nil
}

func (root *cgroupRoot) newLeaf(limits Limits) (cgroupSession, error) {
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.closed {
		return nil, resourceError()
	}
	root.next++
	parent := int(root.file.Fd())
	name := fmt.Sprintf("run-%d-%d", os.Getpid(), root.next)
	if unix.Mkdirat(parent, name, 0o700) != nil {
		return nil, resourceError()
	}
	dir, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	leaf := &cgroupLeaf{root: parent, dir: dir, name: name}
	for _, control := range [][2]string{
		{"memory.max", strconv.FormatInt(limits.MemoryMaxBytes, 10)}, {"memory.oom.group", "1"},
		{"pids.max", strconv.FormatInt(limits.PIDsMax, 10)},
		{"cpu.max", fmt.Sprintf("%d %d", limits.CPUQuotaMicros, root.period)},
	} {
		if err == nil {
			err = leaf.set(control[0], control[1])
		}
	}
	if err == nil {
		return leaf, nil
	}
	if dir >= 0 {
		_ = unix.Close(dir)
	}
	_ = unix.Unlinkat(parent, name, unix.AT_REMOVEDIR)
	return nil, resourceError()
}

func (leaf *cgroupLeaf) set(name, value string) error { return writeControl(leaf.dir, name, value) }
func (leaf *cgroupLeaf) apply(command *exec.Cmd) error {
	if leaf == nil || command == nil || leaf.dir < 0 {
		return resourceError()
	}
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{}
	}
	command.SysProcAttr.UseCgroupFD, command.SysProcAttr.CgroupFD = true, leaf.dir
	return nil
}
func (leaf *cgroupLeaf) kill() error {
	if leaf == nil || leaf.dir < 0 {
		return nil
	}
	err := writeControl(leaf.dir, "cgroup.kill", "1")
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	return err
}
func (leaf *cgroupLeaf) exceeded() (bool, error) {
	exceeded := false
	for _, check := range []struct {
		file string
		keys []string
	}{{"memory.events", []string{"oom", "oom_kill"}}, {"pids.events", []string{"max"}}} {
		content, err := readControl(leaf.dir, check.file)
		if err != nil {
			return false, err
		}
		positive, err := eventPositive(content, check.keys...)
		if err != nil {
			return false, err
		}
		exceeded = exceeded || positive
	}
	return exceeded, nil
}
func (leaf *cgroupLeaf) close(ctx context.Context) error {
	if leaf.closed {
		return nil
	}
	leaf.closed = true
	result := leaf.kill()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for result == nil {
		content, err := readControl(leaf.dir, "cgroup.events")
		populated, parseErr := eventPositive(content, "populated")
		if err = errors.Join(err, parseErr); err != nil {
			result = err
		} else if !populated {
			break
		} else {
			select {
			case <-ctx.Done():
				result = ctx.Err()
			case <-ticker.C:
			}
		}
	}
	result = errors.Join(result, unix.Close(leaf.dir))
	leaf.dir = -1
	return errors.Join(result, unix.Unlinkat(leaf.root, leaf.name, unix.AT_REMOVEDIR))
}
func (root *cgroupRoot) Close() error {
	if root == nil {
		return nil
	}
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.closed {
		return nil
	}
	root.closed = true
	return root.file.Close()
}

func readControl(dir int, name string) (string, error) {
	fd, err := unix.Openat(dir, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", resourceError()
	}
	file := os.NewFile(uintptr(fd), "cgroup-control")
	content, readErr := io.ReadAll(io.LimitReader(file, 4097))
	if closeErr := file.Close(); readErr != nil || closeErr != nil || len(content) > 4096 {
		return "", resourceError()
	}
	return strings.TrimSpace(string(content)), nil
}
func writeControl(dir int, name, value string) error {
	fd, err := unix.Openat(dir, name, unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	count, err := unix.Write(fd, []byte(value))
	if err != nil || count != len(value) {
		return resourceError()
	}
	return nil
}
func eventPositive(content string, required ...string) (bool, error) {
	fields := strings.Fields(content)
	if len(fields) == 0 || len(fields)%2 != 0 || len(required) == 0 {
		return false, resourceError()
	}
	values := make(map[string]uint64, len(fields)/2)
	for index := 0; index < len(fields); index += 2 {
		value, err := strconv.ParseUint(fields[index+1], 10, 64)
		_, duplicate := values[fields[index]]
		if err != nil || duplicate || !validEventName(fields[index]) {
			return false, resourceError()
		}
		values[fields[index]] = value
	}
	positive := false
	for _, name := range required {
		value, found := values[name]
		if !found {
			return false, resourceError()
		}
		positive = positive || value > 0
	}
	return positive, nil
}
func validEventName(name string) bool {
	return name != "" && name[0] >= 'a' && name[0] <= 'z' &&
		strings.Trim(name, "abcdefghijklmnopqrstuvwxyz0123456789_") == ""
}
func cpuPeriod(value string) (int64, error) {
	fields := strings.Fields(value)
	if len(fields) == 2 {
		if period, err := strconv.ParseInt(fields[1], 10, 64); err == nil && period > 0 {
			return period, nil
		}
	}
	return 0, resourceError()
}
func resourceError() error { return &Error{Code: CodeResourceUnsafe} }

func stopSandbox(command *exec.Cmd, leaf cgroupSession, done <-chan error, timeout time.Duration) error {
	var killErr error
	if command != nil && command.Process != nil {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			killErr = err
		}
	}
	if leaf != nil {
		killErr = errors.Join(killErr, leaf.kill())
	}
	if killErr != nil && command != nil && command.Process != nil {
		killErr = errors.Join(killErr, command.Process.Kill())
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return killErr
	case <-timer.C:
		return errors.Join(killErr, context.DeadlineExceeded)
	}
}
