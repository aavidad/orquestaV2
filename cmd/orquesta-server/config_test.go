package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerConfigFromEnvV0UsaPresupuestoDeComandosParaFronteraParalela(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	got := config.SupervisorCommand.DrainLimits.MaxCommands
	if got != orquestadirectorrunner.DirectorCycleMaxCommandsV0 {
		t.Fatalf("max_commands=%d want %d", got, orquestadirectorrunner.DirectorCycleMaxCommandsV0)
	}
}

func TestServerConfigFromEnvV0SupervisorResidenteNoEsperaAgenteLargo(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "")
	t.Setenv("ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != defaultServerSupervisorMaxExternalWaitsV0 {
		t.Fatalf("server drain max_external_waits=%d want=%d", config.SupervisorCommand.DrainLimits.MaxExternalWaits, defaultServerSupervisorMaxExternalWaitsV0)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != 1 {
		t.Fatalf("el supervisor residente debe observar y volver al bucle, no esperar agentes largos: %d", config.SupervisorCommand.DrainLimits.MaxExternalWaits)
	}
	limits := directorLimitsV0()
	if limits.MaxExternalWaits != 120 {
		t.Fatalf("director max_external_waits=%d want=120", limits.MaxExternalWaits)
	}
}

func TestServerConfigFromEnvV0PermiteEsperaResidenteAmpliaConfiguradaV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "70")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != 70 {
		t.Fatalf("max_external_waits=%d want 70", config.SupervisorCommand.DrainLimits.MaxExternalWaits)
	}
}

func TestServerConfigFromEnvV0BloqueaEsperaResidenteDescontroladaV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "900")

	_, err := serverConfigFromEnvV0()
	if err == nil || !strings.Contains(err.Error(), "ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS incompatible") {
		t.Fatalf("serverConfigFromEnvV0 err=%v", err)
	}
}

func TestServerConfigFromEnvV0ExponeSupervisorDesatendidoV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_SUPERVISOR_MAX_TICKS", "6")
	t.Setenv("ORQUESTA_SERVER_ALLOW_REPEATED_RUNS", "true")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxTicks != 6 {
		t.Fatalf("max_ticks=%d want=6", config.SupervisorCommand.MaxTicks)
	}
	if !config.SupervisorCommand.AllowRepeatedRuns {
		t.Fatalf("allow_repeated_runs=%v want=true", config.SupervisorCommand.AllowRepeatedRuns)
	}
	if config.IdleSelfImprovementPriorityScore != orquestaserver.DefaultIdleSelfImprovementPriorityScoreV0 {
		t.Fatalf("idle priority=%d", config.IdleSelfImprovementPriorityScore)
	}
	if config.IdleSelfImprovementMaxRequests != orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0 {
		t.Fatalf("idle max_requests=%d", config.IdleSelfImprovementMaxRequests)
	}
	if config.IdleSelfImprovementTargetQueue != orquestaserver.DefaultIdleSelfImprovementTargetQueueV0 {
		t.Fatalf("idle target_queue=%d", config.IdleSelfImprovementTargetQueue)
	}
}

