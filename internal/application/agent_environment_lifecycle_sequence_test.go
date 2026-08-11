package application

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/ports"
)

type fakeAgentEnvironmentLifecycleReconciler struct {
	quiesceReceipt ports.AgentQuiesceReceipt
	reconcileCalls int
	mutationCalls  int
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) ReconcileQuiesce(
	_ context.Context,
	_ ports.AgentQuiesceRequest,
) (ports.AgentQuiesceReceipt, error) {
	fake.reconcileCalls++
	return fake.quiesceReceipt, nil
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) ReconcilePreserve(
	context.Context,
	ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	fake.reconcileCalls++
	return ports.AgentPreserveReceipt{}, nil
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) ReconcileClose(
	context.Context,
	ports.AgentCloseRequest,
) (ports.AgentCloseReceipt, error) {
	fake.reconcileCalls++
	return ports.AgentCloseReceipt{}, nil
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) Inspect(
	context.Context,
	ports.AgentEnvironmentInspectRequest,
) (ports.AgentEnvironmentInspectReceipt, error) {
	return ports.AgentEnvironmentInspectReceipt{}, nil
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) Quiesce(
	context.Context,
	ports.AgentQuiesceRequest,
) (ports.AgentQuiesceReceipt, error) {
	fake.mutationCalls++
	return ports.AgentQuiesceReceipt{}, nil
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) Preserve(
	context.Context,
	ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	fake.mutationCalls++
	return ports.AgentPreserveReceipt{}, nil
}

func (fake *fakeAgentEnvironmentLifecycleReconciler) Close(
	context.Context,
	ports.AgentCloseRequest,
) (ports.AgentCloseReceipt, error) {
	fake.mutationCalls++
	return ports.AgentCloseReceipt{}, nil
}

var _ ports.AgentEnvironmentLifecycleReconciler = (*fakeAgentEnvironmentLifecycleReconciler)(nil)
var _ ports.AgentEnvironmentLifecycle = (*fakeAgentEnvironmentLifecycleReconciler)(nil)

type fakeAgentEnvironmentLifecycleTerminalWriter struct {
	current      AgentEnvironmentLifecycleSnapshot
	effects      []EffectReceipt
	consumptions []ActionConsumptionReceipt
	writes       int
}

func (fake *fakeAgentEnvironmentLifecycleTerminalWriter) RecordAgentEnvironmentLifecycleTerminal(
	_ context.Context,
	state AgentEnvironmentLifecyclePostEffectState,
) (AgentEnvironmentLifecycleSnapshot, bool, error) {
	if reflect.DeepEqual(fake.current, state.Snapshot) {
		return fake.current, false, nil
	}
	if validateAgentEnvironmentLifecyclePostEffect(state) != nil ||
		fake.current.Revision != state.ExpectedRevision || !fake.current.Effect.NeedsReconciliation() ||
		fake.current.Effect.AttemptRef != state.Attempt.Ref ||
		fake.current.Effect.ActionFence != state.Attempt.ActionFence ||
		fake.current.Effect.ActionRef != state.Snapshot.Effect.ActionRef ||
		fake.current.Effect.ActionKind != state.Snapshot.Effect.ActionKind ||
		fake.current.Effect.DeliveryAttempt != state.Snapshot.Effect.DeliveryAttempt ||
		fake.current.Effect.WorkItemGeneration != state.Snapshot.Effect.WorkItemGeneration ||
		fake.current.Effect.ClaimToken != state.Snapshot.Effect.ClaimToken ||
		fake.current.Effect.WorkerRef != state.Snapshot.Effect.WorkerRef ||
		fake.current.Effect.IdempotencyKey != state.Snapshot.Effect.IdempotencyKey ||
		fake.current.Effect.ExpectedToken != state.Snapshot.Effect.ExpectedToken {
		return AgentEnvironmentLifecycleSnapshot{}, false, errAgentEnvironmentLifecycleStateInvalid
	}
	fake.current = state.Snapshot
	fake.effects = append(fake.effects, *state.EffectReceipt)
	fake.consumptions = append(fake.consumptions, state.ConsumptionReceipt)
	fake.writes++
	return fake.current, true, nil
}

var _ AgentEnvironmentLifecycleTerminalWriter = (*fakeAgentEnvironmentLifecycleTerminalWriter)(nil)

