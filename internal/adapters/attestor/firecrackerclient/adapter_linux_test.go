//go:build linux

package firecrackerclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	launcherprotocol "orquesta/internal/testattestorprotocol/launcher"
	"orquesta/internal/testattestorprotocol/rawdrive"
)

var adapterTestNow = time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

type fakeLauncherClient struct {
	identityMu sync.RWMutex
	identity   launcherprotocol.Identity
	launch     func(
		context.Context,
		launcherprotocol.LaunchRequest,
		*os.File,
		int64,
	) (launcherprotocol.ClientResult, error)
}

func (client *fakeLauncherClient) Identity() launcherprotocol.Identity {
	client.identityMu.RLock()
	defer client.identityMu.RUnlock()
	if client.identity == (launcherprotocol.Identity{}) {
		return launcherprotocol.Identity{
			Ref:    "launcher:test:v1",
			Digest: testDigest("launcher"),
		}
	}
	return client.identity
}

func (client *fakeLauncherClient) setIdentity(identity launcherprotocol.Identity) {
	client.identityMu.Lock()
	client.identity = identity
	client.identityMu.Unlock()
}

func (client *fakeLauncherClient) Launch(
	ctx context.Context,
	request launcherprotocol.LaunchRequest,
	input *os.File,
	maxInput int64,
) (launcherprotocol.ClientResult, error) {
	return client.launch(ctx, request, input, maxInput)
}

type blockingIdentityLauncher struct {
	identity    launcherprotocol.Identity
	blockOnCall int64
	calls       atomic.Int64
	entered     chan struct{}
	release     chan struct{}
}

func (client *blockingIdentityLauncher) Identity() launcherprotocol.Identity {
	if client.calls.Add(1) == client.blockOnCall {
		close(client.entered)
		<-client.release
	}
	return client.identity
}

func (*blockingIdentityLauncher) Launch(
	ctx context.Context,
	_ launcherprotocol.LaunchRequest,
	_ *os.File,
	_ int64,
) (launcherprotocol.ClientResult, error) {
	<-ctx.Done()
	return launcherprotocol.ClientResult{}, ctx.Err()
}

type fakeSnapshotSource struct {
	mu       sync.RWMutex
	ref      string
	digest   string
	payload  []byte
	openErr  error
	closeErr error
	opens    atomic.Int64
	closes   atomic.Int64
}

func (source *fakeSnapshotSource) SnapshotIdentity() (string, string) {
	source.mu.RLock()
	defer source.mu.RUnlock()
	return source.ref, source.digest
}

func (source *fakeSnapshotSource) OpenSnapshotStream(
	context.Context,
	ports.SnapshotVerificationRequest,
) (io.ReadCloser, error) {
	source.opens.Add(1)
	source.mu.RLock()
	defer source.mu.RUnlock()
	if source.openErr != nil {
		return nil, source.openErr
	}
	return &trackedSnapshotStream{
		Reader: bytes.NewReader(append([]byte(nil), source.payload...)),
		source: source,
	}, nil
}

type trackedSnapshotStream struct {
	*bytes.Reader
	source *fakeSnapshotSource
	once   sync.Once
}

type scriptedSnapshotSource struct {
	identity launcherprotocol.Identity
	open     func(context.Context) (io.ReadCloser, error)
}

func (source *scriptedSnapshotSource) SnapshotIdentity() (string, string) {
	return source.identity.Ref, source.identity.Digest
}

func (source *scriptedSnapshotSource) OpenSnapshotStream(
	ctx context.Context,
	_ ports.SnapshotVerificationRequest,
) (io.ReadCloser, error) {
	return source.open(ctx)
}

type blockingReadCloser struct {
	entered chan struct{}
	release chan struct{}
	enter   sync.Once
	close   sync.Once
	closes  atomic.Int64
}

func (stream *blockingReadCloser) Read([]byte) (int, error) {
	stream.enter.Do(func() { close(stream.entered) })
	<-stream.release
	return 0, io.ErrClosedPipe
}

func (stream *blockingReadCloser) Close() error {
	stream.closes.Add(1)
	stream.close.Do(func() { close(stream.release) })
	return nil
}

func (stream *trackedSnapshotStream) Close() error {
	stream.once.Do(func() { stream.source.closes.Add(1) })
	stream.source.mu.RLock()
	defer stream.source.mu.RUnlock()
	return stream.source.closeErr
}

type codedFailure string

func (failure codedFailure) Error() string     { return string(failure) }
func (failure codedFailure) CauseCode() string { return string(failure) }

