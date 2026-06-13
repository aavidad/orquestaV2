package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func persistOperationalDirectorPlanActiveWaitStateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) error {
	if ports.WaitStateWriter == nil || ports.RunStore == nil {
		return nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return err
	}
	filter := appDirectorWaitFilterV0{
		CohortRef:     firstServiceRefV0(activeStep.CohortRef, state.ActiveCohortRef),
		WaveRef:       firstServiceRefV0(activeStep.WaveRef, state.ActiveWaveRef),
		ParentTaskRef: firstServiceRefV0(activeStep.ParentTaskRef, state.ActiveParentTaskRef),
	}
	explicitAgentRefs := compactServiceRefsV0(append(
		append(append([]string(nil), activeStep.AgentRefs...), activeStep.PendingAgentRefs...),
		state.PendingAgentRefs...,
	))
	snapshot := orquestacionnucleoapp.WorkflowTaskWaitSnapshotV0{}
	if !appDirectorWaitFilterEmptyV0(filter) && ports.DirectorTaskStore != nil {
		snapshot, err = orquestacionnucleoapp.BuildWorkflowTaskWaitSnapshotV0(
			ctx,
			ports.DirectorTaskStore,
			run,
			orquestacionnucleoapp.WorkflowTaskWaitFilterV0{
				CohortRef:     filter.CohortRef,
				WaveRef:       filter.WaveRef,
				ParentTaskRef: filter.ParentTaskRef,
			},
		)
		if err != nil {
			return err
		}
	}
	snapshot.AgentRefs = compactServiceRefsV0(append(snapshot.AgentRefs, explicitAgentRefs...))
	snapshot.PendingAgentRefs = compactServiceRefsV0(append(
		snapshot.PendingAgentRefs,
		appDirectorExplicitPendingAgentRefsV0(run, explicitAgentRefs)...,
	))
	if len(snapshot.AgentRefs) == 0 && appDirectorWaitFilterEmptyV0(filter) {
		return nil
	}
	waitRef := firstServiceRefV0(activeStep.WaitRefs...)
	stateToSave, err := appDirectorWorkflowTaskWaitStateV0(
		request.RunRef,
		filter,
		snapshot,
		appDirectorWaitStateMetaV0{
			OccurredAt:       request.OccurredAt,
			CorrelationID:    request.CorrelationID,
			EvidenceRefs:     []string{"evidence-ref-app-director-active-wait-state-v0"},
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	if err != nil {
		return err
	}
	if waitRef != "" {
		stateToSave.WaitRef = waitRef
		stateToSave, err = orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(stateToSave)
		if err != nil {
			return err
		}
	}
	if ports.WaitStateStore != nil {
		existing, err := ports.WaitStateStore.LoadWorkflowTaskWaitStateV0(ctx, request.RunRef, stateToSave.WaitRef)
		if err != nil {
			if !appDirectorWorkflowTaskWaitStateNotFoundV0(err) {
				return err
			}
		} else if existing.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0 {
			return nil
		}
	}
	return ports.WaitStateWriter.SaveWorkflowTaskWaitStateV0(ctx, stateToSave)
}

func operationalDirectorPlanStateActiveWaitAgentRefsV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) ([]string, error) {
	refs, err := operationalDirectorPlanStateActiveWaitScopeAgentRefsV0(ctx, request, ports)
	if err != nil {
		return nil, err
	}
	if ports.RunStore != nil {
		run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return nil, err
		}
		refs = appDirectorRequestedPendingAgentRefsV0(run, refs)
	}
	return refs, nil
}

func operationalDirectorPlanStateActiveWaitScopeAgentRefsV0(
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
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return nil, nil
	}
	return operationalDirectorPlanStateWaitStepScopeAgentRefsV0(state, activeStep), nil
}

func operationalDirectorPlanStateWaitStepScopeAgentRefsV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) []string {
	refs := compactServiceRefsV0(append(step.PendingAgentRefs, state.PendingAgentRefs...))
	if len(refs) == 0 {
		refs = compactServiceRefsV0(step.AgentRefs)
	}
	return refs
}

func operationalDirectorScopedQuiescentLoopV0(
	request ContinueAppDirectorRequestV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	if continueOperationalDirectorPlanRefV0(request) == "" ||
		loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		loop.PendingOutboxCount > 0 ||
		len(loopRequest.WaitAgentRefs) == 0 ||
		appDirectorRunHasPendingAgentRefsV0(loop.Run, loopRequest.WaitAgentRefs) {
		return loop
	}
	loop.Status = orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0
	return loop
}

func appDirectorRunHasPendingAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) bool {
	requested := appDirectorRequestedAgentRefsV0(run)
	for _, agentRef := range compactServiceRefsV0(agentRefs) {
		if len(requested) > 0 && !requested[agentRef] && !appDirectorAgentTerminalInRunV0(run, agentRef) {
			continue
		}
		if startAppDirectorStringInSetV0(run.DeliveredAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.FailedAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.LostAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.ConfirmedStoppedAgents, agentRef) ||
			startAppDirectorStringInSetV0(run.StoppedAgents, agentRef) {
			continue
		}
		return true
	}
	return false
}

func firstServiceRefV0(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
