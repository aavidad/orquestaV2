package sesionesapp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Service struct {
	store Store
}

type StartResult struct {
	Agente               string
	SesionID             int64
	Rol                  string
	PropuestasPendientes []*Propuesta
	Reglas               []*Regla
	Skills               []*Skill
	WorkflowPasos        []string
}

type FinishResult struct {
	Agente        string
	Rol           string
	WorkflowPasos []string
}

type BriefingResult struct {
	Agent                *Agente
	PropuestasPendientes []*Propuesta
	Reglas               []*Regla
	Skills               []*Skill
	Workflows            []*Workflow
}

type SessionBudgetResult struct {
	ID          int64
	Sesion      *Sesion
	Presupuesto *PresupuestoSesion
	Evaluacion  *EvaluacionPresupuesto
}

type StartContextInput struct {
	Agente            string
	NuevoCodex        bool
	ArranqueLimpio    bool
	Proyecto          string
	Conector          string
	CWD               string
	Herramienta       string
	ExternalSessionID string
	ResumePayload     string
	Resumen           string
	Branch            string
	Host              string
	PID               int64
}

type StartContextResult struct {
	Sesion               *Sesion
	SesionPrevia         *Sesion
	Rol                  string
	PropuestasPendientes []*Propuesta
	Reglas               []*Regla
	Skills               []*Skill
	Workflow             *Workflow
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
	if catalogo, err := s.store.ResolveGovernanceCatalogForContext(result.Rol, nil, agente); err == nil && catalogo != nil {
		result.Reglas = catalogo.Reglas
		result.Skills = catalogo.Skills
	}
	if workflow, err := s.store.ResolveGovernanceWorkflowForContext(result.Rol, nil, agente, "inicio-sesion"); err == nil {
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
		if workflow, err := s.store.ResolveGovernanceWorkflowForContext(result.Rol, nil, agente, "fin-sesion"); err == nil {
			result.WorkflowPasos = parseWorkflowSteps(workflow.Pasos)
		}
	}
	if err := s.store.FinishSession(agente); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) ListAgents() ([]*Agente, error) {
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

func (s *Service) GetProject(ref string) (*Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("proyecto obligatorio")
	}
	return s.store.GetProject(ref)
}

func (s *Service) ActivateAssignment(agente, proyectoRef, nota string) (*Proyecto, error) {
	proyectoRef = strings.TrimSpace(proyectoRef)
	if proyectoRef == "" {
		return nil, fmt.Errorf("proyecto obligatorio")
	}
	proyecto, err := s.store.GetProject(proyectoRef)
	if err != nil {
		return nil, err
	}
	if err := s.store.ActivateAssignment(strings.TrimSpace(agente), proyecto.ID, strings.TrimSpace(nota)); err != nil {
		return nil, err
	}
	return proyecto, nil
}

func (s *Service) ListInspectionSessions(filtro FiltroInspeccion) ([]*Sesion, error) {
	return s.store.ListInspectionSessions(filtro)
}

func (s *Service) GetInspectionSession(id int64) (*Sesion, error) {
	return s.store.GetInspectionSessionByID(id)
}

func (s *Service) SaveActiveSession(agente, proyectoRef string, upd SesionUpdate) (*Sesion, error) {
	proyectoID, err := s.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveActiveSession(strings.TrimSpace(agente), proyectoID, upd); err != nil {
		return nil, err
	}
	return s.store.GetActiveSession(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) Continue(agente, proyectoRef, cwd string) (*Sesion, error) {
	proyectoID, err := s.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, err
	}
	return s.store.GetLastSessionWithFilter(strings.TrimSpace(agente), proyectoID, strings.TrimSpace(cwd))
}

func (s *Service) GetLastSession(agente, proyectoRef string) (*Sesion, error) {
	proyectoID, err := s.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, err
	}
	return s.store.GetLastSession(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) GetActiveSession(agente, proyectoRef string) (*Sesion, error) {
	proyectoID, err := s.ResolveProjectID(proyectoRef)
	if err != nil {
		return nil, err
	}
	return s.store.GetActiveSession(strings.TrimSpace(agente), proyectoID)
}

