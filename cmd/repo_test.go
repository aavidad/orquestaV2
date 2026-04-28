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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/supervisionapp"
)

func TestRepoAddLocalUsaWorkspaceRootYResuelveProyectoPorRuta(t *testing.T) {
	workspace := t.TempDir()
	repo := filepath.Join(workspace, "orquestador")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")

	var discoveryRuta string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMaterializarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode materializar: %v", err)
		}
		discoveryRuta = workspace
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo, OrigenRepo: "local", BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       repo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiProyectoDescubrirRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode descubrir: %v", err)
		}
		discoveryRuta = req.Ruta
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         7,
				Slug:       "orquestador",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    repo,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("path", repo)
	out := capturarStdout(t, func() {
		if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
			t.Fatalf("repo add --path via api: %v", err)
		}
	})
	if filepath.Clean(discoveryRuta) != filepath.Clean(workspace) {
		t.Fatalf("discovery root inesperado: got=%s want=%s", discoveryRuta, workspace)
	}
	for _, token := range []string{"Repo local materializado", "orquestador", repo, "Origen: local"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo add local sin %q:\n%s", token, out)
		}
	}
}

func TestRepoAddRemoteClonaYDescubreProyecto(t *testing.T) {
	remoteParent := t.TempDir()
	remoteRepo := filepath.Join(remoteParent, "repo-remoto")
	if err := os.MkdirAll(remoteRepo, 0o755); err != nil {
		t.Fatalf("mkdir remote repo: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "init")
	runGitCmdAPITest(t, remoteRepo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, remoteRepo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(remoteRepo, "README.md"), []byte("hola remoto\n"), 0o644); err != nil {
		t.Fatalf("write remote readme: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "add", "README.md")
	runGitCmdAPITest(t, remoteRepo, "commit", "-m", "init")

	workspace := t.TempDir()
	clonePath := filepath.Join(workspace, "repo-remoto")
	var discoveryRuta string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMaterializarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode materializar remoto: %v", err)
		}
		if req.Git != remoteRepo {
			t.Fatalf("git remoto inesperado: %s", req.Git)
		}
		discoveryRuta = workspace
		if err := os.MkdirAll(clonePath, 0o755); err != nil {
			t.Fatalf("mkdir clone path: %v", err)
		}
		if err := os.WriteFile(filepath.Join(clonePath, "README.md"), []byte("hola remoto\n"), 0o644); err != nil {
			t.Fatalf("write clone readme: %v", err)
		}
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 9, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath, OrigenRepo: "git", RemoteURL: remoteRepo, BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       clonePath,
			RemoteURL:     remoteRepo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoDescubrirRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode descubrir: %v", err)
		}
		discoveryRuta = req.Ruta
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 9, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/9", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         9,
				Slug:       "repo-remoto",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    clonePath,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("git", remoteRepo)
	out := capturarStdout(t, func() {
		if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
			t.Fatalf("repo add --git via api: %v", err)
		}
	})
	if filepath.Clean(discoveryRuta) != filepath.Clean(workspace) {
		t.Fatalf("discovery root inesperado: got=%s want=%s", discoveryRuta, workspace)
	}
	if _, err := os.Stat(filepath.Join(clonePath, "README.md")); err != nil {
		t.Fatalf("clone no materializado en local: %v", err)
	}
	for _, token := range []string{"Repo remoto materializado", "repo-remoto", clonePath, "Origen: git", "Remote URL: " + remoteRepo} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo add remoto sin %q:\n%s", token, out)
		}
	}
}

