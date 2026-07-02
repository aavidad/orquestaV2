package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestStackShutdownCheckpointV0SolicitaAckSiHayAgenteEnVuelo(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if result.CheckpointRecorded || len(result.PendingAgentRefs) != 1 ||
		result.PendingAgentRefs[0] != fixture.agentRef {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(filepath.Join(fixture.runtimeDir, orquestaruntimecodex.CodexShutdownRequestFileNameV0)); err != nil {
		t.Fatalf("shutdown request no escrito: %v", err)
	}
}

func TestStackShutdownCheckpointV0RegistraCuandoTodosLosAgentesResponden(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	request := stackShutdownRequestForAgentV0(fixture.command, fixture.descriptor)
	writeStackShutdownAckForTestV0(t, fixture.runtimeDir, request)

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if !result.CheckpointRecorded ||
		result.CheckpointRef != "checkpoint-ref-shutdown-"+safeStackShutdownRefPartV0(fixture.runRef) ||
		len(result.PendingAgentRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownCheckpointV0NoConsumeAckDeIntentoAnterior(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	firstRequest := stackShutdownRequestForAgentV0(fixture.command, fixture.descriptor)
	writeStackShutdownAckForTestV0(t, fixture.runtimeDir, firstRequest)
	fixture.command.CorrelationID = "corr-shutdown-stack-002"
	fixture.command.Reason = "shutdown controlado por segunda mutacion"

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if result.CheckpointRecorded || len(result.PendingAgentRefs) != 1 ||
		result.PendingAgentRefs[0] != fixture.agentRef {
		t.Fatalf("result=%+v", result)
	}
	if !hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "shutdown_checkpoint_attempt_mismatch") {
		t.Fatalf("evidence sin mismatch de intento: %+v", result.EvidenceRefs)
	}
}

func TestStackShutdownCheckpointV0RegistraSiAgenteEnVueloNoTieneProcesoVivo(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	fixture.preparer.Config.Codex.SnapshotSource = stackShutdownSnapshotSourceForTestV0{}

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if !result.CheckpointRecorded || len(result.PendingAgentRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(filepath.Join(fixture.runtimeDir, orquestaruntimecodex.CodexShutdownRequestFileNameV0)); !os.IsNotExist(err) {
		t.Fatalf("shutdown request no debe escribirse para agente stale: %v", err)
	}
	if !hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "shutdown-agent-stale-no-process") {
		t.Fatalf("evidence sin stale no-process: %+v", result.EvidenceRefs)
	}
}

func TestStackShutdownCheckpointV0MantieneEsperaSiProcesoSigueVivo(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	processRef := "process-ref-shutdown-stack-001"
	sessionRef := "session-ref-shutdown-stack-001"
	launchRef := "launch-ref-shutdown-stack-001"
	recordStackShutdownProcessForTestV0(t, fixture, processRef, sessionRef, launchRef)
	fixture.preparer.Config.Codex.SnapshotSource = stackShutdownSnapshotSourceForTestV0{
		snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{
			processRef: {
				SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
				ProcessRef:    processRef,
				SessionRef:    sessionRef,
				LaunchRef:     launchRef,
				Status:        orquestaruntime.ProcessRuntimeRunningV0,
			},
		},
	}

	result, err := fixture.preparer.PrepareAgentShutdownV0(context.Background(), fixture.command)
	if err != nil {
		t.Fatalf("PrepareAgentShutdownV0: %v", err)
	}
	if result.CheckpointRecorded || len(result.PendingAgentRefs) != 1 ||
		result.PendingAgentRefs[0] != fixture.agentRef {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownStatsV0ExponeLivenessDeAgentesObsoletos(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	fixture.preparer.Config.Codex.SnapshotSource = stackShutdownSnapshotSourceForTestV0{}

	stats, err := stackShutdownStatsReaderV0{
		Config: fixture.preparer.Config,
	}.ReadRunShutdownStatsV0(context.Background(), orquestaservershutdown.RunShutdownStatsRequestV0{
		RunRef: fixture.runRef,
	})
	if err != nil {
		t.Fatalf("ReadRunShutdownStatsV0: %v", err)
	}
	if !stats.ProcessLivenessObserved ||
		stats.AgentsInFlight != 1 ||
		stats.AgentsRunningLive != 0 ||
		stats.AgentsRunningStale != 1 ||
		stats.AgentsLost != 0 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestStackShutdownActiveWorkReaderV0ListaGoalFirstRunningSinBloquearComoBackend(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	fixture.preparer.Config.Stores.AppGoalStateStore = goalStates
	runRef := "run-ref-shutdown-goal-active-001"
	goalRef := "goal-ref-shutdown-active-001"
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:   goalRef,
			RunRef:    runRef,
			Objective: "probar bloqueo de shutdown con goal activo",
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/shutdown_active_goal.md",
				Purpose: "test",
			}},
			EvidenceRefs: []string{"goal-state-ref-shutdown-active"},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "external-goal-ref-shutdown-active-001",
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := stackShutdownActiveWorkReaderV0{
		Config: fixture.preparer.Config,
	}.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{
		MaxItems: 10,
	})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_first" ||
		result.ActiveWorks[0].RunRef != runRef ||
		result.ActiveWorks[0].WorkRef != goalRef ||
		result.ActiveWorks[0].ExternalWorkRef != "external-goal-ref-shutdown-active-001" ||
		result.ActiveWorks[0].Status != orquestagoal.GoalStatusRunningV0 ||
		!hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "goal-state-ref-shutdown-active") ||
		hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "evidence-ref-shutdown-goal-backend-running-state") ||
		hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "evidence-ref-shutdown-goal-backend-active-timeout") {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownRunControlWriterV0ForcedStopMarcaGoalTerminalReplanificable(t *testing.T) {
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	runRef := "run-ref-shutdown-forced-terminal-001"
	goalRef := "goal-ref-shutdown-forced-terminal-001"
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:   goalRef,
			RunRef:    runRef,
			Objective: "probar reconciliacion terminal tras forced stop",
			WriteSet:  []orquestagoal.GoalWriteScopeV0{{Path: "docs/shutdown_forced_terminal.md"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "external-goal-ref-shutdown-forced-terminal-001",
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"goal-state-ref-shutdown-forced-terminal"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	writer := stackShutdownRunControlWriterV0{
		Inner:          fakeStackShutdownRunControlWriterV0{},
		GoalStateStore: goalStates,
	}

	controlState, err := writer.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-operator-forced-stop"},
	})
	if err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	if controlState.Status != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("controlState=%+v", controlState)
	}
	reconciled, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if reconciled.Status != orquestagoal.GoalStatusBlockedV0 ||
		reconciled.LastResult == nil ||
		reconciled.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		len(reconciled.LastResult.Issues) != 1 ||
		reconciled.LastResult.Issues[0].Code != "operator_forced_stop_no_artifacts" ||
		reconciled.LastClosure == nil ||
		!reconciled.LastClosure.NeedsRework ||
		!hasStackShutdownEvidenceForTestV0(reconciled.EvidenceRefs, "evidence-ref-server-shutdown-goal-forced-terminal-reconciled") {
		t.Fatalf("reconciled=%+v", reconciled)
	}
}

