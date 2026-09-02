package sqlite

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestAgentEnvironmentLifecycleStoreQPCSurvivesRestartAndExpiredLease(t *testing.T) {
	ctx := context.Background()
	fixture := newSQLiteAgentEnvironmentLifecycleFixture(t)
	for name, mutate := range map[string]func(*application.AgentEnvironmentLifecycleSnapshot){
		"launch receipt": func(snapshot *application.AgentEnvironmentLifecycleSnapshot) {
			snapshot.LaunchReceiptRef = "physical-receipt:crossed-launch"
		},
		"physical token": func(snapshot *application.AgentEnvironmentLifecycleSnapshot) {
			snapshot.Token = lifecycleStoreToken(
				t, "external:lifecycle-crossed", "active-crossed", snapshot.Token.Fence.String(),
				ports.AgentEnvironmentActive,
			)
		},
	} {
		t.Run("initial crossed "+name, func(t *testing.T) {
			candidate := fixture.initial("active-crossed-candidate", fixture.system.clock.Now())
			mutate(&candidate)
			if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleInitial(
				ctx, application.AgentEnvironmentLifecycleInitialState{
					Snapshot: candidate, OperationAt: candidate.RecordedAt,
				},
			); err == nil || written {
				t.Fatalf("crossed initial written=%v err=%v", written, err)
			}
		})
	}
	var lifecycleRows int
	if err := fixture.system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM agent_environment_lifecycles`,
	).Scan(&lifecycleRows); err != nil || lifecycleRows != 0 {
		t.Fatalf("crossed initial left rows=%d err=%v", lifecycleRows, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err != nil {
		t.Fatalf("crossed initial dirtied recovery: %s", sqliteTestErrorChain(err))
	}
	if err := fixture.system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	fixture.system.repository = openSQLiteV15Repository(t, fixture.system.path, fixture.system.clock.Now)

	// Two distinct physical observations race to create revision 1. Exactly one
	// becomes authority; both callers receive that durable winner.
	initials := []application.AgentEnvironmentLifecycleInitialState{
		{Snapshot: fixture.initial("active-a", fixture.system.clock.Now()), OperationAt: fixture.system.clock.Now()},
		{Snapshot: fixture.initial("active-b", fixture.system.clock.Now().Add(time.Nanosecond)), OperationAt: fixture.system.clock.Now().Add(time.Nanosecond)},
	}
	type initialResult struct {
		snapshot application.AgentEnvironmentLifecycleSnapshot
		written  bool
		err      error
	}
	start := make(chan struct{})
	results := make(chan initialResult, len(initials))
	var group sync.WaitGroup
	for _, candidate := range initials {
		candidate := candidate
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			snapshot, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleInitial(ctx, candidate)
			results <- initialResult{snapshot: snapshot, written: written, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(results)
	created := 0
	var winner application.AgentEnvironmentLifecycleSnapshot
	var returned []application.AgentEnvironmentLifecycleSnapshot
	for result := range results {
		if result.err != nil {
			t.Fatalf("initial CAS: %s", sqliteTestErrorChain(result.err))
		}
		if result.written {
			created++
			winner = result.snapshot
		}
		returned = append(returned, result.snapshot)
	}
	if created != 1 {
		t.Fatalf("initial CAS writers=%d want=1", created)
	}
	for _, snapshot := range returned {
		if snapshot != winner {
			t.Fatalf("initial CAS loser did not receive winner: got=%+v want=%+v", snapshot, winner)
		}
	}
	stored, found, err := fixture.system.repository.GetAgentEnvironmentLifecycle(ctx, fixture.execution.Ref)
	if err != nil || !found || stored.Snapshot != winner {
		t.Fatalf("initial winner found=%v got=%+v want=%+v err=%v", found, stored.Snapshot, winner, err)
	}

	fixture.system.clock.Advance(time.Second)
	quiesce := fixture.addAndClaim(t, winner, application.ActionQuiesceAgent, "claim:lifecycle:q")
	preparedQ, err := application.PrepareAgentEnvironmentLifecycleEffect(winner, quiesce, fixture.system.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	// The same pre-effect CAS may be submitted concurrently, but only one
	// append can create the EffectAttempt.
	type attemptResult struct {
		written bool
		err     error
	}
	attempts := make(chan attemptResult, 2)
	start = make(chan struct{})
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleAttempt(ctx, preparedQ)
			attempts <- attemptResult{written: written, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(attempts)
	attemptWriters := 0
	for result := range attempts {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.written {
			attemptWriters++
		}
	}
	if attemptWriters != 1 {
		t.Fatalf("attempt CAS writers=%d want=1", attemptWriters)
	}

	// Restart with the original claim lease expired. Terminal persistence uses
	// the historical attempted authority and must not require a new live claim.
	if err := fixture.system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	fixture.system.clock.Advance(2 * time.Minute)
	fixture.system.repository = openSQLiteV15Repository(t, fixture.system.path, fixture.system.clock.Now)
	pending, found, err := fixture.system.repository.GetAgentEnvironmentLifecycle(ctx, fixture.execution.Ref)
	if err != nil || !found || !pending.Snapshot.Effect.NeedsReconciliation() || pending.Attempt != preparedQ.Attempt {
		t.Fatalf("restart pending found=%v state=%+v err=%v", found, pending, err)
	}
	replacement, found, err := fixture.system.repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:lifecycle-restart", Token: "claim:lifecycle:q-replacement",
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: fixture.system.policy,
		CapacityCandidates: fixture.system.capacidad,
	})
	if err != nil || !found || replacement.Action.Ref != quiesce.Action.Ref ||
		replacement.Token == quiesce.Token || replacement.Fence <= quiesce.Fence {
		t.Fatalf("replacement claim=%+v original=%+v found=%v err=%v", replacement, quiesce, found, err)
	}

	quiesced := fixture.nextToken(t, pending.Snapshot.Token, "quiesced", ports.AgentEnvironmentQuiesced)
	outcomeQ, err := application.RecordAgentEnvironmentQuiesceOutcome(preparedQ, ports.AgentQuiesceReceipt{
		Subject: preparedQ.Snapshot.Subject, PreviousToken: preparedQ.Snapshot.Token, NextToken: quiesced,
		IdempotencyKey: preparedQ.Attempt.IdempotencyKey, ReceiptRef: "physical-receipt:lifecycle:q",
		ConfirmedAt: fixture.system.clock.Now(),
	}, fixture.system.clock.Now())
	if err != nil || outcomeQ.Terminal == nil {
		t.Fatalf("quiesce outcome=%+v err=%v", outcomeQ, err)
	}
	postQ := *outcomeQ.Terminal
	record := fixture.record(t)
	nextP, err := (&application.Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		record, postQ.Snapshot, application.ActionPreserveAgentEnvironment, postQ.OperationAt)
	if err != nil {
		t.Fatal(err)
	}
	postQ.NextAction = &nextP
	badQ := postQ
	badQ.Snapshot.Effect.PhysicalReceipt = "physical-receipt:lifecycle:q-crossed"
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, badQ); err == nil || written {
		t.Fatalf("crossed Q terminal written=%v err=%v", written, err)
	}
	var qReceipts, qConsumptions, pActions int
	if err := fixture.system.repository.db.QueryRow(`
SELECT (SELECT COUNT(*) FROM effect_receipts WHERE action_ref=?),
       (SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref=?),
       (SELECT COUNT(*) FROM outbox WHERE ref=?)`,
		quiesce.Action.Ref, quiesce.Action.Ref, nextP.Ref,
	).Scan(&qReceipts, &qConsumptions, &pActions); err != nil || qReceipts != 0 || qConsumptions != 0 || pActions != 0 {
		t.Fatalf("crossed Q partial receipt=%d consumption=%d next=%d err=%v",
			qReceipts, qConsumptions, pActions, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err != nil {
		t.Fatalf("crossed Q dirtied recovery: %s", sqliteTestErrorChain(err))
	}
	if err := fixture.system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	fixture.system.repository = openSQLiteV15Repository(t, fixture.system.path, fixture.system.clock.Now)
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, postQ); err != nil || !written {
		t.Fatalf("terminal Q written=%v err=%v", written, err)
	}
	if got, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, postQ); err != nil || written || got != postQ.Snapshot {
		t.Fatalf("terminal Q replay written=%v got=%+v err=%v", written, got, err)
	}

	fixture.system.clock.Advance(time.Second)
	preserve := fixture.claim(t, application.ActionPreserveAgentEnvironment, "claim:lifecycle:p")
	preparedP, err := application.PrepareAgentEnvironmentLifecycleEffect(postQ.Snapshot, preserve, fixture.system.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleAttempt(ctx, preparedP); err != nil || !written {
		t.Fatalf("attempt P written=%v err=%v", written, err)
	}
	fixture.system.clock.Advance(time.Second)
	manifest := fixture.manifest(t, fixture.system.clock.Now())
	fixture.system.clock.Advance(time.Second)
	preserved := fixture.nextToken(t, preparedP.Snapshot.Token, "preserved", ports.AgentEnvironmentPreserved)
	preserveReceipt := ports.AgentPreserveReceipt{
		Subject: preparedP.Snapshot.Subject, PreviousToken: preparedP.Snapshot.Token, NextToken: preserved,
		IdempotencyKey: preparedP.Attempt.IdempotencyKey, Manifest: manifest,
		ReceiptRef: "physical-receipt:lifecycle:p", ConfirmedAt: fixture.system.clock.Now(),
	}
	fact := fixture.preservation(t, preserveReceipt, fixture.system.clock.Now().Add(time.Second))
	fixture.system.clock.Advance(time.Second)
	record = fixture.record(t)
	outcomeP, err := application.RecordAgentEnvironmentPreserveOutcome(
		preparedP, preserveReceipt, &fact, record, fixture.system.clock.Now())
	if err != nil || outcomeP.Terminal == nil {
		t.Fatalf("preserve outcome=%+v err=%v", outcomeP, err)
	}
	postP := *outcomeP.Terminal
	nextC, err := (&application.Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		record, postP.Snapshot, application.ActionCloseAgentEnvironment, postP.OperationAt)
	if err != nil {
		t.Fatal(err)
	}
	postP.NextAction = &nextC
	for name, mutate := range map[string]func(*application.ComprobantePreservacionEntornoAgente){
		"physical receipt": func(crossed *application.ComprobantePreservacionEntornoAgente) {
			crossed.Resultado.ComprobanteRef = "physical-receipt:lifecycle:p-crossed"
			crossed.Resultado.SelloDigest = ports.ResumenSelloPreservacionEntorno(crossed.Resultado)
		},
		"manifest ref": func(crossed *application.ComprobantePreservacionEntornoAgente) {
			crossed.ManifiestoFisicoRef = "physical-manifest:lifecycle-crossed"
		},
		"manifest digest": func(crossed *application.ComprobantePreservacionEntornoAgente) {
			crossed.ManifiestoFisicoDigest = strings.Repeat("9", 64)
		},
	} {
		t.Run("terminal crossed preservation "+name, func(t *testing.T) {
			crossed := fact
			mutate(&crossed)
			badP := postP
			badP.PreservationFact = &crossed
			if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, badP); err == nil || written {
				t.Fatalf("crossed P terminal written=%v err=%v", written, err)
			}
			var facts, receipts, consumptions, actions int
			if err := fixture.system.repository.db.QueryRow(`
SELECT (SELECT COUNT(*) FROM agent_environment_receipts WHERE ref=?),
       (SELECT COUNT(*) FROM effect_receipts WHERE action_ref=?),
       (SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref=?),
       (SELECT COUNT(*) FROM outbox WHERE ref=?)`,
				fact.Ref, preserve.Action.Ref, preserve.Action.Ref, nextC.Ref,
			).Scan(&facts, &receipts, &consumptions, &actions); err != nil ||
				facts != 0 || receipts != 0 || consumptions != 0 || actions != 0 {
				t.Fatalf("crossed P partial fact=%d receipt=%d consumption=%d next=%d err=%v",
					facts, receipts, consumptions, actions, err)
			}
		})
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err != nil {
		t.Fatalf("crossed P dirtied recovery: %s", sqliteTestErrorChain(err))
	}
	if err := fixture.system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	fixture.system.repository = openSQLiteV15Repository(t, fixture.system.path, fixture.system.clock.Now)
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, postP); err != nil || !written {
		t.Fatalf("terminal P written=%v err=%s", written, sqliteTestErrorChain(err))
	}

	fixture.system.clock.Advance(time.Second)
	closeClaim := fixture.claim(t, application.ActionCloseAgentEnvironment, "claim:lifecycle:c")
	preparedC, err := application.PrepareAgentEnvironmentLifecycleEffect(postP.Snapshot, closeClaim, fixture.system.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleAttempt(ctx, preparedC); err != nil || !written {
		t.Fatalf("attempt C written=%v err=%v", written, err)
	}
	fixture.system.clock.Advance(time.Second)
	closed := fixture.nextToken(t, preparedC.Snapshot.Token, "closed", ports.AgentEnvironmentClosed)
	outcomeC, err := application.RecordAgentEnvironmentCloseOutcome(preparedC, ports.AgentCloseReceipt{
		Subject: preparedC.Snapshot.Subject, PreviousToken: preparedC.Snapshot.Token, NextToken: closed,
		Preservation: preparedC.Snapshot.Preservation, IdempotencyKey: preparedC.Attempt.IdempotencyKey,
		ReceiptRef: "physical-receipt:lifecycle:c", ConfirmedAt: fixture.system.clock.Now(),
	}, fixture.system.clock.Now())
	if err != nil || outcomeC.Terminal == nil {
		t.Fatalf("close outcome=%+v err=%v", outcomeC, err)
	}
	postC := *outcomeC.Terminal
	postC.ReadyToFinalize = true
	postC.FinalizationAction = &application.ActionRecord{
		Ref:  "action:observe-finalize:" + preparedC.Snapshot.Subject.ExecutionRef.String(),
		Kind: application.ActionObserveAgent, GoalRef: preparedC.Snapshot.Subject.GoalRef,
		WorkItemRef:        preparedC.Snapshot.Subject.WorkItemRef,
		ExecutionRef:       preparedC.Snapshot.Subject.ExecutionRef,
		PlanGeneration:     preparedC.Snapshot.Subject.PlanGeneration,
		WorkItemGeneration: closeClaim.Action.WorkItemGeneration,
		AvailableAt:        postC.OperationAt,
	}
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, postC); err != nil || !written {
		t.Fatalf("terminal C written=%v err=%s", written, sqliteTestErrorChain(err))
	}
	final, found, err := fixture.system.repository.GetAgentEnvironmentLifecycle(ctx, fixture.execution.Ref)
	if err != nil || !found || !final.ReadyToFinalize || final.NextAction != nil ||
		final.Snapshot.Token.State != ports.AgentEnvironmentClosed || final.Preservation == nil ||
		final.Preservation.Ref != fact.Ref {
		t.Fatalf("final lifecycle found=%v state=%+v err=%s", found, final, sqliteTestErrorChain(err))
	}
	var lifecycleAttempts, lifecycleReceipts, lifecycleConsumptions int
	if err := fixture.system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_attempts
WHERE action_ref IN (?,?,?)`, quiesce.Action.Ref, preserve.Action.Ref, closeClaim.Action.Ref).Scan(&lifecycleAttempts); err != nil {
		t.Fatal(err)
	}
	if err := fixture.system.repository.db.QueryRow(`SELECT COUNT(*) FROM effect_receipts
WHERE action_ref IN (?,?,?)`, quiesce.Action.Ref, preserve.Action.Ref, closeClaim.Action.Ref).Scan(&lifecycleReceipts); err != nil {
		t.Fatal(err)
	}
	if err := fixture.system.repository.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts
