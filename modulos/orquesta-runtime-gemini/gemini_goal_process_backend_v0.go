package orquestaruntimegemini

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
	GeminiGoalWrapperFilePrefixV0  = "gemini_goal_wrapper_"
	GeminiGoalStdoutFilePrefixV0   = "gemini_goal_stdout_"
	GeminiGoalStderrFilePrefixV0   = "gemini_goal_stderr_"
	GeminiGoalProcessStatePrefixV0 = "gemini_goal_process_state_"
	GeminiGoalProcessStateSchemaV0 = "gemini_goal_process_state.v0"

	GeminiGoalEvidenceProcessLaunchedV0 = "evidence-ref-gemini-goal-process-launched"
	GeminiGoalEvidenceProcessRunningV0  = "evidence-ref-gemini-goal-process-running"
	GeminiGoalEvidenceProcessStoppedV0  = "evidence-ref-gemini-goal-process-stopped"
	GeminiGoalEvidenceProcessAdoptedV0  = "evidence-ref-gemini-goal-process-adopted"
	GeminiGoalEvidenceStopRequestedV0   = "evidence-ref-gemini-goal-process-stop-requested"
	GeminiGoalEvidenceStopCompletedV0   = "evidence-ref-gemini-goal-process-stop-completed"
	GeminiGoalEvidenceStopFailedV0      = "evidence-ref-gemini-goal-process-stop-failed"

	ErrGeminiGoalProcessInvalidV0              = "gemini_goal_process_invalid"
	ErrGeminiGoalProcessLaunchFailedV0         = "gemini_goal_process_launch_failed"
	ErrGeminiGoalProcessStoppedWithoutResultV0 = "gemini_goal_process_stopped_without_result"
	ErrGeminiGoalProcessRefMissingV0           = "gemini_goal_process_ref_missing"
	ErrGeminiGoalProcessStopFailedV0           = "gemini_goal_process_stop_failed"
	ErrGeminiGoalProcessStateWriteFailedV0     = "gemini_goal_process_state_write_failed"
	ErrGeminiGoalProcessStateReadFailedV0      = "gemini_goal_process_state_read_failed"
)

type GeminiGoalProcessBackendV0 struct {
	Control        GeminiGoalBackendV0
	Profile        GeminiConnectorProfileV0
	ProcessRuntime *orquestaruntime.ProcessRuntimeConnectorV0

	mu          sync.Mutex
	processRefs map[string]string
}

