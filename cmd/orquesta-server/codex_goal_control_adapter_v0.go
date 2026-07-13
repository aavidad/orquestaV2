package main

import (
	"context"
	"fmt"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeclaude "orquesta/modulos/orquesta-runtime-claude"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimegemini "orquesta/modulos/orquesta-runtime-gemini"
)

type serverCodexGoalControllerV0 interface {
	StopCodexGoalV0(context.Context, orquestaruntimecodexappserver.CodexGoalStopRequestV0) (orquestaruntimecodexappserver.CodexGoalStopResultV0, error)
}

type serverClaudeGoalControllerV0 interface {
	StopClaudeGoalV0(context.Context, orquestaruntimeclaude.ClaudeGoalStopRequestV0) (orquestaruntimeclaude.ClaudeGoalStopResultV0, error)
}

type serverGeminiGoalControllerV0 interface {
	StopGeminiGoalV0(context.Context, orquestaruntimegemini.GeminiGoalStopRequestV0) (orquestaruntimegemini.GeminiGoalStopResultV0, error)
}

type serverCodexGoalBackendControlV0 struct {
	Controller                 serverCodexGoalControllerV0
	WorkspaceAuthorityVerified bool
}

type serverClaudeGoalBackendControlV0 struct {
	Controller serverClaudeGoalControllerV0
}

type serverGeminiGoalBackendControlV0 struct {
	Controller serverGeminiGoalControllerV0
}

func serverGoalBackendControlFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestaappcodexstack.GoalBackendControlPortV0 {
	if backend.ClaudeControl != nil {
		return serverClaudeGoalBackendControlV0{Controller: backend.ClaudeControl}
	}
	if backend.GeminiControl != nil {
		return serverGeminiGoalBackendControlV0{Controller: backend.GeminiControl}
	}
	if backend.Controller != nil {
		return serverCodexGoalBackendControlV0{Controller: backend.Controller}
	}
	return nil
}

type serverAutoprogrammingGoalWorkspaceControlV0 struct {
	AppGoal             orquestaappcodexstack.GoalBackendControlPortV0
	AutoprogrammingGoal orquestaappcodexstack.GoalBackendControlPortV0
	WorkspaceLookup     serverGoalWorkspaceBindingLookupV0
}

func serverGoalBackendControlForAppAndAutoprogrammingV0(
	appGoal serverCodexGoalBackendV0,
	autoprogrammingGoal serverCodexGoalBackendV0,
) orquestaappcodexstack.GoalBackendControlPortV0 {
	appControl := serverGoalBackendControlFromBackendV0(appGoal)
	autoprogrammingControl := serverGoalBackendControlFromBackendV0(autoprogrammingGoal)
	if codexControl, ok := autoprogrammingControl.(serverCodexGoalBackendControlV0); ok {
		codexControl.WorkspaceAuthorityVerified = true
		autoprogrammingControl = codexControl
	}
	lookup := serverCodexGoalWorkspaceLookupFromBackendV0(autoprogrammingGoal)
	if lookup == nil {
		return appControl
	}
	return serverAutoprogrammingGoalWorkspaceControlV0{
		AppGoal:             appControl,
		AutoprogrammingGoal: autoprogrammingControl,
		WorkspaceLookup:     lookup,
	}
}

func (control serverAutoprogrammingGoalWorkspaceControlV0) ControlGoalBackendV0(
	ctx context.Context,
	request orquestaappcodexstack.GoalBackendControlRequestV0,
) (orquestaappcodexstack.GoalBackendControlResultV0, error) {
	found, err := control.WorkspaceLookup.HasGoalWorkspaceBindingV0(ctx, request.GoalRef)
	if err != nil {
		return orquestaappcodexstack.GoalBackendControlResultV0{}, err
	}
	delegate := control.AppGoal
	if found {
		if _, err := control.WorkspaceLookup.ResolveGoalWorkspaceV0(ctx, orquestagoal.GoalObservationRequestV0{
			GoalRef: request.GoalRef, ExternalGoalRef: request.ExternalGoalRef,
			IntentManifestRef: request.IntentManifestRef, IntentManifestSHA256: request.IntentManifestSHA256,
			WorkspaceAuthoritySchemaVersion: request.WorkspaceAuthoritySchemaVersion,
			WorkspaceRef:                    request.WorkspaceRef, ProviderRef: request.ProviderRef,
			RuntimeGenerationRef: request.RuntimeGenerationRef,
		}); err != nil {
			return orquestaappcodexstack.GoalBackendControlResultV0{}, err
		}
		delegate = control.AutoprogrammingGoal
	}
	if delegate == nil {
		return orquestaappcodexstack.GoalBackendControlResultV0{}, fmt.Errorf("autoprogramming_goal_workspace_control_unavailable")
	}
	return delegate.ControlGoalBackendV0(ctx, request)
}

