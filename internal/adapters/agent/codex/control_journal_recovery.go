package codex

import (
	"errors"
	"io/fs"
	"path"
	"strings"

	"orquesta/internal/ports"
)

func (adapter *Adapter) loadWinningStopSignalIntent(runPath string) (stopSignalIntent, bool, error) {
	names, err := adapter.stopJournalNames(runPath)
	if err != nil {
		return stopSignalIntent{}, false, err
	}
	var winner stopSignalIntent
	foundWinner := false
	for _, name := range names {
		if !strings.HasPrefix(name, "stop-") || !strings.HasSuffix(name, ".signal-intent.json") {
			continue
		}
		var intent stopSignalIntent
		found, err := adapter.readPrivateJSON(path.Join(runPath, name), &intent)
		if err != nil {
			return stopSignalIntent{}, false, err
		}
		if !found {
			continue
		}
		if err := validateStopSignalIntent(intent); err != nil {
			return stopSignalIntent{}, false, err
		}
		if name != stopSignalIntentName(intent.Idempotency) {
			return stopSignalIntent{}, false, &Error{Code: CodeStopConflict}
		}
		if err := adapter.validateStopSignalIntentLink(runPath, intent); err != nil {
			return stopSignalIntent{}, false, err
		}
		if foundWinner && intent.Sequence == winner.Sequence && intent.RequestHash != winner.RequestHash {
			return stopSignalIntent{}, false, &Error{Code: CodeStopConflict}
		}
		if !foundWinner || intent.Sequence > winner.Sequence {
			winner, foundWinner = intent, true
		}
	}
	return winner, foundWinner, nil
}

func (adapter *Adapter) loadWinningStopSignalProof(runPath string) (stopSignalProof, bool, error) {
	names, err := adapter.stopJournalNames(runPath)
	if err != nil {
		return stopSignalProof{}, false, err
	}
	var winner stopSignalProof
	foundWinner := false
	for _, name := range names {
		if !strings.HasPrefix(name, "stop-") || !strings.HasSuffix(name, ".signal.json") {
			continue
		}
		var proof stopSignalProof
		found, err := adapter.readPrivateJSON(path.Join(runPath, name), &proof)
		if err != nil {
			return stopSignalProof{}, false, err
		}
		if !found {
			continue
		}
		if err := validateStopSignalProof(proof); err != nil {
			return stopSignalProof{}, false, err
		}
		if name != stopSignalName(proof.Idempotency) {
			return stopSignalProof{}, false, &Error{Code: CodeStopConflict}
		}
		if err := adapter.validateStopSignalProofLink(runPath, proof); err != nil {
			return stopSignalProof{}, false, err
		}
		if foundWinner && proof.Sequence == winner.Sequence && proof.RequestHash != winner.RequestHash {
			return stopSignalProof{}, false, &Error{Code: CodeStopConflict}
		}
		if !foundWinner || proof.Sequence > winner.Sequence {
			winner, foundWinner = proof, true
		}
	}
	return winner, foundWinner, nil
}

func (adapter *Adapter) validateStopSignalIntentLink(runPath string, intent stopSignalIntent) error {
	requestName, _ := stopRecordNames(intent.Idempotency)
	var request stopRequestRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, requestName), &request)
	if err != nil {
		return err
	}
	if !found || request.SchemaVersion != processSchemaVersion || request.RequestHash != intent.RequestHash ||
		request.Idempotency != intent.Idempotency || request.Mode != intent.Mode || request.RequestedAt.IsZero() {
		return &Error{Code: CodeStopConflict}
	}
	return nil
}

func (adapter *Adapter) validateStopSignalProofLink(runPath string, proof stopSignalProof) error {
	var intent stopSignalIntent
	found, err := adapter.readPrivateJSON(path.Join(runPath, stopSignalIntentName(proof.Idempotency)), &intent)
	if err != nil {
		return err
	}
	if !found {
		return &Error{Code: CodeStopConflict}
	}
	if err := validateStopSignalIntent(intent); err != nil || intent.RequestHash != proof.RequestHash ||
		intent.Idempotency != proof.Idempotency || intent.Mode != proof.Mode || intent.Sequence != proof.Sequence {
		return &Error{Code: CodeStopConflict, Cause: err}
	}
	return adapter.validateStopSignalIntentLink(runPath, intent)
}

