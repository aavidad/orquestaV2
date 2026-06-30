package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestStartupCleanupCandidatesV0IncluyeColaReadyConControlTerminal(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	_, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: "run-terminal-queue-ready",
		AppRef: "app-001",
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	_, err = store.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-terminal-queue-ready",
		TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:  "test",
	})
	if err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{
				QueueRef: "queue-main",
			},
		},
	}

	cleanup, err := check.startupCleanupCandidatesV0(ctx)
	if err != nil {
		t.Fatalf("startupCleanupCandidatesV0: %v", err)
	}
	if len(cleanup) != 1 || cleanup[0].Active || !cleanup[0].QueueDirty {
		t.Fatalf("cleanup=%+v", cleanup)
	}
}

func TestStartupDiagnoseV0AsumeRunsReadyActivosSinPurgaV0(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	_, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: "run-ready-active-adoptable",
		AppRef: "app-001",
		Status: orquestarunqueue.RunStatusReadyV0,
	})
	if err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready ||
		result.Status != orquestaserver.StartupCheckStatusReadyV0 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-orquesta-startup-active-runs-adopted") {
		t.Fatalf("result=%+v", result)
	}
}

func TestStartupDiagnoseV0SuprimeAutoprogrammingStaleEnSesionOPESV0(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	runRef := "request-ref-autoprogramming-backlog-scanner-15eeecb9"
	if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:       runRef,
		AppRef:       "app-ref-autoprogramming",
		Status:       orquestarunqueue.RunStatusReadyV0,
		EvidenceRefs: []string{"evidence-prev"},
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	opesDir := filepath.Join(t.TempDir(), "OPES", "opes-salidas", "curso-demo")
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		ServerConfig: orquestaserver.ConfigV0{
			ProjectWorkDir:              opesDir,
			IdleSelfImprovementDisabled: true,
			EffectiveConfig: orquestaserver.ServerEffectiveConfigV0{Settings: []orquestaserver.ServerConfigSettingV0{{
				Key:    envOPESProjectWorkDirV0,
				Value:  "opes-project-workdir-configured",
				Source: "explicit",
				Scope:  "domain_work",
			}}},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready ||
		!strings.Contains(result.Message, startupIdleSelfImprovementDomainSessionSuppressedReasonV0) ||
		!containsStringForTestV0(result.EvidenceRefs, startupIdleSelfImprovementDomainSessionEvidenceV0) {
		t.Fatalf("result=%+v", result)
	}
	visible, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 visible: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("cola ejecutable debe quedar limpia: %+v", visible)
	}
	all, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             "queue-main",
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 all: %v", err)
	}
	if len(all) != 1 ||
		all[0].Status != orquestarunqueue.RunStatusStoppedV0 ||
		all[0].RescueReason != startupIdleSelfImprovementDomainSessionSuppressedReasonV0 ||
		!containsStringForTestV0(all[0].EvidenceRefs, startupIdleSelfImprovementDomainSessionEvidenceV0) ||
		!containsStringForTestV0(all[0].EvidenceRefs, "evidence-ref-orquesta-startup-queue-stopped") {
		t.Fatalf("terminal candidate=%+v", all)
	}
}

