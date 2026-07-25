package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"orquesta/internal/ports"
)

const (
	outputSchemaFileName = "output-schema.json"
	lastMessageFileName  = "last-message.json"
)

var outputSchema = []byte(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["artifact"],
  "properties": {
    "artifact": {"type": "string"}
  }
}
`)

type modelResult struct {
	Artifact string `json:"artifact"`
}

type cappedDiagnostic struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	maximum   int64
	truncated bool
}

func (writer *cappedDiagnostic) Write(payload []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	written := len(payload)
	remaining := writer.maximum - int64(writer.buffer.Len())
	if remaining <= 0 {
		if len(payload) > 0 {
			writer.truncated = true
		}
		return written, nil
	}
	keep := len(payload)
	if int64(keep) > remaining {
		keep = int(remaining)
		writer.truncated = true
	}
	_, _ = writer.buffer.Write(payload[:keep])
	return written, nil
}

func (writer *cappedDiagnostic) snapshot() ([]byte, bool) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return append([]byte(nil), writer.buffer.Bytes()...), writer.truncated
}

func (adapter *Adapter) startExecutionLocked(callerContext context.Context, request ports.AgentLaunchRequest, state *executionState, environment []string, session *resolvedSession) {
	prompt, err := adapter.renderAgentPrompt(request)
	if err != nil {
		adapter.finishWithoutProcessLocked(state, ErrorCode(err), []byte(err.Error()))
		return
	}
	if err := adapter.prepareRuntimeFiles(state.runPath); err != nil {
		adapter.finishWithoutProcessLocked(state, CodeStatePersistenceFailed, []byte(err.Error()))
		return
	}
	workingDirectory, workspaceBound, err := adapter.executionWorkingDirectory(callerContext, request, state.runPath)
	if err != nil {
		adapter.finishWithoutProcessLocked(state, ErrorCode(err), []byte(err.Error()))
		return
	}
	runContext, cancel := context.WithCancelCause(adapter.lifecycle)
	timeout := time.AfterFunc(adapter.config.Timeout, func() {
		cancel(errExecutionTimeout)
	})
	finished := make(chan struct{})
	if callerContext.Done() != nil {
		go func() {
			select {
			case <-callerContext.Done():
				cancel(errExecutionCanceled)
			case <-finished:
			}
		}()
	}

	diagnostic := &cappedDiagnostic{maximum: adapter.config.MaxDiagnosticBytes}
	var (
		command             *exec.Cmd
		gateReader          *os.File
		gateWriter          *os.File
		supervisorInstance  string
		supervisorPublicKey string
		supervised          bool
		diagnosticDone      <-chan struct{}
		diagnosticReader    *os.File
		commandErr          error
		supervisor          supervisorCommand
	)
	if adapter.processControlsEnabled() {
		instance, instanceErr := newSupervisorInstance()
		if instanceErr != nil {
			commandErr = instanceErr
		} else {
			supervisorInstance = instance
			publicKey, privateKey, keyErr := newCompletionSigningKey()
			if keyErr != nil {
				commandErr = keyErr
			}
			supervisorPublicKey = publicKey
			runDirectory := filepath.Join(adapter.rootPath, filepath.FromSlash(state.runPath))
			envelope := supervisorEnvelope{
				SchemaVersion: supervisorEnvelopeSchema, SupervisorInstance: instance,
				ExecutionRef: request.ExecutionRef.String(), RequestHash: state.requestHash,
				SpecHash: request.SpecHash, ArtifactMediaType: request.ArtifactMediaType,
				RuntimeScope:        adapter.config.RuntimeScope,
				CompletionPublicKey: publicKey, CompletionPrivateKey: privateKey,
				Command: adapter.command,
				Arguments: adapter.commandArgumentsWithSession(
					state.runPath, workspaceBound, workspaceBound && len(request.WriteSet) != 0, session,
				),
				Environment: append([]string(nil), environment...), WorkingDirectory: workingDirectory,
				Prompt: prompt, RunDirectory: runDirectory,
				ResultPath:          filepath.Join(runDirectory, lastMessageFileName),
				TimeoutNanos:        int64(adapter.config.Timeout),
				PipeDrainDelayNanos: int64(adapter.config.ProcessPipeDrainDelay),
				MaxDiagnosticBytes:  adapter.config.MaxDiagnosticBytes,
				MaxOutputBytes:      request.MaxOutputBytes,
			}
			if commandErr == nil {
				supervisor, commandErr = platformSupervisorCommand(runContext, envelope)
			}
			clearSupervisorEnvelope(&envelope)
			supervised = commandErr == nil
			if supervised {
				command, gateReader, gateWriter = supervisor.command, supervisor.gateReader, supervisor.gateWriter
				timeout.Stop()
			}
		}
	} else {
		command, gateReader, gateWriter, commandErr = adapter.executionWorkspaceCommand(
			runContext, state.runPath, workspaceBound, workspaceBound && len(request.WriteSet) != 0, session,
		)
	}
	if commandErr != nil {
		timeout.Stop()
		cancel(errExecutionFinished)
		close(finished)
		adapter.finishWithoutProcessLocked(state, CodeProcessStartFailed, []byte(commandErr.Error()))
		return
	}
	if supervised {
		adapter.configureSupervisorCommand(command, runContext)
	} else {
		adapter.configureExecutionCommand(command, runContext, workingDirectory, environment, prompt, diagnostic)
	}

	var ownerLock *os.File
	if adapter.processControlsEnabled() {
		ownerLock, commandErr = adapter.acquireOwnerLock(state.runPath)
		if commandErr != nil {
			_ = gateReader.Close()
			_ = gateWriter.Close()
			if supervised {
				_ = supervisor.sealed.Close()
				_ = supervisor.diagnosticReader.Close()
				_ = supervisor.diagnosticWriter.Close()
			}
			timeout.Stop()
			cancel(errExecutionFinished)
			close(finished)
			adapter.finishWithoutProcessLocked(state, CodeProcessStartFailed, []byte(commandErr.Error()))
			return
		}
	}
	startErr := command.Start()
	clearEnvironment(command.Env)
	command.Env = nil
	if gateReader != nil {
		_ = gateReader.Close()
	}
	if supervised {
		_ = supervisor.sealed.Close()
		_ = supervisor.diagnosticWriter.Close()
		done := make(chan struct{})
		diagnosticDone = done
		diagnosticReader = supervisor.diagnosticReader
		go func() {
			_, _ = io.Copy(diagnostic, supervisor.diagnosticReader)
			_ = supervisor.diagnosticReader.Close()
			close(done)
		}()
	}
	if startErr != nil {
		if gateWriter != nil {
			_ = gateWriter.Close()
		}
		if supervised {
			_ = supervisor.diagnosticReader.Close()
		}
		releaseOwnerLock(ownerLock)
		timeout.Stop()
		cancel(errExecutionFinished)
		close(finished)
		_, _ = diagnostic.Write([]byte(startErr.Error()))
		payload, truncated := diagnostic.snapshot()
		adapter.finishWithoutProcessLockedWithDiagnostic(state, CodeProcessStartFailed, payload, truncated)
		return
	}
	if !adapter.releaseGatedStartLocked(
		command, gateWriter, ownerLock, timeout, cancel, finished, request, state,
		supervisorInstance, supervisorPublicKey,
	) {
		return
	}
	state.status = ports.AgentRunning
	state.cancel = cancel
	state.settled = make(chan struct{})
	adapter.waitGroup.Add(1)
	go adapter.waitForExecution(
		command, runContext, cancel, timeout, finished, request, state, diagnostic, diagnosticDone,
		diagnosticReader,
	)
}

func (adapter *Adapter) configureSupervisorCommand(command *exec.Cmd, runContext context.Context) {
	command.WaitDelay = adapter.config.ProcessPipeDrainDelay
	configureProcessGroup(command, func() error { return context.Cause(runContext) })
	command.Dir = "/"
	command.Env = []string{}
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil
}

func (adapter *Adapter) configureExecutionCommand(
	command *exec.Cmd, runContext context.Context, workingDirectory string,
	environment []string, prompt string, diagnostic *cappedDiagnostic,
) {
	command.WaitDelay = adapter.config.ProcessPipeDrainDelay
	configureProcessGroup(command, func() error { return context.Cause(runContext) })
	command.Dir = workingDirectory
	command.Env = append([]string(nil), environment...)
	if command.Env == nil {
		command.Env = []string{}
	}
	command.Stdin = strings.NewReader(prompt)
	command.Stdout = io.Discard
	command.Stderr = diagnostic
}

func (adapter *Adapter) releaseGatedStartLocked(
	command *exec.Cmd, gateWriter, ownerLock *os.File, timeout *time.Timer,
	cancel context.CancelCauseFunc, finished chan struct{},
	request ports.AgentLaunchRequest, state *executionState, supervisorInstance, supervisorPublicKey string,
) bool {
	if !adapter.processControlsEnabled() {
		return true
	}
	pgid, bootID, birthMarker, err := platformCaptureProcess(command.Process.Pid)
	record := processRecord{
		SchemaVersion: processSchemaVersion, ExecutionRef: request.ExecutionRef.String(),
		RequestHash: state.requestHash, RuntimeScope: adapter.config.RuntimeScope,
		PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birthMarker,
	}
	supervised := supervisorInstance != ""
	if supervised {
		record.SchemaVersion = supervisedProcessSchemaVersion
		record.SupervisorInstance = supervisorInstance
		record.CompletionPublicKey = supervisorPublicKey
	}
	if err != nil {
		adapter.abortGatedStartLocked(command, gateWriter, ownerLock, timeout, cancel, finished, state, err)
		return false
	}
	if err := adapter.persistProcessRecord(state.runPath, record); err != nil {
		adapter.abortGatedStartLocked(command, gateWriter, ownerLock, timeout, cancel, finished, state, err)
		return false
	}
	state.process, state.ownerLock = &record, ownerLock
	var releaseErr error
	if supervised {
		releaseErr = json.NewEncoder(gateWriter).Encode(record)
	} else {
		_, releaseErr = gateWriter.Write([]byte("run\n"))
	}
	if releaseErr != nil {
		adapter.abortGatedStartLocked(command, gateWriter, ownerLock, timeout, cancel, finished, state, releaseErr)
		return false
	}
	if err := gateWriter.Close(); err != nil {
		adapter.abortGatedStartLocked(command, nil, ownerLock, timeout, cancel, finished, state, err)
		return false
	}
	return true
}

func (adapter *Adapter) executionWorkingDirectory(ctx context.Context, request ports.AgentLaunchRequest, runPath string) (string, bool, error) {
	if request.ExecutionWorkspaceRef.String() == "" {
		return filepath.Join(adapter.rootPath, filepath.FromSlash(runPath)), false, nil
	}
	if adapter.workspaceResolver == nil {
		return "", false, &Error{Code: CodeWorkspaceResolverInvalid}
	}
	resolved, err := adapter.workspaceResolver.ResolveExecutionWorkspace(ctx, request.ExecutionWorkspaceRef)
	if err != nil {
		return "", false, &Error{Code: CodeWorkspaceUnavailable, Cause: err}
	}
	if !filepath.IsAbs(resolved) {
		return "", false, &Error{Code: CodeWorkspaceUnsafe}
	}
	info, err := os.Lstat(resolved)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", false, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	return resolved, true, nil
}

// executionCommand remains the legacy private-control command constructor for
// existing non-workspace control tests. Production launch uses the explicit
// workspace variant below.
func (adapter *Adapter) executionCommand(runContext context.Context, runPath string) (*exec.Cmd, *os.File, *os.File, error) {
	return adapter.executionWorkspaceCommand(runContext, runPath, false, false, nil)
}

func (adapter *Adapter) executionWorkspaceCommand(runContext context.Context, runPath string,
	workspaceBound, workspaceWritable bool, session *resolvedSession,
) (*exec.Cmd, *os.File, *os.File, error) {
	arguments := adapter.commandArgumentsWithSession(runPath, workspaceBound, workspaceWritable, session)
	if !adapter.processControlsEnabled() {
		return exec.CommandContext(runContext, adapter.command, arguments...), nil, nil, nil
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}
	script := `unset PWD; if IFS= read -r gate <&3; then [ "$gate" = run ] && exec 3<&- && exec "$@"; fi; exit 0`
	commandArguments := []string{"-c", script, "orquesta-codex-gate", adapter.command}
	if configuredPWD, found := adapter.config.Environment["PWD"]; found {
		script = `saved_pwd=$1; shift; PWD=$saved_pwd; export PWD; if IFS= read -r gate <&3; then [ "$gate" = run ] && exec 3<&- && exec "$@"; fi; exit 0`
		commandArguments = []string{"-c", script, "orquesta-codex-gate", configuredPWD, adapter.command}
	}
	commandArguments = append(commandArguments, arguments...)
	command := exec.CommandContext(runContext, "/bin/sh", commandArguments...)
	command.ExtraFiles = []*os.File{reader}
	return command, reader, writer, nil
}

func (adapter *Adapter) abortGatedStartLocked(
	command *exec.Cmd,
	gateWriter *os.File,
	ownerLock *os.File,
	timeout *time.Timer,
	cancel context.CancelCauseFunc,
	finished chan struct{},
	state *executionState,
	cause error,
) {
	if gateWriter != nil {
		_ = gateWriter.Close()
	}
	cancel(errExecutionCanceled)
	_ = command.Wait()
	_ = cleanupProcessGroup(command)
	timeout.Stop()
	close(finished)
	releaseOwnerLock(ownerLock)
	state.process, state.ownerLock = nil, nil
	adapter.finishWithoutProcessLocked(state, CodeStatePersistenceFailed, []byte(cause.Error()))
}

func (adapter *Adapter) waitForExecution(
	command *exec.Cmd,
	runContext context.Context,
	cancel context.CancelCauseFunc,
	timeout *time.Timer,
	finished chan struct{},
	request ports.AgentLaunchRequest,
	state *executionState,
	diagnostic *cappedDiagnostic,
	diagnosticDone <-chan struct{},
	diagnosticReader *os.File,
) {
	defer adapter.waitGroup.Done()
	waitErr := command.Wait()
	awaitDiagnosticDrain(diagnosticDone, diagnosticReader, adapter.config.ProcessPipeDrainDelay)
	if errors.Is(waitErr, exec.ErrWaitDelay) && command.ProcessState != nil && command.ProcessState.Success() {
		waitErr = nil
	}
	timeout.Stop()
	cause := context.Cause(runContext)
	cancel(errExecutionFinished)
	close(finished)
	// Stop persists its post-signal proof while holding adapter.mu. Snapshot
	// under the same mutex so a fast exit cannot publish a terminal first.
	adapter.mu.Lock()
	proof := state.stopProof.Load()
	adapter.mu.Unlock()
	var cleanupErr error
	if proof != nil && proof.Mode == ports.AgentStopCooperative && cause == nil {
		groupGone, inspectErr := inspectProcessTree(*state.process)
		if inspectErr != nil {
			cleanupErr = inspectErr
		} else {
			adapter.mu.Lock()
			currentProof := state.stopProof.Load()
			if equalStopSignalProof(proof, currentProof) {
				if !groupGone {
					// The cooperative effect ended the leader but not the tree.
					// Keep ownership and leave escalation to an explicit forced
					// request; neither WaitDelay nor cleanup may smuggle SIGKILL.
					state.cancel = nil
					state.status = ports.AgentRunning
					adapter.settleExecutionLocked(state)
					adapter.mu.Unlock()
					return
				}
				diagnosticPayload, diagnosticTruncated := diagnostic.snapshot()
				adapter.completeExecutionLocked(request, state, proof, waitErr, nil, cause, diagnosticPayload, diagnosticTruncated)
				adapter.mu.Unlock()
				return
			}
			// A forced proof superseded the cooperative snapshot while its
			// leader was settling. Re-evaluate as forced and run the normal
			// cleanup path so cleanup failures still outrank stopped.
			proof = currentProof
			adapter.mu.Unlock()
		}
	}
	if cleanupErr == nil {
		supervised := state.process != nil && state.process.SupervisorInstance != ""
		if !supervised {
			cleanupErr = errors.New(CodeProcessCleanupFailed)
			if adapter.processCleanup != nil {
				cleanupErr = adapter.processCleanup(command)
			}
		}
	}
	if cleanupErr != nil {
		_, _ = diagnostic.Write([]byte("process group cleanup: " + cleanupErr.Error()))
	}
	diagnosticPayload, diagnosticTruncated := diagnostic.snapshot()
	adapter.mu.Lock()
	proof = state.stopProof.Load()
	adapter.completeExecutionLocked(request, state, proof, waitErr, cleanupErr, cause, diagnosticPayload, diagnosticTruncated)
	adapter.mu.Unlock()
}

func awaitDiagnosticDrain(done <-chan struct{}, reader *os.File, delay time.Duration) {
	if done == nil {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-done:
		return
	case <-timer.C:
		if reader != nil {
			_ = reader.Close()
		}
	}
	timer.Reset(delay)
	select {
	case <-done:
	case <-timer.C:
	}
}

func equalStopSignalProof(left, right *stopSignalProof) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func (adapter *Adapter) completeExecutionLocked(
	request ports.AgentLaunchRequest,
	state *executionState,
	proof *stopSignalProof,
	waitErr error,
	cleanupErr error,
	cause error,
	diagnosticPayload []byte,
	diagnosticTruncated bool,
) {
	var completion *completionProof
	var completionResult []byte
	unresolvedStop := false
	if proof == nil {
		var stopErr error
		unresolvedStop, stopErr = adapter.hasDurableStopRequest(state.runPath)
		unresolvedStop = unresolvedStop || stopErr != nil
	}
	if !unresolvedStop && state.process != nil && state.process.SupervisorInstance != "" {
		if candidate, result, found := adapter.loadCompletionProof(state); found {
			completion = &candidate
			completionResult = result
			if int64(len(diagnosticPayload)) != candidate.DiagnosticSize ||
				digestBytes(diagnosticPayload) != candidate.DiagnosticHash {
				clearBytes(diagnosticPayload)
				diagnosticPayload = nil
				diagnosticTruncated = false
			} else {
				diagnosticTruncated = candidate.DiagnosticTruncated
			}
		}
	}
	defer clearBytes(diagnosticPayload)
	defer clearBytes(completionResult)
	terminal := adapter.buildTerminal(
		request, state, proof, completion, completionResult, waitErr, cleanupErr, cause,
		diagnosticPayload, diagnosticTruncated,
	)
	if terminal.ErrorCode == CodeExecutionStopped {
		if proof != nil {
			if err := adapter.finishStoppedProcessLocked(state, *proof); err == nil {
				state.cancel = nil
				adapter.settleExecutionLocked(state)
				return
			}
		}
		terminal.ErrorCode = CodeStatePersistenceFailed
	}
	var gateErr error
	terminal, gateErr = adapter.gateCredentialTerminalLocked(state, terminal)
	if gateErr != nil {
		state.terminal = &terminal
		state.status = terminal.Status
		state.terminalDurable = false
		state.cancel = nil
		adapter.releaseProcessOwnershipLocked(state)
		adapter.settleExecutionLocked(state)
		return
	}
	if state.process != nil && state.process.SupervisorInstance != "" {
		clearBytes(terminal.Diagnostic)
		terminal.Diagnostic = nil
		terminal.DiagnosticTruncated = false
	}
	persisted, err := adapter.persistTerminal(state.runPath, terminal, state.receipt.SpecHash, state.maxOutput)
	if err != nil {
		terminal.Status = ports.AgentFailed
		terminal.MediaType = ""
		terminal.Artifact = ""
		terminal.ErrorCode = CodeStatePersistenceFailed
		state.terminal = &terminal
		state.status = ports.AgentFailed
		state.terminalDurable = false
		state.cancel = nil
		adapter.releaseProcessOwnershipLocked(state)
		adapter.settleExecutionLocked(state)
		return
	}
	state.terminal = &persisted
	state.status = persisted.Status
	state.terminalDurable = true
	state.cancel = nil
	adapter.releaseProcessOwnershipLocked(state)
	adapter.settleExecutionLocked(state)
	_ = adapter.removeCompletionArtifacts(state.runPath)
}

func (adapter *Adapter) settleExecutionLocked(state *executionState) {
	if state.settled != nil {
		close(state.settled)
		state.settled = nil
	}
}

func (adapter *Adapter) buildTerminal(
	request ports.AgentLaunchRequest,
	state *executionState,
	proof *stopSignalProof,
	completion *completionProof,
	completionResult []byte,
	waitErr error,
	cleanupErr error,
	cause error,
	diagnostic []byte,
	diagnosticTruncated bool,
) terminalRecord {
	terminal := terminalRecord{
		SchemaVersion:       stateSchemaVersion,
		RequestHash:         state.terminalRequestHash,
		Status:              ports.AgentFailed,
		ObservedAt:          adapter.terminalTime(state.receipt.AcceptedAt),
		Diagnostic:          diagnostic,
		DiagnosticTruncated: diagnosticTruncated,
	}
	switch {
	case errors.Is(cause, errExecutionTimeout):
		terminal.ErrorCode = CodeExecutionTimeout
		return terminal
	case errors.Is(cause, errExecutionCanceled), errors.Is(cause, errAdapterShutdown):
		terminal.ErrorCode = CodeExecutionCanceled
		return terminal
	case cleanupErr != nil:
		terminal.ErrorCode = CodeProcessCleanupFailed
		return terminal
	case proof != nil:
		terminal.ErrorCode = CodeExecutionStopped
		return terminal
	case completion != nil && completion.Cause == supervisorCauseTimeout:
		terminal.ErrorCode = CodeExecutionTimeout
		return terminal
	case state.process != nil && state.process.SupervisorInstance != "" && completion == nil:
		terminal.ErrorCode = CodeExecutionInterrupted
		return terminal
	case completion != nil && (!completion.Exited || completion.ExitCode != 0):
		terminal.ErrorCode = CodeProcessFailed
		return terminal
	case completion == nil && waitErr != nil:
		terminal.ErrorCode = CodeProcessFailed
		return terminal
	}

	var result modelResult
	var code string
	if completion != nil {
		switch {
		case completion.ResultTooLarge:
			code = CodeOutputTooLarge
		case !completion.ResultFound:
			code = CodeOutputMissing
		default:
			result, code = decodeModelResult(completionResult, request.MaxOutputBytes)
		}
	} else {
		result, code = adapter.readModelResult(state.runPath, request.MaxOutputBytes)
	}
	if code != "" {
		terminal.ErrorCode = code
		return terminal
	}
	terminal.Status = ports.AgentCompleted
	terminal.MediaType = request.ArtifactMediaType
	terminal.Artifact = result.Artifact
	return terminal
}

func (adapter *Adapter) finishSupervisedProcessLocked(state *executionState) error {
	if proof, result, found := adapter.loadCompletionProof(state); found {
		state.artifactMediaType = proof.ArtifactMediaType
		clearBytes(result)
	}
	request := ports.AgentLaunchRequest{
		ArtifactMediaType: state.artifactMediaType,
		MaxOutputBytes:    state.maxOutput,
	}
	adapter.completeExecutionLocked(request, state, nil, nil, nil, nil, nil, false)
	if state.terminal == nil || !state.terminalDurable {
		return &Error{Code: CodeStatePersistenceFailed}
	}
	return nil
}

func (adapter *Adapter) terminalTime(fallback time.Time) time.Time {
	observedAt := adapter.config.Now()
	if observedAt.IsZero() {
		return fallback.UTC()
	}
	return observedAt.UTC()
}

func (adapter *Adapter) finishWithoutProcessLocked(state *executionState, code string, diagnostic []byte) {
	writer := &cappedDiagnostic{maximum: adapter.config.MaxDiagnosticBytes}
	_, _ = writer.Write(diagnostic)
	payload, truncated := writer.snapshot()
	adapter.finishWithoutProcessLockedWithDiagnostic(state, code, payload, truncated)
}

func (adapter *Adapter) finishWithoutProcessLockedWithDiagnostic(state *executionState, code string, diagnostic []byte, truncated bool) {
	terminal := terminalRecord{
		SchemaVersion:       stateSchemaVersion,
		RequestHash:         state.terminalRequestHash,
		Status:              ports.AgentFailed,
		ErrorCode:           code,
		ObservedAt:          adapter.terminalTime(state.receipt.AcceptedAt),
		Diagnostic:          append([]byte(nil), diagnostic...),
		DiagnosticTruncated: truncated,
	}
	var gateErr error
	terminal, gateErr = adapter.gateCredentialTerminalLocked(state, terminal)
	if gateErr != nil {
		state.status = terminal.Status
		state.terminal = &terminal
		state.terminalDurable = false
		return
	}
	if persisted, err := adapter.persistTerminal(state.runPath, terminal, state.receipt.SpecHash, state.maxOutput); err == nil {
		terminal = persisted
		state.terminalDurable = true
	} else {
		terminal.ErrorCode = CodeStatePersistenceFailed
		state.terminalDurable = false
	}
	state.status = terminal.Status
	state.terminal = &terminal
}

func (adapter *Adapter) prepareRuntimeFiles(runPath string) error {
	if err := adapter.writePrivateRuntimeFile(path.Join(runPath, outputSchemaFileName), outputSchema); err != nil {
		return err
	}
	return adapter.writePrivateRuntimeFile(path.Join(runPath, lastMessageFileName), nil)
}

func (adapter *Adapter) writePrivateRuntimeFile(filePath string, content []byte) error {
	if info, err := adapter.root.Lstat(filePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return &Error{Code: CodeStatePersistenceFailed}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	file, err := adapter.root.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = adapter.root.Remove(filePath)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	if len(content) > 0 {
		if _, err := file.Write(content); err != nil {
			return &Error{Code: CodeStatePersistenceFailed, Cause: err}
		}
	}
	if err := file.Sync(); err != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	if err := file.Close(); err != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	remove = false
	return nil
}

func (adapter *Adapter) commandArguments(runPath string, workspace ...bool) []string {
	workspaceBound := len(workspace) == 1 && workspace[0]
	workspaceWritable := workspaceBound
	if len(workspace) >= 2 {
		workspaceBound, workspaceWritable = workspace[0], workspace[0] && workspace[1]
	}
	return adapter.commandArgumentsWithSession(runPath, workspaceBound, workspaceWritable, nil)
}

func (adapter *Adapter) commandArgumentsWithSession(runPath string, workspaceBound, workspaceWritable bool, session *resolvedSession) []string {
	schemaPath := filepath.Join(adapter.rootPath, filepath.FromSlash(path.Join(runPath, outputSchemaFileName)))
	lastMessagePath := filepath.Join(adapter.rootPath, filepath.FromSlash(path.Join(runPath, lastMessageFileName)))
	arguments := []string{
		"exec",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--strict-config",
		"--skip-git-repo-check",
		"--sandbox", codexSandbox(workspaceWritable),
		"--output-schema", schemaPath,
		"--output-last-message", lastMessagePath,
		"--config", fmt.Sprintf("model_reasoning_effort=%q", adapter.config.ReasoningEffort),
		"--config", `shell_environment_policy.inherit="all"`,
		"--config", shellEnvironmentIncludeOnly(adapter.config.Environment),
		"--config", shellEnvironmentExclude(adapter.config.MCPBearerTokenEnvVar),
		"--config", `shell_environment_policy.ignore_default_excludes=false`,
		"--config", `shell_environment_policy.experimental_use_profile=false`,
	}
	if adapter.config.Model != "" {
		arguments = append(arguments, "--model", adapter.config.Model)
	}
	return append(arguments, sessionArguments(session, adapter.config.MCPBearerTokenEnvVar)...)
}

func codexSandbox(workspaceBound bool) string {
	if workspaceBound {
		return "workspace-write"
	}
	return "read-only"
}

func shellEnvironmentIncludeOnly(environment map[string]string) string {
	names := make([]string, 0, len(environment))
	for name := range environment {
		names = append(names, name)
	}
	sort.Strings(names)
	payload, _ := json.Marshal(names)
	return "shell_environment_policy.include_only=" + string(payload)
}

func shellEnvironmentExclude(bearerTokenEnvVar string) string {
	payload, _ := json.Marshal([]string{codexAPIKeyEnvironment, openAIAPIKeyEnvironment, bearerTokenEnvVar})
	return "shell_environment_policy.exclude=" + string(payload)
}

func clearEnvironment(environment []string) {
	for index := range environment {
		environment[index] = ""
	}
}

func (adapter *Adapter) readModelResult(runPath string, maxOutputBytes int64) (modelResult, string) {
	filePath := path.Join(runPath, lastMessageFileName)
	info, err := adapter.root.Lstat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return modelResult{}, CodeOutputMissing
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return modelResult{}, CodeOutputInvalid
	}
	if info.Size() > maxOutputBytes {
		return modelResult{}, CodeOutputTooLarge
	}
	payload, err := adapter.root.ReadFile(filePath)
	if err != nil {
		return modelResult{}, CodeOutputInvalid
	}
	if int64(len(payload)) > maxOutputBytes {
		return modelResult{}, CodeOutputTooLarge
	}
	return decodeModelResult(payload, maxOutputBytes)
}

func decodeModelResult(payload []byte, maxOutputBytes int64) (modelResult, string) {
	if int64(len(payload)) > maxOutputBytes {
		return modelResult{}, CodeOutputTooLarge
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var result modelResult
	if err := decoder.Decode(&result); err != nil {
		return modelResult{}, CodeOutputInvalid
	}
	if err := requireJSONEOF(decoder); err != nil {
		return modelResult{}, CodeOutputInvalid
	}
	if len(result.Artifact) == 0 {
		return modelResult{}, CodeOutputInvalid
	}
	if int64(len(result.Artifact)) > maxOutputBytes {
		return modelResult{}, CodeOutputTooLarge
	}
	return result, ""
}
