package orquestaapprunner

import (
	"testing"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestAppRunWorkflowTaskV0ReutilizaPerfilDelPlanner(t *testing.T) {
	plan := mustRunnerPlannerPlanForTestV0(t)
	unit := runnerPlanUnitByTaskRefForTestV0(t, plan, "task-agenda-api")
	contractRef := appRunContractRefV0(plan)

	task, err := appRunWorkflowTaskV0(plan, contractRef, unit)
	if err != nil {
		t.Fatalf("appRunWorkflowTaskV0: %v", err)
	}
	if task.WorkProfileKind != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("work_profile_kind=%s", task.WorkProfileKind)
	}
	if !sameRunnerStringsForTestV0(task.RequiredTests, []string{"go test ./..."}) {
		t.Fatalf("required_tests=%v", task.RequiredTests)
	}
	if !sameRunnerStringsForTestV0(task.DependsOn, []string{"task-agenda-agenda-core", "task-agenda-web"}) {
		t.Fatalf("depends_on=%v", task.DependsOn)
	}
	if len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != contractRef ||
		task.FunctionContractRefs[0].FunctionName != "app_unit_task_agenda_api" {
		t.Fatalf("function_contract_refs=%+v", task.FunctionContractRefs)
	}
}

func mustRunnerPlannerPlanForTestV0(t *testing.T) orquestaappplanner.AppMicrotaskPlanV0 {
	t.Helper()
	plan, err := orquestaappplanner.BuildGoAPIWebMicrotaskPlanV0(orquestaappplanner.AppPlanRequestV0{
		RunRef:  "run-ref-app-runner-plan-001",
		AppRef:  "agenda",
		AppName: "Agenda",
		API:     true,
		Web:     true,
	})
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanV0: %v", err)
	}
	return plan
}

func runnerPlanUnitByTaskRefForTestV0(
	t *testing.T,
	plan orquestaappplanner.AppMicrotaskPlanV0,
	taskRef string,
) orquestaappplanner.AppWorkUnitV0 {
	t.Helper()
	for _, unit := range plan.Units {
		if unit.TaskRef == taskRef {
			return unit
		}
	}
	t.Fatalf("unit %s no encontrada", taskRef)
	return orquestaappplanner.AppWorkUnitV0{}
}

func sameRunnerStringsForTestV0(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
