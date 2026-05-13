package orquestadomainwork

import (
	"encoding/json"
	"strings"
)

func ValidateDomainWorkJobRequestV0(
	request DomainWorkJobRequestV0,
) []DomainWorkIssueV0 {
	request = NormalizeDomainWorkJobRequestV0(request)
	var issues []DomainWorkIssueV0
	issues = append(issues, requiredDomainWorkRefV0(
		request.DomainRef,
		"domain_ref",
		ErrDomainWorkDomainRefRequiredV0,
		ErrDomainWorkDomainRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkRefV0(
		request.WorkKind,
		"work_kind",
		ErrDomainWorkWorkKindRequiredV0,
		ErrDomainWorkWorkKindInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		request.Objective,
		"objective",
		ErrDomainWorkObjectiveRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		request.CorrelationID,
		"correlation_id",
		ErrDomainWorkCorrelationRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		request.IdempotencyKey,
		"idempotency_key",
		ErrDomainWorkIdempotencyRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		request.RequestedBy,
		"requested_by",
		ErrDomainWorkRequestedByRequiredV0,
	)...)
	issues = append(issues, validateDomainWorkRefsV0(request.InterfaceRefs, "interface_refs")...)
	issues = append(issues, validateDomainWorkRefsV0(request.WorkRefs, "work_refs")...)
	issues = append(issues, validateDomainWorkFieldsV0(request.InputFields, "input_fields")...)
	issues = append(issues, validateDomainWorkRefsV0(request.InputRefs, "input_refs")...)
	issues = append(issues, validateDomainWorkRefsV0(request.EvidenceRefs, "evidence_refs")...)
	issues = append(issues, validateDomainWorkExternalRefsV0(request.ExternalRefs)...)
	return issues
}

func ValidateDomainWorkArtifactSubmissionV0(
	submission DomainWorkArtifactSubmissionV0,
) []DomainWorkIssueV0 {
	submission = NormalizeDomainWorkArtifactSubmissionV0(submission)
	var issues []DomainWorkIssueV0
	issues = append(issues, requiredDomainWorkRefV0(
		submission.DomainRef,
		"domain_ref",
		ErrDomainWorkDomainRefRequiredV0,
		ErrDomainWorkDomainRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkRefV0(
		submission.JobRef,
		"job_ref",
		ErrDomainWorkJobRefRequiredV0,
		ErrDomainWorkRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkRefV0(
		submission.ArtifactRef,
		"artifact_ref",
		ErrDomainWorkArtifactRefRequiredV0,
		ErrDomainWorkRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkRefV0(
		submission.ArtifactType,
		"artifact_type",
		ErrDomainWorkArtifactTypeRequiredV0,
		ErrDomainWorkRefInvalidV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		submission.CorrelationID,
		"correlation_id",
		ErrDomainWorkCorrelationRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		submission.IdempotencyKey,
		"idempotency_key",
		ErrDomainWorkIdempotencyRequiredV0,
	)...)
	issues = append(issues, requiredDomainWorkTextV0(
		submission.RequestedBy,
		"requested_by",
		ErrDomainWorkRequestedByRequiredV0,
	)...)
	issues = append(issues, validateDomainWorkFieldsV0(submission.PayloadFields, "payload_fields")...)
	issues = append(issues, validateDomainWorkRefsV0(submission.PayloadRefs, "payload_refs")...)
	issues = append(issues, validateDomainWorkRefsV0(submission.EvidenceRefs, "evidence_refs")...)
	issues = append(issues, validateDomainWorkExternalRefsV0(submission.ExternalRefs)...)
	return issues
}

func requiredDomainWorkRefV0(
	value string,
	field string,
	requiredCode string,
	invalidCode string,
) []DomainWorkIssueV0 {
	if strings.TrimSpace(value) == "" {
		return []DomainWorkIssueV0{{Code: requiredCode, Field: field}}
	}
	if !isCompactDomainWorkRefV0(value) {
		return []DomainWorkIssueV0{{Code: invalidCode, Field: field}}
	}
	return nil
}

func requiredDomainWorkTextV0(
	value string,
	field string,
	code string,
) []DomainWorkIssueV0 {
	if strings.TrimSpace(value) == "" {
		return []DomainWorkIssueV0{{Code: code, Field: field}}
	}
	return nil
}

func validateDomainWorkRefsV0(values []string, field string) []DomainWorkIssueV0 {
	for _, value := range values {
		if !isCompactDomainWorkRefV0(value) {
			return []DomainWorkIssueV0{{Code: ErrDomainWorkRefInvalidV0, Field: field}}
		}
	}
	return nil
}

func validateDomainWorkExternalRefsV0(values []DomainWorkExternalRefV0) []DomainWorkIssueV0 {
	for _, value := range values {
		if !isCompactDomainWorkRefV0(value.Kind) || !isCompactDomainWorkRefV0(value.Ref) {
			return []DomainWorkIssueV0{{Code: ErrDomainWorkRefInvalidV0, Field: "external_refs"}}
		}
	}
	return nil
}

func validateDomainWorkFieldsV0(values []DomainWorkFieldV0, field string) []DomainWorkIssueV0 {
	for _, value := range values {
		if !isCompactDomainWorkRefV0(value.Name) {
			return []DomainWorkIssueV0{{Code: ErrDomainWorkFieldNameInvalidV0, Field: field}}
		}
		if len(value.ValueJSON) > 0 && !json.Valid(value.ValueJSON) {
			return []DomainWorkIssueV0{{Code: ErrDomainWorkFieldJSONInvalidV0, Field: field}}
		}
	}
	return nil
}

func isCompactDomainWorkRefV0(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	return !strings.ContainsAny(trimmed, " /\\\t\n\r")
}
