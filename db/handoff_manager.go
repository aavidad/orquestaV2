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
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// HandoffCandidato describe un agente que cumple las condiciones para ser
// relevado automáticamente.
type HandoffCandidato struct {
	Agente      string        // agente con la sesión inactiva
	TareaID     *int64        // tarea en progreso que tiene asignada
	SesionID    *int64        // sesión activa del agente
	HandleID    *int64        // runtime handle activo (puede ser nil)
	ProyectoID  *int64        // proyecto de la tarea/sesión
	Inactividad time.Duration // tiempo desde el último heartbeat
	Motivo      string        // descripción del motivo de detección
}

// DetectarAgentesAgotados devuelve la lista de agentes que tienen tareas
// en_progreso y cuya sesión lleva inactiva más de pool_handoff_threshold_seconds.
// Excluye agentes que ya tienen una runtime_order de handoff pendiente o ejecutando
// para evitar handoffs duplicados.
func DetectarAgentesAgotados() ([]*HandoffCandidato, error) {
	threshold := time.Duration(configIntOrDefault("pool_handoff_threshold_seconds", 1800)) * time.Second
	if threshold <= 0 {
		threshold = 30 * time.Minute
	}
	cutoff := time.Now().UTC().Add(-threshold)

	rows, err := DB.Query(`
		SELECT
			t.agente,
			t.id            AS tarea_id,
			s.id            AS sesion_id,
			s.proyecto_id,
			s.heartbeat_at,
			h.id            AS handle_id
		FROM tareas t
		JOIN sesiones s ON s.agente = t.agente AND s.activa = 1
		LEFT JOIN runtime_handles h
			ON h.agente = t.agente
			AND h.estado IN ('activo','pausado')
		WHERE t.estado = 'en_progreso'
		  AND t.agente IS NOT NULL
		  AND s.heartbeat_at IS NOT NULL
		  AND s.heartbeat_at <= ?
		  AND t.agente NOT IN (
			  SELECT DISTINCT agente
			  FROM runtime_orders
			  WHERE tipo = 'handoff'
			    AND estado IN ('pendiente','tomada','ejecutando')
		  )
		GROUP BY t.agente
		ORDER BY s.heartbeat_at`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidatos []*HandoffCandidato
	for rows.Next() {
		var (
			c           HandoffCandidato
			tareaID     int64
			sesionID    int64
			proyectoID  nullInt64
			handleID    nullInt64
			heartbeatAt time.Time
		)
		if err := rows.Scan(&c.Agente, &tareaID, &sesionID, &proyectoID, &heartbeatAt, &handleID); err != nil {
			return nil, err
		}
		c.TareaID = &tareaID
		c.SesionID = &sesionID
		if proyectoID.Valid {
			c.ProyectoID = &proyectoID.Int64
		}
		if handleID.Valid {
			c.HandleID = &handleID.Int64
		}
		c.Inactividad = time.Since(heartbeatAt)
		c.Motivo = fmt.Sprintf("heartbeat hace %.0f min (umbral: %.0f min)",
			c.Inactividad.Minutes(), threshold.Minutes())
		candidatos = append(candidatos, &c)
	}
	return candidatos, rows.Err()
}

// SeleccionarAgenteReemplazo elige el mejor agente disponible para reemplazar
// a "origen". Devuelve el nombre del agente o error si no hay ninguno libre.
// Prioriza: programadores habilitados sin sesión activa, por nombre alfabético.
func SeleccionarAgenteReemplazo(origen string, excluir []string) (string, error) {
	excluidos := append([]string{origen}, excluir...)
	placeholders := strings.Repeat(",?", len(excluidos))[1:] // ",?,?" → "?,?"
	args := make([]any, len(excluidos))
	for i, e := range excluidos {
		args[i] = e
	}

	var nombre string
	err := DB.QueryRow(`
		SELECT a.nombre
		FROM agentes a
		WHERE a.rol = 'programador'
		  AND a.habilitado = 1
		  AND a.activo = 0
		  AND a.nombre NOT IN (`+placeholders+`)
		ORDER BY a.nombre
		LIMIT 1`, args...).Scan(&nombre)
	if err != nil {
		return "", fmt.Errorf("no hay agente disponible para reemplazar a %s: %w", origen, err)
	}
	return nombre, nil
}

// GuardarCheckpointHandoff crea un checkpoint automático para el agente saliente
// antes de ejecutar el handoff. El resumen se inyectará en el contexto del nuevo agente.
func GuardarCheckpointHandoff(c *HandoffCandidato, resumen string) (int64, error) {
	if c.SesionID == nil {
		return 0, fmt.Errorf("candidato sin sesión para checkpoint")
	}
	if strings.TrimSpace(resumen) == "" {
		resumen = fmt.Sprintf("Checkpoint automático por handoff: agente %s inactivo %s",
			c.Agente, c.Inactividad.Round(time.Second))
	}

	payload := map[string]any{
		"motivo":      "handoff_automatico",
		"inactividad": c.Inactividad.String(),
	}
	if c.TareaID != nil {
		payload["tarea_id"] = *c.TareaID
	}
	payloadJSON, _ := json.Marshal(payload)

	res, err := DB.Exec(`
		INSERT INTO runtime_checkpoints
			(agente, proyecto_id, sesion_id, checkpoint_kind, resumen, payload_json, resume_strategy, source)
		VALUES (?, ?, ?, 'automatic', ?, ?, 'context_injection', 'handoff_manager')`,
		c.Agente, c.ProyectoID, *c.SesionID, resumen, string(payloadJSON),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("server", "checkpoint_handoff", "runtime_checkpoint", id,
		fmt.Sprintf("agente=%s motivo=%s", c.Agente, c.Motivo))
	return id, nil
}

// construirResumenContinuidad genera el texto que se inyectará al agente entrante
// describiendo el estado de la tarea que recoge.
func construirResumenContinuidad(c *HandoffCandidato, checkpointID int64) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Estás relevando al agente %s por agotamiento/inactividad (%s).\n", c.Agente, c.Motivo))
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
			Audit("server", "handoff_watchdog_error", "agente", 0,
				fmt.Sprintf("agente=%s error=%s", c.Agente, err.Error()))
			continue
		}
		if !procede {
			continue
		}
		destino, err := SeleccionarAgenteReemplazo(c.Agente, excluidos)
		if err != nil {
			// Sin reemplazo disponible: no es error crítico, sólo lo auditamos
			Audit("server", "handoff_sin_reemplazo", "agente", 0,
				fmt.Sprintf("agente=%s motivo=%s", c.Agente, err.Error()))
			continue
		}

		checkpointID, err := GuardarCheckpointHandoff(c, "")
		if err != nil {
			Audit("server", "handoff_checkpoint_error", "agente", 0,
				fmt.Sprintf("agente=%s error=%s", c.Agente, err.Error()))
			continue
		}

		resumen := construirResumenContinuidad(c, checkpointID)
		_, err = CrearHandoffAgenteVivo(c.Agente, destino, c.TareaID, c.Motivo, resumen, "")
		if err != nil {
			Audit("server", "handoff_error", "agente", 0,
				fmt.Sprintf("%s→%s error=%s", c.Agente, destino, err.Error()))
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
	var runtimeID *int64
	var handleID *int64
	if c.HandleID != nil {
		handleID = c.HandleID
		handle, err := GetRuntimeHandle(*c.HandleID)
		if err != nil {
			return err
		}
		if handle != nil && handle.RuntimeID != nil {
			runtimeID = handle.RuntimeID
		}
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
