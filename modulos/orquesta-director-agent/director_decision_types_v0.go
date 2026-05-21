package orquestadirectoragent

const (
	DirectorAgentDecisionSchemaVersionV0  = "director_agent_decision.v0"
	DirectorAgentMicrotaskSchemaVersionV0 = "workflow_task.v0"
	DirectorAgentPlanningPhaseIDV0        = "planificacion_microtareas"

	DirectorAgentCommandRequestBrainstormV0       = "request_brainstorm"
	DirectorAgentCommandOpenPhaseV0               = "open_phase"
	DirectorAgentCommandRequestVoteV0             = "request_vote"
	DirectorAgentCommandAcceptDecisionV0          = "accept_decision"
	DirectorAgentCommandPublishContractV0         = "publish_function_contract"
	DirectorAgentCommandCreateMicrotaskV0         = "create_microtask"
	DirectorAgentCommandAskDirectorV0             = "ask_director"
	DirectorAgentCommandAskUserV0                 = "ask_user"
	DirectorAgentCommandRequestCapacityV0         = "request_capacity"
	DirectorAgentCommandRequestAgentV0            = "request_agent"
	DirectorAgentCommandRequestReviewV0           = "request_review"
	DirectorAgentCommandRecordReviewResultV0      = "record_review_result"
	DirectorAgentCommandAcceptReviewV0            = "accept_review"
	DirectorAgentCommandRequestReworkV0           = "request_rework"
	DirectorAgentCommandRecordReplanDecisionV0    = "record_replan_decision"
	DirectorAgentCommandCloseTaskV0               = "close_task"
	DirectorAgentCommandRegisterFinalValidationV0 = "register_final_validation"
	DirectorAgentCommandCloseRunV0                = "close_run"
	DirectorAgentCommandProposePlanTeamV0         = "propose_autonomous_plan_team"
	DirectorAgentCommandAnswerQuestionV0          = "answer_director_question"

	DirectorAgentCapacityLowV0    = "low"
	DirectorAgentCapacityMediumV0 = "medium"
	DirectorAgentCapacityHighV0   = "high"
	DirectorAgentCapacityXHighV0  = "xhigh"

	DirectorAgentAnswerContinueV0  = "continue"
	DirectorAgentAnswerReplanV0    = "replan"
	DirectorAgentAnswerStopAgentV0 = "stop_agent"

	DirectorAgentReviewStatusAcceptedV0         = "accepted"
	DirectorAgentReviewStatusChangesRequestedV0 = "changes_requested"
	DirectorAgentReviewStatusRejectedV0         = "rejected"

	DirectorAgentReplanActionSplitTaskV0        = "split_task"
	DirectorAgentReplanActionRetryTaskV0        = "retry_task"
	DirectorAgentReplanActionReplaceAgentV0     = "replace_agent"
	DirectorAgentReplanActionEscalateCapacityV0 = "escalate_capacity"
	DirectorAgentReplanActionAskDirectorV0      = "ask_director"
	DirectorAgentReplanActionAbortTaskV0        = "abort_task"

	DirectorAgentTargetDirectorV0 = "director"
	DirectorAgentTargetUserV0     = "user"

	DirectorAgentAutonomousPlanTeamSchemaVersionV0 = "director_agent_autonomous_plan_team.v0"
	DirectorAgentCompactStatsSchemaVersionV0       = "director_agent_compact_stats.v0"
)

