// Estas pruebas verifican que los lectores Git no heredan secretos ni
// configuración mutable del proceso que ejecuta el censo.
package main

import (
	"strings"
	"testing"
)

func TestGitCommandUsesOnlyExplicitNonSensitiveEnvironment(t *testing.T) {
	t.Setenv("ORQUESTA_SECRETO_ARBITRARIO", "no-debe-llegar")
	t.Setenv("SSH_AUTH_SOCK", "/ruta/sensible")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "alias.log")
	t.Setenv("GIT_CONFIG_VALUE_0", "!false")

	command := gitCommand(t.TempDir(), "version")
	allowed := map[string]string{
		"GCM_INTERACTIVE":        "Never",
		"GIT_ATTR_NOSYSTEM":      "1",
		"GIT_CONFIG_GLOBAL":      "/dev/null",
		"GIT_CONFIG_NOSYSTEM":    "1",
		"GIT_CONFIG_SYSTEM":      "/dev/null",
		"GIT_NO_LAZY_FETCH":      "1",
		"GIT_NO_REPLACE_OBJECTS": "1",
		"GIT_OPTIONAL_LOCKS":     "0",
		"GIT_PAGER":              "cat",
		"GIT_TERMINAL_PROMPT":    "0",
		"LANG":                   "C",
		"LC_ALL":                 "C",
	}
	if len(command.Env) != len(allowed) {
		t.Fatalf("entorno Git=%q; se esperaban %d entradas", command.Env, len(allowed))
	}
	for _, assignment := range command.Env {
		name, value, found := strings.Cut(assignment, "=")
		if !found || allowed[name] != value {
			t.Fatalf("variable no autorizada en el proceso Git: %q", assignment)
		}
	}
	arguments := strings.Join(command.Args, "\x00")
	for _, required := range []string{
		"--no-pager",
		"--no-replace-objects",
		"core.hooksPath=/dev/null",
		"core.fsmonitor=false",
		"credential.helper=",
	} {
		if !strings.Contains(arguments, required) {
			t.Fatalf("falta el endurecimiento Git %q en %q", required, command.Args)
		}
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("el entorno mínimo no permite ejecutar Git: %v: %s", err, output)
	}
}
