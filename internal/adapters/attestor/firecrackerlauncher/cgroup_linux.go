//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type runnerCgroup interface {
	prepare(string) error
	kill(string) error
	cleanup(context.Context, string, LaunchRequest, bool) error
	Close() error
}

type cgroupParent struct {
	root   *os.File
	parent *os.File
}

func openCgroupParent(config Config, owner uint32) (*cgroupParent, error) {
	if validateTrustedAncestors(config.CgroupRoot, owner) != nil {
		return nil, launcherError(CodeResourceUnsafe)
	}
	rootFD, err := unix.Openat2(unix.AT_FDCWD, config.CgroupRoot, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeResourceUnsafe)
	}
	root := os.NewFile(uintptr(rootFD), "orquesta-firecracker-cgroup-root")
	fail := func() (*cgroupParent, error) {
		_ = root.Close()
		return nil, launcherError(CodeResourceUnsafe)
	}
	var filesystem unix.Statfs_t
	var rootStat unix.Stat_t
	if unix.Fstatfs(rootFD, &filesystem) != nil ||
		filesystem.Type != unix.CGROUP2_SUPER_MAGIC ||
		unix.Fstat(rootFD, &rootStat) != nil ||
		rootStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		rootStat.Uid != owner || rootStat.Gid != owner ||
		rootStat.Mode&0o022 != 0 ||
		!isCgroupMountRoot(config.CgroupRoot) {
		return fail()
	}
	parentFD, err := unix.Openat2(rootFD, config.ParentCgroup, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return fail()
	}
	parent := os.NewFile(uintptr(parentFD), "orquesta-firecracker-cgroup-parent")
	var parentStat unix.Stat_t
	if unix.Fstat(parentFD, &parentStat) != nil ||
		parentStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		parentStat.Uid != owner || parentStat.Gid != owner ||
		parentStat.Mode&0o777 != 0o755 ||
		!controlSetContains(parentFD, "cgroup.controllers", "cpu", "memory", "pids") ||
		!controlSetContains(parentFD, "cgroup.subtree_control", "cpu", "memory", "pids") ||
		!controlFileEmpty(parentFD, "cgroup.procs") ||
		!cgroupHasNoChildren(parent, config.MaxCleanupEntries) {
		_ = parent.Close()
		return fail()
	}
	return &cgroupParent{root: root, parent: parent}, nil
}

func cgroupHasNoChildren(parent *os.File, maxEntries uint32) bool {
	if parent == nil || maxEntries == 0 {
		return false
	}
	var visited uint32
	for {
		entries, err := parent.ReadDir(32)
		for _, entry := range entries {
			if visited >= maxEntries || entry.IsDir() {
				return false
			}
			visited++
		}
		if errors.Is(err, io.EOF) {
			return true
		}
		if err != nil {
			return false
		}
	}
}

func isCgroupMountRoot(path string) bool {
	parentFD, err := unix.Open(
		filepath.Dir(path),
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return false
	}
	defer unix.Close(parentFD)
	var filesystem unix.Statfs_t
	return unix.Fstatfs(parentFD, &filesystem) == nil &&
		filesystem.Type != unix.CGROUP2_SUPER_MAGIC
}

func controlSetContains(directory int, name string, required ...string) bool {
	content, err := readCgroupControl(directory, name)
	if err != nil {
		return false
	}
	available := make(map[string]struct{})
	for _, value := range strings.Fields(content) {
		available[strings.TrimPrefix(value, "+")] = struct{}{}
	}
	for _, value := range required {
		if _, ok := available[value]; !ok {
			return false
		}
	}
	return true
}

func controlFileEmpty(directory int, name string) bool {
	content, err := readCgroupControl(directory, name)
	return err == nil && strings.TrimSpace(content) == ""
}

func (parent *cgroupParent) prepare(id string) error {
	if parent == nil || parent.parent == nil || !validJailID(id) {
		return launcherError(CodeResourceUnsafe)
	}
	var stat unix.Stat_t
	err := unix.Fstatat(
		int(parent.parent.Fd()),
		id,
		&stat,
		unix.AT_SYMLINK_NOFOLLOW,
	)
	if !errors.Is(err, unix.ENOENT) {
		return launcherError(CodeResourceUnsafe)
	}
	return nil
}

