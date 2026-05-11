package orquestaappdirectorservice

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestStartAppDirectorDecisionDeferredV0RetieneRevisionHastaEntregas(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:        []string{"task-a", "task-b"},
		Deliveries:   []string{"delivery-a"},
	}
	if !startAppDirectorDecisionDeferredV0(run, reviewOpenDecisionForServiceTestV0()) {
		t.Fatalf("expected revision deferred while delivery missing")
	}
	run.Deliveries = append(run.Deliveries, "delivery-b")
	if startAppDirectorDecisionDeferredV0(run, reviewOpenDecisionForServiceTestV0()) {
		t.Fatalf("expected revision allowed when deliveries cover tasks")
	}
}

func TestStartAppDirectorDecisionTransitionPendingV0DetectaTransicion(t *testing.T) {
	err := orquestacoreworkflow.OrchestrationCommandErrorV0{
		Code:  orquestacoreworkflow.ErrTransicionInvalidaV0,
		Field: "payload.delivery_ref",
	}
	if !startAppDirectorDecisionTransitionPendingV0(err) {
		t.Fatalf("expected transition pending")
	}
}

func reviewOpenDecisionForServiceTestV0() orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		CommandType: orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "Entregas listas para revision.",
		},
	}
}
