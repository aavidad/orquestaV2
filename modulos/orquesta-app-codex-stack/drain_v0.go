package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const defaultDrainRunMaxExternalWaitsV0 = 120

type waitAgentRefsDeliverySourceV0 struct {
	Inner         orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0
	WaitAgentRefs []string
}

type waitAgentRefsProgressSourceV0 struct {
	Inner         orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	WaitAgentRefs []string
}

func drainDeliverySourceForWaitAgentRefsV0(
	inner orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0,
	waitAgentRefs []string,
) orquestacionnucleoapp.AgentDeliveryObservationProviderPortV0 {
	if inner == nil || len(compactStringsV0(waitAgentRefs)) == 0 {
		return inner
	}
	return waitAgentRefsDeliverySourceV0{
		Inner:         inner,
		WaitAgentRefs: compactStringsV0(waitAgentRefs),
	}
}

func (source waitAgentRefsDeliverySourceV0) BuildAgentDeliveryObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	observations, err := source.Inner.BuildAgentDeliveryObservationsV0(ctx, request)
	if err != nil {
		return nil, err
	}
	return drainObservationsForWaitAgentRefsV0(observations, source.WaitAgentRefs), nil
}

func drainProgressSourceForWaitAgentRefsV0(
	inner orquestacionnucleoapp.AgentProgressObservationProviderPortV0,
	waitAgentRefs []string,
) orquestacionnucleoapp.AgentProgressObservationProviderPortV0 {
	if inner == nil || len(compactStringsV0(waitAgentRefs)) == 0 {
		return inner
	}
	return waitAgentRefsProgressSourceV0{
		Inner:         inner,
		WaitAgentRefs: compactStringsV0(waitAgentRefs),
	}
}

func (source waitAgentRefsProgressSourceV0) BuildAgentProgressObservationsV0(
	ctx context.Context,
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
) ([]orquestacionnucleoapp.AgentProgressObservationV0, error) {
	observations, err := source.Inner.BuildAgentProgressObservationsV0(ctx, request)
	if err != nil {
		return nil, err
	}
	return drainProgressObservationsForWaitAgentRefsV0(observations, source.WaitAgentRefs), nil
}

type DrainRunRequestV0 struct {
	RunRef                     string
	OccurredAt                 string
	CorrelationID              string
	MaxBursts                  int
	MaxStepsPerBurst           int
	MaxDispatchesPerWait       int
	MaxCommands                int
	MaxOutboxPerCycle          int
	MaxDecisionCycles          int
	MaxExternalWaits           int
	WaitAgentRefs              []string
	OperationalDirectorPlanRef string
}

func (stack StackV0) DrainRunV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (orquestacionnucleoapp.ManagedProgressiveLoopResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request = normalizeDrainRunRequestV0(request)
	if disposition, ok := stack.goalFirstSupervisorDispositionV0(ctx, request.RunRef); ok {
		run := orquestacoreworkflow.OrchestrationRunV0{RunID: strings.TrimSpace(request.RunRef)}
		if stack.Ports.RunStore != nil {
			if loaded, err := stack.Ports.RunStore.LoadRunV0(ctx, request.RunRef); err == nil {
				run = loaded
			}
		}
		return disposition.managedLoopResultV0(run), nil
	}
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}
	for attempt := 1; attempt <= request.MaxExternalWaits+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		control, err := stack.drainRunAttemptControlV0(ctx, request)
		loop := control.Loop
		result.Final = loop
		result.Status = loop.Status
		result.Attempts = append(result.Attempts, orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{
			AttemptNumber: attempt,
			Result:        loop,
		})
		if err != nil {
			return result, err
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if !drainRunHasPendingExternalAgentsV0(loop.Run, request.WaitAgentRefs) || attempt > request.MaxExternalWaits {
			return result, nil
		}
		if stack.Ports.ExternalWaiter == nil {
			return result, nil
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		wait, err := stack.Ports.ExternalWaiter.WaitExternalProgressV0(
			ctx,
			orquestacionnucleoapp.ExternalProgressWaitRequestV0{
				RunRef:        request.RunRef,
				WaitNumber:    attempt,
				LastResult:    loop,
				CorrelationID: request.CorrelationID,
				EvidenceRefs:  []string{"evidence-ref-app-stack-drain-wait"},
				WaitAgentRefs: request.WaitAgentRefs,
			},
		)
		if err != nil {
			return result, err
		}
		result.ExternalWaits = append(result.ExternalWaits, orquestacionnucleoapp.ManagedProgressiveLoopExternalWaitV0{
			WaitNumber:   attempt,
			Continue:     wait.Continue,
			EvidenceRefs: wait.EvidenceRefs,
		})
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if !wait.Continue {
			return result, nil
		}
	}
	return result, nil
}

func normalizeDrainRunRequestV0(request DrainRunRequestV0) DrainRunRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.WaitAgentRefs = compactStringsV0(request.WaitAgentRefs)
	request.OperationalDirectorPlanRef = strings.TrimSpace(request.OperationalDirectorPlanRef)
	if request.OccurredAt == "" {
		request.OccurredAt = "2026-05-10T12:00:00Z"
	}
	if request.MaxBursts <= 0 {
		request.MaxBursts = 16
	}
	if request.MaxStepsPerBurst <= 0 {
		request.MaxStepsPerBurst = 12
	}
	if request.MaxDispatchesPerWait <= 0 {
		request.MaxDispatchesPerWait = 8
	}
	if request.MaxCommands <= 0 {
		request.MaxCommands = 20
	}
	if request.MaxOutboxPerCycle <= 0 {
		request.MaxOutboxPerCycle = 8
	}
	if request.MaxDecisionCycles <= 0 {
		request.MaxDecisionCycles = 4
	}
	if request.MaxExternalWaits <= 0 {
		request.MaxExternalWaits = defaultDrainRunMaxExternalWaitsV0
	}
	return request
}