func TestStartupDiagnoseV0AdoptaProcesosVivosRegistradosV0(t *testing.T) {
	ctx := context.Background()
	queueControl := orquestarunmemory.NewRunMemoryStoreV0()
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	runRef := "run-ready-active-adoptable"
	agentRef := "agent-ref-startup-adoptable"
	if _, err := queueControl.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: runRef,
		AppRef: "app-001",
		Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	if err := stateStore.RecordAgentProcessV0(ctx, orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          runRef,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-startup-adoptable",
		SessionRef:     "session-ref-startup-adoptable",
		LaunchRef:      "launch-ref-startup-adoptable",
		PID:            12345,
		ReadinessRef:   "readiness-ref-startup-adoptable",
		EvidenceRefs:   []string{"evidence-ref-startup-adoptable"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	adopter := &fakeStartupProcessAdopterV0{}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:        queueControl,
				RunControl:      queueControl,
				ProcessRegistry: stateStore,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
			Codex:    orquestaappcodexstack.CodexRuntimeConfigV0{Runtime: adopter},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready ||
		adopter.calls != 1 ||
		adopter.last.PID != 12345 ||
		!strings.Contains(result.Message, "agentes_adoptados=1") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-orquesta-startup-live-process-adoption") {
		t.Fatalf("result=%+v adopter=%+v", result, adopter)
	}
}

func TestStartupCleanupBlockersV0ExponeAccionYAntiguedadSinPathsV0(t *testing.T) {
	now := time.Date(2026, 6, 30, 17, 0, 0, 0, time.UTC)
	got := startupCleanupBlockersV0([]startupCandidateCleanupV0{
		{
			Candidate: orquestarunqueue.RunSchedulingCandidateV0{
				RunRef:       "run-startup-live-001",
				AppRef:       "app-opes-temporal",
				Status:       orquestarunqueue.RunStatusRunningV0,
				UpdatedAt:    now.Add(-10 * time.Minute),
				EvidenceRefs: []string{"evidence-live"},
			},
			Active: true,
		},
		{
			Candidate: orquestarunqueue.RunSchedulingCandidateV0{
				RunRef:    "run-startup-queue-dirty-001",
				AppRef:    "app-opes-temporal",
				Status:    orquestarunqueue.RunStatusReadyV0,
				UpdatedAt: now.Add(-2 * time.Hour),
			},
			QueueDirty:  true,
			QueueStatus: orquestarunqueue.RunStatusStoppedV0,
		},
	}, orquestaserver.StartupCheckCommandV0{OccurredAt: now})

	if len(got) != 2 {
		t.Fatalf("blockers=%+v", got)
	}
	if got[0].RunRef != "run-startup-live-001" ||
		got[0].AgeSeconds != 600 ||
		got[0].ProcessState != "active_or_in_flight" ||
		got[0].Action != "inspect_live_run_or_force_stop_explicit" ||
		!containsStringForTestV0(got[0].EvidenceRefs, "evidence-ref-orquesta-startup-dirty-runs") {
		t.Fatalf("live blocker=%+v", got[0])
	}
	if got[1].RunRef != "run-startup-queue-dirty-001" ||
		got[1].AgeSeconds != 7200 ||
		got[1].QueueStatus != orquestarunqueue.RunStatusStoppedV0 ||
		got[1].ProcessState != "no_live_process_required" ||
		got[1].Action != "reconcile_queue_terminal" {
		t.Fatalf("queue blocker=%+v", got[1])
	}
	for _, blocker := range got {
		joined := strings.Join([]string{
			blocker.RunRef,
			blocker.AppRef,
			blocker.QueueStatus,
			blocker.UpdatedAt,
			blocker.ProcessState,
			blocker.Action,
			strings.Join(blocker.EvidenceRefs, " "),
		}, " ")
		if strings.Contains(joined, "/") || strings.Contains(strings.ToLower(joined), "home") {
			t.Fatalf("blocker filtra path local: %+v", blocker)
		}
	}
}

func TestStartupSelectiveCleanupV0CierraScopeSinForcedStopGlobalV0(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	runRef := "run-opes-stale-safe"
	if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: runRef,
		AppRef: "opes",
		Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:  runRef,
		Status: orquestacoreworkflow.OrchestrationRunStatusActiveV0,
	})
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
				RunStore:   runStore,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		Mode:      startupCleanupModeSelectiveV0,
		ScopeRefs: []string{"opes"},
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 6, 30, 18, 40, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready ||
		result.Status != orquestaserver.StartupCheckStatusReadyV0 ||
		!resultStartupEvidenceContainsV0(result.EvidenceRefs, startupSelectiveCleanupEvidenceV0) {
		t.Fatalf("result=%+v", result)
	}
	control, err := store.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef})
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if control.Status != orquestaruncontrol.RunControlStatusStoppedV0 || control.Forced {
		t.Fatalf("control debe quedar parado sin forced_stop: %+v", control)
	}
	all, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             "queue-main",
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(all) != 1 ||
		all[0].Status != orquestarunqueue.RunStatusStoppedV0 ||
		!containsStringForTestV0(all[0].EvidenceRefs, startupSelectiveCleanupEvidenceV0) {
		t.Fatalf("candidate=%+v", all)
	}
}

