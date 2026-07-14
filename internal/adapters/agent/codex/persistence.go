package codex

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	legacyStateSchemaVersion = 3
	stateSchemaVersion       = 4
	requestFileName          = "request.json"
	launchUpgradeFileName    = "request-v4-upgrade.json"
	terminalFileName         = "terminal.json"
)

type launchRecord struct {
	SchemaVersion     int                    `json:"schema_version"`
	RequestHash       string                 `json:"request_hash"`
	ExecutionRef      string                 `json:"execution_ref"`
	GoalRef           string                 `json:"goal_ref"`
	WorkItemRef       string                 `json:"work_item_ref"`
	PlanGeneration    goal.PlanGeneration    `json:"plan_generation"`
	AppSpecGeneration goal.AppSpecGeneration `json:"app_spec_generation"`
	ExecutionAttempt  uint64                 `json:"execution_attempt"`
	SpecHash          string                 `json:"spec_hash"`
	ProviderRef       string                 `json:"provider_ref"`
	ModelRef          string                 `json:"model_ref"`
	AgentRef          string                 `json:"agent_ref"`
	ExternalRef       string                 `json:"external_ref"`
	IdempotencyKey    string                 `json:"idempotency_key"`
	AcceptedAt        time.Time              `json:"accepted_at"`
	MaxOutputBytes    int64                  `json:"max_output_bytes"`
}

type terminalRecord struct {
	SchemaVersion       int               `json:"schema_version"`
	RequestHash         string            `json:"request_hash"`
	Status              ports.AgentStatus `json:"status"`
	MediaType           string            `json:"media_type,omitempty"`
	Artifact            string            `json:"artifact,omitempty"`
	ErrorCode           string            `json:"error_code,omitempty"`
	ObservedAt          time.Time         `json:"observed_at"`
	Diagnostic          []byte            `json:"diagnostic,omitempty"`
	DiagnosticTruncated bool              `json:"diagnostic_truncated,omitempty"`
}

// launchUpgradeRecord binds the causal fields introduced in schema V4 to one
// immutable V3 launch. The original request.json remains untouched so the
// adapter can still verify its exact historical hash.
type launchUpgradeRecord struct {
	SchemaVersion       int          `json:"schema_version"`
	SourceSchemaVersion int          `json:"source_schema_version"`
	SourceRequestHash   string       `json:"source_request_hash"`
	Launch              launchRecord `json:"launch"`
}

type legacyRequestHashDocument struct {
	SchemaVersion      int      `json:"schema_version"`
	ExecutionRef       string   `json:"execution_ref"`
	GoalRef            string   `json:"goal_ref"`
	WorkItemRef        string   `json:"work_item_ref"`
	SpecHash           string   `json:"spec_hash"`
	ActorRef           string   `json:"actor_ref"`
	ProjectRef         string   `json:"project_ref"`
	Objective          string   `json:"objective"`
	PhaseRef           string   `json:"phase_ref"`
	PhaseKey           string   `json:"phase_key"`
	PhaseTemplateRef   string   `json:"phase_template_ref"`
	PhaseInputRefs     []string `json:"phase_input_refs"`
	PhaseCriterionRefs []string `json:"phase_criterion_refs"`
	RoleKey            string   `json:"role_key"`
	SkillRefs          []string `json:"skill_refs"`
	ToolRefs           []string `json:"tool_refs"`
	CapabilityRefs     []string `json:"capability_refs"`
	WriteSet           []string `json:"write_set"`
	OutputContract     string   `json:"output_contract"`
	ArtifactMediaType  string   `json:"artifact_media_type"`
	IdempotencyKey     string   `json:"idempotency_key"`
	MaxOutputBytes     int64    `json:"max_output_bytes"`
}

