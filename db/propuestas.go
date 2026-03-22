package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type EstadoPropuesta string

const (
	PropuestaAbierta   EstadoPropuesta = "abierta"
	PropuestaConsenso  EstadoPropuesta = "consenso"
	PropuestaRechazada EstadoPropuesta = "rechazada"
	PropuestaBacklog   EstadoPropuesta = "backlog"
)

type Propuesta struct {
	ID           int64
	Codigo       string
	Titulo       string
	Descripcion  string
	Tipo         string
	Estado       EstadoPropuesta
	PropuestoPor string
	Distribuidor string
	ProyectoID   *int64
	ProyectoSlug string
	Proyecto     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CerradaAt    *time.Time
	Votos        []*Voto // cargados aparte
}

// CrearPropuesta inserta una nueva propuesta.
func CrearPropuesta(p *Propuesta) (int64, error) {
	if p.ProyectoID == nil && strings.TrimSpace(p.ProyectoSlug) != "" {
		proyecto, err := EnsureProyectoRef(strings.TrimSpace(p.ProyectoSlug))
		if err != nil {
			return 0, err
		}
		p.ProyectoID = &proyecto.ID
		p.ProyectoSlug = proyecto.Slug
		p.Proyecto = proyecto.Nombre
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Auto-generar código si no se indica
	if p.Codigo == "" {
		var max int
		_ = tx.QueryRow(`SELECT COALESCE(MAX(CAST(SUBSTR(codigo,4) AS INTEGER)),29) FROM propuestas WHERE codigo LIKE 'OP-%'`).Scan(&max)
		p.Codigo = fmt.Sprintf("OP-%03d", max+1)
	}
	res, err := tx.Exec(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, propuesto_por, distribuidor, proyecto_id)
		VALUES (?,?,?,?,?,?,?)`,
		p.Codigo, p.Titulo, p.Descripcion, p.Tipo, p.PropuestoPor, p.Distribuidor, p.ProyectoID,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	// Crear filas de voto pendiente solo para agentes habilitados (no admin, no retirados)
	rows, _ := tx.Query(`SELECT nombre FROM agentes WHERE rol != 'admin' AND habilitado = 1`)
	if rows != nil {
		var votantes []string
		for rows.Next() {
			var nombre string
			_ = rows.Scan(&nombre)
			votantes = append(votantes, nombre)
		}
		rows.Close()
		for _, nombre := range votantes {
			_, _ = tx.Exec(
				`INSERT OR IGNORE INTO votos (propuesta_id, agente, posicion) VALUES (?,?,'pendiente')`,
				id, nombre,
			)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	Audit(p.PropuestoPor, "crear_propuesta", "propuesta", id, p.Codigo+": "+p.Titulo)
	return id, nil
}

// GetPropuesta busca por ID o por código (OP-XXX).
func GetPropuesta(codigoOID string) (*Propuesta, error) {
	var row *sql.Row
	row = DB.QueryRow(`
		SELECT p.id, p.codigo, p.titulo, p.descripcion, p.tipo, p.estado, p.propuesto_por, p.distribuidor,
		       p.proyecto_id, COALESCE(pr.slug,''), COALESCE(pr.nombre,''), p.created_at, p.updated_at, p.cerrada_at
		FROM propuestas p
		LEFT JOIN proyectos pr ON pr.id = p.proyecto_id
		WHERE p.codigo = ? OR CAST(p.id AS TEXT) = ?`, codigoOID, codigoOID)
	return escanearPropuesta(row)
}

// ListarPropuestas devuelve propuestas, opcionalmente filtradas por estado.
func ListarPropuestas(estado *EstadoPropuesta) ([]*Propuesta, error) {
	q := `SELECT p.id, p.codigo, p.titulo, p.descripcion, p.tipo, p.estado, p.propuesto_por, p.distribuidor,
		         p.proyecto_id, COALESCE(pr.slug,''), COALESCE(pr.nombre,''), p.created_at, p.updated_at, p.cerrada_at
		  FROM propuestas p
		  LEFT JOIN proyectos pr ON pr.id = p.proyecto_id`
	args := []any{}
	if estado != nil {
		q += " WHERE estado = ?"
		args = append(args, *estado)
	}
	q += " ORDER BY p.id DESC"

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Propuesta
	for rows.Next() {
		p, err := escanearPropuesta(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func ListarPropuestasProyecto(selector string, estado *EstadoPropuesta) ([]*Propuesta, error) {
	proyecto, err := GetProyectoRef(selector)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*Propuesta{}, nil
		}
		return nil, err
	}

	q := `SELECT p.id, p.codigo, p.titulo, p.descripcion, p.tipo, p.estado, p.propuesto_por, p.distribuidor,
		         p.proyecto_id, COALESCE(pr.slug,''), COALESCE(pr.nombre,''), p.created_at, p.updated_at, p.cerrada_at
		  FROM propuestas p
		  LEFT JOIN proyectos pr ON pr.id = p.proyecto_id
		  WHERE p.proyecto_id = ?`
	args := []any{proyecto.ID}
	if estado != nil {
		q += ` AND p.estado = ?`
		args = append(args, *estado)
	}
	q += ` ORDER BY p.id DESC`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Propuesta
	for rows.Next() {
		p, err := escanearPropuesta(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// CerrarPropuesta actualiza el estado y la fecha de cierre.
func CerrarPropuesta(codigo, nuevoEstado, agente string) error {
	if nuevoEstado != string(PropuestaConsenso) &&
		nuevoEstado != string(PropuestaRechazada) &&
		nuevoEstado != string(PropuestaBacklog) {
		return fmt.Errorf("estado '%s' no válido para cierre", nuevoEstado)
	}
	res, err := DB.Exec(`
		UPDATE propuestas SET estado=?, cerrada_at=CURRENT_TIMESTAMP WHERE codigo=?`,
		nuevoEstado, codigo,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("propuesta '%s' no encontrada", codigo)
	}
	Audit(agente, "cerrar_propuesta", "propuesta", 0, codigo+"→"+nuevoEstado)
	return nil
}

// ReabrirPropuesta vuelve a abrir una propuesta cerrada y reinicia sus votos a pendiente.
func ReabrirPropuesta(codigo, agente string) (int, error) {
	p, err := GetPropuesta(codigo)
	if err != nil {
		return 0, fmt.Errorf("propuesta '%s' no encontrada", codigo)
	}
	if p.Estado == PropuestaAbierta {
		return 0, fmt.Errorf("la propuesta %s ya está abierta", codigo)
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE propuestas
		SET estado='abierta', cerrada_at=NULL
		WHERE id=?`,
		p.ID,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, fmt.Errorf("propuesta '%s' no encontrada", codigo)
	}

	reparados, err := repararVotosPendientesTx(tx, p.ID, true)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(agente, "reabrir_propuesta", "propuesta", p.ID, fmt.Sprintf("%s→abierta votos=%d", codigo, reparados))
	return reparados, nil
}

