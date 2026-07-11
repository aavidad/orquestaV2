package orquestaautoprogramming

import (
	"reflect"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildAutoprogrammingProgrammableWorkV1UsaPerfilPorTipoDeAppYArea(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.WorkProfiles = []AutoprogrammingWorkProfileV1{
			{
				AppKind:     "web_application",
				Area:        "Orchestration Core",
				ProfileKind: orquestacoreworkflow.WorkProfileDocumentationV0,
				FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
					{ContractRef: "contract-ref-web-docs-v1"},
				},
			},
		}
	}))
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	task := result.Work.Base.Tasks[0]
	if task.WorkProfileKind != orquestacoreworkflow.WorkProfileDocumentationV0 {
		t.Fatalf("work_profile_kind=%s", task.WorkProfileKind)
	}
	if result.Work.ProfileBindings[0].Source != "area:orchestration-core" {
		t.Fatalf("profile_bindings=%+v", result.Work.ProfileBindings)
	}
	if !workflowTaskHasContractRefV1Test(task, "contract-ref-web-docs-v1") {
		t.Fatalf("function_contract_refs=%+v", task.FunctionContractRefs)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV1PriorizaPerfilPorTaskRef(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.WorkProfiles = []AutoprogrammingWorkProfileV1{
			{
				AppKind:     "web_application",
				Area:        "Orchestration Core",
				ProfileKind: orquestacoreworkflow.WorkProfileDocumentationV0,
			},
			{
				AppKind:     "web_application",
				TaskRef:     "task-ref-autoprogramming-a",
				ProfileKind: orquestacoreworkflow.WorkProfileRefactorV0,
			},
		}
	}))
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if got := result.Work.Base.Tasks[0].WorkProfileKind; got != orquestacoreworkflow.WorkProfileRefactorV0 {
		t.Fatalf("work_profile_kind=%s", got)
	}
	if result.Work.ProfileBindings[0].Source != "task_ref:task-ref-autoprogramming-a" {
		t.Fatalf("profile_bindings=%+v", result.Work.ProfileBindings)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV1NormalizaAliasDeTipoApp(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.AppKind = "web_app"
		request.WorkProfiles = []AutoprogrammingWorkProfileV1{
			{
				AppKind:     "web_application",
				ProfileKind: orquestacoreworkflow.WorkProfileDocumentationV0,
			},
		}
	}))
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if result.Work.AppKind != "web-application" {
		t.Fatalf("app_kind=%s", result.Work.AppKind)
	}
	if got := result.Work.Base.Tasks[0].WorkProfileKind; got != orquestacoreworkflow.WorkProfileDocumentationV0 {
		t.Fatalf("work_profile_kind=%s", got)
	}
	if result.Work.ProfileBindings[0].Source != "app_kind" {
		t.Fatalf("profile_bindings=%+v", result.Work.ProfileBindings)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV1PropagaSkillRefsDePerfil(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.WorkProfiles = []AutoprogrammingWorkProfileV1{
			{
				AppKind:     "web_application",
				ProfileKind: orquestacoreworkflow.WorkProfileImplementationV0,
				SkillRefs: []string{
					"skill-ref-catalogo-declarado-v0",
				},
			},
		}
	}))
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if !stringsSliceContainsForAutoprogrammingTestV0(
		result.Work.Base.Tasks[0].SkillRefs,
		"skill-ref-catalogo-declarado-v0",
	) {
		t.Fatalf("skill_refs=%v", result.Work.Base.Tasks[0].SkillRefs)
	}
}

func TestValidateAutoprogrammingRequestV1RechazaPerfilSinAppKind(t *testing.T) {
	result := ValidateAutoprogrammingRequestV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.AppKind = ""
		request.WorkProfiles = []AutoprogrammingWorkProfileV1{{
			ProfileKind: orquestacoreworkflow.WorkProfileDocumentationV0,
		}}
	}))
	assertAutoprogrammingRequestIssueV0(t, result, "app_kind_missing")
}

