/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"crypto/sha256"
	"encoding/hex"
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
	ID                 int64
	TipoAgente         string
	Nombre             string
	Descripcion        string
	CuandoUsar         string
	Escenario          string
	Prioridad          int
	AliasesJSON        string
	HerramientasJSON   string
	Origen             string
	NivelRiesgo        string
	RequiereAprobacion bool
	Activa             bool
	CreatedAt          time.Time
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

type GovernanceCatalog struct {
	TipoAgente string      `json:"tipo_agente"`
	ProyectoID *int64      `json:"proyecto_id,omitempty"`
	Reglas     []*Regla    `json:"reglas"`
	Skills     []*Skill    `json:"skills"`
	Workflows  []*Workflow `json:"workflows"`
	Hash       string      `json:"hash"`
}

// ResolveGovernanceCatalog devuelve el catálogo efectivo actual para un rol y
// proyecto. Hoy la resolución real es por rol; el proyecto queda en la firma
// para mantener estable el punto de integración cuando entren overrides.
func ResolveGovernanceCatalog(tipoAgente string, proyectoID *int64) (*GovernanceCatalog, error) {
	tipoAgente = strings.TrimSpace(tipoAgente)
	reglas, err := GetReglasAgente(tipoAgente)
	if err != nil {
		return nil, err
	}
	skills, err := GetSkillsAgente(tipoAgente)
	if err != nil {
		return nil, err
	}
	workflows, err := GetWorkflowsAgente(tipoAgente)
	if err != nil {
		return nil, err
	}
	catalogo := &GovernanceCatalog{
		TipoAgente: tipoAgente,
		ProyectoID: proyectoID,
		Reglas:     reglas,
		Skills:     skills,
		Workflows:  workflows,
	}
	catalogo.Hash = governanceCatalogHash(catalogo)
	return catalogo, nil
}

func governanceCatalogHash(catalogo *GovernanceCatalog) string {
	if catalogo == nil {
		return ""
	}
	input := struct {
		TipoAgente string   `json:"tipo_agente"`
		ProyectoID int64    `json:"proyecto_id"`
		Reglas     []string `json:"reglas"`
		Skills     []string `json:"skills"`
		Workflows  []string `json:"workflows"`
	}{
		TipoAgente: catalogo.TipoAgente,
	}
	if catalogo.ProyectoID != nil {
		input.ProyectoID = *catalogo.ProyectoID
	}
	for _, regla := range catalogo.Reglas {
		if regla == nil {
			continue
		}
		input.Reglas = append(input.Reglas, fmt.Sprintf("%d|%s|%s|%s", regla.ID, regla.Categoria, regla.Titulo, regla.Descripcion))
	}
	for _, skill := range catalogo.Skills {
		if skill == nil {
			continue
		}
		input.Skills = append(input.Skills, fmt.Sprintf("%d|%s|%s|%s|%d", skill.ID, skill.Nombre, skill.Descripcion, skill.CuandoUsar, skill.Prioridad))
	}
	for _, workflow := range catalogo.Workflows {
		if workflow == nil {
			continue
		}
		input.Workflows = append(input.Workflows, fmt.Sprintf("%d|%s|%s|%s", workflow.ID, workflow.Nombre, workflow.Descripcion, workflow.Pasos))
	}
	data, _ := json.Marshal(input)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}

func BuildGovernanceContextSummary(tipoAgente string, proyectoID *int64) (map[string]any, string) {
	catalogo, err := ResolveGovernanceCatalog(tipoAgente, proyectoID)
	if err != nil || catalogo == nil {
		return nil, ""
	}
	contexto := map[string]any{
		"tipo_agente":       strings.TrimSpace(catalogo.TipoAgente),
		"hash":              strings.TrimSpace(catalogo.Hash),
		"reglas":            len(catalogo.Reglas),
		"skills":            len(catalogo.Skills),
		"workflows":         len(catalogo.Workflows),
		"resolucion_actual": "rol",
	}
	if proyectoID != nil {
		contexto["proyecto_id"] = *proyectoID
	}
	resumen := fmt.Sprintf(
		"Catálogo efectivo %s (%d reglas, %d skills, %d workflows)",
		strings.TrimSpace(catalogo.Hash),
		len(catalogo.Reglas),
		len(catalogo.Skills),
		len(catalogo.Workflows),
	)
	return contexto, resumen
}

