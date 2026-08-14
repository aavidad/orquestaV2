package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestConcurrentBudgetReservationsNeverExceedEnvelope(t *testing.T) {
	const contenders = 100
	system := newSQLiteV15System(t, 2)
	for index := 0; index < contenders; index++ {
		system.submit(t, fmt.Sprintf("request:v15-concurrent:%03d", index))
	}
	type result struct {
		claim application.ActionClaim
		found bool
		err   error
	}
	start := make(chan struct{})
	results := make(chan result, contenders)
	var group sync.WaitGroup
	for index := 0; index < contenders; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
				WorkerRef: "worker:v15-concurrent", Token: fmt.Sprintf("claim:v15-concurrent:%03d", index),
				LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
			})
			results <- result{claim: claim, found: found, err: err}
		}(index)
	}
	close(start)
	group.Wait()
	close(results)
	granted := 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent claim: %v", result.err)
		}
		if result.found {
			granted++
			if result.claim.BudgetReservation.Resources.ProcessSlots != 1 {
				t.Fatalf("reservation = %+v", result.claim.BudgetReservation)
			}
		}
	}
	if granted != 2 {
		t.Fatalf("granted = %d want 2", granted)
	}
	var reservations, slots, deferred, deliveryAttempts, effectAttempts int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*), COALESCE(SUM(process_slots), 0) FROM budget_reservations`).Scan(&reservations, &slots); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*), COALESCE(SUM(delivery_attempt), 0) FROM outbox
WHERE last_error_code = 'budget.temporarily_unavailable'`).Scan(&deferred, &deliveryAttempts); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&effectAttempts); err != nil {
		t.Fatal(err)
	}
	if reservations != 2 || slots != 2 || deferred != contenders-2 || deliveryAttempts != 0 || effectAttempts != 0 {
		t.Fatalf("reservations=%d slots=%d deferred=%d delivery_attempts=%d effect_attempts=%d",
			reservations, slots, deferred, deliveryAttempts, effectAttempts)
	}
}

func TestTemporaryQuotaParksActionWithoutTerminalFailure(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	first := system.submit(t, "request:v15-quota:first")
	second := system.submit(t, "request:v15-quota:second")
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15-quota", Token: "claim:v15-quota:first", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found || claim.BudgetReservationRef == "" {
		t.Fatalf("first claim = %+v found=%v err=%v", claim, found, err)
	}
	_, found, err = system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15-quota", Token: "claim:v15-quota:second", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || found {
		t.Fatalf("quota claim found=%v err=%v", found, err)
	}
	var state, executionState, errorCode string
	var attempts int
	if err := system.repository.db.QueryRow(`
SELECT goal.state, execution.state, action.last_error_code, action.delivery_attempt
FROM outbox action JOIN goals goal ON goal.ref = action.goal_ref
JOIN executions execution ON execution.ref = action.execution_ref
WHERE action.ref <> ?`, claim.Action.Ref).Scan(
		&state, &executionState, &errorCode, &attempts,
	); err != nil {
		t.Fatal(err)
	}
	if state != string(goal.GoalStateRunning) || executionState != string(application.ExecutionQueued) ||
		errorCode != "budget.temporarily_unavailable" || attempts != 0 {
		t.Fatalf("quota state=%s execution=%s code=%s attempts=%d", state, executionState, errorCode, attempts)
	}
	if first.Record.Goal.Ref() == second.Record.Goal.Ref() {
		t.Fatal("fixture goals collided")
	}
}

