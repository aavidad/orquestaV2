package orquestaruntimecodexappserver

import (
	"context"
	"errors"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	codexAppServerGoalForcedStopRequestedEvidenceV0 = "evidence-ref-codex-app-server-goal-forced-stop-requested"
	codexAppServerGoalForcedStopSetEvidenceV0       = "evidence-ref-codex-app-server-goal-forced-stop-set-blocked"
	codexAppServerGoalForcedStopSetFailedEvidenceV0 = "evidence-ref-codex-app-server-goal-forced-stop-set-failed"
	codexAppServerGoalForcedStopTmuxRequestedV0     = "evidence-ref-codex-app-server-tmux-forced-stop-requested"
	codexAppServerGoalForcedStopTmuxStoppedV0       = "evidence-ref-codex-app-server-tmux-forced-stop-stopped"
	codexAppServerGoalForcedStopTmuxFailedV0        = "evidence-ref-codex-app-server-tmux-forced-stop-failed"
)

type BackendShutdownPortV0 interface {
	ShutdownV0(context.Context) error
}

type CodexGoalStopRequestV0 struct {
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	Action          string   `json:"action,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	Forced          bool     `json:"forced,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type CodexGoalStopResultV0 struct {
	Status          string   `json:"status,omitempty"`
	GoalRef         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef string   `json:"external_goal_ref,omitempty"`
	GoalStatusSet   bool     `json:"goal_status_set,omitempty"`
	BackendStopped  bool     `json:"backend_stopped,omitempty"`
	IssueCode       string   `json:"issue_code,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

func (backend serverCodexAppServerGoalBackendV0) StopCodexGoalV0(
	ctx context.Context,
	request CodexGoalStopRequestV0,
) (CodexGoalStopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeCodexGoalStopRequestV0(request)
	result := CodexGoalStopResultV0{
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		EvidenceRefs: compactServerStackStringsV0(append(
			append([]string(nil), request.EvidenceRefs...),
			codexAppServerGoalForcedStopRequestedEvidenceV0,
		)),
	}
	if request.ExternalGoalRef == "" {
		result.IssueCode = "codex_app_server_goal_stop_thread_ref_required"
		return normalizeCodexGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	setErr := backend.setCodexGoalBlockedForForcedStopV0(ctx, request, &result)
	shutdownErr := backend.shutdownCodexGoalBackendForForcedStopV0(ctx, &result)
	result = normalizeCodexGoalStopResultV0(result)
	if shutdownErr != nil {
		return result, shutdownErr
	}
	if setErr != nil && !result.BackendStopped {
		return result, setErr
	}
	return result, nil
}

func (backend serverCodexAppServerGoalBackendV0) setCodexGoalBlockedForForcedStopV0(
	ctx context.Context,
	request CodexGoalStopRequestV0,
	result *CodexGoalStopResultV0,
) error {
	if backend.Protocol == nil {
		result.IssueCode = "codex_app_server_protocol_missing"
		result.EvidenceRefs = compactServerStackStringsV0(append(
			result.EvidenceRefs,
			codexAppServerGoalForcedStopSetFailedEvidenceV0,
		))
		return errors.New(result.IssueCode)
	}
	_, err := backend.Protocol.SetGoalV0(ctx, serverCodexAppServerThreadGoalSetParamsV0{
		ThreadID: strings.TrimSpace(request.ExternalGoalRef),
		Status:   "blocked",
	})
	if err != nil {
		result.IssueCode = codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_stop_set_failed")
		result.EvidenceRefs = compactServerStackStringsV0(append(
			result.EvidenceRefs,
			codexAppServerGoalForcedStopSetFailedEvidenceV0,
		))
		return err
	}
	result.GoalStatusSet = true
	result.EvidenceRefs = compactServerStackStringsV0(append(
		result.EvidenceRefs,
		codexAppServerGoalForcedStopSetEvidenceV0,
	))
	return nil
}

func (backend serverCodexAppServerGoalBackendV0) shutdownCodexGoalBackendForForcedStopV0(
	ctx context.Context,
	result *CodexGoalStopResultV0,
) error {
	shutdown := backend.codexAppServerBackendShutdownPortV0()
	if shutdown == nil {
		return nil
	}
	result.EvidenceRefs = compactServerStackStringsV0(append(
		result.EvidenceRefs,
		codexAppServerGoalForcedStopTmuxRequestedV0,
	))
	if err := shutdown.ShutdownV0(ctx); err != nil {
		result.IssueCode = codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_stop_shutdown_failed")
		result.EvidenceRefs = compactServerStackStringsV0(append(
			result.EvidenceRefs,
			codexAppServerGoalForcedStopTmuxFailedV0,
		))
		return err
	}
	result.BackendStopped = true
	result.EvidenceRefs = compactServerStackStringsV0(append(
		result.EvidenceRefs,
		codexAppServerGoalForcedStopTmuxStoppedV0,
	))
	return nil
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerBackendShutdownPortV0() BackendShutdownPortV0 {
	if backend.BackendShutdown != nil {
		return backend.BackendShutdown
	}
	if lazy, ok := backend.Protocol.(serverCodexAppServerLazyTmuxProtocolV0); ok {
		return lazy.Backend
	}
	return nil
}

func normalizeCodexGoalStopRequestV0(request CodexGoalStopRequestV0) CodexGoalStopRequestV0 {
	request.GoalRef = strings.TrimSpace(request.GoalRef)
	request.ExternalGoalRef = strings.TrimSpace(request.ExternalGoalRef)
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.Reason = strings.TrimSpace(request.Reason)
	request.EvidenceRefs = compactServerStackStringsV0(request.EvidenceRefs)
	return request
}

func normalizeCodexGoalStopResultV0(result CodexGoalStopResultV0) CodexGoalStopResultV0 {
	result.Status = strings.TrimSpace(result.Status)
	if result.Status == "" {
		result.Status = orquestagoal.GoalStatusBlockedV0
	}
	result.GoalRef = strings.TrimSpace(result.GoalRef)
	result.ExternalGoalRef = strings.TrimSpace(result.ExternalGoalRef)
	result.IssueCode = strings.TrimSpace(result.IssueCode)
	result.EvidenceRefs = compactServerStackStringsV0(result.EvidenceRefs)
	return result
}