type GeminiGoalStopRequestV0 struct {
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	Action          string   `json:"action,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	Forced          bool     `json:"forced,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type GeminiGoalStopResultV0 struct {
	Status          string   `json:"status,omitempty"`
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	GoalStatusSet   bool     `json:"goal_status_set,omitempty"`
	BackendStopped  bool     `json:"backend_stopped,omitempty"`
	IssueCode       string   `json:"issue_code,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type geminiGoalProcessStateV0 struct {
	SchemaVersion string `json:"schema_version"`
	GoalRef       string `json:"goal_ref"`
	ProcessRef    string `json:"process_ref"`
	SessionRef    string `json:"session_ref,omitempty"`
	LaunchRef     string `json:"launch_ref,omitempty"`
	PID           int    `json:"pid"`
}

func (backend *GeminiGoalProcessBackendV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if backend == nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalProcessInvalidV0, "backend"), errors.New(ErrGeminiGoalProcessInvalidV0)
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
	if issues := ValidateGeminiConnectorProfileV0(profile); len(issues) > 0 {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalProcessInvalidV0, "profile"), errors.New(ErrGeminiGoalProcessInvalidV0)
	}
	if err := backend.materializeGoalWrapperV0(profile, spec.GoalRef); err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalProcessInvalidV0, "wrapper"), err
	}
	runtime := backend.processRuntimeV0()
	snapshot, err := runtime.LaunchV0(ctx, orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: filepath.Join(profile.RuntimeWorkDir, geminiGoalWrapperFileNameV0(spec.GoalRef)),
		Env:         []string{},
		WorkingDir:  profile.ProjectWorkDir,
	})
	if err != nil {
		return geminiGoalInvalidLaunchReceiptV0(spec.GoalRef, ErrGeminiGoalProcessLaunchFailedV0, "process"), err
	}
	if err := backend.saveProcessSnapshotV0(spec.GoalRef, snapshot); err != nil {
		receipt.Issues = append(receipt.Issues, orquestagoal.GoalWorkIssueV0{
			Code:  ErrGeminiGoalProcessStateWriteFailedV0,
			Field: "process_state",
		})
		backend.saveProcessRefV0(spec.GoalRef, snapshot.ProcessRef)
	}
	receipt.EvidenceRefs = compactGeminiGoalStringsV0(append(
		receipt.EvidenceRefs,
		GeminiGoalEvidenceProcessLaunchedV0,
	))
	receipt.EvidenceRefs = compactGeminiGoalStringsV0(append(
		receipt.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	return orquestagoal.NormalizeGoalLaunchReceiptV0(receipt), nil
}

func (backend *GeminiGoalProcessBackendV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if backend == nil {
		return geminiGoalInvalidResultV0(request, ErrGeminiGoalProcessInvalidV0, "backend"), errors.New(ErrGeminiGoalProcessInvalidV0)
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
			Code:  ErrGeminiGoalProcessInvalidV0,
			Field: "process_ref",
		})
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	}
	result.EvidenceRefs = compactGeminiGoalStringsV0(append(
		result.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	if adopted {
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceProcessAdoptedV0))
	}
	switch snapshot.Status {
	case orquestaruntime.ProcessRuntimeRunningV0, orquestaruntime.ProcessRuntimeStoppingV0:
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceProcessRunningV0))
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	case orquestaruntime.ProcessRuntimeStoppedV0:
		result.Status = orquestagoal.GoalStatusBlockedV0
		result.Summary = ErrGeminiGoalProcessStoppedWithoutResultV0
		result.Issues = append(result.Issues, orquestagoal.GoalWorkIssueV0{
			Code:   ErrGeminiGoalProcessStoppedWithoutResultV0,
			Field:  "result",
			Detail: GeminiGoalEvidenceProcessStoppedV0,
		})
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceProcessStoppedV0))
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	default:
		return orquestagoal.NormalizeGoalWorkResultV0(result), nil
	}
}

func (backend *GeminiGoalProcessBackendV0) StopGeminiGoalV0(
	ctx context.Context,
	request GeminiGoalStopRequestV0,
) (GeminiGoalStopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeGeminiGoalStopRequestV0(request)
	result := GeminiGoalStopResultV0{
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		GoalStatusSet:   true,
		EvidenceRefs: compactGeminiGoalStringsV0(append(
			append([]string(nil), request.EvidenceRefs...),
			GeminiGoalEvidenceStopRequestedV0,
		)),
	}
	if backend == nil {
		result.IssueCode = ErrGeminiGoalProcessRefMissingV0
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopFailedV0))
		return normalizeGeminiGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	processRef, adopted := backend.ensureProcessRefV0(ctx, request.GoalRef)
	if processRef == "" {
		result.IssueCode = ErrGeminiGoalProcessRefMissingV0
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopFailedV0))
		return normalizeGeminiGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	if adopted {
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceProcessAdoptedV0))
	}
	snapshot, err := backend.ProcessRuntime.StopV0(ctx, processRef)
	result.EvidenceRefs = compactGeminiGoalStringsV0(append(
		result.EvidenceRefs,
		orquestaruntime.ProcessRuntimeSnapshotEvidenceRefsV0(snapshot)...,
	))
	if err != nil {
		result.IssueCode = ErrGeminiGoalProcessStopFailedV0
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopFailedV0))
		return normalizeGeminiGoalStopResultV0(result), err
	}
	result.BackendStopped = snapshot.Status == orquestaruntime.ProcessRuntimeStoppedV0
	if result.BackendStopped {
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopCompletedV0))
	} else {
		result.IssueCode = ErrGeminiGoalProcessStopFailedV0
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopFailedV0))
	}
	return normalizeGeminiGoalStopResultV0(result), nil
}

func (backend *GeminiGoalProcessBackendV0) normalizedProfileV0() GeminiConnectorProfileV0 {
	profile := backend.Profile
	if profile.SchemaVersion == "" {
		profile.SchemaVersion = GeminiConnectorProfileSchemaVersionV0
	}
	profile.OptIn = true
	if strings.TrimSpace(profile.ProjectWorkDir) == "" {
		profile.ProjectWorkDir = backend.Control.ProjectWorkDir
	}
	if strings.TrimSpace(profile.RuntimeWorkDir) == "" {
		profile.RuntimeWorkDir = backend.Control.RuntimeWorkDir
	}
	if strings.TrimSpace(profile.RuntimeWorkDirPlacement) == "" {
		profile.RuntimeWorkDirPlacement = InferGeminiRuntimeWorkDirPlacementV0(profile.ProjectWorkDir, profile.RuntimeWorkDir)
	}
	if strings.TrimSpace(profile.PromptLocale) == "" {
		profile.PromptLocale = strings.TrimSpace(backend.Control.PromptLocale)
	}
	return profile
}

func (backend *GeminiGoalProcessBackendV0) materializeGoalWrapperV0(
	profile GeminiConnectorProfileV0,
	goalRef string,
) error {
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		return err
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, geminiGoalWrapperFileNameV0(goalRef))
	promptPath := filepath.Join(profile.RuntimeWorkDir, geminiGoalPromptFileNameV0(goalRef))
	stdoutPath := filepath.Join(profile.RuntimeWorkDir, geminiGoalStdoutFileNameV0(goalRef))
	stderrPath := filepath.Join(profile.RuntimeWorkDir, geminiGoalStderrFileNameV0(goalRef))
	wrapper := BuildGeminiGoalWrapperScriptV0(profile, promptPath, stdoutPath, stderrPath)
	return writeGeminiControlFileV0(profile.RuntimeWorkDir, wrapperPath, filepath.Base(wrapperPath), []byte(wrapper), 0o700)
}

func (backend *GeminiGoalProcessBackendV0) processRuntimeV0() *orquestaruntime.ProcessRuntimeConnectorV0 {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.ProcessRuntime == nil {
		backend.ProcessRuntime = orquestaruntime.NewProcessRuntimeConnectorV0()
	}
	return backend.ProcessRuntime
}

func (backend *GeminiGoalProcessBackendV0) saveProcessRefV0(goalRef string, processRef string) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.processRefs == nil {
		backend.processRefs = map[string]string{}
	}
	backend.processRefs[strings.TrimSpace(goalRef)] = strings.TrimSpace(processRef)
}

func (backend *GeminiGoalProcessBackendV0) loadProcessRefV0(goalRef string) string {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return strings.TrimSpace(backend.processRefs[strings.TrimSpace(goalRef)])
}

func (backend *GeminiGoalProcessBackendV0) ensureProcessRefV0(ctx context.Context, goalRef string) (string, bool) {
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

func (backend *GeminiGoalProcessBackendV0) saveProcessSnapshotV0(
	goalRef string,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) error {
	backend.saveProcessRefV0(goalRef, snapshot.ProcessRef)
	state := geminiGoalProcessStateV0{
		SchemaVersion: GeminiGoalProcessStateSchemaV0,
		GoalRef:       strings.TrimSpace(goalRef),
		ProcessRef:    strings.TrimSpace(snapshot.ProcessRef),
		SessionRef:    strings.TrimSpace(snapshot.SessionRef),
		LaunchRef:     strings.TrimSpace(snapshot.LaunchRef),
		PID:           snapshot.PID,
	}
	return writeGeminiJSONFileV0(
		backend.Control.RuntimeWorkDir,
		filepath.Join(backend.Control.RuntimeWorkDir, geminiGoalProcessStateFileNameV0(goalRef)),
		state,
	)
}

func (backend *GeminiGoalProcessBackendV0) loadProcessStateV0(goalRef string) (geminiGoalProcessStateV0, bool) {
	path := filepath.Join(backend.Control.RuntimeWorkDir, geminiGoalProcessStateFileNameV0(goalRef))
	data, err := os.ReadFile(path)
	if err != nil {
		return geminiGoalProcessStateV0{}, false
	}
	var state geminiGoalProcessStateV0
	if err := json.Unmarshal(data, &state); err != nil {
		return geminiGoalProcessStateV0{}, false
	}
	if state.SchemaVersion != GeminiGoalProcessStateSchemaV0 ||
		strings.TrimSpace(state.GoalRef) != strings.TrimSpace(goalRef) ||
		strings.TrimSpace(state.ProcessRef) == "" ||
		state.PID <= 0 {
		return geminiGoalProcessStateV0{}, false
	}
	return state, true
}

func geminiGoalWrapperFileNameV0(goalRef string) string {
	return GeminiGoalWrapperFilePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".sh"
}

func geminiGoalProcessStateFileNameV0(goalRef string) string {
	return GeminiGoalProcessStatePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".json"
}

func geminiGoalStdoutFileNameV0(goalRef string) string {
	return GeminiGoalStdoutFilePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".log"
}

func geminiGoalStderrFileNameV0(goalRef string) string {
	return GeminiGoalStderrFilePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".log"
}

func normalizeGeminiGoalStopRequestV0(request GeminiGoalStopRequestV0) GeminiGoalStopRequestV0 {
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.ExternalGoalRef = strings.TrimSpace(request.ExternalGoalRef)
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.Reason = strings.TrimSpace(request.Reason)
	request.EvidenceRefs = compactGeminiGoalStringsV0(request.EvidenceRefs)
	return request
}

func normalizeGeminiGoalStopResultV0(result GeminiGoalStopResultV0) GeminiGoalStopResultV0 {
	result.Status = strings.TrimSpace(result.Status)
	if result.Status == "" {
		result.Status = orquestagoal.GoalStatusBlockedV0
	}
	result.GoalRef = strings.TrimSpace(result.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(result.ExternalGoalRef)
	result.IssueCode = strings.TrimSpace(result.IssueCode)
	result.EvidenceRefs = compactGeminiGoalStringsV0(result.EvidenceRefs)
	return result
}