func TestStackShutdownActiveWorkReaderV0BloqueaRestosBackendAunqueStateStoreNoRunningV0(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	fixture.preparer.Config.AppGoalObserver = stackShutdownActiveGoalObserverForTestV0{
		result: orquestaservershutdown.ActiveShutdownWorkResultV0{
			ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
				Kind:            "goal_backend",
				WorkRef:         "orquesta-goal-residue-1234567890",
				ExternalWorkRef: "codex-goal-app-server-tmux",
				Status:          orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
				EvidenceRefs:    []string{"evidence-ref-codex-app-server-tmux-residue"},
			}},
			EvidenceRefs: []string{"evidence-ref-codex-app-server-tmux-residue"},
		},
	}
	fixture.preparer.Config.Stores.AppGoalStateStore = newGoalFirstQueueStateStoreForTestV0()

	result, err := stackShutdownActiveWorkReaderV0{
		Config: fixture.preparer.Config,
	}.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{
		MaxItems: 10,
	})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_backend" ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-residue") {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownActiveWorkReaderV0ListaGoalFirstCompletoPendienteObservacion(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	fixture.preparer.Config.Stores.AppGoalStateStore = goalStates
	runRef := "run-ref-shutdown-goal-complete-pending-001"
	goalRef := "goal-ref-shutdown-complete-pending-001"
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:   goalRef,
			RunRef:    runRef,
			Objective: "probar goal complete pendiente de cierre",
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/shutdown_complete_pending_goal.md",
				Purpose: "test",
			}},
			EvidenceRefs: []string{"goal-state-ref-shutdown-complete-pending"},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "external-goal-ref-shutdown-complete-pending-001",
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.Status = orquestagoal.GoalStatusCompleteV0
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := stackShutdownActiveWorkReaderV0{
		Config: fixture.preparer.Config,
	}.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{
		MaxItems: 10,
	})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_first" ||
		result.ActiveWorks[0].RunRef != runRef ||
		result.ActiveWorks[0].WorkRef != goalRef ||
		result.ActiveWorks[0].ExternalWorkRef != "external-goal-ref-shutdown-complete-pending-001" ||
		result.ActiveWorks[0].Status != orquestagoal.GoalStatusCompleteV0 ||
		!hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "goal-state-ref-shutdown-complete-pending") {
		t.Fatalf("result=%+v", result)
	}
}

