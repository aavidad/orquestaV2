package cmd

import (
	"testing"

	"orquesta/db"
)

func TestShouldOpenRecoveryReadOnlyDB(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "status local", args: []string{"status", "--local"}, want: true},
		{name: "runtime ordenes local", args: []string{"runtime", "ordenes", "--local"}, want: true},
		{name: "runtime mailbox local", args: []string{"runtime", "mailbox", "--local"}, want: true},
		{name: "runtime orden nueva no readonly", args: []string{"runtime", "orden-nueva", "--local"}, want: false},
		{name: "tarea listar no cubierta", args: []string{"tarea", "listar", "--local"}, want: false},
	}
	for _, tc := range cases {
		if got := shouldOpenRecoveryReadOnlyDB(tc.args); got != tc.want {
			t.Fatalf("%s: shouldOpenRecoveryReadOnlyDB(%v)=%v want %v", tc.name, tc.args, got, tc.want)
		}
	}
}

func TestOpenDBForCommandRechazaMutacionesEnRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()

	if err := openDBForCommand([]string{"runtime", "orden-nueva"}); err == nil {
		t.Fatalf("se esperaba rechazo para mutacion en recuperacion local")
	}
	if err := openDBForCommand([]string{"tarea", "listar"}); err == nil {
		t.Fatalf("se esperaba rechazo para listado no cubierto en recuperacion local")
	}
}

// TestLocalRecovery verifica que localRecoveryCommandAllowed bloquea explícitamente
// comandos de escritura y permite comandos de lectura en modo local de recuperación.
func TestLocalRecovery(t *testing.T) {
	bloqueados := [][]string{
		{"tarea", "nueva", "--local"},
		{"tarea", "completar", "--local"},
		{"tarea", "iniciar", "--local"},
		{"tarea", "cancelar", "--local"},
		{"runtime", "orden-nueva", "--local"},
		{"runtime", "mailbox-enviar", "--local"},
		{"runtime", "nudge", "--local"},
		{"agente", "control", "--local"},
		{"agente", "ejecutar", "--local"},
	}
	for _, args := range bloqueados {
		if localRecoveryCommandAllowed(args) {
			t.Errorf("localRecoveryCommandAllowed(%v) = true; esperaba false (comando de escritura)", args)
		}
	}
	permitidos := [][]string{
		{"status", "--local"},
		{"runtime", "listar", "--local"},
		{"runtime", "diagnostico", "--local"},
		{"runtime", "transcript", "--local"},
		{"config", "set", "server_autobootstrap_enabled", "true", "--local"},
	}
	for _, args := range permitidos {
		if !localRecoveryCommandAllowed(args) {
			t.Errorf("localRecoveryCommandAllowed(%v) = false; esperaba true (comando de lectura)", args)
		}
	}
}

func TestOpenDBForCommandPermiteConfigSetEnLocalExplicito(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()

	db.Close()
	if err := openDBForCommand([]string{"config", "set", "server_autobootstrap_enabled", "true"}); err != nil {
		t.Fatalf("se esperaba permitir config set en local explicito: %v", err)
	}
	db.Close()
}

func TestEnsureLocalDBRechazaComandosNoDiagnosticosEnRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()
	defer cambiarArgs(t, []string{"orquesta", "runtime", "orden-nueva"})()

	db.Close()
	if err := ensureLocalDB(); err == nil {
		t.Fatalf("se esperaba rechazo en ensureLocalDB para mutacion local")
	}
	db.Close()
}
