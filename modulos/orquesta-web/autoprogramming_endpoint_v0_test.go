package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAutoprogrammingWebEndpointV0RenderizaPantallaOperativa(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/autoprogramming", nil)

	NewAutoprogrammingWebEndpointV0(nil).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, required := range []string{
		"Autoprogramacion",
		"/api/v0/autoprogramming/prepare-run",
		"/api/v0/autoprogramming/status",
		"/api/v0/autoprogramming/goal/observe",
		"/api/v0/autoprogramming/supervise",
		"queue-health-panel",
		"agents_live",
		`href="/ops"`,
		"Codex Goal automatico",
		"goal_migration:goal-first",
		"Acciones requeridas",
		"Acciones seguras",
		"safe-actions-panel",
		"safe_actions",
		"currentSafeActions",
		"hasGoalFirstSafeActionForCurrentRun",
		"goalFirstSafeActions",
		"item.action === 'observe_goal'",
		"item.endpoint === '/api/v0/autoprogramming/goal/observe'",
		"Esta run publica observe_goal; no se usa supervision legacy.",
		"stale_running",
		`name="project_ref"`,
		`name="required_tests"`,
	} {
		if rec.Code != http.StatusOK || !strings.Contains(body, required) {
			t.Fatalf("autoprogramming incompleto status=%d falta=%q body=%s", rec.Code, required, body)
		}
	}
	if strings.Contains(strings.ToLower(body), "local_path") {
		t.Fatalf("pantalla no debe pedir paths locales: %s", body)
	}
}

func TestAutoprogrammingWebEndpointV0MetodoNoSoportado(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/autoprogramming", nil)

	NewAutoprogrammingWebEndpointV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed ||
		rec.Header().Get("Allow") != "GET, OPTIONS" {
		t.Fatalf("status=%d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}

func TestAutoprogrammingWebEndpointV0ConsultaStatusConQueryPublica(t *testing.T) {
	client := &recordingAutoprogrammingStatusClientV0{
		ViewModel: WebAutoprogrammingStatusViewModelV0{
			SchemaVersion: "web_autoprogramming_status.v0",
			Estado:        WebAutoprogrammingPrepareRunEstadoOKV0,
		},
	}
	endpoint := NewAutoprogrammingWebEndpointV0(client)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/autoprogramming?run_ref=run-web-status-001&include_agent_usage=true", nil)

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || client.Query.RunRef != "run-web-status-001" || !client.Query.IncludeAgentUsage {
		t.Fatalf("status=%d query=%+v body=%s", rec.Code, client.Query, rec.Body.String())
	}
	var viewModel WebAutoprogrammingStatusViewModelV0
	if err := json.NewDecoder(rec.Body).Decode(&viewModel); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if viewModel.Estado != WebAutoprogrammingPrepareRunEstadoOKV0 {
		t.Fatalf("view_model=%+v", viewModel)
	}
}

func TestAutoprogrammingWebEndpointV0PreservaSelectorApp(t *testing.T) {
	client := &recordingAutoprogrammingStatusClientV0{ViewModel: WebAutoprogrammingStatusViewModelV0{Estado: WebAutoprogrammingPrepareRunEstadoOKV0}}
	rec := httptest.NewRecorder()
	NewAutoprogrammingWebEndpointV0(client).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/autoprogramming?scope_mode=app&scope=app-web-opaque-001", nil))
	if rec.Code != http.StatusOK || client.Query.ScopeMode != "app" || client.Query.Scope != "app-web-opaque-001" {
		t.Fatalf("status=%d query=%+v", rec.Code, client.Query)
	}
}

func TestAutoprogrammingWebEndpointV0NoPublicaErrorDeCliente(t *testing.T) {
	endpoint := NewAutoprogrammingWebEndpointV0(&recordingAutoprogrammingStatusClientV0{Err: errAutoprogrammingStatusClientV0{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/autoprogramming?run_ref=run-web-status-001", nil)

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), "detalle-privado") ||
		!strings.Contains(rec.Body.String(), WebAutoprogrammingStatusErrTransporteV0) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type recordingAutoprogrammingStatusClientV0 struct {
	Query     WebAutoprogrammingStatusQueryV0
	ViewModel WebAutoprogrammingStatusViewModelV0
	Err       error
}

func (client *recordingAutoprogrammingStatusClientV0) ConsultarAutoprogrammingStatus(
	_ context.Context,
	query WebAutoprogrammingStatusQueryV0,
) (WebAutoprogrammingStatusViewModelV0, error) {
	client.Query = query
	return client.ViewModel, client.Err
}

type errAutoprogrammingStatusClientV0 struct{}

func (errAutoprogrammingStatusClientV0) Error() string { return "detalle-privado" }
