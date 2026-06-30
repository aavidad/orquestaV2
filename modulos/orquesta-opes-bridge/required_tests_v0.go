package orquestaopesbridge

import (
	"context"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type OPESRequiredTestPolicyV0 struct{}

var _ orquestadomainwork.DomainWorkRequiredTestPolicyPortV0 = OPESRequiredTestPolicyV0{}

func (OPESRequiredTestPolicyV0) BuildDomainWorkRequiredTestPlanV0(
	_ context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkRequiredTestPlanV0, error) {
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	tests := append([]orquestadomainwork.DomainWorkRequiredTestV0(nil), request.RequiredTests...)
	if strings.TrimSpace(request.DomainRef) == "opes" && len(tests) == 0 {
		tests = opesRequiredTestsForJobV0(request.WorkKind, opesRequiredTestJobRefV0(request), request.WorkRefs)
	}
	return orquestadomainwork.NormalizeDomainWorkRequiredTestPlanV0(
		orquestadomainwork.DomainWorkRequiredTestPlanV0{
			DomainRef:          request.DomainRef,
			WorkKind:           request.WorkKind,
			JobRef:             opesRequiredTestJobRefV0(request),
			AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
			RequiredTests:      tests,
			ExternalRefs:       append([]orquestadomainwork.DomainWorkExternalRefV0(nil), request.ExternalRefs...),
			EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
		},
	), nil
}

func opesRequiredTestsForJobV0(
	jobType string,
	jobRef string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	safeJob := compactOPESBridgeRefV0(jobRef)
	workKind := compactOPESBridgeRefV0(jobType)
	artifactType := compactOPESBridgeRefV0(expectedArtifactTypeV0(jobType))
	tests := []orquestadomainwork.DomainWorkRequiredTestV0{{
		TestRef: "opes-domain-test-" + workKind + "-" + safeJob,
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto por contrato publico submit_artifact.",
			"El receipt OPES conserva job_ref, delivery_ref y trazabilidad causal.",
		},
		AcceptanceCriteriaRefs: []string{"opes-required-artifact-" + artifactType},
		InputRefs:              compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...)),
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "domain_ref", Ref: "opes"},
			{Kind: "job_ref", Ref: safeJob},
			{Kind: "artifact_type", Ref: artifactType},
		},
		EvidenceRefs: []string{
			"opes-job-" + safeJob,
			"opes-expected-artifact-" + artifactType,
		},
	}}
	if opesFinalPackageWorkKindV0(jobType) {
		tests = append(tests, opesFinalPackageRequiredTestsV0(safeJob, workRefs)...)
	}
	return tests
}

func opesFinalPackageRequiredTestsV0(
	safeJob string,
	workRefs []string,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	inputRefs := compactStringsV0(append([]string{"opes-job-" + safeJob}, workRefs...))
	return []orquestadomainwork.DomainWorkRequiredTestV0{
		{
			TestRef: "opes-extension-minima-nivel-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe informe_extension_temario.json y .md con conteo por tema.",
				"Cada ampliado publicable alcanza el minimo de su nivel: A1 20.250, A2 14.400, B 10.800, C1 7.200, C2 4.500 o AP 3.150 palabras.",
				"Si algun tema no llega, el estado es pendiente_continuar con needs_expansion_min_words_<nivel>.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-extension-minima-nivel"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_evidence", Ref: "informe_extension_temario"},
			},
			EvidenceRefs: []string{
				"opes-rule-minimos-extension-temarios-2026-06-22",
				"opes-expected-evidence-informe-extension-temario",
			},
		},
		{
			TestRef: "opes-derivacion-comunes-maestro-" + safeJob,
			AcceptanceCriteria: []string{
				"Cada tema comun declara matriz de derivacion desde maestro comun A1/A1-A2 o superior validado.",
				"Si no hay temas comunes, existe evidencia explicita de no aplicabilidad.",
				"No se acepta reutilizar un curso vecino como canon cuando exista maestro comun superior.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-derivacion-comunes-maestro"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_evidence", Ref: "matriz_reutilizacion_comunes"},
			},
			EvidenceRefs: []string{
				"opes-rule-comunes-a1-genericos",
				"opes-expected-evidence-matriz-reutilizacion-comunes",
			},
		},
		{
			TestRef: "opes-question-bank-publicable-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe tests.json o question_bank equivalente con preguntas publicables.",
				"El banco de preguntas no esta vacio y contiene al menos una pregunta revisable.",
				"Un paquete con tests.json parseable pero sin preguntas queda pendiente_continuar, aunque tenga revision de agente aceptada.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-question-bank-publicable"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_evidence", Ref: "question_bank"},
			},
			EvidenceRefs: []string{
				"opes-rule-question-bank-publicable",
				"opes-expected-evidence-question-bank",
			},
		},
		{
			TestRef: "opes-final-package-manifest-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe manifest_cierre.json del completed_syllabus_package con schema opes_final_package_evidence_manifest.v0.",
				"El manifest identifica package_ref, manifest_ref, checksum_refs, validation_report_ref y review_matrix_ref del paquete final.",
				"El manifest declara evidencias requeridas para HTML, RAG, audio, tests, visual y QA final.",
				"Si falta una evidencia obligatoria, el estado es pendiente_continuar con followup_refs causales y no listo_para_revision_operador.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-final-package-manifest"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "artifact_type", Ref: opesArtifactTypeCompletedSyllabusPackageV0},
				{Kind: "required_evidence", Ref: "manifest_cierre"},
			},
			EvidenceRefs: []string{
				"opes-rule-final-package-manifest",
				"opes-expected-evidence-manifest-cierre",
				"opes-final-evidence:html",
				"opes-final-evidence:rag",
				"opes-final-evidence:audio",
				"opes-final-evidence:tests",
				"opes-final-evidence:visual",
				"opes-final-evidence:qa",
			},
		},
		{
			TestRef: "opes-visual-reuse-manifest-" + safeJob,
			AcceptanceCriteria: []string{
				"Existe visual_reuse_manifest o evidencia explicita de no aplicabilidad para cursos con comunes/assets visuales reutilizables.",
				"Si reusable_visual_count o common_visual_count es mayor que cero, copied_visual_count/inserted_visual_count reflejan assets importados y ubicados.",
				"Si visual_count=0, el manifest declara visual_requirement_status=not_applicable o visual_zero_justification_ref; no se acepta ready/html_validado sin esa evidencia.",
				"Los assets reutilizados conservan refs opacas, placement_ref/ancla, alt_text y motivo editorial; los rechazados conservan motivo de rechazo.",
			},
			AcceptanceCriteriaRefs: []string{"opes-required-visual-reuse-manifest"},
			InputRefs:              inputRefs,
			ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
				{Kind: "domain_ref", Ref: "opes"},
				{Kind: "job_ref", Ref: safeJob},
				{Kind: "required_evidence", Ref: "visual_reuse_manifest"},
			},
			EvidenceRefs: []string{
				"opes-rule-visual-reuse-common-assets",
				"opes-expected-evidence-visual-reuse-manifest",
				"opes-final-evidence:visual_reuse",
			},
		},
	}
}

func opesFinalPackageWorkKindV0(jobType string) bool {
	switch strings.TrimSpace(jobType) {
	case "finalize_topic_package",
		"finalize_temario_package",
		"close_temario_package",
		"finalize_syllabus_package",
		"close_syllabus_package":
		return true
	default:
		return false
	}
}

func opesRequiredTestJobRefV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) string {
	for _, ref := range request.WorkRefs {
		if strings.TrimSpace(ref) != "" {
			return ref
		}
	}
	return request.IdempotencyKey
}