func TestAgentEnvironmentLifecyclePrepareRequiresDeliveryAndWorkItemGenerations(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	quiesced := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiesced, "revision:claim-quiesced")
	preserveCurrent := fixture.initial
	preserveCurrent.Revision = 3
	preserveCurrent.Token = quiesced
	preserveCurrent.Effect = AgentEnvironmentLifecycleEffectSnapshot{
		ActionRef: "action:claim-quiesce", ActionKind: ActionQuiesceAgent,
		AttemptRef: "effect-attempt:action:claim-quiesce:claim:claim-quiesce", ActionFence: 1,
		DeliveryAttempt: 1, WorkItemGeneration: fixture.base.Action.WorkItemGeneration,
		ClaimToken: "claim:claim-quiesce", WorkerRef: "worker:claim-quiesce",
		IdempotencyKey: "idempotency:claim-quiesce", ExpectedToken: fixture.initial.Token,
		OutcomeToken: quiesced, PhysicalReceipt: "receipt:claim-quiesce",
		EffectReceiptRef: "effect-receipt:claim-quiesce",
	}
	preserveCurrent.RecordedAt = fixture.now.Add(time.Second)
	preserved := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentPreserved, "revision:claim-preserved")
	closeCurrent := preserveCurrent
	closeCurrent.Revision = 5
	closeCurrent.Token = preserved
	closeCurrent.Effect = AgentEnvironmentLifecycleEffectSnapshot{
		ActionRef: "action:claim-preserve", ActionKind: ActionPreserveAgentEnvironment,
		AttemptRef: "effect-attempt:action:claim-preserve:claim:claim-preserve", ActionFence: 2,
		DeliveryAttempt: 1, WorkItemGeneration: fixture.base.Action.WorkItemGeneration,
		ClaimToken: "claim:claim-preserve", WorkerRef: "worker:claim-preserve",
		IdempotencyKey: "idempotency:claim-preserve", ExpectedToken: quiesced,
		OutcomeToken: preserved, PhysicalReceipt: "receipt:claim-preserve",
		EffectReceiptRef: "effect-receipt:claim-preserve",
	}
	closeCurrent.Preservation = ports.AgentPreservationBinding{
		ApplicationReceiptRef: "environment-receipt:claim-preserve",
		PhysicalManifest: ports.AgentPhysicalPreservationBinding{
			ManifestRef: "physical-manifest:claim-preserve", ManifestSHA256: strings.Repeat("a", 64),
		},
	}
	closeCurrent.PreservationReceiptRef = "receipt:claim-preserve"
	closeCurrent.RecordedAt = fixture.now.Add(2 * time.Second)

	for _, test := range []struct {
		name     string
		kind     ActionKind
		snapshot AgentEnvironmentLifecycleSnapshot
	}{
		{name: "quiesce", kind: ActionQuiesceAgent, snapshot: fixture.initial},
		{name: "preserve", kind: ActionPreserveAgentEnvironment, snapshot: preserveCurrent},
		{name: "close", kind: ActionCloseAgentEnvironment, snapshot: closeCurrent},
	} {
		t.Run(test.name, func(t *testing.T) {
			at := test.snapshot.RecordedAt.Add(time.Second)
			valid := lifecycleClaimFor(t, fixture.base, test.snapshot, test.kind, 9, at)
			for _, mutate := range []struct {
				name string
				fn   func(*ActionClaim)
			}{
				{name: "zero delivery attempt", fn: func(claim *ActionClaim) { claim.DeliveryAttempt = 0 }},
				{name: "zero work item generation", fn: func(claim *ActionClaim) {
					claim.Action.WorkItemGeneration = 0
				}},
				{name: "change scoped action", fn: func(claim *ActionClaim) {
					claim.Action.ChangeRef, _ = ports.NewChangeSetRef("change:lifecycle-crossed")
				}},
			} {
				t.Run(mutate.name, func(t *testing.T) {
					claim := valid
					mutate.fn(&claim)
					if _, err := PrepareAgentEnvironmentLifecycleEffect(test.snapshot, claim, at); err == nil {
						t.Fatal("zero claim generation reached EffectAttempt")
					}
				})
			}
		})
	}
}

