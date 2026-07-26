package codex

import (
	"context"
	"errors"
	"os"
	"path"
	"time"

	"orquesta/internal/ports"
)

const (
	quarantineIntentSchema   = 1
	quarantineIntentFileName = "quarantine-intent.json"
)

type quarantineIntent struct {
	SchemaVersion       int       `json:"schema_version"`
	ExecutionRef        string    `json:"execution_ref"`
	RequestHash         string    `json:"request_hash"`
	TerminalRequestHash string    `json:"terminal_request_hash"`
	RuntimeScope        string    `json:"runtime_scope"`
	SupervisorInstance  string    `json:"supervisor_instance,omitempty"`
	SupervisorPID       int       `json:"supervisor_pid"`
	SupervisorPGID      int       `json:"supervisor_pgid"`
	SupervisorBootID    string    `json:"supervisor_boot_id"`
	SupervisorBirth     string    `json:"supervisor_birth_marker"`
	CgroupName          string    `json:"cgroup_name,omitempty"`
	CgroupDevice        uint64    `json:"cgroup_device,omitempty"`
	CgroupInode         uint64    `json:"cgroup_inode,omitempty"`
	ErrorCode           string    `json:"error_code"`
	PreparedAt          time.Time `json:"prepared_at"`
}

type quarantineOperation struct {
	intent quarantineIntent
	done   chan struct{}
	err    error
}

func (adapter *Adapter) quarantineExecutionLocked(
	ctx context.Context,
	state *executionState,
	cause error,
) error {
	if state == nil || ErrorCode(cause) == "" {
		return errors.Join(&Error{Code: CodeStateInvalid}, cause)
	}
	if state.terminal != nil {
		if !state.terminalDurable {
			return errors.Join(&Error{Code: CodeStatePersistenceFailed}, cause)
		}
		return cause
	}
	if operation := state.quarantine; operation != nil {
		if operation.intent.ErrorCode != ErrorCode(cause) {
			return errors.Join(&Error{Code: CodeStateInvalid}, cause)
		}
		return adapter.waitQuarantineOperationLocked(ctx, operation)
	}

	_, found, err := adapter.processRecordForState(state)
	if err != nil {
		return errors.Join(err, cause)
	}
	if !found {
		return adapter.finishQuarantineWithoutProcessLocked(state, cause)
	}
	record, _, err := adapter.ownProcessLocked(state)
	if err != nil {
		return errors.Join(err, cause)
	}
	intent, err := adapter.ensureQuarantineIntent(state, record, ErrorCode(cause))
	if err != nil {
		adapter.releaseProcessOwnershipLocked(state)
		return errors.Join(err, cause)
	}
	operation := &quarantineOperation{intent: intent, done: make(chan struct{})}
	state.quarantine = operation
	if err := adapter.signalQuarantineLocked(record); err != nil {
		adapter.finishQuarantineOperationLocked(state, operation, errors.Join(err, cause))
		return operation.err
	}

	// The owner lock remains held as the stable cross-adapter token. Waiting,
	// draining and cgroup removal can block and therefore happen without
	// adapter.mu; final state publication is serialized again below.
	adapter.mu.Unlock()
	drainErr := adapter.drainQuarantine(context.WithoutCancel(ctx), state, record)
	adapter.mu.Lock()
	if drainErr != nil {
		adapter.finishQuarantineOperationLocked(
			state, operation,
			errors.Join(&Error{Code: CodeProcessCleanupFailed, Cause: drainErr}, cause),
		)
		return operation.err
	}
	if state.quarantine != operation || state.ownerLock == nil || state.process == nil ||
		*state.process != record {
		adapter.finishQuarantineOperationLocked(
			state, operation, errors.Join(&Error{Code: CodeStateInvalid}, cause),
		)
		return operation.err
	}
	replayed, found, err := adapter.loadQuarantineIntent(state.runPath, record)
	if err != nil || !found || replayed != intent {
		adapter.finishQuarantineOperationLocked(
			state, operation,
			errors.Join(err, &Error{Code: CodeStateInvalid}, cause),
		)
		return operation.err
	}
	if err := adapter.recoveryScrub(state.runPath); err != nil {
		adapter.finishQuarantineOperationLocked(state, operation, errors.Join(err, cause))
		return operation.err
	}
	destroyExecutionGuards(state)
	terminal := terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: intent.TerminalRequestHash,
		Status: ports.AgentFailed, ErrorCode: intent.ErrorCode,
		ObservedAt: adapter.terminalTime(state.receipt.AcceptedAt),
	}
	persisted, err := adapter.persistTerminal(
		state.runPath, terminal, state.receipt.SpecHash, state.maxOutput,
	)
	if err != nil {
		adapter.finishQuarantineOperationLocked(state, operation, errors.Join(err, cause))
		return operation.err
	}
	state.status, state.terminal, state.terminalDurable =
		persisted.Status, &persisted, true
	_ = adapter.removeCompletionArtifacts(state.runPath)
	adapter.finishQuarantineOperationLocked(state, operation, cause)
	return operation.err
}

