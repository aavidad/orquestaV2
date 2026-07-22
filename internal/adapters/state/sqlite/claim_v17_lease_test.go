package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestClaimLeaseUsesSelectedActionKind(t *testing.T) {
	const normalLease = 2 * time.Minute
	const attestLease = 20 * time.Minute

	t.Run("normal action", func(t *testing.T) {
		repository, _ := openTestRepository(t)
		state := newCreateFixture(
			t, "claim-normal-lease", "request:claim-normal-lease", "fingerprint:claim-normal-lease",
			"actor:claim-normal-lease", "project:claim-normal-lease",
		)
		if _, _, err := createLegacyGoal(t, repository, state); err != nil {
			t.Fatal(err)
		}
		at := state.Goal.CreatedAt()
		repository.now = func() time.Time { return at }
		claim := claimV17WithLeases(t, repository, "claim:normal-lease", normalLease, attestLease, sqliteRuntimeTestPolicy())
		if claim.Action.Kind == application.ActionAttestTest || !claim.LeaseUntil.Equal(at.Add(normalLease)) {
			t.Fatalf("normal action received attest lease: kind=%s until=%s", claim.Action.Kind, claim.LeaseUntil)
		}
	})

	t.Run("attest action", func(t *testing.T) {
		system, _ := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
		at := system.clock.Now()
		claim := claimV17WithLeases(
			t, system.repository, "claim:attest-lease", normalLease, attestLease, system.policy,
		)
		if claim.Action.Kind != application.ActionAttestTest || !claim.LeaseUntil.Equal(at.Add(attestLease)) {
			t.Fatalf("attest action lease: kind=%s until=%s want=%s", claim.Action.Kind, claim.LeaseUntil, at.Add(attestLease))
		}
	})
}

func TestAttestClaimLeaseZeroFallsBackToLegacyLease(t *testing.T) {
	const legacyLease = 2 * time.Minute
	system, _ := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	at := system.clock.Now()
	claim := claimV17WithLeases(t, system.repository, "claim:attest-fallback", legacyLease, 0, system.policy)
	if claim.Action.Kind != application.ActionAttestTest || !claim.LeaseUntil.Equal(at.Add(legacyLease)) {
		t.Fatalf("zero special lease did not fall back: kind=%s until=%s want=%s",
			claim.Action.Kind, claim.LeaseUntil, at.Add(legacyLease))
	}
}

func TestExpiredAttestClaimFenceCannotWriteAfterSpecialLeaseRecovery(t *testing.T) {
	const attestLease = 20 * time.Minute
	system, _ := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	oldClaim := claimV17WithLeases(
		t, system.repository, "claim:attest-old", 2*time.Minute, attestLease, system.policy,
	)
	system.clock.Advance(attestLease)
	newClaim := claimV17WithLeases(
		t, system.repository, "claim:attest-new", 2*time.Minute, attestLease, system.policy,
	)
	if newClaim.Action.Ref != oldClaim.Action.Ref || newClaim.Fence != oldClaim.Fence+1 ||
		newClaim.DeliveryAttempt != oldClaim.DeliveryAttempt+1 {
		t.Fatalf("attest recovery changed fencing: old=%+v new=%+v", oldClaim, newClaim)
	}

	before := readClaimV17Row(t, system.repository, newClaim.Action.Ref)
	stale := application.ActionQuarantinedState{
		Claim: oldClaim, ErrorCode: "application.stale_attest_claim", OperationAt: system.clock.Now(),
		Event: application.EventRecord{
			Ref: "event:stale-attest-claim", Kind: "action.quarantined",
			GoalRef: oldClaim.Action.GoalRef, WorkItemRef: oldClaim.Action.WorkItemRef,
			ExecutionRef: oldClaim.Action.ExecutionRef, OccurredAt: system.clock.Now(),
		},
	}
	if err := system.repository.QuarantineAction(context.Background(), stale); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("old attest fence wrote after recovery: %v", err)
	}
	after := readClaimV17Row(t, system.repository, newClaim.Action.Ref)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("old attest fence mutated row: before=%+v after=%+v", before, after)
	}
}

func claimV17WithLeases(
	t *testing.T,
	repository *Repository,
	token string,
	legacyLease, attestLease time.Duration,
	policy application.BudgetPolicy,
) application.ActionClaim {
	t.Helper()
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v17-lease", Token: token, LeaseDuration: legacyLease,
		AttestTestLeaseDuration: attestLease, Capabilities: sqliteTestCapabilities(), BudgetPolicy: policy,
	})
	if err != nil || !found {
		t.Fatalf("claim %s: found=%v err=%v", token, found, err)
	}
	return claim
}

type claimV17Row struct {
	token           string
	worker          string
	leaseUntil      int64
	deliveryAttempt int64
	fence           int64
	completedAt     sql.NullInt64
	quarantinedAt   sql.NullInt64
}

func readClaimV17Row(t *testing.T, repository *Repository, actionRef string) claimV17Row {
	t.Helper()
	var row claimV17Row
	if err := repository.db.QueryRow(`
SELECT claim_token,claimed_by,claimed_until,delivery_attempt,fence,completed_at,quarantined_at
FROM outbox WHERE ref=?`, actionRef).Scan(
		&row.token, &row.worker, &row.leaseUntil, &row.deliveryAttempt, &row.fence,
		&row.completedAt, &row.quarantinedAt,
	); err != nil {
		t.Fatal(err)
	}
	return row
}
