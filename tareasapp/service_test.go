package tareasapp

import (
	"errors"
	"reflect"
	"testing"

	"orquesta/db"
)

type stubStore struct {
	proposal *db.Propuesta
	project  *db.Proyecto
	created  *db.Tarea
	tasks    []*db.Tarea
	moved    []int64
	taken    struct {
		id     int64
		agente string
	}
}

func (s *stubStore) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	out := make([]*db.Tarea, 0)
	for _, tarea := range s.tasks {
		if tarea == nil {
			continue
		}
		if filtro.Estado != nil && tarea.Estado != *filtro.Estado {
			continue
		}
		if filtro.Agente != nil {
			if tarea.Agente == nil || *tarea.Agente != *filtro.Agente {
				continue
			}
		}
		if filtro.ProyectoID != nil {
			if tarea.ProyectoID == nil || *tarea.ProyectoID != *filtro.ProyectoID {
				continue
			}
		}
		out = append(out, tarea)
	}
	return out, nil
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
func (s *stubStore) MoveTaskToBacklog(id int64) error {
	s.moved = append(s.moved, id)
	return nil
}
func (s *stubStore) ReassignTask(id int64, nuevoAgente string) error       { return nil }
func (s *stubStore) ListAgents() ([]*db.Agente, error)                     { return nil, nil }
func (s *stubStore) GetProject(ref string) (*db.Proyecto, error) {
	if s.project == nil {
		return nil, errors.New("not found")
	}
	return s.project, nil
}
func (s *stubStore) GetProposal(codigo string) (*db.Propuesta, error) {
	if s.proposal == nil {
		return nil, errors.New("not found")
	}
	return s.proposal, nil
}

func TestCreateResolvesProposalAndAssignsTask(t *testing.T) {
	store := &stubStore{
		proposal: &db.Propuesta{ID: 12, Codigo: "OP-080"},
		project:  &db.Proyecto{ID: 21, Slug: "orquestador"},
	}
	service := NewService(store)

	id, err := service.Create(CreateTaskInput{
		Titulo:          "Mover tareas a servicio",
		Prioridad:       db.PrioridadAlta,
		CreadoPor:       "",
		Agente:          "Codex2",
		Proyecto:        "orquestador",
		PropuestaCodigo: "OP-080",
		Notas:           "nota inicial",
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
	if store.created.ProyectoID == nil || *store.created.ProyectoID != 21 {
		t.Fatalf("proyecto no resuelto en tarea creada: %+v", store.created)
	}
	if store.created.Notas != "nota inicial" {
		t.Fatalf("notas inesperadas: %+v", store.created)
	}
	if store.created.CreadoPor != "alberto" {
		t.Fatalf("creadoPor inesperado: %+v", store.created)
	}
	if store.taken.id != 77 || store.taken.agente != "Codex2" {
		t.Fatalf("asignacion inesperada: %+v", store.taken)
	}
}

func TestCleanActiveFrontMovesOnlyActiveTasksAndKeepsSelected(t *testing.T) {
	agente := "Codex1"
	proyectoID := int64(21)
	store := &stubStore{
		project: &db.Proyecto{ID: proyectoID, Slug: "orquestador"},
		tasks: []*db.Tarea{
			{ID: 1, Estado: db.EstadoEnProgreso, Agente: &agente, ProyectoID: &proyectoID},
			{ID: 2, Estado: db.EstadoBloqueada, Agente: &agente, ProyectoID: &proyectoID},
			{ID: 3, Estado: db.EstadoBacklog, Agente: &agente, ProyectoID: &proyectoID},
			{ID: 4, Estado: db.EstadoCompletada, Agente: &agente, ProyectoID: &proyectoID},
			{ID: 5, Estado: db.EstadoAsignada, ProyectoID: &proyectoID},
		},
	}
	service := NewService(store)

	result, err := service.CleanActiveFront(CleanActiveFrontInput{
		Agente:   agente,
		Proyecto: "orquestador",
		KeepIDs:  []int64{2},
	})
	if err != nil {
		t.Fatalf("CleanActiveFront: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("total inesperado: %+v", result)
	}
	if !reflect.DeepEqual(result.MovedIDs, []int64{1}) {
		t.Fatalf("moved ids inesperados: %+v", result.MovedIDs)
	}
	if !reflect.DeepEqual(store.moved, []int64{1}) {
		t.Fatalf("movimientos inesperados: %+v", store.moved)
	}
}

func TestCleanActiveFrontRequiresAgentOrProject(t *testing.T) {
	service := NewService(&stubStore{})
	if _, err := service.CleanActiveFront(CleanActiveFrontInput{}); err == nil {
		t.Fatal("se esperaba error sin agente ni proyecto")
	}
}
