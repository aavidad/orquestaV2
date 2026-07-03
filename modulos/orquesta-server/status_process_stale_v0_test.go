package orquestaserver

import (
	"testing"
	"time"
)

func TestMarkServerProcessStaleStateV0ExponeCausaTrasReadinessV0(t *testing.T) {
	now := time.Date(2026, 7, 2, 10, 30, 0, 0, time.UTC)
	state := StateV0{
		SchemaVersion:              StateSchemaVersionV0,
		Status:                     "running",
		PID:                        4242,
		Addr:                       "127.0.0.1:36675",
		ProcessRef:                 "process-ref-http-ready-then-dead",
		LastHeartbeatAt:            "2026-07-02T10:29:00Z",
		StartupReady:               true,
		StartupStatus:              StartupCheckStatusReadyV0,
		StartupMessage:             "orquesta_startup_ready",
		StartupEvidenceRefs:        []string{"evidence-ref-startup-ready"},
		SupervisorTickActive:       true,
		ResidentDirectorTickActive: true,
		GoalObserverTickActive:     true,
		ExternalBridgeTickActive:   true,
		ShutdownInProgress:         true,
		ShutdownStatus:             "waiting_backend",
		ShutdownReady:              true,
		ShutdownHTTPStatus:         503,
	}

	stale := MarkServerProcessStaleStateV0(state, now)

	if stale.Status != "stale" ||
		stale.StartupReady ||
		stale.StartupStatus != ServerProcessStaleReasonCodeV0 ||
		stale.StartupMessage != "server_process_not_alive" ||
		stale.LastError != "server_process_not_alive" {
		t.Fatalf("state stale incompleto: %+v", stale)
	}
	if stale.SupervisorTickActive ||
		stale.ResidentDirectorTickActive ||
		stale.GoalObserverTickActive ||
		stale.ExternalBridgeTickActive ||
		stale.ShutdownInProgress ||
		stale.ShutdownStatus != "" ||
		stale.ShutdownReady ||
		stale.ShutdownHTTPStatus != 0 ||
		stale.ShutdownRunsRequested != 0 ||
		stale.ShutdownRunsStopped != 0 ||
		stale.ShutdownAgentsInFlight != 0 ||
		stale.ShutdownCheckpointsPending != 0 ||
		stale.ShutdownCheckpointAgentsPending != 0 ||
		stale.ShutdownAsyncWorkActive != 0 ||
		stale.ShutdownStopTimeoutAt != "" {
		t.Fatalf("actividad viva no limpiada en stale: %+v", stale)
	}
	if stale.StartupOperationalMessage == nil ||
		stale.StartupOperationalMessage.ReasonCode != ServerProcessStaleReasonCodeV0 ||
		stale.StartupOperationalMessage.Status != "stale" {
		t.Fatalf("startup_operational_message=%+v", stale.StartupOperationalMessage)
	}
	if stale.LastErrorOperationalMessage == nil ||
		stale.LastErrorOperationalMessage.ReasonCode != ServerProcessStaleReasonCodeV0 ||
		stale.LastErrorOperationalMessage.Status != "stale" {
		t.Fatalf("last_error_operational_message=%+v", stale.LastErrorOperationalMessage)
	}
	if len(stale.RecentErrors) == 0 ||
		stale.RecentErrors[0].Code != ServerProcessStaleReasonCodeV0 ||
		stale.RecentErrors[0].Scope != "server" ||
		!containsServerStringForTestV0(
			stale.RecentErrors[0].EvidenceRefs,
			"evidence-ref-server-statefile-process-not-alive",
		) {
		t.Fatalf("recent_errors no expone stale verificable: %+v", stale.RecentErrors)
	}

	readiness := NewServerReadinessV0(stale)
	if readiness.Ready ||
		readiness.StartupReady ||
		readiness.Status != "stale" ||
		readiness.AvailabilityStatus != "crashed" ||
		readiness.AvailabilityReason != "server_crashed_after_readiness" ||
		readiness.StartupStatus != ServerProcessStaleReasonCodeV0 ||
		readiness.StartupMessage != "server_process_not_alive" {
		t.Fatalf("readiness no expone stale: %+v", readiness)
	}
	if !containsServerStringForTestV0(
		readiness.AvailabilityNextActions,
		"restart_orquesta_server_goal_first_app_server_tmux",
	) {
		t.Fatalf("readiness availability actions=%+v", readiness.AvailabilityNextActions)
	}

	public := NewServerPublicStatusV0(stale)
	if public.Status != "stale" ||
		public.AvailabilityStatus != "crashed" ||
		public.AvailabilityReason != "server_crashed_after_readiness" ||
		public.StartupReady ||
		public.StartupStatus != ServerProcessStaleReasonCodeV0 ||
		public.LastError != "server_process_not_alive" ||
		public.StartupOperationalMessage == nil ||
		public.StartupOperationalMessage.ReasonCode != ServerProcessStaleReasonCodeV0 ||
		public.LastErrorOperationalMessage == nil ||
		public.LastErrorOperationalMessage.ReasonCode != ServerProcessStaleReasonCodeV0 ||
		len(public.RecentErrors) == 0 ||
		public.RecentErrors[0].Code != ServerProcessStaleReasonCodeV0 {
		t.Fatalf("status publico no expone stale: %+v", public)
	}
}

