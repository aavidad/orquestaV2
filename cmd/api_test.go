/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/i18n"
	"orquesta/internal/a2ui"
	"orquesta/reviewapp"
	"orquesta/runtimeagente"
	"orquesta/supervisionapp"
	"orquesta/tareasapp"
)

func TestCuentaPresupuestoDesdeAgenteUsaObservedUsageCuandoNoHayCuotaReal(t *testing.T) {
	tokens := int64(1570)
	cost := 0.042
	msgs := 3
	turns := 1
	key, item := cuentaPresupuestoDesdeAgente(&db.Agente{
		Nombre:                "Codex6",
		CuentaEmail:           "claude@example.com",
		PresupuestoFuente:     "claude_rust_session_observed",
		ObservedUsageTokens:   &tokens,
		ObservedUsageCostUSD:  &cost,
		ObservedUsageMessages: &msgs,
		ObservedUsageTurns:    &turns,
		ObservedSessionPath:   "/tmp/.claude/sessions/session-1.json",
	})
	if key != "claude@example.com" {
		t.Fatalf("cuenta clave inesperada: %q", key)
	}
	if item.Criterio != "observed_usage" {
		t.Fatalf("criterio inesperado: %+v", item)
	}
	rank, value := cuentaPresupuestoOrden(item)
	if rank != 0 || value != float64(tokens) {
		t.Fatalf("orden inesperado: rank=%d value=%v", rank, value)
	}
	if item.ObservedSessionPath != "/tmp/.claude/sessions/session-1.json" || item.ObservedUsageMessages == nil || *item.ObservedUsageMessages != 3 {
		t.Fatalf("payload observed inesperado: %+v", item)
	}
}

func TestCompactOpenClawAgentsIncluyeCuentaCanonica(t *testing.T) {
	items := compactOpenClawAgents([]*db.Agente{{
		Nombre:        "Codex7",
		CuentaID:      "acc-codex-7",
		CuentaEmail:   "shared@example.com",
		CuentaUsuario: "Codex7",
		EstadoCuota:   "activo",
	}}, nil, nil)
	if len(items) != 1 {
		t.Fatalf("items inesperados: %+v", items)
	}
	if items[0].CuentaID != "acc-codex-7" || items[0].CuentaEmail != "shared@example.com" || items[0].CuentaUsuario != "Codex7" {
		t.Fatalf("cuenta compacta inesperada: %+v", items[0])
	}
}

func TestAPIAgentesObservarCuentaActualizaCuentas(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex7/observar-cuenta", bytes.NewBufferString(`{"email":"berserk@avidad.com","usuario":"berserk","fuente":"manual_observed_identity"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	apiRouterAgentes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST observar-cuenta status=%d body=%s", rec.Code, rec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/agentes/cuentas", nil)
	getRec := httptest.NewRecorder()
	apiHandlerAgentesCuentas(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET cuentas status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	var resp apiAgentesCuentasResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode cuentas: %v", err)
	}
	var item *apiAgenteCuentaItem
	for i := range resp.Agentes {
		if strings.EqualFold(resp.Agentes[i].Nombre, "Codex7") {
			item = &resp.Agentes[i]
			break
		}
	}
	if item == nil {
		t.Fatalf("Codex7 no aparece en cuentas: %+v", resp.Agentes)
	}
	if item.CuentaEmail != "berserk@avidad.com" || item.CuentaUsuario != "berserk" {
		t.Fatalf("cuenta inesperada: %+v", item)
	}
}

func TestAlignOpenClawSessionCandidatesWithStatusPromueveActivosCanonicos(t *testing.T) {
	candidates := []apiOpenClawSessionCandidate{
		{Agente: "Codex3", Activo: false, ExternalSessionID: "sess-codex3-1"},
		{Agente: "Codex4", Activo: false, ExternalSessionID: "sess-codex4-1"},
	}
	aligned := alignOpenClawSessionCandidatesWithStatus(candidates, []*db.Agente{
		{Nombre: "Codex3", Activo: true},
	})
	if !aligned[0].Activo {
		t.Fatalf("Codex3 debe salir activo tras alinear con status: %#v", aligned)
	}
	if aligned[1].Activo {
		t.Fatalf("Codex4 no debe promocionarse sin estar en agentesActivos: %#v", aligned)
	}
}

func TestAlignSupervisorObservedSessionsWithStatusPromueveActivosCanonicos(t *testing.T) {
	items := []*supervisorObservedAgentSessionSummary{
		{Agente: "Codex3", Activo: false, ExternalSessionID: "sess-codex3-1"},
		{Agente: "Codex4", Activo: false, ExternalSessionID: "sess-codex4-1"},
	}
	aligned := alignSupervisorObservedSessionsWithStatus(items, []*db.Agente{
		{Nombre: "Codex3", Activo: true},
	})
	if !aligned[0].Activo {
		t.Fatalf("Codex3 debe salir activo tras alinear observed sessions: %#v", aligned)
	}
	if aligned[1].Activo {
		t.Fatalf("Codex4 no debe promocionarse sin estar en agentesActivos: %#v", aligned)
	}
}

func TestParseGitWorktreeListPorcelain(t *testing.T) {
	raw := strings.Join([]string{
		"worktree /tmp/orquesta",
		"HEAD abcdef0123456789",
		"branch refs/heads/main",
		"",
		"worktree /tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
		"HEAD 1234567890abcdef",
		"branch refs/heads/orq-orquesta-codex3",
		"",
	}, "\n")
	refs := parseGitWorktreeListPorcelain(raw)
	if len(refs) != 2 {
		t.Fatalf("refs inesperadas: %#v", refs)
	}
	if refs[0].Path != "/tmp/orquesta" || refs[0].Head != "abcdef0123456789" {
		t.Fatalf("ref principal inesperada: %#v", refs[0])
	}
	if refs[1].Path != "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3" || refs[1].Head != "1234567890abcdef" {
		t.Fatalf("ref secundaria inesperada: %#v", refs[1])
	}
}

func TestBuildOpenClawWorktreeDriftFromRefs(t *testing.T) {
	worktrees := []*coordinacion.Worktree{
		{Agent: "Codex3", Path: "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3", Branch: "orq-orquesta-codex3"},
		{Agent: "Codex4", Path: "/tmp/orquesta/.orquesta-worktrees/orquesta-codex4", Branch: "orq-orquesta-codex4"},
	}
	drift := buildOpenClawWorktreeDriftFromRefs(worktrees, "aaaaaaaa11111111", map[string]string{
		"/tmp/orquesta/.orquesta-worktrees/orquesta-codex3": "bbbbbbbb22222222",
		"/tmp/orquesta/.orquesta-worktrees/orquesta-codex4": "aaaaaaaa11111111",
	}, map[string]struct{}{
		"codex3": {},
		"codex4": {},
	})
	if len(drift) != 1 {
		t.Fatalf("drift inesperado: %#v", drift)
	}
	if drift[0].Agente != "Codex3" || drift[0].CurrentHead != "bbbbbbbb" || drift[0].ExpectedHead != "aaaaaaaa" {
		t.Fatalf("item drift inesperado: %#v", drift[0])
	}
}

func TestOpenClawOperatorReuseReviewSnapshotHelpers(t *testing.T) {
	review := map[string]any{
		"normalized_events": []openClawNormalizedEvent{{Source: "merge", NormalizedEvent: "merge.pending"}},
		"thread_sessions": map[string]any{
			"observed_agent_sessions": []*supervisorObservedAgentSessionSummary{{Agente: "Codex3"}},
		},
		"pipeline_state": map[string]any{"phase": "supervisor-loop"},
		"worktree_drift": []apiOpenClawWorktreeDrift{{Agente: "Codex4", CommitsBehind: 2}},
		"queue_summary":  apiOpenClawQueueSummary{Total: 3, Safe: 2, Manual: 1},
	}

	events := openClawEventsFromReviewSnapshot(review)
	if len(events) != 1 || events[0].NormalizedEvent != "merge.pending" {
		t.Fatalf("normalized_events inesperado: %#v", events)
	}

	threads := supervisorThreadsFromReviewSnapshot(review)
	observed, _ := threads["observed_agent_sessions"].([]*supervisorObservedAgentSessionSummary)
	if len(observed) != 1 || observed[0].Agente != "Codex3" {
		t.Fatalf("thread_sessions inesperado: %#v", threads)
	}

	pipeline := supervisorPipelineFromReviewSnapshot(review)
	if got, _ := pipeline["phase"].(string); got != "supervisor-loop" {
		t.Fatalf("pipeline_state inesperado: %#v", pipeline)
	}

	drift := openClawWorktreeDriftFromReviewSnapshot(review)
	if len(drift) != 1 || drift[0].Agente != "Codex4" || drift[0].CommitsBehind != 2 {
		t.Fatalf("worktree_drift inesperado: %#v", drift)
	}

	queue := buildOpenClawQueueSummaryFromReviewSnapshot(review)
	if queue.Total != 3 || queue.Safe != 2 || queue.Manual != 1 {
		t.Fatalf("queue_summary inesperado: %#v", queue)
	}
}

func TestListarSupervisorModuleConflictsFromTasksReutilizaEstadoYaCargado(t *testing.T) {
	conflicts := listarSupervisorModuleConflictsFromTasks([]tareaLite{
		{ID: 410, Modulo: "runtime", Agente: "Codex3"},
		{ID: 411, Modulo: "runtime", Agente: "Codex4"},
		{ID: 412, Modulo: "web", Agente: "Codex4"},
	})
	if len(conflicts) != 1 {
		t.Fatalf("conflicts inesperados: %#v", conflicts)
	}
	if conflicts[0].Modulo != "runtime" || len(conflicts[0].Tareas) != 2 {
		t.Fatalf("conflict inesperado: %#v", conflicts[0])
	}
}

func TestBuildProyectoPendingMailboxFiltraPorProyectoYAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-mailbox",
		Nombre:  "Orquestador Mailbox",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	otroProyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "otro-mailbox",
		Nombre:  "Otro Mailbox",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert otro proyecto: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex3",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"supervisor_action":"revisar_worktree_desfasada"}`,
	}); err != nil {
		t.Fatalf("encolar mailbox proyecto: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex3",
		ProyectoID:  &otroProyectoID,
		Kind:        "nudge",
		PayloadJSON: `{}`,
	}); err != nil {
		t.Fatalf("encolar mailbox otro proyecto: %v", err)
	}

	items, err := buildProyectoPendingMailbox(proyectoID, map[string]struct{}{"codex3": {}})
	if err != nil {
		t.Fatalf("buildProyectoPendingMailbox: %v", err)
	}
	if len(items) != 1 || items[0].Agente != "Codex3" || items[0].Count != 1 {
		t.Fatalf("mailbox proyecto inesperada: %+v", items)
	}
	if items[0].SupervisorActionsCSV != "revisar_worktree_desfasada" {
		t.Fatalf("supervisor action inesperada: %+v", items[0])
	}
}

func TestFilterOpenClawWorktreeDriftByAgents(t *testing.T) {
	items := []apiOpenClawWorktreeDrift{
		{Agente: "Codex3", CommitsBehind: 4},
		{Agente: "Codex4", CommitsBehind: 2},
	}
	out := filterOpenClawWorktreeDriftByAgents(items, map[string]struct{}{"codex4": {}})
	if len(out) != 1 || out[0].Agente != "Codex4" {
		t.Fatalf("drift filtrada inesperada: %+v", out)
	}
}

func TestParseGitStatusPorcelainSummary(t *testing.T) {
	dirty, summary, files, overflow := parseGitStatusPorcelainSummary(" M cmd/api.go\n?? cmd/new_file.go\n")
	if !dirty {
		t.Fatalf("deberia marcar dirty")
	}
	if summary != "1 tracked · 1 untracked" {
		t.Fatalf("summary inesperado: %q", summary)
	}
	if len(files) != 2 || files[0] != "cmd/api.go" || files[1] != "cmd/new_file.go" || overflow != 0 {
		t.Fatalf("detalle inesperado: files=%v overflow=%d", files, overflow)
	}
}

var cmdTestDBMu sync.Mutex
var cmdTestBootstrapOnce sync.Once
var cmdTestBootstrapData []byte
var cmdTestBootstrapErr error

func cargarPlantillaDBCmdTest() ([]byte, error) {
	cmdTestBootstrapOnce.Do(func() {
		tmp, err := os.MkdirTemp("", "orquesta-cmd-db-template-*")
		if err != nil {
			cmdTestBootstrapErr = err
			return
		}
		defer os.RemoveAll(tmp)

		anteriorDB := os.Getenv("ORQUESTA_DB")
		anteriorDSN, teniaDSN := os.LookupEnv("ORQUESTA_DB_DSN")
		anteriorDriver, teniaDriver := os.LookupEnv("ORQUESTA_DB_DRIVER")
		anteriorBackend, teniaBackend := os.LookupEnv("ORQUESTA_DB_BACKEND")
		anteriorMaxOpenConns, teniaMaxOpenConns := os.LookupEnv("ORQUESTA_DB_MAX_OPEN_CONNS")
		anteriorBootstrap, teniaBootstrap := os.LookupEnv("ORQUESTA_DB_BOOTSTRAP")
		anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
		anteriorForceLocal, teniaForceLocal := os.LookupEnv("ORQUESTA_FORCE_LOCAL_DB")
		anteriorDisableServer, teniaDisableServer := os.LookupEnv("ORQUESTA_DISABLE_SERVER_CLIENT")
		defer func() {
			db.Close()
			db.DB = nil
			if anteriorDB == "" {
				_ = os.Unsetenv("ORQUESTA_DB")
			} else {
				_ = os.Setenv("ORQUESTA_DB", anteriorDB)
			}
			if teniaDSN {
				_ = os.Setenv("ORQUESTA_DB_DSN", anteriorDSN)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_DSN")
			}
			if teniaDriver {
				_ = os.Setenv("ORQUESTA_DB_DRIVER", anteriorDriver)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
			}
			if teniaBackend {
				_ = os.Setenv("ORQUESTA_DB_BACKEND", anteriorBackend)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
			}
			if teniaMaxOpenConns {
				_ = os.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", anteriorMaxOpenConns)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS")
			}
			if teniaBootstrap {
				_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", anteriorBootstrap)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_BOOTSTRAP")
			}
			if anteriorRoot == "" {
				_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
			} else {
				_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
			}
			if teniaForceLocal {
				_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", anteriorForceLocal)
			} else {
				_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
			}
			if teniaDisableServer {
				_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", anteriorDisableServer)
			} else {
				_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
			}
		}()

		dbPath := filepath.Join(tmp, "orquesta-cmd-template.db")
		_ = os.Setenv("ORQUESTA_DB", dbPath)
		_ = os.Unsetenv("ORQUESTA_DB_DSN")
		_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
		_ = os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS")
		_ = os.Unsetenv("ORQUESTA_DB_BOOTSTRAP")
		_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp)
		_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1")
		_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1")

		if err := db.Open(); err != nil {
			cmdTestBootstrapErr = err
			return
		}
		if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
			cmdTestBootstrapErr = err
			return
		}
		db.Close()
		cmdTestBootstrapData, cmdTestBootstrapErr = os.ReadFile(dbPath)
	})
	return cmdTestBootstrapData, cmdTestBootstrapErr
}

func prepararDBTemporalCmd(t *testing.T) string {
	t.Helper()
	cmdTestDBMu.Lock()
	agentesService = agentesapp.NewService(agentesapp.Repository{}, capacidadService)
	capacidadService.SetAgentResolver(resolvedorAgentePipelineOperativo{rowsProvider: agentesService})
	runtimeMailboxSyncSupervisedHandleFn = db.SincronizarRuntimeHandleSupervisado
	runtimeMailboxLoadWorkerSnapshotFn = runtimeagente.LoadWorkerSnapshotFromMetadataJSON
	runtimeMailboxBuildInteractiveInstructionFn = construirInstruccionMailboxInteractivo
	wakeRuntimeOrdersAfterMailbox = func() bool { return false }
	processRuntimeOrdersWakeFallback = func() {}
	processRuntimeMailboxWakeFallback = func() {}
	resetStatusSnapshotCache()
	resetControlPlaneConfigCache()
	resetRuntimeBudgetObservationBackgroundGate()
	resetAutonomiaActiveSessionsObservationGate()
	resetRuntimeMailboxReevaluationGate()
	resetAutonomiaIdleAutoassignGate()
	autonomiaContinueNudgeGate.Reset()
	resetAutonomiaDegradedTaskGate()
	resetPresupuestoPrimerUsoSesionGate()
	db.ResetRuntimeHandlesHotCache()

	anteriorDB := os.Getenv("ORQUESTA_DB")
	anteriorDSN, teniaDSN := os.LookupEnv("ORQUESTA_DB_DSN")
	anteriorDriver, teniaDriver := os.LookupEnv("ORQUESTA_DB_DRIVER")
	anteriorBackend, teniaBackend := os.LookupEnv("ORQUESTA_DB_BACKEND")
	anteriorMaxOpenConns, teniaMaxOpenConns := os.LookupEnv("ORQUESTA_DB_MAX_OPEN_CONNS")
	anteriorBootstrap, teniaBootstrap := os.LookupEnv("ORQUESTA_DB_BOOTSTRAP")
	anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
	anteriorForceLocal, teniaForceLocal := os.LookupEnv("ORQUESTA_FORCE_LOCAL_DB")
	anteriorDisableServer, teniaDisableServer := os.LookupEnv("ORQUESTA_DISABLE_SERVER_CLIENT")
	t.Cleanup(func() {
		agentesService = agentesapp.NewService(agentesapp.Repository{}, capacidadService)
		capacidadService.SetAgentResolver(resolvedorAgentePipelineOperativo{rowsProvider: agentesService})
		runtimeMailboxSyncSupervisedHandleFn = db.SincronizarRuntimeHandleSupervisado
		runtimeMailboxLoadWorkerSnapshotFn = runtimeagente.LoadWorkerSnapshotFromMetadataJSON
		runtimeMailboxBuildInteractiveInstructionFn = construirInstruccionMailboxInteractivo
		wakeRuntimeOrdersAfterMailbox = wakeControlPlaneRuntimeOrders
		processRuntimeOrdersWakeFallback = runRuntimeOrdersWakeFallback
		processRuntimeMailboxWakeFallback = runRuntimeMailboxWakeFallback
		resetStatusSnapshotCache()
		resetControlPlaneConfigCache()
		resetRuntimeBudgetObservationBackgroundGate()
		resetAutonomiaActiveSessionsObservationGate()
		resetRuntimeMailboxReevaluationGate()
		resetAutonomiaIdleAutoassignGate()
		autonomiaContinueNudgeGate.Reset()
		resetAutonomiaDegradedTaskGate()
		resetPresupuestoPrimerUsoSesionGate()
		db.ResetRuntimeHandlesHotCache()
		db.Close()
		db.DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
		if teniaDSN {
			_ = os.Setenv("ORQUESTA_DB_DSN", anteriorDSN)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_DSN")
		}
		if teniaDriver {
			_ = os.Setenv("ORQUESTA_DB_DRIVER", anteriorDriver)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		}
		if teniaBackend {
			_ = os.Setenv("ORQUESTA_DB_BACKEND", anteriorBackend)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
		}
		if teniaMaxOpenConns {
			_ = os.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", anteriorMaxOpenConns)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS")
		}
		if teniaBootstrap {
			_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", anteriorBootstrap)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_BOOTSTRAP")
		}
		if anteriorRoot == "" {
			_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
		} else {
			_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
		}
		if teniaForceLocal {
			_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", anteriorForceLocal)
		} else {
			_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
		}
		if teniaDisableServer {
			_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", anteriorDisableServer)
		} else {
			_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
		}
		cmdTestDBMu.Unlock()
	})

	db.Close()
	db.DB = nil
	resetStatusSnapshotCache()

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-api-test.db")
	plantilla, err := cargarPlantillaDBCmdTest()
	if err != nil {
		t.Fatalf("cargar plantilla db cmd test: %v", err)
	}
	if err := os.WriteFile(dbPath, plantilla, 0o600); err != nil {
		t.Fatalf("write plantilla db cmd test: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB", dbPath); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_DSN"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_DSN: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_DRIVER"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_DRIVER: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_BACKEND"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_BACKEND: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_MAX_OPEN_CONNS: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_BOOTSTRAP"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_BOOTSTRAP: %v", err)
	}
	if err := os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp); err != nil {
		t.Fatalf("setenv ORQUESTA_WORKSPACE_ROOT: %v", err)
	}
	if err := os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1"); err != nil {
		t.Fatalf("setenv ORQUESTA_FORCE_LOCAL_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1"); err != nil {
		t.Fatalf("setenv ORQUESTA_DISABLE_SERVER_CLIENT: %v", err)
	}
	if err := db.Open(); err != nil {
		t.Fatalf("open db temporal: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base cmd: %v", err)
	}
	registrarRosterBaseCmdTest(t)
	return tmp
}