func (parent *cgroupParent) kill(id string) error {
	leaf, err := parent.openLeaf(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return launcherError(CodeCleanupFailed)
	}
	defer leaf.Close()
	err = writeCgroupControl(int(leaf.Fd()), "cgroup.kill", "1")
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return launcherError(CodeCleanupFailed)
	}
	return nil
}

func (parent *cgroupParent) cleanup(
	ctx context.Context,
	id string,
	request LaunchRequest,
	required bool,
) error {
	if parent == nil || parent.parent == nil || !validJailID(id) {
		return launcherError(CodeCleanupFailed)
	}
	leaf, err := parent.openLeaf(id)
	if errors.Is(err, os.ErrNotExist) && !required {
		return nil
	}
	if err != nil {
		return launcherError(CodeCleanupFailed)
	}
	leafFD := int(leaf.Fd())
	validateErr := validateCgroupLimits(leafFD, request)
	killErr := writeCgroupControl(leafFD, "cgroup.kill", "1")
	if errors.Is(killErr, unix.ENOENT) {
		killErr = nil
	}
	waitErr := waitCgroupEmpty(ctx, leafFD)
	closeErr := leaf.Close()
	removeErr := unix.Unlinkat(
		int(parent.parent.Fd()),
		id,
		unix.AT_REMOVEDIR,
	)
	if errors.Join(validateErr, killErr, waitErr, closeErr, removeErr) != nil {
		return launcherError(CodeCleanupFailed)
	}
	return nil
}

func (parent *cgroupParent) openLeaf(id string) (*os.File, error) {
	fd, err := unix.Openat2(int(parent.parent.Fd()), id, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if errors.Is(err, unix.ENOENT) {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-cgroup-leaf")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Uid != 0 || stat.Gid != 0 ||
		stat.Mode&0o022 != 0 {
		_ = file.Close()
		return nil, launcherError(CodeResourceUnsafe)
	}
	return file, nil
}

func validateCgroupLimits(directory int, request LaunchRequest) error {
	expected := map[string]string{
		"cpu.max":          fmt.Sprintf("%d %d", request.CPUQuotaMicros, request.CPUPeriodMicros),
		"memory.max":       strconv.FormatUint(request.MemoryMaxBytes, 10),
		"memory.swap.max":  "0",
		"memory.oom.group": "1",
		"pids.max":         strconv.FormatUint(uint64(request.PIDsMax), 10),
	}
	for name, want := range expected {
		got, err := readCgroupControl(directory, name)
		if err != nil || got != want {
			return launcherError(CodeResourceUnsafe)
		}
	}
	return nil
}

func waitCgroupEmpty(ctx context.Context, directory int) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		content, err := readCgroupControl(directory, "cgroup.events")
		if err != nil {
			return err
		}
		fields := strings.Fields(content)
		for index := 0; index+1 < len(fields); index += 2 {
			if fields[index] == "populated" && fields[index+1] == "0" {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func readCgroupControl(directory int, name string) (string, error) {
	fd, err := unix.Openat(
		directory,
		name,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return "", err
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-cgroup-control")
	content, readErr := io.ReadAll(io.LimitReader(file, 4097))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(content) > 4096 {
		return "", launcherError(CodeResourceUnsafe)
	}
	return strings.TrimSpace(string(content)), nil
}

func writeCgroupControl(directory int, name, value string) error {
	fd, err := unix.Openat(
		directory,
		name,
		unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	count, err := unix.Write(fd, []byte(value))
	if err != nil || count != len(value) {
		return launcherError(CodeResourceUnsafe)
	}
	return nil
}

func (parent *cgroupParent) Close() error {
	if parent == nil {
		return nil
	}
	return errors.Join(closeFile(parent.parent), closeFile(parent.root))
}

func closeFile(file *os.File) error {
	if file == nil {
		return nil
	}
	return file.Close()
}