func TestAttestPassAndFailProduceCanonicalPortResult(t *testing.T) {
	for _, test := range []struct {
		name     string
		exitCode uint8
		verdict  ports.TestAttestationVerdict
	}{
		{name: "pass", exitCode: 0, verdict: ports.TestAttestationPassed},
		{name: "fail", exitCode: 7, verdict: ports.TestAttestationFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := validSnapshotSource()
			launcher := successfulLauncher(t, test.exitCode, testDigest("assets"))
			adapter := newTestAdapter(t, source, launcher)
			run := validRun(t, adapter.PolicyIdentity())
			result, err := adapter.Attest(context.Background(), run)
			if err != nil {
				t.Fatal(err)
			}
			if result.Verdict != test.verdict ||
				result.Tests[0].ExitCode != int(test.exitCode) ||
				result.AttestorRef != AttestorRef ||
				result.PolicyRef != PolicyRef ||
				ports.ValidateTestAttestationResult(run.Request, result) != nil {
				t.Fatalf("result=%+v", result)
			}
			if source.opens.Load() != 1 || source.closes.Load() != 1 {
				t.Fatalf("snapshot opens=%d closes=%d", source.opens.Load(), source.closes.Load())
			}
			if err := adapter.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAttestSendsCanonicalSealedInputAndExactResources(t *testing.T) {
	source := validSnapshotSource()
	assetDigest := testDigest("assets")
	var observed launcherprotocol.LaunchRequest
	launcher := &fakeLauncherClient{launch: func(
		ctx context.Context,
		request launcherprotocol.LaunchRequest,
		input *os.File,
		maxInput int64,
	) (launcherprotocol.ClientResult, error) {
		observed = request
		if maxInput <= 0 || int64(fileSize(t, input)) != maxInput {
			t.Fatalf("input size=%d max=%d", fileSize(t, input), maxInput)
		}
		assertSealedPrivateFile(t, input, 0o400)
		wire, err := rawdrive.ReadInputDrive(
			io.NewSectionReader(input, 0, maxInput),
			uint64(maxInput),
			io.Discard,
		)
		if err != nil {
			t.Fatal(err)
		}
		if wire.SubjectDigest != request.SubjectDigest ||
			wire.RunNonce != request.Nonce ||
			wire.PolicyDigest != request.PolicyDigest ||
			len(wire.RequiredTests) != 1 ||
			wire.RequiredTests[0].ToolRef != "tool:go" {
			t.Fatalf("wire=%+v request=%+v", wire, request)
		}
		return outputResult(t, request, 0, assetDigest)
	}}
	adapter := newTestAdapter(t, source, launcher)
	run := validRun(t, adapter.PolicyIdentity())
	if _, err := adapter.Attest(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	limits := validLimits()
	if observed.SubjectDigest != run.Request.SubjectDigest ||
		observed.PolicyDigest != run.Request.Subject.PolicyDigest ||
		observed.GuestMemoryMiB != limits.GuestMemoryMiB ||
		observed.MemoryMaxBytes != limits.MemoryMaxBytes ||
		observed.PIDsMax != limits.PIDsMax ||
		observed.CPUQuotaMicros != limits.CPUQuotaMicros ||
		observed.CPUPeriodMicros != launcherprotocol.CPUPeriodMicrosV0 ||
		observed.MaxCapturedOutputBytes != limits.MaxOutputBytes ||
		observed.OutputDriveBytes == 0 ||
		observed.Timeout <= 0 || observed.Timeout > limits.Timeout {
		t.Fatalf("request=%+v", observed)
	}
}

func TestAttestRejectsRequestPolicyToolSnapshotAndSourceDriftBeforeLaunch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ports.TestAttestationRun, *fakeSnapshotSource)
		code   string
	}{
		{
			name: "subject mismatch",
			mutate: func(run *ports.TestAttestationRun, _ *fakeSnapshotSource) {
				run.Snapshot.SubjectDigest = testDigest("other")
			},
			code: "test_attestor.run_subject_mismatch",
		},
		{
			name: "policy mismatch",
			mutate: func(run *ports.TestAttestationRun, _ *fakeSnapshotSource) {
				run.Request.Subject.PolicyDigest = testDigest("other")
				run.Request.SubjectDigest = ports.TestSubjectDigest(run.Request.Subject)
				run.Snapshot.Subject = run.Request.Subject
				run.Snapshot.SubjectDigest = run.Request.SubjectDigest
			},
			code: codePolicyMismatch,
		},
		{
			name: "tool unsupported",
			mutate: func(run *ports.TestAttestationRun, _ *fakeSnapshotSource) {
				ref := run.Request.RequiredTests[0].Ref()
				tool, _ := goal.NewToolRef("tool:shell")
				spec, _ := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
					Ref: ref, ToolRef: tool, Arguments: []string{"test"}, WorkingDirectory: ".",
				})
				run.Request.RequiredTests = []goal.RequiredTestSpec{spec}
				run.Request.Subject.RequiredTestsDigest = goal.RequiredTestsDigest(run.Request.RequiredTests)
				run.Request.SubjectDigest = ports.TestSubjectDigest(run.Request.Subject)
				run.Snapshot.Subject = run.Request.Subject
				run.Snapshot.SubjectDigest = run.Request.SubjectDigest
			},
			code: codeToolUnsupported,
		},
		{
			name: "source identity drift",
			mutate: func(_ *ports.TestAttestationRun, source *fakeSnapshotSource) {
				source.mu.Lock()
				source.digest = testDigest("drift")
				source.mu.Unlock()
			},
			code: codePolicyMismatch,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := validSnapshotSource()
			var launches atomic.Int64
			launcher := &fakeLauncherClient{launch: func(
				context.Context,
				launcherprotocol.LaunchRequest,
				*os.File,
				int64,
			) (launcherprotocol.ClientResult, error) {
				launches.Add(1)
				return launcherprotocol.ClientResult{}, errors.New("unexpected")
			}}
			adapter := newTestAdapter(t, source, launcher)
			run := validRun(t, adapter.PolicyIdentity())
			test.mutate(&run, source)
			_, err := adapter.Attest(context.Background(), run)
			if code := ports.TestAttestorContractErrorCode(err); code != test.code {
				t.Fatalf("code=%q err=%v want=%q", code, err, test.code)
			}
			if launches.Load() != 0 {
				t.Fatalf("launches=%d", launches.Load())
			}
		})
	}
}

