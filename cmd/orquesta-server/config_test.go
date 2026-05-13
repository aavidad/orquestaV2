package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
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
