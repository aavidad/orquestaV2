package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestBootstrapAppSpecCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico(t *testing.T) {
	var received orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != BootstrapAppSpecCliEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(BootstrapAppSpecCliCorrelationHeaderV0); got != "corr-bootstrap-1" {
			t.Fatalf("correlation=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type=%q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("accept=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if received.RequestID != "req-bootstrap-1" ||
			received.CorrelationID != "corr-bootstrap-1" ||
			received.IdempotencyKey != "idem-bootstrap-1" {
			t.Fatalf("request tecnica no normalizada: %+v", received)
		}
		writeBootstrapAppSpecSuccessV0(t, w, received)
	}))
	defer server.Close()

	client := mustNewBootstrapAppSpecCliClientV0(t, server.URL, 1200*time.Millisecond)
	env := client.BootstrapProyectoDesdeAppSpec(context.Background(), CliInvocationContextV0{
		RequestID:      "req-bootstrap-1",
		CorrelationID:  "corr-bootstrap-1",
		IdempotencyKey: "idem-bootstrap-1",
		ServerURL:      server.URL,
		Timeout:        time.Second,
		OutputFormat:   CliOutputFormatJSONV0,
	}, minimalBootstrapAppSpecCommandForCliV0(t))

	if !env.OK || env.RequestID != "req-bootstrap-1" || env.CorrelationID != "corr-bootstrap-1" {
		t.Fatalf("envelope ids/ok inesperado: %+v", env)
	}
	if env.Contract != BootstrapAppSpecCliContractV0 || env.Version != BootstrapAppSpecCliContractVersionV0 {
		t.Fatalf("contrato inesperado: %s %s", env.Contract, env.Version)
	}
	if env.Meta.StatusCode != http.StatusOK || env.Meta.Transporte != CliTransportRESTV0 || env.Meta.Retryable {
		t.Fatalf("meta inesperada: %+v", env.Meta)
	}
	data, ok := env.Data.(orquestadirector.BootstrapProyectoDesdeAppSpecResultV0)
	if !ok {
		t.Fatalf("data type=%T", env.Data)
	}
	if data.ProjectRef != "projectref_0001" ||
		data.AppSpecRef != "appspecref_0001" ||
		data.StartRunCommand.CommandType != orquestacoreworkflow.OrchestrationCommandStartRunV0 ||
		data.WorkflowResult.Events[0].EventType != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		t.Fatalf("data canonica inesperada: %+v", data)
	}
}

func TestBootstrapAppSpecCliClientV0Respuesta400DevuelveErroresPublicos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(orquestadirector.BootstrapProyectoDesdeAppSpecErrorV0{
			Code:    orquestadirector.ErrDirectorBootstrapInvalidoV0,
			Field:   "idempotency_key",
			Message: "idempotency_key requerida",
		})
	}))
	defer server.Close()

	client := mustNewBootstrapAppSpecCliClientV0(t, server.URL, time.Second)
	env := client.BootstrapProyectoDesdeAppSpec(context.Background(), invocationForBootstrapAppSpecCliV0(server.URL), minimalBootstrapAppSpecCommandForCliV0(t))

	if env.OK || env.Data != nil || env.Meta.StatusCode != http.StatusBadRequest || env.Meta.Retryable {
		t.Fatalf("envelope 400 inesperado: %+v", env)
	}
	if len(env.Errores) != 1 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	if err := env.Errores[0]; err.Codigo != orquestadirector.ErrDirectorBootstrapInvalidoV0 ||
		err.Campo != "idempotency_key" ||
		err.MensajeI18N != "orquesta_director.errores."+orquestadirector.ErrDirectorBootstrapInvalidoV0 ||
		err.Detalle != "idempotency_key requerida" {
		t.Fatalf("error publico inesperado: %+v", err)
	}
}