WHERE kind IN ('quiesce_agent','preserve_agent_environment','close_agent_environment')`).Scan(&lifecycleConsumptions); err != nil {
		t.Fatal(err)
	}
	if lifecycleAttempts != 3 || lifecycleReceipts != 3 || lifecycleConsumptions != 3 {
		t.Fatalf("QPC ledgers attempts=%d receipts=%d consumptions=%d",
			lifecycleAttempts, lifecycleReceipts, lifecycleConsumptions)
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err != nil {
		t.Fatalf("recovery rejected QPC lifecycle: %s", sqliteTestErrorChain(err))
	}
}

func TestAgentEnvironmentLifecycleRejectsUngovernedActionBeforeClaimAndRecovery(t *testing.T) {
	ctx := context.Background()
	fixture := newSQLiteAgentEnvironmentLifecycleFixture(t)
	initial := fixture.initial("active-ungoverned", fixture.system.clock.Now())
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleInitial(
		ctx, application.AgentEnvironmentLifecycleInitialState{Snapshot: initial, OperationAt: initial.RecordedAt},
	); err != nil || !written {
		t.Fatalf("initial written=%v err=%v", written, err)
	}
	action, err := (&application.Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		fixture.record(t), initial, application.ActionQuiesceAgent, fixture.system.clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	const insertUngoverned = `
