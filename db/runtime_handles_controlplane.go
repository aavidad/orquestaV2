package db

import (
	"database/sql"
	"encoding/json"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func GetRuntimeHandleBySesionID(sesionID int64) (*RuntimeHandle, error) {
	return consultarConReintentos(func() (*RuntimeHandle, error) {
		h, err := getRuntimeHandleByQuery(runtimeHandleSelectBase()+` WHERE sesion_id = ? ORDER BY id DESC LIMIT 1`, sesionID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return h, err
	})
}

func GetRuntimeHandle(id int64) (*RuntimeHandle, error) {
	return consultarConReintentos(func() (*RuntimeHandle, error) {
		h, err := getRuntimeHandleByQuery(runtimeHandleSelectBase()+` WHERE id = ?`, id)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return reconciliarMetadataRuntimeHandleLeida(h)
	})
}

func ActualizarMetadataRuntimeHandle(id int64, metadataJSON string) error {
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, strings.TrimSpace(metadataJSON), id); err != nil {
		return err
	}
	runtimeHandleHotReset()
	return nil
}

func GetRuntimeHandleActivoAgente(agente string) (*RuntimeHandle, error) {
	return GetRuntimeHandleActivoAgenteProyecto(agente, nil)
}

func GetRuntimeHandleActivoAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	handle, err := seleccionarRuntimeHandleActivo(agente, proyectoID)
	if err != nil || handle == nil {
		return handle, err
	}
	validado, err := validarRuntimeHandleActivo(handle, agente, proyectoID)
	if err != nil || validado == nil {
		return validado, err
	}
	if err := cerrarRuntimeHandlesActivosSupersededPorHandle(validado); err != nil {
		return nil, err
	}
	return validado, nil
}

func seleccionarRuntimeHandleActivo(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	candidates, err := listarRuntimeHandlesActivosCandidatos(agente, proyectoID)
	if err != nil || len(candidates) == 0 {
		return nil, err
	}
	preferido := elegirRuntimeHandleActivoPreferente(candidates)
	if preferido == nil {
		return nil, nil
	}
	return preferido, nil
}

func runtimeHandleExcluidoDelActivoCanonico(handle *RuntimeHandle) bool {
	if handle == nil {
		return true
	}
	switch strings.TrimSpace(handle.Estado) {
	case "activo", "pausado":
		return false
	default:
		return true
	}
}

func elegirRuntimeHandleActivoPreferente(handles []*RuntimeHandle) *RuntimeHandle {
	now := time.Now().UTC()
	var preferido *RuntimeHandle
	for _, handle := range handles {
		if runtimeHandleExcluidoDelActivoCanonico(handle) {
			continue
		}
		if preferido == nil || runtimeHandlePreferible(handle, preferido, now) {
			preferido = handle
		}
	}
	return preferido
}

func runtimeHandlePreferible(candidate, current *RuntimeHandle, now time.Time) bool {
	return runtimeHandlePreferibleConInfo(
		candidate,
		runtimeHandleWorkerInfoFor(candidate, now),
		current,
		runtimeHandleWorkerInfoFor(current, now),
		now,
	)
}

type runtimeHandleWorkerInfo struct {
	priority   int
	recency    time.Time
	fresh      bool
	structured bool
}

func runtimeHandlePreferibleConInfo(candidate *RuntimeHandle, candidateInfo runtimeHandleWorkerInfo, current *RuntimeHandle, currentInfo runtimeHandleWorkerInfo, now time.Time) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	if candidateInfo.priority != currentInfo.priority {
		return candidateInfo.priority > currentInfo.priority
	}
	if candidateInfo.fresh != currentInfo.fresh {
		return candidateInfo.fresh
	}
	if candidateInfo.structured != currentInfo.structured {
		return candidateInfo.structured
	}
	if !candidateInfo.recency.Equal(currentInfo.recency) {
		return candidateInfo.recency.After(currentInfo.recency)
	}
	return candidate.ID > current.ID
}