func AppendGovernanceCatalogPayload(prev string, contexto map[string]any) string {
	prev = strings.TrimSpace(prev)
	if len(contexto) == 0 {
		return prev
	}
	envelope := map[string]any{}
	if prev != "" {
		var parsed any
		if err := json.Unmarshal([]byte(prev), &parsed); err == nil {
			if obj, ok := parsed.(map[string]any); ok {
				for key, value := range obj {
					envelope[key] = value
				}
			} else {
				envelope["resume_previo"] = parsed
			}
		} else {
			envelope["resume_previo_raw"] = prev
		}
	}
	envelope["governance_catalog"] = contexto
	data, err := json.Marshal(envelope)
	if err != nil {
		return prev
	}
	return string(data)
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
		`SELECT id, tipo_agente, nombre, descripcion, cuando_usar, escenario, prioridad, aliases_json, herramientas_json, origen, nivel_riesgo, requiere_aprobacion, activa, created_at
		 FROM skills WHERE tipo_agente = ? AND activa = 1
		 ORDER BY prioridad ASC, escenario ASC, nombre ASC`, tipoAgente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Skill
	for rows.Next() {
		s := &Skill{}
		var requiereAprobacion int
		if err := rows.Scan(&s.ID, &s.TipoAgente, &s.Nombre, &s.Descripcion,
			&s.CuandoUsar, &s.Escenario, &s.Prioridad, &s.AliasesJSON, &s.HerramientasJSON, &s.Origen, &s.NivelRiesgo, &requiereAprobacion, &s.Activa, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.RequiereAprobacion = requiereAprobacion == 1
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
	if err := aplicarPoliticaSeguridadSkill("telegram_admin", s, s.Activa); err != nil {
		return 0, err
	}
	if existente, err := BuscarSkillEquivalente(s, s.ID); err != nil {
		return 0, err
	} else if existente != nil && !sameSkillNaturalKey(existente, s) {
		return 0, &SkillEquivalenteError{Existente: existente}
	}
	res, err := DB.Exec(`
		INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar, escenario, prioridad, aliases_json, herramientas_json, origen, nivel_riesgo, requiere_aprobacion, activa)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(tipo_agente, nombre) DO UPDATE SET
			descripcion = excluded.descripcion,
			cuando_usar = excluded.cuando_usar,
			escenario = excluded.escenario,
			prioridad = excluded.prioridad,
			aliases_json = excluded.aliases_json,
			herramientas_json = excluded.herramientas_json,
			origen = excluded.origen,
			nivel_riesgo = excluded.nivel_riesgo,
			requiere_aprobacion = excluded.requiere_aprobacion,
			activa = excluded.activa
	`, s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Escenario, s.Prioridad, s.AliasesJSON, s.HerramientasJSON, s.Origen, s.NivelRiesgo, boolToInt(s.RequiereAprobacion), s.Activa)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id <= 0 {
		if err := DB.QueryRow(`SELECT id FROM skills WHERE tipo_agente = ? AND nombre = ?`, s.TipoAgente, s.Nombre).Scan(&id); err != nil {
			return 0, err
		}
	}
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

type GovernanceRepository struct{}

func (GovernanceRepository) ListRules(tipoAgente string, activa *bool) ([]*Regla, error) {
	return ListarReglas(tipoAgente, activa)
}

func (GovernanceRepository) SaveRule(r *Regla) (int64, error) {
	return UpsertRegla(r)
}

func (GovernanceRepository) ListSkills(tipoAgente string, activa *bool) ([]*Skill, error) {
	return ListarSkills(tipoAgente, activa)
}

func (GovernanceRepository) SaveSkill(s *Skill) (int64, error) {
	return UpsertSkill(s)
}

func (GovernanceRepository) ListWorkflows(tipoAgente string, activo *bool) ([]*Workflow, error) {
	return ListarWorkflows(tipoAgente, activo)
}

func (GovernanceRepository) SaveWorkflow(w *Workflow) (int64, error) {
	return UpsertWorkflow(w)
}

func GuardarRegla(r *Regla) (int64, error) {
	return UpsertRegla(r)
}

func GuardarSkill(s *Skill) (int64, error) {
	return UpsertSkill(s)
}

func GuardarWorkflow(w *Workflow) (int64, error) {
	return UpsertWorkflow(w)
}

func ListarReglas(tipoAgente string, activa any) ([]*Regla, error) {
	q := `SELECT id, tipo_agente, categoria, titulo, descripcion, activa, created_at FROM reglas WHERE 1=1`
	var args []any
	activaFiltro := resolverFiltroBool(activa)
	if tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if activaFiltro != nil {
		q += ` AND activa = ?`
		args = append(args, *activaFiltro)
	}
	q += ` ORDER BY categoria, id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Regla
	for rows.Next() {
		r := &Regla{}
		if err := rows.Scan(&r.ID, &r.TipoAgente, &r.Categoria, &r.Titulo, &r.Descripcion, &r.Activa, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func CrearRegla(actor string, r *Regla) (int64, error) {
	id, err := UpsertRegla(r)
	if err == nil {
		Audit(actor, "crear_regla", "regla", id, r.Titulo)
		notificarRefreshGobernanza(actor, r.TipoAgente, "crear_regla", map[string]any{
			"regla_id": id,
			"titulo":   r.Titulo,
		})
	}
	return id, err
}

func GetRegla(id int64) (*Regla, error) {
	r := &Regla{}
	err := DB.QueryRow(`SELECT id, tipo_agente, categoria, titulo, descripcion, activa, created_at FROM reglas WHERE id = ?`, id).
		Scan(&r.ID, &r.TipoAgente, &r.Categoria, &r.Titulo, &r.Descripcion, &r.Activa, &r.CreatedAt)
	return r, err
}

func ActualizarRegla(actor string, r *Regla) error {
	_, err := DB.Exec(`UPDATE reglas SET tipo_agente=?, categoria=?, titulo=?, descripcion=?, activa=? WHERE id=?`,
		r.TipoAgente, r.Categoria, r.Titulo, r.Descripcion, r.Activa, r.ID)
	if err == nil {
		Audit(actor, "actualizar_regla", "regla", r.ID, r.Titulo)
		notificarRefreshGobernanza(actor, r.TipoAgente, "actualizar_regla", map[string]any{
			"regla_id": r.ID,
			"titulo":   r.Titulo,
		})
	}
	return err
}

func SetReglaActiva(actor string, id int64, activa bool) error {
	regla, err := GetRegla(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`UPDATE reglas SET activa=? WHERE id=?`, activa, id)
	if err == nil {
		Audit(actor, "set_regla_activa", "regla", id, fmt.Sprintf("activa=%t", activa))
		notificarRefreshGobernanza(actor, regla.TipoAgente, "set_regla_activa", map[string]any{
			"regla_id": id,
			"activa":   activa,
			"titulo":   regla.Titulo,
		})
	}
	return err
}

func ListarSkills(tipoAgente string, activa any) ([]*Skill, error) {
	q := `SELECT id, tipo_agente, nombre, descripcion, cuando_usar, escenario, prioridad, aliases_json, herramientas_json, origen, nivel_riesgo, requiere_aprobacion, activa, created_at FROM skills WHERE 1=1`
	var args []any
	activaFiltro := resolverFiltroBool(activa)
	if tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if activaFiltro != nil {
		q += ` AND activa = ?`
		args = append(args, *activaFiltro)
	}
	q += ` ORDER BY prioridad ASC, escenario ASC, nombre ASC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Skill
	for rows.Next() {
		s := &Skill{}
		var requiereAprobacion int
		if err := rows.Scan(&s.ID, &s.TipoAgente, &s.Nombre, &s.Descripcion, &s.CuandoUsar, &s.Escenario, &s.Prioridad, &s.AliasesJSON, &s.HerramientasJSON, &s.Origen, &s.NivelRiesgo, &requiereAprobacion, &s.Activa, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.RequiereAprobacion = requiereAprobacion == 1
		list = append(list, s)
	}
	return list, rows.Err()
}

func CrearSkill(actor string, s *Skill) (int64, error) {
	if err := aplicarPoliticaSeguridadSkill(actor, s, false); err != nil {
		return 0, err
	}
	if existente, err := BuscarSkillEquivalente(s, 0); err != nil {
		return 0, err
	} else if existente != nil && !sameSkillNaturalKey(existente, s) {
		return 0, &SkillEquivalenteError{Existente: existente}
	}
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar, escenario, prioridad, aliases_json, herramientas_json, origen, nivel_riesgo, requiere_aprobacion, activa)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Escenario, s.Prioridad, s.AliasesJSON, s.HerramientasJSON, s.Origen, s.NivelRiesgo, boolToInt(s.RequiereAprobacion), s.Activa,
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
	notificarRefreshSkillCatalogo(actor, s, "crear")
	return id, nil
}

func GetSkill(id int64) (*Skill, error) {
	s := &Skill{}
	var requiereAprobacion int
	err := DB.QueryRow(`SELECT id, tipo_agente, nombre, descripcion, cuando_usar, escenario, prioridad, aliases_json, herramientas_json, origen, nivel_riesgo, requiere_aprobacion, activa, created_at FROM skills WHERE id = ?`, id).
		Scan(&s.ID, &s.TipoAgente, &s.Nombre, &s.Descripcion, &s.CuandoUsar, &s.Escenario, &s.Prioridad, &s.AliasesJSON, &s.HerramientasJSON, &s.Origen, &s.NivelRiesgo, &requiereAprobacion, &s.Activa, &s.CreatedAt)
	s.RequiereAprobacion = requiereAprobacion == 1
	return s, err
}

func ActualizarSkill(actor string, s *Skill) error {
	if err := aplicarPoliticaSeguridadSkill(actor, s, s.Activa); err != nil {
		return err
	}
	if existente, err := BuscarSkillEquivalente(s, s.ID); err != nil {
		return err
	} else if existente != nil && existente.ID != s.ID {
		return &SkillEquivalenteError{Existente: existente}
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`UPDATE skills SET tipo_agente=?, nombre=?, descripcion=?, cuando_usar=?, escenario=?, prioridad=?, aliases_json=?, herramientas_json=?, origen=?, nivel_riesgo=?, requiere_aprobacion=?, activa=? WHERE id=?`,
		s.TipoAgente, s.Nombre, s.Descripcion, s.CuandoUsar, s.Escenario, s.Prioridad, s.AliasesJSON, s.HerramientasJSON, s.Origen, s.NivelRiesgo, boolToInt(s.RequiereAprobacion), s.Activa, s.ID)
	if err != nil {
		return err
	}
	if err := registrarVersionSkillTx(tx, s, actor, "actualizar"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	Audit(actor, "actualizar_skill", "skill", s.ID, s.Nombre)
	notificarRefreshSkillCatalogo(actor, s, "actualizar")
	return nil
}

func SetSkillActiva(actor string, id int64, activa bool) error {
	skill, err := GetSkill(id)
	if err != nil {
		return err
	}
	skill.Activa = activa
	if err := aplicarPoliticaSeguridadSkill(actor, skill, activa); err != nil {
		return err
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`UPDATE skills SET activa=?, requiere_aprobacion=? WHERE id=?`, skill.Activa, boolToInt(skill.RequiereAprobacion), id)
	if err != nil {
		return err
	}
	if err := registrarVersionSkillTx(tx, skill, actor, "activar"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	Audit(actor, "set_skill_activa", "skill", id, fmt.Sprintf("activa=%t", activa))
	notificarRefreshSkillCatalogo(actor, skill, "activar")
	return nil
}

func SetSkillActivo(actor string, id int64, activa bool) error {
	return SetSkillActiva(actor, id, activa)
}

func ListarWorkflows(tipoAgente string, activo any) ([]*Workflow, error) {
	q := `SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at FROM workflows WHERE 1=1`
	var args []any
	activoFiltro := resolverFiltroBool(activo)
	if tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if activoFiltro != nil {
		q += ` AND activo = ?`
		args = append(args, *activoFiltro)
	}
	q += ` ORDER BY nombre`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Workflow
	for rows.Next() {
		w := &Workflow{}
		if err := rows.Scan(&w.ID, &w.TipoAgente, &w.Nombre, &w.Descripcion, &w.Pasos, &w.Activo, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func CrearWorkflow(actor string, w *Workflow) (int64, error) {
	id, err := UpsertWorkflow(w)
	if err == nil {
		Audit(actor, "crear_workflow", "workflow", id, w.Nombre)
		notificarRefreshGobernanza(actor, w.TipoAgente, "crear_workflow", map[string]any{
			"workflow_id": id,
			"nombre":      w.Nombre,
		})
	}
	return id, err
}

func GetWorkflowPorID(id int64) (*Workflow, error) {
	w := &Workflow{}
	err := DB.QueryRow(`SELECT id, tipo_agente, nombre, descripcion, pasos, activo, created_at FROM workflows WHERE id = ?`, id).
		Scan(&w.ID, &w.TipoAgente, &w.Nombre, &w.Descripcion, &w.Pasos, &w.Activo, &w.CreatedAt)
	return w, err
}

func GetWorkflowByID(id int64) (*Workflow, error) {
	return GetWorkflowPorID(id)
}

func ActualizarWorkflow(actor string, w *Workflow) error {
	_, err := DB.Exec(`UPDATE workflows SET tipo_agente=?, nombre=?, descripcion=?, pasos=?, activo=? WHERE id=?`,
		w.TipoAgente, w.Nombre, w.Descripcion, w.Pasos, w.Activo, w.ID)
	if err == nil {
		Audit(actor, "actualizar_workflow", "workflow", w.ID, w.Nombre)
		notificarRefreshGobernanza(actor, w.TipoAgente, "actualizar_workflow", map[string]any{
			"workflow_id": w.ID,
			"nombre":      w.Nombre,
		})
	}
	return err
}

func SetWorkflowActivo(actor string, id int64, activo bool) error {
	workflow, err := GetWorkflowPorID(id)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`UPDATE workflows SET activo=? WHERE id=?`, activo, id)
	if err == nil {
		Audit(actor, "set_workflow_activo", "workflow", id, fmt.Sprintf("activo=%t", activo))
		notificarRefreshGobernanza(actor, workflow.TipoAgente, "set_workflow_activo", map[string]any{
			"workflow_id": id,
			"activo":      activo,
			"nombre":      workflow.Nombre,
		})
	}
	return err
}

func resolverFiltroBool(v any) *bool {
	switch value := v.(type) {
	case nil:
		return nil
	case bool:
		return &value
	case *bool:
		return value
	default:
		return nil
	}
}
