package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestTerminalAgentLaunchReconciliationPreservesRun7ShapeAndNeverCallsLaunch(t *testing.T) {
	ctx := context.Background()
	system, original, physicalAttempt := seedV27AmbiguousEffectAttempt(t, "terminal-run7-shape")
	quarantinedAt := system.clock.Now().UTC()
	sqliteTestNoError(t, system.repository.QuarantineAction(ctx, application.ActionQuarantinedState{
		Claim: original, ErrorCode: "application.effect_unknown_applied", OperationAt: quarantinedAt,
		Event: application.EventRecord{
			Ref: "event:terminal-run7-quarantine", Kind: "action.quarantined",
			GoalRef: original.Action.GoalRef, WorkItemRef: original.Action.WorkItemRef,
			ExecutionRef: original.Action.ExecutionRef, OccurredAt: quarantinedAt,
		},
	}))

	actionBefore, consumptionBefore := terminalOriginalRows(t, system.repository.db, original.Action.Ref)
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Second)
	request := application.ReconcileTerminalAgentLaunchRequest{
		RequestRef: "request:terminal-run7-reconciliation",
		GoalRef:    original.Action.GoalRef, WorkItemRef: original.Action.WorkItemRef,
		ExecutionRef: original.Action.ExecutionRef, ActionRef: original.Action.Ref,
		EffectIntentRef:    original.Action.EffectIntentRef,
		EffectIntentDigest: original.Action.EffectIntent.Digest,
		EffectAttemptRef:   physicalAttempt.Ref, PlanGeneration: original.Action.PlanGeneration,
		WorkItemGeneration: original.Action.WorkItemGeneration, ActionFence: physicalAttempt.ActionFence,
	}
	authorized, err := system.orchestrator.ReconcileTerminalAgentLaunch(ctx, system.access, request)
	if err != nil || !authorized.Created || authorized.Authority.ActionRef != original.Action.Ref ||
		authorized.Authority.EffectAttemptRef != physicalAttempt.Ref {
		t.Fatalf("authorization=%+v err=%s", authorized, sqliteTestErrorChain(err))
	}
	replayed, err := system.orchestrator.ReconcileTerminalAgentLaunch(ctx, system.access, request)
	if err != nil || replayed.Created || replayed.Authority != authorized.Authority {
		t.Fatalf("authorization replay=%+v err=%s", replayed, sqliteTestErrorChain(err))
	}

	restartSQLiteV15System(t, system)
	reconciler := &terminalLaunchReconciler{clock: system.clock}
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)
	claim, found, err := system.orchestrator.ClaimNextAction(
		ctx, "worker:terminal-run7-reconciliation", application.ActionClaimSelection{ExcludeLaunch: true},
	)
	if err != nil || !found || claim.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch ||
		claim.Action.Ref != original.Action.Ref || claim.RecoveryEffectAttemptRef != physicalAttempt.Ref ||
		claim.Action.EffectIntent.IdempotencyKey != physicalAttempt.IdempotencyKey ||
		claim.Fence <= physicalAttempt.ActionFence {
		t.Fatalf("claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	if _, processErr := system.orchestrator.ProcessClaim(ctx, claim); processErr != nil {
		t.Fatalf("pending continuation process err=%s", sqliteTestErrorChain(processErr))
	}
	launches, reconciliations, continuations, requests := reconciler.snapshot()
	if launches != 0 || reconciliations != 0 || continuations != 0 || len(requests) != 0 {
		t.Fatalf("before authority Launch=%d ReconcileLaunch=%d ContinueV41=%d requests=%+v",
			launches, reconciliations, continuations, requests)
	}
	recordTerminalContinuationV41(
		t, system, original, physicalAttempt, authorized.Authority.Ref, "terminal-run7-shape",
	)
	system.clock.Advance(claim.Action.EffectIntent.QuotaRetryDelay)
	restartSQLiteV15System(t, system)
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)
	continuedClaim, found, err := system.orchestrator.ClaimNextAction(
		ctx, "worker:terminal-run7-continuation", application.ActionClaimSelection{ExcludeLaunch: true},
	)
	if err != nil || !found || continuedClaim.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch {
		t.Fatalf("continued claim=%+v found=%t err=%s", continuedClaim, found, sqliteTestErrorChain(err))
	}
	if result, processErr := system.orchestrator.ProcessClaim(ctx, continuedClaim); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("continued process=%+v err=%s", result, sqliteTestErrorChain(processErr))
	}
	launches, reconciliations, continuations, requests = reconciler.snapshot()
	if launches != 0 || reconciliations != 0 || continuations != 1 || len(requests) != 1 ||
		requests[0].IdempotencyKey != physicalAttempt.IdempotencyKey ||
		requests[0].EffectAuthority.EffectAttemptRef != physicalAttempt.Ref ||
		!requests[0].EffectAuthority.ClaimLeaseUntil.Equal(physicalAttempt.ClaimLeaseUntil) {
		t.Fatalf("Launch=%d ReconcileLaunch=%d ContinueV41=%d requests=%+v",
			launches, reconciliations, continuations, requests)
	}

	actionAfter, consumptionAfter := terminalOriginalRows(t, system.repository.db, original.Action.Ref)
	if actionAfter != actionBefore || consumptionAfter != consumptionBefore {
		t.Fatalf("original evidence changed\naction before=%q\naction after=%q\nreceipt before=%q\nreceipt after=%q",
			actionBefore, actionAfter, consumptionBefore, consumptionAfter)
	}
	var physicalAttempts, physicalReceipts, authorities, jobs, reconciliationAttempts, reconciliationReceipts int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
 (SELECT COUNT(*) FROM effect_attempts WHERE action_ref=?),
 (SELECT COUNT(*) FROM effect_receipts WHERE action_ref=?),
 (SELECT COUNT(*) FROM agent_launch_reconciliation_authorities WHERE action_ref=?),
 (SELECT COUNT(*) FROM agent_launch_reconciliation_jobs WHERE state='completed'),
 (SELECT COUNT(*) FROM agent_launch_reconciliation_attempts),
 (SELECT COUNT(*) FROM agent_launch_reconciliation_receipts WHERE outcome='completed')`,
		original.Action.Ref, original.Action.Ref, original.Action.Ref).Scan(
		&physicalAttempts, &physicalReceipts, &authorities, &jobs, &reconciliationAttempts, &reconciliationReceipts))
	if physicalAttempts != 1 || physicalReceipts != 1 || authorities != 1 || jobs != 1 ||
		reconciliationAttempts != 2 || reconciliationReceipts != 1 {
		t.Fatalf("counts physical_attempt=%d physical_receipt=%d authorities=%d jobs=%d reconciliation_attempt=%d reconciliation_receipt=%d",
			physicalAttempts, physicalReceipts, authorities, jobs, reconciliationAttempts, reconciliationReceipts)
	}
	var executionState, capacityState, capacityCause string
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT execution.state,capacity.state,transition.cause_kind
FROM executions execution
JOIN agent_capacity_reservations capacity ON capacity.execution_ref=execution.ref
JOIN agent_capacity_transitions transition ON transition.ref=capacity.last_transition_ref
WHERE execution.ref=?`, original.Action.ExecutionRef.String()).Scan(&executionState, &capacityState, &capacityCause))
	if executionState != "running" || capacityState != "consumed" || capacityCause != "reconciliation" {
		t.Fatalf("execution=%s capacity=%s cause=%s", executionState, capacityState, capacityCause)
	}
	var observations int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM outbox