func TestStackShutdownActiveWorkReaderV0BloqueaBackendActivoTrasGoalTimeout(t *testing.T) {
	fixture := newStackShutdownCheckpointFixtureV0(t)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	fixture.preparer.Config.Stores.AppGoalStateStore = goalStates
	runRef := "run-ref-shutdown-goal-timeout-backend-active-001"
	goalRef := "goal-ref-shutdown-timeout-backend-active-001"
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:   goalRef,
			RunRef:    runRef,
			Objective: "probar bloqueo de shutdown con backend goal activo tras timeout local",
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path:    "docs/shutdown_goal_timeout.md",
				Purpose: "test",
			}},
			EvidenceRefs: []string{"goal-state-ref-shutdown-timeout-backend-active"},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "external-goal-ref-shutdown-timeout-backend-active-001",
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusBlockedV0,
		GoalRef:       goalRef,
		Summary:       "codex_app_server_goal_active_timeout",
		EvidenceRefs:  []string{"evidence-ref-codex-app-server-goal-active-timeout"},
	}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	result, err := stackShutdownActiveWorkReaderV0{
		Config: fixture.preparer.Config,
	}.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{
		MaxItems: 10,
	})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_backend" ||
		result.ActiveWorks[0].RunRef != runRef ||
		result.ActiveWorks[0].WorkRef != goalRef ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!hasStackShutdownEvidenceForTestV0(result.EvidenceRefs, "evidence-ref-shutdown-goal-backend-active-timeout") {
		t.Fatalf("result=%+v", result)
	}
}

type stackShutdownCheckpointFixtureDataV0 struct {
	runRef     string
	agentRef   string
	runtimeDir string
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
	preparer   stackShutdownCheckpointPreparerV0
	command    orquestaservershutdown.PrepareAgentShutdownCommandV0
}

func newStackShutdownCheckpointFixtureV0(t *testing.T) stackShutdownCheckpointFixtureDataV0 {
	t.Helper()
	runRef := "run-ref-shutdown-stack-001"
	agentRef := "agent-ref-shutdown-stack-001"
	runtimeDir := t.TempDir()
	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  "descriptor-ref-shutdown-stack-001",
		RunID:          runRef,
		AgentRef:       agentRef,
		AckPath:        filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		ProjectWorkDir: t.TempDir(),
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
			RequestID:     agentRef,
			CorrelationID: "corr-shutdown-stack-001",
			RuntimeKind:   "cli",
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				RequestID:     agentRef,
				CorrelationID: "corr-shutdown-stack-001",
				CapacityLevel: "medium",
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: "ack-ref-shutdown-stack-001",
				},
			},
		},
	}
	receiptStore := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor)
	return stackShutdownCheckpointFixtureDataV0{
		runRef:     runRef,
		agentRef:   agentRef,
		runtimeDir: runtimeDir,
		descriptor: descriptor,
		preparer: stackShutdownCheckpointPreparerV0{Config: ConfigV0{
			Stores: StoresV0{
				RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(stackShutdownRunForTestV0(runRef, agentRef)),
				ReceiptStore:    receiptStore,
				ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
				ProcessRegistry: orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
			},
		}},
		command: orquestaservershutdown.PrepareAgentShutdownCommandV0{
			RunRef:        runRef,
			CorrelationID: "corr-shutdown-stack-001",
			RequestedBy:   "test",
			Reason:        "shutdown controlado",
			EvidenceRefs:  []string{"evidence-ref-shutdown-stack"},
		},
	}
}