func TestAgentEnvironmentLifecycleSequenceUsesRealPreservationFact(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	record := fixture.record

	quiesceClaim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 11, fixture.now)
	quiescePre := mustPrepareLifecycle(t, fixture.initial, quiesceClaim, fixture.now)
	appendLifecyclePreparedFacts(&record, quiescePre)
	quiesced := lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentQuiesced, "revision:2")
	quiesceOutcome, err := RecordAgentEnvironmentQuiesceOutcome(quiescePre, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token, NextToken: quiesced,
		IdempotencyKey: quiescePre.Attempt.IdempotencyKey, ReceiptRef: "receipt:quiesce",
		ConfirmedAt: fixture.now.Add(time.Second),
	}, fixture.now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	quiescePost := mustLifecycleTerminalOutcome(t, quiesceOutcome)
	if quiescePost.EffectReceipt.Status != EffectStatusQuiesced {
		t.Fatalf("quiesce post=%+v", quiescePost)
	}
	appendLifecycleTerminalFacts(&record, quiescePost)

	preserveClaim := lifecycleClaimFor(t, fixture.base, quiescePost.Snapshot,
		ActionPreserveAgentEnvironment, 12, fixture.now.Add(3*time.Second))
	preservePre := mustPrepareLifecycle(t, quiescePost.Snapshot, preserveClaim, fixture.now.Add(3*time.Second))
	appendLifecyclePreparedFacts(&record, preservePre)
	manifest := lifecycleManifest(t, preservePre.Snapshot.Subject, fixture.now.Add(4*time.Second))
	preserved := lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentPreserved, "revision:3")
	preserveReceipt := ports.AgentPreserveReceipt{
		Subject: fixture.initial.Subject, PreviousToken: quiesced, NextToken: preserved,
		IdempotencyKey: preservePre.Attempt.IdempotencyKey, Manifest: manifest,
		ReceiptRef: "receipt:preserve", ConfirmedAt: fixture.now.Add(5 * time.Second),
	}
	preservation := lifecyclePreservationFact(t, record, preserveReceipt, fixture.now.Add(6*time.Second))
	preserveOutcome, err := RecordAgentEnvironmentPreserveOutcome(
		preservePre, preserveReceipt, &preservation, record, fixture.now.Add(7*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	preservePost := mustLifecycleTerminalOutcome(t, preserveOutcome)
	if preservePost.PreservationFact == nil {
		t.Fatalf("preserve post=%+v", preservePost)
	}
	if preservePost.Snapshot.Preservation.ApplicationReceiptRef != preservation.Ref ||
		preservePost.Snapshot.Preservation.PhysicalManifest.ManifestRef != manifest.Ref ||
		preservePost.Snapshot.Preservation.PhysicalManifest.ManifestSHA256 != manifest.SHA256 {
		t.Fatalf("preservation binding was not derived from facts: %+v", preservePost.Snapshot.Preservation)
	}
	appendLifecycleTerminalFacts(&record, preservePost)

	closeClaim := lifecycleClaimFor(t, fixture.base, preservePost.Snapshot,
		ActionCloseAgentEnvironment, 13, fixture.now.Add(8*time.Second))
	closePre := mustPrepareLifecycle(t, preservePost.Snapshot, closeClaim, fixture.now.Add(8*time.Second))
	if closePre.RequiredApplicationReceiptRef != preservation.Ref {
		t.Fatalf("close application receipt=%q", closePre.RequiredApplicationReceiptRef)
	}
	appendLifecyclePreparedFacts(&record, closePre)
	closed := lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentClosed, "revision:4")
	closeOutcome, err := RecordAgentEnvironmentCloseOutcome(closePre, ports.AgentCloseReceipt{
		Subject: fixture.initial.Subject, PreviousToken: preserved, NextToken: closed,
		Preservation: preservePost.Snapshot.Preservation, IdempotencyKey: closePre.Attempt.IdempotencyKey,
		ReceiptRef: "receipt:close", ConfirmedAt: fixture.now.Add(9 * time.Second),
	}, fixture.now.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	closePost := mustLifecycleTerminalOutcome(t, closeOutcome)
	if closePost.EffectReceipt.Status != EffectStatusClosed {
		t.Fatalf("close post=%+v", closePost)
	}
	appendLifecycleTerminalFacts(&record, closePost)
	replayed, err := ReplayAgentEnvironmentLifecycleSnapshot(
		closePost.Snapshot, fixture.request, fixture.launch, record, &preservation,
	)
	if err != nil || !reflect.DeepEqual(replayed, closePost.Snapshot) {
		t.Fatalf("exact replay failed: replayed=%+v err=%v", replayed, err)
	}
}

func TestAgentEnvironmentLifecycleAttemptedAndPendingOnlyReconcileSameAttempt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 21, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)

	newClaim := lifecycleClaimFor(t, fixture.base, prepared.Snapshot,
		ActionQuiesceAgent, 22, fixture.now.Add(time.Second))
	if _, err := PrepareAgentEnvironmentLifecycleEffect(prepared.Snapshot, newClaim, fixture.now.Add(time.Second)); err == nil {
		t.Fatal("attempted frontier created a second physical attempt")
	}
	reconcile, err := AgentEnvironmentLifecycleReconciliationRequest(prepared.Snapshot)
	if err != nil || reconcile.AttemptRef != prepared.Attempt.Ref || reconcile.ActionFence != prepared.Attempt.ActionFence ||
		reconcile.InspectRequest.Subject != fixture.initial.Subject || reconcile.ReceiptRequest.Quiesce == nil ||
		reconcile.ReceiptRequest.Quiesce.ExpectedToken != fixture.initial.Token ||
		reconcile.ReceiptRequest.Preserve != nil || reconcile.ReceiptRequest.Close != nil {
		t.Fatalf("attempted reconciliation=%+v err=%v", reconcile, err)
	}

	pendingOutcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken:      lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentQuiescing, "revision:2"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}, fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	pendingReconcile := mustLifecycleReconciliationOutcome(t, pendingOutcome)
	if !reflect.DeepEqual(pendingReconcile, reconcile) {
		t.Fatalf("pending reply changed durable reconciliation authority: pending=%+v attempted=%+v",
			pendingReconcile, reconcile)
	}
	if _, err := PrepareAgentEnvironmentLifecycleEffect(
		prepared.Snapshot, newClaim, fixture.now.Add(2*time.Second),
	); err == nil {
		t.Fatal("non-persisted pending reply created a second attempt")
	}

	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)
	replayed, err := ReplayAgentEnvironmentLifecycleSnapshot(
		prepared.Snapshot, fixture.request, fixture.launch, record, nil,
	)
	if err != nil || !reflect.DeepEqual(replayed, prepared.Snapshot) {
		t.Fatalf("attempted restart after pending failed: replayed=%+v err=%v", replayed, err)
	}
	forgedPending := prepared.Snapshot
	forgedPending.Revision++
	forgedPending.Token = lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiescing, "revision:forged-pending")
	forgedPending.Effect.OutcomeToken = forgedPending.Token
	forgedPending.RecordedAt = fixture.now.Add(time.Second)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		forgedPending, fixture.request, fixture.launch, record, nil,
	); err == nil {
		t.Fatal("persisted pending OutcomeToken replayed")
	}
}

