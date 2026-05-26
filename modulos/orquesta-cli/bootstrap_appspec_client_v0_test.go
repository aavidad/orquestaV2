package orquestacli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBootstrapAppSpecCliClientV0CuarentenaNoHaceHTTP(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(w, "legacy route must not be called", http.StatusInternalServerError)
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

	if called {
		t.Fatalf("cliente llamo ruta legacy %s", BootstrapAppSpecCliLegacyEndpointV0)
	}
	if env.OK || env.Data != nil || env.RequestID != "req-bootstrap-1" || env.CorrelationID != "corr-bootstrap-1" {
		t.Fatalf("envelope ids/ok inesperado: %+v", env)
	}
	if env.Contract != BootstrapAppSpecCliContractV0 || env.Version != BootstrapAppSpecCliContractVersionV0 {
		t.Fatalf("contrato inesperado: %s %s", env.Contract, env.Version)
	}
	if env.Meta.StatusCode != 0 || env.Meta.Transporte != CliTransportRESTV0 || env.Meta.Retryable {
		t.Fatalf("meta inesperada: %+v", env.Meta)
	}
	if len(env.Errores) != 1 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	if err := env.Errores[0]; err.Codigo != CliErrContratoNoConfiguradoV0 ||
		err.Campo != "route_policy" ||
		err.MensajeI18N != CliDefaultErrorMessageNamespaceV0+CliErrContratoNoConfiguradoV0 ||
		err.Detalle != BootstrapAppSpecCliQuarantineDetailV0 {
		t.Fatalf("error publico inesperado: %+v", err)
	}
}

func TestBootstrapAppSpecCliClientV0CuarentenaNoRequiereServerURL(t *testing.T) {
	client := mustNewBootstrapAppSpecCliClientV0(t, "", time.Second)
	env := client.BootstrapProyectoDesdeAppSpec(context.Background(), invocationForBootstrapAppSpecCliV0(""), minimalBootstrapAppSpecCommandForCliV0(t))

	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrContratoNoConfiguradoV0 {
		t.Fatalf("cuarentena no aplicada: %+v", env)
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
}
