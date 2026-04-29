package cmd

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/notificaciones"
)

func TestBuildSupervisorReviewSnapshotWithStatusParalelizaSeccionesIndependientes(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevSignals := supervisorReviewSignalsBuilder
		prevMerges := supervisorReviewMergesBuilder
		prevConflicts := supervisorReviewConflictsBuilder
		prevMailbox := supervisorReviewMailboxBuilder
		prevOutbox := supervisorReviewOutboxBuilder
		prevThreads := supervisorReviewThreadsSnapshotBuilder
		prevSubagents := supervisorReviewSubagentsSnapshotBuilder
		prevPipeline := supervisorReviewPipelineSnapshotBuilder
		prevCriticalRisk := supervisorReviewCriticalRiskBuilder
		prevDrift := openClawWorktreeDriftBuilder
		t.Cleanup(func() {
			supervisorReviewSignalsBuilder = prevSignals
			supervisorReviewMergesBuilder = prevMerges
			supervisorReviewConflictsBuilder = prevConflicts
			supervisorReviewMailboxBuilder = prevMailbox
			supervisorReviewOutboxBuilder = prevOutbox
			supervisorReviewThreadsSnapshotBuilder = prevThreads
			supervisorReviewSubagentsSnapshotBuilder = prevSubagents
			supervisorReviewPipelineSnapshotBuilder = prevPipeline
			supervisorReviewCriticalRiskBuilder = prevCriticalRisk
			openClawWorktreeDriftBuilder = prevDrift
		})

		delay := 80 * time.Millisecond
		supervisorReviewSignalsBuilder = func(limit int) ([]*supervisorReviewSignal, error) {
			time.Sleep(delay)
			return []*supervisorReviewSignal{{Event: &db.RuntimeEvent{ID: 1, Kind: "ready_for_review", CreatedAt: time.Now().UTC()}}}, nil
		}
		supervisorReviewMergesBuilder = func(limit int) ([]*db.GitMerge, error) {
			time.Sleep(delay)
			return []*db.GitMerge{{ID: 7, Estado: "pendiente", CreatedAt: time.Now().UTC()}}, nil
		}
		supervisorReviewConflictsBuilder = func(tasks []tareaLite) ([]supervisorModuleConflict, error) {
			time.Sleep(delay)
			return []supervisorModuleConflict{{Modulo: "api"}}, nil
		}
		supervisorReviewMailboxBuilder = func(agentes []*db.Agente) ([]apiOpenClawMailboxLite, error) {
			time.Sleep(delay)
			return []apiOpenClawMailboxLite{{Agente: "Codex3", Count: 1}}, nil
		}
		supervisorReviewOutboxBuilder = func(limit int) (notificaciones.OutboxSummary, error) {
			time.Sleep(delay)
			return notificaciones.OutboxSummary{}, nil
		}
		supervisorReviewThreadsSnapshotBuilder = func(supervisor, sessionID string, limit int) (map[string]any, error) {
			time.Sleep(delay)
			return map[string]any{"threads": []any{}}, nil
		}
		supervisorReviewSubagentsSnapshotBuilder = func(supervisor, proyectoSlug, sessionID string, limit int) (map[string]any, error) {
			time.Sleep(delay)
			return map[string]any{"subagents": []any{}}, nil
		}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			time.Sleep(delay)
			return nil, nil
		}
		supervisorReviewPipelineSnapshotBuilder = func(supervisor, proyectoSlug string, limit int, status apiStatusResponse, gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (map[string]any, error) {
			time.Sleep(10 * time.Millisecond)
			return map[string]any{"queue_kind": "safe"}, nil
		}
		supervisorReviewCriticalRiskBuilder = func(actions []supervisorRecommendedAction) (*workspaceAutonomyProjectSummary, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, nil
		}

		start := time.Now()
		snapshot, err := buildSupervisorReviewSnapshotWithStatus("OpenClaw", apiStatusResponse{
			Generado:          "2026-04-29T12:00:00Z",
			Agentes:           []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesActivos:    []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex3", Activo: true}},
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshotWithStatus: %v", err)
		}
		if elapsed > 350*time.Millisecond {
			t.Fatalf("snapshot de revisión demasiado lento; parece secuencial, elapsed=%s", elapsed)
		}
		if signals, _ := snapshot["signals"].([]*supervisorReviewSignal); len(signals) != 1 {
			t.Fatalf("signals inesperadas: %#v", snapshot["signals"])
		}
		if merges, _ := snapshot["merges"].([]*db.GitMerge); len(merges) != 1 {
			t.Fatalf("merges inesperados: %#v", snapshot["merges"])
		}
	})
}

