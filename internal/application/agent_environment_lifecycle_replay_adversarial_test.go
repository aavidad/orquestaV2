package application

import (
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestAgentEnvironmentLifecycleInitialReplayRequiresExactLaunchHistory(t *testing.T) {
	for _, test := range []struct {
		name              string
		equalConfirmation bool
	}{
		{name: "normal confirmation after physical acceptance"},
		{name: "recovery confirmation at physical acceptance", equalConfirmation: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentEnvironmentLifecycleFixture(t)
			if test.equalConfirmation {
				execution := lifecycleFixtureExecution(t, &fixture)
				for index := range fixture.record.EffectReceipts {
					if fixture.record.EffectReceipts[index].Ref == execution.LaunchReceiptRef {
						fixture.record.EffectReceipts[index].ConfirmedAt = fixture.launch.AcceptedAt
					}
				}
			}
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				fixture.initial, fixture.request, fixture.launch, fixture.record, nil,
			); err != nil {
				t.Fatalf("exact initial replay failed: %v", err)
			}
		})
	}
}

func TestAgentEnvironmentLifecycleInitialReplayRejectsCrossedExecutionRefsAndLaunchTimes(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*agentEnvironmentLifecycleFixture)
	}{
		{name: "execution intent ref", mutate: func(fixture *agentEnvironmentLifecycleFixture) {
			lifecycleFixtureExecution(t, fixture).EffectIntentRef = "effect-intent:launch:crossed"
		}},
		{name: "execution receipt ref", mutate: func(fixture *agentEnvironmentLifecycleFixture) {
			lifecycleFixtureExecution(t, fixture).LaunchReceiptRef = "effect-receipt:launch:crossed"
		}},
		{name: "execution accepted time", mutate: func(fixture *agentEnvironmentLifecycleFixture) {
			lifecycleFixtureExecution(t, fixture).ProviderAcceptedAt = fixture.launch.AcceptedAt.Add(time.Nanosecond)
		}},
		{name: "effect confirmed before physical acceptance", mutate: func(fixture *agentEnvironmentLifecycleFixture) {
			execution := lifecycleFixtureExecution(t, fixture)
			for index := range fixture.record.EffectReceipts {
				if fixture.record.EffectReceipts[index].Ref == execution.LaunchReceiptRef {
					fixture.record.EffectReceipts[index].ConfirmedAt = fixture.launch.AcceptedAt.Add(-time.Nanosecond)
				}
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentEnvironmentLifecycleFixture(t)
			test.mutate(&fixture)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				fixture.initial, fixture.request, fixture.launch, fixture.record, nil,
			); err == nil {
				t.Fatal("crossed launch authority replayed")
			}
		})
	}
}

func TestAgentEnvironmentLifecycleInitialReplayRejectsLifecycleAttempts(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 71, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)

	t.Run("complete posterior lifecycle attempt", func(t *testing.T) {
		record := fixture.record
		appendLifecyclePreparedFacts(&record, prepared)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			fixture.initial, fixture.request, fixture.launch, record, nil,
		); err == nil {
			t.Fatal("active snapshot ignored a posterior lifecycle attempt")
		}
	})
	t.Run("orphan lifecycle attempt", func(t *testing.T) {
		record := fixture.record
		record.EffectAttempts = append(record.EffectAttempts, prepared.Attempt)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			fixture.initial, fixture.request, fixture.launch, record, nil,
		); err == nil {
			t.Fatal("active snapshot ignored an orphan lifecycle attempt")
		}
	})
	t.Run("complete posterior lifecycle receipt", func(t *testing.T) {
		outcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
			Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
			NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
				ports.AgentEnvironmentQuiesced, "revision:posterior-receipt"),
			IdempotencyKey: prepared.Attempt.IdempotencyKey,
			ReceiptRef:     "receipt:quiesce:posterior", ConfirmedAt: fixture.now.Add(time.Second),
		}, fixture.now.Add(2*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		post := mustLifecycleTerminalOutcome(t, outcome)
		record := fixture.record
		appendLifecyclePreparedFacts(&record, prepared)
		appendLifecycleTerminalFacts(&record, post)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			fixture.initial, fixture.request, fixture.launch, record, nil,
		); err == nil {
			t.Fatal("active snapshot ignored a posterior lifecycle receipt")
		}
	})
}