func TestAttestBindsAndRevalidatesLauncherAndSnapshotIdentity(t *testing.T) {
	t.Run("launcher drift before launch", func(t *testing.T) {
		source := validSnapshotSource()
		var launches atomic.Int64
		launcher := &fakeLauncherClient{launch: func(
			_ context.Context,
			_ launcherprotocol.LaunchRequest,
			_ *os.File,
			_ int64,
		) (launcherprotocol.ClientResult, error) {
			launches.Add(1)
			return launcherprotocol.ClientResult{}, errors.New("unexpected launch")
		}}
		adapter := newTestAdapter(t, source, launcher)
		run := validRun(t, adapter.PolicyIdentity())
		launcher.setIdentity(launcherprotocol.Identity{
			Ref: "launcher:test:v1", Digest: testDigest("launcher-drift"),
		})
		_, err := adapter.Attest(context.Background(), run)
		if ports.TestAttestorContractErrorCode(err) != codePolicyMismatch ||
			launches.Load() != 0 {
			t.Fatalf("err=%v launches=%d", err, launches.Load())
		}
	})

	for _, test := range []struct {
		name   string
		mutate func(*fakeSnapshotSource, *fakeLauncherClient)
	}{
		{
			name: "launcher drift during launch",
			mutate: func(_ *fakeSnapshotSource, launcher *fakeLauncherClient) {
				launcher.setIdentity(launcherprotocol.Identity{
					Ref: "launcher:test:v1", Digest: testDigest("launcher-during"),
				})
			},
		},
		{
			name: "snapshot drift during launch",
			mutate: func(source *fakeSnapshotSource, _ *fakeLauncherClient) {
				source.mu.Lock()
				source.digest = testDigest("snapshot-during")
				source.mu.Unlock()
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := validSnapshotSource()
			var returnedFD int
			var launcher *fakeLauncherClient
			launcher = &fakeLauncherClient{launch: func(
				_ context.Context,
				request launcherprotocol.LaunchRequest,
				_ *os.File,
				_ int64,
			) (launcherprotocol.ClientResult, error) {
				result, err := outputResult(t, request, 0, testDigest("assets"))
				returnedFD = int(result.Output.Fd())
				test.mutate(source, launcher)
				return result, err
			}}
			adapter := newTestAdapter(t, source, launcher)
			run := validRun(t, adapter.PolicyIdentity())
			_, err := adapter.Attest(context.Background(), run)
			if ports.TestAttestorContractErrorCode(err) != codePolicyMismatch {
				t.Fatalf("err=%v", err)
			}
			if _, fdErr := unix.FcntlInt(
				uintptr(returnedFD),
				unix.F_GETFD,
				0,
			); !errors.Is(fdErr, unix.EBADF) {
				t.Fatalf("output fd leaked: %v", fdErr)
			}
		})
	}
}

func TestAttestMapsLauncherPeerTimeoutAndEveryGuestFailureWithoutRawLeak(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"peer", "test_attestor.firecracker_launcher_server_unauthorized", codeLauncherIdentity},
		{"launcher timeout", "test_attestor.firecracker_launcher_execution_timeout", codeExecutionTimeout},
		{"guest input", "test_attestor.firecracker.guest_input_invalid", codeInputInvalid},
		{"guest snapshot", "test_attestor.firecracker.guest_snapshot_invalid", codeSnapshotInvalid},
		{"guest snapshot limit", "test_attestor.firecracker.guest_snapshot_limit", codeSnapshotLimit},
		{"guest materialize", "test_attestor.firecracker.guest_materialize_failed", codeSnapshotInvalid},
		{"guest tool", "test_attestor.firecracker.guest_tool_unsupported", codeToolUnsupported},
		{"guest execution", "test_attestor.firecracker.guest_execution_failed", codeExecutionFailed},
		{"guest timeout", "test_attestor.firecracker.guest_execution_timeout", codeExecutionTimeout},
		{"guest output limit", "test_attestor.firecracker.guest_output_limit", codeOutputLimit},
		{"guest no tests", "test_attestor.firecracker.guest_no_tests", "test_attestor.required_tests_invalid"},
		{"guest identity", "test_attestor.firecracker.guest_identity_failed", codeIsolationIdentity},
		{"guest network", "test_attestor.firecracker.guest_network_available", codeNetworkUnsafe},
		{"guest output write", "test_attestor.firecracker.guest_output_write_failed", codeOutputInvalid},
		{"guest scratch", "test_attestor.firecracker.guest_scratch_unavailable", codeResourceUnsafe},
		{"guest capacity", "test_attestor.firecracker.guest_scratch_capacity", codeResourceLimit},
		{"guest cleanup", "test_attestor.firecracker.guest_cleanup_failed", codeCleanupFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := validSnapshotSource()
			launcher := &fakeLauncherClient{launch: func(
				_ context.Context,
				request launcherprotocol.LaunchRequest,
				_ *os.File,
				_ int64,
			) (launcherprotocol.ClientResult, error) {
				if strings.Contains(test.raw, "_launcher_") {
					return launcherprotocol.ClientResult{}, codedFailure(test.raw)
				}
				return outputError(t, request, test.raw, testDigest("assets"))
			}}
			adapter := newTestAdapter(t, source, launcher)
			_, err := adapter.Attest(
				context.Background(),
				validRun(t, adapter.PolicyIdentity()),
			)
			if code := ports.TestAttestorContractErrorCode(err); code != test.want {
				t.Fatalf("code=%q err=%v want=%q", code, err, test.want)
			}
			if strings.Contains(err.Error(), "firecracker_launcher") ||
				strings.Contains(err.Error(), "firecracker.guest") {
				t.Fatalf("raw error leaked: %v", err)
			}
		})
	}
}

