package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func autoprogrammingBridgeExistingRunResultV0(
	ctx context.Context,
	request AutoprogrammingBridgeRequestV0,
	result AutoprogrammingBridgeResultV0,
	existing orquestacoreworkflow.OrchestrationRunV0,
	expected orquestacoreworkflow.OrchestrationRunV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) (AutoprogrammingBridgeResultV0, error) {
	if err := autoprogrammingBridgeValidateExistingRunV0(existing, expected); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	expectedTaskRefs := compactStringsV0(expected.Tasks)
	storedTasks, err := ports.DirectorTaskStore.LoadWorkflowTasksV0(ctx, expected.RunID, expectedTaskRefs)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	if err := autoprogrammingBridgeValidateStoredTasksV0(storedTasks, result.Work.Tasks); err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Run = existing
	result.Tasks = append([]orquestacoreworkflow.WorkflowTaskV0(nil), storedTasks...)
	result.WaitAgentRefs = autoprogrammingBridgeWaitAgentRefsV0(result.Work.Tasks)
	continueRequest, err := autoprogrammingBridgeContinueRequestWithPlanStateV0(ctx, request, result, ports)
	if err != nil {
		return AutoprogrammingBridgeResultV0{}, err
	}
	result.Continue = continueRequest
	return result, nil
}

func autoprogrammingBridgeValidateStoredTasksV0(
	stored []orquestacoreworkflow.WorkflowTaskV0,
	expected []orquestacoreworkflow.WorkflowTaskV0,
) error {
	expectedByRef := map[string]orquestacoreworkflow.WorkflowTaskV0{}
	for _, task := range expected {
		expectedByRef[strings.TrimSpace(task.TaskID)] = task
	}
	seen := map[string]bool{}
	for _, task := range stored {
		taskRef := strings.TrimSpace(task.TaskID)
		expectedTask, ok := expectedByRef[taskRef]
		if !ok || strings.TrimSpace(task.RunID) != strings.TrimSpace(expectedTask.RunID) {
			return fmt.Errorf("autoprogramming workflow task existente incompatible: %s", task.TaskID)
		}
		if err := orquestacoreworkflow.ValidateWorkflowTaskV0(task); err != nil {
			return fmt.Errorf("autoprogramming workflow task existente invalida: %s", task.TaskID)
		}
		seen[taskRef] = true
	}
	for taskRef := range expectedByRef {
		if !seen[taskRef] {
			return fmt.Errorf("autoprogramming workflow task existente ausente: %s", taskRef)
		}
	}
	return nil
}
