package orquestaruntimeclaude

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	ClaudeGoalWrapperFilePrefixV0  = "claude_goal_wrapper_"
	ClaudeGoalStdoutFilePrefixV0   = "claude_goal_stdout_"
	ClaudeGoalStderrFilePrefixV0   = "claude_goal_stderr_"
	ClaudeGoalProcessStatePrefixV0 = "claude_goal_process_state_"
	ClaudeGoalProcessStateSchemaV0 = "claude_goal_process_state.v0"

	ClaudeGoalEvidenceProcessLaunchedV0 = "evidence-ref-claude-goal-process-launched"
	ClaudeGoalEvidenceProcessRunningV0  = "evidence-ref-claude-goal-process-running"
	ClaudeGoalEvidenceProcessStoppedV0  = "evidence-ref-claude-goal-process-stopped"
	ClaudeGoalEvidenceProcessAdoptedV0  = "evidence-ref-claude-goal-process-adopted"
	ClaudeGoalEvidenceStopRequestedV0   = "evidence-ref-claude-goal-process-stop-requested"
	ClaudeGoalEvidenceStopCompletedV0   = "evidence-ref-claude-goal-process-stop-completed"
	ClaudeGoalEvidenceStopFailedV0      = "evidence-ref-claude-goal-process-stop-failed"

	ErrClaudeGoalProcessInvalidV0              = "claude_goal_process_invalid"
	ErrClaudeGoalProcessLaunchFailedV0         = "claude_goal_process_launch_failed"
	ErrClaudeGoalProcessStoppedWithoutResultV0 = "claude_goal_process_stopped_without_result"
	ErrClaudeGoalProcessRefMissingV0           = "claude_goal_process_ref_missing"
	ErrClaudeGoalProcessStopFailedV0           = "claude_goal_process_stop_failed"
	ErrClaudeGoalProcessStateWriteFailedV0     = "claude_goal_process_state_write_failed"
	ErrClaudeGoalProcessStateReadFailedV0      = "claude_goal_process_state_read_failed"
)

type ClaudeGoalProcessBackendV0 struct {
	Control        ClaudeGoalBackendV0
	Profile        ClaudeConnectorProfileV0
	ProcessRuntime *orquestaruntime.ProcessRuntimeConnectorV0

	mu          sync.Mutex
	processRefs map[string]string
}

