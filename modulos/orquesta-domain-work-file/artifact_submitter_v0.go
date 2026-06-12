package orquestadomainworkfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type domainWorkFileArtifactFingerprintInputV0 struct {
	SchemaVersion  string                                       `json:"schema_version"`
	CorrelationID  string                                       `json:"correlation_id"`
	IdempotencyKey string                                       `json:"idempotency_key"`
	RequestedBy    string                                       `json:"requested_by"`
	DomainRef      string                                       `json:"domain_ref"`
	JobRef         string                                       `json:"job_ref"`
	ArtifactRef    string                                       `json:"artifact_ref"`
	ArtifactType   string                                       `json:"artifact_type"`
	Summary        string                                       `json:"summary"`
	PayloadFields  []orquestadomainwork.DomainWorkFieldV0       `json:"payload_fields"`
	PayloadRefs    []string                                     `json:"payload_refs"`
	ExternalRefs   []orquestadomainwork.DomainWorkExternalRefV0 `json:"external_refs"`
	EvidenceRefs   []string                                     `json:"evidence_refs"`
	CompleteJob    bool                                         `json:"complete_job"`
}

func domainWorkFileArtifactSubmissionFingerprintV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) (string, error) {
	submission = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
	data, err := json.Marshal(domainWorkFileArtifactFingerprintInputV0{
		SchemaVersion:  submission.SchemaVersion,
		CorrelationID:  submission.CorrelationID,
		IdempotencyKey: submission.IdempotencyKey,
		RequestedBy:    submission.RequestedBy,
		DomainRef:      submission.DomainRef,
		JobRef:         submission.JobRef,
		ArtifactRef:    submission.ArtifactRef,
		ArtifactType:   submission.ArtifactType,
		Summary:        submission.Summary,
		PayloadFields:  cloneDomainWorkFileFieldsV0(submission.PayloadFields),
		PayloadRefs:    append([]string(nil), submission.PayloadRefs...),
		ExternalRefs:   cloneDomainWorkFileExternalRefsV0(submission.ExternalRefs),
		EvidenceRefs:   append([]string(nil), submission.EvidenceRefs...),
		CompleteJob:    submission.CompleteJob,
	})
	if err != nil {
		return "", err
	}
	return "sha256-" + domainWorkFileArtifactSHA256HexV0(data), nil
}

func acceptedDomainWorkFileArtifactReceiptV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) orquestadomainwork.DomainWorkArtifactReceiptV0 {
	keyData, _ := json.Marshal(domainWorkFileArtifactKeyV0{
		DomainRef:      submission.DomainRef,
		IdempotencyKey: submission.IdempotencyKey,
	})
	evidenceRefs := append([]string(nil), submission.EvidenceRefs...)
	evidenceRefs = append(evidenceRefs, "evidence-ref-domain-work-file-artifact-accepted")
	return orquestadomainwork.DomainWorkArtifactReceiptV0{
		SchemaVersion:  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:         submission.JobRef,
		ArtifactRef:    submission.ArtifactRef,
		ReceiptRef:     "domain-work-artifact-receipt-sha256-" + domainWorkFileArtifactSHA256HexV0(keyData),
		CorrelationID:  submission.CorrelationID,
		IdempotencyKey: submission.IdempotencyKey,
		ExternalRefs:   cloneDomainWorkFileExternalRefsV0(submission.ExternalRefs),
		EvidenceRefs:   evidenceRefs,
	}
}

func invalidDomainWorkFileArtifactReceiptV0(
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
		ExternalRefs:   cloneDomainWorkFileExternalRefsV0(submission.ExternalRefs),
		EvidenceRefs:   append([]string(nil), submission.EvidenceRefs...),
		Issues:         cloneDomainWorkFileIssuesV0(issues),
	}
}

func domainWorkFileArtifactSHA256HexV0(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