func (adapter *Adapter) waitQuarantineOperationLocked(
	ctx context.Context,
	operation *quarantineOperation,
) error {
	adapter.mu.Unlock()
	select {
	case <-ctx.Done():
		adapter.mu.Lock()
		return ctx.Err()
	case <-operation.done:
		adapter.mu.Lock()
		return operation.err
	}
}

func (adapter *Adapter) finishQuarantineOperationLocked(
	state *executionState,
	operation *quarantineOperation,
	result error,
) {
	if state != nil && state.quarantine == operation {
		state.quarantine = nil
		adapter.releaseProcessOwnershipLocked(state)
	}
	operation.err = result
	close(operation.done)
}

func (adapter *Adapter) finishQuarantineWithoutProcessLocked(
	state *executionState,
	cause error,
) error {
	if err := adapter.recoveryScrub(state.runPath); err != nil {
		return errors.Join(err, cause)
	}
	destroyExecutionGuards(state)
	terminal := terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: state.terminalRequestHash,
		Status: ports.AgentFailed, ErrorCode: ErrorCode(cause),
		ObservedAt: adapter.terminalTime(state.receipt.AcceptedAt),
	}
	persisted, err := adapter.persistTerminal(
		state.runPath, terminal, state.receipt.SpecHash, state.maxOutput,
	)
	if err != nil {
		return errors.Join(err, cause)
	}
	state.status, state.terminal, state.terminalDurable =
		persisted.Status, &persisted, true
	return cause
}

func (adapter *Adapter) ensureQuarantineIntent(
	state *executionState,
	record processRecord,
	errorCode string,
) (quarantineIntent, error) {
	if existing, found, err := adapter.loadQuarantineIntent(state.runPath, record); err != nil || found {
		if err == nil && (existing.TerminalRequestHash != state.terminalRequestHash ||
			existing.ErrorCode != errorCode) {
			err = &Error{Code: CodeStateInvalid}
		}
		return existing, err
	}
	now := adapter.config.Now()
	if now.IsZero() {
		return quarantineIntent{}, &Error{Code: CodeClockInvalid}
	}
	intent := quarantineIntent{
		SchemaVersion: quarantineIntentSchema,
		ExecutionRef:  record.ExecutionRef, RequestHash: record.RequestHash,
		TerminalRequestHash: state.terminalRequestHash,
		RuntimeScope:        record.RuntimeScope, SupervisorInstance: record.SupervisorInstance,
		SupervisorPID: record.PID, SupervisorPGID: record.PGID,
		SupervisorBootID: record.BootID, SupervisorBirth: record.BirthMarker,
		CgroupName: record.CgroupName, CgroupDevice: record.CgroupDevice,
		CgroupInode: record.CgroupInode, ErrorCode: errorCode, PreparedAt: now.UTC(),
	}
	if err := validateQuarantineIntent(intent, record); err != nil {
		return quarantineIntent{}, err
	}
	created, err := adapter.publishJSON(state.runPath, quarantineIntentFileName, intent)
	if err != nil {
		return quarantineIntent{}, err
	}
	if !created {
		existing, found, err := adapter.loadQuarantineIntent(state.runPath, record)
		if err != nil || !found || existing != intent {
			return quarantineIntent{}, errors.Join(err, &Error{Code: CodeStateInvalid})
		}
		return existing, nil
	}
	return intent, nil
}

