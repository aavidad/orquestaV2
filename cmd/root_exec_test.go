/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestExecuteLocalArgsReseteaFlagsEntreEjecuciones(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiTareasResponse{
			Tareas: []*db.Tarea{
				{ID: 1, Titulo: "Cerrar bug de flags", Modulo: "cli", Prioridad: db.PrioridadAlta},
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

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