func TestAgentEnvironmentLifecycleReplayAcceptsExactPlannedNextEffect(t *testing.T) {
	t.Run("active with planned quiesce", func(t *testing.T) {
		fixture := newAgentEnvironmentLifecycleFixture(t)
		claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 72, fixture.now)
		record := fixture.record
		record.EffectIntents = append(record.EffectIntents, claim.Action.EffectIntent)
		record.EffectApprovals = append(record.EffectApprovals, claim.EffectApproval)

		replayed, err := ReplayAgentEnvironmentLifecycleSnapshot(
			fixture.initial, fixture.request, fixture.launch, record, nil,
		)
		if err != nil {
			t.Fatalf("planned quiesce advanced physical frontier: %v", err)
		}
		prepared, err := PrepareAgentEnvironmentLifecycleEffect(replayed, claim, fixture.now)
		if err != nil || prepared.Attempt.IntentRef != claim.Action.EffectIntent.Ref ||
			prepared.Attempt.ApprovalRef != claim.EffectApproval.Ref {
			t.Fatalf("exact planned claim not consumed: prepared=%+v err=%v", prepared, err)
		}
	})

	t.Run("quiesced with planned preserve", func(t *testing.T) {
		fixture := newAgentEnvironmentLifecycleFixture(t)
		quiesceClaim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 73, fixture.now)
		quiescePre := mustPrepareLifecycle(t, fixture.initial, quiesceClaim, fixture.now)
		quiesceOutcome, err := RecordAgentEnvironmentQuiesceOutcome(quiescePre, ports.AgentQuiesceReceipt{
			Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
			NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
				ports.AgentEnvironmentQuiesced, "revision:planned-preserve"),
			IdempotencyKey: quiescePre.Attempt.IdempotencyKey,
			ReceiptRef:     "receipt:quiesce:planned-preserve", ConfirmedAt: fixture.now.Add(time.Second),
		}, fixture.now.Add(2*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		quiescePost := mustLifecycleTerminalOutcome(t, quiesceOutcome)
		record := fixture.record
		appendLifecyclePreparedFacts(&record, quiescePre)
		appendLifecycleTerminalFacts(&record, quiescePost)
		preserveClaim := lifecycleClaimFor(t, fixture.base, quiescePost.Snapshot,
			ActionPreserveAgentEnvironment, 74, fixture.now.Add(3*time.Second))
		record.EffectIntents = append(record.EffectIntents, preserveClaim.Action.EffectIntent)
		record.EffectApprovals = append(record.EffectApprovals, preserveClaim.EffectApproval)

		replayed, err := ReplayAgentEnvironmentLifecycleSnapshot(
			quiescePost.Snapshot, fixture.request, fixture.launch, record, nil,
		)
		if err != nil {
			t.Fatalf("planned preserve advanced physical frontier: %v", err)
		}
		prepared, err := PrepareAgentEnvironmentLifecycleEffect(
			replayed, preserveClaim, fixture.now.Add(3*time.Second),
		)
		if err != nil || prepared.Attempt.IntentRef != preserveClaim.Action.EffectIntent.Ref ||
			prepared.Attempt.ApprovalRef != preserveClaim.EffectApproval.Ref {
			t.Fatalf("exact planned preserve claim not consumed: prepared=%+v err=%v", prepared, err)
		}
	})
}

func TestAgentEnvironmentLifecycleReplayRejectsIncompleteOrCrossedPlannedEffect(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GoalRecord, ActionClaim)
	}{
		{name: "missing approval", mutate: func(record *GoalRecord, claim ActionClaim) {
			record.EffectIntents = append(record.EffectIntents, claim.Action.EffectIntent)
		}},
		{name: "duplicate approval", mutate: func(record *GoalRecord, claim ActionClaim) {
			record.EffectIntents = append(record.EffectIntents, claim.Action.EffectIntent)
			record.EffectApprovals = append(record.EffectApprovals, claim.EffectApproval, claim.EffectApproval)
		}},
		{name: "crossed approval authority", mutate: func(record *GoalRecord, claim ActionClaim) {
			record.EffectIntents = append(record.EffectIntents, claim.Action.EffectIntent)
			approval := claim.EffectApproval
			approval.IntentDigest = claim.EffectApproval.IntentDigest + ":crossed"
			record.EffectApprovals = append(record.EffectApprovals, approval)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentEnvironmentLifecycleFixture(t)
			claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 75, fixture.now)
			record := fixture.record
			test.mutate(&record, claim)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				fixture.initial, fixture.request, fixture.launch, record, nil,
			); err == nil {
				t.Fatal("invalid planned effect replayed")
			}
		})
	}
}

