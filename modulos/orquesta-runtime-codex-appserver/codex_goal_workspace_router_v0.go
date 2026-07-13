package orquestaruntimecodexappserver

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexGoalWorkspaceUnavailableIssueV0 = "codex_goal_workspace_unavailable"

type GoalWorkspaceBindingV0 struct {
	ProjectWorkDir string
	EvidenceRefs   []string
}

type GoalWorkspaceRouterPortV0 interface {
	PrepareCodexGoalWorkspaceV0(context.Context, orquestaruntimecodexgoal.CodexGoalStartPacketV0) (GoalWorkspaceBindingV0, error)
	ResolveCodexGoalWorkspaceV0(context.Context, orquestaruntimecodexgoal.CodexGoalObservationRequestV0) (GoalWorkspaceBindingV0, error)
}

// GoalWorkspaceExecutionBinderPortV0 commits the externally-created Codex
// thread to the already prepared durable authority before launch success is
// exposed to callers. It is optional only for pre-057 packets with no authority.
type GoalWorkspaceExecutionBinderPortV0 interface {
	BindCodexGoalExecutionV0(context.Context, orquestaruntimecodexgoal.CodexGoalObservationRequestV0) error
}

func (backend serverCodexAppServerGoalBackendV0) startCodexGoalInResolvedWorkspaceV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	binding, err := backend.WorkspaceRouter.PrepareCodexGoalWorkspaceV0(ctx, packet)
	if err != nil || !validCodexGoalWorkspaceBindingV0(binding) {
		return codexAppServerStartReceiptV0(packet, "", codexGoalWorkspaceUnavailableIssueV0), errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	resolved := backend
	resolved.CWD = filepath.Clean(strings.TrimSpace(binding.ProjectWorkDir))
	resolved.WorkspaceRouter = nil
	receipt, err := resolved.StartCodexGoalV0(ctx, packet)
	receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, binding.EvidenceRefs...))
	_, hasAuthority, authorityErr := codexAppServerStartAuthorityV0(packet)
	if err == nil && authorityErr == nil && hasAuthority {
		binder, ok := backend.WorkspaceRouter.(GoalWorkspaceExecutionBinderPortV0)
		bindRequest := orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
			SchemaVersion: orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
			GoalRef:       packet.GoalRef, ExternalGoalRef: receipt.ExternalGoalRef,
			IntentManifestRef: packet.IntentManifestRef, IntentManifestSHA256: packet.IntentManifestSHA256,
			WorkspaceAuthoritySchemaVersion: orquestagoal.GoalWorkspaceAuthoritySchemaV0,
			WorkspaceRef:                    packet.WorkspaceRef, ProviderRef: packet.ProviderRef,
			RuntimeGenerationRef: receipt.RuntimeGenerationRef,
		}
		if !ok || strings.TrimSpace(receipt.ExternalGoalRef) == "" || strings.TrimSpace(receipt.RuntimeGenerationRef) == "" || binder.BindCodexGoalExecutionV0(ctx, bindRequest) != nil {
			if resolved.Protocol != nil && strings.TrimSpace(receipt.ExternalGoalRef) != "" {
				_, _ = resolved.Protocol.SetGoalV0(ctx, serverCodexAppServerThreadGoalSetParamsV0{ThreadID: receipt.ExternalGoalRef, Status: "blocked"})
			}
			receipt.IssueCode = codexGoalWorkspaceUnavailableIssueV0
			return receipt, errors.New(codexGoalWorkspaceUnavailableIssueV0)
		}
		receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, "evidence-ref-codex-goal-execution-authority-bound"))
	}
	return receipt, err
}

func (backend serverCodexAppServerGoalBackendV0) observeCodexGoalInResolvedWorkspaceV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	binding, err := backend.WorkspaceRouter.ResolveCodexGoalWorkspaceV0(ctx, request)
	if err != nil || !validCodexGoalWorkspaceBindingV0(binding) {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, codexGoalWorkspaceUnavailableIssueV0), errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	resolved := backend
	resolved.CWD = filepath.Clean(strings.TrimSpace(binding.ProjectWorkDir))
	resolved.WorkspaceRouter = nil
	resolved.workspaceAuthorityVerified = true
	receipt, err := resolved.ObserveCodexGoalV0(ctx, request)
	receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, binding.EvidenceRefs...))
	return receipt, err
}

func (backend serverCodexAppServerGoalBackendV0) fingerprintCodexGoalInResolvedWorkspaceV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalObservationFingerprintV0, bool, error) {
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
	authority := orquestagoal.GoalObservationRequestFromStateV0(state)
	binding, err := backend.WorkspaceRouter.ResolveCodexGoalWorkspaceV0(ctx, orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:                   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:                         authority.GoalRef,
		ExternalGoalRef:                 firstNonEmptyServerStackV0(authority.ExternalGoalRef, state.LaunchReceipt.ExternalGoalRef),
		IntentManifestRef:               authority.IntentManifestRef,
		IntentManifestSHA256:            authority.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: authority.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    authority.WorkspaceRef,
		ProviderRef:                     authority.ProviderRef,
		RuntimeGenerationRef:            authority.RuntimeGenerationRef,
	})
	if err != nil || !validCodexGoalWorkspaceBindingV0(binding) {
		return orquestagoal.GoalObservationFingerprintV0{}, true, errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	resolved := backend
	resolved.CWD = filepath.Clean(strings.TrimSpace(binding.ProjectWorkDir))
	resolved.WorkspaceRouter = nil
	resolved.workspaceAuthorityVerified = true
	return resolved.FingerprintGoalObservationV0(ctx, state)
}

func validCodexGoalWorkspaceBindingV0(binding GoalWorkspaceBindingV0) bool {
	path := strings.TrimSpace(binding.ProjectWorkDir)
	return path != "" && filepath.IsAbs(path)
}