func TestStartupSelectiveCleanupV0NoLimpiaFueraDeScopeV0(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	scopedRunRef := "run-opes-stale-safe"
	otherRunRef := "run-other-live"
	for _, candidate := range []orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: scopedRunRef, AppRef: "opes", Status: orquestarunqueue.RunStatusReadyV0},
		{RunRef: otherRunRef, AppRef: "other-app", Status: orquestarunqueue.RunStatusReadyV0},
	} {
		if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", candidate); err != nil {
			t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
		}
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(
		orquestacoreworkflow.OrchestrationRunV0{RunID: scopedRunRef, Status: orquestacoreworkflow.OrchestrationRunStatusActiveV0},
		orquestacoreworkflow.OrchestrationRunV0{
			RunID:         otherRunRef,
			Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			StartedAgents: []string{"agent-other-live"},
		},
	)
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
				RunStore:   runStore,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		Mode:      startupCleanupModeSelectiveV0,
		ScopeRefs: []string{"opes"},
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 6, 30, 18, 45, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if result.Ready ||
		result.Status != "startup_dirty_runs_detected" ||
		len(result.Blockers) != 1 ||
		result.Blockers[0].RunRef != otherRunRef {
		t.Fatalf("result=%+v", result)
	}
	all, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             "queue-main",
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	statusByRun := map[string]string{}
	for _, candidate := range all {
		statusByRun[candidate.RunRef] = candidate.Status
	}
	if statusByRun[scopedRunRef] != orquestarunqueue.RunStatusStoppedV0 ||
		statusByRun[otherRunRef] != orquestarunqueue.RunStatusReadyV0 {
		t.Fatalf("statusByRun=%+v", statusByRun)
	}
}

func TestStartupCandidateCanBeCompletedByStartupV0AceptaStopSolicitadoSinACKV0(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-startup-stop-sin-ack",
		StartedAgents: []string{"agent-ref-startup-stop-sin-ack"},
		StoppedAgents: []string{"agent-ref-startup-stop-sin-ack"},
	})
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{RunStore: runStore},
		},
		Mode: startupCleanupModeForcedStopV0,
	}

	ready, err := check.startupCandidateCanBeCompletedByStartupV0(ctx, "run-startup-stop-sin-ack")
	if err != nil {
		t.Fatalf("startupCandidateCanBeCompletedByStartupV0: %v", err)
	}
	if !ready {
		t.Fatalf("stop solicitado sin ack no debe bloquear purga de arranque")
	}
}

type fakeStartupProcessAdopterV0 struct {
	calls int
	last  orquestaruntime.ProcessRuntimeSnapshotV0
}

func (fake *fakeStartupProcessAdopterV0) LaunchV0(
	context.Context,
	orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return orquestaruntime.ProcessRuntimeSnapshotV0{}, nil
}

func (fake *fakeStartupProcessAdopterV0) AdoptProcessV0(
	_ context.Context,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	fake.calls++
	fake.last = snapshot
	snapshot.Status = orquestaruntime.ProcessRuntimeRunningV0
	return snapshot, nil
}

