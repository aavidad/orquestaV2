package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

import (
	"orquesta/sesionesapp"
)

type Agente = sesionesapp.Agente
type Sesion = sesionesapp.Sesion
type SesionInicio = sesionesapp.SesionInicio
type SesionUpdate = sesionesapp.SesionUpdate
type FiltroSesionesInspeccion = sesionesapp.FiltroInspeccion

// IniciarSesion marca al agente como activo y crea una sesión.
func IniciarSesion(agente string) (int64, error) {
	s, err := IniciarSesionContexto(SesionInicio{Agente: agente})
	if err != nil {
		return 0, err
	}
	return s.ID, nil
}

func IniciarSesionContexto(in SesionInicio) (*Sesion, error) {
	agente := strings.TrimSpace(in.Agente)
	// Verificar que el agente existe y está habilitado
	agenteCanonico, _, habilitado, err := resolverAgentePorNombreCI(agente)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agente '%s' no registrado; usa 'orquesta config agente-nuevo' para registrarlo", agente)
	}
	if err != nil {
		return nil, err
	}
	agente = agenteCanonico
	if !habilitado {
		return nil, fmt.Errorf("agente '%s' está retirado y no puede iniciar sesión", agente)
	}

	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Cerrar sesiones anteriores abiertas
	if _, err = tx.Exec(`UPDATE sesiones SET activa=0, estado='cerrada', fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, agente); err != nil {
		return nil, err
	}

	// Marcar activo y estado inicial
	_, err = tx.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible', ultima_sesion=CURRENT_TIMESTAMP WHERE nombre=?`, agente)
	if err != nil {
		return nil, err
	}

	// Crear nueva sesión
	res, err := tx.Exec(`
		INSERT INTO sesiones (agente, conector_id, proyecto_id, cwd, herramienta, external_session_id, resume_payload_json, resumen_continuidad, branch, heartbeat_at, host, pid)
		VALUES (?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,?,?)`,
		agente, in.ConectorID, in.ProyectoID, in.CWD, in.Herramienta, in.ExternalSessionID, in.ResumePayloadJSON, in.ResumenContinuidad, in.Branch, in.Host, in.PID,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	Audit(agente, "inicio_sesion", "sesion", id, "")
	sesion, err := GetSesionByID(id)
	if err != nil {
		return nil, err
	}
	if err := asignarPoolCanonicoSesion(sesion); err != nil {
		return nil, err
	}
	if err := UpsertRuntimeDesdeSesion(sesion); err != nil {
		return nil, err
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		return nil, err
	}
	return sesion, nil
}

// FinSesion marca al agente como inactivo y cierra su sesión.
func FinSesion(agente string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE agentes SET activo=0, estado_sesion=NULL WHERE nombre=?`, agente)
	if err != nil {
		return err
	}
	res, err := tx.Exec(`
		UPDATE sesiones SET activa=0, estado='cerrada', fin=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`, agente)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("el agente '%s' no tenía sesión activa", agente)
	}
	if err := MarcarRuntimesCerradosPorAgente(agente); err != nil {
		return err
	}
	if err := MarcarRuntimeHandlesCerradosPorAgente(agente); err != nil {
		return err
	}
	Audit(agente, "fin_sesion", "sesion", 0, "")
	return nil
}

func asignarPoolCanonicoSesion(sesion *Sesion) error {
	if sesion == nil {
		return nil
	}
	herramienta := strings.TrimSpace(sesion.Herramienta)
	conector := strings.TrimSpace(sesion.ConectorSlug)
	if !strings.EqualFold(herramienta, "ollama_pool_local") && !strings.EqualFold(conector, "ollama_pool_local") {
		return nil
	}
	proyectoSlug := strings.TrimSpace(sesion.ProyectoSlug)
	if proyectoSlug == "" {
		return nil
	}
	agente := strings.TrimSpace(sesion.Agente)
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		AgenteNombre: &agente,
		ProyectoSlug: proyectoSlug,
	})
	if err != nil {
		return err
	}
	if resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return nil
	}
	pool, err := GetPool(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil {
		return err
	}
	if !poolUsaConectorPoolLocalCompartido(pool) {
		return nil
	}
	_, err = DB.Exec(`UPDATE sesiones SET pool_id = ? WHERE id = ?`, pool.ID, sesion.ID)
	return err
}

func GetSesionByID(id int64) (*Sesion, error) {
	return consultarConReintentos(func() (*Sesion, error) {
		row := DB.QueryRow(`
			SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
			       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
			       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
			       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
			       s.branch, s.heartbeat_at, s.host, s.pid
			FROM sesiones s
			LEFT JOIN conectores c ON c.id = s.conector_id
			LEFT JOIN proyectos p ON p.id = s.proyecto_id
			WHERE s.id = ?`, id)
		return escanearSesion(row)
	})
}

func ObtenerUltimaSesion(agente string, proyectoID *int64) (*Sesion, error) {
	return ObtenerUltimaSesionConFiltro(agente, proyectoID, "")
}

func ObtenerUltimaSesionConFiltro(agente string, proyectoID *int64, cwd string) (*Sesion, error) {
	q := `
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.agente = ?`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND s.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	if strings.TrimSpace(cwd) != "" {
		q += ` AND s.cwd = ?`
		args = append(args, strings.TrimSpace(cwd))
	}
	q += ` ORDER BY s.id DESC LIMIT 1`
	return escanearSesion(DB.QueryRow(q, args...))
}

func GuardarSesionActiva(agente string, proyectoID *int64, upd SesionUpdate) error {
	q := `SELECT id FROM sesiones WHERE agente = ? AND activa = 1`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY id DESC LIMIT 1`

	var id int64
	if _, err := consultarConReintentos(func() (int64, error) {
		err := DB.QueryRow(q, args...).Scan(&id)
		return id, err
	}); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("el agente '%s' no tiene sesión activa", agente)
		}
		return err
	}

	partes := make([]string, 0, 8)
	updateArgs := make([]any, 0, 10)
	if upd.CWD != nil {
		partes = append(partes, "cwd = ?")
		updateArgs = append(updateArgs, *upd.CWD)
	}
	if upd.Herramienta != nil {
		partes = append(partes, "herramienta = ?")
		updateArgs = append(updateArgs, *upd.Herramienta)
	}
	if upd.ExternalSessionID != nil {
		partes = append(partes, "external_session_id = ?")
		updateArgs = append(updateArgs, *upd.ExternalSessionID)
	}
	if upd.ResumePayloadJSON != nil {
		partes = append(partes, "resume_payload_json = ?")
		updateArgs = append(updateArgs, *upd.ResumePayloadJSON)
	}
	if upd.ResumenContinuidad != nil {
		partes = append(partes, "resumen_continuidad = ?")
		updateArgs = append(updateArgs, *upd.ResumenContinuidad)
	}
	if upd.Branch != nil {
		partes = append(partes, "branch = ?")
		updateArgs = append(updateArgs, *upd.Branch)
	}
	if upd.Host != nil {
		partes = append(partes, "host = ?")
		updateArgs = append(updateArgs, *upd.Host)
	}
	if upd.PID != nil {
		partes = append(partes, "pid = ?")
		updateArgs = append(updateArgs, *upd.PID)
	}
	if upd.Estado != nil {
		partes = append(partes, "estado = ?")
		updateArgs = append(updateArgs, *upd.Estado)
	}
	if upd.Heartbeat {
		partes = append(partes, "heartbeat_at = CURRENT_TIMESTAMP")
		// Solo incrementamos estadísticas de uso estimado, pero ya no bloqueamos localmente (OP-084)
		tick := configIntOrDefault("agent_tick_seconds", 30)
		_, _ = DB.Exec(`
			UPDATE agentes 
			SET consumo_dia_segundos = consumo_dia_segundos + ?, 
			    consumo_semanal_segundos = consumo_semanal_segundos + ?
			WHERE nombre = ?`, tick, tick, agente)
	}
	if len(partes) == 0 {
		return nil
	}

	updateArgs = append(updateArgs, id)
	_, err := DB.Exec(`UPDATE sesiones SET `+strings.Join(partes, ", ")+` WHERE id = ?`, updateArgs...)
	if err == nil {
		sesion, getErr := GetSesionByID(id)
		if getErr != nil {
			err = getErr
		} else {
			err = UpsertRuntimeDesdeSesion(sesion)
			if err == nil {
				err = UpsertRuntimeHandleDesdeSesion(sesion)
			}
		}
	}
	if err == nil {
		Audit(agente, "guardar_sesion", "sesion", id, "")
	}
	return err
}

func ListarSesionesActivas() ([]*Sesion, error) {
	return ListarSesionesActivasOperativas()
}

func listarSesionesAbiertasRaw() ([]*Sesion, error) {
	return consultarConReintentos(func() ([]*Sesion, error) {
		rows, err := DB.Query(`
			SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
			       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
			       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
			       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
			       s.branch, s.heartbeat_at, s.host, s.pid
			FROM sesiones s
			LEFT JOIN conectores c ON c.id = s.conector_id
			LEFT JOIN proyectos p ON p.id = s.proyecto_id
			WHERE s.activa = 1
			ORDER BY s.id DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []*Sesion
		for rows.Next() {
			s, err := escanearSesion(rows)
			if err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, rows.Err()
	})
}

func sessionOperationalStaleDuration() time.Duration {
	seconds := configIntOrDefault("session_operational_stale_seconds", 180)
	if seconds <= 0 {
		tick := configIntOrDefault("agent_tick_seconds", 30)
		if tick <= 0 {
			tick = 30
		}
		seconds = tick * 6
	}
	return time.Duration(seconds) * time.Second
}

func sesionOperativaPorHeartbeat(activa bool, heartbeat *time.Time, inicio time.Time, now time.Time) bool {
	if !activa {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ultima := preferTime(heartbeat, timePtr(inicio), &now)
	if ultima == nil || ultima.IsZero() {
		return false
	}
	return !ultima.Before(now.Add(-sessionOperationalStaleDuration()))
}

func SesionEsOperativa(sesion *Sesion) bool {
	if sesion == nil {
		return false
	}
	return sesionOperativaPorHeartbeat(sesion.Activa, sesion.HeartbeatAt, sesion.Inicio, time.Now().UTC())
}

func runtimeHandleCacheKey(agente string, proyectoID *int64) string {
	key := strings.ToLower(strings.TrimSpace(agente))
	if proyectoID != nil && *proyectoID > 0 {
		key = fmt.Sprintf("%s#%d", key, *proyectoID)
	}
	return key
}

func listarHandlesActivosOperativosMap() (map[string]bool, error) {
	cutoff := sessionOperationalCutoff()
	return consultarConReintentos(func() (map[string]bool, error) {
		rows, err := DB.Query(runtimeHandleSelectBase() + `
		 WHERE estado IN ('activo','pausado')
		 ORDER BY id DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[string]bool{}
		for rows.Next() {
			handle, err := scanRuntimeHandle(rows)
			if err != nil {
				return nil, err
			}
			if !runtimeHandleSostieneSesionOperativaConCutoff(handle, cutoff) {
				continue
			}
			out[runtimeHandleCacheKey(handle.Agente, handle.ProyectoID)] = true
		}
		return out, rows.Err()
	})
}

func listarAgentesConHandleActivoOperativo() (map[string]bool, error) {
	handlesActivos, err := listarHandlesActivosOperativosMap()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(handlesActivos))
	for key, ok := range handlesActivos {
		if !ok {
			continue
		}
		nombre := key
		if idx := strings.Index(nombre, "#"); idx >= 0 {
			nombre = nombre[:idx]
		}
		nombre = strings.ToLower(strings.TrimSpace(nombre))
		if nombre != "" {
			out[nombre] = true
		}
	}
	return out, nil
}

