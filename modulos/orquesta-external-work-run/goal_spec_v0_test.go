package orquestaexternalworkrun

import (
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestBuildExternalWorkGoalWorkSpecV0CompilaContratoNeutral(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.AllowedWriteSet = []string{"deliveries/opes/job-ref-001"}
	request.AppChangeRequest.RequiredTests = []string{"go test -count=1 ./..."}
	request.AppChangeRequest.ExternalWork.RequiredTests = []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef:                "domain-test-ref-qc",
		AcceptanceCriteriaRefs: []string{"criteria-ref-domain-qc"},
		EvidenceRefs:           []string{"evidence-ref-domain-policy"},
	}}

	spec, issues := BuildExternalWorkGoalWorkSpecV0(
		request,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if validationIssues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(validationIssues) != 0 {
		t.Fatalf("goal validation issues=%+v spec=%+v", validationIssues, spec)
	}
	if spec.DirectorKind != orquestagoal.GoalDirectorKindRuntimeGoalV0 ||
		spec.WorkProfileKind != ExternalWorkGoalWorkProfileKindV0 ||
		spec.RunRef != "run-external-work-opes-job-ref-001-change-ref-001" ||
		spec.ProjectRef != "opes" ||
		spec.DomainRef != "opes" ||
		spec.WorkKind != "draft_content_block" {
		t.Fatalf("spec basico inesperado=%+v", spec)
	}
	if len(spec.WriteSet) != 1 ||
		spec.WriteSet[0].Path != "deliveries/opes/job-ref-001" {
		t.Fatalf("write_set=%+v", spec.WriteSet)
	}
	if len(spec.RequiredTests) != 2 ||
		spec.RequiredTests[0].Command != "go test -count=1 ./..." ||
		spec.RequiredTests[1].TestRef != "domain-test-ref-qc" {
		t.Fatalf("required_tests=%+v", spec.RequiredTests)
	}
	if !spec.ClosurePolicy.RequireRequiredTests ||
		!spec.ClosurePolicy.RequireArtifacts ||
		!spec.ClosurePolicy.RequireDomainReceipt {
		t.Fatalf("closure_policy=%+v", spec.ClosurePolicy)
	}
	if len(spec.ArtifactContracts) != 1 ||
		spec.ArtifactContracts[0].ArtifactType != orquestadomainwork.DomainWorkArtifactTypeContentBlockV0 ||
		!spec.ArtifactContracts[0].Required {
		t.Fatalf("artifact_contracts=%+v", spec.ArtifactContracts)
	}
	if !externalWorkRunTestContainsStringV0(spec.AcceptanceCriteria, "markdown valido") {
		t.Fatalf("acceptance_criteria=%+v", spec.AcceptanceCriteria)
	}
}

func TestBuildExternalWorkGoalWorkSpecV0UsaWriteSetLogicoSinPayload(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork.InputFields = append(
		request.AppChangeRequest.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:      "private_notes",
			Value:     "valor-que-no-debe-ser-context-ref",
			Values:    []string{"otro-valor-privado"},
			ValueJSON: []byte(`{"secret":"no-copiar"}`),
		},
	)

	spec, issues := BuildExternalWorkGoalWorkSpecV0(
		request,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(spec.WriteSet) != 1 ||
		!strings.HasPrefix(spec.WriteSet[0].Path, "domain-work/opes/draft_content_block/") {
		t.Fatalf("write_set=%+v", spec.WriteSet)
	}
	for _, ctx := range spec.ContextRefs {
		if strings.Contains(ctx.Ref, "valor-que-no-debe") ||
			strings.Contains(ctx.Ref, "otro-valor-privado") ||
			strings.Contains(ctx.Ref, "no-copiar") {
			t.Fatalf("context_refs contiene payload: %+v", spec.ContextRefs)
		}
	}
	if !externalWorkRunTestContainsContextRefV0(spec.ContextRefs, "input_field", "input-field-private_notes") {
		t.Fatalf("context_refs no declaran nombre de campo: %+v", spec.ContextRefs)
	}
}

func TestBuildExternalWorkGoalWorkSpecV0CompilaDominioNoOPESYWorkKindDesconocido(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.AppRef = "agenda"
	request.AppChangeRequest.ChangeRef = "change-agenda-001"
	request.AppChangeRequest.UserIntent = "Componer resumen externo de agenda."
	request.AppChangeRequest.TargetArea = "resumenes"
	request.AppChangeRequest.AcceptanceCriteria = []string{"resumen trazable"}
	request.AppChangeRequest.ExternalWork.ProjectRef = "domain-ref-agenda"
	request.AppChangeRequest.ExternalWork.JobRef = "job-ref-agenda-001"
	request.AppChangeRequest.ExternalWork.WorkKind = "compose_external_summary"
	request.AppChangeRequest.ExternalWork.WorkRefs = []string{"agenda-ref-week-01"}
	request.AppChangeRequest.ExternalWork.InputFields = []orquestadomainwork.DomainWorkFieldV0{{
		Name:  "source_ref",
		Value: "source-ref-agenda-week",
	}}

	spec, issues := BuildExternalWorkGoalWorkSpecV0(
		request,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if spec.ProjectRef != "domain-ref-agenda" ||
		spec.DomainRef != "domain-ref-agenda" ||
		spec.WorkKind != "compose_external_summary" ||
		spec.ArtifactContracts[0].ArtifactType != orquestadomainwork.DomainWorkArtifactTypeGenericWorkDeliveryV0 {
		t.Fatalf("spec dominio no OPES inesperado=%+v", spec)
	}
	if !strings.HasPrefix(spec.WriteSet[0].Path, "domain-work/domain-ref-agenda/compose_external_summary/") {
		t.Fatalf("write_set=%+v", spec.WriteSet)
	}
}

func TestBuildExternalWorkGoalWorkSpecV0DevuelveIssuesSinExternalWork(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork = nil

	_, issues := BuildExternalWorkGoalWorkSpecV0(
		request,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if len(issues) == 0 ||
		issues[0].Code != ErrExternalWorkRunExternalWorkRequiredV0 ||
		issues[0].Field != "app_change_request.external_work" {
		t.Fatalf("issues=%+v", issues)
	}
}

func externalWorkRunTestContainsStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func externalWorkRunTestContainsContextRefV0(
	values []orquestagoal.GoalContextRefV0,
	kind string,
	ref string,
) bool {
	for _, value := range values {
		if value.Kind == kind && value.Ref == ref {
			return true
		}
	}
	return false
}