type requestHashDocument struct {
	SchemaVersion      int                    `json:"schema_version"`
	ExecutionRef       string                 `json:"execution_ref"`
	GoalRef            string                 `json:"goal_ref"`
	WorkItemRef        string                 `json:"work_item_ref"`
	PlanGeneration     goal.PlanGeneration    `json:"plan_generation"`
	AppSpecGeneration  goal.AppSpecGeneration `json:"app_spec_generation"`
	ExecutionAttempt   uint64                 `json:"execution_attempt"`
	SpecHash           string                 `json:"spec_hash"`
	ActorRef           string                 `json:"actor_ref"`
	ProjectRef         string                 `json:"project_ref"`
	Objective          string                 `json:"objective"`
	PhaseRef           string                 `json:"phase_ref"`
	PhaseKey           string                 `json:"phase_key"`
	PhaseTemplateRef   string                 `json:"phase_template_ref"`
	PhaseInputRefs     []string               `json:"phase_input_refs"`
	PhaseCriterionRefs []string               `json:"phase_criterion_refs"`
	RoleKey            string                 `json:"role_key"`
	SkillRefs          []string               `json:"skill_refs"`
	ToolRefs           []string               `json:"tool_refs"`
	CapabilityRefs     []string               `json:"capability_refs"`
	WriteSet           []string               `json:"write_set"`
	OutputContract     string                 `json:"output_contract"`
	ArtifactMediaType  string                 `json:"artifact_media_type"`
	IdempotencyKey     string                 `json:"idempotency_key"`
	MaxOutputBytes     int64                  `json:"max_output_bytes"`
}

