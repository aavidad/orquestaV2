package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestAgenteDebeEntrarEnReanimacionAutomaticaAceptaCuotaVisibleRecuperadaAntesDeReanimarAt(t *testing.T) {
	now := time.Now().UTC()
	quota := 86
	agente := &db.Agente{
		Nombre:            "Gemini1",
		Habilitado:        true,
		EstadoCuota:       "enfriamiento",
		MotivoPausa:       "worker bloqueado por cuota",
		ReanimarAt:        timePtr(now.Add(45 * time.Minute)),
		PresupuestoEstado: "ok",
		CuotaRestantePct:  &quota,
	}
	if !agenteDebeEntrarEnReanimacionAutomatica(agente, now) {
		t.Fatalf("deberia admitir reanimacion temprana por cuota visible recuperada: %+v", agente)
	}
}

func TestAgenteDebeEntrarEnReanimacionAutomaticaNoAceptaCuotaVisibleStale(t *testing.T) {
	now := time.Now().UTC()
	quota := 86
	agente := &db.Agente{
		Nombre:            "Gemini1",
		Habilitado:        true,
		EstadoCuota:       "enfriamiento",
		MotivoPausa:       "worker bloqueado por cuota",
		ReanimarAt:        timePtr(now.Add(45 * time.Minute)),
		PresupuestoEstado: "ok",
		CuotaRestantePct:  &quota,
		PresupuestoStale:  true,
	}
	if agenteDebeEntrarEnReanimacionAutomatica(agente, now) {
		t.Fatalf("no deberia reanimar antes con presupuesto stale: %+v", agente)
	}
}

func TestPresupuestoAgenteDebeRevalidarseAhoraSiResetVisibleYaVencio(t *testing.T) {
	now := time.Now().UTC()
	quota := 0
	checked := now.Add(-5 * time.Minute)
	reset := now.Add(-time.Minute)
	agente := &db.Agente{
		Nombre:               "Gemini1",
		Habilitado:           true,
		EstadoCuota:          "enfriamiento",
		CuotaRestantePct:     &quota,
		PresupuestoVentana:   "5h",
		PresupuestoCheckedAt: &checked,
		PresupuestoResetAt:   &reset,
	}
	if !presupuestoAgenteDebeRevalidarseAhora(agente, time.Hour, true) {
		t.Fatalf("deberia forzar revalidacion cuando el reset visible ya vencio: %+v", agente)
	}
}

func TestPresupuestoAgenteNoSeRevalidaAntesDeMinAgeSiResetVisibleNoVencio(t *testing.T) {
	now := time.Now().UTC()
	quota := 0
	checked := now.Add(-5 * time.Minute)
	reset := now.Add(30 * time.Minute)
	agente := &db.Agente{
		Nombre:               "Gemini1",
		Habilitado:           true,
		EstadoCuota:          "enfriamiento",
		CuotaRestantePct:     &quota,
		PresupuestoVentana:   "5h",
		PresupuestoCheckedAt: &checked,
		PresupuestoResetAt:   &reset,
	}
	if presupuestoAgenteDebeRevalidarseAhora(agente, time.Hour, true) {
		t.Fatalf("no deberia saltarse minAge mientras el reset visible siga en el futuro: %+v", agente)
	}
}