INSERT INTO outbox(ref,kind,goal_ref,work_item_ref,execution_ref,
 plan_generation,work_item_generation,available_at,governance_version,effect_intent_ref)
VALUES(?,?,?,?,?,?,?,?,0,NULL)`
	arguments := []any{
		action.Ref, string(action.Kind), action.GoalRef.String(), action.WorkItemRef.String(),
		action.ExecutionRef.String(), int64(action.PlanGeneration), int64(action.WorkItemGeneration),
		requiredTime(action.AvailableAt),
	}
	if _, err := fixture.system.repository.db.ExecContext(ctx, insertUngoverned, arguments...); err == nil {
		t.Fatal("DB accepted ungoverned lifecycle action")
	}

	connection, err := fixture.system.repository.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rewriteRecoveryTrigger(t, fixture.system.repository.db, "outbox_governance_insert_guard", func() {
		if _, err := connection.ExecContext(ctx, `PRAGMA ignore_check_constraints=ON`); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(ctx, insertUngoverned, arguments...); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(ctx, `PRAGMA ignore_check_constraints=OFF`); err != nil {
			t.Fatal(err)
		}
	})
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if claim, found, err := fixture.system.repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:ungoverned", Token: "claim:ungoverned", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: fixture.system.policy,
		CapacityCandidates: fixture.system.capacidad,
	}); err != nil || found {
		t.Fatalf("ungoverned action entered claim loop claim=%+v found=%v err=%v", claim, found, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err == nil ||
		(!strings.Contains(sqliteTestErrorChain(err), "sqlite.recovery_v15_action_intent_invalid") &&
			!strings.Contains(sqliteTestErrorChain(err), "sqlite.integrity_check_failed")) {
		t.Fatalf("recovery accepted ungoverned lifecycle action: %s", sqliteTestErrorChain(err))
	}
	if err := fixture.system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, openErr := Open(ctx, Options{
		Path: fixture.system.path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: fixture.system.clock.Now,
	})
	if openErr != nil {
		t.Fatalf("plain reopen changed contract: %s", sqliteTestErrorChain(openErr))
	}
	defer reopened.Close()
	if claim, found, err := reopened.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:ungoverned-reopen", Token: "claim:ungoverned-reopen", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: fixture.system.policy,
		CapacityCandidates: fixture.system.capacidad,
	}); err != nil || found {
		t.Fatalf("reopened ungoverned action entered claim loop claim=%+v found=%v err=%v", claim, found, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, reopened.db); err == nil {
		t.Fatal("reopened recovery accepted ungoverned lifecycle action")
	}
}

func TestAgentEnvironmentLifecycleRejectsGovernedOutOfOrderPreserveAndClose(t *testing.T) {
	t.Run("preserve before quiesce terminal", func(t *testing.T) {
		fixture, preparedQ := prepareSQLiteLifecycleQuiesceAttempt(t, "out-of-order-p")
		postQ := sqliteLifecycleQuiesceTerminal(t, fixture, preparedQ, "out-of-order-p")
		assertSQLiteLifecycleOutOfOrderActionRejected(t, fixture, *postQ.NextAction, preparedQ.Claim.LeaseUntil)
	})

	t.Run("close before preserve terminal", func(t *testing.T) {
		fixture, preparedQ := prepareSQLiteLifecycleQuiesceAttempt(t, "out-of-order-c")
		postQ := sqliteLifecycleQuiesceTerminal(t, fixture, preparedQ, "out-of-order-c")
		if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(
			context.Background(), postQ,
		); err != nil || !written {
			t.Fatalf("terminal Q written=%v err=%v", written, err)
		}
		fixture.system.clock.Advance(time.Second)
		preserve := fixture.claim(t, application.ActionPreserveAgentEnvironment, "claim:out-of-order-c:p")
		preparedP, err := application.PrepareAgentEnvironmentLifecycleEffect(
			postQ.Snapshot, preserve, fixture.system.clock.Now(),
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleAttempt(
			context.Background(), preparedP,
		); err != nil || !written {
			t.Fatalf("attempt P written=%v err=%v", written, err)
		}
		postP := sqliteLifecyclePreserveTerminal(t, fixture, preparedP, "out-of-order-c")
		assertSQLiteLifecycleOutOfOrderActionRejected(t, fixture, *postP.NextAction, preparedP.Claim.LeaseUntil)
	})
}

func TestAgentEnvironmentLifecycleTerminalCASDistinctCandidatesHasOneWinner(t *testing.T) {
	ctx := context.Background()
	fixture, preparedQ := prepareSQLiteLifecycleQuiesceAttempt(t, "terminal-cas")
	candidates := []application.AgentEnvironmentLifecyclePostEffectState{
		sqliteLifecycleQuiesceTerminal(t, fixture, preparedQ, "terminal-cas-a"),
		sqliteLifecycleQuiesceTerminal(t, fixture, preparedQ, "terminal-cas-b"),
	}
	type terminalResult struct {
		candidate int
		snapshot  application.AgentEnvironmentLifecycleSnapshot
		written   bool
		err       error
	}
	start := make(chan struct{})
	results := make(chan terminalResult, len(candidates))
	var group sync.WaitGroup
	for index, candidate := range candidates {
		index, candidate := index, candidate
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			snapshot, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleTerminal(ctx, candidate)
			results <- terminalResult{candidate: index, snapshot: snapshot, written: written, err: err}
		}()
	}
	close(start)
	group.Wait()
	close(results)
	writers, winnerIndex := 0, -1
	var returned []terminalResult
	for result := range results {
		if result.err != nil {
			t.Fatalf("terminal candidate=%d err=%s", result.candidate, sqliteTestErrorChain(result.err))
		}
		if result.written {
			writers++
			winnerIndex = result.candidate
		}
		returned = append(returned, result)
	}
	if writers != 1 {
		t.Fatalf("terminal writers=%d want=1", writers)
	}
	winner := candidates[winnerIndex]
	for _, result := range returned {
		if result.snapshot != winner.Snapshot {
			t.Fatalf("terminal candidate=%d returned=%+v winner=%+v",
				result.candidate, result.snapshot, winner.Snapshot)
		}
	}
	stored, found, err := fixture.system.repository.GetAgentEnvironmentLifecycle(ctx, fixture.execution.Ref)
	if err != nil || !found || stored.Snapshot != winner.Snapshot || stored.NextAction == nil ||
		!sameLifecycleActionIgnoringAttachedApproval(*stored.NextAction, *winner.NextAction) {
		t.Fatalf("terminal winner found=%v stored=%+v err=%v", found, stored, err)
	}
	var receipts, consumptions, nextActions int
	if err := fixture.system.repository.db.QueryRow(`
