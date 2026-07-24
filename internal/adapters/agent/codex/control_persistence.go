package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"strings"
	"time"

	"orquesta/internal/ports"
)

const (
	processSchemaVersion = 1
	processFileName      = "process.json"
	ownerLockFileName    = "owner.lock"
	stopCompletionName   = "stop-completion.json"
)

type processRecord struct {
	SchemaVersion int    `json:"schema_version"`
	ExecutionRef  string `json:"execution_ref"`
	RequestHash   string `json:"request_hash"`
	RuntimeScope  string `json:"runtime_scope"`
	PID           int    `json:"pid"`
	PGID          int    `json:"pgid"`
	BootID        string `json:"boot_id"`
	BirthMarker   string `json:"birth_marker"`
}

type stopRequestRecord struct {
	SchemaVersion int                 `json:"schema_version"`
	RequestHash   string              `json:"request_hash"`
	Idempotency   string              `json:"idempotency_key"`
	Mode          ports.AgentStopMode `json:"mode"`
	RequestedAt   time.Time           `json:"requested_at"`
}

type stopReceiptRecord struct {
	SchemaVersion int                   `json:"schema_version"`
	RequestHash   string                `json:"request_hash"`
	Status        ports.AgentStopStatus `json:"status"`
	ReceiptRef    string                `json:"receipt_ref"`
	ConfirmedAt   time.Time             `json:"confirmed_at"`
}

type stopSignalProof struct {
	SchemaVersion int                 `json:"schema_version"`
	RequestHash   string              `json:"request_hash"`
	Idempotency   string              `json:"idempotency_key"`
	Mode          ports.AgentStopMode `json:"mode"`
	Sequence      uint64              `json:"sequence"`
	SignaledAt    time.Time           `json:"signaled_at"`
}

type stopSignalIntent struct {
	SchemaVersion int                 `json:"schema_version"`
	RequestHash   string              `json:"request_hash"`
	Idempotency   string              `json:"idempotency_key"`
	Mode          ports.AgentStopMode `json:"mode"`
	Sequence      uint64              `json:"sequence"`
	PreparedAt    time.Time           `json:"prepared_at"`
}

type stopCompletionRecord struct {
	SchemaVersion int                 `json:"schema_version"`
	RequestHash   string              `json:"request_hash"`
	Idempotency   string              `json:"idempotency_key"`
	Mode          ports.AgentStopMode `json:"mode"`
	Sequence      uint64              `json:"sequence"`
	ObservedAt    time.Time           `json:"observed_at"`
}

type stopHashDocument struct {
	ExecutionRef      string              `json:"execution_ref"`
	GoalRef           string              `json:"goal_ref"`
	WorkItemRef       string              `json:"work_item_ref"`
	PlanGeneration    uint64              `json:"plan_generation"`
	AppSpecGeneration uint64              `json:"app_spec_generation"`
	ExecutionAttempt  uint64              `json:"execution_attempt"`
	SpecHash          string              `json:"spec_hash"`
	ProviderRef       string              `json:"provider_ref"`
	ModelRef          string              `json:"model_ref"`
	AgentRef          string              `json:"agent_ref"`
	ExternalRef       string              `json:"external_ref"`
	Mode              ports.AgentStopMode `json:"mode"`
	IdempotencyKey    string              `json:"idempotency_key"`
}

