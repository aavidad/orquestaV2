package db

import (
	"database/sql"
	"strings"
)

func GetProyectoOperacionPrepareLite(proyectoID int64) (*ProyectoOperacion, error) {
	if proyectoID == 0 {
		return proyectoOperacionDefault(proyectoID), nil
	}
	if op, ok, err := getProyectoOperacionPrepareLiteReadOnly(proyectoID); ok {
		return op, err
	}
	return GetProyectoOperacion(proyectoID)
}

func getProyectoOperacionPrepareLiteReadOnly(proyectoID int64) (*ProyectoOperacion, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	var (
		op     ProyectoOperacion
		resume int
	)
	err = raw.QueryRow(`
		SELECT proyecto_id, estado_operativo, motivo, objetivo_pct, min_agentes, max_agentes,
		       prioridad, resume_automatico, created_at, updated_at
		FROM proyectos_operacion
		WHERE proyecto_id = ?`, proyectoID,
	).Scan(
		&op.ProyectoID, &op.EstadoOperativo, &op.Motivo, &op.ObjetivoPct, &op.MinAgentes,
		&op.MaxAgentes, &op.Prioridad, &resume, &op.CreatedAt, &op.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return proyectoOperacionDefault(proyectoID), true, nil
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return proyectoOperacionDefault(proyectoID), true, nil
		}
		return nil, true, err
	}
	op.ResumeAutomatico = resume == 1
	normalizarProyectoOperacion(&op)
	return &op, true, nil
}

func ListarTareasContextPrepareLite(agente string, proyectoID int64, limit int) ([]*Tarea, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID == 0 {
		return nil, nil
	}
	if tareas, ok, err := listarTareasContextPrepareLiteReadOnly(agente, proyectoID, limit); ok {
		return tareas, err
	}
	return ListarTareas(FiltroTareas{
		Agente:     strPtrRuntime(agente),
		ProyectoID: &proyectoID,
		Limit:      limit,
	})
}

func listarTareasContextPrepareLiteReadOnly(agente string, proyectoID int64, limit int) ([]*Tarea, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	if limit <= 0 {
		limit = 8
	}
	rows, err := raw.Query(`
		SELECT id, titulo, descripcion, proyecto_id, modulo, estado, agente, propuesta_id, prioridad,
		       dependencias, creado_por, commit_cierre, notas,
		       created_at, updated_at, completada_at, contrato_definido
		FROM tareas
		WHERE agente = ? AND proyecto_id = ? AND estado IN ('asignada','en_progreso','bloqueada')
		ORDER BY CASE WHEN estado='en_progreso' THEN 1 WHEN estado='asignada' THEN 2 ELSE 3 END,
		         CASE prioridad WHEN 'alta' THEN 1 WHEN 'media' THEN 2 ELSE 3 END,
		         id
		LIMIT ?`, agente, proyectoID, limit)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()
	var out []*Tarea
	for rows.Next() {
		tarea, err := escanearTarea(rows)
		if err != nil {
			return nil, true, err
		}
		out = append(out, tarea)
	}
	return out, true, rows.Err()
}

func ListarPropuestasAbiertasPrepareLite(proyectoID int64, limit int) ([]*Propuesta, error) {
	if proyectoID == 0 {
		return nil, nil
	}
	if propuestas, ok, err := listarPropuestasAbiertasPrepareLiteReadOnly(proyectoID, limit); ok {
		return propuestas, err
	}
	estado := PropuestaAbierta
	return ListarPropuestas(&estado, &proyectoID)
}

func listarPropuestasAbiertasPrepareLiteReadOnly(proyectoID int64, limit int) ([]*Propuesta, bool, error) {
	raw, ok, err := openPrepareLiteReadOnlyLocal()
	if !ok || err != nil {
		return nil, ok, err
	}
	defer raw.Close()
	if limit <= 0 {
		limit = 4
	}
	rows, err := raw.Query(`
		SELECT id, codigo, titulo, descripcion, proyecto_id, tipo, estado, propuesto_por, distribuidor,
		       created_at, updated_at, cerrada_at
		FROM propuestas
		WHERE estado = ? AND proyecto_id = ?
		ORDER BY id DESC
		LIMIT ?`, PropuestaAbierta, proyectoID, limit)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()
	var out []*Propuesta
	for rows.Next() {
		propuesta, err := escanearPropuesta(rows)
		if err != nil {
			return nil, true, err
		}
		out = append(out, propuesta)
	}
	return out, true, rows.Err()
}
