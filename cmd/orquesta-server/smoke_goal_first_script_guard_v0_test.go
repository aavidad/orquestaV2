package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type scriptContractGuardV0 struct {
	name    string
	script  string
	wants   []string
	forbids []string
}

func TestSmokeScriptContractsConsolidadosV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, guard := range smokeScriptContractGuardsV0() {
		t.Run(guard.name, func(t *testing.T) {
			text := readOperationalDocGuardV0(t, root, guard.script)
			for _, want := range guard.wants {
				if !strings.Contains(text, want) {
					t.Fatalf("%s no cumple contrato %s: falta %q", guard.script, guard.name, want)
				}
			}
			for _, forbidden := range guard.forbids {
				if strings.Contains(text, forbidden) {
					t.Fatalf("%s incumple contrato %s: contiene %q", guard.script, guard.name, forbidden)
				}
			}
		})
	}
}

func TestSmokeGoalFirstScriptContractGuardV0(t *testing.T) {
	TestSmokeScriptContractsConsolidadosV0(t)
}

func TestF3DrainPidfdEsSemanticoYNoUsaKillPorPIDV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	script := filepath.Join(root, "scripts/lib/pidfd_signal.py")
	program := `
import ast, pathlib, sys
tree=ast.parse(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
calls=[]
for node in ast.walk(tree):
    if not isinstance(node, ast.Call): continue
    if isinstance(node.func, ast.Attribute): name=node.func.attr
    elif isinstance(node.func, ast.Name): name=node.func.id
    else: name=""
    calls.append((name,node.lineno))
names=[name for name,_ in calls]
assert "pidfd_open" in names and "pidfd_send_signal" in names
assert min(line for name,line in calls if name=="pidfd_open") < min(line for name,line in calls if name=="pidfd_send_signal")
assert not ({"kill","system","popen"} & set(names))
`
	command := exec.Command("python3", "-c", program, script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("helper pidfd no cumple AST semantico: %v\n%s", err, output)
	}
}

