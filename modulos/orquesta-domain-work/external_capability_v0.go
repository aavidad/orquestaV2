package orquestadomainwork

import (
	"context"
	"strings"
)

const (
	DomainWorkExternalCapabilityQuerySchemaV0      = "domain_work_external_capability_query.v0"
	DomainWorkExternalCapabilityEvaluationSchemaV0 = "domain_work_external_capability_evaluation.v0"

	DomainWorkExternalCapabilityKindSpeechSynthesisV0 = "speech_synthesis"

	DomainWorkExternalCapabilityReasonArtifactRequiresCapabilityV0 = "artifact_type_requires_external_capability"
	DomainWorkExternalCapabilityReasonMissingV0                    = "external_capability_missing"

	ErrDomainWorkExternalCapabilityMissingV0 = "domain_work_external_capability_missing"
)

type DomainWorkExternalCapabilityRequirementV0 struct {
	CapabilityRef string                    `json:"capability_ref"`
	Kind          string                    `json:"kind"`
	WorkKind      string                    `json:"work_kind,omitempty"`
	ArtifactType  string                    `json:"artifact_type,omitempty"`
	Reason        string                    `json:"reason,omitempty"`
	Required      bool                      `json:"required,omitempty"`
	ExternalRefs  []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs  []string                  `json:"evidence_refs,omitempty"`
}

type DomainWorkExternalCapabilityV0 struct {
	CapabilityRef     string                    `json:"capability_ref"`
	Kind              string                    `json:"kind"`
	Available         bool                      `json:"available"`
	OperationalReason string                    `json:"operational_reason,omitempty"`
	ExternalRefs      []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs      []string                  `json:"evidence_refs,omitempty"`
}

type DomainWorkExternalCapabilityQueryV0 struct {
	SchemaVersion string                                      `json:"schema_version"`
	DomainRef     string                                      `json:"domain_ref"`
	WorkKind      string                                      `json:"work_kind"`
	ArtifactType  string                                      `json:"artifact_type"`
	Requirements  []DomainWorkExternalCapabilityRequirementV0 `json:"requirements,omitempty"`
	ExternalRefs  []DomainWorkExternalRefV0                   `json:"external_refs,omitempty"`
	EvidenceRefs  []string                                    `json:"evidence_refs,omitempty"`
}

type DomainWorkExternalCapabilityEvaluationV0 struct {
	SchemaVersion        string                                      `json:"schema_version"`
	DomainRef            string                                      `json:"domain_ref"`
	WorkKind             string                                      `json:"work_kind"`
	ArtifactType         string                                      `json:"artifact_type"`
	Requirements         []DomainWorkExternalCapabilityRequirementV0 `json:"requirements,omitempty"`
	DeclaredCapabilities []DomainWorkExternalCapabilityV0            `json:"declared_capabilities,omitempty"`
	MatchedCapabilities  []DomainWorkExternalCapabilityV0            `json:"matched_capabilities,omitempty"`
	MissingRequirements  []DomainWorkExternalCapabilityRequirementV0 `json:"missing_requirements,omitempty"`
	Ready                bool                                        `json:"ready"`
	OperationalReason    string                                      `json:"operational_reason,omitempty"`
	Issues               []DomainWorkIssueV0                         `json:"issues,omitempty"`
}

type DomainWorkExternalCapabilitySourcePortV0 interface {
	ListDomainWorkExternalCapabilitiesV0(
		context.Context,
		DomainWorkExternalCapabilityQueryV0,
	) ([]DomainWorkExternalCapabilityV0, error)
}

func BuildDomainWorkExternalCapabilityQueryV0(
	request DomainWorkJobRequestV0,
) DomainWorkExternalCapabilityQueryV0 {
	request = NormalizeDomainWorkJobRequestV0(request)
	artifactType := ExpectedDomainWorkArtifactTypeForWorkKindV0(request.WorkKind)
	return NormalizeDomainWorkExternalCapabilityQueryV0(
		DomainWorkExternalCapabilityQueryV0{
			DomainRef:    request.DomainRef,
			WorkKind:     request.WorkKind,
			ArtifactType: artifactType,
			Requirements: RequiredDomainWorkExternalCapabilitiesForJobV0(request),
			ExternalRefs: append([]DomainWorkExternalRefV0(nil), request.ExternalRefs...),
			EvidenceRefs: append([]string(nil), request.EvidenceRefs...),
		},
	)
}

