package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	var stderr bytes.Buffer
	out := capturarStdout(t, func() {
		if err := executeLocalArgs([]string{"tarea", "listar", "--agente", "Codex1", "--estado", string(db.EstadoAsignada), "--tsv"}, &bytes.Buffer{}, &stderr); err != nil {
			t.Fatalf("run listar tsv: %v stderr=%s", err, stderr.String())
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

func TestTareaListarAceptaEstadosMultiples(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/tareas", func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query()["estado"]
		if len(got) != 2 || got[0] != string(db.EstadoEnProgreso) || got[1] != string(db.EstadoAsignada) {
			t.Fatalf("estados enviados inesperados: %v", got)
		}
		_ = json.NewEncoder(w).Encode(apiTareasResponse{
			Tareas: []*db.Tarea{{
				ID:         17,
				Titulo:     "Auditoria",
				ProyectoID: ptrInt64(7),
				Prioridad:  db.PrioridadAlta,
				Estado:     db.EstadoEnProgreso,
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

	var stderr bytes.Buffer
	out := capturarStdout(t, func() {
		if err := executeLocalArgs([]string{"tarea", "listar", "--estado", string(db.EstadoEnProgreso), "--estado", string(db.EstadoAsignada)}, &bytes.Buffer{}, &stderr); err != nil {
			t.Fatalf("run listar multiestado: %v stderr=%s", err, stderr.String())
		}
	})
	if !strings.Contains(out, "Auditoria") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestListarTareasPorEstadosOR(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	libreID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Libre",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
	})
	if err != nil {
		t.Fatalf("crear libre: %v", err)
	}
	_ = libreID
	reservadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Reservada",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadMedia,
	})
	if err != nil {
		t.Fatalf("crear reservada: %v", err)
	}
	if err := db.TomarTarea(reservadaID, "Codex4"); err != nil {
		t.Fatalf("tomar reservada: %v", err)
	}
	cerradaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrada",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadBaja,
	})
	if err != nil {
		t.Fatalf("crear cerrada: %v", err)
	}
	if err := db.TomarTarea(cerradaID, "Codex4"); err != nil {
		t.Fatalf("tomar cerrada: %v", err)
	}
	if err := db.IniciarTarea(cerradaID, "Codex4"); err != nil {
		t.Fatalf("iniciar cerrada: %v", err)
	}
	if err := db.CompletarTarea(cerradaID, "Codex4", ""); err != nil {
		t.Fatalf("completar cerrada: %v", err)
	}

	tareas, err := listarTareasPorEstadosOR(db.FiltroTareas{}, []db.EstadoTarea{db.EstadoLibre, db.EstadoAsignada})
	if err != nil {
		t.Fatalf("listarTareasPorEstadosOR: %v", err)
	}
	if len(tareas) != 2 {
		t.Fatalf("tareas=%d, want 2", len(tareas))
	}
	got := []string{tareas[0].Titulo, tareas[1].Titulo}
	if !(containsString(got, "Libre") && containsString(got, "Reservada")) {
		t.Fatalf("titulos inesperados: %v", got)
	}
}

func TestAPIHandlerTareasListarRespetaLimit(t *testing.T) {
	prepararDBTemporalCmd(t)

	for i := 0; i < 3; i++ {
		if _, err := db.CrearTarea(&db.Tarea{
			Titulo:    "Tarea limit",
			Modulo:    "controlplane",
			Prioridad: db.PrioridadMedia,
			CreadoPor: "test",
		}); err != nil {
			t.Fatalf("crear tarea %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tareas?limit=1", nil)
	rec := httptest.NewRecorder()
	apiHandlerTareasListar(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiTareasResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Tareas) != 1 {
		t.Fatalf("len(resp.Tareas)=%d want 1", len(resp.Tareas))
	}
}

func ptrInt64(v int64) *int64    { return &v }
func ptrString(v string) *string { return &v }

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
