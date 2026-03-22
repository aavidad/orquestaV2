/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestRegistrarYActualizarDecisionProyecto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	res, err := DB.Exec(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, propuesto_por, distribuidor)
		VALUES ('OP-900', 'Prueba decision', 'decision de prueba', 'implementacion', 'alberto', 'claude')`)
	if err != nil {
		t.Fatalf("insert propuesta: %v", err)
	}
	if id, _ := res.LastInsertId(); id == 0 {
		t.Fatalf("id de propuesta inesperado")
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Relacionar decision",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}

	id, err := RegistrarDecisionProyecto(&DecisionProyecto{
		Proyecto:                "orquestador",
		Titulo:                  "Elegir arquitectura",
		SolucionElegida:         "hexagonal ligera",
		Motivo:                  "reduce acoplamiento",
		AlternativasDescartadas: "seguir en db/",
		Impacto:                 "alto",
		PropuestaCodigo:         "OP-900",
		TareaID:                 &tareaID,
		RegistradoPor:           "Codex3",
	})
	if err != nil {
		t.Fatalf("RegistrarDecisionProyecto: %v", err)
	}

	d, err := GetDecisionProyecto(id)
	if err != nil {
		t.Fatalf("GetDecisionProyecto: %v", err)
	}
	d.Motivo = "reduce acoplamiento y deuda"
	if err := ActualizarDecisionProyecto(d); err != nil {
		t.Fatalf("ActualizarDecisionProyecto: %v", err)
	}

	lista, err := ListarDecisionesProyecto("orquestador")
	if err != nil {
		t.Fatalf("ListarDecisionesProyecto: %v", err)
	}
	if len(lista) != 1 || lista[0].Motivo != "reduce acoplamiento y deuda" {
		t.Fatalf("lista inesperada: %+v", lista)
	}
}

func TestRegistrarYActualizarDocumentoExterno(t *testing.T) {
	abrirDBTemporalMemoria(t)

	id, err := RegistrarDocumentoExterno(&DocumentoExterno{
		Proyecto:      "orquestador",
		TipoDocumento: "manual",
		Ruta:          "/srv/docs/manual.md",
		Resumen:       "Manual externo",
		Estado:        "vigente",
		RegistradoPor: "Codex3",
	})
	if err != nil {
		t.Fatalf("RegistrarDocumentoExterno: %v", err)
	}

	doc, err := GetDocumentoExterno(id)
	if err != nil {
		t.Fatalf("GetDocumentoExterno: %v", err)
	}
	doc.Estado = "archivado"
	if err := ActualizarDocumentoExterno(doc); err != nil {
		t.Fatalf("ActualizarDocumentoExterno: %v", err)
	}

	lista, err := ListarDocumentosExternos("orquestador")
	if err != nil {
		t.Fatalf("ListarDocumentosExternos: %v", err)
	}
	if len(lista) != 1 || lista[0].Estado != "archivado" {
		t.Fatalf("lista inesperada: %+v", lista)
	}
}
