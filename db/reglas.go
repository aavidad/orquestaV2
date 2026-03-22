/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"encoding/json"
	"fmt"
	"strings"
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

func ListarReglas(tipoAgente string, activa *bool) ([]*Regla, error) {
	q := `SELECT id, tipo_agente, categoria, titulo, descripcion, activa, created_at
		FROM reglas WHERE 1=1`
	args := []any{}
	if tipoAgente = strings.TrimSpace(tipoAgente); tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if activa != nil {
		q += ` AND activa = ?`
		args = append(args, *activa)
	}
	q += ` ORDER BY tipo_agente, categoria, id`
	rows, err := DB.Query(q, args...)
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

func GuardarRegla(r *Regla) (int64, error) {
	if r == nil {
		return 0, fmt.Errorf("regla obligatoria")
	}
	r.TipoAgente = strings.TrimSpace(r.TipoAgente)
	r.Categoria = strings.TrimSpace(r.Categoria)
	r.Titulo = strings.TrimSpace(r.Titulo)
	r.Descripcion = strings.TrimSpace(r.Descripcion)
	if r.TipoAgente == "" || r.Categoria == "" || r.Titulo == "" || r.Descripcion == "" {
		return 0, fmt.Errorf("tipo_agente, categoria, titulo y descripcion son obligatorios")
	}
	if !tipoAgenteValido(r.TipoAgente) {
		return 0, fmt.Errorf("tipo_agente invalido: %s", r.TipoAgente)
	}
	if _, err := DB.Exec(`
		INSERT INTO reglas (tipo_agente, categoria, titulo, descripcion, activa)
		VALUES (?,?,?,?,?)
		ON CONFLICT(tipo_agente, titulo) DO UPDATE SET
			categoria=excluded.categoria,
			descripcion=excluded.descripcion,
			activa=excluded.activa`,
		r.TipoAgente, r.Categoria, r.Titulo, r.Descripcion, r.Activa,
	); err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM reglas WHERE tipo_agente = ? AND titulo = ?`,
		r.TipoAgente, r.Titulo).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
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

func ListarSkills(tipoAgente string, activa *bool) ([]*Skill, error) {
	q := `SELECT id, tipo_agente, nombre, descripcion, cuando_usar, activa, created_at
		FROM skills WHERE 1=1`
	args := []any{}
	if tipoAgente = strings.TrimSpace(tipoAgente); tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if activa != nil {
		q += ` AND activa = ?`
		args = append(args, *activa)
	}
	q += ` ORDER BY tipo_agente, nombre`
	rows, err := DB.Query(q, args...)
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

func GuardarSkill(s *Skill) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("skill obligatoria")
	}
	s.TipoAgente = strings.TrimSpace(s.TipoAgente)
	s.Nombre = strings.TrimSpace(s.Nombre)
	s.Descripcion = strings.TrimSpace(s.Descripcion)
	s.CuandoUsar = strings.TrimSpace(s.CuandoUsar)
	if s.TipoAgente == "" || s.Nombre == "" || s.Descripcion == "" {
		return 0, fmt.Errorf("tipo_agente, nombre y descripcion son obligatorios")
	}
	if !tipoAgenteValido(s.TipoAgente) {
		return 0, fmt.Errorf("tipo_agente invalido: %s", s.TipoAgente)
	}
	if _, err := DB.Exec(`
		INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar, activa)
		VALUES (?,?,?,?,?)
		ON CONFLICT(tipo_agente, nombre) DO UPDATE SET
			descripcion=excluded.descripcion,
			cuando_usar=excluded.cuando_usar,
			activa=excluded.activa`,
		s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Activa,
	); err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM skills WHERE tipo_agente = ? AND nombre = ?`,
		s.TipoAgente, s.Nombre).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
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

func ListarWorkflows(tipoAgente string, activo *bool) ([]*Workflow, error) {
	q := `SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at
		FROM workflows WHERE 1=1`
	args := []any{}
	if tipoAgente = strings.TrimSpace(tipoAgente); tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if activo != nil {
		q += ` AND activo = ?`
		args = append(args, *activo)
	}
	q += ` ORDER BY tipo_agente, nombre`
	rows, err := DB.Query(q, args...)
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

func GuardarWorkflow(w *Workflow) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("workflow obligatorio")
	}
	w.TipoAgente = strings.TrimSpace(w.TipoAgente)
	w.Nombre = strings.TrimSpace(w.Nombre)
	w.Descripcion = strings.TrimSpace(w.Descripcion)
	w.Pasos = strings.TrimSpace(w.Pasos)
	if w.TipoAgente == "" || w.Nombre == "" || w.Descripcion == "" || w.Pasos == "" {
		return 0, fmt.Errorf("tipo_agente, nombre, descripcion y pasos son obligatorios")
	}
	if !tipoAgenteValido(w.TipoAgente) {
		return 0, fmt.Errorf("tipo_agente invalido: %s", w.TipoAgente)
	}
	var pasos []string
	if err := json.Unmarshal([]byte(w.Pasos), &pasos); err != nil || len(pasos) == 0 {
		return 0, fmt.Errorf("pasos debe ser un JSON array no vacio")
	}
	if _, err := DB.Exec(`
		INSERT INTO workflows (tipo_agente, nombre, descripcion, pasos, activo)
		VALUES (?,?,?,?,?)
		ON CONFLICT(nombre) DO UPDATE SET
			tipo_agente=excluded.tipo_agente,
			descripcion=excluded.descripcion,
			pasos=excluded.pasos,
			activo=excluded.activo`,
		w.TipoAgente, w.Nombre, w.Descripcion, w.Pasos, w.Activo,
	); err != nil {
		return 0, err
	}
	var id int64
	if err := DB.QueryRow(`SELECT id FROM workflows WHERE nombre = ?`, w.Nombre).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
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

type GovernanceRepository struct{}

func (GovernanceRepository) ListRules(tipoAgente string, activa *bool) ([]*Regla, error) {
	return ListarReglas(tipoAgente, activa)
}

func (GovernanceRepository) SaveRule(r *Regla) (int64, error) {
	return GuardarRegla(r)
}

func (GovernanceRepository) ListSkills(tipoAgente string, activa *bool) ([]*Skill, error) {
	return ListarSkills(tipoAgente, activa)
}

func (GovernanceRepository) SaveSkill(s *Skill) (int64, error) {
	return GuardarSkill(s)
}

func (GovernanceRepository) ListWorkflows(tipoAgente string, activo *bool) ([]*Workflow, error) {
	return ListarWorkflows(tipoAgente, activo)
}

func (GovernanceRepository) SaveWorkflow(w *Workflow) (int64, error) {
	return GuardarWorkflow(w)
}

func tipoAgenteValido(tipo string) bool {
	switch tipo {
	case "programador", "documentador", "admin":
		return true
	default:
		return false
	}
}
