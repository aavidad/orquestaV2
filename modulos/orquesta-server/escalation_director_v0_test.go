package orquestaserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func escalationDirectorTestRuntimeV0(
	t *testing.T,
	command []string,
	maxPerDay int,
	supervisor *fakeSupervisorV0,
	goalStore orquestagoal.GoalWorkStateStorePortV0,
	stopper *fakeGoalCooperativeStopperForTestV0,
	store *memoryStateStoreV0,
	now time.Time,
) *RuntimeV0 {
	t.Helper()
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                      t.TempDir(),
		TickInterval:                  time.Hour,
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           true,
		GoalObserverMaxItems:          5,
		ResidentDirectorEnabled:       false,
		AuditDisabled:                 true,
		EscalationDirectorEnabled:     true,
		EscalationDirectorCommand:     command,
		EscalationDirectorMaxPerDay:   maxPerDay,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStore,
		GoalStopper:    stopper,
		StateStore:     store,
		Clock:          fixedClockV0{now: now},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	return runtime
}

func escalationDirectorAnomalousSupervisorV0(
	state orquestagoal.GoalWorkStateV0,
	goalRef string,
	externalGoalRef string,
) *fakeSupervisorV0 {
	observation := orquestagoal.GoalWorkObserveActiveResultV0{
		Observations: []orquestagoal.GoalWorkObserveResultV0{{
			State: state,
			Result: orquestagoal.GoalWorkResultV0{
				Status:          orquestagoal.GoalStatusCompleteV0,
				GoalRef:         goalRef,
				ExternalGoalRef: externalGoalRef,
				Summary:         "MEJ-TASK-201 implementada: arnes determinista",
			},
		}},
	}
	return &fakeSupervisorV0{
		goalObservationResult: []orquestagoal.GoalWorkObserveActiveResultV0{observation, observation},
	}
}

func TestEscalationDirectorPendingIssuesV0IncluyeReviewYArtefactosParcialesObservadosV0(t *testing.T) {
	result := orquestagoal.GoalWorkObserveActiveResultV0{
		Issues: []orquestagoal.GoalWorkObserveActiveIssueV0{{
			RunRef:  " run-top ",
			GoalRef: " goal-top ",
			Code:    " review_result_payload_invalid_after_delivery ",
			Field:   " review_result ",
			Message: " retry_review ",
		}},
		Observations: []orquestagoal.GoalWorkObserveResultV0{{
			State: orquestagoal.GoalWorkStateV0{
				RunRef:  "run-observed",
				GoalRef: "goal-observed",
			},
			Result: orquestagoal.GoalWorkResultV0{
				Issues: []orquestagoal.GoalWorkIssueV0{
					{
						Code:   "partial_artifacts_written",
						Field:  "materialized_artifacts",
						Detail: "hay artefactos parciales",
					},
					{
						Code:   "observe_goal_failed",
						Field:  "run_ref",
						Detail: "este codigo ya tiene camino automatico",
					},
				},
			},
			Closure: orquestagoal.GoalClosureValidationV0{
				Issues: []orquestagoal.GoalWorkIssueV0{{
					Code:   "review_requires_human",
					Field:  "closure",
					Detail: "la revision humana no debe dejar al loop esperando",
				}},
			},
		}},
	}

	pending := escalationDirectorPendingIssuesV0(result)
	if len(pending) != 3 {
		t.Fatalf("pending=%+v want 3 issues escalables", pending)
	}
	assertEscalationDirectorIssueForTestV0(t, pending, "review_result_payload_invalid_after_delivery", "run-top", "goal-top")
	assertEscalationDirectorIssueForTestV0(t, pending, "partial_artifacts_written", "run-observed", "goal-observed")
	assertEscalationDirectorIssueForTestV0(t, pending, "review_requires_human", "run-observed", "goal-observed")
	for _, issue := range pending {
		if issue.Code == "observe_goal_failed" {
			t.Fatalf("codigo con accion automatica no debia escalarse: %+v", pending)
		}
	}
}

func assertEscalationDirectorIssueForTestV0(
	t *testing.T,
	issues []orquestagoal.GoalWorkObserveActiveIssueV0,
	code string,
	runRef string,
	goalRef string,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code &&
			issue.RunRef == runRef &&
			issue.GoalRef == goalRef {
			return
		}
	}
	t.Fatalf("issue code=%s run=%s goal=%s no encontrado en %+v", code, runRef, goalRef, issues)
}