func TestAgentEnvironmentLifecycleTerminalReceiptAfterLeaseResolvesExactAttempt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 23, fixture.now)
	claim.LeaseUntil = fixture.now.Add(time.Minute)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)

	if _, err := PrepareAgentEnvironmentLifecycleEffect(
		prepared.Snapshot,
		lifecycleClaimFor(t, fixture.base, prepared.Snapshot, ActionQuiesceAgent, 24, claim.LeaseUntil.Add(time.Second)),
		claim.LeaseUntil.Add(time.Second),
	); err == nil {
		t.Fatal("expired unresolved attempt authorized a replacement physical effect")
	}

	confirmedAt := claim.LeaseUntil.Add(time.Second)
	outcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentQuiesced, "revision:after-lease"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:quiesce:after-lease", ConfirmedAt: confirmedAt,
	}, confirmedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	if post.EffectReceipt.AttemptRef != prepared.Attempt.Ref || !post.EffectReceipt.ConfirmedAt.Equal(confirmedAt) {
		t.Fatalf("late terminal receipt did not resolve same attempt: post=%+v", post)
	}
	writer := &fakeAgentEnvironmentLifecycleTerminalWriter{current: prepared.Snapshot}
	physical := &fakeAgentEnvironmentLifecycleReconciler{}
	persisted, changed, err := writer.RecordAgentEnvironmentLifecycleTerminal(context.Background(), post)
	if err != nil || !changed || !reflect.DeepEqual(persisted, post.Snapshot) || writer.writes != 1 ||
		len(writer.effects) != 1 || len(writer.consumptions) != 1 || physical.mutationCalls != 0 {
		t.Fatalf("late terminal CAS persisted=%+v changed=%t writer=%+v physical_calls=%d err=%v",
			persisted, changed, writer, physical.mutationCalls, err)
	}
	if replayedCAS, changed, err := writer.RecordAgentEnvironmentLifecycleTerminal(context.Background(), post); err != nil ||
		changed || !reflect.DeepEqual(replayedCAS, post.Snapshot) || writer.writes != 1 {
		t.Fatalf("terminal CAS replay changed=%t writes=%d snapshot=%+v err=%v",
			changed, writer.writes, replayedCAS, err)
	}
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)
	appendLifecycleTerminalFacts(&record, post)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		post.Snapshot, fixture.request, fixture.launch, record, nil,
	); err != nil {
		t.Fatalf("late receipt history did not replay exact attempt: %v", err)
	}
}