func TestRepoAddYRevisarPlanViaAPI(t *testing.T) {
	workspace := t.TempDir()
	repo := filepath.Join(workspace, "orquestador")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")

	paso := &capacidadapp.PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		Accion:       "arrancar_fase",
		AccionTarea:  "implementar",
		Motivo:       "Hay contrato listo para abrir implementacion",
		FaseActual:   "especificacion",
		FaseObjetivo: "implementacion",
		TareaObjetivo: &capacidadapp.TareaPipelineLocal{
			ID:     17,
			Titulo: "Implementar repo add",
			Estado: string(db.TareaEnProgreso),
		},
		EtapaObjetivo: &capacidadapp.EtapaPipelineLocal{
			Fase:             "implementacion",
			Carril:           "premium_worktree",
			EntregaCanonica:  "git_worktree",
			RequiereWorktree: true,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo, OrigenRepo: "local", BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       repo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/revisar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoRevisarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo revisar: %v", err)
		}
		if req.Proyecto != "orquestador" || !req.Plan {
			t.Fatalf("request repo revisar inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiRepoRevisarResponse{OK: true, Proyecto: &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo}, Paso: paso})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/7", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         7,
				Slug:       "orquestador",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    repo,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	mux.HandleFunc("/api/modelo/pipeline-local/paso", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
			t.Fatalf("proyecto pipeline inesperado: %s", got)
		}
		_ = json.NewEncoder(w).Encode(apiPasoPipelineLocalDeterministaResponse{Paso: paso})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("path", repo)
	if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
		t.Fatalf("repo add via api: %v", err)
	}

	resetCommandFlags(repoRevisarCmd)
	_ = repoRevisarCmd.Flags().Set("proyecto", "orquestador")
	_ = repoRevisarCmd.Flags().Set("plan", "true")
	out := capturarStdout(t, func() {
		if err := repoRevisarCmd.RunE(repoRevisarCmd, nil); err != nil {
			t.Fatalf("repo revisar --plan via api: %v", err)
		}
	})
	for _, token := range []string{"Accion: arrancar_fase", "Fase objetivo: implementacion", "Tarea: #17 Implementar repo add", "Carril: premium_worktree", "Entrega: git_worktree"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo revisar --plan sin %q:\n%s", token, out)
		}
	}
}

