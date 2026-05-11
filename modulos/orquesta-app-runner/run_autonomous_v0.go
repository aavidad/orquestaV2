package orquestaapprunner

import (
	"context"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func runPreparedAutonomousLoopRequestV0(
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopRequestV0,
) orquestacionnucleoapp.AutonomousDirectorLoopRequestV0 {
	return orquestacionnucleoapp.AutonomousDirectorLoopRequestV0{
		Loop:   loop,
		Policy: ports.AutonomousDirectorPolicy,
		Limits: orquestacionnucleoapp.AutonomousDirectorLimitsV0{
			MaxTeamSize:          request.MaxCommands,
			MaxParallelAgents:    request.MaxDispatchesPerWait,
			MaxBursts:            request.MaxBursts,
			MaxStepsPerBurst:     request.MaxStepsPerBurst,
			MaxDispatchesPerWait: request.MaxDispatchesPerWait,
			MaxCommandsPerCycle:  request.MaxCommands,
			MaxOutboxPerCycle:    request.MaxOutboxPerCycle,
		},
	}
}

func appRunnerLoopResultFromAutonomousV0(
	loop orquestacionnucleoapp.AutonomousDirectorLoopResultV0,
) appRunnerLoopResultV0 {
	stats := loop.Stats
	return appRunnerLoopResultV0{
		Managed:           managedResultFromProgressiveLoopV0(loop.Loop),
		DirectorLoopStats: &stats,
	}
}

func runPreparedManagedAutonomousAppLoopV0(
	ctx context.Context,
	request RunPreparedAppOrchestrationRequestV0,
	ports RunPreparedAppOrchestrationPortsV0,
	service orquestacionnucleoapp.ServiceV0,
	autonomousRequest orquestacionnucleoapp.AutonomousDirectorLoopRequestV0,
) (appRunnerLoopResultV0, error) {
	result := appRunnerLoopResultV0{}
	for attempt := 1; attempt <= request.MaxExternalWaits+1; attempt++ {
		loop, err := service.RunAutonomousDirectorLoopV0(ctx, autonomousRequest)
		result = appendAutonomousAttemptV0(result, attempt, loop)
		enrichAutonomousLoopStatsWithProgressV0(
			ctx,
			result.DirectorLoopStats,
			loop.Loop,
			autonomousRequest.Loop,
			ports.ProgressSource,
		)
		if err != nil || loop.Loop.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
			result.Managed.Status = loop.Loop.Status
			return result, err
		}
		if attempt > request.MaxExternalWaits {
			result.Managed.Status = loop.Loop.Status
			return result, nil
		}
		wait, err := ports.ExternalWaiter.WaitExternalProgressV0(
			ctx,
			externalAppRunnerWaitRequestV0(request, attempt, loop.Loop),
		)
		if err != nil {
			result.Managed.Status = loop.Loop.Status
			return result, err
		}
		result.Managed.ExternalWaits = append(
			result.Managed.ExternalWaits,
			appRunnerExternalWaitV0(attempt, wait),
		)
		if !wait.Continue {
			result.Managed.Status = loop.Loop.Status
			return result, nil
		}
	}
	return result, nil
}

func appendAutonomousAttemptV0(
	result appRunnerLoopResultV0,
	attemptNumber int,
	loop orquestacionnucleoapp.AutonomousDirectorLoopResultV0,
) appRunnerLoopResultV0 {
	stats := loop.Stats
	result.DirectorLoopStats = &stats
	result.Managed.Final = loop.Loop
	result.Managed.Attempts = append(result.Managed.Attempts,
		orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{
			AttemptNumber: attemptNumber,
			Result:        loop.Loop,
		},
	)
	return result
}

func enrichAutonomousLoopStatsWithProgressV0(
	ctx context.Context,
	stats *orquestacionnucleoapp.AutonomousDirectorLoopStatsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	request orquestacionnucleoapp.ProgressiveLoopRequestV0,
	progressSource orquestacionnucleoapp.AgentProgressObservationProviderPortV0,
) {
	if stats == nil || progressSource == nil {
		return
	}
	stats.Run = orquestacionnucleoapp.BuildDirectorRunStatsWithPortsV0(
		ctx,
		loop.Run,
		nil,
		progressSource,
		orquestacionnucleoapp.DirectorProgressSourceRequestV0{
			StepNumber:    loop.TotalExecutedSteps,
			MaxSteps:      request.MaxBursts * request.MaxStepsPerBurst,
			OccurredAt:    request.OccurredAt,
			CorrelationID: request.CorrelationID,
			EvidenceRefs:  request.EvidenceRefs,
		},
	)
}

func externalAppRunnerWaitRequestV0(
	request RunPreparedAppOrchestrationRequestV0,
	waitNumber int,
	last orquestacionnucleoapp.ProgressiveLoopResultV0,
) orquestacionnucleoapp.ExternalProgressWaitRequestV0 {
	return orquestacionnucleoapp.ExternalProgressWaitRequestV0{
		RunRef:        request.Prepared.Run.RunID,
		WaitNumber:    waitNumber,
		LastResult:    last,
		CorrelationID: request.CorrelationID,
		EvidenceRefs:  request.Prepared.EvidenceRefs,
	}
}

func appRunnerExternalWaitV0(
	waitNumber int,
	wait orquestacionnucleoapp.ExternalProgressWaitResultV0,
) orquestacionnucleoapp.ManagedProgressiveLoopExternalWaitV0 {
	return orquestacionnucleoapp.ManagedProgressiveLoopExternalWaitV0{
		WaitNumber:   waitNumber,
		Continue:     wait.Continue,
		EvidenceRefs: compactAppRunnerRefsV0(wait.EvidenceRefs),
	}
}
