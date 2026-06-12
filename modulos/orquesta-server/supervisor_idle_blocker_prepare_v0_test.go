package orquestaserver

import (
	"context"
	"strings"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0SupervisorNoPreparaAutomejoraConProveedorAuthBloqueadoV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		results: []fakeSupervisorResultV0{{result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
		}}},
		planRequests: []IdleSelfImprovementRequestV0{{RequestRef: "request-ref-backlog-no-debe-usarse"}},
		selfStarted:  make(chan struct{}, 1),
		blocker: IdleSelfImprovementBlockerResultV0{
			Blocked:        true,
			Reason:         "provider_auth_blocked",
			RunRefs:        []string{"run-ref-auth-blocked-001"},
			EvidenceRefs:   []string{"evidence-ref-auth-config-blocker"},
			Message:        "provider_auth_blocked: restaurar proveedor y reanudar run.",
			RecoveryAction: "restore_provider_credentials_then_resume_run",
			NextActions:    []string{"restore_provider_credentials"},
		},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                       t.TempDir(),
		TickInterval:                   time.Hour,
		IdleSelfImprovementAfter:       time.Minute,
		IdleSelfImprovementMaxRequests: 3,
		IdleSelfImprovementTargetQueue: 4,
		AuditDisabled:                  true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(context.Background())

	if supervisor.planCalls != 0 || supervisor.selfCalls != 0 {
		t.Fatalf("bloqueo proveedor no debe planificar: plan_calls=%d self_calls=%d", supervisor.planCalls, supervisor.selfCalls)
	}
	if !strings.HasPrefix(store.last.IdleSelfImprovementReason, "provider_auth_blocked") ||
		!strings.Contains(store.last.IdleSelfImprovementReason, "run-ref-auth-blocked-001") ||
		!strings.Contains(store.last.IdleSelfImprovementReason, "restore_provider_credentials_then_resume_run") {
		t.Fatalf("reason=%q state=%+v", store.last.IdleSelfImprovementReason, store.last)
	}
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != "provider_auth_blocked" ||
		store.last.IdleSelfImprovementOperationalMessage.Status != "blocked" ||
		store.last.IdleSelfImprovementOperationalMessage.RunRefs[0] != "run-ref-auth-blocked-001" ||
		store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs[0] != "evidence-ref-auth-config-blocker" {
		t.Fatalf("operational_message=%+v", store.last.IdleSelfImprovementOperationalMessage)
	}
}

func TestIdleSelfImprovementAuditPayloadRedactaMensajeDeBlockerV0(t *testing.T) {
	payload := idleSelfImprovementScheduleDecisionV0{
		Reason:          "provider_auth_blocked",
		BlockerRunRefs:  []string{"run-ref-auth-blocked-001"},
		BlockerEvidence: []string{"evidence-ref-auth-config-blocker"},
		BlockerMessage:  "reautorizar token en /home/user/.config/provider",
	}.AuditPayload()

	if payload["blocker_message"] != "diagnostic_message_redacted" {
		t.Fatalf("blocker_message=%q", payload["blocker_message"])
	}
	if payload["blocker_run_refs"].([]string)[0] != "run-ref-auth-blocked-001" ||
		payload["blocker_evidence"].([]string)[0] != "evidence-ref-auth-config-blocker" {
		t.Fatalf("payload compacto perdido: %+v", payload)
	}
}

func TestStatusTrackerV0ConservaAutomejoraAceptadaDuranteBloqueoV0(t *testing.T) {
	now := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted: true, RunRef: "run-ref-1", RequestRef: "request-ref-1",
	}, now)
	state := tracker.MarkIdleSelfImprovementCheckedV0("attempt_blocked", now.Add(time.Minute))
	if !strings.Contains(state.IdleSelfImprovementReason, "run_ref=run-ref-1") ||
		!strings.Contains(state.IdleSelfImprovementReason, "pending=accepted_attempt") {
		t.Fatalf("reason=%q", state.IdleSelfImprovementReason)
	}
}

