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
