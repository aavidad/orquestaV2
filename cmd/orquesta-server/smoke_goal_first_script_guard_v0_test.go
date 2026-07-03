package main

import (
	"os"
	"path/filepath"
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

func TestSmokeGoalFirstAppServerRealPideCleanupGoalBackendsV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"cleanup_goal_backends",
		"assert_app_server_tmux_shutdown_ready",
		"app_server_tmux_shutdown_ready=true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke debe pedir cleanup gobernado de backend goal: falta %q", want)
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

func TestSmokeGoalFirstAppServerRealShutdownToleraTmuxYaCerradoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"tmux session ya estaba cerrada antes del shutdown",
		`if [[ -n "$session_name" ]] && tmux has-session -t "$session_name"`,
		`if [[ -n "$owner_file" && -e "$owner_file" ]]`,
		`if [[ -n "$socket_path" && -e "$socket_path" ]]`,
		`if [[ -n "$pane_pid" ]] && kill -0 "$pane_pid"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke debe tolerar app-server tmux ya cerrado antes de shutdown: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstAppServerRealShutdownLimpiaBackendPropioYReintentaV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"cleanup_app_server_tmux_for_shutdown_retry",
		`"$session_name" != orquesta-goal-*`,
		`"$shutdown_status" == "409" && "$(json_get "$shutdown_response" "status")" == "backend_still_running"`,
		"app_server_tmux_cleanup_retry=session:",
		"stop_app_server_processes_for_socket",
		"kill -KILL",
		"smoke_goal_first_app_server_real_backend_cleanup_retry",
		"POST /api/v0/server/shutdown retry -> HTTP",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke debe limpiar backend tmux propio y reintentar shutdown: falta %q", want)
		}
	}
}

func TestSmokeCommonShutdownCleanupBackendGoalSiWrapperCancelaV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	common := readOperationalDocGuardV0(t, root, "scripts/lib/smoke_common.sh")
	smoke := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		`local runtime_dir="${5:-}"`,
		"cleanup_goal_backends",
		"smoke_cleanup_codex_app_server_tmux_runtime",
		"smoke_stop_codex_app_server_runtime_owned_processes",
		`owner == "orquesta-codex-goal-app-server-tmux-v0"`,
		`session.startswith("orquesta-goal-")`,
		`"codex" not in raw or "app-server" not in raw`,
		"CODEX_HOME=",
		"kill -KILL",
	} {
		if !strings.Contains(common, want) {
			t.Fatalf("smoke common no limpia backend goal propio al cancelar wrapper: falta %q", want)
		}
	}
	if !strings.Contains(smoke, `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"`) {
		t.Fatalf("smoke goal-first no pasa runtime_dir al cleanup comun")
	}
}

func TestSmokesAisladosPasanRuntimeDirAlShutdownComunV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	checks := map[string]string{
		"scripts/smoke_goal_first_app_server_real.sh":            `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"`,
		"scripts/smoke_autoprogramming_supervised.sh":            `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"`,
		"scripts/smoke_codex_required_test_runner_state_file.sh": `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"`,
		"scripts/smoke_orquesta_server_rest_director.sh":         `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"`,
		"scripts/smoke_autoprogramming_bolsa_real.sh":            `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"`,
		"scripts/smoke_external_domain_fake_real.sh":             `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"`,
		"scripts/smoke_external_domain_non_opes_real.sh":         `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"`,
		"scripts/smoke_opes_reviews_providers_real.sh":           `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 40 "$RUNTIME_DIR"`,
		"scripts/lib/opes_agent_smoke_ops.sh":                    `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$RUNTIME_DIR"`,
		"scripts/smoke_orquesta_server_restart_state.sh":         `smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$runtime_dir"`,
	}
	for path, want := range checks {
		text := readOperationalDocGuardV0(t, root, path)
		if !strings.Contains(text, want) {
			t.Fatalf("%s no pasa runtime al shutdown comun: falta %q", path, want)
		}
	}
}

func TestSmokeOPESReviewsProvidersRealExigeOptInLegacyYWorkdirOPESV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_opes_reviews_providers_real.sh")

	for _, want := range []string{
		"ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP",
		"ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP",
		"director_execution_mode=legacy_director_loop",
		"ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true",
		"repo OPES real",
		"external/opes/",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke OPES proveedores debe fijar opt-in legacy y workdir seguro: falta %q", want)
		}
	}
}

