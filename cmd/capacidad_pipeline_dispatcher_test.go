package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/gitaplicacion"
)

type fakeGitStorePipelineDispatcher struct {
	projectID int64
	worktrees []*db.Worktree
	merges    []*db.GitMergeRequest
	saved     *db.GitMergeRequest
}

func (f *fakeGitStorePipelineDispatcher) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return f.worktrees, nil
}

func (f *fakeGitStorePipelineDispatcher) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return nil, nil
}

func (f *fakeGitStorePipelineDispatcher) ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error) {
	return f.merges, nil
}

func (f *fakeGitStorePipelineDispatcher) SaveMerge(item *db.GitMergeRequest) (int64, error) {
	f.saved = item
	return 77, nil
}

func (f *fakeGitStorePipelineDispatcher) ResolveProjectID(slug string) (*int64, error) {
	return &f.projectID, nil
}

func prepararWorktreeGitDispatcherTest(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	run("init")
	run("config", "user.name", "Orquesta Test")
	run("config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README base: %v", err)
	}
	run("add", "README.md")
	run("commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write README dirty: %v", err)
	}
	return repo
}

func TestResolverConectorYModeloStartPipelinePremium(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre       string
		agente       string
		modelo       string
		wantConector string
		wantModelo   string
	}{
		{
			nombre:       "claude usa conector canonico y limpia modelo local",
			agente:       "Claude1",
			modelo:       "qwen3.5:27b-q4_K_M",
			wantConector: "claude-code",
			wantModelo:   "",
		},
		{
			nombre:       "gemini usa conector canonico y limpia modelo local",
			agente:       "Gemini1",
			modelo:       "qwen3.5:27b-q4_K_M",
			wantConector: "gemini-cli",
			wantModelo:   "",
		},
		{
			nombre:       "codex usa conector canonico y conserva modelo compatible",
			agente:       "Codex1",
			modelo:       "codex-mini-latest",
			wantConector: "codex-cli",
			wantModelo:   "codex-mini-latest",
		},
	}
	for _, tc := range casos {
		tc := tc
		t.Run(tc.nombre, func(t *testing.T) {
			t.Parallel()
			gotConector, gotModelo := resolverConectorYModeloStartPipeline(tc.agente, tc.modelo)
			if gotConector != tc.wantConector || gotModelo != tc.wantModelo {
				t.Fatalf("resolverConectorYModeloStartPipeline(%q,%q)=(%q,%q); want (%q,%q)",
					tc.agente, tc.modelo, gotConector, gotModelo, tc.wantConector, tc.wantModelo)
			}
		})
	}
}

func TestDespachadorPipelineOperativoEvitaDuplicarMailboxPendiente(t *testing.T) {
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
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"kind":   "pipeline_local",
		"accion": "continuar_trabajo",
	})
	if _, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: string(payload),
	}); err != nil {
		t.Fatalf("send runtime mailbox: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Claude1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "mailbox_ya_pendiente" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestDespachadorPipelineOperativoRespetaBloqueoCuota(t *testing.T) {
	prepararDBTemporalCmd(t)
	_, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("set estado_cuota: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Claude1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "cuota_bloqueada" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestDespachadorPipelineOperativoIncluyeWriteSetEnMailboxPremium(t *testing.T) {
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
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Codex1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			WriteSet:        []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"},
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nil")
	}
	pendiente := "pendiente"
	toAgente := "Codex1"
	items, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &toAgente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("mailbox inesperada: %+v", items)
	}
	if !strings.Contains(items[0].PayloadJSON, `"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"]`) {
		t.Fatalf("payload sin write_set: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `WRITE_SET: cmd/controlplane_support.go, db/controlplane_entities.go`) {
		t.Fatalf("instruction sin write_set: %s", items[0].PayloadJSON)
	}
}

func TestDespachadorPipelineOperativoNoSobrecargaAgenteEnOtroProyecto(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoActualID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto actual: %v", err)
	}
	proyectoViejoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "tools",
		Nombre:  "Tools",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto viejo: %v", err)
	}
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Claude1", proyectoViejoID, "ocupado"); err != nil {
		t.Fatalf("activar asignacion previa: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Claude1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "agente_ocupado_otro_proyecto" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	pendiente := "pendiente"
	toAgente := "Claude1"
	items, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &toAgente, ProyectoID: &proyectoActualID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("no deberia encolar mailbox para proyecto actual: %+v", items)
	}
}