func TestBuildSupervisorReviewBriefingReutilizaSnapshotParaOverview(t *testing.T) {
	withTempOrquestaDB(t, func() {
		resetStatusSnapshotCache()
		t.Cleanup(resetStatusSnapshotCache)

		prevSignals := supervisorReviewSignalsBuilder
		prevMerges := supervisorReviewMergesBuilder
		prevConflicts := supervisorReviewConflictsBuilder
		prevMailbox := supervisorReviewMailboxBuilder
		prevOutbox := supervisorReviewOutboxBuilder
		prevThreads := supervisorReviewThreadsSnapshotBuilder
		prevSubagents := supervisorReviewSubagentsSnapshotBuilder
		prevPipeline := supervisorReviewPipelineSnapshotBuilder
		prevCriticalRisk := supervisorReviewCriticalRiskBuilder
		prevDrift := openClawWorktreeDriftBuilder
		t.Cleanup(func() {
			supervisorReviewSignalsBuilder = prevSignals
			supervisorReviewMergesBuilder = prevMerges
			supervisorReviewConflictsBuilder = prevConflicts
			supervisorReviewMailboxBuilder = prevMailbox
			supervisorReviewOutboxBuilder = prevOutbox
			supervisorReviewThreadsSnapshotBuilder = prevThreads
			supervisorReviewSubagentsSnapshotBuilder = prevSubagents
			supervisorReviewPipelineSnapshotBuilder = prevPipeline
			supervisorReviewCriticalRiskBuilder = prevCriticalRisk
			openClawWorktreeDriftBuilder = prevDrift
		})

		storeStatusSnapshot(apiStatusResponse{
			Generado:          "2026-04-29T12:00:00Z",
			Agentes:           []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesActivos:    []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex3", Activo: true}},
		}, time.Now().UTC())

		var signalCalls int32
		var mergeCalls int32
		supervisorReviewSignalsBuilder = func(limit int) ([]*supervisorReviewSignal, error) {
			atomic.AddInt32(&signalCalls, 1)
			return []*supervisorReviewSignal{{
				Event:   &db.RuntimeEvent{ID: 1, Kind: "ready_for_review", Message: "Listo para revisar", CreatedAt: time.Now().UTC()},
				Agent:   "Codex3",
				Project: "orquestador",
			}}, nil
		}
		supervisorReviewMergesBuilder = func(limit int) ([]*db.GitMerge, error) {
			atomic.AddInt32(&mergeCalls, 1)
			return []*db.GitMerge{{ID: 4, Estado: "aprobado", ProyectoSlug: "orquestador", SourceBranch: "feat/api", TargetBranch: "main", CreatedAt: time.Now().UTC()}}, nil
		}
		supervisorReviewConflictsBuilder = func(tasks []tareaLite) ([]supervisorModuleConflict, error) {
			return nil, nil
		}
		supervisorReviewMailboxBuilder = func(agentes []*db.Agente) ([]apiOpenClawMailboxLite, error) {
			return nil, nil
		}
		supervisorReviewOutboxBuilder = func(limit int) (notificaciones.OutboxSummary, error) {
			return notificaciones.OutboxSummary{}, nil
		}
		supervisorReviewThreadsSnapshotBuilder = func(supervisor, sessionID string, limit int) (map[string]any, error) {
			return map[string]any{}, nil
		}
		supervisorReviewSubagentsSnapshotBuilder = func(supervisor, proyectoSlug, sessionID string, limit int) (map[string]any, error) {
			return map[string]any{}, nil
		}
		supervisorReviewPipelineSnapshotBuilder = func(supervisor, proyectoSlug string, limit int, status apiStatusResponse, gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (map[string]any, error) {
			return map[string]any{}, nil
		}
		supervisorReviewCriticalRiskBuilder = func(actions []supervisorRecommendedAction) (*workspaceAutonomyProjectSummary, error) {
			return nil, nil
		}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		text, err := buildSupervisorReviewBriefing("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewBriefing: %v", err)
		}
		if atomic.LoadInt32(&signalCalls) != 1 {
			t.Fatalf("signals deberían calcularse una sola vez, got=%d", atomic.LoadInt32(&signalCalls))
		}
		if atomic.LoadInt32(&mergeCalls) != 1 {
			t.Fatalf("merges deberían calcularse una sola vez, got=%d", atomic.LoadInt32(&mergeCalls))
		}
		if !strings.Contains(text, "ready_for_review") || !strings.Contains(text, "feat/api->main") {
			t.Fatalf("briefing sin datos de snapshot reutilizados: %s", text)
		}
	})
}

