package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"orquesta/coordinacion"
	"orquesta/internal/controlruntime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func ReconciliarRuntimeHandlesStale() (int, error) {
	staleSeconds := configIntOrDefault("runtime_handle_stale_seconds", 120)
	if staleSeconds <= 0 {
		staleSeconds = 120
	}
	cutoff := time.Now().UTC().Add(-time.Duration(staleSeconds) * time.Second)
	rows, err := DB.Query(`
		SELECT id
		FROM runtime_handles
		WHERE estado IN ('activo','pausado')
		  AND COALESCE(last_seen_at, created_at) <= ?`, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	reconciled := 0
	for _, id := range ids {
		handle, err := GetRuntimeHandle(id)
		if err != nil {
			return 0, err
		}
		if superseded, err := supersedeRuntimeHandleSiHaceFalta(handle); err != nil {
			return 0, err
		} else if superseded {
			reconciled++
			continue
		}
		if revived, err := refrescarRuntimeHandleSiSigueVivo(handle); err != nil {
			return 0, err
		} else if revived != nil {
			continue
		}
		res, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
			WHERE id = ?
			  AND estado IN ('activo','pausado')`, id)
		if err != nil {
			return 0, err
		}
		rows, _ := res.RowsAffected()
		if rows > 0 {
			runtimeHandleHotReset()
			reconciled++
		}
	}
	return reconciled, nil
}

func ReconciliarRuntimeOrdersStale() (int, error) {
	now := time.Now().UTC()
	staleSeconds := configIntOrDefault("runtime_order_stale_seconds", 120)
	if staleSeconds <= 0 {
		staleSeconds = 120
	}
	cutoff := now.Add(-time.Duration(staleSeconds) * time.Second)
	candidates := make([]*RuntimeOrder, 0, 16)
	for _, estado := range []string{"tomada", "ejecutando"} {
		estado := estado
		orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
		if err != nil {
			return 0, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			if order.LeaseExpiresAt != nil && !order.LeaseExpiresAt.After(now) {
				candidates = append(candidates, order)
				continue
			}
			if runtimeOrderAttemptReference(order).After(cutoff) {
				continue
			}
			candidates = append(candidates, order)
		}
	}
	recovered := 0
	for _, item := range candidates {
		applied, err := reconciliarRuntimeOrderStale(item.ID, item.Tipo, now, cutoff)
		if err != nil {
			return recovered, err
		}
		if applied {
			recovered++
		}
	}
	return recovered, nil
}

func ReconciliarRuntimeOrdersPendientesHandoffExpiradas() (int, error) {
	minutes := configIntOrDefault("runtime_pending_handoff_expiry_minutes", 24*60)
	if minutes <= 0 {
		minutes = 24 * 60
	}
	cutoff := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	rows, err := DB.Query(runtimeOrderSelectBase()+`
		WHERE estado = 'pendiente'
		  AND tipo = 'handoff'
		  AND created_at <= ?
		ORDER BY id ASC`, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	orders := make([]*RuntimeOrder, 0, 16)
	for rows.Next() {
		order, err := scanRuntimeOrder(rows)
		if err != nil {
			return 0, err
		}
		if order != nil {
			orders = append(orders, order)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	expired := 0
	for _, order := range orders {
		if order == nil {
			continue
		}
		if handle, err := GetRuntimeHandleOperativoRecienteAgenteProyecto(strings.TrimSpace(order.Agente), order.ProyectoID); err != nil {
			return expired, err
		} else if handle != nil {
			continue
		}
		if runtime, err := GetRuntimePrincipalAgenteProyecto(strings.TrimSpace(order.Agente), order.ProyectoID); err != nil {
			return expired, err
		} else if runtimePendingHandoffStillActive(runtime, cutoff) {
			continue
		}
		resultado := map[string]any{
			"ok":      false,
			"expired": true,
			"reason":  "pending_handoff_hygiene_timeout",
		}
		data, _ := json.Marshal(resultado)
		if _, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado = 'expirada',
			    resultado_json = ?,
			    error_text = CASE
			        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'handoff pendiente expirada por higiene'
			        ELSE error_text || ?
			    END,
			    finished_at = CURRENT_TIMESTAMP,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL
			WHERE id = ?
			  AND estado = 'pendiente'
			  AND tipo = 'handoff'`, string(data), "\nhandoff pendiente expirada por higiene", order.ID); err != nil {
			return expired, err
		}
		expired++
	}
	return expired, nil
}

