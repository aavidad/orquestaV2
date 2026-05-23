package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
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
	if err := os.WriteFile(filepath.Join(agentDir, "codex_last_message.txt"), []byte("ACK ack-1 completed"), 0o644); err != nil {
		t.Fatalf("write last message: %v", err)
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
	if _, err := os.Stat(filepath.Join(got.RevisionDir, "runtime_archived", "run-ready-completed")); err != nil {
		t.Fatalf("runtime not archived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runtimeDir, "run-ready-completed")); !os.IsNotExist(err) {
		t.Fatalf("runtime source still exists or unexpected err=%v", err)
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
	if _, err := os.Stat(filepath.Join(runtimeDir, runRef)); !os.IsNotExist(err) {
		t.Fatalf("runtime source still exists or unexpected err=%v", err)
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