func TestCompactStartupStateFilesV0ArchivaBasuraActiva(t *testing.T) {
	stateDir := t.TempDir()
	runtimeDir := t.TempDir()
	runStateDir := filepath.Join(stateDir, "run-state")
	if err := os.MkdirAll(runStateDir, 0o755); err != nil {
		t.Fatalf("mkdir run-state: %v", err)
	}
	queue := startupQueueSnapshotV0{
		SchemaVersion: "orquesta.run_file.queue.v0",
		Records: []startupQueueRecordV0{
			{
				RunRef:   "run-stopped",
				QueueRef: "global",
				Candidate: orquestarunqueue.RunSchedulingCandidateV0{
					RunRef: "run-stopped",
					Status: orquestarunqueue.RunStatusStoppedV0,
				},
			},
			{
				RunRef:   "run-ready-completed",
				QueueRef: "global",
				Candidate: orquestarunqueue.RunSchedulingCandidateV0{
					RunRef: "run-ready-completed",
					Status: "ready",
				},
			},
			{
				RunRef:   "run-ready-live",
				QueueRef: "global",
				Candidate: orquestarunqueue.RunSchedulingCandidateV0{
					RunRef: "run-ready-live",
					Status: "ready",
				},
			},
		},
	}
	control := startupControlSnapshotV0{
		SchemaVersion: "orquesta.run_file.control.v0",
		Records: []startupControlRecordV0{
			{
				RunRef: "run-stopped",
				State: orquestaruncontrol.RunControlStateV0{
					RunRef: "run-stopped",
					Status: orquestaruncontrol.RunControlStatusStoppedV0,
				},
			},
			{
				RunRef: "run-running",
				State: orquestaruncontrol.RunControlStateV0{
					RunRef: "run-running",
					Status: orquestaruncontrol.RunControlStatusRunningV0,
				},
			},
		},
	}
	writeStartupJSONForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"), queue)
	writeStartupJSONForTestV0(t, filepath.Join(runStateDir, "control_v0.json"), control)
	agentDir := filepath.Join(runtimeDir, "run-ready-completed", "agent-1")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent: %v", err)
	}
	writeStartupStrictCompletedAckForTestV0(t, agentDir, startupStrictAckFixtureV0{
		RunRef:   "run-ready-completed",
		AgentRef: "agent-1",
	})
	if err := os.WriteFile(filepath.Join(agentDir, "agent_prompt.txt"), []byte("prompt secret_token=abc"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "stderr.log"), []byte("stderr Bearer abc"), 0o644); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	check := serverStartupCheckV0{ServerConfig: orquestaserver.ConfigV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
	}}
	got, err := check.compactStartupStateFilesV0(orquestaserver.StartupCheckCommandV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
		OccurredAt:     time.Date(2026, 5, 18, 15, 30, 31, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("compactStartupStateFilesV0: %v", err)
	}
	if !got.CompactionNeeded ||
		got.QueueRemoved != 2 ||
		got.QueueKept != 1 ||
		got.ControlRemoved != 1 ||
		got.ControlKept != 1 ||
		got.RuntimeArchived != 1 {
		t.Fatalf("compaction=%+v", got)
	}
	queueAfter := readStartupQueueForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"))
	if len(queueAfter.Records) != 1 || queueAfter.Records[0].RunRef != "run-ready-live" {
		t.Fatalf("queue after=%+v", queueAfter)
	}
	controlAfter := readStartupControlForTestV0(t, filepath.Join(runStateDir, "control_v0.json"))
	if len(controlAfter.Records) != 1 || controlAfter.Records[0].RunRef != "run-running" {
		t.Fatalf("control after=%+v", controlAfter)
	}
	if got.RevisionRef == "" || got.RetentionDays == 0 || got.MaxBytes == 0 || got.Artifacts == 0 {
		t.Fatalf("revision policy missing=%+v", got)
	}
	archiveManifest := filepath.Join(got.RevisionDir, "runtime_archived", "run-ready-completed", "manifest.json")
	if _, err := os.Stat(archiveManifest); err != nil {
		t.Fatalf("runtime summary not archived: %v", err)
	}
	archiveBytes, err := os.ReadFile(archiveManifest)
	if err != nil {
		t.Fatalf("read runtime archive manifest: %v", err)
	}
	for _, forbidden := range []string{"secret_token", "Bearer", "agent_prompt.txt", "stderr.log"} {
		if strings.Contains(string(archiveBytes), forbidden) {
			t.Fatalf("runtime archive leaks %q: %s", forbidden, string(archiveBytes))
		}
	}
	manifestBytes, err := os.ReadFile(filepath.Join(got.RevisionDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read revision manifest: %v", err)
	}
	if strings.Contains(string(manifestBytes), got.RevisionDir) || strings.Contains(string(manifestBytes), "revision_dir") {
		t.Fatalf("revision manifest leaks local path: %s", string(manifestBytes))
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "run-ready-completed")); err != nil {
		t.Fatalf("runtime source no queda preservado: %v", err)
	}
}

