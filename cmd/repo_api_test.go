/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestAPIReposMaterializarLocal(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
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

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos/materializar", bytes.NewReader([]byte(`{"path":"`+repo+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiRepoMaterializarResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !payload.OK || payload.Proyecto == nil {
		t.Fatalf("payload materializar inválido: %s", rec.Body.String())
	}
	if payload.Proyecto.OrigenRepo != "local" || filepath.Clean(payload.Proyecto.RutaAbs) != filepath.Clean(repo) {
		t.Fatalf("proyecto materializado inesperado: %+v", payload.Proyecto)
	}
}

func TestAPIReposMejorarYRevisarLocal(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
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

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recMejorar := httptest.NewRecorder()
	reqMejorar := httptest.NewRequest(http.MethodPost, "/api/repos/mejorar", bytes.NewReader([]byte(`{"path":"`+repo+`","titulo":"Mejorar repo API","descripcion":"Completar la app hasta cerrar backlog","despachar":false,"funcion_objetivo":"pkg.Calcular","write_set":["pkg/calculo.go","pkg/calculo_test.go"],"modelos_candidatos":["qwen","gemma"],"preservar_arquitectura":true,"autonomia_persistente":true,"supervisor_agente":"Codex1","reviewer_agente":"Codex2","max_workers":4}`)))
	reqMejorar.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recMejorar, reqMejorar)
	if recMejorar.Code != http.StatusCreated {
		t.Fatalf("status mejorar inesperado: %d body=%s", recMejorar.Code, recMejorar.Body.String())
	}
	var payloadMejorar apiRepoMejorarResponse
	if err := json.Unmarshal(recMejorar.Body.Bytes(), &payloadMejorar); err != nil {
		t.Fatalf("decode mejorar: %v", err)
	}
	if !payloadMejorar.OK || payloadMejorar.Proyecto == nil || payloadMejorar.Tarea == nil {
		t.Fatalf("payload mejorar inválido: %s", recMejorar.Body.String())
	}
	if payloadMejorar.Proyecto.Slug == "" || payloadMejorar.Tarea.Titulo != "Mejorar repo API" {
		t.Fatalf("resultado mejorar inesperado: %+v tarea=%+v", payloadMejorar.Proyecto, payloadMejorar.Tarea)
	}
	if payloadMejorar.Tarea.Prioridad != db.PrioridadMedia {
		t.Fatalf("prioridad por defecto inesperada: %s", payloadMejorar.Tarea.Prioridad)
	}
	if payloadMejorar.Fork == nil {
		t.Fatalf("payload mejorar sin fork parseado: %+v", payloadMejorar)
	}
	if payloadMejorar.Policy == nil || !payloadMejorar.Policy.Enabled {
		t.Fatalf("payload mejorar sin policy persistente: %+v", payloadMejorar)
	}
	if payloadMejorar.Policy.SupervisorAgente != "Codex1" || payloadMejorar.Policy.ReviewerAgente != "Codex2" || payloadMejorar.Policy.MaxWorkers != 4 {
		t.Fatalf("policy persistente inesperada: %+v", payloadMejorar.Policy)
	}
	if payloadMejorar.Fork.FuncionObjetivo != "pkg.Calcular" || len(payloadMejorar.Fork.WriteSet) != 2 || len(payloadMejorar.Fork.ModelosCandidatos) != 2 || !payloadMejorar.Fork.PreservarArquitectura {
		t.Fatalf("fork parseado inesperado: %+v", payloadMejorar.Fork)
	}
	if payloadMejorar.Fork.Materia != "arquitectura" {
		t.Fatalf("materia fork inesperada: %+v", payloadMejorar.Fork)
	}
	if payloadMejorar.Fork.ForkLines != 3 {
		t.Fatalf("fork_lines inesperado: %+v", payloadMejorar.Fork)
	}
	if payloadMejorar.Fork.DecisionMode != "auto" || strings.TrimSpace(payloadMejorar.Fork.DecisionReason) == "" {
		t.Fatalf("decision fork incompleta: %+v", payloadMejorar.Fork)
	}
	if len(payloadMejorar.Fork.SelectedModels) != 3 {
		t.Fatalf("selected_models inesperado: %+v", payloadMejorar.Fork)
	}
	for _, token := range []string{"fork_funcion_v1:", "funcion_objetivo: pkg.Calcular", "write_set: pkg/calculo.go, pkg/calculo_test.go", "modelos_candidatos: qwen, gemma", "materia: arquitectura", "fork_lines: 3", "selected_models: qwen, gemma, llama", "decision_mode: auto", "decision_reason: orquesta decide fork automático", "preservar_arquitectura: true", "autonomia:finish_app", "autonomia_persistente_v1:", "supervisor_agente: Codex1", "reviewer_agente: Codex2", "max_workers: 4"} {
		if !strings.Contains(payloadMejorar.Tarea.Notas, token) {
			t.Fatalf("tarea mejorar sin token %q en notas:\n%s", token, payloadMejorar.Tarea.Notas)
		}
	}

	recRevisar := httptest.NewRecorder()
	reqRevisar := httptest.NewRequest(http.MethodPost, "/api/repos/revisar", bytes.NewReader([]byte(`{"proyecto":"`+payloadMejorar.Proyecto.Slug+`","plan":true}`)))
	reqRevisar.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recRevisar, reqRevisar)
	if recRevisar.Code != http.StatusOK {
		t.Fatalf("status revisar inesperado: %d body=%s", recRevisar.Code, recRevisar.Body.String())
	}
	var payloadRevisar apiRepoRevisarResponse
	if err := json.Unmarshal(recRevisar.Body.Bytes(), &payloadRevisar); err != nil {
		t.Fatalf("decode revisar: %v", err)
	}
	if !payloadRevisar.OK || payloadRevisar.Proyecto == nil {
		t.Fatalf("payload revisar inválido: %s", recRevisar.Body.String())
	}
}

func TestResolveRepoPersistentAutonomyRequestAutoSeleccionaCodexOperativos(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %v proyecto=%+v", err, proyecto)
	}
	for _, nombre := range []string{"Codex1", "Codex2", "Codex3", "Codex4"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}
	if err := db.PausarAgente("Codex1", 30, "cuota bloqueada para fallback"); err != nil {
		t.Fatalf("pausar codex1: %v", err)
	}
	req := normalizeRepoMejorarRequest(apiRepoMejorarRequest{
		Proyecto:             "orquestador",
		Titulo:               "Cerrar app",
		Descripcion:          "Cerrar autonomamente hasta terminar",
		AutonomiaPersistente: true,
		MaxWorkers:           2,
	})
	got, err := resolveRepoPersistentAutonomyRequest(proyecto, req)
	if err != nil {
		t.Fatalf("resolveRepoPersistentAutonomyRequest: %v", err)
	}
	if got.SupervisorAgente != "Codex2" {
		t.Fatalf("supervisor inesperado: %+v", got)
	}
	if got.ReviewerAgente != "Codex3" {
		t.Fatalf("reviewer inesperado: %+v", got)
	}
	for _, nombre := range []string{"Codex2", "Codex3"} {
		asignacion, err := db.GetAsignacionActivaAgente(nombre)
		if err != nil {
			t.Fatalf("asignacion activa %s: %v", nombre, err)
		}
		if asignacion == nil || asignacion.ProyectoID != proyectoID {
			t.Fatalf("asignacion de %s inesperada: %+v", nombre, asignacion)
		}
	}
}

func TestResolveRepoPersistentAutonomyRequestNoUsaPrimeComoWorkerPorDefecto(t *testing.T) {
	prepararDBTemporalCmd(t)
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %v proyecto=%+v", err, proyecto)
	}
	for _, nombre := range []string{"Codex1", "Codex2", "Codex3", "CodexPg1"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}

	req := normalizeRepoMejorarRequest(apiRepoMejorarRequest{
		Proyecto:             "orquestador",
		Titulo:               "Cerrar app",
		Descripcion:          "Cerrar autonomamente hasta terminar",
		AutonomiaPersistente: true,
		MaxWorkers:           2,
	})
	got, err := resolveRepoPersistentAutonomyRequest(proyecto, req)
	if err != nil {
		t.Fatalf("resolveRepoPersistentAutonomyRequest: %v", err)
	}
	if got.SupervisorAgente != "Codex1" {
		t.Fatalf("supervisor inesperado: %+v", got)
	}
	if got.ReviewerAgente != "Codex2" {
		t.Fatalf("reviewer inesperado: %+v", got)
	}
	if got.MaxWorkers != 2 {
		t.Fatalf("max workers inesperado: %+v", got)
	}
	asignacionPrime, err := db.GetAsignacionActivaAgente("CodexPg1")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("get asignacion activa prime: %v", err)
	}
	if err == nil && asignacionPrime != nil {
		t.Fatalf("prime no deberia entrar al pool worker por defecto: %+v", asignacionPrime)
	}
}

func TestAPIReposMaterializarYRevisarRemote(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
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

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recMaterializar := httptest.NewRecorder()
	reqMaterializar := httptest.NewRequest(http.MethodPost, "/api/repos/materializar", bytes.NewReader([]byte(`{"git":"`+remoteRepo+`"}`)))
	reqMaterializar.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recMaterializar, reqMaterializar)
	if recMaterializar.Code != http.StatusCreated {
		t.Fatalf("status materializar remoto inesperado: %d body=%s", recMaterializar.Code, recMaterializar.Body.String())
	}
	var payloadMaterializar apiRepoMaterializarResponse
	if err := json.Unmarshal(recMaterializar.Body.Bytes(), &payloadMaterializar); err != nil {
		t.Fatalf("decode materializar remoto: %v", err)
	}
	if !payloadMaterializar.OK || payloadMaterializar.Proyecto == nil {
		t.Fatalf("payload materializar remoto inválido: %s", recMaterializar.Body.String())
	}
	if payloadMaterializar.Proyecto.OrigenRepo != "git" || payloadMaterializar.Proyecto.RemoteURL != remoteRepo {
		t.Fatalf("proyecto remoto inesperado: %+v", payloadMaterializar.Proyecto)
	}

	recRevisar := httptest.NewRecorder()
	reqRevisar := httptest.NewRequest(http.MethodPost, "/api/repos/revisar", bytes.NewReader([]byte(`{"proyecto":"`+payloadMaterializar.Proyecto.Slug+`","plan":true}`)))
	reqRevisar.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recRevisar, reqRevisar)
	if recRevisar.Code != http.StatusOK {
		t.Fatalf("status revisar remoto inesperado: %d body=%s", recRevisar.Code, recRevisar.Body.String())
	}
	var payloadRevisar apiRepoRevisarResponse
	if err := json.Unmarshal(recRevisar.Body.Bytes(), &payloadRevisar); err != nil {
		t.Fatalf("decode revisar remoto: %v", err)
	}
	if !payloadRevisar.OK || payloadRevisar.Proyecto == nil || payloadRevisar.Paso == nil {
		t.Fatalf("payload revisar remoto inválido: %s", recRevisar.Body.String())
	}
}

func TestAPIReposMejorarRemoteSinDespachar(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
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

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recMejorar := httptest.NewRecorder()
	reqMejorar := httptest.NewRequest(http.MethodPost, "/api/repos/mejorar", bytes.NewReader([]byte(`{"git":"`+remoteRepo+`","titulo":"Mejorar repo remoto API","despachar":false}`)))
	reqMejorar.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recMejorar, reqMejorar)
	if recMejorar.Code != http.StatusCreated {
		t.Fatalf("status mejorar remoto inesperado: %d body=%s", recMejorar.Code, recMejorar.Body.String())
	}
	var payload apiRepoMejorarResponse
	if err := json.Unmarshal(recMejorar.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode mejorar remoto: %v", err)
	}
	if !payload.OK || payload.Proyecto == nil || payload.Tarea == nil {
		t.Fatalf("payload mejorar remoto inválido: %s", recMejorar.Body.String())
	}
	if payload.Proyecto.OrigenRepo != "git" || payload.Proyecto.RemoteURL != remoteRepo {
		t.Fatalf("proyecto remoto inesperado: %+v", payload.Proyecto)
	}
	if payload.Tarea.Titulo != "Mejorar repo remoto API" || payload.Tarea.Prioridad != db.PrioridadMedia {
		t.Fatalf("tarea remota inesperada: %+v", payload.Tarea)
	}
	if payload.Resultado != nil {
		t.Fatalf("no deberia despachar cuando despachar=false: %+v", payload.Resultado)
	}
}

func TestAPIReposMejorarAutoSeleccionaModelosPorMateria(t *testing.T) {
	prepararDBTemporalCmd(t)
	workspace := t.TempDir()
	if err := db.ConfigSet("workspace_root", workspace); err != nil {
		t.Fatalf("config workspace_root: %v", err)
	}
	repo := filepath.Join(workspace, "infra-app")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
	runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
	if err := os.WriteFile(filepath.Join(repo, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "Dockerfile")
	runGitCmdAPITest(t, repo, "commit", "-m", "init")
	if _, err := db.RegistrarBenchmarkScoreAgenteLocal("Qwen1", "ollama-cli", "infraestructura", 9.2, "fuerte en infra"); err != nil {
		t.Fatalf("seed qwen: %v", err)
	}
	if _, err := db.RegistrarBenchmarkScoreAgenteLocal("Gemma1", "ollama-cli", "infraestructura", 8.4, "bueno en infra"); err != nil {
		t.Fatalf("seed gemma: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos/mejorar", bytes.NewReader([]byte(`{"path":"`+repo+`","titulo":"Refactor infra docker","descripcion":"Mejorar infraestructura y docker","despachar":false,"funcion_objetivo":"infra.BuildImage","write_set":["Dockerfile","deploy/docker-compose.yml"],"preservar_arquitectura":true}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiRepoMejorarResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode mejorar auto: %v", err)
	}
	if payload.Fork == nil {
		t.Fatalf("payload sin fork: %s", rec.Body.String())
	}
	if payload.Fork.Materia != "infraestructura" {
		t.Fatalf("materia inesperada: %+v", payload.Fork)
	}
	if payload.Fork.ForkLines != 3 {
		t.Fatalf("fork_lines inesperado: %+v", payload.Fork)
	}
	if len(payload.Fork.SelectedModels) < 2 || payload.Fork.SelectedModels[0] != "qwen" || payload.Fork.SelectedModels[1] != "gemma" {
		t.Fatalf("selected_models inesperado: %+v", payload.Fork)
	}
	if !strings.Contains(payload.Tarea.Notas, "selected_models: qwen, gemma") {
		t.Fatalf("notas sin selected_models esperado:\n%s", payload.Tarea.Notas)
	}
}

