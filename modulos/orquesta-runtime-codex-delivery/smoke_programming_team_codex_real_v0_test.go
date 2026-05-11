package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestProgrammingTeamCodexRealOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROGRAMMING_TEAM_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_PROGRAMMING_TEAM_SMOKE=1 para lanzar equipo de programacion Codex real")
	}
	cfg := codexRealSmokeConfigForTestV0(t)
	if cfg.Timeout < 420*time.Second {
		cfg.Timeout = 420 * time.Second
	}
	if err := programmingTeamSeedProjectV0(cfg.ProjectWorkDir); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	tasks := programmingTeamTasksV0()
	runRef := "run-ref-programming-team-real-001"
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	worktreeStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	processRegistry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	dispatchers := []orquestacionnucleoapp.OutboxDispatcherBindingV0{
		codexDeliveryCapacityDispatcherForTestV0(store, sink, ledger),
	}
	batchDispatchers := programmingTeamBatchDispatchersV0(
		store,
		sink,
		ledger,
		receiptStore,
		processRegistry,
		processRuntime,
		cfg,
		tasks,
		worktreeStore,
	)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	diagnostics := func(failure error) string {
		return programmingTeamSmokeStateDiagnosticsV0(
			ctx,
			failure,
			store,
			taskStore,
			receiptStore,
			sink,
			ledger,
			processRegistry,
			processRuntime,
			worktreeStore,
			runRef,
			tasks,
		)
	}
	result, err := orquestaappdirectorservice.StartAppDirectorV0(
		ctx,
		programmingTeamDirectorStartRequestV0(runRef, cfg),
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:                 store,
			EventSink:                sink,
			OutboxLedger:             ledger,
			Dispatchers:              dispatchers,
			BatchDispatchers:         batchDispatchers,
			DeliverySource:           programmingTeamDeliverySourceV0(receiptStore, worktreeStore),
			ProgressSource:           programmingTeamProgressSourceV0(receiptStore, processRegistry, processRuntime),
			DirectorDecisionSource:   programmingTeamDecisionSourceV0(receiptStore),
			DirectorTaskStore:        taskStore,
			ExternalWaiter:           timedExternalProgressWaiterV0{Interval: 5 * time.Second},
			ReviewGateSource:         nil,
			ReviewReworkReplanSource: nil,
		},
	)
	if err != nil {
		t.Fatalf(
			"programming team director loop: %v\n%s\n%s",
			err,
			diagnostics(err),
			codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir),
		)
	}
	defer mcpFormDirectorTeamSmokeStopProcessesV0(
		t,
		processRuntime,
		processRegistry,
		result.Run.RunID,
		result.StartedAgents,
	)
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("loop status=%s result=%+v\n%s", result.LoopStatus, result, diagnostics(nil))
	}
	for _, agentRef := range programmingTeamProgrammingAgentsV0(tasks) {
		if !codexDeliveryLoopContainsRefV0(result.Run.StartedAgents, agentRef) {
			t.Fatalf("started_agents=%v missing=%s\n%s", result.Run.StartedAgents, agentRef, diagnostics(nil))
		}
	}
	descriptors := mcpFormDirectorTeamSmokeDescriptorsV0(
		t,
		receiptStore,
		result.Run.RunID,
		programmingTeamProgrammingAgentsV0(tasks),
	)
	for _, descriptor := range descriptors {
		ackRef := descriptor.Spec.AgentPacket.DeliveryRefs.AckRef
		if !codexDeliveryLoopContainsRefV0(result.Run.Deliveries, ackRef) {
			t.Fatalf("deliveries=%v missing=%s\n%s", result.Run.Deliveries, ackRef, diagnostics(nil))
		}
	}
	if codexDeliveryLoopEventCountV0(
		sink.EventsV0(),
		orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0,
	) == 0 {
		t.Fatalf("events sin PhaseArtifactRegistered: %+v\n%s", sink.EventsV0(), diagnostics(nil))
	}
	if codexDeliveryLoopEventCountV0(
		sink.EventsV0(),
		orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0,
	) < len(descriptors) {
		t.Fatalf("events sin DeliveryRegistered: %+v\n%s", sink.EventsV0(), diagnostics(nil))
	}
	programmingTeamVerifyProjectV0(t, cfg.ProjectWorkDir)
}

func programmingTeamSeedProjectV0(projectDir string) error {
	if err := os.MkdirAll(filepath.Join(projectDir, "internal", "api"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "web"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "server"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module agenda-smoke\n\ngo 1.22\n"), 0o600)
}

func programmingTeamRunLoopV0(
	t *testing.T,
	ctx context.Context,
	service orquestacionnucleoapp.ServiceV0,
	runRef string,
	dispatchers []orquestacionnucleoapp.OutboxDispatcherBindingV0,
	batchDispatchers []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	t.Helper()
	result, err := service.RunProgressiveLoopV0(ctx, orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-10T00:00:00Z",
		MaxBursts:            8,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 6,
		CorrelationID:        "corr-programming-team-001",
		EvidenceRefs:         []string{"evidence-ref-programming-team-loop"},
		Dispatchers:          dispatchers,
		BatchDispatchers:     batchDispatchers,
	})
	if err != nil {
		t.Fatalf("programming team loop: %v", err)
	}
	return result
}

func programmingTeamRunManagedLoopV0(
	t *testing.T,
	ctx context.Context,
	service orquestacionnucleoapp.ServiceV0,
	runRef string,
	cfg codexRealSmokeConfigV0,
	dispatchers []orquestacionnucleoapp.OutboxDispatcherBindingV0,
	batchDispatchers []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0,
) orquestacionnucleoapp.ManagedProgressiveLoopResultV0 {
	t.Helper()
	interval := 5 * time.Second
	maxWaits := int(cfg.Timeout / interval)
	if maxWaits < 1 {
		maxWaits = 1
	}
	result, err := service.RunManagedProgressiveLoopV0(ctx, orquestacionnucleoapp.ManagedProgressiveLoopRequestV0{
		Loop: orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef:               runRef,
			OccurredAt:           "2026-05-10T00:00:00Z",
			MaxBursts:            8,
			MaxStepsPerBurst:     6,
			MaxDispatchesPerWait: 6,
			CorrelationID:        "corr-programming-team-001",
			EvidenceRefs:         []string{"evidence-ref-programming-team-loop"},
			Dispatchers:          dispatchers,
			BatchDispatchers:     batchDispatchers,
		},
		ExternalWaiter:   timedExternalProgressWaiterV0{Interval: interval},
		MaxExternalWaits: maxWaits,
	})
	if err != nil {
		t.Fatalf("programming team managed loop: %v", err)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("managed loop status=%s result=%+v", result.Status, result)
	}
	return result
}

type timedExternalProgressWaiterV0 struct {
	Interval time.Duration
}

func (waiter timedExternalProgressWaiterV0) WaitExternalProgressV0(
	ctx context.Context,
	_ orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	interval := waiter.Interval
	if interval <= 0 {
		interval = time.Second
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return orquestacionnucleoapp.ExternalProgressWaitResultV0{}, ctx.Err()
	case <-timer.C:
		return orquestacionnucleoapp.ExternalProgressWaitResultV0{
			Continue:     true,
			EvidenceRefs: []string{"evidence-ref-external-wait-001"},
		}, nil
	}
}

func programmingTeamStartedAgentsV0(tasks []programmingTeamTaskV0) []string {
	agents := make([]string, 0, len(tasks))
	for _, task := range tasks {
		agents = append(agents, task.AgentRef)
	}
	return agents
}
