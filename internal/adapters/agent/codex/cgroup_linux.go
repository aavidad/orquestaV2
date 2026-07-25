//go:build linux

package codex

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type cgroupIdentity struct {
	Device uint64
	Inode  uint64
}

type codexCgroupRoot struct {
	mu              sync.Mutex
	rootPath        string
	root            *os.File
	control         *os.File
	rootIdentity    cgroupIdentity
	controlIdentity cgroupIdentity
	closed          bool
}

type codexCgroupLeaf struct {
	root     *codexCgroupRoot
	file     *os.File
	name     string
	identity cgroupIdentity
}

const (
	cgroupAllocationFileName = "cgroup-allocation.json"
	cgroupBoundaryFileName   = "cgroup-boundary.json"
	cgroupJournalSchema      = 1
)

type cgroupAllocationRecord struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	RootDevice    uint64 `json:"root_device"`
	RootInode     uint64 `json:"root_inode"`
}

type cgroupBoundaryRecord struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	RootDevice    uint64 `json:"root_device"`
	RootInode     uint64 `json:"root_inode"`
	ControlDevice uint64 `json:"control_device"`
	ControlInode  uint64 `json:"control_inode"`
	Device        uint64 `json:"device"`
	Inode         uint64 `json:"inode"`
}

func platformCgroupRequired() bool { return true }

