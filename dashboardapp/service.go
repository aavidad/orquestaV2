package dashboardapp

import (
	"time"

	"orquesta/db"
)

type Store interface {
	ListAgents() ([]*db.Agente, error)
	CountTasksByState() (map[string]int, error)
	ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error)
	CountVotes(propuestaID int64) (int, int, int, int, error)
	ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error)
}

type Service struct {
	store Store
}

type ProposalSummary struct {
	Codigo     string
	Titulo     string
	Acuerdo    int
	Desacuerdo int
	Pendiente  int
}

type Summary struct {
	Agents      []*db.Agente
	TaskCounts  map[string]int
	TotalTasks  int
	DoneTasks   int
	PercentDone int
	OpenProps   []ProposalSummary
	ActiveTasks []*db.Tarea
	GeneratedAt time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) BuildSummary() (*Summary, error) {
	agents, err := s.store.ListAgents()
	if err != nil {
		return nil, err
	}
	counts, err := s.store.CountTasksByState()
	if err != nil {
		return nil, err
	}
	totalTasks := 0
	doneTasks := 0
	for estado, count := range counts {
		totalTasks += count
		if estado == string(db.EstadoCompletada) {
			doneTasks = count
		}
	}
	percentDone := 0
	if totalTasks > 0 {
		percentDone = doneTasks * 100 / totalTasks
	}

	openState := db.PropuestaAbierta
	proposals, err := s.store.ListProposals(&openState)
	if err != nil {
		return nil, err
	}
	openProps := make([]ProposalSummary, 0, len(proposals))
	for _, proposal := range proposals {
		acuerdo, desacuerdo, _, pendiente, err := s.store.CountVotes(proposal.ID)
		if err != nil {
			return nil, err
		}
		openProps = append(openProps, ProposalSummary{
			Codigo:     proposal.Codigo,
			Titulo:     proposal.Titulo,
			Acuerdo:    acuerdo,
			Desacuerdo: desacuerdo,
			Pendiente:  pendiente,
		})
	}

	activeState := db.EstadoEnProgreso
	activeTasks, err := s.store.ListTasks(db.FiltroTareas{Estado: &activeState})
	if err != nil {
		return nil, err
	}

	return &Summary{
		Agents:      agents,
		TaskCounts:  counts,
		TotalTasks:  totalTasks,
		DoneTasks:   doneTasks,
		PercentDone: percentDone,
		OpenProps:   openProps,
		ActiveTasks: activeTasks,
		GeneratedAt: time.Now(),
	}, nil
}

type Repository struct{}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}

func (Repository) CountTasksByState() (map[string]int, error) {
	return db.ContarTareasPorEstado()
}

func (Repository) ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error) {
	return db.ListarPropuestas(estado)
}

func (Repository) CountVotes(propuestaID int64) (int, int, int, int, error) {
	return db.ContarVotos(propuestaID)
}

func (Repository) ListTasks(filtro db.FiltroTareas) ([]*db.Tarea, error) {
	return db.ListarTareas(filtro)
}