func TestServerAvailabilityV0ExponeStoppedAccionableV0(t *testing.T) {
	state := StateV0{
		SchemaVersion: StateSchemaVersionV0,
		Status:        "stopped",
		StartupReady:  false,
		StartupStatus: "stopped",
	}

	readiness := NewServerReadinessV0(state)
	if readiness.Ready ||
		readiness.AvailabilityStatus != "stopped" ||
		readiness.AvailabilityReason != "server_stopped" ||
		!containsServerStringForTestV0(readiness.AvailabilityNextActions, "start_orquesta_server_goal_first_app_server_tmux") {
		t.Fatalf("readiness stopped=%+v", readiness)
	}

	public := NewServerPublicStatusV0(state)
	if public.AvailabilityStatus != "stopped" ||
		public.AvailabilityReason != "server_stopped" ||
		!containsServerStringForTestV0(public.AvailabilityEvidenceRefs, "evidence-ref-server-state-stopped") {
		t.Fatalf("public stopped=%+v", public)
	}
}

func TestStatusTrackerStoppedV0LimpiaStartupReadyV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 9, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:18787"}, now)
	tracker.MarkServingV0("127.0.0.1:18787", now)
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status:       StartupCheckStatusReadyV0,
		Message:      "startup_ready",
		EvidenceRefs: []string{"evidence-ref-startup-ready"},
	}, now)

	stopped := tracker.MarkStoppedV0(now.Add(time.Minute))

	if stopped.Status != "stopped" ||
		stopped.StartupReady ||
		stopped.StartupStatus != "stopped" {
		t.Fatalf("MarkStoppedV0 debe limpiar startup ready: %+v", stopped)
	}
	readiness := NewServerReadinessV0(stopped)
	if readiness.Ready || readiness.StartupReady || readiness.StartupStatus != "stopped" {
		t.Fatalf("readiness stopped no debe publicar startup ready: %+v", readiness)
	}
}

func TestStatusTrackerRuntimeStoppedV0LimpiaStartupReadyV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 9, 15, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:18787"}, now)
	tracker.MarkServingV0("127.0.0.1:18787", now)
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status:  StartupCheckStatusReadyV0,
		Message: "startup_ready",
	}, now)
	tracker.MarkRuntimeStoppingV0("async_work_draining", 1, now.Add(time.Minute))

	stopped := tracker.MarkRuntimeStoppedV0(now.Add(2 * time.Minute))

	if stopped.Status != "stopped" ||
		stopped.StartupReady ||
		stopped.StartupStatus != "stopped" ||
		stopped.ShutdownStatus != "stopped" ||
		!stopped.ShutdownReady {
		t.Fatalf("MarkRuntimeStoppedV0 debe limpiar startup ready y conservar shutdown stopped: %+v", stopped)
	}
	public := NewServerPublicStatusV0(stopped)
	if public.StartupReady || public.StartupStatus != "stopped" {
		t.Fatalf("status publico stopped no debe heredar startup ready: %+v", public)
	}
}