func TestRuntimeV0EscalationDirectorAplicaStopYEsIdempotentePorFirmaV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 22, 0, 0, 0, time.UTC)
	const runRef = "run-ref-escalation-director-001"
	const goalRef = "goal-ref-autoprogramming-backlog-t292-escalation-001"
	const externalGoalRef = "external-goal-ref-escalation-director-001"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	invocationsLog := filepath.Join(t.TempDir(), "invocations.log")
	command := []string{
		"/bin/sh", "-c",
		`echo x >> ` + invocationsLog + `; echo '{"decision":"stop","reason":"cierre sin result durable"}'`,
	}
	supervisor := escalationDirectorAnomalousSupervisorV0(state, goalRef, externalGoalRef)
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	store := &memoryStateStoreV0{}
	runtime := escalationDirectorTestRuntimeV0(t, command, 0, supervisor, goalStore, stopper, store, now)
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))

	runtime.runGoalObservationTickV0(context.Background())
	runtime.ForgetGoalObservationFingerprintV0(runRef)
	runtime.runGoalObservationTickV0(context.Background())

	data, err := os.ReadFile(invocationsLog)
	if err != nil {
		t.Fatalf("ReadFile invocations: %v", err)
	}
	if invocations := strings.Count(string(data), "x"); invocations != 1 {
		t.Fatalf("el comando debia invocarse 1 vez (firma idempotente), fue %d", invocations)
	}
	if stopper.calls != 1 ||
		stopper.last.RunRef != runRef ||
		stopper.last.Reason != escalationDirectorStopReasonV0 ||
		stopper.last.RequestedBy != escalationDirectorRequestedByV0 {
		t.Fatalf("stopper calls=%d request=%+v", stopper.calls, stopper.last)
	}
	if store.last.EscalationDirectorLastDecision != escalationDirectorDecisionStopV0 ||
		store.last.EscalationDirectorInvocationsToday != 1 ||
		store.last.EscalationDirectorDay != "2026-07-03" ||
		store.last.EscalationDirectorLastReason != "cierre sin result durable" {
		t.Fatalf("estado escalation=%+v", store.last)
	}
}

func TestRuntimeV0EscalationDirectorRespetaPresupuestoDiarioV0(t *testing.T) {
	now := time.Date(2026, 7, 3, 22, 30, 0, 0, time.UTC)
	const runRef = "run-ref-escalation-director-002"
	const goalRef = "goal-ref-autoprogramming-backlog-t292-escalation-002"
	const externalGoalRef = "external-goal-ref-escalation-director-002"
	state := mustIdleSelfImprovementGoalStateForProgressTestV0(t, runRef, goalRef, externalGoalRef)
	goalStore := newMemoryGoalStateStoreV0()
	if err := goalStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	invocationsLog := filepath.Join(t.TempDir(), "invocations.log")
	command := []string{
		"/bin/sh", "-c",
		`echo x >> ` + invocationsLog + `; echo '{"decision":"defer","reason":"n/a"}'`,
	}
	supervisor := escalationDirectorAnomalousSupervisorV0(state, goalRef, externalGoalRef)
	stopper := &fakeGoalCooperativeStopperForTestV0{}
	store := &memoryStateStoreV0{}
	runtime := escalationDirectorTestRuntimeV0(t, command, 1, supervisor, goalStore, stopper, store, now)
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(idleSelfImprovementPreparedForProgressTestV0(state), now.Add(-3*time.Minute))
	runtime.tracker.MarkEscalationDirectorV0("2026-07-03", 1, "sha256:otra-firma-previa", "stop", "previa", now.Add(-time.Hour))

	runtime.runGoalObservationTickV0(context.Background())

	if _, err := os.Stat(invocationsLog); !os.IsNotExist(err) {
		t.Fatalf("el comando no debia invocarse con presupuesto agotado")
	}
	if store.last.EscalationDirectorLastDecision != "budget_exhausted" ||
		store.last.EscalationDirectorInvocationsToday != 1 {
		t.Fatalf("estado escalation=%+v", store.last)
	}
}

func TestParseEscalationDirectorDecisionV0(t *testing.T) {
	decision, err := parseEscalationDirectorDecisionV0(
		"ruido previo\n{\"decision\":\"Review_OK\",\"reason\":\"falsa alarma\"}\nruido posterior",
	)
	if err != nil || decision.Decision != escalationDirectorDecisionReviewOKV0 || decision.Reason != "falsa alarma" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if _, err := parseEscalationDirectorDecisionV0("sin json"); err == nil {
		t.Fatalf("salida sin JSON debia fallar")
	}
	if _, err := parseEscalationDirectorDecisionV0(`{"decision":"borrar_todo"}`); err == nil {
		t.Fatalf("decision fuera de la lista blanca debia fallar")
	}
}