func TestHierarchicalFairnessBoundsProjectAndGoalStarvation(t *testing.T) {
	const perGoal = 25
	system := newSQLiteV15System(t, 3)
	system.policy.ProjectEnvelopeTemplate.Limit.ProcessSlots = 2
	system.policy.GoalEnvelopeTemplate.Limit.ProcessSlots = 1
	system.orchestrator = newSQLiteV15Orchestrator(
		t, system.repository, system.clock, system.external, system.policy, system.ids,
	)
	secondProject := mustRef(t, "project:v15-second", goal.NewProjectRef)
	secondOwner := testPrincipal(t, "principal:v15-second", "actor:v15-second", identity.PrincipalKindHuman)
	provisionTestAccess(t, system.repository, secondOwner, secondProject, identity.RoleProjectOwner, system.clock.Now())
	secondAccess, err := application.NewAccess(secondOwner, secondProject)
	sqliteTestNoError(t, err)
	goalProjects := make(map[goal.GoalRef]goal.ProjectRef, 4)
	firstAction := make(map[goal.GoalRef]string, 4)
	submit := func(access application.Access, project goal.ProjectRef, ref string) {
		plan := &application.PlanSpec{Phases: []application.PhaseSpec{{
			Ref: "phase-instance:" + ref, Key: "phase:" + ref, TemplateRef: "phase-template:fair",
		}}}
		for index := 0; index < perGoal; index++ {
			plan.WorkItems = append(plan.WorkItems, application.WorkItemSpec{
				Key: fmt.Sprintf("work:%02d", index), Objective: fmt.Sprintf("fair work %s/%02d", ref, index),
				Phase: "phase:" + ref, Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle,
			})
		}
		result, err := system.orchestrator.Submit(context.Background(), access, application.SubmitRequest{
			RequestRef: ref, Statement: "fair governed work " + ref, Confirm: true, Plan: plan,
		})
		sqliteTestNoError(t, err)
		goalRef := result.Record.Goal.Ref()
		goalProjects[goalRef] = project
		if err := system.repository.db.QueryRow(`SELECT MIN(ref) FROM outbox WHERE goal_ref=?`,
			goalRef.String()).Scan(new(string)); err != nil {
			t.Fatal(err)
		}
		var refValue string
		if err := system.repository.db.QueryRow(`SELECT MIN(ref) FROM outbox WHERE goal_ref=?`,
			goalRef.String()).Scan(&refValue); err != nil {
			t.Fatal(err)
		}
		firstAction[goalRef] = refValue
	}
	submit(system.access, system.project, "request:v15-fair:p1:g1")
	submit(system.access, system.project, "request:v15-fair:p1:g2")
	submit(secondAccess, secondProject, "request:v15-fair:p2:g1")
	submit(secondAccess, secondProject, "request:v15-fair:p2:g2")
	type result struct {
		claim application.ActionClaim
		found bool
		err   error
	}
	results := make(chan result, perGoal*4)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := 0; index < perGoal*4; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
				WorkerRef: "worker:v15-fair", Token: fmt.Sprintf("claim:v15-fair:%03d", index),
				LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
			})
			results <- result{claim: claim, found: found, err: err}
		}(index)
	}
	close(start)
	group.Wait()
	close(results)
	projectGrants := make(map[goal.ProjectRef]int)
	goalGrants := make(map[goal.GoalRef]int)
	var grantedClaims []application.ActionClaim
	granted := 0
	for outcome := range results {
		if outcome.err != nil {
			t.Fatalf("fair concurrent claim: %v", outcome.err)
		}
		if !outcome.found {
			continue
		}
		granted++
		grantedClaims = append(grantedClaims, outcome.claim)
		project := goalProjects[outcome.claim.Action.GoalRef]
		projectGrants[project]++
		goalGrants[outcome.claim.Action.GoalRef]++
		if outcome.claim.Action.Ref != firstAction[outcome.claim.Action.GoalRef] {
			t.Fatalf("Goal FIFO violated got=%s want=%s", outcome.claim.Action.Ref,
				firstAction[outcome.claim.Action.GoalRef])
		}
	}
	if granted != 3 || len(projectGrants) != 2 || len(goalGrants) != 3 ||
		projectGrants[system.project] == 0 || projectGrants[secondProject] == 0 ||
		projectGrants[system.project] > 2 || projectGrants[secondProject] > 2 {
		t.Fatalf("hierarchical grants=%d projects=%v goals=%v", granted, projectGrants, goalGrants)
	}
	for goalRef, count := range goalGrants {
		if count != 1 {
			t.Fatalf("Goal limit violated %s=%d", goalRef, count)
		}
	}
	var reservations, slots, deferred, deliveries, attempts int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*),SUM(process_slots) FROM budget_reservations`).Scan(
		&reservations, &slots,
	); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*),COALESCE(SUM(delivery_attempt),0) FROM outbox
WHERE last_error_code='budget.temporarily_unavailable'`).Scan(&deferred, &deliveries); err != nil {
		t.Fatal(err)
	}
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if reservations != 3 || slots != 3 || deferred != 97 || deliveries != 0 || attempts != 0 {
		t.Fatalf("hierarchical ledger reservations=%d slots=%d deferred=%d deliveries=%d attempts=%d",
			reservations, slots, deferred, deliveries, attempts)
	}
	var unserved goal.GoalRef
	for goalRef := range goalProjects {
		if goalGrants[goalRef] == 0 {
			unserved = goalRef
		}
	}
	released := grantedClaims[0]
	record, err := system.repository.GetGoal(context.Background(), released.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, released.Action.ExecutionRef)
	if !found {
		t.Fatal("released execution missing")
	}
	zero := governance.ResourceVector{Currency: released.BudgetReservation.Resources.Currency}
	settlement, err := governance.Reconcile(released.BudgetReservation, governance.ResourceUsage{
		Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	})
	sqliteTestNoError(t, err)
	settlement.SettledAt = system.clock.Now()
	execution.BudgetReservationRef, execution.EffectIntentRef, execution.LaunchReceiptRef = "", "", ""
	if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
		Claim: released, Execution: execution, AvailableAt: system.clock.Now().Add(2 * time.Second),
		ErrorCode: "test.fairness.release", OperationAt: system.clock.Now(),
		BudgetSettlement: &settlement, ClearEffectBinding: true,
	}); err != nil {
		t.Fatal(err)
	}
	restartSQLiteV15System(t, system)
	system.clock.Advance(time.Second)
	next, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15-fair:durable", Token: "claim:v15-fair:durable",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found || next.Action.GoalRef != unserved || next.Action.Ref != firstAction[unserved] {
		t.Fatalf("durable fairness claim=%+v found=%v err=%v unserved=%s", next, found, err, unserved)
	}
}

func TestSaturatedLaunchBacklogCannotStarveRunnableObservation(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	running := system.submit(t, "request:v15-observe-progress:running")
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-observe-progress:launch"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("seed running execution result=%+v err=%v", result, err)
	}

	for wave := 0; wave < 2; wave++ {
		for index := 0; index < 20; index++ {
			system.submit(t, fmt.Sprintf("request:v15-observe-progress:w%d:%02d", wave, index))
		}
		claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v15-observe-progress", Token: fmt.Sprintf("claim:v15-observe-progress:%d", wave),
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
		})
		if err != nil || !found || claim.Action.Kind != application.ActionObserveAgent ||
			claim.Action.GoalRef != running.Record.Goal.Ref() {
			t.Fatalf("wave %d observation progress claim=%+v found=%v err=%v", wave, claim, found, err)
		}
		if wave == 0 {
			record, readErr := system.repository.GetGoal(context.Background(), running.Record.Goal.Ref())
			if readErr != nil {
				t.Fatal(readErr)
			}
			if err := system.repository.RequeueAction(context.Background(), application.ActionRequeuedState{
				Claim: claim, Execution: record.Executions[0], AvailableAt: system.clock.Now(),
				ErrorCode: "test.observation_retry", OperationAt: system.clock.Now(),
			}); err != nil {
				t.Fatalf("requeue observation: %v", err)
			}
			system.clock.Advance(2 * time.Second)
		}
	}
	var deferred, attempts int
	if err := system.repository.db.QueryRow(`
SELECT COUNT(*), COALESCE(SUM(delivery_attempt),0) FROM outbox
WHERE kind='launch_agent' AND last_error_code='budget.temporarily_unavailable'`).Scan(&deferred, &attempts); err != nil {
		t.Fatal(err)
	}
	if deferred != 40 || attempts != 0 {
		t.Fatalf("saturated launch backlog deferred=%d delivery_attempts=%d", deferred, attempts)
	}
}