func (s *Service) FindActiveSessionByExternalSessionID(externalSessionID string) (*Sesion, error) {
	externalSessionID = strings.TrimSpace(externalSessionID)
	if externalSessionID == "" {
		return nil, fmt.Errorf("external_session_id obligatorio")
	}
	activa := true
	sesiones, err := s.store.ListInspectionSessions(FiltroInspeccion{Activa: &activa, Limit: 200})
	if err != nil {
		return nil, err
	}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		if !sesion.Activa {
			continue
		}
		if strings.TrimSpace(sesion.ExternalSessionID) == externalSessionID {
			return sesion, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Service) ResolveBudgetSession(sesionID int64, agente string) (*Sesion, error) {
	if sesionID > 0 {
		return s.store.GetSessionByID(sesionID)
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, fmt.Errorf("debe indicar --sesion o --agente")
	}
	sesion, err := s.store.GetActiveSession(agente, nil)
	if err != nil {
		return nil, err
	}
	if sesion == nil {
		return nil, fmt.Errorf("el agente %s no tiene sesión activa", agente)
	}
	return sesion, nil
}

func (s *Service) GetBudget(sesionID int64, agente string) (*SessionBudgetResult, error) {
	sesion, err := s.ResolveBudgetSession(sesionID, agente)
	if err != nil {
		return nil, err
	}
	p, err := s.store.GetLatestSessionBudget(sesion.ID)
	if err != nil {
		return nil, err
	}
	ev, err := s.store.EvaluateSessionBudget(p)
	if err != nil {
		return nil, err
	}
	return &SessionBudgetResult{
		Sesion:      sesion,
		Presupuesto: p,
		Evaluacion:  ev,
	}, nil
}

func (s *Service) RegisterBudget(sesionID int64, agente string, p *PresupuestoSesion) (*SessionBudgetResult, error) {
	sesion, err := s.ResolveBudgetSession(sesionID, agente)
	if err != nil {
		return nil, err
	}
	p.SesionID = sesion.ID
	id, err := s.store.RegisterSessionBudget(p)
	if err != nil {
		return nil, err
	}
	latest, err := s.store.GetLatestSessionBudget(sesion.ID)
	if err != nil {
		return nil, err
	}
	ev, err := s.store.EvaluateSessionBudget(latest)
	if err != nil {
		return nil, err
	}
	return &SessionBudgetResult{
		ID:          id,
		Sesion:      sesion,
		Presupuesto: latest,
		Evaluacion:  ev,
	}, nil
}

func (s *Service) BuildBriefing(agente string) (*BriefingResult, error) {
	agente = strings.TrimSpace(agente)
	agentes, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	var actual *Agente
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
	if catalogo, err := s.store.ResolveGovernanceCatalogForContext(actual.Rol, nil, actual.Nombre); err == nil && catalogo != nil {
		result.Reglas = catalogo.Reglas
		result.Skills = catalogo.Skills
		result.Workflows = catalogo.Workflows
	}
	return result, nil
}

func (s *Service) StartContext(input StartContextInput) (*StartContextResult, error) {
	agente := strings.TrimSpace(input.Agente)
	if input.NuevoCodex {
		nombre, err := s.store.RegisterCodex()
		if err != nil {
			return nil, fmt.Errorf("registrando codex: %w", err)
		}
		agente = nombre
	}
	if agente == "" {
		return nil, fmt.Errorf("indica el nombre del agente o usa --nuevo-codex")
	}
	if input.ArranqueLimpio {
		input.ExternalSessionID = ""
		input.ResumePayload = ""
		input.Resumen = ""
	}

	var (
		proyectoID *int64
		previo     *Sesion
	)
	if proyectoRef := strings.TrimSpace(input.Proyecto); proyectoRef != "" {
		proyecto, err := s.store.GetProject(proyectoRef)
		if err != nil {
			return nil, err
		}
		proyectoID = &proyecto.ID
		if strings.TrimSpace(input.CWD) == "" {
			input.CWD = proyecto.RutaAbs
		}
		if err := s.store.ActivateAssignment(agente, proyecto.ID, "asignación automática al iniciar sesión"); err != nil {
			return nil, err
		}
		if !input.ArranqueLimpio {
			previo, err = s.store.GetLastSession(agente, &proyecto.ID)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
		}
	}

	var conectorID *int64
	if conectorRef := strings.TrimSpace(input.Conector); conectorRef != "" {
		conector, err := s.store.GetConnector(conectorRef)
		if err != nil {
			return nil, err
		}
		conectorID = &conector.ID
		if strings.TrimSpace(input.Herramienta) == "" {
			input.Herramienta = conector.Slug
		}
	}

	var pid *int64
	if input.PID > 0 {
		pid = &input.PID
	}
	sesion, err := s.store.StartSessionContext(SesionInicio{
		Agente:             agente,
		ConectorID:         conectorID,
		ProyectoID:         proyectoID,
		CWD:                input.CWD,
		Herramienta:        input.Herramienta,
		ExternalSessionID:  input.ExternalSessionID,
		ResumePayloadJSON:  input.ResumePayload,
		ResumenContinuidad: input.Resumen,
		Branch:             input.Branch,
		Host:               input.Host,
		PID:                pid,
	})
	if err != nil {
		return nil, err
	}

	result := &StartContextResult{
		Sesion:       sesion,
		SesionPrevia: previo,
	}
	agentes, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	for _, item := range agentes {
		if item != nil && item.Nombre == sesion.Agente {
			result.Rol = item.Rol
			break
		}
	}
	if result.Rol == "" || result.Rol == "admin" {
		return result, nil
	}
	if result.PropuestasPendientes, err = s.store.ListPendingProposals(sesion.Agente); err != nil {
		return nil, err
	}
	catalogo, err := s.store.ResolveGovernanceCatalogForContext(result.Rol, proyectoID, sesion.Agente)
	if err != nil {
		return nil, err
	}
	result.Reglas = catalogo.Reglas
	result.Skills = catalogo.Skills
	workflow, err := s.store.ResolveGovernanceWorkflowForContext(result.Rol, proyectoID, sesion.Agente, "inicio-sesion")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	result.Workflow = workflow
	return result, nil
}

func parseWorkflowSteps(raw string) []string {
	var pasos []string
	if err := json.Unmarshal([]byte(raw), &pasos); err != nil {
		return nil
	}
	return pasos
}