type DirectorAgentDecisionV0 struct {
	SchemaVersion           string                                 `json:"schema_version"`
	DecisionRef             string                                 `json:"decision_ref"`
	RunID                   string                                 `json:"run_id"`
	PhaseID                 string                                 `json:"phase_id"`
	CommandType             string                                 `json:"command_type"`
	CommandRef              string                                 `json:"command_ref"`
	Summary                 string                                 `json:"summary"`
	EvidenceRefs            []string                               `json:"evidence_refs,omitempty"`
	RequestBrainstorm       *DirectorAgentBrainstormCommandV0      `json:"request_brainstorm,omitempty"`
	OpenPhase               *DirectorAgentOpenPhaseCommandV0       `json:"open_phase,omitempty"`
	RequestVote             *DirectorAgentVoteCommandV0            `json:"request_vote,omitempty"`
	AcceptDecision          *DirectorAgentAcceptDecisionCommandV0  `json:"accept_decision,omitempty"`
	PublishContract         *DirectorAgentPublishContractCommandV0 `json:"publish_function_contract,omitempty"`
	CreateMicrotask         *DirectorAgentCreateMicrotaskCommandV0 `json:"create_microtask,omitempty"`
	AskDirector             *DirectorAgentAskQuestionCommandV0     `json:"ask_director,omitempty"`
	AskUser                 *DirectorAgentAskQuestionCommandV0     `json:"ask_user,omitempty"`
	RequestCapacity         *DirectorAgentRequestCapacityCommandV0 `json:"request_capacity,omitempty"`
	RequestAgent            *DirectorAgentRequestAgentCommandV0    `json:"request_agent,omitempty"`
	RequestReview           *DirectorAgentRequestReviewCommandV0   `json:"request_review,omitempty"`
	RecordReviewResult      *DirectorAgentReviewResultCommandV0    `json:"record_review_result,omitempty"`
	AcceptReview            *DirectorAgentAcceptReviewCommandV0    `json:"accept_review,omitempty"`
	RequestRework           *DirectorAgentRequestReworkCommandV0   `json:"request_rework,omitempty"`
	RecordReplanDecision    *DirectorAgentReplanDecisionCommandV0  `json:"record_replan_decision,omitempty"`
	CloseTask               *DirectorAgentCloseTaskCommandV0       `json:"close_task,omitempty"`
	RegisterFinalValidation *DirectorAgentFinalValidationCommandV0 `json:"register_final_validation,omitempty"`
	CloseRun                *DirectorAgentCloseRunCommandV0        `json:"close_run,omitempty"`
	ProposePlanTeam         *DirectorAgentPlanTeamCommandV0        `json:"propose_autonomous_plan_team,omitempty"`
	AnswerQuestion          *DirectorAgentAnswerQuestionCommandV0  `json:"answer_director_question,omitempty"`
}

