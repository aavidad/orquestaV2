package orquestaserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeV0RestauraEstadoDurableAlRecrearInstanciaV0(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStateStoreV0(filepath.Join(dir, DefaultStateFileV0))
	if err != nil {
		t.Fatalf("NewFileStateStoreV0: %v", err)
	}
	previous := StateV0{
		SchemaVersion:              StateSchemaVersionV0,
		Status:                     "running",
		PID:                        44,
		Addr:                       "127.0.0.1:1",
		ProcessRef:                 "process-ref-old",
		StartedAt:                  "2026-05-26T10:00:00Z",
		LastHeartbeatAt:            "2026-05-26T10:05:00Z",
		LastSupervisorQueueRef:     "queue-ref-automejora",
		LastSupervisorQueueSize:    3,
		SupervisorTicks:            7,
		SupervisorExecutions:       5,
		StatePersistStatus:         "ok",
		StatePersistLastConfirmed:  "2026-05-26T10:05:00Z",
		IdleSelfImprovementCheck:   "2026-05-26T10:04:00Z",
		IdleSelfImprovementReason:  "shutdown_in_progress",
		IdleSelfImprovementFlight:  true,
		IdleSelfImprovementRuns:    3,
		IdleSelfImprovementOK:      2,
		ExternalBridgeStatus:       "idle",
		ExternalBridgeTicks:        4,
		ExternalBridgeLastTickRef:  "bridge-tick-ref-1",
		AuditStatus:                "ok",
		AuditLastConfirmed:         "2026-05-26T10:05:00Z",
		ResponseWriteFailures:      1,
		ResponseWriteLastCode:      "response_write_failed",
		RecentErrors:               []ServerDiagnosticV0{{Code: "state_persist_failed", Scope: "state"}},
		ShutdownAsyncWorkActive:    9,
		ShutdownInProgress:         true,
		SupervisorFrozen:           true,
		SupervisorTickActive:       true,
		StartupReady:               true,
		StartupStatus:              "ready",
		StartupMessage:             "old startup",
		IdleSelfImprovementAfter:   "1m0s",
		IdleSelfImprovementTarget:  3,
		ProjectWorkDir:             "/tmp/old-project",
		RuntimeWorkDir:             "/tmp/old-runtime",
		EffectiveConfig:            ServerEffectiveConfigV0{SchemaVersion: "old"},
		ShutdownReady:              true,
		ShutdownCheckpointsPending: 2,
		ShutdownRunsRequested:      6,
		ShutdownRunsStopped:        5,
		ShutdownAgentsInFlight:     1,
		ShutdownHTTPStatus:         503,
		ShutdownStatus:             "waiting",
		ShutdownStopTimeoutAt:      "2026-05-26T10:07:00Z",
		ShutdownSignalName:         "TERM",
		ShutdownSignalCount:        1,
		ShutdownSignalEscalated:    true,
		LastShutdownAt:             "2026-05-26T10:06:00Z",
		LastSupervisorOperationalMessage: &ServerOperationalMessageV0{
			Scope: "supervisor",
		},
	}
	if err := store.SaveServerStateV0(context.Background(), previous); err != nil {
		t.Fatalf("SaveServerStateV0: %v", err)
	}
	now := time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      dir,
		Addr:          "127.0.0.1:8789",
		AuditDisabled: true,
	}, RuntimeDepsV0{Clock: fixedClockV0{now: now}})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	got := runtime.StateV0()
	if got.Status != "starting" ||
		got.PID != os.Getpid() ||
		got.Addr != "127.0.0.1:8789" ||
		got.ProcessRef == previous.ProcessRef ||
		got.StartedAt != formatTimeV0(now) {
		t.Fatalf("identidad viva no renovada: %+v", got)
	}
	if got.SupervisorTicks != 7 ||
		got.SupervisorExecutions != 5 ||
		got.LastSupervisorQueueRef != "queue-ref-automejora" ||
		got.ExternalBridgeTicks != 4 ||
		got.StatePersistLastConfirmed == "" ||
		got.AuditLastConfirmed == "" ||
		len(got.RecentErrors) != 1 {
		t.Fatalf("estado durable no restaurado: %+v", got)
	}
	if got.SupervisorTickActive ||
		got.SupervisorFrozen ||
		got.ShutdownInProgress ||
		got.ShutdownStatus != "" ||
		got.ShutdownHTTPStatus != 0 ||
		got.ShutdownRunsRequested != 0 ||
		got.ShutdownAgentsInFlight != 0 ||
		got.ShutdownAsyncWorkActive != 0 ||
		got.StartupReady ||
		got.IdleSelfImprovementFlight {
		t.Fatalf("campos volatiles no reiniciados: %+v", got)
	}
	if got.IdleSelfImprovementReason != "" {
		t.Fatalf("idle shutdown reason no limpiado: %+v", got)
	}
	if _, lastAttempt, inFlight, accepted := runtime.tracker.IdleSelfImprovementWindowV0(); !lastAttempt.Equal(time.Date(2026, 5, 26, 10, 4, 0, 0, time.UTC)) || inFlight || accepted {
		t.Fatalf("ventana idle restaurada last=%s in_flight=%v accepted=%v", lastAttempt, inFlight, accepted)
	}
	state := runtime.tracker.MarkIdleSelfImprovementCheckedV0("attempt_blocked", now)
	if state.IdleSelfImprovementRuns != 3 ||
		state.IdleSelfImprovementOK != 2 ||
		state.IdleSelfImprovementReason != "attempt_blocked" {
		t.Fatalf("contadores idle restaurados=%+v", state)
	}
}