SELECT (SELECT COUNT(*) FROM effect_receipts WHERE action_ref=?),
       (SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref=?),
       (SELECT COUNT(*) FROM outbox WHERE ref=?)`,
		preparedQ.Claim.Action.Ref, preparedQ.Claim.Action.Ref, winner.NextAction.Ref,
	).Scan(&receipts, &consumptions, &nextActions); err != nil ||
		receipts != 1 || consumptions != 1 || nextActions != 1 {
		t.Fatalf("terminal CAS ledgers receipt=%d consumption=%d next=%d err=%v",
			receipts, consumptions, nextActions, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err != nil {
		t.Fatalf("terminal CAS recovery: %s", sqliteTestErrorChain(err))
	}
}

func prepareSQLiteLifecycleQuiesceAttempt(
	t *testing.T,
	suffix string,
) (sqliteAgentEnvironmentLifecycleFixture, application.AgentEnvironmentLifecyclePreEffectState) {
	t.Helper()
	ctx := context.Background()
	fixture := newSQLiteAgentEnvironmentLifecycleFixture(t)
	initial := fixture.initial("active-"+suffix, fixture.system.clock.Now())
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleInitial(
		ctx, application.AgentEnvironmentLifecycleInitialState{Snapshot: initial, OperationAt: initial.RecordedAt},
	); err != nil || !written {
		t.Fatalf("initial written=%v err=%v", written, err)
	}
	fixture.system.clock.Advance(time.Second)
	claim := fixture.addAndClaim(t, initial, application.ActionQuiesceAgent, "claim:"+suffix+":q")
	prepared, err := application.PrepareAgentEnvironmentLifecycleEffect(initial, claim, fixture.system.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, written, err := fixture.system.repository.RecordAgentEnvironmentLifecycleAttempt(ctx, prepared); err != nil || !written {
		t.Fatalf("attempt Q written=%v err=%v", written, err)
	}
	return fixture, prepared
}

func sqliteLifecycleQuiesceTerminal(
	t *testing.T,
	fixture sqliteAgentEnvironmentLifecycleFixture,
	prepared application.AgentEnvironmentLifecyclePreEffectState,
	suffix string,
) application.AgentEnvironmentLifecyclePostEffectState {
	t.Helper()
	next := fixture.nextToken(t, prepared.Snapshot.Token, "quiesced-"+suffix, ports.AgentEnvironmentQuiesced)
	outcome, err := application.RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: prepared.Snapshot.Subject, PreviousToken: prepared.Snapshot.Token, NextToken: next,
		IdempotencyKey: prepared.Attempt.IdempotencyKey, ReceiptRef: "physical-receipt:" + suffix + ":q",
		ConfirmedAt: fixture.system.clock.Now(),
	}, fixture.system.clock.Now())
	if err != nil || outcome.Terminal == nil {
		t.Fatalf("quiesce outcome=%+v err=%v", outcome, err)
	}
	post := *outcome.Terminal
	nextAction, err := (&application.Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		fixture.record(t), post.Snapshot, application.ActionPreserveAgentEnvironment, post.OperationAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	post.NextAction = &nextAction
	return post
}

func sqliteLifecyclePreserveTerminal(
	t *testing.T,
	fixture sqliteAgentEnvironmentLifecycleFixture,
	prepared application.AgentEnvironmentLifecyclePreEffectState,
	suffix string,
) application.AgentEnvironmentLifecyclePostEffectState {
	t.Helper()
	fixture.system.clock.Advance(time.Second)
	manifest := fixture.manifest(t, fixture.system.clock.Now())
	fixture.system.clock.Advance(time.Second)
	next := fixture.nextToken(t, prepared.Snapshot.Token, "preserved-"+suffix, ports.AgentEnvironmentPreserved)
	receipt := ports.AgentPreserveReceipt{
		Subject: prepared.Snapshot.Subject, PreviousToken: prepared.Snapshot.Token, NextToken: next,
		IdempotencyKey: prepared.Attempt.IdempotencyKey, Manifest: manifest,
		ReceiptRef: "physical-receipt:" + suffix + ":p", ConfirmedAt: fixture.system.clock.Now(),
	}
	fact := fixture.preservation(t, receipt, fixture.system.clock.Now().Add(time.Second))
	fixture.system.clock.Advance(time.Second)
	outcome, err := application.RecordAgentEnvironmentPreserveOutcome(
		prepared, receipt, &fact, fixture.record(t), fixture.system.clock.Now(),
	)
	if err != nil || outcome.Terminal == nil {
		t.Fatalf("preserve outcome=%+v err=%v", outcome, err)
	}
	post := *outcome.Terminal
	nextAction, err := (&application.Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		fixture.record(t), post.Snapshot, application.ActionCloseAgentEnvironment, post.OperationAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	post.NextAction = &nextAction
	return post
}

func assertSQLiteLifecycleOutOfOrderActionRejected(
	t *testing.T,
	fixture sqliteAgentEnvironmentLifecycleFixture,
	action application.ActionRecord,
	claimLeaseUntil time.Time,
) {
	t.Helper()
	ctx := context.Background()
	// Avoid the active-action unique index masking the lifecycle BEFORE INSERT
	// trigger; governance intent remains exact and only the frontier is future.
	action.WorkItemGeneration++
	transaction, err := beginTransaction(ctx, fixture.system.repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertAction(ctx, transaction, action); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "sqlite.outbox_effect_intent_invalid") {
		_ = transaction.Rollback()
		t.Fatalf("out-of-order %s trigger err=%s", action.Kind, sqliteTestErrorChain(err))
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	var actions, intents, approvals int
	if err := fixture.system.repository.db.QueryRow(`
