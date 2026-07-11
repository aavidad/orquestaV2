package orquestaautoprogramming

type MaterialProgressClassV0 string

const (
	MaterialProgressClassDiffV0    MaterialProgressClassV0 = "diff"
	MaterialProgressClassTestV0    MaterialProgressClassV0 = "test"
	MaterialProgressClassResultV0  MaterialProgressClassV0 = "result"
	MaterialProgressClassReceiptV0 MaterialProgressClassV0 = "receipt"
	MaterialProgressClassNoneV0    MaterialProgressClassV0 = "none"
)

type MaterialProgressActionV0 string

const (
	MaterialProgressActionInvalidV0          MaterialProgressActionV0 = "invalid"
	MaterialProgressActionContinueV0         MaterialProgressActionV0 = "continue"
	MaterialProgressActionWarningV0          MaterialProgressActionV0 = "warning"
	MaterialProgressActionReplanRequiredV0   MaterialProgressActionV0 = "replan_required"
	MaterialProgressActionHardStopRequiredV0 MaterialProgressActionV0 = "hard_stop_required"
)

// MaterialProgressCheckpointV0 is an already classified observation. Adapters
// attest the class; this package never infers it from text, names, or activity.
type MaterialProgressCheckpointV0 struct {
	Sequence           int64                   `json:"sequence"`
	TokensAccumulated  int64                   `json:"tokens_accumulated"`
	ContextRevisionRef string                  `json:"context_revision_ref"`
	MaterialClass      MaterialProgressClassV0 `json:"material_class"`
	EvidenceRefs       []string                `json:"evidence_refs,omitempty"`
}

// MaterialProgressSegmentV0 starts at the last accepted material evidence.
type MaterialProgressSegmentV0 struct {
	StartSequence          int64    `json:"start_sequence"`
	StartTokensAccumulated int64    `json:"start_tokens_accumulated"`
	ContextRevisionRef     string   `json:"context_revision_ref"`
	EvidenceRefs           []string `json:"evidence_refs,omitempty"`
}

type MaterialProgressPolicyV0 struct {
	WarningAfterTokens          int64 `json:"warning_after_tokens"`
	ReplanRequiredAfterTokens   int64 `json:"replan_required_after_tokens"`
	HardStopRequiredAfterTokens int64 `json:"hard_stop_required_after_tokens"`
}

type MaterialProgressInputV0 struct {
	Policy     MaterialProgressPolicyV0     `json:"policy"`
	Segment    MaterialProgressSegmentV0    `json:"segment"`
	Checkpoint MaterialProgressCheckpointV0 `json:"checkpoint"`
}

type MaterialProgressIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type MaterialProgressValidationResultV0 struct {
	Accepted bool                      `json:"accepted"`
	Input    MaterialProgressInputV0   `json:"input"`
	Issues   []MaterialProgressIssueV0 `json:"issues,omitempty"`
}

type MaterialProgressDecisionV0 struct {
	Accepted              bool                      `json:"accepted"`
	Action                MaterialProgressActionV0  `json:"action"`
	MaterialProgressed    bool                      `json:"material_progressed"`
	TokensWithoutMaterial int64                     `json:"tokens_without_material"`
	Segment               MaterialProgressSegmentV0 `json:"segment"`
	Issues                []MaterialProgressIssueV0 `json:"issues,omitempty"`
}
