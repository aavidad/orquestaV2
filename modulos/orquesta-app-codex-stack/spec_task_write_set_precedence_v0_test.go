package orquestaappcodexstack

import (
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgrammingObjectiveV0RespetaWriteSetClosed(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:          "task-ref-write-set-closed-001",
		Title:           "Refactor acotado",
		Summary:         "Resolver refactor acotado.",
		WorkProfileKind: orquestacoreworkflow.WorkProfileRefactorV0,
		WriteSet:        []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{
			"Cambio implementado dentro del write-set declarado.",
		},
	}

	objective := programmingObjectiveV0(task, orquestaruntime.LaunchRuntimeAgentRequestV0{})

	for _, want := range []string{
		"Usa el write-set como alcance de escritura.",
		"no edites fuera",
		"conserva lo util",
		"nota de revision o tarea derivada",
	} {
		if !strings.Contains(objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, objective)
		}
	}
	for _, forbidden := range []string{
		"App Go completa",
		"si debes tocar otros ficheros",
		"justificado en el ACK",
		"justificando cualquier toque fuera del write-set",
		"CONSULTA AL DIRECTOR",
		"decision explicita del director",
		"policy opt-in distinta",
	} {
		if strings.Contains(strings.ToLower(objective), strings.ToLower(forbidden)) {
			t.Fatalf("objective conserva autorizacion incompatible %q:\n%s", forbidden, objective)
		}
	}
}