func TestRuntimeV0PrepareIdleSelfImprovementBatchNoMezclaFalloConRunAceptadoV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		selfResults: []IdleSelfImprovementResultV0{{
			Accepted:   false,
			RequestRef: "request-ref-autoprogramming-backlog-t36-codex-ack-strict-terminal-validation-e71900c9",
			Status:     "error",
			Message:    "autoprogramming run existente incompatible: request-ref-autoprogramming-backlog-t36-codex-ack-strict-terminal-validation-e71900c9",
		}, {
			Accepted:   true,
			RunRef:     "request-ref-autoprogramming-backlog-t197-cli-rest-response-bounds-redaction-parity-4f78dcfb",
			RequestRef: "request-ref-autoprogramming-backlog-t197-cli-rest-response-bounds-redaction-parity-4f78dcfb",
			Status:     "ok",
			Message:    "prepared",
			EvidenceRefs: []string{
				"evidence-ref-t197-prepared",
			},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.prepareIdleSelfImprovementBatchV0(context.Background(), supervisor, []IdleSelfImprovementRequestV0{
		{RequestRef: "request-ref-autoprogramming-backlog-t36-codex-ack-strict-terminal-validation-e71900c9"},
		{RequestRef: "request-ref-autoprogramming-backlog-t197-cli-rest-response-bounds-redaction-parity-4f78dcfb"},
	})

	reason := store.last.IdleSelfImprovementReason
	if !strings.Contains(reason, "run_ref=request-ref-autoprogramming-backlog-t197-cli-rest-response-bounds-redaction-parity-4f78dcfb") ||
		!strings.Contains(reason, "status=ok") ||
		!strings.Contains(reason, "repair_failed_prepare_requests_and_retry") {
		t.Fatalf("reason=%q", reason)
	}
	if strings.Contains(reason, "status=error") ||
		strings.Contains(reason, "autoprogramming run existente incompatible") {
		t.Fatalf("reason mezcla fallo con run aceptado=%q", reason)
	}
}

func TestRuntimeV0PrepareIdleSelfImprovementBatchExponeFalloAccionableV0(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	store := &memoryStateStoreV0{}
	supervisor := &fakeSupervisorV0{
		selfResults: []IdleSelfImprovementResultV0{{
			Accepted:    false,
			RequestRef:  "request-ref-autoprogramming-backlog-t999",
			Status:      "error",
			Message:     "idle_self_improvement_queue_candidate_not_executable",
			NextActions: []string{"planner_should_skip_control_blocked_backlog_ref"},
			EvidenceRefs: []string{
				"evidence-ref-idle-self-improvement-control-blocked",
			},
		}},
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:      t.TempDir(),
		TickInterval:  time.Hour,
		AuditDisabled: true,
	}, RuntimeDepsV0{
		Supervisor: supervisor,
		StateStore: store,
		Clock:      fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	runtime.prepareIdleSelfImprovementBatchV0(context.Background(), supervisor, []IdleSelfImprovementRequestV0{{
		RequestRef: "request-ref-autoprogramming-backlog-t999",
	}})

	reason := store.last.IdleSelfImprovementReason
	if !strings.HasPrefix(reason, "error:prepare_failed") ||
		!strings.Contains(reason, "request_ref=request-ref-autoprogramming-backlog-t999") ||
		!strings.Contains(reason, "message=idle_self_improvement_queue_candidate_not_executable") ||
		!strings.Contains(reason, "planner_should_skip_control_blocked_backlog_ref") {
		t.Fatalf("reason=%q", reason)
	}
	if store.last.IdleSelfImprovementOperationalMessage == nil ||
		store.last.IdleSelfImprovementOperationalMessage.ReasonCode != "prepare_failed" ||
		store.last.IdleSelfImprovementOperationalMessage.RequestRefs[0] != "request-ref-autoprogramming-backlog-t999" ||
		store.last.IdleSelfImprovementOperationalMessage.EvidenceRefs[0] != "evidence-ref-idle-self-improvement-control-blocked" {
		t.Fatalf("operational_message=%+v", store.last.IdleSelfImprovementOperationalMessage)
	}
	if len(store.last.RecentErrors) == 0 ||
		store.last.RecentErrors[0].Code != "idle_self_improvement_prepare_failed" ||
		store.last.RecentErrors[0].EvidenceRefs[0] != "evidence-ref-idle-self-improvement-control-blocked" {
		t.Fatalf("recent_errors=%+v", store.last.RecentErrors)
	}

	runtime.markIdleSelfImprovementCheckedV0(context.Background(), "attempt_blocked", now.Add(time.Second))
	reason = store.last.IdleSelfImprovementReason
	if !strings.Contains(reason, "pending=failed_attempt") ||
		!strings.Contains(reason, "request_ref=request-ref-autoprogramming-backlog-t999") {
		t.Fatalf("reason tras attempt_blocked=%q", reason)
	}
}
