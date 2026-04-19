package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type EstadoReviewGate string

const (
	ReviewGatePendiente  EstadoReviewGate = "pendiente"
	ReviewGateEnRevision EstadoReviewGate = "en_revision"
	ReviewGateCambiosPed EstadoReviewGate = "cambios_pedidos"
	ReviewGateAprobado   EstadoReviewGate = "aprobado"
	ReviewGateBloqueado  EstadoReviewGate = "bloqueado"
)

type ReviewGate struct {
	ID             int64            `json:"id"`
	ProyectoID     *int64           `json:"proyecto_id,omitempty"`
	TareaID        *int64           `json:"tarea_id,omitempty"`
	WorktreeID     *int64           `json:"worktree_id,omitempty"`
	RequestedBy    string           `json:"requested_by"`
	ReviewerAgente string           `json:"reviewer_agente"`
	Estado         EstadoReviewGate `json:"estado"`
	SeverityMax    string           `json:"severity_max"`
	FindingsJSON   string           `json:"findings_json"`
	ResolvedAt     *time.Time       `json:"resolved_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type FiltroReviewGates struct {
	ProyectoID *int64
	TareaID    *int64
	Reviewer   *string
	Estado     *string
	Limit      int
}

func CrearReviewGate(item *ReviewGate) (int64, error) {
	if err := ensureReviewGateSchema(); err != nil {
		return 0, err
	}
	if item == nil {
		return 0, fmt.Errorf("review_gate nil")
	}
	normalizarReviewGate(item)
	id, err := insertReturningID(`
		INSERT INTO review_gates (
			proyecto_id, tarea_id, worktree_id, requested_by, reviewer_agente,
			estado, severity_max, findings_json, resolved_at
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		item.ProyectoID, item.TareaID, item.WorktreeID, strings.TrimSpace(item.RequestedBy),
		strings.TrimSpace(item.ReviewerAgente), item.Estado, strings.TrimSpace(item.SeverityMax),
		strings.TrimSpace(item.FindingsJSON), item.ResolvedAt,
	)
	if err != nil {
		return 0, err
	}
	item.ID = id
	return id, nil
}

func GetReviewGate(id int64) (*ReviewGate, error) {
	exists, err := SchemaObjectExists("table", "review_gates")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	row := DB.QueryRow(`
		SELECT id, proyecto_id, tarea_id, worktree_id, requested_by, reviewer_agente,
		       estado, severity_max, findings_json, resolved_at, created_at, updated_at
		FROM review_gates
		WHERE id = ?`, id)
	item, err := scanReviewGate(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func ListarReviewGates(filter FiltroReviewGates) ([]*ReviewGate, error) {
	exists, err := SchemaObjectExists("table", "review_gates")
	if err != nil {
		return nil, err
	}
	if !exists {
		return []*ReviewGate{}, nil
	}
	q := `
		SELECT id, proyecto_id, tarea_id, worktree_id, requested_by, reviewer_agente,
		       estado, severity_max, findings_json, resolved_at, created_at, updated_at
		FROM review_gates
		WHERE 1=1`
	args := []any{}
	if filter.ProyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.TareaID != nil {
		q += ` AND tarea_id = ?`
		args = append(args, *filter.TareaID)
	}
	if filter.Reviewer != nil {
		q += ` AND reviewer_agente = ?`
		args = append(args, strings.TrimSpace(*filter.Reviewer))
	}
	if filter.Estado != nil {
		q += ` AND estado = ?`
		args = append(args, strings.TrimSpace(*filter.Estado))
	}
	q += ` ORDER BY id DESC`
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	q += ` LIMIT ?`
	args = append(args, filter.Limit)
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ReviewGate
	for rows.Next() {
		item, err := scanReviewGate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func ActualizarReviewGate(item *ReviewGate) error {
	if err := ensureReviewGateSchema(); err != nil {
		return err
	}
	if item == nil || item.ID == 0 {
		return fmt.Errorf("review_gate inválido")
	}
	normalizarReviewGate(item)
	_, err := DB.Exec(`
		UPDATE review_gates
		SET proyecto_id = ?,
		    tarea_id = ?,
		    worktree_id = ?,
		    requested_by = ?,
		    reviewer_agente = ?,
		    estado = ?,
		    severity_max = ?,
		    findings_json = ?,
		    resolved_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		item.ProyectoID, item.TareaID, item.WorktreeID, strings.TrimSpace(item.RequestedBy),
		strings.TrimSpace(item.ReviewerAgente), item.Estado, strings.TrimSpace(item.SeverityMax),
		strings.TrimSpace(item.FindingsJSON), item.ResolvedAt, item.ID,
	)
	return err
}

func ResolverReviewGate(id int64, estado EstadoReviewGate, findingsJSON string, resolvedAt *time.Time) error {
	item, err := GetReviewGate(id)
	if err != nil {
		return err
	}
	if item == nil {
		return sql.ErrNoRows
	}
	item.Estado = estado
	item.FindingsJSON = strings.TrimSpace(findingsJSON)
	switch estado {
	case ReviewGateAprobado, ReviewGateBloqueado:
		item.ResolvedAt = resolvedAt
	default:
		item.ResolvedAt = nil
	}
	return ActualizarReviewGate(item)
}

func ensureReviewGateSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS review_gates (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto_id     INTEGER REFERENCES proyectos(id) ON DELETE CASCADE,
			tarea_id        INTEGER REFERENCES tareas(id) ON DELETE CASCADE,
			worktree_id     INTEGER REFERENCES worktrees(id) ON DELETE SET NULL,
			requested_by    TEXT    NOT NULL DEFAULT '',
			reviewer_agente TEXT    NOT NULL DEFAULT '',
			estado          TEXT    NOT NULL DEFAULT 'pendiente'
			                              CHECK (estado IN ('pendiente','en_revision','cambios_pedidos','aprobado','bloqueado')),
			severity_max    TEXT    NOT NULL DEFAULT '',
			findings_json   TEXT    NOT NULL DEFAULT '[]',
			resolved_at     DATETIME,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_review_gates_proyecto_estado ON review_gates(proyecto_id, estado, id DESC)`)
	return err
}

func normalizarReviewGate(item *ReviewGate) {
	if item == nil {
		return
	}
	if strings.TrimSpace(item.FindingsJSON) == "" {
		item.FindingsJSON = "[]"
	}
	switch item.Estado {
	case ReviewGatePendiente, ReviewGateEnRevision, ReviewGateCambiosPed, ReviewGateAprobado, ReviewGateBloqueado:
	default:
		item.Estado = ReviewGatePendiente
	}
	if item.Estado != ReviewGateAprobado && item.Estado != ReviewGateBloqueado {
		item.ResolvedAt = nil
	}
}

func scanReviewGate(scanner interface{ Scan(dest ...any) error }) (*ReviewGate, error) {
	var (
		item       ReviewGate
		proyectoID sql.NullInt64
		tareaID    sql.NullInt64
		worktreeID sql.NullInt64
		resolvedAt sql.NullTime
	)
	if err := scanner.Scan(
		&item.ID, &proyectoID, &tareaID, &worktreeID, &item.RequestedBy, &item.ReviewerAgente,
		&item.Estado, &item.SeverityMax, &item.FindingsJSON, &resolvedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		item.ProyectoID = &proyectoID.Int64
	}
	if tareaID.Valid {
		item.TareaID = &tareaID.Int64
	}
	if worktreeID.Valid {
		item.WorktreeID = &worktreeID.Int64
	}
	if resolvedAt.Valid {
		item.ResolvedAt = &resolvedAt.Time
	}
	normalizarReviewGate(&item)
	return &item, nil
}