func TestSQLiteRoundTripsAllCriticalityEffortPairsWithoutTextInference(t *testing.T) {
	system := newSQLiteV15System(t, 70)
	criticalities := []governance.SecurityCriticality{
		governance.SecurityCriticalityNormal,
		governance.SecurityCriticalitySensitive,
		governance.SecurityCriticalityCritical,
	}
	efforts := []governance.ReasoningEffort{
		governance.ReasoningEffortLow,
		governance.ReasoningEffortMedium,
		governance.ReasoningEffortHigh,
		governance.ReasoningEffortXHigh,
	}
	plan := &application.PlanSpec{
		Phases: []application.PhaseSpec{{
			Ref: "phase-instance:v15-governance-matrix", Key: "phase:v15-governance-matrix",
			TemplateRef: "phase-template:v15-governance-matrix",
		}},
	}
	for criticalityIndex, criticality := range criticalities {
		for effortIndex, effort := range efforts {
			index := criticalityIndex*len(efforts) + effortIndex
			plan.WorkItems = append(plan.WorkItems, application.WorkItemSpec{
				Key: fmt.Sprintf("work:%02d", index),
				Objective: fmt.Sprintf(
					"adversarial critical xhigh provider-derived auto metadata must stay declared %02d", index,
				),
				Phase: "phase:v15-governance-matrix", Role: "role:worker",
				OutputContract:      goal.OutputContractEvidenceBundle,
				SecurityCriticality: criticality, ReasoningEffort: effort,
			})
		}
	}
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v15-governance-matrix", Statement: "persist explicit governance matrix",
		Confirm: true, Plan: plan,
	})
	if err != nil || !created.Created {
		t.Fatalf("submit governance matrix=%+v err=%v cause=%v", created, err, errors.Unwrap(err))
	}
	assertMatrix := func(record application.GoalRecord) {
		t.Helper()
		items := record.Goal.WorkItems()
		if len(items) != len(criticalities)*len(efforts) {
			t.Fatalf("governance matrix items=%d", len(items))
		}
		for index, item := range items {
			wantCriticality := criticalities[index/len(efforts)]
			wantEffort := efforts[index%len(efforts)]
			if item.SecurityCriticality() != wantCriticality || item.ReasoningEffort() != wantEffort ||
				item.Objective() != fmt.Sprintf("adversarial critical xhigh provider-derived auto metadata must stay declared %02d", index) {
				t.Fatalf("matrix item %d metadata=%q/%q objective=%q", index,
					item.SecurityCriticality(), item.ReasoningEffort(), item.Objective())
			}
		}
	}
	assertMatrix(created.Record)
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	persisted, err := restarted.GetGoal(context.Background(), created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	assertMatrix(persisted)
}

func TestSQLiteBudgetsEffectsRestartRaceAndReplay(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v15-restart-replay")
	first := claimSQLiteV15(t, system, "claim:v15-restart:first")
	system.clock.Advance(2 * time.Minute)
	second := claimSQLiteV15(t, system, "claim:v15-restart:second")
	if first.BudgetReservationRef != second.BudgetReservationRef ||
		first.BudgetReservation != second.BudgetReservation || first.Fence >= second.Fence {
		t.Fatalf("reservation replay first=%+v second=%+v", first, second)
	}
	prepareSQLiteV15Launch(t, system, second)
	attempt := sqliteV15Attempt(second, system.clock.Now())
	type attemptResult struct {
		value   application.EffectAttempt
		created bool
		err     error
	}
	start := make(chan struct{})
	results := make(chan attemptResult, 2)
	for index := 0; index < 2; index++ {
		go func() {
			<-start
			value, created, err := system.repository.RecordEffectAttempt(context.Background(),
				application.RecordEffectAttemptState{Claim: second, Attempt: attempt, OperationAt: system.clock.Now()})
			results <- attemptResult{value, created, err}
		}()
	}
	close(start)
	created := 0
	for index := 0; index < 2; index++ {
		result := <-results
		if result.err != nil || result.value != attempt {
			t.Fatalf("attempt race result=%+v", result)
		}
		if result.created {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("created attempts = %d", created)
	}
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	replayed, replayCreated, err := restarted.RecordEffectAttempt(context.Background(),
		application.RecordEffectAttemptState{Claim: second, Attempt: attempt, OperationAt: system.clock.Now()})
	if err != nil || replayCreated || replayed != attempt {
		t.Fatalf("restart replay=%+v created=%v err=%v", replayed, replayCreated, err)
	}
	var reservations, attempts int
	if err := restarted.db.QueryRow(`SELECT COUNT(*) FROM budget_reservations`).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if err := restarted.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if reservations != 1 || attempts != 1 {
		t.Fatalf("restart facts reservations=%d attempts=%d", reservations, attempts)
	}
}

func TestV15RestartReleasesPreAttemptReservationWhenApprovalExpires(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	created, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v15-expiring-approval", Statement: "explicit sensitive launch",
		Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v15-expiring", Key: "phase:v15-expiring", TemplateRef: "phase-template:v15-expiring",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "work", Objective: "sensitive launch", Phase: "phase:v15-expiring", Role: "role:worker",
				OutputContract:      goal.OutputContractEvidenceBundle,
				SecurityCriticality: governance.SecurityCriticalitySensitive,
				ReasoningEffort:     governance.ReasoningEffortMedium,
			}},
		},
	})
	sqliteTestNoError(t, err)
	intent := created.Record.EffectIntents[0]
	approved, err := system.orchestrator.DecideEffect(context.Background(), system.access, application.DecideEffectRequest{
		RequestRef: "approval:v15-expiring", GoalRef: created.Record.Goal.Ref(), IntentRef: intent.Ref,
		ExpectedIntentDigest: intent.Digest, Decision: application.EffectApproved, Reason: "bounded explicit approval",
	})
	if err != nil || !approved.Created {
		t.Fatalf("approve expiring intent=%+v err=%v", approved, err)
	}
	claim := claimSQLiteV15(t, system, "claim:v15-expiring:first")
	system.clock.Advance(2*time.Hour + 2*time.Minute)
	restartSQLiteV15System(t, system)
	_, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15-expiring:restart", Token: "claim:v15-expiring:restart",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
	})
	if err != nil || found {
		t.Fatalf("expired approval claim found=%v err=%v", found, err)
	}
	assertSQLiteV15StalePreAttemptReleased(t, system, created.Record.Goal.Ref(), claim)
}

