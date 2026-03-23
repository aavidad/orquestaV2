package db

import (
	"database/sql"
	"fmt"
)

// PlanificarTareasAutomaticamente es el motor de autonomía de Orquesta.
// Busca agentes disponibles y les asigna tareas libres de sus proyectos activos,
// además de liberar tareas del backlog cuyas dependencias ya se han cumplido.
func PlanificarTareasAutomaticamente() error {
	// 1. Liberar tareas del backlog que ya no tienen dependencias pendientes
	if err := liberarBacklog(); err != nil {
		return err
	}

	// 2. Buscar agentes disponibles (habilitados y en estado 'disponible')
	agentes, err := ListarAgentesDisponibles()
	if err != nil {
		return err
	}

	for _, ag := range agentes {
		// Buscar el proyecto asignado a este agente
		proyectoID, err := ObtenerProyectoActivoAgente(ag.Nombre)
		if err != nil || proyectoID == 0 {
			continue // El agente no tiene un proyecto asignado ahora mismo
		}

		// Buscar la siguiente tarea libre para ese proyecto
		tarea, err := buscarSiguienteTareaLibre(proyectoID)
		if err != nil || tarea == nil {
			continue // No hay tareas listas para este proyecto
		}

		// Asignar automáticamente
		if err := TomarTarea(tarea.ID, ag.Nombre); err == nil {
			Audit("sistema", "auto_asignacion", "tarea", tarea.ID, 
				fmt.Sprintf("Asignada automáticamente a %s (Autonomía Total)", ag.Nombre))
			fmt.Printf("✓ [Planificador] Tarea #%d ('%s') asignada automáticamente a %s\n", 
				tarea.ID, tarea.Titulo, ag.Nombre)
		}
	}

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

func ListarAgentesDisponibles() ([]*Agente, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol 
		FROM agentes 
		WHERE habilitado = 1 AND estado_sesion = 'disponible'`)
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
