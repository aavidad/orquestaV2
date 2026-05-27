package orquestaappcodexstack

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestDrainObservationCommandV0NormalizaEntregaTardiaAFaseActualV0(t *testing.T) {
	taskRef := "task-ref-drain-late-phase-001"
	agentRef := "agent-ref-drain-late-phase-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:           "run-ref-drain-late-phase-001",
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:           []string{taskRef},
		Agents:          []string{agentRef},
		StartedAgents:   []string{agentRef},
		DeliveredAgents: nil,
	}
	command, err := drainObservationCommandV0(DrainRunRequestV0{
		RunRef:        run.RunID,
		OccurredAt:    "2026-05-26T21:55:00Z",
		CorrelationID: "corr-drain-late-phase-001",
	}, run, orquestacionnucleoapp.AgentDeliveryObservationV0{
		DeliveryRef:  "ack-ref-drain-late-phase-001",
		ArtifactRef:  "ack-ref-drain-late-phase-001",
		TaskID:       taskRef,
		AgentRef:     agentRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		Summary:      "Entrega tardia validada.",
		EvidenceRefs: []string{"evidence-ref-drain-late-phase-001"},
	})
	if err != nil {
		t.Fatalf("drainObservationCommandV0: %v", err)
	}
	var payload orquestacoreworkflow.RegisterDeliveryCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("phase_id=%q want current revision", payload.PhaseID)
	}
}
