package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
)

func (service ServiceV0) RunSupervisedBurstV0(
	ctx context.Context,
	request SupervisedBurstRequestV0,
) (SupervisedBurstResultV0, error) {
	request = normalizeSupervisedBurstRequestV0(request)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := service.validateV0(request); err != nil {
		return SupervisedBurstResultV0{}, err
	}

	workflow := storedWorkflowPortV0{
		RunStore:  service.RunStore,
		EventSink: service.EventSink,
	}
	builder := cycleStepInputBuilderV0{
		RunStore:          service.RunStore,
		CandidateProvider: service.CandidateProvider,
		OutboxLedger:      service.OutboxLedger,
		Workflow:          workflow,
		Request:           request,
		MaxCommands:       boundedServiceMaxCommandsV0(service.MaxCommands),
		MaxOutboxPerCycle: boundedServiceMaxOutboxV0(service.MaxOutboxPerCycle),
	}
	burst, burstErr := orquestadirectorsupervisedburst.RunDirectorSupervisedBurstV0(ctx,
		orquestadirectorsupervisedburst.DirectorSupervisedBurstInputV0{
			StepInputBuilder: builder,
			StepExecutor:     service.stepExecutorV0(),
			Supervisor:       service.supervisorV0(),
			RunRef:           request.RunRef,
			MaxSteps:         request.MaxSteps,
			CorrelationID:    request.CorrelationID,
			EvidenceRefs:     request.EvidenceRefs,
		},
	)
	run, loadErr := service.RunStore.LoadRunV0(ctx, request.RunRef)
	result := SupervisedBurstResultV0{Burst: burst, Run: run}
	if burstErr != nil {
		return result, burstErr
	}
	if loadErr != nil {
		return result, errorV0(ErrNucleoOrquestacionStoreV0, "run_store", loadErr.Error())
	}
	return result, nil
}

func normalizeSupervisedBurstRequestV0(
	request SupervisedBurstRequestV0,
) SupervisedBurstRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	return request
}

func (service ServiceV0) validateV0(request SupervisedBurstRequestV0) error {
	if service.RunStore == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido")
	}
	if service.CandidateProvider == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "candidate_provider", "candidate_provider requerido")
	}
	if service.OutboxLedger == nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "outbox_ledger", "outbox_ledger requerido")
	}
	if request.RunRef == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	if request.OccurredAt == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	if request.MaxSteps <= 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "max_steps", "max_steps debe ser mayor que cero")
	}
	return nil
}

func boundedServiceMaxCommandsV0(value int) int {
	return boundedServicePositiveLimitV0(value, orquestadirectorrunner.DirectorCycleMaxCommandsV0)
}

func boundedServiceMaxOutboxV0(value int) int {
	return boundedServicePositiveLimitV0(value, orquestadirectorrunner.DirectorCycleMaxOutboxV0)
}

func boundedServicePositiveLimitV0(value int, max int) int {
	if value > max {
		return max
	}
	return value
}

func (service ServiceV0) stepExecutorV0() orquestadirectorsupervisedburst.DirectorCycleStepExecutorPortV0 {
	if service.StepExecutor != nil {
		return service.StepExecutor
	}
	return orquestadirectorsupervisedburst.DirectDirectorCycleStepExecutorV0{}
}

func (service ServiceV0) supervisorV0() orquestadirectorsupervisedburst.DirectorSupervisorPolicyPortV0 {
	if service.Supervisor != nil {
		return service.Supervisor
	}
	return orquestadirectorsupervisedburst.DirectDirectorSupervisorPolicyV0{}
}
