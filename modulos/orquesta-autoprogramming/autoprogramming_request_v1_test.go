package orquestaautoprogramming

import (
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
