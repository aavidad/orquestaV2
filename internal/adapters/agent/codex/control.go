package codex

import (
	"context"
	"errors"
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
	select {
	case <-adapter.lifecycle.Done():
		return ports.AgentControlCapabilities{}, &Error{Code: CodeUnavailable}
	default:
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
	operationContext, endOperation, err := adapter.beginOperation(ctx)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	defer endOperation()
	ctx = operationContext

	adapter.mu.Lock()
	handledLegacyV3, legacyV3Err := adapter.quarantineLegacyV3StopLocked(ctx, request)
	adapter.mu.Unlock()
	if handledLegacyV3 {
		return ports.AgentStopReceipt{}, legacyV3Err
	}

	var preparation stopPreparation
	for {
		adapter.mu.Lock()
		preparation, err = adapter.prepareStopLocked(request, requestHash)
		adapter.mu.Unlock()
		if err != nil {
			return ports.AgentStopReceipt{}, err
		}
		if preparation.starting == nil {
			break
		}
		select {
		case <-ctx.Done():
			if errors.Is(context.Cause(ctx), errAdapterShutdown) {
				return ports.AgentStopReceipt{}, &Error{Code: CodeUnavailable}
			}
			return validatedStopReceipt(request, pendingStopReceipt(request))
		case <-preparation.starting:
		}
		if ctx.Err() != nil {
			if errors.Is(context.Cause(ctx), errAdapterShutdown) {
				return ports.AgentStopReceipt{}, &Error{Code: CodeUnavailable}
			}
			return validatedStopReceipt(request, pendingStopReceipt(request))
		}
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
	observedGone, waitErr := waitForExactProcessWithInspector(
		ctx, preparation.process, preparation.settled, adapter.inspectProcessTree,
	)
	if waitErr != nil {
		if errors.Is(context.Cause(ctx), errAdapterShutdown) {
			return ports.AgentStopReceipt{}, &Error{Code: CodeUnavailable}
		}
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

func (adapter *Adapter) quarantineLegacyV3StopLocked(
	ctx context.Context,
	request ports.AgentStopRequest,
) (bool, error) {
	record, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil || !found || record.SchemaVersion != legacyStateSchemaVersion {
		return false, err
	}
	if record.ExecutionRef != request.ExecutionRef.String() ||
		record.SpecHash != request.SpecHash ||
		record.ProviderRef != request.ProviderRef ||
		record.ExternalRef != request.ExternalRef ||
		request.ModelRef != adapter.modelRef() ||
		request.AgentRef != AgentRef {
		return true, &Error{Code: CodeStateInvalid}
	}
	state := adapter.executions[request.ExecutionRef.String()]
	if state == nil {
		state, err = adapter.recoveryState(record, runPath, request.ExecutionRef)
		if err != nil {
			return true, err
		}
		if terminal, found, loadErr := adapter.loadCausalTerminal(
			runPath, record.RequestHash, record.SpecHash, record.MaxOutputBytes,
		); loadErr != nil {
			return true, loadErr
		} else if found {
			state.status, state.terminal, state.terminalDurable =
				terminal.Status, &terminal, true
		}
	}
	if state.terminal != nil {
		return true, &Error{Code: CodeLegacyControlMetadataUnknown}
	}
	adapter.executions[request.ExecutionRef.String()] = state
	return true, adapter.quarantineExecutionLocked(
		ctx, state, &Error{Code: CodeLegacyControlMetadataUnknown},
	)
}

func (adapter *Adapter) controlTargetLocked(request ports.AgentStopRequest) (*executionState, string, error) {
	record, runPath, found, err := adapter.loadLaunchRecord(request.ExecutionRef)
	if err != nil {
		return nil, runPath, err
	}
	if !found {
		return nil, runPath, &Error{Code: CodeExecutionNotFound}
	}
	if record.SchemaVersion == stateSchemaVersion && record.LaunchActionFence == 0 {
		return nil, runPath, &Error{Code: CodeLegacyControlMetadataUnknown}
	}
	terminalRequestHash := record.RequestHash
	stateRequestHash := record.RequestHash
	if state, ok := adapter.executions[request.ExecutionRef.String()]; ok {
		if err := ports.ValidateAgentStopTarget(state.receipt, request); err != nil {
			return nil, runPath, err
		}
		return state, runPath, nil
	}
	if record.SchemaVersion != stateSchemaVersion {
		source, untrustedRequestHash, upgraded, loadErr := adapter.loadLegacyLaunchBinding(
			runPath, record,
		)
		if loadErr != nil {
			return nil, runPath, loadErr
		}
		record = source
		stateRequestHash = source.RequestHash
		process, processFound, processErr := adapter.readProcessRecord(runPath)
		if processErr != nil {
			return nil, runPath, processErr
		}
		if upgraded && processFound && process.RequestHash == untrustedRequestHash {
			stateRequestHash = process.RequestHash
		}
	}
	launch, err := adapter.controlLaunchReceipt(record, request)
	if err != nil {
		return nil, runPath, err
	}
	if err := ports.ValidateAgentStopTarget(launch, request); err != nil {
		return nil, runPath, err
	}
	state := &executionState{
		requestHash: stateRequestHash, terminalRequestHash: terminalRequestHash,
		receipt: launch, maxOutput: record.MaxOutputBytes, runPath: runPath, status: ports.AgentPending,
	}
	if terminal, terminalFound, loadErr := adapter.loadCausalTerminal(
		runPath, terminalRequestHash, record.SpecHash, record.MaxOutputBytes,
	); loadErr != nil {
		return nil, runPath, loadErr
	} else if terminalFound {
		state.status, state.terminal, state.terminalDurable = terminal.Status, &terminal, true
	}
	adapter.executions[request.ExecutionRef.String()] = state
	return state, runPath, nil
}

func (adapter *Adapter) controlLaunchReceipt(
	record launchRecord,
	request ports.AgentStopRequest,
) (ports.AgentLaunchReceipt, error) {
	if record.SchemaVersion == legacyStateSchemaVersion || record.LaunchActionFence == 0 {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeLegacyControlMetadataUnknown}
	}
	return record.receipt(request.ExecutionRef)
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
		if !state.terminalDurable {
			return &Error{Code: CodeStatePersistenceFailed}
		}
		if err := adapter.cleanupTerminalCgroup(state); err != nil {
			return err
		}
		adapter.releaseProcessOwnershipLocked(state)
		return nil
	}
	requireProof := adapter.requireDurableStopProof
	if state.process != nil && state.process.SchemaVersion == cgroupProcessSchemaVersion {
		requireProof = adapter.requireWinningStopProof
	}
	if err := requireProof(state.runPath, proof); err != nil {
		return err
	}
	if proof.Mode == ports.AgentStopCooperative && state.process != nil &&
		state.process.SchemaVersion == cgroupProcessSchemaVersion {
		completion, result, found := adapter.loadCompletionProof(state)
		clearBytes(result)
		if !found {
			return adapter.recoverInterruptedExecutionLocked(state)
		}
		if completion.Cause != supervisorCauseStop ||
			completion.StopRequestHash != proof.RequestHash ||
			completion.StopIdempotency != proof.Idempotency ||
			completion.StopMode != string(proof.Mode) ||
			completion.StopSequence != proof.Sequence {
			return adapter.finishSupervisedProcessWithStopProofLocked(state, proof)
		}
	}
	if _, err := adapter.persistStopCompletion(state.runPath, proof); err != nil {
		return err
	}
	if err := adapter.credentialOutputScrub(state.runPath); err != nil {
		return err
	}
	destroyExecutionGuards(state)
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
	if err := adapter.cleanupTerminalCgroup(state); err != nil {
		return err
	}
	adapter.releaseProcessOwnershipLocked(state)
	return nil
}

func (adapter *Adapter) requireWinningStopProof(runPath string, proof stopSignalProof) error {
	intent, found, err := adapter.loadWinningStopSignalIntent(runPath)
	if err != nil {
		return err
	}
	if !found || intent.RequestHash != proof.RequestHash ||
		intent.Idempotency != proof.Idempotency || intent.Mode != proof.Mode ||
		intent.Sequence != proof.Sequence {
		return &Error{Code: CodeStopConflict}
	}
	return adapter.requireDurableStopProof(runPath, proof)
}

func (adapter *Adapter) requireDurableStopProof(runPath string, proof stopSignalProof) error {
	winner, found, err := adapter.loadWinningStopSignalProof(runPath)
	if err != nil {
		return err
	}
	if !found || winner != proof {
		return &Error{Code: CodeStopConflict}
	}
	return nil
}

func waitForExactProcess(ctx context.Context, record processRecord, settled <-chan struct{}) (bool, error) {
	return waitForExactProcessWithInspector(ctx, record, settled, inspectProcessTree)
}

func waitForExactProcessWithInspector(
	ctx context.Context,
	record processRecord,
	settled <-chan struct{},
	inspect func(processRecord) (bool, error),
) (bool, error) {
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
		gone, err := inspect(record)
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
