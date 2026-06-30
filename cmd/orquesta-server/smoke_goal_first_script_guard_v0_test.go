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

func TestSmokeGoalFirstAppServerRealDisablesResidentGoalObserverV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"export ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false",
		"export ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED=false",
		"/api/v0/apps/director/goal/observe",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke manual puede competir con observador residente: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealRetriesTransientObservationRejectedV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		`grep -q "codex_goal_observation_rejected" "$observe_response"`,
		"goal_status=transient_observation_rejected",
		`continue`,
		"timeout esperando cierre aceptado goal-first",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke no reintenta observe transient rechazado: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealDiagnosesAppServerAuthMissingV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"print_app_server_failure_diagnostics",
		`find "$project_dir/generated-apps" -mindepth 1 -print -quit 2>/dev/null || true`,
		"generated_apps_present=0",
		"Missing bearer or basic authentication",
		"401 Unauthorized",
		"codex_app_server_failure_reason=codex_app_server_auth_missing",
		"codex_app_server_auth_missing=true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke no diagnostica auth ausente del app-server: falta %q", want)
		}
	}
}
