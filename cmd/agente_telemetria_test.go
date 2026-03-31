/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

func setAgentTelemetryHTTPClientForTest(t *testing.T, transport roundTripFunc) {
	t.Helper()
	prevHTTPClient := httpClientOrquesta
	prevServerClient := serverHTTPClient
	prevURL := os.Getenv("ORQUESTA_SERVER_URL")
	httpClientOrquesta = &http.Client{Transport: transport}
	serverHTTPClient = httpClientOrquesta
	if err := os.Setenv("ORQUESTA_SERVER_URL", "http://orquesta.test"); err != nil {
		t.Fatalf("setenv ORQUESTA_SERVER_URL: %v", err)
	}
	resetServerDiscovery()
	t.Cleanup(func() {
		httpClientOrquesta = prevHTTPClient
		serverHTTPClient = prevServerClient
		if prevURL == "" {
			_ = os.Unsetenv("ORQUESTA_SERVER_URL")
		} else {
			_ = os.Setenv("ORQUESTA_SERVER_URL", prevURL)
		}
		resetServerDiscovery()
	})
}

func TestAgenteCuentasCmdRenderizaListado(t *testing.T) {
	setAgentTelemetryHTTPClientForTest(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/api/agentes/cuentas" && req.URL.Query().Get("activos") == "true":
			return newJSONResponse(http.StatusOK, `{"generado":"2026-03-31T18:00:00Z","activos":true,"agentes":[{"nombre":"Codex1","cuenta_email":"codex1@example.com","cuenta_usuario":"codex1_user"}]}`), nil
		default:
			return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	}))
	out := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"agente", "cuentas", "--activos"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute agente cuentas: %v", err)
		}
	})
	if !strings.Contains(out, "Codex1") || !strings.Contains(out, "codex1@example.com") || !strings.Contains(out, "codex1_user") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestAgenteCuentasCmdRenderizaUsuarioSinGuionCuandoNoHayCorreo(t *testing.T) {
	setAgentTelemetryHTTPClientForTest(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/api/agentes/cuentas" && req.URL.Query().Get("activos") == "true":
			return newJSONResponse(http.StatusOK, `{"generado":"2026-03-31T18:00:00Z","activos":true,"agentes":[{"nombre":"Codex1","cuenta_usuario":"Codex1"}]}`), nil
		default:
			return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	}))
	out := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"agente", "cuentas", "--activos"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute agente cuentas: %v", err)
		}
	})
	if !strings.Contains(out, "usuario Codex1") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
	if strings.Contains(out, "— (Codex1)") {
		t.Fatalf("la salida no deberia usar el formato legado:\n%s", out)
	}
}

func TestAgentePresupuestoCmdRenderizaListado(t *testing.T) {
	setAgentTelemetryHTTPClientForTest(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/api/agentes/presupuesto" && req.URL.Query().Get("activos") == "true":
			return newJSONResponse(http.StatusOK, `{"generado":"2026-03-31T18:00:00Z","activos":true,"agentes":[{"Nombre":"Codex1","CuotaRestantePct":18,"PresupuestoVentana":"weekly","PresupuestoResetAt":"2026-04-06T02:00:00Z","PresupuestoDiarioPct":77,"PresupuestoDiarioResetAt":"2026-04-01T02:00:00Z","PresupuestoSemanalPct":18,"PresupuestoSemanalResetAt":"2026-04-06T02:00:00Z"}]}`), nil
		default:
			return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	}))
	out := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"agente", "presupuesto", "--activos"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute agente presupuesto: %v", err)
		}
	})
	if !strings.Contains(out, "Codex1") || !strings.Contains(out, "efectivo 18%") || !strings.Contains(out, "weekly") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestAgenteRankingCuentasCmdRenderizaListado(t *testing.T) {
	setAgentTelemetryHTTPClientForTest(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/api/agentes/ranking-cuentas" && req.URL.Query().Get("activos") == "true":
			return newJSONResponse(http.StatusOK, `{"generado":"2026-03-31T18:00:00Z","activos":true,"cuentas":[{"cuenta_clave":"codex2@example.com","cuenta_email":"codex2@example.com","cuenta_usuario":"codex2","criterio":"remaining_tokens","remaining_tokens":120000,"agentes":["Codex2"]},{"cuenta_clave":"codex1","cuenta_usuario":"codex1","criterio":"cuota_pct","cuota_restante_pct":77,"presupuesto_ventana":"daily","agentes":["Codex1"]}]}`), nil
		default:
			return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	}))
	out := captureOutput(t, func() {
		rootCmd.SetArgs([]string{"agente", "ranking-cuentas", "--activos"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute agente ranking-cuentas: %v", err)
		}
	})
	if !strings.Contains(out, "1.") || !strings.Contains(out, "codex2@example.com") || !strings.Contains(out, "tokens 120000") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
	if !strings.Contains(out, "2.") || !strings.Contains(out, "efectivo 77%") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}
