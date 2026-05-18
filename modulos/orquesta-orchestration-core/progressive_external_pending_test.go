package orquestacionnucleoapp

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestProgressiveStatusAfterMaxBurstsV0WaitExternalSiHayAgentesPendientes(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StartedAgents: []string{"agent-a", "agent-b"},
		PhaseArtifacts: []string{
			"artifact-a",
		},
	}
	if got := progressiveStatusAfterMaxBurstsV0(run, nil, false); got != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s", got)
	}
	run.PhaseArtifacts = append(run.PhaseArtifacts, "artifact-b")
	if got := progressiveStatusAfterMaxBurstsV0(run, nil, false); got != ProgressiveLoopStatusMaxBurstsV0 {
		t.Fatalf("status=%s", got)
	}
}

func TestProgressiveStatusAfterMaxBurstsV0EsperaSoloAgentesObjetivo(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StartedAgents:   []string{"agent-old", "agent-new"},
		DeliveredAgents: []string{"agent-new"},
	}
	if got := progressiveStatusAfterMaxBurstsV0(run, []string{"agent-new"}, false); got != ProgressiveLoopStatusMaxBurstsV0 {
		t.Fatalf("status=%s", got)
	}
	if got := progressiveStatusAfterMaxBurstsV0(run, []string{"agent-old"}, false); got != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s", got)
	}
}

func TestProgressiveStatusAfterMaxBurstsV0ScopeVacioNoEsperaLegacy(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StartedAgents: []string{"agent-old"},
	}
	if got := progressiveStatusAfterMaxBurstsV0(run, nil, true); got != ProgressiveLoopStatusMaxBurstsV0 {
		t.Fatalf("status=%s", got)
	}
}