func runtimePendingHandoffStillActive(runtime *RuntimeInstance, cutoff time.Time) bool {
	if runtime == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "", "cerrado", "fallido":
		return false
	}
	last := runtimeMomentForRecovery(runtime)
	if last.IsZero() {
		return false
	}
	return last.After(cutoff)
}

func ProcesarRuntimeOrdersBatch() (int, error) {
	stale, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		return 0, err
	}
	reconciled, err := reconciliarRuntimeOrdersPendientesMailboxConsumido()
	if err != nil {
		return stale, err
	}
	controlEjecutando, err := reconciliarRuntimeOrdersEjecutandoControlSatisfechas()
	if err != nil {
		return stale + reconciled, err
	}
	controlObsoletas, err := reconciliarRuntimeOrdersPendientesControlObsoletas()
	if err != nil {
		return stale + reconciled + controlEjecutando, err
	}
	processed, err := procesarRuntimeOrdersBatchTipos(runtimeOrderTiposDespachables())
	if err != nil {
		return stale + reconciled + controlEjecutando + controlObsoletas + processed, err
	}
	promoted, err := procesarBootstrapRuntimeOrdersActivosBatch()
	if err != nil {
		return stale + reconciled + controlEjecutando + controlObsoletas + processed + promoted, err
	}
	deferred, err := procesarRuntimeOrdersBatchTipos(runtimeOrderTiposDiferibles())
	return stale + reconciled + controlEjecutando + controlObsoletas + processed + promoted + deferred, err
}

