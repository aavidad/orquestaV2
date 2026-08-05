package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestV28ValidateAgentLaunchRecoveryClaimFencesRestartExpiryAndReclaim(t *testing.T) {
	system, original, _ := seedV27AmbiguousEffectAttempt(t, "validate-before-effect")
	until := original.LeaseUntil
	if original.EffectApproval.ExpiresAt.After(until) {
		until = original.EffectApproval.ExpiresAt
	}
	system.clock.Advance(until.Sub(system.clock.Now()) + time.Nanosecond)
	first := claimV28RecoveryForValidation(t, system, "claim:v28-validate-first")
	before := v28RecoveryLedgerCounts(t, system.repository.db)
	if err := system.repository.ValidateAgentLaunchRecoveryClaim(context.Background(), first); err != nil {
		t.Fatalf("current recovery claim rejected: %s", sqliteTestErrorChain(err))
	}
	restarted := openSQLiteV15Repository(t, system.path, system.clock.Now)
	if err := restarted.ValidateAgentLaunchRecoveryClaim(context.Background(), first); err != nil {
		t.Fatalf("restart rejected current recovery claim: %s", sqliteTestErrorChain(err))
	}
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("read-only validation mutated ledgers before=%+v after=%+v", before, after)
	}

	system.clock.Advance(first.LeaseUntil.Sub(system.clock.Now()))
	outboxBeforeExpiry := readV28RecoveryOutbox(t, system.repository.db, first.Action.Ref)
	if err := restarted.ValidateAgentLaunchRecoveryClaim(context.Background(), first); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("expired recovery claim error=%s", sqliteTestErrorChain(err))
	}
	if outboxAfter := readV28RecoveryOutbox(t, system.repository.db, first.Action.Ref); outboxAfter != outboxBeforeExpiry {
		t.Fatalf("expired validation mutated outbox before=%+v after=%+v", outboxBeforeExpiry, outboxAfter)
	}

	system.clock.Advance(time.Nanosecond)
	second := claimV28RecoveryForValidation(t, system, "claim:v28-validate-second")
	beforeStaleValidation := readV28RecoveryOutbox(t, system.repository.db, first.Action.Ref)
	if err := restarted.ValidateAgentLaunchRecoveryClaim(context.Background(), first); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("reclaimed stale recovery claim error=%s", sqliteTestErrorChain(err))
	}
	if err := restarted.ValidateAgentLaunchRecoveryClaim(context.Background(), second); err != nil {
		t.Fatalf("current reclaimed recovery claim rejected: %s", sqliteTestErrorChain(err))
	}
	if after := readV28RecoveryOutbox(t, system.repository.db, first.Action.Ref); after != beforeStaleValidation {
		t.Fatalf("stale/current validation mutated outbox before=%+v after=%+v", beforeStaleValidation, after)
	}
}

func TestV28ValidateAgentLaunchRecoveryClaimRejectsCrossedBindings(t *testing.T) {
	system, original, _ := seedV27AmbiguousEffectAttempt(t, "validate-crossed")
	until := original.LeaseUntil
	if original.EffectApproval.ExpiresAt.After(until) {
		until = original.EffectApproval.ExpiresAt
	}
	system.clock.Advance(until.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV28RecoveryForValidation(t, system, "claim:v28-validate-crossed")
	before := v28RecoveryLedgerCounts(t, system.repository.db)
	outboxBefore := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref)
	crossedPlacement, err := ports.NewAgentPlacementRef("placement:v28-crossed")
	sqliteTestNoError(t, err)
	tests := map[string]func(application.ActionClaim) application.ActionClaim{
		"attempt ref": func(value application.ActionClaim) application.ActionClaim {
			value.RecoveryEffectAttemptRef = "effect-attempt:v28-crossed"
			return value
		},
		"placement": func(value application.ActionClaim) application.ActionClaim {
			value.ReferenciaColocacion = crossedPlacement
			return value
		},
		"budget reservation": func(value application.ActionClaim) application.ActionClaim {
			value.BudgetReservation.ReservedAt = value.BudgetReservation.ReservedAt.Add(time.Nanosecond)
			return value
		},
		"capacity reservation": func(value application.ActionClaim) application.ActionClaim {
			value.CapacityReservation.Fence++
			return value
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			if err := system.repository.ValidateAgentLaunchRecoveryClaim(context.Background(), mutate(claim)); !application.IsStateError(err, application.StateConflict) {
				t.Fatalf("crossed claim error=%s", sqliteTestErrorChain(err))
			}
		})
	}
	if after := v28RecoveryLedgerCounts(t, system.repository.db); after != before {
		t.Fatalf("crossed validation mutated ledgers before=%+v after=%+v", before, after)
	}
	if after := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref); after != outboxBefore {
		t.Fatalf("crossed validation mutated outbox before=%+v after=%+v", outboxBefore, after)
	}
}