func smokeScriptContractGuardsV0() []scriptContractGuardV0 {
	return []scriptContractGuardV0{
		{
			name:   "goal-first-app-server-shutdown-governed",
			script: "scripts/smoke_goal_first_app_server_real.sh",
			wants: []string{
				"fail_after_app_server_tmux_shutdown_ready 1",
				"assert_app_server_tmux_shutdown_ready",
				"cleanup_goal_backends",
				`smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 25 "$runtime_dir"`,
				`export ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false`,
				"print_app_server_failure_diagnostics",
			},
		},
		{
			name:   "goal-first-high-consumption-wrapper",
			script: "scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh",
			wants: []string{
				"ORQUESTA" + "_GOAL_FIRST_SMOKE_HIGH_CONSUMPTION_MODE=1",
				`ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED:-true}"`,
				`ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS="${ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS:-1}"`,
				`exec "$repo_root/scripts/smoke_goal_first_app_server_real.sh" "$@"`,
			},
		},
		{
			name:   "goal-first-forced-stop-wrapper",
			script: "scripts/smoke_goal_first_forced_stop_backend_real.sh",
			wants: []string{
				"SMOKE" + "_GOAL_FIRST_FORCED_STOP_MODE=1",
				`ORQUESTA_CODEX_GOAL_BACKEND="${ORQUESTA_CODEX_GOAL_BACKEND:-app_server_tmux}"`,
				`ORQUESTA_CODEX_GOAL_TIMEOUT_MS="${ORQUESTA_CODEX_GOAL_TIMEOUT_MS:-600000}"`,
				`exec "$repo_root/scripts/smoke_goal_first_app_server_real.sh" "$@"`,
			},
		},
		{
			name:   "goal-first-shutdown-coordination-wrapper",
			script: "scripts/smoke_goal_first_shutdown_coordination_real.sh",
			wants: []string{
				"SMOKE" + "_GOAL_FIRST_SHUTDOWN_COORDINATION_MODE=1",
				"cleanup_goal_backends",
				"ORQUESTA_GOAL_FIRST_SHUTDOWN_COORDINATION_POLLS",
				`exec "$repo_root/scripts/smoke_goal_first_app_server_real.sh" "$@"`,
			},
			forbids: []string{"/api/v0/runs/control"},
		},
		{
			name:   "claude-process-safe-forced-stop-wrapper",
			script: "scripts/smoke_goal_first_claude_process_server_real.sh",
			wants: []string{
				"SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL",
				"SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE",
				"ORQUESTA_CODEX_GOAL_BACKEND=claude_process",
				"/api/v0/runs/control",
				"cleanup_goal_backends",
				"smoke_goal_first_claude_process_forced_stop_server_real=ok",
			},
		},
		{
			name:   "smoke-common-managed-endpoint",
			script: "scripts/lib/smoke_common.sh",
			wants: []string{
				"smoke_orquesta_base_url_from_env_or_runtime",
				"ORQUESTA_SERVER_URL",
				"ORQUESTA_RUNTIME_DIR",
				"base_url.txt",
			},
			forbids: []string{"127.0.0.1:8787", "localhost:8787"},
		},
		{
			name:   "deploy-script-is-managed-entrypoint",
			script: "scripts/orquesta_server_deploy.sh",
			wants: []string{
				"deploy_not_fast_forward",
				"deploy_config_missing",
				"deploy_runtime_identity_mismatch",
				"orquesta_server_ctl.sh",
				"scripts/lib/isolated_test_env.sh",
				"orquesta_use_isolated_test_env",
			},
		},
		{
			name:   "f5-ctl-validates-runtime-workdir",
			script: "scripts/orquesta_server_ctl.sh",
			wants: []string{
				"/srv/orquesta-self/worktrees/orquesta",
				"ctl_workdir_invalid",
				"ctl_workdir_stale",
				"ctl_workdir_not_aligned",
				"ORQUESTA_CODEX_PROJECT_WORKDIR=\"$WORKDIR\"",
				"ORQUESTA_CODEX_SANDBOX=workspace-write",
			},
			forbids: []string{"/srv/orquesta-self/worktrees/pilot-remoto-1"},
		},
		{
			name:   "f3-drain-script-governed",
			script: "scripts/lib/orquesta_drain_runtime.py",
			wants: []string{
				"orquesta_server_drain_receipt.v2",
				"orquesta_server_drain_inventory.v2",
				"protected_uso_app",
				"protected_uso_app_descendant",
				"orquesta_identity_incomplete",
				"backup_prepared_before_stop",
				"refused_precondition",
				"DRAIN_TEST_MODE",
				"drain_status",
				"/api/v0/server/shutdown",
				`"requested_by": "orquesta-director"`,
				`"cleanup_goal_backends": True`,
				"kill-session",
				"pidfd_helper",
				"urllib.parse.urlsplit",
				"SIGTERM",
				"SIGKILL",
			},
			forbids: []string{"/api/v0/runs/control", "os.kill", "kill -9", "pkill", "killall"},
		},
		{
			name:   "f3-isolated-test-env-profile",
			script: "scripts/lib/isolated_test_env.sh",
			wants: []string{
				"orquesta_use_isolated_test_env",
				"orquesta_session_disk_preflight.sh",
				"--preflight",
				`eval "$exports"`,
				"TMPDIR",
				"GOTMPDIR",
				"GOCACHE",
				"GOMODCACHE",
				"GOPATH",
				"CODEX_HOME",
				"XDG_RUNTIME_DIR",
				"ORQUESTA_FLAKY_HARNESS_CACHE_ROOT",
				"ORQUESTA_TEST_RUNTIME_ROOT",
				"ORQUESTA_TEST_PORT_BASE",
				"ORQUESTA_TEST_PORT_RANGE",
				"ORQUESTA_TEST_PORT_LOCK_DIR",
			},
		},
		{
			name:   "f3-wide-tests-run-in-observable-batches",
			script: "scripts/orquesta_test_batches.sh",
			wants: []string{
				"orquesta_use_isolated_test_env",
				"orquesta_test_batches_receipt.v1",
				"ORQUESTA_TEST_BATCH_SIZE",
				"ORQUESTA_TEST_BATCH_TIMEOUT",
				"--kill-after",
				"two_consecutive_passes_passed",
			},
			forbids: []string{`go test ./...`},
		},
		{
			name:   "f3-common-harness-consumers",
			script: "scripts/smoke_self_programming_composite_goal_first.sh",
			wants:  []string{"scripts/lib/isolated_test_env.sh", "orquesta_use_isolated_test_env"},
		},
		{
			name:   "f3-nightly-consumes-common-harness",
			script: "scripts/orquesta_smoke_nightly.sh",
			wants:  []string{"scripts/lib/isolated_test_env.sh", "orquesta_use_isolated_test_env"},
		},
	}
}

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
		`export ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED:-false}"`,
		"/api/v0/apps/director/goal/observe",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke manual puede competir con observador residente: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstHighConsumptionWrapperActivaObserverYUmbralBajoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	wrapper := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_checkpoint_only_high_consumption_real.sh")
	smoke := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"ORQUESTA" + "_GOAL_FIRST_SMOKE_HIGH_CONSUMPTION_MODE=1",
		`ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED="${ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED:-true}"`,
		`ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS="${ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS:-1}"`,
		`exec "$repo_root/scripts/smoke_goal_first_app_server_real.sh" "$@"`,
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("wrapper BUG-088 incompleto: falta %q", want)
		}
	}
	for _, want := range []string{
		"smoke_goal_first_high_consumption_real=ok",
		"bug088_path=checkpoint_only_replan",
		"bug088_path=no_checkpoint_replan",
		"bug088_path=second_artifact_or_terminal_artifact",
		"bug088_path=second_artifact_or_partial_artifacts",
		"generated-apps/bug088_second_artifact.txt",
		"recommended_action=replan_narrow_context",
		"recommended_action=review_partial_artifacts",
	} {
		if !strings.Contains(smoke, want) {
			t.Fatalf("smoke BUG-088 no reconoce replan alto consumo: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstForcedStopWrapperEjercitaRunControlBackendVivoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	wrapper := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_forced_stop_backend_real.sh")
	smoke := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"ORQUESTA" + "_GOAL_FIRST_SMOKE_HIGH_CONSUMPTION_MODE=1",
		"SMOKE" + "_GOAL_FIRST_FORCED_STOP_MODE=1",
		`ORQUESTA_CODEX_GOAL_BACKEND="${ORQUESTA_CODEX_GOAL_BACKEND:-app_server_tmux}"`,
		`ORQUESTA_CODEX_GOAL_TIMEOUT_MS="${ORQUESTA_CODEX_GOAL_TIMEOUT_MS:-600000}"`,
		`ORQUESTA_CODEX_SANDBOX="${ORQUESTA_CODEX_SANDBOX:-danger-full-access}"`,
		`ORQUESTA_CODEX_APPROVAL_POLICY="${ORQUESTA_CODEX_APPROVAL_POLICY:-never}"`,
		`exec "$repo_root/scripts/smoke_goal_first_app_server_real.sh" "$@"`,
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("wrapper forced-stop incompleto: falta %q", want)
		}
	}
	for _, want := range []string{
		`forced_stop_mode="${SMOKE_GOAL_FIRST_FORCED_STOP_MODE:-0}"`,
		"run_forced_stop_smoke",
		"post_autoprogramming_status_snapshot",
		"/api/v0/autoprogramming/status",
		`post_autoprogramming_status_snapshot "before_forced_stop"`,
		`post_autoprogramming_status_snapshot "after_forced_stop"`,
		`autoprogramming_status_${label}_visible=true`,
		`autoprogramming_status_${label}_not_running=true`,
		"/api/v0/runs/control",
		`"forced": true`,
		`"action": "stop"`,
		`"$control_status_value" != "stopped"`,
		`"$control_final_status" != "stopped"`,
		`"$post_stop_goal_status" == "running"`,
		"evidence-ref-observe-goal-run-control-terminal",
		"smoke_goal_first_forced_stop_backend_real=ok",
	} {
		if !strings.Contains(smoke, want) {
			t.Fatalf("smoke forced-stop no valida control/observe terminal: falta %q", want)
		}
	}
}