func runtimeHandleWorkerInfoFor(handle *RuntimeHandle, now time.Time) runtimeHandleWorkerInfo {
	if handle == nil {
		return runtimeHandleWorkerInfo{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	info := runtimeHandleWorkerInfo{recency: runtimeHandleRecency(handle)}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return info
	}
	driver := strings.ToLower(strings.TrimSpace(snap.Driver()))
	if driver == "" {
		driver = strings.ToLower(strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "driver", "")))
	}
	if driver != "process_pty_cli" {
		info.structured = true
	}
	info.fresh = snap.Alive() && !snap.IsHeartbeatStale(now, time.Minute)
	switch driver {
	case "tmux_cli_session":
		if info.fresh {
			info.priority = 40
		} else {
			info.priority = 30
		}
	case "process_pty_cli":
		if info.fresh {
			info.priority = 0
		} else {
			info.priority = -10
		}
	default:
		if info.fresh {
			info.priority = 5
		} else {
			info.priority = 1
		}
	}
	return info
}

func runtimeHandleWorkerPriority(handle *RuntimeHandle, now time.Time) int {
	return runtimeHandleWorkerInfoFor(handle, now).priority
}

func runtimeHandleRecency(handle *RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return handle.LastSeenAt.UTC()
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt.UTC()
	}
	return handle.CreatedAt.UTC()
}

func runtimeHandleTieneWorkerEstructurado(handle *RuntimeHandle) bool {
	return runtimeHandleWorkerInfoFor(handle, time.Now().UTC()).structured
}

func runtimeHandleTieneWorkerEstructuradoFresco(handle *RuntimeHandle, now time.Time) bool {
	return runtimeHandleWorkerInfoFor(handle, now).fresh
}

func cerrarRuntimeHandlesActivosSuperseded(preferido *RuntimeHandle, candidates []*RuntimeHandle) error {
	if preferido == nil || len(candidates) < 2 {
		return nil
	}
	for _, handle := range candidates {
		if handle == nil || handle.ID == preferido.ID {
			continue
		}
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='cerrado', last_seen_at=CURRENT_TIMESTAMP
			WHERE id = ? AND estado IN ('activo','pausado')`, handle.ID); err != nil {
			return err
		}
	}
	runtimeHandleHotReset()
	return nil
}

func cerrarRuntimeHandlesActivosSupersededPorHandle(preferido *RuntimeHandle) error {
	if preferido == nil {
		return nil
	}
	candidates, err := listarRuntimeHandlesActivosCandidatos(strings.TrimSpace(preferido.Agente), preferido.ProyectoID)
	if err != nil {
		return err
	}
	return cerrarRuntimeHandlesActivosSuperseded(preferido, candidates)
}

func listarRuntimeHandlesActivosCandidatos(agente string, proyectoID *int64) ([]*RuntimeHandle, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	q := runtimeHandleSelectBase() + ` WHERE agente = ? AND estado IN ('activo','pausado')`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += ` AND (proyecto_id = ? OR proyecto_id IS NULL)`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeHandle
	for rows.Next() {
		h, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func validarRuntimeHandleActivo(handle *RuntimeHandle, agente string, proyectoID *int64) (*RuntimeHandle, error) {
	validado, err := validarRuntimeHandleActivoSinFallback(handle)
	if err != nil || validado != nil {
		return validado, err
	}
	return recuperarRuntimeHandleVivoAgenteProyecto(agente, proyectoID)
}

func validarRuntimeHandleActivoSinFallback(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	if runtimeHandlePuedeRepresentarActivoCanonico(handle, now) {
		return handle, nil
	}
	return refrescarRuntimeHandleSiSigueVivo(handle)
}

func recuperarRuntimeHandleVivoAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	candidates, err := listarRuntimeHandlesActivosCandidatos(agente, proyectoID)
	if err != nil {
		return nil, err
	}
	for _, handle := range candidates {
		validado, err := refrescarRuntimeHandleSiSigueVivo(handle)
		if err != nil {
			return nil, err
		}
		if validado != nil {
			return validado, nil
		}
	}
	return nil, nil
}

func refrescarRuntimeHandleSiSigueVivo(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	if revive, estado, pid := runtimeHandleReviveStateFromStructuredWorker(handle, now); revive {
		return revivirRuntimeHandle(handle, estado, pid)
	}
	if runtimeHandleSePuedeValidarLocalmente(handle) {
		return refrescarRuntimeHandleProcesoLocal(handle)
	}
	if revived, err := supersedeRuntimeHandleSiHaceFalta(handle); err != nil {
		return nil, err
	} else if revived {
		return nil, nil
	}
	return nil, nil
}

func refrescarRuntimeHandleProcesoLocal(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil {
		return nil, err
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	vivo, pid, err := controlruntime.ProcesoVivo(obj)
	if err != nil {
		return nil, err
	}
	if vivo {
		pid64 := int64(pid)
		return revivirRuntimeHandle(handle, estadoReviveRuntimeHandle(handle), &pid64)
	}
	return nil, marcarRuntimeHandleFantasma(handle)
}

func revivirRuntimeHandle(handle *RuntimeHandle, estado string, pid *int64) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	if strings.TrimSpace(estado) == "" {
		estado = estadoReviveRuntimeHandle(handle)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?, last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, estado, handle.ID); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	handle.Estado = estado
	now := time.Now().UTC()
	handle.LastSeenAt = &now
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil {
		return handle, err
	}
	processState := "vivo"
	if pid == nil && runtime.PID != nil {
		pid = runtime.PID
	}
	if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state = ?, process_state = ?, pid = COALESCE(?, pid), last_event_at = CURRENT_TIMESTAMP
		WHERE id = ?`, estado, processState, pid, runtime.ID); err != nil {
		return nil, err
	}
	return handle, nil
}

