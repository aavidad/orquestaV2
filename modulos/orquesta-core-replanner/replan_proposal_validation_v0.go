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
)

var forbiddenReplanProposalFragmentsV0 = []string{
	"db", "database", "sql", "dsn", "runtime",
	"provider", "providers", "proveedor", "proveedores",
	"model", "models", "modelo", "modelos",
	"home", "oauth", "prompt", "prompts",
	"transcript", "transcripts", "diff", "diffs",
	"completion", "raw_text", "full_text",
	"filesystem", "git", "docker", "tmux",
	"secret", "secrets", "secreto", "secretos", "token", "password",
	"credential", "credencial", "api_key",
}

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
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenReplanProposalFragmentsV0 {
			if containsForbiddenReplanProposalFragmentV0(lower, fragment) {
				return true
			}
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

func containsForbiddenReplanProposalFragmentV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if hasReplanProposalTokenBoundaryV0(lowerValue, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func hasReplanProposalTokenBoundaryV0(value string, start int, end int) bool {
	before := start == 0 || !isAsciiLetterOrDigitV0(value[start-1])
	after := end >= len(value) || !isAsciiLetterOrDigitV0(value[end])
	return before && after
}

func isAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func replanProposalErrorV0(code string, field string) ReplanProposalErrorV0 {
	return ReplanProposalErrorV0{Code: code, Field: field}
}
