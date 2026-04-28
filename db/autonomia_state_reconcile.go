package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReconciliarEstadoAutonomia sanea residuos operativos que contaminan el loop
// autónomo vivo: supervisión stale, alias no canónicos y órdenes pendientes
// heredadas con payload de handoff fuera de canon.
func ReconciliarEstadoAutonomia() error {
	if err := reconciliarNotasTareasFinishAppAutonomiaPersistente(); err != nil {
		return err
	}
	if err := reconciliarTareasLibresDuplicadasFinishApp(); err != nil {
		return err
	}
	if err := reconciliarTareasWorkerEnSupervisorReservado(); err != nil {
		return err
	}
	if err := reconciliarAsignacionesSupervisorReservadoActivas(); err != nil {
		return err
	}
	if err := reconciliarAsignacionesSupervisorStale(); err != nil {
		return err
	}
	if err := reconciliarRuntimeMailboxWorkerEnSupervisorReservado(); err != nil {
		return err
	}
	if err := reconciliarRuntimeOrdersWorkerEnSupervisorReservado(); err != nil {
		return err
	}
	if err := reconciliarRuntimeMailboxPipelineFantasma(); err != nil {
		return err
	}
	if err := reconciliarRuntimeOrdersPipelineFantasma(); err != nil {
		return err
	}
	if err := reconciliarAsignacionesDuplicadasAgenteProyecto(); err != nil {
		return err
	}
	if err := reconciliarAsignacionesAliasNoCanonico(); err != nil {
		return err
	}
	if err := reconciliarRuntimeMailboxAliasNoCanonico(); err != nil {
		return err
	}
	if err := reconciliarRuntimeOrdersAliasNoCanonico(); err != nil {
		return err
	}
	return nil
}

func reconciliarTareasLibresDuplicadasFinishApp() error {
	rows, err := DB.Query(`
		SELECT id, proyecto_id, estado, COALESCE(notas,'')
		FROM tareas
		WHERE proyecto_id IS NOT NULL
		  AND estado IN ('libre','asignada','en_progreso','bloqueada','backlog')
		  AND LOWER(COALESCE(notas,'')) LIKE '%autonomia:finish_app%'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		proyectoID int64
		estado     string
		notas      string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.proyectoID, &it.estado, &it.notas); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	type keeper struct {
		id   int64
		rank int
	}
	keepers := map[int64]keeper{}
	for _, it := range items {
		rank := finishAppTaskReconcileRank(it.estado)
		if rank >= finishAppTaskReconcileRank(string(EstadoLibre)) {
			continue
		}
		current, ok := keepers[it.proyectoID]
		if !ok || rank < current.rank || (rank == current.rank && it.id < current.id) {
			keepers[it.proyectoID] = keeper{id: it.id, rank: rank}
		}
	}

	for _, it := range items {
		if !strings.EqualFold(strings.TrimSpace(it.estado), string(EstadoLibre)) {
			continue
		}
		keep, ok := keepers[it.proyectoID]
		if !ok || keep.id <= 0 || keep.id == it.id {
			continue
		}
		nota := formatearAnotacionTarea("server", fmt.Sprintf("finish_app libre duplicada; compactada contra tarea #%d activa", keep.id), time.Now().UTC())
		notas := strings.TrimSpace(it.notas)
		if notas != "" {
			notas += "\n" + nota
		} else {
			notas = nota
		}
		if _, err := DB.Exec(`UPDATE tareas SET estado='backlog', notas=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, notas, it.id); err != nil {
			return err
		}
		Audit("sistema", "compactar_finish_app_libre_duplicada", "tarea", it.id,
			fmt.Sprintf("proyecto_id=%d keep_tarea_id=%d", it.proyectoID, keep.id))
	}
	return nil
}

func finishAppTaskReconcileRank(estado string) int {
	switch strings.ToLower(strings.TrimSpace(estado)) {
	case "en_progreso":
		return 0
	case "asignada":
		return 1
	case "bloqueada":
		return 2
	case "libre":
		return 3
	case "backlog":
		return 4
	default:
		return 5
	}
}

