package orquestadirectoroperativo

type OperationalDirectorModeV0 string

const (
	OperationalDirectorModeProgrammingV0 OperationalDirectorModeV0 = "programming"
	OperationalDirectorModeDomainWorkV0  OperationalDirectorModeV0 = "domain_work"
)

type OperationalDirectorContextStatusV0 string

const (
	OperationalDirectorContextSufficientV0   OperationalDirectorContextStatusV0 = "sufficient"
	OperationalDirectorContextInsufficientV0 OperationalDirectorContextStatusV0 = "insufficient"
)

type OperationalDirectorPlanStatusV0 string

const (
	OperationalDirectorPlanReadyV0        OperationalDirectorPlanStatusV0 = "ready"
	OperationalDirectorPlanNeedsContextV0 OperationalDirectorPlanStatusV0 = "needs_context"
)

type OperationalDirectorStepKindV0 string

const (
	OperationalDirectorStepGatherContextV0        OperationalDirectorStepKindV0 = "gather_context"
	OperationalDirectorStepRequestDomainContextV0 OperationalDirectorStepKindV0 = "request_domain_context"
	OperationalDirectorStepSplitWorkV0            OperationalDirectorStepKindV0 = "split_work"
	OperationalDirectorStepLaunchSubagentsV0      OperationalDirectorStepKindV0 = "launch_subagents"
	OperationalDirectorStepWaitSubagentsV0        OperationalDirectorStepKindV0 = "wait_subagents"
	OperationalDirectorStepGovernDelegationV0     OperationalDirectorStepKindV0 = "govern_delegation"
	OperationalDirectorStepReviewDeliveriesV0     OperationalDirectorStepKindV0 = "review_deliveries"
	OperationalDirectorStepRunRequiredTestsV0     OperationalDirectorStepKindV0 = "run_required_tests"
	OperationalDirectorStepReplanOrCloseV0        OperationalDirectorStepKindV0 = "replan_or_close"
)

type OperationalDirectorStepStatusV0 string

const (
	OperationalDirectorStepPendingV0          OperationalDirectorStepStatusV0 = "pending"
	OperationalDirectorStepRunningV0          OperationalDirectorStepStatusV0 = "running"
	OperationalDirectorStepBlockedV0          OperationalDirectorStepStatusV0 = "blocked"
	OperationalDirectorStepChangesRequestedV0 OperationalDirectorStepStatusV0 = "changes_requested"
	OperationalDirectorStepAcceptedV0         OperationalDirectorStepStatusV0 = "accepted"
	OperationalDirectorStepClosedV0           OperationalDirectorStepStatusV0 = "closed"
)

const (
	DefaultOperationalDirectorMaxLoopsV0             = 6
	DefaultOperationalDirectorMaxParallelAgentsV0    = 3
	DefaultOperationalDirectorMaxDelegationDepthV0   = 2
	DefaultOperationalDirectorMaxSubagentsPerAgentV0 = 3
	MaxOperationalDirectorMaxLoopsV0                 = 12
	MaxOperationalDirectorMaxParallelAgentsV0        = 6
	MaxOperationalDirectorMaxDelegationDepthV0       = 3
	MaxOperationalDirectorMaxSubagentsPerAgentV0     = 6
	MaxOperationalDirectorMaxRecursiveAgentsV0       = 4096
)

type OperationalDirectorRequestV0 struct {
	RequestRef               string                             `json:"request_ref"`
	RunRef                   string                             `json:"run_ref"`
	ProjectRef               string                             `json:"project_ref"`
	Objective                string                             `json:"objective"`
	Mode                     OperationalDirectorModeV0          `json:"mode"`
	ContextStatus            OperationalDirectorContextStatusV0 `json:"context_status,omitempty"`
	DomainRefs               []string                           `json:"domain_refs,omitempty"`
	MissingContext           []string                           `json:"missing_context,omitempty"`
	WorktreeRef              string                             `json:"worktree_ref,omitempty"`
	WorktreeIsolated         bool                               `json:"worktree_isolated,omitempty"`
	BranchRef                string                             `json:"branch_ref,omitempty"`
	WriteSet                 []string                           `json:"write_set,omitempty"`
	RequiredTests            []string                           `json:"required_tests,omitempty"`
	MaxLoops                 int                                `json:"max_loops,omitempty"`
	MaxParallelAgents        int                                `json:"max_parallel_agents,omitempty"`
	AllowRecursiveDelegation bool                               `json:"allow_recursive_delegation,omitempty"`
	MaxDelegationDepth       int                                `json:"max_delegation_depth,omitempty"`
	MaxSubagentsPerAgent     int                                `json:"max_subagents_per_agent,omitempty"`
	MaxRecursiveAgents       int                                `json:"max_recursive_agents,omitempty"`
}

type OperationalDirectorPlanResultV0 struct {
	Accepted      bool                         `json:"accepted"`
	ReadyToLaunch bool                         `json:"ready_to_launch"`
	Blocked       bool                         `json:"blocked"`
	Plan          OperationalDirectorPlanV0    `json:"plan,omitempty"`
	Issues        []OperationalDirectorIssueV0 `json:"issues,omitempty"`
}

