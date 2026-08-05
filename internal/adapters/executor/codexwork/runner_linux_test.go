//go:build linux

package codexwork

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	protocol "orquesta/internal/adapters/protocol/codexwork"
)

const helperSeparator = "--codexwork-helper"

func TestRunnerWritesOnlyRawArtifactAndReapsCleanChild(t *testing.T) {
	artifact := "diff --git a/file b/file\n+contenido exacto\n"
	runner, capture := helperRunner(t, "success", artifact, protocol.MaxPacketBytesV1)
	var output bytes.Buffer
	if err := runner.Run(context.Background(), packetReader(t, 2*time.Second), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != artifact {
		t.Fatalf("output=%q want=%q", output.String(), artifact)
	}
	if strings.Contains(output.String(), `"artifact"`) {
		t.Fatal("se publico el wrapper JSON")
	}
	assertReaped(t, capture.command())
}

func TestRunnerRejectsMalformedOversizeAndTrailingPacketsBeforeStart(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		code protocol.Code
	}{
		{name: "malformed", raw: []byte(`{"prompt":"secret"`), code: protocol.CodePacketMalformed},
		{name: "oversize", raw: bytes.Repeat([]byte("x"), protocol.MaxPacketBytesV1+1), code: protocol.CodePacketTooLarge},
		{name: "trailing", raw: append(packetBytes(t, time.Second), []byte(` {}`)...), code: protocol.CodePacketMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var starts int
			runner, err := New(Config{
				Command: "unused", Arguments: []string{"unused"},
				MaxPacketBytes: protocol.MaxPacketBytesV1, MaxFrameBytes: protocol.MaxPacketBytesV1,
				MaxDiagnosticBytes: 1024, CleanupTimeout: 100 * time.Millisecond,
				CommandFactory: func(context.Context, string, ...string) *exec.Cmd {
					starts++
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			err = runner.Run(context.Background(), bytes.NewReader(test.raw), io.Discard)
			if protocol.ErrorCode(err) != test.code {
				t.Fatalf("code=%q err=%v", protocol.ErrorCode(err), err)
			}
			if starts != 0 {
				t.Fatalf("se inicio el hijo %d veces", starts)
			}
		})
	}
}

func TestRunnerRejectsOversizeFrameAndReapsChild(t *testing.T) {
	runner, capture := helperRunner(t, "oversize-frame", "", 256)
	err := runner.Run(context.Background(), packetReader(t, 2*time.Second), io.Discard)
	if protocol.ErrorCode(err) != protocol.CodeFrameTooLarge {
		t.Fatalf("code=%q err=%v", protocol.ErrorCode(err), err)
	}
	assertReaped(t, capture.command())
}

func TestRunnerRejectsInvalidProtocolWithoutLeakingPayload(t *testing.T) {
	runner, capture := helperRunner(t, "invalid-protocol", "", protocol.MaxPacketBytesV1)
	err := runner.Run(context.Background(), packetReader(t, 2*time.Second), io.Discard)
	if protocol.ErrorCode(err) != protocol.CodeMethod {
		t.Fatalf("code=%q err=%v", protocol.ErrorCode(err), err)
	}
	if strings.Contains(err.Error(), "not-allowed-secret") {
		t.Fatal("el error filtro el payload remoto")
	}
	assertReaped(t, capture.command())
}

func TestRunnerTimeoutKillsAndReapsProcessGroup(t *testing.T) {
	runner, capture := helperRunner(t, "hang", "", protocol.MaxPacketBytesV1)
	started := time.Now()
	err := runner.Run(context.Background(), packetReader(t, 40*time.Millisecond), io.Discard)
	if ErrorCode(err) != CodeTimeout {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("el timeout no fue acotado")
	}
	assertReaped(t, capture.command())
}

func TestRunnerParentCancellationKillsAndReapsProcess(t *testing.T) {
	runner, capture := helperRunner(t, "hang", "", protocol.MaxPacketBytesV1)
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan struct{})
	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
		close(cancelled)
	}()
	err := runner.Run(ctx, packetReader(t, 2*time.Second), io.Discard)
	<-cancelled
	if ErrorCode(err) != CodeCancelled {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
	assertReaped(t, capture.command())
}

func TestRunnerCancellationInterruptsInitialClosableReader(t *testing.T) {
	input := newBlockingReadCloser()
	runner, err := New(Config{
		Command: "unused", MaxPacketBytes: protocol.MaxPacketBytesV1,
		MaxFrameBytes: protocol.MaxPacketBytesV1, MaxDiagnosticBytes: 64,
		CleanupTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() { finished <- runner.Run(ctx, input, io.Discard) }()
	<-input.started
	cancel()
	select {
	case runErr := <-finished:
		if ErrorCode(runErr) != CodeCancelled {
			t.Fatalf("code=%q err=%v", ErrorCode(runErr), runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("la lectura inicial no respondio a cancelacion")
	}
	select {
	case <-input.closed:
	default:
		t.Fatal("el lector no quedo cerrado")
	}
}

func TestRunnerPreCancelledContextDoesNotReadNonClosableInput(t *testing.T) {
	runner, err := New(Config{
		Command: "unused", MaxPacketBytes: protocol.MaxPacketBytesV1,
		MaxFrameBytes: protocol.MaxPacketBytesV1, MaxDiagnosticBytes: 64,
		CleanupTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = runner.Run(ctx, panicReader{}, io.Discard)
	if ErrorCode(err) != CodeCancelled {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
}

func TestRunnerInitialDeadlineReturnsTimeoutNotInputFailure(t *testing.T) {
	input := newBlockingReadCloser()
	runner, err := New(Config{
		Command: "unused", MaxPacketBytes: protocol.MaxPacketBytesV1,
		MaxFrameBytes: protocol.MaxPacketBytesV1, MaxDiagnosticBytes: 64,
		CleanupTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err = runner.Run(ctx, input, io.Discard)
	if ErrorCode(err) != CodeTimeout {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
}

func TestRunnerInitialDeadlineInterruptsRealOSPipe(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readEnd.Close()
	defer writeEnd.Close()
	runner, err := New(Config{
		Command: "unused", MaxPacketBytes: protocol.MaxPacketBytesV1,
		MaxFrameBytes: protocol.MaxPacketBytesV1, MaxDiagnosticBytes: 64,
		CleanupTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = runner.Run(ctx, readEnd, io.Discard)
	if ErrorCode(err) != CodeTimeout {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("poll del pipe real no respondio al deadline")
	}
}

func TestRunnerInitialDeadlineInterruptsRealFIFO(t *testing.T) {
	fifoPath := t.TempDir() + "/stdin.fifo"
	if err := unix.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatal(err)
	}
	fifo, err := os.OpenFile(fifoPath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fifo.Close()
	runner, err := New(Config{
		Command: "unused", MaxPacketBytes: protocol.MaxPacketBytesV1,
		MaxFrameBytes: protocol.MaxPacketBytesV1, MaxDiagnosticBytes: 64,
		CleanupTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = runner.Run(ctx, fifo, io.Discard)
	if ErrorCode(err) != CodeTimeout {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("poll del FIFO real no respondio al deadline")
	}
}

func TestRunnerPurgesResidualDescendantBeforePublishingSuccess(t *testing.T) {
	pidPath := t.TempDir() + "/grandchild.pid"
	runner, capture := helperRunner(t, "stderr-descendant", pidPath, protocol.MaxPacketBytesV1)
	var output bytes.Buffer
	if err := runner.Run(context.Background(), packetReader(t, 2*time.Second), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "descendant-artifact" {
		t.Fatalf("artefacto=%q", output.String())
	}
	pidRaw, readErr := os.ReadFile(pidPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var pid int
	if _, scanErr := fmt.Sscanf(string(pidRaw), "%d", &pid); scanErr != nil || pid <= 0 {
		t.Fatalf("pid invalido %q: %v", pidRaw, scanErr)
	}
	assertPIDGone(t, pid)
	assertReaped(t, capture.command())
}

func TestRunnerDoesNotWaitForEOFHeldByStdoutDescendant(t *testing.T) {
	pidPath := t.TempDir() + "/grandchild-stdout.pid"
	runner, capture := helperRunner(t, "stdout-descendant", pidPath, protocol.MaxPacketBytesV1)
	var output bytes.Buffer
	started := time.Now()
	if err := runner.Run(context.Background(), packetReader(t, 2*time.Second), &output); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 1900*time.Millisecond {
		t.Fatal("el EOF heredado convirtio el trabajo valido en espera")
	}
	if output.String() != "descendant-artifact" {
		t.Fatalf("artefacto=%q", output.String())
	}
	pidRaw, readErr := os.ReadFile(pidPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var pid int
	if _, scanErr := fmt.Sscanf(string(pidRaw), "%d", &pid); scanErr != nil || pid <= 0 {
		t.Fatalf("pid invalido %q: %v", pidRaw, scanErr)
	}
	assertPIDGone(t, pid)
	assertReaped(t, capture.command())
}

func TestRunnerRejectsNonZeroChildAfterReapingIt(t *testing.T) {
	runner, capture := helperRunner(t, "nonzero", "", protocol.MaxPacketBytesV1)
	err := runner.Run(context.Background(), packetReader(t, 2*time.Second), io.Discard)
	if ErrorCode(err) != CodeChildFailed {
		t.Fatalf("code=%q err=%v", ErrorCode(err), err)
	}
	assertReaped(t, capture.command())
}

func TestRunnerDrainsButNeverExposesChildStderr(t *testing.T) {
	artifact := "artefacto"
	runner, capture := helperRunner(t, "stderr-success", artifact, protocol.MaxPacketBytesV1)
	var output bytes.Buffer
	if err := runner.Run(context.Background(), packetReader(t, 2*time.Second), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != artifact || strings.Contains(output.String(), "CHILD_SECRET") {
		t.Fatalf("salida no sellada: %q", output.String())
	}
	assertReaped(t, capture.command())
}

func TestRunnerChildReceivesOnlySealedEnvironment(t *testing.T) {
	t.Setenv("ORQUESTA_EXECUTOR_SECRET_MARKER", "NO_FILTRAR")
	runner, capture := helperRunner(t, "environment", "env-ok", protocol.MaxPacketBytesV1)
	var output bytes.Buffer
	if err := runner.Run(context.Background(), packetReader(t, 2*time.Second), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "env-ok" {
		t.Fatalf("artefacto=%q", output.String())
	}
	command := capture.command()
	if command == nil || !slices.Equal(command.Env, sealedEnvironment()) {
		t.Fatalf("env=%q want=%q", command.Env, sealedEnvironment())
	}
	for _, variable := range command.Env {
		if strings.Contains(variable, "NO_FILTRAR") || strings.HasPrefix(variable, "ORQUESTA_EXECUTOR_SECRET_MARKER=") {
			t.Fatalf("se heredo secreto: %q", variable)
		}
	}
	assertReaped(t, command)
}

func TestRunnerRejectsConfigurationThatExpandsCanonicalLimits(t *testing.T) {
	base := Config{
		Command: "agent", MaxPacketBytes: protocol.MaxPacketBytesV1,
		MaxFrameBytes: protocol.MaxPacketBytesV1, MaxDiagnosticBytes: 1, CleanupTimeout: time.Second,
	}
	mutations := []func(*Config){
		func(config *Config) { config.Command = "" },
		func(config *Config) { config.MaxPacketBytes = protocol.MaxPacketBytesV1 + 1 },
		func(config *Config) { config.MaxFrameBytes = protocol.MaxPacketBytesV1 + 1 },
		func(config *Config) { config.MaxDiagnosticBytes = 0 },
		func(config *Config) { config.CleanupTimeout = 0 },
	}
	for index, mutate := range mutations {
		config := base
		mutate(&config)
		if _, err := New(config); ErrorCode(err) != CodeConfigInvalid {
			t.Fatalf("mutation %d: %v", index, err)
		}
	}
}

func TestRunnerCanonicalPacketFrameAndOutputLimitsStayAtOneMiB(t *testing.T) {
	const oneMiB = 1 << 20
	if protocol.MaxPacketBytesV1 != oneMiB || protocol.MaxOutputBytesV1 != oneMiB {
		t.Fatalf(
			"packet/frame=%d output=%d want=%d",
			protocol.MaxPacketBytesV1, protocol.MaxOutputBytesV1, oneMiB,
		)
	}
}

type commandCapture struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

type blockingReadCloser struct {
	started   chan struct{}
	closed    chan struct{}
	startOnce sync.Once
	closeOnce sync.Once
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{started: make(chan struct{}), closed: make(chan struct{})}
}

func (reader *blockingReadCloser) Read([]byte) (int, error) {
	reader.startOnce.Do(func() { close(reader.started) })
	<-reader.closed
	return 0, os.ErrClosed
}

func (reader *blockingReadCloser) Close() error {
	reader.closeOnce.Do(func() { close(reader.closed) })
	return nil
}

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("Read no debe ejecutarse") }

func (capture *commandCapture) set(command *exec.Cmd) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	capture.cmd = command
}

func (capture *commandCapture) command() *exec.Cmd {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.cmd
}

func helperRunner(t *testing.T, mode, artifact string, maxFrame int) (Runner, *commandCapture) {
	t.Helper()
	capture := &commandCapture{}
	runner, err := New(Config{
		Command: os.Args[0],
		Arguments: []string{
			"-test.run=^TestCodexWorkExecutorHelperProcess$", "--", helperSeparator, mode, artifact,
		},
		MaxPacketBytes: protocol.MaxPacketBytesV1, MaxFrameBytes: maxFrame,
		MaxDiagnosticBytes: 64, CleanupTimeout: postKillWait,
		CommandFactory: func(ctx context.Context, command string, arguments ...string) *exec.Cmd {
			child := exec.CommandContext(ctx, command, arguments...)
			capture.set(child)
			return child
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return runner, capture
}

func packetReader(t *testing.T, timeout time.Duration) io.Reader {
	t.Helper()
	return bytes.NewReader(packetBytes(t, timeout))
}

func packetBytes(t *testing.T, timeout time.Duration) []byte {
	t.Helper()
	packet := protocol.WorkPacketV1{
		Schema: protocol.WorkPacketSchemaV1, Prompt: "prompt secreto exacto",
		Model: "gpt-5.6", Effort: "high", TokenBudget: 4096,
		MaxOutputBytes: 64 << 10, TimeBudgetMS: uint64(timeout.Milliseconds()),
		GoalRef: "goal", WorkItemRef: "work", ExecutionRef: "execution", EffectAttemptRef: "attempt",
	}
	encoded, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func assertReaped(t *testing.T, command *exec.Cmd) {
	t.Helper()
	if command == nil || command.Process == nil {
		t.Fatal("no se capturo el proceso")
	}
	deadline := time.Now().Add(time.Second)
	for {
		err := syscall.Kill(command.Process.Pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d sigue vivo: %v", command.Process.Pid, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestCodexWorkExecutorHelperProcess(t *testing.T) {
	index := -1
	for candidate, argument := range os.Args {
		if argument == helperSeparator {
			index = candidate
			break
		}
	}
	if index < 0 {
		return
	}
	if len(os.Args) <= index+2 {
		os.Exit(97)
	}
	mode, artifact := os.Args[index+1], os.Args[index+2]
	reader := bufio.NewReader(os.Stdin)
	initial, err := reader.ReadBytes('\n')
	if err != nil {
		os.Exit(96)
	}
	switch mode {
	case "hang":
		time.Sleep(30 * time.Second)
		return
	case "oversize-frame":
		_, _ = fmt.Fprintln(os.Stdout, strings.Repeat("x", 257))
		return
	case "invalid-protocol":
		_, _ = fmt.Fprintln(os.Stdout, `{"method":"not-allowed-secret","params":{}}`)
		return
	case "nonzero":
		os.Exit(9)
	case "stderr-descendant":
		grandchild := exec.Command("/bin/sleep", "30")
		grandchild.Stderr = os.Stderr
		if err := grandchild.Start(); err != nil {
			os.Exit(88)
		}
		if err := os.WriteFile(artifact, []byte(fmt.Sprint(grandchild.Process.Pid)), 0o600); err != nil {
			os.Exit(87)
		}
		artifact = "descendant-artifact"
	case "stdout-descendant":
		grandchild := exec.Command("/bin/sleep", "30")
		grandchild.Stdout = os.Stdout
		if err := grandchild.Start(); err != nil {
			os.Exit(86)
		}
		if err := os.WriteFile(artifact, []byte(fmt.Sprint(grandchild.Process.Pid)), 0o600); err != nil {
			os.Exit(85)
		}
		artifact = "descendant-artifact"
	case "stderr-success":
		_, _ = fmt.Fprintln(os.Stderr, strings.Repeat("CHILD_SECRET", 1024))
	case "environment":
		if !slices.Equal(os.Environ(), sealedEnvironment()) || os.Getenv("ORQUESTA_EXECUTOR_SECRET_MARKER") != "" {
			os.Exit(84)
		}
	}
	serveSuccessfulProtocol(reader, initial, artifact)
}

func assertPIDGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid residual %d sigue vivo: %v", pid, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func serveSuccessfulProtocol(reader *bufio.Reader, initial []byte, artifact string) {
	decode := func(line []byte) map[string]any {
		var value map[string]any
		if json.Unmarshal(line, &value) != nil {
			os.Exit(94)
		}
		return value
	}
	read := func() map[string]any {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			os.Exit(95)
		}
		return decode(line)
	}
	encoder := json.NewEncoder(os.Stdout)
	request := decode(initial)
	if request["method"] != "initialize" || request["id"] == nil {
		os.Exit(93)
	}
	_ = encoder.Encode(map[string]any{
		"id": request["id"],
		"result": map[string]any{
			"userAgent": "helper", "codexHome": "/sealed",
			"platformFamily": "unix", "platformOs": "linux",
		},
	})
	initialized := read()
	if initialized["method"] != "initialized" {
		os.Exit(92)
	}
	request = read()
	if request["method"] != "thread/start" || request["id"] == nil {
		os.Exit(91)
	}
	_ = encoder.Encode(map[string]any{
		"id": request["id"], "result": map[string]any{"thread": map[string]any{"id": "thread-1"}},
	})
	request = read()
	if request["method"] != "turn/start" || request["id"] == nil {
		os.Exit(90)
	}
	_ = encoder.Encode(map[string]any{
		"id": request["id"],
		"result": map[string]any{
			"turn": map[string]any{"id": "turn-1", "status": "inProgress", "items": []any{}},
		},
	})
	answer, _ := json.Marshal(map[string]string{"artifact": artifact})
	_ = encoder.Encode(map[string]any{
		"method": "turn/completed",
		"params": map[string]any{
			"threadId": "thread-1",
			"turn": map[string]any{
				"id": "turn-1", "status": "completed",
				"items": []any{map[string]any{
					"id": "item-1", "type": "agentMessage", "text": string(answer), "phase": "final_answer",
				}},
			},
		},
	})
	// El ejecutor debe cerrar stdin tras el resultado terminal y el helper debe
	// terminar solo despues de observar ese EOF.
	if _, err := reader.ReadByte(); !errors.Is(err, io.EOF) {
		os.Exit(89)
	}
}