func RequiredDomainWorkExternalCapabilitiesForJobV0(
	request DomainWorkJobRequestV0,
) []DomainWorkExternalCapabilityRequirementV0 {
	request = NormalizeDomainWorkJobRequestV0(request)
	artifactType := ExpectedDomainWorkArtifactTypeForWorkKindV0(request.WorkKind)
	if artifactType != DomainWorkArtifactTypeAudioAssetV0 {
		return []DomainWorkExternalCapabilityRequirementV0{}
	}
	return []DomainWorkExternalCapabilityRequirementV0{
		NormalizeDomainWorkExternalCapabilityRequirementV0(
			DomainWorkExternalCapabilityRequirementV0{
				CapabilityRef: DomainWorkExternalCapabilityKindSpeechSynthesisV0,
				Kind:          DomainWorkExternalCapabilityKindSpeechSynthesisV0,
				WorkKind:      request.WorkKind,
				ArtifactType:  artifactType,
				Reason:        DomainWorkExternalCapabilityReasonArtifactRequiresCapabilityV0,
				Required:      true,
				ExternalRefs:  append([]DomainWorkExternalRefV0(nil), request.ExternalRefs...),
				EvidenceRefs:  append([]string(nil), request.EvidenceRefs...),
			},
		),
	}
}

func EvaluateDomainWorkExternalCapabilitiesV0(
	request DomainWorkJobRequestV0,
	capabilities []DomainWorkExternalCapabilityV0,
) DomainWorkExternalCapabilityEvaluationV0 {
	query := BuildDomainWorkExternalCapabilityQueryV0(request)
	evaluation := DomainWorkExternalCapabilityEvaluationV0{
		SchemaVersion:        DomainWorkExternalCapabilityEvaluationSchemaV0,
		DomainRef:            query.DomainRef,
		WorkKind:             query.WorkKind,
		ArtifactType:         query.ArtifactType,
		Requirements:         append([]DomainWorkExternalCapabilityRequirementV0(nil), query.Requirements...),
		DeclaredCapabilities: compactDomainWorkExternalCapabilitiesV0(capabilities),
	}
	for _, requirement := range evaluation.Requirements {
		requirement = NormalizeDomainWorkExternalCapabilityRequirementV0(requirement)
		if !requirement.Required {
			continue
		}
		match, ok := matchDomainWorkExternalCapabilityV0(
			requirement,
			evaluation.DeclaredCapabilities,
		)
		if ok {
			evaluation.MatchedCapabilities = append(evaluation.MatchedCapabilities, match)
			continue
		}
		evaluation.MissingRequirements = append(evaluation.MissingRequirements, requirement)
		evaluation.Issues = append(evaluation.Issues, DomainWorkIssueV0{
			Code:  ErrDomainWorkExternalCapabilityMissingV0,
			Field: "external_capabilities",
		})
		if evaluation.OperationalReason == "" {
			evaluation.OperationalReason = missingDomainWorkExternalCapabilityReasonV0(
				requirement,
				evaluation.DeclaredCapabilities,
			)
		}
	}
	evaluation.Ready = len(evaluation.MissingRequirements) == 0
	return evaluation
}

func NormalizeDomainWorkExternalCapabilityQueryV0(
	query DomainWorkExternalCapabilityQueryV0,
) DomainWorkExternalCapabilityQueryV0 {
	query.SchemaVersion = defaultDomainWorkSchemaV0(
		query.SchemaVersion,
		DomainWorkExternalCapabilityQuerySchemaV0,
	)
	query.DomainRef = strings.TrimSpace(query.DomainRef)
	query.WorkKind = strings.TrimSpace(query.WorkKind)
	query.ArtifactType = strings.TrimSpace(query.ArtifactType)
	query.Requirements = compactDomainWorkExternalCapabilityRequirementsV0(query.Requirements)
	query.ExternalRefs = compactDomainWorkExternalRefsV0(query.ExternalRefs)
	query.EvidenceRefs = compactDomainWorkStringsV0(query.EvidenceRefs)
	return query
}

func NormalizeDomainWorkExternalCapabilityRequirementV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
) DomainWorkExternalCapabilityRequirementV0 {
	requirement.Kind = normalizeDomainWorkExternalCapabilityKindV0(requirement.Kind)
	requirement.CapabilityRef = strings.TrimSpace(requirement.CapabilityRef)
	if requirement.CapabilityRef == "" {
		requirement.CapabilityRef = requirement.Kind
	}
	requirement.WorkKind = strings.TrimSpace(requirement.WorkKind)
	requirement.ArtifactType = strings.TrimSpace(requirement.ArtifactType)
	requirement.Reason = strings.TrimSpace(requirement.Reason)
	requirement.ExternalRefs = compactDomainWorkExternalRefsV0(requirement.ExternalRefs)
	requirement.EvidenceRefs = compactDomainWorkStringsV0(requirement.EvidenceRefs)
	return requirement
}

