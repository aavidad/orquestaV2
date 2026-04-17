package progresoapp

import (
	"strings"
	"time"
)

type FaseProyecto struct {
	ID          int64     `json:"id"`
	Proyecto    string    `json:"proyecto"`
	Nombre      string    `json:"nombre"`
	Descripcion string    `json:"descripcion"`
	Orden       int64     `json:"orden"`
	Peso        float64   `json:"peso"`
	Estado      string    `json:"estado"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AvanceTarea struct {
	TareaID        int64     `json:"tarea_id"`
	Proyecto       string    `json:"proyecto"`
	FaseID         *int64    `json:"fase_id,omitempty"`
	ProgresoPct    float64   `json:"progreso_pct"`
	ActualizadoPor string    `json:"actualizado_por"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TareaProgresoDetalle struct {
	TareaID      int64      `json:"tarea_id,omitempty"`
	TareaTitulo  string     `json:"tarea_titulo,omitempty"`
	FaseID       *int64     `json:"fase_id,omitempty"`
	FaseNombre   string     `json:"fase_nombre"`
	Proyecto     string     `json:"proyecto"`
	ProgresoPct  float64    `json:"progreso_pct"`
	Actualizado  *time.Time `json:"actualizado,omitempty"`
	Manual       bool       `json:"manual"`
	EstadoTarea  string     `json:"estado_tarea,omitempty"`
	ModuloTarea  string     `json:"modulo_tarea,omitempty"`
	PrioridadRaw string     `json:"prioridad_tarea,omitempty"`
}

type FaseProgresoDetalle struct {
	Fase              *FaseProyecto           `json:"fase"`
	ProgresoPct       float64                 `json:"progreso_pct"`
	Tareas            []*TareaProgresoDetalle `json:"tareas"`
	TareasTotales     int                     `json:"tareas_totales"`
	TareasCompletadas int                     `json:"tareas_completadas"`
}

type ResumenProgresoProyecto struct {
	Proyecto          string                  `json:"proyecto"`
	ProgresoPct       float64                 `json:"progreso_pct"`
	TareasTotales     int                     `json:"tareas_totales"`
	TareasCompletadas int                     `json:"tareas_completadas"`
	Fases             []*FaseProgresoDetalle  `json:"fases"`
	TareasSinFase     []*TareaProgresoDetalle `json:"tareas_sin_fase"`
}

type Store interface {
	CalculateProjectSummary(proyecto string) (*ResumenProgresoProyecto, error)
	ListProjectPhases(proyecto string) ([]*FaseProyecto, error)
	RegisterProjectPhase(fase *FaseProyecto) (int64, error)
	GetProjectPhase(id int64) (*FaseProyecto, error)
	UpdateProjectPhase(fase *FaseProyecto) error
	RegisterTaskProgress(avance *AvanceTarea) error
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

func (s *Service) GetSummary(proyecto string) (*ResumenProgresoProyecto, error) {
	return s.store.CalculateProjectSummary(strings.TrimSpace(proyecto))
}

func (s *Service) ListPhases(proyecto string) ([]*FaseProyecto, error) {
	return s.store.ListProjectPhases(strings.TrimSpace(proyecto))
}

func (s *Service) GetActivePhase(proyecto string) (string, error) {
	phases, err := s.ListPhases(proyecto)
	if err != nil {
		return "", err
	}
	for _, p := range phases {
		if strings.ToLower(p.Estado) == "activa" {
			return p.Nombre, nil
		}
	}
	return "", nil
}

func (s *Service) RegisterPhase(input RegisterPhaseInput) (int64, *FaseProyecto, error) {
	id, err := s.store.RegisterProjectPhase(&FaseProyecto{
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

func (s *Service) UpdatePhase(input UpdatePhaseInput) (*FaseProyecto, error) {
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
	return s.store.RegisterTaskProgress(&AvanceTarea{
		TareaID:        input.TareaID,
		Proyecto:       strings.TrimSpace(input.Proyecto),
		FaseID:         input.FaseID,
		ProgresoPct:    input.ProgresoPct,
		ActualizadoPor: strings.TrimSpace(input.ActualizadoPor),
	})
}