func hashLaunchRequest(request ports.AgentLaunchRequest) (string, error) {
	document := requestHashDocument{
		SchemaVersion:      stateSchemaVersion,
		ExecutionRef:       request.ExecutionRef.String(),
		GoalRef:            request.GoalRef.String(),
		WorkItemRef:        request.WorkItemRef.String(),
		PlanGeneration:     request.PlanGeneration,
		AppSpecGeneration:  request.AppSpecGeneration,
		ExecutionAttempt:   request.ExecutionAttempt,
		SpecHash:           request.SpecHash,
		ActorRef:           request.ActorRef.String(),
		ProjectRef:         request.ProjectRef.String(),
		Objective:          request.Objective,
		PhaseRef:           request.PhaseRef,
		PhaseKey:           request.PhaseKey,
		PhaseTemplateRef:   request.PhaseTemplateRef,
		PhaseInputRefs:     append([]string(nil), request.PhaseInputRefs...),
		PhaseCriterionRefs: append([]string(nil), request.PhaseCriterionRefs...),
		RoleKey:            request.RoleKey,
		SkillRefs:          append([]string(nil), request.SkillRefs...),
		ToolRefs:           append([]string(nil), request.ToolRefs...),
		CapabilityRefs:     append([]string(nil), request.CapabilityRefs...),
		WriteSet:           append([]string(nil), request.WriteSet...),
		OutputContract:     request.OutputContract,
		ArtifactMediaType:  request.ArtifactMediaType,
		IdempotencyKey:     request.IdempotencyKey,
		MaxOutputBytes:     request.MaxOutputBytes,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func hashLegacyLaunchRequest(request ports.AgentLaunchRequest) (string, error) {
	document := legacyRequestHashDocument{
		SchemaVersion:      legacyStateSchemaVersion,
		ExecutionRef:       request.ExecutionRef.String(),
		GoalRef:            request.GoalRef.String(),
		WorkItemRef:        request.WorkItemRef.String(),
		SpecHash:           request.SpecHash,
		ActorRef:           request.ActorRef.String(),
		ProjectRef:         request.ProjectRef.String(),
		Objective:          request.Objective,
		PhaseRef:           request.PhaseRef,
		PhaseKey:           request.PhaseKey,
		PhaseTemplateRef:   request.PhaseTemplateRef,
		PhaseInputRefs:     append([]string(nil), request.PhaseInputRefs...),
		PhaseCriterionRefs: append([]string(nil), request.PhaseCriterionRefs...),
		RoleKey:            request.RoleKey,
		SkillRefs:          append([]string(nil), request.SkillRefs...),
		ToolRefs:           append([]string(nil), request.ToolRefs...),
		CapabilityRefs:     append([]string(nil), request.CapabilityRefs...),
		WriteSet:           append([]string(nil), request.WriteSet...),
		OutputContract:     request.OutputContract,
		ArtifactMediaType:  request.ArtifactMediaType,
		IdempotencyKey:     request.IdempotencyKey,
		MaxOutputBytes:     request.MaxOutputBytes,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func executionPath(executionRef goal.ExecutionRef) string {
	digest := sha256.Sum256([]byte(executionRef.String()))
	return path.Join("executions", hex.EncodeToString(digest[:]))
}

func (adapter *Adapter) ensureLaunchRecord(request ports.AgentLaunchRequest, requestHash string) (launchRecord, string, bool, error) {
	runPath := executionPath(request.ExecutionRef)
	if err := adapter.ensurePrivateDirectory("executions"); err != nil {
		return launchRecord{}, "", false, err
	}
	if err := adapter.ensurePrivateDirectory(runPath); err != nil {
		return launchRecord{}, "", false, err
	}
	acceptedAt := adapter.config.Now()
	if acceptedAt.IsZero() {
		return launchRecord{}, "", false, &Error{Code: CodeClockInvalid}
	}
	candidate := launchRecord{
		SchemaVersion:     stateSchemaVersion,
		RequestHash:       requestHash,
		ExecutionRef:      request.ExecutionRef.String(),
		GoalRef:           request.GoalRef.String(),
		WorkItemRef:       request.WorkItemRef.String(),
		PlanGeneration:    request.PlanGeneration,
		AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt:  request.ExecutionAttempt,
		SpecHash:          request.SpecHash,
		ProviderRef:       ProviderRef,
		ModelRef:          adapter.modelRef(),
		AgentRef:          AgentRef,
		ExternalRef:       "codex:" + path.Base(runPath),
		IdempotencyKey:    request.IdempotencyKey,
		AcceptedAt:        acceptedAt.UTC(),
		MaxOutputBytes:    request.MaxOutputBytes,
	}
	created, err := adapter.publishJSON(runPath, requestFileName, candidate)
	if err != nil {
		return launchRecord{}, "", false, err
	}
	if created {
		return candidate, runPath, true, nil
	}
	existing, found, err := adapter.readLaunchRecord(runPath)
	if err != nil {
		return launchRecord{}, "", false, err
	}
	if !found {
		return launchRecord{}, "", false, &Error{Code: CodeStateInvalid}
	}
	return existing, runPath, false, nil
}

func (adapter *Adapter) loadLaunchRecord(executionRef goal.ExecutionRef) (launchRecord, string, bool, error) {
	runPath := executionPath(executionRef)
	if err := adapter.validatePrivateDirectory(runPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return launchRecord{}, runPath, false, nil
		}
		return launchRecord{}, runPath, false, err
	}
	record, found, err := adapter.readLaunchRecord(runPath)
	if err != nil || !found {
		return launchRecord{}, runPath, found, err
	}
	if err := adapter.syncDirectoryCausally(runPath); err != nil {
		return launchRecord{}, runPath, false, err
	}
	if record.ExecutionRef != executionRef.String() {
		return launchRecord{}, runPath, false, &Error{Code: CodeStateInvalid}
	}
	return record, runPath, true, nil
}

func (adapter *Adapter) readLaunchRecord(runPath string) (launchRecord, bool, error) {
	var record launchRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, requestFileName), &record)
	if err != nil || !found {
		return launchRecord{}, found, err
	}
	if record.RequestHash == "" ||
		record.ExecutionRef == "" ||
		record.SpecHash == "" ||
		record.ProviderRef != ProviderRef ||
		record.ExternalRef == "" ||
		record.IdempotencyKey == "" ||
		record.AcceptedAt.IsZero() ||
		record.MaxOutputBytes <= 0 {
		return launchRecord{}, false, &Error{Code: CodeStateInvalid}
	}
	switch record.SchemaVersion {
	case legacyStateSchemaVersion:
		if record.GoalRef != "" || record.WorkItemRef != "" ||
			record.PlanGeneration != 0 || record.AppSpecGeneration != 0 || record.ExecutionAttempt != 0 ||
			record.ModelRef != "" || record.AgentRef != "" {
			return launchRecord{}, false, &Error{Code: CodeStateInvalid}
		}
	case stateSchemaVersion:
		if err := validateLaunchRecordV4(record); err != nil {
			return launchRecord{}, false, err
		}
	default:
		return launchRecord{}, false, &Error{Code: CodeStateInvalid}
	}
	return record, true, nil
}

func validateLaunchRecordV4(record launchRecord) error {
	if record.SchemaVersion != stateSchemaVersion ||
		record.GoalRef == "" ||
		record.WorkItemRef == "" ||
		record.PlanGeneration == 0 ||
		record.AppSpecGeneration == 0 ||
		record.ExecutionAttempt == 0 ||
		record.ModelRef == "" ||
		record.AgentRef != AgentRef {
		return &Error{Code: CodeStateInvalid}
	}
	return nil
}

func (adapter *Adapter) bindLegacyLaunchRecord(
	runPath string,
	legacy launchRecord,
	request ports.AgentLaunchRequest,
	requestHash string,
) (launchRecord, error) {
	legacyHash, err := hashLegacyLaunchRequest(request)
	if err != nil {
		return launchRecord{}, &Error{Code: CodeStateInvalid, Cause: err}
	}
	if legacy.RequestHash != legacyHash {
		return launchRecord{}, &Error{Code: CodeExecutionConflict}
	}
	candidate := launchRecord{
		SchemaVersion:     stateSchemaVersion,
		RequestHash:       requestHash,
		ExecutionRef:      legacy.ExecutionRef,
		GoalRef:           request.GoalRef.String(),
		WorkItemRef:       request.WorkItemRef.String(),
		PlanGeneration:    request.PlanGeneration,
		AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt:  request.ExecutionAttempt,
		SpecHash:          legacy.SpecHash,
		ProviderRef:       legacy.ProviderRef,
		ModelRef:          adapter.modelRef(),
		AgentRef:          AgentRef,
		ExternalRef:       legacy.ExternalRef,
		IdempotencyKey:    legacy.IdempotencyKey,
		AcceptedAt:        legacy.AcceptedAt,
		MaxOutputBytes:    legacy.MaxOutputBytes,
	}
	upgrade := launchUpgradeRecord{
		SchemaVersion:       stateSchemaVersion,
		SourceSchemaVersion: legacyStateSchemaVersion,
		SourceRequestHash:   legacy.RequestHash,
		Launch:              candidate,
	}
	created, err := adapter.publishJSON(runPath, launchUpgradeFileName, upgrade)
	if err != nil {
		return launchRecord{}, err
	}
	if !created {
		found, readErr := adapter.readPrivateJSON(path.Join(runPath, launchUpgradeFileName), &upgrade)
		if readErr != nil {
			return launchRecord{}, readErr
		}
		if !found {
			return launchRecord{}, &Error{Code: CodeStateInvalid}
		}
	}
	if err := validateLegacyLaunchUpgrade(upgrade, legacy); err != nil {
		return launchRecord{}, err
	}
	if upgrade.Launch.RequestHash != requestHash {
		return launchRecord{}, &Error{Code: CodeExecutionConflict}
	}
	return upgrade.Launch, nil
}

func validateLegacyLaunchUpgrade(upgrade launchUpgradeRecord, legacy launchRecord) error {
	if upgrade.SchemaVersion != stateSchemaVersion ||
		upgrade.SourceSchemaVersion != legacyStateSchemaVersion ||
		upgrade.SourceRequestHash != legacy.RequestHash {
		return &Error{Code: CodeStateInvalid}
	}
	if err := validateLaunchRecordV4(upgrade.Launch); err != nil {
		return err
	}
	launch := upgrade.Launch
	if launch.ExecutionRef != legacy.ExecutionRef ||
		launch.SpecHash != legacy.SpecHash ||
		launch.ProviderRef != legacy.ProviderRef ||
		launch.ExternalRef != legacy.ExternalRef ||
		launch.IdempotencyKey != legacy.IdempotencyKey ||
		!launch.AcceptedAt.Equal(legacy.AcceptedAt) ||
		launch.MaxOutputBytes != legacy.MaxOutputBytes {
		return &Error{Code: CodeStateInvalid}
	}
	return nil
}

func (record launchRecord) receipt(executionRef goal.ExecutionRef) (ports.AgentLaunchReceipt, error) {
	if record.ExecutionRef != executionRef.String() {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid}
	}
	goalRef, err := goal.NewGoalRef(record.GoalRef)
	if err != nil || goalRef.String() != record.GoalRef {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid}
	}
	workItemRef, err := goal.NewWorkItemRef(record.WorkItemRef)
	if err != nil || workItemRef.String() != record.WorkItemRef {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid}
	}
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef:      executionRef,
		GoalRef:           goalRef,
		WorkItemRef:       workItemRef,
		PlanGeneration:    record.PlanGeneration,
		AppSpecGeneration: record.AppSpecGeneration,
		ExecutionAttempt:  record.ExecutionAttempt,
		SpecHash:          record.SpecHash,
		ProviderRef:       record.ProviderRef,
		ModelRef:          record.ModelRef,
		AgentRef:          record.AgentRef,
		ExternalRef:       record.ExternalRef,
		IdempotencyKey:    record.IdempotencyKey,
		AcceptedAt:        record.AcceptedAt,
	}
	request := ports.AgentLaunchRequest{
		ExecutionRef:      executionRef,
		GoalRef:           goalRef,
		WorkItemRef:       workItemRef,
		PlanGeneration:    record.PlanGeneration,
		AppSpecGeneration: record.AppSpecGeneration,
		ExecutionAttempt:  record.ExecutionAttempt,
		SpecHash:          record.SpecHash,
		IdempotencyKey:    record.IdempotencyKey,
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid}
	}
	return receipt, nil
}

