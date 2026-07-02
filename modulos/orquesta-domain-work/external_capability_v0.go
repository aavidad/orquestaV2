package orquestadomainwork

import (
	"context"
	"strconv"
	"strings"
)

const (
	DomainWorkExternalCapabilityQuerySchemaV0      = "domain_work_external_capability_query.v0"
	DomainWorkExternalCapabilityEvaluationSchemaV0 = "domain_work_external_capability_evaluation.v0"

	DomainWorkExternalCapabilityKindSpeechSynthesisV0  = "speech_synthesis"
	DomainWorkExternalCapabilityKindRemoteQAProviderV0 = "remote_qa_provider"

	DomainWorkExternalCapabilityReasonArtifactRequiresCapabilityV0 = "artifact_type_requires_external_capability"
	DomainWorkExternalCapabilityReasonWorkKindRequiresCapabilityV0 = "work_kind_requires_external_capability"
	DomainWorkExternalCapabilityReasonMissingV0                    = "external_capability_missing"

	ErrDomainWorkExternalCapabilityMissingV0 = "domain_work_external_capability_missing"
)

type DomainWorkExternalCapabilityRequirementV0 struct {
	CapabilityRef                    string                    `json:"capability_ref"`
	Kind                             string                    `json:"kind"`
	WorkKind                         string                    `json:"work_kind,omitempty"`
	ArtifactType                     string                    `json:"artifact_type,omitempty"`
	Reason                           string                    `json:"reason,omitempty"`
	Required                         bool                      `json:"required,omitempty"`
	NetworkRequired                  bool                      `json:"network_required,omitempty"`
	AuthStateRequired                bool                      `json:"auth_state_required,omitempty"`
	ToolPathRequired                 bool                      `json:"tool_path_required,omitempty"`
	ProviderQuotaSensitive           bool                      `json:"provider_quota_sensitive,omitempty"`
	CommandTimeoutSeconds            int                       `json:"command_timeout_seconds,omitempty"`
	ProgressHeartbeatRequired        bool                      `json:"progress_heartbeat_required,omitempty"`
	ProviderTimeoutRequired          bool                      `json:"provider_timeout_required,omitempty"`
	ProviderNoProgressTimeoutSeconds int                       `json:"provider_no_progress_timeout_seconds,omitempty"`
	ResumeEvidenceRequired           bool                      `json:"resume_evidence_required,omitempty"`
	NoDuplicateValidOutputsRequired  bool                      `json:"no_duplicate_valid_outputs_required,omitempty"`
	ExternalRefs                     []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs                     []string                  `json:"evidence_refs,omitempty"`
}

