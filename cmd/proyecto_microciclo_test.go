package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/progresoapp"
	"orquesta/supervisionapp"
)

func TestEliminarColisionWorktreeMicrocicloPurgaRefsLegacyConflictivas(t *testing.T) {
	repo := t.TempDir()
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")
	runGitCmdAPITest(t, repo, "branch", "orq-orquestador-gemini1/t526")

	worktreePath := filepath.Join(repo, ".orquesta-worktrees", "orquestador-gemini1")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatalf("mkdir worktree path: %v", err)
	}

	if err := eliminarColisionWorktreeMicrociclo(&db.Proyecto{Slug: "orquestador", RutaAbs: repo}, "Gemini1"); err != nil {
		t.Fatalf("eliminarColisionWorktreeMicrociclo: %v", err)
	}

	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("la ruta de worktree deberia eliminarse, err=%v", err)
	}
	refs := gitRefsProyectoMicrocicloTest(t, repo)
	if strings.Contains(refs, "orq-orquestador-gemini1/t526") {
		t.Fatalf("la ref legacy conflictiva deberia haberse purgado: %s", refs)
	}
}

func gitRefsProyectoMicrocicloTest(t *testing.T, repo string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git for-each-ref: %v output=%s", err, string(out))
	}
	return string(out)
}

func TestProyectoMicrocicloCLIExponeFinishAppEnDispatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/proyectos/orquestador", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir(), Tipo: db.ProyectoRepo, Activo: true},
		})
	})
	mux.HandleFunc("/api/proyectos/orquestador/autonomia/microciclo", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiProyectoMicrocicloResponse{
			Resultado: &proyectoMicrocicloResult{
				Proyecto: &db.Proyecto{ID: 7, Slug: "orquestador", Nombre: "Orquestador", Tipo: db.ProyectoRepo, Activo: true},
				Policy:   &supervisionapp.Policy{SupervisorAgente: "Codex1"},
				Fase:     &progresoapp.FaseProyecto{Nombre: "implementacion"},
				Tarea:    &db.Tarea{ID: 44, Titulo: "Cerrar app completa", Estado: db.TareaAsignada},
				Dispatch: &capacidadapp.ResultadoEjecucionPasoPipelineLocal{
					Despacho: &capacidadapp.DespachoPipelineLocal{
						AgenteSugerido: "Codex1",
						FinishApp:      true,
						SeleccionAgente: &capacidadapp.SeleccionAgentePipeline{
							Agente:     "Codex1",
							Estrategia: "fallback_carril",
							Motivo:     "sin local suficiente",
						},
					},
					DispatchRuntime: &capacidadapp.ResultadoDespachoPipeline{Estado: "pending"},
				},
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(proyectoMicrocicloCmd)
	out := capturarStdout(t, func() {
		if err := proyectoMicrocicloCmd.RunE(proyectoMicrocicloCmd, []string{"orquestador"}); err != nil {
			t.Fatalf("proyecto microciclo via API: %v", err)
		}
	})
	for _, token := range []string{"✓ Microciclo activado en orquestador con Codex1", "Fase:      implementacion", "Tarea:     #44 Cerrar app completa", "Modo dispatch: finish_app", "Estrategia agente: fallback_carril", "Motivo agente: sin local suficiente", "Dispatch:  pending"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida microciclo sin %q:\n%s", token, out)
		}
	}
}
