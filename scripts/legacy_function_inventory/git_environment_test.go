// Esta prueba fija que los procesos Git no heredan secretos ni configuración.
package main

import (
	"strings"
	"testing"
)

func TestGitCommandUsesMinimalExplicitEnvironment(t *testing.T) {
	t.Setenv("INVENTORY_SECRET_SENTINEL", "no-heredar")
	command, err := newGitCommand(t.TempDir(), "rev-parse", "--show-object-format")
	if err != nil {
		t.Fatal(err)
	}
	environment := strings.Join(command.Env, "\n")
	for _, required := range []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_REPLACE_OBJECTS=1",
	} {
		if !strings.Contains(environment, required) {
			t.Fatalf("falta entorno mínimo %q en %q", required, environment)
		}
	}
	if strings.Contains(environment, "INVENTORY_SECRET_SENTINEL") ||
		strings.Contains(environment, "no-heredar") {
		t.Fatalf("el proceso heredaría un secreto: %q", environment)
	}
	arguments := strings.Join(command.Args, "\n")
	if !strings.Contains(arguments, "core.hooksPath=/dev/null") {
		t.Fatalf("los hooks no quedaron anulados: %q", arguments)
	}
}
