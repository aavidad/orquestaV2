//go:build linux

package firecrackerlauncher

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type fakeRunner struct {
	mu       sync.Mutex
	requests []LaunchRequest
	inputs   [][]byte
	output   []byte
	captured uint64
	err      error
	closeErr error
	wait     bool
	closed   bool
}

type closeUnblockedRunner struct {
	entered chan struct{}
	release chan struct{}
	enter   sync.Once
	close   sync.Once
}

func TestClientIdentityBindsCanonicalSocketAndTrustedUID(t *testing.T) {
	first, err := newClient("/run/orquesta/launcher.sock", 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newClient("/run/orquesta/launcher.sock", 0)
	if err != nil {
		t.Fatal(err)
	}
	otherPath, err := newClient("/run/orquesta/other.sock", 0)
	if err != nil {
		t.Fatal(err)
	}
	otherUID, err := newClient("/run/orquesta/launcher.sock", 1000)
	if err != nil {
		t.Fatal(err)
	}
	identity := first.Identity()
	if identity.Ref != "launcher:firecracker:uds:v1" ||
		!validDigest(identity.Digest) ||
		identity != second.Identity() ||
		identity == otherPath.Identity() ||
		identity == otherUID.Identity() {
		t.Fatalf(
			"first=%+v second=%+v path=%+v uid=%+v",
			identity,
			second.Identity(),
			otherPath.Identity(),
			otherUID.Identity(),
		)
	}
	mutated := identity
	mutated.Digest = strings.Repeat("f", 64)
	if first.Identity() != identity {
		t.Fatal("caller mutated stored launcher identity")
	}
}

func (runner *closeUnblockedRunner) Run(
	context.Context,
	LaunchRequest,
	*os.File,
	*os.File,
) (RunResult, error) {
	runner.enter.Do(func() { close(runner.entered) })
	<-runner.release
	return RunResult{}, launcherError(CodeUnavailable)
}

func (runner *closeUnblockedRunner) Close() error {
	runner.close.Do(func() { close(runner.release) })
	return nil
}

func (runner *fakeRunner) Run(
	ctx context.Context,
	request LaunchRequest,
	input *os.File,
	outputDrive *os.File,
) (RunResult, error) {
	if runner.wait {
		<-ctx.Done()
		if runner.err != nil {
			return RunResult{}, runner.err
		}
		return RunResult{}, ctx.Err()
	}
	content, err := io.ReadAll(io.NewSectionReader(input, 0, 1<<20))
	if err != nil {
		return RunResult{}, err
	}
	runner.mu.Lock()
	runner.requests = append(runner.requests, request)
	runner.inputs = append(runner.inputs, content)
	runner.mu.Unlock()
	if runner.err != nil {
		return RunResult{}, runner.err
	}
	if len(runner.output) > 0 {
		if count, err := outputDrive.WriteAt(runner.output, 0); err != nil || count != len(runner.output) {
			return RunResult{}, launcherError(CodeOutputInvalid)
		}
	}
	return RunResult{
		CapturedOutputBytes: runner.captured,
		AssetDigest:         digestBytes([]byte("assets")),
	}, nil
}

func (runner *fakeRunner) Close() error {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.closed = true
	return runner.closeErr
}

func TestUnixLauncherRoundTripUsesPeerCredentialsAndDescriptors(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{output: []byte("ORQ-RESULT")}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() { cancel(); <-done }()

	inputPayload := inputDrivePayloadForTest("ORQ-FIRECRACKER-INPUT")
	input, inputDigest, err := NewSealedInput(inputPayload, config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = inputDigest
	client, err := newClient(config.SocketPath, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Launch(context.Background(), request, input, config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Output.Close()
	if result.Response.Code != responseCodeOK || result.Response.Nonce != request.Nonce ||
		result.Response.AssetDigest != digestBytes([]byte("assets")) {
		t.Fatalf("response=%+v", result.Response)
	}
	inputIdentity, _, err := descriptorMetadata(input)
	if err != nil {
		t.Fatal(err)
	}
	outputIdentity, outputSeals, err := descriptorMetadata(result.Output)
	outputFlags, flagErr := unix.FcntlInt(result.Output.Fd(), unix.F_GETFD, 0)
	if err != nil || inputIdentity == outputIdentity ||
		outputIdentity.mode&0o777 != 0o400 || outputSeals != inputSeals ||
		flagErr != nil || outputFlags&unix.FD_CLOEXEC == 0 {
		t.Fatalf(
			"output ownership/seals invalid: input=%+v output=%+v seals=%x err=%v",
			inputIdentity,
			outputIdentity,
			outputSeals,
			errors.Join(err, flagErr),
		)
	}
	content := make([]byte, len(runner.output))
	if _, err := result.Output.ReadAt(content, 0); err != nil || !strings.HasPrefix(string(content), "ORQ-RESULT") {
		t.Fatalf("output=%q err=%v", content, err)
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.requests) != 1 || runner.requests[0] != request ||
		len(runner.inputs) != 1 || string(runner.inputs[0]) != string(inputPayload) {
		t.Fatalf("runner requests=%+v inputs=%q", runner.requests, runner.inputs)
	}
}

func TestUnixLauncherHarnessUsesPrivateShortUDSTempsWithLongTMPDIR(t *testing.T) {
	longTMPDIR := filepath.Join(t.TempDir(), strings.Repeat("long-", 30))
	if err := os.Mkdir(longTMPDIR, 0o700); err != nil {
		t.Fatal(err)
	}
	if len(filepath.Join(longTMPDIR, "runtime", "launcher.sock")) <
		len(unix.RawSockaddrUnix{}.Path) {
		t.Fatal("long TMPDIR fixture does not exceed sun_path")
	}
	t.Setenv("TMPDIR", longTMPDIR)

	var first, second string
	t.Run("launcher_starts", func(t *testing.T) {
		first = shortUDSTempDirForTest(t)
		second = shortUDSTempDirForTest(t)
		if first == second {
			t.Fatalf("short UDS test directories are not unique: %s", first)
		}
		info, err := os.Stat(first)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("short UDS test directory is not private: mode=%v", info.Mode())
		}
		root := filepath.Join(first, "runtime")
		config := validConfigForTest(root)
		if len(config.SocketPath) >= len(unix.RawSockaddrUnix{}.Path) {
			t.Fatalf("short UDS test socket exceeds sun_path: %s", config.SocketPath)
		}
		prepareRuntimeRoot(t, root)
		server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
		if err != nil {
			t.Fatalf("launcher unavailable with long TMPDIR: %v", err)
		}
		if err := server.Close(); err != nil {
			t.Fatal(err)
		}
	})
	for _, root := range []string{first, second} {
		if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("short UDS test directory remains after cleanup: %s: %v", root, err)
		}
	}
}

func TestCapturedOutputLimitIsIndependentFromOutputDrive(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	config.MaxCapturedOutputBytes = 16 << 20
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{output: []byte("OUTCOME"), captured: 16 << 20}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() { cancel(); <-done }()

	request := validLaunchRequestForTest()
	request.MaxCapturedOutputBytes = 16 << 20
	input, digest, err := NewSealedInput(inputDrivePayloadForTest("input"), config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request.InputDigest = digest
	client, _ := newClient(config.SocketPath, uint32(os.Geteuid()))
	result, err := client.Launch(context.Background(), request, input, config.MaxInputBytes)
	if err != nil {
		t.Fatalf("capture larger than drive was rejected: %v", err)
	}
	defer result.Output.Close()
}

func TestUnixLauncherRejectsUnauthorizedPeerBeforeRunner(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	config.AllowedUID = uint32(os.Geteuid()) + 1
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{output: []byte("must-not-run")}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() { cancel(); <-done }()

	input, digest, err := NewSealedInput(inputDrivePayloadForTest("input"), config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	client, _ := newClient(config.SocketPath, uint32(os.Geteuid()))
	_, err = client.Launch(context.Background(), request, input, config.MaxInputBytes)
	if ErrorCode(err) != CodePeerUnauthorized {
		t.Fatalf("unauthorized launch err=%v", err)
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.requests) != 0 {
		t.Fatalf("unauthorized peer reached runner: %+v", runner.requests)
	}
}

func TestUnixLauncherRejectsMissingAndExcessDescriptorsWithoutLeaks(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{output: []byte("must-not-run")}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() { cancel(); <-done }()

	request := validLaunchRequestForTest()
	input, digest, err := NewSealedInput(inputDrivePayloadForTest("input"), config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request.InputDigest = digest
	payload, _ := marshalRequest(request)
	socket := connectForTest(t, config.SocketPath)
	waitForConnectionsForTest(t, server, 1)
	missingConnectionOwner := registeredConnectionOwnerForTest(t, server)
	if _, err := unix.SendmsgN(socket, payload, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	responsePayload, files, err := receivePacket(socket, 1)
	unix.Close(socket)
	closeFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	response, err := unmarshalResponse(responsePayload)
	if err != nil || response.Code != CodeDescriptorInvalid {
		t.Fatalf("response=%+v err=%v", response, err)
	}
	waitForConnectionsForTest(t, server, 0)
	waitForDescriptorOwnerCountForTest(t, missingConnectionOwner, 0)

	inputOwner := descriptorOwnerForFileForTest(t, input)
	baselineOwned := openDescriptorCountForOwnerForTest(t, inputOwner)
	socket = connectForTest(t, config.SocketPath)
	waitForConnectionsForTest(t, server, 1)
	excessConnectionOwner := registeredConnectionOwnerForTest(t, server)
	if _, err := unix.SendmsgN(
		socket,
		payload,
		unix.UnixRights(int(input.Fd()), int(input.Fd())),
		nil,
		0,
	); err != nil {
		t.Fatal(err)
	}
	responsePayload, files, err = receivePacket(socket, 1)
	unix.Close(socket)
	closeFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	response, err = unmarshalResponse(responsePayload)
	if err != nil || response.Code != CodeDescriptorInvalid {
		t.Fatalf("excess descriptor response=%+v err=%v", response, err)
	}
	// The response plus empty registry form the causal barrier: finishConnection
	// must have reclaimed the accepted socket and both received rights.
	waitForConnectionsForTest(t, server, 0)
	waitForDescriptorOwnerCountForTest(t, excessConnectionOwner, 0)
	waitForDescriptorOwnerCountForTest(t, inputOwner, baselineOwned)
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.requests) != 0 {
		t.Fatalf("invalid descriptors reached runner: %+v", runner.requests)
	}
}

func TestUnixLauncherTimeoutIsStableAndCleanupRemovesSocket(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{wait: true}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	input, digest, _ := NewSealedInput(inputDrivePayloadForTest("input"), config.MaxInputBytes)
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	request.Timeout = 30 * time.Millisecond
	client, _ := newClient(config.SocketPath, uint32(os.Geteuid()))
	_, err = client.Launch(context.Background(), request, input, config.MaxInputBytes)
	if ErrorCode(err) != CodeExecutionTimeout {
		t.Fatalf("timeout err=%v", err)
	}
	cancel()
	<-done
	if _, err := os.Lstat(config.SocketPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket remains after close: %v", err)
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if !runner.closed {
		t.Fatal("runner was not closed")
	}
}

func TestUnixLauncherCleanupFailureDominatesExpiredRequest(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{
		wait: true,
		err:  launcherError(CodeCleanupFailed),
	}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() {
		cancel()
		<-done
	}()
	input, digest, err := NewSealedInput(
		inputDrivePayloadForTest("input"),
		config.MaxInputBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	request.Timeout = 30 * time.Millisecond
	payload, err := marshalRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	socket := connectForTest(t, config.SocketPath)
	defer unix.Close(socket)
	if count, err := unix.SendmsgN(
		socket,
		payload,
		unix.UnixRights(int(input.Fd())),
		nil,
		0,
	); err != nil || count != len(payload) {
		t.Fatalf("send request: count=%d err=%v", count, err)
	}
	responsePayload, files, err := receivePacket(socket, 1)
	closeFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	response, err := unmarshalResponse(responsePayload)
	if err != nil || response.Code != CodeCleanupFailed {
		t.Fatalf("cleanup failure hidden by request timeout: response=%+v err=%v", response, err)
	}
}

func TestDescriptorsRejectRegularFilesAndInputDigestDrift(t *testing.T) {
	regular := filepath.Join(t.TempDir(), "regular")
	if err := os.WriteFile(regular, []byte("input"), 0o400); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(regular)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := validateInputDescriptor(
		context.Background(),
		file,
		1<<20,
		digestBytes([]byte("input")),
	); ErrorCode(err) != CodeInputInvalid {
		t.Fatalf("regular file accepted: %v", err)
	}
	input, _, err := NewSealedInput([]byte("input"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if _, err := validateInputDescriptor(
		context.Background(),
		input,
		1<<20,
		digestBytes([]byte("changed")),
	); ErrorCode(err) != CodeInputInvalid {
		t.Fatalf("digest drift accepted: %v", err)
	}
}

func TestSealedInputStreamsDeclaredSizeAndOutputDriveIsVariable(t *testing.T) {
	payload := strings.Repeat("stream-", 4096)
	input, digest, err := NewSealedInputFromReader(strings.NewReader(payload), int64(len(payload)), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if digest != digestBytes([]byte(payload)) {
		t.Fatalf("digest=%s", digest)
	}
	if _, _, err := NewSealedInputFromReader(strings.NewReader("short"), 6, 6); ErrorCode(err) != CodeInputInvalid {
		t.Fatalf("short stream accepted: %v", err)
	}
	for _, size := range []uint64{0, firecrackerSectorBytes - 1, maxOutputDriveBytes + firecrackerSectorBytes} {
		if output, err := NewOutputDescriptor(size); ErrorCode(err) != CodeOutputInvalid {
			if output != nil {
				_ = output.Close()
			}
			t.Fatalf("unsafe output size accepted: %d err=%v", size, err)
		}
	}
	output, err := NewOutputDescriptor(2 * firecrackerSectorBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	if identity, err := validateEmptyOutputDescriptor(output, 2*firecrackerSectorBytes); err != nil ||
		identity.size != int64(2*firecrackerSectorBytes) {
		t.Fatalf("variable output identity=%+v err=%v", identity, err)
	}
}

func TestServeStopsWithSilentPeer(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	config.CleanupTimeout = 25 * time.Millisecond
	prepareRuntimeRoot(t, root)
	server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	socket := connectForTest(t, config.SocketPath)
	defer unix.Close(socket)
	for attempts := 0; attempts < 100; attempts++ {
		server.mu.Lock()
		accepted := len(server.connections) == 1
		server.mu.Unlock()
		if accepted {
			break
		}
		if attempts == 99 {
			t.Fatal("silent connection was not accepted")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve close: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Serve remained blocked by silent peer")
	}
}

func TestServerCloseStopsRunnerBeforeWaitingForBlockedHandler(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	config.CleanupTimeout = 250 * time.Millisecond
	prepareRuntimeRoot(t, root)
	runner := &closeUnblockedRunner{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	_, serveDone := serveForTest(t, server)
	input, digest, err := NewSealedInput(
		inputDrivePayloadForTest("blocked-runner"),
		config.MaxInputBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	request.Timeout = 2 * time.Second
	client, err := newClient(config.SocketPath, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	clientDone := make(chan error, 1)
	go func() {
		result, launchErr := client.Launch(
			context.Background(),
			request,
			input,
			config.MaxInputBytes,
		)
		if result.Output != nil {
			_ = result.Output.Close()
		}
		clientDone <- launchErr
	}()
	select {
	case <-runner.entered:
	case <-time.After(time.Second):
		t.Fatal("runner was not entered")
	}
	started := time.Now()
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("Close waited for request timeout: %s", elapsed)
	}
	select {
	case <-clientDone:
	case <-time.After(time.Second):
		t.Fatal("client remained blocked after Close")
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve remained blocked after Close")
	}
}

func TestServerIsSingleStartAndRejectsServeAfterClose(t *testing.T) {
	t.Run("double_serve", func(t *testing.T) {
		root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
		config := validConfigForTest(root)
		prepareRuntimeRoot(t, root)
		server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
		if err != nil {
			t.Fatal(err)
		}
		cancel, done := serveForTest(t, server)
		if err := server.Serve(context.Background()); ErrorCode(err) != CodeUnavailable {
			t.Fatalf("second Serve accepted: %v", err)
		}
		cancel()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		server.mu.Lock()
		listener := server.listener
		server.mu.Unlock()
		if listener != -1 {
			t.Fatalf("closed listener retained reusable descriptor: %d", listener)
		}
	})
	t.Run("serve_after_close", func(t *testing.T) {
		root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
		config := validConfigForTest(root)
		prepareRuntimeRoot(t, root)
		server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
		if err != nil {
			t.Fatal(err)
		}
		if err := server.Close(); err != nil {
			t.Fatal(err)
		}
		if err := server.Serve(context.Background()); ErrorCode(err) != CodeUnavailable {
			t.Fatalf("Serve after Close accepted: %v", err)
		}
	})
}

func TestRuntimeRootAndSocketHaveExactGroupAccess(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	assertPathSecurityForTest(
		t,
		root,
		unix.S_IFDIR,
		0o750,
		uint32(os.Geteuid()),
		config.AllowedGID,
	)
	assertPathSecurityForTest(
		t,
		config.SocketPath,
		unix.S_IFSOCK,
		0o660,
		uint32(os.Geteuid()),
		config.AllowedGID,
	)
	marker := filepath.Join(root, runtimeMarkerName)
	var markerStat unix.Stat_t
	if unix.Lstat(marker, &markerStat) != nil || markerStat.Mode&unix.S_IFMT != unix.S_IFREG ||
		markerStat.Mode&0o777 != 0o400 || markerStat.Uid != uint32(os.Geteuid()) {
		t.Fatalf("unsafe marker metadata: %+v", markerStat)
	}
}

func TestMaxConcurrentRunsBoundsAcceptedHandlers(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	config.MaxConcurrentRuns = 1
	config.CleanupTimeout = 250 * time.Millisecond
	prepareRuntimeRoot(t, root)
	server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	first := connectForTest(t, config.SocketPath)
	defer unix.Close(first)
	waitForConnectionsForTest(t, server, 1)
	second := connectForTest(t, config.SocketPath)
	defer unix.Close(second)
	time.Sleep(25 * time.Millisecond)
	server.mu.Lock()
	active := len(server.connections)
	server.mu.Unlock()
	if active != 1 || len(server.slots) != 1 {
		t.Fatalf("concurrency cap bypassed: connections=%d slots=%d", active, len(server.slots))
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve close: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("bounded handlers prevented cooperative close")
	}
}

func TestReceivePacketClosesExcessAndTruncatedRights(t *testing.T) {
	for _, descriptorCount := range []int{3, 8} {
		t.Run(strconv.Itoa(descriptorCount), func(t *testing.T) {
			sockets, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer unix.Close(sockets[0])
			defer unix.Close(sockets[1])
			source, _, err := NewSealedInput(
				inputDrivePayloadForTest("rights-"+strconv.Itoa(descriptorCount)),
				1<<20,
			)
			if err != nil {
				t.Fatal(err)
			}
			defer source.Close()
			rightFDs := make([]int, descriptorCount)
			for index := range rightFDs {
				rightFDs[index] = int(source.Fd())
			}
			sourceOwner := descriptorOwnerForFileForTest(t, source)
			before := openDescriptorCountForOwnerForTest(t, sourceOwner)
			if _, err := unix.SendmsgN(sockets[0], []byte("packet"), unix.UnixRights(rightFDs...), nil, 0); err != nil {
				t.Fatal(err)
			}
			payload, files, err := receivePacket(sockets[1], 2)
			closeFiles(files)
			if ErrorCode(err) != CodeDescriptorInvalid || payload != nil || len(files) != 0 {
				t.Fatalf("excess rights accepted: payload=%q files=%d err=%v", payload, len(files), err)
			}
			after := openDescriptorCountForOwnerForTest(t, sourceOwner)
			if after != before {
				t.Fatalf(
					"received rights for owner %+v leaked: before=%d after=%d",
					sourceOwner,
					before,
					after,
				)
			}
		})
	}
}

func TestLauncherRejectsUnalignedInputDriveBeforeRunner(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() { cancel(); <-done }()
	input, digest, err := NewSealedInput([]byte{1}, config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	payload, _ := marshalRequest(request)
	socket := connectForTest(t, config.SocketPath)
	defer unix.Close(socket)
	if _, err := unix.SendmsgN(
		socket,
		payload,
		unix.UnixRights(int(input.Fd())),
		nil,
		0,
	); err != nil {
		t.Fatal(err)
	}
	responsePayload, files, err := receivePacket(socket, 1)
	closeFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	response, err := unmarshalResponse(responsePayload)
	if err != nil || response.Code != CodeInputInvalid {
		t.Fatalf("unaligned drive response=%+v err=%v", response, err)
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.requests) != 0 {
		t.Fatalf("unaligned drive reached runner: %+v", runner.requests)
	}
}

func TestServerMapsValidationDeadlineToExecutionTimeout(t *testing.T) {
	root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
	config := validConfigForTest(root)
	prepareRuntimeRoot(t, root)
	runner := &fakeRunner{}
	server, err := newServer(config, runner, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := serveForTest(t, server)
	defer func() { cancel(); <-done }()
	input, digest, err := NewSealedInput(inputDrivePayloadForTest("input"), config.MaxInputBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	request.Timeout = time.Nanosecond
	payload, _ := marshalRequest(request)
	socket := connectForTest(t, config.SocketPath)
	defer unix.Close(socket)
	if _, err := unix.SendmsgN(
		socket,
		payload,
		unix.UnixRights(int(input.Fd())),
		nil,
		0,
	); err != nil {
		t.Fatal(err)
	}
	responsePayload, files, err := receivePacket(socket, 1)
	closeFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	response, err := unmarshalResponse(responsePayload)
	if err != nil || response.Code != CodeExecutionTimeout {
		t.Fatalf("validation timeout response=%+v err=%v", response, err)
	}
}

func TestClientAuthenticatesServerBeforeSendingDescriptors(t *testing.T) {
	parent := shortUDSTempDirForTest(t)
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(parent, "fake.sock")
	listener, err := unix.Socket(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(listener)
	if err := unix.Bind(listener, &unix.SockaddrUnix{Name: socketPath}); err != nil {
		t.Fatal(err)
	}
	if err := unix.Listen(listener, 1); err != nil {
		t.Fatal(err)
	}
	received := make(chan int, 1)
	go func() {
		connection, _, acceptErr := unix.Accept4(listener, unix.SOCK_CLOEXEC)
		if acceptErr != nil {
			received <- -1
			return
		}
		defer unix.Close(connection)
		timeout := unix.NsecToTimeval((100 * time.Millisecond).Nanoseconds())
		_ = unix.SetsockoptTimeval(connection, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &timeout)
		payload := make([]byte, maxProtocolPacket)
		oob := make([]byte, unix.CmsgSpace(4))
		count, _, _, _, _ := unix.Recvmsg(connection, payload, oob, unix.MSG_CMSG_CLOEXEC)
		received <- count
	}()
	input, digest, err := NewSealedInput(inputDrivePayloadForTest("input"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	request := validLaunchRequestForTest()
	request.InputDigest = digest
	client, err := newClient(socketPath, uint32(os.Geteuid())+1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Launch(context.Background(), request, input, 1<<20); ErrorCode(err) != CodeServerUnauthorized {
		t.Fatalf("untrusted server accepted: %v", err)
	}
	if count := <-received; count != 0 {
		t.Fatalf("client sent %d bytes before authenticating server", count)
	}
}

func TestInputValidationAndConnectHonorCanceledContext(t *testing.T) {
	input, digest, err := NewSealedInput(inputDrivePayloadForTest("input"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := validateInputDriveDescriptor(ctx, input, 1<<20, digest); ErrorCode(err) != CodeUnavailable {
		t.Fatalf("canceled hash continued: %v", err)
	}
	socket, err := unix.Socket(
		unix.AF_UNIX,
		unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(socket)
	if err := connectSocketContext(ctx, socket, filepath.Join(shortUDSTempDirForTest(t), "absent.sock")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled connect continued: %v", err)
	}
	sourcePath := filepath.Join(t.TempDir(), "input.drive")
	if err := os.WriteFile(sourcePath, inputDrivePayloadForTest("source"), 0o400); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if sealed, _, err := NewSealedInputFromFileContext(
		ctx,
		source,
		int64(firecrackerSectorBytes),
		1<<20,
	); ErrorCode(err) != CodeUnavailable {
		if sealed != nil {
			_ = sealed.Close()
		}
		t.Fatalf("canceled durable source continued: %v", err)
	}
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pipeReader.Close()
	defer pipeWriter.Close()
	if sealed, _, err := NewSealedInputFromFileContext(
		context.Background(),
		pipeReader,
		int64(firecrackerSectorBytes),
		1<<20,
	); ErrorCode(err) != CodeInputInvalid {
		if sealed != nil {
			_ = sealed.Close()
		}
		t.Fatalf("blocking non-durable source accepted: %v", err)
	}
}

func TestServePropagatesRunnerAndSocketCleanupErrors(t *testing.T) {
	if err := joinServeAndCloseErrors(
		launcherError(CodeUnavailable),
		launcherError(CodeCleanupFailed),
	); ErrorCode(err) != CodeCleanupFailed {
		t.Fatalf("cleanup failure lost behind primary failure: %v", err)
	}
	t.Run("runner_close", func(t *testing.T) {
		root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
		config := validConfigForTest(root)
		prepareRuntimeRoot(t, root)
		runner := &fakeRunner{closeErr: errors.New("physical runner cleanup detail")}
		server, err := newServer(config, runner, uint32(os.Geteuid()))
		if err != nil {
			t.Fatal(err)
		}
		cancel, done := serveForTest(t, server)
		cancel()
		if err := <-done; ErrorCode(err) != CodeCleanupFailed {
			t.Fatalf("runner close error hidden: %v", err)
		}
	})
	t.Run("socket_replaced", func(t *testing.T) {
		root := filepath.Join(shortUDSTempDirForTest(t), "runtime")
		config := validConfigForTest(root)
		prepareRuntimeRoot(t, root)
		server, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid()))
		if err != nil {
			t.Fatal(err)
		}
		cancel, done := serveForTest(t, server)
		if err := unix.Unlink(config.SocketPath); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(config.SocketPath, []byte("replacement"), 0o600); err != nil {
			t.Fatal(err)
		}
		cancel()
		if err := <-done; ErrorCode(err) != CodeCleanupFailed {
			t.Fatalf("socket replacement cleanup error hidden: %v", err)
		}
	})
}

func TestRuntimePathRejectsIntermediateSymlink(t *testing.T) {
	base := shortUDSTempDirForTest(t)
	if err := os.Chmod(base, 0o700); err != nil {
		t.Fatal(err)
	}
	realParent := filepath.Join(base, "real")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	realRoot := filepath.Join(realParent, "runtime")
	prepareRuntimeRoot(t, realRoot)
	link := filepath.Join(base, "link")
	if err := os.Symlink(realParent, link); err != nil {
		t.Fatal(err)
	}
	config := validConfigForTest(filepath.Join(link, "runtime"))
	if _, err := newServer(config, &fakeRunner{}, uint32(os.Geteuid())); ErrorCode(err) != CodeRuntimeRootUnsafe {
		t.Fatalf("intermediate runtime symlink accepted: %v", err)
	}
}

func prepareRuntimeRoot(t *testing.T, root string) {
	t.Helper()
	if err := os.Chmod(filepath.Dir(root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o750); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, runtimeMarkerName)
	if err := os.WriteFile(marker, []byte(runtimeMarkerContent), 0o400); err != nil {
		t.Fatal(err)
	}
}

func inputDrivePayloadForTest(label string) []byte {
	payload := make([]byte, firecrackerSectorBytes)
	copy(payload, label)
	return payload
}

func serveForTest(t *testing.T, server *Server) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	for attempts := 0; attempts < 100; attempts++ {
		server.mu.Lock()
		started := server.started
		server.mu.Unlock()
		if started {
			return cancel, done
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	t.Fatal("launcher socket did not become ready")
	return nil, nil
}

func connectForTest(t *testing.T, path string) int {
	t.Helper()
	socket, err := unix.Socket(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Connect(socket, &unix.SockaddrUnix{Name: path}); err != nil {
		unix.Close(socket)
		t.Fatal(err)
	}
	return socket
}

func waitForConnectionsForTest(t *testing.T, server *Server, want int) {
	t.Helper()
	for attempts := 0; attempts < 100; attempts++ {
		server.mu.Lock()
		got := len(server.connections)
		server.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("connections did not reach %d", want)
}

func assertPathSecurityForTest(
	t *testing.T,
	path string,
	fileType uint32,
	mode uint32,
	uid uint32,
	gid uint32,
) {
	t.Helper()
	var stat unix.Stat_t
	if unix.Lstat(path, &stat) != nil || stat.Mode&unix.S_IFMT != fileType ||
		stat.Mode&0o777 != mode || stat.Uid != uid || stat.Gid != gid {
		t.Fatalf("unsafe metadata for %s: %+v", path, stat)
	}
}