func TestV28ValidateAgentLaunchRecoveryClaimRejectsPersistedCapacityTransition(t *testing.T) {
	system, original, _ := seedV27AmbiguousEffectAttempt(t, "validate-capacity-transition")
	until := original.LeaseUntil
	if original.EffectApproval.ExpiresAt.After(until) {
		until = original.EffectApproval.ExpiresAt
	}
	system.clock.Advance(until.Sub(system.clock.Now()) + time.Nanosecond)
	claim := claimV28RecoveryForValidation(t, system, "claim:v28-validate-capacity-transition")
	transaction, err := beginTransaction(context.Background(), system.repository)
	sqliteTestNoError(t, err)
	transitionAt := system.clock.Now().Add(time.Nanosecond)
	sqliteTestNoError(t, persistirTransicionCapacidad(
		context.Background(), transaction, claim.CapacityReservation,
		application.AgentCapacityQuarantined, application.AgentCapacityCauseUnknownApplied,
		"cause:v28-capacity-transition", "", "", transitionAt,
	))
	sqliteTestNoError(t, commit(transaction))
	persistedBefore, placementBefore, found, err := leerReservaCapacidadAccion(
		context.Background(), system.repository.db, claim.Action.Ref,
	)
	sqliteTestNoError(t, err)
	if !found || persistedBefore.State != application.AgentCapacityQuarantined ||
		persistedBefore.Revision != claim.CapacityReservation.Revision+1 ||
		persistedBefore.LastTransitionRef == "" || persistedBefore.LastCauseRef == "" ||
		!persistedBefore.UpdatedAt.Equal(transitionAt) || placementBefore != claim.ReferenciaColocacion {
		t.Fatalf("capacity transition not persisted: reservation=%+v placement=%v found=%t",
			persistedBefore, placementBefore, found)
	}
	outboxBefore := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref)
	if err := system.repository.ValidateAgentLaunchRecoveryClaim(context.Background(), claim); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("transitioned capacity claim error=%s", sqliteTestErrorChain(err))
	}
	persistedAfter, placementAfter, found, err := leerReservaCapacidadAccion(
		context.Background(), system.repository.db, claim.Action.Ref,
	)
	sqliteTestNoError(t, err)
	if !found || persistedAfter != persistedBefore || placementAfter != placementBefore {
		t.Fatalf("validation mutated capacity before=%+v/%v after=%+v/%v found=%t",
			persistedBefore, placementBefore, persistedAfter, placementAfter, found)
	}
	if after := readV28RecoveryOutbox(t, system.repository.db, claim.Action.Ref); after != outboxBefore {
		t.Fatalf("validation mutated outbox before=%+v after=%+v", outboxBefore, after)
	}
}

func claimV28RecoveryForValidation(
	t *testing.T,
	system *sqliteV15System,
	token string,
) application.ActionClaim {
	t.Helper()
	claim, found, err := system.repository.ClaimNextAction(
		context.Background(), application.ClaimRequest{
			WorkerRef: "worker:v28-validate", Token: token,
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(),
			BudgetPolicy: system.policy, ExcludeLaunch: true,
		},
	)
	if err != nil || !found || claim.Disposition != application.ActionClaimDispositionRecoverEffect {
		t.Fatalf("recovery claim=%+v found=%t err=%s", claim, found, sqliteTestErrorChain(err))
	}
	return claim
}
