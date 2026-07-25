//go:build linux

package firecrackerlauncher

import (
	"errors"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

const maxKVMInterruptedSyscallAttempts = 8

type kvmPreflightSyscalls struct {
	open       func() (int, error)
	fstat      func(int, *unix.Stat_t) error
	apiVersion func(int) (int, error)
	close      func(int) error
}

func newPhysicalRunner(config Config) (*physicalRunner, error) {
	if validateConfig(config) != nil {
		return nil, launcherError(CodeConfigInvalid)
	}
	if err := preflightKVM(); err != nil {
		return nil, err
	}
	assets, err := openAssetSet(config, 0)
	if err != nil {
		return nil, err
	}
	namespace, err := openEmptyNetNamespace(
		config.NetNSPath,
		0,
		config.CleanupTimeout,
	)
	if err != nil {
		_ = assets.Close()
		return nil, err
	}
	cgroups, err := openCgroupParent(config, 0)
	if err != nil {
		_ = namespace.Close()
		_ = assets.Close()
		return nil, err
	}
	runtimeRoot, err := openTrustedRuntimeRoot(config.RuntimeRoot, 0, config.AllowedGID)
	if err != nil {
		_ = cgroups.Close()
		_ = namespace.Close()
		_ = assets.Close()
		return nil, err
	}
	runs, err := openRunsRoot(runtimeRoot, config.RuntimeRoot, 0)
	if err != nil {
		_ = runtimeRoot.Close()
		_ = cgroups.Close()
		_ = namespace.Close()
		_ = assets.Close()
		return nil, err
	}
	return newPhysicalRunnerWithDependencies(
		config, assets, namespace, cgroups, runtimeRoot, runs, exec.Command,
	), nil
}

func newPhysicalRunnerWithDependencies(
	config Config,
	assets *assetSet,
	namespace runnerNetNamespace,
	cgroups runnerCgroup,
	runtimeRoot *os.File,
	runs *runsRoot,
	command launcherCommandFactory,
) *physicalRunner {
	return &physicalRunner{
		config: config, assets: assets, namespace: namespace, cgroups: cgroups,
		runtimeRoot: runtimeRoot, runs: runs, command: command,
		active: make(map[*activePhysicalRun]struct{}),
	}
}

func preflightKVM() error {
	return preflightKVMWithSyscalls(kvmPreflightSyscalls{
		open: func() (int, error) {
			return unix.Openat2(unix.AT_FDCWD, "/dev/kvm", &unix.OpenHow{
				Flags:   unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW,
				Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
			})
		},
		fstat: unix.Fstat,
		apiVersion: func(fd int) (int, error) {
			return unix.IoctlGetInt(fd, kvmGetAPIVersionIOCTL)
		},
		close: unix.Close,
	})
}

func preflightKVMWithSyscalls(syscalls kvmPreflightSyscalls) error {
	if syscalls.open == nil || syscalls.fstat == nil ||
		syscalls.apiVersion == nil || syscalls.close == nil {
		return launcherError(CodeKVMOpenUnavailable)
	}
	fd, err := retryKVMOpen(syscalls.open)
	if err != nil {
		return launcherError(CodeKVMOpenUnavailable)
	}
	defer syscalls.close(fd)
	var stat unix.Stat_t
	if retryKVMFstat(fd, &stat, syscalls.fstat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFCHR ||
		stat.Uid != 0 || stat.Nlink != 1 ||
		stat.Mode&0o777 != 0o660 {
		return launcherError(CodeKVMMetadataUnsafe)
	}
	version, err := retryKVMAPIVersion(fd, syscalls.apiVersion)
	if err != nil {
		return launcherError(CodeKVMAPIUnavailable)
	}
	if version != kvmAPIVersion {
		return launcherError(CodeKVMVersionUnsupported)
	}
	return nil
}

func retryKVMOpen(open func() (int, error)) (int, error) {
	var err error
	for range maxKVMInterruptedSyscallAttempts {
		var fd int
		fd, err = open()
		if !errors.Is(err, unix.EINTR) {
			return fd, err
		}
	}
	return -1, err
}

func retryKVMFstat(
	fd int,
	stat *unix.Stat_t,
	fstat func(int, *unix.Stat_t) error,
) error {
	var err error
	for range maxKVMInterruptedSyscallAttempts {
		err = fstat(fd, stat)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
	return err
}

func retryKVMAPIVersion(
	fd int,
	apiVersion func(int) (int, error),
) (int, error) {
	var (
		version int
		err     error
	)
	for range maxKVMInterruptedSyscallAttempts {
		version, err = apiVersion(fd)
		if !errors.Is(err, unix.EINTR) {
			return version, err
		}
	}
	return 0, err
}
