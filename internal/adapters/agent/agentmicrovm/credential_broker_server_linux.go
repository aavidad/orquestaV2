//go:build linux

package agentmicrovm

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const maximumUnixSocketPathBytes = len(unix.RawSockaddrUnix{}.Path) - 1

// CredentialBrokerServer owns only transport lifecycle. Credential authority
// and one-shot consumption remain in CredentialBroker.
type CredentialBrokerServer struct {
	config CredentialBrokerServerConfig
	broker *CredentialBroker
	slots  chan struct{}

	mu          sync.Mutex
	started     bool
	stopping    bool
	listener    *net.UnixListener
	socketRoot  *os.File
	socket      brokerSocketIdentity
	cancel      context.CancelFunc
	connections map[*net.UnixConn]struct{}
	done        chan struct{}
	waitErr     error
	active      sync.WaitGroup
}

type brokerSocketIdentity struct {
	device uint64
	inode  uint64
}

func NewCredentialBrokerServer(
	config CredentialBrokerServerConfig,
	broker *CredentialBroker,
) (*CredentialBrokerServer, error) {
	if broker == nil || !validCredentialBrokerServerConfig(config) {
		return nil, fail(CodeCredentialBrokerServerInvalid, nil)
	}
	return &CredentialBrokerServer{
		config:      config,
		broker:      broker,
		slots:       make(chan struct{}, int(config.MaxConcurrent)),
		connections: make(map[*net.UnixConn]struct{}),
		done:        make(chan struct{}),
	}, nil
}

func validCredentialBrokerServerConfig(config CredentialBrokerServerConfig) bool {
	if config.OwnerUID != uint32(os.Geteuid()) || config.ConnectionTimeout <= 0 ||
		config.MaxConcurrent == 0 || config.MaxConcurrent > maximumCredentialBrokerHandlers ||
		!canonicalBrokerSocketPath(config.SocketPath) {
		return false
	}
	return true
}

func canonicalBrokerSocketPath(path string) bool {
	return path != "" && filepath.IsAbs(path) && filepath.Clean(path) == path &&
		filepath.Dir(path) != path && filepath.Base(path) != "." && filepath.Base(path) != ".." &&
		!strings.ContainsRune(path, '\x00') && len(path) <= maximumUnixSocketPathBytes
}

// Start publishes the socket before returning and starts one resident accept
// loop. A server instance cannot be restarted with a different physical peer.
func (server *CredentialBrokerServer) Start(ctx context.Context) error {
	if server == nil || nilInterface(ctx) || ctx.Err() != nil {
		return fail(CodeCredentialBrokerServerInvalid, nil)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.started || server.stopping || server.broker == nil {
		return fail(CodeCredentialBrokerServerInvalid, nil)
	}
	listener, root, identity, err := openCredentialBrokerSocket(server.config)
	if err != nil {
		return err
	}
	serveCtx, cancel := context.WithCancel(ctx)
	server.started = true
	server.listener = listener
	server.socketRoot = root
	server.socket = identity
	server.cancel = cancel
	go server.serve(serveCtx, listener)
	go func() {
		<-serveCtx.Done()
		server.stopTransport()
	}()
	return nil
}

func (server *CredentialBrokerServer) serve(ctx context.Context, listener *net.UnixListener) {
	var result error
	defer func() { server.finishServe(result) }()
	for {
		select {
		case server.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		connection, err := listener.AcceptUnix()
		if err != nil {
			<-server.slots
			if ctx.Err() != nil || server.isStopping() || errors.Is(err, net.ErrClosed) {
				return
			}
			result = fail(CodeCredentialBrokerServerUnavailable, nil)
			return
		}
		if !server.registerConnection(connection) {
			_ = connection.Close()
			<-server.slots
			return
		}
		go server.serveConnection(ctx, connection)
	}
}

func (server *CredentialBrokerServer) serveConnection(parent context.Context, connection *net.UnixConn) {
	defer server.finishConnection(connection)
	if err := server.authenticatePeer(connection); err != nil {
		return
	}
	connectionCtx, cancel := context.WithTimeout(parent, server.config.ConnectionTimeout)
	defer cancel()
	_ = server.broker.Handle(connectionCtx, connection)
}

func (server *CredentialBrokerServer) authenticatePeer(connection *net.UnixConn) error {
	if connection == nil {
		return fail(CodeCredentialBrokerPeerDenied, nil)
	}
	credential, err := credentialBrokerPeerCredentials(connection)
	// The production peer is a root-owned agente_microvm daemon. A non-root
	// Orquesta cannot read /proc/<root-pid>/exe, so executable hashing would make
	// the real composition impossible. The private socket and exact peer UID are
	// the trust boundary; a positive kernel-supplied PID rejects absent peers.
	if err != nil || credential == nil || credential.Pid <= 0 || credential.Uid != server.config.PeerUID {
		return fail(CodeCredentialBrokerPeerDenied, nil)
	}
	return nil
}

func credentialBrokerPeerCredentials(connection *net.UnixConn) (*unix.Ucred, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return nil, err
	}
	var credential *unix.Ucred
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		credential, controlErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return nil, err
	}
	if controlErr != nil {
		return nil, controlErr
	}
	return credential, nil
}

