package orquestacorereplanner

import (
	"encoding/json"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	maxReplanProposalPayloadBytesV0 = 2048
	maxReplanProposalStringV0       = 600
	maxReplanProposalEvidenceRefsV0 = 20
	coreReplannerDetailBoundaryV0   = "core_replanner"
)

func validateReplanProposalRequiredFieldsV0(proposal ReplanProposalV0) error {
	fields := map[string]string{
		"replan_ref":         proposal.ReplanRef,
		"run_ref":            proposal.RunRef,
		"task_ref":           proposal.TaskRef,
		"source_ref":         proposal.SourceRef,
		"reason_code":        proposal.ReasonCode,
		"recommended_action": string(proposal.RecommendedAction),
		"summary":            proposal.Summary,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return replanProposalErrorV0(ErrReplanProposalInvalidoV0, field)
		}
	}
	return nil
}

func validateReplanProposalActionFieldsV0(proposal ReplanProposalV0) error {
	switch proposal.RecommendedAction {
	case ReplanActionEscalateCapacityV0:
		if strings.TrimSpace(proposal.CapacityRequestRef) == "" {
			return replanProposalErrorV0(ErrReplanProposalInvalidoV0, "capacity_request_ref")
		}
	case ReplanActionReplaceAgentV0:
		if strings.TrimSpace(proposal.ReplacementRole) == "" {
			return replanProposalErrorV0(ErrReplanProposalInvalidoV0, "replacement_role")
		}
	}
	return nil
}

func validateReplanProposalCollectionsV0(proposal ReplanProposalV0) error {
	if len(proposal.EvidenceRefs) > maxReplanProposalEvidenceRefsV0 {
		return replanProposalErrorV0(ErrReplanProposalInvalidoV0, "evidence_refs")
	}
	for _, ref := range proposal.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			return replanProposalErrorV0(ErrReplanProposalInvalidoV0, "evidence_refs")
		}
	}
	return nil
}

func validateReplanProposalCompactPayloadV0(proposal ReplanProposalV0) error {
	data, err := json.Marshal(proposal)
	if err != nil {
		return replanProposalErrorV0(ErrReplanProposalPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanProposalPayloadBytesV0 {
		return replanProposalErrorV0(ErrReplanProposalPayloadInvalidoV0, "payload")
	}
	return nil
}

func replanProposalTextFieldsV0(proposal ReplanProposalV0) []string {
	values := []string{
		proposal.ReplanRef,
		proposal.RunRef,
		proposal.TaskRef,
		proposal.SourceRef,
		proposal.ReasonCode,
		string(proposal.RecommendedAction),
		proposal.CapacityRequestRef,
		proposal.ReplacementRole,
		proposal.Summary,
	}
	return append(values, proposal.EvidenceRefs...)
}

func replanProposalHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxReplanProposalStringV0 {
			return true
		}
	}
	return false
}

func replanProposalHasForbiddenDetailsV0(values []string) bool {
	if orquestarails.ValuesContainOperationalSensitiveDetailForFieldV0(
		coreReplannerDetailBoundaryV0,
		"*",
		values,
	) {
		return true
	}
	for _, value := range values {
		if replanProposalContainsRealHomePathV0(value) {
			return true
		}
	}
	return false
}

func normalizeReplanProposalStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	normalized := make([]string, len(values))
	for index, value := range values {
		normalized[index] = strings.TrimSpace(value)
	}
	return normalized
}

func replanProposalContainsRealHomePathV0(value string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(value, `\`, "/"))
	return replanProposalContainsHomePathPrefixV0(normalized, "/home/") ||
		replanProposalContainsHomePathPrefixV0(normalized, "/users/") ||
		replanProposalContainsHomePathPrefixV0(normalized, "c:/users/")
}

func replanProposalContainsHomePathPrefixV0(value string, prefix string) bool {
	start := 0
	for {
		index := strings.Index(value[start:], prefix)
		if index < 0 {
			return false
		}
		absolute := start + index
		if replanProposalPathPrefixBoundaryV0(value, absolute) &&
			replanProposalPathPrefixHasUserSegmentV0(value, absolute+len(prefix)) {
			return true
		}
		start = absolute + len(prefix)
	}
}

func replanProposalPathPrefixBoundaryV0(value string, index int) bool {
	if index == 0 {
		return true
	}
	switch value[index-1] {
	case ' ', '\t', '\n', '\r', '"', '\'', '`', '=', ':':
		return true
	default:
		return false
	}
}

func replanProposalPathPrefixHasUserSegmentV0(value string, start int) bool {
	if start >= len(value) || value[start] == '/' {
		return false
	}
	end := strings.IndexByte(value[start:], '/')
	if end < 0 {
		return true
	}
	return end > 0
}

func replanProposalErrorV0(code string, field string) ReplanProposalErrorV0 {
	return ReplanProposalErrorV0{Code: code, Field: field}
}
