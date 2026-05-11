package orquestacionnucleoapp

import (
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func mustActiveBrainstormingRunWithStartedAgentV0(
	t *testing.T,
	runRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	capacityRef := "capacity-ref-nucleo-brainstorm-001"
	var run orquestacoreworkflow.OrchestrationRunV0
	run = mustApplyCommandV0(t, run, mustStartRunCommandV0(t, runRef))
	run = mustApplyCommandV0(t, run, mustOpenBrainstormingCommandV0(t, runRef))
	run = mustApplyCommandV0(t, run, mustRequestPhaseCapacityCommandV0(t, runRef, capacityRef))
	run = mustApplyCommandV0(t, run, mustRegisterPhaseCapacityDecisionCommandV0(t, runRef, capacityRef))
	run = mustApplyCommandV0(t, run, mustRequestPhaseAgentCommandV0(t, runRef, capacityRef, agentRef))
	run = mustApplyCommandV0(t, run, mustRegisterPhaseAgentStartedCommandV0(t, runRef, agentRef))
	return run
}

func mustOpenBrainstormingCommandV0(t *testing.T, runRef string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		commandMetaV0(runRef, "cmd-open-brainstorming-001", "idem-open-brainstorming-001"),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			Reason:  "Preparar arquitectura inicial.",
		},
	)
	if err != nil {
		t.Fatalf("open brainstorming command: %v", err)
	}
	return command
}

func mustRequestPhaseCapacityCommandV0(
	t *testing.T,
	runRef string,
	capacityRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRequestCapacityCommandV0(
		commandMetaV0(runRef, "cmd-capacity-brainstorming-001", "idem-capacity-brainstorming-001"),
		orquestacoreworkflow.RequestCapacityCommandPayloadV0{
			CapacityRequestID:          capacityRef,
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			ReasonCode:                 "arquitectura_inicial",
			Summary:                    "Capacidad para razonar arquitectura inicial.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-capacity-brainstorming-001"},
		},
	)
	if err != nil {
		t.Fatalf("request phase capacity command: %v", err)
	}
	return command
}

func mustRegisterPhaseCapacityDecisionCommandV0(
	t *testing.T,
	runRef string,
	capacityRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRegisterCapacityDecisionCommandV0(
		commandMetaV0(runRef, "cmd-capacity-decision-brainstorming-001", "idem-capacity-decision-brainstorming-001"),
		orquestacoreworkflow.RegisterCapacityDecisionCommandPayloadV0{
			CapacityRequestID: capacityRef,
			DecisionRef:       "decision-ref-nucleo-brainstorm-001",
			Tier:              orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort:   orquestacoreworkflow.OrchestrationCapacityHighV0,
			Summary:           "Decision de capacidad para arquitectura inicial.",
			EvidenceRefs:      []string{"evidence-ref-capacity-decision-brainstorming-001"},
		},
	)
	if err != nil {
		t.Fatalf("register phase capacity decision command: %v", err)
	}
	return command
}

func mustRequestPhaseAgentCommandV0(
	t *testing.T,
	runRef string,
	capacityRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRequestAgentCommandV0(
		commandMetaV0(runRef, "cmd-agent-brainstorming-001", "idem-agent-brainstorming-001"),
		orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     agentRef,
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			CapacityRequestRef: capacityRef,
			Role:               "director",
			Summary:            "Agente para proponer arquitectura inicial.",
			EvidenceRefs:       []string{"evidence-ref-agent-brainstorming-001"},
		},
	)
	if err != nil {
		t.Fatalf("request phase agent command: %v", err)
	}
	return command
}

func mustRegisterPhaseAgentStartedCommandV0(
	t *testing.T,
	runRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRegisterAgentStartedCommandV0(
		commandMetaV0(runRef, "cmd-agent-started-brainstorming-001", "idem-agent-started-brainstorming-001"),
		orquestacoreworkflow.RegisterAgentStartedCommandPayloadV0{
			AgentRequestID: agentRef,
			LaunchRef:      "launch-ref-nucleo-brainstorm-001",
			AckRef:         "ack-ref-nucleo-brainstorm-001",
			ReadinessRef:   "readiness-ref-nucleo-brainstorm-001",
			EvidenceRefs:   []string{"evidence-ref-agent-started-brainstorming-001"},
		},
	)
	if err != nil {
		t.Fatalf("register phase agent started command: %v", err)
	}
	return command
}

func phaseArtifactObservationForTestV0() AgentDeliveryObservationV0 {
	return AgentDeliveryObservationV0{
		ArtifactRef:  "artifact-ref-nucleo-brainstorm-001",
		DeliveryRef:  "receipt-ref-nucleo-brainstorm-001",
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		AgentRef:     "agent-ref-nucleo-brainstorm-001",
		Summary:      "Arquitectura inicial documentada con evidencias compactas.",
		EvidenceRefs: []string{"evidence-ref-phase-artifact-nucleo-001"},
	}
}

func containsProjectionRefPartV0(values []string, ref string) bool {
	for _, value := range values {
		if strings.Contains(value, ref) {
			return true
		}
	}
	return false
}
