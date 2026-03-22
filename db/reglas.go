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
	return ListarReglas(tipoAgente, false)
}

func ListarReglas(tipoAgente string, incluirInactivas bool) ([]*Regla, error) {
	q := `
		SELECT id, tipo_agente, categoria, titulo, descripcion, activa, created_at
		FROM reglas
		WHERE 1=1`
	args := []any{}
	if tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if !incluirInactivas {
		q += ` AND activa = 1`
	}
	q += ` ORDER BY tipo_agente, categoria, id`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Regla
	for rows.Next() {
		r, err := escanearRegla(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func GetRegla(id int64) (*Regla, error) {
	row := DB.QueryRow(`
		SELECT id, tipo_agente, categoria, titulo, descripcion, activa, created_at
		FROM reglas
		WHERE id = ?`, id)
	return escanearRegla(row)
}

func CrearRegla(actor string, r *Regla) (int64, error) {
	if r == nil {
		return 0, fmt.Errorf("regla nula")
	}
	if r.TipoAgente == "" || r.Categoria == "" || r.Titulo == "" {
		return 0, fmt.Errorf("tipo_agente, categoria y titulo son obligatorios")
	}
	if err := validarPermisoEdicionCatalogo(actor, "reglas", r.TipoAgente, "crear"); err != nil {
		return 0, err
	}
	if !r.Activa {
		r.Activa = true
	}
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO reglas (tipo_agente, categoria, titulo, descripcion, activa)
		VALUES (?,?,?,?,?)`,
		r.TipoAgente, r.Categoria, r.Titulo, r.Descripcion, r.Activa,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	r.ID = id
	if err := registrarVersionReglaTx(tx, r, actor, "crear"); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(actor, "crear_regla", "regla", id, r.Titulo)
	return id, nil
}

func ActualizarRegla(actor string, r *Regla) error {
	if r == nil {
		return fmt.Errorf("regla nula")
	}
	if r.ID <= 0 {
		return fmt.Errorf("id de regla obligatorio")
	}
	if r.TipoAgente == "" || r.Categoria == "" || r.Titulo == "" {
		return fmt.Errorf("tipo_agente, categoria y titulo son obligatorios")
	}
	actual, err := GetRegla(r.ID)
	if err != nil {
		return err
	}
	if err := validarPermisoEdicionCatalogo(actor, "reglas", actual.TipoAgente, "editar"); err != nil {
		return err
	}
	if actual.TipoAgente != r.TipoAgente {
		if err := validarPermisoEdicionCatalogo(actor, "reglas", r.TipoAgente, "editar"); err != nil {
			return err
		}
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE reglas
		SET tipo_agente=?, categoria=?, titulo=?, descripcion=?, activa=?
		WHERE id=?`,
		r.TipoAgente, r.Categoria, r.Titulo, r.Descripcion, r.Activa, r.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("regla #%d no encontrada", r.ID)
	}
	if err := registrarVersionReglaTx(tx, r, actor, "editar"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	Audit(actor, "actualizar_regla", "regla", r.ID, r.Titulo)
	return nil
}

func SetReglaActiva(actor string, id int64, activa bool) error {
	actual, err := GetRegla(id)
	if err != nil {
		return err
	}
	if err := validarPermisoEdicionCatalogo(actor, "reglas", actual.TipoAgente, "activar"); err != nil {
		return err
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE reglas SET activa=? WHERE id=?`, activa, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("regla #%d no encontrada", id)
	}
	actual.Activa = activa
	if err := registrarVersionReglaTx(tx, actual, actor, map[bool]string{true: "activar", false: "desactivar"}[activa]); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	accion := "desactivar_regla"
	if activa {
		accion = "activar_regla"
	}
	Audit(actor, accion, "regla", id, "")
	return nil
}

// GetSkillsAgente devuelve los skills activos para el rol de un agente.
func GetSkillsAgente(tipoAgente string) ([]*Skill, error) {
	return ListarSkills(tipoAgente, false)
}

func ListarSkills(tipoAgente string, incluirInactivos bool) ([]*Skill, error) {
	q := `
		SELECT id, tipo_agente, nombre, descripcion, cuando_usar, activa, created_at
		FROM skills
		WHERE 1=1`
	args := []any{}
	if tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if !incluirInactivos {
		q += ` AND activa = 1`
	}
	q += ` ORDER BY tipo_agente, nombre`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Skill
	for rows.Next() {
		s, err := escanearSkill(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func GetSkill(id int64) (*Skill, error) {
	row := DB.QueryRow(`
		SELECT id, tipo_agente, nombre, descripcion, cuando_usar, activa, created_at
		FROM skills
		WHERE id = ?`, id)
	return escanearSkill(row)
}

func CrearSkill(actor string, s *Skill) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("skill nulo")
	}
	if s.TipoAgente == "" || s.Nombre == "" {
		return 0, fmt.Errorf("tipo_agente y nombre son obligatorios")
	}
	if err := validarPermisoEdicionCatalogo(actor, "skills", s.TipoAgente, "crear"); err != nil {
		return 0, err
	}
	if !s.Activa {
		s.Activa = true
	}
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar, activa)
		VALUES (?,?,?,?,?)`,
		s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Activa,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	s.ID = id
	if err := registrarVersionSkillTx(tx, s, actor, "crear"); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(actor, "crear_skill", "skill", id, s.Nombre)
	return id, nil
}

func ActualizarSkill(actor string, s *Skill) error {
	if s == nil {
		return fmt.Errorf("skill nulo")
	}
	if s.ID <= 0 {
		return fmt.Errorf("id de skill obligatorio")
	}
	if s.TipoAgente == "" || s.Nombre == "" {
		return fmt.Errorf("tipo_agente y nombre son obligatorios")
	}
	actual, err := GetSkill(s.ID)
	if err != nil {
		return err
	}
	if err := validarPermisoEdicionCatalogo(actor, "skills", actual.TipoAgente, "editar"); err != nil {
		return err
	}
	if actual.TipoAgente != s.TipoAgente {
		if err := validarPermisoEdicionCatalogo(actor, "skills", s.TipoAgente, "editar"); err != nil {
			return err
		}
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE skills
		SET tipo_agente=?, nombre=?, descripcion=?, cuando_usar=?, activa=?
		WHERE id=?`,
		s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Activa, s.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("skill #%d no encontrado", s.ID)
	}
	if err := registrarVersionSkillTx(tx, s, actor, "editar"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	Audit(actor, "actualizar_skill", "skill", s.ID, s.Nombre)
	return nil
}

func SetSkillActivo(actor string, id int64, activo bool) error {
	actual, err := GetSkill(id)
	if err != nil {
		return err
	}
	if err := validarPermisoEdicionCatalogo(actor, "skills", actual.TipoAgente, "activar"); err != nil {
		return err
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE skills SET activa=? WHERE id=?`, activo, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("skill #%d no encontrado", id)
	}
	actual.Activa = activo
	if err := registrarVersionSkillTx(tx, actual, actor, map[bool]string{true: "activar", false: "desactivar"}[activo]); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	accion := "desactivar_skill"
	if activo {
		accion = "activar_skill"
	}
	Audit(actor, accion, "skill", id, "")
	return nil
}

// GetWorkflowsAgente devuelve los workflows activos para el rol de un agente.
func GetWorkflowsAgente(tipoAgente string) ([]*Workflow, error) {
	return ListarWorkflows(tipoAgente, false)
}

func ListarWorkflows(tipoAgente string, incluirInactivos bool) ([]*Workflow, error) {
	q := `
		SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at
		FROM workflows
		WHERE 1=1`
	args := []any{}
	if tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if !incluirInactivos {
		q += ` AND activo = 1`
	}
	q += ` ORDER BY tipo_agente, nombre`

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Workflow
	for rows.Next() {
		w, err := escanearWorkflow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

// GetWorkflow devuelve un workflow concreto por nombre y tipo de agente.
func GetWorkflow(tipoAgente, nombre string) (*Workflow, error) {
	row := DB.QueryRow(
		`SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at
		 FROM workflows WHERE tipo_agente = ? AND nombre = ?`,
		tipoAgente, nombre,
	)
	return escanearWorkflow(row)
}

func GetWorkflowByID(id int64) (*Workflow, error) {
	row := DB.QueryRow(`
		SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at
		FROM workflows
		WHERE id = ?`, id)
	return escanearWorkflow(row)
}

func CrearWorkflow(actor string, w *Workflow) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("workflow nulo")
	}
	if w.TipoAgente == "" || w.Nombre == "" {
		return 0, fmt.Errorf("tipo_agente y nombre son obligatorios")
	}
	if err := validarPermisoEdicionCatalogo(actor, "workflows", w.TipoAgente, "crear"); err != nil {
		return 0, err
	}
	if w.Pasos == "" {
		w.Pasos = "[]"
	}
	if !w.Activo {
		w.Activo = true
	}
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO workflows (tipo_agente, nombre, descripcion, pasos, activo)
		VALUES (?,?,?,?,?)`,
		w.TipoAgente, w.Nombre, w.Descripcion, w.Pasos, w.Activo,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	w.ID = id
	if err := registrarVersionWorkflowTx(tx, w, actor, "crear"); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(actor, "crear_workflow", "workflow", id, w.Nombre)
	return id, nil
}

func ActualizarWorkflow(actor string, w *Workflow) error {
	if w == nil {
		return fmt.Errorf("workflow nulo")
	}
	if w.ID <= 0 {
		return fmt.Errorf("id de workflow obligatorio")
	}
	if w.TipoAgente == "" || w.Nombre == "" {
		return fmt.Errorf("tipo_agente y nombre son obligatorios")
	}
	actual, err := GetWorkflowByID(w.ID)
	if err != nil {
		return err
	}
	if err := validarPermisoEdicionCatalogo(actor, "workflows", actual.TipoAgente, "editar"); err != nil {
		return err
	}
	if actual.TipoAgente != w.TipoAgente {
		if err := validarPermisoEdicionCatalogo(actor, "workflows", w.TipoAgente, "editar"); err != nil {
			return err
		}
	}
	if w.Pasos == "" {
		w.Pasos = "[]"
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE workflows
		SET tipo_agente=?, nombre=?, descripcion=?, pasos=?, activo=?
		WHERE id=?`,
		w.TipoAgente, w.Nombre, w.Descripcion, w.Pasos, w.Activo, w.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow #%d no encontrado", w.ID)
	}
	if err := registrarVersionWorkflowTx(tx, w, actor, "editar"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	Audit(actor, "actualizar_workflow", "workflow", w.ID, w.Nombre)
	return nil
}

func SetWorkflowActivo(actor string, id int64, activo bool) error {
	actual, err := GetWorkflowByID(id)
	if err != nil {
		return err
	}
	if err := validarPermisoEdicionCatalogo(actor, "workflows", actual.TipoAgente, "activar"); err != nil {
		return err
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE workflows SET activo=? WHERE id=?`, activo, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workflow #%d no encontrado", id)
	}
	actual.Activo = activo
	if err := registrarVersionWorkflowTx(tx, actual, actor, map[bool]string{true: "activar", false: "desactivar"}[activo]); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	accion := "desactivar_workflow"
	if activo {
		accion = "activar_workflow"
	}
	Audit(actor, accion, "workflow", id, "")
	return nil
}

func escanearRegla(s scanner) (*Regla, error) {
	r := &Regla{}
	if err := s.Scan(&r.ID, &r.TipoAgente, &r.Categoria, &r.Titulo, &r.Descripcion, &r.Activa, &r.CreatedAt); err != nil {
		return nil, err
	}
	return r, nil
}

func escanearSkill(s scanner) (*Skill, error) {
	skill := &Skill{}
	if err := s.Scan(&skill.ID, &skill.TipoAgente, &skill.Nombre, &skill.Descripcion, &skill.CuandoUsar, &skill.Activa, &skill.CreatedAt); err != nil {
		return nil, err
	}
	return skill, nil
}

func escanearWorkflow(s scanner) (*Workflow, error) {
	w := &Workflow{}
	if err := s.Scan(&w.ID, &w.TipoAgente, &w.Nombre, &w.Descripcion, &w.Pasos, &w.Activo, &w.CreatedAt); err != nil {
		return nil, err
	}
	return w, nil
}
