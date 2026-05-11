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
	if got := progressiveStatusAfterMaxBurstsV0(run); got != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s", got)
	}
	run.PhaseArtifacts = append(run.PhaseArtifacts, "artifact-b")
	if got := progressiveStatusAfterMaxBurstsV0(run); got != ProgressiveLoopStatusMaxBurstsV0 {
		t.Fatalf("status=%s", got)
	}
}