type DirectorAgentBrainstormCommandV0 struct {
	BrainstormRequestID        string   `json:"brainstorm_request_id"`
	PhaseID                    string   `json:"phase_id"`
	TopicRef                   string   `json:"topic_ref"`
	Summary                    string   `json:"summary"`
	MinimumRecommendedCapacity string   `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentOpenPhaseCommandV0 struct {
	PhaseID string `json:"phase_id"`
	Reason  string `json:"reason,omitempty"`
}

type DirectorAgentVoteCommandV0 struct {
	VoteRequestID              string   `json:"vote_request_id"`
	PhaseID                    string   `json:"phase_id"`
	DecisionTopicRef           string   `json:"decision_topic_ref"`
	BrainstormRef              string   `json:"brainstorm_ref"`
	Summary                    string   `json:"summary"`
	MinimumRecommendedCapacity string   `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentAcceptDecisionCommandV0 struct {
	DecisionRef       string   `json:"decision_ref"`
	PhaseID           string   `json:"phase_id"`
	VoteRef           string   `json:"vote_ref"`
	AcceptedOptionRef string   `json:"accepted_option_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentPublishContractCommandV0 struct {
	ContractRef   string   `json:"contract_ref"`
	PhaseID       string   `json:"phase_id"`
	DecisionRef   string   `json:"decision_ref"`
	Summary       string   `json:"summary"`
	FunctionNames []string `json:"function_names,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentCreateMicrotaskCommandV0 struct {
	Task DirectorAgentMicrotaskV0 `json:"task"`
}

type DirectorAgentRequestReviewCommandV0 struct {
	ReviewRequestID string   `json:"review_request_id"`
	PhaseID         string   `json:"phase_id"`
	DeliveryRef     string   `json:"delivery_ref"`
	Summary         string   `json:"summary"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentReviewResultCommandV0 struct {
	ReviewResultRef string   `json:"review_result_ref"`
	ReviewRequestID string   `json:"review_request_id"`
	DeliveryRef     string   `json:"delivery_ref"`
	Status          string   `json:"status"`
	Summary         string   `json:"summary"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
	QualityGateRef  string   `json:"quality_gate_ref,omitempty"`
}

type DirectorAgentAcceptReviewCommandV0 struct {
	AcceptedReviewRef string   `json:"accepted_review_ref"`
	PhaseID           string   `json:"phase_id"`
	ReviewRequestID   string   `json:"review_request_id"`
	DeliveryRef       string   `json:"delivery_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentCloseTaskCommandV0 struct {
	TaskID            string   `json:"task_id"`
	PhaseID           string   `json:"phase_id"`
	DeliveryRef       string   `json:"delivery_ref"`
	AcceptedReviewRef string   `json:"accepted_review_ref"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentFinalValidationCommandV0 struct {
	ValidationRef       string   `json:"validation_ref"`
	PhaseID             string   `json:"phase_id"`
	ClosedTaskRef       string   `json:"closed_task_ref"`
	Summary             string   `json:"summary"`
	RequestKind         string   `json:"request_kind,omitempty"`
	ExecutionMode       string   `json:"execution_mode,omitempty"`
	MinimumDeliverables []string `json:"minimum_deliverables,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentCloseRunCommandV0 struct {
	ClosureRef          string   `json:"closure_ref"`
	PhaseID             string   `json:"phase_id"`
	ValidationRef       string   `json:"validation_ref"`
	Summary             string   `json:"summary"`
	RequestKind         string   `json:"request_kind,omitempty"`
	ExecutionMode       string   `json:"execution_mode,omitempty"`
	MinimumDeliverables []string `json:"minimum_deliverables,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentAnswerQuestionCommandV0 struct {
	AnswerID     string   `json:"answer_id"`
	QuestionID   string   `json:"question_id"`
	Decision     string   `json:"decision"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Unblocks     bool     `json:"unblocks"`
}

type DirectorAgentMicrotaskV0 struct {
	SchemaVersion        string                               `json:"schema_version"`
	TaskID               string                               `json:"task_id"`
	RunID                string                               `json:"run_id"`
	PhaseID              string                               `json:"phase_id"`
	WorkProfileKind      string                               `json:"work_profile_kind,omitempty"`
	Title                string                               `json:"title"`
	Summary              string                               `json:"summary,omitempty"`
	WriteSet             []string                             `json:"write_set"`
	AcceptanceCriteria   []string                             `json:"acceptance_criteria"`
	RequiredTests        []string                             `json:"required_tests,omitempty"`
	DependsOn            []string                             `json:"depends_on,omitempty"`
	ParentTaskRef        string                               `json:"parent_task_ref,omitempty"`
	CohortRef            string                               `json:"cohort_ref,omitempty"`
	WaveRef              string                               `json:"wave_ref,omitempty"`
	DelegationDepth      int                                  `json:"delegation_depth,omitempty"`
	MaxChildAgents       int                                  `json:"max_child_agents,omitempty"`
	ChildTaskRefs        []string                             `json:"child_task_refs,omitempty"`
	FunctionContractRefs []DirectorAgentFunctionContractRefV0 `json:"function_contract_refs,omitempty"`
}

type DirectorAgentFunctionContractRefV0 struct {
	ContractRef  string `json:"contract_ref,omitempty"`
	FunctionName string `json:"function_name,omitempty"`
}

type DirectorAgentPlanTeamCommandV0 struct {
	Plan DirectorAgentAutonomousPlanTeamV0 `json:"plan"`
}

type DirectorAgentAutonomousPlanTeamV0 struct {
	SchemaVersion string                              `json:"schema_version"`
	PlanRef       string                              `json:"plan_ref"`
	RunID         string                              `json:"run_id"`
	PhaseID       string                              `json:"phase_id"`
	GoalRef       string                              `json:"goal_ref"`
	Summary       string                              `json:"summary"`
	Team          []DirectorAgentTeamMemberV0         `json:"team"`
	WorkUnits     []DirectorAgentAutonomousWorkUnitV0 `json:"work_units"`
	EvidenceRefs  []string                            `json:"evidence_refs,omitempty"`
}

type DirectorAgentTeamMemberV0 struct {
	MemberRef          string   `json:"member_ref"`
	Role               string   `json:"role"`
	Capacity           string   `json:"capacity,omitempty"`
	ResponsibilityRefs []string `json:"responsibility_refs,omitempty"`
}

type DirectorAgentAutonomousWorkUnitV0 struct {
	WorkUnitRef          string                               `json:"work_unit_ref"`
	PhaseID              string                               `json:"phase_id"`
	WorkProfileKind      string                               `json:"work_profile_kind,omitempty"`
	Title                string                               `json:"title"`
	Summary              string                               `json:"summary"`
	AssignedMemberRef    string                               `json:"assigned_member_ref"`
	WriteSet             []string                             `json:"write_set"`
	AcceptanceCriteria   []string                             `json:"acceptance_criteria"`
	FunctionContractRefs []DirectorAgentFunctionContractRefV0 `json:"function_contract_refs,omitempty"`
	DependsOn            []string                             `json:"depends_on,omitempty"`
}

type DirectorAgentDecisionIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}
