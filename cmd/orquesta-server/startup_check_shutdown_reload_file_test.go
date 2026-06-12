package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestDiagnoseStartupV0BackupDurableYQuitaStopPendienteDeColaActiva(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runStateDir := filepath.Join(stateDir, "run-state")
	store, err := orquestarunfile.NewRunFileStoreV0(runStateDir)
	if err != nil {
		t.Fatalf("NewRunFileStoreV0: %v", err)
	}
	runRef := "run-shutdown-reload-file-stale-001"
	if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "global", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: runRef,
		AppRef: "app-stale",
		Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	if _, err := store.PutRunControlStateV0(ctx, orquestaruncontrol.RunControlStateV0{
		RunRef: runRef,
		Status: orquestaruncontrol.RunControlStatusStopRequestedV0,
		Forced: true,
	}); err != nil {
		t.Fatalf("PutRunControlStateV0: %v", err)
	}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "global"},
		},
		ServerConfig: orquestaserver.ConfigV0{StateDir: stateDir},
		Mode:         startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		StateDir:   stateDir,
		OccurredAt: time.Date(2026, 6, 11, 8, 10, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready ||
		result.StartupRevision.QueueRemoved != 1 ||
		result.StartupRevision.ControlRemoved != 1 ||
		!resultStartupEvidenceContainsV0(result.EvidenceRefs, "evidence-ref-orquesta-startup-state-compacted") {
		t.Fatalf("result=%+v", result)
	}
	queueAfter := readStartupQueueForTestV0(t, filepath.Join(runStateDir, "queue_v0.json"))
	if len(queueAfter.Records) != 0 {
		t.Fatalf("queue after=%+v", queueAfter)
	}
	removed := readStartupQueueForTestV0(t, filepath.Join(stateDir, "revision", "revision-ref-orquesta-startup-20260611t081000z", "queue_removed.json"))
	if len(removed.Records) != 1 ||
		removed.Records[0].RunRef != runRef ||
		removed.Records[0].PurgeReason != "queue_status_no_ejecutable" {
		t.Fatalf("removed=%+v", removed)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "revision", "revision-ref-orquesta-startup-20260611t081000z", "manifest.json")); err != nil {
		t.Fatalf("manifest durable ausente: %v", err)
	}
}