func sesionTieneHandleActivoOperativo(agente string, proyectoID *int64) (bool, error) {
	handlesActivos, err := listarHandlesActivosOperativosMap()
	if err != nil {
		return false, err
	}
	return handlesActivos[runtimeHandleCacheKey(agente, proyectoID)], nil
}

func listarUltimoHandlePorAgenteProyectoMap() (map[string]*RuntimeHandle, error) {
	return consultarConReintentos(func() (map[string]*RuntimeHandle, error) {
		rows, err := DB.Query(runtimeHandleSelectBase() + ` ORDER BY id DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[string]*RuntimeHandle{}
		for rows.Next() {
			handle, err := scanRuntimeHandle(rows)
			if err != nil {
				return nil, err
			}
			key := runtimeHandleCacheKey(handle.Agente, handle.ProyectoID)
			if _, exists := out[key]; exists {
				continue
			}
			out[key] = handle
		}
		return out, rows.Err()
	})
}

func listarUltimoRuntimePorAgenteProyectoMap() (map[string]*RuntimeInstance, error) {
	return consultarConReintentos(func() (map[string]*RuntimeInstance, error) {
		rows, err := DB.Query(runtimeSelectBase() + `
		 ORDER BY ` + runtimeActividadExpr("r") + ` DESC, r.id DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[string]*RuntimeInstance{}
		for rows.Next() {
			runtime, err := escanearRuntime(rows)
			if err != nil {
				return nil, err
			}
			key := runtimeHandleCacheKey(runtime.Agente, runtime.ProyectoID)
			if _, exists := out[key]; exists {
				continue
			}
			out[key] = runtime
		}
		return out, rows.Err()
	})
}

func runtimeHandleSostieneSesionOperativa(handle *RuntimeHandle) bool {
	return runtimeHandleSostieneSesionOperativaConCutoff(handle, sessionOperationalCutoff())
}

func runtimeHandleSostieneSesionOperativaConCutoff(handle *RuntimeHandle, cutoff time.Time) bool {
	if handle == nil {
		return false
	}
	estado := strings.TrimSpace(handle.Estado)
	if estado != "activo" && estado != "pausado" {
		return false
	}
	ultima := preferTime(handle.LastSeenAt, &handle.UpdatedAt, &handle.CreatedAt)
	if ultima == nil || ultima.Before(cutoff) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return true
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if externalSessionID := strings.TrimSpace(stringFromMap(meta, "external_session_id", "")); externalSessionID != "" {
		return true
	}
	handleRef := strings.TrimSpace(handle.HandleRef)
	if handleRef == "" {
		return false
	}
	if handle.SesionID != nil && handleRef == jsonNumber(*handle.SesionID) {
		return false
	}
	return true
}

func runtimeSostieneSesionOperativaConCutoff(runtime *RuntimeInstance, cutoff time.Time) bool {
	if runtime == nil {
		return false
	}
	logicalState := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
	switch logicalState {
	case "cerrado", "fallido", "bloqueado", "finalizado":
		return false
	}
	processState := strings.ToLower(strings.TrimSpace(runtime.ProcessState))
	switch processState {
	case "finalizado", "fallido", "exited", "dead":
		return false
	}
	ultima := preferTime(runtime.UltimaActividadAt, runtime.LastHeartbeatAt, runtime.LastEventAt, &runtime.UpdatedAt, &runtime.CreatedAt)
	return ultima != nil && !ultima.Before(cutoff)
}

func sesionInvalidadaPorUltimoHandle(sesion *Sesion, handle *RuntimeHandle, cutoff time.Time, now time.Time) bool {
	if sesion == nil || handle == nil {
		return false
	}
	if handle.SesionID != nil && *handle.SesionID != sesion.ID {
		return false
	}
	if runtimeHandleSostieneSesionOperativaConCutoff(handle, cutoff) {
		return false
	}
	estado := strings.TrimSpace(handle.Estado)
	if estado != "cerrado" && estado != "fallido" {
		return false
	}
	ultimaHandle := preferTime(handle.LastSeenAt, &handle.UpdatedAt, &handle.CreatedAt)
	if ultimaHandle == nil || ultimaHandle.Before(cutoff) {
		return false
	}
	ultimaSesion := preferTime(sesion.HeartbeatAt, timePtr(sesion.Inicio), &now)
	if ultimaSesion == nil {
		return false
	}
	return !ultimaHandle.Before(*ultimaSesion)
}

func sesionEsActivaOperativaConMapas(sesion *Sesion, handlesActivos map[string]bool, ultimosHandles map[string]*RuntimeHandle, ultimosRuntimes map[string]*RuntimeInstance, now time.Time) bool {
	if sesion == nil {
		return false
	}
	key := runtimeHandleCacheKey(sesion.Agente, sesion.ProyectoID)
	cutoff := sessionOperationalCutoff()
	runtime := ultimosRuntimes[key]
	if runtime != nil && !runtimeSostieneSesionOperativaConCutoff(runtime, cutoff) {
		return false
	}
	sostieneRuntime := runtime != nil && runtimeSostieneSesionOperativaConCutoff(runtime, cutoff)
	sostieneHandle := handlesActivos[key]
	if sesionOperativaPorHeartbeat(sesion.Activa, sesion.HeartbeatAt, sesion.Inicio, now) {
		if sesionInvalidadaPorUltimoHandle(sesion, ultimosHandles[key], cutoff, now) {
			return false
		}
		return sostieneRuntime || sostieneHandle
	}
	return sostieneHandle
}

func ListarSesionesActivasOperativas() ([]*Sesion, error) {
	sesiones, err := listarSesionesAbiertasRaw()
	if err != nil {
		return nil, err
	}
	handlesActivos, err := listarHandlesActivosOperativosMap()
	if err != nil {
		return nil, err
	}
	ultimosHandles, err := listarUltimoHandlePorAgenteProyectoMap()
	if err != nil {
		return nil, err
	}
	ultimosRuntimes, err := listarUltimoRuntimePorAgenteProyectoMap()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]*Sesion, 0, len(sesiones))
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		if sesionEsActivaOperativaConMapas(sesion, handlesActivos, ultimosHandles, ultimosRuntimes, now) {
			out = append(out, sesion)
		}
	}
	return out, nil
}

func aplicarEstadoVisibleAgente(agente *Agente, sesion *Sesion) {
	if agente == nil {
		return
	}
	if AgenteSinCuotaProveedorEfectivo(agente) {
		normalizarAgenteSinCuotaProveedor(agente)
	}
	proyectarBloqueoVisibleDesdePresupuestoObservado(agente)
	sincronizarReanimacionVisibleDesdePresupuesto(agente)
	if !agenteBloqueadoPorCuotaOperativo(agente) && !strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "activo") {
		mantenerPausaVisible := agenteMotivoPausaOperativaVisible(agente.MotivoPausa)
		if !mantenerPausaVisible && agente.ReanimarAt != nil && agente.ReanimarAt.After(time.Now().UTC()) {
			mantenerPausaVisible = true
		}
		if !mantenerPausaVisible {
			agente.EstadoCuota = "activo"
			agente.ReanimarAt = nil
			if !agenteMotivoPausaOperativaVisible(agente.MotivoPausa) {
				agente.MotivoPausa = ""
			}
		}
	}
	if sesion == nil {
		agente.Activo = false
		agente.EstadoSesion = ""
		return
	}
	if agenteBloqueadoPorCuotaOperativo(agente) {
		agente.Activo = false
		if estado := strings.TrimSpace(agente.EstadoSesion); estado != "" {
			agente.EstadoSesion = estado
		} else {
			agente.EstadoSesion = "pausada"
		}
		return
	}
	if strings.EqualFold(strings.TrimSpace(sesion.Estado), "pausada") {
		agente.Activo = false
		if estado := strings.TrimSpace(agente.EstadoSesion); estado != "" {
			agente.EstadoSesion = estado
		} else {
			agente.EstadoSesion = "pausada"
		}
		return
	}
	agente.Activo = true
	if estado := strings.TrimSpace(agente.EstadoSesion); estado != "" {
		agente.EstadoSesion = estado
		return
	}
	if estado := strings.TrimSpace(sesion.Estado); estado != "" && estado != "activa" {
		agente.EstadoSesion = estado
		return
	}
	agente.EstadoSesion = "disponible"
}

func agenteMotivoPausaOperativaVisible(motivo string) bool {
	motivo = strings.ToLower(strings.TrimSpace(motivo))
	switch motivo {
	case "", "usage limit", "cuota diaria agotada", "presupuesto agotado observado", "presupuesto semanal agotado observado", "ventana corta agotada observada", "créditos agotados observados", "creditos agotados observados":
		return false
	}
	return strings.Contains(motivo, "runtime") ||
		strings.Contains(motivo, "panic") ||
		strings.Contains(motivo, "manual") ||
		strings.Contains(motivo, "pausa") ||
		strings.Contains(motivo, "detenido") ||
		strings.Contains(motivo, "supervisor")
}

func proyectarBloqueoVisibleDesdePresupuestoObservado(a *Agente) {
	if a == nil {
		return
	}
	if AgenteSinCuotaProveedorEfectivo(a) {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "activo") {
		return
	}
	if a.PresupuestoStale &&
		strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "provider_backoff") &&
		a.PresupuestoSemanalPct != nil &&
		*a.PresupuestoSemanalPct > 0 {
		return
	}
	now := time.Now().UTC()
	if strings.EqualFold(strings.TrimSpace(a.PresupuestoEstado), "agotado") {
		a.EstadoCuota = "agotado"
		if strings.TrimSpace(a.MotivoPausa) == "" {
			a.MotivoPausa = "Presupuesto agotado observado"
		}
		if a.PresupuestoResetAt != nil && a.PresupuestoResetAt.After(now) {
			a.ReanimarAt = a.PresupuestoResetAt
		}
		return
	}
	if a.RemainingCredits != nil && *a.RemainingCredits <= 0 {
		a.EstadoCuota = "agotado"
		if strings.TrimSpace(a.MotivoPausa) == "" {
			a.MotivoPausa = "Créditos agotados observados"
		}
		if a.PresupuestoResetAt != nil && a.PresupuestoResetAt.After(now) {
			a.ReanimarAt = a.PresupuestoResetAt
		}
		return
	}
	if a.PresupuestoSemanalPct != nil && *a.PresupuestoSemanalPct <= 0 {
		a.EstadoCuota = "agotado"
		if strings.TrimSpace(a.MotivoPausa) == "" {
			a.MotivoPausa = "Presupuesto semanal agotado observado"
		}
		if a.PresupuestoSemanalResetAt != nil && a.PresupuestoSemanalResetAt.After(now) {
			a.ReanimarAt = a.PresupuestoSemanalResetAt
		}
		return
	}
}

