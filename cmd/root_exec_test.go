/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestExecuteLocalArgsReseteaFlagsEntreEjecuciones(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar bug de flags",
		ProyectoID: &proyectoID,
		Modulo:     "cli",
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "Codex1",
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	outJSON, errJSON, exitJSON := executeLocalCommandCaptured([]string{"tarea", "listar", "--json"})
	if exitJSON != 0 {
		t.Fatalf("primera ejecucion --json: exit=%d stderr=%s", exitJSON, errJSON)
	}
	if !strings.Contains(outJSON, "\"Titulo\": \"Cerrar bug de flags\"") {
		t.Fatalf("salida json inesperada: %s", outJSON)
	}

	outTSV, errTSV, exitTSV := executeLocalCommandCaptured([]string{"tarea", "listar", "--tsv"})
	if exitTSV != 0 {
		t.Fatalf("segunda ejecucion --tsv: exit=%d stderr=%s", exitTSV, errTSV)
	}
	if !strings.Contains(outTSV, "Cerrar bug de flags") {
		t.Fatalf("salida tsv inesperada: %s", outTSV)
	}
	if strings.Contains(errTSV, "usa solo uno de --json o --tsv") {
		t.Fatalf("las flags de salida se contaminaron entre ejecuciones: %s", errTSV)
	}
}
