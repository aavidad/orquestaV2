package panelapp

import (
	"testing"
	"time"
)

type stubStore struct{}

func (stubStore) ListAgents() ([]*Agente, error) {
	return []*Agente{{Nombre: "Codex2", Rol: "programador", Activo: true}}, nil
}

func (stubStore) CountTasksByState() (map[string]int, error) {
	return map[string]int{
		string(EstadoCompletada): 3,
		string(EstadoEnProgreso): 1,
		"asignada":               2,
	}, nil
}

func (stubStore) ListProposals(estado *EstadoPropuesta) ([]*Propuesta, error) {
	return []*Propuesta{{ID: 8, Codigo: "OP-080", Titulo: "Servidor unico"}}, nil
}

func (stubStore) CountVotes(propuestaID int64) (int, int, int, int, error) {
	return 3, 0, 0, 1, nil
}

func (stubStore) ListTasks(filtro FiltroTareas) ([]*Tarea, error) {
	agente := "Codex2"
	return []*Tarea{{
		ID:        7,
		Titulo:    "Mover dashboard a servicio",
		Estado:    EstadoEnProgreso,
		Prioridad: PrioridadTarea("alta"),
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