func registrarRosterBaseCmdTest(t *testing.T) {
	t.Helper()
	roster := []struct {
		nombre string
		rol    string
	}{
		{nombre: "alberto", rol: "admin"},
		{nombre: "Codex1", rol: "programador"},
		{nombre: "Codex2", rol: "programador"},
	}
	for _, item := range roster {
		if err := agentesService.RegisterAgent(item.nombre, item.rol); err != nil {
			t.Fatalf("registrar agente base %s: %v", item.nombre, err)
		}
	}
}

func TestAPIAgentesListaJSON(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); len(got) < 16 || got[:16] != "application/json" {
		t.Fatalf("content-type inesperado: %s", got)
	}
}

func TestAPIAgentesIgnoraSnapshotStatusStaleYLeeCatalogoCanonico(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	if err := db.RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Gemma1: %v", err)
	}
	if err := db.RetirarAgente("Gemma1"); err != nil {
		t.Fatalf("RetirarAgente Gemma1: %v", err)
	}

	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{{
			Nombre:     "Gemma1",
			Rol:        "programador",
			Activo:     true,
			Habilitado: true,
		}},
		Generado: now.Format(time.RFC3339),
	}, now, time.Minute)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	if len(resp.Agentes) == 0 {
		t.Fatalf("respuesta sin agentes: %s", rec.Body.String())
	}
	var gemma *db.Agente
	for _, agente := range resp.Agentes {
		if agente != nil && agente.Nombre == "Gemma1" {
			gemma = agente
			break
		}
	}
	if gemma == nil {
		t.Fatalf("Gemma1 no expuesto via /api/agentes: %+v", resp.Agentes)
	}
	if gemma.Habilitado || gemma.Activo {
		t.Fatalf("/api/agentes no debe heredar flags stale de status: %+v", gemma)
	}
}

func TestAPIRetirarAgenteInvalidaSnapshotStatus(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	if err := db.RegistrarAgente("temporal", "programador"); err != nil {
		t.Fatalf("RegistrarAgente temporal: %v", err)
	}
	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{{
			Nombre:     "temporal",
			Rol:        "programador",
			Activo:     true,
			Habilitado: true,
		}},
		Generado: now.Format(time.RFC3339),
	}, now, time.Minute)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recRetirar := httptest.NewRecorder()
	reqRetirar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/retirar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRetirar, reqRetirar)
	if recRetirar.Code != http.StatusOK {
		t.Fatalf("status retirar inesperado: %d body=%s", recRetirar.Code, recRetirar.Body.String())
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	var temporal *db.Agente
	for _, agente := range statusResp.Agentes {
		if agente != nil && agente.Nombre == "temporal" {
			temporal = agente
			break
		}
	}
	if temporal == nil {
		t.Fatalf("temporal no expuesto via /api/status: %+v", statusResp.Agentes)
	}
	if temporal.Habilitado || temporal.Activo {
		t.Fatalf("/api/status no debe conservar snapshot stale tras retirar: %+v", temporal)
	}
}

func TestAPIServerExponeMetadatosDescubrimiento(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/server", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/server: %v", err)
	}
	if payload.Name != "orquesta" {
		t.Fatalf("name inesperado: %+v", payload)
	}
	if payload.StorageMode != "single-process" || payload.StorageDriver == "" || payload.SQLPlaceholder == "" {
		t.Fatalf("payload de descubrimiento incompleto: %+v", payload)
	}
	if len(payload.Capabilities) == 0 {
		t.Fatalf("capabilities vacias: %+v", payload)
	}
}

func TestAPIServerOperationalExponeResumenOperativo(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexOp", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/server/operational: %v", err)
	}
	if payload.RegisteredAgents < 1 {
		t.Fatalf("registeredAgents inesperado: %+v", payload)
	}
	if strings.TrimSpace(payload.State) == "" || strings.TrimSpace(payload.Reason) == "" {
		t.Fatalf("estado operativo incompleto: %+v", payload)
	}
	if payload.DispatchPending < 0 || payload.DispatchNotified < 0 || payload.DispatchFailed < 0 || payload.DispatchConfirmed < 0 {
		t.Fatalf("contadores dispatch invalidos: %+v", payload)
	}
	if payload.AutonomySupervising < 0 || payload.AutonomyContinuing < 0 || payload.AutonomyPending < 0 || payload.AutonomyConfirmed < 0 || payload.AutonomyHandoffs < 0 {
		t.Fatalf("contadores autonomia invalidos: %+v", payload)
	}
}

func TestAPIServerOperationalExponeHeadersOperativosPorDefecto(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevNow := statusNowFunc
	t.Cleanup(func() {
		statusNowFunc = prevNow
	})
	now := time.Date(2026, 4, 29, 9, 55, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Autonomia:         autonomiaResumen{WorkConfirmed: 1, ByKind: map[string]int{}},
		Generado:          now.Format(time.RFC3339),
	}, now, time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational"); got != "true" {
		t.Fatalf("header operational inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational-State"); got != "ready" {
		t.Fatalf("header state inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational-Reason"); got != "control_plane_responsive" {
		t.Fatalf("header reason inesperado: %q", got)
	}
}

func TestAPIServerOperationalStrictProbeDevuelve503CuandoNoEsOperativo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/server/operational?strict=1", nil)
	rec := httptest.NewRecorder()
	apiWriteServerOperational(rec, req, serverOperationalInfo{
		State:       "degraded",
		Operational: false,
		Reason:      "workers_require_manual_auth",
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational"); got != "false" {
		t.Fatalf("header operational inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational-State"); got != "degraded" {
		t.Fatalf("header state inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational-Reason"); got != "workers_require_manual_auth" {
		t.Fatalf("header reason inesperado: %q", got)
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode strict operational: %v", err)
	}
	if payload.Operational || payload.State != "degraded" || payload.Reason != "workers_require_manual_auth" {
		t.Fatalf("payload estricto inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalStrictProbeMantiene200SiSigueOperativo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/server/operational?probe=true", nil)
	rec := httptest.NewRecorder()
	apiWriteServerOperational(rec, req, serverOperationalInfo{
		State:       "idle",
		Operational: true,
		Reason:      "workers_quota_blocked",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Orquesta-Operational"); got != "true" {
		t.Fatalf("header operational inesperado: %q", got)
	}
	if got := rec.Header().Get("X-Orquesta-Operational-State"); got != "idle" {
		t.Fatalf("header state inesperado: %q", got)
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode strict operational idle: %v", err)
	}
	if !payload.Operational || payload.State != "idle" || strings.TrimSpace(payload.Reason) == "" {
		t.Fatalf("payload idle inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalStrictProbeReutilizaSnapshotStale(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevNow := statusNowFunc
	t.Cleanup(func() {
		statusNowFunc = prevNow
	})
	now := time.Date(2026, 4, 29, 10, 10, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 11, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Autonomia:         autonomiaResumen{ContinuidadPendiente: 1, WorkConfirmed: 1, ByKind: map[string]int{}},
		Generado:          now.Add(-15 * time.Second).Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational?strict=1", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Orquesta-Operational"); got != "true" {
		t.Fatalf("header operational inesperado: %q", got)
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode strict stale operational: %v", err)
	}
	if !payload.Operational || payload.State != "ready" || payload.ActiveAgents != 1 || payload.TasksInProgress != 1 {
		t.Fatalf("payload stale inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalExponeRiesgoCanonicoLigero(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevNow := statusNowFunc
	t.Cleanup(func() {
		statusNowFunc = prevNow
	})
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "Codex1"}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		AutonomyHighlights: []string{
			"integracion_bloqueada=9",
			"riesgo_top=infra(9)",
		},
		CriticalProjectRisk: &workspaceAutonomyProjectSummary{
			Project:    "infra",
			Blocking:   9,
			Highlights: []string{"riesgo=alto", "integracion_bloqueada=9", "runtime_orders=1"},
		},
		Generado: now.Format(time.RFC3339),
	}, now, time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/server/operational: %v", err)
	}
	if len(payload.AutonomyHighlights) != 2 || payload.AutonomyHighlights[0] != "integracion_bloqueada=9" {
		t.Fatalf("autonomyHighlights inesperados: %+v", payload.AutonomyHighlights)
	}
	if payload.CriticalProjectRisk == nil || payload.CriticalProjectRisk.Project != "infra" || payload.CriticalProjectRisk.Blocking != 9 {
		t.Fatalf("criticalProjectRisk inesperado: %+v", payload.CriticalProjectRisk)
	}
}

func TestAPIServerOperationalNormalizaRiesgoCanonicoDelDegradedBuilder(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevBuilder := apiServerOperationalBuilder
	prevTimeout := apiServerOperationalTimeout
	prevFallback := apiServerOperationalStatusFallback
	prevDegraded := apiServerOperationalDegradedBuilder
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	t.Cleanup(func() {
		apiServerOperationalBuilder = prevBuilder
		apiServerOperationalTimeout = prevTimeout
		apiServerOperationalStatusFallback = prevFallback
		apiServerOperationalDegradedBuilder = prevDegraded
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
	})

	apiStatusReadOnlyLiteFetcher = nil
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiServerOperationalTimeout = 20 * time.Millisecond
	apiServerOperationalStatusFallback = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiServerOperationalDegradedBuilder = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:       "degraded",
			Operational: false,
			Reason:      "status_temporarily_degraded",
			CriticalProjectRisk: &workspaceAutonomyProjectSummary{
				Project:    "infra",
				Blocking:   12,
				Highlights: []string{"riesgo=alto", "integracion_bloqueada=12", "runtime_orders=3"},
			},
		}
	}
	apiServerOperationalBuilder = func() (serverOperationalInfo, error) {
		time.Sleep(200 * time.Millisecond)
		return serverOperationalInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode degraded operational: %v", err)
	}
	if payload.CriticalProjectRisk == nil || payload.CriticalProjectRisk.Project != "infra" || payload.CriticalProjectRisk.Blocking != 12 {
		t.Fatalf("criticalProjectRisk inesperado: %+v", payload.CriticalProjectRisk)
	}
	if !containsStringWorkspace(payload.AutonomyHighlights, "integracion_bloqueada=12") || !containsStringWorkspace(payload.AutonomyHighlights, "riesgo_top=infra(12)") {
		t.Fatalf("autonomyHighlights canónicos inesperados: %+v", payload.AutonomyHighlights)
	}
}

func TestAPIServerOperationalReturnsDegradedPayloadWhenBuilderHangs(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevBuilder := apiServerOperationalBuilder
	prevTimeout := apiServerOperationalTimeout
	prevFallback := apiServerOperationalStatusFallback
	prevDegraded := apiServerOperationalDegradedBuilder
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	t.Cleanup(func() {
		apiServerOperationalBuilder = prevBuilder
		apiServerOperationalTimeout = prevTimeout
		apiServerOperationalStatusFallback = prevFallback
		apiServerOperationalDegradedBuilder = prevDegraded
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
	})

	apiStatusReadOnlyLiteFetcher = nil
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiServerOperationalTimeout = 20 * time.Millisecond
	apiServerOperationalStatusFallback = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiServerOperationalDegradedBuilder = func() serverOperationalInfo {
		return serverOperationalInfo{
			State:       "degraded",
			Operational: false,
			Reason:      "status_temporarily_degraded",
		}
	}
	apiServerOperationalBuilder = func() (serverOperationalInfo, error) {
		time.Sleep(200 * time.Millisecond)
		return serverOperationalInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode degraded operational: %v", err)
	}
	if payload.State != "degraded" || strings.TrimSpace(payload.Reason) == "" {
		t.Fatalf("payload degradado inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalUsesFreshSnapshotWhenBuilderHangs(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevBuilder := apiServerOperationalBuilder
	prevTimeout := apiServerOperationalTimeout
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevNow := statusNowFunc
	t.Cleanup(func() {
		apiServerOperationalBuilder = prevBuilder
		apiServerOperationalTimeout = prevTimeout
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusNowFunc = prevNow
	})

	apiStatusReadOnlyLiteFetcher = nil
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "Codex1"}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Generado:          now.Format(time.RFC3339),
	}, now, time.Minute)

	apiServerOperationalTimeout = 20 * time.Millisecond
	apiServerOperationalBuilder = func() (serverOperationalInfo, error) {
		time.Sleep(200 * time.Millisecond)
		return serverOperationalInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode fresh snapshot operational: %v", err)
	}
	if payload.State == "degraded" || payload.Reason == "status_temporarily_degraded" {
		t.Fatalf("deberia reutilizar snapshot fresca sin degradar: %+v", payload)
	}
	if payload.ActiveAgents != 1 || payload.WorkingAgents != 1 {
		t.Fatalf("payload operacional inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalUsesStaleSnapshotWhenNoFreshAvailable(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevBuilder := apiServerOperationalBuilder
	prevTimeout := apiServerOperationalTimeout
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevNow := statusNowFunc
	t.Cleanup(func() {
		apiServerOperationalBuilder = prevBuilder
		apiServerOperationalTimeout = prevTimeout
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusNowFunc = prevNow
	})

	apiStatusReadOnlyLiteFetcher = nil
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:           []*db.Agente{{Nombre: "Codex1"}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex1", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex1", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 1, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
		Generado:          now.Add(-10 * time.Second).Format(time.RFC3339),
	}, now.Add(-2*time.Minute), time.Second)

	apiServerOperationalTimeout = 20 * time.Millisecond
	apiServerOperationalBuilder = func() (serverOperationalInfo, error) {
		time.Sleep(200 * time.Millisecond)
		return serverOperationalInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode fast fallback operational: %v", err)
	}
	if payload.State == "degraded" || payload.Reason == "status_temporarily_degraded" {
		t.Fatalf("deberia reutilizar fallback ligero sin degradar: %+v", payload)
	}
	if payload.ActiveAgents != 1 || payload.WorkingAgents != 1 {
		t.Fatalf("payload operacional inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalUsesConfiguredDegradedBuilderWhenNoSnapshotNorReadOnly(t *testing.T) {
	prevBuilder := apiServerOperationalBuilder
	prevTimeout := apiServerOperationalTimeout
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	t.Cleanup(func() {
		apiServerOperationalBuilder = prevBuilder
		apiServerOperationalTimeout = prevTimeout
		apiStatusReadOnlyLiteFetcher = prevReadOnly
	})

	apiStatusReadOnlyLiteFetcher = nil
	apiServerOperationalTimeout = 20 * time.Millisecond
	apiServerOperationalBuilder = func() (serverOperationalInfo, error) {
		time.Sleep(200 * time.Millisecond)
		return serverOperationalInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode degraded operational: %v", err)
	}
	if payload.State != "idle" || payload.Reason != "no_active_workers" {
		t.Fatalf("deberia usar el degraded builder configurado sin snapshot/read-only: %+v", payload)
	}
}

func TestAPIServerOperationalUsesReadOnlyFallbackWhenFastFallbackUnavailable(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevBuilder := apiServerOperationalBuilder
	prevTimeout := apiServerOperationalTimeout
	prevFallback := apiServerOperationalStatusFallback
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	t.Cleanup(func() {
		apiServerOperationalBuilder = prevBuilder
		apiServerOperationalTimeout = prevTimeout
		apiServerOperationalStatusFallback = prevFallback
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
	})

	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{"en_progreso": 1}, []*db.Tarea{
			{ID: 9, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
		}, nil
	}
	apiServerOperationalStatusFallback = func() (apiStatusResponse, error) {
		if status, ok := fetchStatusReadOnlyLiteDirect(statusFastTimeout); ok {
			return status, nil
		}
		if status, ok := fetchStatusForOperationalFallback(statusFastTimeout); ok {
			return status, nil
		}
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiServerOperationalTimeout = 20 * time.Millisecond
	apiServerOperationalBuilder = func() (serverOperationalInfo, error) {
		time.Sleep(200 * time.Millisecond)
		return serverOperationalInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode read-only fallback operational: %v", err)
	}
	if payload.RegisteredAgents != 1 || payload.TasksInProgress != 1 || payload.WorkingAgents != 1 {
		t.Fatalf("payload operacional read-only inesperado: %+v", payload)
	}
}

func TestAPIServerOperationalIgnoraSnapshotEnTransicionYUsaReadOnlyFresco(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevNow := statusNowFunc
	t.Cleanup(func() {
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		statusNowFunc = prevNow
	})

	now := time.Date(2026, 4, 28, 19, 0, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Stale1", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Stale2", Activo: true, EstadoCuota: "activo"},
		},
		AgentesActivos: []*db.Agente{
			{Nombre: "Stale1", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Stale2", Activo: true, EstadoCuota: "activo"},
		},
		TareasEnProgreso: []tareaLite{
			{ID: 98, Estado: db.TareaEnProgreso, Agente: "Stale1"},
			{ID: 99, Estado: db.TareaEnProgreso, Agente: "Stale2"},
		},
		Autonomia: autonomiaResumen{ContinuidadPendiente: 1},
		Generado:  now.Format(time.RFC3339),
	}, now, time.Minute)

	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{"en_progreso": 1}, []*db.Tarea{
			{ID: 9, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	rec := httptest.NewRecorder()
	apiHandlerServerOperational(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverOperationalInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode operational: %v", err)
	}
	if payload.RegisteredAgents != 1 || payload.WorkingAgents != 1 || payload.TasksInProgress != 1 {
		t.Fatalf("payload operacional inesperado: %+v", payload)
	}
	if payload.State == "degraded" || payload.Reason == "status_temporarily_degraded" {
		t.Fatalf("no deberia degradar si hay read-only fresco disponible: %+v", payload)
	}
}

func TestFetchStatusForOperationalFallbackIgnoraSnapshotQueRequiereRefresh(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevNow := statusNowFunc
	t.Cleanup(func() {
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusNowFunc = prevNow
	})

	now := time.Date(2026, 4, 28, 18, 45, 0, 0, time.UTC)
	statusNowFunc = func() time.Time { return now }
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes:          []*db.Agente{{Nombre: "Stale", Activo: true, EstadoCuota: "activo"}},
		AgentesActivos:   []*db.Agente{{Nombre: "Stale", Activo: true, EstadoCuota: "activo"}},
		TareasEnProgreso: []tareaLite{{ID: 99, Estado: db.TareaEnProgreso, Agente: "Stale"}},
		Autonomia:        autonomiaResumen{ContinuidadPendiente: 1},
		Generado:         now.Format(time.RFC3339),
	}, now, time.Minute)

	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{}, errStatusFetchTimeout
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}
	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "CodexRO", Activo: true, EstadoCuota: "activo"}}, map[string]int{"en_progreso": 1}, []*db.Tarea{
			{ID: 9, Estado: db.TareaEnProgreso, Agente: stringPtr("CodexRO")},
		}, nil
	}

	status, ok := fetchStatusForOperationalFallback(50 * time.Millisecond)
	if !ok {
		t.Fatal("deberia encontrar fallback operativo")
	}
	if len(status.Agentes) == 0 || status.Agentes[0] == nil || status.Agentes[0].Nombre != "CodexRO" {
		t.Fatalf("no deberia reutilizar snapshot stale que requiere refresh: %+v", status.Agentes)
	}
}

func TestFetchStatusReadOnlyLiteDirectReconcilesVisibleSessions(t *testing.T) {
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevSessions := statusVisibleSessionsFetcher
	prevAgentsWithSessions := statusListAgentsWithSessionsFetcher
	resetAgentPanelSnapshotCache()
	t.Cleanup(func() {
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusVisibleSessionsFetcher = prevSessions
		statusListAgentsWithSessionsFetcher = prevAgentsWithSessions
		resetAgentPanelSnapshotCache()
	})

	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{{Nombre: "Codex2", Activo: false, Habilitado: true, EstadoCuota: "activo"}}, map[string]int{}, nil, nil
	}
	statusVisibleSessionsFetcher = func() ([]*db.Sesion, error) {
		return []*db.Sesion{{ID: 7, Agente: "Codex2", Activa: true, Estado: "activa"}}, nil
	}
	statusListAgentsWithSessionsFetcher = func(_ []*db.Sesion) ([]*db.Agente, error) {
		return []*db.Agente{{Nombre: "Codex2", Activo: true, Habilitado: true, EstadoSesion: "disponible", EstadoCuota: "activo"}}, nil
	}

	status, ok := fetchStatusReadOnlyLiteDirect(20 * time.Millisecond)
	if !ok {
		t.Fatalf("deberia construir snapshot read-only")
	}
	if len(status.AgentesActivos) != 1 || status.AgentesActivos[0].Nombre != "Codex2" || !status.AgentesActivos[0].Activo {
		t.Fatalf("agentes activos reconciliados inesperados: %+v", status.AgentesActivos)
	}
}

