package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// PlanificarTareasAutomaticamente es el motor de autonomía de Orquesta.
// Busca agentes planificables y les asigna tareas libres de sus proyectos activos,
// además de liberar tareas del backlog cuyas dependencias ya se han cumplido.
func PlanificarTareasAutomaticamente() error {
	// 1. Liberar tareas del backlog que ya no tienen dependencias pendientes
	if err := liberarBacklog(); err != nil {
		return err
	}

	// 1.a Liberar tareas no iniciadas retenidas por agentes pausados por cuota
	// para que el planificador pueda redistribuirlas sin tocar trabajo en progreso.
	if err := reconciliarTareasAsignadasPorCuota(); err != nil {
		return err
	}

	// 1.b Recuperar tareas huérfanas que siguen asignadas a agentes sin
	// continuidad viva ni runtime activo para ese proyecto.
	if err := reconciliarTareasHuerfanas(); err != nil {
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

func tareaHuerfanaRecuperable(agente string, proyectoID int64) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	if agentePareceOperadorManualFueraDeFlota(agente) {
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
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if nombre == "" {
		return false
	}
	for _, prefix := range []string{"codex", "claude", "gemini", "ollama", "antigravity"} {
		if strings.HasPrefix(nombre, prefix) {
			return false
		}
	}
	return true
}

func ventanaGraciaRecuperacionTareaHuerfana() time.Duration {
	return time.Duration(configIntOrDefault("orphan_task_recovery_grace_seconds", 300)) * time.Second
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
	Workers    int
	Deseados   int
	Deficit    int
	TieneMin   bool
	CargaRatio int
}

func seleccionarProyectoAutomaticoDisponible(agente string) (int64, error) {
	agente = strings.TrimSpace(agente)
	activo := true
	proyectos, err := ListarProyectos(FiltroProyectos{Activo: &activo})
	if err != nil {
		return 0, err
	}
	totalPlanificables, err := contarAgentesPlanificables()
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
		reservado, _, err := agenteReservadoAutonomiaProyecto(agente, proyecto.ID)
		if err != nil {
			return 0, err
		}
		if reservado {
			continue
		}
		disponible, err := ProyectoDisponibleParaAutonomia(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !disponible {
			continue
		}
		tarea, err := buscarSiguienteTareaLibreParaAgente(agente, proyecto.ID)
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
		workers, err := contarWorkersActivosProyecto(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if op.MaxAgentes > 0 && activos >= op.MaxAgentes {
			continue
		}
		admiteWorker, err := proyectoAdmiteWorkerAutonomia(proyecto.ID)
		if err != nil {
			return 0, err
		}
		if !admiteWorker {
			continue
		}
		deseados := cupoDeseadoProyecto(op, totalPlanificables)
		candidato := &candidatoProyectoAutomatico{
			Proyecto:   proyecto,
			Operacion:  op,
			Activos:    activos,
			Workers:    workers,
			Deseados:   deseados,
			Deficit:    deseados - workers,
			TieneMin:   workers < op.MinAgentes,
			CargaRatio: cargaProyecto(workers, deseados),
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
	if candidato.Deficit > 0 || actual.Deficit > 0 {
		if candidato.Deficit != actual.Deficit {
			return candidato.Deficit > actual.Deficit
		}
	}
	if candidato.CargaRatio != actual.CargaRatio {
		return candidato.CargaRatio < actual.CargaRatio
	}
	if candidato.Operacion != nil && actual.Operacion != nil && candidato.Operacion.Prioridad != actual.Operacion.Prioridad {
		return candidato.Operacion.Prioridad > actual.Operacion.Prioridad
	}
	return candidato.Proyecto.ID < actual.Proyecto.ID
}

func contarAgentesPlanificables() (int, error) {
	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		return 0, err
	}
	if len(agentes) == 0 {
		return 1, nil
	}
	return len(agentes), nil
}

func cupoDeseadoProyecto(op *ProyectoOperacion, totalAgentes int) int {
	if totalAgentes <= 0 {
		totalAgentes = 1
	}
	if op == nil {
		return 1
	}
	objetivo := op.ObjetivoPct
	if objetivo <= 0 {
		objetivo = 100
	}
	deseados := (totalAgentes*objetivo + 99) / 100
	if deseados <= 0 {
		deseados = 1
	}
	if op.MinAgentes > deseados {
		deseados = op.MinAgentes
	}
	if op.MaxAgentes > 0 && deseados > op.MaxAgentes {
		deseados = op.MaxAgentes
	}
	if deseados <= 0 {
		return 1
	}
	return deseados
}

func cargaProyecto(activos, deseados int) int {
	if deseados <= 0 {
		deseados = 1
	}
	return (activos * 10000) / deseados
}

func agenteReservadoAutonomiaProyecto(agente string, proyectoID int64) (bool, string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, "", nil
	}
	policy, err := GetProyectoAutonomia(proyectoID)
	if err != nil || policy == nil || !policy.Enabled {
		return false, "", err
	}
	if policy.ReserveSupervisor && strings.EqualFold(agente, strings.TrimSpace(policy.SupervisorAgente)) {
		return true, "supervisor", nil
	}
	if policy.ReserveReviewer && strings.EqualFold(agente, strings.TrimSpace(policy.ReviewerAgente)) {
		return true, "reviewer", nil
	}
	return false, "", nil
}

func contarWorkersActivosProyecto(proyectoID int64) (int, error) {
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		return 0, err
	}
	estado := AsignacionActiva
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, asignacion := range asignaciones {
		if asignacion == nil {
			continue
		}
		reservado, _, err := agenteReservadoAutonomiaProyecto(asignacion.Agente, proyectoID)
		if err != nil {
			return 0, err
		}
		if reservado {
			continue
		}
		infoAgente, err := GetAgente(asignacion.Agente)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return 0, err
		}
		if infoAgente == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(infoAgente.EstadoCuota), "activo") {
			continue
		}
		if !agentePertenecePoolAutobootstrapProyecto(asignacion.Agente, proyecto) {
			continue
		}
		total++
	}
	return total, nil
}