func TestAgentEnvironmentLifecyclePendingReconciliationReusesExactAttempt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 31, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	pendingToken := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiescing, "revision:pending-restart")
	pendingOutcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token, NextToken: pendingToken,
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}, fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	reconciliation := mustLifecycleReconciliationOutcome(t, pendingOutcome)
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)
	replayedAttempted, err := ReplayAgentEnvironmentLifecycleSnapshot(
		prepared.Snapshot, fixture.request, fixture.launch, record, nil,
	)
	if err != nil {
		t.Fatalf("attempted restart after pending failed: %v", err)
	}

	terminalToken := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiesced, "revision:terminal-after-restart")
	receipt := ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token, NextToken: terminalToken,
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:quiesce:after-pending", ConfirmedAt: fixture.now.Add(2 * time.Second),
	}
	fake := &fakeAgentEnvironmentLifecycleReconciler{quiesceReceipt: receipt}
	historical, err := fake.ReconcileQuiesce(context.Background(), *reconciliation.ReceiptRequest.Quiesce)
	if err != nil || historical != receipt || fake.reconcileCalls != 1 || fake.mutationCalls != 0 {
		t.Fatalf("read-only reconciliation crossed mutation boundary: historical=%+v fake=%+v err=%v",
			historical, fake, err)
	}
	inspection := ports.AgentEnvironmentInspectReceipt{Subject: fixture.initial.Subject, Token: terminalToken}
	resolvedAt := fixture.now.Add(3 * time.Second)
	resolved, err := RecordAgentEnvironmentQuiesceReconciliationOutcome(
		replayedAttempted, claim, prepared.Attempt, inspection, historical, resolvedAt,
	)
	if err != nil || resolved.ExpectedRevision != prepared.Snapshot.Revision ||
		resolved.Snapshot.Revision != prepared.Snapshot.Revision+1 || resolved.Attempt != prepared.Attempt ||
		resolved.Snapshot.Effect.AttemptRef != prepared.Attempt.Ref ||
		resolved.Snapshot.Effect.ActionFence != prepared.Attempt.ActionFence ||
		resolved.Snapshot.Effect.IdempotencyKey != prepared.Attempt.IdempotencyKey ||
		resolved.ConsumptionReceipt.EffectReceiptRef != resolved.EffectReceipt.Ref {
		t.Fatalf("pending resolution invented authority: resolved=%+v err=%v", resolved, err)
	}
	repeated, err := RecordAgentEnvironmentQuiesceReconciliationOutcome(
		replayedAttempted, claim, prepared.Attempt, inspection, receipt, resolvedAt,
	)
	if err != nil || !reflect.DeepEqual(repeated, resolved) {
		t.Fatalf("exact reconciliation was not idempotent: repeated=%+v err=%v", repeated, err)
	}
	beforeAttempts := len(record.EffectAttempts)
	appendLifecycleTerminalFacts(&record, resolved)
	if len(record.EffectAttempts) != beforeAttempts {
		t.Fatal("terminal reconciliation created a new attempt")
	}
	replayedTerminal, err := ReplayAgentEnvironmentLifecycleSnapshot(
		resolved.Snapshot, fixture.request, fixture.launch, record, nil,
	)
	if err != nil || !reflect.DeepEqual(replayedTerminal, resolved.Snapshot) {
		t.Fatalf("resolved terminal replay failed: replayed=%+v err=%v", replayedTerminal, err)
	}
}

