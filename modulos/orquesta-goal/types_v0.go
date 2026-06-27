package orquestagoal

import "context"

const (
	GoalWorkSpecSchemaV0          = "orquesta_goal_work_spec.v0"
	GoalWorkLaunchReceiptSchemaV0 = "orquesta_goal_launch_receipt.v0"
	GoalWorkResultSchemaV0        = "orquesta_goal_work_result.v0"
	GoalWorkStateSchemaV0         = "orquesta_goal_work_state.v0"

	GoalDirectorKindRuntimeGoalV0 = "runtime_goal"
	GoalDirectorKindCodexGoalV0   = "codex_goal"

	GoalRuleEnforcementAdvisoryV0 = "advisory"
	GoalRuleEnforcementHardV0     = "hard"

	GoalStatusAcceptedV0 = "accepted"
	GoalStatusInvalidV0  = "invalid"
	GoalStatusRunningV0  = "running"
	GoalStatusCompleteV0 = "complete"
	GoalStatusBlockedV0  = "blocked"
)

const (
	ErrGoalRefRequiredV0       = "goal_ref_required"
	ErrGoalRefInvalidV0        = "goal_ref_invalid"
	ErrGoalObjectiveRequiredV0 = "goal_objective_required"
	ErrGoalDirectorRequiredV0  = "goal_director_kind_required"
	ErrGoalDirectorInvalidV0   = "goal_director_kind_invalid"
	ErrGoalWriteSetRequiredV0  = "goal_write_set_required"
	ErrGoalWriteSetInvalidV0   = "goal_write_set_invalid"
	ErrGoalRefFieldInvalidV0   = "goal_ref_field_invalid"
	ErrGoalRuleInvalidV0       = "goal_rule_invalid"
	ErrGoalStatusInvalidV0     = "goal_status_invalid"
	ErrGoalClosureInvalidV0    = "goal_closure_invalid"
)

type GoalWorkSpecV0 struct {
	SchemaVersion      string                   `json:"schema_version"`
	GoalRef            string                   `json:"goal_ref"`
	RequestRef         string                   `json:"request_ref,omitempty"`
	RunRef             string                   `json:"run_ref,omitempty"`
	ProjectRef         string                   `json:"project_ref,omitempty"`
	DomainRef          string                   `json:"domain_ref,omitempty"`
	WorkKind           string                   `json:"work_kind,omitempty"`
	WorkProfileKind    string                   `json:"work_profile_kind,omitempty"`
	Objective          string                   `json:"objective"`
	DirectorKind       string                   `json:"director_kind"`
	ContextRefs        []GoalContextRefV0       `json:"context_refs,omitempty"`
	RuleRefs           []GoalRuleRefV0          `json:"rule_refs,omitempty"`
	SkillRefs          []string                 `json:"skill_refs,omitempty"`
	WriteSet           []GoalWriteScopeV0       `json:"write_set,omitempty"`
	RequiredTests      []GoalRequiredTestV0     `json:"required_tests,omitempty"`
	AcceptanceCriteria []string                 `json:"acceptance_criteria,omitempty"`
	ArtifactContracts  []GoalArtifactContractV0 `json:"artifact_contracts,omitempty"`
	EvidenceRefs       []string                 `json:"evidence_refs,omitempty"`
	Budget             GoalBudgetV0             `json:"budget,omitempty"`
	ClosurePolicy      GoalClosurePolicyV0      `json:"closure_policy,omitempty"`
	ReworkPolicy       GoalReworkPolicyV0       `json:"rework_policy,omitempty"`
}

type GoalContextRefV0 struct {
	Kind     string `json:"kind,omitempty"`
	Ref      string `json:"ref"`
	Purpose  string `json:"purpose,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type GoalRuleRefV0 struct {
	Kind        string `json:"kind,omitempty"`
	Ref         string `json:"ref"`
	Enforcement string `json:"enforcement,omitempty"`
}

type GoalWriteScopeV0 struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose,omitempty"`
}

type GoalRequiredTestV0 struct {
	TestRef                string   `json:"test_ref"`
	CommandRef             string   `json:"command_ref,omitempty"`
	Command                string   `json:"command,omitempty"`
	AcceptanceCriteria     []string `json:"acceptance_criteria,omitempty"`
	AcceptanceCriteriaRefs []string `json:"acceptance_criteria_refs,omitempty"`
	EvidenceRefs           []string `json:"evidence_refs,omitempty"`
}