func TestFetchStatusReadOnlyLiteDirectPrefierePanelSnapshotFrescoParaVisibilidad(t *testing.T) {
	prevReadOnly := apiStatusReadOnlyLiteFetcher
	prevSessions := statusVisibleSessionsFetcher
	prevAgentsWithSessions := statusListAgentsWithSessionsFetcher
	resetAgentPanelSnapshotCache()
	t.Cleanup(func() {
		apiStatusReadOnlyLiteFetcher = prevReadOnly
		statusVisibleSessionsFetcher = prevSessions
		statusListAgentsWithSessionsFetcher = prevAgentsWithSessions
		resetAgentPanelSnapshotCache()
	})

	apiStatusReadOnlyLiteFetcher = func() ([]*db.Agente, map[string]int, []*db.Tarea, error) {
		return []*db.Agente{
			{Nombre: "Codex1", Activo: true, Habilitado: true, EstadoCuota: "activo"},
			{Nombre: "Codex2", Activo: true, Habilitado: true, EstadoCuota: "activo"},
		}, map[string]int{}, nil, nil
	}
	statusVisibleSessionsFetcher = func() ([]*db.Sesion, error) { return nil, nil }
	statusListAgentsWithSessionsFetcher = func(_ []*db.Sesion) ([]*db.Agente, error) { return nil, nil }
	storeAgentPanelSnapshot([]agentesapp.Row{
		{Agente: &db.Agente{Nombre: "Codex2", Activo: true, Habilitado: true, EstadoCuota: "activo"}, EstadoOperativo: "trabajando"},
	}, time.Now().UTC())

	status, ok := fetchStatusReadOnlyLiteDirect(20 * time.Millisecond)
	if !ok {
		t.Fatalf("deberia construir snapshot read-only")
	}
	if len(status.AgentesActivos) != 1 || status.AgentesActivos[0].Nombre != "Codex2" {
		t.Fatalf("agentes activos deberian alinearse con panel fresco: %+v", status.AgentesActivos)
	}
	if len(status.AgentesTrabajando) != 1 || status.AgentesTrabajando[0].Nombre != "Codex2" {
		t.Fatalf("agentes trabajando deberian alinearse con panel fresco: %+v", status.AgentesTrabajando)
	}
}

func TestAPIAuditFiltraEnSQLPorAccionYEntidadID(t *testing.T) {
	prepararDBTemporalCmd(t)

	db.Audit("Codex1", "pipeline_local_batch", "runtime", 7, "entrada valida")
	db.Audit("Codex1", "otro_batch", "runtime", 7, "accion distinta")
	db.Audit("Codex2", "pipeline_local_batch", "runtime", 8, "entidad_id distinta")

	req := httptest.NewRequest(http.MethodGet, "/api/audit?accion=pipeline_local_batch&entidad=runtime&entidad_id=7&limit=10", nil)
	rec := httptest.NewRecorder()
	apiHandlerAudit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiAuditResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode audit payload: %v", err)
	}
	if len(payload.Audit) != 1 {
		t.Fatalf("audit filtrada inesperada: %+v", payload.Audit)
	}
	if payload.Audit[0].Accion != "pipeline_local_batch" || payload.Audit[0].EntidadID != 7 {
		t.Fatalf("entrada audit inesperada: %+v", payload.Audit[0])
	}
}

func TestAPIAuditReturnsEmptyWhenQueryTimesOut(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevTimeout := apiAuditTimeout
	prevListAudit := apiListAuditFn
	t.Cleanup(func() {
		apiAuditTimeout = prevTimeout
		apiListAuditFn = prevListAudit
	})

	apiAuditTimeout = 20 * time.Millisecond
	apiListAuditFn = func(db.FiltroAuditoria) ([]*db.LogAuditoria, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/audit?accion=control_plane_batch_slow&limit=10", nil)
	rec := httptest.NewRecorder()
	apiHandlerAudit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiAuditResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode audit payload: %v", err)
	}
	if len(payload.Audit) != 0 {
		t.Fatalf("audit degradada inesperada: %+v", payload.Audit)
	}
}

