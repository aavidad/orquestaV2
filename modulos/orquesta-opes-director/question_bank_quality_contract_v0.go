package orquestaopesdirector

import (
	"strconv"
	"strings"
)

const (
	OPESQuestionBankQualityContractSchemaV0 = "opes_question_bank_quality_contract.v0"

	OPESQuestionBankQualityStatusCompleteV0    = "complete"
	OPESQuestionBankQualityStatusNeedsReworkV0 = "needs_rework"

	OPESQuestionBankMinQuestionsV0        = 50
	OPESQuestionBankRequiredOptionsV0     = 4
	OPESQuestionBankRequiredCorrectsV0    = 1
	OPESQuestionBankMinExplanationRunesV0 = 8

	ErrOPESQuestionBankContractRequiredV0         = "opes_question_bank_contract_required"
	ErrOPESQuestionBankQuestionCountBelowMinV0    = "opes_question_bank_question_count_below_min"
	ErrOPESQuestionBankStemRequiredV0             = "opes_question_bank_stem_required"
	ErrOPESQuestionBankOptionsInvalidV0           = "opes_question_bank_options_invalid"
	ErrOPESQuestionBankCorrectAnswerInvalidV0     = "opes_question_bank_correct_answer_invalid"
	ErrOPESQuestionBankExplanationRequiredV0      = "opes_question_bank_explanation_required"
	ErrOPESQuestionBankStructuralReportRequiredV0 = "opes_question_bank_structural_report_required"
	ErrOPESQuestionBankDifficultyReportRequiredV0 = "opes_question_bank_difficulty_report_required"
	ErrOPESQuestionBankThreeModelReviewRequiredV0 = "opes_question_bank_three_model_review_required"
)

type OPESQuestionBankQualityContractRequestV0 struct {
	TopicRef               string
	QuestionCount          int
	Questions              []OPESQuestionBankQuestionV0
	StructuralReportStatus string
	DifficultyReportStatus string
	ThreeModelReviewStatus string
	QuestionBankStatus     string
	EvidenceRefs           []string
}

type OPESQuestionBankQuestionV0 struct {
	QuestionRef       string
	Stem              string
	Options           []OPESQuestionBankOptionV0
	CorrectAnswerRefs []string
	Explanation       string
}

type OPESQuestionBankOptionV0 struct {
	Ref     string
	Text    string
	Correct bool
}

type OPESQuestionBankQualityContractResultV0 struct {
	SchemaVersion string                           `json:"schema_version"`
	Status        string                           `json:"status"`
	TopicRef      string                           `json:"topic_ref,omitempty"`
	QuestionCount int                              `json:"question_count,omitempty"`
	MinQuestions  int                              `json:"min_questions,omitempty"`
	EvidenceRefs  []string                         `json:"evidence_refs,omitempty"`
	Issues        []OPESQuestionBankQualityIssueV0 `json:"issues,omitempty"`
}