func TestV15RestartReleasesRevokedPreAttemptButRetainsPostAttemptReservation(t *testing.T) {
	for _, withAttempt := range []bool{false, true} {
		t.Run(fmt.Sprintf("attempt_%v", withAttempt), func(t *testing.T) {
			system := newSQLiteV15System(t, 1)
			created := system.submit(t, fmt.Sprintf("request:v15-revoked:%v", withAttempt))
			claim := claimSQLiteV15(t, system, fmt.Sprintf("claim:v15-revoked:%v", withAttempt))
			if withAttempt {
				prepareSQLiteV15Launch(t, system, claim)
				attempt := sqliteV15Attempt(claim, system.clock.Now())
				if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
					Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
				}); err != nil {
					t.Fatal(err)
				}
			}
			revokeSQLiteV15Owner(t, system)
			system.clock.Advance(2 * time.Minute)
			restartSQLiteV15System(t, system)
			_, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
				WorkerRef: "worker:v15-revoked:restart", Token: fmt.Sprintf("claim:v15-revoked:restart:%v", withAttempt),
				LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy,
			})
			if err != nil || found {
				t.Fatalf("revoked authority claim found=%v err=%v", found, err)
			}
			record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
			sqliteTestNoError(t, err)
			if !withAttempt {
				assertSQLiteV15StalePreAttemptReleased(t, system, created.Record.Goal.Ref(), claim)
				return
			}
			if len(record.BudgetReservations) != 1 || len(record.BudgetSettlements) != 0 ||
				len(record.EffectAttempts) != 1 || record.Executions[0].BudgetReservationRef != claim.BudgetReservationRef ||
				record.Executions[0].EffectIntentRef != claim.Action.EffectIntentRef {
				t.Fatalf("post-attempt reservation was released record=%+v", record)
			}
			var code string
			var claimed int
			if err := system.repository.db.QueryRow(`
SELECT last_error_code, claim_token IS NOT NULL FROM outbox WHERE ref=?`, claim.Action.Ref).Scan(
				&code, &claimed,
			); err != nil {
				t.Fatal(err)
			}
			beforeCalls := system.external.launchCalls
			result, processErr := system.orchestrator.ProcessNext(context.Background(), "worker:v15-revoked:process")
			if code != "governance.effect_approval_required" || claimed != 0 || processErr != nil || result.Processed ||
				system.external.launchCalls != beforeCalls {
				t.Fatalf("post-attempt parking code=%q claimed=%d result=%+v err=%v calls=%d/%d",
					code, claimed, result, processErr, beforeCalls, system.external.launchCalls)
			}
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
				t.Fatalf("post-attempt stale admission recovery: %v cause=%v", err, errors.Unwrap(err))
			}
		})
	}
}