func reconciliarNotasTareasFinishAppAutonomiaPersistente() error {
	rows, err := DB.Query(`
		SELECT id, proyecto_id, COALESCE(notas,'')
		FROM tareas
		WHERE proyecto_id IS NOT NULL
		  AND estado IN ('libre','asignada','en_progreso','bloqueada','backlog')
		  AND LOWER(COALESCE(notas,'')) LIKE '%autonomia:finish_app%'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		proyectoID int64
		notas      string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.proyectoID, &it.notas); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		policy, err := GetProyectoAutonomia(it.proyectoID)
		if err != nil {
			return err
		}
		if policy == nil || !policy.Enabled {
			continue
		}
		updated := upsertAutonomiaPersistenteTaskBlock(it.notas, policy)
		if strings.TrimSpace(updated) == strings.TrimSpace(it.notas) {
			continue
		}
		if _, err := DB.Exec(`UPDATE tareas SET notas=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, updated, it.id); err != nil {
			return err
		}
		Audit("sistema", "reconciliar_notas_finish_app_autonomia_persistente", "tarea", it.id,
			fmt.Sprintf("proyecto_id=%d supervisor=%s reviewer=%s", it.proyectoID, strings.TrimSpace(policy.SupervisorAgente), strings.TrimSpace(policy.ReviewerAgente)))
	}
	return nil
}

func upsertAutonomiaPersistenteTaskBlock(notas string, policy *ProyectoAutonomia) string {
	notas = strings.TrimSpace(notas)
	block := renderAutonomiaPersistenteTaskBlock(policy)
	if block == "" {
		return notas
	}
	lines := strings.Split(notas, "\n")
	start := -1
	end := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "autonomia_persistente_v1:" {
			continue
		}
		start = i
		end = i + 1
		for end < len(lines) {
			next := lines[end]
			if strings.HasPrefix(next, "  ") || strings.TrimSpace(next) == "" {
				end++
				continue
			}
			break
		}
		break
	}
	blockLines := strings.Split(block, "\n")
	if start >= 0 {
		replaced := append([]string{}, lines[:start]...)
		replaced = append(replaced, blockLines...)
		replaced = append(replaced, lines[end:]...)
		return strings.TrimSpace(strings.Join(replaced, "\n"))
	}
	insertAt := len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) == "autonomia:finish_app" {
			insertAt = i + 1
			break
		}
	}
	merged := append([]string{}, lines[:insertAt]...)
	merged = append(merged, blockLines...)
	merged = append(merged, lines[insertAt:]...)
	return strings.TrimSpace(strings.Join(merged, "\n"))
}

func renderAutonomiaPersistenteTaskBlock(policy *ProyectoAutonomia) string {
	if policy == nil {
		return ""
	}
	lines := []string{
		"autonomia_persistente_v1:",
		fmt.Sprintf("  enabled: %t", policy.Enabled),
		"  supervisor_agente: " + strings.TrimSpace(policy.SupervisorAgente),
		"  reviewer_agente: " + strings.TrimSpace(policy.ReviewerAgente),
		fmt.Sprintf("  max_workers: %d", policy.MaxWorkers),
		fmt.Sprintf("  auto_create_tasks: %t", policy.AutoCreateTasks),
	}
	return strings.Join(lines, "\n")
}

func reconciliarAsignacionesSupervisorReservadoActivas() error {
	estado := AsignacionActiva
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Estado: &estado})
	if err != nil {
		return err
	}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.ProyectoID <= 0 {
			continue
		}
		reservado, rol, err := agenteReservadoAutonomiaProyecto(strings.TrimSpace(asignacion.Agente), asignacion.ProyectoID)
		if err != nil {
			return err
		}
		if !reservado || rol != "supervisor" {
			continue
		}
		nota := strings.ToLower(strings.TrimSpace(asignacion.Nota))
		if nota == "supervision" || nota == "supervision_automatica" || nota == "server_autobootstrap" {
			continue
		}
		if _, err := DB.Exec(`UPDATE asignaciones SET nota='supervision_automatica' WHERE id=?`, asignacion.ID); err != nil {
			return err
		}
		Audit("sistema", "normalizar_supervisor_reservado_activo", "asignacion", asignacion.ID,
			fmt.Sprintf("agente=%s proyecto_id=%d nota_anterior=%s", strings.TrimSpace(asignacion.Agente), asignacion.ProyectoID, strings.TrimSpace(asignacion.Nota)))
	}
	return nil
}