func (adapter *Adapter) observationReceipt(record launchRecord, executionRef goal.ExecutionRef) (ports.AgentLaunchReceipt, error) {
	if record.SchemaVersion == stateSchemaVersion {
		return record.receipt(executionRef)
	}
	if record.SchemaVersion != legacyStateSchemaVersion || record.ExecutionRef != executionRef.String() {
		return ports.AgentLaunchReceipt{}, &Error{Code: CodeStateInvalid}
	}
	// Observe only needs the durable identity/spec/timestamp carried by V3. Goal
	// causality is bound later, if Launch is replayed, by request-v4-upgrade.json.
	return ports.AgentLaunchReceipt{
		ExecutionRef:   executionRef,
		SpecHash:       record.SpecHash,
		ProviderRef:    record.ProviderRef,
		ModelRef:       adapter.modelRef(),
		AgentRef:       AgentRef,
		ExternalRef:    record.ExternalRef,
		IdempotencyKey: record.IdempotencyKey,
		AcceptedAt:     record.AcceptedAt,
	}, nil
}

func (adapter *Adapter) persistTerminal(runPath string, terminal terminalRecord, specHash string, maxOutput int64) (terminalRecord, error) {
	created, err := adapter.publishJSON(runPath, terminalFileName, terminal)
	if err != nil {
		return terminalRecord{}, err
	}
	if created {
		return terminal, nil
	}
	existing, found, err := adapter.loadTerminal(runPath, terminal.RequestHash, specHash, maxOutput)
	if err != nil {
		return terminalRecord{}, err
	}
	if !found {
		return terminalRecord{}, &Error{Code: CodeStateInvalid}
	}
	return existing, nil
}