SELECT (SELECT COUNT(*) FROM outbox WHERE ref=?),
       (SELECT COUNT(*) FROM effect_intents WHERE ref=?),
       (SELECT COUNT(*) FROM effect_approvals WHERE ref=?)`,
		action.Ref, action.EffectIntentRef, action.EffectApproval.Ref,
	).Scan(&actions, &intents, &approvals); err != nil || actions != 0 || intents != 0 || approvals != 0 {
		t.Fatalf("out-of-order %s partial action=%d intent=%d approval=%d err=%v",
			action.Kind, actions, intents, approvals, err)
	}
	rewriteRecoveryTrigger(t, fixture.system.repository.db, "outbox_governance_insert_guard", func() {
		transaction, err := beginTransaction(ctx, fixture.system.repository)
		if err != nil {
			t.Fatal(err)
		}
		if err := insertAction(ctx, transaction, action); err != nil {
			_ = transaction.Rollback()
			t.Fatal(err)
		}
		if err := commit(transaction); err != nil {
			t.Fatal(err)
		}
	})
	var beforeRevision int64
	if err := fixture.system.repository.db.QueryRow(`
SELECT revision FROM agent_environment_lifecycles WHERE execution_ref=?`,
		fixture.execution.Ref.String(),
	).Scan(&beforeRevision); err != nil {
		t.Fatal(err)
	}
	readTx, err := beginReadTransaction(ctx, fixture.system.repository)
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := readClaimCandidateWindow(
		ctx, readTx, claimLeaseUntil.Add(time.Nanosecond), true, false, nil,
	)
	if err != nil {
		_ = readTx.Rollback()
		t.Fatal(err)
	}
	if err := readTx.Rollback(); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if candidate.action.Ref == action.Ref {
			t.Fatalf("out-of-order %s entered claim window", action.Kind)
		}
	}
	if _, _, err := validateRecoveryDatabase(ctx, fixture.system.repository.db); err == nil ||
		(!strings.Contains(sqliteTestErrorChain(err), "sqlite.recovery_agent_environment_lifecycle_history_invalid") &&
			!strings.Contains(sqliteTestErrorChain(err), "sqlite.recovery_agent_environment_lifecycle_action_invalid")) {
		t.Fatalf("recovery accepted/cross-classified out-of-order %s: %s",
			action.Kind, sqliteTestErrorChain(err))
	}
	var mutated, afterRevision int64
	if err := fixture.system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox WHERE ref=?
 AND (claim_token IS NOT NULL OR delivery_attempt<>0 OR fence<>0 OR completed_at IS NOT NULL)`,
		action.Ref,
	).Scan(&mutated); err != nil {
		t.Fatal(err)
	}
	if err := fixture.system.repository.db.QueryRow(`
SELECT revision FROM agent_environment_lifecycles WHERE execution_ref=?`,
		fixture.execution.Ref.String(),
	).Scan(&afterRevision); err != nil || mutated != 0 || afterRevision != beforeRevision {
		t.Fatalf("out-of-order %s mutated=%d revision=%d/%d err=%v",
			action.Kind, mutated, beforeRevision, afterRevision, err)
	}
}

