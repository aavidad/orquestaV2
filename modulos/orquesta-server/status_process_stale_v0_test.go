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
		stale.ShutdownHTTPStatus != 0 {
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
		readiness.StartupStatus != ServerProcessStaleReasonCodeV0 ||
		readiness.StartupMessage != "server_process_not_alive" {
		t.Fatalf("readiness no expone stale: %+v", readiness)
	}

	public := NewServerPublicStatusV0(stale)
	if public.Status != "stale" ||
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
