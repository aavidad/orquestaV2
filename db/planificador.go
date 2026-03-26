package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// PlanificarTareasAutomaticamente es el motor de autonomía de Orquesta.
// Busca agentes planificables y les asigna tareas libres de sus proyectos activos,
// además de liberar tareas del backlog cuyas dependencias ya se han cumplido.
func PlanificarTareasAutomaticamente() error {
	// 1. Liberar tareas del backlog que ya no tienen dependencias pendientes
	if err := liberarBacklog(); err != nil {
		return err
	}

	// 2. Buscar agentes planificables: agentes disponibles de verdad y agentes
	// recién incorporados que aún no han abierto su primera sesión manual.
	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		return err
	}

	for _, ag := range agentes {
		if err := planificarAgenteAutomaticamente(ag); err != nil {
			Audit("sistema", "auto_planificacion_error", "agente", 0,
				fmt.Sprintf("agente=%s error=%s", ag.Nombre, err.Error()))
		}
	}

	return nil
}

func planificarAgenteAutomaticamente(ag *Agente) error {
	if ag == nil {
		return nil
	}
	proyectoID, err := ResolverProyectoPlanificableAgente(ag.Nombre)
	if err != nil || proyectoID == 0 {
		return err // El agente no tiene un proyecto asignado ahora mismo
	}
	disponible, err := ProyectoDisponibleParaAutonomia(proyectoID)
	if err != nil {
		return err
	}
	if !disponible {
		return nil
	}
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		return err
	}

	tieneTrabajo, err := agenteTieneTrabajoArrancable(ag.Nombre, proyectoID)
	if err != nil {
		return err
	}
	if tieneTrabajo {
		return encolarStartAutomaticoSiHaceFalta(ag.Nombre, proyecto, "trabajo_asignado")
	}

	// Buscar la siguiente tarea libre para ese proyecto
	tarea, err := buscarSiguienteTareaLibre(proyectoID)
	if err != nil || tarea == nil {
		return err
	}

	// Asignar automáticamente
	if err := TomarTarea(tarea.ID, ag.Nombre); err != nil {
		return nil
	}
	Audit("sistema", "auto_asignacion", "tarea", tarea.ID,
		fmt.Sprintf("Asignada automáticamente a %s (Autonomía Total)", ag.Nombre))
	fmt.Printf("✓ [Planificador] Tarea #%d ('%s') asignada automáticamente a %s\n",
		tarea.ID, tarea.Titulo, ag.Nombre)
	return encolarStartAutomaticoSiHaceFalta(ag.Nombre, proyecto, fmt.Sprintf("tarea_autoasignada:%d", tarea.ID))
}

func agenteTieneTrabajoArrancable(agente string, proyectoID int64) (bool, error) {
	filtro := FiltroTareas{
		Agente:     &agente,
		ProyectoID: &proyectoID,
	}
	tareas, err := ListarTareas(filtro)
	if err != nil {
		return false, err
	}
	for _, tarea := range tareas {
		switch tarea.Estado {
		case TareaAsignada, TareaEnProgreso:
			return true, nil
		}
	}
	return false, nil
}

func proyectoTieneTrabajoPlanificableParaAgente(agente string, proyectoID int64) (bool, error) {
	tieneTrabajo, err := agenteTieneTrabajoArrancable(agente, proyectoID)
	if err != nil || tieneTrabajo {
		return tieneTrabajo, err
	}
	tarea, err := buscarSiguienteTareaLibre(proyectoID)
	if err != nil {
		return false, err
	}
	return tarea != nil, nil
}

