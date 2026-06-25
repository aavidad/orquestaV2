package orquestacionnucleoapp

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	DirectorRunStatsSchemaVersionV0  = "director_run_stats.v0"
	DirectorLoopStatsSchemaVersionV0 = "director_loop_stats.v0"
)

const (
	DirectorAgentStatusRequestedV0      = "requested"
	DirectorAgentStatusRunningV0        = "running"
	DirectorAgentStatusCompletedV0      = "completed"
	DirectorAgentStatusFailedV0         = "failed"
	DirectorAgentStatusLostV0           = "lost"
	DirectorAgentStatusStopRequestedV0  = "stop_requested"
	DirectorAgentStatusStoppedV0        = "stopped"
	DirectorAgentStatusControlUnknownV0 = "control_unknown"
)

const (
	DirectorAgentControlStateNotLoadedV0  = "not_loaded"
	DirectorAgentControlStateNotNeededV0  = "not_needed"
	DirectorAgentControlStateRegisteredV0 = "registered"
	DirectorAgentControlStateMissingV0    = "missing"
)

const (
	DirectorAgentStopReasonSourceStopRequestV0 = "agent_stop_request"
	DirectorAgentStopReasonSourceAssessmentV0  = "agent_assessment"
)

const (
	DirectorClosureStatusBlockedV0 = "blocked"
	DirectorClosureStatusReadyV0   = "ready"
	DirectorClosureStatusClosedV0  = "closed"
)

const (
	DirectorClosureBlockedByContratosV0            = "contratos"
	DirectorClosureBlockedByProgramacionEntregasV0 = "programacion_entregas"
	DirectorClosureBlockedByRevisionFinalV0        = "revision_final"
	DirectorClosureBlockedByValidacionFinalV0      = "validacion_final"
	DirectorClosureBlockedByFaseCierreV0           = "fase_cierre"
	DirectorClosureBlockedByRunBlockersV0          = "run_blockers"
)

const (
	DirectorRunStopStatusNoneV0       = "none"
	DirectorRunStopStatusRequestedV0  = "stop_requested"
	DirectorRunStopStatusPropagatedV0 = "stop_propagated"
	DirectorRunStopStatusPendingV0    = "stop_pending"
	DirectorRunStopStatusConfirmedV0  = "stop_confirmed"
)

type DirectorRunStatsV0 struct {
	SchemaVersion string                   `json:"schema_version"`
	RunRef        string                   `json:"run_ref"`
	ProjectRef    string                   `json:"project_ref,omitempty"`
	AppSpecRef    string                   `json:"app_spec_ref,omitempty"`
	Status        string                   `json:"status,omitempty"`
	CurrentPhase  string                   `json:"current_phase,omitempty"`
	Counts        DirectorRunStatsCountsV0 `json:"counts"`
	Refs          DirectorRunStatsRefsV0   `json:"refs,omitempty"`
	Progress      DirectorProgressStatsV0  `json:"progress"`
	Closure       DirectorClosureStatsV0   `json:"closure"`
	StopControl   DirectorRunStopControlV0 `json:"stop_control"`
	UsageSummary  *DirectorRunUsageStatsV0 `json:"usage_summary,omitempty"`
	Phases        []DirectorPhaseStatsV0   `json:"phases,omitempty"`
	Agents        []DirectorAgentStatsV0   `json:"agents,omitempty"`
}

