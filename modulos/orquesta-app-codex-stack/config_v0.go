package orquestaappcodexstack

import (
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestatoolcapability "orquesta/modulos/orquesta-tool-capability"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type ConfigV0 struct {
	Enabled                      bool
	Clock                        orquestafactoryhttp.AppSpecHTTPClockV0
	Timeout                      time.Duration
	DirectorLimits               orquestaweb.WebArrancarDirectorAppLimitsV0
	AppIntakeAssistant           orquestaweb.WebNuevaAppIntakeAssistantPortV0
	WizardBotAssistant           orquestaweb.WizardBotLLMAssistPortV0
	DirectorDecisionBudget       orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetV0
	Stores                       StoresV0
	RunQueue                     RunQueueConfigV0
	RunSupervisor                RunSupervisorConfigV0
	Codex                        CodexRuntimeConfigV0
	Gemini                       GeminiRuntimeConfigV0
	Claude                       ClaudeRuntimeConfigV0
	EgressSanitizer              EgressSanitizerConfigV0
	Capacity                     CapacityConfigV0
	ReviewGate                   ReviewGateConfigV0
	RequiredTests                orquestacionnucleoapp.RequiredTestRunnerPortV0
	AutonomousDirectorPolicy     orquestacionnucleoapp.AutonomousDirectorPolicyPortV0
	AppGoalLauncher              orquestagoal.GoalWorkLauncherPortV0
	AppGoalReworkLauncher        orquestagoal.GoalWorkLauncherPortV0
	GoalRequiredTestDependencies GoalRequiredTestDependencyResolverPortV0
	AppGoalObserver              orquestagoal.GoalWorkObservationPortV0
	// AppGoalRequiredTestAttestor is opt-in and must execute outside the
	// implementer identity. It is intentionally not a Codex/runtime default.
	AppGoalRequiredTestSpecBinder                  orquestagoal.GoalRequiredTestSpecBinderPortV0
	AppGoalRequiredTestSnapshotObserver            orquestagoal.GoalRequiredTestFinalSnapshotObserverPortV0
	AppGoalRequiredTestAttestor                    orquestagoal.GoalRequiredTestAttestorPortV0
	AppGoalRequiredTestIdentityVerifier            orquestagoal.GoalRequiredTestIdentityVerifierPortV0
	GoalRequiredTestClaimPolicy                    orquestagoal.GoalRequiredTestAttestationClaimPolicyV0
	AppGoalBackendControl                          GoalBackendControlPortV0
	AppGoalClosureValidator                        orquestagoal.GoalWorkClosureValidatorPortV0
	DomainTests                                    DomainWorkRequiredTestConfigV0
	AppChange                                      orquestaappchange.AppChangePortsV0
	AutoprogrammingPromotion                       AutoprogrammingPromotionConfigV0
	AutoprogrammingStatusDiagnostics               []orquestamcp.MCPAutoprogrammingDiagnosticV0
	AutoprogrammingIdleSelfImprovementBudgetSource orquestamcp.MCPAutoprogrammingIdleSelfImprovementBudgetSourceV0
	AutoprogrammingGoalProgressPolicy              orquestamcp.MCPAutoprogrammingGoalProgressPolicyV0
	ConfigProjectionSettings                       []orquestamcp.MCPConfigProjectionSettingV0
	DomainWork                                     orquestamcp.MCPDomainWorkExecutorPortV0
	CodeContext                                    orquestacontext.CodeContextQueryPortV0
	CodeContextToolLeases                          orquestacontext.CodeContextToolLeaseListPortV0
	RuntimeModels                                  orquestaruntime.RuntimeModelManagerPortV0
	ToolCapabilities                               orquestatoolcapability.CapabilityCatalogPortV0
	DocumentTextExtract                            orquestamcp.MCPDocumentTextExtractExtractorPortV0
	DataProfile                                    orquestamcp.MCPDataProfileProfilerPortV0
	Council                                        orquestamcp.MCPCouncilPortV0
	CouncilPublicErrorClassifier                   orquestamcp.MCPCouncilPublicErrorClassifierV0
	CouncilGate                                    CouncilGateConfigV0
	AutonomyProgram                                orquestamcp.MCPAutonomyProgramPortV0
	CouncilDoubleReview                            CouncilDoubleReviewConfigV0
	DecisionCouncil                                DecisionCouncilConfigV0
	DomainDelivery                                 DomainWorkDeliveryBridgeConfigV0
	ExternalWorkRunGuard                           ExternalWorkRunProjectWorkDirGuardConfigV0
	ExternalWorkRunRuntimeGuard                    ExternalWorkRunRuntimeCompatibilityGuardConfigV0
	GoalObserverResidentEnabled                    bool
	AllowLegacyExternalWorkRun                     bool
	AllowLegacyAutoprogrammingRun                  bool
	// PromoteMaterializedArtifactWithoutAck recupera entregas cuyo agente
	// materializo el write-set pero no escribio ACK (con gate-issue de revision
	// humana). Opt-in: por defecto OFF.
	PromoteMaterializedArtifactWithoutAck bool
}

type DomainWorkRequiredTestConfigV0 struct {
	Policy orquestadomainwork.DomainWorkRequiredTestPolicyPortV0
}

type StoresV0 struct {
	RunStore                         orquestacionnucleoapp.RunStorePortV0
	EventSink                        orquestacionnucleoapp.EventSinkPortV0
	OutboxLedger                     OutboxLedgerPortV0
	TaskStore                        orquestaappdirectorservice.AppDirectorWorkflowTaskStorePortV0
	WaitStateStore                   orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0
	OperationalPlanStateWriter       orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0
	OperationalPlanStateStore        orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0
	RequiredTestEvidenceStore        orquestacionnucleoapp.RequiredTestEvidenceStorePortV0
	AppChangeStore                   orquestaappchange.AppChangeRecordStorePortV0
	ReceiptStore                     CodexReceiptStorePortV0
	ProgressState                    orquestaruntimecodexdelivery.CodexProgressStateStorePortV0
	ProcessRegistry                  orquestacionnucleoapp.AgentProcessRegistryPortV0
	RunControl                       orquestaruncontrol.RunControlPortV0
	RunQueue                         orquestarunqueue.RunQueuePortV0
	AppGoalStateStore                orquestagoal.GoalWorkStateStorePortV0
	GoalRequiredTestAttestationStore orquestagoal.GoalRequiredTestAttestationStorePortV0
	AutoprogrammingBatchStore        orquestaautoprogramming.AutoprogrammingBatchStorePortV0
}

type OutboxLedgerPortV0 interface {
	orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	orquestaoutboxdispatch.PendingOutboxReaderPortV0
	orquestaoutboxdispatch.OutboxDispatchClaimerPortV0
	orquestaoutboxdispatch.OutboxDispatchAckPortV0
}

type CodexReceiptStorePortV0 interface {
	orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	orquestaruntimecodexdelivery.CodexReceiptDescriptorRecorderPortV0
}

type CodexRuntimeConfigV0 struct {
	CommandPath              string
	ProjectWorkDir           string
	RuntimeWorkDir           string
	CodeHomeDir              string
	HomeDir                  string
	PathEnv                  string
	Model                    string
	ModelRouting             CodexModelRoutingConfigV0
	ReasoningEffort          string
	Profile                  string
	Sandbox                  string
	ApprovalPolicy           string
	DirectorSandbox          string
	DirectorApprovalPolicy   string
	InteractiveApprovalOptIn bool
	ExtraArgs                []string
	PromptHints              []string
	SkillInstructions        []orquestaruntimecodex.CodexSkillInstructionV0

	Runtime          orquestaruntime.ExternalAgentProcessRuntimePortV0
	ProcessStopper   orquestacionnucleoapp.ProcessRuntimeStopPortV0
	SnapshotSource   orquestaruntimecodexdelivery.CodexProcessSnapshotSourcePortV0
	ContextSanitizer orquestacontext.ContextSanitizerPortV0

	MaxBatchReady  int
	MaxConcurrency int
	WaitInterval   time.Duration
	ProgressPolicy orquestaruntime.AgentProgressHeartbeatPolicyV0
	ProgressBudget orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0
	UsageMetrics   CodexStackAgentUsageMetricsProviderPortV0
}

// CodexModelRoutingConfigV0 is composition data. TaskRoutes are Director
// declarations keyed by task_ref; no title/prompt heuristic is permitted.
type CodexModelRoutingConfigV0 struct {
	Policy     orquestacapacity.ModelRoutingPolicyV0
	ModelAlias map[string]string
	TaskRoutes map[string]orquestacapacity.ModelRoutingRequestV0
}

type GeminiRuntimeConfigV0 struct {
	Enabled        bool
	CommandPath    string
	ProjectWorkDir string
	RuntimeWorkDir string
	HomeDir        string
	PathEnv        string
	Model          string
	ApprovalMode   string
	OutputFormat   string
	PromptLocale   string
	ExtraArgs      []string
	PromptHints    []string
}

type ClaudeRuntimeConfigV0 struct {
	Enabled        bool
	CommandPath    string
	ProjectWorkDir string
	RuntimeWorkDir string
	HomeDir        string
	PathEnv        string
	// Model and Effort are intentionally not configuration fallbacks. Every
	// Claude launch obtains both values from ModelRouting for its own task_ref.
	PermissionMode string
	OutputFormat   string
	ModelRouting   ClaudeModelRoutingConfigV0
	PromptLocale   string
	ExtraArgs      []string
	PromptHints    []string
}

// ClaudeModelRoutingConfigV0 keeps provider selection separate from the
// neutral task-class decision. Model aliases are resolved only by composition.
type ClaudeModelRoutingConfigV0 struct {
	Policy     orquestacapacity.ModelRoutingPolicyV0
	ModelAlias map[string]string
	TaskRoutes map[string]orquestacapacity.ModelRoutingRequestV0
}

type RunQueueConfigV0 struct {
	QueueRef             string
	DefaultPriorityScore int
	QueueLimit           int
	MaxRunsPerTick       int
}

type RunSupervisorConfigV0 struct {
	MaxTicks          int
	MaxExecutions     int
	StopOnNoExecution bool
	AllowRepeatedRuns bool
}

type CapacityConfigV0 struct {
	Tier            orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	ReasoningEffort orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	PolicyRef       string
	PoolRef         string
	ModelRef        string
	QuotaRef        string
	OccurredAt      string
	RequestedBy     string
	Summary         string
	EvidenceRefs    []string
	DecisionPolicy  orquestacionnucleoapp.CapacityDecisionPolicyPortV0
}

type ReviewGateConfigV0 struct {
	FileEvidence            orquestaruntimecodexdelivery.CodexReviewGateFileEvidenceResultProviderPortV0
	MaxLinesPerFile         int
	FailureStatus           orquestacoreworkflow.ReviewResultStatusV0
	StrictGoLineBudget      bool
	LineBudgetSnapshotStore orquestaruntimeworktree.WorktreeSnapshotStorePortV0
	SnapshotReadBudget      orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0
	Policy                  ReviewGateDeliveryPolicyPortV0
}
