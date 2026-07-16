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
	buffer    bytes.Buffer
	maximum   int64
	truncated bool
}

func (writer *cappedDiagnostic) Write(payload []byte) (int, error) {
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
	return append([]byte(nil), writer.buffer.Bytes()...), writer.truncated
}

func (adapter *Adapter) startExecutionLocked(callerContext context.Context, request ports.AgentLaunchRequest, state *executionState, environment []string) {
	if err := adapter.prepareRuntimeFiles(state.runPath); err != nil {
		adapter.finishWithoutProcessLocked(state, CodeStatePersistenceFailed, []byte(err.Error()))
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
	command, gateReader, gateWriter, commandErr := adapter.executionCommand(runContext, state.runPath)
	if commandErr != nil {
		timeout.Stop()
		cancel(errExecutionFinished)
		close(finished)
		adapter.finishWithoutProcessLocked(state, CodeProcessStartFailed, []byte(commandErr.Error()))
		return
	}
	command.WaitDelay = adapter.config.ProcessPipeDrainDelay
	configureProcessGroup(command, func() error { return context.Cause(runContext) })
	command.Dir = filepath.Join(adapter.rootPath, filepath.FromSlash(state.runPath))
	command.Env = append([]string(nil), environment...)
	if command.Env == nil {
		command.Env = []string{}
	}
	command.Stdin = strings.NewReader(agentPrompt(request))
	command.Stdout = io.Discard
	command.Stderr = diagnostic

	var ownerLock *os.File
	if adapter.processControlsEnabled() {
		ownerLock, commandErr = adapter.acquireOwnerLock(state.runPath)
		if commandErr != nil {
			_ = gateReader.Close()
			_ = gateWriter.Close()
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
	if startErr != nil {
		if gateWriter != nil {
			_ = gateWriter.Close()
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
	if adapter.processControlsEnabled() {
		pgid, bootID, birthMarker, identityErr := platformCaptureProcess(command.Process.Pid)
		record := processRecord{
			SchemaVersion: processSchemaVersion, ExecutionRef: request.ExecutionRef.String(),
			RequestHash: state.requestHash, RuntimeScope: adapter.config.RuntimeScope,
			PID: command.Process.Pid, PGID: pgid, BootID: bootID, BirthMarker: birthMarker,
		}
		if identityErr != nil {
			adapter.abortGatedStartLocked(command, gateWriter, ownerLock, timeout, cancel, finished, state, identityErr)
			return
		}
		if persistErr := adapter.persistProcessRecord(state.runPath, record); persistErr != nil {
			adapter.abortGatedStartLocked(command, gateWriter, ownerLock, timeout, cancel, finished, state, persistErr)
			return
		}
		state.process, state.ownerLock = &record, ownerLock
		if _, releaseErr := gateWriter.Write([]byte("run\n")); releaseErr != nil {
			adapter.abortGatedStartLocked(command, gateWriter, ownerLock, timeout, cancel, finished, state, releaseErr)
			return
		}
		if releaseErr := gateWriter.Close(); releaseErr != nil {
			adapter.abortGatedStartLocked(command, nil, ownerLock, timeout, cancel, finished, state, releaseErr)
			return
		}
	}
	state.status = ports.AgentRunning
	state.cancel = cancel
	state.settled = make(chan struct{})
	adapter.waitGroup.Add(1)
	go adapter.waitForExecution(command, runContext, cancel, timeout, finished, request, state, diagnostic)
}

func (adapter *Adapter) executionCommand(runContext context.Context, runPath string) (*exec.Cmd, *os.File, *os.File, error) {
	arguments := adapter.commandArguments(runPath)
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
) {
	defer adapter.waitGroup.Done()
	waitErr := command.Wait()
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
		cleanupErr = errors.New(CodeProcessCleanupFailed)
		if adapter.processCleanup != nil {
			cleanupErr = adapter.processCleanup(command)
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
	terminal := adapter.buildTerminal(request, state, proof, waitErr, cleanupErr, cause, diagnosticPayload, diagnosticTruncated)
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
	case waitErr != nil:
		terminal.ErrorCode = CodeProcessFailed
		return terminal
	}

	result, code := adapter.readModelResult(state.runPath, request.MaxOutputBytes)
	if code != "" {
		terminal.ErrorCode = code
		return terminal
	}
	terminal.Status = ports.AgentCompleted
	terminal.MediaType = request.ArtifactMediaType
	terminal.Artifact = result.Artifact
	return terminal
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

func (adapter *Adapter) commandArguments(runPath string) []string {
	schemaPath := filepath.Join(adapter.rootPath, filepath.FromSlash(path.Join(runPath, outputSchemaFileName)))
	lastMessagePath := filepath.Join(adapter.rootPath, filepath.FromSlash(path.Join(runPath, lastMessageFileName)))
	arguments := []string{
		"exec",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--strict-config",
		"--skip-git-repo-check",
		"--sandbox", "read-only",
		"--output-schema", schemaPath,
		"--output-last-message", lastMessagePath,
		"--config", fmt.Sprintf("model_reasoning_effort=%q", adapter.config.ReasoningEffort),
		"--config", `shell_environment_policy.inherit="all"`,
		"--config", shellEnvironmentIncludeOnly(adapter.config.Environment),
		"--config", `shell_environment_policy.exclude=["CODEX_API_KEY","OPENAI_API_KEY"]`,
		"--config", `shell_environment_policy.ignore_default_excludes=false`,
		"--config", `shell_environment_policy.experimental_use_profile=false`,
	}
	if adapter.config.Model != "" {
		arguments = append(arguments, "--model", adapter.config.Model)
	}
	return arguments
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

func clearEnvironment(environment []string) {
	for index := range environment {
		environment[index] = ""
	}
}

func agentPrompt(request ports.AgentLaunchRequest) string {
	return "Produce one artifact for the following objective.\n\n" +
		"Objective:\n" + request.Objective + "\n\n" +
		"Phase instance:\n" + request.PhaseRef + "\n\n" +
		"Phase key:\n" + request.PhaseKey + "\n\n" +
		"Phase template:\n" + request.PhaseTemplateRef + "\n\n" +
		"Phase inputs:\n" + strings.Join(request.PhaseInputRefs, "\n") + "\n\n" +
		"Phase criteria:\n" + strings.Join(request.PhaseCriterionRefs, "\n") + "\n\n" +
		"Role:\n" + request.RoleKey + "\n\n" +
		"Skills:\n" + strings.Join(request.SkillRefs, "\n") + "\n\n" +
		"Tools:\n" + strings.Join(request.ToolRefs, "\n") + "\n\n" +
		"Capabilities:\n" + strings.Join(request.CapabilityRefs, "\n") + "\n\n" +
		"Allowed write scopes:\n" + strings.Join(request.WriteSet, "\n") + "\n\n" +
		"Output contract:\n" + request.OutputContract + "\n\n" +
		"Artifact media type:\n" + request.ArtifactMediaType + "\n\n" +
		"Return only the JSON object required by the supplied schema. " +
		"Set artifact to the complete artifact content.\n"
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