type DirectorRunStatsCountsV0 struct {
	TasksTotal              int `json:"tasks_total"`
	TasksClosed             int `json:"tasks_closed"`
	TasksOpen               int `json:"tasks_open"`
	TasksDelivered          int `json:"tasks_delivered"`
	Brainstorms             int `json:"brainstorms"`
	Votes                   int `json:"votes"`
	FunctionContracts       int `json:"function_contracts"`
	Decisions               int `json:"decisions"`
	CapacityRequests        int `json:"capacity_requests"`
	CapacityDecisions       int `json:"capacity_decisions"`
	AgentsRequested         int `json:"agents_requested"`
	AgentsStarted           int `json:"agents_started"`
	AgentsFailed            int `json:"agents_failed"`
	AgentsLost              int `json:"agents_lost"`
	AgentsStopRequested     int `json:"agents_stop_requested"`
	AgentsStopConfirmed     int `json:"agents_stop_confirmed"`
	AgentsDelivered         int `json:"agents_delivered"`
	AgentsInFlight          int `json:"agents_in_flight"`
	AgentsControlRegistered int `json:"agents_control_registered"`
	AgentsControlMissing    int `json:"agents_control_missing"`
	AgentAssessments        int `json:"agent_assessments"`
	AgentLeaseExpirations   int `json:"agent_lease_expirations"`
	ConcurrencyGates        int `json:"concurrency_gates"`
	QualityGates            int `json:"quality_gates"`
	PhaseArtifacts          int `json:"phase_artifacts"`
	Deliveries              int `json:"deliveries"`
	Reviews                 int `json:"reviews"`
	ReviewResults           int `json:"review_results"`
	ReworkRequests          int `json:"rework_requests"`
	ReplanDecisions         int `json:"replan_decisions"`
	AcceptedReviews         int `json:"accepted_reviews"`
	Validations             int `json:"validations"`
	Closures                int `json:"closures"`
	DirectorQuestions       int `json:"director_questions"`
	DirectorAnswers         int `json:"director_answers"`
	Blockers                int `json:"blockers"`
	CommandEffects          int `json:"command_effects"`
}

type DirectorRunStatsRefsV0 struct {
	OpenTasks             []string `json:"open_tasks,omitempty"`
	ClosedTasks           []string `json:"closed_tasks,omitempty"`
	DeliveredTasks        []string `json:"delivered_tasks,omitempty"`
	Brainstorms           []string `json:"brainstorms,omitempty"`
	Votes                 []string `json:"votes,omitempty"`
	FunctionContracts     []string `json:"function_contracts,omitempty"`
	Decisions             []string `json:"decisions,omitempty"`
	CapacityRequests      []string `json:"capacity_requests,omitempty"`
	CapacityDecisions     []string `json:"capacity_decisions,omitempty"`
	AgentsRequested       []string `json:"agents_requested,omitempty"`
	AgentsStarted         []string `json:"agents_started,omitempty"`
	AgentsFailed          []string `json:"agents_failed,omitempty"`
	AgentsLost            []string `json:"agents_lost,omitempty"`
	AgentsStopRequested   []string `json:"agents_stop_requested,omitempty"`
	AgentStopRequests     []string `json:"agent_stop_requests,omitempty"`
	AgentsStopConfirmed   []string `json:"agents_stop_confirmed,omitempty"`
	AgentsDelivered       []string `json:"agents_delivered,omitempty"`
	AgentAssessments      []string `json:"agent_assessments,omitempty"`
	AgentLeaseExpirations []string `json:"agent_lease_expirations,omitempty"`
	ConcurrencyGates      []string `json:"concurrency_gates,omitempty"`
	QualityGates          []string `json:"quality_gates,omitempty"`
	PhaseArtifacts        []string `json:"phase_artifacts,omitempty"`
	Deliveries            []string `json:"deliveries,omitempty"`
	Reviews               []string `json:"reviews,omitempty"`
	ReviewResults         []string `json:"review_results,omitempty"`
	ReworkRequests        []string `json:"rework_requests,omitempty"`
	ReplanDecisions       []string `json:"replan_decisions,omitempty"`
	AcceptedReviews       []string `json:"accepted_reviews,omitempty"`
	Validations           []string `json:"validations,omitempty"`
	Closures              []string `json:"closures,omitempty"`
	DirectorQuestions     []string `json:"director_questions,omitempty"`
	DirectorAnswers       []string `json:"director_answers,omitempty"`
	DirectorAnswered      []string `json:"director_answered_questions,omitempty"`
	Blockers              []string `json:"blockers,omitempty"`
}

type DirectorPhaseStatsV0 struct {
	PhaseID             string `json:"phase_id"`
	Status              string `json:"status,omitempty"`
	RecommendedCapacity string `json:"recommended_capacity,omitempty"`
	EntryCriteria       int    `json:"entry_criteria"`
	ExitCriteria        int    `json:"exit_criteria"`
	EvidenceRequired    int    `json:"evidence_required"`
}

