package supervisionapp

import (
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

type Store interface {
	GetProject(ref string) (*db.Proyecto, error)
	GetProjectAutonomy(proyectoID int64) (*db.ProyectoAutonomia, error)
	ListProjectAutonomy(enabled *bool) ([]*db.ProyectoAutonomia, error)
	UpsertProjectAutonomy(item *db.ProyectoAutonomia) (int64, error)
	RegisterAutonomyCycle(item *db.AutonomiaCiclo) (int64, error)
	ListAutonomyCycles(filter db.FiltroAutonomiaCiclos) ([]*db.AutonomiaCiclo, error)
	MarkProjectAutonomySupervised(proyectoID int64, when time.Time) error
	MarkProjectAutonomyReviewed(proyectoID int64, when time.Time) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Repository envuelve las funciones db compatibles que este paquete espera.
//
// Contrato esperado hoy:
// - db.GetProyecto
// - db.GetProyectoAutonomia
// - db.ListarProyectosAutonomia
// - db.UpsertProyectoAutonomia
// - db.RegistrarAutonomiaCiclo
// - db.ListarAutonomiaCiclos
// - db.MarcarProyectoAutonomiaSupervisado
// - db.MarcarProyectoAutonomiaRevisado
//
// Si los nombres de db cambian, solo hace falta adaptar este wrapper.
type Repository struct {
	getProject                  func(string) (*db.Proyecto, error)
	getProjectAutonomy          func(int64) (*db.ProyectoAutonomia, error)
	listProjectAutonomy         func(*bool) ([]*db.ProyectoAutonomia, error)
	upsertProjectAutonomy       func(*db.ProyectoAutonomia) (int64, error)
	registerAutonomyCycle       func(*db.AutonomiaCiclo) (int64, error)
	listAutonomyCycles          func(db.FiltroAutonomiaCiclos) ([]*db.AutonomiaCiclo, error)
	markProjectAutonomyReviewed func(int64, time.Time) error
	markProjectAutonomySeen     func(int64, time.Time) error
}

func NewRepository() Repository {
	return Repository{
		getProject:                  db.GetProyecto,
		getProjectAutonomy:          db.GetProyectoAutonomia,
		listProjectAutonomy:         db.ListarProyectosAutonomia,
		upsertProjectAutonomy:       db.UpsertProyectoAutonomia,
		registerAutonomyCycle:       db.RegistrarAutonomiaCiclo,
		listAutonomyCycles:          db.ListarAutonomiaCiclos,
		markProjectAutonomySeen:     db.MarcarProyectoAutonomiaSupervisado,
		markProjectAutonomyReviewed: db.MarcarProyectoAutonomiaRevisado,
	}
}

func (r Repository) withDefaults() Repository {
	if r.getProject == nil {
		defaults := NewRepository()
		if r.getProject == nil {
			r.getProject = defaults.getProject
		}
		if r.getProjectAutonomy == nil {
			r.getProjectAutonomy = defaults.getProjectAutonomy
		}
		if r.listProjectAutonomy == nil {
			r.listProjectAutonomy = defaults.listProjectAutonomy
		}
		if r.upsertProjectAutonomy == nil {
			r.upsertProjectAutonomy = defaults.upsertProjectAutonomy
		}
		if r.registerAutonomyCycle == nil {
			r.registerAutonomyCycle = defaults.registerAutonomyCycle
		}
		if r.listAutonomyCycles == nil {
			r.listAutonomyCycles = defaults.listAutonomyCycles
		}
		if r.markProjectAutonomySeen == nil {
			r.markProjectAutonomySeen = defaults.markProjectAutonomySeen
		}
		if r.markProjectAutonomyReviewed == nil {
			r.markProjectAutonomyReviewed = defaults.markProjectAutonomyReviewed
		}
	}
	return r
}

type PolicyInput struct {
	Enabled              bool
	ObjetivoGeneral      string
	DefinitionOfDoneJSON string
	MaxWorkers           int
	SupervisorAgente     string
	ReviewerAgente       string
	ReserveReviewer      bool
	ReserveSupervisor    bool
	ReviewRequired       bool
	AutoCreateTasks      bool
	AutoCloseProject     bool
	EstadoAutonomia      db.EstadoAutonomiaProyecto
}

type CycleInput struct {
	Kind         string
	Agente       string
	SesionID     *int64
	RuntimeID    *int64
	InputJSON    string
	DecisionJSON string
	Resultado    string
}

func (s *Service) GetProjectPolicy(projectRef string) (*db.ProyectoAutonomia, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	return s.store.GetProjectAutonomy(proyecto.ID)
}

func (s *Service) ListEnabledPolicies() ([]*db.ProyectoAutonomia, error) {
	enabled := true
	return s.store.ListProjectAutonomy(&enabled)
}

func (s *Service) UpsertProjectPolicy(projectRef string, in PolicyInput) (*db.ProyectoAutonomia, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	item := &db.ProyectoAutonomia{
		ProyectoID:           proyecto.ID,
		Enabled:              in.Enabled,
		ObjetivoGeneral:      strings.TrimSpace(in.ObjetivoGeneral),
		DefinitionOfDoneJSON: strings.TrimSpace(in.DefinitionOfDoneJSON),
		MaxWorkers:           in.MaxWorkers,
		SupervisorAgente:     strings.TrimSpace(in.SupervisorAgente),
		ReviewerAgente:       strings.TrimSpace(in.ReviewerAgente),
		ReserveReviewer:      in.ReserveReviewer,
		ReserveSupervisor:    in.ReserveSupervisor,
		ReviewRequired:       in.ReviewRequired,
		AutoCreateTasks:      in.AutoCreateTasks,
		AutoCloseProject:     in.AutoCloseProject,
		EstadoAutonomia:      in.EstadoAutonomia,
	}
	if _, err := s.store.UpsertProjectAutonomy(item); err != nil {
		return nil, err
	}
	return s.store.GetProjectAutonomy(proyecto.ID)
}

func (s *Service) RegisterCycle(projectRef string, in CycleInput) (*db.AutonomiaCiclo, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Kind) == "" {
		return nil, fmt.Errorf("kind obligatorio")
	}
	item := &db.AutonomiaCiclo{
		ProyectoID:   proyecto.ID,
		Kind:         strings.TrimSpace(in.Kind),
		Agente:       strings.TrimSpace(in.Agente),
		SesionID:     in.SesionID,
		RuntimeID:    in.RuntimeID,
		InputJSON:    strings.TrimSpace(in.InputJSON),
		DecisionJSON: strings.TrimSpace(in.DecisionJSON),
		Resultado:    strings.TrimSpace(in.Resultado),
	}
	id, err := s.store.RegisterAutonomyCycle(item)
	if err != nil {
		return nil, err
	}
	list, err := s.store.ListAutonomyCycles(db.FiltroAutonomiaCiclos{ProyectoID: &proyecto.ID, Limit: 1})
	if err == nil && len(list) > 0 && list[0].ID == id {
		return list[0], nil
	}
	return item, nil
}

func (s *Service) ListCycles(projectRef string, kind *string, limit int) ([]*db.AutonomiaCiclo, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	return s.store.ListAutonomyCycles(db.FiltroAutonomiaCiclos{
		ProyectoID: &proyecto.ID,
		Kind:       kind,
		Limit:      limit,
	})
}

func (s *Service) MarkSupervised(projectRef string, when time.Time) error {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return err
	}
	return s.store.MarkProjectAutonomySupervised(proyecto.ID, when)
}

