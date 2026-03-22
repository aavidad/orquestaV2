/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestCRUDReglas(t *testing.T) {
	abrirDBTemporalMemoria(t)
	actor := "alberto"

	id, err := CrearRegla(actor, &Regla{
		TipoAgente:  "programador",
		Categoria:   "calidad",
		Titulo:      "Regla temporal",
		Descripcion: "descripcion inicial",
	})
	if err != nil {
		t.Fatalf("CrearRegla: %v", err)
	}

	r, err := GetRegla(id)
	if err != nil {
		t.Fatalf("GetRegla: %v", err)
	}
	if !r.Activa {
		t.Fatalf("la regla deberia nacer activa")
	}

	r.Categoria = "arquitectura"
	r.Titulo = "Regla actualizada"
	r.Descripcion = "descripcion final"
	if err := ActualizarRegla(actor, r); err != nil {
		t.Fatalf("ActualizarRegla: %v", err)
	}
	if err := SetReglaActiva(actor, id, false); err != nil {
		t.Fatalf("SetReglaActiva: %v", err)
	}

	versiones, err := ListarVersionesRegla(id)
	if err != nil {
		t.Fatalf("ListarVersionesRegla: %v", err)
	}
	if len(versiones) != 3 {
		t.Fatalf("se esperaban 3 versiones de regla, got %d", len(versiones))
	}
	if versiones[0].Accion != "crear" || versiones[1].Accion != "editar" || versiones[2].Accion != "desactivar" {
		t.Fatalf("acciones de version inesperadas: %s / %s / %s", versiones[0].Accion, versiones[1].Accion, versiones[2].Accion)
	}

	activos, err := ListarReglas("programador", false)
	if err != nil {
		t.Fatalf("ListarReglas activos: %v", err)
	}
	for _, item := range activos {
		if item.ID == id {
			t.Fatalf("la regla desactivada no deberia salir en activos")
		}
	}

	todas, err := ListarReglas("programador", true)
	if err != nil {
		t.Fatalf("ListarReglas todas: %v", err)
	}
	var encontrada bool
	for _, item := range todas {
		if item.ID == id {
			encontrada = true
			if item.Activa {
				t.Fatalf("la regla deberia estar inactiva")
			}
		}
	}
	if !encontrada {
		t.Fatalf("la regla actualizada no aparece en el listado total")
	}

	if _, err := CrearRegla("codex1", &Regla{
		TipoAgente:  "documentador",
		Categoria:   "calidad",
		Titulo:      "Regla denegada",
		Descripcion: "no deberia poder crearla un programador",
	}); err == nil {
		t.Fatalf("se esperaba denegacion de permisos")
	}
}

func TestCRUDSkillsYWorkflows(t *testing.T) {
	abrirDBTemporalMemoria(t)
	actor := "alberto"

	skillID, err := CrearSkill(actor, &Skill{
		TipoAgente:  "programador",
		Nombre:      "skill-temporal",
		Descripcion: "descripcion skill",
		CuandoUsar:  "cuando haga falta",
	})
	if err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	skill, err := GetSkill(skillID)
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	skill.Descripcion = "descripcion skill editada"
	skill.CuandoUsar = "siempre que aplique"
	if err := ActualizarSkill(actor, skill); err != nil {
		t.Fatalf("ActualizarSkill: %v", err)
	}
	if err := SetSkillActivo(actor, skillID, false); err != nil {
		t.Fatalf("SetSkillActivo: %v", err)
	}

	versionesSkill, err := ListarVersionesSkill(skillID)
	if err != nil {
		t.Fatalf("ListarVersionesSkill: %v", err)
	}
	if len(versionesSkill) != 3 {
		t.Fatalf("se esperaban 3 versiones de skill, got %d", len(versionesSkill))
	}

	skills, err := ListarSkills("programador", true)
	if err != nil {
		t.Fatalf("ListarSkills: %v", err)
	}
	if len(skills) == 0 {
		t.Fatalf("se esperaba al menos un skill")
	}

	workflowID, err := CrearWorkflow(actor, &Workflow{
		TipoAgente:  "programador",
		Nombre:      "workflow-temporal",
		Descripcion: "workflow de prueba",
		Pasos:       `["paso 1","paso 2"]`,
	})
	if err != nil {
		t.Fatalf("CrearWorkflow: %v", err)
	}

	workflow, err := GetWorkflowByID(workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowByID: %v", err)
	}
	workflow.Descripcion = "workflow editado"
	workflow.Pasos = `["paso A"]`
	if err := ActualizarWorkflow(actor, workflow); err != nil {
		t.Fatalf("ActualizarWorkflow: %v", err)
	}
	if err := SetWorkflowActivo(actor, workflowID, false); err != nil {
		t.Fatalf("SetWorkflowActivo: %v", err)
	}

	versionesWorkflow, err := ListarVersionesWorkflow(workflowID)
	if err != nil {
		t.Fatalf("ListarVersionesWorkflow: %v", err)
	}
	if len(versionesWorkflow) != 3 {
		t.Fatalf("se esperaban 3 versiones de workflow, got %d", len(versionesWorkflow))
	}

	workflows, err := ListarWorkflows("programador", true)
	if err != nil {
		t.Fatalf("ListarWorkflows: %v", err)
	}
	var encontrado bool
	for _, item := range workflows {
		if item.ID == workflowID {
			encontrado = true
			if item.Activo {
				t.Fatalf("el workflow deberia quedar inactivo")
			}
			if item.Pasos != `["paso A"]` {
				t.Fatalf("pasos inesperados: %s", item.Pasos)
			}
		}
	}
	if !encontrado {
		t.Fatalf("workflow no encontrado en listado total")
	}
}

func TestGuardarPermisoEdicionCatalogo(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := GuardarPermisoEdicionCatalogo("alberto", &PermisoEdicionCatalogo{
		Entidad:        "reglas",
		Rol:            "programador",
		Alcance:        "todos",
		PuedeCrear:     true,
		PuedeEditar:    true,
		PuedeActivar:   false,
		PuedeVersionar: true,
	}); err != nil {
		t.Fatalf("GuardarPermisoEdicionCatalogo: %v", err)
	}

	permisos, err := ListarPermisosEdicionCatalogo("reglas")
	if err != nil {
		t.Fatalf("ListarPermisosEdicionCatalogo: %v", err)
	}
	var encontrado bool
	for _, p := range permisos {
		if p.Entidad == "reglas" && p.Rol == "programador" {
			encontrado = true
			if p.Alcance != "todos" || !p.PuedeVersionar || p.PuedeActivar {
				t.Fatalf("permiso inesperado: %#v", p)
			}
		}
	}
	if !encontrado {
		t.Fatalf("no se encontró el permiso actualizado")
	}
}
