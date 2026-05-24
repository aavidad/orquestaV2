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
	return []orquestadomainwork.DomainWorkRequiredTestV0{{
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
