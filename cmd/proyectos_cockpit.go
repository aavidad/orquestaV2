package cmd

import (
	"database/sql"
	"strings"

	"orquesta/db"
)

type apiProyectoCockpitResponse struct {
	Cockpit *apiProyectoCockpit `json:"cockpit"`
}

type apiProyectoCockpit struct {
	Proyecto                *db.Proyecto               `json:"proyecto,omitempty"`
	TareasPorEstado         map[string]int             `json:"tareas_por_estado"`
	TareasActivas           []tareaLite                `json:"tareas_activas,omitempty"`
	TareasReservadas        []tareaLite                `json:"tareas_reservadas,omitempty"`
	AgentesActivos          []apiProyectoCockpitAgente `json:"agentes_activos,omitempty"`
	AsignacionesActivas     int                        `json:"asignaciones_activas"`
	PropuestasAbiertas      int                        `json:"propuestas_abiertas"`
	ReviewGatesAbiertas     int                        `json:"review_gates_abiertas"`
	RuntimeMailboxPendiente int                        `json:"runtime_mailbox_pendiente"`
	RuntimeOrdersAbiertas   int                        `json:"runtime_orders_abiertas"`
}

type apiProyectoCockpitAgente struct {
	Nombre string `json:"nombre"`
	Rol    string `json:"rol,omitempty"`
}

func buildProyectoCockpit(ref string) (*apiProyectoCockpit, error) {
	proyecto, err := db.GetProyectoConRutaEfectiva(strings.TrimSpace(ref), "")
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, nil
	}

	cockpit := &apiProyectoCockpit{
		Proyecto:        proyecto,
		TareasPorEstado: map[string]int{},
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyecto.ID})
	if err != nil {
		return nil, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		estado := strings.TrimSpace(string(tarea.Estado))
		if estado == "" {
			estado = "desconocido"
		}
		cockpit.TareasPorEstado[estado]++
		item := tareaLite{
			ID:        tarea.ID,
			Titulo:    strings.TrimSpace(tarea.Titulo),
			Agente:    safeStringPtr(tarea.Agente),
			Estado:    tarea.Estado,
			Modulo:    strings.TrimSpace(tarea.Modulo),
			Prioridad: tarea.Prioridad,
		}
		switch tarea.Estado {
		case db.TareaEnProgreso:
			cockpit.TareasActivas = append(cockpit.TareasActivas, item)
		case db.TareaAsignada:
			cockpit.TareasReservadas = append(cockpit.TareasReservadas, item)
		}
	}

	estadoAsignacion := db.AsignacionActiva
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{
		ProyectoID: &proyecto.ID,
		Estado:     &estadoAsignacion,
	})
	if err != nil {
		return nil, err
	}
	cockpit.AsignacionesActivas = len(asignaciones)

	agentesActivos, err := listarAgentesActivosProyecto(proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.AgentesActivos = agentesActivos

	cockpit.PropuestasAbiertas, err = countProyectoRows(`SELECT COUNT(*) FROM propuestas WHERE proyecto_id=? AND estado='abierta'`, proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.ReviewGatesAbiertas, err = countProyectoRows(`SELECT COUNT(*) FROM review_gates WHERE proyecto_id=? AND estado <> 'aprobado'`, proyecto.ID)
	if err != nil {
		return nil, err
	}
	cockpit.RuntimeMailboxPendiente, err = countProyectoRows(`SELECT COUNT(*) FROM runtime_mailbox WHERE proyecto_id=? AND estado='pendiente'`, proyecto.ID)
	if err != nil {
		return nil, err
	}
	for _, estado := range []string{"pendiente", "tomada", "ejecutando"} {
		n, err := countProyectoRows(`SELECT COUNT(*) FROM runtime_orders WHERE proyecto_id=? AND estado=?`, proyecto.ID, estado)
		if err != nil {
			return nil, err
		}
		cockpit.RuntimeOrdersAbiertas += n
	}

	return cockpit, nil
}

func countProyectoRows(query string, args ...any) (int, error) {
	if db.DB == nil {
		return 0, nil
	}
	var n int
	err := db.DB.QueryRow(query, args...).Scan(&n)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return n, err
}

func listarAgentesActivosProyecto(proyectoID int64) ([]apiProyectoCockpitAgente, error) {
	sesiones, err := db.ListarSesionesActivasOperativas()
	if err != nil {
		return nil, err
	}
	nombresProyecto := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
			continue
		}
		nombre := strings.TrimSpace(sesion.Agente)
		if nombre != "" {
			nombresProyecto[nombre] = struct{}{}
		}
	}
	if len(nombresProyecto) == 0 {
		return nil, nil
	}
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	items := make([]apiProyectoCockpitAgente, 0, len(status.AgentesActivos))
	for _, agente := range status.AgentesActivos {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre == "" || !containsProyectoAgente(nombresProyecto, nombre) {
			continue
		}
		items = append(items, apiProyectoCockpitAgente{
			Nombre: nombre,
			Rol:    strings.TrimSpace(agente.Rol),
		})
	}
	return items, nil
}

func containsProyectoAgente(nombres map[string]struct{}, nombre string) bool {
	_, ok := nombres[strings.TrimSpace(nombre)]
	return ok
}

func safeStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