type sqliteAgentEnvironmentLifecycleFixture struct {
	system      *sqliteV15System
	execution   application.ExecutionRecord
	request     ports.AgentLaunchRequest
	launch      ports.AgentLaunchReceipt
	launchFence uint64
}

func newSQLiteAgentEnvironmentLifecycleFixture(t *testing.T) sqliteAgentEnvironmentLifecycleFixture {
	t.Helper()
	ctx := context.Background()
	system := newSQLiteV15System(t, 2)
	system.external.requierePreservacion = true
	system.orchestrator = newSQLiteV16Orchestrator(t, system)
	created := system.submit(t, "request:lifecycle-store")
	processSQLiteV16Actions(t, system, application.ActionLaunchAgent)
	record, err := system.repository.GetGoal(ctx, created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	if len(record.Executions) != 1 || !record.Executions[0].RequierePreservacionEntorno {
		t.Fatalf("lifecycle execution missing: %+v", record.Executions)
	}
	execution := record.Executions[0]
	request := system.external.launchRequests[execution.Ref]
	launch := system.external.launches[request.IdempotencyKey]
	var launchFence uint64
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.Kind == application.ActionLaunchAgent && receipt.ExecutionRef == execution.Ref {
			launchFence = receipt.Fence
		}
	}
	if launchFence == 0 {
		t.Fatal("launch consumption fence missing")
	}
	// Remove the legacy observation delivery from the active frontier with a
	// normal fenced consumption; lifecycle bootstrap wiring is intentionally
	// outside this adapter test.
	claim, found, err := system.repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:lifecycle-observe", Token: "claim:lifecycle-observe", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: system.policy, CapacityCandidates: system.capacidad,
	})
	if err != nil || !found || claim.Action.Kind != application.ActionObserveAgent {
		t.Fatalf("observe claim=%+v found=%v err=%v", claim, found, err)
	}
	tx, err := beginTransaction(ctx, system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, requireClaim(ctx, tx, claim))
	_, err = tx.ExecContext(ctx, `UPDATE outbox SET completed_at=?,last_error_code='' WHERE ref=?`,
		requiredTime(system.clock.Now()), claim.Action.Ref)
	sqliteTestNoError(t, err)
	consumption := application.ActionConsumptionReceipt{
		ActionRef: claim.Action.Ref, Kind: claim.Action.Kind, GoalRef: claim.Action.GoalRef,
		WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef,
		PlanGeneration: claim.Action.PlanGeneration, WorkItemGeneration: claim.Action.WorkItemGeneration,
		Fence: claim.Fence, DeliveryAttempt: claim.DeliveryAttempt, ClaimToken: claim.Token,
		WorkerRef: claim.WorkerRef, Outcome: application.ActionConsumedCompleted, ConsumedAt: system.clock.Now(),
	}
	sqliteTestNoError(t, insertActionConsumptionReceipt(ctx, tx, consumption, 0, nil))
	sqliteTestNoError(t, commit(tx))
	return sqliteAgentEnvironmentLifecycleFixture{
		system: system, execution: execution, request: request, launch: launch, launchFence: launchFence,
	}
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) initial(revision string, at time.Time) application.AgentEnvironmentLifecycleSnapshot {
	token := lifecycleStoreToken(nil, fixture.execution.ExternalRef, revision, "physical-fence:lifecycle", ports.AgentEnvironmentActive)
	snapshot, err := application.NewAgentEnvironmentLifecycleSnapshot(
		fixture.request, fixture.launch,
		ports.AgentEnvironmentInspectReceipt{Subject: lifecycleStoreSubject(fixture.launch), Token: token}, at)
	if err != nil {
		panic(err)
	}
	return snapshot
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) addAndClaim(
	t *testing.T,
	snapshot application.AgentEnvironmentLifecycleSnapshot,
	kind application.ActionKind,
	token string,
) application.ActionClaim {
	t.Helper()
	action, err := (&application.Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		fixture.record(t), snapshot, kind, fixture.system.clock.Now())
	sqliteTestNoError(t, err)
	tx, err := beginTransaction(context.Background(), fixture.system.repository)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, insertAction(context.Background(), tx, action))
	sqliteTestNoError(t, commit(tx))
	return fixture.claim(t, kind, token)
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) claim(
	t *testing.T,
	kind application.ActionKind,
	token string,
) application.ActionClaim {
	t.Helper()
	claim, found, err := fixture.system.repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:lifecycle", Token: token, LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(), BudgetPolicy: fixture.system.policy,
		CapacityCandidates: fixture.system.capacidad,
	})
	if err != nil || !found || claim.Action.Kind != kind {
		t.Fatalf("claim %s got=%+v found=%v err=%v", kind, claim, found, err)
	}
	return claim
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) record(t *testing.T) application.GoalRecord {
	t.Helper()
	record, err := fixture.system.repository.GetGoal(context.Background(), fixture.execution.GoalRef)
	sqliteTestNoError(t, err)
	return record
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) nextToken(
	t *testing.T,
	previous ports.AgentEnvironmentLifecycleToken,
	revision string,
	state ports.AgentEnvironmentLifecycleState,
) ports.AgentEnvironmentLifecycleToken {
	t.Helper()
	return lifecycleStoreToken(t, fixture.execution.ExternalRef, revision, previous.Fence.String(), state)
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) manifest(
	t *testing.T,
	sealedAt time.Time,
) ports.AgentPhysicalPreservationManifest {
	t.Helper()
	content := []byte(`{"schema":"physical-lifecycle-test"}`)
	digest := sha256.Sum256(content)
	revision, err := ports.NewAgentPhysicalRevision("physical-revision:manifest")
	sqliteTestNoError(t, err)
	return ports.AgentPhysicalPreservationManifest{
		Ref: "physical-manifest:lifecycle", SHA256: fmt.Sprintf("%x", digest), Content: content,
		ContentBytes: uint64(len(content)), WorkRevision: revision,
		Causality: ports.AgentPhysicalPreservationCausality{
			PlanSHA256: strings.Repeat("1", 64), GrantSHA256: strings.Repeat("2", 64),
			KernelSHA256: strings.Repeat("3", 64), InitramfsSHA256: strings.Repeat("4", 64),
		},
		SealedAt: sealedAt,
	}
}