func TestBootstrapAppSpecCliClientV0Status500NoFiltraBodyPrivado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado token=secreto /home/alberto/runtime", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := mustNewBootstrapAppSpecCliClientV0(t, server.URL, time.Second)
	env := client.BootstrapProyectoDesdeAppSpec(context.Background(), invocationForBootstrapAppSpecCliV0(server.URL), minimalBootstrapAppSpecCommandForCliV0(t))

	if env.OK || env.Meta.StatusCode != http.StatusInternalServerError || !env.Meta.Retryable {
		t.Fatalf("envelope 500 inesperado: %+v", env)
	}
	if len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrErrorTransporteV0 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{"stack privado", "token=secreto", "/home/alberto", "runtime"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("envelope filtra detalle privado %q: %s", forbidden, string(raw))
		}
	}
}

func TestBootstrapAppSpecCliClientV0TimeoutDevuelveErrorTransporte(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeBootstrapAppSpecSuccessV0(t, w, minimalBootstrapAppSpecCommandForCliV0(t))
	}))
	defer server.Close()

	client := mustNewBootstrapAppSpecCliClientV0(t, server.URL, 2*time.Millisecond)
	env := client.BootstrapProyectoDesdeAppSpec(context.Background(), invocationForBootstrapAppSpecCliV0(server.URL), minimalBootstrapAppSpecCommandForCliV0(t))

	if env.OK || len(env.Errores) != 1 {
		t.Fatalf("timeout envelope inesperado: %+v", env)
	}
	if env.Errores[0].Codigo != CliErrErrorTransporteV0 || env.Errores[0].Detalle != "timeout" || !env.Meta.Retryable {
		t.Fatalf("timeout error inesperado: errores=%+v meta=%+v", env.Errores, env.Meta)
	}
}

func TestBootstrapAppSpecCliClientV0RespuestaInvalida(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"project_ref": "projectref_0001"})
	}))
	defer server.Close()

	client := mustNewBootstrapAppSpecCliClientV0(t, server.URL, time.Second)
	env := client.BootstrapProyectoDesdeAppSpec(context.Background(), invocationForBootstrapAppSpecCliV0(server.URL), minimalBootstrapAppSpecCommandForCliV0(t))

	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrRespuestaInvalidaV0 {
		t.Fatalf("respuesta invalida no detectada: %+v", env)
	}
	if env.Data != nil || env.Meta.StatusCode != http.StatusOK {
		t.Fatalf("respuesta invalida con data/status inesperado: %+v", env)
	}
}

func TestBootstrapAppSpecCliClientV0RechazaServerURLConCredenciales(t *testing.T) {
	_, err := NewBootstrapAppSpecCliClientV0("https://usuario:secreto@example.test", time.Second)
	if !IsCliClientErrorCodeV0(err, CliErrConfiguracionInvalidaV0) {
		t.Fatalf("error constructor=%v", err)
	}
	var cliErr CliClientErrorV0
	if !errors.As(err, &cliErr) || cliErr.Field != "server_url" || cliErr.Detail != "server_url_no_debe_contener_credenciales" {
		t.Fatalf("error publico inesperado: %+v", cliErr)
	}
	if strings.Contains(cliErr.Detail, "usuario") || strings.Contains(cliErr.Detail, "secreto") {
		t.Fatalf("detalle filtra credenciales: %+v", cliErr)
	}
}

func mustNewBootstrapAppSpecCliClientV0(t *testing.T, serverURL string, timeout time.Duration) *BootstrapAppSpecCliClientV0 {
	t.Helper()
	client, err := NewBootstrapAppSpecCliClientV0(serverURL, timeout)
	if err != nil {
		t.Fatalf("NewBootstrapAppSpecCliClientV0: %v", err)
	}
	return client
}