func runtimeHandleReviveStateFromStructuredWorker(handle *RuntimeHandle, now time.Time) (bool, string, *int64) {
	if !runtimeHandleTieneWorkerEstructuradoFresco(handle, now) {
		return false, "", nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	worker, _ := meta["worker"].(map[string]any)
	if worker == nil {
		return false, "", nil
	}
	alive := boolFromAny(worker["alive"])
	if !alive {
		return false, "", nil
	}
	pid := int64PtrFromObservedAny(worker["pid"])
	switch strings.TrimSpace(stringFromMap(worker, "logical_state", "")) {
	case "pausado":
		return true, "pausado", pid
	case "activo", "ready", "running", "disponible":
		return true, "activo", pid
	default:
		return true, estadoReviveRuntimeHandle(handle), pid
	}
}

func runtimeHandleConfiaEstadoReciente(handle *RuntimeHandle, now time.Time) bool {
	if handle == nil {
		return false
	}
	recency := runtimeHandleRecency(handle)
	return !recency.IsZero() && now.Sub(recency) <= 90*time.Second
}

func runtimeHandlePuedeRepresentarActivoCanonico(handle *RuntimeHandle, now time.Time) bool {
	if handle == nil {
		return false
	}
	switch strings.TrimSpace(handle.Estado) {
	case "activo", "pausado":
	default:
		return false
	}
	if runtimeHandleTieneWorkerEstructuradoFresco(handle, now) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return false
	}
	return runtimeHandleConfiaEstadoReciente(handle, now)
}

func supersedeRuntimeHandleSiHaceFalta(handle *RuntimeHandle) (bool, error) {
	if handle == nil {
		return false, nil
	}
	if debe, err := runtimeHandleDebeCederAWorkerEstructurado(handle); err != nil {
		return false, err
	} else if !debe {
		return false, nil
	}
	return true, marcarRuntimeHandleSuperseded(handle)
}

func runtimeHandleDebeCederAWorkerEstructurado(handle *RuntimeHandle) (bool, error) {
	if handle == nil {
		return false, nil
	}
	candidates, err := listarRuntimeHandlesActivosCandidatos(strings.TrimSpace(handle.Agente), handle.ProyectoID)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	for _, candidate := range candidates {
		if candidate == nil || candidate.ID == handle.ID {
			continue
		}
		if runtimeHandleTieneWorkerEstructuradoFresco(candidate, now) {
			return true, nil
		}
	}
	return false, nil
}