func reconciliarRuntimeMailboxWorkerEnSupervisorReservado() error {
	rows, err := DB.Query(`
		SELECT id, to_agente, proyecto_id, kind, payload_json
		FROM runtime_mailbox
		WHERE estado IN ('pendiente','entregado')
		  AND proyecto_id IS NOT NULL
		  AND trim(COALESCE(to_agente,'')) <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		agente     string
		proyectoID int64
		kind       string
		payloadRaw string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.agente, &it.proyectoID, &it.kind, &it.payloadRaw); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		reservado, rol, err := agenteReservadoAutonomiaProyecto(strings.TrimSpace(it.agente), it.proyectoID)
		if err != nil {
			return err
		}
		if !reservado || (rol != "supervisor" && rol != "reviewer") {
			continue
		}
		payload := mapFromJSON(it.payloadRaw)
		if payloadRuntimeWorkerEnSupervisor(payload, strings.TrimSpace(it.kind)) == 0 {
			continue
		}
		if _, err := DB.Exec(`UPDATE runtime_mailbox SET estado='cancelado' WHERE id=?`, it.id); err != nil {
			return err
		}
		Audit("sistema", "cancelar_mailbox_worker_agente_reservado_autonomia", "runtime_mailbox", it.id,
			fmt.Sprintf("rol=%s agente=%s proyecto_id=%d kind=%s tarea_id=%d", rol, strings.TrimSpace(it.agente), it.proyectoID, strings.TrimSpace(it.kind), payloadRuntimeWorkerEnSupervisor(payload, strings.TrimSpace(it.kind))))
	}
	return nil
}

func reconciliarRuntimeOrdersWorkerEnSupervisorReservado() error {
	rows, err := DB.Query(`
		SELECT id, agente, proyecto_id, tipo, payload_json
		FROM runtime_orders
		WHERE estado IN ('pendiente','notificada','ejecutando','retenida')
		  AND proyecto_id IS NOT NULL
		  AND trim(COALESCE(agente,'')) <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		agente     string
		proyectoID int64
		tipo       string
		payloadRaw string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.agente, &it.proyectoID, &it.tipo, &it.payloadRaw); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		reservado, rol, err := agenteReservadoAutonomiaProyecto(strings.TrimSpace(it.agente), it.proyectoID)
		if err != nil {
			return err
		}
		if !reservado || (rol != "supervisor" && rol != "reviewer") {
			continue
		}
		payload := mapFromJSON(it.payloadRaw)
		tareaID := payloadRuntimeWorkerEnSupervisor(payload, strings.TrimSpace(it.tipo))
		if tareaID == 0 {
			continue
		}
		if _, err := DB.Exec(`UPDATE runtime_orders SET estado='cancelada', error_text='supervisor_reserved_worker_task' WHERE id=?`, it.id); err != nil {
			return err
		}
		Audit("sistema", "cancelar_runtime_order_worker_agente_reservado_autonomia", "runtime_order", it.id,
			fmt.Sprintf("rol=%s agente=%s proyecto_id=%d tipo=%s tarea_id=%d", rol, strings.TrimSpace(it.agente), it.proyectoID, strings.TrimSpace(it.tipo), tareaID))
	}
	return nil
}

func payloadRuntimeWorkerEnSupervisor(payload map[string]any, kind string) int64 {
	if len(payload) == 0 {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "accion", "")), "supervisar_proyecto") {
		return 0
	}
	tareaID := int64FromAny(payload["tarea_objetivo_id"])
	if tareaID <= 0 {
		return 0
	}
	kind = strings.TrimSpace(kind)
	payloadKind := strings.TrimSpace(stringFromMap(payload, "kind", ""))
	source := strings.TrimSpace(stringFromMap(payload, "source", ""))
	if strings.EqualFold(kind, "pipeline_local") || strings.EqualFold(payloadKind, "pipeline_local") || strings.EqualFold(source, "pipeline_local") {
		return tareaID
	}
	return tareaID
}

