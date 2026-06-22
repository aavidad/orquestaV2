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
