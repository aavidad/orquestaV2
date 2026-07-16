package codex

import (
	"errors"
	"os"

	"orquesta/internal/ports"
)

type stopPreparation struct {
	state        *executionState
	process      processRecord
	settled      <-chan struct{}
	observedGone bool
	receipt      ports.AgentStopReceipt
	complete     bool
}

func (adapter *Adapter) prepareStopLocked(
	request ports.AgentStopRequest,
	requestHash string,
) (stopPreparation, error) {
	if adapter.closed {
		return stopPreparation{}, &Error{Code: CodeUnavailable}
	}
	state, runPath, err := adapter.controlTargetLocked(request)
	if err != nil {
		return stopPreparation{}, err
	}
	_, requested, err := adapter.loadStopRequest(runPath, requestHash, request)
	if err != nil {
		return stopPreparation{}, err
	}
	receiptRecord, replayed, err := adapter.loadStopReceiptRecord(runPath, request, requestHash)
	if err != nil {
		return stopPreparation{}, err
	}
	if replayed {
		if !requested || state.terminal == nil {
			return stopPreparation{}, &Error{Code: CodeStopConflict}
		}
		expected, err := adapter.terminalStopStatusLocked(state, request, requestHash, true)
		if err != nil {
			return stopPreparation{}, err
		}
		if receiptRecord.Status != expected {
			return stopPreparation{}, &Error{Code: CodeStopConflict}
		}
		receipt, err := validatedStopReceipt(request, newStopReceipt(
			request, receiptRecord.Status, receiptRecord.ReceiptRef, receiptRecord.ConfirmedAt,
		))
		return stopPreparation{receipt: receipt, complete: true}, err
	}
	if !adapter.processControlsEnabled() {
		receipt, err := validatedStopReceipt(request, unsupportedStopReceipt(request))
		return stopPreparation{receipt: receipt, complete: true}, err
	}
	if state.terminal != nil {
		receipt, err := adapter.confirmTerminalStopLocked(state, request, requestHash, requested)
		return stopPreparation{receipt: receipt, complete: true}, err
	}
	if !requested {
		if _, err := adapter.ensureStopRequest(runPath, requestHash, request); err != nil {
			return stopPreparation{}, err
		}
	}
	return adapter.prepareLiveStopLocked(state, runPath, request, requestHash)
}

func (adapter *Adapter) prepareLiveStopLocked(
	state *executionState,
	runPath string,
	request ports.AgentStopRequest,
	requestHash string,
) (stopPreparation, error) {
	record, _, err := adapter.ownProcessLocked(state)
	if errors.Is(err, errOwnerLockBusy) {
		receipt, validateErr := validatedStopReceipt(request, pendingStopReceipt(request))
		return stopPreparation{receipt: receipt, complete: true}, validateErr
	}
	if err != nil {
		return stopPreparation{}, err
	}
	_, exactProofFound, err := adapter.loadStopSignalProof(runPath, requestHash, request)
	if err != nil {
		return stopPreparation{}, err
	}
	winnerIntent, winnerIntentFound, err := adapter.loadWinningStopSignalIntent(runPath)
	if err != nil {
		return stopPreparation{}, err
	}
	winner, winnerFound, err := adapter.loadWinningStopSignalProof(runPath)
	if err != nil {
		return stopPreparation{}, err
	}
	state.stopProof.Store(nil)
	if winnerFound {
		state.stopProof.Store(&winner)
	}
	treeGone, err := inspectProcessTree(record)
	if err != nil {
		return stopPreparation{}, err
	}
	if err := adapter.signalStopIfNeededLocked(
		state, record, runPath, request, requestHash,
		treeGone, exactProofFound, winnerIntent, winnerIntentFound, winner, winnerFound,
	); err != nil {
		return stopPreparation{}, err
	}
	return stopPreparation{
		state: state, process: record, settled: state.settled, observedGone: treeGone,
	}, nil
}

func (adapter *Adapter) signalStopIfNeededLocked(
	state *executionState,
	record processRecord,
	runPath string,
	request ports.AgentStopRequest,
	requestHash string,
	treeGone, exactProofFound bool,
	winnerIntent stopSignalIntent,
	winnerIntentFound bool,
	winnerProof stopSignalProof,
	winnerProofFound bool,
) error {
	// Cooperative delivery remains at-most-once because replaying SIGTERM can
	// change provider behaviour. Forced delivery is safe to retry against the
	// exact persisted process identity: SIGKILL is idempotent, and retry closes
	// the crash frontier between intent fsync and the syscall.
	intentProven := winnerIntentFound && winnerProofFound &&
		winnerIntent.RequestHash == winnerProof.RequestHash &&
		winnerIntent.Idempotency == winnerProof.Idempotency &&
		winnerIntent.Mode == winnerProof.Mode && winnerIntent.Sequence == winnerProof.Sequence
	if !treeGone && winnerIntentFound && winnerIntent.Mode == ports.AgentStopForced && !intentProven {
		return adapter.signalForcedIntentLocked(state, record, runPath, winnerIntent)
	}
	// Only an explicit forced request may supersede a cooperative intent.
	shouldSignal := !treeGone && !exactProofFound &&
		(!winnerIntentFound || (winnerIntent.Mode == ports.AgentStopCooperative && request.Mode == ports.AgentStopForced))
	if !shouldSignal {
		return nil
	}
	intent, created, err := adapter.prepareStopSignalIntent(runPath, requestHash, request)
	if err != nil || !created {
		// Existing intent fences an unknowable crash-before/after-syscall effect.
		return err
	}
	if err := signalProcessTree(record, request.Mode); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	proof, err := adapter.persistStopSignalProof(runPath, intent)
	if err != nil {
		state.stopProof.Store(nil)
		return err
	}
	state.stopProof.Store(&proof)
	return nil
}

func (adapter *Adapter) signalForcedIntentLocked(
	state *executionState,
	record processRecord,
	runPath string,
	intent stopSignalIntent,
) error {
	if err := signalProcessTree(record, ports.AgentStopForced); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	proof, err := adapter.persistStopSignalProof(runPath, intent)
	if err != nil {
		state.stopProof.Store(nil)
		return err
	}
	state.stopProof.Store(&proof)
	return nil
}
