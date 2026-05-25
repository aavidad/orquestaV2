package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func markOperationalDirectorWorkflowTaskWaitStatesClearedV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) error {
	if ports.WaitStateStore == nil || ports.WaitStateWriter == nil {
		return nil
	}
	for _, step := range state.Steps {
		waitRefs := compactServiceRefsV0(step.WaitRefs)
		for _, waitRef := range waitRefs {
			waitState, err := ports.WaitStateStore.LoadWorkflowTaskWaitStateV0(ctx, request.RunRef, waitRef)
			if err != nil {
				if appDirectorWorkflowTaskWaitStateNotFoundV0(err) {
					continue
				}
				return err
			}
			if !appDirectorWorkflowTaskWaitStateMatchesPlanStepV0(request.RunRef, waitState, state, step) {
				continue
			}
			waitState.Status = orquestacionnucleoapp.WorkflowTaskWaitStateStatusClearedV0
			waitState.PendingAgentRefs = nil
			waitState.EvidenceRefs = compactServiceRefsV0(append(
				waitState.EvidenceRefs,
				"evidence-ref-app-director-wait-state-cleared-v0",
			))
			if strings.TrimSpace(request.OccurredAt) != "" {
				waitState.ObservedAt = request.OccurredAt
			}
			normalized, err := orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(waitState)
			if err != nil {
				return err
			}
			if err := ports.WaitStateWriter.SaveWorkflowTaskWaitStateV0(ctx, normalized); err != nil {
				return err
			}
		}
	}
	return nil
}

func operationalDirectorPlanStateBlockedAfterClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
	extraBlockerRefs ...string,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "operational-closure-blocked"
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return nil
	}
	blockerRefs := operationalDirectorPlanStateClosureBlockerRefsV0(reason, issues)
	blockerRefs = compactServiceRefsV0(append(blockerRefs, extraBlockerRefs...))
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == state.ActiveStepID ||
			(state.ActiveStepID == "" && step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0) {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.BlockerRefs = compactServiceRefsV0(append(nextStep.BlockerRefs, blockerRefs...))
			nextStep.Reason = reason
			if state.ActiveStepID == "" {
				state.ActiveStepID = step.StepID
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.BlockerRefs = compactServiceRefsV0(append(state.BlockerRefs, blockerRefs...))
	state.EvidenceRefs = compactServiceRefsV0(append(
		append(state.EvidenceRefs, extraBlockerRefs...),
		"evidence-ref-app-director-operational-plan-state-closure-blocked-v0",
	))
	state.ClosureReason = reason
	state.Steps = nextSteps
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateClosureEvidenceRefsV0(
	request orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) []string {
	refs := []string{
		request.DeliveryRef,
		request.AcceptedReviewRef,
		request.ValidationRef,
		request.ClosureRef,
	}
	refs = append(refs, request.RequiredTestEvidenceRefs...)
	refs = append(refs, request.EvidenceRefs...)
	return compactServiceRefsV0(refs)
}

func operationalDirectorPlanStateClosureBlockerRefsV0(
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) []string {
	refs := []string{reason}
	for _, issue := range issues {
		if field := strings.TrimSpace(issue.Field); field != "" {
			refs = append(refs, field)
		}
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, code)
		}
	}
	return compactServiceRefsV0(refs)
}