func reconciliarRuntimeMailboxPipelineFantasma() error {
	rows, err := DB.Query(`
		SELECT id, to_agente, proyecto_id, kind, payload_json
		FROM runtime_mailbox
		WHERE estado IN ('pendiente','entregado')
		  AND proyecto_id IS NOT NULL
		  AND trim(COALESCE(to_agente,'')) <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		agente     string
		proyectoID int64
		kind       string
		payloadRaw string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.agente, &it.proyectoID, &it.kind, &it.payloadRaw); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		payload := mapFromJSON(it.payloadRaw)
		if !runtimePayloadEsPipelineFantasma(payload, strings.TrimSpace(it.kind), strings.TrimSpace(it.agente), it.proyectoID) {
			continue
		}
		if _, err := DB.Exec(`UPDATE runtime_mailbox SET estado='cancelado' WHERE id=?`, it.id); err != nil {
			return err
		}
		Audit("sistema", "cancelar_mailbox_pipeline_fantasma", "runtime_mailbox", it.id,
			fmt.Sprintf("agente=%s proyecto_id=%d kind=%s", strings.TrimSpace(it.agente), it.proyectoID, strings.TrimSpace(it.kind)))
	}
	return nil
}

func reconciliarRuntimeOrdersPipelineFantasma() error {
	rows, err := DB.Query(`
		SELECT id, agente, proyecto_id, tipo, payload_json
		FROM runtime_orders
		WHERE estado IN ('pendiente','notificada','ejecutando','retenida')
		  AND proyecto_id IS NOT NULL
		  AND trim(COALESCE(agente,'')) <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		agente     string
		proyectoID int64
		tipo       string
		payloadRaw string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.agente, &it.proyectoID, &it.tipo, &it.payloadRaw); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		payload := mapFromJSON(it.payloadRaw)
		if !runtimePayloadEsPipelineFantasma(payload, strings.TrimSpace(it.tipo), strings.TrimSpace(it.agente), it.proyectoID) {
			continue
		}
		if _, err := DB.Exec(`UPDATE runtime_orders SET estado='cancelada', error_text='ghost_pipeline_without_task' WHERE id=?`, it.id); err != nil {
			return err
		}
		Audit("sistema", "cancelar_runtime_order_pipeline_fantasma", "runtime_order", it.id,
			fmt.Sprintf("agente=%s proyecto_id=%d tipo=%s", strings.TrimSpace(it.agente), it.proyectoID, strings.TrimSpace(it.tipo)))
	}
	return nil
}

func runtimePayloadEsPipelineFantasma(payload map[string]any, kind, agente string, proyectoID int64) bool {
	if len(payload) == 0 || proyectoID <= 0 {
		return false
	}
	kind = strings.TrimSpace(kind)
	payloadKind := strings.TrimSpace(stringFromMap(payload, "kind", ""))
	source := strings.TrimSpace(stringFromMap(payload, "source", ""))
	if !strings.EqualFold(kind, "pipeline_local") && !strings.EqualFold(payloadKind, "pipeline_local") && !strings.EqualFold(source, "pipeline_local") {
		return false
	}
	if int64FromAny(payload["tarea_objetivo_id"]) > 0 {
		return false
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false
	}
	if asignacion, err := GetAsignacionActivaAgente(agente); err == nil && asignacion != nil && asignacion.ProyectoID == proyectoID {
		return false
	}
	tareaActivaID, err := GetTareaActivaIDPorAgenteProyecto(agente, &proyectoID)
	if err != nil {
		return false
	}
	return tareaActivaID == 0
}

