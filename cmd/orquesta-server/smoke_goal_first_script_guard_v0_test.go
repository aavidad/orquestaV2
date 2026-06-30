package main

import (
	"strings"
	"testing"
)

func TestSmokeGoalFirstAppServerRealShutdownEvidenceOnGoalFailureV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"fail_after_app_server_tmux_shutdown_ready 1",
		"assert_app_server_tmux_shutdown_ready",
		"goal-first no cerro aceptado; comprobando shutdown app_server_tmux antes de fallar",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke sin evidencia shutdown en fallo goal-first: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealProcessCounterDoesNotCountItselfV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"current_pid = str(os.getpid())",
		"parts = line.strip().split(None, 1)",
		"parts[0] == current_pid",
		`"codex" in parts[1] and "app-server" in parts[1] and socket in parts[1]`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("contador de procesos app-server puede contarse a si mismo: falta %q", want)
		}
	}
}
