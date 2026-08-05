package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type agentLaunchRecoveryFixture struct {
	record  GoalRecord
	claim   ActionClaim
	request ports.AgentLaunchRequest
	attempt EffectAttempt
}

func newAgentLaunchRecoveryFixture(t *testing.T) agentLaunchRecoveryFixture {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, &scriptedAgent{now: clock.Now})
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:agent-launch-recovery", Statement: "recover exact launch", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	claim, found, err := orchestrator.ClaimNextAction(
		context.Background(), "worker:launch-before-restart", ActionClaimSelection{},
	)
	if err != nil || !found {
		t.Fatalf("claim found=%t err=%v", found, err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	phase, phaseFound := phaseForWorkItem(record.Goal, item)
	if !found || !executionFound || !phaseFound {
		t.Fatal("launch causal records missing")
	}
	execution.BudgetReservationRef = claim.BudgetReservationRef
	execution.EffectIntentRef = claim.Action.EffectIntent.Ref
	record, item, execution, proceed, err := orchestrator.prepareLaunchDispatch(
		context.Background(), claim, record, item, execution,
	)
	if err != nil || !proceed {
		t.Fatalf("prepare proceed=%t err=%v", proceed, err)
	}
	request := agentLaunchRequest(record.Goal, item, execution, phase)
	request.ReferenciaColocacion = claim.ReferenciaColocacion
	request.RequierePreservacionEntorno = execution.RequierePreservacionEntorno
	attempt := EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: claim.Action.EffectIntent.Ref, IntentDigest: claim.Action.EffectIntent.Digest,
		ApprovalRef: claim.EffectApproval.Ref, Subject: claim.Action.EffectIntent.Subject,
		ActionRef: claim.Action.Ref, ActionFence: claim.Fence, WorkerRef: claim.WorkerRef,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		StartedAt:      clock.Now().UTC(), ClaimLeaseUntil: claim.LeaseUntil,
	}
	record.EffectAttempts = append(record.EffectAttempts, attempt)
	claim.Token = "claim:after-restart"
	claim.WorkerRef = "worker:launch-after-restart"
	claim.DeliveryAttempt++
	claim.Fence++
	claim.Disposition = ActionClaimDispositionRecoverEffect
	claim.RecoveryEffectAttemptRef = attempt.Ref
	claim.LeaseUntil = attempt.ClaimLeaseUntil.Add(time.Minute)
	return agentLaunchRecoveryFixture{record: record, claim: claim, request: request, attempt: attempt}
}

func TestAgentLaunchRecoveryBuildsHistoricalAuthorityUnderNewFence(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	request, attempt, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request)
	if err != nil || attempt != fixture.attempt {
		t.Fatalf("attempt=%+v err=%v", attempt, err)
	}
	want := ports.AgentLaunchEffectAuthority{
		AuthorizationReceiptRef: fixture.claim.Action.EffectIntent.Authority.Ref(),
		EffectApprovalRef:       fixture.attempt.ApprovalRef,
		EffectAttemptRef:        fixture.attempt.Ref,
		ActionFence:             fixture.attempt.ActionFence,
		StartedAt:               fixture.attempt.StartedAt,
		ClaimLeaseUntil:         fixture.attempt.ClaimLeaseUntil,
		ApprovalExpiresAt:       fixture.claim.EffectApproval.ExpiresAt,
	}
	if fixture.claim.Fence <= request.EffectAuthority.ActionFence || request.EffectAuthority != want {
		t.Fatalf("claim fence=%d authority=%+v want=%+v", fixture.claim.Fence, request.EffectAuthority, want)
	}
}