func ResolverProyectoPlanificableAgente(agente string) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, nil
	}
	proyectoActivoID, err := ObtenerProyectoActivoAgente(agente)
	if err != nil {
		return 0, err
	}
	if proyectoActivoID != 0 {
		disponible, err := ProyectoDisponibleParaAutonomia(proyectoActivoID)
		if err != nil {
			return 0, err
		}
		tieneTrabajo := false
		if disponible {
			tieneTrabajo, err = proyectoTieneTrabajoPlanificableParaAgente(agente, proyectoActivoID)
		}
		if err != nil {
			return 0, err
		}
		if tieneTrabajo {
			return proyectoActivoID, nil
		}
		if !disponible {
			if err := PausarAsignacion(agente, proyectoActivoID, "proyecto_no_disponible"); err != nil {
				return 0, err
			}
			proyectoActivoID = 0
		} else {
			if err := PausarAsignacion(agente, proyectoActivoID, "sin_trabajo_espera_automatica"); err != nil {
				return 0, err
			}
			proyectoActivoID = 0
		}
	}

	asignacion, err := buscarAsignacionPausadaConTrabajo(agente)
	if err != nil {
		return 0, err
	}
	if asignacion != nil {
		if proyectoActivoID != 0 && proyectoActivoID != asignacion.ProyectoID {
			if err := PausarAsignacion(agente, proyectoActivoID, "sin_trabajo_reactivacion_automatica"); err != nil {
				return 0, err
			}
		}
		if err := ActivarAsignacion(agente, asignacion.ProyectoID, "reactivacion_automatica"); err != nil {
			return 0, err
		}
		return asignacion.ProyectoID, nil
	}

	proyectoAutomaticoID, err := seleccionarProyectoAutomaticoDisponible(agente)
	if err != nil {
		return 0, err
	}
	if proyectoAutomaticoID == 0 {
		return proyectoActivoID, nil
	}
	if proyectoActivoID != 0 && proyectoActivoID != proyectoAutomaticoID {
		if err := PausarAsignacion(agente, proyectoActivoID, "sin_trabajo_rebalanceo_automatico"); err != nil {
			return 0, err
		}
	}
	if err := ActivarAsignacion(agente, proyectoAutomaticoID, "asignacion_automatica_por_politica"); err != nil {
		return 0, err
	}
	return proyectoAutomaticoID, nil
}

func buscarAsignacionPausadaConTrabajo(agente string) (*Asignacion, error) {
	estado := AsignacionPausada
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
		Agente: &agente,
		Estado: &estado,
	})
	if err != nil {
		return nil, err
	}
	for _, asignacion := range asignaciones {
		if asignacion == nil {
			continue
		}
		disponible, err := ProyectoDisponibleParaAutonomia(asignacion.ProyectoID)
		if err != nil {
			return nil, err
		}
		if !disponible {
			continue
		}
		tieneTrabajo, err := proyectoTieneTrabajoPlanificableParaAgente(agente, asignacion.ProyectoID)
		if err != nil {
			return nil, err
		}
		if tieneTrabajo {
			return asignacion, nil
		}
	}
	return nil, nil
}

type candidatoProyectoAutomatico struct {
	Proyecto   *Proyecto
	Operacion  *ProyectoOperacion
	Activos    int
	TieneMin   bool
	CargaRatio int
}

