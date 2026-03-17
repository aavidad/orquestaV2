package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type EstadoTarea string

const (
	// Constantes principales
	TareaLibre      EstadoTarea = "libre"
	TareaAsignada   EstadoTarea = "asignada"
	TareaEnProgreso EstadoTarea = "en_progreso"
	TareaCompletada EstadoTarea = "completada"
	TareaBloqueada  EstadoTarea = "bloqueada"
	TareaCancelada  EstadoTarea = "cancelada"
	TareaBacklog    EstadoTarea = "backlog" // pendiente para el futuro

	// Aliases con prefijo Estado (usados en cmd/)
	EstadoLibre      = TareaLibre
	EstadoAsignada   = TareaAsignada
	EstadoEnProgreso = TareaEnProgreso
	EstadoCompletada = TareaCompletada
	EstadoBloqueada  = TareaBloqueada
	EstadoCancelada  = TareaCancelada
	EstadoBacklog    = TareaBacklog
)

type PrioridadTarea string

const (
	PrioridadAlta  PrioridadTarea = "alta"
	PrioridadMedia PrioridadTarea = "media"
	PrioridadBaja  PrioridadTarea = "baja"
)

type Tarea struct {
	ID           int64
	Titulo       string
	Descripcion  string
	Modulo       string
	Estado       EstadoTarea
	Agente       *string // nil = sin asignar
	PropuestaID  *int64  // nil = no vinculada a propuesta
	Prioridad    PrioridadTarea
	Dependencias []int64
	CreadoPor    string
	CommitCierre string
	Notas        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CompletadaAt *time.Time
}

type FiltroTareas struct {
	Estado      *EstadoTarea
	Agente      *string
	Modulo      *string
	PropuestaID *int64
	Libre       bool // solo las libre (sin agente)
}