func TestAttestRejectsTamperReplayAndUntrustedOutputDescriptor(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*launcherprotocol.ClientResult, launcherprotocol.LaunchRequest)
		code   string
	}{
		{
			name: "nonce",
			mutate: func(result *launcherprotocol.ClientResult, _ launcherprotocol.LaunchRequest) {
				result.Response.Nonce = strings.Repeat("a", 64)
			},
			code: codeOutputInvalid,
		},
		{
			name: "asset",
			mutate: func(result *launcherprotocol.ClientResult, _ launcherprotocol.LaunchRequest) {
				result.Response.AssetDigest = testDigest("other-assets")
			},
			code: codeAssetMismatch,
		},
		{
			name: "digest",
			mutate: func(result *launcherprotocol.ClientResult, _ launcherprotocol.LaunchRequest) {
				result.Response.OutputDigest = testDigest("tampered")
			},
			code: codeOutputInvalid,
		},
		{
			name: "response code",
			mutate: func(result *launcherprotocol.ClientResult, _ launcherprotocol.LaunchRequest) {
				result.Response.Code = "test_attestor.failure"
			},
			code: codeOutputInvalid,
		},
		{
			name: "linked output",
			mutate: func(result *launcherprotocol.ClientResult, request launcherprotocol.LaunchRequest) {
				_ = result.Output.Close()
				file, err := os.CreateTemp(t.TempDir(), "linked-output")
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Truncate(int64(request.OutputDriveBytes)); err != nil {
					t.Fatal(err)
				}
				result.Output = file
			},
			code: codeOutputInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := validSnapshotSource()
			launcher := &fakeLauncherClient{launch: func(
				_ context.Context,
				request launcherprotocol.LaunchRequest,
				_ *os.File,
				_ int64,
			) (launcherprotocol.ClientResult, error) {
				result, err := outputResult(t, request, 0, testDigest("assets"))
				test.mutate(&result, request)
				return result, err
			}}
			adapter := newTestAdapter(t, source, launcher)
			_, err := adapter.Attest(
				context.Background(),
				validRun(t, adapter.PolicyIdentity()),
			)
			if code := ports.TestAttestorContractErrorCode(err); code != test.code {
				t.Fatalf("code=%q err=%v want=%q", code, err, test.code)
			}
		})
	}
}

func TestAttestUsesFreshNonceForReplayAndClosesEveryDescriptor(t *testing.T) {
	source := validSnapshotSource()
	var mu sync.Mutex
	var nonces []string
	var borrowedFDs, returnedFDs []int
	launcher := &fakeLauncherClient{launch: func(
		_ context.Context,
		request launcherprotocol.LaunchRequest,
		input *os.File,
		_ int64,
	) (launcherprotocol.ClientResult, error) {
		result, err := outputResult(t, request, 0, testDigest("assets"))
		mu.Lock()
		nonces = append(nonces, request.Nonce)
		borrowedFDs = append(borrowedFDs, int(input.Fd()))
		returnedFDs = append(returnedFDs, int(result.Output.Fd()))
		mu.Unlock()
		return result, err
	}}
	adapter := newTestAdapter(t, source, launcher)
	run := validRun(t, adapter.PolicyIdentity())
	for index := 0; index < 20; index++ {
		if _, err := adapter.Attest(context.Background(), run); err != nil {
			t.Fatal(err)
		}
	}
	if len(nonces) != 20 {
		t.Fatalf("nonces=%d", len(nonces))
	}
	seen := make(map[string]struct{}, len(nonces))
	for _, nonce := range nonces {
		if _, duplicate := seen[nonce]; duplicate {
			t.Fatalf("replayed nonce %q", nonce)
		}
		seen[nonce] = struct{}{}
	}
	for _, fd := range append(borrowedFDs, returnedFDs...) {
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); !errors.Is(err, unix.EBADF) {
			t.Fatalf("fd %d leaked: %v", fd, err)
		}
	}
	if source.opens.Load() != 20 || source.closes.Load() != 20 {
		t.Fatalf("snapshot opens=%d closes=%d", source.opens.Load(), source.closes.Load())
	}
}

