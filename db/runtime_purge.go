package db

import (
	"fmt"
	"strings"
	"time"
)

func PurgarRuntimeHandlesInactivos(filtro FiltroPurgadoRuntimeHandles) (*PurgaRuntimeHandlesResultado, error) {
	if filtro.Agente == nil && filtro.ProyectoID == nil {
		return nil, fmt.Errorf("debes indicar agente o proyecto para purgar runtime handles")
	}
	estados, err := normalizarEstadosPurgadoRuntimeHandles(filtro.Estados)
	if err != nil {
		return nil, err
	}
	query := `SELECT id FROM runtime_handles WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)`
	args := make([]any, 0, len(estados)+2)
	for _, estado := range estados {
		args = append(args, estado)
	}
	if filtro.Agente != nil && strings.TrimSpace(*filtro.Agente) != "" {
		query += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*filtro.Agente))
	}
	if filtro.ProyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *filtro.ProyectoID)
	}
	query += ` ORDER BY id DESC`

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &PurgaRuntimeHandlesResultado{Estados: estados}, nil
	}
	if err := validarPurgadoRuntimeHandles(ids); err != nil {
		return nil, err
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	argsDelete := int64SliceToAny(ids)
	if _, err := tx.Exec(`UPDATE runtime_orders SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE runtime_transcript SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM runtime_handles WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	return &PurgaRuntimeHandlesResultado{
		Deleted:    len(ids),
		DeletedIDs: ids,
		Estados:    estados,
	}, nil
}

func PurgarRuntimeOrdersTerminales(filtro FiltroPurgadoRuntimeOrders) (*PurgaRuntimeOrdersResultado, error) {
	if filtro.Agente == nil && filtro.ProyectoID == nil {
		return nil, fmt.Errorf("debes indicar agente o proyecto para purgar runtime orders")
	}
	estados, err := normalizarEstadosPurgadoRuntimeOrders(filtro.Estados)
	if err != nil {
		return nil, err
	}
	tipos, err := normalizarTiposPurgadoRuntimeOrders(filtro.Tipos)
	if err != nil {
		return nil, err
	}
	query := `SELECT id FROM runtime_orders WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)`
	args := make([]any, 0, len(estados)+4)
	for _, estado := range estados {
		args = append(args, estado)
	}
	if filtro.Agente != nil && strings.TrimSpace(*filtro.Agente) != "" {
		query += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*filtro.Agente))
	}
	if filtro.ProyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *filtro.ProyectoID)
	}
	if len(tipos) > 0 {
		query += ` AND tipo IN (` + runtimeSQLPlaceholders(len(tipos)) + `)`
		for _, tipo := range tipos {
			args = append(args, tipo)
		}
	}
	if filtro.CreatedBefore != nil && !filtro.CreatedBefore.IsZero() {
		query += ` AND created_at < ?`
		args = append(args, filtro.CreatedBefore.UTC())
	}
	query += ` ORDER BY id DESC`

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &PurgaRuntimeOrdersResultado{Estados: estados, Tipos: tipos}, nil
	}
	if err := validarPurgadoRuntimeOrders(ids); err != nil {
		return nil, err
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	for _, chunk := range runtimeInt64Chunks(ids, 400) {
		argsDelete := int64SliceToAny(chunk)
		if _, err := tx.Exec(`UPDATE runtime_mailbox SET runtime_order_id = NULL WHERE runtime_order_id IN (`+runtimeSQLPlaceholders(len(chunk))+`)`, argsDelete...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM runtime_orders WHERE id IN (`+runtimeSQLPlaceholders(len(chunk))+`)`, argsDelete...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &PurgaRuntimeOrdersResultado{
		Deleted:    len(ids),
		DeletedIDs: ids,
		Estados:    estados,
		Tipos:      tipos,
	}, nil
}