type DomainWorkExternalCapabilityV0 struct {
	CapabilityRef                    string                    `json:"capability_ref"`
	Kind                             string                    `json:"kind"`
	Available                        bool                      `json:"available"`
	OperationalReason                string                    `json:"operational_reason,omitempty"`
	NetworkReady                     bool                      `json:"network_ready,omitempty"`
	AuthStateReady                   bool                      `json:"auth_state_ready,omitempty"`
	ToolPathReady                    bool                      `json:"tool_path_ready,omitempty"`
	ProviderQuotaReady               bool                      `json:"provider_quota_ready,omitempty"`
	CommandTimeoutSeconds            int                       `json:"command_timeout_seconds,omitempty"`
	ProgressHeartbeatReady           bool                      `json:"progress_heartbeat_ready,omitempty"`
	ProviderTimeoutReady             bool                      `json:"provider_timeout_ready,omitempty"`
	ProviderNoProgressTimeoutSeconds int                       `json:"provider_no_progress_timeout_seconds,omitempty"`
	ResumeEvidenceReady              bool                      `json:"resume_evidence_ready,omitempty"`
	NoDuplicateValidOutputsReady     bool                      `json:"no_duplicate_valid_outputs_ready,omitempty"`
	ExternalRefs                     []DomainWorkExternalRefV0 `json:"external_refs,omitempty"`
	EvidenceRefs                     []string                  `json:"evidence_refs,omitempty"`
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
	requirements := []DomainWorkExternalCapabilityRequirementV0{}
	if artifactType == DomainWorkArtifactTypeAudioAssetV0 {
		requirements = append(requirements, NormalizeDomainWorkExternalCapabilityRequirementV0(
			DomainWorkExternalCapabilityRequirementV0{
				CapabilityRef:                    DomainWorkExternalCapabilityKindSpeechSynthesisV0,
				Kind:                             DomainWorkExternalCapabilityKindSpeechSynthesisV0,
				WorkKind:                         request.WorkKind,
				ArtifactType:                     artifactType,
				Reason:                           DomainWorkExternalCapabilityReasonArtifactRequiresCapabilityV0,
				Required:                         true,
				NetworkRequired:                  true,
				ToolPathRequired:                 true,
				ProviderQuotaSensitive:           true,
				CommandTimeoutSeconds:            1800,
				ProgressHeartbeatRequired:        true,
				ProviderTimeoutRequired:          true,
				ProviderNoProgressTimeoutSeconds: 300,
				ResumeEvidenceRequired:           true,
				NoDuplicateValidOutputsRequired:  true,
				ExternalRefs:                     append([]DomainWorkExternalRefV0(nil), request.ExternalRefs...),
				EvidenceRefs:                     append([]string(nil), request.EvidenceRefs...),
			},
		))
	}
	if domainWorkRequiresRemoteQAProviderV0(request.WorkKind, artifactType) {
		requirements = append(requirements, NormalizeDomainWorkExternalCapabilityRequirementV0(
			DomainWorkExternalCapabilityRequirementV0{
				CapabilityRef:          DomainWorkExternalCapabilityKindRemoteQAProviderV0,
				Kind:                   DomainWorkExternalCapabilityKindRemoteQAProviderV0,
				WorkKind:               request.WorkKind,
				ArtifactType:           artifactType,
				Reason:                 DomainWorkExternalCapabilityReasonWorkKindRequiresCapabilityV0,
				Required:               true,
				NetworkRequired:        true,
				AuthStateRequired:      true,
				ProviderQuotaSensitive: true,
				CommandTimeoutSeconds:  1200,
				ExternalRefs:           append([]DomainWorkExternalRefV0(nil), request.ExternalRefs...),
				EvidenceRefs:           append([]string(nil), request.EvidenceRefs...),
			},
		))
	}
	if requirements == nil {
		return []DomainWorkExternalCapabilityRequirementV0{}
	}
	return requirements
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
	if requirement.CommandTimeoutSeconds < 0 {
		requirement.CommandTimeoutSeconds = 0
	}
	if requirement.ProviderNoProgressTimeoutSeconds < 0 {
		requirement.ProviderNoProgressTimeoutSeconds = 0
	}
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
	if capability.CommandTimeoutSeconds < 0 {
		capability.CommandTimeoutSeconds = 0
	}
	if capability.ProviderNoProgressTimeoutSeconds < 0 {
		capability.ProviderNoProgressTimeoutSeconds = 0
	}
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
			boolDomainWorkExternalCapabilityKeyV0(requirement.NetworkRequired) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.AuthStateRequired) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.ToolPathRequired) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.ProviderQuotaSensitive) + "\x00" +
			itoaDomainWorkExternalCapabilityV0(requirement.CommandTimeoutSeconds) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.ProgressHeartbeatRequired) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.ProviderTimeoutRequired) + "\x00" +
			itoaDomainWorkExternalCapabilityV0(requirement.ProviderNoProgressTimeoutSeconds) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.ResumeEvidenceRequired) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(requirement.NoDuplicateValidOutputsRequired) + "\x00" +
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
			boolDomainWorkExternalCapabilityKeyV0(capability.NetworkReady) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.AuthStateReady) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.ToolPathReady) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.ProviderQuotaReady) + "\x00" +
			itoaDomainWorkExternalCapabilityV0(capability.CommandTimeoutSeconds) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.ProgressHeartbeatReady) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.ProviderTimeoutReady) + "\x00" +
			itoaDomainWorkExternalCapabilityV0(capability.ProviderNoProgressTimeoutSeconds) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.ResumeEvidenceReady) + "\x00" +
			boolDomainWorkExternalCapabilityKeyV0(capability.NoDuplicateValidOutputsReady) + "\x00" +
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
		if domainWorkCapabilityMatchesRequirementV0(requirement, capability) &&
			domainWorkCapabilitySatisfiesProfileV0(requirement, capability) {
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
		if !domainWorkCapabilityMatchesRequirementV0(requirement, capability) {
			continue
		}
		if capability.OperationalReason != "" {
			return capability.OperationalReason
		}
		if reason := domainWorkCapabilityProfileMissingReasonV0(requirement, capability); reason != "" {
			return reason
		}
	}
	if requirement.Kind == "" {
		return DomainWorkExternalCapabilityReasonMissingV0
	}
	return DomainWorkExternalCapabilityReasonMissingV0 + ":" + requirement.Kind
}