func TestScriptsQueArrancanServidorTemporalUsanShutdownComunV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	scriptsDir := filepath.Join(root, "scripts")
	err := filepath.WalkDir(scriptsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".sh" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		text := readOperationalDocGuardV0(t, root, rel)
		startsTemporaryServer := scriptStartsTemporaryOrquestaServerV0(text)
		if !startsTemporaryServer {
			return nil
		}
		if !strings.Contains(text, "smoke_shutdown_orquesta_server") {
			t.Fatalf("%s arranca servidor temporal sin shutdown comun", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func scriptStartsTemporaryOrquestaServerV0(text string) bool {
	if scriptRunsTemporaryOrquestaServerCommandV0(text) {
		return true
	}
	if !strings.Contains(text, "ORQUESTA_SERVER_ADDR") {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "$!") {
			return true
		}
	}
	return false
}

func scriptRunsTemporaryOrquestaServerCommandV0(text string) bool {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.Contains(trimmed, "orquesta-server") ||
			!strings.Contains(trimmed, " run") {
			continue
		}
		for _, candidate := range lines[index:] {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" || strings.HasPrefix(candidate, "#") {
				continue
			}
			if shellLineEndsInBackgroundV0(candidate) {
				return true
			}
			if !strings.HasSuffix(candidate, `\`) {
				break
			}
		}
	}
	return false
}

func shellLineEndsInBackgroundV0(line string) bool {
	line = strings.TrimSpace(line)
	return line == "&" || strings.HasSuffix(line, " &")
}

func TestScriptRunsTemporaryOrquestaServerCommandV0DetectaBackgroundMultilineaLargoV0(t *testing.T) {
	text := strings.Join([]string{
		`ORQUESTA_SERVER_ADDR="127.0.0.1:0" \`,
		`ORQUESTA_SERVER_STATE_DIR="$state_dir" \`,
		`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \`,
		`ORQUESTA_CODEX_PROJECT_WORKDIR="$project_dir" \`,
		`ORQUESTA_CODEX_RUNTIME_WORKDIR="$runtime_dir" \`,
		`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true \`,
		`go run ./cmd/orquesta-server run \`,
		`  --some-local-flag \`,
		`  >"$server_log" 2>&1 &`,
	}, "\n")

	if !scriptRunsTemporaryOrquestaServerCommandV0(text) {
		t.Fatalf("detector debe cubrir comandos orquesta-server run en background aunque el & este lejos")
	}
}

func TestScriptRunsTemporaryOrquestaServerCommandV0IgnoraForegroundMultilineaV0(t *testing.T) {
	text := strings.Join([]string{
		`ORQUESTA_SERVER_ADDR="127.0.0.1:0" \`,
		`go run ./cmd/orquesta-server run \`,
		`  --some-local-flag >"$server_log" 2>&1`,
	}, "\n")

	if scriptRunsTemporaryOrquestaServerCommandV0(text) {
		t.Fatalf("detector no debe marcar comandos foreground sin background")
	}
}

func TestScriptsQueDeleganArranqueServidorTemporalInstalanCleanupV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	scriptsDir := filepath.Join(root, "scripts")
	err := filepath.WalkDir(scriptsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".sh" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		text := readOperationalDocGuardV0(t, root, rel)
		if !scriptInvokesOrquestaServerStartHelperV0(text) {
			return nil
		}
		if !strings.Contains(text, "trap ") || !strings.Contains(text, " EXIT") {
			t.Fatalf("%s delega arranque de servidor temporal sin trap EXIT de cleanup", rel)
		}
		if !strings.Contains(text, "smoke_shutdown_orquesta_server") &&
			!strings.Contains(text, "trap smoke_cleanup EXIT") {
			t.Fatalf("%s delega arranque de servidor temporal sin cleanup comun directo o delegado", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func scriptInvokesOrquestaServerStartHelperV0(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "func ") ||
			strings.HasSuffix(trimmed, "() {") {
			continue
		}
		if strings.HasPrefix(trimmed, "smoke_start_orquesta_server") ||
			strings.HasPrefix(trimmed, "start_orquesta_server") {
			return true
		}
	}
	return false
}

func TestScriptsQueUsanShutdownComunPasanRuntimeDirV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	scriptsDir := filepath.Join(root, "scripts")
	err := filepath.WalkDir(scriptsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".sh" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		text := readOperationalDocGuardV0(t, root, rel)
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "smoke_shutdown_orquesta_server ") {
				continue
			}
			if !strings.Contains(trimmed, `"$runtime_dir"`) &&
				!strings.Contains(trimmed, `"$RUNTIME_DIR"`) {
				t.Fatalf("%s invoca shutdown comun sin runtime_dir: %s", rel, trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func TestScriptsConShutdownDirectoPidenCleanupGoalBackendsV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	scriptsDir := filepath.Join(root, "scripts")
	err := filepath.WalkDir(scriptsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".sh" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		text := readOperationalDocGuardV0(t, root, rel)
		lines := strings.Split(text, "\n")
		for index, line := range lines {
			if !strings.Contains(line, "/api/v0/server/shutdown") {
				continue
			}
			windowStart := index - 3
			if windowStart < 0 {
				windowStart = 0
			}
			windowEnd := index + 8
			if windowEnd > len(lines) {
				windowEnd = len(lines)
			}
			window := strings.Join(lines[windowStart:windowEnd], "\n")
			if !strings.Contains(window, "curl") || !strings.Contains(window, "-X POST") {
				continue
			}
			if !strings.Contains(window, "cleanup_goal_backends") {
				t.Fatalf("%s invoca shutdown HTTP directo sin cleanup_goal_backends cerca de linea %d", rel, index+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func TestInicioAgenteNoRecomiendaRuntimeManualV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/inicio_agente.sh")

	if strings.Contains(text, "go run ./cmd/orquesta-server run") {
		t.Fatalf("inicio_agente no debe recomendar runtime manual no gobernado")
	}
	if strings.Contains(text, `ORQUESTA_SERVER_URL:-http://127.0.0.1:8787`) {
		t.Fatalf("inicio_agente no debe asumir puerto historico por defecto")
	}
	for _, want := range []string{
		"orquesta-server start",
		"servidor residente gobernado",
		"smoke_require_orquesta_base_url",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("inicio_agente debe orientar a servidor gestionado: falta %q", want)
		}
	}
}

func TestOrquestaStatusNowUsaEndpointGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/orquesta_status_now.sh")

	for _, forbidden := range []string{
		`ORQUESTA_SERVER_URL:-http://127.0.0.1:8787`,
		"Default: http://127.0.0.1:8787",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("orquesta_status_now no debe asumir puerto historico: %q", forbidden)
		}
	}
	for _, want := range []string{
		"smoke_require_orquesta_base_url",
		"ORQUESTA_RUNTIME_DIR",
		"base_url.txt",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("orquesta_status_now debe usar endpoint gestionado: falta %q", want)
		}
	}
}