func (control serverCodexGoalBackendControlV0) ControlGoalBackendV0(
	ctx context.Context,
	request orquestaappcodexstack.GoalBackendControlRequestV0,
) (orquestaappcodexstack.GoalBackendControlResultV0, error) {
	result, err := control.Controller.StopCodexGoalV0(ctx, orquestaruntimecodexappserver.CodexGoalStopRequestV0{
		GoalRef: request.GoalRef, ExternalGoalRef: request.ExternalGoalRef,
		IntentManifestRef: request.IntentManifestRef, IntentManifestSHA256: request.IntentManifestSHA256,
		WorkspaceAuthoritySchemaVersion: request.WorkspaceAuthoritySchemaVersion,
		WorkspaceRef:                    request.WorkspaceRef, ProviderRef: request.ProviderRef,
		RuntimeGenerationRef:       request.RuntimeGenerationRef,
		WorkspaceAuthorityVerified: control.WorkspaceAuthorityVerified,
		Action:                     request.Action, Reason: request.Reason, Forced: request.Forced, EvidenceRefs: request.EvidenceRefs,
	})
	return orquestaappcodexstack.GoalBackendControlResultV0{
		Status:          result.Status,
		GoalRef:         result.GoalRef,
		ExternalGoalRef: result.ExternalGoalRef,
		GoalStatusSet:   result.GoalStatusSet,
		BackendStopped:  result.BackendStopped,
		IssueCode:       result.IssueCode,
		EvidenceRefs:    result.EvidenceRefs,
	}, err
}

func (control serverGeminiGoalBackendControlV0) ControlGoalBackendV0(
	ctx context.Context,
	request orquestaappcodexstack.GoalBackendControlRequestV0,
) (orquestaappcodexstack.GoalBackendControlResultV0, error) {
	result, err := control.Controller.StopGeminiGoalV0(ctx, orquestaruntimegemini.GeminiGoalStopRequestV0{
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Action:          request.Action,
		Reason:          request.Reason,
		Forced:          request.Forced,
		EvidenceRefs:    request.EvidenceRefs,
	})
	return orquestaappcodexstack.GoalBackendControlResultV0{
		Status:          result.Status,
		GoalRef:         result.GoalRef,
		ExternalGoalRef: result.ExternalGoalRef,
		GoalStatusSet:   result.GoalStatusSet,
		BackendStopped:  result.BackendStopped,
		IssueCode:       result.IssueCode,
		EvidenceRefs:    result.EvidenceRefs,
	}, err
}

func (control serverClaudeGoalBackendControlV0) ControlGoalBackendV0(
	ctx context.Context,
	request orquestaappcodexstack.GoalBackendControlRequestV0,
) (orquestaappcodexstack.GoalBackendControlResultV0, error) {
	result, err := control.Controller.StopClaudeGoalV0(ctx, orquestaruntimeclaude.ClaudeGoalStopRequestV0{
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Action:          request.Action,
		Reason:          request.Reason,
		Forced:          request.Forced,
		EvidenceRefs:    request.EvidenceRefs,
	})
	return orquestaappcodexstack.GoalBackendControlResultV0{
		Status:          result.Status,
		GoalRef:         result.GoalRef,
		ExternalGoalRef: result.ExternalGoalRef,
		GoalStatusSet:   result.GoalStatusSet,
		BackendStopped:  result.BackendStopped,
		IssueCode:       result.IssueCode,
		EvidenceRefs:    result.EvidenceRefs,
	}, err
}