func TestCompactStartupStateFilesV0ApartaReadyConRunAutoprogrammingObsoleto(t *testing.T) {
	stateDir := t.TempDir()
	runtimeDir := t.TempDir()
	runStateDir := filepath.Join(stateDir, "run-state")
	if err := os.MkdirAll(runStateDir, 0o755); err != nil {
		t.Fatalf("mkdir run-state: %v", err)
	}
	staleRunRef := "request-ref-autoprogramming-backlog-stale-assessment-001"
	liveRunRef := "request-ref-autoprogramming-backlog-live-001"
	queue := startupQueueSnapshotV0{
		SchemaVersion: "orquesta.run_file.queue.v0",
		Records: []startupQueueRecordV0{
			{
				RunRef:   staleRunRef,
				QueueRef: "global",
				Candidate: orquestarunqueue.RunSchedulingCandidateV0{
					RunRef: staleRunRef,
					Status: "ready",
				},
			},
			{
				RunRef:   liveRunRef,
				QueueRef: "global",
				Candidate: orquestarunqueue.RunSchedulingCandidateV0{
					RunRef: liveRunRef,
					Status: "ready",
				},
			},
		},
	}
	writeStartupJSONForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"), queue)
	writeStartupJSONForTestV0(t, filepath.Join(runStateDir, "control_v0.json"), startupControlSnapshotV0{})
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
		RootDir: filepath.Join(stateDir, "orchestration-state"),
	})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	agentRef := "agent-ref-autoprogramming-backlog-stale-assessment-001"
	ghostAgentRef := "agent-ref-autoprogramming-backlog-stale-assessment-ghost-001"
	if err := stateStore.SaveRunV0(context.Background(), orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         staleRunRef,
		ProjectRef:    "project-ref-startup-stale-001",
		AppSpecRef:    "appspec-ref-startup-stale-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		}},
		LastEventID:       "evt-startup-stale-001",
		LastSequence:      1,
		Agents:            []string{agentRef, ghostAgentRef},
		StartedAgents:     []string{agentRef, ghostAgentRef},
		StoppedAgents:     []string{agentRef},
		AgentStopRequests: []string{agentRef + "#reason:startup_stale_assessment"},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-startup-stale-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: agentRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		})},
	}); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}

	check := serverStartupCheckV0{ServerConfig: orquestaserver.ConfigV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
	}}
	got, err := check.compactStartupStateFilesV0(orquestaserver.StartupCheckCommandV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
		OccurredAt:     time.Date(2026, 5, 25, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("compactStartupStateFilesV0: %v", err)
	}
	if !got.CompactionNeeded || got.QueueRemoved != 1 || got.QueueKept != 1 {
		t.Fatalf("compaction=%+v", got)
	}
	queueAfter := readStartupQueueForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"))
	if len(queueAfter.Records) != 1 || queueAfter.Records[0].RunRef != liveRunRef {
		t.Fatalf("queue after=%+v", queueAfter)
	}
	removed := readStartupQueueForTestV0(t, filepath.Join(got.RevisionDir, "queue_removed.json"))
	if len(removed.Records) != 1 ||
		removed.Records[0].RunRef != staleRunRef ||
		removed.Records[0].PurgeReason != "ready_run_autoprogramming_obsoleto" {
		t.Fatalf("removed=%+v", removed)
	}
	if _, err := stateStore.LoadRunV0(context.Background(), staleRunRef); err != nil {
		t.Fatalf("run historico no debe borrarse: %v", err)
	}
}