func TestAgentEnvironmentLifecyclePendingReconciliationRejectsDivergence(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 41, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	pendingOutcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentQuiescing, "revision:pending-divergence"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}, fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	mustLifecycleReconciliationOutcome(t, pendingOutcome)
	terminalToken := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiesced, "revision:terminal-divergence")
	inspection := ports.AgentEnvironmentInspectReceipt{Subject: fixture.initial.Subject, Token: terminalToken}
	receipt := ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token, NextToken: terminalToken,
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:quiesce:divergence", ConfirmedAt: fixture.now.Add(2 * time.Second),
	}

	tests := []struct {
		name   string
		mutate func(*ActionClaim, *EffectAttempt, *ports.AgentEnvironmentInspectReceipt, *ports.AgentQuiesceReceipt)
	}{
		{name: "receipt token differs from inspection", mutate: func(_ *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, receipt *ports.AgentQuiesceReceipt,
		) {
			receipt.NextToken = lifecycleToken(t, fixture.initial.Subject.ExternalRef,
				ports.AgentEnvironmentQuiesced, "revision:receipt-peer")
		}},
		{name: "receipt identity crossed", mutate: func(_ *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, receipt *ports.AgentQuiesceReceipt,
		) {
			receipt.Subject.AgentRef = "agent:lifecycle:peer"
		}},
		{name: "receipt previous token crossed", mutate: func(_ *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, receipt *ports.AgentQuiesceReceipt,
		) {
			receipt.PreviousToken = lifecycleToken(t, fixture.initial.Subject.ExternalRef,
				ports.AgentEnvironmentActive, "revision:previous-peer")
		}},
		{name: "receipt idempotency crossed", mutate: func(_ *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, receipt *ports.AgentQuiesceReceipt,
		) {
			receipt.IdempotencyKey = "idempotency:quiesce:peer"
		}},
		{name: "physical fence crossed", mutate: func(_ *ActionClaim, _ *EffectAttempt,
			inspection *ports.AgentEnvironmentInspectReceipt, receipt *ports.AgentQuiesceReceipt,
		) {
			fence, fenceErr := ports.NewAgentPhysicalFence("physical-fence:peer")
			if fenceErr != nil {
				t.Fatal(fenceErr)
			}
			inspection.Token.Fence = fence
			receipt.NextToken = inspection.Token
		}},
		{name: "attempt fence crossed", mutate: func(_ *ActionClaim, attempt *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, _ *ports.AgentQuiesceReceipt,
		) {
			attempt.ActionFence++
		}},
		{name: "delivery attempt crossed", mutate: func(claim *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, _ *ports.AgentQuiesceReceipt,
		) {
			claim.DeliveryAttempt = 999
		}},
		{name: "work item generation crossed", mutate: func(claim *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, _ *ports.AgentQuiesceReceipt,
		) {
			claim.Action.WorkItemGeneration++
		}},
		{name: "claim token crossed", mutate: func(claim *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, _ *ports.AgentQuiesceReceipt,
		) {
			claim.Token = "claim:lifecycle:peer"
		}},
		{name: "worker crossed", mutate: func(claim *ActionClaim, _ *EffectAttempt,
			_ *ports.AgentEnvironmentInspectReceipt, _ *ports.AgentQuiesceReceipt,
		) {
			claim.WorkerRef = "worker:lifecycle:peer"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changedClaim, changedAttempt := claim, prepared.Attempt
			changedInspection, changedReceipt := inspection, receipt
			test.mutate(&changedClaim, &changedAttempt, &changedInspection, &changedReceipt)
			if _, err := RecordAgentEnvironmentQuiesceReconciliationOutcome(
				prepared.Snapshot, changedClaim, changedAttempt, changedInspection, changedReceipt,
				fixture.now.Add(3*time.Second),
			); err == nil {
				t.Fatal("divergent pending reconciliation accepted")
			}
		})
	}
}

func TestAgentEnvironmentLifecyclePreplannedFactsDoNotAdvancePhysicalFrontier(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	plannedAt := fixture.launch.AcceptedAt.Add(-time.Second)
	quiesceClaim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 51, plannedAt)
	preserveClaim := lifecycleClaimFor(t, fixture.base, fixture.initial,
		ActionPreserveAgentEnvironment, 52, plannedAt)
	closeClaim := lifecycleClaimFor(t, fixture.base, fixture.initial,
		ActionCloseAgentEnvironment, 53, plannedAt)
	record := fixture.record
	for _, claim := range []ActionClaim{quiesceClaim, preserveClaim, closeClaim} {
		record.EffectIntents = append(record.EffectIntents, claim.Action.EffectIntent)
		record.EffectApprovals = append(record.EffectApprovals, claim.EffectApproval)
	}
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		fixture.initial, fixture.request, fixture.launch, record, nil,
	); err != nil {
		t.Fatalf("preplanned lifecycle facts advanced active frontier: %v", err)
	}

	quiescePre := mustPrepareLifecycle(t, fixture.initial, quiesceClaim, fixture.now)
	record.EffectAttempts = append(record.EffectAttempts, quiescePre.Attempt)
	quiesceOutcome, err := RecordAgentEnvironmentQuiesceOutcome(quiescePre, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentQuiesced, "revision:preplanned-quiesce"),
		IdempotencyKey: quiescePre.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:quiesce:preplanned", ConfirmedAt: fixture.now.Add(time.Second),
	}, fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	quiescePost := mustLifecycleTerminalOutcome(t, quiesceOutcome)
	appendLifecycleTerminalFacts(&record, quiescePost)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		quiescePost.Snapshot, fixture.request, fixture.launch, record, nil,
	); err != nil {
		t.Fatalf("future preserve/close plans broke quiesced replay: %v", err)
	}

	preservePre := mustPrepareLifecycle(t, quiescePost.Snapshot, preserveClaim, fixture.now.Add(2*time.Second))
	record.EffectAttempts = append(record.EffectAttempts, preservePre.Attempt)
	manifest := lifecycleManifest(t, fixture.initial.Subject, fixture.now.Add(3*time.Second))
	preserveReceipt := ports.AgentPreserveReceipt{
		Subject: fixture.initial.Subject, PreviousToken: quiescePost.Snapshot.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentPreserved, "revision:preplanned-preserve"),
		IdempotencyKey: preservePre.Attempt.IdempotencyKey, Manifest: manifest,
		ReceiptRef: "receipt:preserve:preplanned", ConfirmedAt: fixture.now.Add(4 * time.Second),
	}
	preservation := lifecyclePreservationFact(t, record, preserveReceipt, fixture.now.Add(4*time.Second))
	preserveOutcome, err := RecordAgentEnvironmentPreserveOutcome(
		preservePre, preserveReceipt, &preservation, record, fixture.now.Add(4*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	preservePost := mustLifecycleTerminalOutcome(t, preserveOutcome)
	appendLifecycleTerminalFacts(&record, preservePost)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		preservePost.Snapshot, fixture.request, fixture.launch, record, &preservation,
	); err != nil {
		t.Fatalf("future close plan broke preserved replay: %v", err)
	}

	closePre := mustPrepareLifecycle(t, preservePost.Snapshot, closeClaim, fixture.now.Add(5*time.Second))
	record.EffectAttempts = append(record.EffectAttempts, closePre.Attempt)
	closeOutcome, err := RecordAgentEnvironmentCloseOutcome(closePre, ports.AgentCloseReceipt{
		Subject: fixture.initial.Subject, PreviousToken: preservePost.Snapshot.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentClosed, "revision:preplanned-close"),
		Preservation:   preservePost.Snapshot.Preservation,
		IdempotencyKey: closePre.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:close:preplanned", ConfirmedAt: fixture.now.Add(6 * time.Second),
	}, fixture.now.Add(6*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	closePost := mustLifecycleTerminalOutcome(t, closeOutcome)
	appendLifecycleTerminalFacts(&record, closePost)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		closePost.Snapshot, fixture.request, fixture.launch, record, &preservation,
	); err != nil {
		t.Fatalf("preplanned sequential lifecycle failed exact replay: %v", err)
	}
}

