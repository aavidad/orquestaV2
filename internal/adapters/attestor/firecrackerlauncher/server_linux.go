//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const runtimeMarkerContent = "schema=orquesta_firecracker_launcher_root.v1\n"

type Server struct {
	config      Config
	runner      Runner
	listener    int
	socket      descriptorIdentity
	runtimeRoot *os.File
	slots       chan struct{}

	mu          sync.Mutex
	closed      bool
	started     bool
	connections map[int]struct{}
	close       sync.Once
	active      sync.WaitGroup
	closeErr    error
}

func NewServer(config Config) (*Server, error) {
	if os.Geteuid() != 0 {
		return nil, launcherError(CodePrivilegeRequired)
	}
	// Physical jailer execution remains deliberately unavailable until its
	// cgroup and cleanup implementation is accredited. The transport can be
	// installed safely: every request fails closed with CodeUnavailable.
	return newServer(config, unavailableRunner{}, 0)
}

func newServer(config Config, runner Runner, trustedOwner uint32) (*Server, error) {
	if err := validateConfig(config); err != nil || runner == nil {
		return nil, launcherError(CodeConfigInvalid)
	}
	runtimeRoot, err := openTrustedRuntimeRoot(
		config.RuntimeRoot,
		trustedOwner,
		config.AllowedGID,
	)
	if err != nil {
		return nil, err
	}
	listener, socketIdentity, err := openLauncherSocket(config, trustedOwner, runtimeRoot)
	if err != nil {
		_ = runtimeRoot.Close()
		return nil, err
	}
	return &Server{
		config: config, runner: runner, listener: listener, socket: socketIdentity,
		runtimeRoot: runtimeRoot,
		slots:       make(chan struct{}, int(config.MaxConcurrentRuns)), connections: make(map[int]struct{}),
	}, nil
}

func (server *Server) Serve(ctx context.Context) (result error) {
	if server == nil {
		return launcherError(CodeUnavailable)
	}
	if !server.beginServe() {
		return launcherError(CodeUnavailable)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			server.closeListener()
		case <-done:
		}
	}()
	defer close(done)
	defer func() {
		result = joinServeAndCloseErrors(result, server.Close())
	}()
	for {
		select {
		case server.slots <- struct{}{}:
		case <-ctx.Done():
			return nil
		}
		listener, pinErr := server.pinListener()
		if pinErr != nil {
			<-server.slots
			if ctx.Err() != nil || server.isClosed() {
				return nil
			}
			return launcherError(CodeUnavailable)
		}
		connection, err := server.acceptConnection(ctx, listener)
		_ = unix.Close(listener)
		if err != nil {
			<-server.slots
			server.mu.Lock()
			closed := server.closed
			server.mu.Unlock()
			if closed || ctx.Err() != nil || errors.Is(err, unix.EBADF) || errors.Is(err, unix.EINVAL) {
				return nil
			}
			return launcherError(CodeUnavailable)
		}
		if err := setSocketTimeouts(connection, server.config.CleanupTimeout); err != nil {
			_ = unix.Close(connection)
			<-server.slots
			continue
		}
		if !server.registerConnection(connection) {
			_ = unix.Close(connection)
			<-server.slots
			return nil
		}
		go func() {
			defer server.finishConnection(connection)
			server.handle(ctx, connection)
		}()
	}
}

func (server *Server) beginServe() bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.closed || server.started || server.listener < 0 {
		return false
	}
	server.started = true
	return true
}

func (server *Server) pinListener() (int, error) {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.closed || server.listener < 0 {
		return -1, unix.EBADF
	}
	duplicate, err := unix.FcntlInt(uintptr(server.listener), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	return duplicate, nil
}

func (server *Server) acceptConnection(ctx context.Context, listener int) (int, error) {
	pollDescriptors := []unix.PollFd{{Fd: int32(listener), Events: unix.POLLIN}}
	for {
		if ctx.Err() != nil || server.isClosed() {
			return -1, unix.EBADF
		}
		connection, _, err := unix.Accept4(
			listener,
			unix.SOCK_CLOEXEC,
		)
		if err == nil {
			return connection, nil
		}
		if !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EWOULDBLOCK) &&
			!errors.Is(err, unix.EINTR) {
			return -1, err
		}
		_, err = unix.Poll(pollDescriptors, 10)
		if err != nil && !errors.Is(err, unix.EINTR) {
			return -1, err
		}
	}
}

func (server *Server) isClosed() bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.closed
}

func joinServeAndCloseErrors(serveErr, closeErr error) error {
	// Cleanup failure can mean a residual privileged process or replaced
	// filesystem object, so it must be the public machine code when both fail.
	return errors.Join(closeErr, serveErr)
}