func (adapter *Adapter) loadTerminal(runPath, requestHash, specHash string, maxOutput int64) (terminalRecord, bool, error) {
	var terminal terminalRecord
	found, err := adapter.readPrivateJSON(path.Join(runPath, terminalFileName), &terminal)
	if err != nil || !found {
		return terminalRecord{}, found, err
	}
	if (terminal.SchemaVersion != legacyStateSchemaVersion && terminal.SchemaVersion != stateSchemaVersion) ||
		terminal.RequestHash != requestHash || terminal.ObservedAt.IsZero() {
		return terminalRecord{}, false, &Error{Code: CodeStateInvalid}
	}
	observation := terminal.observation(goal.ExecutionRef{}, specHash)
	observation.ExecutionRef, _ = goal.NewExecutionRef("execution:state-validation")
	if err := ports.ValidateAgentObservation(observation, maxOutput); err != nil {
		return terminalRecord{}, false, &Error{Code: CodeStateInvalid, Cause: err}
	}
	if err := adapter.syncDirectoryCausally(runPath); err != nil {
		return terminalRecord{}, false, err
	}
	return terminal, true, nil
}

func (terminal terminalRecord) observation(executionRef goal.ExecutionRef, specHash string) ports.AgentObservation {
	return ports.AgentObservation{
		ExecutionRef: executionRef,
		SpecHash:     specHash,
		Status:       terminal.Status,
		MediaType:    terminal.MediaType,
		Content:      append([]byte(nil), []byte(terminal.Artifact)...),
		ErrorCode:    terminal.ErrorCode,
		ObservedAt:   terminal.ObservedAt,
	}
}

