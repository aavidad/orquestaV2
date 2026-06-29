package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestNewMCPPreviewDirectorAppResultV0PublicaResumenSinGoalSpecCompleto(t *testing.T) {
	result := NewMCPPreviewDirectorAppResultV0(orquestaappdirectorservice.StartAppDirectorGoalPreviewV0{
		Status:                orquestaappdirectorservice.StartAppDirectorStatusPreviewReadyV0,
		DirectorExecutionMode: "goal_first",
		CorrelationID:         "corr-preview-public-001",
		AppSpec: orquestafactory.AppSpecV0{
			RequestID: "req-preview-public-001",
		},
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID: "run-preview-public-001",
		},
		GoalSpec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      "goal-preview-public-001",
			RunRef:       "run-preview-public-001",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			Objective:    "objetivo privado que no debe serializarse completo",
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "generated-apps/privada",
			}},
			RequiredTests: []orquestagoal.GoalRequiredTestV0{{
				TestRef: "test-preview-public-001",
				Command: "comando privado que no debe filtrarse",
			}},
		},
	})

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(payload)
	for _, forbidden := range []string{
		`"goal_spec"`,
		`"write_set"`,
		`"required_tests"`,
		"objetivo privado",
		"comando privado",
		"generated-apps/privada",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("payload publico filtra %q: %s", forbidden, raw)
		}
	}
	if result.GoalSpecSummary.GoalRef != "goal-preview-public-001" ||
		result.GoalSpecSummary.RunRef != "run-preview-public-001" ||
		result.GoalSpecSummary.WriteSetCount != 1 ||
		result.GoalSpecSummary.RequiredTestCount != 1 ||
		result.GoalSpecSummary.SpecHash == "" {
		t.Fatalf("summary=%+v", result.GoalSpecSummary)
	}
}
