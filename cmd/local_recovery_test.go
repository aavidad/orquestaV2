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

func TestEnsureLocalDBRechazaComandosNoDiagnosticosEnRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()
	defer cambiarArgs(t, []string{"orquesta", "runtime", "orden-nueva"})()

	db.Close()
	if err := ensureLocalDB(); err == nil {
		t.Fatalf("se esperaba rechazo en ensureLocalDB para mutacion local")
	}
	db.Close()
}
