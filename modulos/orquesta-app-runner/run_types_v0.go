package orquestaapprunner

import (
	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	AppOrchestrationRunResultSchemaVersionV0 = "app_orchestration_run_result.v0"
	AppOrchestrationRunStatusRunningV0       = "running"
	AppOrchestrationRunStatusCompleteV0      = "complete"
)

type RunPreparedAppOrchestrationRequestV0 struct {
	Prepared                  AppOrchestrationPreparedV0 `json:"prepared"`
	OccurredAt                string                     `json:"occurred_at"`
	CorrelationID             string                     `json:"correlation_id,omitempty"`
	RequestedBy               string                     `json:"requested_by,omitempty"`
	UseAutonomousDirectorLoop bool                       `json:"use_autonomous_director_loop,omitempty"`
	MaxBursts                 int                        `json:"max_bursts,omitempty"`
	MaxStepsPerBurst          int                        `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait      int                        `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands               int                        `json:"max_commands,omitempty"`
	MaxOutboxPerCycle         int                        `json:"max_outbox_per_cycle,omitempty"`
	MaxExternalWaits          int                        `json:"max_external_waits,omitempty"`
}

type RunPreparedAppOrchestrationPortsV0 struct {
	RunStore                 orquestacionnucleoapp.RunStorePortV0
	EventSink                orquestacionnucleoapp.EventSinkPortV0
	OutboxLedger             orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	Dispatchers              []orquestacionnucleoapp.OutboxDispatcherBindingV0
	BatchDispatchers         []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0
	DeliverySource           orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0
	ReviewGateSource         orquestacionnucleoapp.ReviewGateObservationProviderPortV0
	ProgressSource           orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	LeaseSource              orquestacionnucleoapp.AgentLeaseAssessmentProviderPortV0
	AssessmentReplanSource   orquestacionnucleoapp.AgentAssessmentReplanPlanProviderPortV0
	ExternalWaiter           orquestacionnucleoapp.ExternalProgressWaiterPortV0
	AutonomousDirectorPolicy orquestacionnucleoapp.AutonomousDirectorPolicyPortV0
}

type AppOrchestrationRunResultV0 struct {
	SchemaVersion     string                                               `json:"schema_version"`
	Status            string                                               `json:"status"`
	Run               orquestacoreworkflow.OrchestrationRunV0              `json:"run"`
	Progress          orquestaappplanner.AppPlanProgressV0                 `json:"progress"`
	LoopStatus        orquestacionnucleoapp.ProgressiveLoopStatusV0        `json:"loop_status"`
	StartedAgents     []string                                             `json:"started_agents,omitempty"`
	Attempts          int                                                  `json:"attempts"`
	ExternalWaits     int                                                  `json:"external_waits"`
	DirectorLoopStats *orquestacionnucleoapp.AutonomousDirectorLoopStatsV0 `json:"director_loop_stats,omitempty"`
	EvidenceRefs      []string                                             `json:"evidence_refs,omitempty"`
}
