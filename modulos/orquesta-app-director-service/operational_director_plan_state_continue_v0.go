package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func continueRequestWithOperationalDirectorWaitV0(
	request ContinueAppDirectorRequestV0,
	materialized continueOperationalDirectorMaterializedV0,
) ContinueAppDirectorRequestV0 {
	if len(materialized.Tasks) == 0 {
		return request
	}
	if strings.TrimSpace(request.WaitWaveRef) == "" && strings.TrimSpace(materialized.WaveRef) != "" {
		request.WaitWaveRef = materialized.WaveRef
	}
	if strings.TrimSpace(request.WaitCohortRef) == "" && strings.TrimSpace(materialized.CohortRef) != "" {
		request.WaitCohortRef = materialized.CohortRef
	}
	return request
}

func continueRequestWithOperationalDirectorPlanStateV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (ContinueAppDirectorRequestV0, error) {
	hasWaitScope := continueRequestHasWaitScopeV0(request)
	hasPlanRef := continueOperationalDirectorPlanRefV0(request) != ""
	if hasWaitScope && (!hasPlanRef || ports.OperationalPlanStateStore == nil) {
		return request, nil
	}
	ensured, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(ctx, request, ports)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	request = ensured
	explicitPlanRef := strings.TrimSpace(request.OperationalDirectorPlanRef)
	if explicitPlanRef == "" && ports.OperationalPlanStateStore == nil {
		return request, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return request, nil
	}
	if ports.OperationalPlanStateStore == nil {
		return ContinueAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "ports.operational_plan_state_store"}
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return request, nil
	}
	reopened := false
	state, reopenedRequiredTests, err := continueOperationalDirectorPlanStateAfterBlockedRequiredTestsReplanV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedRequiredTests
	state, recoveredRequiredTests, err := continueOperationalDirectorPlanStateAfterBlockedRequiredTestsEvidenceV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || recoveredRequiredTests
	state, reopenedReviewReplan, err := continueOperationalDirectorPlanStateAfterReviewReworkReplanFollowupsV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedReviewReplan
	state, reopenedLateWaitDelivery, err := continueOperationalDirectorPlanStateAfterBlockedWaitExpiredDeliveryV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedLateWaitDelivery
	state, reopenedDeliveredTasks, err := continueOperationalDirectorPlanStateAfterBlockedWaitDeliveredTasksV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedDeliveredTasks
	state, reopenedClosurePrerequisite, err := continueOperationalDirectorPlanStateAfterBlockedClosurePrerequisiteV0(request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedClosurePrerequisite
	state, reopenedClosureNoProgress, err := continueOperationalDirectorPlanStateAfterClosureOpenTasksNoProgressV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedClosureNoProgress
	state, refreshedClosureScope, err := continueOperationalDirectorPlanStateRefreshStaleClosureScopeV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || refreshedClosureScope
	state, reopenedClosureReplan, err := continueOperationalDirectorPlanStateAfterBlockedClosureIssuesReplanV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	reopened = reopened || reopenedClosureReplan
	if reopened {
		if ports.OperationalPlanStateWriter == nil {
			return ContinueAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "ports.operational_plan_state_writer"}
		}
		if err := ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, state); err != nil {
			return ContinueAppDirectorRequestV0{}, err
		}
	}
	stateScopedRequest := request
	if hasWaitScope {
		stateScopedRequest = continueRequestWithoutWaitScopeV0(stateScopedRequest)
	}
	next, applied, err := continueRequestWithLoadedOperationalDirectorPlanStateV0(ctx, stateScopedRequest, state, ports)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	if !applied && hasWaitScope && explicitPlanRef == "" {
		return request, nil
	}
	canContinueWithoutWaitScope, err := operationalDirectorPlanStateCanContinueWithoutWaitScopeV0(ctx, request, ports, state)
	if err != nil {
		return ContinueAppDirectorRequestV0{}, err
	}
	if explicitPlanRef != "" &&
		(!applied || (!continueRequestHasWaitScopeV0(next) && !canContinueWithoutWaitScope)) {
		canProgress, err := continueOperationalDirectorPlanStateCanProgressBlockedRequiredTestsReplanV0(ctx, request, ports, state)
		if err != nil {
			return ContinueAppDirectorRequestV0{}, err
		}
		canProgressClosureIssues, err := continueOperationalDirectorPlanStateCanProgressBlockedClosureIssuesReplanV0(ctx, request, ports, state)
		if err != nil {
			return ContinueAppDirectorRequestV0{}, err
		}
		canProgress = canProgress || canProgressClosureIssues
		if !canProgress && operationalDirectorPlanStateBlockedRequiredTestsEvidenceMissingV0(state) {
			return request, nil
		}
		if !canProgress && operationalDirectorPlanStateBlockedTerminalWaitV0(state) {
			return request, nil
		}
		if !canProgress {
			return ContinueAppDirectorRequestV0{}, AppDirectorServiceIssueV0{Field: "operational_director_plan_state.active_step"}
		}
	}
	return next, nil
}

func operationalDirectorPlanStateReplanOrCloseRunningV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) bool {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	return ok &&
		state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 &&
		activeStep.Kind == orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 &&
		activeStep.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0
}

func operationalDirectorPlanStateCanContinueWithoutWaitScopeV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (bool, error) {
	if operationalDirectorPlanStateReplanOrCloseRunningV0(state) {
		return true, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return false, nil
	}
	agentRefs := compactServiceRefsV0(append(activeStep.PendingAgentRefs, state.PendingAgentRefs...))
	if len(agentRefs) == 0 {
		agentRefs = compactServiceRefsV0(activeStep.AgentRefs)
	}
	if len(agentRefs) == 0 || ports.RunStore == nil {
		return false, nil
	}
	run, err := ports.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return false, err
	}
	return len(appDirectorRequestedPendingAgentRefsV0(run, agentRefs)) == 0, nil
}

func operationalDirectorPlanStateBlockedRequiredTestsEvidenceMissingV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) bool {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	return ok &&
		state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 &&
		activeStep.Kind == orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0 &&
		activeStep.Status == orquestadirectoroperativo.OperationalDirectorStepBlockedV0 &&
		activeStep.Reason == "required-tests-evidence-missing"
}