func TestForceStopStartupStateV0CompactaDespuesDeMarcarTerminal(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runtimeDir := t.TempDir()
	runStateDir := filepath.Join(stateDir, "run-state")
	store, err := orquestarunfile.NewRunFileStoreV0(runStateDir)
	if err != nil {
		t.Fatalf("NewRunFileStoreV0: %v", err)
	}
	runRef := "run-ready-for-startup-clean"
	if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "global", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: runRef,
		AppRef: "app-clean",
		Status: "ready",
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runtimeDir, runRef, "agent-1"), 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, runRef, "agent-1", "agent_ack.json"), []byte(`{"status":"completed"}`), 0o644); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "global"},
		},
		ServerConfig: orquestaserver.ConfigV0{
			StateDir:       stateDir,
			RuntimeWorkDir: runtimeDir,
		},
		Mode: startupCleanupModeForcedStopV0,
	}
	result, err := check.forceStopStartupStateV0(ctx, orquestaserver.StartupCheckCommandV0{
		StateDir:       stateDir,
		RuntimeWorkDir: runtimeDir,
		OccurredAt:     time.Date(2026, 5, 18, 15, 52, 51, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("forceStopStartupStateV0: %v", err)
	}
	if !result.Ready || !resultStartupEvidenceContainsV0(result.EvidenceRefs, "evidence-ref-orquesta-startup-state-compacted") {
		t.Fatalf("result=%+v", result)
	}
	queueAfter := readStartupQueueForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"))
	if len(queueAfter.Records) != 0 {
		t.Fatalf("queue after=%+v", queueAfter)
	}
	controlAfter := readStartupControlForTestV0(t, filepath.Join(runStateDir, "control_v0.json"))
	if len(controlAfter.Records) != 0 {
		t.Fatalf("control after=%+v", controlAfter)
	}
	if _, err := store.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef}); err == nil {
		t.Fatalf("store en memoria conserva control terminal tras compactar")
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, runRef)); err != nil {
		t.Fatalf("runtime source no queda preservado: %v", err)
	}
}

func TestStoppedStartupQueueCandidateV0MarcaColaTerminal(t *testing.T) {
	now := time.Date(2026, 5, 18, 15, 20, 0, 0, time.UTC)
	candidate := orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        "run-001",
		AppRef:        "app-001",
		Status:        "ready",
		PriorityScore: 70,
		UpdatedAt:     now.Add(-time.Hour),
		EvidenceRefs: []string{
			"evidence-prev",
			" evidence-prev ",
			"",
		},
	}

	got := stoppedStartupQueueCandidateV0(candidate, orquestaserver.StartupCheckCommandV0{
		OccurredAt: now,
	})

	if got.Status != orquestarunqueue.RunStatusStoppedV0 {
		t.Fatalf("status=%q", got.Status)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at=%s", got.UpdatedAt)
	}
	if got.PriorityScore != candidate.PriorityScore || got.RunRef != candidate.RunRef || got.AppRef != candidate.AppRef {
		t.Fatalf("candidate=%+v", got)
	}
	if len(got.EvidenceRefs) != 2 ||
		got.EvidenceRefs[0] != "evidence-prev" ||
		got.EvidenceRefs[1] != "evidence-ref-orquesta-startup-queue-stopped" {
		t.Fatalf("evidence_refs=%#v", got.EvidenceRefs)
	}
}
