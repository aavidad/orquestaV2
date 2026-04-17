package reviewapp

import (
	"database/sql"
	"strings"
	"time"

	"orquesta/db"
)

type Repository struct{}

func NewRepository() Repository {
	return Repository{}
}

func (Repository) GetProject(ref string) (*ProjectRef, error) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, sql.ErrNoRows
	}
	return &ProjectRef{ID: proyecto.ID, Slug: strings.TrimSpace(proyecto.Slug)}, nil
}

func (Repository) GetReviewGate(id int64) (*Gate, error) {
	item, err := db.GetReviewGate(id)
	if err != nil || item == nil {
		return nil, err
	}
	return reviewGateFromDB(item), nil
}

func (Repository) ListReviewGates(filter GateFilter) ([]*Gate, error) {
	dbFilter := db.FiltroReviewGates{
		ProyectoID: filter.ProyectoID,
		TareaID:    nil,
		Reviewer:   filter.ReviewerAgente,
		Estado:     filter.Estado,
		Limit:      filter.Limit,
	}
	items, err := db.ListarReviewGates(dbFilter)
	if err != nil {
		return nil, err
	}
	out := make([]*Gate, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, reviewGateFromDB(item))
	}
	return out, nil
}

func (Repository) CreateReviewGate(gate *Gate) (int64, error) {
	id, err := db.CrearReviewGate(reviewGateToDB(gate))
	if err != nil {
		return 0, err
	}
	if gate != nil {
		gate.ID = id
	}
	return id, nil
}

func (Repository) UpdateReviewGate(gate *Gate) error {
	return db.ActualizarReviewGate(reviewGateToDB(gate))
}

func reviewGateFromDB(item *db.ReviewGate) *Gate {
	if item == nil {
		return nil
	}
	return &Gate{
		ID:             item.ID,
		ProyectoID:     int64FromDB(item.ProyectoID),
		TareaID:        int64PtrFromDB(item.TareaID),
		WorktreeID:     int64PtrFromDB(item.WorktreeID),
		RequestedBy:    strings.TrimSpace(item.RequestedBy),
		ReviewerAgente: strings.TrimSpace(item.ReviewerAgente),
		Estado:         strings.TrimSpace(string(item.Estado)),
		SeverityMax:    strings.TrimSpace(item.SeverityMax),
		FindingsJSON:   strings.TrimSpace(item.FindingsJSON),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		ResolvedAt:     timePtrFromDB(item.ResolvedAt),
	}
}

func reviewGateToDB(item *Gate) *db.ReviewGate {
	if item == nil {
		return nil
	}
	return &db.ReviewGate{
		ID:             item.ID,
		ProyectoID:     int64PtrToDB(item.ProyectoID),
		TareaID:        int64OptionalPtrToDB(item.TareaID),
		WorktreeID:     int64OptionalPtrToDB(item.WorktreeID),
		RequestedBy:    strings.TrimSpace(item.RequestedBy),
		ReviewerAgente: strings.TrimSpace(item.ReviewerAgente),
		Estado:         db.EstadoReviewGate(strings.TrimSpace(item.Estado)),
		SeverityMax:    strings.TrimSpace(item.SeverityMax),
		FindingsJSON:   strings.TrimSpace(item.FindingsJSON),
		ResolvedAt:     timePtrToDB(item.ResolvedAt),
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func int64FromDB(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func int64PtrFromDB(v *int64) *int64 {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func int64PtrToDB(v int64) *int64 {
	if v == 0 {
		return nil
	}
	cp := v
	return &cp
}

func int64OptionalPtrToDB(v *int64) *int64 {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func timePtrFromDB(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func timePtrToDB(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