func TestBuildAutoprogrammingProgrammableWorkV1ConservaFallbackV0(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(nil))
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if got := result.Work.Base.Tasks[0].WorkProfileKind; got != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("work_profile_kind=%s", got)
	}
	if result.Work.ProfileBindings[0].Source != "default-v0" {
		t.Fatalf("profile_bindings=%+v", result.Work.ProfileBindings)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV1GoalReadyConservaAcceptanceChecksYGoalSpecs(t *testing.T) {
	command := "go test -count=1 ./modulos/orquesta-autoprogramming -run TestBUG208AG"
	checks := []AutoprogrammingAcceptanceCheckV0{
		{
			CriterionRef: "criterion-ref-bug-208ag-v1-001",
			Description:  "El primer criterio queda asociado al comando.",
			Command:      command,
		},
		{
			CriterionRef: "criterion-ref-bug-208ag-v1-002",
			Description:  "El segundo criterio comparte el mismo comando.",
			Command:      command,
		},
	}
	result := BuildAutoprogrammingProgrammableWorkV1(validAutoprogrammingRequestV1(func(request *AutoprogrammingRequestV1) {
		request.RequiredTests = []string{command}
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-acceptance-check-v1-001",
			Area:    "autoprogramming",
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
			},
			AcceptanceChecks: checks,
		}}
	}))
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	if len(result.Work.Base.Groups) != 1 ||
		!reflect.DeepEqual(result.Work.Base.Groups[0].AcceptanceChecks, checks) {
		t.Fatalf("acceptance_checks=%+v", result.Work.Base.Groups)
	}
	if len(result.Work.Base.GoalSpecs) != 1 {
		t.Fatalf("stored goal_specs=%+v", result.Work.Base.GoalSpecs)
	}
	if len(result.Work.Base.Profiles) != 0 || len(result.Work.Base.Tasks) != 0 {
		t.Fatalf("legacy workflow surface profiles=%+v tasks=%+v", result.Work.Base.Profiles, result.Work.Base.Tasks)
	}
	if !reflect.DeepEqual(result.Work.Base.Groups[0].Profile, orquestacoreworkflow.WorkProfileV0{}) ||
		!reflect.DeepEqual(result.Work.Base.Groups[0].Task, orquestacoreworkflow.WorkflowTaskV0{}) {
		t.Fatalf("legacy group workflow surface=%+v", result.Work.Base.Groups[0])
	}
	if len(result.Work.ProfileBindings) != 1 || result.Work.ProfileBindings[0].Source != "default-v0" {
		t.Fatalf("profile_bindings=%+v", result.Work.ProfileBindings)
	}
	spec := result.Work.Base.GoalSpecs[0]
	if len(spec.RequiredTests) != 1 ||
		!reflect.DeepEqual(spec.RequiredTests[0].AcceptanceCriteria, []string{
			checks[0].Description,
			checks[1].Description,
		}) ||
		!reflect.DeepEqual(spec.RequiredTests[0].AcceptanceCriteriaRefs, []string{
			checks[0].CriterionRef,
			checks[1].CriterionRef,
		}) ||
		!reflect.DeepEqual(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs, []string{
			checks[0].CriterionRef,
			checks[1].CriterionRef,
		}) {
		t.Fatalf("required_tests=%+v closure_policy=%+v", spec.RequiredTests, spec.ClosurePolicy)
	}
}

func validAutoprogrammingRequestV1(
	mutate func(*AutoprogrammingRequestV1),
) AutoprogrammingRequestV1 {
	request := AutoprogrammingRequestV1{
		AutoprogrammingRequestV0: validAutoprogrammingRequestV0(nil),
		SchemaVersion:            AutoprogrammingRequestSchemaVersionV1,
		AppKind:                  "web-application",
	}
	if mutate != nil {
		mutate(&request)
	}
	return request
}

func workflowTaskHasContractRefV1Test(
	task orquestacoreworkflow.WorkflowTaskV0,
	contractRef string,
) bool {
	for _, ref := range task.FunctionContractRefs {
		if ref.ContractRef == contractRef {
			return true
		}
	}
	return false
}
