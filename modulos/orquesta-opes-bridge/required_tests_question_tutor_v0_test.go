package orquestaopesbridge

import (
	"context"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestOPESRequiredTestPolicyV0QuestionBankExigeQATripleYExplicacionesTutor(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-question-bank",
			IdempotencyKey: "idem-policy-opes-question-bank",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "generate_question_bank",
			WorkRefs:       []string{"job-ref-policy-opes-question-bank"},
			Objective:      "generar banco de tests OPES",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	questionBank := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-question-bank-publicable-job-ref-policy-opes-question-bank",
	)
	if questionBank.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(questionBank, "required_test_name", "question_bank_publicable") ||
		!requiredTestHasExternalRefForTestV0(questionBank, "required_evidence", "question_bank_three_model_review") ||
		!stringInRequiredTestRefsForTestV0(questionBank.AcceptanceCriteriaRefs, "opes-required-question-bank-publicable") ||
		!stringInRequiredTestRefsForTestV0(questionBank.EvidenceRefs, "opes-final-evidence:question_bank_publicable") {
		t.Fatalf("question_bank_required_test=%+v", questionBank)
	}
	criteria := strings.Join(questionBank.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"al menos 50 preguntas",
		"4 opciones A/B/C/D",
		"una unica respuesta correcta exacta",
		"distractores plausibles",
		"explicacion tutor",
		"revision 100% Codex/Gemini/Claude",
		"revision parcial no cierra",
		"pendiente_continuar",
	} {
		if !strings.Contains(criteria, want) {
			t.Fatalf("question_bank_criteria=%q falta %s", criteria, want)
		}
	}
}

func TestOPESRequiredTestPolicyV0TutorAssetsExigeFuentesCanonicasYRAGReconstruido(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-tutor",
			IdempotencyKey: "idem-policy-opes-tutor",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "generate_tutor_assets",
			WorkRefs:       []string{"job-ref-policy-opes-tutor"},
			Objective:      "generar tutor OPES",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	tutor := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-tutor-assets-publicable-job-ref-policy-opes-tutor",
	)
	if tutor.TestRef == "" ||
		!requiredTestHasExternalRefForTestV0(tutor, "required_test_name", "tutor_assets_publicable") ||
		!requiredTestHasExternalRefForTestV0(tutor, "required_evidence", "tutor_qa_report") ||
		!requiredTestHasExternalRefForTestV0(tutor, "required_evidence", "rag_corpus_manifest") ||
		!stringInRequiredTestRefsForTestV0(tutor.AcceptanceCriteriaRefs, "opes-required-tutor-assets-publicable") ||
		!stringInRequiredTestRefsForTestV0(tutor.EvidenceRefs, "opes-final-evidence:tutor_assets_publicable") {
		t.Fatalf("tutor_required_test=%+v", tutor)
	}
	criteria := strings.Join(tutor.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"fuentes canonicas aprobadas",
		"rag/corpus/chunks.jsonl",
		"rag/corpus/summary.json",
		"rag/manifest.json",
		"HTML final/local, tests y tutor fuente limpios",
		"explicacion de fallos de test",
		"ausencia de invencion fuera de fuentes",
		"pendiente_continuar",
	} {
		if !strings.Contains(criteria, want) {
			t.Fatalf("tutor_criteria=%q falta %s", criteria, want)
		}
	}
}

func TestOPESRequiredTestPolicyV0FinalTemarioIncluyeQATestsYTutor(t *testing.T) {
	plan, err := (OPESRequiredTestPolicyV0{}).BuildDomainWorkRequiredTestPlanV0(
		context.Background(),
		orquestadomainwork.DomainWorkJobRequestV0{
			CorrelationID:  "corr-policy-opes-final-tests-tutor",
			IdempotencyKey: "idem-policy-opes-final-tests-tutor",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "finalize_temario_package",
			WorkRefs:       []string{"job-ref-policy-opes-final-tests-tutor"},
			Objective:      "cerrar temario OPES con QA tests y tutor",
		},
	)
	if err != nil {
		t.Fatalf("BuildDomainWorkRequiredTestPlanV0: %v", err)
	}
	for _, want := range []string{
		"opes-question-bank-publicable-job-ref-policy-opes-final-tests-tutor",
		"opes-tutor-assets-publicable-job-ref-policy-opes-final-tests-tutor",
	} {
		if !stringInRequiredTestRefsForTestV0(requiredTestRefsForTestV0(plan.RequiredTests), want) {
			t.Fatalf("required_tests=%v falta %s", requiredTestRefsForTestV0(plan.RequiredTests), want)
		}
	}
	finalManifest := requiredTestByRefForTestV0(
		plan.RequiredTests,
		"opes-final-package-manifest-job-ref-policy-opes-final-tests-tutor",
	)
	if finalManifest.TestRef == "" ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:question_bank_publicable") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:tutor_assets_publicable") ||
		!stringInRequiredTestRefsForTestV0(finalManifest.EvidenceRefs, "opes-final-evidence:tutor") {
		t.Fatalf("final_manifest_required_test=%+v", finalManifest)
	}
	criteria := strings.Join(finalManifest.AcceptanceCriteria, "\n")
	for _, want := range []string{
		"tests",
		"tutor",
		"question_bank_publicable",
		"tutor_assets_publicable",
	} {
		if !strings.Contains(criteria, want) {
			t.Fatalf("final_manifest_criteria=%q falta %s", criteria, want)
		}
	}
}
