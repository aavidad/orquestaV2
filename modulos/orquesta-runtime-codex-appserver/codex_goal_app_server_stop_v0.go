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

type BackendForcedStopShutdownPortV0 interface {
	ShutdownForcedStopV0(context.Context) error
}

type CodexGoalStopRequestV0 struct {
	GoalRef                         string   `json:"goal_ref,omitempty"`
	ExternalGoalRef                 string   `json:"external_goal_ref,omitempty"`
	IntentManifestRef               string   `json:"intent_manifest_ref,omitempty"`
	IntentManifestSHA256            string   `json:"intent_manifest_sha256,omitempty"`
	WorkspaceAuthoritySchemaVersion string   `json:"workspace_authority_schema_version,omitempty"`
	WorkspaceRef                    string   `json:"workspace_ref,omitempty"`
	ProviderRef                     string   `json:"provider_ref,omitempty"`
	RuntimeGenerationRef            string   `json:"runtime_generation_ref,omitempty"`
	Action                          string   `json:"action,omitempty"`
	Reason                          string   `json:"reason,omitempty"`
	Forced                          bool     `json:"forced,omitempty"`
	EvidenceRefs                    []string `json:"evidence_refs,omitempty"`
	WorkspaceAuthorityVerified      bool     `json:"-"`
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
	_, lazyProtocol := backend.Protocol.(serverCodexAppServerLazyTmuxProtocolV0)
	if err := codexAppServerStopAuthorityV0(request, !lazyProtocol); err != nil {
		result.IssueCode = "codex_app_server_goal_stop_authority_invalid"
		return normalizeCodexGoalStopResultV0(result), err
	}
	authority, hasAuthority := codexAppServerExecutionAuthorityFromStopV0(request)
	if hasAuthority {
		if backend.Runtime == nil {
			result.IssueCode = "codex_app_server_goal_authority_runtime_missing"
			return normalizeCodexGoalStopResultV0(result), errors.New(result.IssueCode)
		}
		bound := backend.Runtime.threadBoundToAuthorityV0(request.ExternalGoalRef, authority)
		if !bound && request.WorkspaceAuthorityVerified {
			bound = backend.Runtime.bindThreadAuthorityV0(request.ExternalGoalRef, authority)
		}
		if !bound {
			result.IssueCode = "codex_app_server_goal_thread_authority_mismatch"
			return normalizeCodexGoalStopResultV0(result), errors.New(result.IssueCode)
		}
	}
	if lazy, ok := backend.Protocol.(serverCodexAppServerLazyTmuxProtocolV0); ok {
		return backend.stopCodexGoalWithLazyGenerationV0(ctx, request, result, lazy, authority, hasAuthority)
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

func (backend serverCodexAppServerGoalBackendV0) stopCodexGoalWithLazyGenerationV0(
	ctx context.Context,
	request CodexGoalStopRequestV0,
	result CodexGoalStopResultV0,
	lazy serverCodexAppServerLazyTmuxProtocolV0,
	authority orquestagoal.GoalExecutionAuthorityV0,
	hasAuthority bool,
) (CodexGoalStopResultV0, error) {
	if backend.Runtime == nil || request.RuntimeGenerationRef == "" {
		result.IssueCode = codexAppServerTmuxGenerationConflictV0
		return normalizeCodexGoalStopResultV0(result), errors.New(result.IssueCode)
	}
	var setErr, bindingErr error
	generationRef, generationErr := lazy.withVerifiedGenerationV0(ctx, request.RuntimeGenerationRef, func(protocol serverCodexAppServerProtocolPortV0, leasedGenerationRef string) error {
		bound := backend.Runtime.bindThreadGenerationV0(request.ExternalGoalRef, leasedGenerationRef)
		if hasAuthority {
			bound = backend.Runtime.threadBoundToAuthorityV0(request.ExternalGoalRef, authority)
		}
		if !bound {
			bindingErr = errors.New(codexAppServerTmuxGenerationConflictV0)
			return bindingErr
		}
		scoped := backend
		scoped.Protocol = protocol
		setErr = scoped.setCodexGoalBlockedForForcedStopV0(ctx, request, &result)
		return setErr
	})
	if bindingErr != nil || generationRef != request.RuntimeGenerationRef ||
		(generationErr != nil && (setErr == nil || !errors.Is(generationErr, setErr))) {
		result.IssueCode = codexAppServerTmuxGenerationConflictV0
		return normalizeCodexGoalStopResultV0(result), firstNonNilCodexGoalStopV0(bindingErr, generationErr, errors.New(result.IssueCode))
	}
	result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs, codexAppServerGoalForcedStopTmuxRequestedV0))
	if err := lazy.Backend.shutdownForcedStopGenerationV0(ctx, request.RuntimeGenerationRef); err != nil {
		result.IssueCode = codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_stop_shutdown_failed")
		result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs, codexAppServerGoalForcedStopTmuxFailedV0))
		return normalizeCodexGoalStopResultV0(result), err
	}
	result.BackendStopped = true
	result.EvidenceRefs = compactServerStackStringsV0(append(result.EvidenceRefs, codexAppServerGoalForcedStopTmuxStoppedV0))
	return normalizeCodexGoalStopResultV0(result), nil
}

func firstNonNilCodexGoalStopV0(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
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
	if forcedShutdown, ok := shutdown.(BackendForcedStopShutdownPortV0); ok && forcedShutdown != nil {
		if err := forcedShutdown.ShutdownForcedStopV0(ctx); err != nil {
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
	request.IntentManifestRef = strings.TrimSpace(request.IntentManifestRef)
	request.IntentManifestSHA256 = strings.TrimSpace(request.IntentManifestSHA256)
	request.WorkspaceAuthoritySchemaVersion = strings.TrimSpace(request.WorkspaceAuthoritySchemaVersion)
	request.WorkspaceRef = strings.TrimSpace(request.WorkspaceRef)
	request.ProviderRef = strings.TrimSpace(request.ProviderRef)
	request.RuntimeGenerationRef = strings.TrimSpace(request.RuntimeGenerationRef)
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	request.Reason = strings.TrimSpace(request.Reason)
	request.EvidenceRefs = compactServerStackStringsV0(request.EvidenceRefs)
	return request
}

func codexAppServerStopAuthorityV0(request CodexGoalStopRequestV0, requireDeterministicGeneration bool) error {
	authority, _ := codexAppServerExecutionAuthorityFromStopV0(request)
	return codexAppServerValidateExecutionAuthorityV0(authority, requireDeterministicGeneration)
}

func codexAppServerExecutionAuthorityFromStopV0(request CodexGoalStopRequestV0) (orquestagoal.GoalExecutionAuthorityV0, bool) {
	authority := orquestagoal.NormalizeGoalExecutionAuthorityV0(orquestagoal.GoalExecutionAuthorityV0{
		GoalRef: request.GoalRef, IntentManifestRef: request.IntentManifestRef,
		IntentManifestSHA256:            request.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: request.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    request.WorkspaceRef,
		ProviderRef:                     request.ProviderRef,
		RuntimeGenerationRef:            request.RuntimeGenerationRef,
	})
	return authority, !codexAppServerExecutionAuthorityLegacyV0(authority)
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
