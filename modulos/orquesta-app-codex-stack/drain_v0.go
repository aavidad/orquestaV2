package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type DrainRunRequestV0 struct {
	RunRef               string
	OccurredAt           string
	CorrelationID        string
	MaxBursts            int
	MaxStepsPerBurst     int
	MaxDispatchesPerWait int
	MaxCommands          int
	MaxOutboxPerCycle    int
	MaxDecisionCycles    int
	MaxExternalWaits     int
}

func (stack StackV0) DrainRunV0(
	ctx context.Context,
	request DrainRunRequestV0,
) (orquestacionnucleoapp.ManagedProgressiveLoopResultV0, error) {
	request = normalizeDrainRunRequestV0(request)
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{}
	for attempt := 1; attempt <= request.MaxExternalWaits+1; attempt++ {
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
		if !drainRunHasPendingExternalAgentsV0(loop.Run) || attempt > request.MaxExternalWaits {
			return result, nil
		}
		if stack.Ports.ExternalWaiter == nil {
			return result, nil
		}
		wait, err := stack.Ports.ExternalWaiter.WaitExternalProgressV0(
			ctx,
			orquestacionnucleoapp.ExternalProgressWaitRequestV0{
				RunRef:        request.RunRef,
				WaitNumber:    attempt,
				LastResult:    loop,
				CorrelationID: request.CorrelationID,
				EvidenceRefs:  []string{"evidence-ref-app-stack-drain-wait"},
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
		request.MaxExternalWaits = 6
	}
	return request
}

func (stack StackV0) applyDrainObservationsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	var run orquestacoreworkflow.OrchestrationRunV0
	for _, observation := range observations {
		current, err := stack.Stores.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return run, err
		}
		if drainObservationAlreadyRegisteredV0(current, observation) {
			run = current
			continue
		}
		command, err := drainObservationCommandV0(request, observation)
		if err != nil {
			return run, err
		}
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(
			ctx,
			stack.Stores.RunStore,
			stack.Ports.EventSink,
			command,
		); err != nil {
			return run, drainObservationApplyErrorV0(current, observation, err)
		}
	}
	return stack.Stores.RunStore.LoadRunV0(ctx, request.RunRef)
}

func drainObservationAlreadyRegisteredV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) bool {
	if strings.TrimSpace(observation.PhaseID) == string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		return stringInDrainSetV0(run.Deliveries, observation.DeliveryRef)
	}
	return phaseArtifactInDrainSetV0(run.PhaseArtifacts, observation.ArtifactRef)
}

func stringInDrainSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func phaseArtifactInDrainSetV0(values []string, artifactRef string) bool {
	artifactRef = strings.TrimSpace(artifactRef)
	for _, value := range values {
		head, _, ok := strings.Cut(strings.TrimSpace(value), "#phase:")
		if ok && strings.TrimSpace(head) == artifactRef {
			return true
		}
		if !ok && strings.TrimSpace(value) == artifactRef {
			return true
		}
	}
	return false
}

func drainObservationApplyErrorV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	err error,
) error {
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		return fmt.Errorf(
			"drain observation artifact=%s agent=%s phase=%s current=%s field=%s agents=%v started=%v failed=%v stopped=%v artifacts=%v deliveries=%v: %w",
			observation.ArtifactRef,
			observation.AgentRef,
			observation.PhaseID,
			run.CurrentPhase,
			commandErr.Field,
			compactStringsV0(run.Agents),
			compactStringsV0(run.StartedAgents),
			compactStringsV0(run.FailedAgents),
			compactStringsV0(run.StoppedAgents),
			compactStringsV0(run.PhaseArtifacts),
			compactStringsV0(run.Deliveries),
			err,
		)
	}
	return fmt.Errorf(
		"drain observation artifact=%s agent=%s phase=%s current=%s agents=%v started=%v failed=%v stopped=%v artifacts=%v deliveries=%v: %w",
		observation.ArtifactRef,
		observation.AgentRef,
		observation.PhaseID,
		run.CurrentPhase,
		compactStringsV0(run.Agents),
		compactStringsV0(run.StartedAgents),
		compactStringsV0(run.FailedAgents),
		compactStringsV0(run.StoppedAgents),
		compactStringsV0(run.PhaseArtifacts),
		compactStringsV0(run.Deliveries),
		err,
	)
}

func drainObservationCommandV0(
	request DrainRunRequestV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	meta := orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-stack-drain-" + strings.TrimSpace(observation.DeliveryRef),
		RunID:          request.RunRef,
		IdempotencyKey: "idem-app-stack-drain-" + strings.TrimSpace(observation.DeliveryRef),
		CorrelationID:  request.CorrelationID,
		RequestedBy:    "orquesta-app-codex-stack-drain",
		OccurredAt:     request.OccurredAt,
	}
	if strings.TrimSpace(observation.PhaseID) == string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
		return orquestacoreworkflow.NewRegisterDeliveryCommandV0(meta, orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
			DeliveryRef:  observation.DeliveryRef,
			PhaseID:      observation.PhaseID,
			TaskID:       observation.TaskID,
			AgentRef:     observation.AgentRef,
			Summary:      observation.Summary,
			EvidenceRefs: observation.EvidenceRefs,
		})
	}
	return orquestacoreworkflow.NewRegisterPhaseArtifactCommandV0(meta, orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0{
		ArtifactRef:  observation.ArtifactRef,
		PhaseID:      observation.PhaseID,
		AgentRef:     observation.AgentRef,
		Summary:      observation.Summary,
		EvidenceRefs: observation.EvidenceRefs,
	})
}

func drainProgressiveResultV0(
	status orquestacionnucleoapp.ProgressiveLoopStatusV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	return orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: status,
		Run:    run,
	}
}

func drainRunHasPendingExternalAgentsV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	started := len(compactStringsV0(run.StartedAgents))
	closed := len(compactStringsV0(run.Deliveries)) +
		len(compactStringsV0(run.PhaseArtifacts)) +
		len(compactStringsV0(run.FailedAgents)) +
		len(compactStringsV0(run.StoppedAgents))
	return started > closed
}
