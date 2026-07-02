package orquestaopesbridge

import (
	"context"
	"strings"
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
		"opes-extension_pass-job-ref-policy-opes-final",
		"opes-official_text_qa_pass-job-ref-policy-opes-final",
		"opes-strict_editorial_qa_pass-job-ref-policy-opes-final",
		"opes-derivacion-comunes-maestro-job-ref-policy-opes-final",
		"opes-question-bank-publicable-job-ref-policy-opes-final",
		"opes-final-package-manifest-job-ref-policy-opes-final",
		"opes-visual-reuse-manifest-job-ref-policy-opes-final",
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
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:extension_pass") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:official_text_qa_pass") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:strict_editorial_qa_pass") {
		t.Fatalf("final_manifest_required_test=%+v", finalManifest)
	}
	extensionPass := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-extension_pass-job-ref-policy-opes-final",
	)
	if extensionPass.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(extensionPass, "required_test_name", "extension_pass") ||
		!requiredTestHasExternalRefForTestV0(extensionPass, "required_evidence", "informe_extension_temario") ||
		!stringInRequiredTestRefsForTestV0(extensionPass.AcceptanceCriteriaRefs, "opes-required-extension_pass") ||
		!stringInRequiredTestRefsForTestV0(extensionPass.EvidenceRefs, "opes-final-evidence:extension_pass") {
		t.Fatalf("extension_pass_required_test=%+v", extensionPass)
	}
	officialTextQA := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-official_text_qa_pass-job-ref-policy-opes-final",
	)
	if officialTextQA.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(officialTextQA, "required_test_name", "official_text_qa_pass") ||
		!requiredTestHasExternalRefForTestV0(officialTextQA, "required_evidence", "official_text_qa_report") ||
		!stringInRequiredTestRefsForTestV0(officialTextQA.AcceptanceCriteriaRefs, "opes-required-official_text_qa_pass") ||
		!stringInRequiredTestRefsForTestV0(officialTextQA.EvidenceRefs, "opes-final-evidence:official_text_qa_pass") {
		t.Fatalf("official_text_qa_pass_required_test=%+v", officialTextQA)
	}
	strictEditorialQA := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-strict_editorial_qa_pass-job-ref-policy-opes-final",
	)
	if strictEditorialQA.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(strictEditorialQA, "required_test_name", "strict_editorial_qa_pass") ||
		!requiredTestHasExternalRefForTestV0(strictEditorialQA, "required_evidence", "strict_editorial_qa_report") ||
		!stringInRequiredTestRefsForTestV0(strictEditorialQA.AcceptanceCriteriaRefs, "opes-required-strict_editorial_qa_pass") ||
		!stringInRequiredTestRefsForTestV0(strictEditorialQA.EvidenceRefs, "opes-final-evidence:strict_editorial_qa_pass") {
		t.Fatalf("strict_editorial_qa_pass_required_test=%+v", strictEditorialQA)
	}
	visualReuse := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-visual-reuse-manifest-job-ref-policy-opes-final",
	)
	if visualReuse.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(visualReuse, "required_evidence", "visual_reuse_manifest") ||
		!stringInRequiredTestRefsForTestV0(visualReuse.AcceptanceCriteriaRefs, "opes-required-visual-reuse-manifest") ||
		!stringInRequiredTestRefsForTestV0(visualReuse.EvidenceRefs, "opes-final-evidence:visual_reuse") {
		t.Fatalf("visual_reuse_required_test=%+v", visualReuse)
	}
	criteria := strings.Join(finalManifest.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"manifest_cierre.json",
		"checksum_refs",
		"validation_report_ref",
		"review_matrix_ref",
		"HTML",
		"RAG",
		"audio",
		"tests",
		"visual",
		"qa_passes",
		"qa_report_refs",
		"extension_pass",
		"official_text_qa_pass",
		"strict_editorial_qa_pass",
		"rag/corpus/chunks.jsonl",
		"source_variant",
	} {
		if !strings.Contains(criteria, want) {
			t.Fatalf("final_manifest_criteria=%q falta %s", criteria, want)
		}
	}
	visualCriteria := strings.Join(visualReuse.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"rebuild HTML",
		"html_final/html_ampliado",
		"tema_*.html",
	} {
		if !strings.Contains(visualCriteria, want) {
			t.Fatalf("visual_reuse_criteria=%q falta %s", visualCriteria, want)
		}
	}
	strictCriteria := strings.Join(strictEditorialQA.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"andamiaje interno",
		"contaminacion cruzada",
		"pendiente_rework_editorial",
		"needs_remove_study_scaffolding_and_cross_topic_contamination",
		"nunca ready",
	} {
		if !strings.Contains(strictCriteria, want) {
			t.Fatalf("strict_editorial_qa_criteria=%q falta %s", strictCriteria, want)
		}
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
