package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestDespachadorPipelineOperativoNoBloqueaCuotaVisibleFrescaAunqueEstadoPersistidoSeaAgotado(t *testing.T) {
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         t.TempDir(),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='agotado', motivo_pausa='Cuota diaria agotada' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("set estado_cuota legacy: %v", err)
	}
	remaining := int64(7200)
	reset := time.Now().UTC().Add(3 * time.Hour)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		ResetAt:          &reset,
		RemainingSeconds: &remaining,
		BudgetSource:     "codex_profile_status",
		RawSnapshotJSON:  `{"session_usage":{"remaining_seconds":7200},"rate_limits":{"primary":{"used_percent":39}}}`,
		CheckedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto visible: %v", err)
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
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nil")
	}
	if resultado.Estado == "cuota_bloqueada" || resultado.Estado == "cuota_bloqueada_con_mailbox_pendiente" {
		t.Fatalf("no deberia bloquear por cuota visible fresca: %+v", resultado)
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas: strings.Join([]string{
			"autonomia:microrefactor_loop",
			"fork_funcion_v1:",
			"funcion_objetivo: procesarRuntimeMailboxSessionResumeBatchConMailbox",
			"write_set: cmd/controlplane_support.go, db/controlplane_entities.go",
			"modelos_candidatos: claude-sonnet, gemini-2.5-pro",
			"materia: arquitectura",
			"fork_lines: 2",
			"selected_models: qwen, gemma",
			"decision_mode: auto",
			"decision_reason: orquesta decide fork automático · materia=arquitectura · lineas=2 · modelos=qwen, gemma",
			"preservar_arquitectura: true",
		}, "\n"),
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Codex1",
			TareaObjetivoID: tareaID,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			WriteSet:        []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"},
			SimbolosFoco:    "procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente",
			TestsMinimos:    "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
			FinishApp:       true,
			Paralelismo: &capacidadapp.PoliticaParalelismoPipelineLocal{
				PuedeAbrirSubagentes: true,
				MaxSubagentes:        2,
				Motivo:               "frente amplio con write-set particionable",
			},
			Motivo: "microrefactor_loop",
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
	var payload map[string]any
	if err := json.Unmarshal([]byte(items[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if !strings.Contains(items[0].PayloadJSON, `"write_set":["cmd/controlplane_support.go","db/controlplane_entities.go"]`) {
		t.Fatalf("payload sin write_set: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `"simbolos_foco":"procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente"`) {
		t.Fatalf("payload sin simbolos_foco: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `"tests_minimos":"go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'"`) {
		t.Fatalf("payload sin tests_minimos: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `"paralelismo":{"max_subagentes":2,"motivo":"frente amplio con write-set particionable","puede_abrir_subagentes":true}`) &&
		!strings.Contains(items[0].PayloadJSON, `"paralelismo":{"puede_abrir_subagentes":true,"max_subagentes":2,"motivo":"frente amplio con write-set particionable"}`) {
		t.Fatalf("payload sin paralelismo: %s", items[0].PayloadJSON)
	}
	forkRaw, ok := payload["fork_funcion"].(map[string]any)
	if !ok {
		t.Fatalf("payload sin fork_funcion parseable: %s", items[0].PayloadJSON)
	}
	if strings.TrimSpace(stringFromAny(forkRaw["funcion_objetivo"])) != "procesarRuntimeMailboxSessionResumeBatchConMailbox" {
		t.Fatalf("fork_funcion objetivo inesperado: %+v", forkRaw)
	}
	if strings.TrimSpace(stringFromAny(forkRaw["schema_version"])) != "fork_funcion_v1" {
		t.Fatalf("fork_funcion schema_version inesperada: %+v", forkRaw)
	}
	if !boolFromAny(forkRaw["preservar_arquitectura"]) {
		t.Fatalf("fork_funcion sin preservar_arquitectura: %+v", forkRaw)
	}
	if !reflect.DeepEqual(stringSliceFromAny(forkRaw["write_set"]), []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"}) {
		t.Fatalf("fork_funcion write_set inesperado: %+v", forkRaw)
	}
	if !reflect.DeepEqual(stringSliceFromAny(forkRaw["modelos_candidatos"]), []string{"claude-sonnet", "gemini-2.5-pro"}) {
		t.Fatalf("fork_funcion modelos inesperados: %+v", forkRaw)
	}
	if strings.TrimSpace(stringFromAny(forkRaw["materia"])) != "arquitectura" || intFromAny(forkRaw["fork_lines"]) != 2 {
		t.Fatalf("fork_funcion metadata auto inesperada: %+v", forkRaw)
	}
	if !reflect.DeepEqual(stringSliceFromAny(forkRaw["selected_models"]), []string{"qwen", "gemma"}) {
		t.Fatalf("fork_funcion selected_models inesperados: %+v", forkRaw)
	}
	variantesRaw, ok := payload["variantes_candidatas"].([]any)
	if !ok || len(variantesRaw) != 2 {
		t.Fatalf("payload sin variantes_candidatas parseables: %s", items[0].PayloadJSON)
	}
	var modelos []string
	for i, raw := range variantesRaw {
		variante, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("variante %d no parseable: %+v", i, raw)
		}
		if estado := strings.TrimSpace(stringFromAny(variante["estado"])); estado != "pendiente" {
			t.Fatalf("variante %d estado inesperado: %+v", i, variante)
		}
		if !boolFromAny(variante["preservar_arquitectura"]) {
			t.Fatalf("variante %d sin preservar_arquitectura: %+v", i, variante)
		}
		if !reflect.DeepEqual(stringSliceFromAny(variante["write_set"]), []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"}) {
			t.Fatalf("variante %d write_set inesperado: %+v", i, variante)
		}
		modelos = append(modelos, strings.TrimSpace(stringFromAny(variante["modelo"])))
	}
	if !reflect.DeepEqual(modelos, []string{"qwen", "gemma"}) {
		t.Fatalf("variantes modelos inesperados: %+v", modelos)
	}
	if !strings.Contains(items[0].PayloadJSON, `WRITE_SET: cmd/controlplane_support.go, db/controlplane_entities.go`) {
		t.Fatalf("instruction sin write_set: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `Simbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente`) {
		t.Fatalf("instruction sin simbolos foco: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `Tests minimos: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'`) {
		t.Fatalf("instruction sin tests minimos: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `MODO: finish_app`) {
		t.Fatalf("instruction sin modo finish_app: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `PARALELISMO: puedes abrir hasta 2 subagentes`) {
		t.Fatalf("instruction sin paralelismo: %s", items[0].PayloadJSON)
	}
	if !strings.Contains(items[0].PayloadJSON, `Motivo paralelismo: frente amplio con write-set particionable`) {
		t.Fatalf("instruction sin motivo de paralelismo: %s", items[0].PayloadJSON)
	}
	accion := "pipeline_dispatch_fork_funcion"
	entidad := "tarea"
	audits, err := db.ListarAuditoria(db.FiltroAuditoria{
		Accion:    &accion,
		Entidad:   &entidad,
		EntidadID: &tareaID,
		Limite:    5,
	})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	if len(audits) == 0 {
		t.Fatalf("sin auditoria de fork_funcion para tarea=%d", tareaID)
	}
	if !strings.Contains(strings.TrimSpace(audits[0].Detalle), "funcion=procesarRuntimeMailboxSessionResumeBatchConMailbox") {
		t.Fatalf("auditoria sin funcion objetivo: %+v", audits[0])
	}
	for _, token := range []string{"modelos=qwen,gemma", "fork_lines=2", "decision_mode=auto"} {
		if !strings.Contains(strings.TrimSpace(audits[0].Detalle), token) {
			t.Fatalf("auditoria sin token %q: %+v", token, audits[0])
		}
	}
}

func TestDespachadorPipelineOperativoLanzaSubagentesParaSlicesDisjuntos(t *testing.T) {
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
	prevLauncher := pipelineSubagentLaunchFn
	t.Cleanup(func() {
		pipelineSubagentLaunchFn = prevLauncher
	})
	t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", "echo ok")
	var prompts []string
	var descriptions []string
	pipelineSubagentLaunchFn = func(req supervisorSubagentLaunchRequest) (*supervisorSubagentLaunchResult, error) {
		prompts = append(prompts, req.Prompt)
		descriptions = append(descriptions, req.Description)
		return &supervisorSubagentLaunchResult{Supervisor: "OpenClaw", Proyecto: req.Proyecto}, nil
	}

	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Codex1",
			TareaObjetivoID: 531,
			TareaObjetivo:   "Frente amplio del control plane",
			WriteSet: []string{
				"cmd/controlplane_support.go",
				"db/controlplane_entities.go",
				"cmd/controlplane_support_test.go",
				"db/controlplane_entities_test.go",
			},
			TestsMinimos: "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
			Paralelismo: &capacidadapp.PoliticaParalelismoPipelineLocal{
				PuedeAbrirSubagentes: true,
				MaxSubagentes:        2,
				Motivo:               "frente amplio con write-set particionable",
			},
			Motivo: "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.SubagentsLaunched != 2 {
		t.Fatalf("subagentes lanzados inesperados: %+v", resultado)
	}
	if len(prompts) != 2 || len(descriptions) != 2 {
		t.Fatalf("launches inesperados prompts=%d descriptions=%d", len(prompts), len(descriptions))
	}
	if !strings.Contains(prompts[0], "Trabaja solo dentro de este write_set disjunto:") {
		t.Fatalf("prompt sin contrato de slice: %s", prompts[0])
	}
	if strings.Contains(prompts[0], "cmd/controlplane_support.go, db/controlplane_entities.go, cmd/controlplane_support_test.go, db/controlplane_entities_test.go") {
		t.Fatalf("prompt no deberia contener el write_set completo sin particionar: %s", prompts[0])
	}
	slices := pipelineWriteSetSlices([]string{
		"cmd/controlplane_support.go",
		"db/controlplane_entities.go",
		"cmd/controlplane_support_test.go",
		"db/controlplane_entities_test.go",
	}, 2)
	if len(slices) != 2 {
		t.Fatalf("slices inesperados: %+v", slices)
	}
	if !reflect.DeepEqual(slices[0], []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"}) &&
		!reflect.DeepEqual(slices[0], []string{"cmd/controlplane_support.go", "db/controlplane_entities_test.go"}) {
		t.Fatalf("slice 0 inesperado: %+v", slices[0])
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
}

func TestDespachadorPipelineOperativoLanzaSubagentesForkFuncionParaModelosSeleccionados(t *testing.T) {
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Refactor de fork multi-modelo",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas: strings.Join([]string{
			"fork_funcion_v1:",
			"funcion_objetivo: procesarRuntimeMailboxSessionResumeBatchConMailbox",
			"write_set: cmd/controlplane_support.go, db/controlplane_entities.go",
			"modelos_candidatos: claude-sonnet, gemini-2.5-pro",
			"materia: arquitectura",
			"fork_lines: 2",
			"selected_models: qwen, gemma",
			"decision_mode: auto",
			"decision_reason: orquesta decide fork automático · materia=arquitectura · lineas=2 · modelos=qwen, gemma",
			"preservar_arquitectura: true",
		}, "\n"),
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	prevLauncher := pipelineSubagentLaunchFn
	t.Cleanup(func() {
		pipelineSubagentLaunchFn = prevLauncher
	})
	t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", "echo ok")
	var models []string
	var sources []string
	var prompts []string
	pipelineSubagentLaunchFn = func(req supervisorSubagentLaunchRequest) (*supervisorSubagentLaunchResult, error) {
		models = append(models, strings.TrimSpace(req.Model))
		sources = append(sources, strings.TrimSpace(fmt.Sprint(req.Metadata["parallel_mode"])))
		prompts = append(prompts, req.Prompt)
		return &supervisorSubagentLaunchResult{}, nil
	}
	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Codex1",
			TareaObjetivoID: tareaID,
			TareaObjetivo:   "Refactor de fork multi-modelo",
			WriteSet:        []string{"cmd/controlplane_support.go", "db/controlplane_entities.go"},
			TestsMinimos:    "go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*'",
			Motivo:          "fork_funcion",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nil")
	}
	if resultado.SubagentsLaunched != 2 {
		t.Fatalf("subagents lanzados inesperados: %+v", resultado)
	}
	if !reflect.DeepEqual(models, []string{"qwen", "gemma"}) {
		t.Fatalf("models launch inesperados: %+v", models)
	}
	for _, source := range sources {
		if source != "fork_funcion" {
			t.Fatalf("parallel_mode inesperado: %+v", sources)
		}
	}
	for _, token := range []string{"TRABAJO DE FORK DE FUNCION EN PARALELO.", "Modelo objetivo: qwen", "Funcion objetivo: procesarRuntimeMailboxSessionResumeBatchConMailbox", "Numero de lineas de fork decidido por Orquesta: 2"} {
		if !strings.Contains(prompts[0], token) {
			t.Fatalf("prompt fork sin token %q:\n%s", token, prompts[0])
		}
	}
}

func TestDespachadorPipelineOperativoAseguraOwnershipDeTareaPremium(t *testing.T) {
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:microrefactor_loop",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "orquesta"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "orquesta"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "implementacion",
			Carril:          "premium_worktree",
			PerfilTarea:     "implementacion",
			AgenteSugerido:  "Codex1",
			TareaObjetivoID: tareaID,
			TareaObjetivo:   "Micro-refactorización cíclica del control plane",
			Motivo:          "microrefactor_loop",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil {
		t.Fatalf("resultado nil")
	}
	tarea, err := tareasService.Get(tareaID)
	if err != nil || tarea == nil {
		t.Fatalf("get tarea: %+v err=%v", tarea, err)
	}
	if tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Codex1" {
		t.Fatalf("ownership de tarea inesperado: %+v", tarea)
	}
	if tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("estado de tarea inesperado: %+v", tarea)
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
