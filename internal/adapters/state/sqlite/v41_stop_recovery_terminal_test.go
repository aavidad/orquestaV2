package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

const v40StopRecoveryClaimMigrationSHA256 = "sha256:981455d3c5ce676120c90ed4d2f61573fef7825a47042a8030e2d20c9b5e88e3"

func TestV41MigrationPreservesV40AndProgressivelyWidensTerminalGuards(t *testing.T) {
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	if len(migrations) != recoverySchemaLatest {
		t.Fatalf("migration count=%d latest=%d", len(migrations), recoverySchemaLatest)
	}
	v40, v41 := migrations[recoverySchemaV38StopRecoveryClaim-1],
		migrations[recoverySchemaV38StopRecoveryTerminal-1]
	if v40.name != "040_stop_recovery_claim.sql" || v40.checksum != v40StopRecoveryClaimMigrationSHA256 ||
		v41.name != "041_stop_recovery_terminal.sql" ||
		!strings.Contains(v41.sql, "action.recovery_effect_attempt_ref=attempt.ref") ||
		!strings.Contains(v41.sql, "NEW.kind='stop_agent'") {
		t.Fatalf("migration V40=%q/%q V41=%q", v40.name, v40.checksum, v41.name)
	}
	path := filepath.Join(t.TempDir(), "canonical-v40.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38StopRecoveryClaim)
	var before string
	sqliteTestNoError(t, database.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='effect_receipts_causal_guard'`).Scan(&before))
	if strings.Contains(before, "action.recovery_effect_attempt_ref=attempt.ref") {
		t.Fatalf("canonical V40 already permits terminal Stop recovery: %s", before)
	}
	sqliteTestNoError(t, database.Close())
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: time.Now,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = repository.Close() })
	var version, receiptV40, receiptV41 int
	var checksumV40, terminalGuard string
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT checksum FROM schema_migrations WHERE version=?`,
		recoverySchemaV38StopRecoveryClaim).Scan(&checksumV40))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38StopRecoveryClaim).Scan(&receiptV40))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`,
		recoverySchemaV38StopRecoveryTerminal).Scan(&receiptV41))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='effect_receipts_causal_guard'`).Scan(&terminalGuard))
	if version != recoverySchemaLatest || receiptV40 != 1 || receiptV41 != 1 ||
		checksumV40 != v40StopRecoveryClaimMigrationSHA256 ||
		!strings.Contains(terminalGuard, "action.recovery_effect_attempt_ref=attempt.ref") {
		t.Fatalf("upgrade version=%d receipts=%d/%d checksum=%q guard=%s",
			version, receiptV40, receiptV41, checksumV40, terminalGuard)
	}
}

func TestV41StopRecoveryTerminalCommitsHistoricalReceiptAfterRestart(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v41-stop-terminal")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v41-stop-terminal")
	var attemptsBefore, receiptsBefore, consumptionsBefore int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
(SELECT COUNT(*) FROM effect_attempts),(SELECT COUNT(*) FROM effect_receipts),
(SELECT COUNT(*) FROM action_consumption_receipts)`).Scan(
		&attemptsBefore, &receiptsBefore, &consumptionsBefore))
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV15Orchestrator(
		t, system.repository, system.clock, system.external, system.policy, system.ids,
	)
	system.external.stopStatus = ports.AgentStopped

	result, err := system.orchestrator.ProcessClaim(context.Background(), claim)
	if err != nil || !result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("ProcessClaim()=%+v err=%s", result, sqliteTestErrorChain(err))
	}
	system.external.mu.Lock()
	stopCalls, reconcileCalls := system.external.stopCalls, system.external.stopReconcileCalls
	system.external.mu.Unlock()
	if stopCalls != 0 || reconcileCalls != 1 {
		t.Fatalf("physical Stop calls=%d read-only reconcile calls=%d", stopCalls, reconcileCalls)
	}
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionStopped {
		t.Fatalf("terminal execution=%+v found=%t", execution, found)
	}
	var effect application.EffectReceipt
	for _, candidate := range record.EffectReceipts {
		if candidate.AttemptRef == attempt.Ref {
			effect = candidate
		}
	}
	var consumed application.ActionConsumptionReceipt
	for _, candidate := range record.ConsumptionReceipts {
		if candidate.ActionRef == claim.Action.Ref {
			consumed = candidate
		}
	}
	if effect.AttemptRef != attempt.Ref || effect.ActionFence != attempt.ActionFence ||
		effect.Status != application.EffectStatusStopped || !effect.ConfirmedAt.After(attempt.ClaimLeaseUntil) ||
		consumed.Fence != claim.Fence || consumed.EffectReceiptRef != effect.Ref {
		t.Fatalf("effect=%+v consumed=%+v attempt=%+v claim=%+v", effect, consumed, attempt, claim)
	}
	var attemptsAfter, receiptsAfter, consumptionsAfter int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
(SELECT COUNT(*) FROM effect_attempts),(SELECT COUNT(*) FROM effect_receipts),
(SELECT COUNT(*) FROM action_consumption_receipts)`).Scan(
		&attemptsAfter, &receiptsAfter, &consumptionsAfter))
	if attemptsAfter != attemptsBefore || receiptsAfter != receiptsBefore+1 ||
		consumptionsAfter < consumptionsBefore+1 {
		t.Fatalf("ledger before=%d/%d/%d after=%d/%d/%d", attemptsBefore, receiptsBefore,
			consumptionsBefore, attemptsAfter, receiptsAfter, consumptionsAfter)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("terminal recovery validation: %s", sqliteTestErrorChain(err))
	}
	sqliteTestNoError(t, system.repository.Close())
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	if _, _, err := validateRecoveryDatabase(context.Background(), restarted.db); err != nil {
		t.Fatalf("terminal recovery restart validation: %s", sqliteTestErrorChain(err))
	}
	replayed, err := restarted.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	if len(replayed.EffectReceipts) != len(record.EffectReceipts) ||
		len(replayed.ConsumptionReceipts) != len(record.ConsumptionReceipts) {
		t.Fatalf("restart lost atomic history before=%+v after=%+v", record, replayed)
	}
}

