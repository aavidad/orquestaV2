package orquestaappdirectorservice

import (
	"context"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func runPreparedDirectorAutonomyLoopV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	loop, err := runPreparedDirectorLoopV0(ctx, request, ports, prepared)
	if err != nil || ports.DirectorDecisionSource == nil {
		return loop, err
	}
	for cycle := 0; cycle < request.MaxDecisionCycles; cycle++ {
		next, progressed, err := consumeStartAppDirectorDecisionsV0(
			ctx,
			request,
			ports,
			loop,
		)
		if err != nil || !progressed {
			return next, err
		}
		loop, err = runPreparedDirectorLoopV0(ctx, request, ports, prepared)
		if err != nil {
			return loop, err
		}
	}
	return loop, nil
}

func runPreparedDirectorLoopV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	service := startAppDirectorLoopServiceV0(request, ports, prepared)
	loopRequest, err := startAppDirectorLoopRequestV0(ctx, request, ports, prepared)
	if err != nil {
		return orquestacionnucleoapp.ProgressiveLoopResultV0{}, err
	}
	if ports.ExternalWaiter == nil {
		return service.RunProgressiveLoopV0(ctx, loopRequest)
	}
	managed, err := service.RunManagedProgressiveLoopV0(
		ctx,
		orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
			Loop:             loopRequest,
			ExternalWaiter:   ports.ExternalWaiter,
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	return managed.Final, err
}

func startAppDirectorLoopServiceV0(
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) orquestacionnucleoapp.ServiceV0 {
	provider := composeStartAppDirectorProviderV0(
		prepared.CandidateProvider,
		ports,
		request.RequestedBy,
	)
	return orquestacionnucleoapp.ServiceV0{
		RunStore:           ports.RunStore,
		EventSink:          ports.EventSink,
		CandidateProvider:  provider,
		RunControl:         ports.RunControl,
		RunControlTerminal: ports.RunControlTerminal,
		OutboxLedger:       ports.OutboxLedger,
		MaxCommands:        request.MaxCommands,
		MaxOutboxPerCycle:  request.MaxOutboxPerCycle,
	}
}

func startAppDirectorLoopRequestV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) (orquestacionnucleoapp.ProgressiveLoopRequestV0, error) {
	waitResolution, err := appDirectorResolvedWaitV0(
		ctx,
		prepared.Run.RunID,
		request.WaitAgentRefs,
		appDirectorWaitFilterV0{
			CohortRef:     request.WaitCohortRef,
			WaveRef:       request.WaitWaveRef,
			ParentTaskRef: request.WaitParentTaskRef,
		},
		ports,
		appDirectorWaitStateMetaV0{
			OccurredAt:       request.OccurredAt,
			CorrelationID:    request.CorrelationID,
			EvidenceRefs:     prepared.EvidenceRefs,
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	if err != nil {
		return orquestacionnucleoapp.ProgressiveLoopRequestV0{}, err
	}
	return orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               prepared.Run.RunID,
		OccurredAt:           request.OccurredAt,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		WaitAgentRefs:        waitResolution.AgentRefs,
		WaitScopeApplied:     waitResolution.ScopeApplied,
		CorrelationID:        request.CorrelationID,
		EvidenceRefs:         prepared.EvidenceRefs,
		Dispatchers:          ports.Dispatchers,
		BatchDispatchers:     ports.BatchDispatchers,
	}, nil
}
