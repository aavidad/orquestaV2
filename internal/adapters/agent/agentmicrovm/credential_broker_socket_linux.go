//go:build linux

package agentmicrovm

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

func openCredentialBrokerSocket(config CredentialBrokerServerConfig) (*net.UnixListener, *os.File, brokerSocketIdentity, error) {
	return openCredentialBrokerSocketWithHooks(config, credentialBrokerSocketHooks{})
}

// credentialBrokerSocketHooks is an instance-scoped fault seam. Production
// always supplies the zero value; tests can replace the path at the only
// security-relevant frontier without a process-global hook or timing race.
type credentialBrokerSocketHooks struct {
	afterBindBeforeIdentity   func(string) error
	afterCandidateBeforeProbe func(string) error
}

func openCredentialBrokerSocketWithHooks(
	config CredentialBrokerServerConfig,
	hooks credentialBrokerSocketHooks,
) (*net.UnixListener, *os.File, brokerSocketIdentity, error) {
	rootFD, err := unix.Openat2(unix.AT_FDCWD, filepath.Dir(config.SocketPath), &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK,
		Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, nil, brokerSocketIdentity{}, fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	root := os.NewFile(uintptr(rootFD), "orquesta-credential-broker-root")
	failOpen := func(code string) (*net.UnixListener, *os.File, brokerSocketIdentity, error) {
		_ = root.Close()
		return nil, nil, brokerSocketIdentity{}, fail(code, nil)
	}
	var rootStat unix.Stat_t
	if unix.Fstat(rootFD, &rootStat) != nil || rootStat.Mode&unix.S_IFMT != unix.S_IFDIR ||
		rootStat.Uid != config.OwnerUID || rootStat.Mode&0o700 != 0o700 || rootStat.Mode&0o077 != 0 {
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	name := filepath.Base(config.SocketPath)
	var existing unix.Stat_t
	// Existing sockets also fail closed. Stale recovery after an unclean host
	// stop belongs to an explicit pre-start cleanup with its own ownership and
	// liveness proof; this listener never guesses that a pathname is orphaned.
	if err := unix.Fstatat(rootFD, name, &existing, unix.AT_SYMLINK_NOFOLLOW); !errors.Is(err, unix.ENOENT) {
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	socket, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK, 0)
	if err != nil {
		return failOpen(CodeCredentialBrokerServerUnavailable)
	}
	closeSocket := func() { _ = unix.Close(socket) }
	if err := unix.Bind(socket, &unix.SockaddrUnix{Name: config.SocketPath}); err != nil {
		closeSocket()
		return failOpen(CodeCredentialBrokerServerUnavailable)
	}
	if unix.Fchmodat(rootFD, name, 0o600, 0) != nil || unix.Listen(socket, int(config.MaxConcurrent)) != nil {
		// The pathname has not yet been proven to reach this descriptor. Closing
		// without unlinking is safer than deleting a concurrent replacement;
		// explicit pre-start cleanup governs the possible residue.
		closeSocket()
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	if hooks.afterBindBeforeIdentity != nil && hooks.afterBindBeforeIdentity(config.SocketPath) != nil {
		closeSocket()
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	candidate, err := credentialBrokerBoundSocketIdentity(socket, rootFD, name, config.SocketPath, config.OwnerUID, true)
	if err != nil {
		closeSocket()
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	if hooks.afterCandidateBeforeProbe != nil && hooks.afterCandidateBeforeProbe(config.SocketPath) != nil {
		closeSocket()
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	if err := probeCredentialBrokerBoundSocket(socket, config.SocketPath, config.ConnectionTimeout); err != nil {
		closeSocket()
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	verified, err := credentialBrokerBoundSocketIdentity(socket, rootFD, name, config.SocketPath, config.OwnerUID, true)
	if err != nil || verified != candidate {
		closeSocket()
		return failOpen(CodeCredentialBrokerSocketUnsafe)
	}
	// Only the self-connect accepted by this exact listener promotes the path
	// identity to cleanup authority.
	identity := candidate
	file := os.NewFile(uintptr(socket), "orquesta-credential-broker-listener")
	listenerValue, err := net.FileListener(file)
	_ = file.Close()
	if err != nil {
		_ = removeCredentialBrokerSocket(root, name, identity)
		return failOpen(CodeCredentialBrokerServerUnavailable)
	}
	listener, ok := listenerValue.(*net.UnixListener)
	if !ok {
		_ = listenerValue.Close()
		_ = removeCredentialBrokerSocket(root, name, identity)
		return failOpen(CodeCredentialBrokerServerUnavailable)
	}
	listener.SetUnlinkOnClose(false)
	return listener, root, identity, nil
}

func probeCredentialBrokerBoundSocket(socket int, path string, timeout time.Duration) error {
	probe, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	defer probe.Close()
	accepted, _, err := unix.Accept4(socket, unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK)
	if err != nil {
		return fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	defer unix.Close(accepted)
	credential, err := unix.GetsockoptUcred(accepted, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil || credential == nil || credential.Uid != uint32(os.Geteuid()) ||
		credential.Pid != int32(os.Getpid()) {
		return fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	return nil
}

func credentialBrokerBoundSocketIdentity(
	socket, rootFD int,
	name, path string,
	owner uint32,
	exactMode bool,
) (brokerSocketIdentity, error) {
	var pathStat unix.Stat_t
	if unix.Fstatat(rootFD, name, &pathStat, unix.AT_SYMLINK_NOFOLLOW) != nil {
		return brokerSocketIdentity{}, fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	identity := brokerSocketIdentity{device: uint64(pathStat.Dev), inode: pathStat.Ino}
	if pathStat.Mode&unix.S_IFMT != unix.S_IFSOCK || pathStat.Uid != owner ||
		exactMode && pathStat.Mode&0o777 != 0o600 {
		return identity, fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	pathFD, err := unix.Openat(rootFD, name, unix.O_PATH|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return identity, fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	defer unix.Close(pathFD)
	var openedStat, socketStat unix.Stat_t
	address, addressErr := unix.Getsockname(socket)
	unixAddress, addressOK := address.(*unix.SockaddrUnix)
	if unix.Fstat(pathFD, &openedStat) != nil || unix.Fstat(socket, &socketStat) != nil ||
		openedStat.Mode&unix.S_IFMT != unix.S_IFSOCK || socketStat.Mode&unix.S_IFMT != unix.S_IFSOCK ||
		uint64(openedStat.Dev) != identity.device || openedStat.Ino != identity.inode ||
		addressErr != nil || !addressOK || unixAddress.Name != path {
		return identity, fail(CodeCredentialBrokerSocketUnsafe, nil)
	}
	return identity, nil
}

func removeCredentialBrokerSocket(root *os.File, name string, identity brokerSocketIdentity) error {
	if root == nil || identity == (brokerSocketIdentity{}) {
		return fail(CodeCredentialBrokerCleanupFailed, nil)
	}
	var stat unix.Stat_t
	err := unix.Fstatat(int(root.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil || stat.Mode&unix.S_IFMT != unix.S_IFSOCK ||
		uint64(stat.Dev) != identity.device || stat.Ino != identity.inode {
		return fail(CodeCredentialBrokerCleanupFailed, nil)
	}
	if unix.Unlinkat(int(root.Fd()), name, 0) != nil {
		return fail(CodeCredentialBrokerCleanupFailed, nil)
	}
	return nil
}
