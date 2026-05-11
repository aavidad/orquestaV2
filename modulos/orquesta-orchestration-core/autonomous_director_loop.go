package orquestacionnucleoapp

import "context"

func (service ServiceV0) RunAutonomousDirectorLoopV0(
	ctx context.Context,
	request AutonomousDirectorLoopRequestV0,
) (AutonomousDirectorLoopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if service.RunStore == nil {
		return AutonomousDirectorLoopResultV0{},
			errorV0(ErrNucleoOrquestacionInvalidoV0, "run_store", "run_store requerido")
	}
	loop := normalizeProgressiveLoopRequestV0(request.Loop)
	run, err := service.RunStore.LoadRunV0(ctx, loop.RunRef)
	if err != nil {
		return AutonomousDirectorLoopResultV0{}, err
	}
	policy := request.Policy
	if policy == nil {
		policy = HeuristicAutonomousDirectorPolicyV0{}
	}
	decision, err := policy.DecideAutonomousDirectorV0(ctx, AutonomousDirectorDecisionInputV0{
		Run:    run,
		Limits: request.Limits,
		Requests: AutonomousDirectorRequestHintsV0{
			CorrelationID: loop.CorrelationID,
			EvidenceRefs:  loop.EvidenceRefs,
		},
	})
	if err != nil {
		return AutonomousDirectorLoopResultV0{}, err
	}
	tunedService, tunedLoop := applyAutonomousDirectorDecisionV0(service, loop, decision)
	result, err := tunedService.RunProgressiveLoopV0(ctx, tunedLoop)
	return AutonomousDirectorLoopResultV0{
		Decision: decision,
		Loop:     result,
		Stats:    BuildAutonomousDirectorLoopStatsV0(decision, result),
	}, err
}

func applyAutonomousDirectorDecisionV0(
	service ServiceV0,
	request ProgressiveLoopRequestV0,
	decision AutonomousDirectorDecisionV0,
) (ServiceV0, ProgressiveLoopRequestV0) {
	if decision.MaxCommandsPerCycle > 0 {
		service.MaxCommands = decision.MaxCommandsPerCycle
	}
	if decision.MaxOutboxPerCycle > 0 {
		service.MaxOutboxPerCycle = decision.MaxOutboxPerCycle
	}
	if decision.MaxBursts > 0 {
		request.MaxBursts = decision.MaxBursts
	}
	if decision.MaxStepsPerBurst > 0 {
		request.MaxStepsPerBurst = decision.MaxStepsPerBurst
	}
	if decision.MaxDispatchesPerWait > 0 {
		request.MaxDispatchesPerWait = decision.MaxDispatchesPerWait
	}
	request.BatchDispatchers = tuneAutonomousBatchDispatchersV0(
		request.BatchDispatchers,
		decision.MaxParallelAgents,
	)
	return service, request
}

func tuneAutonomousBatchDispatchersV0(
	dispatchers []OutboxBatchDispatcherBindingV0,
	maxParallelAgents int,
) []OutboxBatchDispatcherBindingV0 {
	if len(dispatchers) == 0 || maxParallelAgents <= 0 {
		return append([]OutboxBatchDispatcherBindingV0(nil), dispatchers...)
	}
	tuned := make([]OutboxBatchDispatcherBindingV0, 0, len(dispatchers))
	for _, dispatcher := range dispatchers {
		dispatcher.MaxReady = maxParallelAgents
		if tuner, ok := dispatcher.Executor.(AutonomousBatchExecutorTunerV0); ok {
			dispatcher.Executor = tuner.WithAutonomousBatchConcurrencyV0(maxParallelAgents)
		}
		tuned = append(tuned, dispatcher)
	}
	return tuned
}