func hashStopRequest(request ports.AgentStopRequest) (string, error) {
	payload, err := json.Marshal(stopHashDocument{
		ExecutionRef: request.ExecutionRef.String(), GoalRef: request.GoalRef.String(),
		WorkItemRef: request.WorkItemRef.String(), PlanGeneration: uint64(request.PlanGeneration),
		AppSpecGeneration: uint64(request.AppSpecGeneration), ExecutionAttempt: request.ExecutionAttempt,
		SpecHash: request.SpecHash, ProviderRef: request.ProviderRef, ModelRef: request.ModelRef,
		AgentRef: request.AgentRef, ExternalRef: request.ExternalRef, Mode: request.Mode,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func stopRecordNames(idempotencyKey string) (string, string) {
	digest := sha256.Sum256([]byte(idempotencyKey))
	base := "stop-" + hex.EncodeToString(digest[:])
	return base + ".request.json", base + ".receipt.json"
}

func stopSignalName(idempotencyKey string) string {
	requestName, _ := stopRecordNames(idempotencyKey)
	return strings.TrimSuffix(requestName, ".request.json") + ".signal.json"
}

func stopSignalIntentName(idempotencyKey string) string {
	requestName, _ := stopRecordNames(idempotencyKey)
	return strings.TrimSuffix(requestName, ".request.json") + ".signal-intent.json"
}

func (adapter *Adapter) persistProcessRecord(runPath string, record processRecord) error {
	created, err := adapter.publishJSON(runPath, processFileName, record)
	if err != nil || created {
		return err
	}
	existing, found, err := adapter.readProcessRecord(runPath)
	if err != nil {
		return err
	}
	if !found || existing != record {
		return &Error{Code: CodeProcessOwnershipInvalid}
	}
	return nil
}

func (adapter *Adapter) readProcessRecord(runPath string) (processRecord, bool, error) {
	var record processRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, processFileName), &record)
	if err != nil || !found {
		return processRecord{}, found, err
	}
	if record.SchemaVersion != processSchemaVersion || record.ExecutionRef == "" ||
		record.RequestHash == "" || record.RuntimeScope == "" || record.PID <= 0 ||
		record.PGID <= 0 || record.BootID == "" || record.BirthMarker == "" {
		return processRecord{}, false, &Error{Code: CodeProcessOwnershipInvalid}
	}
	return record, true, nil
}

func (adapter *Adapter) processRecordForState(state *executionState) (processRecord, bool, error) {
	record, found, err := adapter.readProcessRecord(state.runPath)
	if err != nil || !found {
		return processRecord{}, found, err
	}
	if record.ExecutionRef != state.receipt.ExecutionRef.String() ||
		(record.RequestHash != state.requestHash && record.RequestHash != state.terminalRequestHash) ||
		record.RuntimeScope != adapter.config.RuntimeScope {
		return processRecord{}, false, &Error{Code: CodeProcessOwnershipInvalid}
	}
	return record, true, nil
}

func (adapter *Adapter) ensureStopRequest(runPath, requestHash string, request ports.AgentStopRequest) (stopRequestRecord, error) {
	requestName, _ := stopRecordNames(request.IdempotencyKey)
	now := adapter.config.Now()
	if now.IsZero() {
		return stopRequestRecord{}, &Error{Code: CodeClockInvalid}
	}
	candidate := stopRequestRecord{
		SchemaVersion: processSchemaVersion, RequestHash: requestHash,
		Idempotency: request.IdempotencyKey, Mode: request.Mode, RequestedAt: now.UTC(),
	}
	created, err := adapter.publishJSON(runPath, requestName, candidate)
	if err != nil || created {
		return candidate, err
	}
	var existing stopRequestRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, requestName), &existing)
	if err != nil {
		return stopRequestRecord{}, err
	}
	if !found || existing.SchemaVersion != processSchemaVersion || existing.RequestHash != requestHash ||
		existing.Idempotency != request.IdempotencyKey || existing.Mode != request.Mode || existing.RequestedAt.IsZero() {
		return stopRequestRecord{}, &Error{Code: CodeStopConflict}
	}
	return existing, nil
}

func (adapter *Adapter) loadStopRequest(runPath, requestHash string, request ports.AgentStopRequest) (stopRequestRecord, bool, error) {
	requestName, _ := stopRecordNames(request.IdempotencyKey)
	var record stopRequestRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, requestName), &record)
	if err != nil || !found {
		return stopRequestRecord{}, found, err
	}
	if record.SchemaVersion != processSchemaVersion || record.RequestHash != requestHash ||
		record.Idempotency != request.IdempotencyKey || record.Mode != request.Mode || record.RequestedAt.IsZero() {
		return stopRequestRecord{}, false, &Error{Code: CodeStopConflict}
	}
	return record, true, nil
}