type OPESQuestionBankQualityIssueV0 struct {
	Code         string   `json:"code"`
	Field        string   `json:"field,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func ValidateOPESQuestionBankQualityContractV0(
	request OPESQuestionBankQualityContractRequestV0,
) OPESQuestionBankQualityContractResultV0 {
	request = normalizeOPESQuestionBankQualityContractRequestV0(request)
	result := OPESQuestionBankQualityContractResultV0{
		SchemaVersion: OPESQuestionBankQualityContractSchemaV0,
		Status:        OPESQuestionBankQualityStatusCompleteV0,
		TopicRef:      request.TopicRef,
		QuestionCount: opesQuestionBankQuestionCountV0(request),
		MinQuestions:  OPESQuestionBankMinQuestionsV0,
		EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
	}
	if result.QuestionCount <= 0 {
		result.Issues = append(result.Issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankContractRequiredV0,
			"question_bank",
			"structured question bank contract is required",
		))
	}
	if result.QuestionCount > 0 && result.QuestionCount < OPESQuestionBankMinQuestionsV0 {
		result.Issues = append(result.Issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankQuestionCountBelowMinV0,
			"question_count",
			"OPES public question bank requires at least 50 questions per topic",
		))
	}
	for index, question := range request.Questions {
		result.Issues = append(result.Issues, opesQuestionBankQuestionIssuesV0(index, question)...)
	}
	if !opesQuestionBankReportPassesV0(
		request.StructuralReportStatus,
		result.EvidenceRefs,
		"question_bank_structural_report",
		"question_bank_quality_contract",
	) {
		result.Issues = append(result.Issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankStructuralReportRequiredV0,
			"question_bank_structural_report",
			"question bank needs structural validation evidence",
		))
	}
	if !opesQuestionBankReportPassesV0(
		request.DifficultyReportStatus,
		result.EvidenceRefs,
		"question_bank_difficulty_report",
		"difficulty_report",
		"proximity_report",
	) {
		result.Issues = append(result.Issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankDifficultyReportRequiredV0,
			"question_bank_difficulty_report",
			"question bank needs difficulty and distractor proximity validation",
		))
	}
	if !opesQuestionBankReportPassesV0(
		request.ThreeModelReviewStatus,
		result.EvidenceRefs,
		"question_bank_three_model_review",
		"three_model_review",
		"review_100_codex_gemini_claude",
	) {
		result.Issues = append(result.Issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankThreeModelReviewRequiredV0,
			"question_bank_three_model_review",
			"question bank needs 100% Codex/Gemini/Claude review evidence",
		))
	}
	if len(result.Issues) > 0 {
		result.Status = OPESQuestionBankQualityStatusNeedsReworkV0
	}
	return result
}

func normalizeOPESQuestionBankQualityContractRequestV0(
	request OPESQuestionBankQualityContractRequestV0,
) OPESQuestionBankQualityContractRequestV0 {
	request.TopicRef = strings.TrimSpace(request.TopicRef)
	request.StructuralReportStatus = strings.ToLower(strings.TrimSpace(request.StructuralReportStatus))
	request.DifficultyReportStatus = strings.ToLower(strings.TrimSpace(request.DifficultyReportStatus))
	request.ThreeModelReviewStatus = strings.ToLower(strings.TrimSpace(request.ThreeModelReviewStatus))
	request.QuestionBankStatus = strings.ToLower(strings.TrimSpace(request.QuestionBankStatus))
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	for index := range request.Questions {
		request.Questions[index].QuestionRef = strings.TrimSpace(request.Questions[index].QuestionRef)
		request.Questions[index].Stem = strings.TrimSpace(request.Questions[index].Stem)
		request.Questions[index].CorrectAnswerRefs = compactStringsV0(request.Questions[index].CorrectAnswerRefs)
		request.Questions[index].Explanation = strings.TrimSpace(request.Questions[index].Explanation)
		for optionIndex := range request.Questions[index].Options {
			request.Questions[index].Options[optionIndex].Ref = strings.TrimSpace(request.Questions[index].Options[optionIndex].Ref)
			request.Questions[index].Options[optionIndex].Text = strings.TrimSpace(request.Questions[index].Options[optionIndex].Text)
		}
		request.Questions[index].Options = opesQuestionBankCompactOptionsV0(request.Questions[index].Options)
	}
	return request
}

func opesQuestionBankQuestionCountV0(request OPESQuestionBankQualityContractRequestV0) int {
	if len(request.Questions) > 0 {
		return len(request.Questions)
	}
	if request.QuestionCount > 0 {
		return request.QuestionCount
	}
	return 0
}

func opesQuestionBankQuestionIssuesV0(
	index int,
	question OPESQuestionBankQuestionV0,
) []OPESQuestionBankQualityIssueV0 {
	fieldPrefix := "questions." + strconv.Itoa(index+1)
	var issues []OPESQuestionBankQualityIssueV0
	if strings.TrimSpace(question.Stem) == "" {
		issues = append(issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankStemRequiredV0,
			fieldPrefix+".stem",
			"question stem is required",
		))
	}
	if len(question.Options) != OPESQuestionBankRequiredOptionsV0 || opesQuestionBankHasBlankOrDuplicateOptionsV0(question.Options) {
		issues = append(issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankOptionsInvalidV0,
			fieldPrefix+".options",
			"question requires exactly four non-duplicate options",
		))
	}
	correctKeys, unresolved := opesQuestionBankCorrectAnswerKeysV0(question)
	if len(correctKeys) != OPESQuestionBankRequiredCorrectsV0 || len(unresolved) > 0 {
		issues = append(issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankCorrectAnswerInvalidV0,
			fieldPrefix+".correct_answer",
			"question requires exactly one resolvable correct answer",
		))
	}
	if len([]rune(strings.TrimSpace(question.Explanation))) < OPESQuestionBankMinExplanationRunesV0 {
		issues = append(issues, opesQuestionBankIssueV0(
			ErrOPESQuestionBankExplanationRequiredV0,
			fieldPrefix+".explanation",
			"question requires tutor explanation",
		))
	}
	return issues
}