func proyectoAdmiteWorkerAutonomia(proyectoID int64) (bool, error) {
	return proyectoAdmiteWorkerAutonomiaParaAgente(proyectoID, "")
}

func proyectoAdmiteWorkerAutonomiaParaAgente(proyectoID int64, agente string) (bool, error) {
	if proyectoID <= 0 {
		return true, nil
	}
	policy, err := GetProyectoAutonomia(proyectoID)
	if err != nil || policy == nil || !policy.Enabled || policy.MaxWorkers <= 0 {
		return true, err
	}
	workers, err := contarWorkersActivosProyecto(proyectoID)
	if err != nil {
		return false, err
	}
	agente = strings.TrimSpace(agente)
	if agente != "" {
		estado := AsignacionActiva
		asignaciones, err := ListarAsignaciones(FiltroAsignaciones{
			Agente:     &agente,
			ProyectoID: &proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return false, err
		}
		if len(asignaciones) > 0 {
			reservado, _, err := agenteReservadoAutonomiaProyecto(agente, proyectoID)
			if err != nil {
				return false, err
			}
			if !reservado && workers > 0 {
				workers--
			}
		}
	}
	return workers < policy.MaxWorkers, nil
}

func agentePertenecePoolAutobootstrapProyecto(agente string, proyecto *Proyecto) bool {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyecto == nil {
		return true
	}
	projectSlug := strings.TrimSpace(configOrDefault("server_autobootstrap_project_slug", "orquestador"))
	if projectSlug == "" || !strings.EqualFold(strings.TrimSpace(proyecto.Slug), projectSlug) {
		return true
	}
	permitidos := splitConfigAgentList(configOrDefault("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5"))
	if len(permitidos) == 0 {
		return true
	}
	for _, nombre := range permitidos {
		if strings.EqualFold(agente, nombre) {
			return true
		}
	}
	return false
}

func splitConfigAgentList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
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
	sesionActiva, err := GetSesionActiva(agente, nil)
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
	handleProyecto, err := runtimeHandleCanonicoRecienteConFallback(agente, &proyecto.ID)
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
	handle, err := runtimeHandleCanonicoRecienteConFallback(agente, nil)
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

func existeRuntimeOrderAbierta(agente string, proyectoID *int64, tipos ...string) (bool, error) {
	if len(tipos) == 0 {
		return false, nil
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Agente: stringsPtrTrimmed(agente),
		Estado: &estado,
	})
	if err != nil {
		return false, err
	}
	tiposWanted := make(map[string]struct{}, len(tipos))
	for _, tipo := range tipos {
		tipo = strings.TrimSpace(tipo)
		if tipo != "" {
			tiposWanted[tipo] = struct{}{}
		}
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		if proyectoID != nil && order.ProyectoID != nil && *order.ProyectoID != *proyectoID {
			continue
		}
		if _, ok := tiposWanted[strings.TrimSpace(order.Tipo)]; ok {
			return true, nil
		}
	}
	return false, nil
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

func buscarSiguienteTareaLibreParaAgente(agente string, proyectoID int64) (*Tarea, error) {
	q := FiltroTareas{
		ProyectoID: &proyectoID,
		Libre:      true,
	}
	list, err := ListarTareas(q)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	if strings.TrimSpace(agente) == "" || len(list) == 1 {
		return list[0], nil
	}
	moduloPreferido, err := moduloPreferidoAgenteProyecto(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return nil, err
	}
	modulosOcupados, err := modulosOcupadosProyecto(proyectoID, strings.TrimSpace(agente))
	if err != nil {
		return nil, err
	}
	candidatas := make([]*Tarea, 0, len(list))
	candidatas = append(candidatas, list...)
	sort.SliceStable(candidatas, func(i, j int) bool {
		return mejorTareaLibreParaAgente(candidatas[i], candidatas[j], moduloPreferido, modulosOcupados)
	})
	return candidatas[0], nil
}

func mejorTareaLibreParaAgente(a, b *Tarea, moduloPreferido string, modulosOcupados map[string]bool) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	scoreA := scoreTareaLibreParaAgente(a, moduloPreferido, modulosOcupados)
	scoreB := scoreTareaLibreParaAgente(b, moduloPreferido, modulosOcupados)
	if scoreA != scoreB {
		return scoreA > scoreB
	}
	return a.ID < b.ID
}

