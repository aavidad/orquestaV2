package orquestacoreworkflow

import "testing"

func TestWorkflowTaskFromWorkProfileV0CodeStudyDefaultsPhaseAndCriteria(t *testing.T) {
	task, err := WorkflowTaskFromWorkProfileV0(validWorkProfileV0(WorkProfileCodeStudyV0))
	if err != nil {
		t.Fatalf("WorkflowTaskFromWorkProfileV0: %v", err)
	}
	if task.PhaseID != OrchestrationPhaseBrainstormingArquitecturaV0 {
		t.Fatalf("phase_id=%s", task.PhaseID)
	}
	if task.WorkProfileKind != WorkProfileCodeStudyV0 {
		t.Fatalf("work_profile_kind=%s", task.WorkProfileKind)
	}
	if len(task.AcceptanceCriteria) < 2 ||
		task.AcceptanceCriteria[0] != "Mapa de componentes, riesgos y puntos de cambio documentado." {
		t.Fatalf("criteria=%+v", task.AcceptanceCriteria)
	}
}

func TestWorkflowTaskFromWorkProfileV0RefactorRequiresTestsAndPreservesLineage(t *testing.T) {
	profile := validWorkProfileV0(WorkProfileRefactorV0)
	profile.RequiredTests = []string{"go test -count=1 ./..."}
	profile.ParentTaskRef = "task-ref-parent-001"
	profile.CohortRef = "cohort-ref-profile-001"
	profile.WaveRef = "wave-ref-profile-001"
	profile.DelegationDepth = 2
	profile.MaxDelegationDepth = 4
	profile.MaxChildAgents = 4
	profile.MaxSubagentsPerAgent = 4
	profile.MaxRecursiveAgents = 16
	profile.ChildTaskRefs = []string{"task-ref-child-001"}

	task, err := WorkflowTaskFromWorkProfileV0(profile)
	if err != nil {
		t.Fatalf("WorkflowTaskFromWorkProfileV0: %v", err)
	}
	if task.PhaseID != OrchestrationPhaseProgramacionV0 ||
		task.ParentTaskRef != "task-ref-parent-001" ||
		task.CohortRef != "cohort-ref-profile-001" ||
		task.WaveRef != "wave-ref-profile-001" ||
		task.DelegationDepth != 2 ||
		task.MaxDelegationDepth != 4 ||
		task.MaxChildAgents != 4 ||
		task.MaxSubagentsPerAgent != 4 ||
		task.MaxRecursiveAgents != 16 ||
		len(task.ChildTaskRefs) != 1 ||
		task.ChildTaskRefs[0] != "task-ref-child-001" {
		t.Fatalf("task=%+v", task)
	}
}

func TestValidateWorkProfileV0RejectsImplementationWithoutRequiredTests(t *testing.T) {
	err := ValidateWorkProfileV0(validWorkProfileV0(WorkProfileImplementationV0))
	assertWorkProfileErrorV0(t, err, ErrWorkProfileInvalidoV0, "required_tests")
}

func TestWorkflowTaskFromWorkProfileV0RejectsImplementationWithoutRequiredTests(t *testing.T) {
	_, err := WorkflowTaskFromWorkProfileV0(validWorkProfileV0(WorkProfileImplementationV0))
	assertWorkProfileErrorV0(t, err, ErrWorkProfileInvalidoV0, "required_tests")
}

func TestValidateWorkProfileV0RejectsMissingFunctionRefs(t *testing.T) {
	profile := validWorkProfileV0(WorkProfileCodeStudyV0)
	profile.FunctionContractRefs = nil

	err := ValidateWorkProfileV0(profile)
	assertWorkProfileErrorV0(t, err, ErrWorkProfileInvalidoV0, "function_contract_refs")
}

func TestValidateWorkProfileV0RejectsUnsafeScopeRefsThroughWorkflowTask(t *testing.T) {
	profile := validWorkProfileV0(WorkProfileCodeStudyV0)
	profile.ScopeRefs = []string{"../secrets"}

	err := ValidateWorkProfileV0(profile)
	assertWorkProfileErrorV0(t, err, ErrWorkProfileTaskInvalidaV0, "workflow_task.write_set")
}

func TestValidateWorkflowTaskV0RejectsUnknownWorkProfileKind(t *testing.T) {
	task := validWorkflowTaskV0()
	task.WorkProfileKind = "perfil_desconocido"

	err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
	assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, "work_profile_kind")
}

func TestNormalizeWorkProfileKindV0Aliases(t *testing.T) {
	for _, item := range []struct {
		alias string
		want  WorkProfileKindV0
	}{
		{alias: "programacion", want: WorkProfileImplementationV0},
		{alias: "estudio-codigo", want: WorkProfileCodeStudyV0},
		{alias: "refactorizacion", want: WorkProfileRefactorV0},
		{alias: "pruebas", want: WorkProfileRequiredTestsV0},
		{alias: "external-work", want: WorkProfileDomainWorkV0},
	} {
		got := NormalizeWorkProfileKindV0(WorkProfileKindV0(item.alias))
		if got != item.want {
			t.Fatalf("alias=%s got=%s want=%s", item.alias, got, item.want)
		}
		if _, ok := LookupWorkProfileDefinitionV0(WorkProfileKindV0(item.alias)); !ok {
			t.Fatalf("definition no encontrada para %s", item.alias)
		}
	}
}

func validWorkProfileV0(kind WorkProfileKindV0) WorkProfileV0 {
	return WorkProfileV0{
		SchemaVersion: WorkProfileSchemaVersionV0,
		ProfileRef:    "work-profile-ref-001",
		ProfileKind:   kind,
		TaskRef:       "task-ref-work-profile-001",
		RunRef:        "run-ref-work-profile-001",
		Title:         "Resolver perfil de trabajo",
		Objective:     "Convertir perfil neutral en tarea durable.",
		ScopeRefs:     []string{"docs/work-profile.md"},
		AcceptanceCriteria: []string{
			"Resultado trazable por refs compactas.",
		},
		FunctionContractRefs: []WorkflowFunctionContractRefV0{{
			ContractRef:  "contract-ref-work-profile-001",
			FunctionName: "ApplyWorkProfileV0",
		}},
	}
}

func assertWorkProfileErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("err nil")
	}
	publicErr, ok := err.(WorkProfileErrorV0)
	if !ok {
		t.Fatalf("err=%T %#v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("err=%+v want code=%s field=%s", publicErr, code, field)
	}
}
