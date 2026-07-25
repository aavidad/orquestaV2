//go:build linux

package firecrackerlauncher

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	launchercontract "orquesta/internal/testattestorprotocol/launcher"
)

type Client struct {
	socketPath string
	trustedUID uint32
	identity   launchercontract.Identity
}

type ClientResult = launchercontract.ClientResult

func NewClient(socketPath string) (*Client, error) {
	return newClient(socketPath, 0)
}

func newClient(socketPath string, trustedUID uint32) (*Client, error) {
	if !canonicalAbsolute(socketPath) {
		return nil, launcherError(CodeConfigInvalid)
	}
	return &Client{
		socketPath: socketPath,
		trustedUID: trustedUID,
		identity:   configuredClientIdentity(socketPath, trustedUID),
	}, nil
}

// Identity binds UDS transport semantics, canonical socket path and trusted
// server UID. Real trust still requires composition to protect the path,
// SO_PEERCRED to authenticate the connected peer, and launcher asset checks.
func (client *Client) Identity() launchercontract.Identity {
	if client == nil {
		return launchercontract.Identity{}
	}
	return client.identity
}

func configuredClientIdentity(
	socketPath string,
	trustedUID uint32,
) launchercontract.Identity {
	digest := sha256.New()
	for _, value := range []string{
		"orquesta.test-attestor.firecracker-launcher-client.v1",
		"launcher:firecracker:uds:v1",
		socketPath,
		strconv.FormatUint(uint64(trustedUID), 10),
		"sock-seqpacket;scm-rights;so-peercred;server-uid",
	} {
		writeClientIdentityField(digest, value)
	}
	return launchercontract.Identity{
		Ref:    "launcher:firecracker:uds:v1",
		Digest: hex.EncodeToString(digest.Sum(nil)),
	}
}

func writeClientIdentityField(digest hash.Hash, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = digest.Write(size[:])
	_, _ = digest.Write([]byte(value))
}

func (client *Client) Launch(
	ctx context.Context,
	request LaunchRequest,
	input *os.File,
	maxInputBytes int64,
) (ClientResult, error) {
	requestContext, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	payload, err := marshalRequest(request)
	if err != nil {
		if requestContext.Err() != nil {
			return ClientResult{}, launcherError(contextFailureCode(requestContext))
		}
		return ClientResult{}, err
	}
	inputIdentity, err := validateInputDriveDescriptor(
		requestContext,
		input,
		maxInputBytes,
		request.InputDigest,
	)
	if err != nil {
		if requestContext.Err() != nil {
			return ClientResult{}, launcherError(contextFailureCode(requestContext))
		}
		return ClientResult{}, err
	}
	socket, err := unix.Socket(
		unix.AF_UNIX,
		unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK,
		0,
	)
	if err != nil {
		return ClientResult{}, launcherError(CodeUnavailable)
	}
	defer unix.Close(socket)
	if err := connectSocketContext(requestContext, socket, client.socketPath); err != nil {
		if requestContext.Err() != nil {
			return ClientResult{}, launcherError(contextFailureCode(requestContext))
		}
		return ClientResult{}, launcherError(CodeUnavailable)
	}
	credential, err := unix.GetsockoptUcred(socket, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil || credential.Pid <= 0 || credential.Uid != client.trustedUID {
		return ClientResult{}, launcherError(CodeServerUnauthorized)
	}
	if err := unix.SetNonblock(socket, false); err != nil {
		return ClientResult{}, launcherError(CodeUnavailable)
	}
	rights := unix.UnixRights(int(input.Fd()))
	if count, err := unix.SendmsgN(socket, payload, rights, nil, 0); err != nil || count != len(payload) {
		return ClientResult{}, launcherError(CodeUnavailable)
	}
	type received struct {
		payload []byte
		files   []*os.File
		err     error
	}
	done := make(chan received, 1)
	var closeOnce sync.Once
	closeSocket := func() { closeOnce.Do(func() { _ = unix.Shutdown(socket, unix.SHUT_RDWR) }) }
	go func() {
		message, files, receiveErr := receivePacket(socket, 1)
		done <- received{message, files, receiveErr}
	}()
	var got received
	select {
	case got = <-done:
	case <-requestContext.Done():
		closeSocket()
		got = <-done
		closeFiles(got.files)
		return ClientResult{}, launcherError(contextFailureCode(requestContext))
	}
	if got.err != nil {
		closeFiles(got.files)
		return ClientResult{}, got.err
	}
	response, err := unmarshalResponse(got.payload)
	if err != nil || response.Nonce != request.Nonce {
		closeFiles(got.files)
		return ClientResult{}, launcherError(CodeResponseInvalid)
	}
	if response.Code != responseCodeOK {
		closeFiles(got.files)
		if len(got.files) != 0 {
			return ClientResult{}, launcherError(CodeResponseInvalid)
		}
		return ClientResult{}, launcherError(response.Code)
	}
	if len(got.files) != 1 {
		closeFiles(got.files)
		return ClientResult{}, launcherError(CodeResponseInvalid)
	}
	returnedIdentity, err := validatePublishedOutput(
		got.files[0],
		request.OutputDriveBytes,
		response.OutputDigest,
	)
	if err != nil || inputIdentity == returnedIdentity {
		closeFiles(got.files)
		return ClientResult{}, launcherError(CodeResponseInvalid)
	}
	return ClientResult{Response: response, Output: got.files[0]}, nil
}

func contextFailureCode(ctx context.Context) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return CodeExecutionTimeout
	}
	return CodeUnavailable
}

