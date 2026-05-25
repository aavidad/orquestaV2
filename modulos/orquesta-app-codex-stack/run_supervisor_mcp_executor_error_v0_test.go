package orquestaappcodexstack

import (
	"errors"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackRunSupervisorErrorResultMCPV0IncluyeDiagnosticsYNextAction(t *testing.T) {
	input := orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-supervisor-error-result-001",
		CorrelationID: "corr-supervisor-error-result-001",
		RunRef:        "run-ref-supervisor-error-result-001",
	}
	err := DrainObservationApplyErrorV0{
		RunRef:        "run-ref-supervisor-error-result-001",
		ArtifactRef:   "artifact-ref-001",
		DeliveryRef:   "delivery-ref-001",
		TaskRef:       "task-ref-001",
		AgentRef:      "agent-ref-001",
		PhaseID:       "programacion",
		CurrentPhase:  "programacion",
		CommandCode:   "transicion_invalida",
		Field:         "payload.agent_ref",
		StoppedAgents: []string{"agent-ref-001"},
		Deliveries:    []string{"delivery-ref-prev-001"},
		Cause:         "transicion_invalida: payload.agent_ref",
		Err:           errors.New("transicion_invalida: payload.agent_ref"),
	}

	result := codexStackRunSupervisorErrorResultMCPV0(input, CodexSupervisorResultV0{}, err)

	if result.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "delivery_ack_ingestion_failed" ||
		result.Errores[0].Field != "payload.agent_ref" {
		t.Fatalf("error publico incompleto: %+v", result)
	}
	if len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "drain_observation_apply_failed" ||
		result.Diagnostics[0].Scope != "run:run-ref-supervisor-error-result-001/task:task-ref-001/agent:agent-ref-001" {
		t.Fatalf("diagnostics perdidos: %+v", result.Diagnostics)
	}
	if len(result.NextActions) != 1 ||
		result.NextActions[0] != "ingest_late_ack_or_reconcile_stopped_agent" {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
}