func TestServerConfigFromEnvV0PublicaConfiguracionEfectivaCanonica(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", "10")
	t.Setenv("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", "10")
	t.Setenv("ORQUESTA_SERVER_QUEUE_LIMIT", "10")
	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE", "25")
	t.Setenv("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS", "10")
	t.Setenv("ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED", "true")
	t.Setenv("ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS", "7")
	t.Setenv("ORQUESTA_SERVER_SELF_WATCHDOG_CPU_HIGH_PERCENT", "88")
	t.Setenv("ORQUESTA_SERVER_SELF_WATCHDOG_SUSTAINED_SECONDS", "180")
	t.Setenv("ORQUESTA_SERVER_SELF_WATCHDOG_NO_PROGRESS_SECONDS", "90")
	t.Setenv("ORQUESTA_CODEX_MAX_BATCH_READY", "10")
	t.Setenv("ORQUESTA_CODEX_MAX_CONCURRENCY", "10")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "high")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", "6")
	t.Setenv("ORQUESTA_HERMES_ENABLED", "1")
	t.Setenv("ORQUESTA_HERMES_BASE_URL", "https://hermes.local/mcp")
	t.Setenv("ORQUESTA_HERMES_API_KEY", "secret-hermes-test")
	t.Setenv("ORQUESTA_HERMES_STATUS_TOOL", "hermes.status")
	t.Setenv("ORQUESTA_HERMES_STATUS_CONNECTOR_REF", "hermes-status-ref")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_SERVER_MAX_RUNS_PER_TICK":                  "10",
		"ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK":            "10",
		"ORQUESTA_SERVER_QUEUE_LIMIT":                        "10",
		"ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES":               "10",
		"ORQUESTA_SERVER_DRAIN_MAX_OUTBOX":                   "10",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE": "25",
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS": "10",
		"ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED":          "true",
		"ORQUESTA_SERVER_RESIDENT_DIRECTOR_MAX_ACTIONS":      "7",
		"ORQUESTA_SERVER_SELF_WATCHDOG_DISABLED":             "false",
		"ORQUESTA_SERVER_SELF_WATCHDOG_CPU_HIGH_PERCENT":     "88",
		"ORQUESTA_SERVER_SELF_WATCHDOG_SUSTAINED_SECONDS":    "180",
		"ORQUESTA_SERVER_SELF_WATCHDOG_NO_PROGRESS_SECONDS":  "90",
		"ORQUESTA_CODEX_MAX_BATCH_READY":                     "10",
		"ORQUESTA_CODEX_MAX_CONCURRENCY":                     "10",
		"ORQUESTA_CODEX_REASONING_EFFORT":                    "high",
		"ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT":    "6",
		"ORQUESTA_HERMES_ENABLED":                            "true",
		"ORQUESTA_HERMES_BASE_URL":                           "hermes-base-url-configured",
		"ORQUESTA_HERMES_API_KEY":                            "hermes-api-key-configured",
		"ORQUESTA_HERMES_STATUS_TOOL":                        "hermes.status",
		"ORQUESTA_HERMES_STATUS_CONNECTOR_REF":               "hermes-status-ref",
	} {
		got := effectiveSettingValueForTestV0(settings, key)
		if got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
	for _, key := range []string{"ORQUESTA_HERMES_BASE_URL", "ORQUESTA_HERMES_API_KEY"} {
		setting := effectiveSettingForTestV0(settings, key)
		if !setting.Sensitive {
			t.Fatalf("%s debe quedar marcado como sensible: %+v", key, setting)
		}
	}
}

func TestServerConfigFromEnvV0UsaCapacidadCanonicaAmpliaSetentaPadresSeisHijos(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", "")
	t.Setenv("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", "")
	t.Setenv("ORQUESTA_SERVER_QUEUE_LIMIT", "")
	t.Setenv("ORQUESTA_CODEX_MAX_BATCH_READY", "")
	t.Setenv("ORQUESTA_CODEX_MAX_CONCURRENCY", "")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS", "")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", "")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxRunsPerTick != 70 ||
		config.SupervisorCommand.MaxExecutions != 70 {
		t.Fatalf("supervisor capacity=%d/%d want 70/70", config.SupervisorCommand.MaxRunsPerTick, config.SupervisorCommand.MaxExecutions)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_CODEX_MAX_BATCH_READY":                  "70",
		"ORQUESTA_CODEX_MAX_CONCURRENCY":                  "70",
		"ORQUESTA_SERVER_QUEUE_LIMIT":                     "70",
		"ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS":             "70",
		"ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT": "6",
		"ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET":  "70",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
}