func (server *Server) handle(parent context.Context, connection int) {
	credential, err := unix.GetsockoptUcred(connection, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil || credential.Pid <= 0 || !server.config.peerAllowed(credential.Uid, credential.Gid) {
		// Drain exactly one bounded packet and close any received rights before
		// replying. Closing a SOCK_SEQPACKET peer with unread data can reset it
		// before the stable denial reaches the client.
		payload, files, _ := receivePacket(connection, 1)
		closeFiles(files)
		nonce := strings.Repeat("0", 64)
		if request, decodeErr := unmarshalRequest(payload); decodeErr == nil {
			nonce = request.Nonce
		}
		server.sendFailure(connection, nonce, CodePeerUnauthorized)
		return
	}
	payload, files, err := receivePacket(connection, 1)
	if err != nil {
		server.sendFailure(connection, strings.Repeat("0", 64), safeCode(err, CodeProtocolInvalid))
		return
	}
	defer closeFiles(files)
	if len(files) != 1 {
		server.sendFailure(connection, strings.Repeat("0", 64), CodeDescriptorInvalid)
		return
	}
	request, err := unmarshalRequest(payload)
	if err != nil {
		server.sendFailure(connection, strings.Repeat("0", 64), CodeProtocolInvalid)
		return
	}
	if err := server.config.validateRequest(request); err != nil {
		server.sendFailure(connection, request.Nonce, safeCode(err, CodeResourceUnsafe))
		return
	}
	requestContext, cancel := context.WithTimeout(parent, request.Timeout)
	defer cancel()
	if _, err := validateInputDriveDescriptor(
		requestContext,
		files[0],
		server.config.MaxInputBytes,
		request.InputDigest,
	); err != nil {
		code := safeCode(err, CodeInputInvalid)
		if requestContext.Err() != nil {
			code = contextFailureCode(requestContext)
		}
		server.sendFailure(connection, request.Nonce, code)
		return
	}
	outputDrive, err := NewOutputDescriptor(request.OutputDriveBytes)
	if err != nil {
		server.sendFailure(connection, request.Nonce, CodeOutputInvalid)
		return
	}
	defer outputDrive.Close()
	runResult, runErr := server.runner.Run(requestContext, request, files[0], outputDrive)
	if runErr != nil {
		code := safeCode(runErr, CodeExecutionFailed)
		if requestContext.Err() != nil {
			code = contextFailureCode(requestContext)
		}
		server.sendFailure(connection, request.Nonce, code)
		return
	}
	if !validDigest(runResult.AssetDigest) {
		server.sendFailure(connection, request.Nonce, CodeAssetsUnsafe)
		return
	}
	if runResult.CapturedOutputBytes > request.MaxCapturedOutputBytes {
		server.sendFailure(connection, request.Nonce, CodeResourceUnsafe)
		return
	}
	outputDigest, err := publishOutput(outputDrive, request.OutputDriveBytes)
	if err != nil {
		server.sendFailure(connection, request.Nonce, CodeOutputInvalid)
		return
	}
	response, err := marshalResponse(LaunchResponse{
		Nonce: request.Nonce, Code: responseCodeOK,
		OutputDigest: outputDigest, AssetDigest: runResult.AssetDigest,
	})
	if err != nil || sendPacket(connection, response, outputDrive) != nil {
		return
	}
}

func (server *Server) sendFailure(connection int, nonce, code string) {
	if !validNonce(nonce) {
		nonce = strings.Repeat("0", 64)
	}
	if !validResponseCode(code) || code == responseCodeOK {
		code = CodeExecutionFailed
	}
	payload, err := marshalResponse(LaunchResponse{Nonce: nonce, Code: code})
	if err == nil {
		_ = sendPacket(connection, payload, nil)
	}
}

func safeCode(err error, fallback string) string {
	code := ErrorCode(err)
	if !validResponseCode(code) || code == responseCodeOK {
		return fallback
	}
	return code
}

func (server *Server) Close() error {
	if server == nil {
		return nil
	}
	server.close.Do(func() {
		server.closeListener()
		server.active.Wait()
		server.closeErr = errors.Join(
			normalizeCleanupError(server.runner.Close()),
			removeLauncherSocket(server.runtimeRoot, filepath.Base(server.config.SocketPath), server.socket),
			normalizeCleanupError(server.runtimeRoot.Close()),
		)
	})
	return server.closeErr
}

func normalizeCleanupError(err error) error {
	if err == nil {
		return nil
	}
	if ErrorCode(err) == CodeCleanupFailed {
		return err
	}
	return launcherError(CodeCleanupFailed)
}

func (server *Server) closeListener() {
	server.mu.Lock()
	if server.closed {
		server.mu.Unlock()
		return
	}
	server.closed = true
	listener := server.listener
	server.listener = -1
	if listener >= 0 {
		_ = unix.Shutdown(listener, unix.SHUT_RDWR)
	}
	for connection := range server.connections {
		_ = unix.Shutdown(connection, unix.SHUT_RDWR)
	}
	server.mu.Unlock()
	if listener >= 0 {
		_ = unix.Close(listener)
	}
}

func (server *Server) registerConnection(connection int) bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.closed {
		return false
	}
	server.connections[connection] = struct{}{}
	server.active.Add(1)
	return true
}

