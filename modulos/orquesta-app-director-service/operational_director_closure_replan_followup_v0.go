package orquestaappdirectorservice

import (
	"context"
	"errors"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func operationalDirectorClosureReplannableBlockerRefsV0(refs []string) []string {
	replannable := make([]string, 0, len(refs))
	closureInsufficient := false
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if operationalDirectorClosureIssueReplannableV0(ref) {
			replannable = append(replannable, ref)
			if ref != "required_test_evidence_refs" && ref != "operational_closure_insufficient" {
				closureInsufficient = true
			}
		}
	}
	if closureInsufficient {
		replannable = append(replannable, "operational_closure_insufficient")
	}
	return compactServiceRefsV0(replannable)
}

func continueOperationalDirectorPlanStateAfterBlockedClosurePrerequisiteV0(
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if ports.OperationalPlanStateWriter == nil {
		return state, false, nil
	}
	activeStep, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		activeStep.Kind != orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0 ||
		activeStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 {
		return state, false, nil
	}
	reason := strings.TrimSpace(state.ClosureReason)
	if reason == "" {
		reason = strings.TrimSpace(activeStep.Reason)
	}
	switch reason {
	case "operational-closure-source-unavailable":
		if ports.OperationalClosureSource == nil {
			return state, false, nil
		}
	case "operational-closure-task-store-unavailable":
		if ports.DirectorTaskStore == nil {
			return state, false, nil
		}
	case "operational-closure-outbox-pending":
		// The next loop is the source of truth for pending outbox. Reopen once and
		// let maybeCloseOperationalDirectorV0 block again if the outbox is still pending.
	default:
		return state, false, nil
	}
	return operationalDirectorPlanStateWithClosurePrerequisiteReadyV0(request, state, activeStep)
}

func operationalDirectorPlanStateWithClosurePrerequisiteReadyV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	nextSteps := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(state.Steps))
	for _, step := range state.Steps {
		nextStep := step
		if step.StepID == activeStep.StepID {
			nextStep.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			nextStep.BlockerRefs = nil
			nextStep.Reason = "operational-closure-prerequisite-ready"
			nextStep.EvidenceRefs = compactServiceRefsV0(append(
				nextStep.EvidenceRefs,
				"evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0",
			))
		}
		nextSteps = append(nextSteps, nextStep)
	}
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = activeStep.StepID
	state.PendingAgentRefs = nil
	state.BlockerRefs = nil
	state.ClosureReason = ""
	state.Steps = nextSteps
	state.EvidenceRefs = compactServiceRefsV0(append(
		state.EvidenceRefs,
		"evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0",
	))
	state.UpdatedAt = request.OccurredAt
	next, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(state)
	if err != nil {
		return state, false, err
	}
	return next, true, nil
}

func operationalDirectorPlanReplanFollowupAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	match operationalDirectorPlanReviewReworkReplanMatchV0,
) []string {
	if match.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 &&
		match.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0 {
		return nil
	}
	agentRefs := []string(nil)
	for _, followupRef := range compactServiceRefsV0(match.FollowupRefs) {
		if startAppDirectorStringInSetV0(run.Agents, followupRef) &&
			!startAppDirectorStringInSetV0(activeStep.AgentRefs, followupRef) {
			agentRefs = append(agentRefs, followupRef)
		}
	}
	return compactServiceRefsV0(agentRefs)
}

func operationalDirectorPlanReplanFollowupTasksV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	match operationalDirectorPlanReviewReworkReplanMatchV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, bool, error) {
	if match.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 ||
		len(match.FollowupRefs) == 0 ||
		ports.DirectorTaskStore == nil {
		return nil, false, nil
	}
	followupRefs := compactServiceRefsV0(match.FollowupRefs)
	for _, followupRef := range followupRefs {
		if !startAppDirectorStringInSetV0(run.Tasks, followupRef) {
			return nil, false, nil
		}
	}
	tasks, err := ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, request.RunRef, followupRefs)
	if err != nil {
		if operationalDirectorPlanMissingWorkflowTasksV0(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(tasks) != len(followupRefs) {
		return nil, false, nil
	}
	if _, _, _, ok := operationalDirectorPlanFollowupScopeV0(tasks); !ok {
		return nil, false, nil
	}
	return tasks, true, nil
}

func operationalDirectorPlanMissingWorkflowTasksV0(err error) bool {
	var issue orquestacionnucleoapp.ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "workflow_tasks"
}

func operationalDirectorPlanFollowupScopeV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (string, string, string, bool) {
	if len(tasks) == 0 {
		return "", "", "", false
	}
	waveRef := strings.TrimSpace(tasks[0].WaveRef)
	cohortRef := strings.TrimSpace(tasks[0].CohortRef)
	parentTaskRef := strings.TrimSpace(tasks[0].ParentTaskRef)
	for _, task := range tasks {
		if strings.TrimSpace(task.WaveRef) != waveRef ||
			strings.TrimSpace(task.CohortRef) != cohortRef ||
			strings.TrimSpace(task.ParentTaskRef) != parentTaskRef {
			return "", "", "", false
		}
	}
	return waveRef, cohortRef, parentTaskRef, true
}
