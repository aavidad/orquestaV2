/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package tareasapp

import (
	"fmt"
	"strings"

	"orquesta/db"
)

type Store interface {
	ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error)
	GetTask(id int64) (*db.Tarea, error)
	CreateTask(t *db.Tarea) (int64, error)
	TakeTask(id int64, agente string) error
	StartTask(id int64, agente string) error
	CompleteTask(id int64, agente, commit string) error
	BlockTask(id int64, agente, motivo string) error
	UnblockTask(id int64, agente, resolucion string) error
	AnnotateTask(id int64, agente, nota string) error
	MoveTaskToBacklog(id int64) error
	ReassignTask(id int64, nuevoAgente string) error
	GetProject(ref string) (*db.Proyecto, error)
	GetProposal(codigo string) (*db.Propuesta, error)
	ListAgents() ([]*db.Agente, error)
}

type Service struct {
	store Store
}

type CreateTaskInput struct {
	Titulo          string
	Descripcion     string
	Modulo          string
	Prioridad       db.PrioridadTarea
	CreadoPor       string
	Agente          string
	Proyecto        string
	PropuestaCodigo string
	Notas           string
}

type CleanActiveFrontInput struct {
	Agente   string
	Proyecto string
	KeepIDs  []int64
}

type CleanActiveFrontResult struct {
	Total    int     `json:"total"`
	MovedIDs []int64 `json:"moved_ids"`
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return s.store.ListTasks(filtro)
}

func (s *Service) Get(id int64) (*db.Tarea, error) {
	return s.store.GetTask(id)
}

func (s *Service) ListAgents() ([]*db.Agente, error) {
	return s.store.ListAgents()
}

func (s *Service) ResolveProposalID(codigo string) (*int64, error) {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return nil, nil
	}
	propuesta, err := s.store.GetProposal(codigo)
	if err != nil {
		return nil, err
	}
	return &propuesta.ID, nil
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
	if proyecto == nil {
		return nil, fmt.Errorf("proyecto no encontrado: %s", ref)
	}
	return &proyecto.ID, nil
}

func (s *Service) Create(input CreateTaskInput) (int64, error) {
	tarea := &db.Tarea{
		Titulo:      strings.TrimSpace(input.Titulo),
		Descripcion: strings.TrimSpace(input.Descripcion),
		Modulo:      strings.TrimSpace(input.Modulo),
		Prioridad:   input.Prioridad,
		CreadoPor:   strings.TrimSpace(input.CreadoPor),
		Notas:       input.Notas,
	}
	if tarea.CreadoPor == "" {
		tarea.CreadoPor = "alberto"
	}
	if tarea.Titulo == "" {
		return 0, fmt.Errorf("el titulo es obligatorio")
	}
	if propuestaID, err := s.ResolveProposalID(input.PropuestaCodigo); err != nil {
		return 0, err
	} else {
		tarea.PropuestaID = propuestaID
	}
	if proyectoID, err := s.ResolveProjectID(input.Proyecto); err != nil {
		return 0, err
	} else {
		tarea.ProyectoID = proyectoID
	}
	id, err := s.store.CreateTask(tarea)
	if err != nil {
		return 0, err
	}
	if agente := strings.TrimSpace(input.Agente); agente != "" {
		if err := s.store.TakeTask(id, agente); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Service) Take(id int64, agente string) error {
	tarea, err := s.store.GetTask(id)
	if err != nil {
		return err
	}
	if err := validateTaskDependencies(s.store, tarea); err != nil {
		return err
	}
	return s.store.TakeTask(id, strings.TrimSpace(agente))
}

func (s *Service) Start(id int64, agente string) error {
	tarea, err := s.store.GetTask(id)
	if err != nil {
		return err
	}
	if err := validateTaskDependencies(s.store, tarea); err != nil {
		return err
	}
	return s.store.StartTask(id, strings.TrimSpace(agente))
}

func (s *Service) Complete(id int64, agente, commit string) error {
	return s.store.CompleteTask(id, strings.TrimSpace(agente), strings.TrimSpace(commit))
}

func (s *Service) Block(id int64, agente, motivo string) error {
	return s.store.BlockTask(id, strings.TrimSpace(agente), strings.TrimSpace(motivo))
}

func (s *Service) Unblock(id int64, agente, resolucion string) error {
	return s.store.UnblockTask(id, strings.TrimSpace(agente), strings.TrimSpace(resolucion))
}

func (s *Service) Note(id int64, agente, nota string) error {
	return s.store.AnnotateTask(id, strings.TrimSpace(agente), strings.TrimSpace(nota))
}

func (s *Service) MoveToBacklog(id int64) error {
	return s.store.MoveTaskToBacklog(id)
}

func (s *Service) Reassign(id int64, nuevoAgente string) error {
	return s.store.ReassignTask(id, strings.TrimSpace(nuevoAgente))
}

func (s *Service) CleanActiveFront(input CleanActiveFrontInput) (*CleanActiveFrontResult, error) {
	agente := strings.TrimSpace(input.Agente)
	proyecto := strings.TrimSpace(input.Proyecto)
	if agente == "" && proyecto == "" {
		return nil, fmt.Errorf("agente o proyecto es obligatorio")
	}
	var proyectoID *int64
	if proyecto != "" {
		id, err := s.ResolveProjectID(proyecto)
		if err != nil {
			return nil, err
		}
		proyectoID = id
	}
	keep := map[int64]struct{}{}
	for _, id := range input.KeepIDs {
		if id > 0 {
			keep[id] = struct{}{}
		}
	}
	result := &CleanActiveFrontResult{MovedIDs: make([]int64, 0)}
	states := []db.EstadoTarea{
		db.EstadoLibre,
		db.EstadoAsignada,
		db.EstadoEnProgreso,
		db.EstadoBloqueada,
	}
	seen := map[int64]struct{}{}
	for _, state := range states {
		stateCopy := state
		filtro := db.FiltroTareas{Estado: &stateCopy}
		if agente != "" {
			filtro.Agente = &agente
		}
		if proyectoID != nil {
			filtro.ProyectoID = proyectoID
		}
		tareas, err := s.store.ListTasks(filtro)
		if err != nil {
			return nil, err
		}
		for _, tarea := range tareas {
			if tarea == nil || tarea.ID <= 0 {
				continue
			}
			if _, ok := seen[tarea.ID]; ok {
				continue
			}
			seen[tarea.ID] = struct{}{}
			if _, ok := keep[tarea.ID]; ok {
				continue
			}
			if err := s.store.MoveTaskToBacklog(tarea.ID); err != nil {
				return nil, err
			}
			result.MovedIDs = append(result.MovedIDs, tarea.ID)
		}
	}
	result.Total = len(result.MovedIDs)
	return result, nil
}
