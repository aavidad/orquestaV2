package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type existingDirectorAutonomyLoopResultV0 struct {
	Request     ContinueAppDirectorRequestV0
	Loop        orquestacionnucleoapp.ProgressiveLoopResultV0
	LoopRequest orquestacionnucleoapp.ProgressiveLoopRequestV0
	ManagedLoop orquestacionnucleoapp.ManagedProgressiveLoopResultV0
}

func runExistingDirectorAutonomyLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (existingDirectorAutonomyLoopResultV0, error) {
	loop, loopRequest, managedLoop, err := runExistingDirectorLoopV0(ctx, request, ports)
	if err != nil || ports.DirectorDecisionSource == nil {
		return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
	}
	startRequest := continueAsStartRequestV0(request)
	for cycle := 0; cycle < request.MaxDecisionCycles; cycle++ {
		next, progressed, err := consumeStartAppDirectorDecisionsV0(ctx, startRequest, ports, loop)
		if err != nil || !progressed {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: next, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		request, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
		if err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: next, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		if _, err := ensureOperationalDirectorReviewPhaseV0(ctx, request, ports); err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: next, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
		startRequest = continueAsStartRequestV0(request)
		loop, loopRequest, managedLoop, err = runExistingDirectorLoopV0(ctx, request, ports)
		if err != nil {
			return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, err
		}
	}
	return existingDirectorAutonomyLoopResultV0{Request: request, Loop: loop, LoopRequest: loopRequest, ManagedLoop: managedLoop}, nil
}

func runExistingDirectorLoopV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, orquestacionnucleoapp.ProgressiveLoopRequestV0, orquestacionnucleoapp.ManagedProgressiveLoopResultV0, error) {
	service := existingDirectorLoopServiceV0(request, ports)
	loopRequest, err := existingDirectorLoopRequestV0(ctx, request, ports)
	if err != nil {
		return orquestacionnucleoapp.ProgressiveLoopResultV0{}, loopRequest, orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}, err
	}
	if ports.ExternalWaiter == nil {
		loop, err := service.RunProgressiveLoopV0(ctx, loopRequest)
		return loop, loopRequest, orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}, err
	}
	managed, err := service.RunManagedProgressiveLoopV0(
		ctx,
		orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
			Loop:             loopRequest,
			ExternalWaiter:   ports.ExternalWaiter,
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	return managed.Final, loopRequest, managed, err
}

func existingDirectorLoopServiceV0(
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) orquestacionnucleoapp.ServiceV0 {
	return orquestacionnucleoapp.ServiceV0{
		RunStore:           ports.RunStore,
		EventSink:          ports.EventSink,
		CandidateProvider:  composeStartAppDirectorProviderV0(nil, ports, request.RequestedBy),
		RunControl:         ports.RunControl,
		RunControlTerminal: ports.RunControlTerminal,
		OutboxLedger:       ports.OutboxLedger,
		MaxCommands:        request.MaxCommands,
		MaxOutboxPerCycle:  request.MaxOutboxPerCycle,
	}
}

func existingDirectorLoopRequestV0(
	ctx context.Context,
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) (orquestacionnucleoapp.ProgressiveLoopRequestV0, error) {
	waitResolution, err := appDirectorResolvedWaitV0(
		ctx,
		request.RunRef,
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
			EvidenceRefs:     []string{"evidence-ref-app-director-continue-v0"},
			MaxExternalWaits: request.MaxExternalWaits,
		},
	)
	if err != nil {
		return orquestacionnucleoapp.ProgressiveLoopRequestV0{}, err
	}
	return orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               request.RunRef,
		OccurredAt:           request.OccurredAt,
		MaxBursts:            request.MaxBursts,
		MaxStepsPerBurst:     request.MaxStepsPerBurst,
		MaxDispatchesPerWait: request.MaxDispatchesPerWait,
		WaitAgentRefs:        waitResolution.AgentRefs,
		WaitScopeApplied:     waitResolution.ScopeApplied,
		CorrelationID:        request.CorrelationID,
		EvidenceRefs:         []string{"evidence-ref-app-director-continue-v0"},
		Dispatchers:          ports.Dispatchers,
		BatchDispatchers:     ports.BatchDispatchers,
	}, nil
}

func continueAsStartRequestV0(request ContinueAppDirectorRequestV0) StartAppDirectorRequestV0 {
	return StartAppDirectorRequestV0{
		RunRef:                                  request.RunRef,
		OccurredAt:                              request.OccurredAt,
		CorrelationID:                           request.CorrelationID,
		RequestedBy:                             request.RequestedBy,
		MaxBursts:                               request.MaxBursts,
		MaxStepsPerBurst:                        request.MaxStepsPerBurst,
		MaxDispatchesPerWait:                    request.MaxDispatchesPerWait,
		WaitAgentRefs:                           request.WaitAgentRefs,
		WaitCohortRef:                           request.WaitCohortRef,
		WaitWaveRef:                             request.WaitWaveRef,
		WaitParentTaskRef:                       request.WaitParentTaskRef,
		MaxCommands:                             request.MaxCommands,
		MaxOutboxPerCycle:                       request.MaxOutboxPerCycle,
		MaxDecisionCycles:                       request.MaxDecisionCycles,
		MaxExternalWaits:                        request.MaxExternalWaits,
		OperationalDirectorPlanRef:              request.OperationalDirectorPlanRef,
		OperationalDirectorPlan:                 request.OperationalDirectorPlan,
		OperationalDirectorFunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.OperationalDirectorFunctionContractRefs...),
		OperationalDirectorTargetPhaseID:        request.OperationalDirectorTargetPhaseID,
		OperationalDirectorMaxItems:             request.OperationalDirectorMaxItems,
	}
}