func TestArrancarCodexModuloNoRecomiendaRuntimeManualV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	modulesDir := filepath.Join(root, "modulos")
	err := filepath.WalkDir(modulesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "arrancar_codex.sh" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		text := readOperationalDocGuardV0(t, root, rel)
		if strings.Contains(text, "go run ./cmd/orquesta-server run") {
			t.Fatalf("%s recomienda runtime manual no gobernado", rel)
		}
		if strings.Contains(text, "ruta_vigente:") &&
			!strings.Contains(text, "orquesta-server start") &&
			!strings.Contains(text, "servidor residente/cola OrquestaV2") {
			t.Fatalf("%s debe orientar a servidor gestionado o cola residente", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk modulos: %v", err)
	}
}

func TestUsoActualAppOrquestaRecomiendaServidorGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/uso_actual_app_orquesta.md")
	current, _, _ := strings.Cut(text, "## Contenido historico V1 preservado")

	if strings.Contains(current, "```bash\ngo run ./cmd/orquesta-server run\n```") {
		t.Fatalf("uso actual no debe recomendar runtime manual no gobernado en seccion vigente")
	}
	if strings.Contains(current, "go run ./cmd/orquesta-server run") {
		t.Fatalf("uso actual no debe enseñar el comando runtime manual exacto en seccion vigente")
	}
	if strings.Contains(current, "127.0.0.1:8787") {
		t.Fatalf("uso actual no debe asumir puerto historico en seccion vigente")
	}
	for _, want := range []string{
		"orquesta-server start",
		"orquesta-server stop",
		"queda solo para harnesses",
		"smoke_shutdown_orquesta_server",
		"runtime_dir",
		"orquesta-server status --json",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
	} {
		if !strings.Contains(current, want) {
			t.Fatalf("uso actual debe documentar arranque/parada gestionados: falta %q", want)
		}
	}
}