func TestDespachadorPipelineOperativoRespetaMailboxPendienteEnOtroProyecto(t *testing.T) {
	prepararDBTemporalCmd(t)
	proyectoActualID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto actual: %v", err)
	}
	proyectoViejoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "tools",
		Nombre:  "Tools",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto viejo: %v", err)
	}
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"kind":   "pipeline_local",
		"accion": "continuar_trabajo",
	})
	if _, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoViejoID,
		Kind:        "pipeline_local",
		PayloadJSON: string(payload),
	}); err != nil {
		t.Fatalf("send runtime mailbox: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Claude1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "agente_ocupado_otro_proyecto" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	pendiente := "pendiente"
	toAgente := "Claude1"
	items, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &toAgente, ProyectoID: &proyectoActualID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("no deberia encolar mailbox para proyecto actual: %+v", items)
	}
}

func TestDespachadorPipelineOperativoNoEncolaStartSiYaHayRuntimeActivo(t *testing.T) {
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
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         t.TempDir(),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.GetRuntimeHandleBySesionID(sesion.ID); err != nil {
		t.Fatalf("get handle: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Claude1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "mailbox_encolado_runtime_existente" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	agente := "Claude1"
	orders, err := runtimesService.ListRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "start" {
			t.Fatalf("no deberia encolar start con runtime activo: %+v", order)
		}
	}
}

func TestDespachadorPipelineOperativoBloqueaRuntimeAmbiguo(t *testing.T) {
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
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionA, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         t.TempDir(),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion A: %v", err)
	}
	handleA, err := db.GetRuntimeHandleBySesionID(sesionA.ID)
	if err != nil || handleA == nil {
		t.Fatalf("get handle A: %+v err=%v", handleA, err)
	}
	sesionB, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         t.TempDir(),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion B: %v", err)
	}
	handleB, err := db.GetRuntimeHandleBySesionID(sesionB.ID)
	if err != nil || handleB == nil {
		t.Fatalf("get handle B: %+v err=%v", handleB, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo' WHERE id IN (?, ?)`, handleA.ID, handleB.ID); err != nil {
		t.Fatalf("forzar handles activos: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Claude1",
			TareaObjetivoID: 530,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "runtime_ambiguo" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestDespachadorPipelineOperativoSolicitaMergeEnIntegracion(t *testing.T) {
	rutaWorktree := prepararWorktreeGitDispatcherTest(t)
	store := &fakeGitStorePipelineDispatcher{
		projectID: 9,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   9,
			ProyectoSlug: "orquestador",
			Agente:       "Codex1",
			RutaAbs:      rutaWorktree,
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Estado:       "activa",
		}},
	}
	anterior := gitService
	gitService = gitaplicacion.NewService(store)
	t.Cleanup(func() { gitService = anterior })

	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "integracion",
			Carril:          "determinista_app",
			TareaObjetivoID: 12,
			TareaObjetivo:   "Integrar parser",
			AgenteTarea:     "Codex1",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "merge_solicitado" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.saved == nil {
		t.Fatalf("merge no guardado")
	}
	if store.saved.SourceBranch != "orq/orquestador/codex1" || store.saved.TargetBranch != "main" {
		t.Fatalf("ramas inesperadas: %+v", store.saved)
	}
}

func TestDespachadorPipelineOperativoReutilizaMergePendienteEnIntegracion(t *testing.T) {
	rutaWorktree := prepararWorktreeGitDispatcherTest(t)
	store := &fakeGitStorePipelineDispatcher{
		projectID: 9,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   9,
			ProyectoSlug: "orquestador",
			Agente:       "Codex1",
			RutaAbs:      rutaWorktree,
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Estado:       "activa",
		}},
		merges: []*db.GitMergeRequest{{
			ID:           91,
			ProyectoID:   9,
			ProyectoSlug: "orquestador",
			SourceBranch: "orq/orquestador/codex1",
			TargetBranch: "main",
			Estado:       "pendiente",
		}},
	}
	anterior := gitService
	gitService = gitaplicacion.NewService(store)
	t.Cleanup(func() { gitService = anterior })

	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "integracion",
			Carril:          "determinista_app",
			TareaObjetivoID: 12,
			TareaObjetivo:   "Integrar parser",
			AgenteTarea:     "Codex1",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "merge_ya_pendiente" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.saved != nil {
		t.Fatalf("no deberia guardar un merge nuevo: %+v", store.saved)
	}
}
