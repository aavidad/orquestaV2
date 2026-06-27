package orquestadomainwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	DomainWorkJobFingerprintAlgorithmV0 = "sha256"
)

type DomainWorkJobIdentityV0 struct {
	Fingerprint string
	JobRefBase  string
}

func BuildDomainWorkJobIdentityV0(
	request DomainWorkJobRequestV0,
) (DomainWorkJobIdentityV0, error) {
	normalized := NormalizeDomainWorkJobRequestV0(request)
	fingerprintInput := domainWorkJobFingerprintInputV0{
		SchemaVersion:      normalized.SchemaVersion,
		CorrelationID:      normalized.CorrelationID,
		IdempotencyKey:     normalized.IdempotencyKey,
		RequestedBy:        normalized.RequestedBy,
		DomainRef:          normalized.DomainRef,
		InterfaceRefs:      append([]string(nil), normalized.InterfaceRefs...),
		WorkKind:           normalized.WorkKind,
		WorkRefs:           append([]string(nil), normalized.WorkRefs...),
		Objective:          normalized.Objective,
		InputFields:        cloneDomainWorkIdentityFieldsV0(normalized.InputFields),
		InputRefs:          append([]string(nil), normalized.InputRefs...),
		Constraints:        append([]string(nil), normalized.Constraints...),
		AcceptanceCriteria: append([]string(nil), normalized.AcceptanceCriteria...),
		RequiredTests:      cloneDomainWorkIdentityRequiredTestsV0(normalized.RequiredTests),
		ExternalRefs:       append([]DomainWorkExternalRefV0(nil), normalized.ExternalRefs...),
		EvidenceRefs:       append([]string(nil), normalized.EvidenceRefs...),
	}
	fingerprintData, err := json.Marshal(fingerprintInput)
	if err != nil {
		return DomainWorkJobIdentityV0{}, err
	}
	keyData, err := json.Marshal(domainWorkJobRefKeyV0{
		SchemaVersion:  normalized.SchemaVersion,
		DomainRef:      normalized.DomainRef,
		IdempotencyKey: normalized.IdempotencyKey,
	})
	if err != nil {
		return DomainWorkJobIdentityV0{}, err
	}
	return DomainWorkJobIdentityV0{
		Fingerprint: "sha256-" + domainWorkJobSHA256HexV0(fingerprintData),
		JobRefBase:  "domain-work-job-sha256-" + domainWorkJobSHA256HexV0(keyData),
	}, nil
}

func EquivalentDomainWorkJobRequestsV0(
	left DomainWorkJobRequestV0,
	right DomainWorkJobRequestV0,
) (bool, error) {
	leftIdentity, err := BuildDomainWorkJobIdentityV0(left)
	if err != nil {
		return false, err
	}
	rightIdentity, err := BuildDomainWorkJobIdentityV0(right)
	if err != nil {
		return false, err
	}
	return leftIdentity.Fingerprint == rightIdentity.Fingerprint, nil
}

type domainWorkJobFingerprintInputV0 struct {
	SchemaVersion      string                     `json:"schema_version"`
	CorrelationID      string                     `json:"correlation_id"`
	IdempotencyKey     string                     `json:"idempotency_key"`
	RequestedBy        string                     `json:"requested_by"`
	DomainRef          string                     `json:"domain_ref"`
	InterfaceRefs      []string                   `json:"interface_refs"`
	WorkKind           string                     `json:"work_kind"`
	WorkRefs           []string                   `json:"work_refs"`
	Objective          string                     `json:"objective"`
	InputFields        []DomainWorkFieldV0        `json:"input_fields"`
	InputRefs          []string                   `json:"input_refs"`
	Constraints        []string                   `json:"constraints"`
	AcceptanceCriteria []string                   `json:"acceptance_criteria"`
	RequiredTests      []DomainWorkRequiredTestV0 `json:"required_tests"`
	ExternalRefs       []DomainWorkExternalRefV0  `json:"external_refs"`
	EvidenceRefs       []string                   `json:"evidence_refs"`
}

type domainWorkJobRefKeyV0 struct {
	SchemaVersion  string `json:"schema_version"`
	DomainRef      string `json:"domain_ref"`
	IdempotencyKey string `json:"idempotency_key"`
}

func domainWorkJobSHA256HexV0(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func cloneDomainWorkIdentityFieldsV0(
	fields []DomainWorkFieldV0,
) []DomainWorkFieldV0 {
	out := make([]DomainWorkFieldV0, len(fields))
	for index, field := range fields {
		out[index] = field
		out[index].Values = append([]string(nil), field.Values...)
		out[index].ValueJSON = append([]byte(nil), field.ValueJSON...)
	}
	return out
}

func cloneDomainWorkIdentityRequiredTestsV0(
	tests []DomainWorkRequiredTestV0,
) []DomainWorkRequiredTestV0 {
	out := make([]DomainWorkRequiredTestV0, len(tests))
	for index, test := range tests {
		out[index] = test
		out[index].AcceptanceCriteria = append([]string(nil), test.AcceptanceCriteria...)
		out[index].AcceptanceCriteriaRefs = append([]string(nil), test.AcceptanceCriteriaRefs...)
		out[index].InputRefs = append([]string(nil), test.InputRefs...)
		out[index].ExternalRefs = append([]DomainWorkExternalRefV0(nil), test.ExternalRefs...)
		out[index].EvidenceRefs = append([]string(nil), test.EvidenceRefs...)
	}
	return out
}