func (server *Server) finishConnection(connection int) {
	server.mu.Lock()
	delete(server.connections, connection)
	server.mu.Unlock()
	_ = unix.Close(connection)
	<-server.slots
	server.active.Done()
}

func setSocketTimeouts(socket int, timeout time.Duration) error {
	value := unix.NsecToTimeval(timeout.Nanoseconds())
	if value.Sec == 0 && value.Usec == 0 {
		value.Usec = 1
	}
	if unix.SetsockoptTimeval(socket, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &value) != nil ||
		unix.SetsockoptTimeval(socket, unix.SOL_SOCKET, unix.SO_SNDTIMEO, &value) != nil {
		return launcherError(CodeUnavailable)
	}
	return nil
}

func openLauncherSocket(
	config Config,
	owner uint32,
	runtimeRoot *os.File,
) (int, descriptorIdentity, error) {
	if filepath.Dir(config.SocketPath) != config.RuntimeRoot {
		return -1, descriptorIdentity{}, launcherError(CodeConfigInvalid)
	}
	socketName := filepath.Base(config.SocketPath)
	var existing unix.Stat_t
	if err := unix.Fstatat(
		int(runtimeRoot.Fd()),
		socketName,
		&existing,
		unix.AT_SYMLINK_NOFOLLOW,
	); !errors.Is(err, unix.ENOENT) {
		return -1, descriptorIdentity{}, launcherError(CodeRuntimeRootUnsafe)
	}
	socket, err := unix.Socket(
		unix.AF_UNIX,
		unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK,
		0,
	)
	if err != nil {
		return -1, descriptorIdentity{}, launcherError(CodeUnavailable)
	}
	fail := func(code string) (int, descriptorIdentity, error) {
		_ = unix.Close(socket)
		return -1, descriptorIdentity{}, launcherError(code)
	}
	if err := unix.Bind(socket, &unix.SockaddrUnix{Name: config.SocketPath}); err != nil {
		return fail(CodeUnavailable)
	}
	if unix.Fchownat(
		int(runtimeRoot.Fd()),
		socketName,
		int(owner),
		int(config.AllowedGID),
		unix.AT_SYMLINK_NOFOLLOW,
	) != nil ||
		unix.Fchmodat(int(runtimeRoot.Fd()), socketName, 0o660, 0) != nil ||
		unix.Listen(socket, int(config.MaxConcurrentRuns)) != nil {
		_ = unix.Unlinkat(int(runtimeRoot.Fd()), socketName, 0)
		return fail(CodeRuntimeRootUnsafe)
	}
	var stat unix.Stat_t
	if unix.Fstatat(int(runtimeRoot.Fd()), socketName, &stat, unix.AT_SYMLINK_NOFOLLOW) != nil ||
		stat.Mode&unix.S_IFMT != unix.S_IFSOCK ||
		stat.Uid != owner || stat.Gid != config.AllowedGID || stat.Mode&0o777 != 0o660 {
		_ = unix.Unlinkat(int(runtimeRoot.Fd()), socketName, 0)
		return fail(CodeRuntimeRootUnsafe)
	}
	return socket, descriptorIdentity{device: uint64(stat.Dev), inode: stat.Ino, mode: stat.Mode}, nil
}

func removeLauncherSocket(
	runtimeRoot *os.File,
	name string,
	identity descriptorIdentity,
) error {
	var stat unix.Stat_t
	if runtimeRoot == nil {
		return launcherError(CodeCleanupFailed)
	}
	if err := unix.Fstatat(int(runtimeRoot.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		return launcherError(CodeCleanupFailed)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFSOCK || uint64(stat.Dev) != identity.device || stat.Ino != identity.inode {
		return launcherError(CodeCleanupFailed)
	}
	if unix.Unlinkat(int(runtimeRoot.Fd()), name, 0) != nil {
		return launcherError(CodeCleanupFailed)
	}
	return nil
}