func ProcesarHigieneRuntimesAutonomosBatch() (int, error) {
	mailboxObsoleta, err := reconciliarRuntimeMailboxPendientePorTarea()
	if err != nil {
		return 0, err
	}
	worktreesObsoletas, err := reconciliarWorktreesActivasPorTarea()
	if err != nil {
		return mailboxObsoleta, err
	}
	rows, err := DB.Query(runtimeHandleSelectBase() + `
		WHERE estado='activo' AND metadata_json LIKE '%"modo":"autonomo"%'`)
	if err != nil {
		return mailboxObsoleta + worktreesObsoletas, err
	}
	defer rows.Close()

	var handles []*RuntimeHandle
	for rows.Next() {
		handle, err := scanRuntimeHandle(rows)
		if err != nil {
			return 0, err
		}
		if handle != nil {
			handles = append(handles, handle)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		meta := mapFromJSON(handle.MetadataJSON)
		tareaID := int64FromMap(meta, "tarea_id")
		if tareaID <= 0 {
			continue
		}
		tarea, err := GetTarea(tareaID)
		if err != nil || tarea == nil {
			continue
		}
		if tarea.Estado == TareaCompletada || tarea.Estado == TareaCancelada {
			lastSeen := handle.LastSeenAt
			if lastSeen == nil {
				lastSeen = &handle.UpdatedAt
			}
			idleLimit := time.Duration(configIntOrDefault("runtime_autonomo_idle_timeout_seconds", 300)) * time.Second
			if time.Since(*lastSeen) > idleLimit {
				order := &RuntimeOrder{
					Agente:      strings.TrimSpace(handle.Agente),
					ProyectoID:  handle.ProyectoID,
					Tipo:        "stop",
					PayloadJSON: `{"reason": "higiene_autonoma:tarea_finalizada", "source": "server"}`,
					Estado:      "pending",
				}
				if _, err := EncolarRuntimeOrder(order); err != nil {
					return mailboxObsoleta + worktreesObsoletas + processed, err
				}
				Audit("server", "runtime_higiene_autonomo_stop_encolado", "handle", handle.ID, "agente: "+handle.Agente)
				processed++
			}
		}
	}
	handlesSinTrabajo, err := listarRuntimeHandlesActivosSupervisados()
	if err != nil {
		return mailboxObsoleta + worktreesObsoletas + processed, err
	}
	for _, handle := range handlesSinTrabajo {
		if handle == nil {
			continue
		}
		apagar, err := runtimeHandleDebeApagarsePorHigieneSinTrabajo(handle)
		if err != nil {
			return mailboxObsoleta + worktreesObsoletas + processed, err
		}
		if !apagar {
			continue
		}
		if abierta, err := existeRuntimeOrderAbierta(strings.TrimSpace(handle.Agente), handle.ProyectoID, "stop"); err != nil {
			return processed, err
		} else if abierta {
			continue
		}
		order := &RuntimeOrder{
			Agente:      strings.TrimSpace(handle.Agente),
			ProyectoID:  handle.ProyectoID,
			RuntimeID:   handle.RuntimeID,
			HandleID:    &handle.ID,
			Tipo:        "stop",
			PayloadJSON: `{"reason": "higiene_autonoma:sin_trabajo_pendiente", "source": "server"}`,
			Estado:      "pending",
		}
		if _, err := EncolarRuntimeOrder(order); err != nil {
			return mailboxObsoleta + worktreesObsoletas + processed, err
		}
		Audit("server", "runtime_higiene_stop_sin_trabajo_encolado", "handle", handle.ID,
			"agente: "+handle.Agente)
		processed++
	}
	orphanTMUX, err := purgarSesionesTMUXHuerfanasConCurrentPathBorrado()
	if err != nil {
		return mailboxObsoleta + worktreesObsoletas + processed, err
	}
	processed += orphanTMUX
	worktreesCerradas, err := purgarRutasWorktreeCerradasOFallidas()
	if err != nil {
		return mailboxObsoleta + worktreesObsoletas + processed, err
	}
	processed += worktreesCerradas
	historico, err := PurgarRuntimeHistorico()
	if err != nil {
		return mailboxObsoleta + worktreesObsoletas + processed, err
	}
	processed += contarPurgaRuntimeHistorico(historico)
	agentesInactivos, err := reconciliarAgentesActivosSinVida()
	if err != nil {
		return mailboxObsoleta + worktreesObsoletas + processed, err
	}
	processed += agentesInactivos
	return mailboxObsoleta + worktreesObsoletas + processed, nil
}

func reconciliarRuntimeMailboxPendientePorTarea() (int, error) {
	estado := "pendiente"
	items, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{Estado: &estado})
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, msg := range items {
		if msg == nil {
			continue
		}
		taskID := runtimeMailboxPayloadTaskIDJSON(msg.PayloadJSON)
		if taskID <= 0 {
			continue
		}
		obsoleta, err := runtimeMailboxPendienteObsoletaPorTarea(msg, taskID)
		if err != nil {
			return processed, err
		}
		if !obsoleta {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return processed, err
		}
		if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func runtimeMailboxPendienteObsoletaPorTarea(msg *RuntimeMailboxMessage, taskID int64) (bool, error) {
	if msg == nil || taskID <= 0 {
		return false, nil
	}
	tarea, err := GetTarea(taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	if tarea == nil {
		return true, nil
	}
	if runtimeTaskNoLongerActionable(tarea) {
		return true, nil
	}
	target := strings.TrimSpace(msg.ToAgente)
	if tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) == "" {
		return true, nil
	}
	if target != "" && !strings.EqualFold(strings.TrimSpace(*tarea.Agente), target) {
		return true, nil
	}
	if msg.ProyectoID != nil && tarea.ProyectoID != nil && *msg.ProyectoID != *tarea.ProyectoID {
		return true, nil
	}
	return false, nil
}

func runtimeTaskNoLongerActionable(tarea *Tarea) bool {
	if tarea == nil {
		return true
	}
	switch tarea.Estado {
	case TareaCompletada, TareaCancelada, TareaLibre, TareaBacklog:
		return true
	default:
		return false
	}
}

func reconciliarWorktreesActivasPorTarea() (int, error) {
	estado := coordinacion.WorktreeActive
	items, err := (CoordinationWorktreeSQLRepository{}).ListRaw(coordinacion.WorktreeFilter{State: &estado})
	if err != nil {
		return 0, err
	}
	processed := 0
	grouped := map[string][]*coordinacion.Worktree{}
	for _, item := range items {
		if item == nil || item.TaskID == nil || *item.TaskID <= 0 {
			continue
		}
		tarea, err := GetTarea(*item.TaskID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return processed, err
		}
		if err != nil || tarea == nil || runtimeTaskNoLongerActionable(tarea) ||
			tarea.ProyectoID == nil || *tarea.ProyectoID != item.ProjectID ||
			tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) == "" ||
			!strings.EqualFold(strings.TrimSpace(*tarea.Agente), strings.TrimSpace(item.Agent)) {
			closed, err := cerrarWorktreeActivaPorHigiene(item, "higiene_task_orphaned")
			if err != nil {
				return processed, err
			}
			if closed {
				processed++
			}
			continue
		}
		key := runtimeHygieneTaskGroupKey(item.ProjectID, *item.TaskID)
		grouped[key] = append(grouped[key], item)
	}
	for _, group := range grouped {
		if len(group) <= 1 {
			continue
		}
		keep := seleccionarWorktreeActivaPreferidaPorTarea(group)
		for _, item := range group {
			if item == nil || keep == nil || item.ID == keep.ID {
				continue
			}
			closed, err := cerrarWorktreeActivaPorHigiene(item, "higiene_task_duplicate")
			if err != nil {
				return processed, err
			}
			if closed {
				processed++
			}
		}
	}
	return processed, nil
}

func runtimeHygieneTaskGroupKey(projectID, taskID int64) string {
	return jsonNumber(projectID) + ":" + jsonNumber(taskID)
}

func seleccionarWorktreeActivaPreferidaPorTarea(items []*coordinacion.Worktree) *coordinacion.Worktree {
	candidates := append([]*coordinacion.Worktree(nil), items...)
	sort.SliceStable(candidates, func(i, j int) bool {
		a := candidates[i]
		b := candidates[j]
		scoreA := runtimeHygieneWorktreeScore(a)
		scoreB := runtimeHygieneWorktreeScore(b)
		if scoreA != scoreB {
			return scoreA > scoreB
		}
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.ID > b.ID
	})
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

func runtimeHygieneWorktreeScore(item *coordinacion.Worktree) int {
	if item == nil {
		return -1
	}
	score := 0
	if coordinacion.WorktreePathUsable(item.Path) {
		score += 2
	}
	if strings.TrimSpace(item.Agent) != "" {
		score++
	}
	return score
}

func cerrarWorktreeActivaPorHigiene(item *coordinacion.Worktree, reason string) (bool, error) {
	if item == nil || item.ID <= 0 {
		return false, nil
	}
	closed, err := (CoordinationWorktreeSQLRepository{}).Close(item.ID, time.Now().UTC(), reason)
	if err != nil {
		return false, nil
	}
	if item.Path != "" {
		_ = os.RemoveAll(strings.TrimSpace(item.Path))
	}
	return closed != nil, nil
}

func purgarRutasWorktreeCerradasOFallidas() (int, error) {
	processed := 0
	for _, state := range []coordinacion.WorktreeState{coordinacion.WorktreeClosed, coordinacion.WorktreeFailed} {
		state := state
		items, err := (CoordinationWorktreeSQLRepository{}).ListRaw(coordinacion.WorktreeFilter{State: &state})
		if err != nil {
			return processed, err
		}
		for _, item := range items {
			if item == nil || strings.TrimSpace(item.Path) == "" {
				continue
			}
			shouldRemove, err := runtimeHygieneShouldRemoveClosedWorktreePath(item)
			if err != nil {
				return processed, err
			}
			if !shouldRemove {
				continue
			}
			if err := os.RemoveAll(strings.TrimSpace(item.Path)); err != nil {
				return processed, err
			}
			processed++
		}
	}
	return processed, nil
}

func runtimeHygieneShouldRemoveClosedWorktreePath(item *coordinacion.Worktree) (bool, error) {
	if item == nil || strings.TrimSpace(item.Path) == "" {
		return false, nil
	}
	path := strings.TrimSpace(item.Path)
	if !coordinacion.WorktreePathUsable(path) {
		return false, nil
	}
	proyecto, err := GetProyectoConRutaEfectiva(jsonNumber(item.ProjectID), "")
	if err != nil || proyecto == nil {
		return false, err
	}
	root := filepath.Join(strings.TrimSpace(proyecto.RutaAbs), ".orquesta-worktrees")
	if !coordinacion.PathWithin(root, path) {
		return false, nil
	}
	return true, nil
}

func contarPurgaRuntimeHistorico(resultado *PurgaRuntimeHistoricoResultado) int {
	if resultado == nil {
		return 0
	}
	total := 0
	if resultado.Handles != nil {
		total += resultado.Handles.Deleted
	}
	if resultado.Orders != nil {
		total += resultado.Orders.Deleted
	}
	if resultado.Runtimes != nil {
		total += resultado.Runtimes.Deleted
	}
	return total
}

func listarRuntimeHandlesActivosSupervisados() ([]*RuntimeHandle, error) {
	rows, err := DB.Query(runtimeHandleSelectBase() + `
		WHERE estado='activo'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*RuntimeHandle, 0, 16)
	for rows.Next() {
		handle, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		if handle == nil || !runtimeHandlePareceSupervisadoPorOrquesta(handle) {
			continue
		}
		out = append(out, handle)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func runtimeHandlePareceSupervisadoPorOrquesta(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if strings.EqualFold(strings.TrimSpace(runtimeHandleMetadataString(meta, "modo")), "autonomo") {
		return true
	}
	if strings.TrimSpace(runtimeHandleMetadataString(meta, "supervisor_driver")) != "" {
		return true
	}
	if int64FromMap(meta, "supervisor_owner_pid") > 0 {
		return true
	}
	return false
}

func runtimeHandleDebeApagarsePorHigieneSinTrabajo(handle *RuntimeHandle) (bool, error) {
	if handle == nil || handle.ProyectoID == nil || *handle.ProyectoID <= 0 {
		return false, nil
	}
	agente := strings.TrimSpace(handle.Agente)
	if agente == "" {
		return false, nil
	}
	idleLimit := time.Duration(configIntOrDefault("runtime_autonomo_idle_timeout_seconds", 300)) * time.Second
	if idleLimit <= 0 {
		idleLimit = 5 * time.Minute
	}
	reference := handle.CreatedAt
	if !handle.UpdatedAt.IsZero() && handle.UpdatedAt.Before(reference) {
		reference = handle.UpdatedAt
	}
	if reference.IsZero() {
		reference = time.Now().UTC()
	}
	if time.Since(reference) < idleLimit {
		return false, nil
	}
	if asignacion, err := GetAsignacionActivaAgente(agente); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if asignacion != nil && asignacion.ProyectoID == *handle.ProyectoID {
		return false, nil
	}
	if tareaID, err := GetTareaActivaIDPorAgenteProyecto(agente, handle.ProyectoID); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if tareaID > 0 {
		return false, nil
	}
	estadoPendiente := "pendiente"
	if mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: handle.ProyectoID, Estado: &estadoPendiente, Limit: 1}); err != nil {
		return false, err
	} else if len(mailbox) > 0 {
		return false, nil
	}
	return true, nil
}

func runtimeHandleMetadataString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	raw, ok := meta[key]
	if !ok {
		return ""
	}
	text, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func reconciliarAgentesActivosSinVida() (int, error) {
	rows, err := DB.Query(`SELECT nombre FROM agentes WHERE activo=1 AND habilitado=1`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	names := make([]string, 0, 16)
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return 0, err
		}
		nombre = strings.TrimSpace(nombre)
		if nombre != "" {
			names = append(names, nombre)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0
	for _, agente := range names {
		ok, err := agenteActivoDebeDesactivarsePorHigiene(agente)
		if err != nil {
			return processed, err
		}
		if !ok {
			continue
		}
		if _, err := DB.Exec(`UPDATE agentes SET activo=0, estado_sesion='disponible' WHERE nombre=? AND activo=1`, agente); err != nil {
			return processed, err
		}
		Audit("server", "runtime_higiene_agente_inactivo_sin_vida", "agente", 0, "agente: "+agente)
		processed++
	}
	return processed, nil
}

func agenteActivoDebeDesactivarsePorHigiene(agente string) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	if asignacion, err := GetAsignacionActivaAgente(agente); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if asignacion != nil {
		return false, nil
	}
	if tareaID, err := GetTareaActivaIDPorAgente(agente); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if tareaID > 0 {
		return false, nil
	}
	estadoPendiente := "pendiente"
	if mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estadoPendiente, Limit: 1}); err != nil {
		return false, err
	} else if len(mailbox) > 0 {
		return false, nil
	}
	var sesionesActivas int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE agente=? AND activa=1`, agente).Scan(&sesionesActivas); err != nil {
		return false, err
	}
	if sesionesActivas > 0 {
		return false, nil
	}
	var handlesActivos int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_handles WHERE agente=? AND estado IN ('activo','pausado')`, agente).Scan(&handlesActivos); err != nil {
		return false, err
	}
	if handlesActivos > 0 {
		return false, nil
	}
	var runtimesVivos int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_instances WHERE agente=? AND lower(trim(coalesce(logical_state,''))) IN ('activo','active','running','iniciando','starting','esperando_io','waiting_io','pausado','paused')`, agente).Scan(&runtimesVivos); err != nil {
		return false, err
	}
	if runtimesVivos > 0 {
		return false, nil
	}
	return true, nil
}

func purgarSesionesTMUXHuerfanasConCurrentPathBorrado() (int, error) {
	tmuxCommand, err := runtimeTMUXCommandPathFn()
	if err != nil || strings.TrimSpace(tmuxCommand) == "" {
		return 0, nil
	}
	sessions, err := runtimeTMUXListSessionsFn(tmuxCommand)
	if err != nil {
		return 0, err
	}
	referenciadas, err := runtimeTMUXSessionsActivasReferenciadas()
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, session := range sessions {
		sessionName := strings.TrimSpace(session.SessionName)
		if !strings.HasPrefix(sessionName, "orq-") {
			continue
		}
		if _, ok := referenciadas[sessionName]; ok {
			continue
		}
		currentPath, deleted := controlruntime.NormalizeTMUXCurrentPath(session.CurrentPath)
		if !deleted {
			if strings.TrimSpace(currentPath) == "" {
				continue
			}
			if _, statErr := os.Stat(currentPath); statErr != nil {
				if os.IsNotExist(statErr) {
					deleted = true
				} else {
					continue
				}
			}
		}
		if !deleted {
			continue
		}
		if err := runtimeTMUXKillSessionFn(tmuxCommand, sessionName); err != nil {
			return purged, err
		}
		Audit("server", "runtime_hygiene_orphan_tmux_killed", "runtime_handle", 0, "tmux_session: "+sessionName)
		purged++
	}
	return purged, nil
}

func runtimeTMUXSessionsActivasReferenciadas() (map[string]struct{}, error) {
	rows, err := DB.Query(`SELECT handle_ref, metadata_json FROM runtime_handles WHERE estado='activo' AND LOWER(COALESCE(transporte, ''))='tmux'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var handleRef, metadataJSON string
		if err := rows.Scan(&handleRef, &metadataJSON); err != nil {
			return nil, err
		}
		sessionName := strings.TrimSpace(stringFromMap(mapFromJSON(metadataJSON), "tmux_session", ""))
		if sessionName == "" {
			sessionName = strings.TrimSpace(runtimeTMUXSessionNameFromHandleRef(handleRef))
		}
		if sessionName == "" {
			continue
		}
		out[sessionName] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func runtimeTMUXSessionNameFromHandleRef(handleRef string) string {
	handleRef = strings.TrimSpace(handleRef)
	if handleRef == "" {
		return ""
	}
	if idx := strings.Index(handleRef, "/"); idx >= 0 {
		handleRef = handleRef[:idx]
	}
	return strings.TrimSpace(handleRef)
}