func TestAgentLaunchRecoverySelectorRejectsReceiptsAndAmbiguity(t *testing.T) {
	tests := map[string]func(*agentLaunchRecoveryFixture){
		"receipt plus ambiguous": func(fixture *agentLaunchRecoveryFixture) {
			peer := fixture.attempt
			peer.Ref += ":peer"
			peer.ActionFence++
			fixture.record.EffectAttempts = append(fixture.record.EffectAttempts, peer)
			fixture.claim.RecoveryEffectAttemptRef, fixture.claim.Fence = peer.Ref, peer.ActionFence+1
			fixture.record.EffectReceipts = append(fixture.record.EffectReceipts, recoveryEffectReceipt(fixture.attempt))
		},
		"stale crossed receipt": func(fixture *agentLaunchRecoveryFixture) {
			receipt := recoveryEffectReceipt(fixture.attempt)
			receipt.IntentRef = "effect-intent:crossed"
			fixture.record.EffectReceipts = append(fixture.record.EffectReceipts, receipt)
		},
		"two ambiguous": func(fixture *agentLaunchRecoveryFixture) {
			peer := fixture.attempt
			peer.Ref += ":peer"
			peer.ActionFence++
			fixture.record.EffectAttempts = append(fixture.record.EffectAttempts, peer)
			fixture.claim.Fence = peer.ActionFence + 1
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			mutate(&fixture)
			if _, err := SelectAgentLaunchRecoveryAttempt(fixture.record, fixture.claim); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestAgentLaunchRecoverySelectorSkipsOnlyValidZeroRelease(t *testing.T) {
	t.Run("previous exact release", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		fixture.record.EffectAttempts[0].ActionFence++
		fixture.attempt = fixture.record.EffectAttempts[0]
		fixture.claim.Fence = fixture.attempt.ActionFence + 1
		previous := fixture.attempt
		previous.Ref += ":released"
		previous.ActionFence--
		previous.WorkerRef = "worker:historical-released"
		fixture.record.EffectAttempts = append([]EffectAttempt{previous}, fixture.record.EffectAttempts...)
		fixture.record.BudgetSettlements = append(fixture.record.BudgetSettlements,
			exactZeroReleaseForRecovery(fixture.claim.BudgetReservation, previous))
		selected, err := SelectAgentLaunchRecoveryAttempt(fixture.record, fixture.claim)
		if err != nil || selected != fixture.attempt {
			t.Fatalf("selected=%+v err=%v released=%t reservation_fence=%d attempt_fence=%d",
				selected, err, effectAttemptDefinitelyUnapplied(fixture.record, previous),
				fixture.claim.BudgetReservation.Fence, previous.ActionFence)
		}
	})
	t.Run("malformed released attempt", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		fixture.record.EffectAttempts[0].ActionFence++
		fixture.attempt = fixture.record.EffectAttempts[0]
		fixture.claim.Fence = fixture.attempt.ActionFence + 1
		previous := fixture.attempt
		previous.Ref += ":released"
		previous.ActionFence--
		previous.ClaimLeaseUntil = time.Time{}
		fixture.record.EffectAttempts = append([]EffectAttempt{previous}, fixture.record.EffectAttempts...)
		fixture.record.BudgetSettlements = append(fixture.record.BudgetSettlements,
			exactZeroReleaseForRecovery(fixture.claim.BudgetReservation, previous))
		if _, err := SelectAgentLaunchRecoveryAttempt(fixture.record, fixture.claim); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestAgentLaunchRecoveryRejectsLeaseFenceAndCrossBindings(t *testing.T) {
	tests := map[string]func(*agentLaunchRecoveryFixture){
		"lease missing": func(f *agentLaunchRecoveryFixture) { f.record.EffectAttempts[0].ClaimLeaseUntil = time.Time{} },
		"lease before start": func(f *agentLaunchRecoveryFixture) {
			f.record.EffectAttempts[0].ClaimLeaseUntil = f.record.EffectAttempts[0].StartedAt
		},
		"fence not superior": func(f *agentLaunchRecoveryFixture) { f.claim.Fence = f.attempt.ActionFence },
		"subject crossed": func(f *agentLaunchRecoveryFixture) {
			f.record.EffectAttempts[0].Subject.ExecutionRef = mustExecutionRefRecovery(t, "execution:crossed")
		},
		"session crossed": func(f *agentLaunchRecoveryFixture) {
			f.request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:crossed")
		},
		"placement crossed": func(f *agentLaunchRecoveryFixture) {
			f.request.ReferenciaColocacion, _ = ports.NewAgentPlacementRef("placement:crossed")
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			mutate(&fixture)
			if _, _, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestAgentLaunchRecoveryUsesApprovalValidityAtAttemptStart(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	intent := fixture.claim.Action.EffectIntent
	approver := testPrincipal(t, "principal:recovery-approver", "actor:recovery-approver", identity.PrincipalKindHuman)
	approval := fixture.claim.EffectApproval
	approval.RequestRef = "request:recovery-explicit-approval"
	approval.RequestFingerprint = strings.Repeat("e", 64)
	approval.ProposedBy = intent.ProposedBy
	approval.DecidedBy = approver.Ref
	approval.Source = EffectApprovalSourceExplicitDecision
	approval.Reason = "recover historical launch"
	approval.DecidedAt = fixture.attempt.StartedAt
	approval.ExpiresAt = approval.DecidedAt.Add(intent.ApprovalTTL)
	approval.AuthorizationReceipt = effectTestAuthorization(
		t, approver, intent.Subject.ProjectRef, identity.PermissionEffectsApprove,
		effectApprovalResourceRef(intent.Subject.GoalRef, intent.Ref, intent.Digest), approval.DecidedAt,
	)
	if err := ValidateEffectApproval(intent, approval); err != nil {
		t.Fatalf("explicit approval fixture invalid: %v", err)
	}
	fixture.claim.EffectApproval = approval
	fixture.record.EffectApprovals[0] = approval
	fixture.record.EffectAttempts[0].ApprovalRef = approval.Ref
	fixture.attempt = fixture.record.EffectAttempts[0]
	fixture.claim.LeaseUntil = approval.ExpiresAt.Add(time.Hour)
	if _, _, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request); err != nil {
		t.Fatalf("approval expired now rejected: %v", err)
	}
	fixture.record.EffectAttempts[0].StartedAt = approval.ExpiresAt
	fixture.record.EffectAttempts[0].ClaimLeaseUntil = approval.ExpiresAt.Add(time.Minute)
	fixture.claim.RecoveryEffectAttemptRef = fixture.record.EffectAttempts[0].Ref
	if _, _, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
		t.Fatalf("approval invalid at StartedAt error=%v", err)
	}
}

func TestAgentLaunchRecoveryRequiresReconcilerCapability(t *testing.T) {
	launcher := &scriptedAgent{}
	if reconciler, err := AgentLaunchReconcilerFrom(launcher); reconciler != nil ||
		!errors.Is(err, ErrAgentLaunchRecoveryUnsupported) {
		t.Fatalf("reconciler=%T err=%v", reconciler, err)
	}
	capable := &recoveryCapableLauncher{scriptedAgent: launcher}
	if reconciler, err := AgentLaunchReconcilerFrom(capable); err != nil || reconciler != capable {
		t.Fatalf("reconciler=%T err=%v", reconciler, err)
	}
}

type recoveryCapableLauncher struct{ scriptedAgent *scriptedAgent }

func (launcher *recoveryCapableLauncher) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	return launcher.scriptedAgent.Capabilities(ctx)
}
func (launcher *recoveryCapableLauncher) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	return launcher.scriptedAgent.Launch(ctx, request)
}
func (launcher *recoveryCapableLauncher) ReconcileLaunch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	return launchReceiptForRequest(request, request.EffectAuthority.StartedAt), nil
}

func recoveryEffectReceipt(attempt EffectAttempt) EffectReceipt {
	return EffectReceipt{
		Ref: "effect-receipt:" + attempt.Ref, IntentRef: attempt.IntentRef, IntentDigest: attempt.IntentDigest,
		ApprovalRef: attempt.ApprovalRef, AttemptRef: attempt.Ref, Subject: attempt.Subject,
		ActionRef: attempt.ActionRef, ActionFence: attempt.ActionFence, IdempotencyKey: attempt.IdempotencyKey,
		ExternalRef: "receipt:external:" + attempt.Ref, Status: EffectStatusAccepted,
		Usage: unknownUsage(), ConfirmedAt: attempt.StartedAt,
	}
}

func exactZeroReleaseForRecovery(
	reservation governance.BudgetReservation,
	attempt EffectAttempt,
) governance.BudgetSettlement {
	zero := governance.ResourceVector{Currency: reservation.Resources.Currency}
	return governance.BudgetSettlement{
		ReservationRef: reservation.Ref, CausalAttemptRef: attempt.Ref, Reserved: reservation.Resources,
		Observed: governance.ResourceUsage{
			Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
		},
		Charged: zero, Released: reservation.Resources, Overrun: zero, SettledAt: attempt.StartedAt,
	}
}

func mustExecutionRefRecovery(t *testing.T, value string) goal.ExecutionRef {
	t.Helper()
	parsed, err := goal.NewExecutionRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
