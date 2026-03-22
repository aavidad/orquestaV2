package taskapp

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
	PropuestaCodigo string
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

func (s *Service) Create(input CreateTaskInput) (int64, error) {
	tarea := &db.Tarea{
		Titulo:      strings.TrimSpace(input.Titulo),
		Descripcion: strings.TrimSpace(input.Descripcion),
		Modulo:      strings.TrimSpace(input.Modulo),
		Prioridad:   input.Prioridad,
		CreadoPor:   strings.TrimSpace(input.CreadoPor),
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
	return s.store.TakeTask(id, strings.TrimSpace(agente))
}

func (s *Service) Start(id int64, agente string) error {
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

type Repository struct{}

func (Repository) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return db.ListarTareas(filtro)
}

func (Repository) GetTask(id int64) (*db.Tarea, error) {
	return db.GetTarea(id)
}

func (Repository) CreateTask(t *db.Tarea) (int64, error) {
	return db.CrearTarea(t)
}

func (Repository) TakeTask(id int64, agente string) error {
	return db.TomarTarea(id, agente)
}

func (Repository) StartTask(id int64, agente string) error {
	return db.IniciarTarea(id, agente)
}

func (Repository) CompleteTask(id int64, agente, commit string) error {
	return db.CompletarTarea(id, agente, commit)
}

func (Repository) BlockTask(id int64, agente, motivo string) error {
	return db.BloquearTarea(id, agente, motivo)
}

func (Repository) UnblockTask(id int64, agente, resolucion string) error {
	return db.DesbloquearTarea(id, agente, resolucion)
}

func (Repository) AnnotateTask(id int64, agente, nota string) error {
	return db.AnotarTarea(id, agente, nota)
}

func (Repository) MoveTaskToBacklog(id int64) error {
	return db.EnviarTareaABacklog(id)
}

func (Repository) ReassignTask(id int64, nuevoAgente string) error {
	return db.ReasignarTarea(id, nuevoAgente)
}

func (Repository) GetProposal(codigo string) (*db.Propuesta, error) {
	return db.GetPropuesta(codigo)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}