func (server *CredentialBrokerServer) registerConnection(connection *net.UnixConn) bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.stopping {
		return false
	}
	server.connections[connection] = struct{}{}
	server.active.Add(1)
	return true
}

func (server *CredentialBrokerServer) finishConnection(connection *net.UnixConn) {
	server.mu.Lock()
	delete(server.connections, connection)
	server.mu.Unlock()
	_ = connection.Close()
	<-server.slots
	server.active.Done()
}

func (server *CredentialBrokerServer) isStopping() bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.stopping
}

func (server *CredentialBrokerServer) finishServe(serveErr error) {
	server.stopTransport()
	server.active.Wait()
	server.mu.Lock()
	cleanupErr := removeCredentialBrokerSocket(server.socketRoot, filepath.Base(server.config.SocketPath), server.socket)
	rootErr := error(nil)
	if server.socketRoot != nil {
		rootErr = server.socketRoot.Close()
		server.socketRoot = nil
	}
	if cleanupErr != nil || rootErr != nil {
		server.waitErr = errors.Join(fail(CodeCredentialBrokerCleanupFailed, nil), serveErr)
	} else {
		server.waitErr = serveErr
	}
	close(server.done)
	server.mu.Unlock()
}

func (server *CredentialBrokerServer) stopTransport() {
	server.mu.Lock()
	if server.stopping {
		server.mu.Unlock()
		return
	}
	server.stopping = true
	cancel := server.cancel
	listener := server.listener
	server.listener = nil
	connections := make([]*net.UnixConn, 0, len(server.connections))
	for connection := range server.connections {
		connections = append(connections, connection)
	}
	server.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if listener != nil {
		_ = listener.Close()
	}
	for _, connection := range connections {
		_ = connection.Close()
	}
}

// Shutdown is idempotent. It closes the listener and every accepted
// connection before waiting for exact socket cleanup.
func (server *CredentialBrokerServer) Shutdown(ctx context.Context) error {
	if server == nil || nilInterface(ctx) {
		return fail(CodeCredentialBrokerServerInvalid, nil)
	}
	server.mu.Lock()
	if !server.started {
		server.mu.Unlock()
		return nil
	}
	done := server.done
	server.mu.Unlock()
	server.stopTransport()
	select {
	case <-done:
		return server.waitResult()
	case <-ctx.Done():
		return fail(CodeCredentialBrokerServerUnavailable, nil)
	}
}

// Wait blocks until the resident loop and all accepted exchanges have ended.
func (server *CredentialBrokerServer) Wait() error {
	if server == nil {
		return fail(CodeCredentialBrokerServerInvalid, nil)
	}
	server.mu.Lock()
	if !server.started {
		server.mu.Unlock()
		return fail(CodeCredentialBrokerServerInvalid, nil)
	}
	done := server.done
	server.mu.Unlock()
	<-done
	return server.waitResult()
}

func (server *CredentialBrokerServer) waitResult() error {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.waitErr
}
