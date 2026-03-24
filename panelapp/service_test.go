package panelapp

import (
	"testing"
	"time"

	"orquesta/db"
)

type stubStore struct{}

func (stubStore) ListAgents() ([]*db.Agente, error) {
	return []*db.Agente{{Nombre: "Codex2", Rol: "programador", Activo: true}}, nil
}

func (stubStore) CountTasksByState() (map[string]int, error) {
	return map[string]int{
		string(db.EstadoCompletada): 3,
		string(db.EstadoEnProgreso): 1,
		string(db.EstadoAsignada):   2,
	}, nil
}

func (stubStore) ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error) {
	return []*db.Propuesta{{ID: 8, Codigo: "OP-080", Titulo: "Servidor unico"}}, nil
}

func (stubStore) CountVotes(propuestaID int64) (int, int, int, int, error) {
	return 3, 0, 0, 1, nil
}

func (stubStore) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	agente := "Codex2"
	return []*db.Tarea{{
		ID:        7,
		Titulo:    "Mover dashboard a servicio",
		Estado:    db.EstadoEnProgreso,
		Prioridad: db.PrioridadAlta,
		Agente:    &agente,
		CreatedAt: time.Now(),
	}}, nil
}

func TestBuildSummary(t *testing.T) {
	service := NewService(stubStore{})

	summary, err := service.BuildSummary()
	if err != nil {
		t.Fatalf("BuildSummary: %v", err)
	}
	if summary.TotalTasks != 6 || summary.DoneTasks != 3 || summary.PercentDone != 50 {
		t.Fatalf("conteos inesperados: %+v", summary)
	}
	if len(summary.OpenProps) != 1 || summary.OpenProps[0].Codigo != "OP-080" || summary.OpenProps[0].Pendiente != 1 {
		t.Fatalf("propuestas inesperadas: %+v", summary.OpenProps)
	}
	if len(summary.ActiveTasks) != 1 || summary.ActiveTasks[0].ID != 7 {
		t.Fatalf("tareas activas inesperadas: %+v", summary.ActiveTasks)
	}
}