func agenteTieneVentanaTemporalVisible(a *Agente) bool {
	if a == nil {
		return false
	}
	if a.PresupuestoSesionPct != nil {
		return true
	}
	if presupuestoEsVentanaCorta(strings.TrimSpace(a.PresupuestoVentana)) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "provider_backoff")
}

func agenteBloqueadoPorCuotaOperativo(a *Agente) bool {
	if a == nil {
		return false
	}
	if AgenteSinCuotaProveedorEfectivo(a) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "activo") {
		return false
	}
	if a.ReanimarAt != nil && !a.ReanimarAt.IsZero() && a.ReanimarAt.After(time.Now().UTC()) {
		return true
	}
	if a.RemainingCredits != nil && *a.RemainingCredits <= 0 {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(a.PresupuestoEstado), "agotado") {
		return true
	}
	if a.PresupuestoSemanalPct != nil {
		if *a.PresupuestoSemanalPct <= 0 {
			return true
		}
		if !agenteTieneVentanaTemporalVisible(a) {
			return false
		}
	}
	if a.PresupuestoSesionPct != nil {
		return *a.PresupuestoSesionPct <= 0
	}
	if presupuestoEsVentanaCorta(strings.TrimSpace(a.PresupuestoVentana)) && a.CuotaRestantePct != nil {
		return *a.CuotaRestantePct <= 0
	}
	if a.CuotaRestantePct != nil {
		return *a.CuotaRestantePct <= 0
	}
	return true
}

func sincronizarReanimacionVisibleDesdePresupuesto(a *Agente) {
	if a == nil {
		return
	}
	if AgenteSinCuotaProveedorEfectivo(a) {
		a.ReanimarAt = nil
		return
	}
	if strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "activo") {
		return
	}
	now := time.Now().UTC()
	if a.ReanimarAt != nil && !a.ReanimarAt.IsZero() && a.ReanimarAt.After(now) {
		return
	}
	candidatos := []*time.Time{
		a.PresupuestoResetAt,
		a.PresupuestoSemanalResetAt,
		a.PresupuestoDiarioResetAt,
		a.PresupuestoSesionResetAt,
	}
	for _, candidate := range candidatos {
		if candidate != nil && !candidate.IsZero() && candidate.After(now) {
			a.ReanimarAt = candidate
			return
		}
	}
}

func aplicarEstadoVisibleAgentes(agentes []*Agente, sesiones []*Sesion) {
	if len(agentes) == 0 {
		return
	}
	sesionesPorAgente := make(map[string]*Sesion, len(sesiones))
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		actual, existe := sesionesPorAgente[sesion.Agente]
		if !existe || sesion.ID > actual.ID {
			sesionesPorAgente[sesion.Agente] = sesion
		}
	}
	for _, agente := range agentes {
		aplicarEstadoVisibleAgente(agente, sesionesPorAgente[agente.Nombre])
	}
}

