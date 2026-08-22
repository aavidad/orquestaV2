package application

import (
	"testing"
	"time"

	"orquesta/internal/ports"
)

// A terminal token observed in a public provider response is not itself an
// accreditable receipt. Application must retain the durable attempted frontier
// for reconciliation when Quiesce or Close omits receipt identity/time.
func TestAgentEnvironmentLifecycleTerminalPublicReplyWithoutReceiptFailsClosed(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)

	quiesceClaim := lifecycleClaimFor(t, fixture.base, fixture.initial,
		ActionQuiesceAgent, 201, fixture.now)
	quiesce := mustPrepareLifecycle(t, fixture.initial, quiesceClaim, fixture.now)
	quiesced := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiesced, "revision:public-no-receipt-q")
	if outcome, err := RecordAgentEnvironmentQuiesceOutcome(quiesce, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: quiesced, IdempotencyKey: quiesce.Attempt.IdempotencyKey,
	}, fixture.now.Add(time.Second)); err == nil || outcome.Terminal != nil || outcome.Reconciliation != nil {
		t.Fatalf("terminal Quiesce without public receipt escaped fail-closed: outcome=%+v err=%v", outcome, err)
	}
	if !quiesce.Snapshot.Effect.NeedsReconciliation() {
		t.Fatal("rejected Quiesce reply destroyed unknown-applied reconciliation frontier")
	}

	preservePrepared, preserveReceipt, record := lifecyclePreparedPreserve(t, fixture)
	preservation := lifecyclePreservationFact(t, record, preserveReceipt, fixture.now.Add(6*time.Second))
	preserveOutcome, err := RecordAgentEnvironmentPreserveOutcome(
		preservePrepared, preserveReceipt, &preservation, record, fixture.now.Add(7*time.Second),
	)
	if err != nil || preserveOutcome.Terminal == nil {
		t.Fatalf("prepare Close frontier: outcome=%+v err=%v", preserveOutcome, err)
	}
	preserved := preserveOutcome.Terminal.Snapshot
	closeClaim := lifecycleClaimFor(t, fixture.base, preserved,
		ActionCloseAgentEnvironment, 202, fixture.now.Add(8*time.Second))
	closePrepared := mustPrepareLifecycle(t, preserved, closeClaim, fixture.now.Add(8*time.Second))
	closed := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentClosed, "revision:public-no-receipt-c")
	if outcome, err := RecordAgentEnvironmentCloseOutcome(closePrepared, ports.AgentCloseReceipt{
		Subject: fixture.initial.Subject, PreviousToken: preserved.Token, NextToken: closed,
		Preservation: preserved.Preservation, IdempotencyKey: closePrepared.Attempt.IdempotencyKey,
	}, fixture.now.Add(9*time.Second)); err == nil || outcome.Terminal != nil || outcome.Reconciliation != nil {
		t.Fatalf("terminal Close without public receipt escaped fail-closed: outcome=%+v err=%v", outcome, err)
	}
	if !closePrepared.Snapshot.Effect.NeedsReconciliation() {
		t.Fatal("rejected Close reply destroyed unknown-applied reconciliation frontier")
	}
}