func TestAPIAuditFiltraDesde(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevListAudit := apiListAuditFn
	t.Cleanup(func() {
		apiListAuditFn = prevListAudit
	})

	var got db.FiltroAuditoria
	apiListAuditFn = func(f db.FiltroAuditoria) ([]*db.LogAuditoria, error) {
		got = f
		return []*db.LogAuditoria{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/audit?agente=Codex1&desde=2026-04-23T10:00:00Z&limit=10", nil)
	rec := httptest.NewRecorder()
	apiHandlerAudit(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.Desde == nil || got.Desde.UTC().Format(time.RFC3339) != "2026-04-23T10:00:00Z" {
		t.Fatalf("filtro desde inesperado: %+v", got)
	}
}

func TestAPIRuntimeTranscriptFiltraDesde(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevFn := apiListRuntimeTranscriptFn
	t.Cleanup(func() {
		apiListRuntimeTranscriptFn = prevFn
	})

	var got db.FiltroRuntimeTranscript
	apiListRuntimeTranscriptFn = func(filter db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
		got = filter
		return []*db.RuntimeTranscriptEntry{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runtime-transcript?agente=Codex1&desde=2026-04-23T10:00:00Z&limit=10", nil)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeTranscript(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if got.Desde == nil || got.Desde.UTC().Format(time.RFC3339) != "2026-04-23T10:00:00Z" {
		t.Fatalf("filtro desde inesperado: %+v", got)
	}
}

func TestAPIStatusExponeResumenOperativoCompat(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}
	for _, key := range []string{"agentes", "conteo_tareas", "resumenTareas", "generado", "tareasPorEstado", "agentesActivos", "agentesSaturados", "propuestasAbiertas", "tareasActivas"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("/api/status sin clave %q: %+v", key, payload)
		}
	}
}

func TestAPIAgentesYStatusExponenCuentaYCuotaVisible(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexCuenta", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexCuenta",
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-cuenta-001",
		Host:              "worker-api",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("UpsertRuntimeHandleDesdeSesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE sesion_id=?`,
		`{"auth":{"email":"codexcuenta@example.com"},"username":"codexcuenta_user"}`, sesion.ID); err != nil {
		t.Fatalf("update runtime handle metadata: %v", err)
	}
	remaining := int64(900)
	resetSesion := time.Date(2026, 3, 31, 21, 0, 0, 0, time.UTC)
	inicioSesion := resetSesion.Add(-5 * time.Hour)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		WindowStartedAt:  &inicioSesion,
		ResetAt:          &resetSesion,
		RemainingSeconds: &remaining,
		BudgetSource:     "runtime",
		RawSnapshotJSON:  `{"account":{"email":"codexcuenta@example.com","username":"codexcuenta_user"}}`,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgentes := httptest.NewRecorder()
	reqAgentes := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(recAgentes, reqAgentes)
	if recAgentes.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", recAgentes.Code, recAgentes.Body.String())
	}
	var agentesResp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(recAgentes.Body.Bytes(), &agentesResp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	var agenteAPI *db.Agente
	for _, item := range agentesResp.Agentes {
		if item != nil && item.Nombre == "CodexCuenta" {
			agenteAPI = item
			break
		}
	}
	if agenteAPI == nil {
		t.Fatalf("CodexCuenta no expuesto via /api/agentes: %+v", agentesResp.Agentes)
	}
	if agenteAPI.CuentaEmail != "codexcuenta@example.com" || agenteAPI.CuentaUsuario != "codexcuenta_user" {
		t.Fatalf("cuenta inesperada via /api/agentes: %+v", agenteAPI)
	}
	if agenteAPI.PresupuestoSesionPct == nil || agenteAPI.PresupuestoDiarioPct == nil || agenteAPI.PresupuestoSemanalPct == nil {
		t.Fatalf("desglose de cuota incompleto via /api/agentes: %+v", agenteAPI)
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	var agenteStatus *db.Agente
	for _, item := range statusResp.AgentesActivos {
		if item != nil && item.Nombre == "CodexCuenta" {
			agenteStatus = item
			break
		}
	}
	if agenteStatus == nil {
		t.Fatalf("CodexCuenta no expuesto via /api/status: %+v", statusResp.AgentesActivos)
	}
	if agenteStatus.CuentaEmail != "codexcuenta@example.com" || agenteStatus.CuentaUsuario != "codexcuenta_user" {
		t.Fatalf("cuenta inesperada via /api/status: %+v", agenteStatus)
	}
	if agenteStatus.PresupuestoVentana == "" || agenteStatus.PresupuestoResetAt == nil {
		t.Fatalf("ventana efectiva incompleta via /api/status: %+v", agenteStatus)
	}
}

func TestAPIAgentesPresupuestoYCuentas(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexCuenta", "programador"); err != nil {
		t.Fatalf("RegistrarAgente CodexCuenta: %v", err)
	}
	if err := db.RegistrarAgente("CodexSinCuenta", "programador"); err != nil {
		t.Fatalf("RegistrarAgente CodexSinCuenta: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexCuenta",
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-cuentas-001",
		Host:              "worker-api",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("UpsertRuntimeHandleDesdeSesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE sesion_id=?`,
		`{"auth":{"email":"codexcuenta@example.com"},"username":"codexcuenta_user"}`, sesion.ID); err != nil {
		t.Fatalf("update runtime handle metadata: %v", err)
	}
	remaining := int64(900)
	resetSesion := time.Date(2026, 3, 31, 21, 0, 0, 0, time.UTC)
	inicioSesion := resetSesion.Add(-5 * time.Hour)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		WindowStartedAt:  &inicioSesion,
		ResetAt:          &resetSesion,
		RemainingSeconds: &remaining,
		BudgetSource:     "runtime",
		RawSnapshotJSON:  `{"account":{"email":"codexcuenta@example.com","username":"codexcuenta_user"}}`,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recPresupuesto := httptest.NewRecorder()
	reqPresupuesto := httptest.NewRequest(http.MethodGet, "/api/agentes/presupuesto?activos=true", nil)
	mux.ServeHTTP(recPresupuesto, reqPresupuesto)
	if recPresupuesto.Code != http.StatusOK {
		t.Fatalf("status presupuesto inesperado: %d body=%s", recPresupuesto.Code, recPresupuesto.Body.String())
	}
	var presupuestoResp apiAgentesPresupuestoResponse
	if err := json.Unmarshal(recPresupuesto.Body.Bytes(), &presupuestoResp); err != nil {
		t.Fatalf("decode presupuesto: %v", err)
	}
	if !presupuestoResp.Activos || len(presupuestoResp.Agentes) != 1 || presupuestoResp.Agentes[0].Nombre != "CodexCuenta" {
		t.Fatalf("respuesta presupuesto inesperada: %+v", presupuestoResp)
	}
	if presupuestoResp.Agentes[0].PresupuestoSesionPct == nil || presupuestoResp.Agentes[0].PresupuestoDiarioPct == nil || presupuestoResp.Agentes[0].PresupuestoSemanalPct == nil {
		t.Fatalf("presupuesto sin desglose completo: %+v", presupuestoResp.Agentes[0])
	}

	recPresupuestoAgente := httptest.NewRecorder()
	reqPresupuestoAgente := httptest.NewRequest(http.MethodGet, "/api/agentes/presupuesto?agente=CodexCuenta", nil)
	mux.ServeHTTP(recPresupuestoAgente, reqPresupuestoAgente)
	if recPresupuestoAgente.Code != http.StatusOK {
		t.Fatalf("status presupuesto por agente inesperado: %d body=%s", recPresupuestoAgente.Code, recPresupuestoAgente.Body.String())
	}
	var presupuestoAgenteResp apiAgentesPresupuestoResponse
	if err := json.Unmarshal(recPresupuestoAgente.Body.Bytes(), &presupuestoAgenteResp); err != nil {
		t.Fatalf("decode presupuesto por agente: %v", err)
	}
	if len(presupuestoAgenteResp.Agentes) != 1 || presupuestoAgenteResp.Agentes[0].Nombre != "CodexCuenta" {
		t.Fatalf("respuesta presupuesto por agente inesperada: %+v", presupuestoAgenteResp)
	}

	recCuentas := httptest.NewRecorder()
	reqCuentas := httptest.NewRequest(http.MethodGet, "/api/agentes/cuentas", nil)
	mux.ServeHTTP(recCuentas, reqCuentas)
	if recCuentas.Code != http.StatusOK {
		t.Fatalf("status cuentas inesperado: %d body=%s", recCuentas.Code, recCuentas.Body.String())
	}
	var cuentasResp apiAgentesCuentasResponse
	if err := json.Unmarshal(recCuentas.Body.Bytes(), &cuentasResp); err != nil {
		t.Fatalf("decode cuentas: %v", err)
	}
	if len(cuentasResp.Agentes) < 2 {
		t.Fatalf("respuesta cuentas incompleta: %+v", cuentasResp)
	}
	var cuentaItem, sinCuentaItem *apiAgenteCuentaItem
	for i := range cuentasResp.Agentes {
		item := &cuentasResp.Agentes[i]
		switch item.Nombre {
		case "CodexCuenta":
			cuentaItem = item
		case "CodexSinCuenta":
			sinCuentaItem = item
		}
	}
	if cuentaItem == nil || cuentaItem.CuentaEmail != "codexcuenta@example.com" || cuentaItem.CuentaUsuario != "codexcuenta_user" {
		t.Fatalf("cuenta expuesta inesperada: %+v", cuentaItem)
	}
	if sinCuentaItem == nil {
		t.Fatalf("faltaba agente sin cuenta: %+v", cuentasResp.Agentes)
	}

	recRanking := httptest.NewRecorder()
	reqRanking := httptest.NewRequest(http.MethodGet, "/api/agentes/ranking-cuentas?activos=true", nil)
	mux.ServeHTTP(recRanking, reqRanking)
	if recRanking.Code != http.StatusOK {
		t.Fatalf("status ranking cuentas inesperado: %d body=%s", recRanking.Code, recRanking.Body.String())
	}
	var rankingResp apiAgentesRankingCuentasResponse
	if err := json.Unmarshal(recRanking.Body.Bytes(), &rankingResp); err != nil {
		t.Fatalf("decode ranking cuentas: %v", err)
	}
	if len(rankingResp.Cuentas) != 1 {
		t.Fatalf("ranking de cuentas inesperado: %+v", rankingResp)
	}
	if rankingResp.Cuentas[0].CuentaClave != "codexcuenta@example.com" {
		t.Fatalf("cuenta clave inesperada: %+v", rankingResp.Cuentas[0])
	}
	if rankingResp.Cuentas[0].Criterio != "remaining_seconds" && rankingResp.Cuentas[0].Criterio != "cuota_pct" {
		t.Fatalf("criterio inesperado: %+v", rankingResp.Cuentas[0])
	}
}

func TestAPIStatusOmiteTareasActivasSinProyecto(t *testing.T) {
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
	conProyectoID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Con proyecto",
		Descripcion: "Debe salir en status",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea con proyecto: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Sin proyecto",
		Descripcion: "No debe salir en status",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea sin proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso' WHERE id IN (?, (SELECT MAX(id) FROM tareas))`, conProyectoID, conProyectoID); err != nil {
		t.Fatalf("marcar tareas en_progreso: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}
	if len(payload.TareasActivas) != 1 {
		t.Fatalf("tareasActivas inesperadas: %+v", payload.TareasActivas)
	}
	if payload.TareasActivas[0].ID != conProyectoID {
		t.Fatalf("tarea activa visible inesperada: %+v", payload.TareasActivas[0])
	}
	if got := payload.TareasPorEstado[string(db.TareaEnProgreso)]; got != 1 {
		t.Fatalf("conteo visible de en_progreso inesperado: %+v", payload.TareasPorEstado)
	}
}

func TestAPIAgentesPresupuestoRefrescarOperaPorLaViaCanonica(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/presupuesto/refrescar", bytes.NewReader([]byte(`{"agente":"Codex6"}`)))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status refrescar presupuesto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode refresh presupuesto: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload refresh sin ok: %#v", payload)
	}
	if payload["agente"] != "Codex6" {
		t.Fatalf("agente refresh inesperado: %#v", payload)
	}
}

func TestAPIAgentesYStatusAlineanActivoConSesionReal(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"CodexVisible1", "CodexVisible2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible' WHERE nombre='CodexVisible2'`); err != nil {
		t.Fatalf("marcar activo visible2: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexVisible1",
		CWD:         "/tmp/orquesta-codex-visible1-api",
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto visible1: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexVisible1'`); err != nil {
		t.Fatalf("marcar estado visible1: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgentes := httptest.NewRecorder()
	reqAgentes := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(recAgentes, reqAgentes)
	if recAgentes.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", recAgentes.Code, recAgentes.Body.String())
	}
	var agentesResp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(recAgentes.Body.Bytes(), &agentesResp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	estadoAgentes := map[string]*db.Agente{}
	for _, agente := range agentesResp.Agentes {
		estadoAgentes[agente.Nombre] = agente
	}
	if !estadoAgentes["CodexVisible1"].Activo || estadoAgentes["CodexVisible1"].EstadoSesion != "pensando" {
		t.Fatalf("agente visible1 inesperado via /api/agentes: %+v", estadoAgentes["CodexVisible1"])
	}
	if estadoAgentes["CodexVisible2"].Activo || estadoAgentes["CodexVisible2"].EstadoSesion != "" {
		t.Fatalf("agente visible2 inesperado via /api/agentes: %+v", estadoAgentes["CodexVisible2"])
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	estadoStatus := map[string]*db.Agente{}
	for _, agente := range statusResp.Agentes {
		estadoStatus[agente.Nombre] = agente
	}
	estadoActivosStatus := map[string]*db.Agente{}
	for _, agente := range statusResp.AgentesActivos {
		estadoActivosStatus[agente.Nombre] = agente
	}
	if visible1 := estadoActivosStatus["CodexVisible1"]; visible1 != nil && !visible1.Activo {
		t.Fatalf("CodexVisible1 no deberia perder flag activo via /api/status: %+v", visible1)
	}
	if visible2 := estadoActivosStatus["CodexVisible2"]; visible2 != nil && visible2.Activo {
		t.Fatalf("CodexVisible2 no deberia seguir activo via /api/status: %+v", visible2)
	}
	if visible1 := estadoStatus["CodexVisible1"]; visible1 != nil && visible1.EstadoSesion != "pensando" {
		t.Fatalf("estado de sesion inesperado via /api/status: %+v", visible1)
	}
	if visible2 := estadoStatus["CodexVisible2"]; visible2 != nil && (visible2.Activo || visible2.EstadoSesion != "") {
		t.Fatalf("agente visible2 inesperado via /api/status: %+v", visible2)
	}
}

func TestAPIAgentesYStatusOcultanSesionZombi(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"CodexZombie", "CodexHandle"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	old := "2026-03-22 16:39:43"
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexZombie'`); err != nil {
		t.Fatalf("estado zombie: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)`,
		"CodexZombie", 1, "activa", "codex", "localhost", old,
	); err != nil {
		t.Fatalf("insert sesion zombie: %v", err)
	}
	var handleSesionID int64
	if err := db.DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)
		RETURNING id`,
		"CodexHandle", 1, "activa", "codex", "localhost", old,
	).Scan(&handleSesionID); err != nil {
		t.Fatalf("insert sesion CodexHandle: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at
		) VALUES (?,?,?,?,?,'activo','{}','{}',CURRENT_TIMESTAMP)`,
		"CodexHandle", handleSesionID, "cli", "session", "sess-codex-handle",
	); err != nil {
		t.Fatalf("insert handle CodexHandle: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgentes := httptest.NewRecorder()
	reqAgentes := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(recAgentes, reqAgentes)
	if recAgentes.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", recAgentes.Code, recAgentes.Body.String())
	}
	var agentesResp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(recAgentes.Body.Bytes(), &agentesResp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	estadoAgentes := map[string]*db.Agente{}
	for _, agente := range agentesResp.Agentes {
		estadoAgentes[agente.Nombre] = agente
	}
	if estadoAgentes["CodexZombie"].Activo {
		t.Fatalf("CodexZombie no deberia salir activo via /api/agentes: %+v", estadoAgentes["CodexZombie"])
	}
	if !estadoAgentes["CodexHandle"].Activo {
		t.Fatalf("CodexHandle deberia seguir activo via /api/agentes: %+v", estadoAgentes["CodexHandle"])
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	estadoStatus := map[string]*db.Agente{}
	for _, agente := range statusResp.Agentes {
		estadoStatus[agente.Nombre] = agente
	}
	if zombie := estadoStatus["CodexZombie"]; zombie != nil && zombie.Activo {
		t.Fatalf("CodexZombie no deberia salir activo via /api/status: %+v", zombie)
	}
	if handle := estadoStatus["CodexHandle"]; handle == nil || !handle.Activo {
		t.Fatalf("CodexHandle deberia seguir activo via /api/status: %+v", handle)
	}
}

func TestAPIStatusNoCuentaSesionConHandleFallidoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexFallo", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	var sesionID int64
	if err := db.DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
		RETURNING id`,
		"CodexFallo", 1, "activa", "codex-cli", "localhost",
	).Scan(&sesionID); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,'fallido','{}','{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexFallo", sesionID, "cli", "process", "9999",
	); err != nil {
		t.Fatalf("insert handle fallido: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	for _, agente := range statusResp.Agentes {
		if agente != nil && agente.Nombre == "CodexFallo" && agente.Activo {
			t.Fatalf("CodexFallo no deberia salir activo via /api/status: %+v", agente)
		}
	}
}

func TestAPIStatusNoCuentaAgenteConRuntimePrincipalStaleAunqueMantengaHandleActivo(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexRuntimeStale", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	var sesionID int64
	if err := db.DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
		RETURNING id`,
		"CodexRuntimeStale", 1, "activa", "codex-cli", "localhost",
	).Scan(&sesionID); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:            "CodexRuntimeStale",
		SesionID:          &sesionID,
		Provider:          "openai",
		Connector:         "codex-cli",
		LogicalState:      "esperando_io",
		ProcessState:      "running",
		ExternalSessionID: "sess-runtime-stale",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance: %v", err)
	}
	old := time.Now().UTC().Add(-15 * time.Minute)
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET last_event_at=?, last_heartbeat_at=?, updated_at=?
		WHERE id = ?`,
		old, old, old, runtimeID,
	); err != nil {
		t.Fatalf("stale runtime: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, runtime_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,'7777','activo','{}','{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexRuntimeStale", sesionID, runtimeID, "cli", "process",
	); err != nil {
		t.Fatalf("insert handle activo: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	for _, agente := range statusResp.Agentes {
		if agente != nil && agente.Nombre == "CodexRuntimeStale" && agente.Activo {
			t.Fatalf("CodexRuntimeStale no deberia salir activo via /api/status: %+v", agente)
		}
	}
}

func TestAPIRuntimeA2UIExponeMensajesDeFormaServerFirst(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a2ui-api",
		ResumenContinuidad: "runtime con a2ui",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("get runtime: runtime=%+v err=%v", runtimeInst, err)
	}
	reqA2UI := &a2ui.RenderRequest{
		Type:      a2ui.MessageRender,
		Component: a2ui.ComponentDataTable,
		Props: json.RawMessage(`{
			"title":"Riesgos runtime",
			"columns":["ID","Detalle"],
			"data":[[1,"Handoff pendiente"],[2,"Checkpoint atrasado"]]
		}`),
	}
	msg, err := a2ui.BuildMailboxMessage("Codex0", "Codex1", reqA2UI)
	if err != nil {
		t.Fatalf("build mailbox a2ui: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  msg.FromAgente,
		ToAgente:    msg.ToAgente,
		ProyectoID:  &proyectoID,
		Kind:        msg.Kind,
		PayloadJSON: msg.Payload,
	}); err != nil {
		t.Fatalf("crear mailbox a2ui: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/runtimes/"+itoa(runtimeInst.ID)+"/a2ui?limit=5", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiRuntimeA2UIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode a2ui api: %v", err)
	}
	if payload.Runtime == nil || payload.Runtime.ID != runtimeInst.ID {
		t.Fatalf("runtime inesperado en /api/runtimes/a2ui: %+v", payload.Runtime)
	}
	if len(payload.A2UI) != 1 {
		t.Fatalf("mensajes a2ui inesperados: %+v", payload.A2UI)
	}
	if payload.A2UI[0].Component != string(a2ui.ComponentDataTable) {
		t.Fatalf("componente a2ui inesperado: %+v", payload.A2UI[0])
	}
	if payload.A2UI[0].DataTable == nil || payload.A2UI[0].DataTable.Title != "Riesgos runtime" {
		t.Fatalf("datatable a2ui inesperada: %+v", payload.A2UI[0])
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content-type inesperado: %s", got)
	}
}

func TestAPIAsignacionActivarYSesionInicio(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.GetProyecto("autofirmav2"); err != nil {
		t.Fatalf("get proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	asignacionBody, _ := json.Marshal(apiAsignacionActivarRequest{
		Agente:   "Codex1",
		Proyecto: "autofirmav2",
		Nota:     "slot 1",
	})
	recAsignacion := httptest.NewRecorder()
	reqAsignacion := httptest.NewRequest(http.MethodPost, "/api/asignaciones/activar", bytes.NewReader(asignacionBody))
	mux.ServeHTTP(recAsignacion, reqAsignacion)
	if recAsignacion.Code != http.StatusOK {
		t.Fatalf("status asignacion inesperado: %d body=%s", recAsignacion.Code, recAsignacion.Body.String())
	}

	sesionBody, _ := json.Marshal(apiSesionInicioRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-001",
		Resumen:           "continuar desde API",
	})
	recSesion := httptest.NewRecorder()
	reqSesion := httptest.NewRequest(http.MethodPost, "/api/sesiones/inicio", bytes.NewReader(sesionBody))
	mux.ServeHTTP(recSesion, reqSesion)
	if recSesion.Code != http.StatusCreated {
		t.Fatalf("status sesion inesperado: %d body=%s", recSesion.Code, recSesion.Body.String())
	}
	var resp apiSesionInicioResponse
	if err := json.Unmarshal(recSesion.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode sesion inicio: %v", err)
	}
	if resp.Sesion == nil || resp.Sesion.Agente != "Codex1" {
		t.Fatalf("respuesta de sesion inesperada: %+v", resp.Sesion)
	}
	if resp.Rol == "" {
		t.Fatalf("se esperaba rol en la respuesta de sesion inicio")
	}
	if len(resp.Reglas) == 0 {
		t.Fatalf("se esperaban reglas en la respuesta de sesion inicio")
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "sess-001" {
		t.Fatalf("external session id inesperado: %s", sesion.ExternalSessionID)
	}
}

func TestAPISesionInicioArranqueLimpioOmiteContinuidadPrevia(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    "codex",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		ConectorID:         &conectorID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID:  "sess-previa",
		ResumePayloadJSON:  `{"continuidad":true}`,
		ResumenContinuidad: "seguir desde antes",
	}); err != nil {
		t.Fatalf("iniciar sesion previa: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	sesionBody, _ := json.Marshal(apiSesionInicioRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-nueva",
		ResumePayload:     `{"nueva":true}`,
		Resumen:           "no deberia persistir",
		ArranqueLimpio:    true,
	})
	recSesion := httptest.NewRecorder()
	reqSesion := httptest.NewRequest(http.MethodPost, "/api/sesiones/inicio", bytes.NewReader(sesionBody))
	mux.ServeHTTP(recSesion, reqSesion)
	if recSesion.Code != http.StatusCreated {
		t.Fatalf("status sesion inesperado: %d body=%s", recSesion.Code, recSesion.Body.String())
	}
	var resp apiSesionInicioResponse
	if err := json.Unmarshal(recSesion.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode sesion inicio: %v", err)
	}
	if resp.SesionPrevia != nil {
		t.Fatalf("no deberia exponer sesion previa: %+v", resp.SesionPrevia)
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "" || sesion.ResumePayloadJSON != "" || sesion.ResumenContinuidad != "" {
		t.Fatalf("continuidad no limpiada en arranque limpio: %+v", sesion)
	}
}

func TestAPISesionGuardarPermiteLimpiarContinuidad(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID:  "sess-guardar",
		ResumePayloadJSON:  `{"continuidad":true}`,
		ResumenContinuidad: "seguir guardado",
		Branch:             "feature/x",
	}); err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiSesionGuardarRequest{
		Agente:             "Codex1",
		Proyecto:           "autofirmav2",
		LimpiarContinuidad: true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sesiones/guardar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status guardar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "" || sesion.ResumePayloadJSON != "" || sesion.ResumenContinuidad != "" {
		t.Fatalf("continuidad no limpiada: %+v", sesion)
	}
	if sesion.Branch != "feature/x" {
		t.Fatalf("branch no deberia cambiar: %+v", sesion)
	}
}

func TestAPIAgenteAdoptarContextoPersisteContinuidadYGobernanza(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    "codex",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Completar proyecto",
		Descripcion: "Trabajo pendiente",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiAgenteAdoptarContextoRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		Branch:            "feature/adopcion",
		ExternalSessionID: "sess-codex-actual",
		Resumen:           "seguir este mismo proyecto hasta terminarlo",
		Nota:              "adopcion de la sesion actual",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/adoptar-contexto", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status adoptar contexto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteAdoptarContextoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode adoptar contexto: %v", err)
	}
	if resp.Sesion == nil || resp.Sesion.Agente != "Codex1" {
		t.Fatalf("sesion adoptada inesperada: %+v", resp.Sesion)
	}
	if resp.Sesion.Activa || resp.Sesion.Estado != "pausada" {
		t.Fatalf("la sesion adoptada deberia quedar aparcada hasta el takeover: %+v", resp.Sesion)
	}
	if resp.Checkpoint == nil || resp.Checkpoint.CheckpointKind != "contexto_adoptado" {
		t.Fatalf("checkpoint adoptado inesperado: %+v", resp.Checkpoint)
	}
	if resp.RuntimeOrderID == nil || *resp.RuntimeOrderID <= 0 {
		t.Fatalf("se esperaba runtime_order start para la adopcion: %+v", resp)
	}
	if resp.Gobernanza == nil || strings.TrimSpace(resp.Gobernanza.ResolucionActual) == "" {
		t.Fatalf("catalogo de gobernanza ausente: %+v", resp.Gobernanza)
	}

	sesion, err := db.ObtenerUltimaSesion("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get ultima sesion: %v", err)
	}
	if !strings.Contains(sesion.ResumePayloadJSON, `"governance_catalog"`) || !strings.Contains(sesion.ResumePayloadJSON, `"project_context"`) {
		t.Fatalf("resume payload adoptado incompleto: %s", sesion.ResumePayloadJSON)
	}
	if !strings.Contains(sesion.ResumenContinuidad, "Contexto adoptado por Orquesta") {
		t.Fatalf("resumen continuidad no adoptado: %s", sesion.ResumenContinuidad)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" || !strings.Contains(orders[0].PayloadJSON, `"motivo":"adopt_context"`) {
		t.Fatalf("runtime orders inesperadas tras adopcion: %+v", orders)
	}
}

func TestAPIAgenteAdoptarContextoNoDuplicaRuntimeExternoActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conectorID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		Herramienta:        "codex-remote",
		ExternalSessionID:  "sess-live",
		ResumenContinuidad: "sesion externa viva",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiAgenteAdoptarContextoRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-remote",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-live",
		Resumen:           "seguir este mismo proyecto",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/adoptar-contexto", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status adoptar contexto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteAdoptarContextoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode adoptar contexto: %v", err)
	}
	if resp.RuntimeOrderID != nil {
		t.Fatalf("no deberia encolar start sobre runtime externo vivo: %+v", resp)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia haber runtime orders nuevas: %+v", orders)
	}
}

func TestAPIPropuestaReabrirYRepararVotos(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &db.Propuesta{
		Titulo:       "Propuesta API",
		Descripcion:  "Reabrir y reparar desde API",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if _, err := db.DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor2"); err != nil {
		t.Fatalf("delete voto faltante: %v", err)
	}
	if err := db.CerrarPropuesta(propuesta.Codigo, string(db.PropuestaBacklog), "alberto"); err != nil {
		t.Fatalf("cerrar propuesta: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	reabrirBody, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion: "reabrir",
		Agente: "alberto",
	})
	recReabrir := httptest.NewRecorder()
	reqReabrir := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(reabrirBody))
	mux.ServeHTTP(recReabrir, reqReabrir)
	if recReabrir.Code != http.StatusOK {
		t.Fatalf("status reabrir inesperado: %d body=%s", recReabrir.Code, recReabrir.Body.String())
	}

	reabierta, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta reabierta: %v", err)
	}
	if reabierta.Estado != db.PropuestaAbierta {
		t.Fatalf("estado inesperado tras reabrir: %s", reabierta.Estado)
	}

	if _, err := db.DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1"); err != nil {
		t.Fatalf("delete segundo voto faltante: %v", err)
	}

	repararBody, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion: "reparar_votos",
		Agente: "alberto",
	})
	recReparar := httptest.NewRecorder()
	reqReparar := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(repararBody))
	mux.ServeHTTP(recReparar, reqReparar)
	if recReparar.Code != http.StatusOK {
		t.Fatalf("status reparar inesperado: %d body=%s", recReparar.Code, recReparar.Body.String())
	}

	var total int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1").Scan(&total); err != nil {
		t.Fatalf("contar voto reparado: %v", err)
	}
	if total != 1 {
		t.Fatalf("conteo inesperado tras reparar via API: %d", total)
	}
}

func TestAPIGobernanzaCatalogoEOverrides(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "ctx-proyecto",
		Nombre:  "ctx-proyecto",
		RutaAbs: filepath.Join(t.TempDir(), "ctx-proyecto"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertRegla(&db.Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "server-first",
		Descripcion: "usar API",
		Activa:      true,
	}); err != nil {
		t.Fatalf("upsert regla: %v", err)
	}
	skillID, err := db.UpsertSkill(&db.Skill{
		TipoAgente:  "programador",
		Nombre:      "docker-build",
		Descripcion: "build",
		CuandoUsar:  "siempre",
		Prioridad:   10,
		Activa:      false,
	})
	if err != nil {
		t.Fatalf("upsert skill: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	overrideBody, _ := json.Marshal(apiGovernanceOverrideSaveRequest{
		Actor:      "alberto",
		TipoAgente: "programador",
		ScopeTipo:  db.GovernanceScopeProyecto,
		ScopeRef:   "ctx-proyecto",
		Entidad:    db.GovernanceEntitySkill,
		EntidadID:  skillID,
		Accion:     db.GovernanceActionEnable,
	})
	recOverride := httptest.NewRecorder()
	reqOverride := httptest.NewRequest(http.MethodPost, "/api/gobernanza/overrides", bytes.NewReader(overrideBody))
	mux.ServeHTTP(recOverride, reqOverride)
	if recOverride.Code != http.StatusCreated {
		t.Fatalf("status override inesperado: %d body=%s", recOverride.Code, recOverride.Body.String())
	}

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/gobernanza/overrides?scope_tipo=proyecto&scope_ref=ctx-proyecto&tipo_agente=programador", nil)
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("status listar overrides inesperado: %d body=%s", recList.Code, recList.Body.String())
	}
	var overridesResp apiGovernanceOverridesResponse
	if err := json.Unmarshal(recList.Body.Bytes(), &overridesResp); err != nil {
		t.Fatalf("decode overrides: %v", err)
	}
	if len(overridesResp.Overrides) != 1 || overridesResp.Overrides[0].EntidadID != skillID {
		t.Fatalf("overrides inesperados: %+v", overridesResp.Overrides)
	}

	recCatalogo := httptest.NewRecorder()
	reqCatalogo := httptest.NewRequest(http.MethodGet, "/api/gobernanza/catalogo?agente=Codex1&proyecto=ctx-proyecto", nil)
	mux.ServeHTTP(recCatalogo, reqCatalogo)
	if recCatalogo.Code != http.StatusOK {
		t.Fatalf("status catalogo inesperado: %d body=%s", recCatalogo.Code, recCatalogo.Body.String())
	}
	var catalogoResp apiGovernanceCatalogResponse
	if err := json.Unmarshal(recCatalogo.Body.Bytes(), &catalogoResp); err != nil {
		t.Fatalf("decode catalogo: %v", err)
	}
	if catalogoResp.Catalogo == nil {
		t.Fatalf("catalogo nil")
	}
	if catalogoResp.Catalogo.ProyectoID == nil || *catalogoResp.Catalogo.ProyectoID != proyectoID {
		t.Fatalf("proyecto_id inesperado en catalogo: %+v", catalogoResp.Catalogo)
	}
	if catalogoResp.Catalogo.ResolucionActual != "rol+proyecto" {
		t.Fatalf("resolucion inesperada: %q", catalogoResp.Catalogo.ResolucionActual)
	}
	foundSkill := false
	for _, skill := range catalogoResp.Catalogo.Skills {
		if skill != nil && skill.ID == skillID {
			foundSkill = true
			break
		}
	}
	if !foundSkill {
		t.Fatalf("el catalogo efectivo deberia incluir la skill habilitada por override: %+v", catalogoResp.Catalogo.Skills)
	}
}

func TestAPIStatusIncluyeVotosDePropuestasAbiertas(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &db.Propuesta{
		Titulo:       "Status con votos",
		Descripcion:  "Verificar carga de votos en /api/status",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	p, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if _, err := db.Votar(p.ID, "revisor1", db.VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("votar revisor1: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if len(resp.PropuestasAbiertas) == 0 {
		t.Fatalf("se esperaban propuestas abiertas en status")
	}
	var encontrada *db.Propuesta
	for _, item := range resp.PropuestasAbiertas {
		if item != nil && item.Codigo == propuesta.Codigo {
			encontrada = item
			break
		}
	}
	if encontrada == nil {
		t.Fatalf("no se encontró la propuesta %s en status", propuesta.Codigo)
	}
	if len(encontrada.Votos) == 0 {
		t.Fatalf("status no incluyó votos para la propuesta abierta")
	}
}

func TestAPIPropuestaActualizarAnexaDescripcion(t *testing.T) {
	prepararDBTemporalCmd(t)

	propuesta := &db.Propuesta{
		Codigo:       "OP-910",
		Titulo:       "Propuesta API update",
		Descripcion:  "Base",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	appendText := " + docs"
	body, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion:     "actualizar",
		Agente:     "alberto",
		AnexarDesc: &appendText,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status actualizar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta actualizada: %v", err)
	}
	if updated.Descripcion != "Base + docs" {
		t.Fatalf("descripcion inesperada: %q", updated.Descripcion)
	}
}

func TestAPILenguajePoliticaMatrizYResolver(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.SetLanguagePolicy(&db.LanguagePolicy{
		DefaultLanguage:          "es",
		DocumentationMultilang:   true,
		AppsMultilang:            true,
		DocumentationDefaultLang: "es",
		AppsDefaultLang:          "en",
		AllowedLanguages:         []string{"es", "en", "fr"},
		Notes:                    "politica de prueba",
	}, "Codex3"); err != nil {
		t.Fatalf("SetLanguagePolicy: %v", err)
	}
	if _, err := db.SetLanguageMatrixEntry("project", "orquestador", "apps", "fr", "demo", "Codex3"); err != nil {
		t.Fatalf("SetLanguageMatrixEntry: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recPolicy := httptest.NewRecorder()
	reqPolicy := httptest.NewRequest(http.MethodGet, "/api/lenguaje/politica", nil)
	mux.ServeHTTP(recPolicy, reqPolicy)
	if recPolicy.Code != http.StatusOK {
		t.Fatalf("status politica inesperado: %d body=%s", recPolicy.Code, recPolicy.Body.String())
	}
	var policyResp struct {
		Politica *db.LanguagePolicy `json:"politica"`
	}
	if err := json.Unmarshal(recPolicy.Body.Bytes(), &policyResp); err != nil {
		t.Fatalf("decode politica: %v", err)
	}
	if policyResp.Politica == nil || policyResp.Politica.AppsDefaultLang != "en" {
		t.Fatalf("politica inesperada: %+v", policyResp.Politica)
	}

	recMatrix := httptest.NewRecorder()
	reqMatrix := httptest.NewRequest(http.MethodGet, "/api/lenguaje/matriz", nil)
	mux.ServeHTTP(recMatrix, reqMatrix)
	if recMatrix.Code != http.StatusOK {
		t.Fatalf("status matriz inesperado: %d body=%s", recMatrix.Code, recMatrix.Body.String())
	}
	var matrixResp struct {
		Matriz []*db.LanguageMatrixEntry `json:"matriz"`
	}
	if err := json.Unmarshal(recMatrix.Body.Bytes(), &matrixResp); err != nil {
		t.Fatalf("decode matriz: %v", err)
	}
	if len(matrixResp.Matriz) != 1 || matrixResp.Matriz[0].Language != "fr" {
		t.Fatalf("matriz inesperada: %+v", matrixResp.Matriz)
	}

	recResolve := httptest.NewRecorder()
	reqResolve := httptest.NewRequest(http.MethodGet, "/api/lenguaje/resolver?proyecto=orquestador&contexto=apps", nil)
	mux.ServeHTTP(recResolve, reqResolve)
	if recResolve.Code != http.StatusOK {
		t.Fatalf("status resolver inesperado: %d body=%s", recResolve.Code, recResolve.Body.String())
	}
	var resolveResp struct {
		Resolucion *db.LanguageResolution `json:"resolucion"`
	}
	if err := json.Unmarshal(recResolve.Body.Bytes(), &resolveResp); err != nil {
		t.Fatalf("decode resolucion: %v", err)
	}
	if resolveResp.Resolucion == nil || resolveResp.Resolucion.Idioma != "fr" {
		t.Fatalf("resolucion inesperada: %+v", resolveResp.Resolucion)
	}
}

func TestAPIProyectoDescubrirYGestionAgentes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(filepath.Join(proyectoDir, ".git"), 0o755); err != nil {
		t.Fatalf("crear proyecto demo: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	descubrirBody, _ := json.Marshal(apiProyectoDescubrirRequest{Ruta: tmp})
	recDescubrir := httptest.NewRecorder()
	reqDescubrir := httptest.NewRequest(http.MethodPost, "/api/proyectos/descubrir", bytes.NewReader(descubrirBody))
	mux.ServeHTTP(recDescubrir, reqDescubrir)
	if recDescubrir.Code != http.StatusCreated {
		t.Fatalf("status descubrir inesperado: %d body=%s", recDescubrir.Code, recDescubrir.Body.String())
	}

	recProyecto := httptest.NewRecorder()
	reqProyecto := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo", nil)
	mux.ServeHTTP(recProyecto, reqProyecto)
	if recProyecto.Code != http.StatusOK {
		t.Fatalf("status proyecto inesperado: %d body=%s", recProyecto.Code, recProyecto.Body.String())
	}

	actualizarBody, _ := json.Marshal(apiProyectoActualizarRequest{
		RutaAbs: filepath.Join(tmp, "demo-renombrado"),
	})
	recActualizar := httptest.NewRecorder()
	reqActualizar := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo", bytes.NewReader(actualizarBody))
	mux.ServeHTTP(recActualizar, reqActualizar)
	if recActualizar.Code != http.StatusOK {
		t.Fatalf("status actualizar proyecto inesperado: %d body=%s", recActualizar.Code, recActualizar.Body.String())
	}
	var actualizarResp map[string]*db.Proyecto
	if err := json.Unmarshal(recActualizar.Body.Bytes(), &actualizarResp); err != nil {
		t.Fatalf("decode actualizar proyecto: %v", err)
	}
	if actualizarResp["proyecto"] == nil || actualizarResp["proyecto"].RutaAbs != filepath.Join(tmp, "demo-renombrado") {
		t.Fatalf("proyecto actualizado inesperado: %+v", actualizarResp["proyecto"])
	}

	agenteBody, _ := json.Marshal(apiAgenteRequest{Nombre: "temporal", Rol: "programador"})
	recAlta := httptest.NewRecorder()
	reqAlta := httptest.NewRequest(http.MethodPost, "/api/agentes", bytes.NewReader(agenteBody))
	mux.ServeHTTP(recAlta, reqAlta)
	if recAlta.Code != http.StatusCreated {
		t.Fatalf("status alta agente inesperado: %d body=%s", recAlta.Code, recAlta.Body.String())
	}

	agenteAutoBody, _ := json.Marshal(apiAgenteRequest{Proveedor: "claude", Rol: "programador"})
	recAltaAuto := httptest.NewRecorder()
	reqAltaAuto := httptest.NewRequest(http.MethodPost, "/api/agentes", bytes.NewReader(agenteAutoBody))
	mux.ServeHTTP(recAltaAuto, reqAltaAuto)
	if recAltaAuto.Code != http.StatusCreated {
		t.Fatalf("status alta agente auto inesperado: %d body=%s", recAltaAuto.Code, recAltaAuto.Body.String())
	}
	var altaAutoResp map[string]any
	if err := json.Unmarshal(recAltaAuto.Body.Bytes(), &altaAutoResp); err != nil {
		t.Fatalf("decode alta auto: %v", err)
	}
	if altaAutoResp["nombre"] != "Claude1" {
		t.Fatalf("nombre auto inesperado: %+v", altaAutoResp)
	}

	recRetirar := httptest.NewRecorder()
	reqRetirar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/retirar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRetirar, reqRetirar)
	if recRetirar.Code != http.StatusOK {
		t.Fatalf("status retirar inesperado: %d body=%s", recRetirar.Code, recRetirar.Body.String())
	}

	recRehabilitar := httptest.NewRecorder()
	reqRehabilitar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/rehabilitar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRehabilitar, reqRehabilitar)
	if recRehabilitar.Code != http.StatusOK {
		t.Fatalf("status rehabilitar inesperado: %d body=%s", recRehabilitar.Code, recRehabilitar.Body.String())
	}
	var rehabResp apiAgenteResetReanimacionResponse
	if err := json.Unmarshal(recRehabilitar.Body.Bytes(), &rehabResp); err != nil {
		t.Fatalf("decode rehabilitar: %v", err)
	}
	if !rehabResp.OK || rehabResp.Agente != "temporal" || !rehabResp.Running {
		t.Fatalf("respuesta rehabilitar inesperada: %+v", rehabResp)
	}
}

func TestAPIProyectoGetUsaFallbackPrepareLiteTrasTimeout(t *testing.T) {
	prevTimeout := apiRuntimeProjectLookupTimeout
	prevPrimary := apiProjectLookupWithRouteFn
	prevFallback := apiProjectLookupWithRoutePrepareLiteFn
	t.Cleanup(func() {
		apiRuntimeProjectLookupTimeout = prevTimeout
		apiProjectLookupWithRouteFn = prevPrimary
		apiProjectLookupWithRoutePrepareLiteFn = prevFallback
	})

	apiRuntimeProjectLookupTimeout = 5 * time.Millisecond
	apiProjectLookupWithRouteFn = func(ref, cwdHint string) (*db.Proyecto, error) {
		time.Sleep(25 * time.Millisecond)
		return &db.Proyecto{ID: 1, Slug: ref, RutaAbs: "/slow", Tipo: db.ProyectoRepo}, nil
	}
	apiProjectLookupWithRoutePrepareLiteFn = func(ref, cwdHint string) (*db.Proyecto, error) {
		return &db.Proyecto{ID: 7, Slug: ref, RutaAbs: "/fast", Tipo: db.ProyectoRepo}, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status proyecto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]*db.Proyecto
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode proyecto: %v", err)
	}
	if payload["proyecto"] == nil || payload["proyecto"].RutaAbs != "/fast" {
		t.Fatalf("fallback prepare-lite no aplicado: %+v", payload["proyecto"])
	}
}

func TestAPIProyectoFusionar(t *testing.T) {
	prepararDBTemporalCmd(t)

	destinoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear destino: %v", err)
	}
	origenID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear origen: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Tarea origen",
		ProyectoID: &origenID,
		Modulo:     "orquestacion",
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "Codex1",
	}); err != nil {
		t.Fatalf("crear tarea origen: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoFusionRequest{Origen: "orquesta", ArchivarOrigen: true})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquestador/fusionar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status fusionar proyecto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiProyectoFusionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode fusion proyecto: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.DestinoID != destinoID {
		t.Fatalf("resultado de fusion inesperado: %+v", resp.Resultado)
	}

	tarea, err := db.GetTarea(1)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.ProyectoID == nil || *tarea.ProyectoID != destinoID {
		t.Fatalf("tarea no movida al destino: %+v", tarea.ProyectoID)
	}
	origenArchivado, err := db.GetProyecto(resp.Resultado.SlugArchivado)
	if err != nil {
		t.Fatalf("get origen archivado: %v", err)
	}
	if origenArchivado == nil || origenArchivado.Activo {
		t.Fatalf("origen no archivado: %+v", origenArchivado)
	}
}

func TestAPIAgenteResetReanimacionLimpiaContinuidadYCancelaBootstrap(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                "/tmp/orquestador",
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-old",
		ResumePayloadJSON:  `{"runtime_order":{"id":109242},"checkpoint":{"id":40275}}`,
		ResumenContinuidad: "Continuidad vieja",
		Branch:             "feature/old",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		Estado:      "pendiente",
		PayloadJSON: `{"motivo":"cambio de turno"}`,
	})
	if err != nil {
		t.Fatalf("crear handoff: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex1/reset-reanimacion", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status reset inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiAgenteResetReanimacionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK || resp.Agente != "Codex1" || !resp.Running {
		t.Fatalf("respuesta reset inesperada: %+v", resp)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		order, err := db.GetRuntimeOrder(orderID)
		if err != nil {
			t.Fatalf("runtime order: %v", err)
		}
		sesion, err := db.ObtenerUltimaSesion("Codex1", &proyectoID)
		if err != nil {
			t.Fatalf("ultima sesion: %v", err)
		}
		checkpoint, err := db.UltimoRuntimeCheckpoint("Codex1", &proyectoID)
		if err != nil {
			t.Fatalf("ultimo checkpoint: %v", err)
		}
		if order != nil && order.Estado == "cancelada" &&
			sesion != nil &&
			strings.TrimSpace(sesion.ExternalSessionID) == "" &&
			strings.TrimSpace(sesion.ResumePayloadJSON) == "" &&
			strings.TrimSpace(sesion.ResumenContinuidad) == "" &&
			checkpoint != nil && checkpoint.CheckpointKind == "manual_rehabilitation" {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("reset-reanimacion no aplico los efectos esperados a tiempo")
}

func TestAPIAgenteResetReanimacionReabreFrenteBloqueadoRecuperable(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reabrir frente al salir de cuota",
		Descripcion: "Trabajo premium recuperable",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_cuota: worker bloqueado por cuota"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeActual, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeActual == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeActual, err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("fallar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='degradado', process_state='fallido' WHERE id=?`, runtimeActual.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='activo', motivo_pausa='' WHERE nombre='Gemini1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/Gemini1/reset-reanimacion", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status reset inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiAgenteResetReanimacionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK || resp.Agente != "Gemini1" || !resp.Running {
		t.Fatalf("respuesta reset inesperada: %+v", resp)
	}

	agente := "Gemini1"
	estado := "pendiente"
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
		if err != nil {
			t.Fatalf("listar orders: %v", err)
		}
		if tarea != nil && tarea.Estado == db.TareaEnProgreso && len(orders) > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("deberia reabrir el frente y encolar control de reactivacion")
}

func TestAPIAgentesReanimacionesListaVencidasYFuturas(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar Claude1: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion codex: %v", err)
	}
	if err := db.ActivarAsignacion("Claude1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion claude: %v", err)
	}
	tareaCodex, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reactivar Codex",
		Descripcion: "frente vencido",
		ProyectoID:  &proyectoID,
		Modulo:      "cmd",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea codex: %v", err)
	}
	if err := db.TomarTarea(tareaCodex, "Codex1"); err != nil {
		t.Fatalf("tomar tarea codex: %v", err)
	}
	if err := db.BloquearTarea(tareaCodex, "Codex1", "worker bloqueado por cuota"); err != nil {
		t.Fatalf("bloquear tarea codex: %v", err)
	}
	tareaClaude, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reactivar Claude",
		Descripcion: "frente futuro",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea claude: %v", err)
	}
	if err := db.TomarTarea(tareaClaude, "Claude1"); err != nil {
		t.Fatalf("tomar tarea claude: %v", err)
	}
	if err := db.BloquearTarea(tareaClaude, "Claude1", "worker bloqueado por cuota"); err != nil {
		t.Fatalf("bloquear tarea claude: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='runtime_panic', reanimar_at=datetime('now','-5 minutes') WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("actualizar codex: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='runtime_panic', reanimar_at=datetime('now','+30 minutes') WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("actualizar claude: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recDue := httptest.NewRecorder()
	reqDue := httptest.NewRequest(http.MethodGet, "/api/agentes/reanimaciones", nil)
	mux.ServeHTTP(recDue, reqDue)
	if recDue.Code != http.StatusOK {
		t.Fatalf("status due inesperado: %d body=%s", recDue.Code, recDue.Body.String())
	}
	var dueResp apiAgenteReanimationsResponse
	if err := json.Unmarshal(recDue.Body.Bytes(), &dueResp); err != nil {
		t.Fatalf("decode due: %v", err)
	}
	if len(dueResp.Rows) != 1 || dueResp.Rows[0].Name != "Codex1" || !dueResp.Rows[0].Due {
		t.Fatalf("rows due inesperadas: %+v", dueResp.Rows)
	}
	if len(dueResp.Rows[0].Leases) != 1 || dueResp.Rows[0].Leases[0].TaskID != tareaCodex {
		t.Fatalf("leases due inesperadas: %+v", dueResp.Rows[0].Leases)
	}

	recAll := httptest.NewRecorder()
	reqAll := httptest.NewRequest(http.MethodGet, "/api/agentes/reanimaciones?all=true", nil)
	mux.ServeHTTP(recAll, reqAll)
	if recAll.Code != http.StatusOK {
		t.Fatalf("status all inesperado: %d body=%s", recAll.Code, recAll.Body.String())
	}
	var allResp apiAgenteReanimationsResponse
	if err := json.Unmarshal(recAll.Body.Bytes(), &allResp); err != nil {
		t.Fatalf("decode all: %v", err)
	}
	if len(allResp.Rows) != 2 {
		t.Fatalf("rows all=%d, want 2", len(allResp.Rows))
	}
	if !allResp.Rows[0].Due || allResp.Rows[1].Due {
		t.Fatalf("orden due/future inesperado: %+v", allResp.Rows)
	}

	if _, err := db.DB.Exec(`UPDATE agentes SET habilitado=0 WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("deshabilitar claude: %v", err)
	}
	recActive := httptest.NewRecorder()
	reqActive := httptest.NewRequest(http.MethodGet, "/api/agentes/reanimaciones?all=true&activos=true", nil)
	mux.ServeHTTP(recActive, reqActive)
	if recActive.Code != http.StatusOK {
		t.Fatalf("status active inesperado: %d body=%s", recActive.Code, recActive.Body.String())
	}
	var activeResp apiAgenteReanimationsResponse
	if err := json.Unmarshal(recActive.Body.Bytes(), &activeResp); err != nil {
		t.Fatalf("decode active: %v", err)
	}
	if len(activeResp.Rows) != 1 || activeResp.Rows[0].Name != "Codex1" {
		t.Fatalf("rows active inesperadas: %+v", activeResp.Rows)
	}
}

func TestAPIProyectoFusionarRechazaDesactivarArchivoOrigen(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear destino: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear origen: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoFusionRequest{Origen: "orquesta", ArchivarOrigen: false})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquestador/fusionar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado al desactivar archivado: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "todavía no está soportado") {
		t.Fatalf("error inesperado: %s", rec.Body.String())
	}
}

func TestAPIProyectoFabricarApp(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoFabricarAppRequest{
		Tipo:        "web_api",
		Nombre:      "Demo App",
		Descripcion: "Aplicacion demo",
		Frontend:    true,
		API:         true,
		Docker:      true,
		I18n:        true,
		Idiomas:     []string{"es", "en"},
		Por:         "Codex3",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/fabricar-app", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status fabricar-app inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiProyectoFabricarAppResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode fabricar-app: %v", err)
	}
	if !resp.OK || resp.Slug != "demo-app" || resp.Created < 4 {
		t.Fatalf("respuesta fabricar-app inesperada: %+v", resp)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != resp.Created {
		t.Fatalf("tareas creadas=%d respuesta=%d", len(tareas), resp.Created)
	}
}

func TestAPIProyectoIdiomas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	rutaProyecto := filepath.Join(tmp, "demo-app")
	if err := os.MkdirAll(rutaProyecto, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	if _, err := i18n.MaterializeProjectSkeleton(i18n.ProjectSkeletonSpec{
		RootDir:         rutaProyecto,
		DefaultLanguage: "es",
		Languages:       []string{"es"},
		Domains:         []string{"common", "errors"},
	}); err != nil {
		t.Fatalf("materialize i18n: %v", err)
	}
	enCommon := filepath.Join(rutaProyecto, "i18n", "es", "common.json")
	customRaw := []byte("{\n  \"custom\": \"persist\"\n}\n")
	if err := os.WriteFile(enCommon, customRaw, 0o644); err != nil {
		t.Fatalf("rewrite common.json: %v", err)
	}

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: rutaProyecto,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoIdiomasRequest{
		Idiomas: []string{"en", "fr"},
		Por:     "Codex3",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/idiomas", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status idiomas inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiProyectoIdiomasResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode idiomas: %v", err)
	}
	if !resp.OK || resp.Slug != "demo-app" || resp.Created < 3 || len(resp.Idiomas) != 2 {
		t.Fatalf("respuesta idiomas inesperada: %+v", resp)
	}

	for _, key := range []string{"i18n_expand_en", "documentacion_expand_en", "qa_i18n_expand_en", "i18n_expand_fr"} {
		if id := db.GetTareaIDBlueprintKey(proyectoID, key); id <= 0 {
			t.Fatalf("faltaba blueprint %s", key)
		}
	}
	for _, rel := range []string{"i18n/en/common.json", "i18n/en/errors.json", "i18n/fr/common.json"} {
		if _, err := os.Stat(filepath.Join(rutaProyecto, rel)); err != nil {
			t.Fatalf("falta fichero expandido %s: %v", rel, err)
		}
	}
	raw, err := os.ReadFile(enCommon)
	if err != nil {
		t.Fatalf("leer common.json es: %v", err)
	}
	if string(raw) != string(customRaw) {
		t.Fatalf("common.json existente sobrescrito:\n%s", string(raw))
	}
}

func TestAPIProyectoSharedContext(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if proyectoID <= 0 {
		t.Fatalf("id proyecto invalido")
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	createBody, _ := json.Marshal(apiProyectoSharedContextCreateRequest{
		Tipo:    "restriccion",
		Titulo:  "Preservar hexagonalidad",
		Detalle: "No mover dominio a adaptadores",
		Peso:    3,
		Origen:  "humano",
	})
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/contexto-compartido", bytes.NewReader(createBody))
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("status create shared context inesperado: %d body=%s", createRec.Code, createRec.Body.String())
	}

	var createResp apiProyectoSharedContextMutationResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create shared context: %v", err)
	}
	if !createResp.OK || createResp.ID <= 0 {
		t.Fatalf("respuesta create shared context inesperada: %+v", createResp)
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/contexto-compartido", nil)
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("status list shared context inesperado: %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listResp apiProyectoSharedContextResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list shared context: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("items shared context inesperados: %d", len(listResp.Items))
	}
	if listResp.Items[0].Titulo != "Preservar hexagonalidad" {
		t.Fatalf("titulo shared context inesperado: %+v", listResp.Items[0])
	}
	if !strings.Contains(listResp.Summary, "Preservar hexagonalidad") {
		t.Fatalf("summary shared context inesperado: %q", listResp.Summary)
	}
}

func TestAPIProyectoOperacionGetYPost(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, nombre := range []string{"CodexSupervisor", "CodexReview"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/operacion", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("status get operacion inesperado: %d body=%s", recGet.Code, recGet.Body.String())
	}
	var getResp apiProyectoOperacionResponse
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get operacion: %v", err)
	}
	if getResp.Operacion == nil || getResp.Operacion.EstadoOperativo != db.ProyectoOperativoActivo {
		t.Fatalf("operacion inicial inesperada: %+v", getResp.Operacion)
	}

	body, _ := json.Marshal(apiProyectoOperacionSetRequest{
		EstadoOperativo:  string(db.ProyectoOperativoEsperandoHumano),
		Motivo:           "esperando aprobacion",
		ObjetivoPct:      60,
		MinAgentes:       1,
		MaxAgentes:       3,
		Prioridad:        250,
		ResumeAutomatico: true,
	})
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/operacion", bytes.NewReader(body))
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("status post operacion inesperado: %d body=%s", recPost.Code, recPost.Body.String())
	}
	var postResp apiProyectoOperacionResponse
	if err := json.Unmarshal(recPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode post operacion: %v", err)
	}
	if postResp.Operacion == nil || postResp.Operacion.EstadoOperativo != db.ProyectoOperativoEsperandoHumano || postResp.Operacion.ObjetivoPct != 60 {
		t.Fatalf("operacion guardada inesperada: %+v", postResp.Operacion)
	}
}

func TestAPIProyectoAutonomiaGetPostYCiclos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/autonomia", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("status get autonomia inesperado: %d body=%s", recGet.Code, recGet.Body.String())
	}
	var getResp apiProyectoAutonomiaResponse
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get autonomia: %v", err)
	}
	if getResp.Policy == nil || getResp.Policy.EstadoAutonomia != db.AutonomiaProyectoActiva || getResp.Policy.Enabled {
		t.Fatalf("autonomia inicial inesperada: %+v", getResp.Policy)
	}

	body, _ := json.Marshal(apiProyectoAutonomiaSaveRequest{
		Enabled:              true,
		ObjetivoGeneral:      "terminar el proyecto sin intervención humana",
		DefinitionOfDoneJSON: `{"tests":"green"}`,
		MaxWorkers:           3,
		SupervisorAgente:     "CodexSupervisor",
		ReviewerAgente:       "CodexReview",
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      string(db.AutonomiaProyectoActiva),
	})
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/autonomia", bytes.NewReader(body))
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("status post autonomia inesperado: %d body=%s", recPost.Code, recPost.Body.String())
	}
	var postResp apiProyectoAutonomiaResponse
	if err := json.Unmarshal(recPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode post autonomia: %v", err)
	}
	if postResp.Policy == nil || !postResp.Policy.Enabled || postResp.Policy.MaxWorkers != 3 {
		t.Fatalf("autonomia guardada inesperada: %+v", postResp.Policy)
	}
	if postResp.Policy.SupervisorAgente != "CodexSupervisor" {
		t.Fatalf("supervisor de autonomia inesperado: %+v", postResp.Policy)
	}
	if strings.TrimSpace(postResp.Policy.ReviewerAgente) == "" {
		t.Fatalf("reviewer de autonomia vacío: %+v", postResp.Policy)
	}

	if _, err := supervisionService.RegisterCycle("demo-app", supervisionapp.CycleInput{
		Kind:         "supervision",
		Agente:       "Codex3",
		InputJSON:    `{"reason":"periodic"}`,
		DecisionJSON: `{"decision":"seguir"}`,
		Resultado:    "ok",
	}); err != nil {
		t.Fatalf("RegisterCycle: %v", err)
	}

	recCycles := httptest.NewRecorder()
	reqCycles := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/autonomia/ciclos?kind=supervision&limit=5", nil)
	mux.ServeHTTP(recCycles, reqCycles)
	if recCycles.Code != http.StatusOK {
		t.Fatalf("status ciclos inesperado: %d body=%s", recCycles.Code, recCycles.Body.String())
	}
	var cyclesResp struct {
		Cycles []*supervisionapp.Cycle `json:"cycles"`
	}
	if err := json.Unmarshal(recCycles.Body.Bytes(), &cyclesResp); err != nil {
		t.Fatalf("decode ciclos: %v", err)
	}
	if len(cyclesResp.Cycles) != 1 || cyclesResp.Cycles[0].Kind != "supervision" {
		t.Fatalf("ciclos inesperados: %+v", cyclesResp.Cycles)
	}
}

func TestAPIProyectoAutonomiaPostAutocompletaReviewerPersistente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, nombre := range []string{"Codex1", "Codex2", "Codex3"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", nombre, err)
		}
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoAutonomiaSaveRequest{
		Enabled:              true,
		ObjetivoGeneral:      "terminar el proyecto sin intervención humana",
		DefinitionOfDoneJSON: `{"tests":"green"}`,
		MaxWorkers:           2,
		SupervisorAgente:     "codex1",
		ReviewerAgente:       "",
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      string(db.AutonomiaProyectoActiva),
	})
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/autonomia", bytes.NewReader(body))
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("status post autonomia inesperado: %d body=%s", recPost.Code, recPost.Body.String())
	}
	var postResp apiProyectoAutonomiaResponse
	if err := json.Unmarshal(recPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode post autonomia: %v", err)
	}
	if postResp.Policy == nil {
		t.Fatalf("policy ausente: %s", recPost.Body.String())
	}
	if postResp.Policy.SupervisorAgente != "Codex1" {
		t.Fatalf("supervisor inesperado: %+v", postResp.Policy)
	}
	if postResp.Policy.ReviewerAgente == "" || postResp.Policy.ReviewerAgente == "codex1" {
		t.Fatalf("reviewer no autocompletado correctamente: %+v", postResp.Policy)
	}
	if postResp.Policy.ReviewerAgente != "Codex2" {
		t.Fatalf("reviewer inesperado: %+v", postResp.Policy)
	}
}

func TestAPIProyectoMicrocicloActivaOperacionAutonomiaYTarea(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: filepath.Join(tmp, "orquesta"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoAPIID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "api",
		Nombre:  "api",
		RutaAbs: filepath.Join(tmp, "api"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto api: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoAPIID, "frente previo"); err != nil {
		t.Fatalf("activar asignacion previa: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente: "Codex1",
		Notas:  "slice=transcript",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode microciclo: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Policy == nil || resp.Resultado.Tarea == nil || resp.Resultado.Fase == nil {
		t.Fatalf("resultado incompleto: %+v", resp.Resultado)
	}
	if resp.Resultado.Dispatch == nil {
		t.Fatalf("dispatch inesperado: %+v", resp.Resultado)
	}
	if resp.Resultado.Policy.SupervisorAgente != "Codex1" || !resp.Resultado.Policy.Enabled {
		t.Fatalf("policy inesperada: %+v", resp.Resultado.Policy)
	}
	if resp.Resultado.Fase.Nombre != "implementacion" || !strings.EqualFold(resp.Resultado.Fase.Estado, "activa") {
		t.Fatalf("fase inesperada: %+v", resp.Resultado.Fase)
	}
	if !strings.Contains(resp.Resultado.Tarea.Notas, microcicloRefactorNotasTag) {
		t.Fatalf("tarea semilla sin tag: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "No abras shims de compatibilidad") {
		t.Fatalf("la tarea semilla debe prohibir shims de compatibilidad fuera del frente: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "tmux_cli_session") || !strings.Contains(resp.Resultado.Tarea.Descripcion, "process_pty_cli") {
		t.Fatalf("la tarea semilla debe fijar TMUX como carril canonico premium y relegar PTY legacy: %+v", resp.Resultado.Tarea)
	}
	op, err := db.GetProyectoOperacion(resp.Resultado.Proyecto.ID)
	if err != nil {
		t.Fatalf("get operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoActivo || !op.ResumeAutomatico || op.MaxAgentes != 3 {
		t.Fatalf("operacion inesperada: %+v", op)
	}
	if resp.Resultado.Policy.MaxWorkers != 3 {
		t.Fatalf("policy max_workers inesperado: %+v", resp.Resultado.Policy)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Codex1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacion.ProyectoSlug != "orquesta" {
		t.Fatalf("el microciclo deberia fijar la asignacion activa sobre el proyecto objetivo: %+v", asignacion)
	}
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status microciclo segundo intento inesperado: %d body=%s", rec2.Code, rec2.Body.String())
	}
	var resp2 apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode microciclo 2: %v", err)
	}
	if resp2.Resultado == nil || !resp2.Resultado.TareaReutilizada || resp2.Resultado.Tarea.ID != resp.Resultado.Tarea.ID {
		t.Fatalf("la tarea semilla no se reutilizo: first=%+v second=%+v", resp.Resultado, resp2.Resultado)
	}
}

func TestAPIProyectoMicrocicloLimpiaPruebasAntesDeActivar(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:      "Tarea vieja de prueba",
		Descripcion: "Debe ir a backlog antes del nuevo microciclo",
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
		Agente:      "Codex1",
		Proyecto:    "orquesta",
		Notas:       "tmp",
	})
	if err != nil {
		t.Fatalf("crear tarea vieja: %v", err)
	}
	if err := tareasService.Start(tareaID, "Codex1"); err != nil {
		t.Fatalf("start tarea vieja: %v", err)
	}
	proyecto, err := db.GetProyecto("orquesta")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyecto.ID,
		CWD:         tmp,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("fallar handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyecto.ID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"ruido viejo"}`,
	}); err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	rutaWorktreeVieja := filepath.Join(tmp, ".orquesta-worktrees", "wt-codex1")
	if err := os.MkdirAll(rutaWorktreeVieja, 0o755); err != nil {
		t.Fatalf("mkdir worktree vieja: %v", err)
	}
	resWT, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyecto.ID,
		"Codex1",
		"wt-codex1",
		rutaWorktreeVieja,
		"orq/orquesta/codex1",
		"HEAD",
		"activa",
		"prueba_vieja",
	)
	if err != nil {
		t.Fatalf("crear worktree vieja: %v", err)
	}
	worktreeViejaID, _ := resWT.LastInsertId()

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	body, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo limpio inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	tareaVieja, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea vieja: %v", err)
	}
	if tareaVieja.Estado != db.EstadoBacklog {
		t.Fatalf("la tarea vieja deberia pasar a backlog: %+v", tareaVieja)
	}
	pendiente := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex1"), ProyectoID: &proyecto.ID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	for _, msg := range mailbox {
		if msg != nil && strings.Contains(msg.PayloadJSON, "ruido viejo") {
			t.Fatalf("la mailbox vieja deberia quedar consumida: %+v", msg)
		}
	}
	handles, err := db.ListarRuntimeHandles(strPtr("Codex1"))
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	for _, item := range handles {
		if item != nil && item.ID == handle.ID {
			t.Fatalf("el handle fallido viejo deberia purgarse: %+v", item)
		}
	}
	worktreeVieja, err := db.CoordinationWorktreeRepository().GetByID(worktreeViejaID)
	if err != nil {
		t.Fatalf("get worktree vieja: %v", err)
	}
	if worktreeVieja == nil || worktreeVieja.State != coordinacion.WorktreeClosed {
		t.Fatalf("la worktree vieja deberia quedar cerrada: %+v", worktreeVieja)
	}
}

