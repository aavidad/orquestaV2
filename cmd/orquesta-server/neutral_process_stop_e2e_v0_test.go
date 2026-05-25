package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	t66RunRef     = "run-ref-neutral-process-stop"
	t66AgentRef   = "agent-ref-neutral-process-stop"
	t66OccurredAt = "2026-05-25T16:30:00Z"
)

func TestNeutralProcessStopE2EV0(t *testing.T) {
	if os.Getenv("ORQUESTA_T66_CHILD") == "1" {
		neutralProcessStopChildV0()
		return
	}
	ctx := context.Background()
	config := neutralProcessStopConfigV0(t)
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	snapshot := neutralProcessStopLaunchV0(t, ctx, processRuntime, config.ProjectWorkDir)
	t.Cleanup(func() { neutralProcessStopCleanupV0(processRuntime, snapshot.ProcessRef) })
	neutralProcessStopSeedRunV0(t, ctx, stack.Stores.RunStore, stack.Stores.EventSink)
	neutralProcessStopRecordProcessV0(t, ctx, stack.Stores.ProcessRegistry, snapshot)

	first, err := orquestadirectorcycle.ExecuteDirectorCycleStepV0(
		ctx,
		neutralProcessStopCycleInputV0(t, stack.Stores.RunStore, stack.Stores.OutboxLedger, snapshot),
	)
	if err != nil {
		t.Fatalf("cycle step: %v", err)
	}
	if first.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		first.OutboxSavedCount != 1 {
		t.Fatalf("cycle result=%+v", first)
	}

	dispatch := neutralProcessStopDispatchWithStoresV0(t, neutralProcessStopStoresV0{
		RunStore:        stack.Stores.RunStore,
		EventSink:       stack.Stores.EventSink,
		ProcessRegistry: stack.Stores.ProcessRegistry,
		OutboxLedger:    stack.Stores.OutboxLedger,
	}, processRuntime)
	if dispatch.Status != orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchedV0 ||
		dispatch.Execution.DispatchRef == "" {
		t.Fatalf("dispatch=%+v", dispatch)
	}
	neutralProcessStopAssertStoppedV0(t, processRuntime, snapshot.ProcessRef)

	restartedStack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("restart buildStackFromEnvV0: %v", err)
	}
	replay, err := orquestaoutboxdispatch.RunOutboxDispatchOnceV0(
		orquestaoutboxdispatch.RunOutboxDispatchOnceInputV0{
			RunID:       t66RunRef,
			TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MessageType: orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
			Reader:      restartedStack.Stores.OutboxLedger,
			Claimer:     restartedStack.Stores.OutboxLedger,
			Executor:    neutralProcessStopNoopExecutorV0{},
			Acker:       restartedStack.Stores.OutboxLedger,
		},
	)
	if err != nil {
		t.Fatalf("replay dispatch: %v", err)
	}
	if replay.Status != orquestaoutboxdispatch.RunOutboxDispatchOnceNoPendingV0 {
		t.Fatalf("replay dispatch=%+v", replay)
	}
	neutralProcessStopAssertRunConfirmedV0(t, ctx, restartedStack.Stores.RunStore)
	neutralProcessStopAssertNoSensitiveStateV0(t, config.StateDir, os.Args[0])
}

func neutralProcessStopChildV0() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	<-done
}

func neutralProcessStopConfigV0(t *testing.T) orquestaserver.ConfigV0 {
	t.Helper()
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	return config
}

func neutralProcessStopLaunchV0(
	t *testing.T,
	ctx context.Context,
	runtime *orquestaruntime.ProcessRuntimeConnectorV0,
	workDir string,
) orquestaruntime.ProcessRuntimeSnapshotV0 {
	t.Helper()
	snapshot, err := runtime.LaunchV0(ctx, orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: os.Args[0],
		Args:        []string{"-test.run=TestNeutralProcessStopE2EV0"},
		Env:         []string{"ORQUESTA_T66_CHILD=1"},
		WorkingDir:  workDir,
	})
	if err != nil {
		t.Fatalf("LaunchV0: %v", err)
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeRunningV0 ||
		snapshot.ProcessRef == "" ||
		snapshot.SessionRef == "" ||
		snapshot.LaunchRef == "" {
		t.Fatalf("launch snapshot=%+v", snapshot)
	}
	return snapshot
}

func neutralProcessStopSeedRunV0(
	t *testing.T,
	ctx context.Context,
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
) {
	t.Helper()
	_ = sink
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         t66RunRef,
		ProjectRef:    "project-ref-neutral-process-stop",
		AppSpecRef:    "appspec-ref-neutral-process-stop",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-neutral-process-stop"},
		Agents:        []string{t66AgentRef},
		StartedAgents: []string{t66AgentRef},
		LastEventID:   "event-ref-neutral-process-stop-seed",
		LastSequence:  4,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		}},
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run); len(issues) > 0 {
		t.Fatalf("seed run invalid=%+v", issues)
	}
	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
}

func neutralProcessStopRecordProcessV0(
	t *testing.T,
	ctx context.Context,
	registry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) {
	t.Helper()
	err := registry.RecordAgentProcessV0(ctx, orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          t66RunRef,
		AgentRequestID: t66AgentRef,
		ProcessRef:     snapshot.ProcessRef,
		SessionRef:     snapshot.SessionRef,
		LaunchRef:      snapshot.LaunchRef,
		ReadinessRef:   "readiness-ref-neutral-process-stop",
		EvidenceRefs:   []string{"evidence-ref-neutral-process-stop-registry"},
	})
	if err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
}

func neutralProcessStopCycleInputV0(
	t *testing.T,
	store orquestacionnucleoapp.RunStorePortV0,
	ledger orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) orquestadirectorcycle.DirectorCycleStepInputV0 {
	t.Helper()
	run, err := store.LoadRunV0(context.Background(), t66RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	report := neutralProcessStopProgressReportV0(t, snapshot)
	return orquestadirectorcycle.DirectorCycleStepInputV0{
		Scheduler:    orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:     neutralProcessStoredWorkflowV0{store: store},
		OutboxLedger: ledger,
		CycleRef:     "cycle-ref-neutral-process-stop",
		TickRef:      "tick-ref-neutral-process-stop",
		RunRef:       t66RunRef,
		OccurredAt:   t66OccurredAt,
		Run:          run,
		ProgressSupervisionCandidates: []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{{
			CandidateRef: "candidate-ref-neutral-process-stop",
			SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
				CommandMeta:   neutralProcessCommandMetaV0("cmd-neutral-process-assess", "assess"),
				Report:        report,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:       "task-ref-neutral-process-stop",
				AssessmentRef: "assessment-ref-neutral-process-stop",
			},
			EvidenceRefs: []string{"evidence-ref-neutral-process-stop-candidate"},
		}},
		EvidenceRefs: []string{"evidence-ref-neutral-process-stop-cycle"},
	}
}