func stackShutdownRunForTestV0(
	runRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-shutdown-stack-001",
		AppSpecRef:    "app-spec-ref-shutdown-stack-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-shutdown-stack-001"},
		Agents:        []string{agentRef},
		StartedAgents: []string{agentRef},
	}
}

func recordStackShutdownProcessForTestV0(
	t *testing.T,
	fixture stackShutdownCheckpointFixtureDataV0,
	processRef string,
	sessionRef string,
	launchRef string,
) {
	t.Helper()
	if err := fixture.preparer.Config.Stores.ProcessRegistry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          fixture.runRef,
		AgentRequestID: fixture.agentRef,
		ProcessRef:     processRef,
		SessionRef:     sessionRef,
		LaunchRef:      launchRef,
		ReadinessRef:   "readiness-ref-shutdown-stack-001",
		EvidenceRefs:   []string{"evidence-ref-shutdown-process-record"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
}

type stackShutdownSnapshotSourceForTestV0 struct {
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
}

type stackShutdownActiveGoalObserverForTestV0 struct {
	result orquestaservershutdown.ActiveShutdownWorkResultV0
}

type fakeStackShutdownRunControlWriterV0 struct{}

func (fakeStackShutdownRunControlWriterV0) PauseRunV0(
	context.Context,
	orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fakeStackShutdownRunControlWriterV0) ResumeRunV0(
	context.Context,
	orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{}, nil
}

func (fakeStackShutdownRunControlWriterV0) StopRunV0(
	_ context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{
		RunRef: command.RunRef,
		Status: orquestaruncontrol.RunControlStatusStoppedV0,
		Forced: command.Forced,
	}, nil
}

func (fakeStackShutdownRunControlWriterV0) CancelRunV0(
	_ context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return orquestaruncontrol.RunControlStateV0{
		RunRef: command.RunRef,
		Status: orquestaruncontrol.RunControlStatusCanceledV0,
		Forced: command.Forced,
	}, nil
}

func (observer stackShutdownActiveGoalObserverForTestV0) ObserveGoalWorkV0(
	context.Context,
	orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return orquestagoal.GoalWorkResultV0{}, nil
}

func (observer stackShutdownActiveGoalObserverForTestV0) ReadActiveShutdownWorkV0(
	context.Context,
	orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	return observer.result, nil
}

func (source stackShutdownSnapshotSourceForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	if snapshot, ok := source.snapshots[processRef]; ok {
		return snapshot, nil
	}
	return orquestaruntime.ProcessRuntimeSnapshotV0{}, orquestaruntime.ProcessRuntimeErrorV0{
		Code:       orquestaruntime.ProcessRuntimeNoEncontradoV0,
		MessageKey: "orquesta.runtime.process.process_runtime_no_encontrado",
		Field:      "process_ref",
	}
}

func writeStackShutdownAckForTestV0(
	t *testing.T,
	runtimeDir string,
	request orquestaruntimecodex.CodexShutdownRequestV0,
) {
	t.Helper()
	ack := orquestaruntimecodex.CodexShutdownCheckpointAckV0{
		SchemaVersion:      orquestaruntimecodex.CodexShutdownCheckpointAckSchemaVersionV0,
		RunRef:             request.RunRef,
		AgentRef:           request.AgentRef,
		ShutdownAttemptRef: request.ShutdownAttemptRef,
		CheckpointRef:      request.CheckpointRef,
		Status:             orquestaruntimecodex.CodexShutdownCheckpointStatusReadyV0,
		EvidenceRefs:       []string{"evidence-ref-agent-checkpoint"},
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(runtimeDir, orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}

func hasStackShutdownEvidenceForTestV0(refs []string, want string) bool {
	for _, ref := range refs {
		if strings.Contains(ref, want) {
			return true
		}
	}
	return false
}