func TestBuildSupervisorRevisionSnapshotWithStatusOmiteSeccionesRicas(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevSignals := supervisorReviewSignalsBuilder
		prevMerges := supervisorReviewMergesBuilder
		prevConflicts := supervisorReviewConflictsBuilder
		prevMailbox := supervisorReviewMailboxBuilder
		prevOutbox := supervisorReviewOutboxBuilder
		prevThreads := supervisorReviewThreadsSnapshotBuilder
		prevSubagents := supervisorReviewSubagentsSnapshotBuilder
		prevPipeline := supervisorReviewPipelineSnapshotBuilder
		prevCriticalRisk := supervisorReviewCriticalRiskBuilder
		prevDrift := openClawWorktreeDriftBuilder
		t.Cleanup(func() {
			supervisorReviewSignalsBuilder = prevSignals
			supervisorReviewMergesBuilder = prevMerges
			supervisorReviewConflictsBuilder = prevConflicts
			supervisorReviewMailboxBuilder = prevMailbox
			supervisorReviewOutboxBuilder = prevOutbox
			supervisorReviewThreadsSnapshotBuilder = prevThreads
			supervisorReviewSubagentsSnapshotBuilder = prevSubagents
			supervisorReviewPipelineSnapshotBuilder = prevPipeline
			supervisorReviewCriticalRiskBuilder = prevCriticalRisk
			openClawWorktreeDriftBuilder = prevDrift
		})

		var threadsCalls int32
		var subagentsCalls int32
		var pipelineCalls int32
		var driftCalls int32

		supervisorReviewSignalsBuilder = func(limit int) ([]*supervisorReviewSignal, error) {
			return []*supervisorReviewSignal{{Event: &db.RuntimeEvent{ID: 1, Kind: "ready_for_review", CreatedAt: time.Now().UTC()}}}, nil
		}
		supervisorReviewMergesBuilder = func(limit int) ([]*db.GitMerge, error) {
			return []*db.GitMerge{{ID: 2, Estado: "pendiente", CreatedAt: time.Now().UTC()}}, nil
		}
		supervisorReviewConflictsBuilder = func(tasks []tareaLite) ([]supervisorModuleConflict, error) {
			return []supervisorModuleConflict{{Modulo: "api"}}, nil
		}
		supervisorReviewMailboxBuilder = func(agentes []*db.Agente) ([]apiOpenClawMailboxLite, error) {
			return []apiOpenClawMailboxLite{{Agente: "Codex3", Count: 1}}, nil
		}
		supervisorReviewOutboxBuilder = func(limit int) (notificaciones.OutboxSummary, error) {
			return notificaciones.OutboxSummary{}, nil
		}
		supervisorReviewThreadsSnapshotBuilder = func(supervisor, sessionID string, limit int) (map[string]any, error) {
			atomic.AddInt32(&threadsCalls, 1)
			return map[string]any{"threads": []any{}}, nil
		}
		supervisorReviewSubagentsSnapshotBuilder = func(supervisor, proyectoSlug, sessionID string, limit int) (map[string]any, error) {
			atomic.AddInt32(&subagentsCalls, 1)
			return map[string]any{"subagents": []any{}}, nil
		}
		supervisorReviewPipelineSnapshotBuilder = func(supervisor, proyectoSlug string, limit int, status apiStatusResponse, gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (map[string]any, error) {
			atomic.AddInt32(&pipelineCalls, 1)
			return map[string]any{"queue_kind": "safe"}, nil
		}
		supervisorReviewCriticalRiskBuilder = func(actions []supervisorRecommendedAction) (*workspaceAutonomyProjectSummary, error) {
			return nil, nil
		}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			atomic.AddInt32(&driftCalls, 1)
			return nil, nil
		}

		snapshot, err := buildSupervisorRevisionSnapshotWithStatus("OpenClaw", apiStatusResponse{
			Generado:          "2026-04-29T12:00:00Z",
			Agentes:           []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesActivos:    []*db.Agente{{Nombre: "Codex3", Activo: true}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex3", Activo: true}},
		})
		if err != nil {
			t.Fatalf("buildSupervisorRevisionSnapshotWithStatus: %v", err)
		}
		if atomic.LoadInt32(&threadsCalls) != 0 || atomic.LoadInt32(&subagentsCalls) != 0 || atomic.LoadInt32(&pipelineCalls) != 0 || atomic.LoadInt32(&driftCalls) != 0 {
			t.Fatalf("revision estrecha no deberia llamar secciones ricas: threads=%d subagents=%d pipeline=%d drift=%d", threadsCalls, subagentsCalls, pipelineCalls, driftCalls)
		}
		if _, ok := snapshot["thread_sessions"]; ok {
			t.Fatalf("snapshot estrecho no deberia incluir thread_sessions: %#v", snapshot)
		}
		if _, ok := snapshot["subagents"]; ok {
			t.Fatalf("snapshot estrecho no deberia incluir subagents: %#v", snapshot)
		}
		if _, ok := snapshot["pipeline_state"]; ok {
			t.Fatalf("snapshot estrecho no deberia incluir pipeline_state: %#v", snapshot)
		}
		if _, ok := snapshot["worktree_drift"]; ok {
			t.Fatalf("snapshot estrecho no deberia incluir worktree_drift: %#v", snapshot)
		}
		if queue, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction); len(queue) == 0 {
			t.Fatalf("safe_action_queue vacia: %#v", snapshot)
		}
	})
}