func TestReadmesOperativosNoRecomiendanRuntimeManualV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"modulos/orquesta-server/README.md",
		"modulos/orquesta-opes-bridge/README.md",
		"modulos/orquesta-core-workflow/README.md",
	} {
		t.Run(rel, func(t *testing.T) {
			text := readOperationalDocGuardV0(t, root, rel)
			if strings.Contains(text, "go run ./cmd/orquesta-server run") {
				t.Fatalf("%s recomienda runtime manual no gobernado", rel)
			}
			for _, want := range []string{"orquesta-server start", "orquesta-server stop"} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s debe documentar arranque/parada gestionados: falta %q", rel, want)
				}
			}
		})
	}
}

func TestPruebasServidorNoRecomiendaPuertoHistoricoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "modulos/orquesta-server/docs/pruebas.md")
	if strings.Contains(text, "curl http://127.0.0.1:8787") {
		t.Fatalf("pruebas servidor no debe recomendar puerto historico con curl directo")
	}
	if strings.Contains(text, "Prueba manual recomendada:\n  - `go run ./cmd/orquesta-server run`") {
		t.Fatalf("pruebas servidor no debe recomendar runtime manual como camino operativo")
	}
	for _, want := range []string{
		"orquesta-server start",
		"orquesta-server status --json",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"ORQUESTA_SERVER_URL",
		"orquesta-server stop",
		"smoke_shutdown_orquesta_server",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("pruebas servidor debe documentar endpoint gestionado: falta %q", want)
		}
	}
}

func TestHandoffTerminarOrquestaUsaServidorGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/handoff_terminar_orquesta_2026-06-19.md")
	current, _, _ := strings.Cut(text, "# ESTADO DE CIERRE Y PENDIENTES")

	if strings.Contains(current, "127.0.0.1:8799") {
		t.Fatalf("handoff vigente no debe fijar puerto manual para reproduccion")
	}
	for _, forbidden := range []string{
		"  ./orquesta-server run\n# luego:",
		"POST el spec a /api/v0/autoprogramming/prepare-run",
	} {
		if strings.Contains(current, forbidden) {
			t.Fatalf("handoff vigente conserva reproduccion manual no gobernada: %q", forbidden)
		}
	}
	for _, want := range []string{
		"orquesta-server start",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"ORQUESTA_SERVER_URL",
		"orquesta-server status --json",
		"orquesta-server stop",
		"smoke_shutdown_orquesta_server",
		"cleanup_goal_backends",
	} {
		if !strings.Contains(current, want) {
			t.Fatalf("handoff vigente debe usar servidor gestionado: falta %q", want)
		}
	}
}

func TestHandoffGoalFirstParadaUsaServidorGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/handoff_orquesta_goal_first_parada_2026-06-26.md")
	current, _, _ := strings.Cut(text, "## Cierre local posterior")

	if strings.Contains(current, "servidor web local `127.0.0.1:8787`") {
		t.Fatalf("handoff goal-first parada no debe pedir reinicio por puerto historico")
	}
	for _, want := range []string{
		"orquesta-server start",
		"orquesta-server status --json",
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"orquesta-server stop",
		"No asumir el puerto historico `8787`",
	} {
		if !strings.Contains(current, want) {
			t.Fatalf("handoff goal-first parada debe usar servidor gestionado: falta %q", want)
		}
	}
}

