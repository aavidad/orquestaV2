package orquestaappdirectorservice

import (
	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	StartAppDirectorResultSchemaVersionV0 = "start_app_director_result.v0"
	StartAppDirectorStatusStartedV0       = "started"
	StartAppDirectorStatusPendingV0       = "pending"
	StartAppDirectorStatusInvalidV0       = "invalid"
)

type StartAppDirectorRequestV0 struct {
	RunRef               string                           `json:"run_ref,omitempty"`
	ProjectRef           string                           `json:"project_ref,omitempty"`
	OccurredAt           string                           `json:"occurred_at,omitempty"`
	CorrelationID        string                           `json:"correlation_id,omitempty"`
	RequestedBy          string                           `json:"requested_by,omitempty"`
	AppSpecRequest       orquestafactory.AppSpecRequestV0 `json:"app_spec_request"`
	MaxBursts            int                              `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int                              `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int                              `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int                              `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int                              `json:"max_outbox_per_cycle,omitempty"`
	MaxDecisionCycles    int                              `json:"max_decision_cycles,omitempty"`
	MaxExternalWaits     int                              `json:"max_external_waits,omitempty"`
}

type StartAppDirectorPortsV0 struct {
	RunStore                 orquestacionnucleoapp.RunStorePortV0
	EventSink                orquestacionnucleoapp.EventSinkPortV0
	OutboxLedger             orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	Dispatchers              []orquestacionnucleoapp.OutboxDispatcherBindingV0
	BatchDispatchers         []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0
	DeliverySource           orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0
	ReviewGateSource         orquestacionnucleoapp.ReviewGateObservationProviderPortV0
	ReviewReworkReplanSource orquestacionnucleoapp.ReviewReworkReplanPlanProviderPortV0
	ProgressSource           orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	RunControl               orquestaruncontrol.RunControlReaderPortV0
	LeaseSource              orquestacionnucleoapp.AgentLeaseAssessmentProviderPortV0
	AssessmentReplanSource   orquestacionnucleoapp.AgentAssessmentReplanPlanProviderPortV0
	DirectorDecisionSource   orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0
	DirectorTaskStore        AppDirectorWorkflowTaskStorePortV0
	ExternalWaiter           orquestacionnucleoapp.ExternalProgressWaiterPortV0
}

type AppDirectorWorkflowTaskStorePortV0 interface {
	orquestadirectoragentworkflow.DirectorAgentWorkflowTaskStorePortV0
	orquestacionnucleoapp.WorkflowTaskStorePortV0
}

type StartAppDirectorResultV0 struct {
	SchemaVersion    string                                        `json:"schema_version"`
	Status           string                                        `json:"status"`
	CorrelationID    string                                        `json:"correlation_id,omitempty"`
	AppSpec          orquestafactory.AppSpecV0                     `json:"app_spec,omitempty"`
	Run              orquestacoreworkflow.OrchestrationRunV0       `json:"run,omitempty"`
	DirectorTask     orquestaappdirectorintake.AppDirectorTaskV0   `json:"director_task,omitempty"`
	DirectorTasks    []orquestaappdirectorintake.AppDirectorTaskV0 `json:"director_tasks,omitempty"`
	LoopStatus       orquestacionnucleoapp.ProgressiveLoopStatusV0 `json:"loop_status,omitempty"`
	StartedAgents    []string                                      `json:"started_agents,omitempty"`
	ValidationIssues []orquestafactory.ValidationIssue             `json:"validation_issues,omitempty"`
	EvidenceRefs     []string                                      `json:"evidence_refs,omitempty"`
}

type AppDirectorServiceIssueV0 struct {
	Field string `json:"field"`
}

func (issue AppDirectorServiceIssueV0) Error() string {
	return "app_director_service_invalido: " + issue.Field
}