func TestAttestTimeoutCancelCloseAndConcurrencyAreBounded(t *testing.T) {
	t.Run("snapshot open deadline", func(t *testing.T) {
		source := &scriptedSnapshotSource{
			identity: launcherprotocol.Identity{
				Ref: "snapshot:test:v1", Digest: testDigest("snapshot-open"),
			},
			open: func(ctx context.Context) (io.ReadCloser, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		}
		launcher := successfulLauncher(t, 0, testDigest("assets"))
		config := validConfig(source, launcher)
		config.Limits.Timeout = 25 * time.Millisecond
		config.Limits.CleanupTimeout = 5 * time.Millisecond
		adapter, err := newAdapter(config, 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		_, err = adapter.Attest(
			context.Background(),
			validRun(t, adapter.PolicyIdentity()),
		)
		if code := ports.TestAttestorContractErrorCode(err); code != codeExecutionTimeout {
			t.Fatalf("code=%q err=%v", code, err)
		}
	})

	t.Run("snapshot stream closes on caller cancel", func(t *testing.T) {
		stream := &blockingReadCloser{
			entered: make(chan struct{}),
			release: make(chan struct{}),
		}
		source := &scriptedSnapshotSource{
			identity: launcherprotocol.Identity{
				Ref: "snapshot:test:v1", Digest: testDigest("snapshot-blocking"),
			},
			open: func(context.Context) (io.ReadCloser, error) {
				return stream, nil
			},
		}
		adapter := newTestAdapter(
			t,
			source,
			successfulLauncher(t, 0, testDigest("assets")),
		)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, err := adapter.Attest(ctx, validRun(t, adapter.PolicyIdentity()))
			done <- err
		}()
		<-stream.entered
		cancel()
		if code := ports.TestAttestorContractErrorCode(<-done); code != codeUnavailable {
			t.Fatalf("code=%q", code)
		}
		if stream.closes.Load() != 1 {
			t.Fatalf("snapshot closes=%d", stream.closes.Load())
		}
	})

	t.Run("timeout", func(t *testing.T) {
		source := validSnapshotSource()
		launcher := &fakeLauncherClient{launch: func(
			ctx context.Context,
			_ launcherprotocol.LaunchRequest,
			_ *os.File,
			_ int64,
		) (launcherprotocol.ClientResult, error) {
			<-ctx.Done()
			return launcherprotocol.ClientResult{}, ctx.Err()
		}}
		config := validConfig(source, launcher)
		config.Limits.Timeout = 25 * time.Millisecond
		config.Limits.CleanupTimeout = 5 * time.Millisecond
		adapter, err := newAdapter(config, 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		_, err = adapter.Attest(context.Background(), validRun(t, adapter.PolicyIdentity()))
		if code := ports.TestAttestorContractErrorCode(err); code != codeExecutionTimeout {
			t.Fatalf("code=%q err=%v", code, err)
		}
	})

	t.Run("close cancels", func(t *testing.T) {
		source := validSnapshotSource()
		started := make(chan struct{})
		launcher := &fakeLauncherClient{launch: func(
			ctx context.Context,
			_ launcherprotocol.LaunchRequest,
			_ *os.File,
			_ int64,
		) (launcherprotocol.ClientResult, error) {
			close(started)
			<-ctx.Done()
			return launcherprotocol.ClientResult{}, ctx.Err()
		}}
		adapter := newTestAdapter(t, source, launcher)
		done := make(chan error, 1)
		go func() {
			_, err := adapter.Attest(
				context.Background(),
				validRun(t, adapter.PolicyIdentity()),
			)
			done <- err
		}()
		<-started
		if err := adapter.Close(); err != nil {
			t.Fatal(err)
		}
		if code := ports.TestAttestorContractErrorCode(<-done); code != codeUnavailable {
			t.Fatalf("code=%q", code)
		}
		if _, err := adapter.Attest(context.Background(), ports.TestAttestationRun{}); ports.TestAttestorContractErrorCode(err) != codeUnavailable {
			t.Fatalf("closed err=%v", err)
		}
	})

	t.Run("cleanup deadline bounds uncooperative dependency", func(t *testing.T) {
		source := validSnapshotSource()
		started := make(chan struct{})
		release := make(chan struct{})
		launcher := &fakeLauncherClient{launch: func(
			context.Context,
			launcherprotocol.LaunchRequest,
			*os.File,
			int64,
		) (launcherprotocol.ClientResult, error) {
			close(started)
			<-release
			return launcherprotocol.ClientResult{},
				codedFailure("test_attestor.firecracker_launcher_unavailable")
		}}
		config := validConfig(source, launcher)
		config.Limits.CleanupTimeout = 20 * time.Millisecond
		adapter, err := newAdapter(config, 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, attestErr := adapter.Attest(
				context.Background(),
				validRun(t, adapter.PolicyIdentity()),
			)
			done <- attestErr
		}()
		<-started
		before := time.Now()
		closeErr := adapter.Close()
		elapsed := time.Since(before)
		if ports.TestAttestorContractErrorCode(closeErr) != codeCleanupFailed ||
			elapsed < config.Limits.CleanupTimeout ||
			elapsed > 10*config.Limits.CleanupTimeout {
			t.Fatalf("close err=%v elapsed=%s", closeErr, elapsed)
		}
		select {
		case err := <-done:
			t.Fatalf("attest returned before dependency release: %v", err)
		default:
		}
		close(release)
		if code := ports.TestAttestorContractErrorCode(<-done); code != codeUnavailable {
			t.Fatalf("attest code=%q", code)
		}
		if ports.TestAttestorContractErrorCode(adapter.Close()) != codeCleanupFailed {
			t.Fatal("cleanup timeout was not stable")
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		source := validSnapshotSource()
		var active, maximum atomic.Int64
		launcher := &fakeLauncherClient{launch: func(
			_ context.Context,
			request launcherprotocol.LaunchRequest,
			_ *os.File,
			_ int64,
		) (launcherprotocol.ClientResult, error) {
			current := active.Add(1)
			for {
				seen := maximum.Load()
				if current <= seen || maximum.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			active.Add(-1)
			return outputResult(t, request, 0, testDigest("assets"))
		}}
		config := validConfig(source, launcher)
		config.Limits.MaxConcurrentRuns = 2
		adapter, err := newAdapter(config, 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		run := validRun(t, adapter.PolicyIdentity())
		var wait sync.WaitGroup
		failures := make(chan error, 16)
		for index := 0; index < 16; index++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				_, attestErr := adapter.Attest(context.Background(), run)
				failures <- attestErr
			}()
		}
		wait.Wait()
		close(failures)
		for attestErr := range failures {
			if attestErr != nil {
				t.Fatal(attestErr)
			}
		}
		if maximum.Load() != 2 {
			t.Fatalf("max concurrent=%d", maximum.Load())
		}
	})
}

func TestCloseAccountsForAttestBlockedDuringBeginIdentityValidation(t *testing.T) {
	t.Run("waits for registered attest", func(t *testing.T) {
		source := validSnapshotSource()
		launcher := &blockingIdentityLauncher{
			identity: launcherprotocol.Identity{
				Ref: "launcher:test:v1", Digest: testDigest("launcher"),
			},
			blockOnCall: 2,
			entered:     make(chan struct{}),
			release:     make(chan struct{}),
		}
		defer close(launcher.release)
		config := validConfig(source, launcher)
		config.Limits.CleanupTimeout = time.Second
		adapter, err := newAdapter(config, 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		run := validRun(t, adapter.PolicyIdentity())
		attestDone := make(chan error, 1)
		go func() {
			_, attestErr := adapter.Attest(context.Background(), run)
			attestDone <- attestErr
		}()
		<-launcher.entered

		closeDone := make(chan error, 1)
		go func() { closeDone <- adapter.Close() }()
		waitForAdapterClosed(t, adapter)
		adapter.mu.Lock()
		active := adapter.active
		adapter.mu.Unlock()
		if active != 1 {
			t.Fatalf("active=%d want=1", active)
		}
		select {
		case closeErr := <-closeDone:
			t.Fatalf("Close returned before registered Attest drained: %v", closeErr)
		default:
		}

		launcher.release <- struct{}{}
		if code := ports.TestAttestorContractErrorCode(<-attestDone); code != codeUnavailable {
			t.Fatalf("attest code=%q", code)
		}
		if closeErr := <-closeDone; closeErr != nil {
			t.Fatalf("close err=%v", closeErr)
		}
	})

	t.Run("cleanup timeout remains bounded", func(t *testing.T) {
		source := validSnapshotSource()
		launcher := &blockingIdentityLauncher{
			identity: launcherprotocol.Identity{
				Ref: "launcher:test:v1", Digest: testDigest("launcher"),
			},
			blockOnCall: 2,
			entered:     make(chan struct{}),
			release:     make(chan struct{}),
		}
		defer close(launcher.release)
		config := validConfig(source, launcher)
		config.Limits.CleanupTimeout = 20 * time.Millisecond
		adapter, err := newAdapter(config, 1000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		run := validRun(t, adapter.PolicyIdentity())
		attestDone := make(chan error, 1)
		go func() {
			_, attestErr := adapter.Attest(context.Background(), run)
			attestDone <- attestErr
		}()
		<-launcher.entered

		before := time.Now()
		closeErr := adapter.Close()
		elapsed := time.Since(before)
		if ports.TestAttestorContractErrorCode(closeErr) != codeCleanupFailed ||
			elapsed < config.Limits.CleanupTimeout ||
			elapsed > 10*config.Limits.CleanupTimeout {
			t.Fatalf("close err=%v elapsed=%s", closeErr, elapsed)
		}
		select {
		case attestErr := <-attestDone:
			t.Fatalf("Attest returned while launcher identity remained blocked: %v", attestErr)
		default:
		}

		launcher.release <- struct{}{}
		if code := ports.TestAttestorContractErrorCode(<-attestDone); code != codeUnavailable {
			t.Fatalf("attest code=%q", code)
		}
		if ports.TestAttestorContractErrorCode(adapter.Close()) != codeCleanupFailed {
			t.Fatal("cleanup timeout was not stable")
		}
	})
}

func waitForAdapterClosed(t *testing.T, adapter *Adapter) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		adapter.mu.Lock()
		closed := adapter.closed
		adapter.mu.Unlock()
		if closed {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("Close did not mark adapter closed")
		default:
		}
		time.Sleep(time.Millisecond)
	}
}

func TestConfigAndPolicyBindAssetSourceAndEveryResource(t *testing.T) {
	source := validSnapshotSource()
	launcher := successfulLauncher(t, 0, testDigest("assets"))
	base := validConfig(source, launcher)
	first, err := newAdapter(base, 1000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	basePolicy := first.PolicyIdentity()
	mutations := []func(*Config){
		func(config *Config) { config.ExpectedAssetDigest = testDigest("other") },
		func(config *Config) { config.Limits.Timeout++ },
		func(config *Config) { config.Limits.CleanupTimeout++ },
		func(config *Config) { config.Limits.MaxOutputBytes++ },
		func(config *Config) { config.Limits.MaxSubjectBytes++ },
		func(config *Config) { config.Limits.MaxConcurrentRuns++ },
		func(config *Config) { config.Limits.GuestMemoryMiB++ },
		func(config *Config) { config.Limits.MemoryMaxBytes++ },
		func(config *Config) { config.Limits.PIDsMax++ },
		func(config *Config) { config.Limits.CPUQuotaMicros++ },
	}
	for index, mutate := range mutations {
		changed := validConfig(source, launcher)
		mutate(&changed)
		adapter, newErr := newAdapter(changed, 1000, 1000)
		if newErr != nil {
			t.Fatalf("mutation %d: %v", index, newErr)
		}
		if adapter.PolicyIdentity().Digest == basePolicy.Digest {
			t.Fatalf("mutation %d did not bind policy", index)
		}
	}
	otherLauncher := successfulLauncher(t, 0, testDigest("assets"))
	otherLauncher.setIdentity(launcherprotocol.Identity{
		Ref: "launcher:test:v1", Digest: testDigest("launcher-other"),
	})
	changedLauncher, err := newAdapter(
		validConfig(source, otherLauncher),
		1000,
		1000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if changedLauncher.PolicyIdentity().Digest == basePolicy.Digest {
		t.Fatal("launcher identity did not bind policy")
	}
	source.mu.Lock()
	source.digest = testDigest("source-other")
	source.mu.Unlock()
	changed, err := newAdapter(validConfig(source, launcher), 1000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if changed.PolicyIdentity().Digest == basePolicy.Digest {
		t.Fatal("source identity did not bind policy")
	}
}

func TestAdapterRejectsPrivilegedHostIdentity(t *testing.T) {
	source := validSnapshotSource()
	config := validConfig(source, successfulLauncher(t, 0, testDigest("assets")))
	for _, identity := range []struct {
		name string
		euid int
		egid int
	}{
		{name: "root euid", euid: 0, egid: 1000},
		{name: "root egid", euid: 1000, egid: 0},
	} {
		t.Run(identity.name, func(t *testing.T) {
			adapter, err := newAdapter(config, identity.euid, identity.egid)
			if adapter != nil ||
				ports.TestAttestorContractErrorCode(err) != codeIsolationIdentity {
				t.Fatalf("adapter=%v err=%v", adapter, err)
			}
		})
	}
}

func TestRawFailureTranslationIsTotalAndNeverLeaksTransportCodes(t *testing.T) {
	rawCodes := []string{
		"test_attestor.firecracker_launcher_unavailable",
		"test_attestor.firecracker_launcher_config_invalid",
		"test_attestor.firecracker_launcher_privilege_required",
		"test_attestor.firecracker_launcher_peer_unauthorized",
		"test_attestor.firecracker_launcher_server_unauthorized",
		"test_attestor.firecracker_launcher_protocol_invalid",
		"test_attestor.firecracker_launcher_descriptor_invalid",
		"test_attestor.firecracker_launcher_input_invalid",
		"test_attestor.firecracker_launcher_output_invalid",
		"test_attestor.firecracker_launcher_assets_unsafe",
		"test_attestor.firecracker_launcher_network_unsafe",
		"test_attestor.firecracker_launcher_runtime_root_unsafe",
		"test_attestor.firecracker_launcher_resource_unsafe",
		"test_attestor.firecracker_launcher_execution_failed",
		"test_attestor.firecracker_launcher_execution_timeout",
		"test_attestor.firecracker_launcher_cleanup_failed",
		"test_attestor.firecracker_launcher_response_invalid",
		rawdrive.CodeInvalid,
		rawdrive.CodeLimit,
		rawdrive.CodeTruncated,
		rawdrive.CodeTampered,
		rawdrive.CodeTrailingData,
		rawdrive.CodeAlignment,
		rawdrive.CodeIO,
		rawdrive.CodeMetadataInvalid,
		rawdrive.CodeSnapshotInvalid,
		rawdrive.CodeOutputInvalid,
		rawdrive.CodeOutputIncomplete,
		"test_attestor.firecracker.guest_input_invalid",
		"test_attestor.firecracker.guest_snapshot_invalid",
		"test_attestor.firecracker.guest_snapshot_limit",
		"test_attestor.firecracker.guest_materialize_failed",
		"test_attestor.firecracker.guest_tool_unsupported",
		"test_attestor.firecracker.guest_execution_failed",
		"test_attestor.firecracker.guest_execution_timeout",
		"test_attestor.firecracker.guest_output_limit",
		"test_attestor.firecracker.guest_no_tests",
		"test_attestor.firecracker.guest_identity_failed",
		"test_attestor.firecracker.guest_network_available",
		"test_attestor.firecracker.guest_output_write_failed",
		"test_attestor.firecracker.guest_scratch_unavailable",
		"test_attestor.firecracker.guest_scratch_capacity",
		"test_attestor.firecracker.guest_cleanup_failed",
	}
	if len(rawCodes) != len(rawCauseTranslations) {
		t.Fatalf("declared=%d translated=%d", len(rawCodes), len(rawCauseTranslations))
	}
	for _, raw := range rawCodes {
		translated, found := rawCauseTranslations[raw]
		if !found || translated == "" ||
			strings.Contains(translated, "firecracker") ||
			strings.Contains(translated, "raw_drive") {
			t.Fatalf("raw=%q translated=%q found=%v", raw, translated, found)
		}
		if code := ports.TestAttestorContractErrorCode(translateFailure(codedFailure(raw))); code != translated {
			t.Fatalf("raw=%q code=%q want=%q", raw, code, translated)
		}
	}
	if code := ports.TestAttestorContractErrorCode(
		translateFailure(codedFailure("test_attestor.future_raw")),
	); code != codeUnavailable {
		t.Fatalf("unknown code=%q", code)
	}
}

func TestExternalContractErrorsAreRebuiltThroughCauseAllowlist(t *testing.T) {
	known := &ports.TestAttestorContractError{Code: codeExecutionFailed}
	translated := translateFailure(known)
	if translated == known ||
		ports.TestAttestorContractErrorCode(translated) != codeExecutionFailed {
		t.Fatalf("known=%p translated=%T %v", known, translated, translated)
	}
	unknown := &ports.TestAttestorContractError{
		Code: "test_attestor.secret_external_detail",
	}
	translated = translateFailure(unknown)
	if translated == unknown ||
		ports.TestAttestorContractErrorCode(translated) != codeUnavailable ||
		strings.Contains(translated.Error(), "secret_external_detail") {
		t.Fatalf("unknown=%p translated=%T %v", unknown, translated, translated)
	}

	source := validSnapshotSource()
	launcher := &fakeLauncherClient{launch: func(
		context.Context,
		launcherprotocol.LaunchRequest,
		*os.File,
		int64,
	) (launcherprotocol.ClientResult, error) {
		return launcherprotocol.ClientResult{}, unknown
	}}
	adapter := newTestAdapter(t, source, launcher)
	_, err := adapter.Attest(
		context.Background(),
		validRun(t, adapter.PolicyIdentity()),
	)
	if ports.TestAttestorContractErrorCode(err) != codeUnavailable ||
		strings.Contains(err.Error(), "secret_external_detail") {
		t.Fatalf("adapter leaked external cause: %v", err)
	}
}

func successfulLauncher(
	t *testing.T,
	exitCode uint8,
	assetDigest string,
) *fakeLauncherClient {
	t.Helper()
	return &fakeLauncherClient{launch: func(
		_ context.Context,
		request launcherprotocol.LaunchRequest,
		_ *os.File,
		_ int64,
	) (launcherprotocol.ClientResult, error) {
		return outputResult(t, request, exitCode, assetDigest)
	}}
}

func outputResult(
	t *testing.T,
	request launcherprotocol.LaunchRequest,
	exitCode uint8,
	assetDigest string,
) (launcherprotocol.ClientResult, error) {
	t.Helper()
	verdict := rawdrive.VerdictPassed
	if exitCode != 0 {
		verdict = rawdrive.VerdictFailed
	}
	return encodedOutput(t, request, rawdrive.Output{
		SubjectDigest: request.SubjectDigest,
		RunNonce:      request.Nonce,
		Verdict:       verdict,
		Outcomes: []rawdrive.RequiredTestOutcome{{
			RequiredTestRef: "required-test:go",
			ExitCode:        exitCode,
			OutputDigest:    testDigest("test-output"),
		}},
	}, assetDigest)
}

func outputError(
	t *testing.T,
	request launcherprotocol.LaunchRequest,
	code, assetDigest string,
) (launcherprotocol.ClientResult, error) {
	t.Helper()
	return encodedOutput(t, request, rawdrive.Output{
		SubjectDigest: request.SubjectDigest,
		RunNonce:      request.Nonce,
		ErrorCode:     code,
	}, assetDigest)
}

func encodedOutput(
	t *testing.T,
	request launcherprotocol.LaunchRequest,
	output rawdrive.Output,
	assetDigest string,
) (launcherprotocol.ClientResult, error) {
	t.Helper()
	fd, err := unix.MemfdCreate(
		"orquesta-firecracker-client-test-output",
		unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING,
	)
	if err != nil {
		t.Fatal(err)
	}
	file := os.NewFile(uintptr(fd), "orquesta-firecracker-client-test-output")
	if err := unix.Ftruncate(fd, int64(request.OutputDriveBytes)); err != nil {
		t.Fatal(err)
	}
	if err := rawdrive.WriteOutputDrive(file, request.OutputDriveBytes, output); err != nil {
		t.Fatal(err)
	}
	if unix.Fchmod(fd, 0o400) != nil {
		t.Fatal("chmod output")
	}
	if _, err := unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, sealedFileSeals); err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	if _, err := io.Copy(
		digest,
		io.NewSectionReader(file, 0, int64(request.OutputDriveBytes)),
	); err != nil {
		t.Fatal(err)
	}
	return launcherprotocol.ClientResult{
		Response: launcherprotocol.LaunchResponse{
			Nonce:        request.Nonce,
			Code:         "ok",
			OutputDigest: hex.EncodeToString(digest.Sum(nil)),
			AssetDigest:  assetDigest,
		},
		Output: file,
	}, nil
}

func newTestAdapter(
	t *testing.T,
	source SnapshotStreamSource,
	launcher launcherprotocol.Client,
) *Adapter {
	t.Helper()
	adapter, err := newAdapter(validConfig(source, launcher), 1000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Error(err)
		}
	})
	return adapter
}

func validConfig(
	source SnapshotStreamSource,
	launcher launcherprotocol.Client,
) Config {
	return Config{
		Launcher:            launcher,
		SnapshotSource:      source,
		Now:                 func() time.Time { return adapterTestNow },
		ExpectedAssetDigest: testDigest("assets"),
		Limits:              validLimits(),
	}
}

func validLimits() Limits {
	return Limits{
		Timeout:           time.Minute,
		CleanupTimeout:    time.Second,
		MaxOutputBytes:    1 << 20,
		MaxSubjectBytes:   8 << 20,
		MaxConcurrentRuns: 4,
		GuestMemoryMiB:    4096,
		MemoryMaxBytes:    5 << 30,
		PIDsMax:           512,
		CPUQuotaMicros:    200_000,
	}
}

func validSnapshotSource() *fakeSnapshotSource {
	return &fakeSnapshotSource{
		ref:     "snapshot:gitlocal:v1",
		digest:  testDigest("snapshot-source"),
		payload: append(append([]byte(nil), snapshotMagic...), []byte("authorized-snapshot")...),
	}
}

func validRun(t *testing.T, policy PolicyIdentity) ports.TestAttestationRun {
	t.Helper()
	goalRef, _ := goal.NewGoalRef("goal:test")
	item, _ := goal.NewWorkItemRef("work-item:test")
	execution, _ := goal.NewExecutionRef("execution:test")
	workspace, _ := ports.NewExecutionWorkspaceRef("workspace:test")
	change, _ := ports.NewChangeSetRef("change:test")
	repository, _ := identity.NewRepositoryRef("repository:test")
	ref, _ := goal.NewRequiredTestRef("required-test:go")
	tool, _ := goal.NewToolRef("tool:go")
	spec, _ := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: ref, ToolRef: tool, Arguments: []string{"test", "./..."}, WorkingDirectory: ".",
	})
	writeSet := []string{"internal"}
	subject := ports.TestSubject{
		GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution,
		ExecutionAttempt: 1, PlanGeneration: 1, WorkItemGeneration: 1,
		AppSpecGeneration: 1, AppSpecHash: testDigest("app"),
		WorkspaceRef: workspace, WorkspaceBindingDigest: testDigest("workspace"),
		ChangeSetRef: change, ChangeSetDigest: testDigest("change"),
		RepositoryRef: repository, ObjectFormat: ports.GitObjectFormatSHA1,
		BaseOID: strings.Repeat("1", 40), ParentOID: strings.Repeat("2", 40),
		HeadOID: strings.Repeat("3", 40), TreeOID: strings.Repeat("4", 40),
		DiffDigest: testDigest("diff"), WriteSetDigest: ports.WorkspaceWriteSetDigest(writeSet),
		RequiredTestsDigest: goal.RequiredTestsDigest([]goal.RequiredTestSpec{spec}),
		PolicyRef:           policy.Ref, PolicyDigest: policy.Digest,
	}
	request := ports.TestAttestationRequest{
		Subject: subject, SubjectDigest: ports.TestSubjectDigest(subject),
		RequiredTests:  []goal.RequiredTestSpec{spec},
		IdempotencyKey: "attest:test", RequestedAt: adapterTestNow,
	}
	return ports.TestAttestationRun{
		Request: request,
		Snapshot: ports.SnapshotVerificationRequest{
			Subject: subject, SubjectDigest: request.SubjectDigest,
			ChangedPaths: []string{"internal/main.go"}, WriteSet: writeSet,
		},
	}
}

func assertSealedPrivateFile(t *testing.T, file *os.File, mode uint32) {
	t.Helper()
	var stat unix.Stat_t
	if unix.Fstat(int(file.Fd()), &stat) != nil ||
		stat.Nlink != 0 || stat.Mode&0o777 != mode {
		t.Fatalf("stat=%+v", stat)
	}
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&sealedFileSeals != sealedFileSeals {
		t.Fatalf("seals=%x err=%v", seals, err)
	}
}

func fileSize(t *testing.T, file *os.File) uint64 {
	t.Helper()
	var stat unix.Stat_t
	if unix.Fstat(int(file.Fd()), &stat) != nil {
		t.Fatal("fstat")
	}
	return uint64(stat.Size)
}

func testDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