func (fixture sqliteAgentEnvironmentLifecycleFixture) preservation(
	t *testing.T,
	receipt ports.AgentPreserveReceipt,
	registeredAt time.Time,
) application.ComprobantePreservacionEntornoAgente {
	t.Helper()
	packageRef, err := goal.NewArtifactRef("artifact:sha256:" + strings.Repeat("a", 64))
	sqliteTestNoError(t, err)
	inventoryRef, err := goal.NewArtifactRef("artifact:sha256:" + strings.Repeat("b", 64))
	sqliteTestNoError(t, err)
	result := ports.ResultadoPreservacionEntornoAgente{
		Estado: ports.EntornoAgentePreservadoPendienteRevision, EjecucionRef: fixture.execution.Ref,
		IntentoEjecucion: fixture.execution.AttemptNo, IdentidadExterna: fixture.execution.ExternalRef,
		Cerca: fixture.launchFence, PaqueteRef: packageRef, PaqueteDigest: strings.Repeat("a", 64),
		InventarioRef: inventoryRef, InventarioDigest: strings.Repeat("b", 64),
		ConfiguracionDigest: strings.Repeat("c", 64), RootFSDigest: strings.Repeat("d", 64),
		ComprobanteRef: receipt.ReceiptRef, SelladoEn: receipt.Manifest.SealedAt,
		PreservadoEn: receipt.ConfirmedAt,
	}
	result.SelloDigest = ports.ResumenSelloPreservacionEntorno(result)
	return application.ComprobantePreservacionEntornoAgente{
		Ref: "environment-receipt:lifecycle", ClaveIdempotencia: "environment-preservation:lifecycle",
		ProyectoRef: fixture.system.project,
		ObjetivoRef: receipt.Subject.GoalRef, ItemRef: receipt.Subject.WorkItemRef,
		EjecucionRef:   receipt.Subject.ExecutionRef,
		AlcanceEspacio: application.PreservacionEntornoSinEspacioTrabajo,
		Resultado:      result, RegistradoEn: registeredAt,
		ManifiestoFisicoRef: receipt.Manifest.Ref, ManifiestoFisicoDigest: receipt.Manifest.SHA256,
	}
}