// EvaluarConsenso revisa los votos de una propuesta.
// Solo cuentan los agentes habilitados (habilitado=1) con rol de votante (programador/documentador).
// Si un agente se retira, sus votos pendientes ya fueron eliminados por RetirarAgente.
//
// Reglas:
//   - Cualquier voto desacuerdo → NO hay consenso automático (Alberto decide).
//   - Si todos los votantes habilitados han votado acuerdo → consenso automático.
//
// Devuelve true si se alcanzó consenso y se cerró la propuesta.
func EvaluarConsenso(propuestaID int64, agente string) (bool, error) {
	// Solo votos de agentes habilitados
	rows, err := DB.Query(`
		SELECT v.posicion
		FROM votos v
		JOIN agentes a ON a.nombre = v.agente
		WHERE v.propuesta_id = ? AND a.habilitado = 1 AND a.rol != 'admin'`,
		propuestaID,
	)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	total, acuerdo, pendiente, desacuerdo := 0, 0, 0, 0
	for rows.Next() {
		var pos string
		_ = rows.Scan(&pos)
		total++
		switch pos {
		case "acuerdo":
			acuerdo++
		case "pendiente":
			pendiente++
		case "desacuerdo":
			desacuerdo++
		}
	}

	// Cualquier desacuerdo bloquea → Alberto arbitra
	if desacuerdo > 0 {
		return false, nil
	}
	// Consenso unánime: todos los votantes habilitados han dicho acuerdo
	if total > 0 && pendiente == 0 && acuerdo == total {
		var codigo string
		_ = DB.QueryRow(`SELECT codigo FROM propuestas WHERE id=?`, propuestaID).Scan(&codigo)
		_ = CerrarPropuesta(codigo, "consenso", agente)
		return true, nil
	}
	return false, nil
}

// PropuestasPendientesVoto devuelve las propuestas donde el agente tiene voto pendiente.
func PropuestasPendientesVoto(agente string) ([]*Propuesta, error) {
	rows, err := DB.Query(`
		SELECT p.id, p.codigo, p.titulo, p.descripcion, p.tipo, p.estado, p.propuesto_por,
		       p.distribuidor, p.proyecto_id, COALESCE(pr.slug,''), COALESCE(pr.nombre,''), p.created_at, p.updated_at, p.cerrada_at
		FROM propuestas p
		JOIN votos v ON v.propuesta_id = p.id
		LEFT JOIN proyectos pr ON pr.id = p.proyecto_id
		WHERE v.agente = ? AND v.posicion = 'pendiente' AND p.estado = 'abierta'
		ORDER BY p.id`, agente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Propuesta
	for rows.Next() {
		p, err := escanearPropuesta(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func VotantesHabilitados() ([]string, error) {
	rows, err := DB.Query(`SELECT nombre FROM agentes WHERE rol != 'admin' AND habilitado = 1 ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return nil, err
		}
		list = append(list, nombre)
	}
	return list, rows.Err()
}

// ─── helpers ────────────────────────────────────────────────────────────────

func escanearPropuesta(s scanner) (*Propuesta, error) {
	var p Propuesta
	var proyectoID sql.NullInt64
	var cerradaAt sql.NullTime
	err := s.Scan(
		&p.ID, &p.Codigo, &p.Titulo, &p.Descripcion, &p.Tipo, &p.Estado,
		&p.PropuestoPor, &p.Distribuidor, &proyectoID, &p.ProyectoSlug, &p.Proyecto,
		&p.CreatedAt, &p.UpdatedAt, &cerradaAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		p.ProyectoID = &proyectoID.Int64
	}
	if cerradaAt.Valid {
		p.CerradaAt = &cerradaAt.Time
	}
	return &p, nil
}
