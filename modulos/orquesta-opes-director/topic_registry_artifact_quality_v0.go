package orquestaopesdirector

import (
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const topicRegistryArtifactQualityNeedsReworkRefV0 = "artifact-quality-needs-rework"

func topicRegistryArtifactQualityResultForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESArtifactQualityContractResultV0, bool) {
	request, ok := topicRegistryArtifactQualityRequestForRecordV0(record)
	if !ok {
		return OPESArtifactQualityContractResultV0{}, false
	}
	return ValidateOPESArtifactQualityContractV0(request), true
}

func topicRegistryArtifactQualityRequestForRecordV0(
	record OPESCausalArtifactRecordV0,
) (OPESArtifactQualityContractRequestV0, bool) {
	workKind := strings.TrimSpace(fieldStringV0(record.PayloadFields, "source_work_kind", "work_kind"))
	artifactType := topicRegistryEffectiveArtifactTypeForWorkKindV0(workKind, record.ArtifactType)
	if len(opesArtifactQualityRequirementsV0(artifactType)) == 0 {
		return OPESArtifactQualityContractRequestV0{}, false
	}
	return OPESArtifactQualityContractRequestV0{
		WorkKind:     workKind,
		ArtifactType: artifactType,
		TopicRef:     fieldStringV0(record.PayloadFields, "topic_id", "topic_ref"),
		Fields:       record.PayloadFields,
		EvidenceRefs: topicRegistryArtifactQualityEvidenceRefsForRecordV0(record),
	}, true
}

func topicRegistryArtifactQualityPendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	if len(topicRegistryRequiredEvidencePendingRefsForRecordV0(record)) > 0 &&
		topicRegistryRequiredEvidenceShouldReworkV0(record) {
		return nil
	}
	result, ok := topicRegistryArtifactQualityResultForRecordV0(record)
	if !ok || result.Status != OPESArtifactQualityStatusNeedsReworkV0 || !topicRegistryArtifactQualityShouldReworkV0(record, result) {
		return nil
	}
	refs := []string{topicRegistryArtifactQualityNeedsReworkRefV0}
	for _, issue := range result.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			refs = append(refs, "artifact-quality-"+safeRefV0(code))
		}
	}
	return compactStringsV0(refs)
}

func topicRegistryArtifactQualityShouldReworkV0(
	record OPESCausalArtifactRecordV0,
	result OPESArtifactQualityContractResultV0,
) bool {
	status := firstNonEmptyV0(
		fieldStringV0(record.PayloadFields, "operational_status"),
		fieldStringV0(record.PayloadFields, "status"),
		fieldStringV0(record.PayloadFields, "estado"),
		fieldStringV0(record.PayloadFields, "decision"),
	)
	if topicRegistryExplicitPartialStatusV0(status) {
		return false
	}
	if record.CompleteJob {
		return true
	}
	if explicit, ok := topicRegistryExplicitOperationalStatusV0(status); ok && explicit == "complete" {
		return true
	}
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "ready", "listo", "lista", "listo_para_revision_operador", "html_validado":
		return true
	}
	for _, ref := range result.EvidenceRefs {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(ref)), "opes-final-evidence:") {
			return true
		}
	}
	return false
}

func topicRegistryArtifactQualityFieldsForRecordV0(record OPESCausalArtifactRecordV0) []orquestadomainwork.DomainWorkFieldV0 {
	result, ok := topicRegistryArtifactQualityResultForRecordV0(record)
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
		{Name: "artifact_quality_schema", Value: result.SchemaVersion},
		{Name: "artifact_quality_status", Value: result.Status},
		{Name: "artifact_quality_work_kind", Value: result.WorkKind},
		{Name: "artifact_quality_artifact_type", Value: result.ArtifactType},
		{Name: "artifact_quality_issue_count", Value: strconv.Itoa(len(result.Issues))},
		{Name: "artifact_quality_issue_refs", Values: compactStringsV0(issueRefs)},
		{Name: "artifact_quality_evidence_refs", Values: compactStringsV0(result.EvidenceRefs)},
	}
}

func topicRegistryArtifactQualityEvidenceRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := append([]string(nil), record.EvidenceRefs...)
	refs = append(refs, record.PayloadRefs...)
	refs = append(refs, fieldStringsV0(
		record.PayloadFields,
		"evidence_refs",
		"validation_refs",
		"required_test_evidence_refs",
		"qa_report_refs",
		"artifact_quality_evidence_refs",
		"review_evidence_refs",
		"source_refs",
	)...)
	return compactStringsV0(refs)
}