func seleccionarProyectoAutomaticoDisponible(agente string) (int64, error) {
	_ = strings.TrimSpace(agente)
	activo := true
	proyectos, err := ListarProyectos(FiltroProyectos{Activo: &activo})
	if err != nil {
		return 0, err
	}
	activosPorProyecto, err := ContarAsignacionesActivasPorProyecto()
	if err != nil {
		return 0, err
	}
	var mejor *candidatoProyectoAutomatico
	for _, proyecto := range proyectos {
		if proyecto == nil || proyecto.ID == 0 {
			continue
		}
		disponible, err := ProyectoDisponibleParaAutonomia(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !disponible {
			continue
		}
		tarea, err := buscarSiguienteTareaLibre(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if tarea == nil {
			continue
		}
		op, err := GetProyectoOperacion(proyecto.ID)
		if err != nil {
			return 0, err
		}
		activos := activosPorProyecto[proyecto.ID]
		if op.MaxAgentes > 0 && activos >= op.MaxAgentes {
			continue
		}
		objetivo := op.ObjetivoPct
		if objetivo <= 0 {
			objetivo = 100
		}
		candidato := &candidatoProyectoAutomatico{
			Proyecto:   proyecto,
			Operacion:  op,
			Activos:    activos,
			TieneMin:   activos < op.MinAgentes,
			CargaRatio: (activos * 10000) / objetivo,
		}
		if mejorProyectoAutomatico(candidato, mejor) {
			mejor = candidato
		}
	}
	if mejor == nil || mejor.Proyecto == nil {
		return 0, nil
	}
	return mejor.Proyecto.ID, nil
}

func mejorProyectoAutomatico(candidato *candidatoProyectoAutomatico, actual *candidatoProyectoAutomatico) bool {
	if candidato == nil || candidato.Proyecto == nil {
		return false
	}
	if actual == nil || actual.Proyecto == nil {
		return true
	}
	if candidato.TieneMin != actual.TieneMin {
		return candidato.TieneMin
	}
	if candidato.CargaRatio != actual.CargaRatio {
		return candidato.CargaRatio < actual.CargaRatio
	}
	if candidato.Operacion != nil && actual.Operacion != nil && candidato.Operacion.Prioridad != actual.Operacion.Prioridad {
		return candidato.Operacion.Prioridad > actual.Operacion.Prioridad
	}
	return candidato.Proyecto.ID < actual.Proyecto.ID
}

func encolarStartAutomaticoSiHaceFalta(agente string, proyecto *Proyecto, motivo string) error {
	if proyecto == nil {
		return nil
	}
	sesionActiva, err := GetSesionActiva(agente, nil)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if sesionActiva != nil {
		return nil
	}
	handle, err := GetRuntimeHandleActivoAgente(agente)
	if err != nil {
		return err
	}
	if handle != nil {
		return nil
	}
	pendiente, err := existeRuntimeOrderAbierta(agente, nil, "start", "resume", "handoff")
	if err != nil {
		return err
	}
	if pendiente {
		return nil
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"accion":   "start",
		"proyecto": proyecto.Slug,
		"motivo":   motivo,
		"por":      "sistema",
	})
	if err != nil {
		return err
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      agente,
		ProyectoID:  &proyecto.ID,
		Tipo:        "start",
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		return err
	}
	Audit("sistema", "auto_start_runtime", "runtime_order", orderID,
		fmt.Sprintf("agente=%s proyecto=%s motivo=%s", agente, proyecto.Slug, motivo))
	return nil
}

func existeRuntimeOrderAbierta(agente string, proyectoID *int64, tipos ...string) (bool, error) {
	if len(tipos) == 0 {
		return false, nil
	}
	placeholders, tipoArgs := runtimeOrderPlaceholders(tipos)
	args := make([]any, 0, len(tipoArgs)+4)
	args = append(args, strings.TrimSpace(agente))
	args = append(args, tipoArgs...)
	q := `
		SELECT COUNT(*)
		FROM runtime_orders
		WHERE agente = ?
		  AND tipo IN (` + placeholders + `)
		  AND estado IN ('pendiente','tomada','ejecutando')`
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	var count int
	err := DB.QueryRow(q, args...).Scan(&count)
	return count > 0, err
}

func liberarBacklog() error {
	rows, err := DB.Query("SELECT id FROM tareas WHERE estado = 'backlog'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}

	for _, id := range ids {
		t, err := GetTarea(id)
		if err != nil {
			continue
		}
		// Si las dependencias están OK, la pasamos a libre
		if err := ValidarDependencias(t); err == nil {
			_, err = DB.Exec("UPDATE tareas SET estado = 'libre' WHERE id = ?", id)
			if err == nil {
				Audit("sistema", "liberar_backlog", "tarea", id, "Tarea liberada del backlog (dependencias cumplidas)")
			}
		}
	}
	return nil
}

func ListarAgentesPlanificables() ([]*Agente, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol 
		FROM agentes 
		WHERE habilitado = 1
		  AND rol = 'programador'
		  AND COALESCE(estado_cuota, 'activo') = 'activo'
		  AND COALESCE(estado_sesion, '') IN ('', 'disponible', 'esperando')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Agente
	for rows.Next() {
		var a Agente
		if err := rows.Scan(&a.Nombre, &a.Rol); err == nil {
			list = append(list, &a)
		}
	}
	return list, nil
}

// ListarAgentesDisponibles se mantiene por compatibilidad semántica con código
// anterior; ahora delega en la selección planificable oficial.
func ListarAgentesDisponibles() ([]*Agente, error) {
	return ListarAgentesPlanificables()
}

func ObtenerProyectoActivoAgente(agente string) (int64, error) {
	var id int64
	err := DB.QueryRow(`
		SELECT proyecto_id 
		FROM asignaciones 
		WHERE agente = ? AND estado = 'activa' 
		ORDER BY id DESC LIMIT 1`, agente).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func buscarSiguienteTareaLibre(proyectoID int64) (*Tarea, error) {
	q := FiltroTareas{
		ProyectoID: &proyectoID,
		Libre:      true,
	}
	list, err := ListarTareas(q)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	// ListarTareas ya ordena por prioridad
	return list[0], nil
}