func TestAgentEnvironmentLifecycleReplayRejectsPartialOrDuplicateLaunchChain(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GoalRecord, ExecutionRecord)
	}{
		{name: "missing intent", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			record.EffectIntents = removeLifecycleIntent(record.EffectIntents, execution.EffectIntentRef)
		}},
		{name: "missing approval", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			intent, _ := exactAgentEnvironmentIntentByRef(record.EffectIntents, execution.EffectIntentRef)
			for _, attempt := range record.EffectAttempts {
				if attempt.IntentRef == intent.Ref {
					record.EffectApprovals = removeLifecycleApproval(record.EffectApprovals, attempt.ApprovalRef)
					return
				}
			}
		}},
		{name: "missing attempt", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			intent, _ := exactAgentEnvironmentIntentByRef(record.EffectIntents, execution.EffectIntentRef)
			record.EffectAttempts = removeLifecycleAttemptForIntent(record.EffectAttempts, intent.Ref)
		}},
		{name: "missing receipt", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			record.EffectReceipts = removeLifecycleReceipt(record.EffectReceipts, execution.LaunchReceiptRef)
		}},
		{name: "duplicate intent", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			intent, _ := exactAgentEnvironmentIntentByRef(record.EffectIntents, execution.EffectIntentRef)
			record.EffectIntents = append(record.EffectIntents, intent)
		}},
		{name: "duplicate approval", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			intent, _ := exactAgentEnvironmentIntentByRef(record.EffectIntents, execution.EffectIntentRef)
			for _, approval := range record.EffectApprovals {
				if approval.IntentRef == intent.Ref {
					record.EffectApprovals = append(record.EffectApprovals, approval)
					return
				}
			}
		}},
		{name: "duplicate attempt", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			for _, attempt := range record.EffectAttempts {
				if attempt.IntentRef == execution.EffectIntentRef {
					record.EffectAttempts = append(record.EffectAttempts, attempt)
					return
				}
			}
		}},
		{name: "duplicate receipt", mutate: func(record *GoalRecord, execution ExecutionRecord) {
			for _, receipt := range record.EffectReceipts {
				if receipt.Ref == execution.LaunchReceiptRef {
					record.EffectReceipts = append(record.EffectReceipts, receipt)
					return
				}
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentEnvironmentLifecycleFixture(t)
			execution, err := exactAgentEnvironmentExecution(fixture.record, fixture.initial)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&fixture.record, execution)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				fixture.initial, fixture.request, fixture.launch, fixture.record, nil,
			); err == nil {
				t.Fatal("partial or duplicate launch history replayed")
			}
		})
	}
}

