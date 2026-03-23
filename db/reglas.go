/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"fmt"
	"time"
)

// Regla representa una regla de comportamiento para un tipo de agente.
type Regla struct {
	ID          int64
	TipoAgente  string
	Categoria   string
	Titulo      string
	Descripcion string
	Activa      bool
	CreatedAt   time.Time
}

// Skill representa una habilidad/comando disponible para un tipo de agente.
type Skill struct {
	ID          int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	CuandoUsar  string
	Activa      bool
	CreatedAt   time.Time
}

// Workflow representa un flujo de trabajo paso a paso para un tipo de agente.
type Workflow struct {
	ID          int64
	TipoAgente  string
	Nombre      string
	Descripcion string
	Pasos       string // JSON array de strings
	Activo      bool
	CreatedAt   time.Time
}

// GetReglasAgente devuelve las reglas activas para el rol de un agente.
func GetReglasAgente(tipoAgente string) ([]*Regla, error) {
	rows, err := DB.Query(
		`SELECT id, tipo_agente, categoria, titulo, descripcion, activa, created_at
		 FROM reglas WHERE tipo_agente = ? AND activa = 1
		 ORDER BY categoria, id`, tipoAgente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Regla
	for rows.Next() {
		r := &Regla{}
		if err := rows.Scan(&r.ID, &r.TipoAgente, &r.Categoria, &r.Titulo,
			&r.Descripcion, &r.Activa, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// GetSkillsAgente devuelve los skills activos para el rol de un agente.
func GetSkillsAgente(tipoAgente string) ([]*Skill, error) {
	rows, err := DB.Query(
		`SELECT id, tipo_agente, nombre, descripcion, cuando_usar, activa, created_at
		 FROM skills WHERE tipo_agente = ? AND activa = 1
		 ORDER BY nombre`, tipoAgente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Skill
	for rows.Next() {
		s := &Skill{}
		if err := rows.Scan(&s.ID, &s.TipoAgente, &s.Nombre, &s.Descripcion,
			&s.CuandoUsar, &s.Activa, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// GetWorkflowsAgente devuelve los workflows activos para el rol de un agente.
func GetWorkflowsAgente(tipoAgente string) ([]*Workflow, error) {
	rows, err := DB.Query(
		`SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at
		 FROM workflows WHERE tipo_agente = ? AND activo = 1
		 ORDER BY nombre`, tipoAgente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Workflow
	for rows.Next() {
		w := &Workflow{}
		if err := rows.Scan(&w.ID, &w.TipoAgente, &w.Nombre, &w.Descripcion,
			&w.Pasos, &w.Activo, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

// GetWorkflow devuelve un workflow concreto por nombre y tipo de agente.
func GetWorkflow(tipoAgente, nombre string) (*Workflow, error) {
	w := &Workflow{}
	err := DB.QueryRow(
		`SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at
		 FROM workflows WHERE tipo_agente = ? AND nombre = ?`,
		tipoAgente, nombre,
	).Scan(&w.ID, &w.TipoAgente, &w.Nombre, &w.Descripcion,
		&w.Pasos, &w.Activo, &w.CreatedAt)
	return w, err
}

// UpsertRegla crea o actualiza una regla por título y tipo de agente.
func UpsertRegla(r *Regla) (int64, error) {
	res, err := DB.Exec(`
		INSERT INTO reglas (tipo_agente, categoria, titulo, descripcion, activa)
		VALUES (?,?,?,?,?)
		ON CONFLICT(tipo_agente, titulo) DO UPDATE SET
			categoria = excluded.categoria,
			descripcion = excluded.descripcion,
			activa = excluded.activa
	`, r.TipoAgente, r.Categoria, r.Titulo, r.Descripcion, r.Activa)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("telegram_admin", "upsert_regla", "regla", id, fmt.Sprintf("%s: %s", r.TipoAgente, r.Titulo))
	return id, nil
}

// UpsertSkill crea o actualiza una habilidad por nombre y tipo de agente.
func UpsertSkill(s *Skill) (int64, error) {
	res, err := DB.Exec(`
		INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar, activa)
		VALUES (?,?,?,?,?)
		ON CONFLICT(tipo_agente, nombre) DO UPDATE SET
			descripcion = excluded.descripcion,
			cuando_usar = excluded.cuando_usar,
			activa = excluded.activa
	`, s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Activa)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("telegram_admin", "upsert_skill", "skill", id, fmt.Sprintf("%s: %s", s.TipoAgente, s.Nombre))
	return id, nil
}

// UpsertWorkflow crea o actualiza un flujo de trabajo.
func UpsertWorkflow(w *Workflow) (int64, error) {
	res, err := DB.Exec(`
		INSERT INTO workflows (tipo_agente, nombre, descripcion, pasos, activo)
		VALUES (?,?,?,?,?)
		ON CONFLICT(nombre) DO UPDATE SET
			tipo_agente = excluded.tipo_agente,
			descripcion = excluded.descripcion,
			pasos = excluded.pasos,
			activo = excluded.activo
	`, w.TipoAgente, w.Nombre, w.Descripcion, w.Pasos, w.Activo)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	Audit("telegram_admin", "upsert_workflow", "workflow", id, fmt.Sprintf("%s: %s", w.TipoAgente, w.Nombre))
	return id, nil
}
