package orquestaweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestRESTSolicitarNuevaAppClientV0EnviaPOSTJSONCorrelacionYTimeout(t *testing.T) {
	var received orquestafactory.AppSpecRequestV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != SolicitarNuevaAppEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(SolicitarNuevaAppCorrelationV0); got != "req-client-1" {
			t.Fatalf("correlation=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeSuccessV0(t, w)
	}))
	defer server.Close()

	timeout := 1200 * time.Millisecond
	client := NewRESTSolicitarNuevaAppClientV0(server.URL, timeout)

	vm, err := client.SolicitarNuevaApp(context.Background(), minimalFormForClientV0("req-client-1"))
	if err != nil {
		t.Fatalf("SolicitarNuevaApp error: %v", err)
	}
	if client.Timeout != timeout || client.HTTPClient.Timeout != timeout {
		t.Fatalf("timeout no configurado: client=%s http=%s", client.Timeout, client.HTTPClient.Timeout)
	}
	if received.SchemaVersion != orquestafactory.AppSpecRequestSchemaV0 ||
		received.Source != WebNuevaAppSourceV0 ||
		received.RequestID != "req-client-1" ||
		received.Locale != "es" ||
		received.Nombre != "Agenda" ||
		received.Objetivo != "Coordinar ensayos" ||
		received.TipoApp != "web" {
		t.Fatalf("request inesperada: %+v", received)
	}
	if vm.RequestID != "req-client-1" || vm.Estado != WebNuevaAppEstadoValida {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTSolicitarNuevaAppClientV0CreaRequestIDSiFalta(t *testing.T) {
	var received orquestafactory.AppSpecRequestV0
	var correlation string
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlation = r.Header.Get(SolicitarNuevaAppCorrelationV0)
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	form := minimalFormForClientV0("")

	if _, err := client.SolicitarNuevaApp(context.Background(), form); err != nil {
		t.Fatalf("SolicitarNuevaApp error: %v", err)
	}
	if received.RequestID == "" || correlation == "" || correlation != received.RequestID {
		t.Fatalf("request_id/correlation no propagados: request_id=%q correlation=%q", received.RequestID, correlation)
	}
}

func TestRESTSolicitarNuevaAppClientV0AceptaContextNil(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	vm, err := client.SolicitarNuevaApp(nil, minimalFormForClientV0("req-context-nil"))
	if err != nil {
		t.Fatalf("SolicitarNuevaApp: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoValida {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTSolicitarNuevaAppClientV0TransportaProjectSourceEnJSON(t *testing.T) {
	var received orquestafactory.AppSpecRequestV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	form := minimalFormForClientV0("req-project-source")
	form.ProjectSource = WebNuevaAppProjectSourceFormV0{
		Kind:       "github",
		GitURL:     "https://example.test/agenda.git",
		Branch:     "main",
		ProjectRef: "project-ref-agenda",
	}

	if _, err := client.SolicitarNuevaApp(context.Background(), form); err != nil {
		t.Fatalf("SolicitarNuevaApp error: %v", err)
	}
	if received.RequestID != "req-project-source" ||
		received.ProjectSource.Kind != "github" ||
		received.ProjectSource.GitURL != "https://example.test/agenda.git" ||
		received.ProjectSource.Branch != "main" ||
		received.ProjectSource.ProjectRef != "project-ref-agenda" {
		t.Fatalf("payload project_source inesperado: %+v", received)
	}
}

func TestRESTSolicitarNuevaAppClientV0Respuesta2xxGeneraViewModelConSpecYBacklog(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	vm, err := client.SolicitarNuevaApp(context.Background(), minimalFormForClientV0("req-success"))
	if err != nil {
		t.Fatalf("SolicitarNuevaApp error: %v", err)
	}

	if vm.Estado != WebNuevaAppEstadoValida || vm.ResumenApp.Nombre != "Agenda" || vm.ResumenApp.Slug != "agenda" {
		t.Fatalf("resumen=%+v estado=%q", vm.ResumenApp, vm.Estado)
	}
	if len(vm.Fases) != 1 || vm.Fases[0].ID != "discovery" {
		t.Fatalf("fases=%+v", vm.Fases)
	}
	if len(vm.Microtareas) != 1 || vm.Microtareas[0].ID != "BLG-001" {
		t.Fatalf("microtareas=%+v", vm.Microtareas)
	}
	if vm.AppSpecSchemaVersion != orquestafactory.AppSpecSchemaV0 || vm.BacklogSchemaVersion != orquestafactory.BacklogInicialPropuestoSchemaV0 {
		t.Fatalf("schemas: app=%q backlog=%q", vm.AppSpecSchemaVersion, vm.BacklogSchemaVersion)
	}
}

func TestRESTSolicitarNuevaAppClientV0Respuesta400GeneraViewModelInvalido(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errores": []orquestafactory.ValidationIssue{{
				Code:    orquestafactory.ErrIdiomaInvalido,
				Field:   "locale",
				Message: "locale BCP 47 invalido",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	vm, err := client.SolicitarNuevaApp(context.Background(), minimalFormForClientV0("req-invalid"))
	if err != nil {
		t.Fatalf("400 publico no debe ser error de transporte: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoInvalida || vm.RequestID != "req-invalid" || vm.Locale != "es" {
		t.Fatalf("vm=%+v", vm)
	}
	if len(vm.ErroresPublicos) != 1 || vm.ErroresPublicos[0].Code != orquestafactory.ErrIdiomaInvalido {
		t.Fatalf("errores_publicos=%+v", vm.ErroresPublicos)
	}
	if len(vm.Fases) != 0 || len(vm.Microtareas) != 0 {
		t.Fatalf("400 no debe inventar backlog: fases=%+v microtareas=%+v", vm.Fases, vm.Microtareas)
	}
}

func TestRESTSolicitarNuevaAppClientV0Status500DevuelveErrorPublicoEstable(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	_, err := client.SolicitarNuevaApp(context.Background(), minimalFormForClientV0("req-500"))
	if err == nil {
		t.Fatal("expected error")
	}
	var clientErr WebNuevaAppClientErrorV0
	if !errors.As(err, &clientErr) {
		t.Fatalf("error type=%T", err)
	}
	if clientErr.Code != WebNuevaAppErrTransporteV0 || clientErr.StatusCode != http.StatusInternalServerError || err.Error() != WebNuevaAppErrTransporteV0 {
		t.Fatalf("error publico inesperado: %+v error=%q", clientErr, err.Error())
	}
}

func TestRESTSolicitarNuevaAppClientV0TransportErrorYTimeoutDevuelvenErrorPublicoEstable(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	time.Sleep(time.Millisecond)
	defer cancel()
	_, err := client.SolicitarNuevaApp(ctx, minimalFormForClientV0("req-timeout"))
	if !IsWebNuevaAppClientErrorCodeV0(err, WebNuevaAppErrTransporteV0) {
		t.Fatalf("timeout error=%v", err)
	}

	server.Close()
	client = NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	_, err = client.SolicitarNuevaApp(context.Background(), minimalFormForClientV0("req-transport"))
	if !IsWebNuevaAppClientErrorCodeV0(err, WebNuevaAppErrTransporteV0) {
		t.Fatalf("transport error=%v", err)
	}
}

func minimalFormForClientV0(requestID string) WebNuevaAppFormV0 {
	return WebNuevaAppFormV0{
		RequestID: requestID,
		Locale:    "es",
		Nombre:    "Agenda",
		Objetivo:  "Coordinar ensayos",
		TipoApp:   "web",
	}
}

func writeSuccessV0(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"app_spec": validSpecForClientV0(),
		"backlog":  backlogForClientV0(),
	}); err != nil {
		t.Fatalf("encode success: %v", err)
	}
}

func validSpecForClientV0() orquestafactory.AppSpecV0 {
	spec := validSpecForViewModelV0()
	spec.RequestID = "req-client-1"
	spec.SpecID = "spec-agenda-req-client-1"
	return spec
}

func backlogForClientV0() orquestafactory.BacklogInicialPropuestoV0 {
	backlog := backlogForViewModelV0()
	backlog.SpecID = "spec-agenda-req-client-1"
	return backlog
}
