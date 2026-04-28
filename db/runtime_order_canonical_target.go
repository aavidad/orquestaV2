package db

import (
	"database/sql"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

func enriquecerResumeConContextoProyectoDB(resume *runtimeagente.ResumeContext, agente string, proyecto *Proyecto) {
	if resume == nil || proyecto == nil {
		return
	}
	if strings.TrimSpace(resume.ExternalSessionID) == "" &&
		strings.TrimSpace(resume.ResumePayloadJSON) == "" &&
		strings.TrimSpace(resume.ResumenContinuidad) == "" {
		return
	}
	contexto, resumen := BuildProjectContextSummary(agente, proyecto)
	if len(contexto) == 0 {
		return
	}
	if payload := AppendProjectContextPayload(resume.ResumePayloadJSON, contexto); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen != "" && !strings.Contains(resume.ResumenContinuidad, resumen) {
		if strings.TrimSpace(resume.ResumenContinuidad) == "" {
			resume.ResumenContinuidad = resumen
		} else {
			resume.ResumenContinuidad += ". " + resumen
		}
	}
}

func runtimeInstanceIfExists(runtimeID *int64) (*RuntimeInstance, error) {
	if runtimeID == nil || *runtimeID <= 0 {
		return nil, nil
	}
	runtime, err := GetRuntime(*runtimeID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func sesionIfExists(sesionID *int64) (*Sesion, error) {
	if sesionID == nil || *sesionID <= 0 {
		return nil, nil
	}
	if DB == nil {
		return nil, nil
	}
	sesion, err := GetSesionByID(*sesionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sesion, err
}

func resolverDestinoRuntimeOrderCanonico(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) (*RuntimeInstance, *RuntimeHandle, error) {
	if order == nil {
		return runtime, handle, nil
	}
	if handle != nil {
		validado, err := validarRuntimeHandleActivo(handle, order.Agente, order.ProyectoID)
		if err != nil {
			return nil, nil, err
		}
		if validado != nil && !runtimeHandleExcluidoDelActivoCanonico(validado) {
			handle = validado
			runtime, err = runtimeHandleRuntime(validado)
			if err != nil {
				return nil, nil, err
			}
			if err := refrescarRuntimeOrderDestinoCanonico(order, runtime, validado); err != nil {
				return nil, nil, err
			}
			if err := cerrarRuntimeHandlesActivosSupersededPorHandle(validado); err != nil {
				return nil, nil, err
			}
			return runtime, validado, nil
		}
		if runtimeOrderConservaHandleExplicitoParaControl(order, handle) {
			if runtime == nil {
				runtime, err = runtimeHandleRuntime(handle)
				if err != nil {
					return nil, nil, err
				}
			}
			return runtime, handle, nil
		}
		handle = nil
		runtime = nil
	}

	var (
		candidato *RuntimeHandle
		err       error
	)
	candidato, err = runtimeHandleCanonicoRecienteConFallback(order.Agente, order.ProyectoID)
	if err != nil || candidato == nil {
		return runtime, handle, err
	}
	runtime, err = runtimeHandleRuntime(candidato)
	if err != nil {
		return nil, nil, err
	}
	if err := refrescarRuntimeOrderDestinoCanonico(order, runtime, candidato); err != nil {
		return nil, nil, err
	}
	if err := cerrarRuntimeHandlesActivosSupersededPorHandle(candidato); err != nil {
		return nil, nil, err
	}
	return runtime, candidato, nil
}

func runtimeOrderConservaHandleExplicitoParaControl(order *RuntimeOrder, handle *RuntimeHandle) bool {
	if order == nil || handle == nil || order.HandleID == nil || *order.HandleID != handle.ID {
		return false
	}
	if runtimeHandleExcluidoDelActivoCanonico(handle) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "pause", "stop":
		return true
	case "resume":
		return strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") || RuntimeHandlePauseRequiresFreshStart(handle)
	default:
		return false
	}
}

func refrescarRuntimeOrderDestinoCanonico(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if order == nil || handle == nil {
		return nil
	}
	runtimeID := int64(0)
	if runtime != nil && runtime.ID > 0 {
		runtimeID = runtime.ID
	} else if resolvedRuntime, err := runtimeHandleRuntime(handle); err != nil {
		return err
	} else if resolvedRuntime != nil && resolvedRuntime.ID > 0 {
		runtimeID = resolvedRuntime.ID
	} else if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtimeID = *handle.RuntimeID
	}
	changed := order.HandleID == nil || *order.HandleID != handle.ID
	if !changed {
		switch {
		case runtimeID == 0 && order.RuntimeID != nil && *order.RuntimeID > 0:
			changed = true
		case runtimeID > 0 && (order.RuntimeID == nil || *order.RuntimeID != runtimeID):
			changed = true
		}
	}
	if changed && order.ID > 0 {
		if err := actualizarRuntimeOrderDestino(order.ID, runtime, handle); err != nil {
			return err
		}
	}
	order.HandleID = &handle.ID
	if runtimeID > 0 {
		order.RuntimeID = &runtimeID
	} else {
		order.RuntimeID = nil
	}
	return nil
}

func runtimePrincipalAgenteProyecto(agente string, proyectoID *int64) (*RuntimeInstance, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	handle, err := runtimeHandleCanonicoRecienteConFallback(agente, proyectoID)
	if err != nil {
		return nil, err
	}
	if handle != nil {
		runtime, err := runtimeHandleRuntime(handle)
		if err != nil {
			return nil, err
		}
		if runtime != nil {
			return runtime, nil
		}
	}
	q := runtimeSelectBase() + ` WHERE r.agente = ?`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND r.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY ` + runtimeActividadExpr("r") + ` DESC, r.id DESC LIMIT 16`
	items, err := consultarConReintentos(func() ([]*RuntimeInstance, error) {
		rows, err := DB.Query(q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var runtimes []*RuntimeInstance
		for rows.Next() {
			runtime, err := escanearRuntime(rows)
			if err != nil {
				return nil, err
			}
			runtimes = append(runtimes, runtime)
		}
		return runtimes, rows.Err()
	})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for _, runtime := range items {
		if runtimePuedeSerPrincipalSinHandle(runtime, now) {
			return runtime, nil
		}
	}
	return nil, nil
}

func runtimePuedeSerPrincipalSinHandle(runtime *RuntimeInstance, now time.Time) bool {
	if runtime == nil {
		return false
	}
	if !runtimeSostieneSesionOperativaConCutoff(runtime, sessionOperationalCutoff()) {
		return false
	}
	if strings.TrimSpace(runtime.ExternalSessionID) != "" {
		return true
	}
	if runtime.SesionID == nil || *runtime.SesionID <= 0 {
		return false
	}
	sesion, err := sesionIfExists(runtime.SesionID)
	if err != nil || sesion == nil {
		return false
	}
	return sesionOperativaPorHeartbeat(sesion.Activa, sesion.HeartbeatAt, sesion.Inicio, now)
}