func TestSmokeGoalFirstShutdownCoordinationRealNoEsNoopV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_shutdown_coordination_real.sh")
	delegated := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")
	combined := text + "\n" + delegated

	for _, want := range []string{
		"app_server_tmux",
		"/api/v0/autoprogramming/status",
		"/api/v0/server/shutdown",
		`"${runs_requested:-0}" -lt 1`,
		`"${runs_stopped:-0}" -lt "${runs_requested:-0}"`,
		`"$run_control_all_stopped" != "true"`,
		"shutdown_coordination_runs_requested=$runs_requested",
		"shutdown_coordination_run_control_statuses=$run_control_statuses",
		"shutdown_coordination_all_runs_stopped=$run_control_all_stopped",
		"app_server_tmux_processes_alive=0",
		"smoke_goal_first_shutdown_coordination_real=ok",
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("smoke shutdown/status goal-first no debe ser no-op: falta %q", want)
		}
	}
	if !scriptContainsJSONBoolFieldV0(text, "cleanup_goal_backends", true) {
		t.Fatalf("smoke shutdown/status goal-first debe pedir cleanup_goal_backends=true")
	}
}

func TestSmokeGoalFirstShutdownCoordinationRealNoUsaRunsControlComoCaminoPrincipalV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_shutdown_coordination_real.sh")

	if strings.Contains(text, "/api/v0/runs/control") {
		t.Fatalf("smoke shutdown/status goal-first debe coordinar por status/shutdown, no por runs/control")
	}
}

