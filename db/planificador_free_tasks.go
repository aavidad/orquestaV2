package db

import (
	"sort"
	"strings"

	"orquesta/planificadorpolicy"
)

func buscarSiguienteTareaLibre(proyectoID int64) (*Tarea, error) {
	q := FiltroTareas{
		ProyectoID: &proyectoID,
		Libre:      true,
		Limit:      1,
	}
	list, err := ListarTareas(q)
	if err != nil || len(list) == 0 {
		return nil, err
	}
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
	preferirMicroprogramacion, err := agentePrefiereTrabajoMicroprogramacion(strings.TrimSpace(agente), proyectoID)
	if err != nil {
		return nil, err
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
		return mejorTareaLibreParaAgente(candidatas[i], candidatas[j], moduloPreferido, modulosOcupados, preferirMicroprogramacion)
	})
	return candidatas[0], nil
}

func mejorTareaLibreParaAgente(a, b *Tarea, moduloPreferido string, modulosOcupados map[string]bool, preferirMicroprogramacion bool) bool {
	return planificadorpolicy.PreferFreeTaskCandidate(
		freeTaskCandidateSnapshot(a),
		freeTaskCandidateSnapshot(b),
		moduloPreferido,
		modulosOcupados,
		preferirMicroprogramacion,
	)
}

func scoreTareaLibreParaAgente(t *Tarea, moduloPreferido string, modulosOcupados map[string]bool, preferirMicroprogramacion bool) int {
	return planificadorpolicy.ScoreFreeTaskCandidate(
		freeTaskCandidateSnapshot(t),
		moduloPreferido,
		modulosOcupados,
		preferirMicroprogramacion,
	)
}

func freeTaskCandidateSnapshot(task *Tarea) *planificadorpolicy.FreeTaskCandidateSnapshot {
	if task == nil {
		return nil
	}
	snapshot := &planificadorpolicy.FreeTaskCandidateSnapshot{
		ID:              task.ID,
		Module:          task.Modulo,
		Priority:        string(task.Prioridad),
		ContractDefined: task.ContratoDefinido,
	}
	if hasSpec, err := tareaTieneEspecificacionActiva(task.ID); err == nil {
		snapshot.HasActiveSpec = hasSpec
	}
	return snapshot
}

func agentePrefiereTrabajoMicroprogramacion(agente string, proyectoID int64) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, nil
	}
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		return false, err
	}
	poolSlug, err := resolverPoolLocalCompartidoAgenteProyecto(agente, strings.TrimSpace(proyecto.Slug))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(poolSlug) != "", nil
}

func tareaTieneEspecificacionActiva(tareaID int64) (bool, error) {
	if tareaID <= 0 {
		return false, nil
	}
	var total int
	if err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM especificaciones_funcion
		WHERE tarea_id = ?
		  AND estado = 'activa'`, tareaID).Scan(&total); err != nil {
		return false, err
	}
	return total > 0, nil
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