func TestRunbookPruebasLocalesUsaEndpointGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/runbooks/pruebas_locales_orquesta_2026-05-25.md")
	if strings.Contains(text, "http://127.0.0.1:8787") {
		t.Fatalf("runbook pruebas locales no debe asumir puerto historico")
	}
	for _, want := range []string{
		"orquesta-server start",
		"orquesta-server status --json",
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"$ORQUESTA_SERVER_URL/api/v0/autoprogramming/prepare-run",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("runbook pruebas locales debe usar endpoint gestionado: falta %q", want)
		}
	}
}

func TestRunbookAutoprogramacionCLIUsaEndpointGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/runbooks/autoprogramacion_cli_2026-05-23.md")
	if strings.Contains(text, "http://127.0.0.1:8787") {
		t.Fatalf("runbook autoprogramacion CLI no debe asumir puerto historico")
	}
	for _, want := range []string{
		"orquesta-server status --json",
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		`--server-url "$ORQUESTA_SERVER_URL"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("runbook autoprogramacion CLI debe usar endpoint gestionado: falta %q", want)
		}
	}
}

func TestRunbookPanelOpsUsaEndpointGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/runbooks/panel_ops_orquesta_2026-05-24.md")
	if strings.Contains(text, "http://127.0.0.1:8787/ops") {
		t.Fatalf("runbook panel ops no debe asumir puerto historico")
	}
	for _, want := range []string{
		"orquesta-server status --json",
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"$ORQUESTA_SERVER_URL/ops",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("runbook panel ops debe usar endpoint gestionado: falta %q", want)
		}
	}
}

func TestLauncherOPESA1NoUsaPuertoHistoricoPorDefectoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/opes_a1_finalpkg_registry_launcher.py")

	if strings.Contains(text, `os.environ.get("ORQUESTA_BASE_URL", "http://127.0.0.1:8787")`) {
		t.Fatalf("launcher OPES A1 no debe caer al puerto historico 8787 por defecto")
	}
	for _, want := range []string{
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR",
		"base_url.txt",
		"--orquesta-base-url required",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("launcher OPES A1 debe resolver endpoint gestionado: falta %q", want)
		}
	}
}

func TestSmokesOPESRESTDirectosUsanEndpointOrquestaGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	common := readOperationalDocGuardV0(t, root, "scripts/lib/smoke_common.sh")
	for _, want := range []string{
		"smoke_orquesta_base_url_from_env_or_runtime",
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR",
		"base_url.txt",
	} {
		if !strings.Contains(common, want) {
			t.Fatalf("smoke_common no resuelve endpoint Orquesta gestionado: falta %q", want)
		}
	}
	for _, rel := range []string{
		"scripts/smoke_opes_domain_work_real.sh",
		"scripts/smoke_opes_visual_asset_real.sh",
	} {
		t.Run(rel, func(t *testing.T) {
			text := readOperationalDocGuardV0(t, root, rel)
			if strings.Contains(text, `ORQUESTA_BASE_URL="${ORQUESTA_BASE_URL:-http://127.0.0.1`) {
				t.Fatalf("%s no debe caer a puerto historico por defecto", rel)
			}
			if !strings.Contains(text, `ORQUESTA_BASE_URL="$(smoke_require_orquesta_base_url ORQUESTA_SERVER_URL)"`) {
				t.Fatalf("%s debe exigir endpoint Orquesta gestionado", rel)
			}
		})
	}
}

func TestSmokesOPESLargosAceptanEndpointOrquestaGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"scripts/smoke_opes_plan_temario_operadores.sh",
		"scripts/smoke_opes_derivatives_rest.sh",
	} {
		t.Run(rel, func(t *testing.T) {
			text := readOperationalDocGuardV0(t, root, rel)
			for _, want := range []string{
				"smoke_orquesta_base_url_from_env_or_runtime",
				"ORQUESTA_RUNTIME_DIR/base_url.txt",
				"falta endpoint Orquesta gestionado",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s debe aceptar endpoint Orquesta gestionado: falta %q", rel, want)
				}
			}
			if strings.Contains(text, "falta ORQUESTA_BASE_URL explicito") {
				t.Fatalf("%s no debe exigir solo ORQUESTA_BASE_URL", rel)
			}
		})
	}
}

