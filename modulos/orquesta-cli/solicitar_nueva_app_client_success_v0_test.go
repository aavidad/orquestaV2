package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestSolicitarNuevaAppCliClientV0ExitoPropagaCorrelacionYEnvelopeCanonico(t *testing.T) {
	var received orquestafactory.AppSpecRequestV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != SolicitarNuevaAppCliEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(SolicitarNuevaAppCliCorrelationHeaderV0); got != "corr-cli-1" {
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
		if received.Source != SolicitarNuevaAppCliSourceV0 || received.RequestID != "req-cli-1" {
			t.Fatalf("request tecnica no normalizada: %+v", received)
		}
		writeSolicitarNuevaAppSuccessV0(t, w, received, false)
	}))
	defer server.Close()

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, 1200*time.Millisecond)
	env := client.SolicitarNuevaApp(context.Background(), CliInvocationContextV0{
		RequestID:     "req-cli-1",
		CorrelationID: "corr-cli-1",
		ServerURL:     server.URL,
		Timeout:       time.Second,
		OutputFormat:  CliOutputFormatJSONV0,
	}, minimalAppSpecRequestForCliV0())

	if !env.OK || env.RequestID != "req-cli-1" || env.CorrelationID != "corr-cli-1" {
		t.Fatalf("envelope ids/ok inesperado: %+v", env)
	}
	if env.Contract != CliContractSolicitarNuevaAppV0 || env.Version != CliContractVersionSolicitarAppV0 {
		t.Fatalf("contrato inesperado: %s %s", env.Contract, env.Version)
	}
	if env.Meta.StatusCode != http.StatusOK || env.Meta.Transporte != CliTransportRESTV0 || env.Meta.Retryable {
		t.Fatalf("meta inesperada: %+v", env.Meta)
	}
	data, ok := env.Data.(SolicitarNuevaAppCliResultV0)
	if !ok {
		t.Fatalf("data type=%T", env.Data)
	}
	if data.AppSpec.SchemaVersion != orquestafactory.AppSpecSchemaV0 ||
		data.Backlog.SchemaVersion != orquestafactory.BacklogInicialPropuestoSchemaV0 ||
		data.AppSpec.RequestID != "req-cli-1" ||
		data.AppSpec.App.Nombre != "Agenda" {
		t.Fatalf("data canonica inesperada: %+v", data)
	}
}

func TestSolicitarNuevaAppCliClientV0AceptaAliasRemotoPeroSalidaEsCanonica(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received orquestafactory.AppSpecRequestV0
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeSolicitarNuevaAppSuccessV0(t, w, received, true)
	}))
	defer server.Close()

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, time.Second)
	env := client.SolicitarNuevaApp(context.Background(), invocationForCliClientV0(server.URL), minimalAppSpecRequestForCliV0())
	if !env.OK {
		t.Fatalf("alias remoto deberia ser aceptado: %+v", env)
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(raw, []byte(`"spec":`)) || bytes.Contains(raw, []byte(`"backlog_inicial_propuesto":`)) {
		t.Fatalf("salida CLI usa alias no canonico: %s", string(raw))
	}
	if !bytes.Contains(raw, []byte(`"app_spec":`)) || !bytes.Contains(raw, []byte(`"backlog":`)) {
		t.Fatalf("salida CLI no contiene shape canonico: %s", string(raw))
	}
}
