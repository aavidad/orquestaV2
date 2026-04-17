package supervisionapp

import (
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

type EstadoAutonomiaProyecto = db.EstadoAutonomiaProyecto

const (
	AutonomiaProyectoActiva          = db.AutonomiaProyectoActiva
	AutonomiaProyectoEsperandoReview = db.AutonomiaProyectoEsperandoReview
	AutonomiaProyectoEsperandoHumano = db.AutonomiaProyectoEsperandoHumano
	AutonomiaProyectoCerrando        = db.AutonomiaProyectoCerrando
	AutonomiaProyectoCerrado         = db.AutonomiaProyectoCerrado
)

type Store interface {
	GetProject(ref string) (*db.Proyecto, error)
	GetProjectAutonomy(proyectoID int64) (*Policy, error)
	ListProjectAutonomy(enabled *bool) ([]*Policy, error)
	UpsertProjectAutonomy(item *Policy) (int64, error)
	RegisterAutonomyCycle(item *Cycle) (int64, error)
	ListAutonomyCycles(filter CycleFilter) ([]*Cycle, error)
	MarkProjectAutonomySupervised(proyectoID int64, when time.Time) error
	MarkProjectAutonomyReviewed(proyectoID int64, when time.Time) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
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
	EstadoAutonomia      EstadoAutonomiaProyecto
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

type Policy struct {
	ProyectoID           int64                   `json:"proyecto_id"`
	ProyectoSlug         string                  `json:"proyecto_slug,omitempty"`
	Enabled              bool                    `json:"enabled"`
	ObjetivoGeneral      string                  `json:"objetivo_general"`
	DefinitionOfDoneJSON string                  `json:"definition_of_done_json"`
	MaxWorkers           int                     `json:"max_workers"`
	SupervisorAgente     string                  `json:"supervisor_agente"`
	ReviewerAgente       string                  `json:"reviewer_agente"`
	ReserveReviewer      bool                    `json:"reserve_reviewer"`
	ReserveSupervisor    bool                    `json:"reserve_supervisor"`
	ReviewRequired       bool                    `json:"review_required"`
	AutoCreateTasks      bool                    `json:"auto_create_tasks"`
	AutoCloseProject     bool                    `json:"auto_close_project"`
	EstadoAutonomia      EstadoAutonomiaProyecto `json:"estado_autonomia"`
	LastSupervisionAt    *time.Time              `json:"last_supervision_at,omitempty"`
	LastReviewAt         *time.Time              `json:"last_review_at,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

type CycleFilter struct {
	ProyectoID *int64
	Kind       *string
	Agente     *string
	Limit      int
}

type Cycle struct {
	ID           int64     `json:"id"`
	ProyectoID   int64     `json:"proyecto_id"`
	ProyectoSlug string    `json:"proyecto_slug,omitempty"`
	Kind         string    `json:"kind"`
	Agente       string    `json:"agente"`
	SesionID     *int64    `json:"sesion_id,omitempty"`
	RuntimeID    *int64    `json:"runtime_id,omitempty"`
	InputJSON    string    `json:"input_json"`
	DecisionJSON string    `json:"decision_json"`
	Resultado    string    `json:"resultado"`
	CreatedAt    time.Time `json:"created_at"`
}

func ParseEstadoAutonomiaProyecto(raw string) (EstadoAutonomiaProyecto, error) {
	estado := EstadoAutonomiaProyecto(strings.TrimSpace(raw))
	if estado == "" {
		return "", nil
	}
	switch estado {
	case AutonomiaProyectoActiva,
		AutonomiaProyectoEsperandoReview,
		AutonomiaProyectoEsperandoHumano,
		AutonomiaProyectoCerrando,
		AutonomiaProyectoCerrado:
		return estado, nil
	default:
		return "", fmt.Errorf("estado_autonomia inválido: %q (valores válidos: %q, %q, %q, %q, %q)",
			raw,
			AutonomiaProyectoActiva,
			AutonomiaProyectoEsperandoReview,
			AutonomiaProyectoEsperandoHumano,
			AutonomiaProyectoCerrando,
			AutonomiaProyectoCerrado,
		)
	}
}

func (s *Service) GetProjectPolicy(projectRef string) (*Policy, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	return s.store.GetProjectAutonomy(proyecto.ID)
}

func (s *Service) ListEnabledPolicies() ([]*Policy, error) {
	enabled := true
	return s.store.ListProjectAutonomy(&enabled)
}

func (s *Service) PersistPolicyState(policy *Policy, estado EstadoAutonomiaProyecto) error {
	if policy == nil || policy.EstadoAutonomia == estado {
		return nil
	}
	estadoNormalizado, err := ParseEstadoAutonomiaProyecto(string(estado))
	if err != nil {
		return err
	}
	cp := *policy
	cp.EstadoAutonomia = estadoNormalizado
	if _, err := s.store.UpsertProjectAutonomy(&cp); err != nil {
		return err
	}
	policy.EstadoAutonomia = estadoNormalizado
	return nil
}

func (s *Service) UpsertProjectPolicy(projectRef string, in PolicyInput) (*Policy, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	estadoAutonomia, err := ParseEstadoAutonomiaProyecto(string(in.EstadoAutonomia))
	if err != nil {
		return nil, err
	}
	item := &Policy{
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
		EstadoAutonomia:      estadoAutonomia,
	}
	if _, err := s.store.UpsertProjectAutonomy(item); err != nil {
		return nil, err
	}
	return s.store.GetProjectAutonomy(proyecto.ID)
}

func (s *Service) RegisterCycle(projectRef string, in CycleInput) (*Cycle, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Kind) == "" {
		return nil, fmt.Errorf("kind obligatorio")
	}
	item := &Cycle{
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
	filter := CycleFilter{ProyectoID: &proyecto.ID, Limit: 1}
	list, err := s.store.ListAutonomyCycles(filter)
	if err == nil && len(list) > 0 && list[0].ID == id {
		return list[0], nil
	}
	return item, nil
}

func (s *Service) ListCycles(projectRef string, kind *string, limit int) ([]*Cycle, error) {
	proyecto, err := s.store.GetProject(strings.TrimSpace(projectRef))
	if err != nil {
		return nil, err
	}
	return s.store.ListAutonomyCycles(CycleFilter{
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
