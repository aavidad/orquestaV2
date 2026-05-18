package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type WorkflowTaskWaitFilterV0 struct {
	CohortRef     string
	WaveRef       string
	ParentTaskRef string
}

type WorkflowTaskWaitSnapshotV0 struct {
	Filter           WorkflowTaskWaitFilterV0
	TaskRefs         []string
	AgentRefs        []string
	PendingAgentRefs []string
}

func WorkflowTaskWaitAgentRefsV0(
	ctx context.Context,
	store WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	filter WorkflowTaskWaitFilterV0,
) ([]string, error) {
	snapshot, err := BuildWorkflowTaskWaitSnapshotV0(ctx, store, run, filter)
	if err != nil {
		return nil, err
	}
	return snapshot.AgentRefs, nil
}

func BuildWorkflowTaskWaitSnapshotV0(
	ctx context.Context,
	store WorkflowTaskStorePortV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	filter WorkflowTaskWaitFilterV0,
) (WorkflowTaskWaitSnapshotV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	filter = normalizeWorkflowTaskWaitFilterV0(filter)
	snapshot := WorkflowTaskWaitSnapshotV0{Filter: filter}
	if workflowTaskWaitFilterEmptyV0(filter) {
		return snapshot, nil
	}
	if store == nil {
		return WorkflowTaskWaitSnapshotV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"workflow_task_store",
			"workflow_task_store requerido",
		)
	}
	taskRefs := workflowTaskOpenRefsForWaitV0(run)
	if len(taskRefs) == 0 {
		return snapshot, nil
	}
	tasks, err := store.LoadWorkflowTasksV0(ctx, run.RunID, taskRefs)
	if err != nil {
		return WorkflowTaskWaitSnapshotV0{}, err
	}
	matchedTaskRefs := make([]string, 0, len(tasks))
	agentRefs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if !workflowTaskMatchesWaitFilterV0(task, filter) {
			continue
		}
		matchedTaskRefs = append(matchedTaskRefs, task.TaskID)
		agentRefs = append(agentRefs, WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	snapshot.TaskRefs = compactStringsV0(matchedTaskRefs)
	snapshot.AgentRefs = compactStringsV0(agentRefs)
	snapshot.PendingAgentRefs = workflowTaskWaitPendingAgentRefsV0(run, snapshot.AgentRefs)
	return snapshot, nil
}

func workflowTaskOpenRefsForWaitV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	closed := compactStringsV0(run.ClosedTasks)
	refs := make([]string, 0, len(run.Tasks))
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if stringInSetV0(taskRef, closed) {
			continue
		}
		refs = append(refs, taskRef)
	}
	return refs
}

func normalizeWorkflowTaskWaitFilterV0(
	filter WorkflowTaskWaitFilterV0,
) WorkflowTaskWaitFilterV0 {
	return WorkflowTaskWaitFilterV0{
		CohortRef:     strings.TrimSpace(filter.CohortRef),
		WaveRef:       strings.TrimSpace(filter.WaveRef),
		ParentTaskRef: strings.TrimSpace(filter.ParentTaskRef),
	}
}

func workflowTaskWaitFilterEmptyV0(filter WorkflowTaskWaitFilterV0) bool {
	return filter.CohortRef == "" && filter.WaveRef == "" && filter.ParentTaskRef == ""
}

func workflowTaskMatchesWaitFilterV0(
	task orquestacoreworkflow.WorkflowTaskV0,
	filter WorkflowTaskWaitFilterV0,
) bool {
	if filter.CohortRef != "" && strings.TrimSpace(task.CohortRef) != filter.CohortRef {
		return false
	}
	if filter.WaveRef != "" && strings.TrimSpace(task.WaveRef) != filter.WaveRef {
		return false
	}
	if filter.ParentTaskRef != "" && strings.TrimSpace(task.ParentTaskRef) != filter.ParentTaskRef {
		return false
	}
	return true
}

func workflowTaskWaitPendingAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRefs []string,
) []string {
	delivered := compactStringsV0(run.DeliveredAgents)
	failed := compactStringsV0(run.FailedAgents)
	lost := compactStringsV0(run.LostAgents)
	confirmedStopped := compactStringsV0(run.ConfirmedStoppedAgents)
	stopped := compactStringsV0(run.StoppedAgents)
	pending := make([]string, 0, len(agentRefs))
	for _, agentRef := range compactStringsV0(agentRefs) {
		if stringInSetV0(agentRef, delivered) ||
			stringInSetV0(agentRef, failed) ||
			stringInSetV0(agentRef, lost) ||
			stringInSetV0(agentRef, confirmedStopped) ||
			stringInSetV0(agentRef, stopped) {
			continue
		}
		pending = append(pending, agentRef)
	}
	return pending
}
