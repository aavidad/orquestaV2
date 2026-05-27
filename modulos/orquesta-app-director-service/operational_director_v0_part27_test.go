package orquestaappdirectorservice

import (
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func serviceOperationalDirectorPlanStateEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	eventType string,
	payload any,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return orquestacoreworkflow.OrchestrationEventV0{
		EventID:        eventType + "-plan-state-review-" + string(rune('0'+sequence)),
		EventType:      eventType,
		RunID:          runRef,
		Sequence:       sequence,
		OccurredAt:     "2026-05-17T14:19:00Z",
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        raw,
	}
}

func serviceOperationalDirectorPlanStateStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no encontrado en %+v", stepID, state.Steps)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func serviceOperationalDirectorPlanForContinueTestV0(
	t *testing.T,
	runRef string,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	return serviceOperationalDirectorPlanForContinueWithParallelTestV0(t, runRef, 1)
}

func serviceOperationalDirectorPlanWithParallelLaunchesForTestV0(
	t *testing.T,
	runRef string,
	maxParallelAgents int,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	return serviceOperationalDirectorPlanForContinueWithParallelTestV0(t, runRef, maxParallelAgents)
}

func serviceOperationalDirectorPlanForContinueWithParallelTestV0(
	t *testing.T,
	runRef string,
	maxParallelAgents int,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:        "req-app-director-operational-plan-001",
		RunRef:            runRef,
		ProjectRef:        "orquesta",
		Objective:         "Cerrar tramo launch wait del Director Operativo.",
		Mode:              orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		WorktreeRef:       "worktree-ref-app-director-operational-plan",
		WorktreeIsolated:  true,
		BranchRef:         "branch-ref-app-director-operational-plan",
		MaxParallelAgents: maxParallelAgents,
		WriteSet:          serviceOperationalDirectorPlanWriteSetForTestV0(maxParallelAgents),
		RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-app-director-service"},
	})
	if !result.Accepted || !result.ReadyToLaunch {
		t.Fatalf("plan no listo: %+v", result)
	}
	return result.Plan
}

func serviceOperationalDirectorPlanWriteSetForTestV0(maxParallelAgents int) []string {
	if maxParallelAgents <= 1 {
		return []string{"docs/director_operational_plan_test.md"}
	}
	return []string{
		"docs/director_operational_plan_test.md",
		"modulos/orquesta-app-director-service/operational_director_v0.go",
		"modulos/orquesta-app-director-service/operational_director_v0_test.go",
	}
}
