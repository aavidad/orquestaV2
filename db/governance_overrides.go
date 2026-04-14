package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/gobernanzapolicy"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	GovernanceScopeProyecto = gobernanzapolicy.ScopeProject
	GovernanceScopeAgente   = gobernanzapolicy.ScopeAgent

	GovernanceEntityRegla    = gobernanzapolicy.EntityRule
	GovernanceEntitySkill    = gobernanzapolicy.EntitySkill
	GovernanceEntityWorkflow = gobernanzapolicy.EntityWorkflow

	GovernanceActionEnable  = gobernanzapolicy.ActionEnable
	GovernanceActionDisable = gobernanzapolicy.ActionDisable
)

type GovernanceOverride struct {
	ID         int64     `json:"id"`
	TipoAgente string    `json:"tipo_agente"`
	ScopeTipo  string    `json:"scope_tipo"`
	ScopeRef   string    `json:"scope_ref"`
	Entidad    string    `json:"entidad"`
	EntidadID  int64     `json:"entidad_id"`
	Accion     string    `json:"accion"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func GuardarGovernanceOverride(actor string, item *GovernanceOverride) (int64, error) {
	if err := ensureGovernanceOverrideSchema(); err != nil {
		return 0, err
	}
	if item == nil {
		return 0, fmt.Errorf("override obligatorio")
	}
	spec, err := gobernanzapolicy.NormalizeOverrideSpec(item.ScopeTipo, item.Entidad, item.Accion)
	if err != nil {
		return 0, err
	}
	item.ScopeTipo = spec.ScopeType
	item.ScopeRef = strings.TrimSpace(item.ScopeRef)
	item.Entidad = spec.Entity
	item.Accion = spec.Action
	item.TipoAgente = strings.TrimSpace(item.TipoAgente)
	if item.ScopeRef == "" {
		return 0, fmt.Errorf("scope_ref obligatorio")
	}
	if item.EntidadID <= 0 {
		return 0, fmt.Errorf("entidad_id obligatorio")
	}

	tipoAgenteReal, err := governanceEntidadTipoAgente(item.Entidad, item.EntidadID)
	if err != nil {
		return 0, err
	}
	if item.TipoAgente == "" {
		item.TipoAgente = tipoAgenteReal
	}
	if item.TipoAgente != tipoAgenteReal {
		return 0, fmt.Errorf("tipo_agente no coincide con la entidad")
	}

	_, err = DB.Exec(`
		INSERT INTO governance_overrides (
			tipo_agente, scope_tipo, scope_ref, entidad, entidad_id, accion
		) VALUES (?,?,?,?,?,?)
		ON CONFLICT(scope_tipo, scope_ref, entidad, entidad_id) DO UPDATE SET
			tipo_agente=excluded.tipo_agente,
			accion=excluded.accion,
			updated_at=CURRENT_TIMESTAMP`,
		item.TipoAgente,
		item.ScopeTipo,
		item.ScopeRef,
		item.Entidad,
		item.EntidadID,
		item.Accion,
	)
	if err != nil {
		return 0, err
	}

	var id int64
	if err := DB.QueryRow(`
		SELECT id
		FROM governance_overrides
		WHERE scope_tipo = ? AND scope_ref = ? AND entidad = ? AND entidad_id = ?`,
		item.ScopeTipo, item.ScopeRef, item.Entidad, item.EntidadID,
	).Scan(&id); err != nil {
		return 0, err
	}
	Audit(actor, "guardar_governance_override", "governance_override", id,
		fmt.Sprintf("tipo_agente=%s scope=%s:%s entidad=%s/%d accion=%s", item.TipoAgente, item.ScopeTipo, item.ScopeRef, item.Entidad, item.EntidadID, item.Accion))
	notificarRefreshGobernanzaScope(actor, item.TipoAgente, item.ScopeTipo, item.ScopeRef, "governance_override", map[string]any{
		"override_id": id,
		"entidad":     item.Entidad,
		"entidad_id":  item.EntidadID,
		"accion":      item.Accion,
	})
	return id, nil
}

func ListarGovernanceOverrides(scopeTipo, scopeRef, tipoAgente, entidad string) ([]*GovernanceOverride, error) {
	exists, err := governanceOverrideSchemaExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	q := `
		SELECT id, tipo_agente, scope_tipo, scope_ref, entidad, entidad_id, accion, created_at, updated_at
		FROM governance_overrides
		WHERE 1=1`
	args := []any{}
	if scopeTipo = strings.TrimSpace(scopeTipo); scopeTipo != "" {
		q += ` AND scope_tipo = ?`
		args = append(args, scopeTipo)
	}
	if scopeRef = strings.TrimSpace(scopeRef); scopeRef != "" {
		q += ` AND scope_ref = ?`
		args = append(args, scopeRef)
	}
	if tipoAgente = strings.TrimSpace(tipoAgente); tipoAgente != "" {
		q += ` AND tipo_agente = ?`
		args = append(args, tipoAgente)
	}
	if entidad = strings.TrimSpace(entidad); entidad != "" {
		q += ` AND entidad = ?`
		args = append(args, entidad)
	}
	q += ` ORDER BY
		CASE scope_tipo WHEN 'proyecto' THEN 1 WHEN 'agente' THEN 2 ELSE 99 END,
		scope_ref, entidad, entidad_id, id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*GovernanceOverride
	for rows.Next() {
		item := &GovernanceOverride{}
		if err := rows.Scan(
			&item.ID,
			&item.TipoAgente,
			&item.ScopeTipo,
			&item.ScopeRef,
			&item.Entidad,
			&item.EntidadID,
			&item.Accion,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func ResolveGovernanceCatalogForContext(tipoAgente string, proyectoID *int64, agente string) (*GovernanceCatalog, error) {
	tipoAgente = strings.TrimSpace(tipoAgente)
	baseReglas, err := GetReglasAgente(tipoAgente)
	if err != nil {
		return nil, err
	}
	baseSkills, err := GetSkillsAgente(tipoAgente)
	if err != nil {
		return nil, err
	}
	baseWorkflows, err := GetWorkflowsAgente(tipoAgente)
	if err != nil {
		return nil, err
	}

	reglasMap := map[int64]*Regla{}
	for _, item := range baseReglas {
		if item != nil {
			reglasMap[item.ID] = item
		}
	}
	skillsMap := map[int64]*Skill{}
	for _, item := range baseSkills {
		if item != nil {
			skillsMap[item.ID] = item
		}
	}
	workflowsMap := map[int64]*Workflow{}
	for _, item := range baseWorkflows {
		if item != nil {
			workflowsMap[item.ID] = item
		}
	}

	projectScopeRef, agentScopeRef, err := governanceLayerRefs(proyectoID, agente)
	if err != nil {
		return nil, err
	}
	layers, resolucion := gobernanzapolicy.ResolveOverrideLayers(projectScopeRef, agentScopeRef)
	for _, layer := range layers {
		overrides, err := ListarGovernanceOverrides(layer.ScopeType, layer.ScopeRef, tipoAgente, "")
		if err != nil {
			return nil, err
		}
		if len(overrides) == 0 {
			continue
		}
		resolucion = gobernanzapolicy.AppendResolutionScope(resolucion, layer.ScopeType)
		for _, item := range overrides {
			if err := applyGovernanceOverride(item, reglasMap, skillsMap, workflowsMap); err != nil {
				return nil, err
			}
		}
	}

	catalogo := &GovernanceCatalog{
		TipoAgente:       tipoAgente,
		ProyectoID:       proyectoID,
		ResolucionActual: resolucion,
		Reglas:           collectGovernanceRules(reglasMap),
		Skills:           collectGovernanceSkills(skillsMap),
		Workflows:        collectGovernanceWorkflows(workflowsMap),
	}
	catalogo.Hash = governanceCatalogHash(catalogo)
	return catalogo, nil
}

func ResolveGovernanceWorkflowForContext(tipoAgente string, proyectoID *int64, agente, nombre string) (*Workflow, error) {
	catalogo, err := ResolveGovernanceCatalogForContext(tipoAgente, proyectoID, agente)
	if err != nil || catalogo == nil {
		return nil, err
	}
	nombre = strings.TrimSpace(nombre)
	for _, workflow := range catalogo.Workflows {
		if workflow != nil && strings.TrimSpace(workflow.Nombre) == nombre {
			return workflow, nil
		}
	}
	return nil, sql.ErrNoRows
}

func ensureGovernanceOverrideSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS governance_overrides (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			tipo_agente TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
			scope_tipo  TEXT NOT NULL CHECK (scope_tipo IN ('proyecto','agente')),
			scope_ref   TEXT NOT NULL,
			entidad     TEXT NOT NULL CHECK (entidad IN ('regla','skill','workflow')),
			entidad_id  INTEGER NOT NULL,
			accion      TEXT NOT NULL CHECK (accion IN ('enable','disable')),
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(scope_tipo, scope_ref, entidad, entidad_id)
		)`)
	return err
}

func governanceOverrideSchemaExists() (bool, error) {
	return SchemaObjectExists("table", "governance_overrides")
}

func governanceLayerRefs(proyectoID *int64, agente string) (string, string, error) {
	projectScopeRef := ""
	if proyectoID != nil && *proyectoID > 0 {
		proyecto, err := GetProyecto(strconv.FormatInt(*proyectoID, 10))
		if err != nil && err != sql.ErrNoRows {
			return "", "", err
		}
		if proyecto != nil && strings.TrimSpace(proyecto.Slug) != "" {
			projectScopeRef = strings.TrimSpace(proyecto.Slug)
		}
	}
	return projectScopeRef, strings.TrimSpace(agente), nil
}

func applyGovernanceOverride(item *GovernanceOverride, reglas map[int64]*Regla, skills map[int64]*Skill, workflows map[int64]*Workflow) error {
	if item == nil {
		return nil
	}
	switch item.Entidad {
	case GovernanceEntityRegla:
		if item.Accion == GovernanceActionDisable {
			delete(reglas, item.EntidadID)
			return nil
		}
		regla, err := GetRegla(item.EntidadID)
		if err != nil {
			return err
		}
		if regla != nil && regla.TipoAgente == item.TipoAgente {
			reglas[regla.ID] = regla
		}
	case GovernanceEntitySkill:
		if item.Accion == GovernanceActionDisable {
			delete(skills, item.EntidadID)
			return nil
		}
		skill, err := GetSkill(item.EntidadID)
		if err != nil {
			return err
		}
		if skill != nil && skill.TipoAgente == item.TipoAgente {
			skills[skill.ID] = skill
		}
	case GovernanceEntityWorkflow:
		if item.Accion == GovernanceActionDisable {
			delete(workflows, item.EntidadID)
			return nil
		}
		workflow, err := GetWorkflowByID(item.EntidadID)
		if err != nil {
			return err
		}
		if workflow != nil && workflow.TipoAgente == item.TipoAgente {
			workflows[workflow.ID] = workflow
		}
	}
	return nil
}

func collectGovernanceRules(items map[int64]*Regla) []*Regla {
	list := make([]*Regla, 0, len(items))
	for _, item := range items {
		if item != nil {
			list = append(list, item)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Categoria != list[j].Categoria {
			return list[i].Categoria < list[j].Categoria
		}
		return list[i].ID < list[j].ID
	})
	return list
}

func collectGovernanceSkills(items map[int64]*Skill) []*Skill {
	list := make([]*Skill, 0, len(items))
	for _, item := range items {
		if item != nil {
			list = append(list, item)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Prioridad != list[j].Prioridad {
			return list[i].Prioridad < list[j].Prioridad
		}
		if list[i].Escenario != list[j].Escenario {
			return list[i].Escenario < list[j].Escenario
		}
		return list[i].Nombre < list[j].Nombre
	})
	return list
}

func collectGovernanceWorkflows(items map[int64]*Workflow) []*Workflow {
	list := make([]*Workflow, 0, len(items))
	for _, item := range items {
		if item != nil {
			list = append(list, item)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Nombre < list[j].Nombre
	})
	return list
}

func governanceEntidadTipoAgente(entidad string, entidadID int64) (string, error) {
	switch strings.TrimSpace(entidad) {
	case GovernanceEntityRegla:
		item, err := GetRegla(entidadID)
		if err != nil {
			return "", err
		}
		return item.TipoAgente, nil
	case GovernanceEntitySkill:
		item, err := GetSkill(entidadID)
		if err != nil {
			return "", err
		}
		return item.TipoAgente, nil
	case GovernanceEntityWorkflow:
		item, err := GetWorkflowByID(entidadID)
		if err != nil {
			return "", err
		}
		return item.TipoAgente, nil
	default:
		return "", fmt.Errorf("entidad invalida: %s", entidad)
	}
}

func notificarRefreshGobernanzaScope(actor, tipoAgente, scopeTipo, scopeRef, motivo string, extra map[string]any) {
	if strings.TrimSpace(tipoAgente) == "" {
		return
	}
	sesiones, err := ListarSesionesActivasOperativas()
	if err != nil {
		Audit(actor, "governance_refresh_error", "governance_override", 0, err.Error())
		return
	}
	seen := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		agente, err := GetAgente(sesion.Agente)
		if err != nil || agente == nil || agente.Rol != tipoAgente {
			continue
		}
		if !sesionCoincideGovernanceScope(sesion, scopeTipo, scopeRef) {
			continue
		}
		if _, ok := seen[sesion.Agente]; ok {
			continue
		}
		seen[sesion.Agente] = struct{}{}

		catalogo, err := ResolveGovernanceCatalogForContext(tipoAgente, sesion.ProyectoID, sesion.Agente)
		if err != nil || catalogo == nil {
			if err != nil {
				Audit(actor, "governance_refresh_error", "governance_override", 0, err.Error())
			}
			continue
		}
		payloadMap := map[string]any{
			"tipo_agente": tipoAgente,
			"motivo":      motivo,
			"hash":        catalogo.Hash,
			"reglas":      len(catalogo.Reglas),
			"skills":      len(catalogo.Skills),
			"workflows":   len(catalogo.Workflows),
			"scope_tipo":  scopeTipo,
			"scope_ref":   scopeRef,
		}
		for key, value := range extra {
			payloadMap[key] = value
		}
		payload, _ := json.Marshal(payloadMap)
		if _, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
			FromAgente:  actor,
			ToAgente:    sesion.Agente,
			ProyectoID:  sesion.ProyectoID,
			Kind:        MailboxKindGovernanceRefresh,
			PayloadJSON: string(payload),
		}); err != nil {
			Audit(actor, "governance_refresh_error", "governance_override", 0, err.Error())
		}
	}
}

func sesionCoincideGovernanceScope(sesion *Sesion, scopeTipo, scopeRef string) bool {
	if sesion == nil {
		return false
	}
	switch strings.TrimSpace(scopeTipo) {
	case GovernanceScopeAgente:
		return strings.TrimSpace(sesion.Agente) == strings.TrimSpace(scopeRef)
	case GovernanceScopeProyecto:
		if sesion.ProyectoID == nil || *sesion.ProyectoID == 0 {
			return false
		}
		proyecto, err := GetProyecto(strconv.FormatInt(*sesion.ProyectoID, 10))
		if err != nil || proyecto == nil {
			return false
		}
		return strings.TrimSpace(proyecto.Slug) == strings.TrimSpace(scopeRef)
	default:
		return false
	}
}
