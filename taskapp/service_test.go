package taskapp

import (
	"errors"
	"testing"

	"orquesta/db"
)

type stubStore struct {
	proposal *db.Propuesta
	created  *db.Tarea
	taken    struct {
		id     int64
		agente string
	}
}

func (s *stubStore) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return nil, nil
}

func (s *stubStore) GetTask(id int64) (*db.Tarea, error) {
	return &db.Tarea{ID: id, Titulo: "demo"}, nil
}

func (s *stubStore) CreateTask(t *db.Tarea) (int64, error) {
	s.created = t
	return 77, nil
}

func (s *stubStore) TakeTask(id int64, agente string) error {
	s.taken.id = id
	s.taken.agente = agente
	return nil
}

func (s *stubStore) StartTask(id int64, agente string) error               { return nil }
func (s *stubStore) CompleteTask(id int64, agente, commit string) error    { return nil }
func (s *stubStore) BlockTask(id int64, agente, motivo string) error       { return nil }
func (s *stubStore) UnblockTask(id int64, agente, resolucion string) error { return nil }
func (s *stubStore) AnnotateTask(id int64, agente, nota string) error      { return nil }
func (s *stubStore) MoveTaskToBacklog(id int64) error                      { return nil }
func (s *stubStore) ReassignTask(id int64, nuevoAgente string) error       { return nil }
func (s *stubStore) ListAgents() ([]*db.Agente, error)                     { return nil, nil }
func (s *stubStore) GetProposal(codigo string) (*db.Propuesta, error) {
	if s.proposal == nil {
		return nil, errors.New("not found")
	}
	return s.proposal, nil
}

func TestCreateResolvesProposalAndAssignsTask(t *testing.T) {
	store := &stubStore{proposal: &db.Propuesta{ID: 12, Codigo: "OP-080"}}
	service := NewService(store)

	id, err := service.Create(CreateTaskInput{
		Titulo:          "Mover tareas a servicio",
		Prioridad:       db.PrioridadAlta,
		CreadoPor:       "",
		Agente:          "Codex2",
		PropuestaCodigo: "OP-080",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != 77 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.created == nil || store.created.PropuestaID == nil || *store.created.PropuestaID != 12 {
		t.Fatalf("propuesta no resuelta en tarea creada: %+v", store.created)
	}
	if store.created.CreadoPor != "alberto" {
		t.Fatalf("creadoPor inesperado: %+v", store.created)
	}
	if store.taken.id != 77 || store.taken.agente != "Codex2" {
		t.Fatalf("asignacion inesperada: %+v", store.taken)
	}
}
