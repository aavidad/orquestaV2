/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestListarHistorialVotacionesProyecto(t *testing.T) {
	abrirDBTemporalPropuestas(t)

	if err := RetirarAgente("claude"); err != nil {
		t.Fatalf("RetirarAgente claude: %v", err)
	}
	if err := RetirarAgente("antigravity"); err != nil {
		t.Fatalf("RetirarAgente antigravity: %v", err)
	}
	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex1: %v", err)
	}
	if err := RegistrarAgente("codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex2: %v", err)
	}

	id, err := CrearPropuesta(&Propuesta{
		Codigo:       "OP-903",
		Titulo:       "Historial por proyecto",
		Descripcion:  "Debe quedar asociado al proyecto",
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
		ProyectoSlug: "orquestador",
	})
	if err != nil {
		t.Fatalf("CrearPropuesta: %v", err)
	}

	if _, err := Votar(id, "codex1", VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("Votar codex1: %v", err)
	}
	if _, err := Votar(id, "codex2", VotoDesacuerdo, "falta trazabilidad"); err != nil {
		t.Fatalf("Votar codex2: %v", err)
	}

	propuestas, err := ListarPropuestasProyecto("orquestador", nil)
	if err != nil {
		t.Fatalf("ListarPropuestasProyecto: %v", err)
	}
	if len(propuestas) != 1 {
		t.Fatalf("se esperaba 1 propuesta para el proyecto, got %d", len(propuestas))
	}
	if propuestas[0].ProyectoSlug != "orquestador" {
		t.Fatalf("slug de proyecto inesperado: %+v", propuestas[0])
	}
	if propuestas[0].ProyectoID == nil {
		t.Fatalf("se esperaba proyecto_id en la propuesta")
	}

	historial, err := ListarHistorialVotacionesProyecto("orquestador")
	if err != nil {
		t.Fatalf("ListarHistorialVotacionesProyecto: %v", err)
	}
	if len(historial) != 1 {
		t.Fatalf("se esperaba 1 entrada de historial, got %d", len(historial))
	}
	item := historial[0]
	if item.Propuesta.Codigo != "OP-903" {
		t.Fatalf("codigo inesperado: %+v", item.Propuesta)
	}
	if item.Acuerdo != 1 || item.Desacuerdo != 1 || item.Abstencion != 0 || item.Pendiente != 0 {
		t.Fatalf("conteos inesperados: %+v", item)
	}
	if len(item.Votos) != 2 {
		t.Fatalf("se esperaban 2 votos, got %d", len(item.Votos))
	}
}
