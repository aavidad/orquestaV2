package orquestaappcodexstack

import (
	"errors"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestDrainObservationApplyErrorV0ExponeContextoAccionable(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:                  "run-ref-drain-error-001",
		CurrentPhase:           orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Agents:                 []string{"agent-ref-001"},
		StartedAgents:          []string{"agent-ref-001"},
		StoppedAgents:          []string{"agent-ref-001"},
		ConfirmedStoppedAgents: []string{"agent-ref-001"},
		Tasks:                  []string{"task-ref-001"},
		Deliveries:             []string{"delivery-ref-prev-001"},
		PhaseArtifacts:         []string{"artifact-ref-prev-001"},
	}
	observation := orquestacionnucleoapp.AgentDeliveryObservationV0{
		ArtifactRef: "artifact-ref-001",
		DeliveryRef: "delivery-ref-001",
		TaskID:      "task-ref-001",
		AgentRef:    "agent-ref-001",
		PhaseID:     "programacion",
	}
	err := drainObservationApplyErrorV0(run, observation, orquestacoreworkflow.OrchestrationCommandErrorV0{
		Code:  "transicion_invalida",
		Field: "payload.agent_ref",
	})

	var drainErr DrainObservationApplyErrorV0
	if !errors.As(err, &drainErr) {
		t.Fatalf("error no tipado: %T %v", err, err)
	}
	if drainErr.RunRef != "run-ref-drain-error-001" ||
		drainErr.DeliveryRef != "delivery-ref-001" ||
		drainErr.TaskRef != "task-ref-001" ||
		drainErr.Field != "payload.agent_ref" ||
		drainErr.CommandCode != "transicion_invalida" {
		t.Fatalf("contexto perdido: %+v", drainErr)
	}
	if got := drainErr.NextActionV0(); got != "ingest_late_ack_or_reconcile_stopped_agent" {
		t.Fatalf("next_action=%s", got)
	}
	message := drainErr.Error()
	if !strings.Contains(message, "confirmed_stopped=[agent-ref-001]") ||
		!strings.Contains(message, "field=payload.agent_ref") ||
		!strings.Contains(message, "delivery=delivery-ref-001") {
		t.Fatalf("mensaje sin auditoria completa: %s", message)
	}
}