func marcarRuntimeHandleSuperseded(handle *RuntimeHandle) error {
	if handle == nil {
		return nil
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='cerrado', last_seen_at=CURRENT_TIMESTAMP
		WHERE id = ?`, handle.ID); err != nil {
		return err
	}
	runtimeHandleHotReset()
	return nil
}

func runtimeHandleSePuedeValidarLocalmente(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if runtimepolicy.RuntimeHandleUsaLegacyProcessPTY(mapFromJSON(handle.MetadataJSON)) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return true
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil || runtime.PID == nil {
		return false
	}
	return *runtime.PID > 0
}

func marcarRuntimeHandleFantasma(handle *RuntimeHandle) error {
	if handle == nil {
		return nil
	}
	if err := detenerRuntimeHandleObsoleto(handle); err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
		WHERE id = ?`, handle.ID); err != nil {
		return err
	}
	runtimeHandleHotReset()
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido',
		    process_state='finalizado',
		    last_event_at=CURRENT_TIMESTAMP
		WHERE id = ?`, runtime.ID); err != nil {
		return err
	}
	return nil
}

func detenerRuntimeHandleObsoleto(handle *RuntimeHandle) error {
	if handle == nil {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") ||
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "mcp_http") {
		return nil
	}
	runtime, _ := runtimeHandleRuntime(handle)
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	if detenido, err := controlruntime.DetenerSesionTMUXMetadata(obj.MetadataJSON); detenido || err != nil {
		return err
	}
	if runtimeHandleObsoletoRefiereProcesoActual(obj) {
		return nil
	}
	aplicado, pid, err := controlruntime.DetenerProceso(obj)
	if err != nil && !errorDetenerSesionObsoletaIgnorable(err) {
		return err
	}
	if !aplicado {
		if resolvedPID, ok, resolveErr := controlruntime.ResolverPID(obj); resolveErr != nil && !errorDetenerSesionObsoletaIgnorable(resolveErr) {
			return resolveErr
		} else if ok && resolvedPID > 0 {
			if runtimeHandleObsoletoRefierePIDActual(int64(resolvedPID)) {
				return nil
			}
			fallbackPID := int64(resolvedPID)
			aplicado, pid, err = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: &fallbackPID})
			if err != nil && !errorDetenerSesionObsoletaIgnorable(err) {
				return err
			}
		}
	}
	if pid > 0 {
		pid64 := int64(pid)
		for i := 0; i < 5; i++ {
			vivo, _, aliveErr := controlruntime.ProcesoVivo(controlruntime.ObjetivoProceso{PID: &pid64})
			if aliveErr != nil {
				if errorDetenerSesionObsoletaIgnorable(aliveErr) {
					return nil
				}
				return aliveErr
			}
			if !vivo {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}

func runtimeHandleObsoletoRefiereProcesoActual(obj controlruntime.ObjetivoProceso) bool {
	meta := mapFromJSON(obj.MetadataJSON)
	looksLikeTMUX := strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") ||
		strings.TrimSpace(stringFromMap(meta, "tmux_session", "")) != "" ||
		strings.TrimSpace(stringFromMap(meta, "tmux_pane_id", "")) != ""
	if obj.PID != nil &&
		runtimeHandleObsoletoRefierePIDActual(*obj.PID) &&
		!looksLikeTMUX {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(obj.HandleKind), "process") {
		if pid, err := strconv.ParseInt(strings.TrimSpace(obj.HandleRef), 10, 64); err == nil && runtimeHandleObsoletoRefierePIDActual(pid) {
			return true
		}
	}
	return false
}

func runtimeHandleObsoletoRefierePIDActual(pid int64) bool {
	return pid > 0 && pid == int64(os.Getpid())
}

func errorDetenerSesionObsoletaIgnorable(err error) bool {
	if err == nil {
		return true
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case raw == "":
		return true
	case strings.Contains(raw, "can't find session"):
		return true
	case strings.Contains(raw, "no server running"):
		return true
	case strings.Contains(raw, "failed to connect to server"):
		return true
	case strings.Contains(raw, "no such process"):
		return true
	case strings.Contains(raw, "process already finished"):
		return true
	case strings.Contains(raw, "os: process already finished"):
		return true
	default:
		return false
	}
}

func estadoReviveRuntimeHandle(handle *RuntimeHandle) string {
	if handle == nil {
		return "activo"
	}
	if handle.SesionID != nil {
		if sesion, err := sesionIfExists(handle.SesionID); err == nil && sesion != nil {
			switch strings.TrimSpace(sesion.Estado) {
			case "pausada":
				return "pausado"
			case "activa":
				return "activo"
			}
		}
	}
	return "activo"
}

func runtimeHandleRuntime(handle *RuntimeHandle) (*RuntimeInstance, error) {
	if handle == nil {
		return nil, nil
	}
	if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtime, err := GetRuntime(*handle.RuntimeID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if runtime != nil {
			return runtime, nil
		}
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		runtime, err := GetRuntimeBySesionID(*handle.SesionID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return runtime, err
	}
	return nil, nil
}

func ListarRuntimeHandles(agente *string) ([]*RuntimeHandle, error) {
	if agente != nil && strings.TrimSpace(*agente) != "" {
		if _, err := GetRuntimeHandleCanonicoRecienteAgente(strings.TrimSpace(*agente)); err != nil {
			return nil, err
		}
	}
	handles, err := listarRuntimeHandlesLeidos(agente)
	if err != nil {
		return nil, err
	}
	if err := reconciliarRuntimeHandlesInvalidosEnLista(handles); err != nil {
		return nil, err
	}
	if err := reconciliarRuntimeHandlesDuplicadosEnLista(handles); err != nil {
		return nil, err
	}
	return handles, nil
}

func ListarRuntimeHandlesPasivos(agente *string) ([]*RuntimeHandle, error) {
	return listarRuntimeHandlesLeidos(agente)
}

func listarRuntimeHandlesLeidos(agente *string) ([]*RuntimeHandle, error) {
	q := runtimeHandleSelectBase() + ` WHERE 1=1`
	var args []any
	if agente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*agente)
		if err != nil {
			return nil, err
		}
		q += ` AND agente = ?`
		args = append(args, agenteCanonico)
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeHandle
	for rows.Next() {
		h, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i, h := range out {
		out[i], err = compactarMetadataRuntimeHandleEnMemoria(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func reconciliarRuntimeHandlesDuplicadosEnLista(handles []*RuntimeHandle) error {
	if len(handles) < 2 {
		return nil
	}
	grupos := map[string][]*RuntimeHandle{}
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		estado := strings.TrimSpace(handle.Estado)
		if estado != "activo" && estado != "pausado" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(handle.Agente))
		grupos[key] = append(grupos[key], handle)
	}
	now := time.Now().UTC()
	for _, grupo := range grupos {
		if len(grupo) < 2 {
			continue
		}
		preferido := elegirRuntimeHandleActivoPreferente(grupo)
		if preferido == nil || !runtimeHandleWorkerInfoFor(preferido, now).fresh {
			continue
		}
		ids := make([]int64, 0, len(grupo)-1)
		for _, handle := range grupo {
			if handle == nil || handle.ID == preferido.ID {
				continue
			}
			ids = append(ids, handle.ID)
		}
		if len(ids) == 0 {
			continue
		}
		args := make([]any, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='cerrado',
			    last_seen_at=CURRENT_TIMESTAMP
			WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`) AND estado IN ('activo','pausado')`, args...); err != nil {
			return err
		}
		runtimeHandleHotReset()
		for _, handle := range grupo {
			if handle == nil || handle.ID == preferido.ID {
				continue
			}
			handle.Estado = "cerrado"
			ts := now
			handle.LastSeenAt = &ts
		}
	}
	return nil
}

func reconciliarRuntimeHandlesInvalidosEnLista(handles []*RuntimeHandle) error {
	if len(handles) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		switch strings.TrimSpace(handle.Estado) {
		case "activo", "pausado":
		default:
			continue
		}
		if !runtimeHandleExternalSessionIncompatible(handle) {
			continue
		}
		if err := MarcarRuntimeHandleCanalRoto(handle, nil, "external_session_id incompatible with tmux premium runtime"); err != nil {
			return err
		}
		handle.Estado = "fallido"
		ts := now
		handle.LastSeenAt = &ts
	}
	return nil
}

func ListarRuntimeHandlesParaTranscript() ([]*RuntimeHandle, error) {
	if snapshot, err := runtimeHandleHotSnapshotCanonico(); err == nil && len(snapshot) > 0 {
		seen := make(map[int64]struct{}, len(snapshot))
		out := make([]*RuntimeHandle, 0, len(snapshot))
		for _, handle := range snapshot {
			if handle == nil {
				continue
			}
			switch strings.TrimSpace(handle.Estado) {
			case "activo", "pausado", "fallido":
			default:
				continue
			}
			if strings.TrimSpace(handle.MetadataJSON) == "" || !strings.Contains(handle.MetadataJSON, "log_path") {
				continue
			}
			if handle.ID > 0 {
				if _, ok := seen[handle.ID]; ok {
					continue
				}
				seen[handle.ID] = struct{}{}
			}
			compacted, compactErr := compactarMetadataRuntimeHandleEnMemoria(handle)
			if compactErr != nil {
				return nil, compactErr
			}
			out = append(out, compacted)
		}
		sort.SliceStable(out, func(i, j int) bool {
			return runtimeHandleRecency(out[i]).After(runtimeHandleRecency(out[j]))
		})
		if len(out) > 0 {
			return out, nil
		}
	}
	rows, err := DB.Query(runtimeHandleSelectBase() + `
		WHERE estado IN ('activo','pausado','fallido')
		  AND metadata_json <> ''
		  AND metadata_json LIKE '%log_path%'
		ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeHandle
	for rows.Next() {
		h, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i, h := range out {
		out[i], err = compactarMetadataRuntimeHandleEnMemoria(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func ListarRuntimeHandlesParaPresupuesto() ([]*RuntimeHandle, error) {
	if snapshot, err := runtimeHandleHotSnapshotCanonico(); err == nil && len(snapshot) > 0 {
		out := make([]*RuntimeHandle, 0, len(snapshot))
		for _, handle := range snapshot {
			if handle == nil || strings.TrimSpace(handle.MetadataJSON) == "" {
				continue
			}
			compacted, compactErr := compactarMetadataRuntimeHandleEnMemoria(handle)
			if compactErr != nil {
				return nil, compactErr
			}
			out = append(out, compacted)
		}
		sort.SliceStable(out, func(i, j int) bool {
			left := preferTime(out[i].LastSeenAt, &out[i].UpdatedAt, &out[i].CreatedAt)
			right := preferTime(out[j].LastSeenAt, &out[j].UpdatedAt, &out[j].CreatedAt)
			switch {
			case left == nil && right == nil:
				return out[i].ID > out[j].ID
			case left == nil:
				return false
			case right == nil:
				return true
			case !left.Equal(*right):
				return left.After(*right)
			default:
				return out[i].ID > out[j].ID
			}
		})
		return out, nil
	}

	rows, err := DB.Query(runtimeHandleSelectBase() + `
		WHERE estado IN ('activo','pausado')
		  AND metadata_json <> ''
		ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeHandle
	for rows.Next() {
		h, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i, h := range out {
		out[i], err = compactarMetadataRuntimeHandleEnMemoria(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func getRuntimeHandleByQuery(query string, args ...any) (*RuntimeHandle, error) {
	row := DB.QueryRow(query, args...)
	return scanRuntimeHandle(row)
}

func getRuntimeHandleRaw(id int64) (*RuntimeHandle, error) {
	h, err := getRuntimeHandleByQuery(runtimeHandleSelectBase()+` WHERE id = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func reconciliarMetadataRuntimeHandleLeida(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	refreshed, err := compactarMetadataHandleRuntimePersistida(handle)
	if err != nil {
		return nil, err
	}
	return normalizarRuntimeHandleTMUXCanonicoEnMemoria(refreshed), nil
}

func compactarMetadataRuntimeHandleEnMemoria(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		return handle, nil
	}
	before, _ := json.Marshal(meta)
	compactarMetadataRuntimeHandle(meta)
	after, _ := json.Marshal(meta)
	if string(before) == string(after) {
		return normalizarRuntimeHandleTMUXCanonicoEnMemoria(handle), nil
	}
	clone := *handle
	clone.MetadataJSON = string(after)
	return normalizarRuntimeHandleTMUXCanonicoEnMemoria(&clone), nil
}

func normalizarRuntimeHandleTMUXCanonicoEnMemoria(handle *RuntimeHandle) *RuntimeHandle {
	if handle == nil {
		return nil
	}
	if !runtimeHandleEsTMUXCanonico(handle) {
		return handle
	}
	canonicalRef := runtimeHandleTMUXSessionRefObserved(handle)
	if canonicalRef == "" {
		return handle
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") &&
		strings.TrimSpace(handle.HandleRef) == canonicalRef {
		return handle
	}
	clone := *handle
	clone.Transporte = "tmux"
	clone.HandleKind = "session"
	clone.HandleRef = canonicalRef
	return &clone
}

func int64PtrFromObservedAny(v any) *int64 {
	switch val := v.(type) {
	case int64:
		if val <= 0 {
			return nil
		}
		return &val
	case int:
		if val <= 0 {
			return nil
		}
		n := int64(val)
		return &n
	case float64:
		if val <= 0 {
			return nil
		}
		n := int64(val)
		return &n
	case json.Number:
		if n, err := val.Int64(); err == nil && n > 0 {
			return &n
		}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil && n > 0 {
			return &n
		}
	}
	return nil
}
