package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/planificadorpolicy"
	"strings"
	"time"
)

// PlanificarTareasAutomaticamente es el motor de autonomía de Orquesta.
// Busca agentes planificables y les asigna tareas libres de sus proyectos activos,
// además de liberar tareas del backlog cuyas dependencias ya se han cumplido.
func PlanificarTareasAutomaticamente() error {
	if err := PrepararPlanificacionAutomatica(); err != nil {
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

// PrepararPlanificacionAutomatica deja el estado del backlog y de las tareas
// asignadas listo para un scheduler superior, sin autoasignar trabajo nuevo a
// agentes concretos. Se usa en el loop determinista de Orquesta para convivir
// con el pipeline nuevo sin duplicar asignaciones legacy.
func PrepararPlanificacionAutomatica() error {
	// 1. Liberar tareas del backlog que ya no tienen dependencias pendientes
	if err := liberarBacklog(); err != nil {
		return err
	}

	// 1.a Liberar tareas no iniciadas retenidas por agentes pausados por cuota
	// para que el planificador pueda redistribuirlas sin tocar trabajo en progreso.
	if err := reconciliarTareasAsignadasPorCuota(); err != nil {
		return err
	}

	// 1.a Saneamos primero el estado autónomo vivo para que el planificador
	// no construya decisiones sobre residuos de supervisor worker, aliases
	// no canónicos u órdenes pendientes heredadas fuera de canon.
	if err := ReconciliarEstadoAutonomia(); err != nil {
		return err
	}

	// 1.b Recuperar tareas huérfanas que siguen asignadas a agentes sin
	// continuidad viva ni runtime activo para ese proyecto.
	if err := reconciliarTareasHuerfanas(); err != nil {
		return err
	}
	return nil
}

// IntentarAutoasignarTareaAgente asigna una tarea libre al agente dentro del
// proyecto indicado si cumple la política de autonomía y no tiene ya trabajo
// arrancable. Está pensada para sesiones activas e idle que piden más trabajo.
func IntentarAutoasignarTareaAgente(agente string, proyectoID int64) (*Tarea, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return nil, nil
	}
	disponible, err := ProyectoDisponibleParaAutonomia(proyectoID)
	if err != nil || !disponible {
		return nil, err
	}
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		return nil, err
	}
	if !agentePertenecePoolAutobootstrapProyecto(agente, proyecto) {
		return nil, nil
	}
	tieneTrabajo, err := agenteTieneTrabajoArrancable(agente, proyectoID)
	if err != nil || tieneTrabajo {
		return nil, err
	}
	reservado, _, err := agenteReservadoAutonomiaProyecto(agente, proyectoID)
	if err != nil || reservado {
		return nil, err
	}
	admiteWorker, err := proyectoAdmiteWorkerAutonomiaParaAgente(proyectoID, agente)
	if err != nil || !admiteWorker {
		return nil, err
	}
	tarea, err := buscarSiguienteTareaLibreParaAgente(agente, proyectoID)
	if err != nil || tarea == nil {
		return nil, err
	}
	if err := TomarTarea(tarea.ID, agente); err != nil {
		return nil, nil
	}
	asignada, err := GetTarea(tarea.ID)
	if err != nil {
		return nil, err
	}
	Audit("sistema", "auto_asignacion_sesion_activa", "tarea", tarea.ID,
		fmt.Sprintf("Asignada automáticamente a %s desde sesión activa", agente))
	return asignada, nil
}

