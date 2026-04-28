package db

import (
	"database/sql"

	"orquesta/storage"
)

func ListarEstadoLigeroReadOnly() ([]*Agente, map[string]int, []*Tarea, error) {
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	if !supportsPrepareLiteReadOnlyBackend(cfg.Driver) {
		agentes, err := ListarAgentesEstadoLigero()
		if err != nil {
			return nil, nil, nil, err
		}
		cuentas, err := ContarTareasPorEstado()
		if err != nil {
			return nil, nil, nil, err
		}
		tareas, err := listarTareasActivasReadOnlyFallback()
		if err != nil {
			return nil, nil, nil, err
		}
		return agentes, cuentas, tareas, nil
	}

	disabled := false
	skipPost := true
	cfg = applyOpenOptions(cfg, OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
		ReadOnly:           true,
	})
	cfg.MaxOpenConns = 1

	raw, err := backend.Open(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	defer raw.Close()

	agentes, err := listarAgentesRawOnDB(raw, normalizedDriverName(cfg.Driver))
	if err != nil {
		return nil, nil, nil, err
	}
	cuentas, err := contarTareasPorEstadoOnDB(raw, normalizedDriverName(cfg.Driver))
	if err != nil {
		return nil, nil, nil, err
	}
	tareas, err := listarTareasActivasOnDB(raw, normalizedDriverName(cfg.Driver))
	if err != nil {
		return nil, nil, nil, err
	}
	return agentes, cuentas, tareas, nil
}

func listarAgentesRawOnDB(raw *sql.DB, driver string) ([]*Agente, error) {
	rows, err := raw.Query(storage.RebindQuery(driver, `
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes ORDER BY nombre`))
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
}

func contarTareasPorEstadoOnDB(raw *sql.DB, driver string) (map[string]int, error) {
	rows, err := raw.Query(storage.RebindQuery(driver, `SELECT estado, COUNT(*) FROM tareas GROUP BY estado`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int)
	for rows.Next() {
		var estado string
		var n int
		if err := rows.Scan(&estado, &n); err != nil {
			return nil, err
		}
		m[estado] = n
	}
	return m, rows.Err()
}

func listarTareasActivasOnDB(raw *sql.DB, driver string) ([]*Tarea, error) {
	rows, err := raw.Query(storage.RebindQuery(driver, `
		SELECT id, titulo, proyecto_id, modulo, estado, agente, prioridad
		FROM tareas
		WHERE estado IN (?, ?, ?)
		ORDER BY id`), TareaAsignada, TareaEnProgreso, TareaBloqueada)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*Tarea, 0, 32)
	for rows.Next() {
		t := &Tarea{}
		var proyectoID sql.NullInt64
		var agente sql.NullString
		if err := rows.Scan(&t.ID, &t.Titulo, &proyectoID, &t.Modulo, &t.Estado, &agente, &t.Prioridad); err != nil {
			return nil, err
		}
		if proyectoID.Valid {
			t.ProyectoID = &proyectoID.Int64
		}
		if agente.Valid {
			t.Agente = &agente.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func listarTareasActivasReadOnlyFallback() ([]*Tarea, error) {
	items := make([]*Tarea, 0, 32)
	for _, estado := range []EstadoTarea{TareaAsignada, TareaEnProgreso, TareaBloqueada} {
		found, err := ListarTareas(FiltroTareas{Estado: &estado})
		if err != nil {
			return nil, err
		}
		items = append(items, found...)
	}
	return items, nil
}