func PurgarRuntimeHistorico() (*PurgaRuntimeHistoricoResultado, error) {
	now := time.Now().UTC()
	handlesCutoff := runtimeHistoricoRetentionCutoff(now, "runtime_handles_retention_minutes", "runtime_handles_retention_hours", 24)
	ordersCutoff := runtimeHistoricoRetentionCutoff(now, "runtime_orders_retention_minutes", "runtime_orders_retention_hours", 72)
	runtimesCutoff := runtimeHistoricoRetentionCutoff(now, "runtime_instances_retention_minutes", "runtime_instances_retention_hours", 72)
	handleIDs, err := listarRuntimeHandlesPurgablesHistoricos([]string{"cerrado", "fallido"}, handlesCutoff)
	if err != nil {
		return nil, err
	}
	orderIDs, err := listarRuntimeOrdersPurgablesHistoricos([]string{"completada", "fallida", "expirada", "cancelada"}, ordersCutoff)
	if err != nil {
		return nil, err
	}
	mailboxIDs, err := listarRuntimeMailboxPurgablesHistoricos([]string{"consumido", "cancelado", "expirado"}, ordersCutoff)
	if err != nil {
		return nil, err
	}
	runtimeIDs, err := listarRuntimeInstancesPurgablesHistoricos([]string{"cerrado", "bloqueado", "degradado"}, runtimesCutoff)
	if err != nil {
		return nil, err
	}
	if err := validarPurgadoRuntimeHandles(handleIDs); err != nil {
		if runtimeHistoricoHandlePurgeSkippable(err) {
			handleIDs = nil
		} else {
			return nil, err
		}
	}
	if err := validarPurgadoRuntimeOrders(orderIDs); err != nil {
		return nil, err
	}
	if err := validarPurgadoRuntimeInstances(runtimeIDs); err != nil {
		return nil, err
	}
	result := &PurgaRuntimeHistoricoResultado{
		Handles:  &PurgaRuntimeHandlesResultado{Estados: []string{"cerrado", "fallido"}},
		Orders:   &PurgaRuntimeOrdersResultado{Estados: []string{"completada", "fallida", "expirada", "cancelada"}},
		Runtimes: &PurgaRuntimeInstancesResultado{Estados: []string{"cerrado", "bloqueado", "degradado"}},
	}
	if len(handleIDs) == 0 && len(orderIDs) == 0 && len(mailboxIDs) == 0 && len(runtimeIDs) == 0 {
		return result, nil
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	if len(handleIDs) > 0 {
		args := int64SliceToAny(handleIDs)
		if _, err := tx.Exec(`UPDATE runtime_orders SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(handleIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE runtime_transcript SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(handleIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM runtime_handles WHERE id IN (`+runtimeSQLPlaceholders(len(handleIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		result.Handles.Deleted = len(handleIDs)
		result.Handles.DeletedIDs = handleIDs
	}
	if len(orderIDs) > 0 {
		args := int64SliceToAny(orderIDs)
		argsMailboxDelete := make([]any, 0, len(orderIDs)+1)
		argsMailboxDelete = append(argsMailboxDelete, args...)
		argsMailboxDelete = append(argsMailboxDelete, ordersCutoff.UTC())
		if _, err := tx.Exec(`DELETE FROM runtime_mailbox
			WHERE runtime_order_id IN (`+runtimeSQLPlaceholders(len(orderIDs))+`)
			  AND estado IN ('consumido','cancelado','expirado')
			  AND COALESCE(consumed_at, delivered_at, created_at) < ?`, argsMailboxDelete...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE runtime_mailbox SET runtime_order_id = NULL WHERE runtime_order_id IN (`+runtimeSQLPlaceholders(len(orderIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM runtime_orders WHERE id IN (`+runtimeSQLPlaceholders(len(orderIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		result.Orders.Deleted = len(orderIDs)
		result.Orders.DeletedIDs = orderIDs
	}
	if len(mailboxIDs) > 0 {
		args := int64SliceToAny(mailboxIDs)
		if _, err := tx.Exec(`DELETE FROM runtime_mailbox WHERE id IN (`+runtimeSQLPlaceholders(len(mailboxIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if len(runtimeIDs) > 0 {
		args := int64SliceToAny(runtimeIDs)
		if _, err := tx.Exec(`DELETE FROM runtime_instances WHERE id IN (`+runtimeSQLPlaceholders(len(runtimeIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		result.Runtimes.Deleted = len(runtimeIDs)
		result.Runtimes.DeletedIDs = runtimeIDs
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func runtimeHistoricoHandlePurgeSkippable(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "no se pueden purgar handles con runtime orders vivas asociadas")
}

func runtimeHistoricoRetentionCutoff(now time.Time, minutesKey, hoursKey string, fallbackHours int64) time.Time {
	now = now.UTC()
	if minutes := configInt64Fallback(minutesKey, -1); minutes >= 0 {
		cutoff := now.Add(-time.Duration(minutes) * time.Minute)
		if cutoff.Before(now) {
			return cutoff
		}
		return now
	}
	hours := configInt64Fallback(hoursKey, fallbackHours)
	if hours <= 0 {
		hours = fallbackHours
	}
	cutoff := now.Add(-time.Duration(hours) * time.Hour)
	if cutoff.Before(now) {
		return cutoff
	}
	return now.Add(-time.Duration(fallbackHours) * time.Hour)
}

func listarRuntimeHandlesPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_handles WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND COALESCE(last_seen_at, updated_at, created_at) < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listarRuntimeOrdersPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_orders WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND created_at < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listarRuntimeMailboxPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_mailbox WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND runtime_order_id IS NULL
		AND COALESCE(consumed_at, delivered_at, created_at) < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listarRuntimeInstancesPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_instances WHERE logical_state IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND COALESCE(last_heartbeat_at, last_event_at, updated_at, created_at) < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func normalizarEstadosPurgadoRuntimeHandles(estados []string) ([]string, error) {
	if len(estados) == 0 {
		return []string{"cerrado", "fallido"}, nil
	}
	seen := make(map[string]struct{}, len(estados))
	out := make([]string, 0, len(estados))
	for _, raw := range estados {
		estado := strings.ToLower(strings.TrimSpace(raw))
		switch estado {
		case "cerrado", "fallido":
		case "activo", "pausado":
			return nil, fmt.Errorf("no se permite purgar handles en estado %q; deten el agente primero", estado)
		default:
			return nil, fmt.Errorf("estado de runtime handle no soportado: %s", strings.TrimSpace(raw))
		}
		if _, ok := seen[estado]; ok {
			continue
		}
		seen[estado] = struct{}{}
		out = append(out, estado)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("debes indicar al menos un estado purgable")
	}
	return out, nil
}

func normalizarEstadosPurgadoRuntimeOrders(estados []string) ([]string, error) {
	if len(estados) == 0 {
		return []string{"completada", "fallida", "expirada", "cancelada"}, nil
	}
	seen := make(map[string]struct{}, len(estados))
	out := make([]string, 0, len(estados))
	for _, raw := range estados {
		estado := strings.ToLower(strings.TrimSpace(raw))
		switch estado {
		case "completada", "fallida", "expirada", "cancelada":
		case "pendiente", "tomada", "ejecutando":
			return nil, fmt.Errorf("no se permite purgar runtime orders en estado %q; espera a que terminen o cancelalas primero", estado)
		default:
			return nil, fmt.Errorf("estado de runtime order no soportado: %s", strings.TrimSpace(raw))
		}
		if _, ok := seen[estado]; ok {
			continue
		}
		seen[estado] = struct{}{}
		out = append(out, estado)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("debes indicar al menos un estado purgable")
	}
	return out, nil
}

func normalizarTiposPurgadoRuntimeOrders(tipos []string) ([]string, error) {
	if len(tipos) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(tipos))
	out := make([]string, 0, len(tipos))
	for _, raw := range tipos {
		tipo := strings.TrimSpace(raw)
		if tipo == "" {
			continue
		}
		if !runtimeOrderTipoDespachable(tipo) && tipo != "handoff" {
			return nil, fmt.Errorf("tipo de runtime order no soportado para purga: %s", tipo)
		}
		if _, ok := seen[tipo]; ok {
			continue
		}
		seen[tipo] = struct{}{}
		out = append(out, tipo)
	}
	return out, nil
}

func validarPurgadoRuntimeHandles(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := DB.Query(`
		SELECT id, estado FROM runtime_orders
		WHERE handle_id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND estado IN ('pendiente','tomada','ejecutando')`,
		int64SliceToAny(ids)...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var vivos []string
	for rows.Next() {
		var (
			id     int64
			estado string
		)
		if err := rows.Scan(&id, &estado); err != nil {
			return err
		}
		vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(estado)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar handles con runtime orders vivas asociadas: %s", strings.Join(vivos, ", "))
	}
	return nil
}

func validarPurgadoRuntimeOrders(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var vivos []string
	for _, chunk := range runtimeInt64Chunks(ids, 400) {
		rows, err := DB.Query(`
			SELECT id, estado FROM runtime_orders
			WHERE id IN (`+runtimeSQLPlaceholders(len(chunk))+`)
			  AND estado IN ('pendiente','tomada','ejecutando')`,
			int64SliceToAny(chunk)...,
		)
		if err != nil {
			return err
		}
		for rows.Next() {
			var (
				id     int64
				estado string
			)
			if err := rows.Scan(&id, &estado); err != nil {
				rows.Close()
				return err
			}
			vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(estado)))
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar runtime orders vivas: %s", strings.Join(vivos, ", "))
	}
	return nil
}

func runtimeInt64Chunks(ids []int64, chunkSize int) [][]int64 {
	if len(ids) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 400
	}
	out := make([][]int64, 0, (len(ids)+chunkSize-1)/chunkSize)
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[start:end])
	}
	return out
}

func validarPurgadoRuntimeInstances(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	args := int64SliceToAny(ids)
	rows, err := DB.Query(`
		SELECT id, logical_state FROM runtime_instances
		WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND logical_state NOT IN ('cerrado','bloqueado','degradado')`,
		args...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var vivos []string
	for rows.Next() {
		var (
			id           int64
			logicalState string
		)
		if err := rows.Scan(&id, &logicalState); err != nil {
			return err
		}
		vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(logicalState)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar runtime instances vivas: %s", strings.Join(vivos, ", "))
	}
	rows, err = DB.Query(`
		SELECT id, estado FROM runtime_handles
		WHERE runtime_id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND estado IN ('activo','pausado')`,
		args...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	vivos = vivos[:0]
	for rows.Next() {
		var (
			id     int64
			estado string
		)
		if err := rows.Scan(&id, &estado); err != nil {
			return err
		}
		vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(estado)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar runtime instances con handles vivos asociados: %s", strings.Join(vivos, ", "))
	}
	return nil
}

func int64SliceToAny(ids []int64) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, id)
	}
	return out
}

func runtimeSQLPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}
