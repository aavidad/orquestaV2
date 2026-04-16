package cmd

import (
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestRowPermiteAutoRecuperacionBloqueaCuotaVisible(t *testing.T) {
	now := time.Now().UTC()
	cuota := 25
	row := agentesapp.Row{
		Agente: &db.Agente{
			Nombre:               "Codex1",
			EstadoCuota:          "enfriamiento",
			MotivoPausa:          "Cuota diaria agotada",
			CuotaRestantePct:     &cuota,
			PresupuestoEstado:    "observado_stale",
			PresupuestoCheckedAt: &now,
		},
		BlockedTasks:              1,
		WorkerAlive:               true,
		WorkerHeartbeat:           &now,
		WorkerUpdatedAt:           &now,
		WorkerDriver:              "tmux_cli_session",
		WorkerMailboxDeliveryMode: "bootstrap_only",
		WorkerState:               "ready",
		WorkerTMUXSession:         "orq-codex1",
	}

	if rowPermiteAutoRecuperacion(row, now) {
		t.Fatalf("no deberia permitir autorecuperacion mientras la cuota visible siga bloqueando")
	}
}

func TestRowPermiteAutoRecuperacionAceptaContinuidadSinCuotaVisible(t *testing.T) {
	now := time.Now().UTC()
	row := agentesapp.Row{
		Agente: &db.Agente{
			Nombre:      "Codex1",
			EstadoCuota: "activo",
		},
		BlockedTasks:              1,
		WorkerAlive:               true,
		WorkerHeartbeat:           &now,
		WorkerUpdatedAt:           &now,
		WorkerDriver:              "tmux_cli_session",
		WorkerMailboxDeliveryMode: "bootstrap_only",
		WorkerState:               "ready",
		WorkerTMUXSession:         "orq-codex1",
	}

	if !rowPermiteAutoRecuperacion(row, now) {
		t.Fatalf("deberia permitir autorecuperacion cuando la sesion sigue viva y no hay cuota visible")
	}
}