WHERE kind='observe_agent' AND execution_ref=? AND completed_at IS NULL`,
		original.Action.ExecutionRef.String()).Scan(&observations))
	if observations != 1 {
		t.Fatalf("observe actions=%d", observations)
	}
	status, err := system.repository.Status(ctx, system.project)
	if err != nil || status.QuarantinedActions != 0 {
		t.Fatalf("status=%+v err=%s", status, sqliteTestErrorChain(err))
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("reconciled recovery database invalid: %s", sqliteTestErrorChain(err))
	}
}

func TestMigrationV40AddsTerminalLaunchReconciliationJournalToV39(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "terminal-launch-reconciliation-v39.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38LaunchRuntimeDigests)
	defer database.Close()

	var before int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&before))
	if before != 39 {
		t.Fatalf("source schema=%d", before)
	}
	var journalTables int
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE type='table' AND name LIKE 'agent_launch_reconciliation_%'`).Scan(&journalTables))
	if journalTables != 0 {
		t.Fatalf("V39 already has reconciliation tables=%d", journalTables)
	}

	sqliteTestNoError(t, applyMigrationsWithPolicy(ctx, database, migrationPolicy{
		bounded: true, expectFrom: recoverySchemaV38LaunchRuntimeDigests,
		expectTo: recoverySchemaV38TerminalLaunchReconciliation,
	}))
	var after, receipt, strictTables, causalGuard int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&after))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM schema_migrations