func TestServerConfigFromEnvV0ModoSerialCapaAgentesCodexV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_CODEX_EXECUTION_MODE", "serial")
	t.Setenv("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", "20")
	t.Setenv("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", "20")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_DISPATCHES", "20")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_COMMANDS", "20")
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_OUTBOX", "20")
	t.Setenv("ORQUESTA_CODEX_MAX_BATCH_READY", "20")
	t.Setenv("ORQUESTA_CODEX_MAX_CONCURRENCY", "20")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS", "20")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.MaxRunsPerTick != 1 ||
		config.SupervisorCommand.MaxExecutions != 1 ||
		config.SupervisorCommand.DrainLimits.MaxDispatchesPerWait != 1 ||
		config.SupervisorCommand.DrainLimits.MaxCommands != 1 ||
		config.SupervisorCommand.DrainLimits.MaxOutboxPerCycle != 1 {
		t.Fatalf("serial supervisor=%+v", config.SupervisorCommand)
	}
	runtimeConfig := codexRuntimeConfigV0(config, nil)
	if runtimeConfig.MaxBatchReady != 1 || runtimeConfig.MaxConcurrency != 1 {
		t.Fatalf("serial codex limits batch=%d concurrency=%d", runtimeConfig.MaxBatchReady, runtimeConfig.MaxConcurrency)
	}
	settings := config.EffectiveConfig.Settings
	for key, want := range map[string]string{
		"ORQUESTA_CODEX_EXECUTION_MODE":       "serial",
		"ORQUESTA_CODEX_MAX_BATCH_READY":      "1",
		"ORQUESTA_CODEX_MAX_CONCURRENCY":      "1",
		"ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS": "1",
	} {
		if got := effectiveSettingValueForTestV0(settings, key); got != want {
			t.Fatalf("%s=%q want %q settings=%+v", key, got, want, settings)
		}
	}
}

func TestServerStackIdleSelfImprovementFiltraSeccionesNoTxxV0(t *testing.T) {
	supervisor := serverStackSupervisorV0{}
	result, err := supervisor.FilterIdleSelfImprovementRequestsV0(context.Background(), orquestaserver.IdleSelfImprovementRequestFilterRequestV0{
		Requests: []orquestaserver.IdleSelfImprovementRequestV0{{
			RequestRef: "request-ref-autoprogramming-backlog-tareas-futuras-tras-estabilizar-la-automejora-0cdbd385",
		}, {
			RequestRef:    "request-ref-autoprogramming-backlog-t33-alias-reparado-a1b2c3d4",
			SuggestedArea: "t33-alias-reparado",
			ContextRefs:   []string{"backlog_section:tareas-futuras-tras-estabilizar-la-automejora"},
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-t33-autoprogramming-backlog-ack-correlation-a1b2c3d4",
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-scanner-abc123",
		}},
	})
	if err != nil {
		t.Fatalf("FilterIdleSelfImprovementRequestsV0: %v", err)
	}
	if len(result.Requests) != 2 ||
		result.Requests[0].RequestRef != "request-ref-autoprogramming-backlog-t33-autoprogramming-backlog-ack-correlation-a1b2c3d4" ||
		result.Requests[1].RequestRef != "request-ref-autoprogramming-backlog-scanner-abc123" ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-idle-self-improvement-backlog-non-task-section-filtered") {
		t.Fatalf("result=%+v", result)
	}
}