func (s *Service) MarkReviewed(projectRef string, when time.Time) error {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return err
	}
	return s.store.MarkProjectAutonomyReviewed(proyecto.ID, when)
}

func (r Repository) GetProject(ref string) (*db.Proyecto, error) {
	repo := r.withDefaults()
	return repo.getProject(ref)
}

func (r Repository) GetProjectAutonomy(proyectoID int64) (*db.ProyectoAutonomia, error) {
	repo := r.withDefaults()
	return repo.getProjectAutonomy(proyectoID)
}

func (r Repository) ListProjectAutonomy(enabled *bool) ([]*db.ProyectoAutonomia, error) {
	repo := r.withDefaults()
	return repo.listProjectAutonomy(enabled)
}

func (r Repository) UpsertProjectAutonomy(item *db.ProyectoAutonomia) (int64, error) {
	repo := r.withDefaults()
	return repo.upsertProjectAutonomy(item)
}

func (r Repository) RegisterAutonomyCycle(item *db.AutonomiaCiclo) (int64, error) {
	repo := r.withDefaults()
	return repo.registerAutonomyCycle(item)
}

func (r Repository) ListAutonomyCycles(filter db.FiltroAutonomiaCiclos) ([]*db.AutonomiaCiclo, error) {
	repo := r.withDefaults()
	return repo.listAutonomyCycles(filter)
}

func (r Repository) MarkProjectAutonomySupervised(proyectoID int64, when time.Time) error {
	repo := r.withDefaults()
	return repo.markProjectAutonomySeen(proyectoID, when)
}

func (r Repository) MarkProjectAutonomyReviewed(proyectoID int64, when time.Time) error {
	repo := r.withDefaults()
	return repo.markProjectAutonomyReviewed(proyectoID, when)
}
