package db

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardarGovernanceOverrideAutocompletaTipoAgenteYHaceUpsert(t *testing.T) {
	abrirDBTemporalMemoria(t)

	reglaID := mustUpsertGovernanceRuleTest(t, &Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "regla-override-upsert",
		Descripcion: "Regla base para override",
		Activa:      true,
	})

	id, err := GuardarGovernanceOverride("codex-test", &GovernanceOverride{
		ScopeTipo: " proyecto ",
		ScopeRef:  " demo-proyecto ",
		Entidad:   " regla ",
		EntidadID: reglaID,
		Accion:    " enable ",
	})
	if err != nil {
		t.Fatalf("GuardarGovernanceOverride insert: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}

	items, err := ListarGovernanceOverrides(GovernanceScopeProyecto, "demo-proyecto", "programador", GovernanceEntityRegla)
	if err != nil {
		t.Fatalf("ListarGovernanceOverrides insert: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("overrides inesperados tras insert: %+v", items)
	}
	if items[0].ID != id {
		t.Fatalf("id listado inesperado: got=%d want=%d", items[0].ID, id)
	}
	if items[0].TipoAgente != "programador" {
		t.Fatalf("tipo_agente no autocompletado: %+v", items[0])
	}
	if items[0].ScopeTipo != GovernanceScopeProyecto || items[0].ScopeRef != "demo-proyecto" {
		t.Fatalf("scope no normalizado: %+v", items[0])
	}
	if items[0].Entidad != GovernanceEntityRegla || items[0].Accion != GovernanceActionEnable {
		t.Fatalf("override insertado inesperado: %+v", items[0])
	}

	idActualizado, err := GuardarGovernanceOverride("codex-test", &GovernanceOverride{
		TipoAgente: "programador",
		ScopeTipo:  GovernanceScopeProyecto,
		ScopeRef:   "demo-proyecto",
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     GovernanceActionDisable,
	})
	if err != nil {
		t.Fatalf("GuardarGovernanceOverride update: %v", err)
	}
	if idActualizado != id {
		t.Fatalf("el upsert deberia conservar el id: got=%d want=%d", idActualizado, id)
	}

	items, err = ListarGovernanceOverrides(GovernanceScopeProyecto, "demo-proyecto", "programador", GovernanceEntityRegla)
	if err != nil {
		t.Fatalf("ListarGovernanceOverrides update: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("overrides inesperados tras update: %+v", items)
	}
	if items[0].Accion != GovernanceActionDisable {
		t.Fatalf("accion no actualizada: %+v", items[0])
	}
}

func TestResolveGovernanceCatalogForContextRespetaPrecedenciaRolProyectoAgente(t *testing.T) {
	abrirDBTemporalMemoria(t)

	const (
		rol    = "programador"
		agente = "codex-precedencia"
	)

	proyectoID := mustUpsertGovernanceProjectTest(t, "gobernanza-precedencia")
	reglaBaseID := mustUpsertGovernanceRuleTest(t, &Regla{
		TipoAgente:  rol,
		Categoria:   "arquitectura",
		Titulo:      "regla-base-precedencia",
		Descripcion: "Activa por rol",
		Activa:      true,
	})
	skillCapasID := mustUpsertGovernanceSkillTest(t, &Skill{
		TipoAgente:  rol,
		Nombre:      "skill-capas-precedencia",
		Descripcion: "Solo aparece si un override la habilita",
		CuandoUsar:  "probar precedencia",
		Prioridad:   20,
		Activa:      false,
	})
	workflowProyectoID := mustUpsertGovernanceWorkflowTest(t, &Workflow{
		TipoAgente:  rol,
		Nombre:      "workflow-proyecto-precedencia",
		Descripcion: "Solo aparece si proyecto lo habilita",
		Pasos:       `["uno","dos"]`,
		Activo:      false,
	})

	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeProyecto,
		ScopeRef:   "gobernanza-precedencia",
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaBaseID,
		Accion:     GovernanceActionDisable,
		TipoAgente: rol,
	})
	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeProyecto,
		ScopeRef:   "gobernanza-precedencia",
		Entidad:    GovernanceEntitySkill,
		EntidadID:  skillCapasID,
		Accion:     GovernanceActionEnable,
		TipoAgente: rol,
	})
	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeProyecto,
		ScopeRef:   "gobernanza-precedencia",
		Entidad:    GovernanceEntityWorkflow,
		EntidadID:  workflowProyectoID,
		Accion:     GovernanceActionEnable,
		TipoAgente: rol,
	})
	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeAgente,
		ScopeRef:   agente,
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaBaseID,
		Accion:     GovernanceActionEnable,
		TipoAgente: rol,
	})
	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeAgente,
		ScopeRef:   agente,
		Entidad:    GovernanceEntitySkill,
		EntidadID:  skillCapasID,
		Accion:     GovernanceActionDisable,
		TipoAgente: rol,
	})

	catalogo, err := ResolveGovernanceCatalogForContext(rol, &proyectoID, agente)
	if err != nil {
		t.Fatalf("ResolveGovernanceCatalogForContext: %v", err)
	}
	if catalogo == nil {
		t.Fatalf("catalogo nil")
	}
	if catalogo.ResolucionActual != "rol+proyecto+agente" {
		t.Fatalf("resolucion inesperada: %q", catalogo.ResolucionActual)
	}
	if !governanceCatalogHasRuleTitleTest(catalogo, "regla-base-precedencia") {
		t.Fatalf("la capa agente deberia reactivar la regla base: %+v", catalogo.Reglas)
	}
	if governanceCatalogHasSkillNameTest(catalogo, "skill-capas-precedencia") {
		t.Fatalf("la capa agente deberia prevalecer sobre el enable de proyecto: %+v", catalogo.Skills)
	}
	if !governanceCatalogHasWorkflowNameTest(catalogo, "workflow-proyecto-precedencia") {
		t.Fatalf("el enable de proyecto deberia mantenerse: %+v", catalogo.Workflows)
	}
}

