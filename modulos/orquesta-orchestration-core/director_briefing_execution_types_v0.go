package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

const (
	DirectorBriefingExecutionStatusRunStepV0            = "run_step_executed"
	DirectorBriefingExecutionStatusOutboxDispatchedV0   = "outbox_dispatched"
	DirectorBriefingExecutionStatusOutboxNoProgressV0   = "outbox_no_progress"
	DirectorBriefingExecutionStatusExternalAppliedV0    = "external_action_applied"
	DirectorBriefingExecutionStatusExternalFailedV0     = "external_action_failed"
	DirectorBriefingExecutionStatusExternalPendingV0    = "external_action_pending"
	DirectorBriefingExecutionStatusCloseOrIdlePendingV0 = "close_or_idle_pending"
)

const (
	ErrDirectorBriefingExecutionInvalidV0 = "director_briefing_execution_invalid"
)

type DirectorBriefingExternalActionExecutorPortV0 interface {
	ExecuteDirectorBriefingExternalActionV0(
		ctx context.Context,
		request DirectorBriefingExternalActionRequestV0,
	) (DirectorBriefingExternalActionResultV0, error)
}

type DirectorBriefingExecutionRequestV0 struct {
	Briefing              orquestadirectorsupervisor.DirectorSupervisorBriefingV0
	ActionRef             string
	OccurredAt            string
	CorrelationID         string
	EvidenceRefs          []string
	WaitAgentRefs         []string
	WaitScopeApplied      bool
	MaxDispatchesPerWait  int
	Dispatchers           []OutboxDispatcherBindingV0
	BatchDispatchers      []OutboxBatchDispatcherBindingV0
	ExternalActionHandler DirectorBriefingExternalActionExecutorPortV0
}

type DirectorBriefingExecutionResultV0 struct {
	Status          string                                                           `json:"status"`
	RunRef          string                                                           `json:"run_ref,omitempty"`
	Action          orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0 `json:"action"`
	Burst           *orquestadirectorsupervisedburst.DirectorSupervisedBurstResultV0 `json:"burst,omitempty"`
	Run             *orquestacoreworkflow.OrchestrationRunV0                         `json:"run,omitempty"`
	Dispatches      []OutboxDispatchOnceResultV0                                     `json:"dispatches,omitempty"`
	BatchDispatches []OutboxDispatchBatchRunResultV0                                 `json:"batch_dispatches,omitempty"`
	External        *DirectorBriefingExternalActionResultV0                          `json:"external,omitempty"`
	EvidenceRefs    []string                                                         `json:"evidence_refs,omitempty"`
}

type DirectorBriefingExternalActionRequestV0 struct {
	Briefing      orquestadirectorsupervisor.DirectorSupervisorBriefingV0
	Action        orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0
	OccurredAt    string
	CorrelationID string
	EvidenceRefs  []string
}

type DirectorBriefingExternalActionResultV0 struct {
	Status       string   `json:"status"`
	RunRef       string   `json:"run_ref,omitempty"`
	ActionRef    string   `json:"action_ref,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