func scoreTareaLibreParaAgente(t *Tarea, moduloPreferido string, modulosOcupados map[string]bool) int {
	if t == nil {
		return -1 << 30
	}
	score := 0
	modulo := strings.TrimSpace(t.Modulo)
	if modulo != "" && moduloPreferido != "" && strings.EqualFold(modulo, moduloPreferido) {
		score += 1000
	}
	if modulo != "" && !modulosOcupados[strings.ToLower(modulo)] {
		score += 200
	}
	switch t.Prioridad {
	case PrioridadAlta:
		score += 30
	case PrioridadMedia:
		score += 20
	default:
		score += 10
	}
	return score
}

func moduloPreferidoAgenteProyecto(agente string, proyectoID int64) (string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return "", nil
	}
	rows, err := DB.Query(`
		SELECT modulo
		FROM tareas
		WHERE proyecto_id = ?
		  AND agente = ?
		  AND trim(modulo) <> ''
		  AND estado IN ('asignada','en_progreso','bloqueada','completada')
		ORDER BY
		  CASE estado
		    WHEN 'en_progreso' THEN 0
		    WHEN 'bloqueada' THEN 1
		    WHEN 'asignada' THEN 2
		    ELSE 3
		  END,
		  updated_at DESC,
		  id DESC
		LIMIT 5`, proyectoID, agente)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var modulo string
		if err := rows.Scan(&modulo); err != nil {
			return "", err
		}
		modulo = strings.TrimSpace(modulo)
		if modulo != "" {
			return modulo, nil
		}
	}
	return "", rows.Err()
}

func modulosOcupadosProyecto(proyectoID int64, agente string) (map[string]bool, error) {
	out := make(map[string]bool)
	if proyectoID <= 0 {
		return out, nil
	}
	rows, err := DB.Query(`
		SELECT DISTINCT COALESCE(modulo,''), COALESCE(agente,'')
		FROM tareas
		WHERE proyecto_id = ?
		  AND trim(COALESCE(modulo,'')) <> ''
		  AND trim(COALESCE(agente,'')) <> ''
		  AND estado IN ('asignada','en_progreso','bloqueada')`, proyectoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var modulo string
		var asignado string
		if err := rows.Scan(&modulo, &asignado); err != nil {
			return nil, err
		}
		modulo = strings.ToLower(strings.TrimSpace(modulo))
		asignado = strings.TrimSpace(asignado)
		if modulo == "" || strings.EqualFold(asignado, agente) {
			continue
		}
		out[modulo] = true
	}
	return out, rows.Err()
}