func TestServerStackIdleSelfImprovementPermiteAliasesFederadosV0(t *testing.T) {
	supervisor := serverStackSupervisorV0{}
	result, err := supervisor.FilterIdleSelfImprovementRequestsV0(context.Background(), orquestaserver.IdleSelfImprovementRequestFilterRequestV0{
		Requests: []orquestaserver.IdleSelfImprovementRequestV0{{
			RequestRef:  "request-ref-autoprogramming-backlog-apg-001-doc-loader",
			ContextRefs: []string{"backlog_local_alias:APG-001"},
		}, {
			RequestRef:  "request-ref-autoprogramming-backlog-srv-task-006-capacidad-libre",
			ContextRefs: []string{"backlog_local_alias:SRV-TASK-006"},
		}, {
			RequestRef:  "request-ref-autoprogramming-backlog-mcp-012-roadmap",
			ContextRefs: []string{"backlog_local_alias:MCP-012"},
		}, {
			RequestRef: "request-ref-autoprogramming-backlog-tareas-futuras-001",
		}},
	})
	if err != nil {
		t.Fatalf("FilterIdleSelfImprovementRequestsV0: %v", err)
	}
	if len(result.Requests) != 3 ||
		result.Requests[0].RequestRef != "request-ref-autoprogramming-backlog-apg-001-doc-loader" ||
		result.Requests[1].RequestRef != "request-ref-autoprogramming-backlog-srv-task-006-capacidad-libre" ||
		result.Requests[2].RequestRef != "request-ref-autoprogramming-backlog-mcp-012-roadmap" {
		t.Fatalf("result=%+v", result)
	}
}

func TestServerConfigFromEnvV0AislaEstadoYDejaRuntimeEscribiblePorDefecto(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", "")
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if pathIsInsideForTestV0(t, config.StateDir, projectDir) {
		t.Fatalf("state dir dentro del proyecto: %s", config.StateDir)
	}
	if filepath.Dir(filepath.Dir(config.StateDir)) != filepath.Join(filepath.Dir(projectDir), ".orquesta-control") {
		t.Fatalf("state dir inesperado: %s", config.StateDir)
	}
	if !pathIsInsideForTestV0(t, config.RuntimeWorkDir, projectDir) {
		t.Fatalf("runtime dir fuera del proyecto: %s", config.RuntimeWorkDir)
	}
	if filepath.Base(config.RuntimeWorkDir) != ".orquesta-runtime" {
		t.Fatalf("runtime dir inesperado: %s", config.RuntimeWorkDir)
	}
}

func TestCodexDirectorWaveConfigFromEnvV0UsaSubagentesCanonicos(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", "6")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET", "42")
	stderr := strings.Builder{}

	config, err := codexDirectorWaveConfigFromArgsV0([]string{
		"--dry-run",
		"--allow-recursive-delegation",
		"--max-delegation-depth", "1",
		"--wave-ref", "wave-env-subagents",
		"--project-dir", t.TempDir(),
		"--runtime-dir", filepath.Join(t.TempDir(), "runtime"),
		"--command", os.Args[0],
		"--objective", "probar fanout configurable",
		"--write-set", "cmd/orquesta-server",
		"--required-tests", "go test -count=1 ./cmd/orquesta-server",
		"--branch-ref", "branch-env-subagents",
		"--worktree-ref", "worktree-env-subagents",
	}, &stderr)
	if err != nil {
		t.Fatalf("codexDirectorWaveConfigFromArgsV0: %v stderr=%s", err, stderr.String())
	}
	if config.MaxSubagentsPerAgent != 6 || config.RecursiveAgentBudget != 42 {
		t.Fatalf("subagentes=%d budget=%d", config.MaxSubagentsPerAgent, config.RecursiveAgentBudget)
	}
}

func effectiveSettingValueForTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) string {
	return effectiveSettingForTestV0(settings, key).Value
}

func effectiveSettingForTestV0(settings []orquestaserver.ServerConfigSettingV0, key string) orquestaserver.ServerConfigSettingV0 {
	for _, setting := range settings {
		if setting.Key == key {
			return setting
		}
	}
	return orquestaserver.ServerConfigSettingV0{}
}

