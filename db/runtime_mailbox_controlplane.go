package db

import (
	"database/sql"
	"encoding/json"
	"strings"
)

func EnviarRuntimeMailbox(msg *RuntimeMailboxMessage) (int64, error) {
	if msg == nil || strings.TrimSpace(msg.FromAgente) == "" || strings.TrimSpace(msg.ToAgente) == "" || strings.TrimSpace(msg.Kind) == "" {
		return 0, sql.ErrNoRows
	}
	var err error
	msg.FromAgente, err = CanonicalizeAgentName(msg.FromAgente)
	if err != nil {
		return 0, err
	}
	msg.ToAgente, err = CanonicalizeAgentName(msg.ToAgente)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(msg.PayloadJSON) == "" {
		msg.PayloadJSON = "{}"
	}
	if payloadJSON, err := normalizarRuntimeMailboxPayloadJSON(msg.PayloadJSON, msg.FromAgente, msg.ToAgente); err != nil {
		return 0, err
	} else {
		msg.PayloadJSON = payloadJSON
	}
	id, err := insertReturningID(`
		INSERT INTO runtime_mailbox (
			from_agente, to_agente, proyecto_id, runtime_order_id, kind, payload_json, estado
		) VALUES (?,?,?,?,?,?,?)`,
		msg.FromAgente, msg.ToAgente, msg.ProyectoID, msg.RuntimeOrderID, msg.Kind, msg.PayloadJSON, defaultMailboxEstado(msg.Estado),
	)
	if err != nil {
		return 0, err
	}
	if runtimeMailboxKindSupersedible(strings.TrimSpace(msg.Kind)) {
		if _, err := ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func normalizarRuntimeMailboxPayloadJSON(raw, fromAgente, toAgente string) (string, error) {
	payload := mapFromJSON(raw)
	if strings.TrimSpace(fromAgente) != "" {
		payload["from_agente"] = strings.TrimSpace(fromAgente)
	}
	if strings.TrimSpace(toAgente) != "" {
		payload["to_agente"] = strings.TrimSpace(toAgente)
	}
	for _, key := range []string{"agente_origen", "agente_destino", "origin_agent", "contra_agente"} {
		text := stringFromMap(payload, key, "")
		if strings.TrimSpace(text) == "" {
			continue
		}
		canonico, err := CanonicalizeAgentName(text)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(canonico) != "" {
			payload[key] = strings.TrimSpace(canonico)
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ListarRuntimeMailbox(filter FiltroRuntimeMailbox) ([]*RuntimeMailboxMessage, error) {
	q := runtimeMailboxSelectBase() + ` WHERE 1=1`
	var args []any
	if filter.ToAgente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.ToAgente)
		if err != nil {
			return nil, err
		}
		q += ` AND to_agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.FromAgente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.FromAgente)
		if err != nil {
			return nil, err
		}
		q += ` AND from_agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.ProyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.Estado != nil {
		q += ` AND estado = ?`
		args = append(args, strings.TrimSpace(*filter.Estado))
	}
	q += ` ORDER BY id DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeMailboxMessage
	for rows.Next() {
		msg, err := scanRuntimeMailbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func ResumirRuntimeMailboxPorAgente() ([]*RuntimeMailboxPanelSummary, error) {
	rows, err := DB.Query(`
		SELECT agente, SUM(total_count) AS total_count, SUM(pending_count) AS pending_count
		FROM (
			SELECT to_agente AS agente,
			       COUNT(*) AS total_count,
			       SUM(CASE WHEN estado='pendiente' THEN 1 ELSE 0 END) AS pending_count
			FROM runtime_mailbox
			WHERE TRIM(COALESCE(to_agente, '')) <> ''
			GROUP BY to_agente
			UNION ALL
			SELECT from_agente AS agente,
			       COUNT(*) AS total_count,
			       SUM(CASE WHEN estado='pendiente' THEN 1 ELSE 0 END) AS pending_count
			FROM runtime_mailbox
			WHERE TRIM(COALESCE(from_agente, '')) <> ''
			GROUP BY from_agente
		) mailbox
		GROUP BY agente
		ORDER BY agente`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeMailboxPanelSummary
	for rows.Next() {
		var item RuntimeMailboxPanelSummary
		if err := rows.Scan(&item.Agente, &item.Total, &item.Pending); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

func ListarUltimaAutonomiaMailboxPorAgente() ([]*RuntimeMailboxMessage, error) {
	rows, err := DB.Query(`
		SELECT m.id, m.from_agente, m.to_agente, m.proyecto_id, m.runtime_order_id, m.kind, m.payload_json,
		       m.estado, m.created_at, m.delivered_at, m.consumed_at
		FROM runtime_mailbox m
		JOIN (
			SELECT to_agente, MAX(id) AS last_id
			FROM runtime_mailbox
			WHERE kind='autonomia'
			  AND TRIM(COALESCE(to_agente, '')) <> ''
			GROUP BY to_agente
		) agg ON agg.last_id = m.id
		ORDER BY m.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeMailboxMessage
	for rows.Next() {
		msg, err := scanRuntimeMailbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func GetRuntimeMailboxByRuntimeOrderID(runtimeOrderID int64) (*RuntimeMailboxMessage, error) {
	row := DB.QueryRow(runtimeMailboxSelectBase()+`
		WHERE runtime_order_id = ?
		ORDER BY id DESC
		LIMIT 1`, runtimeOrderID)
	msg, err := scanRuntimeMailbox(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return msg, err
}

func GetRuntimeMailbox(id int64) (*RuntimeMailboxMessage, error) {
	row := DB.QueryRow(runtimeMailboxSelectBase()+`
		WHERE id = ?
		LIMIT 1`, id)
	msg, err := scanRuntimeMailbox(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return msg, err
}

func MarcarRuntimeMailboxEntregado(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_mailbox
		SET estado='entregado', delivered_at=CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

func RearmarRuntimeMailboxPendiente(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_mailbox
		SET estado='pendiente',
		    delivered_at=NULL,
		    consumed_at=NULL
		WHERE id = ?
		  AND estado='entregado'`, id)
	return err
}

func MarcarRuntimeMailboxConsumido(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_mailbox
		SET estado='consumido', consumed_at=CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

func ConsumirRuntimeMailboxPendienteSupersedido(toAgente string, proyectoID *int64, kind string, keepID int64) (int64, error) {
	toAgente = strings.TrimSpace(toAgente)
	kind = strings.TrimSpace(kind)
	if toAgente == "" || kind == "" || keepID <= 0 {
		return 0, nil
	}
	args := []any{toAgente}
	q := `
		UPDATE runtime_mailbox
		SET estado='consumido',
		    delivered_at=COALESCE(delivered_at, CURRENT_TIMESTAMP),
		    consumed_at=CURRENT_TIMESTAMP
		WHERE estado='pendiente'
		  AND to_agente = ?`
	if family := runtimeMailboxSupersedeFamily(kind); family != "" {
		q += ` AND kind IN (` + runtimeMailboxFamilyPlaceholders(family) + `)`
		args = append(args, runtimeMailboxFamilyKinds(family)...)
	} else {
		q += ` AND kind = ?`
		args = append(args, kind)
	}
	q += ` AND id < ?`
	args = append(args, keepID)
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	res, err := DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}

func runtimeMailboxSupersedeFamily(kind string) string {
	switch strings.TrimSpace(kind) {
	case "autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return "guidance"
	default:
		return ""
	}
}

func runtimeMailboxFamilyKinds(family string) []any {
	switch strings.TrimSpace(family) {
	case "guidance":
		return []any{"autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh}
	default:
		return nil
	}
}

func runtimeMailboxFamilyPlaceholders(family string) string {
	kinds := runtimeMailboxFamilyKinds(family)
	if len(kinds) == 0 {
		return "?"
	}
	return runtimeSQLPlaceholders(len(kinds))
}

func runtimeMailboxKindSupersedible(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "instruction", "autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}
