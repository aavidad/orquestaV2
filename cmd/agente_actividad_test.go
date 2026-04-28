package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/gitestadisticasapp"
)

func TestParseStatsSinceAceptaDuracionYRFC3339(t *testing.T) {
	now := time.Now().UTC()
	since, err := parseStatsSince("1h")
	if err != nil {
		t.Fatalf("parseStatsSince duracion: %v", err)
	}
	if delta := now.Sub(since); delta < 59*time.Minute || delta > 61*time.Minute {
		t.Fatalf("ventana inesperada: %s", delta)
	}

	want := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	got, err := parseStatsSince("2026-04-23T10:00:00Z")
	if err != nil {
		t.Fatalf("parseStatsSince RFC3339: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("timestamp inesperado: got=%s want=%s", got, want)
	}
}

func TestResolveAgentActivityCWDPrefiereSesionActivaYProyecto(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-time.Hour)
	detail := &agentesapp.Detail{
		Row: agentesapp.Row{
			Sesion: &db.Sesion{
				ID:           10,
				ProyectoSlug: "orquestador",
				CWD:          "/tmp/orquestador-activa",
				HeartbeatAt:  &now,
			},
		},
		Sesiones: []*db.Sesion{
			{
				ID:           11,
				ProyectoSlug: "api",
				CWD:          "/tmp/api",
				HeartbeatAt:  &now,
			},
			{
				ID:           12,
				ProyectoSlug: "orquestador",
				CWD:          "/tmp/orquestador-vieja",
				HeartbeatAt:  &older,
			},
		},
	}
	got := resolveAgentActivityCWD(detail, "orquestador")
	if got != "/tmp/orquestador-activa" {
		t.Fatalf("cwd inesperado: %q", got)
	}
}

func TestResolveAgentActivityCWDPrefiereWorktreeActivaCanonica(t *testing.T) {
	prev := resolveAgentActivityWorktree
	resolveAgentActivityWorktree = func(project, agent string) string {
		if project == "orquestador" && agent == "Codex1" {
			return "/tmp/orquestador-worktree-canonica"
		}
		return ""
	}
	t.Cleanup(func() {
		resolveAgentActivityWorktree = prev
	})

	detail := &agentesapp.Detail{
		Row: agentesapp.Row{
			Agente: &db.Agente{Nombre: "Codex1"},
			Sesion: &db.Sesion{
				ID:           10,
				ProyectoSlug: "orquestador",
				CWD:          "/tmp/orquestador-activa",
			},
		},
	}
	got := resolveAgentActivityCWD(detail, "orquestador")
	if got != "/tmp/orquestador-worktree-canonica" {
		t.Fatalf("cwd inesperado: %q", got)
	}
}

func TestAutonomyEventSummaryFromEventCompactaDeltaUtil(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	taskID := int64(41)
	ev, ok := autonomyEventSummaryFromEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Source:    "control_plane",
		Reason:    "worker_degradado",
		TaskID:    &taskID,
		CreatedAt: now,
		StateDelta: map[string]any{
			"agente_origen":  "Gemma1",
			"agente_destino": "Codex1",
		},
		ArtifactsRef: []string{
			"runtime_checkpoint:41",
			"runtime_checkpoint:41",
			"/tmp/orquesta/artifacts/checkpoints/runtime-checkpoint-000041.json",
			"artifact:very-long-reference-that-should-be-trimmed-because-it-is-clearly-too-large-to-show-whole-without-noise-for-control-surfaces",
			"artifact:overflow:4",
		},
	})
	if !ok {
		t.Fatal("deberia resumir autonomy event valido")
	}
	if ev.Kind != "task_reassigned" || ev.Source != "control_plane" || ev.OriginAgent != "Gemma1" || ev.TargetAgent != "Codex1" {
		t.Fatalf("summary inesperado: %+v", ev)
	}
	if len(ev.Artifacts) != 3 || ev.Artifacts[0] != "runtime_checkpoint:41" || !strings.Contains(ev.Artifacts[1], "runtime-checkpoint-000041.json") || ev.ArtifactsMore != 1 {
		t.Fatalf("artifacts compactos inesperados: %+v", ev)
	}
}

