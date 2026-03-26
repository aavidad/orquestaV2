package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"orquesta/db"
)

func TestTareaListarTSVParaScripts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiTareasResponse{
			Tareas: []*db.Tarea{{
				ID:          17,
				Titulo:      "Revisar conector hexagonal",
				Descripcion: "Sin acceso directo a SQLite",
				ProyectoID:  ptrInt64(7),
				Modulo:      "controlplane",
				Prioridad:   db.PrioridadAlta,
				Estado:      db.EstadoAsignada,
				Agente:      ptrString("Codex1"),
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

	for _, item := range []struct {
		name  string
		value string
	}{
		{name: "agente", value: "Codex1"},
		{name: "estado", value: string(db.EstadoAsignada)},
		{name: "tsv", value: "true"},
		{name: "json", value: "false"},
	} {
		if err := tareaListarCmd.Flags().Set(item.name, item.value); err != nil {
			t.Fatalf("set flag %s: %v", item.name, err)
		}
	}
	t.Cleanup(func() {
		_ = tareaListarCmd.Flags().Set("agente", "")
		_ = tareaListarCmd.Flags().Set("estado", "")
		_ = tareaListarCmd.Flags().Set("tsv", "false")
		_ = tareaListarCmd.Flags().Set("json", "false")
	})

	out := capturarStdout(t, func() {
		if err := tareaListarCmd.RunE(tareaListarCmd, nil); err != nil {
			t.Fatalf("run listar tsv: %v", err)
		}
	})
	want := "Revisar conector hexagonal"
	if !strings.Contains(out, "\t"+want) {
		t.Fatalf("salida TSV inesperada:\n%s", out)
	}
	if !strings.Contains(out, "\tasignada\talta\tCodex1\torquestador\tcontrolplane\t") {
		t.Fatalf("salida TSV sin columnas esperadas:\n%s", out)
	}
}

func ptrInt64(v int64) *int64    { return &v }
func ptrString(v string) *string { return &v }