func TestRunbookOPESPlanTemarioUsaServidorGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md")
	if strings.Contains(text, "Si `cmd/orquesta-server run` esta activo") {
		t.Fatalf("runbook OPES plan_temario no debe presentar runtime manual como ruta activa")
	}
	for _, want := range []string{
		"servidor residente gestionado",
		"orquesta-server start",
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"no se",
		"asume un runtime manual ni un puerto historico",
		"director_execution_mode=legacy_director_loop",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("runbook OPES plan_temario debe usar servidor gestionado: falta %q", want)
		}
	}
}

func TestOperationalDocsRuntimeManualMentionsAreHistoricalOrHarnessV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	allowed := map[string]string{
		"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md":                        "fuente historica",
		"docs/handoff_terminar_orquesta_2026-06-19.md":                                   "harness local de bajo nivel",
		"docs/incidencia_opes_project_workdir_revision_operario_2026-06-12.md":           "incidencia",
		"docs/rail_errors_observados_2026-05-23.md":                                      "errores de rail observados",
		"docs/runbooks/incidencia_opes_modo_automatico_desactivado_2026-06-13.md":        "incidencia",
		"docs/runbooks/incidencia_startup_lock_stale_codex_a2_informatica_2026-06-13.md": "incidencia",
		"modulos/orquesta-server/docs/pruebas.md":                                        "harnesses aislados",
	}
	needles := []string{
		"go run ./cmd/orquesta-server run",
		"./orquesta-server run",
		"cmd/orquesta-server run",
		"127.0.0.1:8787",
	}
	for _, base := range []string{"docs", "modulos"} {
		err := filepath.WalkDir(filepath.Join(root, base), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel == "docs/inventario_bugs_orquesta_2026-06-30.md" {
				return nil
			}
			text := readOperationalDocGuardV0(t, root, rel)
			hasMention := false
			for _, needle := range needles {
				if strings.Contains(text, needle) {
					hasMention = true
					break
				}
			}
			if !hasMention {
				return nil
			}
			reason, ok := allowed[rel]
			if !ok {
				t.Fatalf("%s contiene runtime manual/puerto historico sin clasificacion historica o harness", rel)
			}
			if !strings.Contains(strings.ToLower(text), reason) && !strings.Contains(text, reason) {
				t.Fatalf("%s permitido por %q, pero el documento no declara ese contexto", rel, reason)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
}

func TestMatrizOPESDerivadosUsaServidorGestionadoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/matriz_pruebas_reales_y_smoke_2026-05-17.md")
	start := strings.Index(text, "| OPES-DER-RESTO |")
	if start < 0 {
		t.Fatalf("matriz sin fila OPES-DER-RESTO")
	}
	rest := text[start:]
	end := strings.Index(rest, "\n| EXT-NO-OPES |")
	if end < 0 {
		t.Fatalf("matriz OPES-DER-RESTO sin cierre esperado")
	}
	row := rest[:end]

	if strings.Contains(row, "go run ./cmd/orquesta-server run") {
		t.Fatalf("OPES-DER-RESTO no debe recomendar runtime manual no gestionado")
	}
	for _, want := range []string{
		"orquesta-server start",
		"ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux",
		"ORQUESTA_RUNTIME_DIR/base_url.txt",
		"ORQUESTA_SERVER_URL",
		"orquesta-server stop",
	} {
		if !strings.Contains(row, want) {
			t.Fatalf("OPES-DER-RESTO debe documentar servidor gestionado: falta %q", want)
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

func TestSmokeGoalFirstAppServerRealPrintsPublicReadinessDiagnosticsOnStartupFailureV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"print_server_readiness_failure_diagnostics",
		`curl -sS -m 2 -o "$response_file" -w "%{http_code}" "http://$addr/api/v0/server/readiness"`,
		"readiness_http_status=",
		`"startup_status",`,
		`print(f"readiness_{key}={code}")`,
		`print(f"readiness_diagnostic_{index}_code={code}")`,
		`print(f"readiness_diagnostic_{index}_message={message}")`,
		"codex_app_server_auth_missing",
		`forbidden = (`,
		`"token", "secret", "bearer", "authorization",`,
		`print_server_readiness_failure_diagnostics "$server_addr"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke no imprime diagnostico publico de readiness al fallar startup: falta %q", want)
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
