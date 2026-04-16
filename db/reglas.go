/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"fmt"
	"strings"
	"time"

	"orquesta/gobernanzapolicy"
)

type governanceTransitionCoordinator interface {
	afterCrearRegla(actor string, r *Regla, id int64) error
	afterActualizarRegla(actor string, r *Regla) error
	afterSetReglaActiva(actor string, r *Regla, activa bool) error
	afterCrearSkill(actor string, s *Skill, id int64) error
	afterActualizarSkill(actor string, s *Skill) error
	afterSetSkillActiva(actor string, s *Skill, activa bool) error
	afterCrearWorkflow(actor string, w *Workflow, id int64) error
	afterActualizarWorkflow(actor string, w *Workflow) error
	afterSetWorkflowActivo(actor string, w *Workflow, activo bool) error
}

type defaultGovernanceTransitionCoordinator struct{}

var defaultGovernanceTransitioner governanceTransitionCoordinator = defaultGovernanceTransitionCoordinator{}

func (defaultGovernanceTransitionCoordinator) afterCrearRegla(actor string, r *Regla, id int64) error {
	if r == nil {
		return nil
	}
	Audit(actor, "crear_regla", "regla", id, r.Titulo)
	notificarRefreshGobernanza(actor, r.TipoAgente, "crear_regla", map[string]any{
		"regla_id": id,
		"titulo":   r.Titulo,
	})
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterActualizarRegla(actor string, r *Regla) error {
	if r == nil {
		return nil
	}
	Audit(actor, "actualizar_regla", "regla", r.ID, r.Titulo)
	notificarRefreshGobernanza(actor, r.TipoAgente, "actualizar_regla", map[string]any{
		"regla_id": r.ID,
		"titulo":   r.Titulo,
	})
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterSetReglaActiva(actor string, r *Regla, activa bool) error {
	if r == nil {
		return nil
	}
	Audit(actor, "set_regla_activa", "regla", r.ID, fmt.Sprintf("activa=%t", activa))
	notificarRefreshGobernanza(actor, r.TipoAgente, "set_regla_activa", map[string]any{
		"regla_id": r.ID,
		"activa":   activa,
		"titulo":   r.Titulo,
	})
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterCrearSkill(actor string, s *Skill, id int64) error {
	if s == nil {
		return nil
	}
	Audit(actor, "crear_skill", "skill", id, s.Nombre)
	notificarRefreshSkillCatalogo(actor, s, "crear")
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterActualizarSkill(actor string, s *Skill) error {
	if s == nil {
		return nil
	}
	Audit(actor, "actualizar_skill", "skill", s.ID, s.Nombre)
	notificarRefreshSkillCatalogo(actor, s, "actualizar")
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterSetSkillActiva(actor string, s *Skill, activa bool) error {
	if s == nil {
		return nil
	}
	Audit(actor, "set_skill_activa", "skill", s.ID, fmt.Sprintf("activa=%t", activa))
	notificarRefreshSkillCatalogo(actor, s, "activar")
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterCrearWorkflow(actor string, w *Workflow, id int64) error {
	if w == nil {
		return nil
	}
	Audit(actor, "crear_workflow", "workflow", id, w.Nombre)
	notificarRefreshGobernanza(actor, w.TipoAgente, "crear_workflow", map[string]any{
		"workflow_id": id,
		"nombre":      w.Nombre,
	})
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterActualizarWorkflow(actor string, w *Workflow) error {
	if w == nil {
		return nil
	}
	Audit(actor, "actualizar_workflow", "workflow", w.ID, w.Nombre)
	notificarRefreshGobernanza(actor, w.TipoAgente, "actualizar_workflow", map[string]any{
		"workflow_id": w.ID,
		"nombre":      w.Nombre,
	})
	return nil
}

func (defaultGovernanceTransitionCoordinator) afterSetWorkflowActivo(actor string, w *Workflow, activo bool) error {
	if w == nil {
		return nil
	}
	Audit(actor, "set_workflow_activo", "workflow", w.ID, fmt.Sprintf("activo=%t", activo))
	notificarRefreshGobernanza(actor, w.TipoAgente, "set_workflow_activo", map[string]any{
		"workflow_id": w.ID,
		"activo":      activo,
		"nombre":      w.Nombre,
	})
	return nil
}

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
	TipoAgente       string      `json:"tipo_agente"`
	ProyectoID       *int64      `json:"proyecto_id,omitempty"`
	ResolucionActual string      `json:"resolucion_actual"`
	Reglas           []*Regla    `json:"reglas"`
	Skills           []*Skill    `json:"skills"`
	Workflows        []*Workflow `json:"workflows"`
	Hash             string      `json:"hash"`
}

// ResolveGovernanceCatalog devuelve el catálogo efectivo actual para un rol y
// proyecto. La resolución actual parte del catálogo por rol y puede aplicar
// overrides posteriores por proyecto.
func ResolveGovernanceCatalog(tipoAgente string, proyectoID *int64) (*GovernanceCatalog, error) {
	return ResolveGovernanceCatalogForContext(tipoAgente, proyectoID, "")
}

func governanceCatalogHash(catalogo *GovernanceCatalog) string {
	return gobernanzapolicy.HashCatalog(governanceCatalogSnapshot(catalogo))
}

func BuildGovernanceContextSummary(tipoAgente string, proyectoID *int64) (map[string]any, string) {
	return BuildGovernanceContextSummaryForContext(tipoAgente, proyectoID, "")
}

func BuildGovernanceContextSummaryFromCatalog(catalogo *GovernanceCatalog) (map[string]any, string) {
	return gobernanzapolicy.BuildContextSummaryFromCatalog(governanceCatalogSnapshot(catalogo))
}

func BuildGovernanceContextSummaryForContext(tipoAgente string, proyectoID *int64, agente string) (map[string]any, string) {
	catalogo, err := ResolveGovernanceCatalogForContext(tipoAgente, proyectoID, agente)
	if err != nil || catalogo == nil {
		return nil, ""
	}
	return BuildGovernanceContextSummaryFromCatalog(catalogo)
}

func AppendGovernanceCatalogPayload(prev string, contexto map[string]any) string {
	if len(contexto) == 0 {
		return strings.TrimSpace(prev)
	}
	return MergeResumePayloadEnvelope(prev, map[string]any{
		"governance_catalog": contexto,
	})
}

func governanceCatalogSnapshot(catalogo *GovernanceCatalog) *gobernanzapolicy.CatalogSnapshot {
	if catalogo == nil {
		return nil
	}
	snapshot := &gobernanzapolicy.CatalogSnapshot{
		AgentType:  catalogo.TipoAgente,
		ProjectID:  catalogo.ProyectoID,
		Resolution: catalogo.ResolucionActual,
		Hash:       catalogo.Hash,
		Rules:      make([]gobernanzapolicy.RuleSnapshot, 0, len(catalogo.Reglas)),
		Skills:     make([]gobernanzapolicy.SkillSnapshot, 0, len(catalogo.Skills)),
		Workflows:  make([]gobernanzapolicy.WorkflowSnapshot, 0, len(catalogo.Workflows)),
	}
	for _, regla := range catalogo.Reglas {
		if regla == nil {
			continue
		}
		snapshot.Rules = append(snapshot.Rules, gobernanzapolicy.RuleSnapshot{
			ID:          regla.ID,
			Category:    regla.Categoria,
			Title:       regla.Titulo,
			Description: regla.Descripcion,
		})
	}
	for _, skill := range catalogo.Skills {
		if skill == nil {
			continue
		}
		snapshot.Skills = append(snapshot.Skills, gobernanzapolicy.SkillSnapshot{
			ID:          skill.ID,
			Name:        skill.Nombre,
			Description: skill.Descripcion,
			WhenToUse:   skill.CuandoUsar,
			Priority:    skill.Prioridad,
		})
	}
	for _, workflow := range catalogo.Workflows {
		if workflow == nil {
			continue
		}
		snapshot.Workflows = append(snapshot.Workflows, gobernanzapolicy.WorkflowSnapshot{
			ID:          workflow.ID,
			Name:        workflow.Nombre,
			Description: workflow.Descripcion,
			Steps:       workflow.Pasos,
		})
	}
	return snapshot
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
		if err := defaultGovernanceTransitioner.afterCrearRegla(actor, r, id); err != nil {
			return 0, err
		}
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
		err = defaultGovernanceTransitioner.afterActualizarRegla(actor, r)
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
		err = defaultGovernanceTransitioner.afterSetReglaActiva(actor, regla, activa)
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
	if err := defaultGovernanceTransitioner.afterCrearSkill(actor, s, id); err != nil {
		return 0, err
	}
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
	if err := defaultGovernanceTransitioner.afterActualizarSkill(actor, s); err != nil {
		return err
	}
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
	return defaultGovernanceTransitioner.afterSetSkillActiva(actor, skill, activa)
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
		if err := defaultGovernanceTransitioner.afterCrearWorkflow(actor, w, id); err != nil {
			return 0, err
		}
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
		err = defaultGovernanceTransitioner.afterActualizarWorkflow(actor, w)
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
		err = defaultGovernanceTransitioner.afterSetWorkflowActivo(actor, workflow, activo)
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