func assertSQLiteV15StalePreAttemptReleased(
	t *testing.T, system *sqliteV15System, goalRef goal.GoalRef, claim application.ActionClaim,
) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil || len(record.BudgetReservations) != 1 || len(record.BudgetSettlements) != 1 ||
		len(record.EffectAttempts) != 0 || record.Executions[0].BudgetReservationRef != "" ||
		record.Executions[0].EffectIntentRef != "" {
		t.Fatalf("stale pre-attempt release record=%+v err=%v", record, err)
	}
	settlement := record.BudgetSettlements[0]
	zero := governance.ResourceVector{Currency: settlement.Reserved.Currency}
	if settlement.ReservationRef != claim.BudgetReservationRef || settlement.Charged != zero ||
		settlement.Released != settlement.Reserved || settlement.Observed.Known != governance.AllResourceDimensions ||
		settlement.Observed.Quality != governance.UsageQualityExact {
		t.Fatalf("stale pre-attempt settlement=%+v", settlement)
	}
	var code string
	var claimed int
	if err := system.repository.db.QueryRow(`SELECT last_error_code,claim_token IS NOT NULL FROM outbox WHERE ref=?`,
		claim.Action.Ref).Scan(&code, &claimed); err != nil {
		t.Fatal(err)
	}
	if code != "governance.effect_approval_required" || claimed != 0 {
		t.Fatalf("parked stale action code=%q claimed=%d", code, claimed)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("stale pre-attempt recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func revokeSQLiteV15Owner(t *testing.T, system *sqliteV15System) {
	t.Helper()
	owner := testPrincipal(t, "principal:v15-owner", "actor:v15-owner", identity.PrincipalKindHuman)
	second := testPrincipal(t, "principal:v15-second-owner", "actor:v15-second-owner", identity.PrincipalKindHuman)
	grantTestMembership(t, system.repository, owner, second, system.project, identity.RoleProjectOwner,
		"membership:v15-second-owner", system.clock.Now())
	authorization := authorizeTest(t, system.repository, second, system.project,
		identity.PermissionProjectMembershipManage, owner.Ref.String(), "authorization:v15-revoke-owner", system.clock.Now())
	request := testRevokeRequest(t, "membership:v15-revoke-owner", second, owner.Ref, system.project, 1, system.clock.Now())
	_, _, changed, err := system.repository.RevokeMembership(context.Background(), application.MembershipRevokeState{
		AuthorizationReceipt: authorization, Request: request,
	})
	if err != nil || !changed {
		t.Fatalf("revoke V15 owner changed=%v err=%v", changed, err)
	}
}

func restartSQLiteV15System(t *testing.T, system *sqliteV15System) {
	t.Helper()
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.repository = restarted
	system.orchestrator = newSQLiteV15Orchestrator(t, restarted, system.clock, system.external, system.policy, system.ids)
}

func TestV15DefinitelyUnappliedRequeueRestartsWithFreshReservation(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v15-definitely-unapplied-restart")
	system.external.launchErr = sqliteV15DefinitelyUnapplied{}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-definite:first")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("definite rejection result=%+v err=%v", result, err)
	}
	requeued, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(requeued.BudgetReservations) != 1 || len(requeued.BudgetSettlements) != 1 ||
		len(requeued.EffectAttempts) != 1 || requeued.Executions[0].BudgetReservationRef != "" ||
		requeued.Executions[0].EffectIntentRef != "" {
		t.Fatalf("requeue ledger=%+v err=%v", requeued, err)
	}
	settlement := requeued.BudgetSettlements[0]
	zero := governance.ResourceVector{Currency: settlement.Reserved.Currency}
	if settlement.Charged != zero || settlement.Released != settlement.Reserved ||
		settlement.Observed.Known != governance.AllResourceDimensions ||
		settlement.Observed.Quality != governance.UsageQualityExact ||
		settlement.CausalAttemptRef != requeued.EffectAttempts[0].Ref {
		t.Fatalf("requeue settlement not exact release: %+v", settlement)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("requeue recovery: %v cause=%v", err, errors.Unwrap(err))
	}
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.repository = restarted
	system.orchestrator = newSQLiteV15Orchestrator(t, restarted, system.clock, system.external, system.policy, system.ids)
	system.external.mu.Lock()
	system.external.launchErr = nil
	system.external.mu.Unlock()
	system.clock.Advance(2 * time.Second)
	result, err = system.orchestrator.ProcessNext(context.Background(), "worker:v15-definite:retry")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("retry launch result=%+v err=%v", result, err)
	}
	after, err := restarted.GetGoal(context.Background(), created.Record.Goal.Ref())
	system.external.mu.Lock()
	physical, calls := len(system.external.launches), system.external.launchCalls
	system.external.mu.Unlock()
	if err != nil || physical != 1 || calls != 2 || len(after.BudgetReservations) != 2 ||
		len(after.BudgetSettlements) != 1 || len(after.EffectAttempts) != 2 || len(after.EffectReceipts) != 1 ||
		after.BudgetReservations[0].Ref == after.BudgetReservations[1].Ref ||
		after.Executions[0].BudgetReservationRef != after.BudgetReservations[1].Ref {
		t.Fatalf("retry facts physical=%d calls=%d record=%+v err=%v", physical, calls, after, err)
	}
}

func TestV15CancelClaimRaceSettlesUncalledReservation(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v15-cancel-claimed")
	claim := claimSQLiteV15(t, system, "claim:v15-cancel-claimed")
	record, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	result, err := system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: "request:v15-cancel-control", Operation: application.ControlCancel,
		Target: application.ControlTargetGoal, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		Reason: "cancel claimed launch before external call",
	})
	if err != nil || !result.Created || result.Control.Status != application.ControlConfirmed {
		t.Fatalf("cancel result=%+v err=%v", result, err)
	}
	system.external.mu.Lock()
	launches := len(system.external.launches)
	system.external.mu.Unlock()
	after, err := system.repository.GetGoal(context.Background(), record.Goal.Ref())
	if err != nil || launches != 0 || len(after.BudgetSettlements) != 1 {
		t.Fatalf("cancel ledger launches=%d settlements=%d err=%v", launches, len(after.BudgetSettlements), err)
	}
	settlement := after.BudgetSettlements[0]
	if settlement.ReservationRef != claim.BudgetReservationRef ||
		settlement.Observed.Known != governance.AllResourceDimensions ||
		settlement.Observed.Quality != governance.UsageQualityExact ||
		settlement.Charged.ProcessSlots != 0 || settlement.Released.ProcessSlots != 1 {
		t.Fatalf("released reservation = %+v", settlement)
	}
	system.submit(t, "request:v15-after-cancel")
	next := claimSQLiteV15(t, system, "claim:v15-after-cancel")
	if next.BudgetReservationRef == claim.BudgetReservationRef {
		t.Fatal("next logical action reused canceled reservation")
	}
}

