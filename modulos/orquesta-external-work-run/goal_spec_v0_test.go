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

func TestBuildExternalWorkGoalWorkSpecV0InlineaInputFieldsOperativosSeguros(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork.InputFields = append(
		request.AppChangeRequest.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "course_root_abs",
			Value: "/home/alberto/Trabajo/OPES/opes-salidas/curso",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "topic_dir_abs",
			Value: "/home/alberto/Trabajo/OPES/opes-salidas/curso/tema_001",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:  "program_json_abs",
			Value: "/home/alberto/Trabajo/OPES/programas/programa.json",
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_read_refs",
			Values: []string{"temario/tema_001.md", "normativa/ley-ref-001"},
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_outputs",
			Values: []string{"04_markdown/tema_001.md", "paquete_final/tests.json"},
		},
		orquestadomainwork.DomainWorkFieldV0{
			Name:      "output_contract",
			ValueJSON: []byte(`{"artifact_type":"content_block","min_words":1200}`),
		},
	)

	spec, issues := BuildExternalWorkGoalWorkSpecV0(
		request,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	context := externalWorkRunTestContextTextV0(spec.ContextRefs)
	for _, want := range []string{
		"kind=input_field_payload",
		"app_change_payload:run-external-work-opes-job-ref-001-change-ref-001:change-ref-001:external_work.input_fields.course_root_abs",
		"AppChangeRecordFilterV0",
		"input_fields.course_root_abs",
		"local_path_ref:",
		"basename=curso",
		"input_fields.topic_dir_abs",
		"basename=tema_001",
		"input_fields.program_json_abs",
		"basename=programa.json",
		"input_fields.required_read_refs",
		"temario/tema_001.md",
		"normativa/ley-ref-001",
		"input_fields.required_outputs",
		"04_markdown/tema_001.md",
		"paquete_final/tests.json",
		"input_fields.output_contract",
		`"artifact_type":"content_block"`,
		"payload_ref",
	} {
		if !strings.Contains(context, want) {
			t.Fatalf("context no contiene %q:\n%s", want, context)
		}
	}
	for _, forbidden := range []string{
		"/home/alberto/Trabajo/OPES",
		"/home-redacted/",
	} {
		if strings.Contains(context, forbidden) {
			t.Fatalf("context contiene ruta local cruda %q:\n%s", forbidden, context)
		}
	}
	if !externalWorkRunTestContainsStringV0(
		spec.AcceptanceCriteria,
		"Usar los input_fields inlineados en context_refs[input_field_value] como contrato operativo compacto; los campos omitidos por redaccion o presupuesto quedan como payload_ref y solo deben resolverse si son imprescindibles para el artefacto. Bloquear con rework de dominio solo si falta un input imprescindible, no por un campo accesorio omitido.",
	) {
		t.Fatalf("acceptance_criteria=%+v", spec.AcceptanceCriteria)
	}
}

func TestBuildExternalWorkGoalWorkSpecV0RedactaInputFieldsSensibles(t *testing.T) {
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
	context := externalWorkRunTestContextTextV0(spec.ContextRefs)
	for _, ctx := range spec.ContextRefs {
		if strings.Contains(ctx.Ref, "valor-que-no-debe") ||
			strings.Contains(ctx.Ref, "otro-valor-privado") ||
			strings.Contains(ctx.Ref, "no-copiar") ||
			strings.Contains(ctx.Purpose, "valor-que-no-debe") ||
			strings.Contains(ctx.Purpose, "otro-valor-privado") ||
			strings.Contains(ctx.Purpose, "no-copiar") ||
			strings.Contains(ctx.Ref, "private_notes") ||
			strings.Contains(ctx.Purpose, "private_notes") ||
			strings.Contains(context, "secret") {
			t.Fatalf("context_refs contiene payload: %+v", spec.ContextRefs)
		}
	}
}

func TestBuildExternalWorkGoalWorkSpecV0NoInlineaInputFieldsMasivos(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	largeValues := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		largeValues = append(largeValues, "regla editorial extensa que debe quedar solo por payload_ref "+string(rune('a'+i)))
	}
	request.AppChangeRequest.ExternalWork.InputFields = append(
		request.AppChangeRequest.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "opes_global_editorial_policy",
			Values: largeValues,
		},
	)

	spec, issues := BuildExternalWorkGoalWorkSpecV0(
		request,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	context := externalWorkRunTestContextTextV0(spec.ContextRefs)
	for _, want := range []string{
		"kind=input_field_payload",
		"external_work.input_fields.opes_global_editorial_policy",
		"input_fields.opes_global_editorial_policy no inlineado por sensibilidad, tamano o presupuesto",
	} {
		if !strings.Contains(context, want) {
			t.Fatalf("context no contiene %q:\n%s", want, context)
		}
	}
	if strings.Contains(context, "regla editorial extensa que debe quedar solo por payload_ref") {
		t.Fatalf("context contiene campo masivo inlineado:\n%s", context)
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

func externalWorkRunTestContextTextV0(
	values []orquestagoal.GoalContextRefV0,
) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, value.Ref+" kind="+value.Kind+" "+value.Purpose)
	}
	return strings.Join(parts, "\n")
}
