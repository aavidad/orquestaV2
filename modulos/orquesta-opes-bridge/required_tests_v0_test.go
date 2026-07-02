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
	if !stringInRequiredTestRefsForTestV0(
		requiredTestRefsForTestV0(tests),
		"opes-domain-test-assemble_topic-job-ref-assemble-001",
	) ||
		!stringInRequiredTestRefsForTestV0(
			requiredTestRefsForTestV0(tests),
			"opes-topic-text-publicable-job-ref-assemble-001",
		) ||
		requiredTestByRefForTestV0(tests, "opes-domain-test-assemble_topic-job-ref-assemble-001").ExternalRefs[2].Ref != "assembled_topic" {
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
	if err != nil ||
		!stringInRequiredTestRefsForTestV0(
			requiredTestRefsForTestV0(plan.RequiredTests),
			"opes-domain-test-draft_content_block-job-ref-policy-opes",
		) ||
		!stringInRequiredTestRefsForTestV0(
			requiredTestRefsForTestV0(plan.RequiredTests),
			"opes-topic-text-publicable-job-ref-policy-opes",
		) {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestOPESRequiredTestPolicyV0SecuenciaCompletaTieneValidadoresEspecificos(t *testing.T) {
	expectedSpecific := map[string]string{
		"plan_temario":                  "document-plan-contract",
		"update_topic_registry":         "topic-registry-update",
		"research_exam_precedents":      "source-research-traceable",
		"draft_content_block":           "topic-text-publicable",
		"generate_visual_asset":         "didactic-visual-publicable",
		"generate_question_bank":        "question-bank-publicable",
		"review_legal":                  "review-report-actionable",
		"review_pedagogical":            "review-report-actionable",
		"review_quality":                "review-report-actionable",
		"review_codex":                  "review-report-actionable",
		"review_gemini":                 "review-report-actionable",
		"review_claude":                 "review-report-actionable",
		"review_pair_codex_gemini":      "review-report-actionable",
		"review_pair_codex_claude":      "review-report-actionable",
		"review_pair_gemini_claude":     "review-report-actionable",
		"review_director_consolidation": "review-report-actionable",
		"validate_topic":                "review-report-actionable",
		"assemble_topic":                "topic-text-publicable",
		"generate_audio_asset":          "audio-tts-resumable",
		"generate_tutor_assets":         "tutor-assets-publicable",
		"generate_learning_games":       "interactive-practice-publicable",
		"visual_asset_reuse":            "visual-reuse-manifest",
		"generate_html_site":            "html-site-publicable",
		"generate_help_manual_assets":   "help-manual-publicable",
		"finalize_temario_package":      "final-package-manifest",
	}
	for _, workKind := range OPESFullTemarioJobTypeSequenceV0() {
		t.Run(workKind, func(t *testing.T) {
			jobRef := "job-ref-sequence-" + workKind
			plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
				context.Background(),
				orquestadomainwork.DomainWorkJobRequestV0{
					CorrelationID:  "corr-policy-opes-sequence-" + workKind,
					IdempotencyKey: "idem-policy-opes-sequence-" + workKind,
					RequestedBy:    "orquesta",
					DomainRef:      "opes",
					WorkKind:       workKind,
					WorkRefs:       []string{jobRef},
					Objective:      "validar secuencia OPES completa",
				},
			)
			if err != nil {
				t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
			}
			safeJob := compactOPESBridgeRefV0(jobRef)
			baseRef := "opes-domain-test-" + compactOPESBridgeRefV0(workKind) + "-" + safeJob
			specificName, ok := expectedSpecific[workKind]
			if !ok {
				t.Fatalf("work_kind sin required-test especifico declarado: %s", workKind)
			}
			specificRef := "opes-" + compactOPESBridgeRefV0(specificName) + "-" + safeJob
			refs := requiredTestRefsForTestV0(plan.RequiredTests)
			if !stringInRequiredTestRefsForTestV0(refs, baseRef) ||
				!stringInRequiredTestRefsForTestV0(refs, specificRef) {
				t.Fatalf("work_kind=%s required_tests=%v falta base=%s specific=%s", workKind, refs, baseRef, specificRef)
			}
		})
	}
}

func TestOPESRequiredTestPolicyV0TextoTemaExigeCalidadPublicableEInventarioParcial(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-topic-text",
			IdempotencyKey: "idem-policy-opes-topic-text",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "assemble_topic",
			WorkRefs:       []string{"job-ref-policy-opes-topic-text"},
			Objective:      "ensamblar texto publicable OPES",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	topicText := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-topic-text-publicable-job-ref-policy-opes-topic-text",
	)
	if topicText.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(topicText, "required_test_name", "topic-text-publicable") ||
		!requiredTestHasExternalRefForTestV0(topicText, "required_evidence", "topic_quality_contract_result") ||
		!stringInRequiredTestRefsForTestV0(topicText.AcceptanceCriteriaRefs, "opes-required-topic-text-publicable") ||
		!stringInRequiredTestRefsForTestV0(topicText.EvidenceRefs, "opes-final-evidence:topic_quality_contract_pass") {
		t.Fatalf("topic_text_required_test=%+v", topicText)
	}
	criteria := strings.Join(topicText.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"OPESTopicQualityContractV0",
		"contador canonico",
		"anclas visibles {#...}",
		"tablas colapsadas",
		"mojibake",
		"metacomentarios",
		"invalid_artifact_paths",
		"valid_artifact_paths",
		"followup/rework causal",
	} {
		if !strings.Contains(criteria, want) {
			t.Fatalf("topic_text_criteria=%q falta %s", criteria, want)
		}
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
