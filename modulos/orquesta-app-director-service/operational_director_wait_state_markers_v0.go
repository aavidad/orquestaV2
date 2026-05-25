package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func markOperationalDirectorWorkflowTaskWaitStateExpiredV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) error {
	if ports.WaitStateStore == nil || ports.WaitStateWriter == nil {
		return nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return nil
	}
	waitRefs := compactServiceRefsV0(activeStep.WaitRefs)
	if len(waitRefs) == 0 {
		return nil
	}
	waitState, err := ports.WaitStateStore.LoadWorkflowTaskWaitStateV0(ctx, request.RunRef, waitRefs[0])
	if err != nil {
		if appDirectorWorkflowTaskWaitStateNotFoundV0(err) {
			return nil
		}
		return err
	}
	if !appDirectorWorkflowTaskWaitStateMatchesPlanStepV0(request.RunRef, waitState, state, activeStep) {
		return nil
	}
	waitState.Status = orquestacionnucleoapp.WorkflowTaskWaitStateStatusExpiredV0
	waitState.PendingAgentRefs = nil
	waitState.Attempt = request.MaxExternalWaits + 1
	waitState.EvidenceRefs = compactServiceRefsV0(append(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0"))
	if strings.TrimSpace(request.OccurredAt) != "" {
		waitState.ObservedAt = request.OccurredAt
	}
	normalized, err := orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(waitState)
	if err != nil {
		return err
	}
	return ports.WaitStateWriter.SaveWorkflowTaskWaitStateV0(ctx, normalized)
}

func markOperationalDirectorWorkflowTaskWaitStateContinuedV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) error {
	if ports.WaitStateStore == nil || ports.WaitStateWriter == nil {
		return nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return nil
	}
	waitRefs := compactServiceRefsV0(activeStep.WaitRefs)
	if len(waitRefs) == 0 {
		return nil
	}
	waitState, err := ports.WaitStateStore.LoadWorkflowTaskWaitStateV0(ctx, request.RunRef, waitRefs[0])
	if err != nil {
		if appDirectorWorkflowTaskWaitStateNotFoundV0(err) {
			return nil
		}
		return err
	}
	if !appDirectorWorkflowTaskWaitStateMatchesPlanStepV0(request.RunRef, waitState, state, activeStep) {
		return nil
	}
	waitState.Status = orquestacionnucleoapp.WorkflowTaskWaitStateStatusContinuedV0
	waitState.PendingAgentRefs = nil
	waitState.EvidenceRefs = compactServiceRefsV0(append(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-continued-v0"))
	if strings.TrimSpace(request.OccurredAt) != "" {
		waitState.ObservedAt = request.OccurredAt
	}
	normalized, err := orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(waitState)
	if err != nil {
		return err
	}
	return ports.WaitStateWriter.SaveWorkflowTaskWaitStateV0(ctx, normalized)
}

func operationalDirectorPlanStateAfterWaitTerminalWithoutDeliveryV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	agentRefs := compactServiceRefsV0(append(activeStep.PendingAgentRefs, state.PendingAgentRefs...))
	if len(agentRefs) == 0 {
		agentRefs = compactServiceRefsV0(activeStep.AgentRefs)
	}
	if len(agentRefs) == 0 ||
		allServiceRefsInSetV0(agentRefs, run.DeliveredAgents) ||
		appDirectorRunHasPendingAgentRefsV0(run, agentRefs) {
		return state, false, nil
	}
	const reason = "wait-subagents-terminal-without-delivery"
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.PendingAgentRefs = nil
			nextStep.BlockerRefs = []string{reason}
			nextStep.Reason = reason
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, "evidence-ref-app-director-wait-subagents-terminal-without-delivery-v0"))
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{reason}
	state.Steps = nextSteps
	state.ClosureReason = reason
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-terminal-without-delivery-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanStateAfterWaitConsumedV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	pendingAgentRefs := compactServiceRefsV0(append(activeStep.PendingAgentRefs, state.PendingAgentRefs...))
	if len(pendingAgentRefs) == 0 || !allServiceRefsInSetV0(pendingAgentRefs, run.DeliveredAgents) {
		return state, false, nil
	}
	reviewStepID := ""
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		switch {
		case step.StepID == activeStep.StepID:
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			nextStep.PendingAgentRefs = nil
			nextStep.Reason = "wait-subagents-consumed"
		case step.Kind == orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0 &&
			(reviewStepID == "" || step.StepID == state.ActiveStepID):
			reviewStepID = step.StepID
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.TaskRefs = append([]string(nil), activeStep.TaskRefs...)
			nextStep.AgentRefs = append([]string(nil), activeStep.AgentRefs...)
			nextStep.WaveRef = activeStep.WaveRef
			nextStep.CohortRef = activeStep.CohortRef
			nextStep.ParentTaskRef = activeStep.ParentTaskRef
			nextStep.DeliveryRefs = nil
			nextStep.ReviewResultRefs = nil
			nextStep.AcceptedReviewRefs = nil
			nextStep.ReworkRequestRefs = nil
			nextStep.ReplanDecisionRefs = nil
			nextStep.BlockerRefs = nil
			nextStep.Reason = "wait-subagents-consumed"
		}
		nextSteps = append(nextSteps, nextStep)
	}
	if reviewStepID == "" {
		return state, false, nil
	}
	state.ActiveStepID = reviewStepID
	state.PendingAgentRefs = nil
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-consumed-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}
