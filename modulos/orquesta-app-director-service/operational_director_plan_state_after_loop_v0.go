package orquestaappdirectorservice

import (
	"context"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func updateOperationalDirectorPlanStateAfterLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) error {
	_, err := applyOperationalDirectorPlanStateAfterLoopV0(ctx, request, ports, loop)
	return err
}

func applyOperationalDirectorPlanStateAfterLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) (bool, error) {
	if ports.OperationalPlanStateStore == nil ||
		ports.OperationalPlanStateWriter == nil ||
		!operationalDirectorPlanStateLoopCanAdvanceV0(loop.Status) {
		return false, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	next := state
	changed := false
	afterWait, waitChanged, err := operationalDirectorPlanStateAfterWaitConsumedV0(request, next, loop.Run)
	if err != nil {
		return false, err
	}
	if waitChanged {
		next = afterWait
		changed = true
	}
	afterTerminalWait, terminalWaitChanged, err := operationalDirectorPlanStateAfterWaitTerminalWithoutDeliveryV0(request, next, loop.Run)
	if err != nil {
		return false, err
	}
	if terminalWaitChanged {
		next = afterTerminalWait
		changed = true
	}
	afterDeliveredTasks, deliveredTasksChanged, err := operationalDirectorPlanStateAfterWaitDeliveredTasksV0(request, next, loop.Run)
	if err != nil {
		return false, err
	}
	if deliveredTasksChanged {
		next = afterDeliveredTasks
		changed = true
	}
	// Avance incremental por sub-ola (opt-in). Solo se ejecuta si las barreras de
	// ola completa de arriba no avanzaron: actua sobre entregas parciales para que
	// las tareas independientes fluyan a review sin esperar a las hermanas.
	if request.StreamingSubwaveEnabled {
		afterSubset, subsetChanged, err := operationalDirectorPlanStateAfterWaitDeliveredSubsetV0(request, next, loop.Run)
		if err != nil {
			return false, err
		}
		if subsetChanged {
			next = afterSubset
			changed = true
		}
	}
	afterReview, reviewChanged, err := operationalDirectorPlanStateAfterReviewAcceptedV0(ctx, request, ports, next, loop)
	if err != nil {
		return false, err
	}
	if reviewChanged {
		next = afterReview
		changed = true
	}
	afterReviewReplan, reviewReplanChanged, err := operationalDirectorPlanStateAfterReviewReworkReplanV0(ctx, request, ports, next, loop)
	if err != nil {
		return false, err
	}
	if reviewReplanChanged {
		next = afterReviewReplan
		changed = true
	}
	afterTests, testsChanged, err := operationalDirectorPlanStateAfterRequiredTestsV0(ctx, request, ports, next, loop)
	if err != nil {
		return false, err
	}
	if testsChanged {
		next = afterTests
		changed = true
	}
	if !changed {
		return false, nil
	}
	if waitChanged || deliveredTasksChanged {
		if err := markOperationalDirectorWorkflowTaskWaitStateContinuedV0(ctx, request, ports, state); err != nil {
			return false, err
		}
	}
	return true, ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func operationalDirectorPlanStateLoopCanAdvanceV0(
	status orquestacionnucleoapp.ProgressiveLoopStatusV0,
) bool {
	switch status {
	case orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0:
		return true
	default:
		return false
	}
}

func applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	managedLoop orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) (bool, error) {
	if !managedProgressiveLoopExternalWaitExhaustedV0(request, managedLoop) ||
		ports.OperationalPlanStateStore == nil ||
		ports.OperationalPlanStateWriter == nil {
		return false, nil
	}
	planRef := continueOperationalDirectorPlanRefV0(request)
	if planRef == "" {
		return false, nil
	}
	state, err := ports.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, request.RunRef, planRef)
	if err != nil {
		return false, err
	}
	next, changed, err := operationalDirectorPlanStateWithWaitSubagentsExpiredV0(request, state)
	if err != nil || !changed {
		return false, err
	}
	if err := markOperationalDirectorWorkflowTaskWaitStateExpiredV0(ctx, request, ports, state); err != nil {
		return false, err
	}
	return true, ports.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, next)
}

func managedProgressiveLoopExternalWaitExhaustedV0(
	request ContinueAppDirectorRequestV0,
	managedLoop orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) bool {
	if request.MaxExternalWaits <= 0 {
		return false
	}
	if managedLoop.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		managedLoop.Final.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		return false
	}
	maxExternalWaits := request.MaxExternalWaits
	if maxExternalWaits < 0 {
		maxExternalWaits = 0
	}
	if len(managedLoop.Attempts) != maxExternalWaits+1 ||
		len(managedLoop.ExternalWaits) != maxExternalWaits {
		return false
	}
	for index, attempt := range managedLoop.Attempts {
		if attempt.AttemptNumber != index+1 ||
			attempt.Result.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
			return false
		}
	}
	for index, wait := range managedLoop.ExternalWaits {
		if wait.WaitNumber != index+1 || !wait.Continue {
			return false
		}
	}
	return len(managedLoop.Attempts) > 0
}

func operationalDirectorPlanStateBlockedTerminalWaitV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) bool {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	return ok &&
		state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 &&
		activeStep.Kind == orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 &&
		activeStep.Status == orquestadirectoroperativo.OperationalDirectorStepBlockedV0 &&
		activeStep.Reason == "wait-subagents-terminal-without-delivery"
}

func operationalDirectorPlanStateWithWaitSubagentsExpiredV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		return state, false, nil
	}
	const reason = "external-wait-exhausted"
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			nextStep.PendingAgentRefs = nil
			nextStep.BlockerRefs = []string{reason}
			nextStep.Reason = reason
			nextStep.EvidenceRefs = compactServiceRefsV0(append(nextStep.EvidenceRefs, "evidence-ref-app-director-wait-subagents-expired-v0"))
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{reason}
	state.Steps = nextSteps
	state.ClosureReason = reason
	state.EvidenceRefs = compactServiceRefsV0(append(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-expired-v0"))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}
