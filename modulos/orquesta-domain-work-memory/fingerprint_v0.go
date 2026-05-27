package orquestadomainworkmemory

import (
	"encoding/json"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func domainWorkMemoryRequestFingerprintV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) (string, error) {
	identity, err := orquestadomainwork.BuildDomainWorkJobIdentityV0(request)
	if err != nil {
		return "", err
	}
	return identity.Fingerprint, nil
}

type domainWorkMemoryRequestFingerprintInputV0 struct {
	SchemaVersion      string                                       `json:"schema_version"`
	CorrelationID      string                                       `json:"correlation_id"`
	IdempotencyKey     string                                       `json:"idempotency_key"`
	RequestedBy        string                                       `json:"requested_by"`
	DomainRef          string                                       `json:"domain_ref"`
	InterfaceRefs      []string                                     `json:"interface_refs"`
	WorkKind           string                                       `json:"work_kind"`
	WorkRefs           []string                                     `json:"work_refs"`
	Objective          string                                       `json:"objective"`
	InputFields        []orquestadomainwork.DomainWorkFieldV0       `json:"input_fields"`
	InputRefs          []string                                     `json:"input_refs"`
	Constraints        []string                                     `json:"constraints"`
	AcceptanceCriteria []string                                     `json:"acceptance_criteria"`
	ExternalRefs       []orquestadomainwork.DomainWorkExternalRefV0 `json:"external_refs"`
	EvidenceRefs       []string                                     `json:"evidence_refs"`
}

func domainWorkMemoryRequestLegacyFingerprintV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) (string, error) {
	value := domainWorkMemoryRequestFingerprintInputV0{
		SchemaVersion:      request.SchemaVersion,
		CorrelationID:      request.CorrelationID,
		IdempotencyKey:     request.IdempotencyKey,
		RequestedBy:        request.RequestedBy,
		DomainRef:          request.DomainRef,
		InterfaceRefs:      append([]string(nil), request.InterfaceRefs...),
		WorkKind:           request.WorkKind,
		WorkRefs:           append([]string(nil), request.WorkRefs...),
		Objective:          request.Objective,
		InputFields:        cloneDomainWorkMemoryFieldsV0(request.InputFields),
		InputRefs:          append([]string(nil), request.InputRefs...),
		Constraints:        append([]string(nil), request.Constraints...),
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		ExternalRefs:       cloneDomainWorkMemoryExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
