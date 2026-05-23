package orquestaappdirectorintake

const HumanDirectorWorkIntakeSchemaVersionV0 = "human_director_work_intake.v0"
const HumanDirectorReviewablePlanSchemaVersionV0 = "human_director_reviewable_plan.v0"

const (
	HumanDirectorPlanStatusReadyForReviewV0 = "ready_for_review"
	HumanDirectorPlanStatusInvalidV0        = "invalid"

	HumanDirectorPlanActionExecuteNowV0      = "execute_now"
	HumanDirectorPlanActionStudyBeforeV0     = "study_before"
	HumanDirectorPlanActionPostponeOverlapV0 = "postpone_overlap"
	HumanDirectorPlanActionRequestReviewV0   = "request_review"
)

type HumanDirectorWorkIntakeRequestV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	RequestRef    string                     `json:"request_ref"`
	ProjectRef    string                     `json:"project_ref"`
	WorktreeRef   string                     `json:"worktree_ref,omitempty"`
	BranchRef     string                     `json:"branch_ref,omitempty"`
	OccurredAt    string                     `json:"occurred_at,omitempty"`
	CorrelationID string                     `json:"correlation_id,omitempty"`
	RequestedBy   string                     `json:"requested_by,omitempty"`
	Request       HumanDirectorWorkRequestV0 `json:"request"`
	Rules         []string                   `json:"rules,omitempty"`
	Limits        HumanDirectorWorkLimitsV0  `json:"limits,omitempty"`
	Hints         HumanDirectorWorkHintsV0   `json:"hints,omitempty"`
	ContextRefs   []string                   `json:"context_refs,omitempty"`
}

type HumanDirectorWorkRequestV0 struct {
	Title              string   `json:"title,omitempty"`
	Objective          string   `json:"objective"`
	Context            []string `json:"context,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	CompactRules       []string `json:"compact_rules,omitempty"`
}

type HumanDirectorWorkLimitsV0 struct {
	MaxAgents          int `json:"max_agents,omitempty"`
	MaxDepth           int `json:"max_depth,omitempty"`
	MaxFanout          int `json:"max_fanout,omitempty"`
	MaxSteps           int `json:"max_steps,omitempty"`
	MaxWriteSetEntries int `json:"max_write_set_entries,omitempty"`
}

type HumanDirectorWorkHintsV0 struct {
	Areas           []string `json:"areas,omitempty"`
	WriteSet        []string `json:"write_set,omitempty"`
	OpaqueRefs      []string `json:"opaque_refs,omitempty"`
	CanRepairSafely []string `json:"can_repair_safely,omitempty"`
	ReviewNeeded    []string `json:"review_needed,omitempty"`
	DeferredReason  string   `json:"deferred_reason,omitempty"`
}

type HumanDirectorReviewablePlanResultV0 struct {
	Accepted bool                          `json:"accepted"`
	Plan     HumanDirectorReviewablePlanV0 `json:"plan"`
	Issues   []HumanDirectorPlanIssueV0    `json:"issues,omitempty"`
}

type HumanDirectorReviewablePlanV0 struct {
	SchemaVersion  string                    `json:"schema_version"`
	RequestRef     string                    `json:"request_ref"`
	ProjectRef     string                    `json:"project_ref"`
	WorktreeRef    string                    `json:"worktree_ref,omitempty"`
	BranchRef      string                    `json:"branch_ref,omitempty"`
	Status         string                    `json:"status"`
	ReviewRequired bool                      `json:"review_required"`
	Goal           string                    `json:"goal,omitempty"`
	Rules          []string                  `json:"rules,omitempty"`
	Limits         HumanDirectorWorkLimitsV0 `json:"limits,omitempty"`
	ContextRefs    []string                  `json:"context_refs,omitempty"`
	Steps          []HumanDirectorPlanStepV0 `json:"steps"`
	EvidenceRefs   []string                  `json:"evidence_refs,omitempty"`
}

type HumanDirectorPlanStepV0 struct {
	StepRef            string   `json:"step_ref"`
	Action             string   `json:"action"`
	Title              string   `json:"title"`
	Objective          string   `json:"objective,omitempty"`
	Area               string   `json:"area,omitempty"`
	WriteSet           []string `json:"write_set,omitempty"`
	DependsOnStepRefs  []string `json:"depends_on_step_refs,omitempty"`
	Reason             string   `json:"reason,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	ContextRefs        []string `json:"context_refs,omitempty"`
	SafeRepairAllowed  bool     `json:"safe_repair_allowed,omitempty"`
}

type HumanDirectorPlanIssueV0 struct {
	Code   string `json:"code"`
	Field  string `json:"field"`
	Detail string `json:"detail,omitempty"`
}
