package orquestaapprunner

import (
	"context"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func runPreparedAppLoopV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
) (appRunnerLoopResultV0, error) {
	service := runPreparedAppServiceV0(request, ports)
	loopRequest := runPreparedAppLoopRequestV0(request, ports)
	if request.UseAutonomousDirectorLoop {
		return runPreparedAutonomousAppLoopV0(ctx, request, ports, service, loopRequest)
	}
	if ports.ExternalWaiter == nil {
		loop, err := service.RunProgressiveLoopV0(ctx, loopRequest)
		return appRunnerLoopResultV0{
			Managed: managedResultFromProgressiveLoopV0(loop),
		}, err
	}
	managed, err := service.RunManagedProgressiveLoopV0(
		ctx,
		orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
			Loop:             loopRequest,
			ExternalWaiter:   ports.ExternalWaiter,
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	return appRunnerLoopResultV0{Managed: managed}, err
}

func runPreparedAppServiceV0(
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
) orquestacionnucleoapp.ServiceV0 {
	return orquestacionnucleoapp.ServiceV0{
		RunStore:  ports.RunStore,
		EventSink: ports.EventSink,
		CandidateProvider: composePreparedAppProviderV0(
			basePreparedAppProviderV0(request),
			ports,
			request.RequestedBy,
		),
		OutboxLedger:      ports.OutboxLedger,
		MaxCommands:       request.MaxCommands,
		MaxOutboxPerCycle: request.MaxOutboxPerCycle,
	}
}

func runPreparedAppLoopRequestV0(
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
) orquestacionnucleoapp.ProgressiveLoopRequestV0 {
	return orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               request.Prepared.Run.RunID,
		OccurredAt:           request.OccurredAt,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		CorrelationID:        request.CorrelationID,
		EvidenceRefs:         request.Prepared.EvidenceRefs,
		Dispatchers:          ports.Dispatchers,
		BatchDispatchers:     ports.BatchDispatchers,
	}
}

func managedResultFromProgressiveLoopV0(
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) orquestacionnucleoapp.ManagedProgressiveLoopResultV0 {
	return orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: loop.Status,
		Final:  loop,
		Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
			AttemptNumber: 1,
			Result:        loop,
		}},
	}
}

type appRunnerLoopResultV0 struct {
	Managed           orquestacionnucleoapp.ManagedProgressiveLoopResultV0
	DirectorLoopStats *orquestacionnucleoapp.AutonomousDirectorLoopStatsV0
}

func runPreparedAutonomousAppLoopV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
	service orquestacionnucleoapp.ServiceV0,
	loopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0,
) (appRunnerLoopResultV0, error) {
	autonomousRequest := runPreparedAutonomousLoopRequestV0(request, ports, loopRequest)
	if ports.ExternalWaiter == nil {
		loop, err := service.RunAutonomousDirectorLoopV0(ctx, autonomousRequest)
		result := appRunnerLoopResultFromAutonomousV0(loop)
		enrichAutonomousLoopStatsWithProgressV0(
			ctx,
			result.DirectorLoopStats,
			loop.Loop,
			loopRequest,
			ports.ProgressSource,
		)
		return result, err
	}
	return runPreparedManagedAutonomousAppLoopV0(
		ctx,
		request,
		ports,
		service,
		autonomousRequest,
	)
}
