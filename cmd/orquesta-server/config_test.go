package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestServerConfigFromEnvV0AislaControlFueraDelProyectoPorDefecto(t *testing.T) {
	projectDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", "")
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	for name, dir := range map[string]string{
		"state":   config.StateDir,
		"runtime": config.RuntimeWorkDir,
	} {
		if pathIsInsideForTestV0(t, dir, projectDir) {
			t.Fatalf("%s dir dentro del proyecto: %s", name, dir)
		}
		if filepath.Dir(filepath.Dir(dir)) != filepath.Join(filepath.Dir(projectDir), ".orquesta-control") {
			t.Fatalf("%s dir inesperado: %s", name, dir)
		}
	}
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

func TestDomainWorkExecutorFromEnvV0ConectaOPESOptIn(t *testing.T) {
	var received struct {
		CorrelationID  string `json:"correlation_id"`
		IdempotencyKey string `json:"idempotency_key"`
		RequestedBy    string `json:"requested_by"`
		JobType        string `json:"job_type"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
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

	executor := domainWorkExecutorFromEnvV0()
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
	if executor := domainWorkExecutorFromEnvV0(); executor != nil {
		t.Fatalf("executor debe ser nil sin ORQUESTA_OPES_BASE_URL")
	}
}