func lifecycleStoreSubject(receipt ports.AgentLaunchReceipt) ports.AgentEnvironmentLifecycleSubject {
	return ports.AgentEnvironmentLifecycleSubject{
		ExecutionRef: receipt.ExecutionRef, GoalRef: receipt.GoalRef, WorkItemRef: receipt.WorkItemRef,
		PlanGeneration: receipt.PlanGeneration, AppSpecGeneration: receipt.AppSpecGeneration,
		ExecutionAttempt: receipt.ExecutionAttempt, SpecHash: receipt.SpecHash,
		ProviderRef: receipt.ProviderRef, ModelRef: receipt.ModelRef, AgentRef: receipt.AgentRef,
		ExternalRef: receipt.ExternalRef,
	}
}

func lifecycleStoreToken(
	t *testing.T,
	physical, revision, fence string,
	state ports.AgentEnvironmentLifecycleState,
) ports.AgentEnvironmentLifecycleToken {
	if t != nil {
		t.Helper()
	}
	physicalToken, err := ports.NewAgentPhysicalToken(physical)
	if err != nil {
		panic(err)
	}
	physicalRevision, err := ports.NewAgentPhysicalRevision(revision)
	if err != nil {
		panic(err)
	}
	physicalFence, err := ports.NewAgentPhysicalFence(fence)
	if err != nil {
		panic(err)
	}
	return ports.AgentEnvironmentLifecycleToken{
		PhysicalToken: physicalToken, Revision: physicalRevision, Fence: physicalFence, State: state,
	}
}
