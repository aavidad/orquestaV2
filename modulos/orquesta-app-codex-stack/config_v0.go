package orquestaappcodexstack

import (
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type ConfigV0 struct {
	Enabled        bool
	Clock          orquestafactoryhttp.AppSpecHTTPClockV0
	Timeout        time.Duration
	DirectorLimits orquestaweb.WebArrancarDirectorAppLimitsV0
	Stores         StoresV0
	RunQueue       RunQueueConfigV0
	RunSupervisor  RunSupervisorConfigV0
	Codex          CodexRuntimeConfigV0
	Capacity       CapacityConfigV0
	ReviewGate     ReviewGateConfigV0
	RequiredTests  orquestacionnucleoapp.RequiredTestRunnerPortV0
	AppChange      orquestaappchange.AppChangePortsV0
	DomainWork     orquestamcp.MCPDomainWorkExecutorPortV0
	DomainDelivery DomainWorkDeliveryBridgeConfigV0
}

type StoresV0 struct {
	RunStore                   orquestacionnucleoapp.RunStorePortV0
	EventSink                  orquestacionnucleoapp.EventSinkPortV0
	OutboxLedger               OutboxLedgerPortV0
	TaskStore                  orquestaappdirectorservice.AppDirectorWorkflowTaskStorePortV0
	WaitStateStore             orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0
	OperationalPlanStateWriter orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0
	OperationalPlanStateStore  orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0
	RequiredTestEvidenceStore  orquestacionnucleoapp.RequiredTestEvidenceStorePortV0
	AppChangeStore             orquestaappchange.AppChangeRecordStorePortV0
	ReceiptStore               CodexReceiptStorePortV0
	ProgressState              orquestaruntimecodexdelivery.CodexProgressStateStorePortV0
	ProcessRegistry            orquestacionnucleoapp.AgentProcessRegistryPortV0
	RunControl                 orquestaruncontrol.RunControlPortV0
	RunQueue                   orquestarunqueue.RunQueuePortV0
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
	CommandPath            string
	ProjectWorkDir         string
	RuntimeWorkDir         string
	CodeHomeDir            string
	HomeDir                string
	PathEnv                string
	Model                  string
	ReasoningEffort        string
	Profile                string
	Sandbox                string
	ApprovalPolicy         string
	DirectorSandbox        string
	DirectorApprovalPolicy string
	ExtraArgs              []string
	PromptHints            []string

	Runtime        orquestaruntime.ExternalAgentProcessRuntimePortV0
	ProcessStopper orquestacionnucleoapp.ProcessRuntimeStopPortV0
	SnapshotSource orquestaruntimecodexdelivery.CodexProcessSnapshotSourcePortV0

	MaxBatchReady  int
	MaxConcurrency int
	WaitInterval   time.Duration
	ProgressPolicy orquestaruntime.AgentProgressHeartbeatPolicyV0
	ProgressBudget orquestaruntimecodexdelivery.CodexBudgetActivityPolicyV0
	UsageMetrics   CodexStackAgentUsageMetricsProviderPortV0
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
	OccurredAt      string
	RequestedBy     string
	Summary         string
	EvidenceRefs    []string
}

type ReviewGateConfigV0 struct {
	FileEvidence    orquestaruntimecodexdelivery.CodexReviewGateFileEvidenceResultProviderPortV0
	MaxLinesPerFile int
	FailureStatus   orquestacoreworkflow.ReviewResultStatusV0
}
