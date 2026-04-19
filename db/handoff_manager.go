/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

// Package db — HandoffManager (OP-089)
//
// El HandoffManager implementa el relevo automático de agentes por agotamiento
// de tokens o inactividad prolongada. El flujo es:
//
//  1. El control plane llama a ProcesarHandoffsBatch() cada N segundos.
//  2. DetectarAgentesAgotados() busca agentes con tareas en_progreso cuyo
//     heartbeat de sesión lleva más de pool_handoff_threshold_seconds sin
//     actualizarse.
//  3. Por cada candidato, SeleccionarAgenteReemplazo() elige el siguiente agente
//     disponible (habilitado, sin sesión activa, distinto del origen).
//  4. GuardarCheckpointHandoff() guarda un checkpoint automático con el contexto
//     del agente saliente.
//  5. CrearHandoffAgenteVivo() reasigna la tarea y crea la RuntimeOrder de tipo
//     "handoff" para el agente entrante con el resumen de continuidad.
//  6. El agente entrante, al arrancar su sesión, encuentra la orden en su buzón
//     y recibe el contexto inyectado automáticamente.
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// HandoffCandidato describe un agente que cumple las condiciones para ser
// relevado automáticamente.
type HandoffCandidato struct {
	Agente         string        // agente con la sesión inactiva o en presupuesto crítico
	TareaID        *int64        // tarea en progreso que tiene asignada
	SesionID       *int64        // sesión activa del agente
	HandleID       *int64        // runtime handle activo (puede ser nil)
	ProyectoID     *int64        // proyecto de la tarea/sesión
	Inactividad    time.Duration // tiempo desde el último heartbeat
	Motivo         string        // descripción del motivo de detección
	Disparador     string        // watchdog | presupuesto
	RequiereSondeo bool          // true si debe emitirse sync_status/nudge antes del handoff
}