WHERE version=40 AND name='040_terminal_agent_launch_reconciliation.sql'`).Scan(&receipt))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_table_list
WHERE name LIKE 'agent_launch_reconciliation_%' AND strict=1`).Scan(&strictTables))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM sqlite_schema
WHERE type='trigger' AND name='effect_receipts_causal_guard'
 AND sql LIKE '%agent_launch_reconciliation_attempts%'`).Scan(&causalGuard))
	if after != 40 || receipt != 1 || strictTables != 4 || causalGuard != 1 {
		t.Fatalf("V40 shape version=%d receipt=%d strict_tables=%d causal_guard=%d",
			after, receipt, strictTables, causalGuard)
	}
	if _, _, err := validateRecoveryDatabase(ctx, database); err != nil {
		t.Fatalf("migrated V40 recovery database invalid: %s", sqliteTestErrorChain(err))
	}
}

func TestTerminalAgentLaunchReconciliationRejectsCrossedHistoricalAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*application.ReconcileTerminalAgentLaunchRequest)
	}{
		{"digest", func(request *application.ReconcileTerminalAgentLaunchRequest) {
			request.EffectIntentDigest = strings.Repeat("b", 64)
		}},
		{"attempt", func(request *application.ReconcileTerminalAgentLaunchRequest) {
			request.EffectAttemptRef += ":crossed"
		}},
		{"fence", func(request *application.ReconcileTerminalAgentLaunchRequest) {
			request.ActionFence++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			system, original, physicalAttempt := seedV27AmbiguousEffectAttempt(t, "terminal-crossed-"+test.name)
			quarantinedAt := system.clock.Now().UTC()
			sqliteTestNoError(t, system.repository.QuarantineAction(ctx, application.ActionQuarantinedState{
				Claim: original, ErrorCode: "application.effect_unknown_applied", OperationAt: quarantinedAt,
				Event: application.EventRecord{
					Ref: "event:terminal-crossed-" + test.name, Kind: "action.quarantined",
					GoalRef: original.Action.GoalRef, WorkItemRef: original.Action.WorkItemRef,
					ExecutionRef: original.Action.ExecutionRef, OccurredAt: quarantinedAt,
				},
			}))
			actionBefore, consumptionBefore := terminalOriginalRows(t, system.repository.db, original.Action.Ref)
			request := application.ReconcileTerminalAgentLaunchRequest{
				RequestRef: "request:terminal-crossed-" + test.name,
				GoalRef:    original.Action.GoalRef, WorkItemRef: original.Action.WorkItemRef,
				ExecutionRef: original.Action.ExecutionRef, ActionRef: original.Action.Ref,
				EffectIntentRef: original.Action.EffectIntentRef, EffectIntentDigest: original.Action.EffectIntent.Digest,
				EffectAttemptRef: physicalAttempt.Ref, PlanGeneration: original.Action.PlanGeneration,
				WorkItemGeneration: original.Action.WorkItemGeneration, ActionFence: physicalAttempt.ActionFence,
			}
			test.mutate(&request)
			if _, err := system.orchestrator.ReconcileTerminalAgentLaunch(ctx, system.access, request); err == nil {
				t.Fatal("crossed historical authority accepted")
			}
			var authorities, jobs, physicalAttempts int
			sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
 (SELECT COUNT(*) FROM agent_launch_reconciliation_authorities),
 (SELECT COUNT(*) FROM agent_launch_reconciliation_jobs),
 (SELECT COUNT(*) FROM effect_attempts WHERE action_ref=?)`, original.Action.Ref).Scan(
				&authorities, &jobs, &physicalAttempts))
			if authorities != 0 || jobs != 0 || physicalAttempts != 1 {
				t.Fatalf("crossed request mutated ledger authorities=%d jobs=%d physical_attempts=%d",
					authorities, jobs, physicalAttempts)
			}
			actionAfter, consumptionAfter := terminalOriginalRows(t, system.repository.db, original.Action.Ref)
			if actionAfter != actionBefore || consumptionAfter != consumptionBefore {
				t.Fatal("crossed request changed original quarantine evidence")
			}
		})
	}
}