func TestRepoAddYRevisarDespachaViaAPI(t *testing.T) {
	workspace := t.TempDir()
	repo := filepath.Join(workspace, "orquestador")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")

	resultado := &capacidadapp.ResultadoEjecucionPasoPipelineLocal{
		Paso: &capacidadapp.PasoPipelineLocalDeterminista{
			ProyectoSlug: "orquestador",
			Accion:       "arrancar_fase",
			AccionTarea:  "implementar",
			Motivo:       "Hay trabajo listo para despachar",
			FaseObjetivo: "implementacion",
			EtapaObjetivo: &capacidadapp.EtapaPipelineLocal{
				Fase:            "implementacion",
				Carril:          "premium_worktree",
				EntregaCanonica: "git_worktree",
			},
		},
		TareaActualizada: &capacidadapp.TareaPipelineLocal{
			ID:     23,
			Titulo: "Mejorar runtime mailbox",
			Estado: string(db.TareaEnProgreso),
		},
		Despacho: &capacidadapp.DespachoPipelineLocal{
			ProyectoSlug:    "orquestador",
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			EntregaCanonica: "git_worktree",
			FinishApp:       true,
			AgenteSugerido:  "Codex1",
			SeleccionAgente: &capacidadapp.SeleccionAgentePipeline{
				Agente:     "Codex1",
				Estrategia: "fallback_carril",
				Motivo:     "fallback de carril: no habia local con fitness suficiente; se usa el agente preferido del carril premium_worktree (Codex1)",
			},
		},
		DispatchRuntime: &capacidadapp.ResultadoDespachoPipeline{
			Estado: "pending",
			Motivo: "dispatch durable encolado",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo, OrigenRepo: "local", BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       repo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/revisar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoRevisarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo revisar despachar: %v", err)
		}
		if req.Proyecto != "orquestador" || req.Plan {
			t.Fatalf("request repo revisar despachar inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiRepoRevisarResponse{OK: true, Proyecto: &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo}, Resultado: resultado})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/7", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         7,
				Slug:       "orquestador",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    repo,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	mux.HandleFunc("/api/modelo/pipeline-local/despachar", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
			t.Fatalf("proyecto despachar inesperado: %s", got)
		}
		_ = json.NewEncoder(w).Encode(apiEjecutarPasoPipelineLocalDeterministaResponse{Resultado: resultado})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("path", repo)
	if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
		t.Fatalf("repo add via api: %v", err)
	}

	resetCommandFlags(repoRevisarCmd)
	_ = repoRevisarCmd.Flags().Set("proyecto", "orquestador")
	out := capturarStdout(t, func() {
		if err := repoRevisarCmd.RunE(repoRevisarCmd, nil); err != nil {
			t.Fatalf("repo revisar via api: %v", err)
		}
	})
	for _, token := range []string{"Accion: arrancar_fase", "Tarea actualizada: #23 Mejorar runtime mailbox", "Agente sugerido: Codex1", "Estrategia agente: fallback_carril", "Motivo agente: fallback de carril: no habia local con fitness suficiente; se usa el agente preferido del carril premium_worktree (Codex1)", "Estado dispatch: pending", "Motivo dispatch: dispatch durable encolado"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo revisar sin %q:\n%s", token, out)
		}
	}
}

func TestRepoAddRemoteYRevisarPlanViaAPI(t *testing.T) {
	remoteParent := t.TempDir()
	remoteRepo := filepath.Join(remoteParent, "repo-remoto")
	if err := os.MkdirAll(remoteRepo, 0o755); err != nil {
		t.Fatalf("mkdir remote repo: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "init")
	runGitCmdAPITest(t, remoteRepo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, remoteRepo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(remoteRepo, "README.md"), []byte("hola remoto\n"), 0o644); err != nil {
		t.Fatalf("write remote readme: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "add", "README.md")
	runGitCmdAPITest(t, remoteRepo, "commit", "-m", "init")

	workspace := t.TempDir()
	clonePath := filepath.Join(workspace, "repo-remoto")
	paso := &capacidadapp.PasoPipelineLocalDeterminista{
		ProyectoSlug: "repo-remoto",
		Accion:       "arrancar_fase",
		AccionTarea:  "implementar",
		Motivo:       "El repo remoto materializado ya puede entrar al pipeline",
		FaseActual:   "especificacion",
		FaseObjetivo: "implementacion",
		TareaObjetivo: &capacidadapp.TareaPipelineLocal{
			ID:     31,
			Titulo: "Primer frente del repo remoto",
			Estado: string(db.TareaAsignada),
		},
		EtapaObjetivo: &capacidadapp.EtapaPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			EntregaCanonica: "git_worktree",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMaterializarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode materializar remoto revisar: %v", err)
		}
		if req.Git != remoteRepo {
			t.Fatalf("git remoto revisar inesperado: %s", req.Git)
		}
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 11, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath, OrigenRepo: "git", RemoteURL: remoteRepo, BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       clonePath,
			RemoteURL:     remoteRepo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/revisar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoRevisarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo revisar remoto: %v", err)
		}
		if req.Proyecto != "repo-remoto" || !req.Plan {
			t.Fatalf("request repo revisar remoto inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiRepoRevisarResponse{OK: true, Proyecto: &db.Proyecto{ID: 11, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath}, Paso: paso})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoDescubrirRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode descubrir remoto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 11, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/11", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto remoto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         11,
				Slug:       "repo-remoto",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    clonePath,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	mux.HandleFunc("/api/modelo/pipeline-local/paso", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("proyecto"); got != "repo-remoto" {
			t.Fatalf("proyecto pipeline remoto inesperado: %s", got)
		}
		_ = json.NewEncoder(w).Encode(apiPasoPipelineLocalDeterministaResponse{Paso: paso})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("git", remoteRepo)
	if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
		t.Fatalf("repo add --git via api: %v", err)
	}

	resetCommandFlags(repoRevisarCmd)
	_ = repoRevisarCmd.Flags().Set("proyecto", "repo-remoto")
	_ = repoRevisarCmd.Flags().Set("plan", "true")
	out := capturarStdout(t, func() {
		if err := repoRevisarCmd.RunE(repoRevisarCmd, nil); err != nil {
			t.Fatalf("repo revisar remoto --plan via api: %v", err)
		}
	})
	for _, token := range []string{"Accion: arrancar_fase", "Fase objetivo: implementacion", "Tarea: #31 Primer frente del repo remoto", "Carril: premium_worktree", "Entrega: git_worktree"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo revisar remoto --plan sin %q:\n%s", token, out)
		}
	}
}

func TestRepoAddYMejorarDespachaViaAPI(t *testing.T) {
	workspace := t.TempDir()
	repo := filepath.Join(workspace, "orquestador")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")

	resultado := &capacidadapp.ResultadoEjecucionPasoPipelineLocal{
		Paso: &capacidadapp.PasoPipelineLocalDeterminista{
			ProyectoSlug: "orquestador",
			Accion:       "arrancar_fase",
			AccionTarea:  "implementar",
			Motivo:       "La mejora sembrada abre el siguiente frente útil",
			FaseObjetivo: "implementacion",
		},
		TareaActualizada: &capacidadapp.TareaPipelineLocal{
			ID:     44,
			Titulo: "Mejorar repo add",
			Estado: string(db.TareaEnProgreso),
		},
		Despacho: &capacidadapp.DespachoPipelineLocal{
			ProyectoSlug:    "orquestador",
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			EntregaCanonica: "git_worktree",
			FinishApp:       true,
			AgenteSugerido:  "Codex1",
			SeleccionAgente: &capacidadapp.SeleccionAgentePipeline{
				Agente:     "Codex1",
				Estrategia: "fallback_carril",
				Motivo:     "fallback de carril: no habia local con fitness suficiente; se usa el agente preferido del carril premium_worktree (Codex1)",
			},
		},
		DispatchRuntime: &capacidadapp.ResultadoDespachoPipeline{
			Estado: "pending",
			Motivo: "dispatch durable encolado",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo, OrigenRepo: "local", BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       repo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/mejorar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMejorarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo mejorar: %v", err)
		}
		if req.Proyecto != "orquestador" || req.Titulo != "Mejorar repo add" || req.Modulo != "cmd" {
			t.Fatalf("request repo mejorar inesperada: %+v", req)
		}
		if req.FuncionObjetivo != "cmd.repoAddServer" || len(req.WriteSet) != 2 || len(req.ModelosCandidatos) != 2 || !req.PreservarArquitectura || !req.FinishApp || !req.AutonomiaPersistente {
			t.Fatalf("request repo mejorar fork inesperada: %+v", req)
		}
		if req.SupervisorAgente != "Codex1" || req.ReviewerAgente != "Codex2" || req.MaxWorkers != 4 {
			t.Fatalf("request repo mejorar autonomia inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiRepoMejorarResponse{
			OK:       true,
			Proyecto: &db.Proyecto{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo},
			Tarea:    &db.Tarea{ID: 44, Titulo: req.Titulo, Estado: db.TareaAsignada},
			Fork: &apiRepoFunctionForkSpec{
				FuncionObjetivo:       "cmd.repoAddServer",
				WriteSet:              []string{"cmd/repo.go", "cmd/repo_test.go"},
				ModelosCandidatos:     []string{"qwen", "gemma"},
				Materia:               "arquitectura",
				ForkLines:             3,
				SelectedModels:        []string{"qwen", "gemma", "llama"},
				PreservarArquitectura: true,
				DecisionMode:          "auto",
				DecisionReason:        "orquesta decide fork automático · materia=arquitectura · lineas=3 · modelos=qwen, gemma, llama · preserva arquitectura",
			},
			Policy: &supervisionapp.Policy{
				Enabled:          true,
				SupervisorAgente: "Codex1",
				ReviewerAgente:   "Codex2",
				MaxWorkers:       4,
			},
			Resultado: resultado,
		})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 7, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/7", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         7,
				Slug:       "orquestador",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    repo,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("path", repo)
	if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
		t.Fatalf("repo add via api: %v", err)
	}

	resetCommandFlags(repoMejorarCmd)
	_ = repoMejorarCmd.Flags().Set("proyecto", "orquestador")
	_ = repoMejorarCmd.Flags().Set("titulo", "Mejorar repo add")
	_ = repoMejorarCmd.Flags().Set("descripcion", "Cerrar alta y pipeline de repos")
	_ = repoMejorarCmd.Flags().Set("modulo", "cmd")
	_ = repoMejorarCmd.Flags().Set("funcion", "cmd.repoAddServer")
	_ = repoMejorarCmd.Flags().Set("write-set", "cmd/repo.go,cmd/repo_test.go")
	_ = repoMejorarCmd.Flags().Set("modelos", "qwen,gemma")
	_ = repoMejorarCmd.Flags().Set("preservar-arquitectura", "true")
	_ = repoMejorarCmd.Flags().Set("finish-app", "true")
	_ = repoMejorarCmd.Flags().Set("autonomia-persistente", "true")
	_ = repoMejorarCmd.Flags().Set("supervisor", "Codex1")
	_ = repoMejorarCmd.Flags().Set("reviewer", "Codex2")
	_ = repoMejorarCmd.Flags().Set("max-workers", "4")
	out := capturarStdout(t, func() {
		if err := repoMejorarCmd.RunE(repoMejorarCmd, nil); err != nil {
			t.Fatalf("repo mejorar via api: %v", err)
		}
	})
	for _, token := range []string{"Tarea #44 creada para orquestador", "Fork función: cmd.repoAddServer", "Materia fork: arquitectura", "Líneas fork: 3", "Modelos elegidos: qwen, gemma, llama", "Restricción fork: preservar arquitectura", "Motivo fork: orquesta decide fork automático", "Autonomía persistente: activa", "Supervisor Codex: Codex1", "Reviewer Codex: Codex2", "Workers máximos: 4", "Tarea actualizada: #44 Mejorar repo add", "Agente sugerido: Codex1", "Modo dispatch: finish_app", "Estrategia agente: fallback_carril", "Motivo agente: fallback de carril: no habia local con fitness suficiente; se usa el agente preferido del carril premium_worktree (Codex1)", "Estado dispatch: pending"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo mejorar sin %q:\n%s", token, out)
		}
	}
}

func TestRepoAddRemoteYMejorarSinDespacharViaAPI(t *testing.T) {
	remoteParent := t.TempDir()
	remoteRepo := filepath.Join(remoteParent, "repo-remoto")
	if err := os.MkdirAll(remoteRepo, 0o755); err != nil {
		t.Fatalf("mkdir remote repo: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "init")
	runGitCmdAPITest(t, remoteRepo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, remoteRepo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(remoteRepo, "README.md"), []byte("hola remoto\n"), 0o644); err != nil {
		t.Fatalf("write remote readme: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "add", "README.md")
	runGitCmdAPITest(t, remoteRepo, "commit", "-m", "init")

	workspace := t.TempDir()
	clonePath := filepath.Join(workspace, "repo-remoto")
	var despacharInvocado bool

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 11, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath, OrigenRepo: "git", RemoteURL: remoteRepo, BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       clonePath,
			RemoteURL:     remoteRepo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/mejorar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMejorarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo mejorar remoto: %v", err)
		}
		despacharInvocado = req.Despachar != nil && *req.Despachar
		_ = json.NewEncoder(w).Encode(apiRepoMejorarResponse{
			OK:       true,
			Proyecto: &db.Proyecto{ID: 11, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath},
			Tarea:    &db.Tarea{ID: 61, Titulo: req.Titulo, Estado: db.TareaAsignada},
		})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 11, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/11", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto remoto: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         11,
				Slug:       "repo-remoto",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    clonePath,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoAddCmd)
	_ = repoAddCmd.Flags().Set("git", remoteRepo)
	if err := repoAddCmd.RunE(repoAddCmd, nil); err != nil {
		t.Fatalf("repo add --git via api: %v", err)
	}

	resetCommandFlags(repoMejorarCmd)
	_ = repoMejorarCmd.Flags().Set("proyecto", "repo-remoto")
	_ = repoMejorarCmd.Flags().Set("titulo", "Sembrar mejora remota")
	_ = repoMejorarCmd.Flags().Set("despachar", "false")
	out := capturarStdout(t, func() {
		if err := repoMejorarCmd.RunE(repoMejorarCmd, nil); err != nil {
			t.Fatalf("repo mejorar remoto via api: %v", err)
		}
	})
	if despacharInvocado {
		t.Fatal("no deberia despachar cuando --despachar=false")
	}
	if !strings.Contains(out, "Tarea #61 creada para repo-remoto") {
		t.Fatalf("salida repo mejorar remoto inesperada:\n%s", out)
	}
}

func TestRepoRevisarConPathMaterializaYPlanificaViaAPI(t *testing.T) {
	workspace := t.TempDir()
	repo := filepath.Join(workspace, "orquestador")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")

	paso := &capacidadapp.PasoPipelineLocalDeterminista{
		ProyectoSlug: "orquestador",
		Accion:       "arrancar_fase",
		AccionTarea:  "implementar",
		Motivo:       "El repo local materializado al vuelo ya puede entrar en implementacion",
		FaseObjetivo: "implementacion",
		TareaObjetivo: &capacidadapp.TareaPipelineLocal{
			ID:     71,
			Titulo: "Revisar repo por path",
			Estado: string(db.TareaAsignada),
		},
		EtapaObjetivo: &capacidadapp.EtapaPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			EntregaCanonica: "git_worktree",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMaterializarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode materializar path revisar: %v", err)
		}
		if req.Path != repo {
			t.Fatalf("path revisar inesperado: %s", req.Path)
		}
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 21, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo, OrigenRepo: "local", BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       repo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/revisar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoRevisarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo revisar path: %v", err)
		}
		if req.Proyecto != "orquestador" || !req.Plan {
			t.Fatalf("request repo revisar path inesperada: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(apiRepoRevisarResponse{OK: true, Proyecto: &db.Proyecto{ID: 21, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo}, Paso: paso})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 21, Slug: "orquestador", Tipo: db.ProyectoRepo, RutaAbs: repo},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/21", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto path: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         21,
				Slug:       "orquestador",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    repo,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	mux.HandleFunc("/api/modelo/pipeline-local/paso", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("proyecto"); got != "orquestador" {
			t.Fatalf("proyecto revisar path inesperado: %s", got)
		}
		_ = json.NewEncoder(w).Encode(apiPasoPipelineLocalDeterministaResponse{Paso: paso})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoRevisarCmd)
	_ = repoRevisarCmd.Flags().Set("path", repo)
	_ = repoRevisarCmd.Flags().Set("plan", "true")
	out := capturarStdout(t, func() {
		if err := repoRevisarCmd.RunE(repoRevisarCmd, nil); err != nil {
			t.Fatalf("repo revisar --path via api: %v", err)
		}
	})
	for _, token := range []string{"Repo local materializado", "Proyecto: #21 orquestador", "Accion: arrancar_fase", "Tarea: #71 Revisar repo por path"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo revisar --path sin %q:\n%s", token, out)
		}
	}
}

func TestRepoMejorarConGitMaterializaYCreaTareaViaAPI(t *testing.T) {
	remoteParent := t.TempDir()
	remoteRepo := filepath.Join(remoteParent, "repo-remoto")
	if err := os.MkdirAll(remoteRepo, 0o755); err != nil {
		t.Fatalf("mkdir remote repo: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "init")
	runGitCmdAPITest(t, remoteRepo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, remoteRepo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(remoteRepo, "README.md"), []byte("hola remoto\n"), 0o644); err != nil {
		t.Fatalf("write remote readme: %v", err)
	}
	runGitCmdAPITest(t, remoteRepo, "add", "README.md")
	runGitCmdAPITest(t, remoteRepo, "commit", "-m", "init")

	workspace := t.TempDir()
	clonePath := filepath.Join(workspace, "repo-remoto")
	var despacharInvocado bool

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/repos/materializar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMaterializarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode materializar git mejorar: %v", err)
		}
		if req.Git != remoteRepo {
			t.Fatalf("git mejorar inesperado: %s", req.Git)
		}
		_ = json.NewEncoder(w).Encode(apiRepoMaterializarResponse{
			OK:            true,
			Proyecto:      &db.Proyecto{ID: 33, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath, OrigenRepo: "git", RemoteURL: remoteRepo, BranchBase: "master", Activo: true},
			DiscoveryRoot: workspace,
			RutaAbs:       clonePath,
			RemoteURL:     remoteRepo,
			BranchBase:    "master",
		})
	})
	mux.HandleFunc("/api/repos/mejorar", func(w http.ResponseWriter, r *http.Request) {
		var req apiRepoMejorarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode repo mejorar git: %v", err)
		}
		despacharInvocado = req.Despachar != nil && *req.Despachar
		_ = json.NewEncoder(w).Encode(apiRepoMejorarResponse{
			OK:       true,
			Proyecto: &db.Proyecto{ID: 33, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath},
			Tarea:    &db.Tarea{ID: 91, Titulo: req.Titulo, Estado: db.TareaAsignada},
		})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": workspace}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyectos": []*db.Proyecto{
				{ID: 1, Slug: filepath.Base(workspace), Tipo: db.ProyectoRaiz, RutaAbs: workspace},
				{ID: 33, Slug: "repo-remoto", Tipo: db.ProyectoRepo, RutaAbs: clonePath},
			},
		})
	})
	mux.HandleFunc("/api/proyectos/33", func(w http.ResponseWriter, r *http.Request) {
		var req apiProyectoActualizarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode update proyecto git mejorar: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"proyecto": &db.Proyecto{
				ID:         33,
				Slug:       "repo-remoto",
				Tipo:       db.ProyectoRepo,
				RutaAbs:    clonePath,
				OrigenRepo: req.OrigenRepo,
				RemoteURL:  req.RemoteURL,
				BranchBase: req.BranchBase,
				Activo:     true,
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(repoMejorarCmd)
	_ = repoMejorarCmd.Flags().Set("git", remoteRepo)
	_ = repoMejorarCmd.Flags().Set("titulo", "Mejora directa sobre repo remoto")
	_ = repoMejorarCmd.Flags().Set("despachar", "false")
	out := capturarStdout(t, func() {
		if err := repoMejorarCmd.RunE(repoMejorarCmd, nil); err != nil {
			t.Fatalf("repo mejorar --git via api: %v", err)
		}
	})
	if despacharInvocado {
		t.Fatal("no deberia despachar cuando --despachar=false")
	}
	for _, token := range []string{"Repo remoto materializado", "Proyecto: #33 repo-remoto", "Tarea #91 creada para repo-remoto"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida repo mejorar --git sin %q:\n%s", token, out)
		}
	}
}
