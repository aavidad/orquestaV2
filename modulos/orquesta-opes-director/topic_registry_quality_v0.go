package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const topicRegistryQualityNeedsReworkRefV0 = "topic-quality-needs-rework"

func topicRegistryQualityResultForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESTopicQualityContractResultV0, bool) {
	request, ok := topicRegistryQualityRequestForRecordV0(record)
	if !ok {
		return OPESTopicQualityContractResultV0{}, false
	}
	return ValidateOPESTopicQualityContractV0(request), true
}

func topicRegistryQualityRequestForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESTopicQualityContractRequestV0, bool) {
	fields := record.PayloadFields
	request := OPESTopicQualityContractRequestV0{
		TopicRef: fieldStringV0(fields, "topic_id", "topic_ref"),
		Level: firstNonEmptyV0(
			fieldStringV0(fields, "level"),
			fieldStringV0(fields, "topic_level"),
			fieldStringV0(fields, "quality_level"),
			OPESTopicQualityLevelBV0,
		),
		Text: firstNonEmptyV0(
			fieldStringV0(fields, "topic_text"),
			fieldStringV0(fields, "public_text"),
			fieldStringV0(fields, "markdown"),
			fieldStringV0(fields, "text"),
			fieldStringV0(fields, "content"),
			fieldStringV0(fields, "tema_ampliado"),
		),
		CanonicalWordCount: fieldIntV0(
			fields,
			"canonical_word_count",
			"canonical_words",
			"strict_word_count",
			"strict_public_word_count",
		),
		CanonicalWordCountText: fieldStringV0(fields, "canonical_word_count_text", "strict_word_count_text"),
		WordCount:              fieldIntV0(fields, "word_count", "words", "declared_word_count"),
		WordCountText:          fieldStringV0(fields, "word_count_text", "wc_output"),
		WordCountSource:        fieldStringV0(fields, "word_count_source", "word_counter"),
		ReportStatus: firstNonEmptyV0(
			fieldStringV0(fields, "topic_quality_status"),
			fieldStringV0(fields, "quality_status"),
			fieldStringV0(fields, "qa_status"),
			fieldStringV0(fields, "strict_editorial_qa_status"),
		),
		ReportText: firstNonEmptyV0(
			fieldStringV0(fields, "topic_quality_report"),
			fieldStringV0(fields, "quality_report"),
			fieldStringV0(fields, "qa_report"),
			fieldStringV0(fields, "strict_editorial_qa_report"),
		),
		RequireDidacticVisual: topicRegistryQualityBoolFieldV0(fields, "require_didactic_visual", "visual_required"),
		EvidenceRefs: compactStringsV0(append(
			append([]string(nil), record.EvidenceRefs...),
			fieldStringsV0(fields, "topic_quality_evidence_refs", "qa_report_refs", "validation_refs")...,
		)),
	}
	if !topicRegistryRecordDeclaresTopicQualityV0(record, request) {
		return OPESTopicQualityContractRequestV0{}, false
	}
	return request, true
}

func topicRegistryRecordDeclaresTopicQualityV0(
	record OPESCausalArtifactRecordV0,
	request OPESTopicQualityContractRequestV0,
) bool {
	if strings.TrimSpace(request.Text) != "" ||
		request.CanonicalWordCount > 0 ||
		strings.TrimSpace(request.CanonicalWordCountText) != "" ||
		request.WordCount > 0 ||
		strings.TrimSpace(request.WordCountText) != "" ||
		strings.TrimSpace(request.ReportStatus) != "" ||
		strings.TrimSpace(request.ReportText) != "" {
		return true
	}
	values := append([]string(nil), record.EvidenceRefs...)
	values = append(values, record.PayloadRefs...)
	values = append(values, fieldStringsV0(record.PayloadFields, "topic_quality_evidence_refs", "qa_report_refs", "validation_refs")...)
	for _, value := range values {
		if topicRegistryQualityRefLooksExplicitV0(value) {
			return true
		}
	}
	return false
}

func topicRegistryQualityRefLooksExplicitV0(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	for _, token := range []string{
		"topic_quality",
		"strict_editorial",
		"official_text_qa",
		"extension_pass",
		"informe_texto_publico",
		"andamiaje_interno",
		"metacomentarios_examen",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func topicRegistryQualityPendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	result, ok := topicRegistryQualityResultForRecordV0(record)
	if !ok || result.Status != OPESTopicQualityStatusNeedsReworkV0 {
		return nil
	}
	refs := []string{topicRegistryQualityNeedsReworkRefV0}
	for _, issue := range result.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, "topic-quality-"+safeRefV0(code))
		}
	}
	return compactStringsV0(refs)
}

func topicRegistryQualityFieldsForRecordV0(record OPESCausalArtifactRecordV0) []orquestadomainwork.DomainWorkFieldV0 {
	result, ok := topicRegistryQualityResultForRecordV0(record)
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
		{Name: "topic_quality_status", Value: result.Status},
		{Name: "topic_quality_issue_refs", Values: compactStringsV0(issueRefs)},
		{Name: "topic_quality_evidence_refs", Values: compactStringsV0(result.EvidenceRefs)},
	}
}

func topicRegistryQualityBoolFieldV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) bool {
	value := strings.ToLower(fieldStringV0(fields, names...))
	switch value {
	case "true", "1", "yes", "si", "required", "obligatorio":
		return true
	default:
		return false
	}
}