func TestBuildSupervisorRevisionReadSnapshotReutilizaSnapshotPersistido(t *testing.T) {
	prevBuilder := supervisorRevisionSnapshotBuilder
	defer func() {
		supervisorRevisionSnapshotBuilder = prevBuilder
		resetSupervisorRevisionSnapshotCache()
	}()
	resetSupervisorRevisionSnapshotCache()

	now := time.Now().UTC()
	storeSupervisorRevisionSnapshotWithTTL("OpenClaw", map[string]any{
		"supervisor":   "OpenClaw",
		"review_gates": []*db.ReviewGate{{ID: 7, Estado: db.ReviewGatePendiente}},
	}, now, time.Minute, time.Minute)

	supervisorRevisionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		time.Sleep(2 * time.Second)
		return map[string]any{"supervisor": supervisor}, nil
	}

	start := time.Now()
	snapshot, err := buildSupervisorRevisionReadSnapshot("OpenClaw")
	if err != nil {
		t.Fatalf("buildSupervisorRevisionReadSnapshot: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("lectura de snapshot de revision demasiado lenta: %s", elapsed)
	}
	gates, _ := snapshot["review_gates"].([]*db.ReviewGate)
	if len(gates) != 1 || gates[0].ID != 7 {
		t.Fatalf("snapshot reutilizado inesperado: %#v", snapshot)
	}
}

func TestBuildSupervisorRevisionReadSnapshotColdStartUsaStatusYNoBloquea(t *testing.T) {
	prevBuilder := supervisorRevisionSnapshotBuilder
	defer func() {
		supervisorRevisionSnapshotBuilder = prevBuilder
		resetSupervisorRevisionSnapshotCache()
		resetStatusSnapshotCache()
	}()
	resetSupervisorRevisionSnapshotCache()
	resetStatusSnapshotCache()

	now := time.Now().UTC()
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Generado:          now.Format(time.RFC3339),
		Agentes:           []*db.Agente{{Nombre: "Codex10", Activo: true, Habilitado: true}},
		AgentesActivos:    []*db.Agente{{Nombre: "Codex10", Activo: true, Habilitado: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex10", Activo: true, Habilitado: true}},
		TareasEnProgreso:  []tareaLite{{ID: 40, Agente: "Codex10", Estado: db.TareaEnProgreso}},
		TareasPorEstado:   map[string]int{"en_progreso": 1},
	}, now, time.Minute)

	release := make(chan struct{})
	supervisorRevisionSnapshotBuilder = func(supervisor string) (map[string]any, error) {
		<-release
		return map[string]any{"supervisor": supervisor, "review_gates": []*db.ReviewGate{{ID: 99}}}, nil
	}

	start := time.Now()
	snapshot, err := buildSupervisorRevisionReadSnapshot("OpenClaw")
	elapsed := time.Since(start)
	close(release)
	if err != nil {
		t.Fatalf("buildSupervisorRevisionReadSnapshot: %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("cold start no deberia bloquearse por revision rica, elapsed=%s", elapsed)
	}
	if snapshot["supervisor"] != "OpenClaw" {
		t.Fatalf("snapshot inesperado: %#v", snapshot)
	}
	gates, _ := snapshot["review_gates"].([]*db.ReviewGate)
	if len(gates) != 0 {
		t.Fatalf("cold start deberia devolver snapshot minima, gates=%#v", gates)
	}
}