func TestResolveGovernanceWorkflowForContextUsaElCatalogoEfectivo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	const (
		rol    = "programador"
		agente = "codex-workflow-ctx"
		nombre = "workflow-contexto-efectivo"
	)

	proyectoID := mustUpsertGovernanceProjectTest(t, "gobernanza-workflow")
	workflowID := mustUpsertGovernanceWorkflowTest(t, &Workflow{
		TipoAgente:  rol,
		Nombre:      nombre,
		Descripcion: "Workflow resuelto por contexto",
		Pasos:       `["analizar","aplicar"]`,
		Activo:      false,
	})

	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeProyecto,
		ScopeRef:   "gobernanza-workflow",
		Entidad:    GovernanceEntityWorkflow,
		EntidadID:  workflowID,
		Accion:     GovernanceActionEnable,
		TipoAgente: rol,
	})

	workflow, err := ResolveGovernanceWorkflowForContext(rol, &proyectoID, agente, nombre)
	if err != nil {
		t.Fatalf("ResolveGovernanceWorkflowForContext enable proyecto: %v", err)
	}
	if workflow == nil || workflow.ID != workflowID {
		t.Fatalf("workflow inesperado tras enable de proyecto: %+v", workflow)
	}

	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeAgente,
		ScopeRef:   agente,
		Entidad:    GovernanceEntityWorkflow,
		EntidadID:  workflowID,
		Accion:     GovernanceActionDisable,
		TipoAgente: rol,
	})

	workflow, err = ResolveGovernanceWorkflowForContext(rol, &proyectoID, agente, nombre)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("se esperaba sql.ErrNoRows tras disable de agente; got workflow=%+v err=%v", workflow, err)
	}
	if workflow != nil {
		t.Fatalf("no deberia resolverse workflow tras disable de agente: %+v", workflow)
	}
}