// CrearTarea inserta una nueva tarea y devuelve su ID.
func CrearTarea(t *Tarea) (int64, error) {
	deps, _ := json.Marshal(t.Dependencias)
	res, err := DB.Exec(`
		INSERT INTO tareas (titulo, descripcion, modulo, prioridad, dependencias, creado_por, notas, propuesta_id)
		VALUES (?,?,?,?,?,?,?,?)`,
		t.Titulo, t.Descripcion, t.Modulo, t.Prioridad, string(deps), t.CreadoPor, t.Notas, t.PropuestaID,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit(t.CreadoPor, "crear_tarea", "tarea", id, t.Titulo)
	return id, nil
}

// GetTarea devuelve una tarea por ID.
func GetTarea(id int64) (*Tarea, error) {
	row := DB.QueryRow(`
		SELECT id, titulo, descripcion, modulo, estado, agente, propuesta_id, prioridad,
		       dependencias, creado_por, commit_cierre, notas,
		       created_at, updated_at, completada_at
		FROM tareas WHERE id = ?`, id)
	return escanearTarea(row)
}

// ListarTareas devuelve tareas según filtros opcionales.
func ListarTareas(f FiltroTareas) ([]*Tarea, error) {
	q := `SELECT id, titulo, descripcion, modulo, estado, agente, propuesta_id, prioridad,
		         dependencias, creado_por, commit_cierre, notas,
		         created_at, updated_at, completada_at
		  FROM tareas WHERE 1=1`
	args := []any{}

	if f.Libre {
		q += " AND estado = 'libre'"
	} else if f.Estado != nil {
		q += " AND estado = ?"
		args = append(args, *f.Estado)
	}
	if f.Agente != nil {
		q += " AND agente = ?"
		args = append(args, *f.Agente)
	}
	if f.Modulo != nil {
		q += " AND modulo = ?"
		args = append(args, *f.Modulo)
	}
	if f.PropuestaID != nil {
		q += " AND propuesta_id = ?"
		args = append(args, *f.PropuestaID)
	}
	q += " ORDER BY CASE prioridad WHEN 'alta' THEN 1 WHEN 'media' THEN 2 ELSE 3 END, id"

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Tarea
	for rows.Next() {
		t, err := escanearTarea(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// TomarTarea asigna una tarea libre (o backlog) a un agente.
func TomarTarea(id int64, agente string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	if t.Estado != TareaLibre && t.Estado != TareaBacklog {
		return fmt.Errorf("la tarea #%d está en estado '%s', solo se pueden tomar tareas 'libre' o 'backlog'", id, t.Estado)
	}
	_, err = DB.Exec(
		`UPDATE tareas SET estado='asignada', agente=? WHERE id=?`,
		agente, id,
	)
	if err == nil {
		Audit(agente, "tomar_tarea", "tarea", id, t.Titulo)
	}
	return err
}

// IniciarTarea marca una tarea como en_progreso.
func IniciarTarea(id int64, agente string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	if t.Estado != TareaAsignada {
		return fmt.Errorf("la tarea #%d está en estado '%s', debe estar 'asignada' para iniciarla", id, t.Estado)
	}
	_, err = DB.Exec(
		`UPDATE tareas SET estado='en_progreso', agente=? WHERE id=?`,
		agente, id,
	)
	if err == nil {
		Audit(agente, "iniciar_tarea", "tarea", id, t.Titulo)
	}
	return err
}

// CompletarTarea marca una tarea como completada.
func CompletarTarea(id int64, agente, commit string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	if t.Estado != TareaEnProgreso && t.Estado != TareaAsignada {
		return fmt.Errorf("la tarea #%d está en estado '%s', debe estar en progreso o asignada", id, t.Estado)
	}
	_, err = DB.Exec(
		`UPDATE tareas SET estado='completada', commit_cierre=?, completada_at=CURRENT_TIMESTAMP WHERE id=?`,
		commit, id,
	)
	if err == nil {
		Audit(agente, "completar_tarea", "tarea", id, t.Titulo+" commit="+commit)
	}
	return err
}

// BloquearTarea marca una tarea como bloqueada y registra el bloqueo.
func BloquearTarea(id int64, agente, motivo string) error {
	t, err := GetTarea(id)
	if err != nil {
		return fmt.Errorf("tarea #%d no encontrada", id)
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`UPDATE tareas SET estado='bloqueada' WHERE id=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`INSERT INTO bloqueos (tarea_id, motivo, bloqueado_por) VALUES (?,?,?)`,
		id, motivo, agente,
	); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	Audit(agente, "bloquear_tarea", "tarea", id, t.Titulo+": "+motivo)
	return nil
}

// DesbloquearTarea resuelve el bloqueo y libera la tarea.
func DesbloquearTarea(id int64, agente, resolucion string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE tareas SET estado='libre', agente=NULL WHERE id=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`
		UPDATE bloqueos SET resuelto=1, resolucion=?, resuelto_at=CURRENT_TIMESTAMP
		WHERE tarea_id=? AND resuelto=0`, resolucion, id); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	Audit(agente, "desbloquear_tarea", "tarea", id, resolucion)
	return nil
}

// AnotarTarea añade una nota a una tarea.
func AnotarTarea(id int64, agente, nota string) error {
	_, err := DB.Exec(
		`UPDATE tareas SET notas = notas || char(10) || ? || ' [' || datetime('now') || ' ' || ? || ']' WHERE id=?`,
		nota, agente, id,
	)
	return err
}

// ContarTareasPorEstado devuelve un mapa estado→cantidad.
func ContarTareasPorEstado() (map[string]int, error) {
	rows, err := DB.Query(`SELECT estado, COUNT(*) FROM tareas GROUP BY estado`)
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

// ─── helpers ────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func escanearTarea(s scanner) (*Tarea, error) {
	var t Tarea
	var depsJSON string
	var agente sql.NullString
	var propuestaID sql.NullInt64
	var completadaAt sql.NullTime
	err := s.Scan(
		&t.ID, &t.Titulo, &t.Descripcion, &t.Modulo, &t.Estado, &agente, &propuestaID,
		&t.Prioridad, &depsJSON, &t.CreadoPor, &t.CommitCierre, &t.Notas,
		&t.CreatedAt, &t.UpdatedAt, &completadaAt,
	)
	if err != nil {
		return nil, err
	}
	if agente.Valid {
		t.Agente = &agente.String
	}
	if propuestaID.Valid {
		t.PropuestaID = &propuestaID.Int64
	}
	_ = json.Unmarshal([]byte(depsJSON), &t.Dependencias)
	if completadaAt.Valid {
		t.CompletadaAt = &completadaAt.Time
	}
	return &t, nil
}
