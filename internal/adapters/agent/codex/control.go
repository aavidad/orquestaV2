package codex

import (
	"context"
	"time"

	"orquesta/internal/ports"
)

func (adapter *Adapter) ControlCapabilities(ctx context.Context) (ports.AgentControlCapabilities, error) {
	if adapter == nil {
		return ports.AgentControlCapabilities{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentControlCapabilities{}, err
	}
	adapter.mu.Lock()
	closed := adapter.closed
	enabled := adapter.processControlsEnabled()
	adapter.mu.Unlock()
	if closed {
		return ports.AgentControlCapabilities{}, &Error{Code: CodeUnavailable}
	}
	return ports.AgentControlCapabilities{CooperativeStop: enabled, ForcedStop: enabled}, nil
}

func (adapter *Adapter) Stop(ctx context.Context, request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	if adapter == nil {
		return ports.AgentStopReceipt{}, &Error{Code: CodeUnavailable}
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if err := ports.ValidateAgentStopRequest(request); err != nil {
		return ports.AgentStopReceipt{}, err
	}
	requestHash, err := hashStopRequest(request)
	if err != nil {
		return ports.AgentStopReceipt{}, &Error{Code: CodeStateInvalid, Cause: err}
	}

	adapter.mu.Lock()
	preparation, err := adapter.prepareStopLocked(request, requestHash)
	adapter.mu.Unlock()
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if preparation.complete {
		return preparation.receipt, nil
	}
	// Cooperative stop must never wait on provider settlement. One exact
	// inspection happened under prepare; a live tree returns pending and leaves
	// later observation or forced escalation reachable to the caller.
	if request.Mode == ports.AgentStopCooperative && !preparation.observedGone {
		return validatedStopReceipt(request, pendingStopReceipt(request))
	}
	observedGone, waitErr := waitForExactProcess(ctx, preparation.process, preparation.settled)
	if waitErr != nil {
		if ctx.Err() != nil {
			return validatedStopReceipt(request, pendingStopReceipt(request))
		}
		return ports.AgentStopReceipt{}, waitErr
	}
	if !observedGone {
		return validatedStopReceipt(request, pendingStopReceipt(request))
	}

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if preparation.state.terminal == nil {
		if proof := preparation.state.stopProof.Load(); proof != nil {
			if err := adapter.finishStoppedProcessLocked(preparation.state, *proof); err != nil {
				return ports.AgentStopReceipt{}, err
			}
		} else {
			if _, err := adapter.hasDurableStopRequest(preparation.state.runPath); err != nil {
				return ports.AgentStopReceipt{}, err
			}
			if err := adapter.recoverInterruptedExecutionLocked(preparation.state); err != nil {
				return ports.AgentStopReceipt{}, err
			}
		}
	}
	return adapter.confirmTerminalStopLocked(preparation.state, request, requestHash, true)
}

func (adapter *Adapter) controlTargetLocked(request ports.AgentStopRequest) (*executionState, string, error) {
	record, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil {
		return nil, runPath, err
	}
	if !found || record.SchemaVersion != stateSchemaVersion {
		return nil, runPath, &Error{Code: CodeExecutionNotFound}
	}
	launch, err := record.receipt(request.ExecutionRef)
	if err != nil {
		return nil, runPath, err
	}
	if err := ports.ValidateAgentStopTarget(launch, request); err != nil {
		return nil, runPath, err
	}
	if state, ok := adapter.executions[request.ExecutionRef.String()]; ok {
		if state.requestHash != record.RequestHash {
			return nil, runPath, &Error{Code: CodeStateInvalid}
		}
		return state, runPath, nil
	}
	state := &executionState{
		requestHash: record.RequestHash, terminalRequestHash: record.RequestHash,
		receipt: launch, maxOutput: record.MaxOutputBytes, runPath: runPath, status: ports.AgentPending,
	}
	if terminal, terminalFound, loadErr := adapter.loadCausalTerminal(runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes); loadErr != nil {
		return nil, runPath, loadErr
	} else if terminalFound {
		state.status, state.terminal, state.terminalDurable = terminal.Status, &terminal, true
	}
	adapter.executions[request.ExecutionRef.String()] = state
	return state, runPath, nil
}

func (adapter *Adapter) ownProcessLocked(state *executionState) (processRecord, processIdentityState, error) {
	if state.process != nil && state.ownerLock != nil {
		identity, err := platformInspectProcess(*state.process)
		if err != nil {
			return *state.process, identity, &Error{Code: CodeProcessInspectionFailed, Cause: err}
		}
		if identity == processIdentityMismatch && err == nil {
			err = &Error{Code: CodeProcessIdentityMismatch}
		}
		return *state.process, identity, err
	}
	record, found, err := adapter.processRecordForState(state)
	if err != nil {
		return processRecord{}, processIdentityMismatch, err
	}
	if !found {
		return processRecord{}, processIdentityMismatch, &Error{Code: CodeProcessOwnershipInvalid}
	}
	owner, err := adapter.acquireOwnerLock(state.runPath)
	if err != nil {
		return processRecord{}, processIdentityMismatch, err
	}
	identity, err := platformInspectProcess(record)
	if err != nil {
		releaseOwnerLock(owner)
		return processRecord{}, identity, &Error{Code: CodeProcessInspectionFailed, Cause: err}
	}
	if identity == processIdentityMismatch {
		releaseOwnerLock(owner)
		return processRecord{}, identity, &Error{Code: CodeProcessIdentityMismatch}
	}
	state.process, state.ownerLock = &record, owner
	state.status = ports.AgentRunning
	return record, identity, nil
}

func (adapter *Adapter) confirmTerminalStopLocked(state *executionState, request ports.AgentStopRequest, requestHash string, requested bool) (ports.AgentStopReceipt, error) {
	if !requested {
		if _, err := adapter.ensureStopRequest(state.runPath, requestHash, request); err != nil {
			return ports.AgentStopReceipt{}, err
		}
	}
	status, err := adapter.terminalStopStatusLocked(state, request, requestHash, requested)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	receipt, err := adapter.persistStopReceipt(state.runPath, request, requestHash, status)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	return validatedStopReceipt(request, receipt)
}

func (adapter *Adapter) terminalStopStatusLocked(
	state *executionState,
	request ports.AgentStopRequest,
	requestHash string,
	requested bool,
) (ports.AgentStopStatus, error) {
	if state == nil || state.terminal == nil {
		return "", &Error{Code: CodeStateInvalid}
	}
	if state.terminal.Status == ports.AgentCompleted {
		return ports.AgentStopAlreadyCompleted, nil
	}
	if state.terminal.ErrorCode != CodeExecutionStopped {
		return ports.AgentStopAlreadyFailed, nil
	}
	completion, found, err := adapter.loadWinningStopCompletion(state.runPath)
	if err != nil {
		return "", err
	}
	if found && completion.RequestHash == requestHash && completion.Idempotency == request.IdempotencyKey && completion.Mode == request.Mode {
		return ports.AgentStopped, nil
	}
	return ports.AgentStopAlreadyStopped, nil
}

func (adapter *Adapter) finishStoppedProcessLocked(state *executionState, proof stopSignalProof) error {
	if state.terminal != nil {
		adapter.releaseProcessOwnershipLocked(state)
		return nil
	}
	if _, err := adapter.persistStopCompletion(state.runPath, proof); err != nil {
		return err
	}
	if err := adapter.credentialOutputScrub(state.runPath); err != nil {
		return err
	}
	terminal := terminalRecord{
		SchemaVersion: stateSchemaVersion, RequestHash: state.terminalRequestHash,
		Status: ports.AgentFailed, ErrorCode: CodeExecutionStopped,
		ObservedAt: adapter.terminalTime(state.receipt.AcceptedAt),
	}
	persisted, err := adapter.persistTerminal(state.runPath, terminal, state.receipt.SpecHash, state.maxOutput)
	if err != nil {
		return err
	}
	terminal, state.terminalDurable = persisted, true
	state.status, state.terminal = terminal.Status, &terminal
	adapter.releaseProcessOwnershipLocked(state)
	return nil
}

func waitForExactProcess(ctx context.Context, record processRecord, settled <-chan struct{}) (bool, error) {
	if settled != nil {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-settled:
		}
	}
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		gone, err := inspectProcessTree(record)
		if err == nil && gone {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
		}
	}
}

func validatedStopReceipt(request ports.AgentStopRequest, receipt ports.AgentStopReceipt) (ports.AgentStopReceipt, error) {
	if err := ports.ValidateAgentStopReceipt(request, receipt); err != nil {
		return ports.AgentStopReceipt{}, &Error{Code: CodeStateInvalid, Cause: err}
	}
	return receipt, nil
}