func openCodexCgroupRoot(configured string) (*codexCgroupRoot, error) {
	if configured == "" || !filepath.IsAbs(configured) || filepath.Clean(configured) != configured {
		return nil, cgroupError(CodeCgroupRootInvalid, nil)
	}
	const mountPath = "/sys/fs/cgroup"
	relative, err := filepath.Rel(mountPath, configured)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, cgroupError(CodeCgroupRootInvalid, err)
	}
	mount, err := openCgroupMount(mountPath)
	if err != nil {
		return nil, cgroupError(CodeCgroupRootInvalid, err)
	}
	root, identity, err := openCgroupDirectoryBeneath(int(mount.Fd()), relative, true)
	_ = mount.Close()
	if err != nil {
		return nil, cgroupError(CodeCgroupRootInvalid, err)
	}
	fail := func(cause error) (*codexCgroupRoot, error) {
		_ = root.Close()
		return nil, cause
	}
	selfPath, err := currentCgroupPath()
	if err != nil {
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	controlRelative, err := filepath.Rel(configured, selfPath)
	if err != nil || controlRelative == "." || controlRelative == ".." ||
		strings.HasPrefix(controlRelative, ".."+string(filepath.Separator)) {
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	if empty, err := cgroupProcessesEmpty(int(root.Fd())); err != nil || !empty {
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	if err := preflightCgroupWrite(int(root.Fd()), "cgroup.procs", "cgroup.kill"); err != nil {
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	control, controlIdentity, err := openCgroupDirectoryBeneath(int(root.Fd()), controlRelative, false)
	if err != nil {
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	controller := &codexCgroupRoot{
		rootPath: configured, root: root, control: control,
		rootIdentity: identity, controlIdentity: controlIdentity,
	}
	if err := preflightCgroupWrite(int(control.Fd()), "cgroup.procs"); err != nil {
		_ = control.Close()
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	if member, err := cgroupHasPID(int(control.Fd()), os.Getpid()); err != nil || !member {
		_ = control.Close()
		return fail(cgroupError(CodeCgroupRootInvalid, err))
	}
	return controller, nil
}

func openCgroupMount(name string) (*os.File, error) {
	fd, err := unix.Openat2(unix.AT_FDCWD, name, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, err
	}
	var fs unix.Statfs_t
	if unix.Fstatfs(fd, &fs) != nil || fs.Type != unix.CGROUP2_SUPER_MAGIC {
		_ = unix.Close(fd)
		return nil, errors.New("cgroup v2 mount required")
	}
	return os.NewFile(uintptr(fd), "orquesta-cgroup2-mount"), nil
}

func openCgroupDirectoryBeneath(parent int, name string, requireOwner bool) (*os.File, cgroupIdentity, error) {
	fd, err := unix.Openat2(parent, name, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, cgroupIdentity{}, err
	}
	file := os.NewFile(uintptr(fd), "orquesta-codex-control-cgroup")
	identity, err := inspectCgroupFD(fd, requireOwner)
	if err != nil {
		_ = file.Close()
		return nil, cgroupIdentity{}, err
	}
	return file, identity, nil
}

func currentCgroupPath() (string, error) {
	payload, err := os.ReadFile("/proc/self/cgroup")
	if err != nil || len(payload) == 0 || len(payload) > 4096 {
		return "", errors.New("invalid self cgroup")
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "0::/") {
		return "", errors.New("cgroup v2 required")
	}
	return filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(lines[0], "0::/")), nil
}

func openCgroupDirectory(parent int, name string, requireOwner bool) (*os.File, cgroupIdentity, error) {
	fd, err := unix.Openat2(parent, name, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, cgroupIdentity{}, err
	}
	file := os.NewFile(uintptr(fd), "orquesta-codex-cgroup")
	identity, err := inspectCgroupFD(fd, requireOwner)
	if err != nil {
		_ = file.Close()
		return nil, cgroupIdentity{}, err
	}
	return file, identity, nil
}

func inspectCgroupFD(fd int, requireOwner bool) (cgroupIdentity, error) {
	var fs unix.Statfs_t
	var stat unix.Stat_t
	if unix.Fstatfs(fd, &fs) != nil || fs.Type != unix.CGROUP2_SUPER_MAGIC ||
		unix.Fstat(fd, &stat) != nil || stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Mode&0o022 != 0 || (requireOwner && stat.Uid != uint32(os.Geteuid())) {
		return cgroupIdentity{}, errors.New("unsafe cgroup directory")
	}
	for _, control := range []string{"cgroup.procs", "cgroup.events"} {
		controlFD, err := unix.Openat(fd, control, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return cgroupIdentity{}, err
		}
		_ = unix.Close(controlFD)
	}
	killFD, err := unix.Openat(fd, "cgroup.kill", unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return cgroupIdentity{}, err
	}
	_ = unix.Close(killFD)
	return cgroupIdentity{Device: uint64(stat.Dev), Inode: stat.Ino}, nil
}

func preflightCgroupWrite(dir int, controls ...string) error {
	for _, control := range controls {
		fd, err := unix.Openat(dir, control, unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return err
		}
		if err := unix.Close(fd); err != nil {
			return err
		}
	}
	return nil
}

func newCgroupName() (string, error) {
	var material [32]byte
	if _, err := io.ReadFull(rand.Reader, material[:]); err != nil {
		return "", err
	}
	return "execution-" + hex.EncodeToString(material[:]), nil
}

func (root *codexCgroupRoot) newLeaf(name string) (*codexCgroupLeaf, error) {
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.closed || !validCgroupName(name) {
		return nil, cgroupError(CodeCgroupRootInvalid, nil)
	}
	if err := unix.Mkdirat(int(root.root.Fd()), name, 0o700); err != nil {
		return nil, cgroupError(CodeCgroupCreateFailed, err)
	}
	remove := true
	defer func() {
		if remove {
			_ = unix.Unlinkat(int(root.root.Fd()), name, unix.AT_REMOVEDIR)
		}
	}()
	file, identity, err := openCgroupDirectory(int(root.root.Fd()), name, true)
	if err != nil {
		return nil, cgroupError(CodeCgroupCreateFailed, err)
	}
	leaf := &codexCgroupLeaf{root: root, file: file, name: name, identity: identity}
	populated, err := cgroupPopulated(int(file.Fd()))
	if err != nil || populated {
		_ = file.Close()
		return nil, cgroupError(CodeCgroupCreateFailed, err)
	}
	remove = false
	return leaf, nil
}

func (leaf *codexCgroupLeaf) apply(command *os.Process) error {
	if leaf == nil || command == nil {
		return cgroupError(CodeCgroupCreateFailed, nil)
	}
	member, err := cgroupHasPID(int(leaf.file.Fd()), command.Pid)
	if err != nil || !member {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	return nil
}

func (leaf *codexCgroupLeaf) close() error {
	if leaf == nil || leaf.file == nil {
		return nil
	}
	err := leaf.file.Close()
	leaf.file = nil
	return err
}

func (leaf *codexCgroupLeaf) destroy(timeout time.Duration) error {
	if leaf == nil {
		return nil
	}
	var result error
	if leaf.file != nil {
		if err := writeCgroupControl(int(leaf.file.Fd()), "cgroup.kill", "1"); err != nil &&
			!errors.Is(err, unix.ENOENT) {
			result = errors.Join(result, err)
		}
		result = errors.Join(result, waitCgroupEmpty(leaf.file, timeout))
		result = errors.Join(result, leaf.close())
	}
	if leaf.root != nil && leaf.root.root != nil {
		result = errors.Join(result, unix.Unlinkat(int(leaf.root.root.Fd()), leaf.name, unix.AT_REMOVEDIR))
	}
	return result
}

func waitCgroupEmpty(file *os.File, timeout time.Duration) error {
	if file == nil || timeout <= 0 {
		return cgroupError(CodeCgroupDrainFailed, nil)
	}
	deadline := time.Now().Add(timeout)
	for {
		populated, err := cgroupPopulated(int(file.Fd()))
		if err != nil || !populated {
			return err
		}
		if !time.Now().Before(deadline) {
			return cgroupError(CodeCgroupDrainFailed, nil)
		}
		time.Sleep(time.Millisecond)
	}
}

func (root *codexCgroupRoot) populateRecord(record *processRecord, leaf *codexCgroupLeaf) {
	record.CgroupName = leaf.name
	record.CgroupRootDevice = root.rootIdentity.Device
	record.CgroupRootInode = root.rootIdentity.Inode
	record.CgroupControlDevice = root.controlIdentity.Device
	record.CgroupControlInode = root.controlIdentity.Inode
	record.CgroupDevice = leaf.identity.Device
	record.CgroupInode = leaf.identity.Inode
}

func (root *codexCgroupRoot) leafForRecord(record processRecord) (*os.File, error) {
	if root == nil || root.root == nil ||
		record.CgroupRootDevice != root.rootIdentity.Device ||
		record.CgroupRootInode != root.rootIdentity.Inode ||
		!validCgroupName(record.CgroupName) {
		return nil, cgroupError(CodeCgroupIdentityMismatch, nil)
	}
	file, identity, err := openCgroupDirectory(int(root.root.Fd()), record.CgroupName, true)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil, os.ErrNotExist
		}
		return nil, cgroupError(CodeCgroupIdentityMismatch, err)
	}
	if identity.Device != record.CgroupDevice || identity.Inode != record.CgroupInode {
		_ = file.Close()
		return nil, cgroupError(CodeCgroupIdentityMismatch, nil)
	}
	return file, nil
}

func validCgroupName(name string) bool {
	if !strings.HasPrefix(name, "execution-") || len(name) != len("execution-")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(name, "execution-"))
	return err == nil
}

func (root *codexCgroupRoot) populated(record processRecord) (bool, error) {
	file, err := root.leafForRecord(record)
	if err != nil {
		return false, cgroupError(CodeCgroupIdentityMismatch, err)
	}
	defer file.Close()
	return cgroupPopulated(int(file.Fd()))
}

func (root *codexCgroupRoot) remove(record processRecord) error {
	file, err := root.leafForRecord(record)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	populated, observeErr := cgroupPopulated(int(file.Fd()))
	closeErr := file.Close()
	if observeErr != nil || populated {
		return errors.Join(observeErr, closeErr, cgroupError(CodeCgroupDrainFailed, nil))
	}
	if closeErr != nil {
		return closeErr
	}
	if err := unix.Unlinkat(int(root.root.Fd()), record.CgroupName, unix.AT_REMOVEDIR); err != nil &&
		!errors.Is(err, unix.ENOENT) {
		return cgroupError(CodeCgroupDrainFailed, err)
	}
	return nil
}

func (root *codexCgroupRoot) drain(record processRecord, timeout time.Duration) error {
	file, err := root.leafForRecord(record)
	if err != nil {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	defer file.Close()
	if err := writeCgroupControl(int(file.Fd()), "cgroup.kill", "1"); err != nil {
		return cgroupError(CodeCgroupDrainFailed, err)
	}
	return waitCgroupEmpty(file, timeout)
}

func (root *codexCgroupRoot) kill(record processRecord) error {
	file, err := root.leafForRecord(record)
	if err != nil {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	defer file.Close()
	if err := writeCgroupControl(int(file.Fd()), "cgroup.kill", "1"); err != nil {
		return cgroupError(CodeCgroupDrainFailed, err)
	}
	return nil
}

func (root *codexCgroupRoot) close() error {
	if root == nil {
		return nil
	}
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.closed {
		return nil
	}
	root.closed = true
	return errors.Join(root.control.Close(), root.root.Close())
}

func duplicateCgroupFile(file *os.File, name string) (*os.File, error) {
	fd, err := unix.FcntlInt(file.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func validateSupervisorCgroups(record processRecord, control, work *os.File) error {
	controlIdentity, err := inspectCgroupFD(int(control.Fd()), false)
	if err != nil || controlIdentity.Device != record.CgroupControlDevice ||
		controlIdentity.Inode != record.CgroupControlInode {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	workIdentity, err := inspectCgroupFD(int(work.Fd()), true)
	if err != nil || workIdentity.Device != record.CgroupDevice || workIdentity.Inode != record.CgroupInode {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	member, err := cgroupHasPID(int(work.Fd()), os.Getpid())
	if err != nil || !member {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	return nil
}

func moveSupervisorToControl(record processRecord, control, work *os.File) error {
	if err := writeCgroupControl(int(control.Fd()), "cgroup.procs", strconv.Itoa(os.Getpid())); err != nil {
		return cgroupError(CodeCgroupDrainFailed, err)
	}
	member, err := cgroupHasPID(int(control.Fd()), os.Getpid())
	if err != nil || !member {
		return cgroupError(CodeCgroupIdentityMismatch, err)
	}
	populated, err := cgroupPopulated(int(work.Fd()))
	if err != nil || populated {
		return cgroupError(CodeCgroupDrainFailed, err)
	}
	return nil
}

func cgroupHasPID(dir, pid int) (bool, error) {
	content, err := readCgroupControl(dir, "cgroup.procs")
	if err != nil {
		return false, err
	}
	target := strconv.Itoa(pid)
	for _, value := range strings.Fields(content) {
		if value == target {
			return true, nil
		}
	}
	return false, nil
}

func cgroupProcessesEmpty(dir int) (bool, error) {
	content, err := readCgroupControl(dir, "cgroup.procs")
	return strings.TrimSpace(content) == "", err
}

func cgroupPopulated(dir int) (bool, error) {
	content, err := readCgroupControl(dir, "cgroup.events")
	if err != nil {
		return false, err
	}
	fields := strings.Fields(content)
	for index := 0; index+1 < len(fields); index += 2 {
		if fields[index] == "populated" {
			switch fields[index+1] {
			case "0":
				return false, nil
			case "1":
				return true, nil
			default:
				return false, errors.New("invalid populated event")
			}
		}
	}
	return false, errors.New("missing populated event")
}

func readCgroupControl(dir int, name string) (string, error) {
	fd, err := unix.Openat(dir, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", err
	}
	file := os.NewFile(uintptr(fd), "orquesta-cgroup-control")
	payload, readErr := io.ReadAll(io.LimitReader(file, 4097))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(payload) > 4096 {
		return "", errors.Join(readErr, closeErr)
	}
	return strings.TrimSpace(string(payload)), nil
}

func writeCgroupControl(dir int, name, value string) error {
	fd, err := unix.Openat(dir, name, unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	written, err := unix.Write(fd, []byte(value))
	if err != nil || written != len(value) {
		return errors.Join(err, io.ErrShortWrite)
	}
	return nil
}

func cgroupError(code string, cause error) error {
	return &Error{Code: code, Cause: cause}
}

func (adapter *Adapter) allocateExecutionCgroup(runPath string) (*codexCgroupLeaf, error) {
	if adapter == nil || adapter.cgroups == nil {
		return nil, cgroupError(CodeCgroupRootRequired, nil)
	}
	if err := adapter.cleanupOrphanedCgroup(runPath, adapter.config.SupervisorStartTimeout); err != nil {
		return nil, err
	}
	name, err := newCgroupName()
	if err != nil {
		return nil, cgroupError(CodeCgroupCreateFailed, err)
	}
	allocation := cgroupAllocationRecord{
		SchemaVersion: cgroupJournalSchema, Name: name,
		RootDevice: adapter.cgroups.rootIdentity.Device,
		RootInode:  adapter.cgroups.rootIdentity.Inode,
	}
	created, err := adapter.publishJSON(runPath, cgroupAllocationFileName, allocation)
	if err != nil || !created {
		return nil, errors.Join(err, cgroupError(CodeCgroupCreateFailed, nil))
	}
	leaf, err := adapter.cgroups.newLeaf(name)
	if err != nil {
		return nil, err
	}
	boundary := cgroupBoundaryRecord{
		SchemaVersion: cgroupJournalSchema, Name: name,
		RootDevice: adapter.cgroups.rootIdentity.Device, RootInode: adapter.cgroups.rootIdentity.Inode,
		ControlDevice: adapter.cgroups.controlIdentity.Device, ControlInode: adapter.cgroups.controlIdentity.Inode,
		Device: leaf.identity.Device, Inode: leaf.identity.Inode,
	}
	created, err = adapter.publishJSON(runPath, cgroupBoundaryFileName, boundary)
	if err != nil || !created {
		_ = leaf.destroy(adapter.config.SupervisorStartTimeout)
		return nil, errors.Join(err, cgroupError(CodeCgroupCreateFailed, nil))
	}
	return leaf, nil
}

func (adapter *Adapter) cleanupOrphanedCgroup(runPath string, timeout time.Duration) error {
	if adapter == nil || adapter.cgroups == nil {
		return nil
	}
	if adapter.beforeCgroupCleanup != nil {
		if err := adapter.beforeCgroupCleanup(runPath); err != nil {
			return err
		}
	}
	var allocation cgroupAllocationRecord
	found, err := adapter.readPrivateJSON(
		filepath.ToSlash(filepath.Join(runPath, cgroupAllocationFileName)), &allocation,
	)
	if err != nil || !found {
		return err
	}
	if allocation.SchemaVersion != cgroupJournalSchema || !validCgroupName(allocation.Name) ||
		allocation.RootDevice != adapter.cgroups.rootIdentity.Device ||
		allocation.RootInode != adapter.cgroups.rootIdentity.Inode {
		return cgroupError(CodeCgroupIdentityMismatch, nil)
	}
	var boundary cgroupBoundaryRecord
	boundaryFound, err := adapter.readPrivateJSON(
		filepath.ToSlash(filepath.Join(runPath, cgroupBoundaryFileName)), &boundary,
	)
	if err != nil {
		return err
	}
	if boundaryFound {
		if boundary.SchemaVersion != cgroupJournalSchema || boundary.Name != allocation.Name ||
			boundary.RootDevice != allocation.RootDevice || boundary.RootInode != allocation.RootInode ||
			boundary.ControlDevice != adapter.cgroups.controlIdentity.Device ||
			boundary.ControlInode != adapter.cgroups.controlIdentity.Inode ||
			boundary.Device == 0 || boundary.Inode == 0 {
			return cgroupError(CodeCgroupIdentityMismatch, nil)
		}
		record := processRecord{
			SchemaVersion: cgroupProcessSchemaVersion, CgroupName: boundary.Name,
			CgroupRootDevice: boundary.RootDevice, CgroupRootInode: boundary.RootInode,
			CgroupControlDevice: boundary.ControlDevice, CgroupControlInode: boundary.ControlInode,
			CgroupDevice: boundary.Device, CgroupInode: boundary.Inode,
		}
		file, openErr := adapter.cgroups.leafForRecord(record)
		if openErr == nil {
			if killErr := writeCgroupControl(int(file.Fd()), "cgroup.kill", "1"); killErr != nil {
				_ = file.Close()
				return cgroupError(CodeCgroupDrainFailed, killErr)
			}
			if waitErr := waitCgroupEmpty(file, timeout); waitErr != nil {
				_ = file.Close()
				return waitErr
			}
			_ = file.Close()
			if removeErr := adapter.cgroups.remove(record); removeErr != nil {
				return removeErr
			}
		} else if !errors.Is(openErr, os.ErrNotExist) {
			return openErr
		}
	} else {
		file, _, openErr := openCgroupDirectoryBeneath(
			int(adapter.cgroups.root.Fd()), allocation.Name, true,
		)
		if openErr == nil {
			populated, observeErr := cgroupPopulated(int(file.Fd()))
			_ = file.Close()
			if observeErr != nil || populated {
				return errors.Join(observeErr, cgroupError(CodeCgroupIdentityMismatch, nil))
			}
			if removeErr := unix.Unlinkat(
				int(adapter.cgroups.root.Fd()), allocation.Name, unix.AT_REMOVEDIR,
			); removeErr != nil {
				return cgroupError(CodeCgroupDrainFailed, removeErr)
			}
		} else if !errors.Is(openErr, unix.ENOENT) {
			return cgroupError(CodeCgroupIdentityMismatch, openErr)
		}
	}
	for _, name := range []string{cgroupBoundaryFileName, cgroupAllocationFileName} {
		if err := adapter.root.Remove(filepath.ToSlash(filepath.Join(runPath, name))); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return adapter.syncDirectoryCausally(runPath)
}

func (adapter *Adapter) cleanupTerminalCgroup(state *executionState) error {
	if state == nil || adapter == nil || adapter.cgroups == nil {
		return nil
	}
	return adapter.cleanupOrphanedCgroup(state.runPath, adapter.config.SupervisorStartTimeout)
}
