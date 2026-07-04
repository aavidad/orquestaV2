package orquestaruntimeclaude

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
	ClaudeGoalWrapperFilePrefixV0 = "claude_goal_wrapper_"
	ClaudeGoalStdoutFilePrefixV0  = "claude_goal_stdout_"
	ClaudeGoalStderrFilePrefixV0  = "claude_goal_stderr_"

	ClaudeGoalEvidenceProcessLaunchedV0 = "evidence-ref-claude-goal-process-launched"
	ClaudeGoalEvidenceProcessRunningV0  = "evidence-ref-claude-goal-process-running"
	ClaudeGoalEvidenceProcessStoppedV0  = "evidence-ref-claude-goal-process-stopped"

	ErrClaudeGoalProcessInvalidV0              = "claude_goal_process_invalid"
	ErrClaudeGoalProcessLaunchFailedV0         = "claude_goal_process_launch_failed"
	ErrClaudeGoalProcessStoppedWithoutResultV0 = "claude_goal_process_stopped_without_result"
)

type ClaudeGoalProcessBackendV0 struct {
	Control        ClaudeGoalBackendV0
	Profile        ClaudeConnectorProfileV0
	ProcessRuntime *orquestaruntime.ProcessRuntimeConnectorV0

	mu          sync.Mutex
	processRefs map[string]string
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
	backend.saveProcessRefV0(spec.GoalRef, snapshot.ProcessRef)
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
	processRef := backend.loadProcessRefV0(request.GoalRef)
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

func claudeGoalWrapperFileNameV0(goalRef string) string {
	return ClaudeGoalWrapperFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".sh"
}

func claudeGoalStdoutFileNameV0(goalRef string) string {
	return ClaudeGoalStdoutFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".log"
}

func claudeGoalStderrFileNameV0(goalRef string) string {
	return ClaudeGoalStderrFilePrefixV0 + claudeGoalSafeRefV0(goalRef) + ".log"
}
