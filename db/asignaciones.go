package db

import (
	"database/sql"
	"time"
)

type AsignacionTransitionCoordinator interface {
	AfterActivarAsignacion(agente string, proyectoID int64, nota string) error
	AfterPausarAsignacion(agente string, proyectoID int64, nota string) error
}

type defaultAsignacionTransitionCoordinator struct{}

var defaultAsignacionTransitioner AsignacionTransitionCoordinator = defaultAsignacionTransitionCoordinator{}

func SetAsignacionTransitionCoordinator(next AsignacionTransitionCoordinator) {
	if next == nil {
		next = defaultAsignacionTransitionCoordinator{}
	}
	defaultAsignacionTransitioner = next
}

func (defaultAsignacionTransitionCoordinator) AfterActivarAsignacion(agente string, proyectoID int64, nota string) error {
	_ = agente
	_ = proyectoID
	_ = nota
	return nil
}

func (defaultAsignacionTransitionCoordinator) AfterPausarAsignacion(agente string, proyectoID int64, nota string) error {
	_ = agente
	_ = proyectoID
	_ = nota
	return nil
}

type EstadoAsignacion string

const (
	AsignacionPlanificada EstadoAsignacion = "planificada"
	AsignacionActiva      EstadoAsignacion = "activa"
	AsignacionPausada     EstadoAsignacion = "pausada"
	AsignacionCerrada     EstadoAsignacion = "cerrada"
)

type Asignacion struct {
	ID             int64
	Agente         string
	ProyectoID     int64
	ProyectoSlug   string
	ProyectoNombre string
	Estado         EstadoAsignacion
	Nota           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CerradaAt      *time.Time
}

type FiltroAsignaciones struct {
	Agente     *string
	ProyectoID *int64
	Estado     *EstadoAsignacion
}

func ActivarAsignacion(agente string, proyectoID int64, nota string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = activarAsignacionTx(tx, agente, proyectoID, nota); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	if err = defaultAsignacionTransitioner.AfterActivarAsignacion(agente, proyectoID, nota); err != nil {
		return err
	}
	return nil
}

func PausarAsignacion(agente string, proyectoID int64, nota string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	changed, err := pausarAsignacionTx(tx, agente, proyectoID, nota)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if err = defaultAsignacionTransitioner.AfterPausarAsignacion(agente, proyectoID, nota); err != nil {
		return err
	}
	return nil
}

func activarAsignacionTx(tx *Tx, agente string, proyectoID int64, nota string) error {
	if _, err := tx.Exec(`
		UPDATE asignaciones
		SET estado='cerrada', cerrada_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('planificada','activa') AND proyecto_id <> ?`,
		agente, proyectoID,
	); err != nil {
		return err
	}

	var existenteID int64
	err := tx.QueryRow(`
		SELECT id FROM asignaciones
		WHERE agente=? AND proyecto_id=? AND estado IN ('planificada','activa','pausada')
		ORDER BY id DESC LIMIT 1`,
		agente, proyectoID,
	).Scan(&existenteID)
	switch err {
	case nil:
		_, err = tx.Exec(`UPDATE asignaciones SET estado='activa', nota=?, cerrada_at=NULL WHERE id=?`, nota, existenteID)
	case sql.ErrNoRows:
		_, err = tx.Exec(`
			INSERT INTO asignaciones (agente, proyecto_id, estado, nota)
			VALUES (?,?,'activa',?)`,
			agente, proyectoID, nota,
		)
	default:
		return err
	}
	return err
}

func pausarAsignacionTx(tx *Tx, agente string, proyectoID int64, nota string) (bool, error) {
	res, err := tx.Exec(`
		UPDATE asignaciones
		SET estado='pausada', nota=?, cerrada_at=NULL
		WHERE agente=? AND proyecto_id=? AND estado='activa'`,
		nota, agente, proyectoID,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func ListarAsignaciones(f FiltroAsignaciones) ([]*Asignacion, error) {
	q := `
		SELECT a.id, a.agente, a.proyecto_id, p.slug, p.nombre, a.estado, a.nota, a.created_at, a.updated_at, a.cerrada_at
		FROM asignaciones a
		JOIN proyectos p ON p.id = a.proyecto_id
		WHERE 1=1`
	var args []any
	if f.Agente != nil {
		q += ` AND a.agente = ?`
		args = append(args, *f.Agente)
	}
	if f.ProyectoID != nil {
		q += ` AND a.proyecto_id = ?`
		args = append(args, *f.ProyectoID)
	}
	if f.Estado != nil {
		q += ` AND a.estado = ?`
		args = append(args, *f.Estado)
	}
	switch {
	case f.Agente != nil:
		q += ` ORDER BY
			CASE a.estado
				WHEN 'activa' THEN 0
				WHEN 'planificada' THEN 1
				WHEN 'pausada' THEN 2
				ELSE 9
			END,
			a.updated_at DESC,
			a.id DESC`
	case f.ProyectoID != nil:
		q += ` ORDER BY a.agente, a.id DESC`
	default:
		q += ` ORDER BY p.slug, a.agente, a.id DESC`
	}

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Asignacion
	for rows.Next() {
		a, err := escanearAsignacion(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func GetAsignacionActivaAgente(agente string) (*Asignacion, error) {
	estado := AsignacionActiva
	list, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agente, Estado: &estado})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, sql.ErrNoRows
	}
	return list[0], nil
}

func ContarAsignacionesActivasPorProyecto() (map[int64]int, error) {
	return consultarConReintentos(func() (map[int64]int, error) {
		rows, err := DB.Query(`
			SELECT proyecto_id, COUNT(*)
			FROM asignaciones
			WHERE estado = 'activa'
			GROUP BY proyecto_id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := make(map[int64]int)
		for rows.Next() {
			var proyectoID int64
			var n int
			if err := rows.Scan(&proyectoID, &n); err != nil {
				return nil, err
			}
			out[proyectoID] = n
		}
		return out, rows.Err()
	})
}

func escanearAsignacion(s scanner) (*Asignacion, error) {
	var a Asignacion
	var cerrada sql.NullTime
	err := s.Scan(
		&a.ID, &a.Agente, &a.ProyectoID, &a.ProyectoSlug, &a.ProyectoNombre,
		&a.Estado, &a.Nota, &a.CreatedAt, &a.UpdatedAt, &cerrada,
	)
	if err != nil {
		return nil, err
	}
	if cerrada.Valid {
		a.CerradaAt = &cerrada.Time
	}
	return &a, nil
}