func pathIsInsideForTestV0(t *testing.T, child string, parent string) bool {
	t.Helper()
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func TestCodexRuntimeConfigV0UsaUmbralesConservadoresPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_STALLED_TICKS", "")
	t.Setenv("ORQUESTA_CODEX_LOOP_TICKS", "")
	t.Setenv("ORQUESTA_CODEX_WAIT_INTERVAL_MS", "")
	t.Setenv("ORQUESTA_CODEX_NO_ACTIVITY_SECONDS", "")
	t.Setenv("ORQUESTA_CODEX_MAX_EXPECTED_SECONDS", "")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.WaitInterval != 2*time.Second {
		t.Fatalf("wait_interval=%s want 2s", config.WaitInterval)
	}
	if config.ProgressPolicy.StalledAfterNoProgressTicks != defaultCodexStalledTicksV0 {
		t.Fatalf(
			"stalled_ticks=%d want %d",
			config.ProgressPolicy.StalledAfterNoProgressTicks,
			defaultCodexStalledTicksV0,
		)
	}
	if config.ProgressPolicy.LoopAfterRepeatedActions != defaultCodexLoopTicksV0 {
		t.Fatalf(
			"loop_ticks=%d want %d",
			config.ProgressPolicy.LoopAfterRepeatedActions,
			defaultCodexLoopTicksV0,
		)
	}
	if config.ProgressBudget.NoActivityLimit != 10*time.Minute {
		t.Fatalf("no_activity=%s want 10m", config.ProgressBudget.NoActivityLimit)
	}
	if config.ProgressBudget.MaxExpected != 20*time.Minute {
		t.Fatalf("max_expected=%s want 20m", config.ProgressBudget.MaxExpected)
	}
	if config.ReasoningEffort != "medium" {
		t.Fatalf("reasoning_effort=%q want medium", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0PermiteSobrescribirUmbralesDeProgreso(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_STALLED_TICKS", "7")
	t.Setenv("ORQUESTA_CODEX_LOOP_TICKS", "11")
	t.Setenv("ORQUESTA_CODEX_WAIT_INTERVAL_MS", "500")
	t.Setenv("ORQUESTA_CODEX_NO_ACTIVITY_SECONDS", "17")
	t.Setenv("ORQUESTA_CODEX_MAX_EXPECTED_SECONDS", "19")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.WaitInterval != 500*time.Millisecond {
		t.Fatalf("wait_interval=%s want 500ms", config.WaitInterval)
	}
	if config.ProgressPolicy.StalledAfterNoProgressTicks != 7 {
		t.Fatalf("stalled_ticks=%d want 7", config.ProgressPolicy.StalledAfterNoProgressTicks)
	}
	if config.ProgressPolicy.LoopAfterRepeatedActions != 11 {
		t.Fatalf("loop_ticks=%d want 11", config.ProgressPolicy.LoopAfterRepeatedActions)
	}
	if config.ProgressBudget.NoActivityLimit != 17*time.Second {
		t.Fatalf("no_activity=%s want 17s", config.ProgressBudget.NoActivityLimit)
	}
	if config.ProgressBudget.MaxExpected != 19*time.Second {
		t.Fatalf("max_expected=%s want 19s", config.ProgressBudget.MaxExpected)
	}
}

func TestCodexRuntimeConfigV0PermiteSobrescribirReasoningEffort(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "high")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.ReasoningEffort != "high" {
		t.Fatalf("reasoning_effort=%q want high", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0ConservaReasoningEffortMedium(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "medium")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.ReasoningEffort != "medium" {
		t.Fatalf("reasoning_effort=%q want medium", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0InyectaToolbeltOperativoDelServidor(t *testing.T) {
	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		Addr:           "127.0.0.1:18787",
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	joined := strings.Join(config.PromptHints, "\n")
	for _, want := range []string{
		"http://127.0.0.1:18787",
		"/api/v0/autoprogramming/supervise",
		"orquesta.operator.operations.v0",
		"artifact_submission",
		"codigo reutilizable",
		"palabras genericas como clave",
		"$caveman full",
		"razonamiento medium",
		"checkpoint/handoff durable",
		"no metas HTTP, MCP, Codex, OPES",
		"centraliza configuracion/env",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("prompt hints no contienen %q: %s", want, joined)
		}
	}
}

func TestCodexStackCapacityConfigFromEnvV0ConservaReasoningCodexMedium(t *testing.T) {
	t.Setenv("ORQUESTA_CAPACITY_REASONING_EFFORT", "")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "medium")

	config := codexStackCapacityConfigFromEnvV0()

	if config.ReasoningEffort != orquestacoreworkflow.OrchestrationCapacityMediumV0 {
		t.Fatalf("reasoning_effort=%q want medium", config.ReasoningEffort)
	}
	if config.Tier != orquestacoreworkflow.OrchestrationCapacityMediumV0 {
		t.Fatalf("tier=%q want medium", config.Tier)
	}
	if config.OccurredAt == "" || config.RequestedBy != "orquesta-server" {
		t.Fatalf("capacity config incompleta: %+v", config)
	}
}

func TestCodexStackCapacityConfigFromEnvV0PermiteSobrescribirCapacidad(t *testing.T) {
	t.Setenv("ORQUESTA_CAPACITY_TIER", "high")
	t.Setenv("ORQUESTA_CAPACITY_REASONING_EFFORT", "low")
	t.Setenv("ORQUESTA_CODEX_REASONING_EFFORT", "medium")

	config := codexStackCapacityConfigFromEnvV0()

	if config.Tier != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("tier=%q want high", config.Tier)
	}
	if config.ReasoningEffort != orquestacoreworkflow.OrchestrationCapacityLowV0 {
		t.Fatalf("reasoning_effort=%q want low", config.ReasoningEffort)
	}
}

func TestCodexRuntimeConfigV0PermiteSandboxAmplioConfigurable(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_SANDBOX", "workspace-write")
	t.Setenv("ORQUESTA_CODEX_APPROVAL_POLICY", "never")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_SANDBOX", "danger-full-access")
	t.Setenv("ORQUESTA_CODEX_DIRECTOR_APPROVAL_POLICY", "on-request")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "workspace-write" ||
		config.ApprovalPolicy != "never" ||
		config.DirectorSandbox != "danger-full-access" ||
		config.DirectorApprovalPolicy != "on-request" {
		t.Fatalf("config=%+v", config)
	}
}

func TestCodexRuntimeConfigV0RespetaSandboxAmplioDeOperador(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_SANDBOX", "danger-full-access")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestCodexRuntimeConfigV0UsaDangerFullAccessPorDefecto(t *testing.T) {
	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestServerConfigFromEnvV0NoMutaStartupCleanupGlobalPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_STARTUP_CLEANUP_MODE", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if got := os.Getenv("ORQUESTA_STARTUP_CLEANUP_MODE"); got != "" {
		t.Fatalf("startup cleanup global mutado=%q", got)
	}
	counts := daemonStartEnvCountsV0(serverDaemonStartEnvPolicyV0(os.Environ(), config))
	if !strings.Contains(counts, "defaulted=") {
		t.Fatalf("daemon policy sin defaults: %s", counts)
	}
}

func TestServerConfigFromEnvV0LecturasRepetidasNoContaminanDefaultsV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_STARTUP_CLEANUP_MODE", "")
	t.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "")

	first, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("first serverConfigFromEnvV0: %v", err)
	}
	second, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("second serverConfigFromEnvV0: %v", err)
	}
	if got := os.Getenv("ORQUESTA_STARTUP_CLEANUP_MODE"); got != "" {
		t.Fatalf("startup cleanup global mutado=%q", got)
	}
	if got := os.Getenv("ORQUESTA_DETAIL_PROHIBITED_RAILS"); got != "" {
		t.Fatalf("detail rails global mutado=%q", got)
	}
	if first.EffectiveConfig.Settings[0].Key != second.EffectiveConfig.Settings[0].Key {
		t.Fatalf("lecturas no estables")
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaOPESOptIn(t *testing.T) {
	var received struct {
		CorrelationID  string `json:"correlation_id"`
		IdempotencyKey string `json:"idempotency_key"`
		RequestedBy    string `json:"requested_by"`
		JobType        string `json:"job_type"`
	}
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "job-ref-opes-env-001",
			"status":          "accepted",
			"correlation_id":  "corr-opes-env-001",
			"idempotency_key": "idem-opes-env-001",
		})
	}))
	defer server.Close()
	t.Setenv("ORQUESTA_OPES_BASE_URL", server.URL)
	t.Setenv("ORQUESTA_OPES_TIMEOUT_SECONDS", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor OPES no configurado")
	}
	result, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-opes-env-001",
		CorrelationID: "corr-opes-env-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-opes-env-001",
			CorrelationID:  "corr-opes-env-001",
			IdempotencyKey: "idem-opes-env-001",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "draft_content_block",
			Objective:      "crear bloque documental",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-ref-opes-env-001" {
		t.Fatalf("result=%+v", result)
	}
	if received.JobType != "draft_content_block" ||
		received.RequestedBy != "orquesta" {
		t.Fatalf("received=%+v", received)
	}
}