func TestBuildGovernanceContextSummaryForContextIncluyeOverridesAgente(t *testing.T) {
	abrirDBTemporalMemoria(t)

	const (
		rol    = "programador"
		agente = "codex-summary-ctx"
	)

	proyectoID := mustUpsertGovernanceProjectTest(t, "gobernanza-summary")
	reglaID := mustUpsertGovernanceRuleTest(t, &Regla{
		TipoAgente:  rol,
		Categoria:   "arquitectura",
		Titulo:      "regla-summary-ctx",
		Descripcion: "override agent-aware",
		Activa:      true,
	})
	mustGuardarGovernanceOverrideTest(t, &GovernanceOverride{
		ScopeTipo:  GovernanceScopeAgente,
		ScopeRef:   agente,
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     GovernanceActionDisable,
		TipoAgente: rol,
	})

	catalogo, err := ResolveGovernanceCatalogForContext(rol, &proyectoID, agente)
	if err != nil {
		t.Fatalf("ResolveGovernanceCatalogForContext: %v", err)
	}
	contexto, resumen := BuildGovernanceContextSummaryForContext(rol, &proyectoID, agente)
	if len(contexto) == 0 {
		t.Fatalf("contexto vacio")
	}
	if got := contexto["resolucion_actual"]; got != "rol+agente" {
		t.Fatalf("resolucion inesperada: %#v", got)
	}
	if got := contexto["reglas"]; got != len(catalogo.Reglas) {
		t.Fatalf("conteo de reglas inesperado: got=%#v want=%d", got, len(catalogo.Reglas))
	}
	if governanceCatalogHasRuleTitleTest(catalogo, "regla-summary-ctx") {
		t.Fatalf("la regla deshabilitada por agente no deberia seguir en el catalogo: %+v", catalogo.Reglas)
	}
	if !strings.Contains(resumen, fmt.Sprintf("%d reglas", len(catalogo.Reglas))) {
		t.Fatalf("resumen inesperado: %s", resumen)
	}
}

func TestListarGovernanceOverridesSinTablaDevuelveVacio(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if _, err := DB.Exec(`DROP TABLE IF EXISTS governance_overrides`); err != nil {
		t.Fatalf("drop governance_overrides: %v", err)
	}

	items, err := ListarGovernanceOverrides(GovernanceScopeProyecto, "demo", "programador", GovernanceEntityRegla)
	if err != nil {
		t.Fatalf("ListarGovernanceOverrides sin tabla: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("sin tabla deberia devolver vacio: %+v", items)
	}
}

func mustGuardarGovernanceOverrideTest(t *testing.T, item *GovernanceOverride) int64 {
	t.Helper()
	id, err := GuardarGovernanceOverride("codex-test", item)
	if err != nil {
		t.Fatalf("GuardarGovernanceOverride(%+v): %v", item, err)
	}
	if id == 0 {
		t.Fatalf("GuardarGovernanceOverride devolvio id invalido para %+v", item)
	}
	return id
}

func mustUpsertGovernanceProjectTest(t *testing.T, slug string) int64 {
	t.Helper()
	id, err := UpsertProyecto(&Proyecto{
		Slug:    slug,
		Nombre:  slug,
		RutaAbs: filepath.Join(t.TempDir(), slug),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto(%s): %v", slug, err)
	}
	return id
}

func mustUpsertGovernanceRuleTest(t *testing.T, item *Regla) int64 {
	t.Helper()
	id, err := UpsertRegla(item)
	if err != nil {
		t.Fatalf("UpsertRegla(%+v): %v", item, err)
	}
	if id <= 0 {
		t.Fatalf("UpsertRegla devolvio id invalido para %+v", item)
	}
	return id
}

func mustUpsertGovernanceSkillTest(t *testing.T, item *Skill) int64 {
	t.Helper()
	id, err := UpsertSkill(item)
	if err != nil {
		t.Fatalf("UpsertSkill(%+v): %v", item, err)
	}
	if id <= 0 {
		t.Fatalf("UpsertSkill devolvio id invalido para %+v", item)
	}
	return id
}

func mustUpsertGovernanceWorkflowTest(t *testing.T, item *Workflow) int64 {
	t.Helper()
	id, err := UpsertWorkflow(item)
	if err != nil {
		t.Fatalf("UpsertWorkflow(%+v): %v", item, err)
	}
	if id <= 0 {
		t.Fatalf("UpsertWorkflow devolvio id invalido para %+v", item)
	}
	return id
}

func governanceCatalogHasRuleTitleTest(catalogo *GovernanceCatalog, titulo string) bool {
	for _, regla := range catalogo.Reglas {
		if regla != nil && regla.Titulo == titulo {
			return true
		}
	}
	return false
}

func governanceCatalogHasSkillNameTest(catalogo *GovernanceCatalog, nombre string) bool {
	for _, skill := range catalogo.Skills {
		if skill != nil && skill.Nombre == nombre {
			return true
		}
	}
	return false
}

func governanceCatalogHasWorkflowNameTest(catalogo *GovernanceCatalog, nombre string) bool {
	for _, workflow := range catalogo.Workflows {
		if workflow != nil && workflow.Nombre == nombre {
			return true
		}
	}
	return false
}
