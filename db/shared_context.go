package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SharedContextItem struct {
	ID          int64
	ProyectoID  *int64
	Agente      string
	Tipo        string
	Titulo      string
	Detalle     string
	PayloadJSON string
	Peso        float64
	Origen      string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FiltroSharedContextItems struct {
	ProyectoID *int64
	Agente     *string
	Tipo       *string
	Activos    bool
	Limit      int
}

func CreateSharedContextItem(item *SharedContextItem) (int64, error) {
	if item == nil {
		return 0, fmt.Errorf("shared context nil")
	}
	titulo := strings.TrimSpace(item.Titulo)
	if titulo == "" {
		return 0, fmt.Errorf("titulo obligatorio")
	}
	tipo := strings.TrimSpace(item.Tipo)
	if tipo == "" {
		tipo = "nota"
	}
	payload := strings.TrimSpace(item.PayloadJSON)
	if payload == "" {
		payload = "{}"
	}
	if !json.Valid([]byte(payload)) {
		return 0, fmt.Errorf("payload_json invalido")
	}
	peso := item.Peso
	if peso <= 0 {
		peso = 5
	}
	res, err := DB.Exec(`
		INSERT INTO shared_context_items (proyecto_id, agente, tipo, titulo, detalle, payload_json, peso, origen, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ProyectoID,
		strings.TrimSpace(item.Agente),
		tipo,
		titulo,
		strings.TrimSpace(item.Detalle),
		payload,
		peso,
		strings.TrimSpace(item.Origen),
		item.ExpiresAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListSharedContextItems(filtro FiltroSharedContextItems) ([]*SharedContextItem, error) {
	where := make([]string, 0, 4)
	args := make([]any, 0, 6)
	if filtro.ProyectoID != nil && *filtro.ProyectoID > 0 {
		where = append(where, "proyecto_id = ?")
		args = append(args, *filtro.ProyectoID)
	}
	if filtro.Agente != nil && strings.TrimSpace(*filtro.Agente) != "" {
		where = append(where, "(agente = '' OR lower(agente) = lower(?))")
		args = append(args, strings.TrimSpace(*filtro.Agente))
	}
	if filtro.Tipo != nil && strings.TrimSpace(*filtro.Tipo) != "" {
		where = append(where, "tipo = ?")
		args = append(args, strings.TrimSpace(*filtro.Tipo))
	}
	if filtro.Activos {
		where = append(where, "(expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)")
	}
	sqlText := `
		SELECT id, proyecto_id, agente, tipo, titulo, detalle, payload_json, peso, origen, expires_at, created_at, updated_at
		FROM shared_context_items`
	if len(where) > 0 {
		sqlText += " WHERE " + strings.Join(where, " AND ")
	}
	sqlText += " ORDER BY peso DESC, updated_at DESC, id DESC"
	if filtro.Limit > 0 {
		sqlText += fmt.Sprintf(" LIMIT %d", filtro.Limit)
	}
	rows, err := DB.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SharedContextItem
	for rows.Next() {
		item, err := scanSharedContextItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func BuildSharedContextSummary(agente string, proyecto *Proyecto) ([]map[string]any, string) {
	if proyecto == nil || proyecto.ID <= 0 {
		return nil, ""
	}
	agente = strings.TrimSpace(agente)
	items, err := ListSharedContextItems(FiltroSharedContextItems{
		ProyectoID: &proyecto.ID,
		Agente:     &agente,
		Activos:    true,
		Limit:      5,
	})
	if err != nil || len(items) == 0 {
		return nil, ""
	}
	out := make([]map[string]any, 0, len(items))
	titulos := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, map[string]any{
			"id":      item.ID,
			"tipo":    strings.TrimSpace(item.Tipo),
			"titulo":  strings.TrimSpace(item.Titulo),
			"detalle": strings.TrimSpace(item.Detalle),
			"peso":    item.Peso,
			"agente":  strings.TrimSpace(item.Agente),
			"origen":  strings.TrimSpace(item.Origen),
			"payload": rawJSONOrStringDB(item.PayloadJSON),
		})
		titulos = append(titulos, strings.TrimSpace(item.Titulo))
	}
	if len(out) == 0 {
		return nil, ""
	}
	return out, "Contexto compartido: " + strings.Join(titulos, " | ")
}

func AppendSharedContextPayload(prev string, items []map[string]any) string {
	if len(items) == 0 {
		return strings.TrimSpace(prev)
	}
	return MergeResumePayloadEnvelope(prev, map[string]any{
		"shared_context": items,
	})
}

func scanSharedContextItem(s scanner) (*SharedContextItem, error) {
	var (
		item      SharedContextItem
		projectID sql.NullInt64
		expiresAt sql.NullTime
	)
	if err := s.Scan(
		&item.ID,
		&projectID,
		&item.Agente,
		&item.Tipo,
		&item.Titulo,
		&item.Detalle,
		&item.PayloadJSON,
		&item.Peso,
		&item.Origen,
		&expiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if projectID.Valid {
		item.ProyectoID = &projectID.Int64
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	return &item, nil
}