func (adapter *Adapter) persistStopReceipt(runPath string, request ports.AgentStopRequest, requestHash string, status ports.AgentStopStatus) (ports.AgentStopReceipt, error) {
	_, receiptName := stopRecordNames(request.IdempotencyKey)
	now := adapter.config.Now()
	if now.IsZero() {
		return ports.AgentStopReceipt{}, &Error{Code: CodeClockInvalid}
	}
	record := stopReceiptRecord{
		SchemaVersion: processSchemaVersion, RequestHash: requestHash, Status: status,
		ReceiptRef: "codex-stop:" + strings.TrimPrefix(requestHash, "sha256:"), ConfirmedAt: now.UTC(),
	}
	created, err := adapter.publishJSON(runPath, receiptName, record)
	if err != nil {
		return ports.AgentStopReceipt{}, err
	}
	if !created {
		existing, found, loadErr := adapter.loadStopReceiptRecord(runPath, request, requestHash)
		if loadErr != nil {
			return ports.AgentStopReceipt{}, loadErr
		}
		if !found {
			return ports.AgentStopReceipt{}, &Error{Code: CodeStateInvalid}
		}
		record = existing
	}
	return newStopReceipt(request, record.Status, record.ReceiptRef, record.ConfirmedAt), nil
}

func (adapter *Adapter) loadStopReceiptRecord(runPath string, request ports.AgentStopRequest, requestHash string) (stopReceiptRecord, bool, error) {
	_, receiptName := stopRecordNames(request.IdempotencyKey)
	var record stopReceiptRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, receiptName), &record)
	if err != nil || !found {
		return stopReceiptRecord{}, found, err
	}
	if record.SchemaVersion != processSchemaVersion || record.RequestHash != requestHash ||
		record.ReceiptRef != "codex-stop:"+strings.TrimPrefix(requestHash, "sha256:") || record.ConfirmedAt.IsZero() {
		return stopReceiptRecord{}, false, &Error{Code: CodeStopConflict}
	}
	switch record.Status {
	case ports.AgentStopped, ports.AgentStopAlreadyStopped, ports.AgentStopAlreadyCompleted, ports.AgentStopAlreadyFailed:
	default:
		return stopReceiptRecord{}, false, &Error{Code: CodeStateInvalid}
	}
	return record, true, nil
}

func (adapter *Adapter) loadStopSignalProof(runPath, requestHash string, request ports.AgentStopRequest) (stopSignalProof, bool, error) {
	var proof stopSignalProof
	found, err := adapter.readPrivateJSON(path.Join(runPath, stopSignalName(request.IdempotencyKey)), &proof)
	if err != nil || !found {
		return stopSignalProof{}, found, err
	}
	if err := validateStopSignalProof(proof); err != nil || proof.RequestHash != requestHash ||
		proof.Idempotency != request.IdempotencyKey || proof.Mode != request.Mode {
		return stopSignalProof{}, false, &Error{Code: CodeStopConflict, Cause: err}
	}
	if err := adapter.validateStopSignalProofLink(runPath, proof); err != nil {
		return stopSignalProof{}, false, err
	}
	return proof, true, nil
}