func TestV41RecoveryRejectsCrossedTerminalStopAuthority(t *testing.T) {
	for _, test := range []struct {
		name, trigger, update, want string
	}{
		{"historical receipt fence", "effect_receipts_immutable_update",
			`UPDATE effect_receipts SET action_fence=action_fence+1 WHERE action_ref=?`,
			"sqlite.recovery_v15_effect_receipt_invalid"},
		{"recovery consumption token", "action_consumption_receipts_immutable_update",
			`UPDATE action_consumption_receipts SET claim_token=claim_token||':crossed' WHERE action_ref=?`,
			"sqlite.recovery_v27_effect_receipt_outside_claim_lease"},
	} {
		t.Run(test.name, func(t *testing.T) {
			system, original, _, _ := seedSQLiteAgentProviderStopRequest(t, "v41-stop-crossed-"+test.name)
			system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
			claim := claimV40StopRecovery(t, system, "claim:v41-stop-crossed-"+test.name)
			system.external.stopStatus = ports.AgentStopped
			if _, err := system.orchestrator.ProcessClaim(context.Background(), claim); err != nil {
				t.Fatalf("settle terminal Stop: %s", sqliteTestErrorChain(err))
			}
			rewriteRecoveryTrigger(t, system.repository.db, test.trigger, func() {
				_, err := system.repository.db.Exec(test.update, claim.Action.Ref)
				sqliteTestNoError(t, err)
			})
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
				!recoveryErrorContains(err, test.want) {
				t.Fatalf("crossed terminal authority recovered: %s", sqliteTestErrorChain(err))
			}
		})
	}
}

func TestV41StopRecoveryTerminalHasSingleAtomicCASWinner(t *testing.T) {
	system, original, attempt, _ := seedSQLiteAgentProviderStopRequest(t, "v41-stop-terminal-cas")
	system.clock.Advance(original.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV40StopRecovery(t, system, "claim:v41-stop-terminal-cas")
	peer := openSQLiteV15Repository(t, system.path, system.clock.Now)
	peerOrchestrator := newSQLiteV15Orchestrator(t, peer, system.clock, system.external, system.policy, system.ids)
	start, gate := make(chan struct{}, 2), make(chan struct{})
	system.external.stopReconcileStart, system.external.stopReconcileGate = start, gate
	type outcome struct{ err error }
	results := make(chan outcome, 2)
	var workers sync.WaitGroup
	for _, orchestrator := range []*application.Orchestrator{system.orchestrator, peerOrchestrator} {
		workers.Add(1)
		go func(orchestrator *application.Orchestrator) {
			defer workers.Done()
			_, err := orchestrator.ProcessClaim(context.Background(), claim)
			results <- outcome{err: err}
		}(orchestrator)
	}
	<-start
	<-start
	close(gate)
	workers.Wait()
	close(results)
	successes, conflicts := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			successes++
		case application.IsStateError(result.err, application.StateConflict):
			conflicts++
		default:
			t.Fatalf("concurrent terminal error=%s", sqliteTestErrorChain(result.err))
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("terminal winners=%d conflicts=%d", successes, conflicts)
	}
	system.external.mu.Lock()
	stopCalls, reconcileCalls := system.external.stopCalls, system.external.stopReconcileCalls
	system.external.mu.Unlock()
	if stopCalls != 0 || reconcileCalls != 2 {
		t.Fatalf("physical Stop calls=%d read-only reconciles=%d", stopCalls, reconcileCalls)
	}
	var effects, consumptions int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT
(SELECT COUNT(*) FROM effect_receipts WHERE attempt_ref=?),
(SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref=?)`,
		attempt.Ref, claim.Action.Ref).Scan(&effects, &consumptions))
	if effects != 1 || consumptions != 1 {
		t.Fatalf("atomic terminal history effects=%d consumptions=%d", effects, consumptions)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("CAS winner recovery validation: %s", sqliteTestErrorChain(err))
	}
}