func (adapter *Adapter) loadQuarantineIntent(
	runPath string,
	record processRecord,
) (quarantineIntent, bool, error) {
	var intent quarantineIntent
	found, err := adapter.readPrivateJSON(path.Join(runPath, quarantineIntentFileName), &intent)
	if err != nil || !found {
		return quarantineIntent{}, found, err
	}
	if err := validateQuarantineIntent(intent, record); err != nil {
		return quarantineIntent{}, false, err
	}
	return intent, true, nil
}

func validateQuarantineIntent(intent quarantineIntent, record processRecord) error {
	if intent.SchemaVersion != quarantineIntentSchema ||
		intent.ExecutionRef == "" || intent.ExecutionRef != record.ExecutionRef ||
		intent.RequestHash == "" || intent.RequestHash != record.RequestHash ||
		intent.TerminalRequestHash == "" ||
		intent.RuntimeScope == "" || intent.RuntimeScope != record.RuntimeScope ||
		intent.SupervisorInstance != record.SupervisorInstance ||
		intent.SupervisorPID != record.PID || intent.SupervisorPGID != record.PGID ||
		intent.SupervisorBootID != record.BootID ||
		intent.SupervisorBirth != record.BirthMarker ||
		intent.CgroupName != record.CgroupName ||
		intent.CgroupDevice != record.CgroupDevice ||
		intent.CgroupInode != record.CgroupInode ||
		!validQuarantineErrorCode(intent.ErrorCode) || intent.PreparedAt.IsZero() {
		return &Error{Code: CodeStateInvalid}
	}
	return nil
}

func validQuarantineErrorCode(code string) bool {
	switch code {
	case CodeLegacyExecutionRequiresNewAttempt,
		CodeLegacyControlMetadataUnknown,
		CodeCredentialInvalid,
		CodeCredentialUnavailable,
		CodeSessionInvalid,
		CodeSessionUnavailable,
		CodeSecretLeak:
		return true
	default:
		return false
	}
}

func (adapter *Adapter) signalQuarantineLocked(record processRecord) error {
	if record.SchemaVersion != cgroupProcessSchemaVersion {
		return adapter.signalProcessTree(record, ports.AgentStopForced)
	}
	if adapter.cgroups == nil {
		return &Error{Code: CodeCgroupRootRequired}
	}
	if err := platformSignalCgroupQuarantine(record); err != nil &&
		!errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (adapter *Adapter) drainQuarantine(
	ctx context.Context,
	state *executionState,
	record processRecord,
) error {
	timeout := adapter.config.SupervisorStartTimeout
	if timeout <= 0 {
		timeout = adapter.config.ProcessPipeDrainDelay
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	gone, waitErr := waitForExactProcessWithInspector(
		waitCtx, record, nil, adapter.inspectProcessTree,
	)
	cancel()
	if !gone && errors.Is(waitErr, context.DeadlineExceeded) &&
		record.SchemaVersion == cgroupProcessSchemaVersion {
		killErr := platformKillExactProcess(record)
		if killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return errors.Join(waitErr, killErr)
		}
		if err := adapter.cgroups.kill(record); err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.Join(waitErr, err)
		}
		waitCtx, cancel = context.WithTimeout(ctx, adapter.config.Timeout)
		gone, waitErr = waitForExactProcessWithInspector(
			waitCtx, record, nil, adapter.inspectProcessTree,
		)
		cancel()
	}
	if waitErr != nil || !gone {
		if waitErr == nil {
			waitErr = &Error{Code: CodeProcessCleanupFailed}
		}
		return waitErr
	}
	if record.SchemaVersion == cgroupProcessSchemaVersion {
		// The leaf may already have been removed by a replay that completed
		// cleanup but crashed before terminal publication.
		if _, err := adapter.cgroups.leafForRecord(record); err == nil {
			if err := adapter.cgroups.drain(record, adapter.config.Timeout); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return adapter.cleanupTerminalCgroup(state)
}