func TestV15DispatchingCancelAndDefiniteLaunchRejectionSettleAtomically(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v15-cancel-dispatching")
	started, release := make(chan struct{}), make(chan struct{})
	system.external.mu.Lock()
	system.external.launchErr = sqliteV15DefinitelyUnapplied{}
	system.external.launchStart, system.external.launchGate = started, release
	system.external.mu.Unlock()

	type processed struct {
		result application.ProcessResult
		err    error
	}
	done := make(chan processed, 1)
	go func() {
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-cancel-dispatching")
		done <- processed{result: result, err: err}
	}()
	<-started
	dispatching, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || dispatching.Executions[0].State != application.ExecutionDispatching ||
		len(dispatching.EffectAttempts) != 1 || len(dispatching.BudgetSettlements) != 0 {
		t.Fatalf("prepared frontier record=%+v err=%v", dispatching, err)
	}
	control, err := system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: "request:v15-cancel-dispatching-control", Operation: application.ControlCancel,
		Target: application.ControlTargetGoal, GoalRef: dispatching.Goal.Ref(),
		ExpectedGoalRevision: dispatching.Goal.Revision(), ExpectedPlanGeneration: dispatching.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: dispatching.Goal.AppSpec().Generation(), ExpectedSpecHash: dispatching.Goal.SpecHash(),
		Reason: "cancel while launch adapter is crossing its boundary",
	})
	if err != nil || !control.Created || control.Control.Status != application.ControlRequested {
		t.Fatalf("pending cancel result=%+v err=%v", control, err)
	}
	close(release)
	processedLaunch := <-done
	if processedLaunch.err != nil || !processedLaunch.result.Processed ||
		processedLaunch.result.Action != application.ActionLaunchAgent {
		t.Fatalf("rejected launch result=%+v err=%v", processedLaunch.result, processedLaunch.err)
	}

	closed, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || closed.Goal.State() != goal.GoalStateCanceled ||
		closed.Executions[0].State != application.ExecutionFailed || len(closed.BudgetSettlements) != 1 ||
		len(closed.EffectAttempts) != 1 || len(closed.EffectReceipts) != 0 || len(closed.ConsumptionReceipts) != 2 {
		t.Fatalf("atomic close record=%+v err=%v", closed, err)
	}
	settlement := closed.BudgetSettlements[0]
	consumption := closed.ConsumptionReceipts[0]
	if consumption.Kind != application.ActionLaunchAgent {
		consumption = closed.ConsumptionReceipts[1]
	}
	if settlement.Observed.Known != governance.AllResourceDimensions ||
		settlement.Observed.Quality != governance.UsageQualityExact || settlement.Charged.ProcessSlots != 0 ||
		settlement.Released.ProcessSlots != 1 || consumption.ErrorCode != "agent.launch_definitely_not_applied" ||
		consumption.EffectReceiptRef != "" {
		t.Fatalf("definite rejection ledger settlement=%+v consumption=%+v", settlement, consumption)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected atomic close: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestV15MigrationParksLegacyEffectsWithoutRetroactiveApproval(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy-v14", "state.sqlite")
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatal(err)
	}
	database := openRawV10TestDatabase(t, path)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	if err := applyRecoveryMigrationPrefix(ctx, database, migrations[:recoverySchemaV14]); err != nil {
		t.Fatal(err)
	}
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC)}
	legacy := &Repository{db: database, path: path, now: clock.Now}
	project := mustRef(t, "project:v15-legacy", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:v15-legacy", "actor:v15-legacy", identity.PrincipalKindHuman)
	provisionTestAccess(t, legacy, owner, project, identity.RoleProjectOwner, clock.Now())
	policy := sqliteTestBudgetPolicy(clock.Now())
	state := newCreateFixture(
		t, "v15-legacy", "request:v15-legacy", "fingerprint:v15-legacy",
		owner.ActorRef.String(), project.String(),
	)
	state.RequestedBy = owner.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, legacy, owner, project, identity.PermissionGoalsCreate, project.String(),
		"authorization:v15-legacy:create", clock.Now(),
	)
	created, wasCreated, err := legacy.CreateGoal(ctx, state)
	if err != nil || !wasCreated {
		t.Fatalf("legacy create: created=%v err=%v cause=%v", wasCreated, err, errors.Unwrap(err))
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("migrate pre-010 state: %v cause=%v", err, errors.Unwrap(err))
	}
	t.Cleanup(func() { _ = migrated.Close() })
	claim, found, err := migrated.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:v15-legacy", Token: "claim:v15-legacy", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: policy,
	})
	if err != nil || found || claim.Action.Ref != "" {
		t.Fatalf("legacy claim=%+v found=%v err=%v", claim, found, err)
	}
	var active, quarantined, governanceVersion int
	var code string
	if err := migrated.db.QueryRow(`
SELECT completed_at IS NULL, quarantined_at IS NOT NULL, governance_version, last_error_code
FROM outbox WHERE goal_ref = ?`, created.Goal.Ref().String()).Scan(
		&active, &quarantined, &governanceVersion, &code,
	); err != nil {
		t.Fatal(err)
	}
	persisted, err := migrated.GetGoal(ctx, created.Goal.Ref())
	if err != nil || active != 1 || quarantined != 0 || governanceVersion != 0 ||
		code != "governance.legacy_reauthorization_required" || persisted.Goal.State() != goal.GoalStateRunning ||
		len(persisted.EffectIntents) != 0 || len(persisted.ConsumptionReceipts) != 0 {
		t.Fatalf("legacy evidence active=%d quarantine=%d governance=%d code=%s record=%+v err=%v",
			active, quarantined, governanceVersion, code, persisted, err)
	}
}