func invocationForBootstrapAppSpecCliV0(serverURL string) CliInvocationContextV0 {
	return CliInvocationContextV0{
		RequestID:      "req-bootstrap-test",
		CorrelationID:  "corr-bootstrap-test",
		IdempotencyKey: "idem-bootstrap-test",
		ServerURL:      serverURL,
		Timeout:        time.Second,
		OutputFormat:   CliOutputFormatJSONV0,
	}
}

func minimalBootstrapAppSpecCommandForCliV0(t *testing.T) orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0 {
	t.Helper()
	req := minimalAppSpecRequestForCliV0()
	req.RequestID = "req-bootstrap-test"
	req.Source = SolicitarNuevaAppCliSourceV0
	spec, issues := orquestafactory.SolicitarNuevaAppV0(req, fixedSolicitarNuevaAppCliClockV0())
	if len(issues) > 0 {
		t.Fatalf("fixture spec invalida: %+v", issues)
	}
	backlog, issues := orquestafactory.GenerarBacklogInicialPropuestoV0(spec)
	if len(issues) > 0 {
		t.Fatalf("fixture backlog invalido: %+v", issues)
	}
	return orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0{
		AppSpec:        spec,
		Backlog:        backlog,
		RequestedBy:    "cli-test",
		OccurredAt:     "2026-05-04T10:30:00Z",
		IdempotencyKey: "idem-bootstrap-test",
	}
}

func writeBootstrapAppSpecSuccessV0(t *testing.T, w http.ResponseWriter, cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	result := validBootstrapAppSpecResultForCliV0(cmd)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func validBootstrapAppSpecResultForCliV0(cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) orquestadirector.BootstrapProyectoDesdeAppSpecResultV0 {
	startRun := orquestacoreworkflow.OrchestrationCommandV0{
		CommandID:      "cmd-start-run-001",
		CommandType:    orquestacoreworkflow.OrchestrationCommandStartRunV0,
		RunID:          "runref_0001",
		IdempotencyKey: firstNonEmptyBootstrapAppSpecV0(cmd.IdempotencyKey, "idem-bootstrap-test"),
		CorrelationID:  cmd.CorrelationID,
		RequestedBy:    cmd.RequestedBy,
		OccurredAt:     firstNonEmptyBootstrapAppSpecV0(cmd.OccurredAt, "2026-05-04T10:30:00Z"),
		PayloadVersion: orquestacoreworkflow.OrchestrationCommandPayloadVersionV0,
		Payload:        json.RawMessage(`{"project_ref":"projectref_0001","app_spec_ref":"appspecref_0001"}`),
	}
	event := orquestacoreworkflow.OrchestrationEventV0{
		EventID:        "evt-run-started-001",
		EventType:      orquestacoreworkflow.OrchestrationEventRunStartedV0,
		RunID:          "runref_0001",
		Sequence:       1,
		IdempotencyKey: startRun.IdempotencyKey,
		CorrelationID:  cmd.CorrelationID,
		CausationID:    startRun.CommandID,
		OccurredAt:     startRun.OccurredAt,
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        json.RawMessage(`{"project_ref":"projectref_0001","app_spec_ref":"appspecref_0001","requested_by":"cli-test"}`),
	}
	return orquestadirector.BootstrapProyectoDesdeAppSpecResultV0{
		RegistroAceptado: orquestadirector.BootstrapRegistroAceptadoV0{
			RegistroID:       "registro_0001",
			ProjectRef:       "projectref_0001",
			AppSpecRef:       "appspecref_0001",
			Estado:           "borrador",
			BootstrapVersion: "v0",
			RequestID:        cmd.RequestID,
			CorrelationID:    cmd.CorrelationID,
		},
		ProjectRef:      "projectref_0001",
		AppSpecRef:      "appspecref_0001",
		StartRunCommand: startRun,
		WorkflowResult:  orquestacoreworkflow.OrchestrationCommandResultV0{Events: []orquestacoreworkflow.OrchestrationEventV0{event}, Outbox: []orquestacoreworkflow.OutboxMessageV0{}},
	}
}
