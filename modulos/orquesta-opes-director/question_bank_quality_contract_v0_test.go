package orquestaopesdirector

import "testing"

func TestValidateOPESQuestionBankQualityContractV0BloqueaMenosDe50Preguntas(t *testing.T) {
	result := ValidateOPESQuestionBankQualityContractV0(OPESQuestionBankQualityContractRequestV0{
		TopicRef:               "tema-001",
		QuestionCount:          49,
		StructuralReportStatus: "pass",
		DifficultyReportStatus: "pass",
		ThreeModelReviewStatus: "pass",
		EvidenceRefs:           []string{"opes-final-evidence:question_bank_publicable"},
	})
	if result.Status != OPESQuestionBankQualityStatusNeedsReworkV0 ||
		!questionBankIssueCodeForTestV0(result.Issues, ErrOPESQuestionBankQuestionCountBelowMinV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESQuestionBankQualityContractV0BloqueaOpcionesIncompletasORespuestaAmbigua(t *testing.T) {
	result := ValidateOPESQuestionBankQualityContractV0(OPESQuestionBankQualityContractRequestV0{
		TopicRef:               "tema-002",
		StructuralReportStatus: "pass",
		DifficultyReportStatus: "pass",
		ThreeModelReviewStatus: "pass",
		Questions: []OPESQuestionBankQuestionV0{{
			Stem: "Pregunta de prueba",
			Options: []OPESQuestionBankOptionV0{
				{Text: "Opcion A", Correct: true},
				{Text: "Opcion B", Correct: true},
			},
			Explanation: "Explicacion tutor",
		}},
	})
	if result.Status != OPESQuestionBankQualityStatusNeedsReworkV0 ||
		!questionBankIssueCodeForTestV0(result.Issues, ErrOPESQuestionBankQuestionCountBelowMinV0) ||
		!questionBankIssueCodeForTestV0(result.Issues, ErrOPESQuestionBankOptionsInvalidV0) ||
		!questionBankIssueCodeForTestV0(result.Issues, ErrOPESQuestionBankCorrectAnswerInvalidV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESQuestionBankQualityContractV0AceptaBancoPublicable(t *testing.T) {
	result := ValidateOPESQuestionBankQualityContractV0(OPESQuestionBankQualityContractRequestV0{
		TopicRef:  "tema-003",
		Questions: validQuestionBankQuestionsForDirectorTestV0(50),
		EvidenceRefs: []string{
			"question_bank_structural_report",
			"question_bank_difficulty_report",
			"question_bank_three_model_review",
		},
	})
	if result.Status != OPESQuestionBankQualityStatusCompleteV0 || len(result.Issues) > 0 {
		t.Fatalf("result=%+v", result)
	}
}

func validQuestionBankQuestionsForDirectorTestV0(count int) []OPESQuestionBankQuestionV0 {
	questions := make([]OPESQuestionBankQuestionV0, 0, count)
	for i := 0; i < count; i++ {
		questions = append(questions, OPESQuestionBankQuestionV0{
			QuestionRef: "q",
			Stem:        "Pregunta publicable sobre contenido canonico del tema",
			Options: []OPESQuestionBankOptionV0{
				{Ref: "A", Text: "Respuesta correcta desarrollada"},
				{Ref: "B", Text: "Distractor plausible relacionado"},
				{Ref: "C", Text: "Distractor parcial razonable"},
				{Ref: "D", Text: "Distractor tecnico alternativo"},
			},
			CorrectAnswerRefs: []string{"A"},
			Explanation:       "Explicacion tutor para aciertos y fallos",
		})
	}
	return questions
}

func questionBankIssueCodeForTestV0(issues []OPESQuestionBankQualityIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
