package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (stack StackV0) applyDrainObservationsV0(
	ctx context.Context,
	request DrainRunRequestV0,
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	var run orquestacoreworkflow.OrchestrationRunV0
	for _, observation := range observations {
		if !drainObservationMatchesWaitAgentRefsV0(observation, request.WaitAgentRefs) {
			continue
		}
		current, err := stack.Stores.RunStore.LoadRunV0(ctx, request.RunRef)
		if err != nil {
			return run, err
		}
		if drainObservationAlreadyRegisteredV0(current, observation) {
			run = current
			continue
		}
		command, err := drainObservationCommandV0(request, current, observation)
		if err != nil {
			return run, err
		}
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(
			ctx, stack.Stores.RunStore, stack.Ports.EventSink, command,
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
	if drainObservationIsTaskDeliveryV0(run, observation) {
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
	out := DrainObservationApplyErrorV0{
		RunRef:                 strings.TrimSpace(run.RunID),
		ArtifactRef:            strings.TrimSpace(observation.ArtifactRef),
		DeliveryRef:            strings.TrimSpace(observation.DeliveryRef),
		TaskRef:                strings.TrimSpace(observation.TaskID),
		AgentRef:               strings.TrimSpace(observation.AgentRef),
		PhaseID:                strings.TrimSpace(observation.PhaseID),
		CurrentPhase:           strings.TrimSpace(string(run.CurrentPhase)),
		Agents:                 compactStringsV0(run.Agents),
		StartedAgents:          compactStringsV0(run.StartedAgents),
		FailedAgents:           compactStringsV0(run.FailedAgents),
		LostAgents:             compactStringsV0(run.LostAgents),
		StoppedAgents:          compactStringsV0(run.StoppedAgents),
		ConfirmedStoppedAgents: compactStringsV0(run.ConfirmedStoppedAgents),
		PhaseArtifacts:         compactStringsV0(run.PhaseArtifacts),
		Deliveries:             compactStringsV0(run.Deliveries),
		Cause:                  strings.TrimSpace(err.Error()),
		Err:                    err,
	}
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		out.CommandCode = strings.TrimSpace(commandErr.Code)
		out.Field = strings.TrimSpace(commandErr.Field)
	}
	return out
}

type DrainObservationApplyErrorV0 struct {
	RunRef                 string
	ArtifactRef            string
	DeliveryRef            string
	TaskRef                string
	AgentRef               string
	PhaseID                string
	CurrentPhase           string
	CommandCode            string
	Field                  string
	Agents                 []string
	StartedAgents          []string
	FailedAgents           []string
	LostAgents             []string
	StoppedAgents          []string
	ConfirmedStoppedAgents []string
	PhaseArtifacts         []string
	Deliveries             []string
	Cause                  string
	Err                    error
}

func (err DrainObservationApplyErrorV0) Error() string {
	parts := compactStringsV0([]string{
		"drain_observation_apply_failed",
		"run=" + err.RunRef,
		"artifact=" + err.ArtifactRef,
		"delivery=" + err.DeliveryRef,
		"task=" + err.TaskRef,
		"agent=" + err.AgentRef,
		"phase=" + err.PhaseID,
		"current=" + err.CurrentPhase,
		"code=" + err.CommandCode,
		"field=" + err.Field,
	})
	details := []string{
		fmt.Sprintf("agents=%v", compactStringsV0(err.Agents)),
		fmt.Sprintf("started=%v", compactStringsV0(err.StartedAgents)),
		fmt.Sprintf("failed=%v", compactStringsV0(err.FailedAgents)),
		fmt.Sprintf("lost=%v", compactStringsV0(err.LostAgents)),
		fmt.Sprintf("stopped=%v", compactStringsV0(err.StoppedAgents)),
		fmt.Sprintf("confirmed_stopped=%v", compactStringsV0(err.ConfirmedStoppedAgents)),
		fmt.Sprintf("artifacts=%v", compactStringsV0(err.PhaseArtifacts)),
		fmt.Sprintf("deliveries=%v", compactStringsV0(err.Deliveries)),
	}
	message := strings.Join(append(parts, details...), " ")
	if strings.TrimSpace(err.Cause) != "" {
		message += ": " + strings.TrimSpace(err.Cause)
	}
	return message
}

func (err DrainObservationApplyErrorV0) Unwrap() error {
	return err.Err
}

func (err DrainObservationApplyErrorV0) NextActionV0() string {
	field := strings.TrimSpace(err.Field)
	switch {
	case field == "payload.agent_ref" && drainObservationStringInSetV0(err.StoppedAgents, err.AgentRef):
		return "ingest_late_ack_or_reconcile_stopped_agent"
	case field == "payload.agent_ref" && drainObservationStringInSetV0(err.ConfirmedStoppedAgents, err.AgentRef):
		return "ingest_late_ack_or_reconcile_confirmed_stopped_agent"
	case field == "payload.agent_ref":
		return "reconcile_agent_ref_or_agent_lifecycle"
	case field == "payload.task_id" || field == "payload.task_ref":
		return "repair_task_ref_or_rematerialize_workflow_task_store"
	case field == "phase" || field == "payload.phase_id":
		return "normalize_phase_or_reopen_valid_phase"
	default:
		return "inspect_run_state_and_requeue_recoverable_observation"
	}
}

func drainObservationStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func drainObservationCommandV0(
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	phaseID := drainObservationPhaseIDV0(run, observation)
	meta := orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-app-stack-drain-" + strings.TrimSpace(observation.DeliveryRef),
		RunID:          request.RunRef,
		IdempotencyKey: "idem-app-stack-drain-" + strings.TrimSpace(observation.DeliveryRef),
		CorrelationID:  request.CorrelationID,
		RequestedBy:    "orquesta-app-codex-stack-drain",
		OccurredAt:     request.OccurredAt,
	}
	if drainObservationIsTaskDeliveryV0(run, observation) {
		return orquestacoreworkflow.NewRegisterDeliveryCommandV0(meta, orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
			DeliveryRef:  observation.DeliveryRef,
			PhaseID:      phaseID,
			TaskID:       observation.TaskID,
			AgentRef:     observation.AgentRef,
			Summary:      observation.Summary,
			EvidenceRefs: observation.EvidenceRefs,
		})
	}
	return orquestacoreworkflow.NewRegisterPhaseArtifactCommandV0(meta, orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0{
		ArtifactRef:  observation.ArtifactRef,
		PhaseID:      phaseID,
		AgentRef:     observation.AgentRef,
		Summary:      observation.Summary,
		EvidenceRefs: observation.EvidenceRefs,
	})
}

func drainObservationIsTaskDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) bool {
	taskRef := strings.TrimSpace(observation.TaskID)
	for _, value := range run.Tasks {
		if strings.TrimSpace(value) == taskRef {
			return strings.TrimSpace(observation.DeliveryRef) != ""
		}
	}
	return false
}

func drainObservationPhaseIDV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
) string {
	phaseID := strings.TrimSpace(observation.PhaseID)
	if phaseID != "" {
		return phaseID
	}
	return strings.TrimSpace(string(run.CurrentPhase))
}