type GoalArtifactContractV0 struct {
	ArtifactRef  string   `json:"artifact_ref"`
	ArtifactType string   `json:"artifact_type"`
	Required     bool     `json:"required,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type GoalBudgetV0 struct {
	TokenBudget       int `json:"token_budget,omitempty"`
	MaxRuntimeSeconds int `json:"max_runtime_seconds,omitempty"`
	MaxReworkGoals    int `json:"max_rework_goals,omitempty"`
	MaxSubgoals       int `json:"max_subgoals,omitempty"`
}

type GoalClosurePolicyV0 struct {
	RequireRequiredTests bool     `json:"require_required_tests,omitempty"`
	RequireArtifacts     bool     `json:"require_artifacts,omitempty"`
	RequireDomainReceipt bool     `json:"require_domain_receipt,omitempty"`
	RequiredEvidenceRefs []string `json:"required_evidence_refs,omitempty"`
}

type GoalReworkPolicyV0 struct {
	PreferNewGoal     bool `json:"prefer_new_goal,omitempty"`
	MaxReworkGoals    int  `json:"max_rework_goals,omitempty"`
	PreserveArtifacts bool `json:"preserve_artifacts,omitempty"`
}

type GoalWorkIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type GoalLaunchReceiptV0 struct {
	SchemaVersion   string            `json:"schema_version"`
	Status          string            `json:"status"`
	GoalRef         string            `json:"goal_ref,omitempty"`
	ExternalGoalRef string            `json:"external_goal_ref,omitempty"`
	EvidenceRefs    []string          `json:"evidence_refs,omitempty"`
	Issues          []GoalWorkIssueV0 `json:"issues,omitempty"`
}

type GoalWorkResultV0 struct {
	SchemaVersion       string                     `json:"schema_version"`
	Status              string                     `json:"status"`
	GoalRef             string                     `json:"goal_ref,omitempty"`
	ExternalGoalRef     string                     `json:"external_goal_ref,omitempty"`
	Summary             string                     `json:"summary,omitempty"`
	ArtifactRefs        []string                   `json:"artifact_refs,omitempty"`
	RequiredTestResults []GoalRequiredTestResultV0 `json:"required_test_results,omitempty"`
	DomainReceiptRefs   []string                   `json:"domain_receipt_refs,omitempty"`
	EvidenceRefs        []string                   `json:"evidence_refs,omitempty"`
	Issues              []GoalWorkIssueV0          `json:"issues,omitempty"`
}

type GoalWorkStateV0 struct {
	SchemaVersion   string                   `json:"schema_version"`
	RunRef          string                   `json:"run_ref"`
	GoalRef         string                   `json:"goal_ref"`
	ExternalGoalRef string                   `json:"external_goal_ref,omitempty"`
	Status          string                   `json:"status"`
	Spec            GoalWorkSpecV0           `json:"spec"`
	LaunchReceipt   GoalLaunchReceiptV0      `json:"launch_receipt"`
	LastResult      *GoalWorkResultV0        `json:"last_result,omitempty"`
	LastClosure     *GoalClosureValidationV0 `json:"last_closure,omitempty"`
	EvidenceRefs    []string                 `json:"evidence_refs,omitempty"`
}

type GoalWorkStateListRequestV0 struct {
	RunRefs    []string `json:"run_refs,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	ActiveOnly bool     `json:"active_only,omitempty"`
	MaxItems   int      `json:"max_items,omitempty"`
}

type GoalRequiredTestResultV0 struct {
	TestRef      string   `json:"test_ref"`
	Status       string   `json:"status"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type GoalObservationRequestV0 struct {
	GoalRef         string `json:"goal_ref"`
	ExternalGoalRef string `json:"external_goal_ref,omitempty"`
}

type GoalClosureValidationV0 struct {
	Status       string            `json:"status"`
	Accepted     bool              `json:"accepted,omitempty"`
	NeedsRework  bool              `json:"needs_rework,omitempty"`
	EvidenceRefs []string          `json:"evidence_refs,omitempty"`
	Issues       []GoalWorkIssueV0 `json:"issues,omitempty"`
}

type GoalWorkLauncherPortV0 interface {
	LaunchGoalWorkV0(context.Context, GoalWorkSpecV0) (GoalLaunchReceiptV0, error)
}

type GoalWorkObservationPortV0 interface {
	ObserveGoalWorkV0(context.Context, GoalObservationRequestV0) (GoalWorkResultV0, error)
}

type GoalWorkClosureValidatorPortV0 interface {
	ValidateGoalWorkClosureV0(context.Context, GoalWorkSpecV0, GoalWorkResultV0) (GoalClosureValidationV0, error)
}

type GoalWorkStateStorePortV0 interface {
	SaveGoalWorkStateV0(context.Context, GoalWorkStateV0) error
	LoadGoalWorkStateV0(context.Context, string) (GoalWorkStateV0, error)
}

type GoalWorkStateListPortV0 interface {
	ListGoalWorkStatesV0(context.Context, GoalWorkStateListRequestV0) ([]GoalWorkStateV0, error)
}