func TestAgentEnvironmentLifecycleReplayRejectsStaleRollbackPeersAndPosteriorFacts(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 81, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	outcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentQuiesced, "revision:adversarial"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
		ReceiptRef:     "receipt:quiesce:adversarial", ConfirmedAt: fixture.now.Add(time.Second),
	}, fixture.now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)
	appendLifecycleTerminalFacts(&record, post)

	t.Run("stale snapshot revision", func(t *testing.T) {
		stale := post.Snapshot
		stale.Revision--
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			stale, fixture.request, fixture.launch, record, nil,
		); err == nil {
			t.Fatal("stale lifecycle snapshot replayed")
		}
	})
	t.Run("rollback missing terminal receipt", func(t *testing.T) {
		rollback := record
		rollback.EffectReceipts = removeLifecycleReceipt(rollback.EffectReceipts, post.EffectReceipt.Ref)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, rollback, nil,
		); err == nil {
			t.Fatal("terminal snapshot replayed from rolled-back ledger")
		}
	})
	t.Run("rollback missing terminal consumption", func(t *testing.T) {
		rollback := record
		rollback.ConsumptionReceipts = removeLifecycleConsumption(
			rollback.ConsumptionReceipts, post.ConsumptionReceipt.ActionRef,
		)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, rollback, nil,
		); err == nil {
			t.Fatal("terminal snapshot replayed without causal consumption")
		}
	})
	t.Run("duplicate terminal consumption", func(t *testing.T) {
		duplicate := record
		duplicate.ConsumptionReceipts = append(duplicate.ConsumptionReceipts, post.ConsumptionReceipt)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, duplicate, nil,
		); err == nil {
			t.Fatal("terminal snapshot replayed duplicate consumption")
		}
	})
	t.Run("crossed terminal consumption", func(t *testing.T) {
		crossed := record
		crossed.ConsumptionReceipts = append([]ActionConsumptionReceipt(nil), record.ConsumptionReceipts...)
		for index := range crossed.ConsumptionReceipts {
			if crossed.ConsumptionReceipts[index].ActionRef == post.ConsumptionReceipt.ActionRef {
				crossed.ConsumptionReceipts[index].EffectReceiptRef = "effect-receipt:crossed"
			}
		}
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, crossed, nil,
		); err == nil {
			t.Fatal("terminal snapshot replayed crossed consumption")
		}
	})
	crossedChange, err := ports.NewChangeSetRef("change:crossed")
	if err != nil {
		t.Fatal(err)
	}
	crossedMailbox, err := NewMailboxMessageRef("mailbox-message:crossed")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []struct {
		name   string
		mutate func(*ActionConsumptionReceipt)
	}{
		{name: "work item generation", mutate: func(receipt *ActionConsumptionReceipt) {
			receipt.WorkItemGeneration++
		}},
		{name: "change ref", mutate: func(receipt *ActionConsumptionReceipt) {
			receipt.ChangeRef = crossedChange
		}},
		{name: "mailbox ref", mutate: func(receipt *ActionConsumptionReceipt) {
			receipt.MailboxMessageRef = crossedMailbox
		}},
		{name: "delivery attempt", mutate: func(receipt *ActionConsumptionReceipt) {
			receipt.DeliveryAttempt = 0
		}},
		{name: "before effect confirmation", mutate: func(receipt *ActionConsumptionReceipt) {
			receipt.ConsumedAt = post.EffectReceipt.ConfirmedAt.Add(-time.Nanosecond)
		}},
	} {
		t.Run("tampered consumption "+mutation.name, func(t *testing.T) {
			crossed := record
			crossed.ConsumptionReceipts = append([]ActionConsumptionReceipt(nil), record.ConsumptionReceipts...)
			for index := range crossed.ConsumptionReceipts {
				if crossed.ConsumptionReceipts[index].ActionRef == post.ConsumptionReceipt.ActionRef {
					mutation.mutate(&crossed.ConsumptionReceipts[index])
				}
			}
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				post.Snapshot, fixture.request, fixture.launch, crossed, nil,
			); err == nil {
				t.Fatal("terminal snapshot replayed tampered lifecycle consumption")
			}
		})
	}
	t.Run("snapshot time differs from consumption", func(t *testing.T) {
		crossed := post.Snapshot
		crossed.RecordedAt = crossed.RecordedAt.Add(time.Nanosecond)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			crossed, fixture.request, fixture.launch, record, nil,
		); err == nil {
			t.Fatal("terminal snapshot replayed outside consumption CAS time")
		}
	})
	for _, mutation := range []struct {
		name   string
		mutate func(*AgentEnvironmentLifecycleEffectSnapshot)
	}{
		{name: "delivery attempt", mutate: func(effect *AgentEnvironmentLifecycleEffectSnapshot) {
			effect.DeliveryAttempt++
		}},
		{name: "work item generation", mutate: func(effect *AgentEnvironmentLifecycleEffectSnapshot) {
			effect.WorkItemGeneration++
		}},
		{name: "claim token", mutate: func(effect *AgentEnvironmentLifecycleEffectSnapshot) {
			effect.ClaimToken = "claim:crossed"
		}},
		{name: "worker", mutate: func(effect *AgentEnvironmentLifecycleEffectSnapshot) {
			effect.WorkerRef = "worker:crossed"
		}},
	} {
		t.Run("snapshot crossed "+mutation.name, func(t *testing.T) {
			crossed := post.Snapshot
			mutation.mutate(&crossed.Effect)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				crossed, fixture.request, fixture.launch, record, nil,
			); err == nil {
				t.Fatal("terminal snapshot crossed lifecycle claim binding")
			}
		})
	}
	t.Run("next attempt before prior consumption", func(t *testing.T) {
		forgedPrior := post.Snapshot
		forgedPrior.RecordedAt = post.EffectReceipt.ConfirmedAt
		preserveAt := post.EffectReceipt.ConfirmedAt.Add(time.Nanosecond)
		preserveClaim := lifecycleClaimFor(t, fixture.base, forgedPrior,
			ActionPreserveAgentEnvironment, 83, preserveAt)
		preserve := mustPrepareLifecycle(t, forgedPrior, preserveClaim, preserveAt)
		candidate := record
		appendLifecyclePreparedFacts(&candidate, preserve)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			preserve.Snapshot, fixture.request, fixture.launch, candidate, nil,
		); err == nil {
			t.Fatal("next attempt replayed between physical confirmation and prior CAS consumption")
		}
	})
	t.Run("peer attempt", func(t *testing.T) {
		peerRecord := record
		peer := prepared.Attempt
		peer.Ref += ":peer"
		peer.WorkerRef = "worker:lifecycle:peer"
		peerRecord.EffectAttempts = append(peerRecord.EffectAttempts, peer)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, peerRecord, nil,
		); err == nil {
			t.Fatal("peer lifecycle attempt replayed")
		}
	})
	t.Run("peer receipt", func(t *testing.T) {
		peerRecord := record
		peer := *post.EffectReceipt
		peer.Ref += ":peer"
		peerRecord.EffectReceipts = append(peerRecord.EffectReceipts, peer)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, peerRecord, nil,
		); err == nil {
			t.Fatal("peer lifecycle receipt replayed")
		}
	})
	t.Run("posterior lifecycle stage", func(t *testing.T) {
		posterior := record
		preserveClaim := lifecycleClaimFor(t, fixture.base, post.Snapshot,
			ActionPreserveAgentEnvironment, 82, fixture.now.Add(3*time.Second))
		preserve := mustPrepareLifecycle(t, post.Snapshot, preserveClaim, fixture.now.Add(3*time.Second))
		appendLifecyclePreparedFacts(&posterior, preserve)
		if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
			post.Snapshot, fixture.request, fixture.launch, posterior, nil,
		); err == nil {
			t.Fatal("snapshot ignored posterior lifecycle stage")
		}
	})
}