func reconciliarTareasAsignadasPorCuota() error {
	rows, err := DB.Query(`
		SELECT t.id, t.proyecto_id, t.agente, COALESCE(a.estado_cuota, 'activo')
		FROM tareas t
		JOIN agentes a ON a.nombre = t.agente
		WHERE t.proyecto_id IS NOT NULL
		  AND t.agente IS NOT NULL
		  AND trim(t.agente) <> ''
		  AND t.estado = 'asignada'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type tareaPorCuota struct {
		id          int64
		proyectoID  int64
		agente      string
		estadoCuota string
	}
	var observadas []tareaPorCuota
	for rows.Next() {
		var item tareaPorCuota
		if err := rows.Scan(&item.id, &item.proyectoID, &item.agente, &item.estadoCuota); err != nil {
			return err
		}
		observadas = append(observadas, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	candidatas := make([]tareaPorCuota, 0, len(observadas))
	for _, item := range observadas {
		infoAgente, err := GetAgente(strings.TrimSpace(item.agente))
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}
		if infoAgente == nil {
			continue
		}
		item.estadoCuota = strings.TrimSpace(infoAgente.EstadoCuota)
		if item.estadoCuota != "enfriamiento" && item.estadoCuota != "agotado" {
			continue
		}
		candidatas = append(candidatas, item)
	}

	for _, item := range candidatas {
		anotacion := formatearAnotacionTarea("server",
			fmt.Sprintf("tarea liberada automáticamente por cuota %s de %s", strings.TrimSpace(item.estadoCuota), strings.TrimSpace(item.agente)),
			time.Now().UTC())
		if _, err := DB.Exec(`
			UPDATE tareas
			SET estado='libre',
			    agente=NULL,
			    notas=COALESCE(notas,'') || ?
			WHERE id=?`,
			anotacion, item.id,
		); err != nil {
			return err
		}
		Audit("sistema", "liberar_tarea_por_cuota", "tarea", item.id,
			fmt.Sprintf("agente=%s proyecto_id=%d estado_cuota=%s", strings.TrimSpace(item.agente), item.proyectoID, strings.TrimSpace(item.estadoCuota)))
	}
	return nil
}

func reconciliarTareasWorkerEnSupervisorReservado() error {
	rows, err := DB.Query(`
		SELECT id, proyecto_id, agente
		FROM tareas
		WHERE proyecto_id IS NOT NULL
		  AND agente IS NOT NULL
		  AND trim(agente) <> ''
		  AND estado IN ('asignada','en_progreso')`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type tareaSupervisor struct {
		id         int64
		proyectoID int64
		agente     string
	}
	var candidatas []tareaSupervisor
	for rows.Next() {
		var item tareaSupervisor
		if err := rows.Scan(&item.id, &item.proyectoID, &item.agente); err != nil {
			return err
		}
		candidatas = append(candidatas, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, item := range candidatas {
		reservado, rol, err := agenteReservadoAutonomiaProyecto(strings.TrimSpace(item.agente), item.proyectoID)
		if err != nil || !reservado {
			if err != nil {
				return err
			}
			continue
		}
		if rol != "supervisor" && rol != "reviewer" {
			continue
		}
		tarea, err := GetTarea(item.id)
		if err != nil {
			return err
		}
		if tareaAgenteReservadoAutonomiaDebePreservarse(tarea, rol) {
			continue
		}
		anotacion := formatearAnotacionTarea("server",
			fmt.Sprintf("tarea worker liberada automáticamente del %s reservado %s", rol, strings.TrimSpace(item.agente)),
			time.Now().UTC())
		if _, err := DB.Exec(`
			UPDATE tareas
			SET estado='libre',
			    agente=NULL,
			    notas=COALESCE(notas,'') || ?
			WHERE id=?`,
			anotacion, item.id,
		); err != nil {
			return err
		}
		Audit("sistema", "liberar_tarea_agente_reservado_autonomia", "tarea", item.id,
			fmt.Sprintf("rol=%s agente=%s proyecto_id=%d", rol, strings.TrimSpace(item.agente), item.proyectoID))
	}
	return nil
}

func tareaAgenteReservadoAutonomiaDebePreservarse(tarea *Tarea, rol string) bool {
	if tarea == nil {
		return false
	}
	notas := strings.ToLower(strings.TrimSpace(tarea.Notas))
	switch strings.TrimSpace(strings.ToLower(rol)) {
	case "supervisor":
		switch {
		case strings.Contains(notas, "autonomia:post_remediation_blocked"):
			return true
		case strings.Contains(notas, "autonomia:needs_replan"):
			return true
		case strings.Contains(notas, "autonomia:blocked"):
			return true
		}
	case "reviewer":
		switch {
		case strings.Contains(notas, "review_gate"):
			return true
		case strings.Contains(notas, "review_feedback"):
			return true
		case strings.Contains(notas, "ready_for_review"):
			return true
		}
	}
	return false
}

func reconciliarAsignacionesSupervisorStale() error {
	enabled := true
	policies, err := ListarProyectosAutonomia(&enabled)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if policy == nil || !policy.Enabled || !policy.ReserveSupervisor || policy.ProyectoID <= 0 {
			continue
		}
		supervisor := strings.TrimSpace(policy.SupervisorAgente)
		if supervisor == "" {
			continue
		}
		estado := AsignacionActiva
		asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
			ProyectoID: &policy.ProyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return err
		}
		for _, asignacion := range asignaciones {
			if asignacion == nil {
				continue
			}
			nota := strings.ToLower(strings.TrimSpace(asignacion.Nota))
			if nota != "supervision" && nota != "supervision_automatica" && nota != "server_autobootstrap" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(asignacion.Agente), supervisor) {
				continue
			}
			if err := PausarAsignacion(strings.TrimSpace(asignacion.Agente), policy.ProyectoID, "supervision_stale_policy_mismatch"); err != nil {
				return err
			}
			Audit("sistema", "pausar_supervision_stale", "asignacion", asignacion.ID,
				fmt.Sprintf("agente=%s proyecto_id=%d supervisor_policy=%s", strings.TrimSpace(asignacion.Agente), policy.ProyectoID, supervisor))
		}
	}
	return nil
}

func reconciliarAsignacionesAliasNoCanonico() error {
	rows, err := DB.Query(`
		SELECT id, agente, proyecto_id, estado
		FROM asignaciones
		WHERE trim(COALESCE(agente,'')) <> ''
		  AND estado='activa'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type aliasActivo struct {
		id         int64
		agente     string
		proyectoID int64
		estado     string
	}
	var items []aliasActivo
	for rows.Next() {
		var item aliasActivo
		if err := rows.Scan(&item.id, &item.agente, &item.proyectoID, &item.estado); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, item := range items {
		canonico, err := CanonicalizeAgentName(strings.TrimSpace(item.agente))
		if err != nil {
			continue
		}
		if canonico == "" || canonico == strings.TrimSpace(item.agente) {
			continue
		}
		if _, err := DB.Exec(`UPDATE asignaciones SET estado='pausada', nota=?, cerrada_at=NULL WHERE id=?`,
			"alias_no_canonico:"+canonico, item.id); err != nil {
			return err
		}
		Audit("sistema", "pausar_asignacion_alias_no_canonico", "asignacion", item.id,
			fmt.Sprintf("agente=%s canonico=%s proyecto_id=%d", strings.TrimSpace(item.agente), canonico, item.proyectoID))
	}
	return nil
}

func reconciliarTareasHuerfanas() error {
	rows, err := DB.Query(`
		SELECT id, proyecto_id, agente, estado, updated_at
		FROM tareas
		WHERE proyecto_id IS NOT NULL
		  AND agente IS NOT NULL
		  AND trim(agente) <> ''
		  AND estado IN ('asignada','en_progreso')`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type tareaHuerfana struct {
		id         int64
		proyectoID int64
		agente     string
		estado     string
		updatedAt  time.Time
	}
	var candidatas []tareaHuerfana
	for rows.Next() {
		var item tareaHuerfana
		if err := rows.Scan(&item.id, &item.proyectoID, &item.agente, &item.estado, &item.updatedAt); err != nil {
			return err
		}
		candidatas = append(candidatas, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	limiteReciente := time.Now().UTC().Add(-ventanaGraciaRecuperacionTareaHuerfana())
	for _, item := range candidatas {
		if item.updatedAt.After(limiteReciente) {
			continue
		}
		recuperable, err := tareaHuerfanaRecuperable(strings.TrimSpace(item.agente), item.proyectoID)
		if err != nil {
			return err
		}
		if !recuperable {
			continue
		}
		tarea, err := GetTarea(item.id)
		if err != nil {
			return err
		}
		if tareaHuerfanaDebeConservarFrenteAcotado(tarea) {
			continue
		}
		anotacion := formatearAnotacionTarea("server", "tarea recuperada automáticamente por continuidad huérfana de "+strings.TrimSpace(item.agente), time.Now().UTC())
		if _, err := DB.Exec(`
			UPDATE tareas
			SET estado='libre',
			    agente=NULL,
			    notas=COALESCE(notas,'') || ?
			WHERE id=?`,
			anotacion, item.id,
		); err != nil {
			return err
		}
		if err := PausarAsignacion(strings.TrimSpace(item.agente), item.proyectoID, "tarea_huerfana_recuperada"); err != nil {
			return err
		}
		Audit("sistema", "recuperar_tarea_huerfana", "tarea", item.id,
			fmt.Sprintf("agente=%s proyecto_id=%d estado_previo=%s", strings.TrimSpace(item.agente), item.proyectoID, strings.TrimSpace(item.estado)))
	}

	return nil
}

func tareaHuerfanaDebeConservarFrenteAcotado(tarea *Tarea) bool {
	if tarea == nil {
		return false
	}
	return planificadorpolicy.ShouldPreserveOrphanTaskForFocusedFront(&planificadorpolicy.OrphanTaskContextSnapshot{
		Title:       strings.TrimSpace(tarea.Titulo),
		Description: strings.TrimSpace(tarea.Descripcion),
		Notes:       strings.TrimSpace(tarea.Notas),
	})
}

func tareaHuerfanaRecuperable(agente string, proyectoID int64) (bool, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return false, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	if agentePareceOperadorManualFueraDeFlota(agente) {
		return false, nil
	}
	if reservado, _, err := agenteReservadoAutonomiaProyecto(agente, proyectoID); err != nil {
		return false, err
	} else if reservado {
		return false, nil
	}
	if asignacion, err := GetAsignacionActivaAgente(agente); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if asignacion != nil && asignacion.ProyectoID != proyectoID {
		return false, nil
	}
	if sesion, err := GetSesionActiva(agente, &proyectoID); err != nil && err != sql.ErrNoRows {
		return false, err
	} else if sesion != nil {
		return false, nil
	}
	handleOperativo, err := runtimeHandleOperativoRecienteConFallback(agente, &proyectoID)
	if err != nil {
		return false, err
	}
	if handleOperativo != nil {
		return false, nil
	}
	var protegida int
	if err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM sesiones
		WHERE agente=?
		  AND proyecto_id=?
		  AND estado='pausada'
		  AND fin IS NULL`,
		agente, proyectoID,
	).Scan(&protegida); err != nil {
		return false, err
	}
	if protegida > 0 {
		return false, nil
	}
	pendiente, err := existeRuntimeOrderAbierta(agente, &proyectoID, "start", "resume", "handoff", "pause")
	if err != nil {
		return false, err
	}
	if pendiente {
		return false, nil
	}
	return true, nil
}

func agentePareceOperadorManualFueraDeFlota(nombre string) bool {
	return planificadorpolicy.LooksLikeManualOperatorOutsideFleet(nombre)
}

func ventanaGraciaRecuperacionTareaHuerfana() time.Duration {
	return planificadorpolicy.OrphanTaskRecoveryGraceWindow(
		configIntOrDefault("orphan_task_recovery_grace_seconds", 300),
	)
}

func planificarAgenteAutomaticamente(ag *Agente) error {
	if ag == nil {
		return nil
	}
	if permite, _, err := cuentaCompartidaPermiteActivacion(ag); err != nil {
		return err
	} else if !permite {
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
	if !agentePertenecePoolAutobootstrapProyecto(ag.Nombre, proyecto) {
		return nil
	}

	tieneTrabajo, err := agenteTieneTrabajoArrancable(ag.Nombre, proyectoID)
	if err != nil {
		return err
	}
	if tieneTrabajo {
		return EncolarStartAutomaticoSiHaceFalta(ag.Nombre, proyecto, "trabajo_asignado")
	}
	reservado, _, err := agenteReservadoAutonomiaProyecto(ag.Nombre, proyectoID)
	if err != nil {
		return err
	}
	if reservado {
		return nil
	}
	admiteWorker, err := proyectoAdmiteWorkerAutonomiaParaAgente(proyectoID, ag.Nombre)
	if err != nil {
		return err
	}
	if !admiteWorker {
		return nil
	}

	// Buscar la siguiente tarea libre para ese proyecto, evitando solapes de módulo
	// cuando exista otra opción y manteniendo afinidad con el frente reciente del agente.
	tarea, err := buscarSiguienteTareaLibreParaAgente(ag.Nombre, proyectoID)
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
	return EncolarStartAutomaticoSiHaceFalta(ag.Nombre, proyecto, fmt.Sprintf("tarea_autoasignada:%d", tarea.ID))
}

func agenteTieneTrabajoArrancable(agente string, proyectoID int64) (bool, error) {
	filtro := FiltroTareas{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Limit:      1,
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
	tareaActivaID, err := GetTareaActivaIDPorAgente(agente)
	if err != nil {
		return 0, err
	}
	if tareaActivaID > 0 {
		tareaActiva, err := GetTarea(tareaActivaID)
		if err != nil {
			return 0, err
		}
		if tareaActiva != nil && tareaActiva.ProyectoID != nil && *tareaActiva.ProyectoID > 0 {
			proyectoTareaID := *tareaActiva.ProyectoID
			disponible, err := ProyectoDisponibleParaAutonomia(proyectoTareaID)
			if err != nil {
				return 0, err
			}
			if disponible {
				if proyectoActivoID != 0 && proyectoActivoID != proyectoTareaID {
					if err := PausarAsignacion(agente, proyectoActivoID, "sesion_activa_en_otro_proyecto_con_tarea_viva"); err != nil {
						return 0, err
					}
				}
				if proyectoActivoID != proyectoTareaID {
					if err := ActivarAsignacion(agente, proyectoTareaID, "tarea_activa_prioritaria"); err != nil {
						return 0, err
					}
				}
				return proyectoTareaID, nil
			}
		}
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

func agentePertenecePoolAutobootstrapProyecto(agente string, proyecto *Proyecto) bool {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyecto == nil {
		return true
	}
	projectSlug := strings.TrimSpace(configOrDefault("server_autobootstrap_project_slug", "orquestador"))
	permitidos := splitConfigAgentList(configOrDefault("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5"))
	return planificadorpolicy.AgentAllowedForAutobootstrapProject(agente, strings.TrimSpace(proyecto.Slug), projectSlug, permitidos)
}

func splitConfigAgentList(raw string) []string {
	return planificadorpolicy.SplitAgentList(raw)
}

func EncolarStartAutomaticoSiHaceFalta(agente string, proyecto *Proyecto, motivo string) error {
	if proyecto == nil {
		return nil
	}
	if permite, ocupadoPor, err := CuentaCompartidaPermiteActivacionAgente(agente); err != nil {
		return err
	} else if !permite {
		Audit("sistema", "auto_start_runtime_cuenta_ocupada", "agente", 0,
			fmt.Sprintf("agente=%s proyecto=%s ocupado_por=%s motivo=%s", agente, proyecto.Slug, ocupadoPor, motivo))
		return nil
	}
	if permite, poolSlug, err := PoolLocalCompartidoPermiteActivacionAgenteProyecto(agente, proyecto.Slug); err != nil {
		return err
	} else if !permite {
		Audit("sistema", "auto_start_runtime_pool_local_sin_capacidad", "agente", 0,
			fmt.Sprintf("agente=%s proyecto=%s pool=%s motivo=%s", agente, proyecto.Slug, poolSlug, motivo))
		return nil
	}
	sesionActiva, err := GetSesionActivaOperativa(agente, nil)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if sesionActiva != nil && strings.TrimSpace(sesionActiva.Estado) != "pausada" {
		return nil
	}
	pendiente, err := existeRuntimeOrderAbierta(agente, nil, "start", "resume", "handoff")
	if err != nil {
		return err
	}
	if pendiente {
		return nil
	}
	handleProyecto, err := runtimeHandleOperativoRecienteConFallback(agente, &proyecto.ID)
	if err != nil {
		return err
	}
	if handleProyecto != nil {
		if strings.TrimSpace(handleProyecto.Estado) == "pausado" {
			payloadJSON, err := json.Marshal(map[string]any{
				"accion":   "resume",
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
				RuntimeID:   handleProyecto.RuntimeID,
				HandleID:    &handleProyecto.ID,
				Tipo:        "resume",
				PayloadJSON: string(payloadJSON),
			})
			if err != nil {
				return err
			}
			Audit("sistema", "auto_resume_runtime", "runtime_order", orderID,
				fmt.Sprintf("agente=%s proyecto=%s motivo=%s", agente, proyecto.Slug, motivo))
		}
		return nil
	}
	handle, err := runtimeHandleOperativoRecienteConFallback(agente, nil)
	if err != nil {
		return err
	}
	if handle != nil {
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
		if tareaBacklogDebeSeguirCompactada(t) {
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

func tareaBacklogDebeSeguirCompactada(t *Tarea) bool {
	if t == nil || t.ID <= 0 || t.ProyectoID == nil || *t.ProyectoID <= 0 {
		return false
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(t.Notas)), "autonomia:finish_app") {
		return false
	}
	var existe int
	if err := DB.QueryRow(`
		SELECT 1
		FROM tareas
		WHERE proyecto_id = ?
		  AND id <> ?
		  AND estado IN ('asignada','en_progreso','bloqueada')
		  AND LOWER(COALESCE(notas,'')) LIKE '%autonomia:finish_app%'
		LIMIT 1`,
		*t.ProyectoID, t.ID,
	).Scan(&existe); err != nil {
		return false
	}
	return existe == 1
}

func ListarAgentesPlanificables() ([]*Agente, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol 
		FROM agentes 
		WHERE habilitado = 1
		  AND rol = 'programador'
		  AND COALESCE(estado_sesion, '') IN ('', 'disponible', 'esperando')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidatos []*Agente
	for rows.Next() {
		var a Agente
		if err := rows.Scan(&a.Nombre, &a.Rol); err == nil {
			copia := a
			candidatos = append(candidatos, &copia)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var list []*Agente
	cuentasReservadas := map[string]int{}
	cuentaCeiling := cuentaCompartidaCeiling()
	for _, a := range candidatos {
		if a == nil {
			continue
		}
		infoAgente, err := GetAgente(a.Nombre)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		if infoAgente == nil || !strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "activo") {
			continue
		}
		if sesionActiva, err := GetSesionActiva(a.Nombre, nil); err != nil && err != sql.ErrNoRows {
			return nil, err
		} else if sesionActiva != nil && strings.TrimSpace(sesionActiva.Estado) != "pausada" {
			continue
		}
		handle, err := runtimeHandleCanonicoRecienteConFallback(a.Nombre, nil)
		if err != nil {
			return nil, err
		}
		if handle != nil && strings.TrimSpace(handle.Estado) != "pausado" {
			continue
		}
		if permite, _, err := cuentaCompartidaPermiteActivacion(infoAgente); err != nil {
			return nil, err
		} else if !permite {
			continue
		}
		if cuentaKey := CuentaClaveAgente(infoAgente); cuentaKey != "" {
			if _, ok := cuentasReservadas[cuentaKey]; !ok {
				ocupantes, err := agentesOcupandoCuentaCompartida(cuentaKey, a.Nombre)
				if err != nil {
					return nil, err
				}
				cuentasReservadas[cuentaKey] = len(ocupantes)
			}
			if cuentasReservadas[cuentaKey] >= cuentaCeiling {
				continue
			}
			cuentasReservadas[cuentaKey]++
		}
		list = append(list, a)
	}
	return priorizarAgentesPlanificables(list)
}

type candidatoPlanificable struct {
	Agente       *Agente
	PoolSlug     string
	Prioridad    int
	ProyectoID   int64
	ProyectoSlug string
}

func priorizarAgentesPlanificables(list []*Agente) ([]*Agente, error) {
	if len(list) <= 1 {
		return list, nil
	}
	candidatos := make([]candidatoPlanificable, 0, len(list))
	snapshots := make([]planificadorpolicy.PlannableAgentCandidateSnapshot, 0, len(list))
	agentesByName := make(map[string]*Agente, len(list))
	for _, agente := range list {
		if agente == nil {
			continue
		}
		agentesByName[strings.ToLower(strings.TrimSpace(agente.Nombre))] = agente
		candidato, err := describirCandidatoPlanificable(agente)
		if err != nil {
			return nil, err
		}
		candidatos = append(candidatos, candidato)
		snapshots = append(snapshots, planificadorpolicy.PlannableAgentCandidateSnapshot{
			AgentName:   strings.TrimSpace(agente.Nombre),
			PoolSlug:    strings.TrimSpace(candidato.PoolSlug),
			Priority:    candidato.Prioridad,
			ProjectSlug: strings.TrimSpace(candidato.ProyectoSlug),
		})
	}

	poolsDisponibles := map[string]int{}
	for _, candidato := range candidatos {
		if strings.TrimSpace(candidato.PoolSlug) != "" {
			if _, ok := poolsDisponibles[candidato.PoolSlug]; ok {
				continue
			}
			disponible, err := capacidadDisponiblePoolLocalCompartido(candidato.PoolSlug)
			if err != nil {
				return nil, err
			}
			poolsDisponibles[candidato.PoolSlug] = disponible
		}
	}

	orderedNames := planificadorpolicy.PrioritizePlannableCandidates(snapshots, poolsDisponibles)
	salida := make([]*Agente, 0, len(orderedNames))
	for _, name := range orderedNames {
		agente := agentesByName[strings.ToLower(strings.TrimSpace(name))]
		if agente == nil {
			continue
		}
		salida = append(salida, agente)
	}
	return salida, nil
}

func describirCandidatoPlanificable(agente *Agente) (candidatoPlanificable, error) {
	out := candidatoPlanificable{Agente: agente}
	if agente == nil {
		return out, nil
	}
	proyectoID, prioridad, err := resolverProyectoPreferentePlanificableAgenteModo(strings.TrimSpace(agente.Nombre), false)
	if err != nil {
		return out, err
	}
	out.ProyectoID = proyectoID
	out.Prioridad = prioridad
	if proyectoID <= 0 {
		return out, nil
	}
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		return out, err
	}
	if proyecto == nil {
		return out, nil
	}
	out.ProyectoSlug = strings.TrimSpace(proyecto.Slug)
	poolSlug, err := resolverPoolLocalCompartidoAgenteProyecto(strings.TrimSpace(agente.Nombre), out.ProyectoSlug)
	if err != nil {
		return out, err
	}
	out.PoolSlug = poolSlug
	return out, nil
}

func resolverProyectoPreferentePlanificableAgente(agente string) (int64, int, error) {
	return resolverProyectoPreferentePlanificableAgenteModo(agente, true)
}

func resolverProyectoPreferentePlanificableAgenteModo(agente string, incluirAutomatico bool) (int64, int, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, 0, nil
	}
	var (
		proyectoActivoID      int64
		proyectoActivoTrabajo bool
		proyectoPausadoID     int64
		proyectoAutomaticoID  int64
	)
	var err error
	if proyectoActivoID, err = ObtenerProyectoActivoAgente(agente); err != nil {
		return 0, 0, err
	} else if proyectoActivoID != 0 {
		disponible, err := ProyectoDisponibleParaAutonomia(proyectoActivoID)
		if err != nil {
			return 0, 0, err
		}
		if disponible {
			tieneTrabajo, err := proyectoTieneTrabajoPlanificableParaAgente(agente, proyectoActivoID)
			if err != nil {
				return 0, 0, err
			}
			proyectoActivoTrabajo = tieneTrabajo
		}
	}
	if asignacion, err := buscarAsignacionPausadaConTrabajo(agente); err != nil {
		return 0, 0, err
	} else if asignacion != nil {
		proyectoPausadoID = asignacion.ProyectoID
	}
	if incluirAutomatico {
		if proyectoAutomaticoID, err = seleccionarProyectoAutomaticoDisponible(agente); err != nil {
			return 0, 0, err
		}
	}
	resolution := planificadorpolicy.ResolvePreferredProject(
		proyectoActivoID,
		proyectoActivoTrabajo,
		proyectoPausadoID,
		proyectoAutomaticoID,
		incluirAutomatico,
	)
	return resolution.ProjectID, resolution.Priority, nil
}

func resolverPoolLocalCompartidoAgenteProyecto(agente, proyectoSlug string) (string, error) {
	agente = strings.TrimSpace(agente)
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	if agente == "" || proyectoSlug == "" {
		return "", nil
	}
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		AgenteNombre: &agente,
		ProyectoSlug: proyectoSlug,
	})
	if err != nil || resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return "", err
	}
	pool, err := GetPool(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil {
		return "", err
	}
	if !poolUsaConectorPoolLocalCompartido(pool) {
		return "", nil
	}
	return strings.TrimSpace(pool.Slug), nil
}

func capacidadDisponiblePoolLocalCompartido(poolSlug string) (int, error) {
	poolSlug = strings.TrimSpace(poolSlug)
	if poolSlug == "" {
		return 0, nil
	}
	pool, err := GetPool(poolSlug)
	if err != nil {
		return 0, err
	}
	resumen, err := ListarPoolsResumen(boolPtr(true))
	if err != nil {
		return 0, err
	}
	for _, item := range resumen {
		if item == nil || item.Pool == nil || item.Pool.ID != pool.ID {
			continue
		}
		reservasPendientes, err := reservasPendientesPoolLocalCompartido(pool.ID, poolSlug)
		if err != nil {
			return 0, err
		}
		disponible := item.CapacidadDisponible - reservasPendientes
		if disponible < 0 {
			disponible = 0
		}
		return disponible, nil
	}
	return 0, nil
}

// ListarAgentesDisponibles se mantiene por compatibilidad semántica con código
// anterior; ahora delega en la selección planificable oficial.
func ListarAgentesDisponibles() ([]*Agente, error) {
	return ListarAgentesPlanificables()
}

func ObtenerProyectoActivoAgente(agente string) (int64, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return 0, err
	}
	var id int64
	err = DB.QueryRow(`
		SELECT proyecto_id 
		FROM asignaciones 
		WHERE agente = ? AND estado = 'activa' 
		ORDER BY id DESC LIMIT 1`, agente).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}
