package gobernanzaapp

import (
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/db"
)

type Store interface {
	ListRules(tipoAgente string, activa *bool) ([]*db.Regla, error)
	SaveRule(r *db.Regla) (int64, error)
	ListSkills(tipoAgente string, activa *bool) ([]*db.Skill, error)
	SaveSkill(s *db.Skill) (int64, error)
	ListWorkflows(tipoAgente string, activo *bool) ([]*db.Workflow, error)
	SaveWorkflow(w *db.Workflow) (int64, error)
	ListOverrides(scopeTipo, scopeRef, tipoAgente, entidad string) ([]*db.GovernanceOverride, error)
	SaveOverride(actor string, item *db.GovernanceOverride) (int64, error)
	ResolveCatalogForContext(tipoAgente string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error)
	GetProject(ref string) (*db.Proyecto, error)
	GetAgent(nombre string) (*db.Agente, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type SaveRuleInput struct {
	TipoAgente  string
	Categoria   string
	Titulo      string
	Descripcion string
	Activa      bool
}

type SaveSkillInput struct {
	TipoAgente         string
	Nombre             string
	Descripcion        string
	CuandoUsar         string
	Escenario          string
	Prioridad          int
	AliasesJSON        string
	HerramientasJSON   string
	Origen             string
	NivelRiesgo        string
	RequiereAprobacion bool
	Activa             bool
}

type SaveWorkflowInput struct {
	TipoAgente  string
	Nombre      string
	Descripcion string
	Pasos       []string
	Activo      bool
}

type SaveOverrideInput struct {
	Actor      string
	TipoAgente string
	ScopeTipo  string
	ScopeRef   string
	Entidad    string
	EntidadID  int64
	Accion     string
}

func (s *Service) ListRules(tipoAgente string, activa *bool) ([]*db.Regla, error) {
	return s.store.ListRules(strings.TrimSpace(tipoAgente), activa)
}

func (s *Service) SaveRule(input SaveRuleInput) (int64, error) {
	return s.store.SaveRule(&db.Regla{
		TipoAgente:  strings.TrimSpace(input.TipoAgente),
		Categoria:   strings.TrimSpace(input.Categoria),
		Titulo:      strings.TrimSpace(input.Titulo),
		Descripcion: strings.TrimSpace(input.Descripcion),
		Activa:      input.Activa,
	})
}

func (s *Service) ListSkills(tipoAgente string, activa *bool) ([]*db.Skill, error) {
	return s.store.ListSkills(strings.TrimSpace(tipoAgente), activa)
}

func (s *Service) SaveSkill(input SaveSkillInput) (int64, error) {
	return s.store.SaveSkill(&db.Skill{
		TipoAgente:         strings.TrimSpace(input.TipoAgente),
		Nombre:             strings.TrimSpace(input.Nombre),
		Descripcion:        strings.TrimSpace(input.Descripcion),
		CuandoUsar:         strings.TrimSpace(input.CuandoUsar),
		Escenario:          strings.TrimSpace(input.Escenario),
		Prioridad:          input.Prioridad,
		AliasesJSON:        strings.TrimSpace(input.AliasesJSON),
		HerramientasJSON:   strings.TrimSpace(input.HerramientasJSON),
		Origen:             strings.TrimSpace(input.Origen),
		NivelRiesgo:        strings.TrimSpace(input.NivelRiesgo),
		RequiereAprobacion: input.RequiereAprobacion,
		Activa:             input.Activa,
	})
}

func (s *Service) ListWorkflows(tipoAgente string, activo *bool) ([]*db.Workflow, error) {
	return s.store.ListWorkflows(strings.TrimSpace(tipoAgente), activo)
}

func (s *Service) SaveWorkflow(input SaveWorkflowInput) (int64, error) {
	if len(input.Pasos) == 0 {
		return 0, fmt.Errorf("pasos obligatorios")
	}
	pasosJSON, err := json.Marshal(input.Pasos)
	if err != nil {
		return 0, err
	}
	return s.store.SaveWorkflow(&db.Workflow{
		TipoAgente:  strings.TrimSpace(input.TipoAgente),
		Nombre:      strings.TrimSpace(input.Nombre),
		Descripcion: strings.TrimSpace(input.Descripcion),
		Pasos:       string(pasosJSON),
		Activo:      input.Activo,
	})
}

func (s *Service) ListOverrides(scopeTipo, scopeRef, tipoAgente, agente, entidad string) ([]*db.GovernanceOverride, error) {
	rol, err := s.resolveTipoAgente(tipoAgente, agente)
	if err != nil {
		return nil, err
	}
	return s.store.ListOverrides(strings.TrimSpace(scopeTipo), strings.TrimSpace(scopeRef), rol, strings.TrimSpace(entidad))
}

func (s *Service) SaveOverride(input SaveOverrideInput) (int64, error) {
	rol, err := s.resolveTipoAgente(input.TipoAgente, "")
	if err != nil {
		return 0, err
	}
	return s.store.SaveOverride(strings.TrimSpace(input.Actor), &db.GovernanceOverride{
		TipoAgente: rol,
		ScopeTipo:  strings.TrimSpace(input.ScopeTipo),
		ScopeRef:   strings.TrimSpace(input.ScopeRef),
		Entidad:    strings.TrimSpace(input.Entidad),
		EntidadID:  input.EntidadID,
		Accion:     strings.TrimSpace(input.Accion),
	})
}

func (s *Service) ResolveCatalogForContext(tipoAgente, proyectoRef, agente string) (*db.GovernanceCatalog, error) {
	rol, err := s.resolveTipoAgente(tipoAgente, agente)
	if err != nil {
		return nil, err
	}
	proyectoID, err := s.resolveProyectoID(proyectoRef)
	if err != nil {
		return nil, err
	}
	return s.store.ResolveCatalogForContext(rol, proyectoID, strings.TrimSpace(agente))
}

func (s *Service) resolveTipoAgente(tipoAgente, agente string) (string, error) {
	if rol := strings.TrimSpace(tipoAgente); rol != "" {
		return rol, nil
	}
	if nombre := strings.TrimSpace(agente); nombre != "" {
		item, err := s.store.GetAgent(nombre)
		if err != nil {
			return "", err
		}
		if item == nil || strings.TrimSpace(item.Rol) == "" {
			return "", fmt.Errorf("agente '%s' no encontrado", nombre)
		}
		return strings.TrimSpace(item.Rol), nil
	}
	return "", fmt.Errorf("debes indicar tipo_agente o agente")
}

func (s *Service) resolveProyectoID(ref string) (*int64, error) {
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

type Repository struct{}

func (Repository) ListRules(tipoAgente string, activa *bool) ([]*db.Regla, error) {
	return db.ListarReglas(tipoAgente, activa)
}

func (Repository) SaveRule(r *db.Regla) (int64, error) {
	return db.CrearRegla("", r)
}

func (Repository) ListSkills(tipoAgente string, activa *bool) ([]*db.Skill, error) {
	return db.ListarSkills(tipoAgente, activa)
}

func (Repository) SaveSkill(s *db.Skill) (int64, error) {
	return db.CrearSkill("", s)
}

func (Repository) ListWorkflows(tipoAgente string, activo *bool) ([]*db.Workflow, error) {
	return db.ListarWorkflows(tipoAgente, activo)
}

func (Repository) SaveWorkflow(w *db.Workflow) (int64, error) {
	return db.CrearWorkflow("", w)
}

func (Repository) ListOverrides(scopeTipo, scopeRef, tipoAgente, entidad string) ([]*db.GovernanceOverride, error) {
	return db.ListarGovernanceOverrides(scopeTipo, scopeRef, tipoAgente, entidad)
}

func (Repository) SaveOverride(actor string, item *db.GovernanceOverride) (int64, error) {
	return db.GuardarGovernanceOverride(actor, item)
}

func (Repository) ResolveCatalogForContext(tipoAgente string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error) {
	return db.ResolveGovernanceCatalogForContext(tipoAgente, proyectoID, agente)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) GetAgent(nombre string) (*db.Agente, error) {
	return db.GetAgente(nombre)
}
