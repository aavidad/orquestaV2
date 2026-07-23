package mcpinterface

// Sealed V04 compatibility names. They are passive wire DTOs only: V20 tool
// registration, validation and dispatch derive exclusively from the canonical
// command registry in internal/commands.
const (
	ToolGoalsAmend    = "orquesta.goals.amend"
	ToolGoalsCreate   = "orquesta.goals.create"
	ToolGoalsGet      = "orquesta.goals.get"
	ToolGoalsList     = "orquesta.goals.list"
	ToolArtifactsRead = "orquesta.artifacts.read"
	ToolSystemStatus  = "orquesta.system.status"
)

type CreateGoalInput struct {
	ProjectRef          string `json:"project_ref"`
	RequestRef          string `json:"request_ref,omitempty"`
	Statement           string `json:"statement,omitempty"`
	NormalizedObjective string `json:"normalized_objective,omitempty"`
	Confirm             bool   `json:"confirm,omitempty"`
}

type AmendGoalInput struct {
	ProjectRef             string `json:"project_ref"`
	RequestRef             string `json:"request_ref,omitempty"`
	SourceGoalRef          string `json:"source_goal_ref,omitempty"`
	ExpectedSourceRevision uint64 `json:"expected_source_revision,omitempty"`
	ExpectedSourceSpecHash string `json:"expected_source_spec_hash,omitempty"`
	Statement              string `json:"statement,omitempty"`
	NormalizedObjective    string `json:"normalized_objective,omitempty"`
	Reason                 string `json:"reason,omitempty"`
	Confirm                bool   `json:"confirm,omitempty"`
}

type GetGoalInput struct {
	ProjectRef string `json:"project_ref"`
	GoalRef    string `json:"goal_ref,omitempty"`
}
