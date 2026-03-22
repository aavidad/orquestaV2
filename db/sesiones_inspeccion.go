package db

import (
	"fmt"
	"strings"
)

// FiltroSesionesInspeccion permite consultar sesiones sin tocar rutas de escritura.
type FiltroSesionesInspeccion struct {
	Agente     *string
	ProyectoID *int64
	Activa     *bool
	Estado     *string
}

// ListarSesionesInspeccion devuelve sesiones con filtros por agente, proyecto, activa y estado.
func ListarSesionesInspeccion(f FiltroSesionesInspeccion) ([]*Sesion, error) {
	q := `
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE 1=1`
	var args []any

	if f.Agente != nil && strings.TrimSpace(*f.Agente) != "" {
		q += ` AND s.agente = ?`
		args = append(args, strings.TrimSpace(*f.Agente))
	}
	if f.ProyectoID != nil {
		q += ` AND s.proyecto_id = ?`
		args = append(args, *f.ProyectoID)
	}
	if f.Activa != nil {
		q += ` AND s.activa = ?`
		args = append(args, *f.Activa)
	}
	if f.Estado != nil && strings.TrimSpace(*f.Estado) != "" {
		q += ` AND s.estado = ?`
		args = append(args, strings.TrimSpace(*f.Estado))
	}

	q += ` ORDER BY s.id DESC`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Sesion
	for rows.Next() {
		sesion, err := escanearSesion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sesion)
	}
	return out, rows.Err()
}

// GetSesionInspeccionByID devuelve una sesión concreta para inspección read-only.
func GetSesionInspeccionByID(id int64) (*Sesion, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id de sesión inválido")
	}
	row := DB.QueryRow(`
		SELECT s.id, s.agente, s.conector_id, COALESCE(c.slug,''), COALESCE(c.nombre,''),
		       s.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       s.inicio, s.fin, s.activa, s.estado, s.cwd, s.herramienta,
		       s.external_session_id, s.resume_payload_json, s.resumen_continuidad,
		       s.branch, s.heartbeat_at, s.host, s.pid
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		LEFT JOIN proyectos p ON p.id = s.proyecto_id
		WHERE s.id = ?`, id)
	return escanearSesion(row)
}

func listarSesionesPorEstadoActivo(estado string, activa bool) ([]*Sesion, error) {
	return ListarSesionesInspeccion(FiltroSesionesInspeccion{
		Estado: &estado,
		Activa: &activa,
	})
}