func connectSocketContext(ctx context.Context, socket int, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := unix.Connect(socket, &unix.SockaddrUnix{Name: path})
	if err == nil || errors.Is(err, unix.EISCONN) {
		return nil
	}
	if !errors.Is(err, unix.EINPROGRESS) && !errors.Is(err, unix.EAGAIN) &&
		!errors.Is(err, unix.EALREADY) {
		return err
	}
	pollDescriptors := []unix.PollFd{{Fd: int32(socket), Events: unix.POLLOUT}}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		wait := 10 * time.Millisecond
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return context.DeadlineExceeded
			}
			if remaining < wait {
				wait = remaining
			}
		}
		ready, err := unix.Poll(pollDescriptors, max(1, int(wait/time.Millisecond)))
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		if ready == 0 {
			continue
		}
		socketError, err := unix.GetsockoptInt(socket, unix.SOL_SOCKET, unix.SO_ERROR)
		if err != nil {
			return err
		}
		if socketError != 0 {
			return syscallError(socketError)
		}
		return nil
	}
}

func syscallError(value int) error {
	return unix.Errno(value)
}

func receivePacket(socket, maxFiles int) ([]byte, []*os.File, error) {
	payload := make([]byte, maxProtocolPacket+1)
	oob := make([]byte, unix.CmsgSpace(max(1, maxFiles+1)*4))
	count, oobCount, flags, _, err := unix.Recvmsg(socket, payload, oob, unix.MSG_CMSG_CLOEXEC)
	if err != nil {
		return nil, nil, launcherError(CodeUnavailable)
	}
	files, rightsErr := parseRights(oob[:oobCount])
	if count == 0 || count > maxProtocolPacket || flags&unix.MSG_TRUNC != 0 {
		closeFiles(files)
		return nil, nil, launcherError(CodeProtocolInvalid)
	}
	if rightsErr != nil || flags&unix.MSG_CTRUNC != 0 || len(files) > maxFiles {
		closeFiles(files)
		return nil, nil, launcherError(CodeDescriptorInvalid)
	}
	return payload[:count], files, nil
}

func parseRights(oob []byte) ([]*os.File, error) {
	if len(oob) == 0 {
		return nil, nil
	}
	messages, err := unix.ParseSocketControlMessage(oob)
	valid := err == nil && len(messages) > 0
	var rights []int
	for index := range messages {
		message := &messages[index]
		if message.Header.Level != unix.SOL_SOCKET || message.Header.Type != unix.SCM_RIGHTS {
			valid = false
			continue
		}
		messageRights, parseErr := unix.ParseUnixRights(message)
		if parseErr != nil {
			valid = false
			continue
		}
		rights = append(rights, messageRights...)
	}
	files := make([]*os.File, len(rights))
	for index, fd := range rights {
		unix.CloseOnExec(fd)
		files[index] = os.NewFile(uintptr(fd), "orquesta-firecracker-launcher-right")
	}
	if !valid {
		closeFiles(files)
		return nil, launcherError(CodeDescriptorInvalid)
	}
	return files, nil
}

func sendPacket(socket int, payload []byte, file *os.File) error {
	var rights []byte
	if file != nil {
		rights = unix.UnixRights(int(file.Fd()))
	}
	count, err := unix.SendmsgN(socket, payload, rights, nil, 0)
	if err != nil || count != len(payload) {
		return launcherError(CodeUnavailable)
	}
	return nil
}

func closeFiles(files []*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}

var _ launchercontract.Client = (*Client)(nil)
