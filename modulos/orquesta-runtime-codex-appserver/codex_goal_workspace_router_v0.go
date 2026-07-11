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
	WritableRoots  []string
	EvidenceRefs   []string
}

type GoalWorkspaceRouterPortV0 interface {
	PrepareCodexGoalWorkspaceV0(context.Context, orquestaruntimecodexgoal.CodexGoalStartPacketV0) (GoalWorkspaceBindingV0, error)
	ResolveCodexGoalWorkspaceV0(context.Context, orquestaruntimecodexgoal.CodexGoalObservationRequestV0) (GoalWorkspaceBindingV0, error)
}

func (backend serverCodexAppServerGoalBackendV0) startCodexGoalInResolvedWorkspaceV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	binding, err := backend.WorkspaceRouter.PrepareCodexGoalWorkspaceV0(ctx, packet)
	if err != nil {
		return codexAppServerStartReceiptV0(packet, "", codexGoalWorkspaceUnavailableIssueV0), errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	var ok bool
	binding, ok = normalizeCodexGoalWorkspaceBindingV0(binding)
	if !ok {
		return codexAppServerStartReceiptV0(packet, "", codexGoalWorkspaceUnavailableIssueV0), errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	resolved := backend
	resolved.CWD = binding.ProjectWorkDir
	resolved.WritableRoots = binding.WritableRoots
	resolved.WorkspaceRouter = nil
	receipt, err := resolved.StartCodexGoalV0(ctx, packet)
	receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, binding.EvidenceRefs...))
	return receipt, err
}

func (backend serverCodexAppServerGoalBackendV0) observeCodexGoalInResolvedWorkspaceV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	binding, err := backend.WorkspaceRouter.ResolveCodexGoalWorkspaceV0(ctx, request)
	if err != nil {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, codexGoalWorkspaceUnavailableIssueV0), errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	var ok bool
	binding, ok = normalizeCodexGoalWorkspaceBindingV0(binding)
	if !ok {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, codexGoalWorkspaceUnavailableIssueV0), errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	resolved := backend
	resolved.CWD = binding.ProjectWorkDir
	resolved.WritableRoots = binding.WritableRoots
	resolved.WorkspaceRouter = nil
	receipt, err := resolved.ObserveCodexGoalV0(ctx, request)
	receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, binding.EvidenceRefs...))
	return receipt, err
}

func (backend serverCodexAppServerGoalBackendV0) fingerprintCodexGoalInResolvedWorkspaceV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalObservationFingerprintV0, bool, error) {
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
	binding, err := backend.WorkspaceRouter.ResolveCodexGoalWorkspaceV0(ctx, orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		SchemaVersion:   orquestaruntimecodexgoal.CodexGoalObservationRequestSchemaV0,
		GoalRef:         state.GoalRef,
		ExternalGoalRef: firstNonEmptyServerStackV0(state.ExternalGoalRef, state.LaunchReceipt.ExternalGoalRef),
	})
	if err != nil {
		return orquestagoal.GoalObservationFingerprintV0{}, true, errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	var ok bool
	binding, ok = normalizeCodexGoalWorkspaceBindingV0(binding)
	if !ok {
		return orquestagoal.GoalObservationFingerprintV0{}, true, errors.New(codexGoalWorkspaceUnavailableIssueV0)
	}
	resolved := backend
	resolved.CWD = binding.ProjectWorkDir
	resolved.WritableRoots = binding.WritableRoots
	resolved.WorkspaceRouter = nil
	return resolved.FingerprintGoalObservationV0(ctx, state)
}

func validCodexGoalWorkspaceBindingV0(binding GoalWorkspaceBindingV0) bool {
	_, ok := normalizeCodexGoalWorkspaceBindingV0(binding)
	return ok
}

func normalizeCodexGoalWorkspaceBindingV0(binding GoalWorkspaceBindingV0) (GoalWorkspaceBindingV0, bool) {
	projectWorkDir := strings.TrimSpace(binding.ProjectWorkDir)
	if projectWorkDir == "" || !filepath.IsAbs(projectWorkDir) {
		return GoalWorkspaceBindingV0{}, false
	}
	binding.ProjectWorkDir = filepath.Clean(projectWorkDir)
	normalizedRoots := make([]string, 0, len(binding.WritableRoots))
	for _, root := range binding.WritableRoots {
		root = strings.TrimSpace(root)
		if root == "" || !filepath.IsAbs(root) {
			return GoalWorkspaceBindingV0{}, false
		}
		normalizedRoots = append(normalizedRoots, filepath.Clean(root))
	}
	binding.WritableRoots = compactServerStackStringsV0(normalizedRoots)
	return binding, true
}
