package orquestaappcodexstack

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestNormalizeCompositeGoAppParallelWriteSetsV0ReparaRutasAgregadasCompartidas(t *testing.T) {
	decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
		parallelWriteSetDecisionForTestV0("task-rx-000", "RX-000 bootstrap", nil, []string{
			"go.mod",
			"cmd/server/main.go",
			"internal/platform/**",
			"internal/modules/**",
			"README.md",
		}),
		parallelWriteSetDecisionForTestV0("task-rx-001", "RX-001 identity-base", []string{"task-rx-000"}, []string{
			"internal/modules/identity/**",
			"internal/platform/api/**",
			"docs/openapi.yaml",
			"docs/**",
			"i18n/**",
			"README.md",
		}),
		parallelWriteSetDecisionForTestV0("task-rx-002", "RX-002 residents-core", []string{"task-rx-000"}, []string{
			"internal/modules/residents/**",
			"internal/platform/api/**",
			"docs/openapi.yaml",
			"docs/**",
			"i18n/**",
			"README.md",
		}),
		parallelWriteSetDecisionForTestV0("task-rx-003", "RX-003 care-events", []string{"task-rx-000"}, []string{
			"internal/modules/care/**",
			"internal/modules/residents/**",
			"internal/platform/api/**",
			"docs/openapi.yaml",
			"docs/**",
			"i18n/**",
			"README.md",
		}),
	}

	normalized := normalizeCompositeDirectorDecisionBatchV0(
		orquestacoreworkflow.OrchestrationRunV0{},
		decisions,
	)
	identity := normalized[1].CreateMicrotask.Task
	residents := normalized[2].CreateMicrotask.Task
	care := normalized[3].CreateMicrotask.Task

	assertContainsForParallelWriteSetTestV0(t, identity.WriteSet, "internal/platform/api/fragments/identity-base")
	assertContainsForParallelWriteSetTestV0(t, identity.WriteSet, "docs/openapi/fragments/identity-base.yaml")
	assertContainsForParallelWriteSetTestV0(t, identity.WriteSet, "docs/modules/identity-base.md")
	assertContainsForParallelWriteSetTestV0(t, identity.WriteSet, "i18n/modules/identity-base")
	assertContainsForParallelWriteSetTestV0(t, identity.AcceptanceCriteria, compositeParallelWriteSetCriterionV0)

	assertContainsForParallelWriteSetTestV0(t, residents.WriteSet, "internal/modules/residents/**")
	assertContainsForParallelWriteSetTestV0(t, care.WriteSet, "internal/modules/care/**")
	assertContainsForParallelWriteSetTestV0(t, care.WriteSet, "internal/modules/care-events/integrations/residents")
	assertNoDuplicateSharedWriteSetForParallelWriteSetTestV0(t, []orquestadirectoragent.DirectorAgentMicrotaskV0{
		identity,
		residents,
		care,
	})
}

func parallelWriteSetDecisionForTestV0(
	taskID string,
	title string,
	dependsOn []string,
	writeSet []string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		RunID:         "run-ref-parallel-write-set",
		PhaseID:       "planificacion_microtareas",
		DecisionRef:   "decision-" + taskID,
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-" + taskID,
		Summary:       title,
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion:      orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:             taskID,
				RunID:              "run-ref-parallel-write-set",
				PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Title:              title,
				Summary:            "Crear app Go modular con API y pruebas.",
				WriteSet:           writeSet,
				AcceptanceCriteria: []string{"objetivo_actual: crear_app_completa normal"},
				RequiredTests:      []string{"go test ./..."},
				DependsOn:          dependsOn,
			},
		},
	}
}

func assertNoDuplicateSharedWriteSetForParallelWriteSetTestV0(
	t *testing.T,
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) {
	t.Helper()
	counts := map[string]int{}
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			if compositePathCanSerializeSiblingTasksV0(compositeNormalizePathTokenV0(path)) {
				counts[compositeNormalizePathTokenV0(path)]++
			}
		}
	}
	for path, count := range counts {
		if count > 1 {
			t.Fatalf("write_set compartido %s aparece %d veces", path, count)
		}
	}
}

func assertContainsForParallelWriteSetTestV0(t *testing.T, values []string, want string) {
	t.Helper()
	if !stringInSetV0(values, want) {
		t.Fatalf("%q no encontrado en %v", want, values)
	}
}
