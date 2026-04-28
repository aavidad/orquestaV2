package cmd

import (
	"testing"

	"orquesta/db"
)

func TestBuildOpenClawCapacitySummaryOmiteSupervisorReservado(t *testing.T) {
	prevConfigGet := statusConfigGet
	prevAutonomy := statusListAutonomyFetcher
	t.Cleanup(func() {
		statusConfigGet = prevConfigGet
		statusListAutonomyFetcher = prevAutonomy
	})

	statusConfigGet = func(string) (string, error) { return "Codex2", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) { return nil, nil }

	got := buildOpenClawCapacitySummary(
		[]*db.Agente{{Nombre: "Codex2"}, {Nombre: "Codex4"}},
		[]*db.Agente{{Nombre: "Codex2"}},
		nil,
		map[string]int{string(db.TareaLibre): 3},
	)

	if got.WorkersConectados != 1 || got.WorkersOciosos != 1 || got.WorkersDisponibles != 1 || got.CapacidadLibre != 1 {
		t.Fatalf("capacity summary deberia omitir supervisor reservado: %+v", got)
	}
}

func TestPreferredSupervisorWorkerOmiteSupervisorReservado(t *testing.T) {
	prevConfigGet := statusConfigGet
	prevAutonomy := statusListAutonomyFetcher
	t.Cleanup(func() {
		statusConfigGet = prevConfigGet
		statusListAutonomyFetcher = prevAutonomy
	})

	statusConfigGet = func(string) (string, error) { return "Codex2", nil }
	statusListAutonomyFetcher = func() ([]*db.ProyectoAutonomia, error) { return nil, nil }

	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{{Nombre: "Codex2"}, {Nombre: "Codex4"}},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex2"},
		},
	}

	if got := preferredSupervisorWorker(status); got != "Codex4" {
		t.Fatalf("deberia preferir worker real y no supervisor reservado: %q", got)
	}
}