func TestAgentEnvironmentLifecyclePendingReplayRejectsRelatedReceipt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 91, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	pendingOutcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken: lifecycleToken(t, fixture.initial.Subject.ExternalRef,
			ports.AgentEnvironmentQuiescing, "revision:pending-adversarial"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}, fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	mustLifecycleReconciliationOutcome(t, pendingOutcome)
	receipt, err := effectReceipt(prepared.Claim, prepared.Attempt,
		"receipt:quiesce:unexpected-terminal", EffectStatusQuiesced, unknownUsage(), fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)
	record.EffectReceipts = append(record.EffectReceipts, receipt)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		prepared.Snapshot, fixture.request, fixture.launch, record, nil,
	); err == nil {
		t.Fatal("attempted lifecycle snapshot ignored terminal receipt without consumption")
	}
}

func removeLifecycleIntent(records []EffectIntent, ref string) []EffectIntent {
	result := make([]EffectIntent, 0, len(records))
	for _, record := range records {
		if record.Ref != ref {
			result = append(result, record)
		}
	}
	return result
}

func removeLifecycleApproval(records []EffectApproval, ref string) []EffectApproval {
	result := make([]EffectApproval, 0, len(records))
	for _, record := range records {
		if record.Ref != ref {
			result = append(result, record)
		}
	}
	return result
}

func removeLifecycleAttemptForIntent(records []EffectAttempt, intentRef string) []EffectAttempt {
	result := make([]EffectAttempt, 0, len(records))
	for _, record := range records {
		if record.IntentRef != intentRef {
			result = append(result, record)
		}
	}
	return result
}

func removeLifecycleReceipt(records []EffectReceipt, ref string) []EffectReceipt {
	result := make([]EffectReceipt, 0, len(records))
	for _, record := range records {
		if record.Ref != ref {
			result = append(result, record)
		}
	}
	return result
}

func removeLifecycleConsumption(
	records []ActionConsumptionReceipt,
	actionRef string,
) []ActionConsumptionReceipt {
	result := make([]ActionConsumptionReceipt, 0, len(records))
	for _, record := range records {
		if record.ActionRef != actionRef {
			result = append(result, record)
		}
	}
	return result
}
