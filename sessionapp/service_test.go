package sessionapp

import (
	"testing"

	"orquesta/db"
)

type stubStore struct {
	startedAgente  string
	finishedAgente string
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