type ClaudeGoalStopRequestV0 struct {
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	Action          string   `json:"action,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	Forced          bool     `json:"forced,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type ClaudeGoalStopResultV0 struct {
	Status          string   `json:"status,omitempty"`
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	GoalStatusSet   bool     `json:"goal_status_set,omitempty"`
	BackendStopped  bool     `json:"backend_stopped,omitempty"`
	IssueCode       string   `json:"issue_code,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type claudeGoalProcessStateV0 struct {
	SchemaVersion string `json:"schema_version"`
	GoalRef       string `json:"goal_ref"`
	ProcessRef    string `json:"process_ref"`
	SessionRef    string `json:"session_ref,omitempty"`
	LaunchRef     string `json:"launch_ref,omitempty"`
	PID           int    `json:"pid"`
}

func (backend *ClaudeGoalProcessBackendV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if backend == nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalProcessInvalidV0, "backend"), errors.New(ErrClaudeGoalProcessInvalidV0)
	}
	receipt, err := backend.Control.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		return receipt, err
	}
	if receipt.Status != orquestagoal.GoalStatusRunningV0 {
		return receipt, nil
	}
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	profile := backend.normalizedProfileV0()
	if issues := ValidateClaudeConnectorProfileV0(profile); len(issues) > 0 {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalProcessInvalidV0, "profile"), errors.New(ErrClaudeGoalProcessInvalidV0)
	}
	if err := backend.materializeGoalWrapperV0(profile, spec.GoalRef); err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalProcessInvalidV0, "wrapper"), err
	}
	runtime := backend.processRuntimeV0()
	snapshot, err := runtime.LaunchV0(ctx, orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: filepath.Join(profile.RuntimeWorkDir, claudeGoalWrapperFileNameV0(spec.GoalRef)),
		Env:         []string{},
		WorkingDir:  profile.ProjectWorkDir,
	})
	if err != nil {
		return claudeGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrClaudeGoalProcessLaunchFailedV0, "process"), err
	}
	if err := backend.saveProcessSnapshotV0(spec.GoalRef, snapshot); err != nil {
		receipt.Issues = append(receipt.Issues, orquestagoal.GoalWorkIssueV0{
			Code:  ErrClaudeGoalProcessStateWriteFailedV0,
			Field: "process_state",
		})
		backend.saveProcessRefV0(spec.GoalRef, snapshot.ProcessRef)
	}
	receipt.EvidenceRefs = compactClaudeGoalStringsV0(append(
		receipt.EvidenceRefs,
		ClaudeGoalEvidenceProcessLaunchedV0,
	))
	receipt.EvidenceRefs = compactClaudeGoalStringsV0(append(
		receipt.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	return orquestagoal.NormalizeGoalLaunchReceiptV0(receipt), nil
}

func (backend *ClaudeGoalProcessBackendV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if backend == nil {
		return claudeGoalInvalidResultV0(request, ErrClaudeGoalProcessInvalidV0, "backend"), errors.New(ErrClaudeGoalProcessInvalidV0)
	}
	result, err := backend.Control.ObserveGoalWorkV0(ctx, request)
	if err != nil || result.Status != orquestagoal.GoalStatusRunningV0 {
		return result, err
	}
	processRef, adopted := backend.ensureProcessRefV0(ctx, request.GoalRef)
	if processRef == "" || backend.ProcessRuntime == nil {
		return result, nil
	}
	snapshot, err := backend.ProcessRuntime.SnapshotV0(processRef)
	if err != nil {
		result.Issues = append(result.Issues, orquestagoal.GoalWorkIssueV0{
			Code:  ErrClaudeGoalProcessInvalidV0,
			Field: "process_ref",
		})
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	}
	result.EvidenceRefs = compactClaudeGoalStringsV0(append(
		result.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	if adopted {
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceProcessAdoptedV0))
	}
	switch snapshot.Status {
	case orquestaruntime.ProcessRuntimeRunningV0, orquestaruntime.ProcessRuntimeStoppingV0:
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceProcessRunningV0))
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	case orquestaruntime.ProcessRuntimeStoppedV0:
		result.Status = orquestagoal.GoalStatusBlockedV0
		result.Summary = ErrClaudeGoalProcessStoppedWithoutResultV0
		result.Issues = append(result.Issues, orquestagoal.GoalWorkIssueV0{
			Code:   ErrClaudeGoalProcessStoppedWithoutResultV0,
			Field:  "result",
			Detail: ClaudeGoalEvidenceProcessStoppedV0,
		})
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceProcessStoppedV0))
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	default:
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	}
}

func (backend *ClaudeGoalProcessBackendV0) StopClaudeGoalV0(
	ctx context.Context,
	request ClaudeGoalStopRequestV0,
) (ClaudeGoalStopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeClaudeGoalStopRequestV0(request)
	result := ClaudeGoalStopResultV0{
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		GoalStatusSet:   true,
		EvidenceRefs: compactClaudeGoalStringsV0(append(
			append([]string(nil), request.EvidenceRefs...),
			ClaudeGoalEvidenceStopRequestedV0,
		)),
	}
	if backend == nil {
		result.IssueCode = ErrClaudeGoalProcessRefMissingV0
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceStopFailedV0))
		return normalizeClaudeGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	processRef, adopted := backend.ensureProcessRefV0(ctx, request.GoalRef)
	if processRef == "" {
		result.IssueCode = ErrClaudeGoalProcessRefMissingV0
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceStopFailedV0))
		return normalizeClaudeGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	if adopted {
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceProcessAdoptedV0))
	}
	snapshot, err := backend.ProcessRuntime.StopV0(ctx, processRef)
	result.EvidenceRefs = compactClaudeGoalStringsV0(append(
		result.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	if err != nil {
		result.IssueCode = ErrClaudeGoalProcessStopFailedV0
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceStopFailedV0))
		return normalizeClaudeGoalStopResultV0(result), err
	}
	result.BackendStopped = snapshot.Status == orquestaruntime.ProcessRuntimeStoppedV0
	if result.BackendStopped {
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceStopCompletedV0))
	} else {
		result.IssueCode = ErrClaudeGoalProcessStopFailedV0
		result.EvidenceRefs = compactClaudeGoalStringsV0(append(result.EvidenceRefs, ClaudeGoalEvidenceStopFailedV0))
	}
	return normalizeClaudeGoalStopResultV0(result), nil
}

func (backend *ClaudeGoalProcessBackendV0) normalizedProfileV0() ClaudeConnectorProfileV0 {
	profile := backend.Profile
	if profile.SchemaVersion == "" {
		profile.SchemaVersion = ClaudeConnectorProfileSchemaVersionV0
	}
	profile.OptIn = true
	if strings.TrimSpace(profile.ProjectWorkDir) == "" {
		profile.ProjectWorkDir = backend.Control.ProjectWorkDir
	}
	if strings.TrimSpace(profile.RuntimeWorkDir) == "" {
		profile.RuntimeWorkDir = backend.Control.RuntimeWorkDir
	}
	if strings.TrimSpace(profile.RuntimeWorkDirPlacement) == "" {
		profile.RuntimeWorkDirPlacement = InferClaudeRuntimeWorkDirPlacementV0(profile.ProjectWorkDir, profile.RuntimeWorkDir)
	}
	if strings.TrimSpace(profile.PromptLocale) == "" {
		profile.PromptLocale = strings.TrimSpace(backend.Control.PromptLocale)
	}
	return profile
}

func (backend *ClaudeGoalProcessBackendV0) materializeGoalWrapperV0(
	profile ClaudeConnectorProfileV0,
	goalRef string,
) error {
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		return err
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, claudeGoalWrapperFileNameV0(goalRef))
	promptPath := filepath.Join(profile.RuntimeWorkDir, claudeGoalPromptFileNameV0(goalRef))
	stdoutPath := filepath.Join(profile.RuntimeWorkDir, claudeGoalStdoutFileNameV0(goalRef))
	stderrPath := filepath.Join(profile.RuntimeWorkDir, claudeGoalStderrFileNameV0(goalRef))
	wrapper := BuildClaudeGoalWrapperScriptV0(profile, promptPath, stdoutPath, stderrPath)
	return writeClaudeControlFileV0(profile.RuntimeWorkDir, wrapperPath, filepath.Base(wrapperPath), []byte(wrapper), 0o700)
}

func (backend *ClaudeGoalProcessBackendV0) processRuntimeV0() *orquestaruntime.ProcessRuntimeConnectorV0 {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.ProcessRuntime == nil {
		backend.ProcessRuntime = orquestaruntime.NewProcessRuntimeConnectorV0()
	}
	return backend.ProcessRuntime
}

func (backend *ClaudeGoalProcessBackendV0) saveProcessRefV0(goalRef string, processRef string) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.processRefs == nil {
		backend.processRefs = map[string]string{}
	}
	backend.processRefs[strings.TrimSpace(goalRef)] = strings.TrimSpace(processRef)
}

func (backend *ClaudeGoalProcessBackendV0) loadProcessRefV0(goalRef string) string {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return strings.TrimSpace(backend.processRefs[strings.TrimSpace(goalRef)])
}

func (backend *ClaudeGoalProcessBackendV0) ensureProcessRefV0(ctx context.Context, goalRef string) (string, bool) {
	if backend == nil {
		return "", false
	}
	if processRef := backend.loadProcessRefV0(goalRef); processRef != "" {
		return processRef, false
	}
	state, ok := backend.loadProcessStateV0(goalRef)
	if !ok {
		return "", false
	}
	runtime := backend.processRuntimeV0()
	snapshot, err := runtime.AdoptProcessV0(ctx, orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    state.ProcessRef,
		SessionRef:    state.SessionRef,
		LaunchRef:     state.LaunchRef,
		PID:           state.PID,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	})
	if err != nil {
		return "", false
	}
	backend.saveProcessRefV0(goalRef, snapshot.ProcessRef)
	return snapshot.ProcessRef, true
}

func (backend *ClaudeGoalProcessBackendV0) saveProcessSnapshotV0(
	goalRef string,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) error {
	backend.saveProcessRefV0(goalRef, snapshot.ProcessRef)
	state := claudeGoalProcessStateV0{
		SchemaVersion: ClaudeGoalProcessStateSchemaV0,
		GoalRef:       strings.TrimSpace(goalRef),
		ProcessRef:    strings.TrimSpace(snapshot.ProcessRef),
		SessionRef:    strings.TrimSpace(snapshot.SessionRef),
		LaunchRef:     strings.TrimSpace(snapshot.LaunchRef),
		PID:           snapshot.PID,
	}
	return writeClaudeJSONFileV0(
		backend.Control.RuntimeWorkDir,
		filepath.Join(backend.Control.RuntimeWorkDir, claudeGoalProcessStateFileNameV0(goalRef)),
		state,
	)
}

func (backend *ClaudeGoalProcessBackendV0) loadProcessStateV0(goalRef string) (claudeGoalProcessStateV0, bool) {
	path := filepath.Join(backend.Control.RuntimeWorkDir, claudeGoalProcessStateFileNameV0(goalRef))
	data, err := os.ReadFile(path)
	if err != nil {
		return claudeGoalProcessStateV0{}, false
	}
	var state claudeGoalProcessStateV0
	if err := json.Unmarshal(data, &state); err != nil {
		return claudeGoalProcessStateV0{}, false
	}
	if state.SchemaVersion != ClaudeGoalProcessStateSchemaV0 ||
		strings.TrimSpace(state.GoalRef) != strings.TrimSpace(goalRef) ||
		strings.TrimSpace(state.ProcessRef) == "" ||
		state.PID <= 0 {
		return claudeGoalProcessStateV0{}, false
	}
	return state, true
}

func claudeGoalWrapperFileNameV0(goalRef string) string {
	return ClaudeGoalWrapperFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".sh"
}

func claudeGoalProcessStateFileNameV0(goalRef string) string {
	return ClaudeGoalProcessStatePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".json"
}

func claudeGoalStdoutFileNameV0(goalRef string) string {
	return ClaudeGoalStdoutFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".log"
}

func claudeGoalStderrFileNameV0(goalRef string) string {
	return ClaudeGoalStderrFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".log"
}

func normalizeClaudeGoalStopRequestV0(request ClaudeGoalStopRequestV0) ClaudeGoalStopRequestV0 {
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.ExternalGoalRef = strings.TrimSpace(request.ExternalGoalRef)
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.Reason = strings.TrimSpace(request.Reason)
	request.EvidenceRefs = compactClaudeGoalStringsV0(request.EvidenceRefs)
	return request
}

func normalizeClaudeGoalStopResultV0(result ClaudeGoalStopResultV0) ClaudeGoalStopResultV0 {
	result.Status = strings.TrimSpace(result.Status)
	if result.Status == "" {
		result.Status = orquestagoal.GoalStatusBlockedV0
	}
	result.GoalRef = strings.TrimSpace(result.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(result.ExternalGoalRef)
	result.IssueCode = strings.TrimSpace(result.IssueCode)
	result.EvidenceRefs = compactClaudeGoalStringsV0(result.EvidenceRefs)
	return result
}
