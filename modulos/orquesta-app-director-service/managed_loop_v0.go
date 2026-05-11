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
	loopRequest := startAppDirectorLoopRequestV0(request, ports, prepared)
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
		RunStore:          ports.RunStore,
		EventSink:         ports.EventSink,
		CandidateProvider: provider,
		RunControl:        ports.RunControl,
		OutboxLedger:      ports.OutboxLedger,
		MaxCommands:       request.MaxCommands,
		MaxOutboxPerCycle: request.MaxOutboxPerCycle,
	}
}

func startAppDirectorLoopRequestV0(
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
) orquestacionnucleoapp.ProgressiveLoopRequestV0 {
	return orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               prepared.Run.RunID,
		OccurredAt:           request.OccurredAt,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		CorrelationID:        request.CorrelationID,
		EvidenceRefs:         prepared.EvidenceRefs,
		Dispatchers:          ports.Dispatchers,
		BatchDispatchers:     ports.BatchDispatchers,
	}
}
