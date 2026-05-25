package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func operationalDirectorPlanStateBlockedAtReplanOrCloseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	reason string,
	issues []orquestacionnucleoapp.ErrorV0,
) error {
	atReplanOrClose, err := operationalDirectorPlanStateIsAtReplanOrCloseV0(ctx, request, ports)
	if err != nil || !atReplanOrClose {
		return err
	}
	return operationalDirectorPlanStateBlockedAfterClosureV0(ctx, request, ports, reason, issues)
}

func operationalDirectorPlanStateIsAtReplanOrCloseV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return false, nil
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return false, nil
	}
	return step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0, nil
}

func operationalDirectorPlanStateBlocksClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (bool, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return false, nil
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 {
		return true, nil
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		return false, nil
	}
	if step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return false, nil
	}
	return true, nil
}

func operationalDirectorPlanStateRequiredTestEvidenceRefsForClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) ([]string, error) {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil {
		return nil, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return nil, err
	}
	refs := []string(nil)
	for _, step := range state.Steps {
		refs = append(refs, step.RequiredTestEvidenceRefs...)
	}
	return compactServiceRefsV0(refs), nil
}

func operationalDirectorPlanStateClosedAfterClosureV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	closureRequest orquestacionnucleoapp.OperationalDirectorClosureRequestV0,
) error {
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" || ports.OperationalPlanStateStore == nil || ports.OperationalPlanStateWriter == nil {
		return nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		if err := markOperationalDirectorWorkflowTaskWaitStatesClearedV0(ctx, request, ports, state); err != nil {
			return err
		}
		return nil
	}
	openState := state
	reason := "operational-closure-succeeded"
	closureRefs := operationalDirectorPlanStateClosureEvidenceRefsV0(closureRequest)
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == state.ActiveStepID ||
			(state.ActiveStepID == "" && step.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0) {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepClosedV0
			nextStep.RequiredTestEvidenceRefs = compactServiceRefsV0(append(
				nextStep.RequiredTestEvidenceRefs,
				closureRequest.RequiredTestEvidenceRefs...,
			))
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, closureRefs...))
			nextStep.BlockerRefs = nil
			nextStep.Reason = reason
			if state.ActiveStepID == "" {
				state.ActiveStepID = step.StepID
			}
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.EvidenceRefs = compactServiceRefsV0(append(
		append(state.EvidenceRefs, closureRefs...),
		"evidence-ref-app-director-operational-plan-state-closure-succeeded-v0",
	))
	state.ClosureReason = reason
	state.Steps = nextSteps
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return err
	}
	if err := markOperationalDirectorWorkflowTaskWaitStatesClearedV0(ctx, request, ports, openState); err != nil {
		return err
	}
	return ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}
