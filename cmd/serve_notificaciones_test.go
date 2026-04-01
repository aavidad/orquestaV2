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
	"net/url"
	"strconv"
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

func TestWebOpenClawMuestraOperatorReviewYEntregas(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_url", "https://openclaw.local/gateway"); err != nil {
		t.Fatalf("config openclaw url: %v", err)
	}
	if err := db.ConfigSet("openclaw_gateway_operator", "OpenClaw"); err != nil {
		t.Fatalf("config openclaw operator: %v", err)
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
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Operador OpenClaw",
		Modulo:    "openclaw",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(taskID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(taskID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		TareaID:         &taskID,
		Estado:          db.ReviewGatePendiente,
		ReviewerAgente:  "OpenClaw",
		SeverityMax:     "media",
		RequestedBy:     "Codex3",
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}
	if gateID <= 0 {
		t.Fatalf("review gate invalido: %d", gateID)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", webHandlerOpenClaw)
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openclaw", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{
		"OpenClaw Operator",
		"Acción siguiente",
		"Review e integración",
		"OpenClaw Gateway y notificaciones",
		"gateway down",
		"Operador OpenClaw",
		"OpenClaw",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("pagina openclaw sin %q:\n%s", token, body)
		}
	}
}

func TestWebOpenClawAccionResuelveReviewGate(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Review OpenClaw",
		Modulo:    "openclaw",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "OpenClaw",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		TareaID:         &taskID,
		RequestedBy:     "Codex3",
		ReviewerAgente:  "OpenClaw",
		Estado:          db.ReviewGatePendiente,
		SeverityMax:     "media",
		FindingsJSON:    `[{"severity":"media","title":"falta prueba"}]`,
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":            {"review_gate"},
		"gate_id":         {strconv.FormatInt(gateID, 10)},
		"estado":          {"aprobado"},
		"reviewer_agente": {"OpenClaw"},
		"findings_json":   {`[{"severity":"baja","title":"ok"}]`},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Review+gate") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateAprobado {
		t.Fatalf("gate no resuelto: %#v", gate)
	}
	if gate.ResolvedAt == nil {
		t.Fatalf("gate aprobado sin resolved_at: %#v", gate)
	}
}

func TestWebOpenClawAccionReseteaReanimacionDeAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	reanimarAt := time.Now().UTC().Add(45 * time.Minute)
	if _, err := db.DB.Exec(`
		UPDATE agentes
		SET activo = 0,
		    estado_cuota = 'enfriamiento',
		    motivo_pausa = 'usage limit',
		    reanimar_at = ?,
		    limite_semanal_segundos = 604800,
		    consumo_semanal_segundos = 3600,
		    limite_dia_segundos = 18000,
		    consumo_dia_segundos = 1200
		WHERE nombre = ?`, reanimarAt, "Codex5"); err != nil {
		t.Fatalf("preparar agente: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	registerAPIRoutes(mux)

	form := url.Values{
		"kind":   {"agente"},
		"agente": {"Codex5"},
		"accion": {"reset-reanimacion"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("accion openclaw status=%d body=%s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Acci%C3%B3n+reset-reanimacion") {
		t.Fatalf("redirect inesperado: %s", location)
	}
	agente, err := db.GetAgente("Codex5")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente == nil {
		t.Fatalf("agente nil")
	}
	if strings.TrimSpace(agente.EstadoCuota) != "activo" {
		t.Fatalf("estado_cuota no reseteado: %#v", agente)
	}
	if agente.ReanimarAt != nil {
		t.Fatalf("reanmiar_at no limpiado: %#v", agente)
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