// ListarAgentes devuelve todos los agentes registrados con su estado de cuota.
func listarAgentesRaw() ([]*Agente, error) {
	return consultarConReintentos(func() ([]*Agente, error) {
		rows, err := DB.Query(`
			SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
			       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
			       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
			       reanimar_at, motivo_pausa
			FROM agentes ORDER BY nombre`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []*Agente
		for rows.Next() {
			a := &Agente{}
			var ultima sql.NullTime
			var lastReset sql.NullTime
			var reanimar sql.NullTime
			var motivo sql.NullString
			if err := rows.Scan(
				&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima,
				&a.ConsumoDiaSegundos, &a.ConsumoSemanalSegundos, &a.LimiteDiaSegundos,
				&a.LimiteSemanalSegundos, &lastReset, &a.EstadoCuota,
				&reanimar, &motivo,
			); err != nil {
				return nil, err
			}
			if ultima.Valid {
				a.UltimaSesion = &ultima.Time
			}
			if lastReset.Valid {
				a.LastUsageResetAt = &lastReset.Time
			}
			if reanimar.Valid {
				a.ReanimarAt = &reanimar.Time
			}
			a.MotivoPausa = motivo.String
			list = append(list, a)
		}
		return list, rows.Err()
	})
}

func ListarAgentesConSesionesActivas(sesiones []*Sesion) ([]*Agente, error) {
	list, err := listarAgentesRaw()
	if err != nil {
		return nil, err
	}
	enriquecerAgentesConPresupuesto(list)
	aplicarEstadoVisibleAgentes(list, sesiones)
	agentesConHandleActivo, err := listarAgentesConHandleActivoOperativo()
	if err != nil {
		return nil, err
	}
	for _, agente := range list {
		if agente == nil || agente.Activo || !agente.Habilitado {
			continue
		}
		estadoCuota := strings.TrimSpace(agente.EstadoCuota)
		if strings.EqualFold(estadoCuota, "enfriamiento") || strings.EqualFold(estadoCuota, "agotado") {
			continue
		}
		if !agentesConHandleActivo[strings.ToLower(strings.TrimSpace(agente.Nombre))] {
			continue
		}
		agente.Activo = true
		if strings.TrimSpace(agente.EstadoSesion) == "" {
			agente.EstadoSesion = "disponible"
		}
	}
	return list, nil
}

func ListarAgentes() ([]*Agente, error) {
	sesionesActivas, err := ListarSesionesActivasOperativas()
	if err != nil {
		return nil, err
	}
	return ListarAgentesConSesionesActivas(sesionesActivas)
}

func ListarAgentesCuentasLigero() ([]*Agente, error) {
	list, err := listarAgentesRaw()
	if err != nil {
		return nil, err
	}
	for _, agente := range list {
		if agente == nil {
			continue
		}
		aplicarIdentidadCuentaAgente(agente, ultimaIdentidadCuentaDesdePresupuestos(agente.Nombre))
		if strings.TrimSpace(agente.CuentaID) == "" || strings.TrimSpace(agente.CuentaEmail) == "" || strings.TrimSpace(agente.CuentaUsuario) == "" {
			aplicarIdentidadCuentaAgente(agente, UltimaIdentidadCuentaObservadaAgente(agente.Nombre))
		}
	}
	return list, nil
}

func GetAgente(nombre string) (*Agente, error) {
	nombreCanonico, _, _, err := resolverAgentePorNombreCI(strings.TrimSpace(nombre))
	if err != nil {
		return nil, err
	}
	row := DB.QueryRow(`
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes WHERE nombre = ?`, nombreCanonico)
	a := &Agente{}
	var ultima sql.NullTime
	var lastReset sql.NullTime
	var reanimar sql.NullTime
	var motivo sql.NullString
	if err := row.Scan(
		&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima,
		&a.ConsumoDiaSegundos, &a.ConsumoSemanalSegundos, &a.LimiteDiaSegundos,
		&a.LimiteSemanalSegundos, &lastReset, &a.EstadoCuota,
		&reanimar, &motivo,
	); err != nil {
		return nil, err
	}
	if ultima.Valid {
		a.UltimaSesion = &ultima.Time
	}
	if lastReset.Valid {
		a.LastUsageResetAt = &lastReset.Time
	}
	if reanimar.Valid {
		a.ReanimarAt = &reanimar.Time
	}
	a.MotivoPausa = motivo.String
	enriquecerAgenteConPresupuesto(a)
	sesionActiva, err := GetSesionActivaOperativa(a.Nombre, nil)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	aplicarEstadoVisibleAgente(a, sesionActiva)
	return a, nil
}

func enriquecerAgentesConPresupuesto(list []*Agente) {
	enriquecerAgentesSinCuotaProveedor(list)
	for _, agente := range list {
		enriquecerAgenteConPresupuesto(agente)
	}
}

func enriquecerAgenteConPresupuesto(a *Agente) {
	if a == nil {
		return
	}
	enriquecerAgenteSinCuotaProveedor(a)
	if AgenteSinCuotaProveedorEfectivo(a) {
		return
	}
	diario := presupuestoDerivadoDiarioAgente(a)
	semanal := presupuestoDerivadoSemanalAgente(a)
	a.PresupuestoDiarioPct = diario.pct
	a.PresupuestoDiarioResetAt = diario.resetAt
	a.PresupuestoSemanalPct = semanal.pct
	a.PresupuestoSemanalResetAt = semanal.resetAt
	candidato := seleccionarPresupuestoEfectivo([]presupuestoAgenteCandidato{diario, semanal})
	aplicarPresupuestoEfectivoAgente(a, candidato)

	p, _, err := UltimoPresupuestoAgente(a.Nombre)
	if err == nil && p != nil {
		presupuestoCuota := p
		if !PresupuestoSesionAportaCuota(p) {
			if conCuota, _, quotaErr := UltimoPresupuestoAgenteConCuota(a.Nombre); quotaErr == nil && conCuota != nil {
				presupuestoCuota = conCuota
			}
		}
		aplicarUsoObservadoAgente(a, p)
		ev, err := EvaluarPresupuestoSesion(presupuestoCuota)
		if err == nil {
			fresco := PresupuestoSesionFresco(presupuestoCuota)
			a.PresupuestoEstado = strings.TrimSpace(ev.Estado)
			a.PresupuestoFuente = strings.TrimSpace(presupuestoCuota.BudgetSource)
			a.PresupuestoCheckedAt = &presupuestoCuota.CheckedAt
			a.PresupuestoStale = !fresco
			a.PresupuestoSesionResetAt = presupuestoCuota.ResetAt
			a.RemainingSeconds = presupuestoCuota.RemainingSeconds
			a.RemainingMessages = presupuestoCuota.RemainingMessages
			a.RemainingTokens = presupuestoCuota.RemainingTokens
			a.RemainingCredits = presupuestoCuota.RemainingCredits
			aplicarIdentidadCuentaAgente(a, identidadCuentaDesdePresupuesto(p))
			var sesion presupuestoAgenteCandidato
			if observedPrimary, observedSecondary := presupuestosObservadosDesdeSnapshot(presupuestoCuota); observedPrimary.pct != nil || observedSecondary.pct != nil {
				if observedPrimary.pct != nil {
					a.PresupuestoSesionPct = observedPrimary.pct
					if observedPrimary.resetAt != nil {
						a.PresupuestoSesionResetAt = observedPrimary.resetAt
					}
					if fresco {
						sesion = observedPrimary
					}
				}
				if observedSecondary.pct != nil {
					a.PresupuestoSemanalPct = observedSecondary.pct
					if observedSecondary.resetAt != nil {
						a.PresupuestoSemanalResetAt = observedSecondary.resetAt
					}
					if fresco {
						semanal = observedSecondary
					}
				}
			}
			if ev.RemainingRatio != nil && fresco && sesion.pct == nil {
				pct := int(math.Round(*ev.RemainingRatio * 100))
				if pct < 0 {
					pct = 0
				}
				if pct > 100 {
					pct = 100
				}
				a.PresupuestoSesionPct = &pct
				sesion = presupuestoAgenteCandidato{
					pct:        &pct,
					windowKind: strings.TrimSpace(presupuestoCuota.WindowKind),
					resetAt:    presupuestoCuota.ResetAt,
					source:     strings.TrimSpace(presupuestoCuota.BudgetSource),
				}
			} else if strings.EqualFold(strings.TrimSpace(ev.Estado), "agotado") && fresco && sesion.pct == nil {
				pct := 0
				a.PresupuestoSesionPct = &pct
				sesion = presupuestoAgenteCandidato{
					pct:        &pct,
					windowKind: strings.TrimSpace(presupuestoCuota.WindowKind),
					resetAt:    presupuestoCuota.ResetAt,
					source:     strings.TrimSpace(presupuestoCuota.BudgetSource),
				}
			}
			candidato = seleccionarPresupuestoEfectivo([]presupuestoAgenteCandidato{sesion, diario, semanal})
			aplicarPresupuestoEfectivoAgente(a, candidato)
			if a.PresupuestoStale && strings.TrimSpace(a.PresupuestoEstado) == "ok" {
				a.PresupuestoEstado = "observado_stale"
			}
			if a.PresupuestoStale &&
				strings.EqualFold(strings.TrimSpace(presupuestoCuota.BudgetSource), "provider_backoff") &&
				a.PresupuestoSemanalPct != nil &&
				*a.PresupuestoSemanalPct > 0 {
				a.PresupuestoEstado = "observado_stale"
			}
			if !a.PresupuestoStale &&
				strings.EqualFold(strings.TrimSpace(presupuestoCuota.BudgetSource), "provider_backoff") &&
				strings.EqualFold(strings.TrimSpace(a.PresupuestoEstado), "agotado") {
				// Un provider_backoff agotado invalida los porcentajes derivados de uso
				// local: mantenerlos visibles produce contradicciones como "semanal 95%"
				// junto a "efectivo 0%". En este caso solo vale el bloqueo observado.
				a.PresupuestoSesionPct = nil
				a.PresupuestoDiarioPct = nil
				a.PresupuestoSemanalPct = nil
			}
		}
	}
	if strings.TrimSpace(a.CuentaID) == "" || strings.TrimSpace(a.CuentaEmail) == "" {
		aplicarIdentidadCuentaAgente(a, ultimaIdentidadCuentaDesdePresupuestos(a.Nombre))
	}
	if strings.TrimSpace(a.CuentaID) == "" || strings.TrimSpace(a.CuentaEmail) == "" {
		nombre := a.Nombre
		if handles, err := listarRuntimeHandlesIdentidadCuenta(&nombre); err == nil && len(handles) > 0 {
			aplicarIdentidadCuentaAgente(a, identidadCuentaDesdeHandle(handles[0]))
		}
	}
	if strings.TrimSpace(a.CuentaID) == "" || strings.TrimSpace(a.CuentaEmail) == "" {
		aplicarIdentidadCuentaAgente(a, UltimaIdentidadCuentaObservadaAgente(a.Nombre))
	}
	reconciliarPresupuestoCanonicoPorCuenta(a)
	proyectarEstadoCuotaVisibleDesdePresupuesto(a)
	sincronizarPresupuestoEfectivoVisible(a)
	reconciliarBloqueoPresupuestoFresco(a)
	reconciliarBloqueoPresupuestoStale(a)
}

func listarRuntimeHandlesIdentidadCuenta(agente *string) ([]*RuntimeHandle, error) {
	handles, err := ListarRuntimeHandlesCanonicosRecientes(agente)
	if err != nil {
		return nil, err
	}
	if mejor := mejorRuntimeHandleIdentidadCuenta(handles); mejor != nil {
		return []*RuntimeHandle{mejor}, nil
	}
	q := runtimeHandleSelectBase() + ` WHERE 1=1`
	var args []any
	if agente != nil && strings.TrimSpace(*agente) != "" {
		q += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*agente))
	}
	q += ` ORDER BY COALESCE(last_seen_at, updated_at, created_at) DESC, id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*RuntimeHandle{}
	for rows.Next() {
		handle, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, handle)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if mejor := mejorRuntimeHandleIdentidadCuenta(out); mejor != nil {
		return []*RuntimeHandle{mejor}, nil
	}
	return out, nil
}

func mejorRuntimeHandleIdentidadCuenta(handles []*RuntimeHandle) *RuntimeHandle {
	var mejor *RuntimeHandle
	mejorTieneIdentidad := false
	now := time.Now().UTC()
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		identidad := identidadCuentaDesdeHandle(handle)
		tieneIdentidad := !identidadCuentaVacia(identidad)
		if mejor == nil {
			mejor = handle
			mejorTieneIdentidad = tieneIdentidad
			continue
		}
		if tieneIdentidad != mejorTieneIdentidad {
			if tieneIdentidad {
				mejor = handle
				mejorTieneIdentidad = true
			}
			continue
		}
		if tieneIdentidad && identidadCuentaEsMejor(identidad, identidadCuentaDesdeHandle(mejor)) {
			mejor = handle
			mejorTieneIdentidad = true
			continue
		}
		if runtimeHandlePreferible(handle, mejor, now) {
			mejor = handle
			mejorTieneIdentidad = tieneIdentidad
		}
	}
	return mejor
}

func reconciliarPresupuestoCanonicoPorCuenta(a *Agente) {
	if a == nil || (strings.TrimSpace(a.CuentaID) == "" && strings.TrimSpace(a.CuentaEmail) == "") {
		return
	}
	p, _, err := UltimoPresupuestoCuentaCanonicaConCuota(strings.TrimSpace(a.CuentaID), strings.TrimSpace(a.CuentaEmail))
	if err != nil || p == nil {
		return
	}
	ev, err := EvaluarPresupuestoSesion(p)
	if err != nil {
		return
	}
	fresco := PresupuestoSesionFresco(p)
	derivedDaily := presupuestoDerivadoDiarioAgente(a)
	derivedWeekly := presupuestoDerivadoSemanalAgente(a)
	derivedDailyPct := derivedDaily.pct
	derivedDailyReset := derivedDaily.resetAt
	derivedWeeklyPct := derivedWeekly.pct
	derivedWeeklyReset := derivedWeekly.resetAt
	a.PresupuestoFuente = strings.TrimSpace(p.BudgetSource)
	a.PresupuestoCheckedAt = &p.CheckedAt
	a.PresupuestoStale = !fresco
	a.PresupuestoEstado = strings.TrimSpace(ev.Estado)
	a.PresupuestoSesionResetAt = p.ResetAt
	a.RemainingSeconds = p.RemainingSeconds
	a.RemainingMessages = p.RemainingMessages
	a.RemainingTokens = p.RemainingTokens
	a.RemainingCredits = p.RemainingCredits
	var sesion presupuestoAgenteCandidato
	var semanal presupuestoAgenteCandidato
	if observedPrimary, observedSecondary := presupuestosObservadosDesdeSnapshot(p); observedPrimary.pct != nil || observedSecondary.pct != nil {
		if observedPrimary.pct != nil {
			a.PresupuestoSesionPct = observedPrimary.pct
			if observedPrimary.resetAt != nil {
				a.PresupuestoSesionResetAt = observedPrimary.resetAt
			}
			if fresco {
				sesion = observedPrimary
			}
		}
		if observedSecondary.pct != nil {
			a.PresupuestoSemanalPct = observedSecondary.pct
			if observedSecondary.resetAt != nil {
				a.PresupuestoSemanalResetAt = observedSecondary.resetAt
			}
			if fresco {
				semanal = observedSecondary
			}
		}
	}
	if ev.RemainingRatio != nil && fresco && sesion.pct == nil {
		pct := int(math.Round(*ev.RemainingRatio * 100))
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		a.PresupuestoSesionPct = &pct
		sesion = presupuestoAgenteCandidato{
			pct:        &pct,
			windowKind: strings.TrimSpace(p.WindowKind),
			resetAt:    p.ResetAt,
			source:     strings.TrimSpace(p.BudgetSource),
		}
	} else if strings.EqualFold(strings.TrimSpace(ev.Estado), "agotado") && fresco && sesion.pct == nil && presupuestoEsVentanaCorta(strings.TrimSpace(p.WindowKind)) {
		pct := 0
		a.PresupuestoSesionPct = &pct
		sesion = presupuestoAgenteCandidato{
			pct:        &pct,
			windowKind: strings.TrimSpace(p.WindowKind),
			resetAt:    p.ResetAt,
			source:     strings.TrimSpace(p.BudgetSource),
		}
	}
	if a.PresupuestoStale && strings.TrimSpace(a.PresupuestoEstado) == "ok" {
		a.PresupuestoEstado = "observado_stale"
	}
	if a.PresupuestoStale && strings.EqualFold(strings.TrimSpace(p.BudgetSource), "provider_backoff") {
		a.PresupuestoEstado = "observado_stale"
	}
	weeklyEffective := presupuestoAgenteCandidato{
		pct:        derivedWeeklyPct,
		windowKind: "weekly",
		resetAt:    derivedWeeklyReset,
		source:     "derived_weekly",
	}
	if semanal.pct != nil && fresco {
		weeklyEffective = presupuestoAgenteCandidato{
			pct:        semanal.pct,
			windowKind: "weekly",
			resetAt:    firstNonNilTime(semanal.resetAt, derivedWeeklyReset),
			source:     strings.TrimSpace(p.BudgetSource),
		}
	}
	candidato := seleccionarPresupuestoEfectivo([]presupuestoAgenteCandidato{
		sesion,
		{
			pct:        derivedDailyPct,
			windowKind: "daily",
			resetAt:    derivedDailyReset,
			source:     "derived_daily",
		},
		weeklyEffective,
	})
	aplicarPresupuestoEfectivoAgente(a, candidato)
}

func aplicarUsoObservadoAgente(a *Agente, p *PresupuestoSesion) {
	if a == nil || p == nil {
		return
	}
	raw := mapFromJSON(p.RawSnapshotJSON)
	if raw == nil {
		return
	}
	sessionUsage, _ := raw["session_usage"].(map[string]any)
	if sessionUsage == nil {
		return
	}
	if totalRaw, ok := sessionUsage["total_tokens"]; ok {
		total := int64FromAny(totalRaw)
		a.ObservedUsageTokens = &total
	}
	if cost, ok := float64FromAny(sessionUsage["estimated_cost_usd"]); ok {
		a.ObservedUsageCostUSD = &cost
	}
	if count, ok := intFromAnyWithBool(sessionUsage["message_count"]); ok {
		a.ObservedUsageMessages = &count
	}
	if turns, ok := intFromAnyWithBool(sessionUsage["turns"]); ok {
		a.ObservedUsageTurns = &turns
	}
	if updatedAt := timeFromAny(sessionUsage["updated_at"]); updatedAt != nil {
		a.ObservedUsageUpdatedAt = updatedAt
	}
	a.ObservedSessionPath = strings.TrimSpace(stringFromMap(sessionUsage, "session_path", ""))
}

func proyectarEstadoCuotaVisibleDesdePresupuesto(a *Agente) {
	if a == nil {
		return
	}
	if a.SinCuotaProveedor {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "activo") {
		return
	}
	if a.PresupuestoCheckedAt == nil || a.PresupuestoStale {
		return
	}
	if a.CuotaRestantePct != nil && *a.CuotaRestantePct <= 0 {
		switch {
		case presupuestoEsVentanaSemanal(a.PresupuestoVentana):
			a.EstadoCuota = "agotado"
			if a.PresupuestoResetAt != nil && a.PresupuestoResetAt.After(time.Now().UTC()) {
				a.ReanimarAt = a.PresupuestoResetAt
			}
			if strings.TrimSpace(a.MotivoPausa) == "" {
				a.MotivoPausa = "Presupuesto semanal agotado observado"
			}
			return
		case presupuestoEsVentanaCorta(a.PresupuestoVentana):
			a.EstadoCuota = "enfriamiento"
			if a.PresupuestoResetAt != nil && a.PresupuestoResetAt.After(time.Now().UTC()) {
				a.ReanimarAt = a.PresupuestoResetAt
			}
			if strings.TrimSpace(a.MotivoPausa) == "" {
				a.MotivoPausa = "Ventana corta agotada observada"
			}
			return
		}
	}
	switch strings.ToLower(strings.TrimSpace(a.PresupuestoEstado)) {
	case "agotado":
		a.EstadoCuota = "agotado"
		if a.PresupuestoResetAt != nil && a.PresupuestoResetAt.After(time.Now().UTC()) {
			a.ReanimarAt = a.PresupuestoResetAt
		}
		if strings.TrimSpace(a.MotivoPausa) == "" {
			a.MotivoPausa = "Presupuesto agotado observado"
		}
	}
}

func sincronizarPresupuestoEfectivoVisible(a *Agente) {
	if a == nil {
		return
	}
	if AgenteSinCuotaProveedorEfectivo(a) {
		a.CuotaRestantePct = nil
		a.PresupuestoVentana = ""
		a.PresupuestoResetAt = nil
		return
	}
	now := time.Now().UTC()
	zero := 0
	if a.PresupuestoSemanalPct != nil && *a.PresupuestoSemanalPct <= 0 {
		a.CuotaRestantePct = &zero
		a.PresupuestoVentana = "weekly"
		if a.PresupuestoSemanalResetAt != nil {
			a.PresupuestoResetAt = a.PresupuestoSemanalResetAt
		}
		if a.ReanimarAt == nil || a.ReanimarAt.IsZero() || !a.ReanimarAt.After(now) {
			if a.PresupuestoSemanalResetAt != nil && a.PresupuestoSemanalResetAt.After(now) {
				a.ReanimarAt = a.PresupuestoSemanalResetAt
			}
		}
		return
	}
	if a.PresupuestoSesionPct != nil && *a.PresupuestoSesionPct <= 0 {
		a.CuotaRestantePct = &zero
		if strings.TrimSpace(a.PresupuestoVentana) == "" || !presupuestoEsVentanaSemanal(a.PresupuestoVentana) {
			if window := strings.TrimSpace(a.PresupuestoVentana); window == "" || !presupuestoEsVentanaCorta(window) {
				a.PresupuestoVentana = "5h"
			}
		}
		if a.PresupuestoSesionResetAt != nil {
			a.PresupuestoResetAt = a.PresupuestoSesionResetAt
		}
		if a.ReanimarAt == nil || a.ReanimarAt.IsZero() || !a.ReanimarAt.After(now) {
			if a.PresupuestoSesionResetAt != nil && a.PresupuestoSesionResetAt.After(now) {
				a.ReanimarAt = a.PresupuestoSesionResetAt
			}
		}
		return
	}
	providerBackoffStale := a.PresupuestoStale && strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "provider_backoff")
	if (strings.EqualFold(strings.TrimSpace(a.PresupuestoEstado), "agotado") && !providerBackoffStale) ||
		(a.RemainingCredits != nil && *a.RemainingCredits <= 0 && !providerBackoffStale) {
		a.CuotaRestantePct = &zero
		if presupuestoEsVentanaSemanal(strings.TrimSpace(a.PresupuestoVentana)) {
			if a.PresupuestoSemanalResetAt != nil {
				a.PresupuestoResetAt = a.PresupuestoSemanalResetAt
			}
		} else if a.PresupuestoSesionResetAt != nil {
			a.PresupuestoResetAt = a.PresupuestoSesionResetAt
		}
	}
}

func reconciliarBloqueoPresupuestoStale(a *Agente) {
	if a == nil {
		return
	}
	if AgenteSinCuotaProveedorEfectivo(a) {
		return
	}
	if !a.PresupuestoStale {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "provider_backoff") {
		return
	}
	if a.PresupuestoSemanalPct == nil || *a.PresupuestoSemanalPct <= 0 {
		return
	}
	if a.CuotaRestantePct == nil || *a.CuotaRestantePct <= 0 {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "agotado") &&
		!strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "enfriamiento") {
		return
	}
	a.EstadoCuota = "activo"
	a.PresupuestoEstado = "observado_stale"
	a.ReanimarAt = nil
	if strings.Contains(strings.ToLower(strings.TrimSpace(a.MotivoPausa)), "presupuesto agotado observado") ||
		strings.Contains(strings.ToLower(strings.TrimSpace(a.MotivoPausa)), "cuota agotada") {
		a.MotivoPausa = "Bloqueo de cuota stale pendiente de revalidación"
	}
}

func reconciliarBloqueoPresupuestoFresco(a *Agente) {
	if a == nil {
		return
	}
	if AgenteSinCuotaProveedorEfectivo(a) {
		return
	}
	if a.PresupuestoCheckedAt == nil || a.PresupuestoStale {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "agotado") &&
		!strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "enfriamiento") {
		return
	}
	if agenteMotivoPausaOperativaVisible(a.MotivoPausa) {
		return
	}
	if a.PresupuestoSemanalPct == nil || *a.PresupuestoSemanalPct <= 0 {
		return
	}
	if a.CuotaRestantePct == nil || *a.CuotaRestantePct <= 0 {
		return
	}
	a.EstadoCuota = "activo"
	a.ReanimarAt = nil
	motivo := strings.ToLower(strings.TrimSpace(a.MotivoPausa))
	if strings.Contains(motivo, "presupuesto") ||
		strings.Contains(motivo, "cuota") ||
		strings.Contains(motivo, "ventana corta") ||
		strings.Contains(motivo, "credit") ||
		strings.Contains(motivo, "crédit") {
		a.MotivoPausa = ""
	}
}

type presupuestoAgenteCandidato struct {
	pct        *int
	windowKind string
	resetAt    *time.Time
	source     string
}

func firstNonNilPct(primary, fallback *int) *int {
	if primary != nil {
		return primary
	}
	return fallback
}

func firstNonNilTime(primary, fallback *time.Time) *time.Time {
	if primary != nil {
		return primary
	}
	return fallback
}

func mejorPresupuestoDerivadoAgente(a *Agente) presupuestoAgenteCandidato {
	return seleccionarPresupuestoEfectivo([]presupuestoAgenteCandidato{
		presupuestoDerivadoDiarioAgente(a),
		presupuestoDerivadoSemanalAgente(a),
	})
}

func seleccionarPresupuestoEfectivo(candidatos []presupuestoAgenteCandidato) presupuestoAgenteCandidato {
	var weekly presupuestoAgenteCandidato
	var short presupuestoAgenteCandidato
	var fallback presupuestoAgenteCandidato
	for _, item := range candidatos {
		if item.pct == nil {
			continue
		}
		if presupuestoEsVentanaSemanal(item.windowKind) {
			weekly = item
			continue
		}
		if short.pct == nil && presupuestoEsVentanaCorta(item.windowKind) {
			short = item
			continue
		}
		if fallback.pct == nil {
			fallback = item
		}
	}
	if weekly.pct != nil {
		if *weekly.pct <= 0 {
			return weekly
		}
		if short.pct != nil {
			return short
		}
		return weekly
	}
	if short.pct != nil {
		return short
	}
	if fallback.pct != nil {
		return fallback
	}
	return presupuestoAgenteCandidato{}
}

func presupuestoEsVentanaSemanal(windowKind string) bool {
	return strings.EqualFold(strings.TrimSpace(windowKind), "weekly")
}

func presupuestoEsVentanaCorta(windowKind string) bool {
	windowKind = strings.ToLower(strings.TrimSpace(windowKind))
	switch windowKind {
	case "5h", "session", "provider":
		return true
	}
	return strings.HasSuffix(windowKind, "m")
}

func aplicarPresupuestoEfectivoAgente(a *Agente, candidato presupuestoAgenteCandidato) {
	if a == nil || candidato.pct == nil {
		return
	}
	a.CuotaRestantePct = candidato.pct
	a.PresupuestoVentana = candidato.windowKind
	a.PresupuestoResetAt = candidato.resetAt
	if strings.TrimSpace(a.PresupuestoFuente) == "" {
		a.PresupuestoFuente = candidato.source
	}
}

type identidadCuentaAgente struct {
	accountID  string
	usuario    string
	email      string
	fuente     string
	observedAt *time.Time
}

func aplicarIdentidadCuentaAgente(a *Agente, identidad identidadCuentaAgente) {
	if a == nil {
		return
	}
	identidad = normalizarIdentidadCuentaAgente(identidad)
	if identidadCuentaVacia(identidad) {
		return
	}
	actual := identidadCuentaDesdeAgente(a)
	if strings.TrimSpace(identidad.accountID) == "" &&
		strings.TrimSpace(identidad.email) == "" &&
		strings.TrimSpace(identidad.usuario) != "" &&
		identidadCuentaObservedAt(identidad).After(identidadCuentaObservedAt(actual)) {
		a.CuentaUsuario = strings.TrimSpace(identidad.usuario)
		if strings.TrimSpace(identidad.fuente) != "" {
			a.CuentaFuente = strings.TrimSpace(identidad.fuente)
		}
		if identidad.observedAt != nil && !identidad.observedAt.IsZero() {
			ts := identidad.observedAt.UTC()
			a.CuentaObservadaAt = &ts
		}
		return
	}
	if strings.TrimSpace(a.CuentaID) == "" && strings.TrimSpace(identidad.accountID) != "" {
		a.CuentaID = strings.TrimSpace(identidad.accountID)
	}
	if strings.TrimSpace(a.CuentaEmail) == "" && strings.TrimSpace(identidad.email) != "" {
		a.CuentaEmail = strings.TrimSpace(identidad.email)
	}
	if strings.TrimSpace(a.CuentaUsuario) == "" {
		if usuario := strings.TrimSpace(identidad.usuario); usuario != "" {
			a.CuentaUsuario = usuario
		} else if email := strings.TrimSpace(identidad.email); email != "" {
			if at := strings.Index(email, "@"); at > 0 {
				a.CuentaUsuario = email[:at]
			}
		}
	}
	if strings.TrimSpace(a.CuentaFuente) == "" && strings.TrimSpace(identidad.fuente) != "" {
		a.CuentaFuente = strings.TrimSpace(identidad.fuente)
	}
	if a.CuentaObservadaAt == nil && identidad.observedAt != nil && !identidad.observedAt.IsZero() {
		ts := identidad.observedAt.UTC()
		a.CuentaObservadaAt = &ts
	}
	if !identidadCuentaEsMejor(identidad, actual) {
		return
	}
	if strings.TrimSpace(identidad.accountID) != "" {
		a.CuentaID = strings.TrimSpace(identidad.accountID)
	}
	if strings.TrimSpace(identidad.email) != "" {
		a.CuentaEmail = strings.TrimSpace(identidad.email)
	}
	if strings.TrimSpace(identidad.usuario) != "" {
		incomingAt := identidadCuentaObservedAt(identidad)
		currentAt := identidadCuentaObservedAt(actual)
		if strings.TrimSpace(actual.usuario) == "" || !incomingAt.Before(currentAt) {
			a.CuentaUsuario = strings.TrimSpace(identidad.usuario)
		}
	} else if email := strings.TrimSpace(identidad.email); email != "" {
		if strings.TrimSpace(actual.usuario) == "" {
			if at := strings.Index(email, "@"); at > 0 {
				a.CuentaUsuario = email[:at]
			}
		}
	}
	if strings.TrimSpace(identidad.fuente) != "" {
		a.CuentaFuente = strings.TrimSpace(identidad.fuente)
	}
	if identidad.observedAt != nil && !identidad.observedAt.IsZero() {
		ts := identidad.observedAt.UTC()
		a.CuentaObservadaAt = &ts
	}
}

func identidadCuentaDesdeAgente(a *Agente) identidadCuentaAgente {
	if a == nil {
		return identidadCuentaAgente{}
	}
	return normalizarIdentidadCuentaAgente(identidadCuentaAgente{
		accountID:  strings.TrimSpace(a.CuentaID),
		usuario:    strings.TrimSpace(a.CuentaUsuario),
		email:      strings.TrimSpace(a.CuentaEmail),
		fuente:     strings.TrimSpace(a.CuentaFuente),
		observedAt: a.CuentaObservadaAt,
	})
}

func identidadCuentaMasReciente(actual, incoming *time.Time) bool {
	if incoming == nil || incoming.IsZero() {
		return false
	}
	if actual == nil || actual.IsZero() {
		return true
	}
	return incoming.UTC().After(actual.UTC())
}

func normalizarIdentidadCuentaAgente(identidad identidadCuentaAgente) identidadCuentaAgente {
	identidad.accountID = strings.TrimSpace(identidad.accountID)
	identidad.usuario = strings.TrimSpace(identidad.usuario)
	identidad.email = strings.TrimSpace(identidad.email)
	identidad.fuente = strings.TrimSpace(identidad.fuente)
	if identidad.usuario == "" && identidad.email != "" {
		if at := strings.Index(identidad.email, "@"); at > 0 {
			identidad.usuario = identidad.email[:at]
		}
	}
	if identidad.observedAt != nil && !identidad.observedAt.IsZero() {
		ts := identidad.observedAt.UTC()
		identidad.observedAt = &ts
	}
	return identidad
}

func identidadCuentaVacia(identidad identidadCuentaAgente) bool {
	return strings.TrimSpace(identidad.accountID) == "" &&
		strings.TrimSpace(identidad.email) == "" &&
		strings.TrimSpace(identidad.usuario) == ""
}

func identidadCuentaEsMejor(candidate, current identidadCuentaAgente) bool {
	candidate = normalizarIdentidadCuentaAgente(candidate)
	current = normalizarIdentidadCuentaAgente(current)
	if identidadCuentaVacia(current) {
		return !identidadCuentaVacia(candidate)
	}
	timeCandidate := identidadCuentaObservedAt(candidate)
	timeCurrent := identidadCuentaObservedAt(current)
	if !timeCandidate.IsZero() && !timeCurrent.IsZero() &&
		timeCandidate.Before(timeCurrent) &&
		strings.TrimSpace(candidate.accountID) == "" {
		return false
	}
	scoreCandidate := identidadCuentaScore(candidate)
	scoreCurrent := identidadCuentaScore(current)
	if scoreCandidate != scoreCurrent {
		return scoreCandidate > scoreCurrent
	}
	if !timeCandidate.Equal(timeCurrent) {
		return timeCandidate.After(timeCurrent)
	}
	return false
}

func identidadCuentaScore(identidad identidadCuentaAgente) int {
	score := 0
	if strings.TrimSpace(identidad.accountID) != "" {
		score += 1000
	}
	if strings.TrimSpace(identidad.email) != "" {
		score += 200
	}
	if strings.TrimSpace(identidad.usuario) != "" {
		score += 50
	}
	switch strings.ToLower(strings.TrimSpace(identidad.fuente)) {
	case "codex_profile_status":
		score += 90
	case "codex_auth":
		score += 80
	case "claude_rust_session_observed":
		score += 75
	case "codex_token_count_observed":
		score += 70
	case "runtime_handle":
		score += 60
	case "runtime_handle_profile":
		score += 45
	case "manual_observed_identity":
		score += 40
	case "runtime_order_send_instruction":
		score += 10
	}
	return score
}

func identidadCuentaObservedAt(identidad identidadCuentaAgente) time.Time {
	if identidad.observedAt == nil || identidad.observedAt.IsZero() {
		return time.Time{}
	}
	return identidad.observedAt.UTC()
}

func identidadCuentaDesdePresupuesto(p *PresupuestoSesion) identidadCuentaAgente {
	if p == nil {
		return identidadCuentaAgente{}
	}
	return identidadCuentaDesdeMapa(
		mapFromJSON(p.RawSnapshotJSON),
		strings.TrimSpace(p.BudgetSource),
		&p.CheckedAt,
	)
}

func ultimaIdentidadCuentaDesdePresupuestos(agente string) identidadCuentaAgente {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return identidadCuentaAgente{}
	}
	rows, err := DB.Query(`
		SELECT p.raw_snapshot_json, p.budget_source, p.checked_at
		FROM presupuestos_sesion p
		JOIN sesiones s ON s.id = p.sesion_id
		WHERE s.agente = ?
		ORDER BY p.checked_at DESC, p.id DESC
		LIMIT 10`, agente)
	if err != nil {
		return identidadCuentaAgente{}
	}
	defer rows.Close()

	var merged identidadCuentaAgente
	for rows.Next() {
		var rawJSON string
		var budgetSource string
		var checkedAt time.Time
		if err := rows.Scan(&rawJSON, &budgetSource, &checkedAt); err != nil {
			return identidadCuentaAgente{}
		}
		identidad := normalizarIdentidadCuentaAgente(identidadCuentaDesdeMapa(mapFromJSON(rawJSON), strings.TrimSpace(budgetSource), &checkedAt))
		if identidadCuentaVacia(identidad) {
			continue
		}
		if identidadCuentaVacia(merged) {
			merged = identidad
			continue
		}
		if strings.TrimSpace(merged.accountID) == "" && strings.TrimSpace(identidad.accountID) != "" {
			merged.accountID = strings.TrimSpace(identidad.accountID)
		}
		if strings.TrimSpace(merged.email) == "" && strings.TrimSpace(identidad.email) != "" {
			merged.email = strings.TrimSpace(identidad.email)
		}
		if strings.TrimSpace(merged.usuario) == "" && strings.TrimSpace(identidad.usuario) != "" {
			merged.usuario = strings.TrimSpace(identidad.usuario)
		}
	}
	return normalizarIdentidadCuentaAgente(merged)
}

func identidadCuentaDesdeHandle(handle *RuntimeHandle) identidadCuentaAgente {
	if handle == nil {
		return identidadCuentaAgente{}
	}
	identidad := identidadCuentaDesdeMapa(
		mapFromJSON(handle.MetadataJSON),
		"runtime_handle",
		handle.LastSeenAt,
	)
	if strings.TrimSpace(identidad.accountID) != "" || strings.TrimSpace(identidad.usuario) != "" || strings.TrimSpace(identidad.email) != "" {
		return identidad
	}
	perfil := perfilCuentaDesdeRenderedCommandHandle(handle.MetadataJSON)
	if strings.TrimSpace(perfil) == "" {
		return identidadCuentaAgente{}
	}
	return identidadCuentaAgente{
		accountID:  strings.TrimSpace(identidad.accountID),
		usuario:    perfil,
		fuente:     "runtime_handle_profile",
		observedAt: handle.LastSeenAt,
	}
}

func identidadCuentaDesdeMapa(raw map[string]any, fuente string, observedAt *time.Time) identidadCuentaAgente {
	if raw == nil {
		return identidadCuentaAgente{}
	}
	email := buscarCadenaRecursiva(raw,
		"account_email",
		"user_email",
		"identified_email",
		"login_email",
		"email",
		"correo",
		"mail",
	)
	usuario := buscarCadenaRecursiva(raw,
		"account_user",
		"account_name",
		"user_name",
		"username",
		"login",
		"usuario",
	)
	accountID := strings.TrimSpace(buscarCadenaRecursiva(raw,
		"account_id",
		"accountId",
		"acct_id",
		"user_id",
		"sub",
	))
	if accountID == "" && strings.TrimSpace(email) == "" && strings.TrimSpace(usuario) == "" {
		return identidadCuentaAgente{}
	}
	return identidadCuentaAgente{
		accountID:  accountID,
		usuario:    strings.TrimSpace(usuario),
		email:      strings.TrimSpace(email),
		fuente:     strings.TrimSpace(fuente),
		observedAt: observedAt,
	}
}

func perfilCuentaDesdeRenderedCommandHandle(metadataJSON string) string {
	meta := mapFromJSON(metadataJSON)
	if meta == nil {
		return ""
	}
	rendered := strings.TrimSpace(stringFromMap(meta, "rendered_command", ""))
	if rendered == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)codex-perfil'\s+'([^']+)'`)
	match := re.FindStringSubmatch(rendered)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func buscarCadenaRecursiva(raw any, claves ...string) string {
	switch value := raw.(type) {
	case map[string]any:
		for _, clave := range claves {
			if text := strings.TrimSpace(stringFromMap(value, clave, "")); text != "" {
				return text
			}
		}
		for _, nested := range value {
			if text := buscarCadenaRecursiva(nested, claves...); text != "" {
				return text
			}
		}
	case []any:
		for _, nested := range value {
			if text := buscarCadenaRecursiva(nested, claves...); text != "" {
				return text
			}
		}
	}
	return ""
}

