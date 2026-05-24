package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func checkReplayStepScopeSnapshotV0(
	step OperationalDirectorPlanStepStateV0,
	taskByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
	waitByRef map[string]WorkflowTaskWaitStateV0,
) []ErrorV0 {
	issues := make([]ErrorV0, 0)
	for _, taskRef := range step.TaskRefs {
		task, ok := taskByRef[taskRef]
		if !ok {
			continue
		}
		issues = append(issues, checkReplayPlanStateKeepsRefV0("plan_state.wave_ref", step.WaveRef, task.WaveRef)...)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("plan_state.cohort_ref", step.CohortRef, task.CohortRef)...)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("plan_state.parent_task_ref", step.ParentTaskRef, task.ParentTaskRef)...)
	}
	for _, waitRef := range step.WaitRefs {
		waitState, ok := waitByRef[waitRef]
		if !ok {
			continue
		}
		issues = append(issues, checkReplayPlanStateKeepsRefV0("plan_state.wave_ref", step.WaveRef, waitState.WaveRef)...)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("plan_state.cohort_ref", step.CohortRef, waitState.CohortRef)...)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("plan_state.parent_task_ref", step.ParentTaskRef, waitState.ParentTaskRef)...)
		issues = append(issues, checkReplaySetSnapshotV0("wait.task_refs", step.TaskRefs, waitState.TaskRefs)...)
		issues = append(issues, checkReplaySetSnapshotV0("wait.agent_refs", step.AgentRefs, waitState.AgentRefs)...)
		issues = append(issues, checkReplaySetSnapshotV0("wait.pending_agent_refs", step.PendingAgentRefs, waitState.PendingAgentRefs)...)
	}
	return issues
}

func checkReplayActiveScopeSnapshotV0(state OperationalDirectorPlanStateV0) []ErrorV0 {
	if strings.TrimSpace(state.ActiveStepID) == "" {
		return nil
	}
	for _, step := range state.Steps {
		if step.StepID != state.ActiveStepID {
			continue
		}
		issues := make([]ErrorV0, 0)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("active_wave_ref", state.ActiveWaveRef, step.WaveRef)...)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("active_cohort_ref", state.ActiveCohortRef, step.CohortRef)...)
		issues = append(issues, checkReplayPlanStateKeepsRefV0("active_parent_task_ref", state.ActiveParentTaskRef, step.ParentTaskRef)...)
		return issues
	}
	return nil
}

func checkReplayPlanStateKeepsRefV0(field string, planValue string, storedValue string) []ErrorV0 {
	if strings.TrimSpace(planValue) != "" || strings.TrimSpace(storedValue) == "" {
		return nil
	}
	return []ErrorV0{errorV0(ErrNucleoOrquestacionStoreV0, field, "metadata viva ausente en plan state")}
}

func checkReplaySetSnapshotV0(field string, planValues []string, storedValues []string) []ErrorV0 {
	planSet := compactStringsV0(planValues)
	storedSet := compactStringsV0(storedValues)
	if len(planSet) == 0 && len(storedSet) == 0 {
		return nil
	}
	if len(planSet) != len(storedSet) {
		return []ErrorV0{errorV0(ErrNucleoOrquestacionStoreV0, field, "metadata viva no coincide con plan state")}
	}
	for _, value := range planSet {
		if !stringInSetV0(value, storedSet) {
			return []ErrorV0{errorV0(ErrNucleoOrquestacionStoreV0, field, "metadata viva no coincide con plan state")}
		}
	}
	return nil
}
