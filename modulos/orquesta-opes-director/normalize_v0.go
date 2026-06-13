package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func normalizeArtifactRecordV0(
	record OPESCausalArtifactRecordV0,
	defaultDomainRef string,
) OPESCausalArtifactRecordV0 {
	record.IdempotencyKey = strings.TrimSpace(record.IdempotencyKey)
	record.Status = strings.TrimSpace(record.Status)
	record.RunRef = strings.TrimSpace(record.RunRef)
	record.TaskRef = strings.TrimSpace(record.TaskRef)
	record.DeliveryRef = strings.TrimSpace(record.DeliveryRef)
	record.CorrelationID = strings.TrimSpace(record.CorrelationID)
	record.DomainRef = strings.TrimSpace(record.DomainRef)
	if record.DomainRef == "" {
		record.DomainRef = strings.TrimSpace(defaultDomainRef)
	}
	record.JobRef = strings.TrimSpace(record.JobRef)
	record.ArtifactRef = strings.TrimSpace(record.ArtifactRef)
	record.ArtifactType = strings.TrimSpace(record.ArtifactType)
	record.Summary = strings.TrimSpace(record.Summary)
	record.PayloadFields = cloneFieldsV0(record.PayloadFields)
	record.PayloadRefs = compactStringsV0(record.PayloadRefs)
	record.ExternalRefs = normalizeExternalRefsV0(record.ExternalRefs)
	record.ReceiptRef = strings.TrimSpace(record.ReceiptRef)
	record.EvidenceRefs = compactStringsV0(record.EvidenceRefs)
	record.IssueRefs = compactStringsV0(record.IssueRefs)
	record.RecordedAt = strings.TrimSpace(record.RecordedAt)
	return record
}

func normalizeExternalRefsV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "normalize",
			CorrelationID:  "normalize",
			IdempotencyKey: "normalize",
			RequestedBy:    "normalize",
			DomainRef:      OPESCausalProducerDefaultDomainRefV0,
			WorkKind:       "normalize",
			Objective:      "normalize",
			ExternalRefs:   refs,
		},
	).ExternalRefs
}
