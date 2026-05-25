package orquestaappdirectorservice

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func mergeDirectorDecisionOperationalPlanStateV0(
	request ContinueAppDirectorRequestV0,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) (orquestacionnucleoapp.OperationalDirectorPlanStateV0, bool, error) {
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return state, false, nil
	}
	newTasks := directorDecisionPlanStateNewTasksV0(state, tasks)
	if len(newTasks) == 0 {
		return state, false, nil
	}
	scope := directorDecisionPlanStateTaskScopeV0(newTasks)
	waitStepID := operationalDirectorPlanStateStepIDByKindV0(
		state,
		orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
	)
	if waitStepID == "" {
		return state, false, AppDirectorServiceIssueV0{Field: "operational_director_plan_state.wait_subagents"}
	}
	taskRefs := continueOperationalDirectorTaskRefsV0(newTasks)
	agentRefs := continueOperationalDirectorAgentRefsV0(newTasks)
	waitRef := appDirectorWaitRefV0(state.RunRef, scope, request.CorrelationID)

	next := state
	next.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	next.ActiveStepID = waitStepID
	next.ActiveWaveRef = scope.WaveRef
	next.ActiveCohortRef = scope.CohortRef
	next.ActiveParentTaskRef = scope.ParentTaskRef
	next.PendingAgentRefs = append([]string(nil), agentRefs...)
	next.BlockerRefs = nil
	next.ClosureReason = ""
	next.RequiredTestRefs = compactServiceRefsV0(append(next.RequiredTestRefs, directorDecisionRequiredTestsV0(newTasks)...))
	next.EvidenceRefs = compactServiceRefsV0(append(
		next.EvidenceRefs,
		"evidence-ref-app-director-operational-plan-state-director-decision-merge-v0",
	))
	next.UpdatedAt = request.OccurredAt
	next.Steps = mergeDirectorDecisionPlanStateStepsV0(next.Steps, waitStepID, scope, taskRefs, agentRefs, waitRef)
	normalized, err := orquestacionnucleoapp.NewOperationalDirectorPlanStateV0(next)
	if err != nil {
		return state, false, err
	}
	return normalized, true, nil
}

func directorDecisionPlanStateNewTasksV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	known := directorDecisionPlanStateTaskRefsV0(state)
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		if startAppDirectorStringInSetV0(known, task.TaskID) {
			continue
		}
		out = append(out, task)
	}
	return out
}

func directorDecisionPlanStateTaskRefsV0(
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
) []string {
	refs := make([]string, 0)
	for _, step := range state.Steps {
		refs = append(refs, step.TaskRefs...)
	}
	return compactServiceRefsV0(refs)
}

func directorDecisionPlanStateTaskScopeV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) appDirectorWaitFilterV0 {
	return appDirectorWaitFilterV0{
		WaveRef:       directorDecisionPlanStateCommonRefV0(tasks, func(task orquestacoreworkflow.WorkflowTaskV0) string { return task.WaveRef }),
		CohortRef:     directorDecisionPlanStateCommonRefV0(tasks, func(task orquestacoreworkflow.WorkflowTaskV0) string { return task.CohortRef }),
		ParentTaskRef: directorDecisionPlanStateCommonRefV0(tasks, func(task orquestacoreworkflow.WorkflowTaskV0) string { return task.ParentTaskRef }),
	}
}

func directorDecisionPlanStateCommonRefV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
	pick func(orquestacoreworkflow.WorkflowTaskV0) string,
) string {
	ref := ""
	for _, task := range tasks {
		value := strings.TrimSpace(pick(task))
		if value == "" {
			continue
		}
		if ref == "" {
			ref = value
			continue
		}
		if ref != value {
			return ""
		}
	}
	return ref
}

func mergeDirectorDecisionPlanStateStepsV0(
	steps []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	waitStepID string,
	scope appDirectorWaitFilterV0,
	taskRefs []string,
	agentRefs []string,
	waitRef string,
) []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	next := make([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0, 0, len(steps))
	for _, step := range steps {
		merged := step
		switch step.Kind {
		case orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0:
			merged.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			merged.TaskRefs = compactServiceRefsV0(append(merged.TaskRefs, taskRefs...))
			merged.AgentRefs = compactServiceRefsV0(append(merged.AgentRefs, agentRefs...))
		case orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0:
			if step.StepID == waitStepID {
				merged = directorDecisionPlanStateMergedWaitStepV0(step, scope, taskRefs, agentRefs, waitRef)
			}
		}
		next = append(next, merged)
	}
	return next
}

func directorDecisionPlanStateMergedWaitStepV0(
	step orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	scope appDirectorWaitFilterV0,
	taskRefs []string,
	agentRefs []string,
	waitRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
	step.WaveRef = scope.WaveRef
	step.CohortRef = scope.CohortRef
	step.ParentTaskRef = scope.ParentTaskRef
	step.TaskRefs = append([]string(nil), taskRefs...)
	step.WaitRefs = []string{waitRef}
	step.AgentRefs = append([]string(nil), agentRefs...)
	step.PendingAgentRefs = append([]string(nil), agentRefs...)
	step.BlockerRefs = []string{"wait-subagents"}
	step.Reason = "director-decision-plan-state-merge"
	step.EvidenceRefs = compactServiceRefsV0(append(
		step.EvidenceRefs,
		"evidence-ref-app-director-wait-subagents-director-decision-merge-v0",
	))
	return step
}
