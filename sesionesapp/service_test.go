package sesionesapp

import (
	"testing"

	"orquesta/db"
)

type stubStore struct {
	startedAgente  string
	finishedAgente string
	project        *db.Proyecto
	inspection     []*db.Sesion
	active         *db.Sesion
	last           *db.Sesion
	saved          struct {
		agente     string
		proyectoID *int64
		upd        db.SesionUpdate
	}
}

func (s *stubStore) RegisterCodex() (string, error) { return "codex9", nil }
func (s *stubStore) StartSession(agente string) (int64, error) {
	s.startedAgente = agente
	return 17, nil
}
func (s *stubStore) FinishSession(agente string) error {
	s.finishedAgente = agente
	return nil
}
func (s *stubStore) GetProject(ref string) (*db.Proyecto, error) { return s.project, nil }
func (s *stubStore) ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	return s.inspection, nil
}
func (s *stubStore) GetInspectionSessionByID(id int64) (*db.Sesion, error) {
	return &db.Sesion{ID: id, Agente: "Codex2"}, nil
}
func (s *stubStore) SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error {
	s.saved.agente = agente
	s.saved.proyectoID = proyectoID
	s.saved.upd = upd
	return nil
}
func (s *stubStore) GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return s.active, nil
}
func (s *stubStore) GetLastSessionWithFilter(agente string, proyectoID *int64, cwd string) (*db.Sesion, error) {
	return s.last, nil
}
func (s *stubStore) ListAgents() ([]*db.Agente, error) {
	return []*db.Agente{{Nombre: "codex9", Rol: "programador"}, {Nombre: "Codex2", Rol: "programador"}}, nil
}
func (s *stubStore) ListPendingProposals(agente string) ([]*db.Propuesta, error) {
	return []*db.Propuesta{{Codigo: "OP-080", Titulo: "Servidor unico"}}, nil
}
func (s *stubStore) ListRules(rol string) ([]*db.Regla, error) {
	return []*db.Regla{{Categoria: "calidad", Titulo: "No romper tests"}}, nil
}
func (s *stubStore) ListSkills(rol string) ([]*db.Skill, error) {
	return []*db.Skill{{Nombre: "rg", CuandoUsar: "buscar rapido"}}, nil
}
func (s *stubStore) GetWorkflow(rol, nombre string) (*db.Workflow, error) {
	return &db.Workflow{Pasos: `["1. votar","2. tomar tarea"]`}, nil
}
func (s *stubStore) ListWorkflows(rol string) ([]*db.Workflow, error) {
	return []*db.Workflow{{Nombre: "inicio-sesion", Descripcion: "arranque"}, {Nombre: "fin-sesion", Descripcion: "cierre"}}, nil
}

func TestStartConstruyeBriefing(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	result, err := service.Start("", true)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if result.Agente != "codex9" || result.SesionID != 17 || result.Rol != "programador" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if len(result.PropuestasPendientes) != 1 || len(result.Reglas) != 1 || len(result.Skills) != 1 || len(result.WorkflowPasos) != 2 {
		t.Fatalf("briefing incompleto: %+v", result)
	}
}

func TestFinishDevuelveChecklistYCierraSesion(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	result, err := service.Finish("Codex2")
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if result.Agente != "Codex2" || len(result.WorkflowPasos) != 2 || store.finishedAgente != "Codex2" {
		t.Fatalf("resultado inesperado: %+v store=%+v", result, store)
	}
}

func TestBuildBriefing(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)

	result, err := service.BuildBriefing("Codex2")
	if err != nil {
		t.Fatalf("BuildBriefing: %v", err)
	}
	if result.Agent == nil || result.Agent.Nombre != "Codex2" {
		t.Fatalf("agente inesperado: %+v", result.Agent)
	}
	if len(result.PropuestasPendientes) != 1 || len(result.Reglas) != 1 || len(result.Skills) != 1 || len(result.Workflows) != 2 {
		t.Fatalf("briefing inesperado: %+v", result)
	}
}

func TestSaveAndContinueSession(t *testing.T) {
	projectID := int64(31)
	store := &stubStore{
		project: &db.Proyecto{ID: projectID, Slug: "orquestador"},
		active:  &db.Sesion{ID: 41, Agente: "Codex2"},
		last:    &db.Sesion{ID: 42, Agente: "Codex2"},
	}
	service := NewService(store)

	saved, err := service.SaveActiveSession("Codex2", "orquestador", db.SesionUpdate{Heartbeat: true})
	if err != nil {
		t.Fatalf("SaveActiveSession: %v", err)
	}
	if saved == nil || saved.ID != 41 {
		t.Fatalf("sesion guardada inesperada: %+v", saved)
	}
	if store.saved.agente != "Codex2" || store.saved.proyectoID == nil || *store.saved.proyectoID != projectID || !store.saved.upd.Heartbeat {
		t.Fatalf("save inesperado: %+v", store.saved)
	}

	continued, err := service.Continue("Codex2", "orquestador", "/tmp/x")
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	if continued == nil || continued.ID != 42 {
		t.Fatalf("sesion continuada inesperada: %+v", continued)
	}
}