func TestTerminalAgentLaunchReconciliationRestartsAndHasOneClaimWinner(t *testing.T) {
	ctx := context.Background()
	system, original, physicalAttempt := seedV27AmbiguousEffectAttempt(t, "terminal-restart-race")
	quarantinedAt := system.clock.Now().UTC()
	sqliteTestNoError(t, system.repository.QuarantineAction(ctx, application.ActionQuarantinedState{
		Claim: original, ErrorCode: "application.effect_unknown_applied", OperationAt: quarantinedAt,
		Event: application.EventRecord{
			Ref: "event:terminal-restart-race-quarantine", Kind: "action.quarantined",
			GoalRef: original.Action.GoalRef, WorkItemRef: original.Action.WorkItemRef,
			ExecutionRef: original.Action.ExecutionRef, OccurredAt: quarantinedAt,
		},
	}))
	actionBefore, consumptionBefore := terminalOriginalRows(t, system.repository.db, original.Action.Ref)
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Second)
	_, err := system.orchestrator.ReconcileTerminalAgentLaunch(ctx, system.access,
		application.ReconcileTerminalAgentLaunchRequest{
			RequestRef: "request:terminal-restart-race", GoalRef: original.Action.GoalRef,
			WorkItemRef: original.Action.WorkItemRef, ExecutionRef: original.Action.ExecutionRef,
			ActionRef: original.Action.Ref, EffectIntentRef: original.Action.EffectIntentRef,
			EffectIntentDigest: original.Action.EffectIntent.Digest, EffectAttemptRef: physicalAttempt.Ref,
			PlanGeneration:     original.Action.PlanGeneration,
			WorkItemGeneration: original.Action.WorkItemGeneration, ActionFence: physicalAttempt.ActionFence,
		})
	sqliteTestNoError(t, err)

	reconciler := &terminalLaunchReconciler{clock: system.clock}
	restartSQLiteV15System(t, system)
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)
	first := raceRestartRecoveryClaims(t, system.orchestrator)
	if first.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch {
		t.Fatalf("first disposition=%s", first.Disposition)
	}
	if _, err := system.orchestrator.ProcessClaim(ctx, first); err != nil {
		t.Fatalf("temporary reconciliation: %s", sqliteTestErrorChain(err))
	}
	launches, reconciliations, continuations, firstRequests := reconciler.snapshot()
	if launches != 0 || reconciliations != 0 || continuations != 0 || len(firstRequests) != 0 {
		t.Fatalf("before authority Launch=%d ReconcileLaunch=%d ContinueV41=%d requests=%+v",
			launches, reconciliations, continuations, firstRequests)
	}
	var pending, attempts int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
 (SELECT COUNT(*) FROM agent_launch_reconciliation_jobs WHERE state='pending' AND claim_token IS NULL),
 (SELECT COUNT(*) FROM agent_launch_reconciliation_attempts)`).Scan(&pending, &attempts))
	if pending != 1 || attempts != 1 {
		t.Fatalf("after requeue pending=%d attempts=%d", pending, attempts)
	}
	recordTerminalContinuationV41(
		t, system, original, physicalAttempt,
		first.TerminalReconciliationRef, "terminal-restart-race",
	)

	system.clock.Advance(first.Action.EffectIntent.QuotaRetryDelay)
	restartSQLiteV15System(t, system)
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)
	second := raceRestartRecoveryClaims(t, system.orchestrator)
	if second.Disposition != application.ActionClaimDispositionReconcileTerminalLaunch ||
		second.Fence <= first.Fence || second.DeliveryAttempt != first.DeliveryAttempt+1 {
		t.Fatalf("second=%+v first=%+v", second, first)
	}
	if _, err := system.orchestrator.ProcessClaim(ctx, second); err != nil {
		t.Fatalf("successful reconciliation: %s", sqliteTestErrorChain(err))
	}
	launches, reconciliations, continuations, requests := reconciler.snapshot()
	if launches != 0 || reconciliations != 0 || continuations != 1 || len(requests) != 1 ||
		requests[0].IdempotencyKey != physicalAttempt.IdempotencyKey ||
		requests[0].EffectAuthority.EffectAttemptRef != physicalAttempt.Ref {
		t.Fatalf("Launch=%d ReconcileLaunch=%d ContinueV41=%d requests=%+v",
			launches, reconciliations, continuations, requests)
	}
	actionAfter, consumptionAfter := terminalOriginalRows(t, system.repository.db, original.Action.Ref)
	if actionAfter != actionBefore || consumptionAfter != consumptionBefore {
		t.Fatal("restart recovery changed original quarantine evidence")
	}
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM agent_launch_reconciliation_attempts`).Scan(&attempts))
	if attempts != 2 {
		t.Fatalf("reconciliation attempts=%d", attempts)
	}
}

