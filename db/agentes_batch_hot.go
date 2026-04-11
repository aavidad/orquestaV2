package db

import (
	"database/sql"
	"strings"
)

// ListarAgentesConSesionOperativa carga por lote los agentes pedidos, enriquece
// presupuesto una sola vez por fila y proyecta el estado visible usando la
// sesión ya observada por el caller. Evita el patrón N x GetAgente() -> N x
// GetSesionActivaOperativa() en los batches calientes.
func ListarAgentesConSesionOperativa(nombres []string, sesiones []*Sesion) (map[string]*Agente, error) {
	seen := make(map[string]string, len(nombres))
	args := make([]any, 0, len(nombres))
	for _, nombre := range nombres {
		nombre = strings.TrimSpace(nombre)
		if nombre == "" {
			continue
		}
		key := strings.ToLower(nombre)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = nombre
		args = append(args, nombre)
	}
	if len(args) == 0 {
		return map[string]*Agente{}, nil
	}

	query := `
		SELECT nombre, rol, activo, habilitado, COALESCE(estado_sesion,''), ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos, limite_dia_segundos,
		       limite_semanal_segundos, last_usage_reset_at, estado_cuota,
		       reanimar_at, motivo_pausa
		FROM agentes
		WHERE nombre IN (` + runtimeSQLPlaceholders(len(args)) + `)`

	list, err := consultarConReintentos(func() ([]*Agente, error) {
		rows, err := DB.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		out := make([]*Agente, 0, len(args))
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
			out = append(out, a)
		}
		return out, rows.Err()
	})
	if err != nil {
		return nil, err
	}

	enriquecerAgentesConPresupuesto(list)

	sesionByAgent := make(map[string]*Sesion, len(sesiones))
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(sesion.Agente))
		if key == "" {
			continue
		}
		if current, ok := sesionByAgent[key]; ok && current != nil && current.ID > sesion.ID {
			continue
		}
		sesionByAgent[key] = sesion
	}

	out := make(map[string]*Agente, len(list))
	for _, agente := range list {
		if agente == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(agente.Nombre))
		aplicarEstadoVisibleAgente(agente, sesionByAgent[key])
		out[key] = agente
	}
	return out, nil
}
