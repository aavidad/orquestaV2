package orquestaappplanner

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestWorkflowTaskForUnitV0ConvierteUnidadAPerfilNeutral(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "api")

	task, err := WorkflowTaskForUnitV0(plan, unit)
	if err != nil {
		t.Fatalf("WorkflowTaskForUnitV0: %v", err)
	}
	if task.WorkProfileKind != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("work_profile_kind=%s", task.WorkProfileKind)
	}
	if task.TaskID != unit.TaskRef || task.RunID != plan.RunRef || task.PhaseID != unit.PhaseID {
		t.Fatalf("refs task=%+v unit=%+v", task, unit)
	}
	if !sameStringSetForTestV0(task.WriteSet, []string{"cmd/server"}) {
		t.Fatalf("write_set=%v", task.WriteSet)
	}
	if !sameStringSetForTestV0(task.RequiredTests, []string{"go test ./..."}) {
		t.Fatalf("required_tests=%v", task.RequiredTests)
	}
	if !sameStringSetForTestV0(task.DependsOn, []string{"task-agenda-agenda-core", "task-agenda-web"}) {
		t.Fatalf("depends_on=%v", task.DependsOn)
	}
	if len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != "contract-task-agenda-api" {
		t.Fatalf("function_contract_refs=%+v", task.FunctionContractRefs)
	}
}

func TestWorkflowTaskForUnitV0ConvierteTodosLosPlanesGenerados(t *testing.T) {
	requests := []AppPlanRequestV0{
		{
			RunRef:  "run-ref-standard-001",
			AppRef:  "agenda",
			AppName: "Agenda",
			API:     true,
			Web:     true,
		},
		{
			RunRef:  "run-ref-large-001",
			AppRef:  "erp",
			AppName: "ERP",
			API:     true,
			Web:     true,
			Scale:   AppPlanScaleLargeV0,
		},
	}
	for _, request := range requests {
		plan, err := BuildGoAPIWebMicrotaskPlanV0(request)
		if err != nil {
			t.Fatalf("BuildGoAPIWebMicrotaskPlanV0(%s): %v", request.AppRef, err)
		}
		for _, unit := range plan.Units {
			task, err := WorkflowTaskForUnitV0(plan, unit)
			if err != nil {
				t.Fatalf("%s WorkflowTaskForUnitV0(%s): %v", request.AppRef, unit.TaskRef, err)
			}
			if task.WorkProfileKind == "" || len(task.FunctionContractRefs) != 1 {
				t.Fatalf("%s task incompleta: %+v", unit.TaskRef, task)
			}
		}
	}
}

func TestWorkProfileForUnitV0UsaRolesExistentesSinPerfilParalelo(t *testing.T) {
	plan, err := BuildGoAPIWebMicrotaskPlanV0(AppPlanRequestV0{
		RunRef:  "run-ref-large-001",
		AppRef:  "erp",
		AppName: "ERP",
		API:     true,
		Web:     true,
		Scale:   AppPlanScaleLargeV0,
	})
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanV0: %v", err)
	}

	assertUnitWorkProfileKindForTestV0(t, plan, "architecture", orquestacoreworkflow.WorkProfileCodeStudyV0)
	assertUnitWorkProfileKindForTestV0(t, plan, "docs", orquestacoreworkflow.WorkProfileDocumentationV0)
	assertUnitWorkProfileKindForTestV0(t, plan, "review", orquestacoreworkflow.WorkProfileReviewV0)
	assertUnitWorkProfileKindForTestV0(t, plan, "integration", orquestacoreworkflow.WorkProfileImplementationV0)
}

func TestValidateAppWorkUnitV0RechazaPerfilDesconocido(t *testing.T) {
	unit := appUnitByKeyForTestV0(t, mustAppPlanForTestV0(t), "api")
	unit.WorkProfileKind = "perfil_desconocido"

	err := validateAppWorkUnitV0(unit)
	if issue, ok := err.(AppPlannerIssueV0); !ok || issue.Field != "work_profile_kind" {
		t.Fatalf("err=%v", err)
	}
}

func TestWorkProfileForUnitV0RechazaDependenciaSinUnidadEnPlan(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "api")
	unit.DependsOnDeliveries = append(unit.DependsOnDeliveries, "ack-agenda-no-existe")

	_, err := WorkProfileForUnitV0(plan, unit)
	if issue, ok := err.(AppPlannerIssueV0); !ok || issue.Field != "depends_on_deliveries" {
		t.Fatalf("err=%v", err)
	}
}

func TestWorkProfileForUnitV0RechazaUnidadFueraDelPlan(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "api")
	unit.TaskRef = "task-agenda-externa"

	_, err := WorkProfileForUnitV0(plan, unit)
	if issue, ok := err.(AppPlannerIssueV0); !ok || issue.Field != "task_ref" {
		t.Fatalf("err=%v", err)
	}
}

func assertUnitWorkProfileKindForTestV0(
	t *testing.T,
	plan AppMicrotaskPlanV0,
	key string,
	want orquestacoreworkflow.WorkProfileKindV0,
) {
	t.Helper()
	unit := appUnitByKeyForTestV0(t, plan, key)
	if unit.WorkProfileKind != want {
		t.Fatalf("%s unit work_profile_kind=%s want=%s", key, unit.WorkProfileKind, want)
	}
	profile, err := WorkProfileForUnitV0(plan, unit)
	if err != nil {
		t.Fatalf("%s WorkProfileForUnitV0: %v", key, err)
	}
	if profile.ProfileKind != want {
		t.Fatalf("%s profile_kind=%s want=%s", key, profile.ProfileKind, want)
	}
	task, err := WorkflowTaskForUnitV0(plan, unit)
	if err != nil {
		t.Fatalf("%s WorkflowTaskForUnitV0: %v", key, err)
	}
	if task.WorkProfileKind != want {
		t.Fatalf("%s task work_profile_kind=%s want=%s", key, task.WorkProfileKind, want)
	}
}
