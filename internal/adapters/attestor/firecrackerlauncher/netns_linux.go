//go:build linux

package firecrackerlauncher

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const netNamespaceProbeArgument = "--orquesta-firecracker-probe-empty-netns"

type pinnedNetNamespace struct {
	file     *os.File
	identity descriptorIdentity
	path     string
	owner    uint32
	timeout  time.Duration
}

func init() {
	if len(os.Args) == 2 && os.Args[1] == netNamespaceProbeArgument {
		os.Exit(netNamespaceProbeMain())
	}
}

func openEmptyNetNamespace(
	path string,
	owner uint32,
	timeout time.Duration,
) (*pinnedNetNamespace, error) {
	if timeout <= 0 {
		return nil, launcherError(CodeNetworkUnsafe)
	}
	if validateTrustedAncestors(path, owner) != nil {
		return nil, launcherError(CodeNetworkUnsafe)
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, launcherError(CodeNetworkUnsafe)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-empty-netns")
	var stat unix.Stat_t
	var filesystem unix.Statfs_t
	if unix.Fstat(fd, &stat) != nil || unix.Fstatfs(fd, &filesystem) != nil ||
		filesystem.Type != unix.NSFS_MAGIC ||
		stat.Mode&unix.S_IFMT != unix.S_IFREG ||
		stat.Uid != owner || stat.Gid != owner || stat.Nlink != 1 ||
		stat.Mode&0o777 != 0o600 {
		_ = file.Close()
		return nil, launcherError(CodeNetworkUnsafe)
	}
	namespace := &pinnedNetNamespace{
		file:    file,
		path:    path,
		owner:   owner,
		timeout: timeout,
		identity: descriptorIdentity{
			device: uint64(stat.Dev), inode: stat.Ino, mode: stat.Mode,
		},
	}
	if err := probeEmptyNetNamespace(file, timeout); err != nil {
		_ = file.Close()
		return nil, launcherError(CodeNetworkUnsafe)
	}
	return namespace, nil
}

func probeEmptyNetNamespace(target *os.File, timeout time.Duration) error {
	if target == nil {
		return launcherError(CodeNetworkUnsafe)
	}
	originalFD, err := unix.Open(
		"/proc/self/ns/net",
		unix.O_RDONLY|unix.O_CLOEXEC,
		0,
	)
	if err != nil {
		return launcherError(CodeNetworkUnsafe)
	}
	defer unix.Close(originalFD)
	if sameNamespaceFD(originalFD, int(target.Fd())) {
		return launcherError(CodeNetworkUnsafe)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "/proc/self/exe", netNamespaceProbeArgument)
	command.Env = []string{}
	command.ExtraFiles = []*os.File{target}
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, Pdeathsig: syscall.SIGKILL,
	}
	if command.Run() != nil || ctx.Err() != nil {
		return launcherError(CodeNetworkUnsafe)
	}
	return nil
}

func netNamespaceProbeMain() int {
	if len(os.Args) != 2 {
		return 125
	}
	target := os.NewFile(3, "orquesta-firecracker-probe-netns")
	if target == nil {
		return 125
	}
	defer target.Close()
	runtime.LockOSThread()
	// Deliberately do not unlock: this subprocess exits immediately, so a
	// namespace-bound thread can never return to a long-lived Go thread pool.
	if unix.Setns(int(target.Fd()), unix.CLONE_NEWNET) != nil {
		return 125
	}
	if verifyCurrentEmptyNetNamespace() != nil {
		return 125
	}
	return 0
}

func verifyCurrentEmptyNetNamespace() error {
	interfaces, err := net.Interfaces()
	if err != nil || len(interfaces) != 1 ||
		interfaces[0].Name != "lo" ||
		interfaces[0].Flags&net.FlagLoopback == 0 ||
		interfaces[0].Flags&net.FlagUp != 0 {
		return launcherError(CodeNetworkUnsafe)
	}
	addresses, err := interfaces[0].Addrs()
	if err != nil || len(addresses) != 0 ||
		!ipv4RoutesEmpty() || !procFileEmpty("/proc/net/ipv6_route") ||
		!procFileEmpty("/proc/net/if_inet6") {
		return launcherError(CodeNetworkUnsafe)
	}
	return nil
}

func sameNamespaceFD(left, right int) bool {
	var leftStat, rightStat unix.Stat_t
	return unix.Fstat(left, &leftStat) == nil &&
		unix.Fstat(right, &rightStat) == nil &&
		leftStat.Dev == rightStat.Dev && leftStat.Ino == rightStat.Ino
}

func ipv4RoutesEmpty() bool {
	content, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return false
	}
	lines := bytes.Split(bytes.TrimSpace(content), []byte{'\n'})
	return len(lines) == 1 && bytes.HasPrefix(lines[0], []byte("Iface"))
}

func procFileEmpty(path string) bool {
	content, err := os.ReadFile(path)
	return err == nil && len(bytes.TrimSpace(content)) == 0
}

func (namespace *pinnedNetNamespace) revalidate() error {
	if namespace == nil || namespace.file == nil {
		return launcherError(CodeNetworkUnsafe)
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, namespace.path, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return launcherError(CodeNetworkUnsafe)
	}
	current := os.NewFile(uintptr(fd), "orquesta-firecracker-current-empty-netns")
	defer current.Close()
	var stat unix.Stat_t
	var filesystem unix.Statfs_t
	if unix.Fstat(fd, &stat) != nil ||
		unix.Fstatfs(fd, &filesystem) != nil ||
		filesystem.Type != unix.NSFS_MAGIC ||
		uint64(stat.Dev) != namespace.identity.device ||
		stat.Ino != namespace.identity.inode ||
		stat.Mode != namespace.identity.mode ||
		stat.Uid != namespace.owner || stat.Gid != namespace.owner ||
		stat.Nlink != 1 {
		return launcherError(CodeNetworkUnsafe)
	}
	return probeEmptyNetNamespace(current, namespace.timeout)
}

func (namespace *pinnedNetNamespace) Close() error {
	if namespace == nil || namespace.file == nil {
		return nil
	}
	return namespace.file.Close()
}
