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

func (adapter *Adapter) startExecutionLocked(callerContext context.Context, request ports.AgentLaunchRequest, state *executionState) {
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
	command := exec.CommandContext(runContext, adapter.command, adapter.commandArguments(state.runPath)...)
	command.WaitDelay = adapter.config.ProcessPipeDrainDelay
	configureProcessGroup(command)
	command.Dir = filepath.Join(adapter.rootPath, filepath.FromSlash(state.runPath))
	command.Env = append([]string(nil), adapter.environment...)
	if command.Env == nil {
		command.Env = []string{}
	}
	command.Stdin = strings.NewReader(agentPrompt(request))
	command.Stdout = io.Discard
	command.Stderr = diagnostic

	if err := command.Start(); err != nil {
		timeout.Stop()
		cancel(errExecutionFinished)
		close(finished)
		_, _ = diagnostic.Write([]byte(err.Error()))
		payload, truncated := diagnostic.snapshot()
		adapter.finishWithoutProcessLockedWithDiagnostic(state, CodeProcessStartFailed, payload, truncated)
		return
	}
	state.status = ports.AgentRunning
	state.cancel = cancel
	adapter.waitGroup.Add(1)
	go adapter.waitForExecution(command, runContext, cancel, timeout, finished, request, state, diagnostic)
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
	cleanupErr := errors.New(CodeProcessCleanupFailed)
	if adapter.processCleanup != nil {
		cleanupErr = adapter.processCleanup(command)
	}
	timeout.Stop()
	cause := context.Cause(runContext)
	cancel(errExecutionFinished)
	close(finished)
	if cleanupErr != nil {
		_, _ = diagnostic.Write([]byte("process group cleanup: " + cleanupErr.Error()))
	}
	diagnosticPayload, diagnosticTruncated := diagnostic.snapshot()
	terminal := adapter.buildTerminal(request, state, waitErr, cleanupErr, cause, diagnosticPayload, diagnosticTruncated)

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	persisted, err := adapter.persistTerminal(state.runPath, terminal, state.maxOutput)
	if err != nil {
		terminal.Status = ports.AgentFailed
		terminal.MediaType = ""
		terminal.Artifact = ""
		terminal.ErrorCode = CodeStatePersistenceFailed
		state.terminal = &terminal
		state.status = ports.AgentFailed
		state.terminalDurable = false
		state.cancel = nil
		return
	}
	state.terminal = &persisted
	state.status = persisted.Status
	state.terminalDurable = true
	state.cancel = nil
}

func (adapter *Adapter) buildTerminal(
	request ports.AgentLaunchRequest,
	state *executionState,
	waitErr error,
	cleanupErr error,
	cause error,
	diagnostic []byte,
	diagnosticTruncated bool,
) terminalRecord {
	terminal := terminalRecord{
		SchemaVersion:       stateSchemaVersion,
		RequestHash:         state.requestHash,
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
	case waitErr != nil:
		terminal.ErrorCode = CodeProcessFailed
		return terminal
	case cleanupErr != nil:
		terminal.ErrorCode = CodeProcessCleanupFailed
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
		RequestHash:         state.requestHash,
		Status:              ports.AgentFailed,
		ErrorCode:           code,
		ObservedAt:          adapter.terminalTime(state.receipt.AcceptedAt),
		Diagnostic:          append([]byte(nil), diagnostic...),
		DiagnosticTruncated: truncated,
	}
	if persisted, err := adapter.persistTerminal(state.runPath, terminal, state.maxOutput); err == nil {
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
		"--skip-git-repo-check",
		"--sandbox", "read-only",
		"--output-schema", schemaPath,
		"--output-last-message", lastMessagePath,
		"--config", fmt.Sprintf("model_reasoning_effort=%q", adapter.config.ReasoningEffort),
	}
	if adapter.config.Model != "" {
		arguments = append(arguments, "--model", adapter.config.Model)
	}
	return arguments
}

func agentPrompt(request ports.AgentLaunchRequest) string {
	return "Produce one artifact for the following objective.\n\n" +
		"Objective:\n" + request.Objective + "\n\n" +
		"Phase:\n" + request.PhaseKey + "\n\n" +
		"Role:\n" + request.RoleKey + "\n\n" +
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
