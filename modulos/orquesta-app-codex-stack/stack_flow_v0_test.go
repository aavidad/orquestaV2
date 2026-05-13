package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestBuildStackV0RequiresOptInAndExplicitPorts(t *testing.T) {
	if _, err := BuildStackV0(ConfigV0{}); err == nil {
		t.Fatalf("esperaba opt-in requerido")
	}
}

func TestCodexStackV0GatewayAPIYWebArrancanEquipoDirectorConRuntimeInyectado(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	if stack.Ports.ReviewGateSource == nil {
		t.Fatalf("review gate source no conectado")
	}

	director := postDirectorAPIV0(t, stack)
	if len(director.StartedAgents) != 4 || runtime.launchCountV0() != 4 {
		t.Fatalf("director=%+v launches=%d", director, runtime.launchCountV0())
	}
	assertStatsForRunV0(t, stack.Handler, director.RunRef, 4)
	assertPhaseArtifactEventsForRunV0(t, stack, director.RunRef, 4)

	form := codexStackFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Director arrancado") {
		t.Fatalf("web status=%d body=%s", rec.Code, rec.Body.String())
	}
	if runtime.launchCountV0() != 8 {
		t.Fatalf("launches tras web=%d", runtime.launchCountV0())
	}
}

func TestBuildStackV0CableaDomainWorkOptIn(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	input := orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-codex-stack-domain-001",
		CorrelationID: "corr-codex-stack-domain-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-codex-stack-domain-001",
			CorrelationID:  "corr-codex-stack-domain-001",
			IdempotencyKey: "idem-codex-stack-domain-001",
			RequestedBy:    "orquesta",
			DomainRef:      "opes",
			WorkKind:       "draft_content_block",
			Objective:      "crear bloque documental",
		},
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/domain-work", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if domainWork.called != 1 || domainWork.input.JobRequest.WorkKind != "draft_content_block" {
		t.Fatalf("domain work no delegado=%+v", domainWork)
	}
}

type fakeCodexStackDomainWorkExecutorV0 struct {
	called int
	input  orquestamcp.MCPDomainWorkToolInputV0
}

func (executor *fakeCodexStackDomainWorkExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPDomainWorkToolInputV0,
) (orquestamcp.MCPDomainWorkToolResultV0, error) {
	executor.called++
	executor.input = input
	return orquestamcp.MCPDomainWorkToolResultV0{
		Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        input.Action,
		Job: &orquestadomainwork.DomainWorkJobV0{
			SchemaVersion: orquestadomainwork.DomainWorkJobSchemaV0,
			Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:        "job-ref-codex-stack-domain-001",
			DomainRef:     input.JobRequest.DomainRef,
			WorkKind:      input.JobRequest.WorkKind,
		},
	}, nil
}