func TestAPIProyectoMicrocicloCreaTareaNuevaTrasLimpiarPruebas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoMicrocicloRequest{Agente: "Codex1"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo inicial inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode microciclo inicial: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Tarea == nil {
		t.Fatalf("resultado inicial inesperado: %+v", resp.Resultado)
	}

	bodyLimpio, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(bodyLimpio))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status microciclo limpio inesperado: %d body=%s", rec2.Code, rec2.Body.String())
	}
	var resp2 apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode microciclo limpio: %v", err)
	}
	if resp2.Resultado == nil || resp2.Resultado.Tarea == nil {
		t.Fatalf("resultado limpio inesperado: %+v", resp2.Resultado)
	}
	if resp2.Resultado.TareaReutilizada || resp2.Resultado.Tarea.ID == resp.Resultado.Tarea.ID {
		t.Fatalf("deberia crear una tarea semilla nueva tras limpiar pruebas: inicial=%+v limpio=%+v", resp.Resultado, resp2.Resultado)
	}
	if resp2.Resultado.Tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea semilla deberia volver a en_progreso: %+v", resp2.Resultado.Tarea)
	}
}

func TestAPIProyectoMicrocicloLimpioReutilizaTareaSiAgenteEstaEnCuota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoMicrocicloRequest{Agente: "Codex1"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo inicial inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode microciclo inicial: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Tarea == nil {
		t.Fatalf("resultado inicial inesperado: %+v", resp.Resultado)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("set estado_cuota: %v", err)
	}

	bodyLimpio, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(bodyLimpio))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status microciclo limpio en cuota inesperado: %d body=%s", rec2.Code, rec2.Body.String())
	}
	var resp2 apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode microciclo limpio en cuota: %v", err)
	}
	if resp2.Resultado == nil || resp2.Resultado.Tarea == nil {
		t.Fatalf("resultado limpio en cuota inesperado: %+v", resp2.Resultado)
	}
	if !resp2.Resultado.TareaReutilizada || resp2.Resultado.Tarea.ID != resp.Resultado.Tarea.ID {
		t.Fatalf("deberia reutilizar la tarea semilla existente bajo cuota: inicial=%+v limpio=%+v", resp.Resultado, resp2.Resultado)
	}
	if resp2.Resultado.Dispatch == nil || resp2.Resultado.Dispatch.DispatchRuntime == nil || resp2.Resultado.Dispatch.DispatchRuntime.Estado != "cuota_bloqueada" {
		t.Fatalf("el microciclo en cuota deberia devolver dispatch bloqueado: %+v", resp2.Resultado.Dispatch)
	}
}