func scriptContainsJSONBoolFieldV0(text string, field string, value bool) bool {
	normalized := strings.NewReplacer(`\`, "", " ", "", "\t", "", "\n", "").Replace(text)
	want := `"` + field + `":` + strings.ToLower(strconv.FormatBool(value))
	return strings.Contains(normalized, want)
}

func TestSmokeGoalFirstClaudeProcessServerRealEsOptInYLimpiaBackendV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_claude_process_server_real.sh")

	for _, want := range []string{
		"SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REAL",
		"SMOKE_CLAUDE_GOAL_PROCESS_SERVER_SAFE_MODE",
		"SMOKE_CLAUDE_GOAL_PROCESS_SERVER_CONTROL_MODE",
		"--safe-mode",
		"ORQUESTA_CODEX_GOAL_BACKEND=claude_process",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DISABLED=true",
		"ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false",
		"ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED=false",
		"/api/v0/apps/director",
		"/api/v0/apps/director/goal/observe",
		"/api/v0/runs/control",
		`"forced": true`,
		"goal_control_signal_confirmed",
		"evidence-ref-claude-goal-process-stop-completed",
		"cleanup_goal_backends",
		"stop_claude_processes_from_runtime_manifest",
		"claude_goal_process_state_",
		"claude_goal_wrapper_",
		"smoke_goal_first_claude_process_server_real=ok",
		"smoke_goal_first_claude_process_forced_stop_server_real=ok",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke Claude process server real incompleto: falta %q", want)
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

func TestSmokeGoalFirstAppServerRealShutdownReadyUsaSenalCooperativaSiServidorSigueVivoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")

	for _, want := range []string{
		"shutdown_ready=true pero proceso servidor sigue vivo; enviando senal cooperativa local",
		`smoke_shutdown_orquesta_server "$server_pid" "" 0 25 "$runtime_dir"`,
		"el servidor siguio vivo tras shutdown_ready=true y senal cooperativa",
		`server_pid=""`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("smoke debe cerrar proceso temporal con senal cooperativa tras shutdown_ready: falta %q", want)
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

func TestSmokeOPESExternalWorkAgentRealUsaShutdownDelegadoConRuntimeDirV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	wrapper := readOperationalDocGuardV0(t, root, "scripts/smoke_opes_external_work_agent_real.sh")
	ops := readOperationalDocGuardV0(t, root, "scripts/lib/opes_agent_smoke_ops.sh")

	for _, want := range []string{
		"source \"$repo_root/scripts/lib/opes_agent_smoke_ops.sh\"",
		`RUNTIME_DIR="$PROJECT_DIR/.orquesta-runtime"`,
		"trap smoke_cleanup EXIT",
		"smoke_start_orquesta_server",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("smoke OPES external-work debe delegar arranque/cleanup gobernado: falta %q", want)
		}
	}
	for _, want := range []string{
		`smoke_shutdown_orquesta_server "$server_pid" "$base_url" 5 30 "$RUNTIME_DIR"`,
		`export ORQUESTA_CODEX_RUNTIME_WORKDIR="$RUNTIME_DIR"`,
		`export ORQUESTA_OPES_BASE_URL`,
		`export ORQUESTA_SERVER_ADDR="127.0.0.1:0"`,
		`server_pid="$!"`,
	} {
		if !strings.Contains(ops, want) {
			t.Fatalf("helper OPES external-work debe pasar runtime_dir al shutdown comun: falta %q", want)
		}
	}
	if strings.Contains(wrapper, "\nOPES_BASE_URL=") || strings.Contains(wrapper, `"$OPES_BASE_URL`) ||
		strings.Contains(ops, `"$OPES_BASE_URL`) {
		t.Fatalf("smoke OPES external-work debe usar ORQUESTA_OPES_BASE_URL como nombre canonico")
	}
}

func TestSmokeOPESReviewsProvidersRealExigeOptInLegacyYWorkdirOPESV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_opes_reviews_providers_real.sh")

	for _, want := range []string{
		"ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP",
		"ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP",
		`"director_execution_mode": "legacy_director_loop"`,
		"Sunset legacy_director_loop",
		"export ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true",
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

func TestScriptStartsTemporaryOrquestaServerV0DetectaPIDConAddrGestionadoV0(t *testing.T) {
	text := strings.Join([]string{
		`ORQUESTA_SERVER_ADDR="127.0.0.1:0" \`,
		`ORQUESTA_SERVER_STATE_DIR="$state_dir" \`,
		`orquesta-server run >"$server_log" 2>&1 &`,
		`server_pid="$!"`,
	}, "\n")

	if !scriptStartsTemporaryOrquestaServerV0(text) {
		t.Fatalf("detector debe cubrir servidor temporal con ORQUESTA_SERVER_ADDR y server_pid=$!")
	}
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

func TestScriptsConShutdownDirectoPidenContratoShutdownV0(t *testing.T) {
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
		for _, command := range scriptShutdownCurlCommandsV0(text) {
			for _, required := range []string{
				"cleanup_goal_backends",
				"idempotency_key",
			} {
				if !strings.Contains(command, required) {
					t.Fatalf("%s invoca shutdown HTTP directo sin %s:\n%s", rel, required, command)
				}
			}
			if !strings.Contains(command, "requested_by") || !strings.Contains(command, "orquesta-director") {
				t.Fatalf("%s invoca shutdown HTTP directo sin autoridad del Director:\n%s", rel, command)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func TestSmokeGoalFirstAppServerRealEsperaTrasKillKillSocketV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "scripts/smoke_goal_first_app_server_real.sh")
	want := strings.Join([]string{
		`for pid in $(app_server_process_pids_for_socket "$socket_path" || true); do`,
		`kill -KILL "$pid"`,
		`for _ in $(seq 1 40); do`,
		`app_server_process_count_for_socket "$socket_path"`,
		`return 1`,
	}, "\n")
	position := 0
	for _, needle := range strings.Split(want, "\n") {
		next := strings.Index(text[position:], needle)
		if next < 0 {
			t.Fatalf("smoke debe esperar procesos tras SIGKILL; falta %q", needle)
		}
		position += next + len(needle)
	}
}

func scriptShutdownCurlCommandsV0(text string) []string {
	lines := strings.Split(text, "\n")
	var commands []string
	for index, line := range lines {
		if !strings.Contains(line, "curl") {
			continue
		}
		commandLines := []string{line}
		if shellLineContinuesV0(line) {
			for _, candidate := range lines[index+1:] {
				commandLines = append(commandLines, candidate)
				if !shellLineContinuesV0(candidate) {
					break
				}
			}
		}
		command := strings.Join(commandLines, "\n")
		if strings.Contains(command, "/api/v0/server/shutdown") &&
			scriptCurlCommandUsesPostV0(command) {
			commands = append(commands, command)
		}
	}
	return commands
}

func scriptCurlCommandUsesPostV0(command string) bool {
	fields := scriptShellCommandFieldsV0(command)
	for index, field := range fields {
		if field == "-X" && index+1 < len(fields) && strings.EqualFold(fields[index+1], "POST") {
			return true
		}
		if strings.HasPrefix(field, "-X") && strings.EqualFold(strings.TrimPrefix(field, "-X"), "POST") {
			return true
		}
		if field == "--request" && index+1 < len(fields) && strings.EqualFold(fields[index+1], "POST") {
			return true
		}
		if strings.HasPrefix(field, "--request=") && strings.EqualFold(strings.TrimPrefix(field, "--request="), "POST") {
			return true
		}
	}
	return false
}

func scriptShellCommandFieldsV0(command string) []string {
	rawFields := strings.Fields(command)
	fields := make([]string, 0, len(rawFields))
	for _, field := range rawFields {
		if field == `\` {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

func shellLineContinuesV0(line string) bool {
	return strings.HasSuffix(strings.TrimSpace(line), `\`)
}

func TestScriptShutdownCurlCommandsV0DetectaCleanupLejanoV0(t *testing.T) {
	text := strings.Join([]string{
		`shutdown_status="$(curl -sS -m "$request_timeout" \`,
		`  -o "$shutdown_response" \`,
		`  -w "%{http_code}" \`,
		`  -X POST \`,
		`  "$base_url/api/v0/server/shutdown" \`,
		`  -H "Content-Type: application/json" \`,
		`  -H "Accept: application/json" \`,
		`  -H "X-Correlation-ID: $request_id-shutdown" \`,
		`  --data-binary "{\"request_id\":\"$request_id-shutdown\",\"idempotency_key\":\"idem-shutdown\",\"requested_by\":\"orquesta-director\",\"cleanup_goal_backends\":true}"`,
		`)"`,
	}, "\n")

	commands := scriptShutdownCurlCommandsV0(text)
	if len(commands) != 1 || !strings.Contains(commands[0], "cleanup_goal_backends") {
		t.Fatalf("detector debe cubrir shutdown multilinea completo: %+v", commands)
	}
}

func TestScriptShutdownCurlCommandsV0DetectaPostSeparadoPorContinuacionV0(t *testing.T) {
	text := strings.Join([]string{
		`shutdown_status="$(curl -sS \`,
		`  -X \`,
		`  POST \`,
		`  "$base_url/api/v0/server/shutdown" \`,
		`  --data-binary "{\"request_id\":\"req-shutdown\",\"idempotency_key\":\"idem-shutdown\",\"requested_by\":\"orquesta-director\",\"cleanup_goal_backends\":true}"`,
		`)"`,
	}, "\n")

	commands := scriptShutdownCurlCommandsV0(text)
	if len(commands) != 1 || !strings.Contains(commands[0], "cleanup_goal_backends") {
		t.Fatalf("detector debe cubrir -X y POST separados por continuacion: %+v", commands)
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

func TestHandoffCierreSesionNoReabreBUG077V0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	text := readOperationalDocGuardV0(t, root, "docs/handoff_cierre_sesion_orquesta_2026-07-02.md")
	openSection := handoffCierreSesionOpenBugsSectionV0(text)

	if strings.Contains(openSection, "BUG-ORQ-20260701-077") {
		t.Fatalf("handoff cierre sesion no debe reabrir BUG-077 sin evidencia nueva")
	}
	for _, line := range strings.Split(openSection, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") &&
			(strings.Contains(trimmed, "BUG-077") || strings.Contains(trimmed, "BUG-ORQ-20260701-077")) {
			t.Fatalf("handoff cierre sesion no debe listar BUG-077 como abierto: %s", trimmed)
		}
	}
	for _, want := range []string{
		"BUG-077 queda cerrado",
		"no usar `777e027c` como HEAD vigente",
		"git fetch origin trabajo/plataforma-agentes && git rebase origin/trabajo/plataforma-agentes",
		"git status --short --branch",
		"HEAD remoto/origin observado en el corte original",
		"estado vigente sin fetch/rebase",
		"evidencia historica del corte original",
		"nueva prueba focal",
		"TestScriptStartsTemporaryOrquestaServerV0DetectaPIDConAddrGestionadoV0",
		"TestSmokeOPESExternalWorkAgentRealUsaShutdownDelegadoConRuntimeDirV0",
		`ORQUESTA_SERVER_ADDR`,
		`server_pid="$!"`,
		"trap smoke_cleanup EXIT",
		"scripts/lib/opes_agent_smoke_ops.sh",
		"RUNTIME_DIR",
		"smoke_shutdown_orquesta_server",
		"orquesta-server stop",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("handoff cierre sesion debe conservar cierre/guarda BUG-077: falta %q", want)
		}
	}
	for _, forbidden := range []string{
		"go run ./cmd/orquesta-server run",
		"127.0.0.1:8787",
		"localhost:8787",
		"HEAD remoto/origin: `777e027c`",
		"HEAD remoto/origin vigente",
		"El arbol local queda limpio en `777e027c`.",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("handoff cierre sesion reabre ruta manual BUG-077: contiene %q", forbidden)
		}
	}
}

