package sesionesapp

import (
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/db"
)

type Store interface {
	RegisterCodex() (string, error)
	StartSession(agente string) (int64, error)
	FinishSession(agente string) error
	GetProject(ref string) (*db.Proyecto, error)
	ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error)
	GetInspectionSessionByID(id int64) (*db.Sesion, error)
	SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error
	GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error)
	GetLastSessionWithFilter(agente string, proyectoID *int64, cwd string) (*db.Sesion, error)
	ListAgents() ([]*db.Agente, error)
	ListPendingProposals(agente string) ([]*db.Propuesta, error)
	ListRules(rol string) ([]*db.Regla, error)
	ListSkills(rol string) ([]*db.Skill, error)
	GetWorkflow(rol, nombre string) (*db.Workflow, error)
	ListWorkflows(rol string) ([]*db.Workflow, error)
}

type Service struct {
	store Store
}

type StartResult struct {
	Agente               string
	SesionID             int64
	Rol                  string
	PropuestasPendientes []*db.Propuesta
	Reglas               []*db.Regla
	Skills               []*db.Skill
	WorkflowPasos        []string
}

type FinishResult struct {
	Agente        string
	Rol           string
	WorkflowPasos []string
}

type BriefingResult struct {
	Agent                *db.Agente
	PropuestasPendientes []*db.Propuesta
	Reglas               []*db.Regla
	Skills               []*db.Skill
	Workflows            []*db.Workflow
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Start(agente string, nuevoCodex bool) (*StartResult, error) {
	agente = strings.TrimSpace(agente)
	if nuevoCodex {
		nombre, err := s.store.RegisterCodex()
		if err != nil {
			return nil, err
		}
		agente = nombre
	}
	sesionID, err := s.store.StartSession(agente)
	if err != nil {
		return nil, err
	}
	result := &StartResult{Agente: agente, SesionID: sesionID}

	agentes, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	for _, item := range agentes {
		if item.Nombre == agente {
			result.Rol = item.Rol
			break
		}
	}
	if result.Rol == "" || result.Rol == "admin" {
		return result, nil
	}
	if pendientes, err := s.store.ListPendingProposals(agente); err == nil {
		result.PropuestasPendientes = pendientes
	}
	if reglas, err := s.store.ListRules(result.Rol); err == nil {
		result.Reglas = reglas
	}
	if skills, err := s.store.ListSkills(result.Rol); err == nil {
		result.Skills = skills
	}
	if workflow, err := s.store.GetWorkflow(result.Rol, "inicio-sesion"); err == nil {
		result.WorkflowPasos = parseWorkflowSteps(workflow.Pasos)
	}
	return result, nil
}

func (s *Service) Finish(agente string) (*FinishResult, error) {
	agente = strings.TrimSpace(agente)
	result := &FinishResult{Agente: agente}
	agentes, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	for _, item := range agentes {
		if item.Nombre == agente {
			result.Rol = item.Rol
			break
		}
	}
	if result.Rol != "" && result.Rol != "admin" {
		if workflow, err := s.store.GetWorkflow(result.Rol, "fin-sesion"); err == nil {
			result.WorkflowPasos = parseWorkflowSteps(workflow.Pasos)
		}
	}
	if err := s.store.FinishSession(agente); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) ListAgents() ([]*db.Agente, error) {
	return s.store.ListAgents()
}

func (s *Service) ResolveProjectID(ref string) (*int64, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, nil
	}
	proyecto, err := s.store.GetProject(ref)
	if err != nil {
		return nil, err
	}
	return &proyecto.ID, nil
}

func (s *Service) ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	return s.store.ListInspectionSessions(filtro)
}

func (s *Service) GetInspectionSession(id int64) (*db.Sesion, error) {
	return s.store.GetInspectionSessionByID(id)
}

func (s *Service) SaveActiveSession(agente, proyectoRef string, upd db.SesionUpdate) (*db.Sesion, error) {
	proyectoID, err := s.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveActiveSession(strings.TrimSpace(agente), proyectoID, upd); err != nil {
		return nil, err
	}
	return s.store.GetActiveSession(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) Continue(agente, proyectoRef, cwd string) (*db.Sesion, error) {
	proyectoID, err := s.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, err
	}
	return s.store.GetLastSessionWithFilter(strings.TrimSpace(agente), proyectoID, strings.TrimSpace(cwd))
}

func (s *Service) BuildBriefing(agente string) (*BriefingResult, error) {
	agente = strings.TrimSpace(agente)
	agentes, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	var actual *db.Agente
	for _, item := range agentes {
		if item.Nombre == agente {
			actual = item
			break
		}
	}
	if actual == nil {
		return nil, fmt.Errorf("agente no encontrado: %s", agente)
	}
	result := &BriefingResult{Agent: actual}
	if actual.Rol == "" || actual.Rol == "admin" {
		return result, nil
	}
	if pendientes, err := s.store.ListPendingProposals(actual.Nombre); err == nil {
		result.PropuestasPendientes = pendientes
	}
	if reglas, err := s.store.ListRules(actual.Rol); err == nil {
		result.Reglas = reglas
	}
	if skills, err := s.store.ListSkills(actual.Rol); err == nil {
		result.Skills = skills
	}
	if workflows, err := s.store.ListWorkflows(actual.Rol); err == nil {
		result.Workflows = workflows
	}
	return result, nil
}

func parseWorkflowSteps(raw string) []string {
	var pasos []string
	if err := json.Unmarshal([]byte(raw), &pasos); err != nil {
		return nil
	}
	return pasos
}

type Repository struct{}

func (Repository) RegisterCodex() (string, error) {
	return db.RegistrarCodex()
}

func (Repository) StartSession(agente string) (int64, error) {
	return db.IniciarSesion(agente)
}

func (Repository) FinishSession(agente string) error {
	return db.FinSesion(agente)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	return db.ListarSesionesInspeccion(filtro)
}

func (Repository) GetInspectionSessionByID(id int64) (*db.Sesion, error) {
	return db.GetSesionInspeccionByID(id)
}

func (Repository) SaveActiveSession(agente string, proyectoID *int64, upd db.SesionUpdate) error {
	return db.GuardarSesionActiva(agente, proyectoID, upd)
}

func (Repository) GetActiveSession(agente string, proyectoID *int64) (*db.Sesion, error) {
	return db.GetSesionActiva(agente, proyectoID)
}

func (Repository) GetLastSessionWithFilter(agente string, proyectoID *int64, cwd string) (*db.Sesion, error) {
	return db.ObtenerUltimaSesionConFiltro(agente, proyectoID, cwd)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}

func (Repository) ListPendingProposals(agente string) ([]*db.Propuesta, error) {
	return db.PropuestasPendientesVoto(agente)
}

func (Repository) ListRules(rol string) ([]*db.Regla, error) {
	return db.GetReglasAgente(rol)
}

func (Repository) ListSkills(rol string) ([]*db.Skill, error) {
	return db.GetSkillsAgente(rol)
}

func (Repository) GetWorkflow(rol, nombre string) (*db.Workflow, error) {
	return db.GetWorkflow(rol, nombre)
}

func (Repository) ListWorkflows(rol string) ([]*db.Workflow, error) {
	return db.GetWorkflowsAgente(rol)
}