type DirectorClosureStatsV0 struct {
	Status      string   `json:"status"`
	Blocked     bool     `json:"blocked"`
	Ready       bool     `json:"ready"`
	Closed      bool     `json:"closed"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	BlockerRefs []string `json:"blocker_refs,omitempty"`
}

type DirectorRunStopControlV0 struct {
	Status                 string   `json:"status"`
	Requested              bool     `json:"requested"`
	Propagated             bool     `json:"propagated"`
	Pending                bool     `json:"pending"`
	Confirmed              bool     `json:"confirmed"`
	RunControlStatus       string   `json:"run_control_status,omitempty"`
	CheckpointRecorded     bool     `json:"checkpoint_recorded,omitempty"`
	Forced                 bool     `json:"forced,omitempty"`
	StartedAgents          int      `json:"started_agents"`
	StopRequestedAgents    int      `json:"stop_requested_agents"`
	StopConfirmedAgents    int      `json:"stop_confirmed_agents"`
	StopPendingAgents      int      `json:"stop_pending_agents"`
	PendingAgentRefs       []string `json:"pending_agent_refs,omitempty"`
	StopRequestedAgentRefs []string `json:"stop_requested_agent_refs,omitempty"`
	StopConfirmedAgentRefs []string `json:"stop_confirmed_agent_refs,omitempty"`
	EvidenceRefs           []string `json:"evidence_refs,omitempty"`
}

type DirectorAgentStatsV0 struct {
	AgentRequestID    string                       `json:"agent_request_id"`
	Status            string                       `json:"status"`
	Requested         bool                         `json:"requested"`
	Started           bool                         `json:"started"`
	Failed            bool                         `json:"failed"`
	Lost              bool                         `json:"lost"`
	StopRequested     bool                         `json:"stop_requested"`
	StopConfirmed     bool                         `json:"stop_confirmed"`
	StopReasonCode    string                       `json:"stop_reason_code,omitempty"`
	StopReasonSource  string                       `json:"stop_reason_source,omitempty"`
	StopReasonRef     string                       `json:"stop_reason_ref,omitempty"`
	Completed         bool                         `json:"completed"`
	InFlight          bool                         `json:"in_flight"`
	NeedsAttention    bool                         `json:"needs_attention"`
	ControlRegistered bool                         `json:"control_registered"`
	ControlState      string                       `json:"control_state,omitempty"`
	CanStop           bool                         `json:"can_stop"`
	LastProgress      *DirectorAgentProgressV0     `json:"last_progress,omitempty"`
	Usage             *DirectorAgentUsageStatsV0   `json:"usage,omitempty"`
	Process           *DirectorAgentProcessStatsV0 `json:"process,omitempty"`
}

type DirectorAgentProcessStatsV0 struct {
	ProcessRef   string   `json:"process_ref,omitempty"`
	SessionRef   string   `json:"session_ref,omitempty"`
	LaunchRef    string   `json:"launch_ref,omitempty"`
	ReadinessRef string   `json:"readiness_ref,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type AutonomousDirectorLoopStatsV0 struct {
	SchemaVersion string                         `json:"schema_version"`
	Run           DirectorRunStatsV0             `json:"run"`
	Decision      AutonomousDirectorDecisionV0   `json:"decision"`
	Loop          DirectorProgressiveLoopStatsV0 `json:"loop"`
}

type DirectorProgressiveLoopStatsV0 struct {
	Status              string                                                     `json:"status,omitempty"`
	Bursts              int                                                        `json:"bursts"`
	Dispatches          int                                                        `json:"dispatches"`
	BatchDispatches     int                                                        `json:"batch_dispatches"`
	TotalExecutedSteps  int                                                        `json:"total_executed_steps"`
	FirstPendingCount   int                                                        `json:"first_pending_count"`
	PendingOutboxCount  int                                                        `json:"pending_outbox_count"`
	RecommendedCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0 `json:"recommended_capacity,omitempty"`
}