// DetectarAgentesAgotados devuelve la lista de agentes que tienen tareas
// en_progreso y cumplen condiciones de relevo automático por watchdog o
// presupuesto. Excluye agentes que ya tienen una runtime_order de handoff
// pendiente o ejecutando para evitar duplicados.
func DetectarAgentesAgotados() ([]*HandoffCandidato, error) {
	threshold := time.Duration(configIntOrDefault("pool_handoff_threshold_seconds", 1800)) * time.Second
	if threshold <= 0 {
		threshold = 30 * time.Minute
	}
	cutoff := time.Now().UTC().Add(-threshold)

	rows, err := DB.Query(`
		SELECT
			s.agente,
			s.id            AS sesion_id,
			s.proyecto_id,
			s.heartbeat_at
		FROM sesiones s
		WHERE s.activa = 1
		  AND EXISTS (
			  SELECT 1
			  FROM tareas t
			  WHERE t.agente = s.agente
			    AND t.estado = 'en_progreso'
		  )
		  AND s.agente NOT IN (
			  SELECT DISTINCT agente
			  FROM runtime_orders
			  WHERE tipo = 'handoff'
			    AND estado IN ('pendiente','tomada','ejecutando')
		  )
		ORDER BY s.heartbeat_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var detectados []*HandoffCandidato
	for rows.Next() {
		var (
			c           HandoffCandidato
			sesionID    int64
			proyectoID  nullInt64
			heartbeatAt sql.NullTime
		)
		if err := rows.Scan(&c.Agente, &sesionID, &proyectoID, &heartbeatAt); err != nil {
			return nil, err
		}
		c.SesionID = &sesionID
		if proyectoID.Valid {
			c.ProyectoID = &proyectoID.Int64
		}
		if heartbeatAt.Valid {
			c.Inactividad = time.Since(heartbeatAt.Time)
		}
		detectados = append(detectados, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	candidatos := make([]*HandoffCandidato, 0, len(detectados))
	for _, c := range detectados {
		tarea, err := seleccionarTareaHandoffAgente(c.Agente, c.ProyectoID)
		if err != nil {
			return nil, err
		}
		if tarea == nil {
			continue
		}
		c.TareaID = &tarea.ID
		if tarea.ProyectoID != nil {
			c.ProyectoID = tarea.ProyectoID
		}
		handle, err := runtimeHandleCanonicoRecienteConFallback(c.Agente, c.ProyectoID)
		if err != nil {
			return nil, err
		}
		if handle != nil {
			c.HandleID = &handle.ID
			if c.ProyectoID == nil && handle.ProyectoID != nil {
				c.ProyectoID = handle.ProyectoID
			}
		} else if strings.TrimSpace(c.Disparador) == "watchdog" {
			c.RequiereSondeo = false
		}
		disparado, err := enriquecerCandidatoHandoff(c, cutoff, threshold)
		if err != nil {
			return nil, err
		}
		if !disparado {
			continue
		}
		candidatos = append(candidatos, c)
	}
	return candidatos, nil
}

func seleccionarTareaHandoffAgente(agente string, proyectoID *int64) (*Tarea, error) {
	q := `
		SELECT id, titulo, descripcion, proyecto_id, modulo, estado, agente, propuesta_id, prioridad,
		       dependencias, creado_por, commit_cierre, notas,
		       created_at, updated_at, completada_at, contrato_definido
		FROM tareas
		WHERE agente = ?
		  AND estado = 'en_progreso'`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += `
		ORDER BY
		  CASE WHEN proyecto_id = ? THEN 0 ELSE 1 END,
		  updated_at DESC,
		  id DESC
		LIMIT 1`
		args = append(args, *proyectoID)
	} else {
		q += `
		ORDER BY updated_at DESC, id DESC
		LIMIT 1`
	}
	tarea, err := escanearTarea(DB.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tarea, err
}

func enriquecerCandidatoHandoff(c *HandoffCandidato, cutoff time.Time, threshold time.Duration) (bool, error) {
	if c == nil {
		return false, nil
	}
	if watchdogYaSatisfechoPorEnfriamiento(c) {
		return false, nil
	}
	if c.SesionID != nil {
		presupuesto, err := UltimoPresupuestoSesion(*c.SesionID)
		if err != nil && err != sql.ErrNoRows {
			return false, err
		}
		if err == nil && presupuesto != nil && presupuestoSesionFresco(presupuesto) {
			ev, err := EvaluarPresupuestoSesion(presupuesto)
			if err != nil {
				return false, err
			}
			if ev != nil && ev.DebeHandoff {
				c.Disparador = "presupuesto"
				c.RequiereSondeo = false
				c.Motivo = strings.TrimSpace(ev.Motivo)
				return true, nil
			}
		}
	}
	if activa, err := watchdogActividadRuntimeReciente(c, cutoff); err != nil {
		return false, err
	} else if activa {
		return false, nil
	}
	if c.Inactividad >= threshold || (c.Inactividad > 0 && time.Now().UTC().Add(-c.Inactividad).Before(cutoff)) {
		c.Disparador = "watchdog"
		c.RequiereSondeo = c.HandleID != nil
		c.Motivo = fmt.Sprintf("heartbeat hace %.0f min (umbral: %.0f min)",
			c.Inactividad.Minutes(), threshold.Minutes())
		if c.HandleID == nil {
			c.Motivo += "; sin runtime handle activo"
		}
		return true, nil
	}
	return false, nil
}

func watchdogYaSatisfechoPorEnfriamiento(c *HandoffCandidato) bool {
	if c == nil || c.HandleID == nil || strings.TrimSpace(c.Agente) == "" {
		return false
	}
	handle, err := GetRuntimeHandle(*c.HandleID)
	if err != nil || handle == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return false
	}
	agente, err := GetAgente(strings.TrimSpace(c.Agente))
	if err != nil || agente == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "enfriamiento")
}

func watchdogActividadRuntimeReciente(c *HandoffCandidato, cutoff time.Time) (bool, error) {
	if c == nil || c.HandleID == nil {
		return false, nil
	}
	handle, err := GetRuntimeHandle(*c.HandleID)
	if err != nil || handle == nil {
		return false, err
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil {
		return false, err
	}
	if strings.EqualFold(strings.TrimSpace(runtime.LogicalState), "cerrado") {
		return false, nil
	}
	ultima := runtimeUltimaActividad(runtime)
	if ultima == nil {
		return false, nil
	}
	return !ultima.Before(cutoff), nil
}

func runtimeUltimaActividad(runtime *RuntimeInstance) *time.Time {
	if runtime == nil {
		return nil
	}
	candidatas := []time.Time{}
	for _, ts := range []*time.Time{runtime.LastEventAt, runtime.LastHeartbeatAt} {
		if ts != nil && !ts.IsZero() {
			candidatas = append(candidatas, ts.UTC())
		}
	}
	if len(candidatas) == 0 {
		return nil
	}
	ultima := candidatas[0]
	for _, ts := range candidatas[1:] {
		if ts.After(ultima) {
			ultima = ts
		}
	}
	return &ultima
}

func presupuestoSesionFresco(p *PresupuestoSesion) bool {
	return PresupuestoSesionFresco(p)
}

// SeleccionarAgenteReemplazo elige el mejor agente disponible para reemplazar
// a "origen". Devuelve el nombre del agente o error si no hay ninguno libre.
// Prioriza agentes del mismo rol, habilitados y sin sesión activa.
func SeleccionarAgenteReemplazo(origen string, excluir []string) (string, error) {
	return SeleccionarAgenteReemplazoParaProyecto(origen, nil, excluir)
}

// SeleccionarAgenteReemplazoParaProyecto aplica la misma política básica de
// reemplazo, pero si conoce el proyecto exige compatibilidad de gobernanza
// efectiva entre origen y destino para evitar handoffs entre agentes no equivalentes.
func SeleccionarAgenteReemplazoParaProyecto(origen string, proyectoID *int64, excluir []string) (string, error) {
	excluidos := append([]string{origen}, excluir...)
	placeholders := strings.Repeat(",?", len(excluidos))[1:] // ",?,?" → "?,?"
	rol, hashEsperado, err := resolverRolYHashGobernanzaHandoff(strings.TrimSpace(origen), proyectoID)
	if err != nil {
		return "", err
	}

	args := make([]any, 0, len(excluidos)+1)
	args = append(args, rol)
	for i, e := range excluidos {
		_ = i
		args = append(args, e)
	}

	rows, err := DB.Query(`
		SELECT a.nombre
		FROM agentes a
		WHERE a.rol = ?
		  AND a.habilitado = 1
		  AND a.activo = 0
		  AND a.nombre NOT IN (`+placeholders+`)
		ORDER BY a.nombre
	`, args...)
	if err != nil {
		return "", err
	}
	candidatos := make([]string, 0, 8)
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			rows.Close()
			return "", err
		}
		candidatos = append(candidatos, nombre)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	for _, nombre := range candidatos {
		if hashEsperado != "" {
			_, hashDestino, err := resolverRolYHashGobernanzaHandoff(strings.TrimSpace(nombre), proyectoID)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(hashDestino) != hashEsperado {
				continue
			}
		}
		return nombre, nil
	}
	if hashEsperado != "" {
		return "", fmt.Errorf("no hay agente disponible compatible con la gobernanza efectiva de %s", origen)
	}
	return "", fmt.Errorf("no hay agente disponible para reemplazar a %s", origen)
}

func ValidarCompatibilidadGobernanzaHandoff(origen, destino string, proyectoID *int64) error {
	rolOrigen, hashOrigen, err := resolverRolYHashGobernanzaHandoff(strings.TrimSpace(origen), proyectoID)
	if err != nil {
		return err
	}
	rolDestino, hashDestino, err := resolverRolYHashGobernanzaHandoff(strings.TrimSpace(destino), proyectoID)
	if err != nil {
		return err
	}
	if rolOrigen != rolDestino {
		return fmt.Errorf("el destino %s no comparte rol con %s", destino, origen)
	}
	if hashOrigen != "" && hashDestino != "" && hashOrigen != hashDestino {
		return fmt.Errorf("el destino %s no es compatible con la gobernanza efectiva de %s", destino, origen)
	}
	return nil
}

func resolverRolYHashGobernanzaHandoff(agente string, proyectoID *int64) (string, string, error) {
	item, err := GetAgente(strings.TrimSpace(agente))
	if err != nil {
		return "", "", fmt.Errorf("no se pudo resolver el agente %s: %w", agente, err)
	}
	rol := "programador"
	if item != nil && strings.TrimSpace(item.Rol) != "" {
		rol = strings.TrimSpace(item.Rol)
	}
	catalogo, err := ResolveGovernanceCatalogForContext(rol, proyectoID, strings.TrimSpace(agente))
	if err != nil {
		return "", "", fmt.Errorf("no se pudo resolver la gobernanza del agente %s: %w", agente, err)
	}
	hash := ""
	if catalogo != nil {
		hash = strings.TrimSpace(catalogo.Hash)
	}
	return rol, hash, nil
}

// GuardarCheckpointHandoff crea un checkpoint automático para el agente saliente
// antes de ejecutar el handoff. El resumen se inyectará en el contexto del nuevo agente.
func GuardarCheckpointHandoff(c *HandoffCandidato, resumen string) (int64, error) {
	if c.SesionID == nil {
		return 0, fmt.Errorf("candidato sin sesión para checkpoint")
	}
	if strings.TrimSpace(resumen) == "" {
		resumen = fmt.Sprintf("Checkpoint automático por handoff: agente %s (%s)",
			c.Agente, descripcionMotivoHandoff(c))
	}

	payload := map[string]any{
		"motivo":     "handoff_automatico",
		"disparador": strings.TrimSpace(c.Disparador),
		"detalle":    strings.TrimSpace(c.Motivo),
	}
	if c.Inactividad > 0 {
		payload["inactividad"] = c.Inactividad.String()
	}
	if c.TareaID != nil {
		payload["tarea_id"] = *c.TareaID
	}
	payloadJSON, _ := json.Marshal(payload)

	id, err := insertReturningID(`
		INSERT INTO runtime_checkpoints
			(agente, proyecto_id, sesion_id, checkpoint_kind, resumen, payload_json, resume_strategy, source)
		VALUES (?, ?, ?, 'automatic', ?, ?, 'context_injection', 'handoff_manager')`,
		c.Agente, c.ProyectoID, *c.SesionID, resumen, string(payloadJSON),
	)
	if err != nil {
		return 0, err
	}
	Audit("server", "checkpoint_handoff", "runtime_checkpoint", id,
		fmt.Sprintf("agente=%s motivo=%s", c.Agente, c.Motivo))
	return id, nil
}

// construirResumenContinuidad genera el texto que se inyectará al agente entrante
// describiendo el estado de la tarea que recoge.
func construirResumenContinuidad(c *HandoffCandidato, checkpointID int64) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Estás relevando al agente %s por %s.\n", c.Agente, descripcionMotivoHandoff(c)))
	if c.TareaID != nil {
		tarea, _ := GetTarea(*c.TareaID)
		if tarea != nil {
			sb.WriteString(fmt.Sprintf("Tarea #%d: %s (estado: %s).\n", tarea.ID, tarea.Titulo, tarea.Estado))
			if tarea.Notas != "" {
				sb.WriteString("Notas previas:\n" + tarea.Notas + "\n")
			}
		}
	}
	sb.WriteString(fmt.Sprintf("Checkpoint guardado: #%d. Revisa el worktree y continúa donde lo dejó el agente anterior.", checkpointID))
	return sb.String()
}

func descripcionMotivoHandoff(c *HandoffCandidato) string {
	if c == nil {
		return "handoff automático"
	}
	switch strings.TrimSpace(c.Disparador) {
	case "presupuesto":
		if strings.TrimSpace(c.Motivo) != "" {
			return "presupuesto crítico (" + strings.TrimSpace(c.Motivo) + ")"
		}
		return "presupuesto crítico"
	case "watchdog":
		if strings.TrimSpace(c.Motivo) != "" {
			return "watchdog (" + strings.TrimSpace(c.Motivo) + ")"
		}
		return "watchdog"
	default:
		if strings.TrimSpace(c.Motivo) != "" {
			return strings.TrimSpace(c.Motivo)
		}
		return "handoff automático"
	}
}

// ProcesarHandoffsBatch detecta agentes agotados, selecciona reemplazos y
// ejecuta el handoff automático para cada candidato.
// Devuelve el número de handoffs iniciados.
func ProcesarHandoffsBatch() (int, error) {
	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		return 0, err
	}

	procesados := 0
	var excluidos []string // agentes ya usados como destino en este batch

	for _, c := range candidatos {
		procede, err := asegurarSondeoPrevioHandoff(c)
		if err != nil {
			Audit("server", "handoff_preflight_error", "agente", 0,
				fmt.Sprintf("agente=%s disparador=%s error=%s", c.Agente, c.Disparador, err.Error()))
			continue
		}
		if !procede {
			continue
		}
		destino, err := SeleccionarAgenteReemplazoParaProyecto(c.Agente, c.ProyectoID, excluidos)
		if err != nil {
			// Sin reemplazo disponible: no es error crítico, sólo lo auditamos
			Audit("server", "handoff_sin_reemplazo", "agente", 0,
				fmt.Sprintf("agente=%s disparador=%s motivo=%s", c.Agente, c.Disparador, err.Error()))
			continue
		}

		checkpointID, err := GuardarCheckpointHandoff(c, "")
		if err != nil {
			Audit("server", "handoff_checkpoint_error", "agente", 0,
				fmt.Sprintf("agente=%s disparador=%s error=%s", c.Agente, c.Disparador, err.Error()))
			continue
		}

		resumen := construirResumenContinuidad(c, checkpointID)
		if c.HandleID == nil {
			_, err = CrearHandoffAgenteStale(c.Agente, destino, c.TareaID, c.Motivo, resumen, "")
		} else {
			_, err = CrearHandoffAgenteVivo(c.Agente, destino, c.TareaID, c.Motivo, resumen, "")
		}
		if err != nil {
			Audit("server", "handoff_error", "agente", 0,
				fmt.Sprintf("%s→%s disparador=%s error=%s", c.Agente, destino, c.Disparador, err.Error()))
			continue
		}

		excluidos = append(excluidos, destino)
		procesados++
	}

	return procesados, nil
}

func asegurarSondeoPrevioHandoff(c *HandoffCandidato) (bool, error) {
	if c == nil {
		return false, nil
	}
	if !c.RequiereSondeo {
		return true, nil
	}
	reciente, err := existeSondeoWatchdogReciente(c.Agente, c.ProyectoID)
	if err != nil {
		return false, err
	}
	if reciente {
		return true, nil
	}
	if err := encolarSondeoWatchdog(c); err != nil {
		return false, err
	}
	return false, nil
}

func existeSondeoWatchdogReciente(agente string, proyectoID *int64) (bool, error) {
	desde := time.Now().UTC().Add(-time.Minute)
	q := runtimeOrderSelectBase() + `
		WHERE agente = ?
		  AND tipo IN ('sync_status','nudge')
		  AND created_at >= ?`
	args := []any{strings.TrimSpace(agente), desde}
	if proyectoID != nil {
		q += ` AND (proyecto_id = ? OR proyecto_id IS NULL)`
		args = append(args, *proyectoID)
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		order, err := scanRuntimeOrder(rows)
		if err != nil {
			return false, err
		}
		if order == nil {
			continue
		}
		if strings.TrimSpace(order.Tipo) == "sync_status" {
			return true, nil
		}
		if strings.TrimSpace(order.Tipo) == "nudge" && strings.Contains(strings.ToLower(order.PayloadJSON), `"kind":"watchdog"`) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func encolarSondeoWatchdog(c *HandoffCandidato) error {
	if c == nil || c.HandleID == nil {
		return nil
	}
	var runtimeID *int64
	var handleID *int64
	handleID = c.HandleID
	handle, err := GetRuntimeHandle(*c.HandleID)
	if err != nil {
		return err
	}
	if runtime, err := runtimeHandleRuntime(handle); err != nil {
		return err
	} else if runtime != nil && runtime.ID > 0 {
		runtimeID = &runtime.ID
	} else if handle != nil && handle.RuntimeID != nil {
		runtimeID = handle.RuntimeID
	}
	payloadSync, _ := json.Marshal(map[string]any{
		"motivo":      "watchdog_heartbeat_stale",
		"inactividad": c.Inactividad.String(),
	})
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      c.Agente,
		ProyectoID:  c.ProyectoID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "sync_status",
		PayloadJSON: string(payloadSync),
	}); err != nil {
		return err
	}
	payloadNudge, _ := json.Marshal(map[string]any{
		"from_agente": "server",
		"to_agente":   c.Agente,
		"kind":        "watchdog",
		"texto":       fmt.Sprintf("Orquesta detecta heartbeat obsoleto (%s). Confirma estado o reanuda tick.", c.Motivo),
	})
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      c.Agente,
		ProyectoID:  c.ProyectoID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "nudge",
		PayloadJSON: string(payloadNudge),
	}); err != nil {
		return err
	}
	Audit("server", "watchdog_runtime_sondeo", "agente", 0,
		fmt.Sprintf("agente=%s motivo=%s", c.Agente, c.Motivo))
	return nil
}

// ─── helper local ────────────────────────────────────────────────────────────

// nullInt64 es un alias local para evitar importar database/sql en este helper.
type nullInt64 struct {
	Int64 int64
	Valid bool
}

func (n *nullInt64) Scan(value any) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	switch v := value.(type) {
	case int64:
		n.Int64 = v
		n.Valid = true
	case []byte:
		var i int64
		if _, err := fmt.Sscan(string(v), &i); err != nil {
			return err
		}
		n.Int64 = i
		n.Valid = true
	default:
		return fmt.Errorf("nullInt64: tipo inesperado %T", value)
	}
	return nil
}