func handoffCierreSesionOpenBugsSectionV0(text string) string {
	start := strings.Index(text, "Abiertos")
	if start < 0 {
		return ""
	}
	rest := text[start:]
	end := strings.Index(rest, "\n## Agente remoto")
	if end < 0 {
		return rest
	}
	return rest[:end]
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

	for _, forbidden := range []string{
		`os.environ.get("ORQUESTA_BASE_URL", "http://127.0.0.1:8787")`,
		"http://127.0.0.1:8787",
		"http://localhost:8787",
		"127.0.0.1:8787",
		"localhost:8787",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("launcher OPES A1 no debe caer al puerto historico 8787 por defecto: contiene %q", forbidden)
		}
	}
	for _, want := range []string{
		"ORQUESTA_SERVER_URL",
		"ORQUESTA_BASE_URL",
		"ORQUESTA_RUNTIME_DIR",
		"base_url.txt",
		"--orquesta-base-url required",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("launcher OPES A1 debe resolver endpoint gestionado: falta %q", want)
		}
	}
	ordered := []string{
		`for name in ("ORQUESTA_SERVER_URL", "ORQUESTA_BASE_URL"):`,
		`runtime_dir = os.environ.get("ORQUESTA_RUNTIME_DIR", "").strip()`,
		`return (Path(runtime_dir) / "base_url.txt").read_text(encoding="utf-8").strip()`,
		`return ""`,
	}
	lastIndex := -1
	for _, want := range ordered {
		index := strings.Index(text, want)
		if index < 0 {
			t.Fatalf("launcher OPES A1 debe conservar precedencia endpoint gestionado: falta %q", want)
		}
		if index <= lastIndex {
			t.Fatalf("launcher OPES A1 debe resolver endpoint gestionado en orden; %q aparece fuera de orden", want)
		}
		lastIndex = index
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

func TestSmokeCommonEndpointGestionadoSinPuertoHistoricoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	common := readOperationalDocGuardV0(t, root, "scripts/lib/smoke_common.sh")

	if strings.Contains(common, "127.0.0.1:8787") ||
		strings.Contains(common, "localhost:8787") {
		t.Fatalf("smoke_common no debe caer al puerto historico 8787")
	}
	ordered := []string{
		`if [[ -n "${ORQUESTA_SERVER_URL:-}" ]]; then`,
		`if [[ -n "${ORQUESTA_BASE_URL:-}" ]]; then`,
		`if [[ -n "${ORQUESTA_RUNTIME_DIR:-}" && -s "${ORQUESTA_RUNTIME_DIR%/}/base_url.txt" ]]; then`,
		`return 1`,
	}
	lastIndex := -1
	for _, want := range ordered {
		index := strings.Index(common, want)
		if index < 0 {
			t.Fatalf("smoke_common debe resolver endpoint gestionado: falta %q", want)
		}
		if index <= lastIndex {
			t.Fatalf("smoke_common debe preservar precedencia endpoint gestionado; %q aparece fuera de orden", want)
		}
		lastIndex = index
	}
	for _, want := range []string{
		`printf '%s\n' "${ORQUESTA_SERVER_URL%/}"`,
		`printf '%s\n' "${ORQUESTA_BASE_URL%/}"`,
		`base_url="$(head -n 1 "${ORQUESTA_RUNTIME_DIR%/}/base_url.txt" | tr -d '[:space:]')"`,
		"smoke Orquesta bloqueado: define $label o ORQUESTA_RUNTIME_DIR con base_url.txt",
	} {
		if !strings.Contains(common, want) {
			t.Fatalf("smoke_common debe conservar resolvedor gestionado sin fallback historico: falta %q", want)
		}
	}
}

