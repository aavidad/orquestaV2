package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const autoprogrammingBridgeOperationalTaskSourceRefV0 = "operational_director.task_source:autoprogramming"

func normalizeAutoprogrammingBridgeRequestV0(
	request AutoprogrammingBridgeRequestV0,
) AutoprogrammingBridgeRequestV0 {
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	if request.CorrelationID == "" {
		request.CorrelationID = "corr-" + strings.TrimSpace(request.Request.RequestRef)
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-codex-stack-autoprogramming"
	}
	return request
}

func validateAutoprogrammingBridgePortsV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) error {
	if ports.RunStore == nil {
		return fmt.Errorf("ports.run_store requerido")
	}
	if ports.DirectorTaskStore == nil {
		return fmt.Errorf("ports.director_task_store requerido")
	}
	return nil
}

func autoprogrammingBridgeOperationalTasksV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, orquestaautoprogramming.AutoprogrammingRequestIssueV0) {
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		task.ContextRefs = compactStringsV0(append(task.ContextRefs, autoprogrammingBridgeOperationalTaskSourceRefV0))
		repaired, issue := orquestaautoprogramming.EnsureAutoprogrammingWorkflowTaskAcceptedByCoreV0(task)
		if issue.Code != "" {
			return out, issue
		}
		out = append(out, repaired)
	}
	return out, orquestaautoprogramming.AutoprogrammingRequestIssueV0{}
}
