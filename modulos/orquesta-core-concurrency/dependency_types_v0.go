package orquestacoreconcurrency

type WorksetDependencyIssueKindV0 string

const (
	WorksetDependencyIssueKindMissingV0   WorksetDependencyIssueKindV0 = "dependency_missing"
	WorksetDependencyIssueKindSelfV0      WorksetDependencyIssueKindV0 = "dependency_self"
	WorksetDependencyIssueKindCycleV0     WorksetDependencyIssueKindV0 = "dependency_cycle"
	WorksetDependencyIssueKindNotClosedV0 WorksetDependencyIssueKindV0 = "dependency_not_closed"
)

type WorksetDependencyIssueV0 struct {
	IssueRef          string                             `json:"issue_ref"`
	ClaimRefs         []string                           `json:"claim_refs"`
	DependencyRefs    []string                           `json:"dependency_refs,omitempty"`
	IssueKind         WorksetDependencyIssueKindV0       `json:"issue_kind"`
	RecommendedAction WorksetConflictRecommendedActionV0 `json:"recommended_action"`
	Summary           string                             `json:"summary"`
	EvidenceRefs      []string                           `json:"evidence_refs,omitempty"`
}

type WorksetDependencyEvaluationV0 struct {
	ReadyClaimRefs   []string                   `json:"ready_claim_refs,omitempty"`
	BlockedClaimRefs []string                   `json:"blocked_claim_refs,omitempty"`
	Issues           []WorksetDependencyIssueV0 `json:"issues,omitempty"`
}