type OperationalDirectorPlanV0 struct {
	PlanRef              string                          `json:"plan_ref"`
	RequestRef           string                          `json:"request_ref"`
	RunRef               string                          `json:"run_ref"`
	ProjectRef           string                          `json:"project_ref"`
	Mode                 OperationalDirectorModeV0       `json:"mode"`
	Status               OperationalDirectorPlanStatusV0 `json:"status"`
	Objective            string                          `json:"objective"`
	LoopBudget           int                             `json:"loop_budget"`
	MaxParallelAgents    int                             `json:"max_parallel_agents"`
	RecursiveDelegation  bool                            `json:"recursive_delegation,omitempty"`
	MaxDelegationDepth   int                             `json:"max_delegation_depth,omitempty"`
	MaxSubagentsPerAgent int                             `json:"max_subagents_per_agent,omitempty"`
	MaxRecursiveAgents   int                             `json:"max_recursive_agents,omitempty"`
	WriteSet             []string                        `json:"write_set,omitempty"`
	RequiredTests        []string                        `json:"required_tests,omitempty"`
	DomainRefs           []string                        `json:"domain_refs,omitempty"`
	MissingContext       []string                        `json:"missing_context,omitempty"`
	Steps                []OperationalDirectorStepV0     `json:"steps"`
}

type OperationalDirectorStepV0 struct {
	StepID             string                          `json:"step_id"`
	Kind               OperationalDirectorStepKindV0   `json:"kind"`
	Status             OperationalDirectorStepStatusV0 `json:"status"`
	Title              string                          `json:"title"`
	WorkProfileKind    string                          `json:"work_profile_kind,omitempty"`
	ParentStepID       string                          `json:"parent_step_id,omitempty"`
	DelegationDepth    int                             `json:"delegation_depth,omitempty"`
	MaxChildAgents     int                             `json:"max_child_agents,omitempty"`
	DependsOn          []string                        `json:"depends_on,omitempty"`
	ChildStepIDs       []string                        `json:"child_step_ids,omitempty"`
	DomainRefs         []string                        `json:"domain_refs,omitempty"`
	WriteSet           []string                        `json:"write_set,omitempty"`
	RequiredTests      []string                        `json:"required_tests,omitempty"`
	AcceptanceCriteria []string                        `json:"acceptance_criteria,omitempty"`
	EvidenceRefs       []string                        `json:"evidence_refs,omitempty"`
}

type OperationalDirectorIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type OperationalDirectorAgentBudgetV0 struct {
	MaxAgents       int  `json:"max_agents,omitempty"`
	PlannedAgents   int  `json:"planned_agents"`
	Exceeded        bool `json:"exceeded,omitempty"`
	RecursiveLaunch bool `json:"recursive_launch,omitempty"`
}

type OperationalDirectorWaveWorkV0 struct {
	PlanRef            string                          `json:"plan_ref"`
	RequestRef         string                          `json:"request_ref"`
	RunRef             string                          `json:"run_ref"`
	ProjectRef         string                          `json:"project_ref"`
	Mode               OperationalDirectorModeV0       `json:"mode"`
	Status             OperationalDirectorPlanStatusV0 `json:"status"`
	ReadyToLaunch      bool                            `json:"ready_to_launch"`
	MaxParallelItems   int                             `json:"max_parallel_items"`
	RecursiveChildren  bool                            `json:"recursive_children,omitempty"`
	MaxRecursiveAgents int                             `json:"max_recursive_agents,omitempty"`
	Waves              []OperationalDirectorWaveV0     `json:"waves,omitempty"`
	Issues             []OperationalDirectorIssueV0    `json:"issues,omitempty"`
}

type OperationalDirectorWaveV0 struct {
	WaveID    string                          `json:"wave_id"`
	Index     int                             `json:"index"`
	DependsOn []string                        `json:"depends_on,omitempty"`
	Items     []OperationalDirectorWorkItemV0 `json:"items"`
}

type OperationalDirectorWorkItemV0 struct {
	ItemID             string                        `json:"item_id"`
	SourceStepID       string                        `json:"source_step_id"`
	Kind               OperationalDirectorStepKindV0 `json:"kind"`
	Title              string                        `json:"title"`
	WorkProfileKind    string                        `json:"work_profile_kind,omitempty"`
	DependsOn          []string                      `json:"depends_on,omitempty"`
	ParentItemID       string                        `json:"parent_item_id,omitempty"`
	DelegationDepth    int                           `json:"delegation_depth,omitempty"`
	MaxChildItems      int                           `json:"max_child_items,omitempty"`
	ChildItemIDs       []string                      `json:"child_item_ids,omitempty"`
	DomainRefs         []string                      `json:"domain_refs,omitempty"`
	WriteSet           []string                      `json:"write_set,omitempty"`
	RequiredTests      []string                      `json:"required_tests,omitempty"`
	AcceptanceCriteria []string                      `json:"acceptance_criteria,omitempty"`
	EvidenceRefs       []string                      `json:"evidence_refs,omitempty"`
}
