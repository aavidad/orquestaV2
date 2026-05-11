package orquestacli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSolicitarNuevaAppCliClientV0RechazaServerURLConCredenciales(t *testing.T) {
	_, err := NewSolicitarNuevaAppCliClientV0("https://usuario:secreto@example.test", time.Second)
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

	client := &SolicitarNuevaAppCliClientV0{BaseURL: "https://usuario:secreto@example.test", Timeout: time.Second}
	env := client.SolicitarNuevaApp(context.Background(), CliInvocationContextV0{
		RequestID:    "req-creds",
		OutputFormat: CliOutputFormatJSONV0,
	}, minimalAppSpecRequestForCliV0())
	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrConfiguracionInvalidaV0 {
		t.Fatalf("envelope credenciales inesperado: %+v", env)
	}
	if strings.Contains(env.Errores[0].Detalle, "usuario") || strings.Contains(env.Errores[0].Detalle, "secreto") {
		t.Fatalf("envelope filtra credenciales: %+v", env.Errores[0])
	}
}

func TestSolicitarNuevaAppCliClientV0DryRunRechazadoSinLlamarServidor(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatal("server no debe llamarse con dry_run no contratado")
	}))
	defer server.Close()

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, time.Second)
	inv := invocationForCliClientV0(server.URL)
	inv.DryRun = true
	env := client.SolicitarNuevaApp(context.Background(), inv, minimalAppSpecRequestForCliV0())

	if called {
		t.Fatal("server llamado")
	}
	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrOpcionInvalidaV0 || env.Errores[0].Campo != "dry_run" {
		t.Fatalf("dry_run envelope inesperado: %+v", env)
	}
}
