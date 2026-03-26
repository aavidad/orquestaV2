package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestTareaListarTSVEsEstableParaScripts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiTareasResponse{
			Tareas: []*db.Tarea{{
				ID:          33,
				Titulo:      "Refactor\tPTY\nconnector",
				ProyectoID:  ptrInt64(7),
				Modulo:      "controlplane",
				Prioridad:   db.PrioridadAlta,
				Estado:      db.EstadoAsignada,
				Agente:      ptrString("Codex1"),
				Descripcion: "salida estable para scripts",
			}},
		})
	})
	mux.HandleFunc("/api/proyectos", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiProyectosResponse{
			Proyectos: []*db.Proyecto{{ID: 7, Slug: "orquestador"}},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := tareaListarCmd.Flags().Set("agente", "Codex1"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := tareaListarCmd.Flags().Set("estado", "asignada"); err != nil {
		t.Fatalf("set estado: %v", err)
	}
	if err := tareaListarCmd.Flags().Set("tsv", "true"); err != nil {
		t.Fatalf("set tsv: %v", err)
	}
	t.Cleanup(func() {
		_ = tareaListarCmd.Flags().Set("agente", "")
		_ = tareaListarCmd.Flags().Set("estado", "")
		_ = tareaListarCmd.Flags().Set("tsv", "false")
	})

	out := capturarStdout(t, func() {
		if err := tareaListarCmd.RunE(tareaListarCmd, nil); err != nil {
			t.Fatalf("run tarea listar --tsv: %v", err)
		}
	})

	line := strings.TrimSpace(out)
	fields := strings.Split(line, "\t")
	if len(fields) != 7 {
		t.Fatalf("esperaba 7 columnas TSV, got=%d line=%q", len(fields), line)
	}
	if fields[0] != "33" || fields[1] != "asignada" || fields[2] != string(db.PrioridadAlta) {
		t.Fatalf("cabecera TSV inesperada: %v", fields[:3])
	}
	if fields[3] != "Codex1" || fields[4] != "orquestador" || fields[5] != "controlplane" {
		t.Fatalf("metadatos TSV inesperados: %v", fields[3:6])
	}
	if strings.Contains(fields[6], "\n") || strings.Contains(fields[6], "\t") {
		t.Fatalf("el titulo TSV no deberia contener tabs ni saltos de linea: %q", fields[6])
	}
	if fields[6] != "Refactor PTY connector" {
		t.Fatalf("titulo TSV inesperado: %q", fields[6])
	}
}
