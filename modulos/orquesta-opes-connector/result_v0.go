package orquestaopesconnector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func domainWorkJobFromOPESV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	response opesJobResponseV0,
) orquestadomainwork.DomainWorkJobV0 {
	if response.ID == "" {
		return invalidDomainWorkJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrOPESJobRefMissingV0,
			Field: "id",
		}})
	}
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:         response.ID,
		DomainRef:      request.DomainRef,
		WorkKind:       firstNonEmptyV0(request.WorkKind, response.Type),
		CorrelationID:  firstNonEmptyV0(response.CorrelationID, request.CorrelationID),
		IdempotencyKey: firstNonEmptyV0(response.IdempotencyKey, request.IdempotencyKey),
		ExternalRefs:   externalRefsFromMapV0(response.ExternalRefs, request.ExternalRefs),
		EvidenceRefs:   []string{"opes-job-ref-" + response.ID},
	}
}

func domainWorkArtifactReceiptFromOPESV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	response opesArtifactResponseV0,
) orquestadomainwork.DomainWorkArtifactReceiptV0 {
	if issues := validateOPESArtifactReceiptResponseV0(submission, response); len(issues) > 0 {
		return invalidDomainWorkArtifactReceiptV0(submission, issues)
	}
	return orquestadomainwork.DomainWorkArtifactReceiptV0{
		SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:         firstNonEmptyV0(response.JobID, submission.JobRef),
		ArtifactRef:    submission.ArtifactRef,
		ReceiptRef:     firstNonEmptyV0(response.ID, response.ArtifactID),
		CorrelationID:  firstNonEmptyV0(response.CorrelationID, submission.CorrelationID),
		IdempotencyKey: firstNonEmptyV0(response.IdempotencyKey, submission.IdempotencyKey),
		ExternalRefs:   externalRefsFromMapV0(response.ExternalRefs, submission.ExternalRefs),
		EvidenceRefs:   []string{"opes-artifact-ref-" + firstNonEmptyV0(response.ID, response.ArtifactID, submission.ArtifactRef)},
	}
}

func validateOPESArtifactReceiptResponseV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	response opesArtifactResponseV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	issues := []orquestadomainwork.DomainWorkIssueV0{}
	if firstNonEmptyV0(response.ID, response.ArtifactID) == "" {
		issues = append(issues, opesReceiptIssueV0(ErrOPESResponseInvalidV0, "artifact_id"))
	}
	if response.JobID != "" && response.JobID != submission.JobRef {
		issues = append(issues, opesReceiptIssueV0(ErrOPESResponseInvalidV0, "job_id"))
	}
	if response.Artifact.JobID != "" && response.Artifact.JobID != submission.JobRef {
		issues = append(issues, opesReceiptIssueV0(ErrOPESResponseInvalidV0, "artifact.job_id"))
	}
	if response.Artifact.Type != "" && response.Artifact.Type != submission.ArtifactType {
		issues = append(issues, opesReceiptIssueV0(ErrOPESResponseInvalidV0, "artifact.type"))
	}
	if submission.CompleteJob && response.Job.Status != "" && !opesJobStatusTerminalForCompletionV0(response.Job.Status) {
		issues = append(issues, opesReceiptIssueV0(ErrOPESResponseInvalidV0, "job.status"))
	}
	return issues
}

func opesJobStatusTerminalForCompletionV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "done", "settled":
		return true
	default:
		return false
	}
}

func opesReceiptIssueV0(code string, field string) orquestadomainwork.DomainWorkIssueV0 {
	return orquestadomainwork.DomainWorkIssueV0{Code: code, Field: field}
}

func invalidDomainWorkJobV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	issues []orquestadomainwork.DomainWorkIssueV0,
) orquestadomainwork.DomainWorkJobV0 {
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
		DomainRef:      request.DomainRef,
		WorkKind:       request.WorkKind,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
		Issues:         issues,
	}
}

func invalidDomainWorkArtifactReceiptV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	issues []orquestadomainwork.DomainWorkIssueV0,
) orquestadomainwork.DomainWorkArtifactReceiptV0 {
	return orquestadomainwork.DomainWorkArtifactReceiptV0{
		SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
		JobRef:         submission.JobRef,
		ArtifactRef:    submission.ArtifactRef,
		CorrelationID:  submission.CorrelationID,
		IdempotencyKey: submission.IdempotencyKey,
		Issues:         issues,
	}
}

func externalRefsFromMapV0(
	values map[string]string,
	fallback []orquestadomainwork.DomainWorkExternalRefV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	if len(values) == 0 {
		return append([]orquestadomainwork.DomainWorkExternalRefV0(nil), fallback...)
	}
	refs := make([]orquestadomainwork.DomainWorkExternalRefV0, 0, len(values))
	for kind, ref := range values {
		refs = append(refs, orquestadomainwork.DomainWorkExternalRefV0{Kind: kind, Ref: ref})
	}
	return refs
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
