package db

import (
	"strings"
	"testing"
)

func TestResolveGovernanceCatalogYHash(t *testing.T) {
	abrirDBTemporalMemoria(t)
	const rol = "programador"
	const (
		ruleTitle    = "Server first catalogo hash"
		skillName    = "skill-catalogo-hash"
		workflowName = "workflow-catalogo-hash"
	)

	if _, err := UpsertRegla(&Regla{
		TipoAgente:  rol,
		Categoria:   "arquitectura",
		Titulo:      ruleTitle,
		Descripcion: "Todo flujo normal usa API",
		Activa:      true,
	}); err != nil {
		t.Fatalf("UpsertRegla: %v", err)
	}
	if _, err := UpsertSkill(&Skill{
		TipoAgente:  rol,
		Nombre:      skillName,
		Descripcion: "Buscar rapido",
		CuandoUsar:  "explorar codigo",
		Prioridad:   10,
		Activa:      true,
	}); err != nil {
		t.Fatalf("UpsertSkill: %v", err)
	}
	if _, err := UpsertWorkflow(&Workflow{
		TipoAgente:  rol,
		Nombre:      workflowName,
		Descripcion: "Arranque",
		Pasos:       `["leer","programar"]`,
		Activo:      true,
	}); err != nil {
		t.Fatalf("UpsertWorkflow: %v", err)
	}

	proyectoID := int64(7)
	catalogo1, err := ResolveGovernanceCatalog(rol, &proyectoID)
	if err != nil {
		t.Fatalf("ResolveGovernanceCatalog 1: %v", err)
	}
	if catalogo1 == nil {
		t.Fatalf("catalogo inesperado: %+v", catalogo1)
	}
	if catalogo1.Hash == "" {
		t.Fatalf("hash vacio: %+v", catalogo1)
	}
	if !containsRuleTitle(catalogo1.Reglas, ruleTitle) || !containsSkillName(catalogo1.Skills, skillName) || !containsWorkflowName(catalogo1.Workflows, workflowName) {
		t.Fatalf("catalogo no contiene los artefactos insertados: %+v", catalogo1)
	}

	catalogo2, err := ResolveGovernanceCatalog(rol, &proyectoID)
	if err != nil {
		t.Fatalf("ResolveGovernanceCatalog 2: %v", err)
	}
	if catalogo1.Hash != catalogo2.Hash {
		t.Fatalf("hash inestable: %s vs %s", catalogo1.Hash, catalogo2.Hash)
	}
	proyectoID2 := int64(8)
	catalogoMismoRolOtroProyecto, err := ResolveGovernanceCatalog(rol, &proyectoID2)
	if err != nil {
		t.Fatalf("ResolveGovernanceCatalog otro proyecto: %v", err)
	}
	if catalogo1.Hash != catalogoMismoRolOtroProyecto.Hash {
		t.Fatalf("el hash no deberia variar por proyecto mientras la resolucion siga siendo por rol: %s vs %s", catalogo1.Hash, catalogoMismoRolOtroProyecto.Hash)
	}

	if _, err := UpsertRegla(&Regla{
		TipoAgente:  rol,
		Categoria:   "arquitectura",
		Titulo:      ruleTitle,
		Descripcion: "Todo flujo normal usa API y no toca DB local",
		Activa:      true,
	}); err != nil {
		t.Fatalf("UpsertRegla actualizada: %v", err)
	}
	catalogo3, err := ResolveGovernanceCatalog(rol, &proyectoID)
	if err != nil {
		t.Fatalf("ResolveGovernanceCatalog 3: %v", err)
	}
	if catalogo3.Hash == catalogo1.Hash {
		t.Fatalf("el hash deberia cambiar tras modificar el catalogo: %s", catalogo3.Hash)
	}
}

func TestAppendGovernanceCatalogPayloadConservaEnvelope(t *testing.T) {
	prev := `{"project_context":{"slug":"demo"}}`
	payload := AppendGovernanceCatalogPayload(prev, map[string]any{
		"tipo_agente": "programador",
		"hash":        "abc123",
	})
	if payload == prev {
		t.Fatalf("payload sin cambios: %s", payload)
	}
	if payload == "" {
		t.Fatalf("payload vacio")
	}
	for _, token := range []string{`"governance_catalog"`, `"abc123"`, `"programador"`, `"project_context"`, `"demo"`} {
		if !strings.Contains(payload, token) {
			t.Fatalf("payload incompleto, falta %s: %s", token, payload)
		}
	}
}

func containsRuleTitle(reglas []*Regla, titulo string) bool {
	for _, regla := range reglas {
		if regla != nil && regla.Titulo == titulo {
			return true
		}
	}
	return false
}

func containsSkillName(skills []*Skill, nombre string) bool {
	for _, skill := range skills {
		if skill != nil && skill.Nombre == nombre {
			return true
		}
	}
	return false
}

func containsWorkflowName(workflows []*Workflow, nombre string) bool {
	for _, workflow := range workflows {
		if workflow != nil && workflow.Nombre == nombre {
			return true
		}
	}
	return false
}