func TestAPIProyectoMicrocicloLimpioNoReutilizaTareaSiCuotaVisibleEstaVigente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoDir := filepath.Join(tmp, "repo-microciclo-visible")
	if err := os.MkdirAll(proyectoDir, 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: proyectoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoMicrocicloRequest{Agente: "Codex1"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo inicial inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode microciclo inicial: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Tarea == nil {
		t.Fatalf("resultado inicial inesperado: %+v", resp.Resultado)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='agotado', motivo_pausa='Cuota diaria agotada' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("set estado_cuota legacy: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         proyectoDir,
		Herramienta: "codex-cli",
	})
	if err != nil || sesion == nil {
		t.Fatalf("iniciar sesion visible: %+v err=%v", sesion, err)
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
	if err := db.MarcarRuntimesCerradosPorAgente("Codex1"); err != nil {
		t.Fatalf("cerrar runtimes sinteticos: %v", err)
	}
	if err := db.MarcarRuntimeHandlesCerradosPorAgente("Codex1"); err != nil {
		t.Fatalf("cerrar handles sinteticos: %v", err)
	}
	if err := db.AparcarSesionActiva("Codex1", nil); err != nil {
		t.Fatalf("aparcar sesion activa sintetica: %v", err)
	}

	bodyLimpio, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(bodyLimpio))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status microciclo limpio con cuota visible inesperado: %d body=%s", rec2.Code, rec2.Body.String())
	}
	var resp2 apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode microciclo limpio con cuota visible: %v", err)
	}
	if resp2.Resultado == nil || resp2.Resultado.Tarea == nil {
		t.Fatalf("resultado limpio con cuota visible inesperado: %+v", resp2.Resultado)
	}
	if resp2.Resultado.TareaReutilizada || resp2.Resultado.Tarea.ID == resp.Resultado.Tarea.ID {
		t.Fatalf("no deberia reutilizar la tarea semilla vieja si la cuota visible ya esta vigente: inicial=%+v limpio=%+v", resp.Resultado, resp2.Resultado)
	}
	if resp2.Resultado.Dispatch == nil || resp2.Resultado.Dispatch.DispatchRuntime == nil || strings.HasPrefix(resp2.Resultado.Dispatch.DispatchRuntime.Estado, "cuota_bloqueada") {
		t.Fatalf("el microciclo no deberia quedar bloqueado por cuota legacy con presupuesto visible fresco: %+v", resp2.Resultado.Dispatch)
	}
}

func TestAPIProyectoMicrocicloSincronizaDirtyWorkspaceEnWorktreeActiva(t *testing.T) {
	t.Setenv("ORQUESTA_SYNC_DIRTY_WORKSPACE_TO_WORKTREE", "true")

	tmp := prepararDBTemporalCmd(t)
	repo := filepath.Join(tmp, "repo-sync")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("dirty-local\n"), 0o644); err != nil {
		t.Fatalf("write dirty: %v", err)
	}

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	body, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo sync inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	worktree, err := gitService.ResolveActiveWorktree("orquesta", "Codex1")
	if err != nil {
		t.Fatalf("ResolveActiveWorktree: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(worktree.RutaAbs, "README.md"))
	if err != nil {
		t.Fatalf("leer README sincronizado: %v", err)
	}
	if string(raw) != "dirty-local\n" {
		t.Fatalf("la worktree deberia heredar el dirty workspace local, got=%q", string(raw))
	}
}

func TestAPIProyectoMicrocicloNoSincronizaDirtyWorkspacePorDefecto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := filepath.Join(tmp, "repo-clean")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("dirty-local\n"), 0o644); err != nil {
		t.Fatalf("write dirty: %v", err)
	}

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	body, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo clean inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	worktree, err := gitService.ResolveActiveWorktree("orquesta", "Codex1")
	if err != nil {
		t.Fatalf("ResolveActiveWorktree: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(worktree.RutaAbs, "README.md"))
	if err != nil {
		t.Fatalf("leer README worktree: %v", err)
	}
	if string(raw) != "base\n" {
		t.Fatalf("la worktree premium deberia arrancar limpia por defecto, got=%q", string(raw))
	}
}

func TestAPIProyectoMicrocicloEscribeInboxEnWorktreeActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := filepath.Join(tmp, "repo-inbox")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	body, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente:         "Codex1",
		LimpiarPruebas: true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo inbox inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode microciclo inbox: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Tarea == nil {
		t.Fatalf("resultado microciclo inbox inesperado: %+v", resp.Resultado)
	}

	worktree, err := gitService.ResolveActiveWorktree("orquesta", "Codex1")
	if err != nil {
		t.Fatalf("ResolveActiveWorktree: %v", err)
	}
	rawInbox, err := os.ReadFile(filepath.Join(worktree.RutaAbs, ".orquesta-inbox.md"))
	if err != nil {
		t.Fatalf("leer inbox worktree: %v", err)
	}
	if !strings.Contains(string(rawInbox), "# Microtarea Activa de Orquesta") {
		t.Fatalf("la inbox deberia materializarse en la worktree activa: %s", string(rawInbox))
	}
	if !strings.Contains(string(rawInbox), fmt.Sprintf("`#%d %s`", resp.Resultado.Tarea.ID, resp.Resultado.Tarea.Titulo)) {
		t.Fatalf("la inbox deberia reflejar la tarea activa: %s", string(rawInbox))
	}
}

