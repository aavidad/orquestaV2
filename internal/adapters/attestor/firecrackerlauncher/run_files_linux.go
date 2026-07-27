//go:build linux

package firecrackerlauncher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	runsDirectoryName   = "runs"
	runMarkerName       = ".orquesta-firecracker-run"
	runMarkerSchema     = "orquesta_firecracker_launcher_run.v1"
	firecrackerConfig   = "firecracker.json"
	kernelJailName      = "vmlinux"
	guestJailName       = "guest.cpio.gz"
	inputDriveJailName  = "input.drive"
	outputDriveJailName = "output.drive"
)

type runsRoot struct {
	file  *os.File
	path  string
	owner uint32
}

type runWorkspace struct {
	runs            *runsRoot
	directory       *os.File
	identity        descriptorIdentity
	id              string
	path            string
	marker          string
	jailerPath      string
	firecrackerPath string
	output          *os.File
	outputIdentity  unix.Stat_t
	cleanupTimeout  time.Duration
	cleanupEntries  uint32
	cleanupDepth    uint32
	cleanup         sync.Once
	cleanupErr      error
}

type firecrackerConfiguration struct {
	BootSource struct {
		KernelImagePath string `json:"kernel_image_path"`
		InitrdPath      string `json:"initrd_path"`
		BootArgs        string `json:"boot_args"`
	} `json:"boot-source"`
	Drives  []firecrackerDrive `json:"drives"`
	Machine struct {
		VCPUCount  uint32 `json:"vcpu_count"`
		MemoryMiB  uint32 `json:"mem_size_mib"`
		SMT        bool   `json:"smt"`
		TrackDirty bool   `json:"track_dirty_pages"`
		HugePages  string `json:"huge_pages"`
	} `json:"machine-config"`
	NetworkInterfaces []struct{} `json:"network-interfaces"`
	Vsock             any        `json:"vsock,omitempty"`
}

type firecrackerDrive struct {
	DriveID    string `json:"drive_id"`
	PathOnHost string `json:"path_on_host"`
	IsRoot     bool   `json:"is_root_device"`
	IsReadOnly bool   `json:"is_read_only"`
	CacheType  string `json:"cache_type"`
	IOEngine   string `json:"io_engine"`
}

