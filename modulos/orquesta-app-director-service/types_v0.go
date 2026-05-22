package orquestaappdirectorservice

import (
	"context"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	StartAppDirectorResultSchemaVersionV0 = "start_app_director_result.v0"
	StartAppDirectorStatusStartedV0       = "started"
	StartAppDirectorStatusPendingV0       = "pending"
	StartAppDirectorStatusInvalidV0       = "invalid"
)

type StartAppDirectorRequestV0 struct {
	RunRef                                  string                                               `json:"run_ref,omitempty"`
	ProjectRef                              string                                               `json:"project_ref,omitempty"`
	OccurredAt                              string                                               `json:"occurred_at,omitempty"`
	CorrelationID                           string                                               `json:"correlation_id,omitempty"`
	RequestedBy                             string                                               `json:"requested_by,omitempty"`
	AppSpecRequest                          orquestafactory.AppSpecRequestV0                     `json:"app_spec_request"`
	MaxBursts                               int                                                  `json:"max_bursts,omitempty"`
	MaxStepsPerBurst                        int                                                  `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait                    int                                                  `json:"max_dispatches_per_wait,omitempty"`
	WaitAgentRefs                           []string                                             `json:"wait_agent_refs,omitempty"`
	WaitCohortRef                           string                                               `json:"wait_cohort_ref,omitempty"`
	WaitWaveRef                             string                                               `json:"wait_wave_ref,omitempty"`
	WaitParentTaskRef                       string                                               `json:"wait_parent_task_ref,omitempty"`
	MaxCommands                             int                                                  `json:"max_commands,omitempty"`
	MaxOutboxPerCycle                       int                                                  `json:"max_outbox_per_cycle,omitempty"`
	MaxDecisionCycles                       int                                                  `json:"max_decision_cycles,omitempty"`
	MaxExternalWaits                        int                                                  `json:"max_external_waits,omitempty"`
	OperationalDirectorPlanRef              string                                               `json:"operational_director_plan_ref,omitempty"`
	OperationalDirectorPlan                 orquestadirectoroperativo.OperationalDirectorPlanV0  `json:"operational_director_plan,omitempty"`
	OperationalDirectorFunctionContractRefs []orquestacoreworkflow.WorkflowFunctionContractRefV0 `json:"operational_director_function_contract_refs,omitempty"`
	OperationalDirectorTargetPhaseID        orquestacoreworkflow.OrchestrationPhaseIDV0          `json:"operational_director_target_phase_id,omitempty"`
	OperationalDirectorMaxItems             int                                                  `json:"operational_director_max_items,omitempty"`
}

type StartAppDirectorPortsV0 struct {
	RunStore                    orquestacionnucleoapp.RunStorePortV0
	EventSink                   orquestacionnucleoapp.EventSinkPortV0
	EventReader                 orquestacionnucleoapp.RunEventReaderPortV0
	OutboxLedger                orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	Dispatchers                 []orquestacionnucleoapp.OutboxDispatcherBindingV0
	BatchDispatchers            []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0
	DeliverySource              orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0
	ReviewGateSource            orquestacionnucleoapp.ReviewGateObservationProviderPortV0
	ReviewReworkReplanSource    orquestacionnucleoapp.ReviewReworkReplanPlanProviderPortV0
	ProgressSource              orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	RunControl                  orquestaruncontrol.RunControlReaderPortV0
	RunControlTerminal          orquestaruncontrol.RunControlTerminalWriterPortV0
	LeaseSource                 orquestacionnucleoapp.AgentLeaseAssessmentProviderPortV0
	AssessmentReplanSource      orquestacionnucleoapp.AgentAssessmentReplanPlanProviderPortV0
	DirectorDecisionSource      orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0
	DirectorTaskStore           AppDirectorWorkflowTaskStorePortV0
	WorkflowTaskProfileResolver orquestacionnucleoapp.WorkflowTaskProfileResolverPortV0
	WorkflowTaskDefaultCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	WaitStateWriter             orquestacionnucleoapp.WorkflowTaskWaitStateWriterPortV0
	OperationalPlanStateWriter  orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0
	OperationalPlanStateStore   orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0
	RequiredTestEvidenceStore   orquestacionnucleoapp.RequiredTestEvidenceReaderPortV0
	RequiredTestRunner          orquestacionnucleoapp.RequiredTestRunnerPortV0
	ExternalWaiter              orquestacionnucleoapp.ExternalProgressWaiterPortV0
	OperationalClosureSource    AppDirectorOperationalClosureSourcePortV0
}

type AppDirectorWorkflowTaskStorePortV0 interface {
	orquestadirectoragentworkflow.DirectorAgentWorkflowTaskStorePortV0
	orquestacionnucleoapp.WorkflowTaskStorePortV0
}

type AppDirectorOperationalClosureSourcePortV0 interface {
	BuildOperationalDirectorClosureRequestV0(
		ctx context.Context,
		request AppDirectorOperationalClosureRequestV0,
	) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error)
}

type AppDirectorOperationalClosureRequestV0 struct {
	Run                      orquestacoreworkflow.OrchestrationRunV0
	LoopStatus               orquestacionnucleoapp.ProgressiveLoopStatusV0
	OccurredAt               string
	CorrelationID            string
	RequestedBy              string
	WaitAgentRefs            []string
	WaitScopeApplied         bool
	WaitCohortRef            string
	WaitWaveRef              string
	WaitParentTaskRef        string
	RequiredTestEvidenceRefs []string
	EvidenceRefs             []string
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