func presupuestoDerivadoDiarioAgente(a *Agente) presupuestoAgenteCandidato {
	pct := porcentajeRestante(remainingBounded(a.ConsumoDiaSegundos, a.LimiteDiaSegundos), a.LimiteDiaSegundos)
	if pct == nil {
		return presupuestoAgenteCandidato{}
	}
	return presupuestoAgenteCandidato{
		pct:        pct,
		windowKind: "daily",
		resetAt:    nextDailyResetAt(),
		source:     "derived_daily",
	}
}

func presupuestosObservadosDesdeSnapshot(p *PresupuestoSesion) (presupuestoAgenteCandidato, presupuestoAgenteCandidato) {
	if p == nil {
		return presupuestoAgenteCandidato{}, presupuestoAgenteCandidato{}
	}
	raw := mapFromJSON(p.RawSnapshotJSON)
	if raw == nil {
		return presupuestoAgenteCandidato{}, presupuestoAgenteCandidato{}
	}
	rateLimits, _ := raw["rate_limits"].(map[string]any)
	if rateLimits == nil {
		return presupuestoAgenteCandidato{}, presupuestoAgenteCandidato{}
	}
	return presupuestoObservadoDesdeMapa(rateLimits, "primary", strings.TrimSpace(p.BudgetSource)),
		presupuestoObservadoDesdeMapa(rateLimits, "secondary", strings.TrimSpace(p.BudgetSource))
}

