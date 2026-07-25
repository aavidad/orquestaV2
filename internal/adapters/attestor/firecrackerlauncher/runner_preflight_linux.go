//go:build linux

package firecrackerlauncher

import (
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

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
	fd, err := unix.Openat2(unix.AT_FDCWD, "/dev/kvm", &unix.OpenHow{
		Flags:   unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return launcherError(CodeUnavailable)
	}
	defer unix.Close(fd)
	var stat unix.Stat_t
	version, ioctlErr := unix.IoctlGetInt(fd, kvmGetAPIVersionIOCTL)
	if unix.Fstat(fd, &stat) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFCHR ||
		stat.Uid != 0 || stat.Nlink != 1 ||
		stat.Mode&0o777 != 0o660 ||
		version != kvmAPIVersion || ioctlErr != nil {
		return launcherError(CodeUnavailable)
	}
	return nil
}