func (adapter *Adapter) prepareStopSignalIntent(runPath, requestHash string, request ports.AgentStopRequest) (stopSignalIntent, bool, error) {
	if existing, found, err := adapter.loadStopSignalIntent(runPath, requestHash, request); err != nil || found {
		return existing, false, err
	}
	winner, found, err := adapter.loadWinningStopSignalIntent(runPath)
	if err != nil {
		return stopSignalIntent{}, false, err
	}
	now := adapter.config.Now()
	if now.IsZero() {
		return stopSignalIntent{}, false, &Error{Code: CodeClockInvalid}
	}
	sequence := uint64(1)
	if found {
		sequence = winner.Sequence + 1
		if sequence == 0 {
			return stopSignalIntent{}, false, &Error{Code: CodeStateInvalid}
		}
	}
	intent := stopSignalIntent{
		SchemaVersion: processSchemaVersion, RequestHash: requestHash,
		Idempotency: request.IdempotencyKey, Mode: request.Mode,
		Sequence: sequence, PreparedAt: now.UTC(),
	}
	created, err := adapter.publishJSON(runPath, stopSignalIntentName(request.IdempotencyKey), intent)
	if err != nil {
		return stopSignalIntent{}, false, err
	}
	if !created {
		existing, found, err := adapter.loadStopSignalIntent(runPath, requestHash, request)
		if err != nil || !found {
			return stopSignalIntent{}, false, errors.Join(err, &Error{Code: CodeStopConflict})
		}
		return existing, false, nil
	}
	return intent, true, nil
}

func (adapter *Adapter) loadStopSignalIntent(runPath, requestHash string, request ports.AgentStopRequest) (stopSignalIntent, bool, error) {
	var intent stopSignalIntent
	found, err := adapter.readPrivateJSON(path.Join(runPath, stopSignalIntentName(request.IdempotencyKey)), &intent)
	if err != nil || !found {
		return stopSignalIntent{}, found, err
	}
	if err := validateStopSignalIntent(intent); err != nil || intent.RequestHash != requestHash ||
		intent.Idempotency != request.IdempotencyKey || intent.Mode != request.Mode {
		return stopSignalIntent{}, false, &Error{Code: CodeStopConflict, Cause: err}
	}
	if err := adapter.validateStopSignalIntentLink(runPath, intent); err != nil {
		return stopSignalIntent{}, false, err
	}
	return intent, true, nil
}

func (adapter *Adapter) persistStopSignalProof(runPath string, intent stopSignalIntent) (stopSignalProof, error) {
	now := adapter.config.Now()
	if now.IsZero() {
		return stopSignalProof{}, &Error{Code: CodeClockInvalid}
	}
	proof := stopSignalProof{
		SchemaVersion: processSchemaVersion, RequestHash: intent.RequestHash,
		Idempotency: intent.Idempotency, Mode: intent.Mode,
		Sequence: intent.Sequence, SignaledAt: now.UTC(),
	}
	created, err := adapter.publishJSON(runPath, stopSignalName(intent.Idempotency), proof)
	if err != nil {
		return stopSignalProof{}, err
	}
	if !created {
		var existing stopSignalProof
		found, err := adapter.readPrivateJSON(path.Join(runPath, stopSignalName(intent.Idempotency)), &existing)
		if err != nil || !found || existing != proof {
			return stopSignalProof{}, errors.Join(err, &Error{Code: CodeStopConflict})
		}
		return existing, nil
	}
	return proof, nil
}

func validateStopSignalIntent(intent stopSignalIntent) error {
	if intent.SchemaVersion != processSchemaVersion || intent.RequestHash == "" || intent.Idempotency == "" ||
		(intent.Mode != ports.AgentStopCooperative && intent.Mode != ports.AgentStopForced) ||
		intent.Sequence == 0 || intent.PreparedAt.IsZero() {
		return &Error{Code: CodeStateInvalid}
	}
	return nil
}

func validateStopSignalProof(proof stopSignalProof) error {
	if proof.SchemaVersion != processSchemaVersion || proof.RequestHash == "" || proof.Idempotency == "" ||
		(proof.Mode != ports.AgentStopCooperative && proof.Mode != ports.AgentStopForced) ||
		proof.Sequence == 0 || proof.SignaledAt.IsZero() {
		return &Error{Code: CodeStateInvalid}
	}
	return nil
}