func presupuestoObservadoDesdeMapa(rateLimits map[string]any, key, source string) presupuestoAgenteCandidato {
	window, _ := rateLimits[key].(map[string]any)
	if window == nil {
		return presupuestoAgenteCandidato{}
	}
	usedPercent, ok := float64FromAny(window["used_percent"])
	if !ok {
		return presupuestoAgenteCandidato{}
	}
	pct := int(math.Round(100 - usedPercent))
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return presupuestoAgenteCandidato{
		pct:        &pct,
		windowKind: normalizarWindowKindObservado(key, intFromAny(window["window_minutes"])),
		resetAt:    timeFromAny(window["resets_at"]),
		source:     source,
	}
}

func normalizarWindowKindObservado(key string, minutes int) string {
	switch {
	case strings.EqualFold(strings.TrimSpace(key), "secondary") || minutes >= 7*24*60:
		return "weekly"
	case minutes == 300:
		return "5h"
	case minutes > 0:
		return fmt.Sprintf("%dm", minutes)
	default:
		return strings.TrimSpace(key)
	}
}

func float64FromAny(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		out, err := v.Float64()
		if err == nil {
			return out, true
		}
	case string:
		out, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return out, true
		}
	}
	return 0, false
}

func intFromAny(raw any) int {
	switch v := raw.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		out, err := v.Int64()
		if err == nil {
			return int(out)
		}
	case string:
		out, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return out
		}
	}
	return 0
}

