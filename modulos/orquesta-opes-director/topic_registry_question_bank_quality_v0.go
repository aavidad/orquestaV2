package orquestaopesdirector

import (
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const topicRegistryQuestionBankQualityNeedsReworkRefV0 = "question-bank-quality-needs-rework"

func topicRegistryQuestionBankQualityResultForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESQuestionBankQualityContractResultV0, bool) {
	request, ok := topicRegistryQuestionBankQualityRequestForRecordV0(record)
	if !ok {
		return OPESQuestionBankQualityContractResultV0{}, false
	}
	return ValidateOPESQuestionBankQualityContractV0(request), true
}

func topicRegistryQuestionBankQualityRequestForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESQuestionBankQualityContractRequestV0, bool) {
	if !topicRegistryRecordLooksQuestionBankV0(record) {
		return OPESQuestionBankQualityContractRequestV0{}, false
	}
	fields := record.PayloadFields
	questions := topicRegistryQuestionBankQuestionsForFieldsV0(fields)
	request := OPESQuestionBankQualityContractRequestV0{
		TopicRef: fieldStringV0(fields, "topic_id", "topic_ref"),
		QuestionCount: fieldIntV0(
			fields,
			"question_count",
			"questions_count",
			"total_questions",
			"question_bank_question_count",
		),
		Questions: questions,
		StructuralReportStatus: firstNonEmptyV0(
			fieldStringV0(fields, "question_bank_structural_status"),
			fieldStringV0(fields, "structural_report_status"),
			fieldStringV0(fields, "question_bank_quality_status"),
		),
		DifficultyReportStatus: firstNonEmptyV0(
			fieldStringV0(fields, "question_bank_difficulty_status"),
			fieldStringV0(fields, "difficulty_report_status"),
			fieldStringV0(fields, "distractor_proximity_status"),
		),
		ThreeModelReviewStatus: firstNonEmptyV0(
			fieldStringV0(fields, "question_bank_three_model_review_status"),
			fieldStringV0(fields, "three_model_review_status"),
			fieldStringV0(fields, "review_100_status"),
		),
		QuestionBankStatus: firstNonEmptyV0(
			fieldStringV0(fields, "question_bank_status"),
			fieldStringV0(fields, "question_bank_quality_status"),
			fieldStringV0(fields, "qa_status"),
		),
		EvidenceRefs: compactStringsV0(append(
			append(append([]string(nil), record.EvidenceRefs...), record.PayloadRefs...),
			fieldStringsV0(
				fields,
				"evidence_refs",
				"validation_refs",
				"required_test_evidence_refs",
				"qa_report_refs",
				"question_bank_evidence_refs",
				"question_bank_validation_refs",
			)...,
		)),
	}
	if request.QuestionCount <= 0 && len(questions) > 0 {
		request.QuestionCount = len(questions)
	}
	return request, true
}

func topicRegistryRecordLooksQuestionBankV0(record OPESCausalArtifactRecordV0) bool {
	if record.ArtifactType == orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0 {
		return true
	}
	switch strings.TrimSpace(fieldStringV0(record.PayloadFields, "source_work_kind", "work_kind")) {
	case "generate_question_bank", "generate_topic_tests", "create_topic_tests":
		return true
	default:
		return false
	}
}

func topicRegistryQuestionBankQualityPendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	if len(topicRegistryRequiredEvidencePendingRefsForRecordV0(record)) > 0 &&
		topicRegistryRequiredEvidenceShouldReworkV0(record) {
		return nil
	}
	result, ok := topicRegistryQuestionBankQualityResultForRecordV0(record)
	if !ok || result.Status != OPESQuestionBankQualityStatusNeedsReworkV0 {
		return nil
	}
	refs := []string{topicRegistryQuestionBankQualityNeedsReworkRefV0}
	for _, issue := range result.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, "question-bank-quality-"+safeRefV0(code))
		}
	}
	return compactStringsV0(refs)
}

func topicRegistryQuestionBankQualityFieldsForRecordV0(record OPESCausalArtifactRecordV0) []orquestadomainwork.DomainWorkFieldV0 {
	result, ok := topicRegistryQuestionBankQualityResultForRecordV0(record)
	if !ok {
		return nil
	}
	issueRefs := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			issueRefs = append(issueRefs, code)
		}
	}
	return []orquestadomainwork.DomainWorkFieldV0{
		{Name: "question_bank_quality_status", Value: result.Status},
		{Name: "question_bank_question_count", Value: strconv.Itoa(result.QuestionCount)},
		{Name: "question_bank_quality_issue_refs", Values: compactStringsV0(issueRefs)},
		{Name: "question_bank_quality_evidence_refs", Values: compactStringsV0(result.EvidenceRefs)},
	}
}