func reconciliarAsignacionesDuplicadasAgenteProyecto() error {
	rows, err := DB.Query(`
		SELECT id, agente, proyecto_id, estado
		FROM asignaciones
		WHERE estado IN ('planificada','activa','pausada')
		  AND trim(COALESCE(agente,'')) <> ''
		ORDER BY proyecto_id, agente, id DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		agente     string
		proyectoID int64
		estado     string
	}
	var items []item
	grupos := map[string][]item{}
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.agente, &it.proyectoID, &it.estado); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, it := range items {
		canonico, err := CanonicalizeAgentName(strings.TrimSpace(it.agente))
		if err != nil {
			return err
		}
		clave := fmt.Sprintf("%d:%s", it.proyectoID, strings.TrimSpace(canonico))
		grupos[clave] = append(grupos[clave], it)
	}

	for _, items := range grupos {
		if len(items) <= 1 {
			continue
		}
		keep := items[0]
		for _, it := range items[1:] {
			if reconciliarAsignacionStateRank(strings.TrimSpace(it.estado)) > reconciliarAsignacionStateRank(strings.TrimSpace(keep.estado)) {
				keep = it
			}
		}
		for _, it := range items {
			if it.id == keep.id {
				continue
			}
			if _, err := DB.Exec(`
				UPDATE asignaciones
				SET estado='cerrada',
				    nota=?,
				    cerrada_at=CURRENT_TIMESTAMP
				WHERE id=?`,
				fmt.Sprintf("duplicada_compactada:%d", keep.id),
				it.id,
			); err != nil {
				return err
			}
			Audit("sistema", "compactar_asignacion_duplicada", "asignacion", it.id,
				fmt.Sprintf("keep_id=%d agente=%s proyecto_id=%d estado=%s",
					keep.id,
					strings.TrimSpace(keep.agente),
					keep.proyectoID,
					strings.TrimSpace(it.estado),
				))
		}
	}
	return nil
}

func reconciliarAsignacionStateRank(estado string) int {
	switch strings.TrimSpace(estado) {
	case "activa":
		return 3
	case "planificada":
		return 2
	case "pausada":
		return 1
	default:
		return 0
	}
}

func reconciliarRuntimeOrdersAliasNoCanonico() error {
	rows, err := DB.Query(`
		SELECT id, agente, tipo, payload_json
		FROM runtime_orders
		WHERE estado IN ('pendiente','tomada','ejecutando')
		  AND trim(COALESCE(agente,'')) <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id         int64
		agente     string
		tipo       string
		payloadRaw string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.agente, &it.tipo, &it.payloadRaw); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		agenteCanonico, err := CanonicalizeAgentName(strings.TrimSpace(it.agente))
		if err != nil {
			return err
		}

		payloadChanged := false
		payload, err := runtimeOrderPayloadMap(it.payloadRaw)
		if err != nil {
			return err
		}

		switch strings.TrimSpace(it.tipo) {
		case "handoff":
			for _, key := range []string{"agente_origen", "agente_destino"} {
				text, ok := payload[key].(string)
				if !ok || strings.TrimSpace(text) == "" {
					continue
				}
				canonico, err := CanonicalizeAgentName(text)
				if err != nil {
					return err
				}
				if canonico != "" && canonico != strings.TrimSpace(text) {
					payload[key] = canonico
					payloadChanged = true
				}
			}
		}

		agentChanged := agenteCanonico != "" && agenteCanonico != strings.TrimSpace(it.agente)
		if !agentChanged && !payloadChanged {
			continue
		}

		payloadJSON := it.payloadRaw
		if payloadChanged {
			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			payloadJSON = string(data)
		}
		agentePersistido := strings.TrimSpace(it.agente)
		if strings.TrimSpace(agenteCanonico) != "" {
			agentePersistido = strings.TrimSpace(agenteCanonico)
		}
		if _, err := DB.Exec(
			`UPDATE runtime_orders SET agente=?, payload_json=? WHERE id=?`,
			agentePersistido,
			payloadJSON,
			it.id,
		); err != nil {
			return err
		}
		Audit("sistema", "runtime_order_alias_reconciled", "runtime_order", it.id,
			fmt.Sprintf("agente=%s canonico=%s tipo=%s payload_changed=%t",
				strings.TrimSpace(it.agente),
				agentePersistido,
				strings.TrimSpace(it.tipo),
				payloadChanged,
			))
	}
	return nil
}

func reconciliarRuntimeMailboxAliasNoCanonico() error {
	rows, err := DB.Query(`
		SELECT id, from_agente, to_agente, payload_json
		FROM runtime_mailbox
		WHERE estado IN ('pendiente','entregado')
		  AND trim(COALESCE(to_agente,'')) <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id        int64
		fromAgent string
		toAgent   string
		payload   string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.fromAgent, &it.toAgent, &it.payload); err != nil {
			return err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, it := range items {
		fromCanonico, err := CanonicalizeAgentName(strings.TrimSpace(it.fromAgent))
		if err != nil && !strings.EqualFold(strings.TrimSpace(it.fromAgent), "server") {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(it.fromAgent), "server") {
			fromCanonico = "server"
		}
		toCanonico, err := CanonicalizeAgentName(strings.TrimSpace(it.toAgent))
		if err != nil {
			return err
		}
		payloadJSON, err := normalizarRuntimeMailboxPayloadJSON(it.payload, fromCanonico, toCanonico)
		if err != nil {
			return err
		}
		if _, err := DB.Exec(`
			UPDATE runtime_mailbox
			SET from_agente = ?, to_agente = ?, payload_json = ?
			WHERE id = ?`,
			fromCanonico,
			toCanonico,
			payloadJSON,
			it.id,
		); err != nil {
			return err
		}
	}
	return nil
}
