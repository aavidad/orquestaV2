package orquestacionnucleoapp

import (
	"context"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	ErrNucleoOrquestacionInvalidoV0 = "nucleo_orquestacion_invalido"
	ErrNucleoOrquestacionStoreV0    = "nucleo_orquestacion_store"
)

type RunStorePortV0 interface {
	LoadRunV0(ctx context.Context, runRef string) (orquestacoreworkflow.OrchestrationRunV0, error)
	SaveRunV0(ctx context.Context, run orquestacoreworkflow.OrchestrationRunV0) error
}

type EventSinkPortV0 interface {
	AppendRunEventsV0(
		ctx context.Context,
		runRef string,
		events []orquestacoreworkflow.OrchestrationEventV0,
	) error
}

type RunEventReaderPortV0 interface {
	LoadRunEventsV0(ctx context.Context, runRef string) ([]orquestacoreworkflow.OrchestrationEventV0, error)
}

type CandidateProviderPortV0 interface {
	BuildSchedulerCandidatesV0(
		ctx context.Context,
		request SchedulerCandidateRequestV0,
	) (SchedulerCandidateSetV0, error)
}

type SchedulerCandidateRequestV0 struct {
	Run              orquestacoreworkflow.OrchestrationRunV0
	StepNumber       int
	MaxSteps         int
	OccurredAt       string
	PreviousStep     *orquestadirectorcycle.DirectorCycleStepResultV0
	CorrelationID    string
	EvidenceRefs     []string
	PreviousDecision string
	WaitAgentRefs    []string
}

type SchedulerCandidateSetV0 struct {
	LeaseActionCandidates         []orquestadirectorscheduler.SchedulableLeaseActionCandidateV0
	PhaseArtifactCandidates       []orquestadirectorscheduler.SchedulablePhaseArtifactCandidateV0
	DeliveryCandidates            []orquestadirectorscheduler.SchedulableDeliveryCandidateV0
	ReviewGateCandidates          []orquestadirectorscheduler.SchedulableReviewGateCandidateV0
	ProgressSupervisionCandidates []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0
	ReplanFollowupCandidates      []orquestadirectorscheduler.SchedulableReplanFollowupCandidateV0
	WorkClaims                    []orquestacoreconcurrency.WorksetClaimV0
	WorkCandidates                []orquestadirectorscheduler.SchedulableWorkCandidateV0
	EvidenceRefs                  []string
}

type ServiceV0 struct {
	RunStore           RunStorePortV0
	EventSink          EventSinkPortV0
	CandidateProvider  CandidateProviderPortV0
	RunControl         orquestaruncontrol.RunControlReaderPortV0
	RunControlTerminal orquestaruncontrol.RunControlTerminalWriterPortV0
	OutboxLedger       orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	StepExecutor       orquestadirectorsupervisedburst.DirectorCycleStepExecutorPortV0
	Supervisor         orquestadirectorsupervisedburst.DirectorSupervisorPolicyPortV0
	MaxCommands        int
	MaxOutboxPerCycle  int
}

type SupervisedBurstRequestV0 struct {
	RunRef        string
	OccurredAt    string
	MaxSteps      int
	CorrelationID string
	EvidenceRefs  []string
	WaitAgentRefs []string
}

type SupervisedBurstResultV0 struct {
	Burst orquestadirectorsupervisedburst.DirectorSupervisedBurstResultV0
	Run   orquestacoreworkflow.OrchestrationRunV0
}

type ErrorV0 struct {
	Code    string
	Field   string
	Message string
}

func (err ErrorV0) Error() string {
	return err.Code
}
