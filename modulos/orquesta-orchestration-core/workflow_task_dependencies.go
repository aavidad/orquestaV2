package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func workflowTaskDependenciesSatisfiedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, dependency := range task.DependsOn {
		if !workflowTaskDependencySatisfiedV0(run, dependency) {
			return false
		}
	}
	return true
}

func workflowTaskDependencySatisfiedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	dependency string,
) bool {
	dependency = strings.TrimSpace(dependency)
	if dependency == "" {
		return true
	}
	return stringInSetV0(dependency, run.DeliveredTasks) ||
		stringInSetV0(dependency, run.ClosedTasks)
}