func TestV15MigratedV14RunningParentDoesNotInventObservationOrStopAuthority(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy-v14-multi", "state.sqlite")
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatal(err)
	}
	database := openRawV10TestDatabase(t, path)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	if err := applyRecoveryMigrationPrefix(ctx, database, migrations[:recoverySchemaV14]); err != nil {
		t.Fatal(err)
	}
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 18, 7, 30, 0, 0, time.UTC)}
	legacy := &Repository{db: database, path: path, now: clock.Now}
	project := mustRef(t, "project:v15-legacy-multi", goal.NewProjectRef)
	owner := testPrincipal(t, "principal:v15-legacy-multi", "actor:v15-legacy-multi", identity.PrincipalKindHuman)
	provisionTestAccess(t, legacy, owner, project, identity.RoleProjectOwner, clock.Now())
	access, err := application.NewAccess(owner, project)
	sqliteTestNoError(t, err)
	policy := sqliteTestBudgetPolicy(clock.Now())
	external := newSQLiteV15External(clock)
	ids := &sqliteV15IDs{}
	orchestrator := newSQLiteV15Orchestrator(t, legacy, clock, external, policy, ids)
	created, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:v15-legacy-multi", Statement: "finish legacy parent and park successor", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v15-legacy-multi", Key: "phase:v15-legacy-multi",
				TemplateRef: "phase-template:v15-legacy-multi",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "parent", Objective: "legacy running parent", Phase: "phase:v15-legacy-multi", Role: "role:worker", OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "child", Objective: "legacy parked child", Phase: "phase:v15-legacy-multi", Role: "role:reviewer", Dependencies: []string{"parent"}, OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	if err != nil && !application.IsStateError(err, application.StateConflict) {
		t.Fatal(err)
	}
	legacyGoalRef := created.Record.Goal.Ref()
	if legacyGoalRef.String() == "" {
		var value string
		if err := database.QueryRow(`SELECT ref FROM goals WHERE project_ref=?`, project.String()).Scan(&value); err != nil {
			t.Fatal(err)
		}
		legacyGoalRef = mustRef(t, value, goal.NewGoalRef)
	}
	record, err := legacy.GetGoal(ctx, legacyGoalRef)
	sqliteTestNoError(t, err)
	parent := record.Goal.WorkItems()[0]
	execution := record.Executions[0]
	claim := application.ActionClaim{
		Action: application.ActionRecord{
			Ref: "action:launch:" + execution.Ref.String(), Kind: application.ActionLaunchAgent,
			GoalRef: record.Goal.Ref(), WorkItemRef: parent.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: record.Goal.PlanGeneration(), WorkItemGeneration: parent.Revision(),
			AvailableAt: record.Goal.CreatedAt(),
		},
		Token: "claim:v15-legacy-parent", WorkerRef: "worker:v15-legacy-parent",
		DeliveryAttempt: 1, Fence: 1, LeaseUntil: clock.Now().Add(time.Minute),
	}
	if _, err := database.Exec(`
INSERT INTO work_item_fences(goal_ref,work_item_ref,fence) VALUES(?,?,1)`,
		record.Goal.Ref().String(), parent.Ref().String(),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
UPDATE outbox SET claim_token=?,claimed_by=?,claimed_until=?,delivery_attempt=1,fence=1
WHERE ref=?`, claim.Token, claim.WorkerRef, requiredTime(claim.LeaseUntil), claim.Action.Ref); err != nil {
		t.Fatal(err)
	}
	started, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), parent.Revision(), parent.Ref(), claim.Action.ExecutionRef, clock.Now(),
	)
	sqliteTestNoError(t, err)
	execution.State = application.ExecutionDispatching
	if err := legacy.RecordLaunchPrepared(ctx, application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: started, Execution: execution,
		OperationAt: clock.Now(), Event: application.EventRecord{
			Ref: "event:v15-legacy-parent-dispatching", Kind: "execution.dispatching", GoalRef: started.Ref(),
			WorkItemRef: parent.Ref(), ExecutionRef: execution.Ref, OccurredAt: clock.Now(),
		},
	}); err != nil {
		t.Fatalf("prepare V14 parent: %v cause=%v", err, errors.Unwrap(err))
	}
	execution.State = application.ExecutionRunning
	execution.ProviderRef, execution.ModelRef, execution.AgentRef = "provider:codex", "model:codex", "agent:codex"
	execution.ExternalRef = "external:" + execution.Ref.String()
	execution.StartedAt, execution.ProviderAcceptedAt = clock.Now(), clock.Now()
	execution.DeadlineAt = clock.Now().Add(time.Hour)
	startedParent, _ := started.WorkItem(parent.Ref())
	if err := legacy.RecordLaunchAccepted(ctx, application.LaunchAcceptedState{
		Claim: claim, Execution: execution, OperationAt: clock.Now(),
		NextAction: application.ActionRecord{
			Ref: "action:observe:" + execution.Ref.String(), Kind: application.ActionObserveAgent,
			GoalRef: started.Ref(), WorkItemRef: parent.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: started.PlanGeneration(), WorkItemGeneration: startedParent.Revision(), AvailableAt: clock.Now(),
		},
		Event: application.EventRecord{
			Ref: "event:v15-legacy-parent-accepted", Kind: "execution.accepted", GoalRef: started.Ref(),
			WorkItemRef: parent.Ref(), ExecutionRef: execution.Ref, OccurredAt: clock.Now(),
		},
	}); err != nil {
		t.Fatalf("persist V14 running parent: %v cause=%v", err, errors.Unwrap(err))
	}
	running, err := legacy.GetGoal(ctx, legacyGoalRef)
	sqliteTestNoError(t, err)
	controlResult, err := orchestrator.Control(ctx, access, application.ControlRequest{
		RequestRef: "control:v15-legacy-running-stop", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: parent.Ref(), ExpectedWorkItemRevision: running.Goal.WorkItems()[0].Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "legacy stop must not deadlock observation",
	})
	if err != nil || controlResult.Control.Status != application.ControlRequested {
		t.Fatalf("request V14 stop result=%+v err=%v", controlResult, err)
	}
	external.launches[execution.IdempotencyKey] = ports.AgentLaunchReceipt{
		ExecutionRef: execution.Ref, SpecHash: execution.SpecHash,
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := Open(ctx, Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now})
	if err != nil {
		t.Fatalf("migrate V14 multi: %v cause=%v", err, errors.Unwrap(err))
	}
	t.Cleanup(func() { _ = migrated.Close() })
	migratedOrchestrator := newSQLiteV15Orchestrator(t, migrated, clock, external, policy, ids)
	result, err := migratedOrchestrator.ProcessNext(ctx, "worker:v15-legacy-observe")
	if err == nil || err.Error() != "application.agent_observation_authority_invalid" ||
		!result.Processed || result.Action != application.ActionObserveAgent || external.observeCalls != 0 {
		t.Fatalf("migrated observation authority result=%+v observe_calls=%d err=%v", result, external.observeCalls, err)
	}
	after, err := migrated.GetGoal(ctx, legacyGoalRef)
	sqliteTestNoError(t, err)
	items := after.Goal.WorkItems()
	if after.Goal.State() != goal.GoalStateRunning || len(items) != 2 ||
		items[0].State() != goal.WorkItemStateRunning || items[1].State() != goal.WorkItemStatePending ||
		len(after.Executions) != 1 || len(after.WorkItemAuthorities) != 0 || len(after.EffectIntents) != 0 ||
		len(after.EffectApprovals) != 0 || len(after.EffectAttempts) != 0 || len(after.EffectReceipts) != 0 ||
		len(after.Controls) != 1 || after.Controls[0].Status != application.ControlRequested {
		t.Fatalf("migrated authority was synthesized or lifecycle changed: %+v", after)
	}
	result, err = migratedOrchestrator.ProcessNext(ctx, "worker:v15-legacy-local-stop")
	if err != nil || result.Processed || external.stopCalls != 0 {
		t.Fatalf("parked migrated stop result=%+v calls=%d err=%v", result, external.stopCalls, err)
	}
	after, err = migrated.GetGoal(ctx, legacyGoalRef)
	if err != nil || after.Controls[0].Status != application.ControlRequested || after.Controls[0].ReceiptRef != "" {
		t.Fatalf("V14 stop invented settlement record=%+v err=%v", after, err)
	}
	var active, quarantined int
	if err := migrated.db.QueryRow(`
SELECT COUNT(*) FILTER (WHERE completed_at IS NULL),
       COUNT(*) FILTER (WHERE quarantined_at IS NOT NULL)
FROM outbox WHERE goal_ref=?`, legacyGoalRef.String()).Scan(&active, &quarantined); err != nil {
		t.Fatal(err)
	}
	if active != 1 || quarantined != 1 {
		t.Fatalf("legacy authority actions active=%d quarantined=%d", active, quarantined)
	}
	var stopCode string
	if err := migrated.db.QueryRow(`SELECT last_error_code FROM outbox WHERE goal_ref=? AND kind='stop_agent'`,
		legacyGoalRef.String()).Scan(&stopCode); err != nil || stopCode != "governance.legacy_reauthorization_required" {
		t.Fatalf("legacy stop parking code=%q err=%v", stopCode, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, migrated.db); err != nil {
		t.Fatalf("migrated quarantined authority recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestV15RecoveryRejectsBudgetEffectCausalTampering(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v15-tamper")
	claim := claimSQLiteV15(t, system, "claim:v15-tamper")
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, _, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid V15 recovery: %v cause=%v", err, errors.Unwrap(err))
	}
	rewriteRecoveryTrigger(t, system.repository.db, "budget_reservations_immutable_update", func() {
		mustV10Exec(t, system.repository.db, `
UPDATE budget_reservations SET policy_hash = ? WHERE ref = ?`,
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			claim.BudgetReservationRef,
		)
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v15_budget_reservation_invalid") {
		t.Fatalf("causal tamper accepted: %v", err)
	}
}

func TestV15BackupRestorePreservesBudgetsAndEffects(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	created := system.submit(t, "request:v15-backup")
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-backup"); err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch result=%+v err=%v", result, err)
	}
	system.clock.Advance(2 * time.Second)
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v15-backup"); err != nil || !result.Processed || result.Action != application.ActionObserveAgent {
		t.Fatalf("observe result=%+v err=%v", result, err)
	}
	before, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || len(before.BudgetSettlements) != 1 || len(before.EffectReceipts) != 1 {
		t.Fatalf("closed governed record=%+v err=%v", before, err)
	}
	recovery, _, _ := newV09TestRecovery(t, system.repository, system.clock.Now(), nil)
	backup, err := recovery.CreateBackup(context.Background())
	if err != nil {
		t.Fatalf("V15 backup: %v cause=%v", err, errors.Unwrap(err))
	}
	if _, err := recovery.VerifyBackup(context.Background(), backup.Ref); err != nil {
		t.Fatal(err)
	}
	target, _ := application.NewRecoveryTargetRef("recovery-target:v15-budget-effects")
	if _, err := recovery.RestoreBackup(context.Background(), backup.Ref, target); err != nil {
		t.Fatal(err)
	}
	path, _ := recovery.TargetPath(target)
	restored, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: system.clock.Now,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restored.Close() })
	after, err := restored.GetGoal(context.Background(), created.Record.Goal.Ref())
	if err != nil || !reflect.DeepEqual(before.BudgetEnvelopes, after.BudgetEnvelopes) ||
		!reflect.DeepEqual(before.BudgetReservations, after.BudgetReservations) ||
		!reflect.DeepEqual(before.BudgetSettlements, after.BudgetSettlements) ||
		!reflect.DeepEqual(before.EffectIntents, after.EffectIntents) ||
		!reflect.DeepEqual(before.EffectApprovals, after.EffectApprovals) ||
		!reflect.DeepEqual(before.EffectAttempts, after.EffectAttempts) ||
		!reflect.DeepEqual(before.EffectReceipts, after.EffectReceipts) {
		t.Fatalf("V15 restore differs: err=%v\nbefore=%+v\nafter=%+v", err, before, after)
	}
}

func claimSQLiteV15(t *testing.T, system *sqliteV15System, token string) application.ActionClaim {
	t.Helper()
	claim, found, err := system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v15", Token: token, LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found {
		t.Fatalf("claim %s found=%v err=%v", token, found, err)
	}
	return claim
}

func sqliteV15Attempt(claim application.ActionClaim, at time.Time) application.EffectAttempt {
	intent := claim.Action.EffectIntent
	return application.EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: intent.Ref, IntentDigest: intent.Digest, ApprovalRef: claim.EffectApproval.Ref,
		Subject: intent.Subject, ActionRef: claim.Action.Ref, ActionFence: claim.Fence,
		WorkerRef: claim.WorkerRef, IdempotencyKey: intent.IdempotencyKey, StartedAt: at.UTC(),
		ClaimLeaseUntil: claim.LeaseUntil.UTC(),
	}
}
