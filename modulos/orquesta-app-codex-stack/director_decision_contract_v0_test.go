package orquestaappcodexstack

import (
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestDirectorDecisionInstructionsV0UsaPoliticaPositivaDeRefsOpacas(t *testing.T) {
	instructions := strings.Join(directorDecisionInstructionsV0(
		orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:   "run-ref-director-decision-policy-001",
			TaskRef: "task-ref-director-decision-policy-001",
		},
	), "\n")

	for _, forbidden := range []string{
		"No escribas terminos prohibidos",
		"provider, model, db, sql",
		"runtime, adapter, adaptador",
	} {
		if strings.Contains(instructions, forbidden) {
			t.Fatalf("prompt conserva lista negativa %q en:\n%s", forbidden, instructions)
		}
	}
	for _, want := range []string{
		"refs opacas",
		"runtime",
		"provider",
		"no escribas valores reales",
		"rutas privadas",
		"credenciales",
		"prompts crudos",
		"conversaciones crudas",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("prompt no incluye politica esperada %q en:\n%s", want, instructions)
		}
	}
}
