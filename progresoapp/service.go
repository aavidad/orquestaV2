package progresoapp

import (
	"strings"

	"orquesta/db"
)

type Store interface {
	CalculateProjectSummary(proyecto string) (*db.ResumenProgresoProyecto, error)
	ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error)
	RegisterProjectPhase(fase *db.FaseProyecto) (int64, error)
	GetProjectPhase(id int64) (*db.FaseProyecto, error)
	UpdateProjectPhase(fase *db.FaseProyecto) error
	RegisterTaskProgress(avance *db.AvanceTarea) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type RegisterPhaseInput struct {
	Proyecto    string
	Nombre      string
	Descripcion string
	Orden       int64
	Peso        float64
	Estado      string
}

type UpdatePhaseInput struct {
	ID          int64
	Proyecto    *string
	Nombre      *string
	Descripcion *string
	Orden       *int64
	Peso        *float64
	Estado      *string
}

type RegisterTaskProgressInput struct {
	TareaID        int64
	Proyecto       string
	FaseID         *int64
	ProgresoPct    float64
	ActualizadoPor string
}

func (s *Service) GetSummary(proyecto string) (*db.ResumenProgresoProyecto, error) {
	return s.store.CalculateProjectSummary(strings.TrimSpace(proyecto))
}

func (s *Service) ListPhases(proyecto string) ([]*db.FaseProyecto, error) {
	return s.store.ListProjectPhases(strings.TrimSpace(proyecto))
}

func (s *Service) RegisterPhase(input RegisterPhaseInput) (int64, *db.FaseProyecto, error) {
	id, err := s.store.RegisterProjectPhase(&db.FaseProyecto{
		Proyecto:    strings.TrimSpace(input.Proyecto),
		Nombre:      strings.TrimSpace(input.Nombre),
		Descripcion: strings.TrimSpace(input.Descripcion),
		Orden:       input.Orden,
		Peso:        input.Peso,
		Estado:      strings.TrimSpace(input.Estado),
	})
	if err != nil {
		return 0, nil, err
	}
	fase, err := s.store.GetProjectPhase(id)
	if err != nil {
		return 0, nil, err
	}
	return id, fase, nil
}

func (s *Service) UpdatePhase(input UpdatePhaseInput) (*db.FaseProyecto, error) {
	fase, err := s.store.GetProjectPhase(input.ID)
	if err != nil {
		return nil, err
	}
	if input.Proyecto != nil {
		fase.Proyecto = strings.TrimSpace(*input.Proyecto)
	}
	if input.Nombre != nil {
		fase.Nombre = strings.TrimSpace(*input.Nombre)
	}
	if input.Descripcion != nil {
		fase.Descripcion = strings.TrimSpace(*input.Descripcion)
	}
	if input.Orden != nil {
		fase.Orden = *input.Orden
	}
	if input.Peso != nil {
		fase.Peso = *input.Peso
	}
	if input.Estado != nil {
		fase.Estado = strings.TrimSpace(*input.Estado)
	}
	if err := s.store.UpdateProjectPhase(fase); err != nil {
		return nil, err
	}
	return fase, nil
}

func (s *Service) RegisterTaskProgress(input RegisterTaskProgressInput) error {
	return s.store.RegisterTaskProgress(&db.AvanceTarea{
		TareaID:        input.TareaID,
		Proyecto:       strings.TrimSpace(input.Proyecto),
		FaseID:         input.FaseID,
		ProgresoPct:    input.ProgresoPct,
		ActualizadoPor: strings.TrimSpace(input.ActualizadoPor),
	})
}

type Repository struct{}

func (Repository) CalculateProjectSummary(proyecto string) (*db.ResumenProgresoProyecto, error) {
	return db.CalcularResumenProgresoProyecto(proyecto)
}

func (Repository) ListProjectPhases(proyecto string) ([]*db.FaseProyecto, error) {
	return db.ListarFasesProyecto(proyecto)
}

func (Repository) RegisterProjectPhase(fase *db.FaseProyecto) (int64, error) {
	return db.RegistrarFaseProyecto(fase)
}

func (Repository) GetProjectPhase(id int64) (*db.FaseProyecto, error) {
	return db.GetFaseProyecto(id)
}

func (Repository) UpdateProjectPhase(fase *db.FaseProyecto) error {
	return db.ActualizarFaseProyecto(fase)
}

func (Repository) RegisterTaskProgress(avance *db.AvanceTarea) error {
	return db.RegistrarAvanceTarea(avance)
}
