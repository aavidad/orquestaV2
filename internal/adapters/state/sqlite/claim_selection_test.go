package sqlite

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestClaimSelectionDefaultIncludesLaunchAcrossRestart(t *testing.T) {
	repository, path := openTestRepository(t)
	seedClaimSelectionCandidate(t, repository, "default-a", application.ActionLaunchAgent)
	seedClaimSelectionCandidate(t, repository, "default-b", application.ActionLaunchAgent)

	now := time.Date(2026, 7, 14, 12, 0, 0, 123456789, time.UTC)
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:selection-default-explicit",
		Token:     "claim:selection-default-explicit", LeaseDuration: time.Minute,
		ExcludeLaunch: false, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found || claim.Action.Kind != application.ActionLaunchAgent ||
		claim.Action.Ref != "action:launch:selection-default-a" ||
		claim.DeliveryAttempt != 1 || claim.Fence != 1 || !claim.LeaseUntil.Equal(now.Add(time.Minute)) {
		t.Fatalf("explicit default claim = %+v found=%v err=%v", claim, found, err)
	}

	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("restart repository: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	claim, found, err = repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:selection-default-zero",
		Token:     "claim:selection-default-zero", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found || claim.Action.Kind != application.ActionLaunchAgent ||
		claim.Action.Ref != "action:launch:selection-default-b" ||
		claim.DeliveryAttempt != 1 || claim.Fence != 1 || !claim.LeaseUntil.Equal(now.Add(time.Minute)) {
		t.Fatalf("zero-value default after restart = %+v found=%v err=%v", claim, found, err)
	}
}

func TestClaimSelectionExcludeLaunchPreservesStopAndObserveProgress(t *testing.T) {
	system := newSQLiteV15System(t, 100)
	ctx := context.Background()
	observeSubmission := system.submit(t, "request:claim-selection-observe")
	if result, err := system.orchestrator.ProcessNext(ctx, "worker:claim-selection-observe-launch"); err != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch observe fixture: result=%+v err=%v", result, err)
	}
	stopSubmission := system.submit(t, "request:claim-selection-stop")
	if result, err := system.orchestrator.ProcessNext(ctx, "worker:claim-selection-stop-launch"); err != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch stop fixture: result=%+v err=%v", result, err)
	}
	runningStop, err := system.repository.GetGoal(ctx, stopSubmission.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	stopItem, stopExecution := runningStop.Goal.WorkItems()[0], runningStop.Executions[0]
	if _, err := system.orchestrator.Control(ctx, system.access, application.ControlRequest{
		RequestRef: "control:claim-selection-stop", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: runningStop.Goal.Ref(),
		ExpectedGoalRevision: runningStop.Goal.Revision(), ExpectedPlanGeneration: runningStop.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: runningStop.Goal.AppSpec().Generation(), ExpectedSpecHash: runningStop.Goal.SpecHash(),
		WorkItemRef: stopItem.Ref(), ExpectedWorkItemRevision: stopItem.Revision(),
		ExecutionRef: stopExecution.Ref, ExpectedExecutionAttempt: stopExecution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "prove stop priority while launch selection is excluded",
	}); err != nil {
		t.Fatalf("request stop fixture: %v", err)
	}
	const launchWave = 73
	for index := 0; index < launchWave; index++ {
		system.submit(t, fmt.Sprintf("request:claim-selection-wave:%03d", index))
	}

	request := application.ClaimRequest{
		WorkerRef: "worker:selection-without-launch",
		Token:     "claim:selection-without-launch-stop", LeaseDuration: time.Minute,
		ExcludeLaunch: true, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
	}
	claim, found, err := system.repository.ClaimNextAction(context.Background(), request)
	if err != nil || !found || claim.Action.Kind != application.ActionStopAgent ||
		claim.Action.ExecutionRef != stopExecution.Ref {
		t.Fatalf("stop-priority claim = %+v found=%v err=%v", claim, found, err)
	}

	stats := &claimWindowQueryStats{}
	request.Token = "claim:selection-without-launch-observe"
	claim, found, err = system.repository.ClaimNextAction(claimWindowStatsContext(stats), request)
	if err != nil || !found || claim.Action.Kind != application.ActionObserveAgent ||
		claim.Action.ExecutionRef != observeSubmission.Record.Executions[0].Ref {
		t.Fatalf("observe after launch wave = %+v found=%v err=%v", claim, found, err)
	}
	stats.assertBounded(t, 1, 1)

	before := readExcludedLaunchMutationState(t, system.repository)
	request.Token = "claim:selection-without-launch-empty"
	claim, found, err = system.repository.ClaimNextAction(context.Background(), request)
	if err != nil || found || claim.Action.Ref != "" {
		t.Fatalf("launch-only backlog claim = %+v found=%v err=%v", claim, found, err)
	}
	after := readExcludedLaunchMutationState(t, system.repository)
	if after != before {
		t.Fatalf("excluded launches mutated: before=%+v after=%+v", before, after)
	}
	if after.launches != launchWave || after.claimed != 0 || after.deliveryAttempts != 0 ||
		after.reservations != 0 || after.effectAttempts != 0 || after.lastErrors != 0 {
		t.Fatalf("excluded launch state = %+v", after)
	}
}