func TestAgentEnvironmentLifecycleReplayRejectsAttemptBeforeExecutionStarted(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	executionStartedAt := fixture.launch.AcceptedAt.Add(2 * time.Second)
	lifecycleFixtureExecution(t, &fixture).StartedAt = executionStartedAt
	attemptedAt := fixture.launch.AcceptedAt.Add(1500 * time.Millisecond)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 61, attemptedAt)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, attemptedAt)
	outcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentQuiesced, "revision:before-execution-start"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:quiesce:before-execution-start",
		ConfirmedAt:    executionStartedAt.Add(time.Second),
	}, executionStartedAt.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)
	appendLifecycleTerminalFacts(&record, post)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		post.Snapshot, fixture.request, fixture.launch, record, nil,
	); err == nil {
		t.Fatal("lifecycle attempt before durable execution start replayed")
	}
}

func TestAgentEnvironmentLifecyclePreserveAndCloseResolvePendingWithoutNewAttempts(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	preservePre, preserveReceipt, record := lifecyclePreparedPreserve(t, fixture)
	preserveAttemptedReconciliation, err := AgentEnvironmentLifecycleReconciliationRequest(preservePre.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	preservePendingToken := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentPreserving, "revision:preserve-pending-restart")
	preservePendingOutcome, err := RecordAgentEnvironmentPreserveOutcome(
		preservePre,
		ports.AgentPreserveReceipt{
			Subject: fixture.initial.Subject, PreviousToken: preservePre.Snapshot.Effect.ExpectedToken,
			NextToken: preservePendingToken, IdempotencyKey: preservePre.Attempt.IdempotencyKey,
		},
		nil, record, fixture.now.Add(4*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	preserveReconciliation := mustLifecycleReconciliationOutcome(t, preservePendingOutcome)
	if !reflect.DeepEqual(preserveReconciliation, preserveAttemptedReconciliation) ||
		preserveReconciliation.ReceiptRequest.Preserve == nil ||
		preserveReconciliation.ReceiptRequest.Quiesce != nil || preserveReconciliation.ReceiptRequest.Close != nil ||
		preserveReconciliation.ReceiptRequest.Preserve.Subject != fixture.initial.Subject ||
		preserveReconciliation.ReceiptRequest.Preserve.ExpectedToken != preservePre.Snapshot.Effect.ExpectedToken ||
		preserveReconciliation.ReceiptRequest.Preserve.IdempotencyKey != preservePre.Attempt.IdempotencyKey {
		t.Fatalf("preserve reconciliation request crossed: %+v", preserveReconciliation)
	}
	replayedPending, err := ReplayAgentEnvironmentLifecycleSnapshot(
		preservePre.Snapshot, fixture.request, fixture.launch, record, nil,
	)
	if err != nil {
		t.Fatalf("preserve pending restart replay failed: %v", err)
	}
	preservation := lifecyclePreservationFact(t, record, preserveReceipt, fixture.now.Add(6*time.Second))
	preserveResolved, err := RecordAgentEnvironmentPreserveReconciliationOutcome(
		replayedPending, preservePre.Claim, preservePre.Attempt,
		ports.AgentEnvironmentInspectReceipt{Subject: fixture.initial.Subject, Token: preserveReceipt.NextToken},
		preserveReceipt, &preservation, record, fixture.now.Add(7*time.Second),
	)
	if err != nil || preserveResolved.Attempt != preservePre.Attempt ||
		preserveResolved.ExpectedRevision != preservePre.Snapshot.Revision ||
		preserveResolved.ConsumptionReceipt.EffectReceiptRef != preserveResolved.EffectReceipt.Ref {
		t.Fatalf("preserve pending resolution failed: resolved=%+v err=%v", preserveResolved, err)
	}
	appendLifecycleTerminalFacts(&record, preserveResolved)

	closeClaim := lifecycleClaimFor(t, fixture.base, preserveResolved.Snapshot,
		ActionCloseAgentEnvironment, 71, fixture.now.Add(8*time.Second))
	closePre := mustPrepareLifecycle(t, preserveResolved.Snapshot, closeClaim, fixture.now.Add(8*time.Second))
	appendLifecyclePreparedFacts(&record, closePre)
	closeAttemptedReconciliation, err := AgentEnvironmentLifecycleReconciliationRequest(closePre.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	closePendingToken := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentClosing, "revision:close-pending-restart")
	closePendingOutcome, err := RecordAgentEnvironmentCloseOutcome(closePre, ports.AgentCloseReceipt{
		Subject: fixture.initial.Subject, PreviousToken: preserveResolved.Snapshot.Token,
		NextToken: closePendingToken, Preservation: preserveResolved.Snapshot.Preservation,
		IdempotencyKey: closePre.Attempt.IdempotencyKey,
	}, fixture.now.Add(9*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	closeReconciliation := mustLifecycleReconciliationOutcome(t, closePendingOutcome)
	if !reflect.DeepEqual(closeReconciliation, closeAttemptedReconciliation) ||
		closeReconciliation.ReceiptRequest.Close == nil ||
		closeReconciliation.ReceiptRequest.Quiesce != nil || closeReconciliation.ReceiptRequest.Preserve != nil ||
		closeReconciliation.ReceiptRequest.Close.ExpectedToken != closePre.Snapshot.Effect.ExpectedToken ||
		closeReconciliation.ReceiptRequest.Close.Preservation != closePre.Snapshot.Preservation {
		t.Fatalf("close reconciliation lost preservation binding: %+v", closeReconciliation)
	}
	replayedClosePending, err := ReplayAgentEnvironmentLifecycleSnapshot(
		closePre.Snapshot, fixture.request, fixture.launch, record, &preservation,
	)
	if err != nil {
		t.Fatalf("close pending restart replay failed: %v", err)
	}
	closedToken := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentClosed, "revision:close-terminal-restart")
	closeReceipt := ports.AgentCloseReceipt{
		Subject: fixture.initial.Subject, PreviousToken: closePre.Snapshot.Effect.ExpectedToken,
		NextToken: closedToken, Preservation: preserveResolved.Snapshot.Preservation,
		IdempotencyKey: closePre.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:close:after-pending", ConfirmedAt: fixture.now.Add(10 * time.Second),
	}
	closeResolved, err := RecordAgentEnvironmentCloseReconciliationOutcome(
		replayedClosePending, closePre.Claim, closePre.Attempt,
		ports.AgentEnvironmentInspectReceipt{Subject: fixture.initial.Subject, Token: closedToken},
		closeReceipt, fixture.now.Add(11*time.Second),
	)
	if err != nil || closeResolved.Attempt != closePre.Attempt ||
		closeResolved.ExpectedRevision != closePre.Snapshot.Revision ||
		closeResolved.ConsumptionReceipt.EffectReceiptRef != closeResolved.EffectReceipt.Ref {
		t.Fatalf("close pending resolution failed: resolved=%+v err=%v", closeResolved, err)
	}
	appendLifecycleTerminalFacts(&record, closeResolved)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		closeResolved.Snapshot, fixture.request, fixture.launch, record, &preservation,
	); err != nil {
		t.Fatalf("preserve/close reconciled sequence failed replay: %v", err)
	}
}