func intFromAnyWithBool(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		out, err := v.Int64()
		if err == nil {
			return int(out), true
		}
	case string:
		out, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return out, true
		}
	}
	return 0, false
}

func timeFromAny(raw any) *time.Time {
	switch v := raw.(type) {
	case float64:
		if v <= 0 {
			return nil
		}
		ts := time.Unix(int64(v), 0).UTC()
		return &ts
	case int64:
		if v <= 0 {
			return nil
		}
		ts := time.Unix(v, 0).UTC()
		return &ts
	case json.Number:
		out, err := v.Int64()
		if err == nil && out > 0 {
			ts := time.Unix(out, 0).UTC()
			return &ts
		}
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if out, err := strconv.ParseInt(v, 10, 64); err == nil && out > 0 {
			ts := time.Unix(out, 0).UTC()
			return &ts
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if ts, err := time.Parse(layout, v); err == nil {
				ts = ts.UTC()
				return &ts
			}
		}
	}
	return nil
}

func presupuestoDerivadoSemanalAgente(a *Agente) presupuestoAgenteCandidato {
	pct := porcentajeRestante(remainingBounded(a.ConsumoSemanalSegundos, a.LimiteSemanalSegundos), a.LimiteSemanalSegundos)
	if pct == nil {
		return presupuestoAgenteCandidato{}
	}
	return presupuestoAgenteCandidato{
		pct:        pct,
		windowKind: "weekly",
		resetAt:    nextWeeklyResetAt(a.LastUsageResetAt),
		source:     "derived_weekly",
	}
}

func remainingBounded(consumo int, limite int) int {
	if limite <= 0 {
		return 0
	}
	remaining := limite - consumo
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

func porcentajeRestante(remaining int, total int) *int {
	if total <= 0 {
		return nil
	}
	pct := int(math.Round(float64(remaining) * 100 / float64(total)))
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return &pct
}

func nextDailyResetAt() *time.Time {
	now := time.Now().UTC()
	reset := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)
	if !now.Before(reset) {
		reset = reset.Add(24 * time.Hour)
	}
	return &reset
}

func nextWeeklyResetAt(lastReset *time.Time) *time.Time {
	if lastReset != nil && !lastReset.IsZero() {
		next := lastReset.UTC().Add(7 * 24 * time.Hour)
		return &next
	}
	now := time.Now().UTC()
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC).AddDate(0, 0, daysUntilMonday)
	return &next
}

func resolverAgentePorNombreCI(nombre string) (string, string, bool, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "", "", false, sql.ErrNoRows
	}
	var (
		nombreCanonico string
		rol            string
		habilitado     bool
	)
	err := DB.QueryRow(`
		SELECT nombre, rol, habilitado
		FROM agentes
		WHERE nombre = ?
		LIMIT 1`, nombre,
	).Scan(&nombreCanonico, &rol, &habilitado)
	if err == nil {
		return nombreCanonico, rol, habilitado, nil
	}
	if err != sql.ErrNoRows {
		return "", "", false, err
	}
	var coincidencias int
	if err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM agentes
		WHERE lower(nombre) = lower(?)`, nombre,
	).Scan(&coincidencias); err != nil {
		return "", "", false, err
	}
	switch coincidencias {
	case 0:
		return "", "", false, sql.ErrNoRows
	case 1:
	default:
		return "", "", false, fmt.Errorf("nombre de agente ambiguo por mayusculas/minusculas: %s", nombre)
	}
	err = DB.QueryRow(`
		SELECT nombre, rol, habilitado
		FROM agentes
		WHERE lower(nombre) = lower(?)
		LIMIT 1`, nombre,
	).Scan(&nombreCanonico, &rol, &habilitado)
	if err != nil {
		return "", "", false, err
	}
	return nombreCanonico, rol, habilitado, nil
}

// CheckReanimaciones busca agentes cuya fecha de reanimación ha vencido.
func CheckReanimaciones() ([]*Agente, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes
		WHERE reanimar_at IS NOT NULL AND reanimar_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Agente
	for rows.Next() {
		a := &Agente{}
		var ultima, lastReset, reanimar sql.NullTime
		var motivo sql.NullString
		if err := rows.Scan(
			&a.Nombre, &a.Rol, &a.Activo, &a.Habilitado, &a.EstadoSesion, &ultima,
			&a.ConsumoDiaSegundos, &a.ConsumoSemanalSegundos, &a.LimiteDiaSegundos,
			&a.LimiteSemanalSegundos, &lastReset, &a.EstadoCuota,
			&reanimar, &motivo,
		); err != nil {
			return nil, err
		}
		if ultima.Valid {
			a.UltimaSesion = &ultima.Time
		}
		if lastReset.Valid {
			a.LastUsageResetAt = &lastReset.Time
		}
		if reanimar.Valid {
			a.ReanimarAt = &reanimar.Time
		}
		a.MotivoPausa = motivo.String
		list = append(list, a)
	}
	return list, rows.Err()
}

// ResetReanimacion limpia los campos de reanimación de un agente.
func ResetReanimacion(nombre string) error {
	_, err := DB.Exec(`UPDATE agentes SET reanimar_at = NULL, motivo_pausa = NULL, estado_cuota = 'activo' WHERE nombre = ?`, nombre)
	return err
}

// EliminarAgente borra físicamente un agente de la base de datos.
func EliminarAgente(nombre string) error {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return fmt.Errorf("agente obligatorio")
	}
	tareasActivas, err := listarTareasActivasAgente(nombre)
	if err != nil {
		return err
	}
	if len(tareasActivas) > 0 {
		return fmt.Errorf("no se puede eliminar el agente '%s': tiene %d tarea(s) activas (%s)", nombre, len(tareasActivas), resumirTareasActivasAgente(tareasActivas))
	}
	res, err := DB.Exec(`DELETE FROM agentes WHERE nombre = ?`, nombre)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("agente '%s' no encontrado", nombre)
	}
	return nil
}

func listarTareasActivasAgente(nombre string) ([]*Tarea, error) {
	tareas, err := ListarTareas(FiltroTareas{Agente: &nombre})
	if err != nil {
		return nil, err
	}
	activas := make([]*Tarea, 0, len(tareas))
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case TareaCompletada, TareaCancelada, TareaBacklog:
			continue
		default:
			activas = append(activas, tarea)
		}
	}
	return activas, nil
}

func resumirTareasActivasAgente(tareas []*Tarea) string {
	if len(tareas) == 0 {
		return ""
	}
	partes := make([]string, 0, min(len(tareas), 3))
	for i, tarea := range tareas {
		if i == 3 {
			partes = append(partes, "...")
			break
		}
		partes = append(partes, fmt.Sprintf("#%d %s", tarea.ID, tarea.Estado))
	}
	return strings.Join(partes, ", ")
}

// PausarAgente establece una pausa forzada (por rate-limit externo) para un agente.
func PausarAgente(nombre string, minutos int, motivo string) error {
	reanimar := time.Now().Add(time.Duration(minutos) * time.Minute)
	_, err := DB.Exec(`
		UPDATE agentes 
		SET estado_cuota = 'enfriamiento', reanimar_at = ?, motivo_pausa = ? 
		WHERE nombre = ?`, reanimar, motivo, nombre)
	return err
}

func PausarAgenteHasta(nombre string, reanimar time.Time, motivo string) error {
	if reanimar.IsZero() {
		reanimar = time.Now().UTC()
	}
	_, err := DB.Exec(`
		UPDATE agentes
		SET estado_cuota = 'enfriamiento', reanimar_at = ?, motivo_pausa = ?
		WHERE nombre = ?`, reanimar.UTC(), motivo, nombre)
	return err
}

// SetEstadoSesion actualiza el estado de actividad de un agente en sesión.
func SetEstadoSesion(agente, estado string) {
	if DB == nil || agente == "" {
		return
	}
	_, _ = DB.Exec(`UPDATE agentes SET estado_sesion=? WHERE nombre=? AND activo=1`, estado, agente)
}

func sessionOperationalGrace() time.Duration {
	tickSeconds := configIntOrDefault("agent_tick_seconds", 30)
	if tickSeconds <= 0 {
		tickSeconds = 30
	}
	grace := time.Duration(tickSeconds*4) * time.Second

	handleStaleSeconds := configIntOrDefault("runtime_handle_stale_seconds", 120)
	if handleStaleSeconds <= 0 {
		handleStaleSeconds = 120
	}
	handleGrace := time.Duration(handleStaleSeconds) * time.Second
	if handleGrace > grace {
		grace = handleGrace
	}
	if grace < 2*time.Minute {
		grace = 2 * time.Minute
	}
	return grace
}

func sessionOperationalCutoff() time.Time {
	return time.Now().UTC().Add(-sessionOperationalGrace())
}

func sessionOperationalCutoffSQL() string {
	return sessionOperationalCutoff().Format("2006-01-02 15:04:05")
}

func GetSesionAbierta(agente string, proyectoID *int64) (*Sesion, error) {
	q := `
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.agente = ? AND s.activa = 1`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND s.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY s.id DESC LIMIT 1`
	return consultarConReintentos(func() (*Sesion, error) {
		return escanearSesion(DB.QueryRow(q, args...))
	})
}