func (adapter *Adapter) ensurePrivateDirectory(directory string) error {
	directory = path.Clean(directory)
	err := adapter.validatePrivateDirectory(directory)
	if err == nil {
		return adapter.syncDirectoryCausally(directory)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	parent := path.Dir(directory)
	if parent != "." && parent != directory {
		if err := adapter.ensurePrivateDirectory(parent); err != nil {
			return err
		}
	}
	if err := adapter.root.Mkdir(directory, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	if err := adapter.validatePrivateDirectory(directory); err != nil {
		return err
	}
	return adapter.syncDirectoryCausally(directory)
}

func (adapter *Adapter) validatePrivateDirectory(directory string) error {
	info, err := adapter.root.Lstat(directory)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return &Error{Code: CodeStateInvalid}
	}
	return nil
}

func (adapter *Adapter) publishJSON(directory, name string, value any) (bool, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	payload = append(payload, '\n')
	temporaryName, err := temporaryPath(directory, name)
	if err != nil {
		return false, err
	}
	temporary, err := adapter.root.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = adapter.root.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(payload); err != nil {
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	if err := temporary.Sync(); err != nil {
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	if err := temporary.Close(); err != nil {
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	finalName := path.Join(directory, name)
	if err := adapter.root.Link(temporaryName, finalName); err != nil {
		if _, statErr := adapter.root.Lstat(finalName); statErr == nil {
			if syncErr := adapter.syncDirectoryCausally(directory); syncErr != nil {
				return false, syncErr
			}
			return false, nil
		}
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	if err := adapter.root.Remove(temporaryName); err != nil {
		return false, &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	removeTemporary = false
	if err := adapter.syncDirectoryCausally(directory); err != nil {
		return false, err
	}
	return true, nil
}

func (adapter *Adapter) readPrivateJSON(filePath string, target any) (bool, error) {
	info, err := adapter.root.Lstat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, &Error{Code: CodeStateInvalid, Cause: err}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return false, &Error{Code: CodeStateInvalid}
	}
	payload, err := adapter.root.ReadFile(filePath)
	if err != nil {
		return false, &Error{Code: CodeStateInvalid, Cause: err}
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false, &Error{Code: CodeStateInvalid, Cause: err}
	}
	if err := requireJSONEOF(decoder); err != nil {
		return false, &Error{Code: CodeStateInvalid, Cause: err}
	}
	return true, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("trailing JSON value")
	}
	return err
}

func temporaryPath(directory, name string) (string, error) {
	var suffix [12]byte
	if _, err := io.ReadFull(rand.Reader, suffix[:]); err != nil {
		return "", &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	return path.Join(directory, fmt.Sprintf(".%s-%s.tmp", name, hex.EncodeToString(suffix[:]))), nil
}

func (adapter *Adapter) syncDirectory(directory string) error {
	syncFn := adapter.syncDirectoryFn
	if syncFn == nil {
		syncFn = syncCodexDirectory
	}
	if err := syncFn(adapter.root, directory); err != nil {
		return &Error{Code: CodeStatePersistenceFailed, Cause: err}
	}
	return nil
}

func syncCodexDirectory(root *os.Root, directory string) error {
	handle, err := root.Open(directory)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}

func (adapter *Adapter) syncDirectoryCausally(directory string) error {
	return syncDirectoryAndParent(directory, adapter.syncDirectory)
}

func syncDirectoryAndParent(directory string, syncDirectory func(string) error) error {
	directory = path.Clean(directory)
	if err := syncDirectory(directory); err != nil {
		return err
	}
	parent := path.Dir(directory)
	if parent == directory {
		return nil
	}
	return syncDirectory(parent)
}
