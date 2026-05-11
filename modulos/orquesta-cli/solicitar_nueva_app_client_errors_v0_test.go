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

func TestSolicitarNuevaAppCliClientV0Respuesta400DevuelveErroresPublicos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, time.Second)
	env := client.SolicitarNuevaApp(context.Background(), invocationForCliClientV0(server.URL), minimalAppSpecRequestForCliV0())

	if env.OK || env.Data != nil || env.Meta.StatusCode != http.StatusBadRequest || env.Meta.Retryable {
		t.Fatalf("envelope 400 inesperado: %+v", env)
	}
	if len(env.Errores) != 1 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	if err := env.Errores[0]; err.Codigo != orquestafactory.ErrIdiomaInvalido ||
		err.Campo != "locale" ||
		err.MensajeI18N != "orquesta_factory.errores."+orquestafactory.ErrIdiomaInvalido ||
		err.Detalle != "locale BCP 47 invalido" {
		t.Fatalf("error publico inesperado: %+v", err)
	}
}

func TestSolicitarNuevaAppCliClientV0Status500NoFiltraBodyPrivado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado token=secreto /home/alberto/runtime", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, time.Second)
	env := client.SolicitarNuevaApp(context.Background(), invocationForCliClientV0(server.URL), minimalAppSpecRequestForCliV0())

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

func TestSolicitarNuevaAppCliClientV0TimeoutDevuelveErrorTransporte(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeSolicitarNuevaAppSuccessV0(t, w, minimalAppSpecRequestForCliV0(), false)
	}))
	defer server.Close()

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, 2*time.Millisecond)
	env := client.SolicitarNuevaApp(context.Background(), invocationForCliClientV0(server.URL), minimalAppSpecRequestForCliV0())

	if env.OK || len(env.Errores) != 1 {
		t.Fatalf("timeout envelope inesperado: %+v", env)
	}
	if env.Errores[0].Codigo != CliErrErrorTransporteV0 || env.Errores[0].Detalle != "timeout" || !env.Meta.Retryable {
		t.Fatalf("timeout error inesperado: errores=%+v meta=%+v", env.Errores, env.Meta)
	}
}

func TestSolicitarNuevaAppCliClientV0RespuestaInvalida(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"app_spec": map[string]any{}})
	}))
	defer server.Close()

	client := mustNewSolicitarNuevaAppCliClientV0(t, server.URL, time.Second)
	env := client.SolicitarNuevaApp(context.Background(), invocationForCliClientV0(server.URL), minimalAppSpecRequestForCliV0())

	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrRespuestaInvalidaV0 {
		t.Fatalf("respuesta invalida no detectada: %+v", env)
	}
	if env.Data != nil || env.Meta.StatusCode != http.StatusOK {
		t.Fatalf("respuesta invalida con data/status inesperado: %+v", env)
	}
}