func TestDomainWorkExecutorFromEnvV0SinOPESQuedaApagado(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor != nil {
		t.Fatalf("executor debe ser nil sin ORQUESTA_OPES_BASE_URL ni OPES_BASE_URL")
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaOPESBaseURLFallback(t *testing.T) {
	var received struct {
		JobType string `json:"job_type"`
	}
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              "job-ref-opes-fallback-001",
			"status":          "accepted",
			"correlation_id":  "corr-opes-fallback-001",
			"idempotency_key": "idem-opes-fallback-001",
		})
	}))
	defer server.Close()
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", server.URL)
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor OPES fallback no configurado")
	}
	result, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-opes-fallback-001",
		CorrelationID: "corr-opes-fallback-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-opes-fallback-001",
			CorrelationID:  "corr-opes-fallback-001",
			IdempotencyKey: "idem-opes-fallback-001",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "plan_temario",
			Objective:      "crear plan de temario",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-ref-opes-fallback-001" ||
		received.JobType != "plan_temario" {
		t.Fatalf("result=%+v received=%+v", result, received)
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaFileCreatorOptIn(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: stateDir,
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor file no configurado")
	}
	input := orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-file-env-001",
		CorrelationID: "corr-file-env-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-file-env-001",
			CorrelationID:  "corr-file-env-001",
			IdempotencyKey: "idem-file-env-001",
			RequestedBy:    "orquesta",
			DomainRef:      "dominio-demo",
			WorkKind:       "generate_content_package",
			Objective:      "crear paquete generico",
		},
	}
	first, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute first: %v", err)
	}
	second, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute second: %v", err)
	}
	if first.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		first.Job == nil ||
		first.Job.JobRef == "" ||
		second.Job == nil ||
		second.Job.JobRef != first.Job.JobRef {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if !pathExistsForTestV0(filepath.Join(stateDir, "domain-work-jobs", "domain_work_jobs_v0.json")) {
		t.Fatalf("snapshot domain-work file no creado")
	}

	submitResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
	})
	if err != nil {
		t.Fatalf("Execute submit: %v", err)
	}
	if submitResult.Estado != orquestamcp.MCPDomainWorkEstadoErrorV0 ||
		len(submitResult.Errores) != 1 ||
		submitResult.Errores[0].Code != orquestamcp.MCPDomainWorkSubmitterUnavailableV0 {
		t.Fatalf("submitResult=%+v", submitResult)
	}
}