func (adapter *Adapter) persistStopCompletion(runPath string, proof stopSignalProof) (stopCompletionRecord, error) {
	winner, found, err := adapter.loadWinningStopSignalProof(runPath)
	if err != nil {
		return stopCompletionRecord{}, err
	}
	if !found || winner != proof {
		return stopCompletionRecord{}, &Error{Code: CodeStopConflict}
	}
	now := adapter.config.Now()
	if now.IsZero() {
		return stopCompletionRecord{}, &Error{Code: CodeClockInvalid}
	}
	candidate := stopCompletionRecord{
		SchemaVersion: processSchemaVersion, RequestHash: proof.RequestHash,
		Idempotency: proof.Idempotency, Mode: proof.Mode, Sequence: proof.Sequence,
		ObservedAt: now.UTC(),
	}
	created, err := adapter.publishJSON(runPath, stopCompletionName, candidate)
	if err != nil || created {
		return candidate, err
	}
	existing, found, err := adapter.loadStopCompletion(runPath)
	if err != nil {
		return stopCompletionRecord{}, err
	}
	if !found || existing.RequestHash != candidate.RequestHash || existing.Idempotency != candidate.Idempotency ||
		existing.Mode != candidate.Mode || existing.Sequence != candidate.Sequence {
		return stopCompletionRecord{}, &Error{Code: CodeStopConflict}
	}
	return existing, nil
}

func (adapter *Adapter) loadStopCompletion(runPath string) (stopCompletionRecord, bool, error) {
	var completion stopCompletionRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, stopCompletionName), &completion)
	if err != nil || !found {
		return stopCompletionRecord{}, found, err
	}
	if completion.SchemaVersion != processSchemaVersion || completion.RequestHash == "" || completion.Idempotency == "" ||
		(completion.Mode != ports.AgentStopCooperative && completion.Mode != ports.AgentStopForced) ||
		completion.Sequence == 0 || completion.ObservedAt.IsZero() {
		return stopCompletionRecord{}, false, &Error{Code: CodeStateInvalid}
	}
	return completion, true, nil
}

func (adapter *Adapter) loadWinningStopCompletion(runPath string) (stopCompletionRecord, bool, error) {
	completion, found, err := adapter.loadStopCompletion(runPath)
	if err != nil || !found {
		return stopCompletionRecord{}, found, err
	}
	winner, winnerFound, err := adapter.loadWinningStopSignalProof(runPath)
	if err != nil {
		return stopCompletionRecord{}, false, err
	}
	if !winnerFound || completion.RequestHash != winner.RequestHash ||
		completion.Idempotency != winner.Idempotency || completion.Mode != winner.Mode || completion.Sequence != winner.Sequence {
		return stopCompletionRecord{}, false, &Error{Code: CodeStopConflict}
	}
	return completion, true, nil
}

func (adapter *Adapter) loadCausalTerminal(runPath, requestHash, specHash string, maxOutput int64) (terminalRecord, bool, error) {
	terminal, found, err := adapter.loadTerminal(runPath, requestHash, specHash, maxOutput)
	if err != nil || !found || terminal.ErrorCode != CodeExecutionStopped {
		return terminal, found, err
	}
	_, completionFound, err := adapter.loadWinningStopCompletion(runPath)
	if err != nil {
		return terminalRecord{}, false, err
	}
	if !completionFound {
		return terminalRecord{}, false, &Error{Code: CodeStateInvalid}
	}
	return terminal, true, nil
}

func (adapter *Adapter) hasDurableStopRequest(runPath string) (bool, error) {
	names, err := adapter.stopJournalNames(runPath)
	if err != nil {
		return false, err
	}
	for _, name := range names {
		if !strings.HasPrefix(name, "stop-") {
			continue
		}
		for _, suffix := range []string{".request.json", ".signal-intent.json", ".signal.json"} {
			if strings.HasSuffix(name, suffix) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (adapter *Adapter) stopJournalNames(runPath string) ([]string, error) {
	directory, err := adapter.root.Open(runPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	names, err := directory.Readdirnames(-1)
	if err != nil {
		_ = directory.Close()
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	if err := directory.Close(); err != nil {
		return nil, &Error{Code: CodeStateInvalid, Cause: err}
	}
	return names, nil
}