func domainWorkCapabilityMatchesRequirementV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) bool {
	if capability.Kind != requirement.Kind {
		return false
	}
	return domainWorkCapabilityMatchesRequirementIdentityV0(requirement, capability)
}

func domainWorkCapabilityMatchesRequirementIdentityV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) bool {
	if requirement.CapabilityRef == "" || requirement.CapabilityRef == requirement.Kind {
		return true
	}
	return capability.CapabilityRef == requirement.CapabilityRef
}

func normalizeDomainWorkExternalCapabilityKindV0(value string) string {
	switch strings.TrimSpace(value) {
	case "tts", "text_to_speech", "text-to-speech":
		return DomainWorkExternalCapabilityKindSpeechSynthesisV0
	case "qa", "qa_provider", "remote-review", "remote_review_provider":
		return DomainWorkExternalCapabilityKindRemoteQAProviderV0
	default:
		return strings.TrimSpace(value)
	}
}

func domainWorkRequiresRemoteQAProviderV0(workKind string, artifactType string) bool {
	switch artifactType {
	case DomainWorkArtifactTypeAgentReviewReportV0, DomainWorkArtifactTypeAgentPairReviewReportV0:
		return true
	}
	switch strings.TrimSpace(workKind) {
	case "review_agent_independent", "review_independent_agent",
		"review_agent_pair", "review_peer_pair", "review_pair",
		"review_codex", "review_gemini", "review_claude",
		"review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		return true
	default:
		return false
	}
}

func domainWorkCapabilitySatisfiesProfileV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) bool {
	if requirement.NetworkRequired && !capability.NetworkReady {
		return false
	}
	if requirement.AuthStateRequired && !capability.AuthStateReady {
		return false
	}
	if requirement.ToolPathRequired && !capability.ToolPathReady {
		return false
	}
	if requirement.ProviderQuotaSensitive && !capability.ProviderQuotaReady {
		return false
	}
	if requirement.CommandTimeoutSeconds > 0 &&
		capability.CommandTimeoutSeconds > 0 &&
		capability.CommandTimeoutSeconds < requirement.CommandTimeoutSeconds {
		return false
	}
	if !domainWorkCapabilityOperationalEvidenceReadyV0(requirement, capability) {
		return false
	}
	return true
}