func openRunsRoot(runtimeRoot *os.File, runtimePath string, owner uint32) (*runsRoot, error) {
	if runtimeRoot == nil || !canonicalAbsolute(runtimePath) {
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	if err := unix.Mkdirat(int(runtimeRoot.Fd()), runsDirectoryName, 0o700); err != nil &&
		!errors.Is(err, unix.EEXIST) {
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	fd, err := unix.Openat2(int(runtimeRoot.Fd()), runsDirectoryName, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-runs-root")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Uid != owner || stat.Gid != owner ||
		stat.Mode&0o777 != 0o700 {
		_ = file.Close()
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	entries, err := file.ReadDir(1)
	if err != nil && !errors.Is(err, io.EOF) || len(entries) != 0 {
		_ = file.Close()
		return nil, launcherError(CodeCleanupFailed)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, launcherError(CodeRuntimeRootUnsafe)
	}
	return &runsRoot{
		file: file, path: filepath.Join(runtimePath, runsDirectoryName), owner: owner,
	}, nil
}

func runID(nonce string) (string, error) {
	raw, err := hex.DecodeString(nonce)
	if err != nil || len(raw) != sha256.Size {
		return "", launcherError(CodeProtocolInvalid)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	id := "orq-" + string(bytes.ToLower([]byte(encoded)))
	if !validJailID(id) {
		return "", launcherError(CodeProtocolInvalid)
	}
	return id, nil
}

func validJailID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, character := range id {
		if character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '-' {
			continue
		}
		return false
	}
	return true
}

func createRunWorkspace(
	ctx context.Context,
	runs *runsRoot,
	config Config,
	assets *assetSet,
	request LaunchRequest,
	input *os.File,
) (*runWorkspace, error) {
	id, err := runID(request.Nonce)
	if err != nil || runs == nil || runs.file == nil || assets == nil {
		return nil, launcherError(CodeExecutionFailed)
	}
	if err := unix.Mkdirat(int(runs.file.Fd()), id, 0o700); err != nil {
		return nil, launcherError(CodeExecutionFailed)
	}
	fd, err := unix.Openat2(int(runs.file.Fd()), id, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	workspace := &runWorkspace{
		runs: runs, id: id, path: filepath.Join(runs.path, id),
		cleanupTimeout: config.CleanupTimeout,
		cleanupEntries: config.MaxCleanupEntries,
		cleanupDepth:   config.MaxCleanupDepth,
	}
	if err != nil {
		_ = unix.Unlinkat(int(runs.file.Fd()), id, unix.AT_REMOVEDIR)
		return nil, launcherError(CodeExecutionFailed)
	}
	workspace.directory = os.NewFile(uintptr(fd), "orquesta-firecracker-run")
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		stat.Uid != runs.owner || stat.Gid != runs.owner ||
		stat.Mode&0o777 != 0o700 {
		_ = workspace.Cleanup()
		return nil, launcherError(CodeExecutionFailed)
	}
	workspace.identity = descriptorIdentity{
		device: uint64(stat.Dev), inode: stat.Ino, mode: stat.Mode,
	}
	workspace.marker = fmt.Sprintf(
		"schema=%s\nid=%s\nnonce=%s\ncgroup=%s/%s\n",
		runMarkerSchema,
		id,
		request.Nonce,
		config.ParentCgroup,
		id,
	)
	if err := workspace.populate(ctx, config, assets, request, input); err != nil {
		cleanupErr := workspace.Cleanup()
		if cleanupErr != nil {
			return nil, cleanupErr
		}
		return nil, err
	}
	return workspace, nil
}

func (workspace *runWorkspace) populate(
	ctx context.Context,
	config Config,
	assets *assetSet,
	request LaunchRequest,
	input *os.File,
) error {
	if err := createBytesAt(
		workspace.directory,
		runMarkerName,
		[]byte(workspace.marker),
		0o400,
		workspace.runs.owner,
		workspace.runs.owner,
	); err != nil {
		return err
	}
	for _, directory := range []string{
		"jail",
		"jail/firecracker",
		"jail/firecracker/" + workspace.id,
		"jail/firecracker/" + workspace.id + "/root",
	} {
		if err := mkdirBeneath(
			workspace.directory,
			directory,
			0o700,
			workspace.runs.owner,
		); err != nil {
			return err
		}
	}
	// Jailer 1.16.1 accepts this pre-existing root and chowns it to JailUID:JailGID
	// before dropping privileges. Keeping it root-only until then prevents the
	// jailed uid from replacing files while the launcher populates the jail.
	if err := copyAssetAt(
		ctx,
		workspace.directory,
		"jailer",
		assets.jailer,
		0o500,
		workspace.runs.owner,
		workspace.runs.owner,
	); err != nil {
		return err
	}
	// Jailer bind-mounts this source and executes it only after dropping to
	// JailUID:JailGID. The exact jail group needs execute permission, while
	// neither the jail identity nor unrelated users may modify or execute it.
	if err := copyAssetAt(
		ctx,
		workspace.directory,
		"firecracker",
		assets.firecracker,
		0o550,
		workspace.runs.owner,
		config.JailGID,
	); err != nil {
		return err
	}
	workspace.jailerPath = filepath.Join(workspace.path, "jailer")
	workspace.firecrackerPath = filepath.Join(workspace.path, "firecracker")
	root, err := workspace.openJailRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	for _, specification := range []struct {
		name  string
		asset *pinnedAsset
	}{
		{kernelJailName, assets.kernel},
		{guestJailName, assets.guest},
	} {
		if err := copyAssetAt(
			ctx,
			root,
			specification.name,
			specification.asset,
			0o400,
			config.JailUID,
			config.JailGID,
		); err != nil {
			return err
		}
	}
	if err := copyInputAt(
		ctx,
		root,
		inputDriveJailName,
		input,
		request.InputDigest,
		config.JailUID,
		config.JailGID,
	); err != nil {
		return err
	}
	output, identity, err := createOutputAt(
		root,
		outputDriveJailName,
		request.OutputDriveBytes,
		config.JailUID,
		config.JailGID,
	)
	if err != nil {
		return err
	}
	workspace.output, workspace.outputIdentity = output, identity
	configuration, err := buildFirecrackerConfig(request)
	if err != nil {
		return launcherError(CodeExecutionFailed)
	}
	if err := createBytesAt(
		root,
		firecrackerConfig,
		configuration,
		0o400,
		config.JailUID,
		config.JailGID,
	); err != nil {
		return err
	}
	if root.Sync() != nil || workspace.directory.Sync() != nil {
		return launcherError(CodeExecutionFailed)
	}
	return nil
}

func buildFirecrackerConfig(request LaunchRequest) ([]byte, error) {
	var config firecrackerConfiguration
	config.BootSource.KernelImagePath = "/" + kernelJailName
	config.BootSource.InitrdPath = "/" + guestJailName
	config.BootSource.BootArgs = "8250.nr_uarts=0 reboot=k panic=1 pci=off init=/init"
	config.Drives = []firecrackerDrive{
		{
			DriveID: "orquesta-input", PathOnHost: "/" + inputDriveJailName,
			IsReadOnly: true, CacheType: "Unsafe", IOEngine: "Sync",
		},
		{
			DriveID: "orquesta-output", PathOnHost: "/" + outputDriveJailName,
			IsReadOnly: false, CacheType: "Unsafe", IOEngine: "Sync",
		},
	}
	config.Machine.VCPUCount = uint32(
		(request.CPUQuotaMicros + request.CPUPeriodMicros - 1) /
			request.CPUPeriodMicros,
	)
	config.Machine.MemoryMiB = request.GuestMemoryMiB
	config.Machine.HugePages = "None"
	config.NetworkInterfaces = []struct{}{}
	config.Vsock = nil
	payload, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}

func (workspace *runWorkspace) openJailRoot() (*os.File, error) {
	relative := filepath.Join("jail", "firecracker", workspace.id, "root")
	fd, err := unix.Openat2(int(workspace.directory.Fd()), relative, &unix.OpenHow{
		Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS |
			unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeExecutionFailed)
	}
	return os.NewFile(uintptr(fd), "orquesta-firecracker-jail-root"), nil
}

func mkdirBeneath(root *os.File, relative string, mode, owner uint32) error {
	parent := root
	parts := bytes.Split([]byte(filepath.ToSlash(relative)), []byte{'/'})
	var opened []*os.File
	defer func() {
		for _, file := range opened {
			_ = file.Close()
		}
	}()
	for _, raw := range parts {
		name := string(raw)
		if name == "" || name == "." || name == ".." {
			return launcherError(CodeExecutionFailed)
		}
		if err := unix.Mkdirat(int(parent.Fd()), name, mode); err != nil &&
			!errors.Is(err, unix.EEXIST) {
			return launcherError(CodeExecutionFailed)
		}
		fd, err := unix.Openat(
			int(parent.Fd()),
			name,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
			0,
		)
		if err != nil {
			return launcherError(CodeExecutionFailed)
		}
		file := os.NewFile(uintptr(fd), "orquesta-firecracker-run-directory")
		var stat unix.Stat_t
		if unix.Fstat(fd, &stat) != nil ||
			stat.Mode&unix.S_IFMT != unix.S_IFDIR ||
			stat.Uid != owner || stat.Gid != owner ||
			stat.Mode&0o777 != mode {
			_ = file.Close()
			return launcherError(CodeExecutionFailed)
		}
		opened = append(opened, file)
		parent = file
	}
	return nil
}