func terminalOriginalRows(t *testing.T, database *sql.DB, actionRef string) (string, string) {
	t.Helper()
	var action, receipt string
	sqliteTestNoError(t, database.QueryRow(`SELECT printf('%s|%s|%s|%s|%s|%d|%d|%s|%s|%d|%d|%d|%d|%s|%s',
ref,kind,goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation,
coalesce(claim_token,''),coalesce(claimed_by,''),coalesce(claimed_until,0),delivery_attempt,fence,
coalesce(completed_at,0),coalesce(quarantined_at,0),last_error_code) FROM outbox WHERE ref=?`, actionRef).Scan(&action))
	sqliteTestNoError(t, database.QueryRow(`SELECT printf('%s|%s|%s|%s|%s|%d|%d|%d|%d|%s|%s|%s|%s|%d|%s',
action_ref,kind,goal_ref,work_item_ref,execution_ref,plan_generation,work_item_generation,fence,
delivery_attempt,claim_token,worker_ref,outcome,error_code,consumed_at,coalesce(effect_receipt_ref,''))
FROM action_consumption_receipts WHERE action_ref=?`, actionRef).Scan(&receipt))
	return action, receipt
}

type terminalLaunchReconciler struct {
	clock           *sqliteMembershipClock
	temporaryFirst  bool
	mu              sync.Mutex
	launches        int
	reconciliations int
	continuations   int
	requests        []ports.AgentLaunchRequest
}

func (reconciler *terminalLaunchReconciler) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteTestCapabilities(), nil
}

func (reconciler *terminalLaunchReconciler) Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	reconciler.launches++
	return ports.AgentLaunchReceipt{}, errors.New("terminal reconciliation called Launch")
}

func (reconciler *terminalLaunchReconciler) ReconcileLaunch(
	_ context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	reconciler.reconciliations++
	return ports.AgentLaunchReceipt{}, errors.New("terminal continuation called ReconcileLaunch")
}

func (reconciler *terminalLaunchReconciler) ContinueExpiredAgentLaunchV41(
	_ context.Context,
	request ports.AgentLaunchRequest,
	_ application.ExpiredAgentLaunchContinuationRecordV41,
) (ports.AgentLaunchReceipt, error) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	reconciler.continuations++
	reconciler.requests = append(reconciler.requests, cloneRestartRecoveryRequest(request))
	if reconciler.temporaryFirst && len(reconciler.requests) == 1 {
		return ports.AgentLaunchReceipt{}, restartRecoveryTemporaryError{}
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: sqliteTestCapabilities().ProviderRef, ModelRef: sqliteTestCapabilities().ModelRef,
		AgentRef:                    sqliteTestCapabilities().AgentRef,
		ExternalRef:                 "external:terminal-reconciled:" + request.ExecutionRef.String(),
		IdempotencyKey:              request.IdempotencyKey,
		ReceiptRef:                  "provider-receipt:terminal-reconciled:" + request.ExecutionRef.String(),
		AcceptedAt:                  reconciler.clock.Now().UTC(),
		RequierePreservacionEntorno: request.RequierePreservacionEntorno,
	}, nil
}

func (reconciler *terminalLaunchReconciler) snapshot() (int, int, int, []ports.AgentLaunchRequest) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	requests := append([]ports.AgentLaunchRequest(nil), reconciler.requests...)
	return reconciler.launches, reconciler.reconciliations, reconciler.continuations, requests
}

func recordTerminalContinuationV41(
	t *testing.T,
	system *sqliteV15System,
	original application.ActionClaim,
	physicalAttempt application.EffectAttempt,
	reconciliationAuthorityRef string,
	suffix string,
) application.ExpiredAgentLaunchContinuationRecordV41 {
	t.Helper()
	var reconciliationAttemptRef string
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT ref
FROM agent_launch_reconciliation_attempts WHERE authority_ref=? ORDER BY started_at DESC LIMIT 1`,
		reconciliationAuthorityRef).Scan(&reconciliationAttemptRef))
	fixture := newV41ContinuationFixture(
		t, system, original, physicalAttempt, reconciliationAuthorityRef,
		reconciliationAttemptRef, suffix,
	)
	record := v41ContinuationRecord(fixture)
	stored, created, err := system.repository.RecordExpiredAgentLaunchContinuationV41(
		context.Background(), record,
	)
	if err != nil || !created || stored.ReconciliationAttemptRef != reconciliationAttemptRef {
		t.Fatalf("record continuation=%+v created=%t err=%s", stored, created, sqliteTestErrorChain(err))
	}
	return stored
}