func TestSummarizeAgentActivityCuentaAutonomyEvents(t *testing.T) {
	summary := summarizeAgentActivity(nil, nil, nil, []autonomyEventSummary{
		{Kind: "task_reassigned"},
		{Kind: "task_reassigned"},
		{Kind: "worker_recovery_requested"},
	})
	if summary.AutonomyEvents != 3 {
		t.Fatalf("autonomy events inesperados: %+v", summary)
	}
	if summary.AutonomyByKind["task_reassigned"] != 2 || summary.AutonomyByKind["worker_recovery_requested"] != 1 {
		t.Fatalf("autonomy by kind inesperado: %+v", summary.AutonomyByKind)
	}
	if summary.LastAutonomyEventAt == nil {
		t.Fatalf("deberia conservar ultimo autonomy event: %+v", summary)
	}
}

func TestCompactAutonomyEventSummariesCuentaYRecortaRecientes(t *testing.T) {
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	recent, byKind, lastAt, total := compactAutonomyEventSummaries([]autonomyEventSummary{
		{Kind: "task_reassigned", CreatedAt: now.Add(-2 * time.Minute)},
		{Kind: "task_reactivated", CreatedAt: now.Add(-1 * time.Minute)},
		{Kind: "task_reassigned", CreatedAt: now.Add(-3 * time.Minute)},
	}, 2)
	if total != 3 {
		t.Fatalf("total inesperado: %d", total)
	}
	if len(recent) != 2 || recent[0].Kind != "task_reactivated" {
		t.Fatalf("recientes inesperados: %+v", recent)
	}
	if byKind["task_reassigned"] != 2 || byKind["task_reactivated"] != 1 {
		t.Fatalf("conteo por kind inesperado: %+v", byKind)
	}
	if lastAt == nil || !lastAt.Equal(now.Add(-1*time.Minute)) {
		t.Fatalf("lastAt inesperado: %v", lastAt)
	}
}

func TestBuildAgentActivityReportLocalIncluyeAutonomyCompactaPorAgente(t *testing.T) {
	prepararDBTemporalCmd(t)

	repoDir := t.TempDir()
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", projectID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		CWD:         filepath.Join(repoDir, "worktree"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	taskA := int64(41)
	taskB := int64(42)
	taskC := int64(43)
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Actor:     "orquesta",
		ProjectID: &projectID,
		TaskID:    &taskA,
		Source:    "control_plane",
		Reason:    "worker_degradado",
		StateDelta: map[string]any{
			"agente_origen":  "Codex1",
			"agente_destino": "Codex4",
		},
		ArtifactsRef: []string{"runtime_checkpoint:41"},
	}); err != nil {
		t.Fatalf("registrar autonomy event task_reassigned: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reactivated",
		Actor:     "orquesta",
		ProjectID: &projectID,
		TaskID:    &taskB,
		Source:    "control_plane",
		Reason:    "resume",
		StateDelta: map[string]any{
			"agente": "Codex1",
		},
	}); err != nil {
		t.Fatalf("registrar autonomy event task_reactivated: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "handoff_requested",
		Actor:     "orquesta",
		ProjectID: &projectID,
		TaskID:    &taskC,
		Source:    "control_plane",
		Reason:    "other_agent",
		StateDelta: map[string]any{
			"agente_origen":  "Codex4",
			"agente_destino": "Codex3",
		},
	}); err != nil {
		t.Fatalf("registrar autonomy event handoff_requested: %v", err)
	}

	report, err := buildAgentActivityReportLocal("Codex1", "orquestador", time.Now().UTC().Add(-time.Hour), 0, 0)
	if err != nil {
		t.Fatalf("buildAgentActivityReportLocal: %v", err)
	}
	if report.Summary.AutonomyEvents != 2 {
		t.Fatalf("autonomy events inesperados: %+v", report.Summary)
	}
	if len(report.Autonomy) != 2 {
		t.Fatalf("autonomy reciente inesperada: %+v", report.Autonomy)
	}
	if len(report.Autonomy) != 2 {
		t.Fatalf("autonomy reciente inesperada: %+v", report.Autonomy)
	}
	foundArtifacts := false
	for _, item := range report.Autonomy {
		if len(item.Artifacts) > 0 {
			foundArtifacts = true
			break
		}
	}
	if !foundArtifacts {
		t.Fatalf("autonomy deberia conservar artifacts compactos: %+v", report.Autonomy)
	}
	if report.Summary.AutonomyByKind["task_reassigned"] != 1 || report.Summary.AutonomyByKind["task_reactivated"] != 1 {
		t.Fatalf("autonomy_by_kind inesperado: %+v", report.Summary.AutonomyByKind)
	}
	if report.Summary.LastAutonomyEventAt == nil {
		t.Fatalf("falta ultimo autonomy event: %+v", report.Summary)
	}
}

