package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type EstadoEntregaNotificacion string

const (
	EntregaNotificacionPendiente EstadoEntregaNotificacion = "pendiente"
	EntregaNotificacionEntregada EstadoEntregaNotificacion = "entregada"
	EntregaNotificacionFallida   EstadoEntregaNotificacion = "fallida"
)

type EntregaNotificacion struct {
	ID          int64                     `json:"id"`
	Canal       string                    `json:"canal"`
	Destino     string                    `json:"destino"`
	TipoEvento  string                    `json:"tipo_evento"`
	Estado      EstadoEntregaNotificacion `json:"estado"`
	Intentos    int                       `json:"intentos"`
	UltimoError string                    `json:"ultimo_error,omitempty"`
	Evento      EventoNotificacion        `json:"evento"`
	NextRetryAt *time.Time                `json:"next_retry_at,omitempty"`
	DeliveredAt *time.Time                `json:"delivered_at,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

type FiltroEntregasNotificacion struct {
	Canal       string
	Estado      string
	Limit       int
	DueOnly     bool
	ActivasOnly bool
}

func CrearEntregaNotificacion(canal, destino string, ev EventoNotificacion) (int64, error) {
	if DB == nil {
		return 0, nil
	}
	if err := ensureNotificacionesOutboxSchema(); err != nil {
		return 0, err
	}
	eventoJSON, err := json.Marshal(ev)
	if err != nil {
		return 0, err
	}
	id, err := insertReturningID(`
		INSERT INTO notificaciones_entregas (
			canal, destino, tipo_evento, estado, intentos, ultimo_error, evento_json
		) VALUES (?,?,?,?,?,?,?)`,
		strings.TrimSpace(canal),
		strings.TrimSpace(destino),
		strings.TrimSpace(ev.Tipo),
		EntregaNotificacionPendiente,
		1,
		"",
		string(eventoJSON),
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func MarcarEntregaNotificacionEntregada(id int64) error {
	if DB == nil || id <= 0 {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE notificaciones_entregas
		SET estado = ?, ultimo_error = '', next_retry_at = NULL, delivered_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		EntregaNotificacionEntregada, id,
	)
	return err
}

func MarcarEntregaNotificacionFallida(id int64, ultimoError string, nextRetryAt time.Time) error {
	if DB == nil || id <= 0 {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE notificaciones_entregas
		SET estado = ?, ultimo_error = ?, next_retry_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		EntregaNotificacionFallida,
		strings.TrimSpace(ultimoError),
		nextRetryAt,
		id,
	)
	return err
}

func IncrementarIntentoEntregaNotificacion(id int64) (int, error) {
	if DB == nil {
		return 0, nil
	}
	if id <= 0 {
		return 0, fmt.Errorf("id de entrega inválido")
	}
	if _, err := DB.Exec(`UPDATE notificaciones_entregas SET intentos = intentos + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
		return 0, err
	}
	var intentos int
	if err := DB.QueryRow(`SELECT intentos FROM notificaciones_entregas WHERE id = ?`, id).Scan(&intentos); err != nil {
		return 0, err
	}
	return intentos, nil
}

func ListarEntregasNotificacion(filter FiltroEntregasNotificacion) ([]*EntregaNotificacion, error) {
	if DB == nil {
		return []*EntregaNotificacion{}, nil
	}
	if err := ensureNotificacionesOutboxSchema(); err != nil {
		return nil, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	q := `
		SELECT id, canal, destino, tipo_evento, estado, intentos, ultimo_error, evento_json, next_retry_at, delivered_at, created_at, updated_at
		FROM notificaciones_entregas
		WHERE 1=1`
	args := []any{}
	if canal := strings.TrimSpace(filter.Canal); canal != "" {
		q += ` AND canal = ?`
		args = append(args, canal)
	}
	if estado := strings.TrimSpace(filter.Estado); estado != "" {
		q += ` AND estado = ?`
		args = append(args, estado)
	}
	if filter.ActivasOnly {
		q += ` AND estado IN (?, ?)`
		args = append(args, EntregaNotificacionPendiente, EntregaNotificacionFallida)
	}
	if filter.DueOnly {
		q += ` AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)`
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, filter.Limit)
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*EntregaNotificacion
	for rows.Next() {
		item, err := scanEntregaNotificacion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func GetEntregaNotificacion(id int64) (*EntregaNotificacion, error) {
	if DB == nil {
		return nil, nil
	}
	if err := ensureNotificacionesOutboxSchema(); err != nil {
		return nil, err
	}
	row := DB.QueryRow(`
		SELECT id, canal, destino, tipo_evento, estado, intentos, ultimo_error, evento_json, next_retry_at, delivered_at, created_at, updated_at
		FROM notificaciones_entregas
		WHERE id = ?`, id)
	item, err := scanEntregaNotificacion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func ensureNotificacionesOutboxSchema() error {
	if DB == nil {
		return nil
	}
	return ensureRenderedSchemaStatements(renderNotificacionesOutboxSchemaForDriver(CurrentStorageDriver()))
}

func renderNotificacionesOutboxSchemaForDriver(driver string) []string {
	return []string{
		renderDriverColumnSyntax(driver, `
		CREATE TABLE IF NOT EXISTS notificaciones_entregas (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			canal         TEXT    NOT NULL,
			destino       TEXT    NOT NULL DEFAULT '',
			tipo_evento   TEXT    NOT NULL DEFAULT '',
			estado        TEXT    NOT NULL DEFAULT 'pendiente'
			                           CHECK (estado IN ('pendiente','entregada','fallida')),
			intentos      INTEGER NOT NULL DEFAULT 1,
			ultimo_error  TEXT    NOT NULL DEFAULT '',
			evento_json   TEXT    NOT NULL DEFAULT '{}',
			next_retry_at DATETIME,
			delivered_at  DATETIME,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`),
		`CREATE INDEX IF NOT EXISTS idx_notificaciones_entregas_canal_estado_id ON notificaciones_entregas(canal, estado, id DESC)`,
	}
}

func scanEntregaNotificacion(scanner interface{ Scan(dest ...any) error }) (*EntregaNotificacion, error) {
	var (
		item        EntregaNotificacion
		eventoJSON  string
		nextRetryAt sql.NullTime
		deliveredAt sql.NullTime
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Canal,
		&item.Destino,
		&item.TipoEvento,
		&item.Estado,
		&item.Intentos,
		&item.UltimoError,
		&eventoJSON,
		&nextRetryAt,
		&deliveredAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if strings.TrimSpace(eventoJSON) != "" {
		_ = json.Unmarshal([]byte(eventoJSON), &item.Evento)
	}
	if nextRetryAt.Valid {
		item.NextRetryAt = &nextRetryAt.Time
	}
	if deliveredAt.Valid {
		item.DeliveredAt = &deliveredAt.Time
	}
	return &item, nil
}
