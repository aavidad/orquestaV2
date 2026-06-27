package orquestaappcodexstack

import (
	"fmt"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func validateCompositeDirectorDecisionBatchV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	requestKind string,
	objectiveHints []string,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	if err := validateCompositeDirectorDecisionRefsV0(run, decisions); err != nil {
		return err
	}
	tasks := compositeProgrammingMicrotasksV0(decisions)
	if err := validateCompositeProgrammingMicrotaskAnchorsV0(
		requestKind,
		objectiveHints,
		tasks,
	); err != nil {
		return err
	}
	initialTasks := compositeInitialProgrammingMicrotasksV0(tasks)
	if len(initialTasks) == 0 || !compositeLooksLikeGoAppPlanV0(initialTasks) {
		return nil
	}
	if !compositeTasksRequireGoTestAllV0(initialTasks) {
		return fmt.Errorf("director_decisions invalidas: app Go sin required_tests go test ./... requerido")
	}
	if compositeGoPlanIsIncrementalV0(run, initialTasks) {
		return nil
	}
	if !compositeTasksCoverWriteSetV0(initialTasks, "go.mod") {
		return fmt.Errorf("director_decisions invalidas: app Go sin tarea para go.mod")
	}
	if !compositeTasksCoverCmdEntrypointV0(initialTasks) {
		return fmt.Errorf("director_decisions invalidas: app Go sin tarea para cmd/server")
	}
	if !compositeTasksDependOnBootstrapV0(initialTasks) {
		return fmt.Errorf("director_decisions invalidas: app Go sin depends_on hacia bootstrap")
	}
	return nil
}

func compositeGoPlanIsIncrementalV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	existing := compositeStringSetV0(run.Tasks)
	if len(existing) == 0 {
		return false
	}
	for _, task := range tasks {
		if compositeTaskCoversGoModV0(task) {
			continue
		}
		if !compositeTaskDependsOnAnyV0(task, existing) {
			return false
		}
	}
	return true
}