func NormalizeDomainWorkExternalCapabilityV0(
	capability DomainWorkExternalCapabilityV0,
) DomainWorkExternalCapabilityV0 {
	capability.Kind = normalizeDomainWorkExternalCapabilityKindV0(capability.Kind)
	capability.CapabilityRef = strings.TrimSpace(capability.CapabilityRef)
	if capability.CapabilityRef == "" {
		capability.CapabilityRef = capability.Kind
	}
	capability.OperationalReason = strings.TrimSpace(capability.OperationalReason)
	capability.ExternalRefs = compactDomainWorkExternalRefsV0(capability.ExternalRefs)
	capability.EvidenceRefs = compactDomainWorkStringsV0(capability.EvidenceRefs)
	return capability
}

func compactDomainWorkExternalCapabilityRequirementsV0(
	values []DomainWorkExternalCapabilityRequirementV0,
) []DomainWorkExternalCapabilityRequirementV0 {
	seen := map[string]struct{}{}
	out := make([]DomainWorkExternalCapabilityRequirementV0, 0, len(values))
	for _, value := range values {
		requirement := NormalizeDomainWorkExternalCapabilityRequirementV0(value)
		if requirement.Kind == "" {
			continue
		}
		key := requirement.CapabilityRef + "\x00" +
			requirement.Kind + "\x00" +
			requirement.WorkKind + "\x00" +
			requirement.ArtifactType + "\x00" +
			requirement.Reason + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.Required) + "\x00" +
			joinDomainWorkExternalRefsV0(requirement.ExternalRefs) + "\x00" +
			strings.Join(requirement.EvidenceRefs, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, requirement)
	}
	if out == nil {
		return []DomainWorkExternalCapabilityRequirementV0{}
	}
	return out
}

func compactDomainWorkExternalCapabilitiesV0(
	values []DomainWorkExternalCapabilityV0,
) []DomainWorkExternalCapabilityV0 {
	seen := map[string]struct{}{}
	out := make([]DomainWorkExternalCapabilityV0, 0, len(values))
	for _, value := range values {
		capability := NormalizeDomainWorkExternalCapabilityV0(value)
		if capability.Kind == "" {
			continue
		}
		key := capability.CapabilityRef + "\x00" +
			capability.Kind + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.Available) + "\x00" +
			capability.OperationalReason + "\x00" +
			joinDomainWorkExternalRefsV0(capability.ExternalRefs) + "\x00" +
			strings.Join(capability.EvidenceRefs, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, capability)
	}
	if out == nil {
		return []DomainWorkExternalCapabilityV0{}
	}
	return out
}

func matchDomainWorkExternalCapabilityV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capabilities []DomainWorkExternalCapabilityV0,
) (DomainWorkExternalCapabilityV0, bool) {
	for _, capability := range capabilities {
		if !capability.Available {
			continue
		}
		if capability.Kind == requirement.Kind ||
			capability.CapabilityRef == requirement.CapabilityRef {
			return capability, true
		}
	}
	return DomainWorkExternalCapabilityV0{}, false
}

func missingDomainWorkExternalCapabilityReasonV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capabilities []DomainWorkExternalCapabilityV0,
) string {
	for _, capability := range capabilities {
		if capability.Kind != requirement.Kind &&
			capability.CapabilityRef != requirement.CapabilityRef {
			continue
		}
		if capability.OperationalReason != "" {
			return capability.OperationalReason
		}
	}
	if requirement.Kind == "" {
		return DomainWorkExternalCapabilityReasonMissingV0
	}
	return DomainWorkExternalCapabilityReasonMissingV0 + ":" + requirement.Kind
}

func normalizeDomainWorkExternalCapabilityKindV0(value string) string {
	switch strings.TrimSpace(value) {
	case "tts", "text_to_speech", "text-to-speech":
		return DomainWorkExternalCapabilityKindSpeechSynthesisV0
	default:
		return strings.TrimSpace(value)
	}
}

func boolDomainWorkExternalCapabilityKeyV0(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
