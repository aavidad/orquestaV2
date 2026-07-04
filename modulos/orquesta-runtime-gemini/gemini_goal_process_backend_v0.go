package orquestaruntimegemini

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	GeminiGoalWrapperFilePrefixV0 = "gemini_goal_wrapper_"
	GeminiGoalStdoutFilePrefixV0  = "gemini_goal_stdout_"
	GeminiGoalStderrFilePrefixV0  = "gemini_goal_stderr_"

	GeminiGoalEvidenceProcessLaunchedV0 = "evidence-ref-gemini-goal-process-launched"
	GeminiGoalEvidenceProcessRunningV0  = "evidence-ref-gemini-goal-process-running"
	GeminiGoalEvidenceProcessStoppedV0  = "evidence-ref-gemini-goal-process-stopped"
	GeminiGoalEvidenceStopRequestedV0   = "evidence-ref-gemini-goal-process-stop-requested"
	GeminiGoalEvidenceStopCompletedV0   = "evidence-ref-gemini-goal-process-stop-completed"
	GeminiGoalEvidenceStopFailedV0      = "evidence-ref-gemini-goal-process-stop-failed"

	ErrGeminiGoalProcessInvalidV0              = "gemini_goal_process_invalid"
	ErrGeminiGoalProcessLaunchFailedV0         = "gemini_goal_process_launch_failed"
	ErrGeminiGoalProcessStoppedWithoutResultV0 = "gemini_goal_process_stopped_without_result"
	ErrGeminiGoalProcessRefMissingV0           = "gemini_goal_process_ref_missing"
	ErrGeminiGoalProcessStopFailedV0           = "gemini_goal_process_stop_failed"
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
	backend.saveProcessRefV0(spec.GoalRef, snapshot.ProcessRef)
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
	processRef := backend.loadProcessRefV0(request.GoalRef)
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
	if backend == nil || backend.ProcessRuntime == nil {
		result.IssueCode = ErrGeminiGoalProcessRefMissingV0
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopFailedV0))
		return normalizeGeminiGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	processRef := backend.loadProcessRefV0(request.GoalRef)
	if processRef == "" {
		result.IssueCode = ErrGeminiGoalProcessRefMissingV0
		result.EvidenceRefs = compactGeminiGoalStringsV0(append(result.EvidenceRefs, GeminiGoalEvidenceStopFailedV0))
		return normalizeGeminiGoalStopResultV0(result), errors.New(result.IssueCode)
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

func geminiGoalWrapperFileNameV0(goalRef string) string {
	return GeminiGoalWrapperFilePrefixV0 + geminiGoalSafeRefV0(goalRef) + ".sh"
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