func TestAPIProyectoMicrocicloEncolaStartSiAgenteNoTieneRuntime(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	body, _ := json.Marshal(apiProyectoMicrocicloRequest{
		Agente: "Claude1",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquesta/autonomia/microciclo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status microciclo inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoMicrocicloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode microciclo: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.Policy == nil || resp.Resultado.Tarea == nil {
		t.Fatalf("resultado microciclo inesperado: %+v", resp.Resultado)
	}
	if !strings.EqualFold(resp.Resultado.Policy.SupervisorAgente, "Claude1") {
		t.Fatalf("el microciclo deberia fijar a Claude1 como supervisor: %+v", resp.Resultado.Policy)
	}
	if resp.Resultado.Tarea.ID <= 0 {
		t.Fatalf("el microciclo deberia dejar una tarea semilla usable: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "Write-set exclusivo:") {
		t.Fatalf("el microciclo deberia acotar el frente con write-set: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "Trabaja solo dentro de ese write_set") {
		t.Fatalf("el microciclo deberia dejar el write_set como contrato exclusivo: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "No ensanches firmas ni APIs del nucleo") {
		t.Fatalf("el microciclo deberia bloquear ensanches de API fuera del frente: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "Antes de ampliar validacion") {
		t.Fatalf("el microciclo deberia forzar patch pequeno antes de ensanchar validacion: %+v", resp.Resultado.Tarea)
	}
	if !strings.Contains(resp.Resultado.Tarea.Descripcion, "no esperes una microtarea nueva") {
		t.Fatalf("el microciclo deberia forzar continuidad sobre el siguiente caso adyacente dentro del mismo write-set: %+v", resp.Resultado.Tarea)
	}
	if resp.Resultado.Dispatch == nil || resp.Resultado.Dispatch.Despacho == nil {
		t.Fatalf("el microciclo deberia construir un despacho usable: %+v", resp.Resultado.Dispatch)
	}
	if !strings.EqualFold(strings.TrimSpace(resp.Resultado.Dispatch.Despacho.AgenteSugerido), "Claude1") {
		t.Fatalf("el microciclo deberia forzar Claude1 como agente sugerido: %+v", resp.Resultado.Dispatch.Despacho)
	}
	if resp.Resultado.Dispatch.DispatchRuntime == nil || strings.EqualFold(strings.TrimSpace(resp.Resultado.Dispatch.DispatchRuntime.Estado), "sin_agente") {
		t.Fatalf("el microciclo no deberia quedarse sin agente en el dispatch runtime: %+v", resp.Resultado.Dispatch.DispatchRuntime)
	}
}

func TestConstruirInboxMicrocicloMarkdownImpulsaSiguienteCasoAdyacente(t *testing.T) {
	inbox := construirInboxMicrocicloMarkdown(&db.Proyecto{Slug: "orquesta"}, &db.Tarea{
		ID:          1,
		Titulo:      "Slice premium",
		Descripcion: "Descripcion de prueba",
	})
	if !strings.Contains(inbox, "no esperes otra microtarea") {
		t.Fatalf("la inbox debe impedir que el agente espere otra microtarea: %s", inbox)
	}
	if !strings.Contains(inbox, "Primer paso obligatorio") {
		t.Fatalf("la inbox debe forzar un primer patch pequeno antes del broad scan: %s", inbox)
	}
	if !strings.Contains(inbox, "Primer movimiento obligatorio") {
		t.Fatalf("la inbox debe forzar una primera accion directa de patch: %s", inbox)
	}
	if !strings.Contains(inbox, "Si el simbolo exacto no esta claro tras leer la inbox, usa entonces: rg -n") {
		t.Fatalf("la inbox debe dejar rg solo como fallback despues de intentar el primer patch: %s", inbox)
	}
	if !strings.Contains(inbox, "No gastes el primer ciclo en git diff, git status") {
		t.Fatalf("la inbox debe prohibir perder el primer ciclo en git diff/status: %s", inbox)
	}
	if !strings.Contains(inbox, "siguiente caso adyacente mas pequeno y verificable") {
		t.Fatalf("la inbox debe empujar el siguiente caso adyacente dentro del write-set: %s", inbox)
	}
	if !strings.Contains(inbox, "No cierres diciendo que esperas otra microtarea") {
		t.Fatalf("la inbox debe prohibir cerrar esperando otra microtarea: %s", inbox)
	}
}

func runGitCmdAPITest(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v output=%s", args, err, string(out))
	}
}

func TestAPIRuntimeCanonicalizaNombreAgenteEnMailboxYPurgas(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
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
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "claude1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"hola"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	reqList := httptest.NewRequest(http.MethodGet, "/api/runtime-mailbox?to_agente=Claude1&estado=pendiente", nil)
	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("listar mailbox status=%d body=%s", recList.Code, recList.Body.String())
	}
	var listed struct {
		Mailbox []*db.RuntimeMailboxMessage `json:"mailbox"`
	}
	if err := json.Unmarshal(recList.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode listar mailbox: %v", err)
	}
	if len(listed.Mailbox) != 1 || listed.Mailbox[0] == nil || listed.Mailbox[0].ToAgente != "claude1" {
		t.Fatalf("mailbox canonicalizada inesperada: %+v", listed.Mailbox)
	}

	body, _ := json.Marshal(apiRuntimeMailboxClearRequest{ToAgente: "Claude1", Estados: []string{"pendiente"}})
	reqClear := httptest.NewRequest(http.MethodPost, "/api/runtime-mailbox/limpiar", bytes.NewReader(body))
	reqClear.Header.Set("Content-Type", "application/json")
	recClear := httptest.NewRecorder()
	mux.ServeHTTP(recClear, reqClear)
	if recClear.Code != http.StatusOK {
		t.Fatalf("limpiar mailbox status=%d body=%s", recClear.Code, recClear.Body.String())
	}
	var cleared apiRuntimeMailboxClearResponse
	if err := json.Unmarshal(recClear.Body.Bytes(), &cleared); err != nil {
		t.Fatalf("decode limpiar mailbox: %v", err)
	}
	if cleared.Cleared != 1 {
		t.Fatalf("mailbox limpiada inesperada: %+v", cleared)
	}
}

func TestAPIReviewGatesCrearListarYResolver(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	createBody, _ := json.Marshal(apiReviewGateCreateRequest{
		Proyecto:       "demo-app",
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReview",
		SeverityMax:    "medium",
		FindingsJSON:   `[{"kind":"coverage","detail":"falta regression"}]`,
	})
	recCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/review-gates", bytes.NewReader(createBody))
	mux.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("status create gate inesperado: %d body=%s", recCreate.Code, recCreate.Body.String())
	}
	var createResp struct {
		Gate *reviewapp.Gate `json:"gate"`
	}
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create gate: %v", err)
	}
	if createResp.Gate == nil || createResp.Gate.ID == 0 || createResp.Gate.Estado != reviewapp.GateStatePending {
		t.Fatalf("gate creada inesperada: %+v", createResp.Gate)
	}

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/review-gates?proyecto=demo-app&estado=pendiente&limit=5", nil)
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("status list gates inesperado: %d body=%s", recList.Code, recList.Body.String())
	}
	var listResp struct {
		Gates []*reviewapp.Gate `json:"gates"`
	}
	if err := json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list gates: %v", err)
	}
	if len(listResp.Gates) != 1 || listResp.Gates[0].ID != createResp.Gate.ID {
		t.Fatalf("listado de gates inesperado: %+v", listResp.Gates)
	}

	resolveBody, _ := json.Marshal(apiReviewGateResolveRequest{
		Estado:         reviewapp.GateStateApproved,
		ReviewerAgente: "CodexReview",
		FindingsJSON:   `[]`,
	})
	recResolve := httptest.NewRecorder()
	reqResolve := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/review-gates/%d/resolver", createResp.Gate.ID), bytes.NewReader(resolveBody))
	mux.ServeHTTP(recResolve, reqResolve)
	if recResolve.Code != http.StatusOK {
		t.Fatalf("status resolve gate inesperado: %d body=%s", recResolve.Code, recResolve.Body.String())
	}
	var resolveResp struct {
		Gate *reviewapp.Gate `json:"gate"`
	}
	if err := json.Unmarshal(recResolve.Body.Bytes(), &resolveResp); err != nil {
		t.Fatalf("decode resolve gate: %v", err)
	}
	if resolveResp.Gate == nil || resolveResp.Gate.Estado != reviewapp.GateStateApproved || resolveResp.Gate.ResolvedAt == nil {
		t.Fatalf("gate resuelta inesperada: %+v", resolveResp.Gate)
	}
}

func TestAPIAgenteControlPlaneAcciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := db.IniciarSesion("Codex1"); err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	if _, err := db.IniciarSesion("Codex2"); err != nil {
		t.Fatalf("iniciar sesion destino: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Handoff API",
		Modulo:    "orquestador",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	pausaBody, _ := json.Marshal(apiAgentePausarRequest{Agente: "Codex1", Minutos: 15, Motivo: "rate limit"})
	recPausa := httptest.NewRecorder()
	reqPausa := httptest.NewRequest(http.MethodPost, "/api/agente/pausar", bytes.NewReader(pausaBody))
	mux.ServeHTTP(recPausa, reqPausa)
	if recPausa.Code != http.StatusOK {
		t.Fatalf("status pausar inesperado: %d body=%s", recPausa.Code, recPausa.Body.String())
	}

	recReset := httptest.NewRecorder()
	reqReset := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex1/reset-reanimacion", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("status reset inesperado: %d body=%s", recReset.Code, recReset.Body.String())
	}

	handoffBody, _ := json.Marshal(apiRuntimeHandoffRequest{
		AgenteOrigen:  "Codex1",
		AgenteDestino: "Codex2",
		TareaID:       tareaID,
		Motivo:        "cambio de turno",
		Resumen:       "seguir desde api",
	})
	recHandoff := httptest.NewRecorder()
	reqHandoff := httptest.NewRequest(http.MethodPost, "/api/agente/handoff", bytes.NewReader(handoffBody))
	mux.ServeHTTP(recHandoff, reqHandoff)
	if recHandoff.Code != http.StatusCreated {
		t.Fatalf("status handoff inesperado: %d body=%s", recHandoff.Code, recHandoff.Body.String())
	}

	var resp struct {
		ID      int64 `json:"id"`
		OrderID int64 `json:"order_id"`
	}
	if err := json.Unmarshal(recHandoff.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode handoff: %v", err)
	}
	if resp.OrderID == 0 && resp.ID == 0 {
		t.Fatalf("order id inesperado: %d", resp.OrderID)
	}

	recEliminar := httptest.NewRecorder()
	reqEliminar := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex3/eliminar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recEliminar, reqEliminar)
	if recEliminar.Code != http.StatusOK {
		t.Fatalf("status eliminar inesperado: %d body=%s", recEliminar.Code, recEliminar.Body.String())
	}
}

func TestAPIAgenteFusionarCreaRespaldoYMueveReferencias(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("registrar codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Fusion API",
		Modulo:    "orquestador",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "codex1",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	backupDir := filepath.Join(tmp, "backups")
	body, _ := json.Marshal(apiAgenteFusionRequest{
		Destino:          "Codex1",
		DestinoRespaldo:  backupDir,
		EtiquetaRespaldo: "fusion-api",
		Retener:          2,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/codex1/fusionar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status fusion inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteFusionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode fusion: %v", err)
	}
	if resp.RutaRespaldo == "" {
		t.Fatalf("ruta respaldo vacia")
	}
	if _, err := os.Stat(resp.RutaRespaldo); err != nil {
		t.Fatalf("respaldo no creado: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" {
		t.Fatalf("agente tarea inesperado tras fusion: %+v", tarea.Agente)
	}
	var agentesOrigen int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM agentes WHERE nombre = ?`, "codex1").Scan(&agentesOrigen); err != nil {
		t.Fatalf("count agente origen: %v", err)
	}
	if agentesOrigen != 0 {
		t.Fatalf("el agente origen deberia haberse eliminado")
	}
}

func TestAPIAgenteEliminarBloqueaTareasActivas(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar Codex5: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "No borrar agente con trabajo vivo",
		Modulo:    "controlplane",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex5"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex5/eliminar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status eliminar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var apiErr apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !strings.Contains(strings.ToLower(apiErr.Error), "tarea") {
		t.Fatalf("mensaje inesperado: %s", apiErr.Error)
	}
	if _, err := db.GetAgente("Codex5"); err != nil {
		t.Fatalf("el agente no deberia haberse eliminado: %v", err)
	}
}

func TestAPIAgenteTickAutoPausaPorAgotamiento(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-tick-auto-pausa",
		ResumenContinuidad: "tick final",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"agente":     "Codex1",
		"proyecto":   "orquestador",
		"cuota_pct":  3,
		"finalizado": true,
		"motivo":     "rate limit de proveedor",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/tick", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status tick inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado_cuota inesperado: %s", agente.EstadoCuota)
	}
	if agente.ReanimarAt == nil {
		t.Fatalf("reanimar_at no deberia ser nil")
	}
	if agente.MotivoPausa == "" || agente.MotivoPausa != "Auto-pausa por agotamiento: rate limit de proveedor" {
		t.Fatalf("motivo_pausa inesperado: %q", agente.MotivoPausa)
	}
}

func TestAPIAgentePrepararDevuelveBundle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if _, err := db.UpsertEntidadMemoria(&db.EntidadMemoria{
		Nombre:        "Core_API",
		Tipo:          string(db.EntidadMemoriaAPI),
		ValorJSON:     `{"version":"v2"}`,
		MetadataJSON:  `{"fuente":"manual"}`,
		VerificadoPor: "Codex1",
		ProyectoID:    &proyecto.ID,
	}); err != nil {
		t.Fatalf("upsert entidad memoria: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agente/preparar?agente=Codex1&proyecto=orquestador", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status preparar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var out agentePrepararOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode preparar: %v", err)
	}
	if out.Agente != "Codex1" {
		t.Fatalf("agente inesperado: %s", out.Agente)
	}
	if out.Proyecto.Slug != "orquestador" {
		t.Fatalf("proyecto inesperado: %s", out.Proyecto.Slug)
	}
	if out.Conector.Slug != "codex-cli" {
		t.Fatalf("conector inesperado: %s", out.Conector.Slug)
	}
	if out.Plan == nil {
		t.Fatalf("plan no deberia ser nil")
	}
	if strings.TrimSpace(out.BootstrapPrompt) == "" || out.Plan.BootstrapPrompt != out.BootstrapPrompt {
		t.Fatalf("bootstrap prompt inesperado: top=%q plan=%q", out.BootstrapPrompt, out.Plan.BootstrapPrompt)
	}
	if out.EstadoCuota == "" {
		t.Fatalf("estado_cuota no deberia venir vacio")
	}
	if len(out.Memoria) != 1 || out.Memoria[0].Nombre != "Core_API" {
		t.Fatalf("memoria inesperada: %+v", out.Memoria)
	}
}

func TestAPIAgentePrepararIntegraBootstrapRuntime(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyecto.ID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-previa"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "continuidad previa",
		ResumePayloadJSON:  `{"previo":true}`,
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	handoffPayload, err := json.Marshal(db.HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		Motivo:             "traspaso",
		ResumenContinuidad: "handoff listo",
		ExternalSessionID:  "sess-handoff",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyecto.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyecto.ID,
		Tipo:       "nudge",
		PayloadJSON: `{
			"to_agente":"Codex1",
			"from_agente":"Coordinador",
			"kind":"handoff_note",
			"texto":"revisa el checkpoint"
		}`,
	}); err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}
	if _, err := db.ProcesarRuntimeOrdersBasicasBatch(); err != nil {
		t.Fatalf("procesar runtime orders basicas: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyecto.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint reciente",
		Branch:         "feature/bootstrap",
		CWD:            filepath.Join(tmp, "orquestador", "checkpoint"),
		PayloadJSON:    `{"archivos":["a.go","b.go"]}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "test:bootstrap",
	}); err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyecto.ID,
		"Codex1",
		"wt-codex1",
		filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "wt-codex1"),
		"feature/wt-codex1",
		"HEAD",
		"activa",
		"continuidad",
	); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "wt-codex1"), 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if _, err := db.CrearPropuesta(&db.Propuesta{
		Codigo:       "OP-901",
		Titulo:       "Revisar autenticacion",
		Descripcion:  "Pendiente de votacion",
		ProyectoID:   &proyecto.ID,
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "orquesta",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	reglaID, err := db.UpsertRegla(&db.Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "regla-api-prepare-governance",
		Descripcion: "forzar reconcile de governance en prepare",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("upsert regla: %v", err)
	}
	if _, err := db.GuardarGovernanceOverride("tester", &db.GovernanceOverride{
		TipoAgente: "programador",
		ScopeTipo:  db.GovernanceScopeProyecto,
		ScopeRef:   "orquestador",
		Entidad:    db.GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     db.GovernanceActionDisable,
	}); err != nil {
		t.Fatalf("guardar override gobernanza: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agente/preparar?agente=Codex1&proyecto=orquestador", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status preparar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var out agentePrepararOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode preparar: %v", err)
	}
	if out.Bootstrap == nil {
		t.Fatalf("bootstrap no deberia ser nil")
	}
	if out.Bootstrap.Order == nil || out.Bootstrap.Order.Tipo != "handoff" {
		t.Fatalf("order bootstrap inesperada: %+v", out.Bootstrap.Order)
	}
	if len(out.Bootstrap.Mailbox) == 0 {
		t.Fatalf("mailbox bootstrap vacio: %+v", out.Bootstrap.Mailbox)
	}
	hasHandoffNote := false
	hasGovernanceRefresh := false
	for _, msg := range out.Bootstrap.Mailbox {
		if msg == nil {
			continue
		}
		switch msg.Kind {
		case "handoff_note":
			hasHandoffNote = true
		case db.MailboxKindGovernanceRefresh:
			hasGovernanceRefresh = true
		}
	}
	if !hasHandoffNote {
		t.Fatalf("mailbox bootstrap sin handoff_note: %+v", out.Bootstrap.Mailbox)
	}
	if !hasGovernanceRefresh {
		t.Fatalf("mailbox bootstrap sin governance_refresh: %+v", out.Bootstrap.Mailbox)
	}
	if out.Bootstrap.Checkpoint == nil || out.Bootstrap.Checkpoint.Branch != "feature/bootstrap" {
		t.Fatalf("checkpoint bootstrap inesperado: %+v", out.Bootstrap.Checkpoint)
	}
	if out.Plan == nil || out.Plan.Modo != "resume" {
		t.Fatalf("plan de arranque inesperado: %+v", out.Plan)
	}
	if strings.TrimSpace(out.BootstrapPrompt) == "" || out.Plan.BootstrapPrompt != out.BootstrapPrompt {
		t.Fatalf("bootstrap prompt inesperado: top=%q plan=%q", out.BootstrapPrompt, out.Plan.BootstrapPrompt)
	}
	if out.Plan.WorkingDir != filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "wt-codex1") {
		t.Fatalf("working_dir inesperado: %s", out.Plan.WorkingDir)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "handoff listo") {
		t.Fatalf("continuity prompt sin handoff: %s", out.Plan.ContinuityPrompt)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "runtime_order=handoff") || !strings.Contains(out.Plan.ContinuityPrompt, "mailbox=2") || !strings.Contains(out.Plan.ContinuityPrompt, "checkpoint#") {
		t.Fatalf("continuity prompt sin resumen bootstrap: %s", out.Plan.ContinuityPrompt)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "project_context") || !strings.Contains(out.Plan.ContinuityPrompt, "Worktree activa en feature/wt-codex1") || !strings.Contains(out.Plan.ContinuityPrompt, "1 propuesta(s) abiertas") {
		t.Fatalf("continuity prompt sin mapa operativo: %s", out.Plan.ContinuityPrompt)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "governance_catalog") || !strings.Contains(out.Plan.ContinuityPrompt, "Catálogo efectivo") {
		t.Fatalf("continuity prompt sin gobernanza efectiva: %s", out.Plan.ContinuityPrompt)
	}

	order, err := db.GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order == nil || order.Estado != "pendiente" {
		t.Fatalf("runtime order no deberia consumirse tras preparar: %+v", order)
	}

	estadoConsumido := "pendiente"
	toAgente := "Codex1"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyecto.ID,
		Estado:     &estadoConsumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailbox) != len(out.Bootstrap.Mailbox) {
		t.Fatalf("mailbox pendiente inesperado: got=%d want=%d %+v", len(mailbox), len(out.Bootstrap.Mailbox), mailbox)
	}
}

func TestAPIProyectosLecturaUsaRutaEfectivaAunquePersistaRutaHistorica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaHistorica,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recProyecto := httptest.NewRecorder()
	reqProyecto := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador", nil)
	mux.ServeHTTP(recProyecto, reqProyecto)
	if recProyecto.Code != http.StatusOK {
		t.Fatalf("status proyecto inesperado: %d body=%s", recProyecto.Code, recProyecto.Body.String())
	}
	var proyectoResp apiProyectoResponse
	if err := json.Unmarshal(recProyecto.Body.Bytes(), &proyectoResp); err != nil {
		t.Fatalf("decode proyecto: %v", err)
	}
	if proyectoResp.Proyecto == nil || proyectoResp.Proyecto.RutaAbs != rutaActual {
		t.Fatalf("proyecto con ruta inesperada: %+v", proyectoResp.Proyecto)
	}

	recListado := httptest.NewRecorder()
	reqListado := httptest.NewRequest(http.MethodGet, "/api/proyectos", nil)
	mux.ServeHTTP(recListado, reqListado)
	if recListado.Code != http.StatusOK {
		t.Fatalf("status listado inesperado: %d body=%s", recListado.Code, recListado.Body.String())
	}
	var listadoResp apiProyectosResponse
	if err := json.Unmarshal(recListado.Body.Bytes(), &listadoResp); err != nil {
		t.Fatalf("decode listado: %v", err)
	}
	if len(listadoResp.Proyectos) != 1 || listadoResp.Proyectos[0] == nil || listadoResp.Proyectos[0].RutaAbs != rutaActual {
		t.Fatalf("listado con ruta inesperada: %+v", listadoResp.Proyectos)
	}

	recOverview := httptest.NewRecorder()
	reqOverview := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/overview", nil)
	mux.ServeHTTP(recOverview, reqOverview)
	if recOverview.Code != http.StatusOK {
		t.Fatalf("status overview inesperado: %d body=%s", recOverview.Code, recOverview.Body.String())
	}
	var overviewResp apiProyectoOverviewResponse
	if err := json.Unmarshal(recOverview.Body.Bytes(), &overviewResp); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if overviewResp.Overview == nil || overviewResp.Overview.Proyecto == nil || overviewResp.Overview.Proyecto.RutaAbs != rutaActual {
		t.Fatalf("overview con ruta inesperada: %+v", overviewResp.Overview)
	}
}