func GetSesionActiva(agente string, proyectoID *int64) (*Sesion, error) {
	q := `
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.agente = ? AND s.activa = 1
		  AND datetime(COALESCE(s.heartbeat_at, s.inicio)) >= datetime(?)`
	args := []any{agente, sessionOperationalCutoffSQL()}
	if proyectoID != nil {
		q += ` AND s.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY s.id DESC LIMIT 1`
	return consultarConReintentos(func() (*Sesion, error) {
		return escanearSesion(DB.QueryRow(q, args...))
	})
}

func GetSesionActivaOperativa(agente string, proyectoID *int64) (*Sesion, error) {
	sesion, err := GetSesionAbierta(agente, proyectoID)
	if err != nil {
		return nil, err
	}
	handlesActivos, err := listarHandlesActivosOperativosMap()
	if err != nil {
		return nil, err
	}
	ultimosHandles, err := listarUltimoHandlePorAgenteProyectoMap()
	if err != nil {
		return nil, err
	}
	ultimosRuntimes, err := listarUltimoRuntimePorAgenteProyectoMap()
	if err != nil {
		return nil, err
	}
	if !sesionEsActivaOperativaConMapas(sesion, handlesActivos, ultimosHandles, ultimosRuntimes, time.Now().UTC()) {
		return nil, sql.ErrNoRows
	}
	return sesion, nil
}

func SesionActivaDeAgente(agente string) (*Sesion, error) {
	return GetSesionActiva(agente, nil)
}

// RegistrarAgente añade un nuevo agente al sistema.
func RegistrarAgente(nombre, rol string) error {
	_, err := DB.Exec(
		`INSERT INTO agentes (nombre, rol, habilitado, retirado) VALUES (?,?,1,0)
		 ON CONFLICT(nombre) DO UPDATE SET rol=excluded.rol,
		   habilitado=CASE WHEN agentes.retirado=1 THEN 0 ELSE 1 END`,
		nombre, rol,
	)
	return err
}

func escanearSesion(s scanner) (*Sesion, error) {
	var sesion Sesion
	var conectorID sql.NullInt64
	var proyectoID sql.NullInt64
	var fin sql.NullTime
	var heartbeat sql.NullTime
	var pid sql.NullInt64
	err := s.Scan(
		&sesion.ID, &sesion.Agente, &conectorID, &sesion.ConectorSlug, &sesion.ConectorNombre,
		&proyectoID, &sesion.ProyectoSlug, &sesion.ProyectoNombre,
		&sesion.Inicio, &fin, &sesion.Activa, &sesion.Estado, &sesion.CWD, &sesion.Herramienta,
		&sesion.ExternalSessionID, &sesion.ResumePayloadJSON, &sesion.ResumenContinuidad,
		&sesion.Branch, &heartbeat, &sesion.Host, &pid,
	)
	if err != nil {
		return nil, err
	}
	if conectorID.Valid {
		sesion.ConectorID = &conectorID.Int64
	}
	if proyectoID.Valid {
		sesion.ProyectoID = &proyectoID.Int64
	}
	if fin.Valid {
		sesion.Fin = &fin.Time
	}
	if heartbeat.Valid {
		sesion.HeartbeatAt = &heartbeat.Time
	}
	if pid.Valid {
		sesion.PID = &pid.Int64
	}
	return &sesion, nil
}

// RetirarAgente deshabilita a un agente y libera su trabajo operativo.
// Pausa sus asignaciones activas/planificadas, devuelve al pool las tareas asignadas
// o en progreso y elimina votos pendientes en propuestas abiertas para no bloquear
// el consenso. La continuidad fina del runtime se gestiona en el control plane;
// aquí solo se deja el estado de BD en una situación replanificable.
func RetirarAgente(nombre string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	notaAsignacion := "agente_retirado"
	if _, err = tx.Exec(`
		UPDATE asignaciones
		SET estado='pausada',
		    nota=CASE
		    	WHEN trim(COALESCE(nota,''))='' THEN ?
		    	ELSE nota || char(10) || ?
		    END,
		    cerrada_at=NULL
		WHERE agente=? AND estado IN ('planificada','activa')`,
		notaAsignacion, notaAsignacion, nombre,
	); err != nil {
		return err
	}

	anotacionRetiro := formatearAnotacionTarea("server", "agente retirado automáticamente: "+nombre, time.Now().UTC())
	if _, err = tx.Exec(`
		UPDATE tareas
		SET estado='libre',
		    agente=NULL,
		    notas=COALESCE(notas,'') || ?
		WHERE agente=? AND estado IN ('asignada','en_progreso')`,
		anotacionRetiro, nombre,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		UPDATE runtime_mailbox
		SET estado='cancelado',
		    delivered_at=COALESCE(delivered_at, CURRENT_TIMESTAMP),
		    consumed_at=COALESCE(consumed_at, CURRENT_TIMESTAMP)
		WHERE to_agente=? AND estado IN ('pendiente','entregado')`,
		nombre,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		UPDATE runtime_orders
		SET estado='cancelada',
		    error_text=CASE
		    	WHEN trim(COALESCE(error_text,''))='' THEN 'agente_retirado'
		    	ELSE error_text
		    END,
		    finished_at=CURRENT_TIMESTAMP,
		    claimed_by='',
		    lease_token='',
		    lease_expires_at=NULL
		WHERE agente=?
		  AND estado IN ('pendiente','tomada','ejecutando')
		  AND tipo NOT IN ('pause','stop')`,
		nombre,
	); err != nil {
		return err
	}

	// Deshabilitar + marcar retirado + cerrar sesión activa
	if _, err = tx.Exec(
		`UPDATE agentes SET habilitado=0, retirado=1, activo=0, estado_sesion='' WHERE nombre=?`, nombre); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`UPDATE sesiones SET activa=0, fin=CURRENT_TIMESTAMP, estado='cerrada' WHERE agente=? AND activa=1`, nombre); err != nil {
		return err
	}
	// Eliminar votos pendientes en propuestas abiertas (para no bloquear consenso)
	if _, err = tx.Exec(`
		DELETE FROM votos
		WHERE agente=? AND posicion='pendiente'
		  AND propuesta_id IN (SELECT id FROM propuestas WHERE estado='abierta')`, nombre); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	Audit("server", "retirar_agente", "agente", 0, nombre)
	return nil
}

// RehabilitarAgente reactiva a un agente retirado.
func RehabilitarAgente(nombre string) error {
	res, err := DB.Exec(`UPDATE agentes SET habilitado=1, retirado=0 WHERE nombre=?`, nombre)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agente '%s' no encontrado", nombre)
	}
	Audit("server", "rehabilitar_agente", "agente", 0, nombre)
	return nil
}

// RegistrarAgenteAuto registra un agente con nombre canónico derivado del proveedor.
// El nombre se asigna por orden usando el siguiente número libre tras los existentes.
func RegistrarAgenteAuto(proveedor, rol string) (string, error) {
	prefijo, err := prefijoCanonicoAgente(proveedor)
	if err != nil {
		return "", err
	}
	rol = strings.TrimSpace(rol)
	if rol == "" {
		rol = "programador"
	}
	rows, err := DB.Query(`SELECT nombre FROM agentes WHERE lower(nombre) LIKE ? ORDER BY lower(nombre)`, strings.ToLower(prefijo)+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	maxN := 0
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return "", err
		}
		var n int
		suffix := strings.TrimPrefix(strings.ToLower(nombre), strings.ToLower(prefijo))
		if _, err := fmt.Sscanf(suffix, "%d", &n); err == nil && n > maxN {
			maxN = n
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	nombre := fmt.Sprintf("%s%d", prefijo, maxN+1)
	if err := RegistrarAgente(nombre, rol); err != nil {
		return "", err
	}
	return nombre, nil
}

// RegistrarCodex registra un agente Codex y devuelve el nombre asignado (Codex1, Codex2…).
func RegistrarCodex() (string, error) {
	return RegistrarAgenteAuto("codex", "programador")
}

func prefijoCanonicoAgente(proveedor string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(proveedor)) {
	case "", "codex", "openai":
		return "Codex", nil
	case "claude", "anthropic":
		return "Claude", nil
	case "gemini", "google":
		return "Gemini", nil
	}

	var sb strings.Builder
	for _, r := range strings.TrimSpace(proveedor) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	normalizado := sb.String()
	if normalizado == "" {
		return "", fmt.Errorf("proveedor de agente obligatorio")
	}
	lower := strings.ToLower(normalizado)
	if lower == "codex" {
		return "Codex", nil
	}
	if lower == "claude" {
		return "Claude", nil
	}
	if lower == "gemini" {
		return "Gemini", nil
	}
	return strings.ToUpper(normalizado[:1]) + strings.ToLower(normalizado[1:]), nil
}

// AuditEntry es una entrada del log de auditoría.
type AuditEntry struct {
	Agente    string
	Accion    string
	Entidad   string
	EntidadID int64
	Detalle   string
	CreatedAt time.Time
}

// AuditLog devuelve las últimas N entradas del log.
func AuditLog(limit int) ([]AuditEntry, error) {
	rows, err := DB.Query(
		`SELECT agente, accion, entidad, entidad_id, detalle, created_at
		 FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.Agente, &e.Accion, &e.Entidad, &e.EntidadID, &e.Detalle, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
