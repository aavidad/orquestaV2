package orquestacoreconcurrency

import (
	"sort"
	"strings"
)

const WorksetClaimSchemaVersionV0 = "workset_claim.v0"

type WorksetClaimV0 struct {
	SchemaVersion  string       `json:"schema_version"`
	ClaimRef       string       `json:"claim_ref"`
	RunRef         string       `json:"run_ref"`
	TaskRef        string       `json:"task_ref"`
	GroupRef       string       `json:"group_ref,omitempty"`
	AgentRequestID string       `json:"agent_request_id,omitempty"`
	ReadSet        []ScopeRefV0 `json:"read_set,omitempty"`
	WriteSet       []ScopeRefV0 `json:"write_set"`
	DependsOn      []string     `json:"depends_on,omitempty"`
	EvidenceRefs   []string     `json:"evidence_refs,omitempty"`
}

func NormalizeWorksetClaimV0(claim WorksetClaimV0) (WorksetClaimV0, []WorksetClaimIssueV0) {
	normalized := WorksetClaimV0{
		SchemaVersion:  normalizeWorksetClaimSchemaVersionV0(claim.SchemaVersion),
		ClaimRef:       strings.TrimSpace(claim.ClaimRef),
		RunRef:         strings.TrimSpace(claim.RunRef),
		TaskRef:        strings.TrimSpace(claim.TaskRef),
		GroupRef:       strings.TrimSpace(claim.GroupRef),
		AgentRequestID: strings.TrimSpace(claim.AgentRequestID),
		DependsOn:      normalizeOpaqueRefsV0(claim.DependsOn),
		EvidenceRefs:   normalizeOpaqueRefsV0(claim.EvidenceRefs),
	}

	var issues []WorksetClaimIssueV0
	var readIssues []WorksetClaimIssueV0
	normalized.ReadSet, readIssues = normalizeExistingScopeRefsV0("read_set", claim.ReadSet)
	issues = append(issues, readIssues...)

	var writeIssues []WorksetClaimIssueV0
	normalized.WriteSet, writeIssues = normalizeExistingScopeRefsV0("write_set", claim.WriteSet)
	issues = append(issues, writeIssues...)

	issues = append(issues, validateNormalizedWorksetClaimV0(normalized)...)
	return normalized, issues
}

func ValidateWorksetClaimV0(claim WorksetClaimV0) []WorksetClaimIssueV0 {
	_, issues := NormalizeWorksetClaimV0(claim)
	return issues
}

func normalizeExistingScopeRefsV0(field string, refs []ScopeRefV0) ([]ScopeRefV0, []WorksetClaimIssueV0) {
	raw := make([]string, 0, len(refs))
	for _, ref := range refs {
		raw = append(raw, ref.Ref)
	}
	return normalizeScopeRefsV0(field, raw)
}

func normalizeWorksetClaimSchemaVersionV0(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return WorksetClaimSchemaVersionV0
	}
	return trimmed
}

func normalizeOpaqueRefsV0(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		value := strings.TrimSpace(ref)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}