func TestScriptsNoAsumenPuertoOrquestaHistoricoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	scriptsDir := filepath.Join(root, "scripts")
	forbidden := []string{
		"http://127.0.0.1:8787",
		"http://localhost:8787",
		"127.0.0.1:8787",
		"localhost:8787",
	}
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
		for _, needle := range forbidden {
			if strings.Contains(text, needle) {
				t.Fatalf("%s no debe asumir puerto historico Orquesta: %q", rel, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func TestScriptsConEndpointGestionadoCarganSmokeCommonV0(t *testing.T) {
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
		if rel == "scripts/lib/smoke_common.sh" {
			return nil
		}
		text := readOperationalDocGuardV0(t, root, rel)
		if !strings.Contains(text, "smoke_require_orquesta_base_url") &&
			!strings.Contains(text, "smoke_orquesta_base_url_from_env_or_runtime") {
			return nil
		}
		if !strings.Contains(text, "smoke_common.sh") {
			t.Fatalf("%s usa endpoint gestionado sin cargar scripts/lib/smoke_common.sh", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
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
				`export ORQUESTA_SERVER_URL="$ORQUESTA_BASE_URL_EFFECTIVE"`,
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s debe aceptar endpoint Orquesta gestionado: falta %q", rel, want)
				}
			}
			if strings.Contains(text, "falta ORQUESTA_BASE_URL explicito") {
				t.Fatalf("%s no debe exigir solo ORQUESTA_BASE_URL", rel)
			}
			if strings.Contains(text, `export ORQUESTA_BASE_URL="$ORQUESTA_BASE_URL_EFFECTIVE"`) {
				t.Fatalf("%s no debe exportar el alias ORQUESTA_BASE_URL al servidor", rel)
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
			// docs/historico/ es archivo historico por definicion (T272):
			// su INDICE declara el contexto; no exige clasificacion por fila.
			if strings.HasPrefix(filepath.ToSlash(rel), "docs/historico/") {
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

func TestOrquestaServerScriptsNoCopianBinarioNiArrancanFueraDeCtlODeployV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	scriptsDir := filepath.Join(root, "scripts")
	allowed := map[string]bool{
		"scripts/orquesta_server_ctl.sh":    true,
		"scripts/orquesta_server_deploy.sh": true,
	}
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
		rel = filepath.ToSlash(rel)
		if allowed[rel] {
			return nil
		}
		text := readOperationalDocGuardV0(t, root, rel)
		if scriptCopiesManagedOrquestaServerBinaryV0(text) {
			t.Fatalf("%s copia binario gestionado fuera de ctl/deploy", rel)
		}
		if scriptStartsManagedOrquestaServerOutsideCtlV0(text) {
			t.Fatalf("%s arranca servidor gestionado fuera de ctl/deploy", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
}

func scriptCopiesManagedOrquestaServerBinaryV0(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.Contains(trimmed, "orquesta-server") {
			continue
		}
		if (strings.HasPrefix(trimmed, "cp ") || strings.Contains(trimmed, " cp ")) &&
			(strings.Contains(trimmed, "/srv/orquesta-self/runtime") || strings.Contains(trimmed, "ORQUESTA_CTL_BINARY")) {
			return true
		}
	}
	return false
}

func scriptStartsManagedOrquestaServerOutsideCtlV0(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, "orquesta_server_ctl.sh") || strings.Contains(trimmed, "$DEPLOY_CTL") {
			continue
		}
		if strings.Contains(trimmed, "nohup") && strings.Contains(trimmed, "orquesta-server") && strings.Contains(trimmed, " run") {
			return true
		}
	}
	return false
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
		`-f "$HOME/.codex/auth.json"`,
		`-f "$HOME/.codex/config.toml"`,
		`export ORQUESTA_CODEX_CODE_HOME="$HOME/.codex"`,
		"codex_app_server_auth_source=CODEX_HOME",
		"codex_app_server_auth_source=ORQUESTA_CODEX_CODE_HOME",
		"codex_app_server_auth_source=default_codex_home",
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

func TestSmokesNoExportanAliasIdleSelfImprovementAfterV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"scripts/smoke_goal_first_app_server_real.sh",
		"scripts/smoke_goal_first_claude_process_server_real.sh",
		"scripts/smoke_autoprogramming_bolsa_real.sh",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		if strings.Contains(text, "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER=") {
			t.Fatalf("%s no debe exportar el alias idle sin unidad", rel)
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
		`orquesta_use_isolated_test_env "$smoke_root/test-env"`,
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
