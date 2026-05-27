package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func (materializer OperationalDirectorPlanMaterializerV0) resolveOperationalDirectorMaterializedTaskRefsV0(
	ctx context.Context,
	request OperationalDirectorPlanMaterializeRequestV0,
	items []orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	itemTaskRefs map[string]string,
	itemWaveRefs map[string]string,
) (map[string]string, error) {
	taskStore, ok := materializer.TaskWriter.(WorkflowTaskStorePortV0)
	if !ok || len(items) == 0 {
		return itemTaskRefs, nil
	}
	resolved := map[string]string{}
	for key, value := range itemTaskRefs {
		resolved[key] = value
	}
	for pass := 0; pass < 2; pass++ {
		changed := false
		for _, item := range items {
			task, err := operationalDirectorWorkflowTaskFromItemV0(request, item, resolved, itemWaveRefs)
			if err != nil {
				return nil, err
			}
			conflict, err := operationalDirectorWorkflowTaskStoreConflictV0(ctx, taskStore, task)
			if err != nil {
				return nil, err
			}
			if !conflict {
				continue
			}
			nextRef := operationalDirectorWorkflowTaskConflictRefV0(task)
			if resolved[item.ItemID] == nextRef {
				continue
			}
			resolved[item.ItemID] = nextRef
			changed = true
		}
		if !changed {
			return resolved, nil
		}
	}
	return resolved, nil
}

func operationalDirectorWorkflowTaskStoreConflictV0(
	ctx context.Context,
	store WorkflowTaskStorePortV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (bool, error) {
	existing, err := store.LoadWorkflowTasksV0(ctx, task.RunID, []string{task.TaskID})
	if err != nil {
		if operationalDirectorWorkflowTaskMissingV0(err) {
			return false, nil
		}
		return false, err
	}
	if len(existing) == 0 {
		return false, nil
	}
	return !reflect.DeepEqual(existing[0], task), nil
}

func operationalDirectorWorkflowTaskMissingV0(err error) bool {
	var issue ErrorV0
	return errors.As(err, &issue) &&
		issue.Code == ErrNucleoOrquestacionStoreV0 &&
		issue.Field == "workflow_tasks" &&
		strings.Contains(issue.Message, "microtarea no encontrada")
}

func operationalDirectorWorkflowTaskConflictRefV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) string {
	data, err := json.Marshal(task)
	if err != nil {
		data = []byte(task.TaskID)
	}
	return strings.TrimSpace(task.TaskID) + "-contract-" +
		deterministicRefDigestPrefixV0("workflow_task_conflict", 32, string(data))
}
