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

func TestSmokeGoalFirstAppServerRealRespetaPresupuestoNuevaAppV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		`export ORQUESTA_CODEX_GOAL_TIMEOUT_MS="${ORQUESTA_CODEX_GOAL_TIMEOUT_MS:-600000}"`,
		`polls="${ORQUESTA_GOAL_FIRST_SMOKE_POLLS:-120}"`,
		`sleep_seconds="${ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS:-5}"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke Nueva App no respeta presupuesto goal-first de 600s: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealUsaSandboxEfectivoEnWorkspaceAisladoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		`export ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-danger-full-access}"`,
		"workspace-write en app-server no materializa herramientas locales",
		"proyecto, runtime y CODEX_HOME son temporales bajo smoke_root",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke Nueva App debe usar sandbox efectivo en workspace aislado: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealDiagnosesAppServerAuthMissingV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"print_app_server_failure_diagnostics",
		`find "$project_dir/generated-apps" -type f -print -quit 2>/dev/null || true`,
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

func TestSmokeGoalFirstAppServerRealProjectsCodeHomeFromCodexHomeV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"configure_smoke_codex_code_home_source",
		`-f "$CODEX_HOME/auth.json"`,
		`-f "$CODEX_HOME/config.toml"`,
		`export ORQUESTA_CODEX_CODE_HOME="$CODEX_HOME"`,
		"codex_app_server_auth_source=CODEX_HOME",
		"codex_app_server_auth_source=ORQUESTA_CODEX_CODE_HOME",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke no proyecta CODEX_HOME autenticado al app-server tmux: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealNoLanzaAutomejoraIdleV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE=0",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS=0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke Nueva App debe aislar automejora idle: falta %q", want)
		}
	}
}

func TestSmokeSelfProgrammingCompositeGoalFirstGuardsV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_self_programming_composite_goal_first.sh")

	for _, want := range []string{
		"ORQUESTA_SELF_PROGRAMMING_COMPOSITE_SMOKE_CONFIRM",
		"app_server_tmux",
		"stdio|app_server_proxy",
		`"$(id -u)" == "0"`,
		"ORQUESTA_SERVER_SELF_PROGRAMMING_ONLY=true",
		"ORQUESTA_SERVER_SELF_PROGRAMMING_ROOT",
		"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED=true",
		"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR",
		`trap 'smoke_temp_root_cleanup "$smoke_root"`,
		`export GOMODCACHE="$gomodcache"`,
		`export GOPATH="$gopath"`,
		"ORQUESTA_OPES_BASE_URL",
		"OPES_BASE_URL",
		"ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL",
		"smoke_goal_first_app_server_real.sh",
		"ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1",
		"TestCodexStackAutoprogrammingPromotionV0GoalFirstE2ERepoTemporalReplayV0",
		"TestGitStagingPromotionConnectorV0PromocionaYArchivaSinBorrarV0",
		"smoke_self_programming_composite_goal_first=ok",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke self-programming compuesto sin guarda/evidencia requerida: falta %q", want)
		}
	}
}
