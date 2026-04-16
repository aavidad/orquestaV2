package cmd

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestResetReanimacionResultadoDistingueCooldownSostenido(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Claude1")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetSecondary := now.Add(72 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":0,"window_minutes":240,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "4h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("registrar presupuesto visible: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='worker bloqueado por cuota' WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("marcar cuota visible agotada: %v", err)
	}

	resultado, err := (dbAutomationService{}).ResetReanimacionResultado("Claude1")
	if err != nil {
		t.Fatalf("ResetReanimacionResultado: %v", err)
	}
	if resultado != resetReanimacionResultadoCooldownSostenido {
		t.Fatalf("resultado inesperado: %s", resultado)
	}
}

func TestResetReanimacionResultadoCancelaBootstrapVivoSiCooldownSigueSostenido(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Claude1")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(4 * time.Hour)
	resetSecondary := now.Add(72 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":0,"window_minutes":240,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "4h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("registrar presupuesto visible: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='worker bloqueado por cuota' WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("marcar cuota visible agotada: %v", err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:  "Claude1",
		Tipo:    "start",
		Estado:  "ejecutando",
		PayloadJSON: `{"accion":"start","motivo":"manual_rehabilitation"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}

	resultado, err := (dbAutomationService{}).ResetReanimacionResultado("Claude1")
	if err != nil {
		t.Fatalf("ResetReanimacionResultado: %v", err)
	}
	if resultado != resetReanimacionResultadoCooldownSostenido {
		t.Fatalf("resultado inesperado: %s", resultado)
	}

	order, err := db.GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil || order.Estado != "cancelada" {
		t.Fatalf("la start order deberia quedar cancelada: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"quota_blocked":true`) || !strings.Contains(order.ResultadoJSON, `"superseded_reason":"quota_cooldown"`) {
		t.Fatalf("resultado de cancelacion inesperado: %+v", order)
	}
}
