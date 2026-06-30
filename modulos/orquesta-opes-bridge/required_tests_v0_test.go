package orquestaopesbridge

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestBuildExternalWorkRunRequestV0DeclaraRequiredTestsOPES(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-assemble-001",
		Type: "assemble_topic",
	}, JobRunConfigV0{})
	if !ok {
		t.Fatalf("request no construida")
	}
	tests := req.AppChangeRequest.ExternalWork.RequiredTests
	if len(tests) != 1 ||
		tests[0].TestRef != "opes-domain-test-assemble_topic-job-ref-assemble-001" ||
		tests[0].ExternalRefs[2].Ref != "assembled_topic" {
		t.Fatalf("required_tests=%+v", tests)
	}
	if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(orquestadomainwork.DomainWorkJobRequestV0{
		CorrelationID:  "corr-required-tests-opes",
		IdempotencyKey: "idem-required-tests-opes",
		RequestedBy:    "orquesta",
		DomainRef:      "opes",
		WorkKind:       "assemble_topic",
		Objective:      "validar required tests OPES",
		RequiredTests:  tests,
	}); len(issues) != 0 {
		t.Fatalf("required_tests invalidos=%+v", issues)
	}
}

func TestOPESRequiredTestPolicyV0SintetizaSiFaltanDeclarados(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes",
			IdempotencyKey: "idem-policy-opes",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "draft_content_block",
			WorkRefs:       []string{"job-ref-policy-opes"},
			Objective:      "validar policy OPES",
		},
	)
	if err != nil || len(plan.RequiredTests) != 1 ||
		plan.RequiredTests[0].TestRef != "opes-domain-test-draft_content_block-job-ref-policy-opes" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestOPESRequiredTestPolicyV0FinalTemarioExigeMinimosYComunes(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-final",
			IdempotencyKey: "idem-policy-opes-final",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "finalize_temario_package",
			WorkRefs:       []string{"job-ref-policy-opes-final"},
			Objective:      "cerrar temario OPES",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	got := requiredTestRefsForTestV0(plan.RequiredTests)
	for _, want := range []string{
		"opes-domain-test-finalize_temario_package-job-ref-policy-opes-final",
		"opes-extension-minima-nivel-job-ref-policy-opes-final",
		"opes-derivacion-comunes-maestro-job-ref-policy-opes-final",
		"opes-question-bank-publicable-job-ref-policy-opes-final",
		"opes-final-package-manifest-job-ref-policy-opes-final",
	} {
		if !stringInRequiredTestRefsForTestV0(got, want) {
			t.Fatalf("required_tests=%v falta %s", got, want)
		}
	}
	finalManifest := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-final-package-manifest-job-ref-policy-opes-final",
	)
	if finalManifest.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(finalManifest, "artifact_type", "completed_syllabus_package") ||
		!requiredTestHasExternalRefForTestV0(finalManifest, "required_evidence", "manifest_cierre") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.AcceptanceCriteriaRefs, "opes-required-final-package-manifest") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-expected-evidence-manifest-cierre") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:qa") {
		t.Fatalf("final_manifest_required_test=%+v", finalManifest)
	}
}

func requiredTestRefsForTestV0(
	tests []orquestadomainwork.DomainWorkRequiredTestV0,
) []string {
	out := make([]string, 0, len(tests))
	for _, test := range tests {
		out = append(out, test.TestRef)
	}
	return out
}

func stringInRequiredTestRefsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func requiredTestByRefForTestV0(
	tests []orquestadomainwork.DomainWorkRequiredTestV0,
	ref string,
) orquestadomainwork.DomainWorkRequiredTestV0 {
	for _, test := range tests {
		if test.TestRef == ref {
			return test
		}
	}
	return orquestadomainwork.DomainWorkRequiredTestV0{}
}

func requiredTestHasExternalRefForTestV0(
	test orquestadomainwork.DomainWorkRequiredTestV0,
	kind string,
	ref string,
) bool {
	for _, externalRef := range test.ExternalRefs {
		if externalRef.Kind == kind && externalRef.Ref == ref {
			return true
		}
	}
	return false
}