func TestAPIProyectoOverviewSoportaDecisionesLegacy(t *testing.T) {
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
	if _, err := db.DB.Exec(`DROP TABLE decisiones_proyecto`); err != nil {
		t.Fatalf("drop decisiones_proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`
		CREATE TABLE decisiones_proyecto (
			id                       INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto                 TEXT    NOT NULL,
			titulo                   TEXT    NOT NULL,
			solucion_elegida         TEXT    NOT NULL DEFAULT '',
			motivo                   TEXT    NOT NULL DEFAULT '',
			alternativas_descartadas TEXT    NOT NULL DEFAULT '',
			impacto                  TEXT    NOT NULL DEFAULT 'medio'
			                              CHECK (impacto IN ('alto','medio','bajo')),
			propuesta_codigo         TEXT    NOT NULL DEFAULT '',
			tarea_id                 INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			registrado_por           TEXT    NOT NULL DEFAULT '',
			created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("create legacy decisiones_proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO decisiones_proyecto (proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas, impacto, registrado_por)
		VALUES (?,?,?,?,?,?,?)`,
		"orquestador",
		"ADR legacy",
		"Bridge OpenClaw",
		"Compatibilidad con BD recuperada",
		"Sin alternativa",
		"medio",
		"orquesta",
	); err != nil {
		t.Fatalf("insert legacy decision: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/overview", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status overview legacy inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode overview legacy: %v", err)
	}
	if resp.Overview == nil || resp.Overview.Proyecto == nil || resp.Overview.Proyecto.ID != proyectoID {
		t.Fatalf("overview legacy sin proyecto esperado: %+v", resp.Overview)
	}
	if len(resp.Overview.Decisiones) != 1 || resp.Overview.Decisiones[0] == nil || resp.Overview.Decisiones[0].Titulo != "ADR legacy" {
		t.Fatalf("overview legacy sin decisiones esperadas: %+v", resp.Overview)
	}
}

func TestAPIProyectoCockpitExponeResumenOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
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
	if err := db.ActivarAsignacion("Codex3", proyectoID, "worker"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	libreID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Libre",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "pendiente",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}
	if err := db.TomarTarea(libreID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(libreID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	reservadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reservada",
		ProyectoID:  &proyectoID,
		Modulo:      "api",
		Prioridad:   db.PrioridadMedia,
		CreadoPor:   "orquesta",
		Descripcion: "reservada",
	})
	if err != nil {
		t.Fatalf("crear tarea reservada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente='Codex3' WHERE id=?`, reservadaID); err != nil {
		t.Fatalf("forzar reservada: %v", err)
	}
	if _, err := db.CrearPropuesta(&db.Propuesta{
		Titulo:       "Abrir frente",
		Descripcion:  "resumen",
		ProyectoID:   &proyectoID,
		Tipo:         "arquitectura",
		PropuestoPor: "orquesta",
		Distribuidor: "orquesta",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGatePendiente,
	}); err != nil {
		t.Fatalf("crear review gate: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex3",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{}`,
	}); err != nil {
		t.Fatalf("encolar mailbox: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{}`,
	}); err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Actor:     "orquesta",
		ProjectID: &proyectoID,
		Source:    "control_plane",
		Reason:    "worker_degradado",
		StateDelta: map[string]any{
			"agente":         "Codex3",
			"agente_destino": "Codex4",
		},
		ArtifactsRef: []string{"runtime_checkpoint:41"},
	}); err != nil {
		t.Fatalf("registrar autonomy event: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/cockpit", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status cockpit inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoCockpitResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode cockpit: %v", err)
	}
	if resp.Cockpit == nil || resp.Cockpit.Proyecto == nil || resp.Cockpit.Proyecto.ID != proyectoID {
		t.Fatalf("cockpit sin proyecto esperado: %+v", resp.Cockpit)
	}
	if got := resp.Cockpit.TareasPorEstado["en_progreso"]; got != 1 {
		t.Fatalf("tareas en progreso inesperadas: %+v", resp.Cockpit.TareasPorEstado)
	}
	if got := resp.Cockpit.TareasPorEstado["asignada"]; got != 1 {
		t.Fatalf("tareas reservadas inesperadas: %+v", resp.Cockpit.TareasPorEstado)
	}
	if len(resp.Cockpit.AgentesActivos) != 1 || resp.Cockpit.AgentesActivos[0].Nombre != "Codex3" {
		t.Fatalf("agentes activos inesperados: %+v", resp.Cockpit.AgentesActivos)
	}
	if resp.Cockpit.PropuestasAbiertas != 1 || resp.Cockpit.ReviewGatesAbiertas != 1 || resp.Cockpit.RuntimeMailboxPendiente != 1 || resp.Cockpit.RuntimeOrdersAbiertas != 1 {
		t.Fatalf("resumen operativo inesperado: %+v", resp.Cockpit)
	}
	if resp.Cockpit.IntegrationRisk != "critico" || resp.Cockpit.IntegrationRiskScore != 10 {
		t.Fatalf("riesgo de integracion inesperado: %+v", resp.Cockpit)
	}
	if got := strings.Join(resp.Cockpit.IntegrationHighlights, " | "); got != "review_gates=1 | runtime_orders=1 | mailbox_rt=1 | propuestas_abiertas=1" {
		t.Fatalf("integration highlights inesperados: %q", got)
	}
	if resp.Cockpit.AutonomyEvents != 1 || len(resp.Cockpit.Autonomy) != 1 || resp.Cockpit.Autonomy[0].Kind != "task_reassigned" {
		t.Fatalf("autonomy en cockpit inesperada: %+v", resp.Cockpit)
	}
	if len(resp.Cockpit.AutonomyHighlights) == 0 || resp.Cockpit.AutonomyHighlights[0] != "task_reassigned=1" {
		t.Fatalf("autonomy highlights en cockpit inesperados: %+v", resp.Cockpit.AutonomyHighlights)
	}
	if len(resp.Cockpit.Autonomy[0].Artifacts) != 1 || resp.Cockpit.Autonomy[0].Artifacts[0] != "runtime_checkpoint:41" {
		t.Fatalf("artifacts autonomy en cockpit inesperados: %+v", resp.Cockpit.Autonomy)
	}
	if len(resp.Cockpit.MailboxPendiente) != 1 || resp.Cockpit.MailboxPendiente[0].Agente != "Codex3" {
		t.Fatalf("mailbox pendiente de cockpit inesperada: %+v", resp.Cockpit.MailboxPendiente)
	}
}

func TestAPIProyectoControlConsolidaCockpit(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Refactor runtime mailbox",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "en progreso",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Integración bloqueada",
		ProyectoID:  &proyectoID,
		Modulo:      "api",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "bloqueada",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='bloqueada', agente='Codex3' WHERE id=?`, tareaBloqueadaID); err != nil {
		t.Fatalf("activar tarea bloqueada: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Actor:     "orquesta",
		ProjectID: &proyectoID,
		TaskID:    &tareaID,
		Source:    "control_plane",
		Reason:    "worker_degradado",
		StateDelta: map[string]any{
			"agente_destino": "Codex3",
		},
		ArtifactsRef: []string{"runtime_checkpoint:77"},
	}); err != nil {
		t.Fatalf("registrar autonomy event: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/control?desde=24h", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status control inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoControlResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode control: %v", err)
	}
	if resp.Control == nil || resp.Control.Project == nil || resp.Control.Project.ID != proyectoID {
		t.Fatalf("control sin proyecto esperado: %+v", resp.Control)
	}
	if resp.Control.Cockpit == nil || resp.Control.Cockpit.Proyecto == nil || resp.Control.Cockpit.Proyecto.ID != proyectoID {
		t.Fatalf("control sin cockpit consolidado: %+v", resp.Control)
	}
	if got := resp.Control.TaskCounts["en_progreso"]; got != 1 {
		t.Fatalf("task_counts inesperado: %+v", resp.Control.TaskCounts)
	}
	if got := resp.Control.Cockpit.TareasPorEstado["en_progreso"]; got != 1 {
		t.Fatalf("cockpit consolidado inesperado: %+v", resp.Control.Cockpit.TareasPorEstado)
	}
	if resp.Control.Cockpit.IntegrationRisk != "alto" || resp.Control.Cockpit.IntegrationRiskScore != 5 {
		t.Fatalf("cockpit consolidado sin riesgo canónico: %+v", resp.Control.Cockpit)
	}
	if got := strings.Join(resp.Control.Cockpit.IntegrationHighlights, " | "); got != "bloqueadas=1" {
		t.Fatalf("integration highlights de cockpit consolidado inesperados: %q", got)
	}
	if len(resp.Control.Autonomy) != 1 || resp.Control.Autonomy[0].Kind != "task_reassigned" {
		t.Fatalf("control sin autonomy consolidada: %+v", resp.Control.Autonomy)
	}
	if len(resp.Control.Autonomy[0].Artifacts) != 1 || resp.Control.Autonomy[0].Artifacts[0] != "runtime_checkpoint:77" {
		t.Fatalf("control sin artifacts autonomy: %+v", resp.Control.Autonomy)
	}
}

func TestAPIWorkspaceControlExponeVistaGlobal(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
	}()

	statusService = stubStatusService{response: apiStatusResponse{
		TareasPorEstado: map[string]int{"en_progreso": 2},
		DeudaDispatch:   deudaDispatchResumen{Total: 1, Pendientes: 1},
		Autonomia: autonomiaResumen{
			Supervisando: 1,
			Count:        1,
			ByKind:       map[string]int{"task_reassigned": 1},
		},
		WorkersConectados:   2,
		WorkersTrabajando:   1,
		SupervisoresActivos: 1,
	}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "orquestador"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		ts := time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC)
		return &apiProyectoCockpit{
			Proyecto:        &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			TareasPorEstado: map[string]int{"en_progreso": 2},
			AutonomyEvents:  1,
			AutonomyByKind:  map[string]int{"task_reassigned": 1},
			AutonomyLastAt:  &ts,
			Autonomy: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   ts,
				TargetAgent: "Codex4",
			}},
		}, nil
	}
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:   since,
			Progress: projectControlProgress{
				State:          "activo",
				StateReason:    "hay trabajo en progreso",
				AttentionScore: 2,
				AttentionLabel: "bajo",
			},
			Agents: []projectControlAgentRow{{
				Name:             "Codex4",
				OperationalState: "trabajando",
				OpenTasks:        2,
			}},
			Autonomy: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   time.Date(2026, 4, 24, 14, 0, 0, 0, time.UTC),
				TargetAgent: "Codex4",
			}},
			Git: projectControlGitAggregate{
				TouchedFiles:        []string{"cmd/workspace_control.go"},
				PendingAddedLines:   3,
				PendingDeletedLines: 1,
			},
		}, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workspace/control?desde=2026-04-24T13:00:00Z", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status workspace control inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiWorkspaceControlResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode workspace control: %v", err)
	}
	if resp.Control == nil || resp.Control.ActiveProjects != 1 || len(resp.Control.Projects) != 1 {
		t.Fatalf("control global inesperado: %+v", resp.Control)
	}
	if !resp.Control.Since.Equal(time.Date(2026, 4, 24, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("since global inesperado: %+v", resp.Control)
	}
	if resp.Control.Operational.State != "activo" || resp.Control.Git.FilesChanged != 1 || len(resp.Control.Agents) != 1 {
		t.Fatalf("agregado global nuevo inesperado: %+v", resp.Control)
	}
	if len(resp.Control.Timeline) != 1 || resp.Control.Timeline[0].Project != "orquestador" || resp.Control.Timeline[0].TargetAgent != "Codex4" {
		t.Fatalf("timeline global inesperada: %+v", resp.Control.Timeline)
	}
	if len(resp.Control.ProjectControls) != 1 || resp.Control.ProjectControls[0].Project == nil || resp.Control.ProjectControls[0].Project.Slug != "orquestador" {
		t.Fatalf("project controls globales inesperados: %+v", resp.Control.ProjectControls)
	}
	if resp.Control.AutonomySurface == nil || resp.Control.AutonomySurface.Events != 1 {
		t.Fatalf("autonomy surface global inesperada: %+v", resp.Control)
	}
	if len(resp.Control.AutonomyHighlights) == 0 || resp.Control.AutonomyHighlights[0] != "task_reassigned=1" {
		t.Fatalf("autonomy highlights global inesperados: %+v", resp.Control)
	}
	if len(resp.Control.AutonomyRecent) != 1 || resp.Control.AutonomyRecent[0].Project != "orquestador" || resp.Control.AutonomyRecent[0].Kind != "task_reassigned" {
		t.Fatalf("autonomy recent global inesperada: %+v", resp.Control.AutonomyRecent)
	}
	if len(resp.Control.AutonomyProjects) != 1 || resp.Control.AutonomyProjects[0].Project != "orquestador" || resp.Control.AutonomyProjects[0].Events != 1 {
		t.Fatalf("autonomy projects global inesperados: %+v", resp.Control.AutonomyProjects)
	}
	if resp.Control.WorkersConectados != 2 || resp.Control.SupervisoresActivos != 1 {
		t.Fatalf("workers/supervisor global inesperados: %+v", resp.Control)
	}
}

func TestAPIWorkspaceControlUsa24hPorDefectoCuandoNoSePasaDesde(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
	}()

	statusService = stubStatusService{response: apiStatusResponse{}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "orquestador"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		return &apiProyectoCockpit{Proyecto: &db.Proyecto{Slug: slug, Nombre: "Orquestador"}}, nil
	}
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:   since,
		}, nil
	}

	before := time.Now().UTC()
	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workspace/control", nil)
	mux.ServeHTTP(rec, req)
	after := time.Now().UTC()
	if rec.Code != http.StatusOK {
		t.Fatalf("status workspace control inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiWorkspaceControlResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode workspace control: %v", err)
	}
	if resp.Control == nil {
		t.Fatal("control global vacio")
	}
	minWant := before.Add(-workspaceControlDefaultWindow).Add(-2 * time.Second)
	maxWant := after.Add(-workspaceControlDefaultWindow).Add(2 * time.Second)
	if resp.Control.Since.Before(minWant) || resp.Control.Since.After(maxWant) {
		t.Fatalf("since global por defecto inesperado: got=%s want_between=[%s,%s]", resp.Control.Since, minWant, maxWant)
	}
}

func TestAPIWorkspaceControlRechazaDesdeInvalido(t *testing.T) {
	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workspace/control?desde=xxx", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "valor --desde inválido") {
		t.Fatalf("body inesperado para desde invalido: %s", rec.Body.String())
	}
}

func TestAPIProyectoCockpitAlineaAgentesActivosConStatusVisible(t *testing.T) {
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
	for _, agente := range []string{"Codex3", "Codex1"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{Agente: agente, ProyectoID: &proyectoID})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", agente, err)
		}
		if agente == "Codex1" {
			reset := time.Now().UTC().Add(2 * time.Hour)
			zeroMessages := int64(0)
			if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
				SesionID:          sesion.ID,
				WindowKind:        "provider",
				ResetAt:           &reset,
				RemainingMessages: &zeroMessages,
				BudgetSource:      "provider_backoff",
				RawSnapshotJSON:   `{"source":"runtime_order_send_instruction","account_user":"Codex1"}`,
				CheckedAt:         time.Now().UTC(),
			}); err != nil {
				t.Fatalf("bloquear presupuesto Codex1: %v", err)
			}
		}
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/cockpit", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status cockpit inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoCockpitResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode cockpit: %v", err)
	}
	if len(resp.Cockpit.AgentesActivos) != 1 || resp.Cockpit.AgentesActivos[0].Nombre != "Codex3" {
		t.Fatalf("cockpit deberia alinear agentes visibles con status: %+v", resp.Cockpit.AgentesActivos)
	}
}

func TestAPIAgenteActividadExponeVistaUnica(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         t.TempDir(),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime control",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "en progreso",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	db.Audit("Codex1", "guardar_sesion", "sesion", 1, "tick")
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "worker_recovery_requested",
		Actor:     "orquesta",
		ProjectID: &proyectoID,
		TaskID:    &tareaID,
		Source:    "worker_recovery_reactivate",
		Reason:    "agente_sin_runtime_activo",
		StateDelta: map[string]any{
			"agente":         "Codex1",
			"control_action": "start",
		},
		ArtifactsRef: []string{"runtime_checkpoint:99"},
	}); err != nil {
		t.Fatalf("registrar autonomy event: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes/Codex1/actividad?desde=24h&proyecto=orquestador", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status actividad inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiAgenteActividadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode actividad: %v", err)
	}
	if resp.Activity == nil {
		t.Fatalf("actividad vacia")
	}
	if got := strings.TrimSpace(resp.Activity.Agent); got != "Codex1" {
		t.Fatalf("agente inesperado: %q", got)
	}
	if resp.Activity.Summary.OpenTasks != 1 {
		t.Fatalf("open tasks inesperadas: %+v", resp.Activity.Summary)
	}
	if resp.Activity.Summary.AuditEntries <= 0 {
		t.Fatalf("deberia incluir auditoria: %+v", resp.Activity.Summary)
	}
	if resp.Activity.Summary.AutonomyEvents != 1 || len(resp.Activity.Autonomy) != 1 || resp.Activity.Autonomy[0].Kind != "worker_recovery_requested" {
		t.Fatalf("actividad sin autonomy consolidada: %+v", resp.Activity)
	}
	if len(resp.Activity.Autonomy[0].Artifacts) != 1 || resp.Activity.Autonomy[0].Artifacts[0] != "runtime_checkpoint:99" {
		t.Fatalf("actividad sin artifacts autonomy: %+v", resp.Activity.Autonomy)
	}
}

func TestAPIRuntimeProcessMailboxReseteaThrottleDeReevaluacion(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

	if !runtimeMailboxShouldReevaluate("session_resume", 41, 33) {
		t.Fatalf("primer intento deberia permitirse")
	}
	if runtimeMailboxShouldReevaluate("session_resume", 41, 33) {
		t.Fatalf("segundo intento inmediato deberia quedar throttled")
	}

	body := bytes.NewReader([]byte(`{"to_agente":"claude1"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-mailbox", body)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessMailbox(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status process-mailbox inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	if !runtimeMailboxShouldReevaluate("session_resume", 41, 33) {
		t.Fatalf("el handler process-mailbox deberia limpiar el throttle para reevaluar de inmediato")
	}
}

func TestAPIRuntimeProcessAutonomiaReseteaThrottlesDeAutoasignacionYDegradados(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetAutonomiaIdleAutoassignGate()
	resetAutonomiaDegradedTaskGate()

	if !autonomiaIdleAutoassignShouldAttempt("Codex1", 3, 0) {
		t.Fatalf("primer intento idle deberia permitirse")
	}
	if autonomiaIdleAutoassignShouldAttempt("Codex1", 3, 0) {
		t.Fatalf("segundo intento idle inmediato deberia quedar throttled")
	}
	if !autonomiaDegradedTaskGate.AllowAt("41", autonomiaDegradedTaskCooldown, time.Now().UTC()) {
		t.Fatalf("primer intento de degradado deberia permitirse")
	}
	if autonomiaDegradedTaskGate.AllowAt("41", autonomiaDegradedTaskCooldown, time.Now().UTC()) {
		t.Fatalf("segundo intento degradado inmediato deberia quedar throttled")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-autonomia", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessAutonomia(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status process-autonomia inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	if !autonomiaIdleAutoassignShouldAttempt("Codex1", 3, 0) {
		t.Fatalf("process-autonomia deberia limpiar el throttle de autoasignacion idle")
	}
	if !autonomiaDegradedTaskGate.AllowAt("41", autonomiaDegradedTaskCooldown, time.Now().UTC()) {
		t.Fatalf("process-autonomia deberia limpiar el throttle de degradados")
	}
}

func TestAPIRuntimeProcessReanimacionesReabreFrenteVencido(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reabrir por batch de reanimaciones",
		Descripcion: "Trabajo premium recuperable",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_cuota: worker bloqueado por cuota"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='runtime_panic' WHERE nombre='Gemini1'`); err != nil {
		t.Fatalf("marcar reanimacion vencida: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-reanimations", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessReanimaciones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status process-reanimations inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea != nil && tarea.Estado == db.TareaEnProgreso {
			agente := "Gemini1"
			estado := "pendiente"
			orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
			if err != nil {
				t.Fatalf("listar orders: %v", err)
			}
			if len(orders) > 0 {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea final: %v", err)
	}
	t.Fatalf("la tarea deberia quedar reabierta y con orden pendiente: %+v", tarea)
}

func TestAPIRuntimeProcessReanimacionesDistingueCooldownSostenidoPorCuota(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Claude1")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(6 * time.Hour)
	resetSecondary := now.Add(72 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":0,"window_minutes":360,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "6h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("registrar presupuesto visible: %v", err)
	}
	if _, err := db.DB.Exec(`
		UPDATE agentes
		SET reanimar_at=CURRENT_TIMESTAMP,
		    estado_cuota='enfriamiento',
		    motivo_pausa='worker bloqueado por cuota'
		WHERE nombre='Claude1'
	`); err != nil {
		t.Fatalf("marcar cuota visible agotada: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-reanimations", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessReanimaciones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status process-reanimations inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiRuntimeProcessReanimationsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Accepted || !resp.Running {
		t.Fatalf("respuesta inesperada: %+v", resp)
	}
	if resp.Candidates != 0 || resp.Reactivated != 0 || resp.CooldownSustained != 0 || resp.Errors != 0 || resp.Count != 0 {
		t.Fatalf("contadores inesperados: %+v", resp)
	}
	agente, err := db.GetAgente("Claude1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		agente, err = db.GetAgente("Claude1")
		if err != nil {
			t.Fatalf("get agente: %v", err)
		}
		if agente != nil && agente.ReanimarAt != nil && agente.PresupuestoResetAt != nil && agente.ReanimarAt.Equal(agente.PresupuestoResetAt.UTC()) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("el cooldown deberia sostenerse hasta el reset visible: %+v", agente)
}

func TestAPIRuntimeProcessReanimacionesWaitDevuelveContadoresReales(t *testing.T) {
	prev := runtimeProcessReanimationsBatchFn
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{
			OK:                true,
			Count:             1,
			Candidates:        3,
			Reactivated:       1,
			CooldownSustained: 1,
			CapacityBlocked:   1,
		}
	}
	t.Cleanup(func() {
		runtimeProcessReanimationsBatchFn = prev
	})

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-reanimations", bytes.NewReader([]byte(`{"wait":true}`)))
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessReanimaciones(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status process-reanimations wait inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiRuntimeProcessReanimationsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Count != 1 || resp.Candidates != 3 || resp.Reactivated != 1 || resp.CooldownSustained != 1 || resp.CapacityBlocked != 1 {
		t.Fatalf("contadores inesperados: %+v", resp)
	}
	if resp.Accepted || resp.Running {
		t.Fatalf("wait no deberia responder como background: %+v", resp)
	}
}
