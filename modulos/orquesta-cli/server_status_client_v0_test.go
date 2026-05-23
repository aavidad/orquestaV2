package orquestacli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerStatusCliClientV0ConsultaHTTPGetSinFallbackLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != ServerStatusCliEndpointV0 {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get(CliCorrelationHeaderV0); got != "corr-server-status-001" {
			t.Fatalf("correlation=%q", got)
		}
		_ = json.NewEncoder(w).Encode(orquestaserver.StateV0{
			SchemaVersion: orquestaserver.StateSchemaVersionV0,
			Status:        "running",
			Addr:          "127.0.0.1:8787",
		})
	}))
	defer server.Close()

	client, err := NewServerStatusCliClientV0(server.URL, time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	env := client.ConsultarEstadoServidor(context.Background(), CliInvocationContextV0{
		RequestID:     "req-server-status-001",
		CorrelationID: "corr-server-status-001",
	})

	if !env.OK || env.Contract != CliContractServerStatusV0 {
		t.Fatalf("env=%+v", env)
	}
}

func TestServerStatusCliClientV0RechazaRespuestaInvalida(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		_, _ = w.Write([]byte(`{"schema_version":"orquesta_server_state.v0"}`))
	}))
	defer server.Close()

	client, err := NewServerStatusCliClientV0(server.URL, time.Second)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	env := client.ConsultarEstadoServidor(context.Background(), CliInvocationContextV0{})

	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrRespuestaInvalidaV0 {
		t.Fatalf("env=%+v", env)
	}
}
