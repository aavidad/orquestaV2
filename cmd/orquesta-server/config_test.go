package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestServerConfigFromEnvV0DaMargenRealALosAgentesPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", "")
	t.Setenv("ORQUESTA_DIRECTOR_MAX_EXTERNAL_WAITS", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if config.SupervisorCommand.DrainLimits.MaxExternalWaits != 120 {
		t.Fatalf("server drain max_external_waits=%d want=120", config.SupervisorCommand.DrainLimits.MaxExternalWaits)
	}
	limits := directorLimitsV0()
	if limits.MaxExternalWaits != 120 {
		t.Fatalf("director max_external_waits=%d want=120", limits.MaxExternalWaits)
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
	if config.ReasoningEffort != "xhigh" {
		t.Fatalf("reasoning_effort=%q want xhigh", config.ReasoningEffort)
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

func TestCodexRuntimeConfigV0PermitePermisosEspecificosDelDirector(t *testing.T) {
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode body: %v", err)
		}
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

func pathExistsForTestV0(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
