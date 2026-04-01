/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestWebDashMuestraEstadoOpenClawNotificaciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.ConfigSet("openclaw_gateway_url", "https://openclaw.local/gateway"); err != nil {
		t.Fatalf("config openclaw url: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_operator", "alberto"); err != nil {
		t.Fatalf("config openclaw operator: %v", err)
	}
	if err := db.ConfigSet("telegram_token", "tg-token"); err != nil {
		t.Fatalf("config telegram token: %v", err)
	}
	if err := db.ConfigSet("telegram_chat_id", "12345"); err != nil {
		t.Fatalf("config telegram chat id: %v", err)
	}
	entregaID, err := db.CrearEntregaNotificacion("openclaw_gateway", "https://openclaw.local/gateway", db.EventoNotificacion{
		Tipo:  "mensaje",
		Texto: "hola openclaw",
	})
	if err != nil {
		t.Fatalf("crear entrega notificacion: %v", err)
	}
	if err := db.MarcarEntregaNotificacionFallida(entregaID, "gateway down", time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatalf("marcar entrega fallida: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dash status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"OpenClaw / notificaciones", "OpenClaw Gateway", "https://openclaw.local/gateway", "Telegram", "12345", "Entregas recientes", "fallida", "gateway down"} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard sin %q:\n%s", token, body)
		}
	}
}

func TestWebDashMuestraCuentaYVentanasDeCuotaAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	remaining := int64(900)
	resetSesion := time.Date(2026, 3, 31, 21, 0, 0, 0, time.UTC)
	inicioSesion := resetSesion.Add(-5 * time.Hour)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		WindowStartedAt:  &inicioSesion,
		ResetAt:          &resetSesion,
		RemainingSeconds: &remaining,
		BudgetSource:     "runtime",
		RawSnapshotJSON:  `{"account":{"email":"codex1@example.com","username":"codex1_user"}}`,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dash status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"cuenta: codex1@example.com",
		"usuario: codex1_user",
		"efectivo",
		"sesión",
		"diario",
		"semanal",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard sin %q:\n%s", token, body)
		}
	}
}