func TestBuildAgentActivityReportLocalIncluyeRiesgoCanonicoDelProyecto(t *testing.T) {
	prepararDBTemporalCmd(t)

	repoDir := t.TempDir()
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", projectID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		CWD:         filepath.Join(repoDir, "worktree"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloqueada",
		ProyectoID:  &projectID,
		Modulo:      "api",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "bloqueada",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='bloqueada', agente='Codex1' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &projectID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGatePendiente,
	}); err != nil {
		t.Fatalf("crear review gate: %v", err)
	}
	if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
		Kind:      "task_reassigned",
		Actor:     "orquesta",
		ProjectID: &projectID,
		TaskID:    &tareaID,
		Source:    "control_plane",
		Reason:    "worker_degradado",
	}); err != nil {
		t.Fatalf("registrar autonomy event: %v", err)
	}

	report, err := buildAgentActivityReportLocal("Codex1", "", time.Now().UTC().Add(-time.Hour), 0, 0)
	if err != nil {
		t.Fatalf("buildAgentActivityReportLocal: %v", err)
	}
	if report.Project != "orquestador" {
		t.Fatalf("proyecto activo inferido inesperado: %q", report.Project)
	}
	if report.Summary.IntegrationRisk != "critico" || report.Summary.IntegrationRiskScore != 9 {
		t.Fatalf("riesgo canónico inesperado: %+v", report.Summary)
	}
	if got := strings.Join(report.Summary.IntegrationHighlights, " | "); got != "bloqueadas=1 | review_gates=1" {
		t.Fatalf("highlights de riesgo inesperados: %q", got)
	}
	if got := strings.Join(report.Summary.AutonomyHighlights, " | "); !strings.Contains(got, "task_reassigned=1") {
		t.Fatalf("highlights de autonomía del proyecto inesperados: %q", got)
	}
}

