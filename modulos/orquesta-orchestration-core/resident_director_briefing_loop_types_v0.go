package orquestacionnucleoapp

import (
	"context"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

const (
	ResidentDirectorBriefingLoopStatusCompletedV0       = "completed"
	ResidentDirectorBriefingLoopStatusNeedsDirectorV0   = "needs_director"
	ResidentDirectorBriefingLoopStatusExternalPendingV0 = "external_pending"
	ResidentDirectorBriefingLoopStatusExternalFailedV0  = "external_failed"
	ResidentDirectorBriefingLoopStatusNoProgressV0      = "no_progress"
	ResidentDirectorBriefingLoopStatusBudgetExhaustedV0 = "budget_exhausted"
)

const (
	ErrResidentDirectorBriefingLoopInvalidV0 = "resident_director_briefing_loop_invalid"
)

type ResidentDirectorBriefingSourcePortV0 interface {
	BuildResidentDirectorBriefingV0(
		ctx context.Context,
		request ResidentDirectorBriefingBuildRequestV0,
	) (orquestadirectorsupervisor.DirectorSupervisorBriefingV0, error)
}

type ResidentDirectorBriefingBuildRequestV0 struct {
	RunRef          string                              `json:"run_ref"`
	ObjectiveRef    string                              `json:"objective_ref,omitempty"`
	ContextRefs     []string                            `json:"context_refs,omitempty"`
	StepNumber      int                                 `json:"step_number"`
	PreviousStep    *ResidentDirectorBriefingLoopStepV0 `json:"previous_step,omitempty"`
	CorrelationID   string                              `json:"correlation_id,omitempty"`
	EvidenceRefs    []string                            `json:"evidence_refs,omitempty"`
	LastExecutionID string                              `json:"last_execution_id,omitempty"`
}

type ResidentDirectorBriefingLoopRequestV0 struct {
	RunRef                string
	ObjectiveRef          string
	ContextRefs           []string
	OccurredAt            string
	CorrelationID         string
	EvidenceRefs          []string
	MaxActions            int
	MaxDispatchesPerWait  int
	WaitAgentRefs         []string
	WaitScopeApplied      bool
	BriefingSource        ResidentDirectorBriefingSourcePortV0
	Dispatchers           []OutboxDispatcherBindingV0
	BatchDispatchers      []OutboxBatchDispatcherBindingV0
	ExternalActionHandler DirectorBriefingExternalActionExecutorPortV0
}

type ResidentDirectorBriefingLoopResultV0 struct {
	Status          string                                                   `json:"status"`
	RunRef          string                                                   `json:"run_ref"`
	ExecutedActions int                                                      `json:"executed_actions"`
	Steps           []ResidentDirectorBriefingLoopStepV0                     `json:"steps,omitempty"`
	LastBriefing    *orquestadirectorsupervisor.DirectorSupervisorBriefingV0 `json:"last_briefing,omitempty"`
	LastExecution   *DirectorBriefingExecutionResultV0                       `json:"last_execution,omitempty"`
	EvidenceRefs    []string                                                 `json:"evidence_refs,omitempty"`
}

type ResidentDirectorBriefingLoopStepV0 struct {
	StepNumber int                                                     `json:"step_number"`
	Briefing   orquestadirectorsupervisor.DirectorSupervisorBriefingV0 `json:"briefing"`
	Execution  *DirectorBriefingExecutionResultV0                      `json:"execution,omitempty"`
}
