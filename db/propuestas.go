package db

import (
	"database/sql"
	"fmt"
	"strconv"
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
	ProyectoID   *int64
	Tipo         string
	Estado       EstadoPropuesta
	PropuestoPor string
	Distribuidor string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CerradaAt    *time.Time
	Votos        []*Voto // cargados aparte
}

func asegurarVotosPendientesPropuesta(propuestaID int64) (int, error) {
	rows, err := DB.Query(`SELECT nombre FROM agentes WHERE rol != 'admin' AND habilitado = 1`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var agentes []string
	for rows.Next() {
		var nombre string
		if err := rows.Scan(&nombre); err != nil {
			return 0, err
		}
		agentes = append(agentes, nombre)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	insertados := 0
	stmt := insertIgnoreValuesSQL("votos", []string{"propuesta_id", "agente", "posicion"}, []string{"propuesta_id", "agente"})
	for _, nombre := range agentes {
		res, err := DB.Exec(
			stmt,
			propuestaID, nombre, VotoPendiente,
		)
		if err != nil {
			return insertados, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			insertados += int(n)
		}
	}
	return insertados, nil
}

// CrearPropuesta inserta una nueva propuesta.
func CrearPropuesta(p *Propuesta) (int64, error) {
	// Auto-generar código si no se indica
	if p.Codigo == "" {
		var max int
		_ = DB.QueryRow(`SELECT COALESCE(MAX(CAST(SUBSTR(codigo,4) AS INTEGER)),29) FROM propuestas WHERE codigo LIKE 'OP-%'`).Scan(&max)
		p.Codigo = fmt.Sprintf("OP-%03d", max+1)
	}
	res, err := DB.Exec(`
		INSERT INTO propuestas (codigo, titulo, descripcion, proyecto_id, tipo, propuesto_por, distribuidor)
		VALUES (?,?,?,?,?,?,?)`,
		p.Codigo, p.Titulo, p.Descripcion, p.ProyectoID, p.Tipo, p.PropuestoPor, p.Distribuidor,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	if _, err := asegurarVotosPendientesPropuesta(id); err != nil {
		return id, err
	}

	Audit(p.PropuestoPor, "crear_propuesta", "propuesta", id, p.Codigo+": "+p.Titulo)

	CanalNotificaciones <- EventoNotificacion{
		Tipo:   "propuesta",
		ID:     id,
		Codigo: p.Codigo,
		Texto:  p.Titulo,
	}
	return id, nil
}

// GetPropuesta busca por ID o por código (OP-XXX).
func GetPropuesta(codigoOID string) (*Propuesta, error) {
	var row *sql.Row
	row = DB.QueryRow(`
		SELECT id, codigo, titulo, descripcion, proyecto_id, tipo, estado, propuesto_por, distribuidor,
		       created_at, updated_at, cerrada_at
		FROM propuestas WHERE codigo = ? OR CAST(id AS TEXT) = ?`, codigoOID, codigoOID)
	return escanearPropuesta(row)
}

// ListarPropuestas devuelve propuestas, opcionalmente filtradas por estado.
func ListarPropuestas(estado *EstadoPropuesta, proyectoID *int64) ([]*Propuesta, error) {
	q := `SELECT id, codigo, titulo, descripcion, proyecto_id, tipo, estado, propuesto_por, distribuidor,
		         created_at, updated_at, cerrada_at
		  FROM propuestas`
	args := []any{}
	var filtros []string
	if estado != nil {
		filtros = append(filtros, "estado = ?")
		args = append(args, *estado)
	}
	if proyectoID != nil {
		filtros = append(filtros, "proyecto_id = ?")
		args = append(args, *proyectoID)
	}
	if len(filtros) > 0 {
		q += " WHERE " + strings.Join(filtros, " AND ")
	}
	q += " ORDER BY id DESC"

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
	if nuevoEstado == string(PropuestaConsenso) {
		propuesta, err := GetPropuesta(codigo)
		if err != nil {
			return err
		}
		ok, detalle, err := puedeCerrarEnConsenso(propuesta.ID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("la propuesta '%s' no cumple la política de consenso: %s", codigo, detalle)
		}
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

func ReabrirPropuesta(codigo, agente string) (int, error) {
	propuesta, err := GetPropuesta(codigo)
	if err != nil {
		return 0, err
	}

	res, err := DB.Exec(`
		UPDATE propuestas SET estado=?, cerrada_at=NULL WHERE id=?`,
		PropuestaAbierta, propuesta.ID,
	)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, fmt.Errorf("propuesta '%s' no encontrada", codigo)
	}

	insertados, err := asegurarVotosPendientesPropuesta(propuesta.ID)
	if err != nil {
		return insertados, err
	}
	Audit(agente, "reabrir_propuesta", "propuesta", propuesta.ID, propuesta.Codigo)
	return insertados, nil
}

func RepararVotosPendientesPropuesta(codigo, agente string) (int, error) {
	propuesta, err := GetPropuesta(codigo)
	if err != nil {
		return 0, err
	}
	insertados, err := asegurarVotosPendientesPropuesta(propuesta.ID)
	if err != nil {
		return insertados, err
	}
	Audit(agente, "reparar_votos_propuesta", "propuesta", propuesta.ID, propuesta.Codigo)
	return insertados, nil
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
	ok, _, err := puedeCerrarEnConsenso(propuestaID)
	if err != nil {
		return false, err
	}
	if ok {
		var codigo string
		_ = DB.QueryRow(`SELECT codigo FROM propuestas WHERE id=?`, propuestaID).Scan(&codigo)
		_ = CerrarPropuesta(codigo, "consenso", agente)
		return true, nil
	}
	return false, nil
}

func puedeCerrarEnConsenso(propuestaID int64) (bool, string, error) {
	minVotes := configIntOrDefault("propuesta_min_votes", 2)
	minNonAuthorVotes := configIntOrDefault("propuesta_min_non_author_votes", 2)

	var autor string
	if err := DB.QueryRow(`SELECT propuesto_por FROM propuestas WHERE id = ?`, propuestaID).Scan(&autor); err != nil {
		return false, "", err
	}

	rows, err := DB.Query(`
		SELECT v.agente, v.posicion
		FROM votos v
		JOIN agentes a ON a.nombre = v.agente
		WHERE v.propuesta_id = ? AND a.habilitado = 1 AND a.rol != 'admin'`,
		propuestaID,
	)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()

	total, acuerdo, pendiente, desacuerdo, nonAuthorAgreement := 0, 0, 0, 0, 0
	for rows.Next() {
		var agente, pos string
		if err := rows.Scan(&agente, &pos); err != nil {
			return false, "", err
		}
		total++
		switch pos {
		case "acuerdo":
			acuerdo++
			if !strings.EqualFold(strings.TrimSpace(agente), strings.TrimSpace(autor)) {
				nonAuthorAgreement++
			}
		case "pendiente":
			pendiente++
		case "desacuerdo":
			desacuerdo++
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", err
	}

	if desacuerdo > 0 {
		return false, fmt.Sprintf("hay %d voto(s) en desacuerdo", desacuerdo), nil
	}
	if total == 0 {
		return false, "no hay votantes habilitados", nil
	}
	if acuerdo < minVotes {
		return false, fmt.Sprintf("solo hay %d voto(s) de acuerdo y el mínimo es %d", acuerdo, minVotes), nil
	}
	if nonAuthorAgreement < minNonAuthorVotes {
		return false, fmt.Sprintf("solo hay %d voto(s) de acuerdo de agentes no autores y el mínimo es %d", nonAuthorAgreement, minNonAuthorVotes), nil
	}
	if pendiente > 0 {
		return true, fmt.Sprintf("hay %d voto(s) pendiente(s), pero ya se cumple el mínimo válido de consenso", pendiente), nil
	}
	return true, "", nil
}

func configIntOrDefault(key string, defaultValue int) int {
	raw, err := ConfigGet(key)
	if err != nil {
		return defaultValue
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		return defaultValue
	}
	return n
}

// PropuestasPendientesVoto devuelve las propuestas donde el agente tiene voto pendiente.
func PropuestasPendientesVoto(agente string) ([]*Propuesta, error) {
	return PropuestasPendientesVotoProyecto(agente, nil)
}

func PropuestasPendientesVotoProyecto(agente string, proyectoID *int64) ([]*Propuesta, error) {
	q := `
		SELECT p.id, p.codigo, p.titulo, p.descripcion, p.proyecto_id, p.tipo, p.estado, p.propuesto_por,
		       p.distribuidor, p.created_at, p.updated_at, p.cerrada_at
		FROM propuestas p
		JOIN votos v ON v.propuesta_id = p.id
		WHERE v.agente = ? AND v.posicion = 'pendiente' AND p.estado = 'abierta'
	`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND p.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY p.id`

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

// ─── helpers ────────────────────────────────────────────────────────────────

func escanearPropuesta(s scanner) (*Propuesta, error) {
	var p Propuesta
	var proyectoID sql.NullInt64
	var cerradaAt sql.NullTime
	err := s.Scan(
		&p.ID, &p.Codigo, &p.Titulo, &p.Descripcion, &proyectoID, &p.Tipo, &p.Estado,
		&p.PropuestoPor, &p.Distribuidor,
		&p.CreatedAt, &p.UpdatedAt, &cerradaAt,
	)
	if err != nil {
		return nil, err
	}
	if cerradaAt.Valid {
		p.CerradaAt = &cerradaAt.Time
	}
	if proyectoID.Valid {
		p.ProyectoID = &proyectoID.Int64
	}
	return &p, nil
}