func domainWorkCapabilityProfileMissingReasonV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) string {
	kind := strings.TrimSpace(requirement.Kind)
	if kind == "" {
		kind = strings.TrimSpace(capability.Kind)
	}
	reasonFor := func(check string) string {
		return DomainWorkExternalCapabilityReasonMissingV0 + ":" + kind + ":" + strings.TrimSpace(check)
	}
	if requirement.NetworkRequired && !capability.NetworkReady {
		return reasonFor("network_ready")
	}
	if requirement.AuthStateRequired && !capability.AuthStateReady {
		return reasonFor("auth_state_ready")
	}
	if requirement.ToolPathRequired && !capability.ToolPathReady {
		return reasonFor("tool_path_ready")
	}
	if requirement.ProviderQuotaSensitive && !capability.ProviderQuotaReady {
		return reasonFor("provider_quota_ready")
	}
	if requirement.CommandTimeoutSeconds > 0 &&
		capability.CommandTimeoutSeconds > 0 &&
		capability.CommandTimeoutSeconds < requirement.CommandTimeoutSeconds {
		return reasonFor("command_timeout_seconds")
	}
	if !domainWorkCapabilityOperationalEvidenceReadyV0(requirement, capability) {
		if domainWorkCapabilityHasPartialResumeEvidenceV0(capability) {
			if requirement.ResumeEvidenceRequired && !capability.ResumeEvidenceReady {
				return reasonFor("resume_evidence_ready")
			}
			if requirement.NoDuplicateValidOutputsRequired && !capability.NoDuplicateValidOutputsReady {
				return reasonFor("no_duplicate_valid_outputs_ready")
			}
		}
		if requirement.ProgressHeartbeatRequired && !capability.ProgressHeartbeatReady {
			return reasonFor("progress_heartbeat_ready")
		}
		if requirement.ProviderTimeoutRequired && !capability.ProviderTimeoutReady {
			return reasonFor("provider_timeout_ready")
		}
		if requirement.ProviderNoProgressTimeoutSeconds > 0 {
			if capability.ProviderNoProgressTimeoutSeconds <= 0 {
				return reasonFor("provider_no_progress_timeout_seconds")
			}
			if capability.ProviderNoProgressTimeoutSeconds > requirement.ProviderNoProgressTimeoutSeconds {
				return reasonFor("provider_no_progress_timeout_seconds")
			}
		}
		if requirement.ResumeEvidenceRequired && !capability.ResumeEvidenceReady {
			return reasonFor("resume_evidence_ready")
		}
		if requirement.NoDuplicateValidOutputsRequired && !capability.NoDuplicateValidOutputsReady {
			return reasonFor("no_duplicate_valid_outputs_ready")
		}
	}
	return ""
}

func domainWorkCapabilityOperationalEvidenceReadyV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) bool {
	providerRequired := domainWorkCapabilityProviderSupervisionRequiredV0(requirement)
	resumeRequired := domainWorkCapabilityResumeEvidenceRequiredV0(requirement)
	providerReady := providerRequired && domainWorkCapabilityProviderSupervisionReadyV0(requirement, capability)
	resumeReady := resumeRequired && domainWorkCapabilityResumeEvidenceReadyV0(requirement, capability)
	switch {
	case providerRequired && resumeRequired:
		return providerReady || resumeReady
	case providerRequired:
		return providerReady
	case resumeRequired:
		return resumeReady
	default:
		return true
	}
}

func domainWorkCapabilityProviderSupervisionRequiredV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
) bool {
	return requirement.ProgressHeartbeatRequired ||
		requirement.ProviderTimeoutRequired ||
		requirement.ProviderNoProgressTimeoutSeconds > 0
}

func domainWorkCapabilityResumeEvidenceRequiredV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
) bool {
	return requirement.ResumeEvidenceRequired || requirement.NoDuplicateValidOutputsRequired
}

func domainWorkCapabilityProviderSupervisionReadyV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) bool {
	if requirement.ProgressHeartbeatRequired && !capability.ProgressHeartbeatReady {
		return false
	}
	if requirement.ProviderTimeoutRequired && !capability.ProviderTimeoutReady {
		return false
	}
	if requirement.ProviderNoProgressTimeoutSeconds > 0 {
		if capability.ProviderNoProgressTimeoutSeconds <= 0 {
			return false
		}
		if capability.ProviderNoProgressTimeoutSeconds > requirement.ProviderNoProgressTimeoutSeconds {
			return false
		}
	}
	return true
}

func domainWorkCapabilityResumeEvidenceReadyV0(
	requirement DomainWorkExternalCapabilityRequirementV0,
	capability DomainWorkExternalCapabilityV0,
) bool {
	if requirement.ResumeEvidenceRequired && !capability.ResumeEvidenceReady {
		return false
	}
	if requirement.NoDuplicateValidOutputsRequired && !capability.NoDuplicateValidOutputsReady {
		return false
	}
	return requirement.ResumeEvidenceRequired || requirement.NoDuplicateValidOutputsRequired
}

func domainWorkCapabilityHasPartialResumeEvidenceV0(
	capability DomainWorkExternalCapabilityV0,
) bool {
	return capability.ResumeEvidenceReady || capability.NoDuplicateValidOutputsReady
}

func itoaDomainWorkExternalCapabilityV0(value int) string {
	return strconv.Itoa(value)
}

func boolDomainWorkExternalCapabilityKeyV0(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
