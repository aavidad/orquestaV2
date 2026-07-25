//go:build linux

package bubblewrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"syscall"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func contextFailure(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &Error{Code: CodeExecutionTimeout}
	}
	return &Error{Code: CodeUnavailable}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) && exit.ExitCode() >= 0 {
		return exit.ExitCode()
	}
	return 255
}

type cappedOutputs struct {
	mu             sync.Mutex
	stdout, stderr bytes.Buffer
	remaining      int64
	overflow       chan struct{}
	once           sync.Once
}

type cappedWriter struct {
	outputs *cappedOutputs
	buffer  *bytes.Buffer
}

func (output cappedWriter) Write(value []byte) (int, error) {
	output.outputs.mu.Lock()
	defer output.outputs.mu.Unlock()
	remaining := output.outputs.remaining
	keep := min(int64(len(value)), remaining)
	if keep > 0 {
		_, _ = output.buffer.Write(value[:keep])
		output.outputs.remaining -= keep
	}
	if int64(len(value)) > remaining {
		output.outputs.once.Do(func() { close(output.outputs.overflow) })
	}
	return len(value), nil
}

func (output *cappedOutputs) digest() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	digest := sha256.New()
	for _, value := range []string{"orquesta.test-output.v1", output.stdout.String(), output.stderr.String()} {
		writeDigestField(digest, value)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func (adapter *Adapter) runTest(ctx context.Context, snapshot *sandboxSnapshot, spec goal.RequiredTestSpec) (outcome ports.RequiredTestOutcome, resultErr error) {
	leaf, err := adapter.cgroups.newLeaf(adapter.config.Limits)
	if err != nil {
		return outcome, err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), adapter.config.Limits.CleanupTimeout)
		defer cancel()
		if leaf.close(cleanupCtx) != nil {
			outcome, resultErr = ports.RequiredTestOutcome{}, &Error{Code: CodeCleanupFailed}
		}
	}()
	command, output, arguments, err := adapter.command(snapshot, spec)
	if err != nil {
		return outcome, err
	}
	if leaf.apply(command) != nil {
		if arguments.Close() != nil {
			return outcome, &Error{Code: CodeCleanupFailed}
		}
		return outcome, &Error{Code: CodeExecutionFailed}
	}
	if command.Start() != nil {
		if arguments.Close() != nil {
			return outcome, &Error{Code: CodeCleanupFailed}
		}
		return outcome, &Error{Code: CodeExecutionFailed}
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	if arguments.Close() != nil {
		_ = stopSandbox(command, leaf, done, adapter.config.Limits.CleanupTimeout)
		return outcome, &Error{Code: CodeCleanupFailed}
	}
	waitErr, failure := waitSandboxCompletion(ctx, done, output.overflow)
	if failure != nil {
		if stopErr := stopSandbox(command, leaf, done, adapter.config.Limits.CleanupTimeout); stopErr != nil {
			failure = &Error{Code: CodeCleanupFailed}
		}
	}
	exceeded, observationErr := leaf.exceeded()
	if errors.Is(waitErr, exec.ErrWaitDelay) {
		return outcome, &Error{Code: CodeCleanupFailed}
	}
	if ErrorCode(failure) == CodeCleanupFailed {
		return outcome, failure
	}
	if observationErr != nil {
		return outcome, resourceError()
	}
	if failure != nil {
		return outcome, failure
	}
	if exceeded {
		return outcome, &Error{Code: CodeResourceLimit}
	}
	return ports.RequiredTestOutcome{RequiredTestRef: spec.Ref(), ExitCode: exitCode(waitErr), OutputDigest: output.digest()}, nil
}

func waitSandboxCompletion(ctx context.Context, done <-chan error, overflow <-chan struct{}) (waitErr, failure error) {
	select {
	case waitErr = <-done:
		select {
		case <-overflow:
			failure = &Error{Code: CodeOutputLimit}
		default:
		}
	case <-ctx.Done():
		failure = contextFailure(ctx)
	case <-overflow:
		failure = &Error{Code: CodeOutputLimit}
	}
	return waitErr, failure
}

func (adapter *Adapter) command(snapshot *sandboxSnapshot, spec goal.RequiredTestSpec) (*exec.Cmd, *cappedOutputs, *sealedArguments, error) {
	metadata, err := validatePinned(adapter.inputs.bubblewrap, adapter.inputs.bubblewrapMeta.uid, false)
	if err != nil || metadata != adapter.inputs.bubblewrapMeta {
		return nil, nil, nil, &Error{Code: CodeBinaryUnsafe}
	}
	if err := adapter.inputs.validateToolchain(); err != nil {
		return nil, nil, nil, err
	}
	arguments, err := newSealedArguments(adapter.config.Limits.MaxSubjectBytes)
	if err != nil {
		return nil, nil, nil, err
	}
	goRun, err := reopenSealedExecutable(adapter.inputs.goBin)
	if err != nil {
		_ = arguments.Close()
		return nil, nil, nil, &Error{Code: CodeToolchainUnsafe}
	}
	arguments.auxiliary = append(arguments.auxiliary, goRun)
	files := []*os.File{adapter.inputs.bubblewrap, adapter.inputs.root, goRun}
	fail := func(err error) (*exec.Cmd, *cappedOutputs, *sealedArguments, error) {
		_ = arguments.Close()
		return nil, nil, nil, err
	}
	if err := appendSandboxPreamble(arguments); err != nil {
		return fail(err)
	}
	for _, directory := range snapshot.directories {
		if err := arguments.add("--dir", path.Join(workspaceMount, directory)); err != nil {
			return fail(err)
		}
	}
	for _, entry := range snapshot.entries {
		if entry.file == nil {
			err = arguments.add("--symlink", entry.target, path.Join(workspaceMount, entry.path))
		} else {
			files = append(files, entry.file)
			err = arguments.add("--perms", modePermissions(entry.mode), "--ro-bind-data",
				strconv.Itoa(len(files)+2), path.Join(workspaceMount, entry.path))
		}
		if err != nil {
			return fail(err)
		}
	}
	if err := appendSandboxTail(arguments, adapter.config.Limits, spec); err != nil {
		return fail(err)
	}
	if err := arguments.seal(); err != nil {
		return fail(err)
	}
	files = append(files, arguments.File)
	argumentFD := strconv.Itoa(len(files) + 2)
	command := exec.Command("/proc/self/fd/3", append([]string{"--args", argumentFD, "--", goCommand}, spec.Arguments()...)...)
	command.Args[0], command.Env, command.ExtraFiles = adapter.config.BubblewrapCommand, []string{}, files
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	command.WaitDelay = adapter.config.Limits.CleanupTimeout
	output := &cappedOutputs{remaining: adapter.config.Limits.MaxOutputBytes, overflow: make(chan struct{})}
	command.Stdout, command.Stderr = cappedWriter{output, &output.stdout}, cappedWriter{output, &output.stderr}
	return command, output, arguments, nil
}