func TestParseRepoFunctionForkSpecFromNotesToleraRuido(t *testing.T) {
	raw := "contexto previo\n\nfork_funcion_v1:\n  funcion_objetivo: app.Calcular\n  write_set: app/calculo.go, app/calculo_test.go\n  modelos_candidatos: qwen, gemma\n  materia: arquitectura\n  fork_lines: 3\n  selected_models: qwen, gemma, llama\n  decision_mode: auto\n  decision_reason: orquesta decide fork automático · materia=arquitectura\n  preservar_arquitectura: true\n\notro bloque:\n  x: y\n"
	spec := parseRepoFunctionForkSpecFromNotes(raw)
	if spec == nil {
		t.Fatalf("spec nil")
	}
	if spec.FuncionObjetivo != "app.Calcular" || len(spec.WriteSet) != 2 || len(spec.ModelosCandidatos) != 2 || !spec.PreservarArquitectura {
		t.Fatalf("spec inesperada: %+v", spec)
	}
	if spec.Materia != "arquitectura" || spec.ForkLines != 3 || len(spec.SelectedModels) != 3 || spec.DecisionMode != "auto" || strings.TrimSpace(spec.DecisionReason) == "" {
		t.Fatalf("spec enriquecida inesperada: %+v", spec)
	}
}