func TestAgentActivityReportJSONIncluyeAutonomy(t *testing.T) {
	report := &agentActivityReport{
		Agent: "Codex1",
		Autonomy: []autonomyEventSummary{{
			Kind:      "worker_recovery_requested",
			CreatedAt: time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
			Agent:     "Codex1",
			Artifacts: []string{"runtime_checkpoint:41"},
		}},
		Summary: agentActivitySummary{
			AutonomyEvents:        1,
			AutonomyByKind:        map[string]int{"worker_recovery_requested": 1},
			AutonomyHighlights:    []string{"worker_recovery_requested=1"},
			IntegrationRisk:       "alto",
			IntegrationRiskScore:  6,
			IntegrationHighlights: []string{"review_gates=1", "runtime_orders=1"},
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	text := string(raw)
	if !containsAll(text, `"autonomy"`, `"worker_recovery_requested"`, `"autonomy_events":1`, `"autonomy_highlights":["worker_recovery_requested=1"]`, `"integration_risk":"alto"`, `"integration_risk_score":6`, `"integration_highlights":["review_gates=1","runtime_orders=1"]`, `"artifacts":["runtime_checkpoint:41"]`) {
		t.Fatalf("json sin autonomy esperado: %s", text)
	}
}

func TestImprimirAgenteActividadMuestraRiesgoYAutonomiaCanonicos(t *testing.T) {
	report := &agentActivityReport{
		Agent:     "Codex1",
		Project:   "orquestador",
		Since:     time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC),
		Generated: time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
		Summary: agentActivitySummary{
			IntegrationRisk:       "critico",
			IntegrationRiskScore:  14,
			IntegrationHighlights: []string{"bloqueadas=1", "review_gates=1"},
			AutonomyHighlights:    []string{"task_reassigned=2", "last_at=2026-04-25T09:00:00Z"},
		},
	}
	out := capturarStdout(t, func() {
		if err := imprimirAgenteActividad(report); err != nil {
			t.Fatalf("imprimirAgenteActividad: %v", err)
		}
	})
	if !containsAll(out,
		"Riesgo:    critico · integracion_bloqueada=14 · causas bloqueadas=1 | review_gates=1",
		"Autonomía: task_reassigned=2 | last_at=2026-04-25T09:00:00Z",
	) {
		t.Fatalf("salida cli sin coherencia canónica:\n%s", out)
	}
}

func TestImprimirAgenteActividadMuestraResumenGitConHead(t *testing.T) {
	committedAt := time.Date(2026, 4, 28, 8, 30, 0, 0, time.UTC)
	report := &agentActivityReport{
		Agent:     "Codex1",
		Since:     time.Date(2026, 4, 28, 7, 0, 0, 0, time.UTC),
		Generated: time.Date(2026, 4, 28, 9, 0, 0, 0, time.UTC),
		Git: &agentGitActivityStats{
			Branch:           "main",
			RepoRoot:         "/repo/orquesta",
			PendingShortStat: "2 files changed, 10 insertions(+), 3 deletions(-)",
			HeadCommit: &gitestadisticasapp.CommitSummary{
				HashShort:   "abc1234",
				Subject:     "feat: exponer head commit",
				CommittedAt: &committedAt,
			},
		},
	}
	out := capturarStdout(t, func() {
		if err := imprimirAgenteActividad(report); err != nil {
			t.Fatalf("imprimirAgenteActividad: %v", err)
		}
	})
	if !containsAll(out,
		"Git:       rama=main",
		"Repo:      /repo/orquesta",
		"HEAD:      abc1234 feat: exponer head commit · 2026-04-28 08:30:00Z",
		"Diff:      2 files changed, 10 insertions(+), 3 deletions(-)",
	) {
		t.Fatalf("salida cli git inesperada:\n%s", out)
	}
}

func TestImprimirAgenteActividadMuestraTareasActualesCadenciaYTests(t *testing.T) {
	lastTest := time.Date(2026, 4, 28, 9, 10, 0, 0, time.UTC)
	report := &agentActivityReport{
		Agent:     "Codex4",
		Since:     time.Date(2026, 4, 28, 8, 0, 0, 0, time.UTC),
		Generated: time.Date(2026, 4, 28, 9, 30, 0, 0, time.UTC),
		Git:       &agentGitActivityStats{},
		Summary: agentActivitySummary{
			CurrentTasks:        []string{"#40 [en_progreso] runtime mailbox/session_resume modulo=cmd"},
			DominantOrder:       "#91 send_instruction [ejecutando] tarea=#40 accion=continuar_trabajo",
			CommitCadence:       "atrasada",
			CommitCadenceDetail: "diff pendiente grande (9 fichero(s), +420/-35) sin commits recientes",
			TranscriptBySignal:  map[string]int{"tests": 2},
			LastTestSignalAt:    &lastTest,
		},
	}
	out := capturarStdout(t, func() {
		if err := imprimirAgenteActividad(report); err != nil {
			t.Fatalf("imprimirAgenteActividad: %v", err)
		}
	})
	if !containsAll(out,
		"Actuales:",
		"#40 [en_progreso] runtime mailbox/session_resume modulo=cmd",
		"Orden:     #91 send_instruction [ejecutando] tarea=#40 accion=continuar_trabajo",
		"Cadencia:  atrasada · diff pendiente grande (9 fichero(s), +420/-35) sin commits recientes",
		"Tests:     señales=2 · ultimo=2026-04-28 09:10:00",
	) {
		t.Fatalf("salida cli sin visibilidad ampliada:\n%s", out)
	}
}

func TestSummarizeAgentActivityExponeLeasesYUltimoTest(t *testing.T) {
	lastTest := time.Date(2026, 4, 28, 9, 15, 0, 0, time.UTC)
	detail := &agentesapp.Detail{
		Row: agentesapp.Row{OpenTasks: 1},
		Entity: &agentesapp.AgentEntity{
			CurrentTask: &agentesapp.TaskFocus{
				TaskID: 40,
				Title:  "runtime mailbox/session_resume",
				State:  db.TareaEnProgreso,
				Module: "cmd",
			},
			DominantOrder: &agentesapp.OrderFocus{
				OrderID: 91,
				Type:    "send_instruction",
				State:   "ejecutando",
				TaskID:  40,
				Action:  "continuar_trabajo",
			},
			Leases: []agentesapp.WorkLease{{
				TaskID: 40,
				Title:  "runtime mailbox/session_resume",
				State:  db.TareaEnProgreso,
				Module: "cmd",
			}},
		},
	}
	summary := summarizeAgentActivity(detail, nil, []*db.RuntimeTranscriptEntry{
		{NormalizedText: "go test ./cmd", CreatedAt: lastTest},
	}, nil)
	if len(summary.CurrentTasks) != 1 || summary.CurrentTasks[0] != "#40 [en_progreso] runtime mailbox/session_resume modulo=cmd" {
		t.Fatalf("current tasks inesperadas: %+v", summary.CurrentTasks)
	}
	if summary.DominantOrder != "#91 send_instruction [ejecutando] tarea=#40 accion=continuar_trabajo" {
		t.Fatalf("dominant order inesperada: %q", summary.DominantOrder)
	}
	if summary.LastTestSignalAt == nil || !summary.LastTestSignalAt.Equal(lastTest.UTC()) {
		t.Fatalf("last test signal inesperado: %+v", summary.LastTestSignalAt)
	}
}

func TestSummarizeAgentCommitCadenceDetectaDiffGrandeSinCommits(t *testing.T) {
	state, detail := summarizeAgentCommitCadence(&agentGitActivityStats{
		PendingFiles:        []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go", "h.go"},
		PendingAddedLines:   401,
		PendingDeletedLines: 12,
		RecentCommitCount:   0,
	})
	if state != "atrasada" {
		t.Fatalf("state=%q", state)
	}
	if !strings.Contains(detail, "diff pendiente grande") {
		t.Fatalf("detail inesperado: %s", detail)
	}
}

func TestClassifyAgentActivityTranscriptSignalReduceSinClasificar(t *testing.T) {
	cases := []struct {
		name string
		item *db.RuntimeTranscriptEntry
		want string
	}{
		{
			name: "respeta clasificacion persistida",
			item: &db.RuntimeTranscriptEntry{Classification: "progress_update", NormalizedText: "texto cualquiera"},
			want: "progress_update",
		},
		{
			name: "detecta tests",
			item: &db.RuntimeTranscriptEntry{NormalizedText: "go test ./cmd ok"},
			want: "tests",
		},
		{
			name: "detecta git",
			item: &db.RuntimeTranscriptEntry{NormalizedText: "git commit -m cambio"},
			want: "git_activity",
		},
		{
			name: "detecta cambios de codigo",
			item: &db.RuntimeTranscriptEntry{NormalizedText: "*** Update File: cmd/agente_actividad.go"},
			want: "code_change",
		},
		{
			name: "usa stream como fallback",
			item: &db.RuntimeTranscriptEntry{Stream: "stdout", Text: "mensaje neutro"},
			want: "stdout",
		},
		{
			name: "ultima opcion sin clasificar",
			item: &db.RuntimeTranscriptEntry{},
			want: "sin_clasificar",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyAgentActivityTranscriptSignal(tc.item); got != tc.want {
				t.Fatalf("signal=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestSummarizeAgentActivityUsaSignalDerivada(t *testing.T) {
	summary := summarizeAgentActivity(nil, nil, []*db.RuntimeTranscriptEntry{
		{NormalizedText: "go test ./cmd"},
		{NormalizedText: "git push origin candidato_refactor"},
		{Stream: "stdout", Text: "mensaje neutro"},
	}, nil)

	if summary.TranscriptBySignal["tests"] != 1 {
		t.Fatalf("tests=%d summary=%+v", summary.TranscriptBySignal["tests"], summary.TranscriptBySignal)
	}
	if summary.TranscriptBySignal["git_activity"] != 1 {
		t.Fatalf("git_activity=%d summary=%+v", summary.TranscriptBySignal["git_activity"], summary.TranscriptBySignal)
	}
	if summary.TranscriptBySignal["stdout"] != 1 {
		t.Fatalf("stdout=%d summary=%+v", summary.TranscriptBySignal["stdout"], summary.TranscriptBySignal)
	}
	if summary.TranscriptBySignal["sin_clasificar"] != 0 {
		t.Fatalf("sin_clasificar no esperado: %+v", summary.TranscriptBySignal)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