func TestClaimSelectionExcludeLaunchFilterGuardsWindowAndKeyset(t *testing.T) {
	const filter = `AND (? = 0 OR o.kind <> 'launch_agent' OR EXISTS (
      SELECT 1 FROM effect_attempts recovery_attempt
      WHERE recovery_attempt.action_ref=o.ref
        AND recovery_attempt.intent_ref=o.effect_intent_ref
  ))`
	if claimCandidateWindowSize != 16 {
		t.Fatalf("candidate window = %d, want 16", claimCandidateWindowSize)
	}
	if strings.Count(claimCandidatesQuery, filter) != 1 {
		t.Fatalf("launch exclusion filter count = %d, want 1", strings.Count(claimCandidatesQuery, filter))
	}
	filterAt := strings.Index(claimCandidatesQuery, filter)
	keysetAt := strings.Index(claimCandidatesQuery, "AND (? = 0 OR (")
	orderAt := strings.LastIndex(claimCandidatesQuery, "ORDER BY CASE")
	limitAt := strings.LastIndex(claimCandidatesQuery, "LIMIT ?")
	if filterAt < 0 || keysetAt < 0 || orderAt < 0 || limitAt < 0 ||
		filterAt >= keysetAt || keysetAt >= orderAt || orderAt >= limitAt {
		t.Fatalf("filter/keyset/order/limit order = %d/%d/%d/%d", filterAt, keysetAt, orderAt, limitAt)
	}
}

func seedClaimSelectionCandidate(
	t *testing.T,
	repository *Repository,
	suffix string,
	kind application.ActionKind,
) {
	t.Helper()
	fixtureSuffix := "selection-" + suffix
	state := newCreateFixture(
		t, fixtureSuffix, "request:"+fixtureSuffix, "fingerprint:"+fixtureSuffix,
		"actor:claim-selection", "project:claim-selection",
	)
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create %s candidate: %v", kind, err)
	}
	if kind != application.ActionLaunchAgent {
		t.Fatalf("unsupported queued claim selection candidate kind %s", kind)
	}
}

type excludedLaunchMutationState struct {
	launches, claimed, deliveryAttempts, reservations int
	fairnessCursors, effectAttempts, effectApprovals  int
	lastErrors                                        int
}

func readExcludedLaunchMutationState(t *testing.T, repository *Repository) excludedLaunchMutationState {
	t.Helper()
	var state excludedLaunchMutationState
	if err := repository.db.QueryRow(`
SELECT COUNT(*),
       COALESCE(SUM(CASE WHEN claim_token IS NOT NULL THEN 1 ELSE 0 END), 0),
       COALESCE(SUM(delivery_attempt), 0),
       COALESCE(SUM(CASE WHEN last_error_code<>'' THEN 1 ELSE 0 END), 0)
FROM outbox
WHERE kind='launch_agent' AND completed_at IS NULL`).Scan(
		&state.launches, &state.claimed, &state.deliveryAttempts, &state.lastErrors,
	); err != nil {
		t.Fatal(err)
	}
	for query, destination := range map[string]*int{
		`SELECT COUNT(*) FROM budget_reservations reservation
JOIN outbox action ON action.ref=reservation.action_ref
WHERE action.kind='launch_agent' AND action.completed_at IS NULL`: &state.reservations,
		`SELECT COUNT(*) FROM fairness_cursors`: &state.fairnessCursors,
		`SELECT COUNT(*) FROM effect_attempts attempt
JOIN outbox action ON action.ref=attempt.action_ref
WHERE action.kind='launch_agent' AND action.completed_at IS NULL`: &state.effectAttempts,
		`SELECT COUNT(*) FROM effect_approvals approval
JOIN effect_intents intent ON intent.ref=approval.intent_ref
JOIN outbox action ON action.ref=intent.action_ref
WHERE action.kind='launch_agent' AND action.completed_at IS NULL`: &state.effectApprovals,
	} {
		if err := repository.db.QueryRow(query).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	return state
}