func TestDomainWorkExecutorFromEnvV0ConectaHTTPNeutralOptIn(t *testing.T) {
	var gotJobRequest orquestadomainwork.DomainWorkJobRequestV0
	var gotSubmission orquestadomainwork.DomainWorkArtifactSubmissionV0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs":
			if err := json.NewDecoder(r.Body).Decode(&gotJobRequest); err != nil {
				t.Fatalf("decode job: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"job": map[string]any{
					"schema_version":  orquestadomainwork.DomainWorkJobSchemaV0,
					"status":          orquestadomainwork.DomainWorkStatusAcceptedV0,
					"job_ref":         "job-ref-http-neutral-001",
					"domain_ref":      gotJobRequest.DomainRef,
					"work_kind":       gotJobRequest.WorkKind,
					"correlation_id":  gotJobRequest.CorrelationID,
					"idempotency_key": gotJobRequest.IdempotencyKey,
				},
			})
		case "/artifacts":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmission); err != nil {
				t.Fatalf("decode submission: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"receipt": map[string]any{
					"schema_version":  orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
					"status":          orquestadomainwork.DomainWorkStatusAcceptedV0,
					"job_ref":         gotSubmission.JobRef,
					"artifact_ref":    gotSubmission.ArtifactRef,
					"receipt_ref":     "receipt-ref-http-neutral-001",
					"correlation_id":  gotSubmission.CorrelationID,
					"idempotency_key": gotSubmission.IdempotencyKey,
				},
			})
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_DIR", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", server.URL)
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_EGRESS_MODE", "smoke_local")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_CREATE_PATH", "/jobs")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_SUBMIT_PATH", "/artifacts")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("domainWorkExecutorFromEnvV0: %v", err)
	}
	if executor == nil {
		t.Fatalf("executor HTTP neutral no configurado")
	}
	createResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "request-ref-http-neutral-001",
			CorrelationID:  "corr-http-neutral-001",
			IdempotencyKey: "idem-http-neutral-001",
			RequestedBy:    "test",
			DomainRef:      "domain-ref-http-neutral-001",
			WorkKind:       "compose_external_summary",
			Objective:      "crear job por HTTP neutral",
		},
	})
	if err != nil {
		t.Fatalf("Execute create: %v", err)
	}
	if createResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		createResult.Job == nil ||
		createResult.Job.JobRef != "job-ref-http-neutral-001" {
		t.Fatalf("createResult=%+v", createResult)
	}
	submitResult, err := executor.Execute(context.Background(), orquestamcp.MCPDomainWorkToolInputV0{
		Action: orquestamcp.MCPDomainWorkActionSubmitArtifactV0,
		ArtifactSubmission: orquestadomainwork.DomainWorkArtifactSubmissionV0{
			RequestID:      "request-ref-http-neutral-submit-001",
			CorrelationID:  "corr-http-neutral-001",
			IdempotencyKey: "idem-http-neutral-submit-001",
			RequestedBy:    "test",
			DomainRef:      "domain-ref-http-neutral-001",
			JobRef:         createResult.Job.JobRef,
			ArtifactRef:    "artifact-ref-http-neutral-001",
			ArtifactType:   "work_delivery",
		},
	})
	if err != nil {
		t.Fatalf("Execute submit: %v", err)
	}
	if submitResult.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		submitResult.Receipt == nil ||
		submitResult.Receipt.ReceiptRef != "receipt-ref-http-neutral-001" ||
		gotSubmission.JobRef != "job-ref-http-neutral-001" {
		t.Fatalf("submitResult=%+v got=%+v", submitResult, gotSubmission)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaBackendAmbiguo(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "domain_work_backend_ambiguous" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaFallbackAmbiguo(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "domain_work_backend_ambiguous" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func TestDomainWorkExecutorFromEnvV0RechazaHTTPConOtroBackend(t *testing.T) {
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv("ORQUESTA_DOMAIN_WORK_FILE_ENABLED", "1")
	t.Setenv("ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL", "http://127.0.0.1:1")

	executor, err := domainWorkExecutorFromEnvV0(orquestaserver.ConfigV0{
		StateDir: t.TempDir(),
	})
	if err == nil || err.Error() != "domain_work_backend_ambiguous" || executor != nil {
		t.Fatalf("executor=%v err=%v", executor, err)
	}
}

func pathExistsForTestV0(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
