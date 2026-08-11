package application

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestAgentEnvironmentLifecycleReplayRejectsCrossedLaunchTokenFenceAndAttempt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 31, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)

	for name, mutate := range map[string]func(*AgentEnvironmentLifecycleSnapshot, *ports.AgentLaunchReceipt){
		"launch receipt": func(value *AgentEnvironmentLifecycleSnapshot, _ *ports.AgentLaunchReceipt) {
			value.LaunchReceiptRef = "receipt:launch:crossed"
		},
		"token": func(value *AgentEnvironmentLifecycleSnapshot, _ *ports.AgentLaunchReceipt) {
			value.Token = lifecycleToken(t, value.Subject.ExternalRef, ports.AgentEnvironmentActive, "revision:crossed")
		},
		"fence": func(value *AgentEnvironmentLifecycleSnapshot, _ *ports.AgentLaunchReceipt) {
			value.Effect.ActionFence++
		},
		"attempt": func(value *AgentEnvironmentLifecycleSnapshot, _ *ports.AgentLaunchReceipt) {
			value.Effect.AttemptRef = "effect-attempt:crossed"
		},
		"claim token": func(value *AgentEnvironmentLifecycleSnapshot, _ *ports.AgentLaunchReceipt) {
			value.Effect.ClaimToken = "claim:crossed"
		},
		"worker": func(value *AgentEnvironmentLifecycleSnapshot, _ *ports.AgentLaunchReceipt) {
			value.Effect.WorkerRef = "worker:crossed"
		},
		"physical launch": func(_ *AgentEnvironmentLifecycleSnapshot, launch *ports.AgentLaunchReceipt) {
			launch.ExternalRef = "external:crossed"
		},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot, launch := prepared.Snapshot, fixture.launch
			mutate(&snapshot, &launch)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				snapshot, fixture.request, launch, record, nil,
			); err == nil {
				t.Fatal("crossed lifecycle snapshot replayed")
			}
		})
	}
}

func TestAgentEnvironmentLifecyclePreserveRejectsInventedOrCrossedApplicationFact(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	prepared, receipt, record := lifecyclePreparedPreserve(t, fixture)

	if _, err := RecordAgentEnvironmentPreserveOutcome(
		prepared, receipt, nil, record, fixture.now.Add(7*time.Second),
	); err == nil {
		t.Fatal("terminal preserve accepted without application fact")
	}
	invented := ComprobantePreservacionEntornoAgente{Ref: "environment-receipt:invented"}
	if _, err := RecordAgentEnvironmentPreserveOutcome(
		prepared, receipt, &invented, record, fixture.now.Add(7*time.Second),
	); err == nil {
		t.Fatal("terminal preserve accepted invented strings")
	}
	real := lifecyclePreservationFact(t, record, receipt, fixture.now.Add(6*time.Second))
	for name, mutate := range map[string]func(*ComprobantePreservacionEntornoAgente){
		"goal": func(value *ComprobantePreservacionEntornoAgente) {
			value.ObjetivoRef, _ = goal.NewGoalRef("goal:crossed")
		},
		"item": func(value *ComprobantePreservacionEntornoAgente) {
			value.ItemRef, _ = goal.NewWorkItemRef("work-item:crossed")
		},
		"execution": func(value *ComprobantePreservacionEntornoAgente) {
			value.EjecucionRef, _ = goal.NewExecutionRef("execution:crossed")
			value.Resultado.EjecucionRef = value.EjecucionRef
		},
		"attempt": func(value *ComprobantePreservacionEntornoAgente) {
			value.Resultado.IntentoEjecucion++
		},
		"external": func(value *ComprobantePreservacionEntornoAgente) {
			value.Resultado.IdentidadExterna = "external:crossed"
		},
		"manifest ref": func(value *ComprobantePreservacionEntornoAgente) {
			value.ManifiestoFisicoRef = "physical-manifest:crossed"
		},
		"manifest digest": func(value *ComprobantePreservacionEntornoAgente) {
			value.ManifiestoFisicoDigest = strings.Repeat("f", 64)
		},
	} {
		t.Run("crossed "+name, func(t *testing.T) {
			crossed := real
			mutate(&crossed)
			if _, err := RecordAgentEnvironmentPreserveOutcome(
				prepared, receipt, &crossed, record, fixture.now.Add(7*time.Second),
			); err == nil {
				t.Fatal("terminal preserve accepted crossed application subject")
			}
		})
	}
	crossedRecord := record
	crossedRecord.ConsumptionReceipts = nil
	if _, err := RecordAgentEnvironmentPreserveOutcome(
		prepared, receipt, &real, crossedRecord, fixture.now.Add(7*time.Second),
	); err == nil {
		t.Fatal("terminal preserve accepted crossed application causality")
	}
	outcome, err := RecordAgentEnvironmentPreserveOutcome(
		prepared, receipt, &real, record, fixture.now.Add(7*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	forged := post.Snapshot
	forged.Preservation.ApplicationReceiptRef = "environment-receipt:forged"
	appendLifecyclePreparedFacts(&record, prepared)
	appendLifecycleTerminalFacts(&record, post)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		forged, fixture.request, fixture.launch, record, &real,
	); err == nil {
		t.Fatal("forged application receipt ref replayed")
	}
	for name, mutate := range map[string]func(*ComprobantePreservacionEntornoAgente){
		"manifest ref": func(value *ComprobantePreservacionEntornoAgente) {
			value.ManifiestoFisicoRef = "physical-manifest:replay-crossed"
		},
		"manifest digest": func(value *ComprobantePreservacionEntornoAgente) {
			value.ManifiestoFisicoDigest = strings.Repeat("e", 64)
		},
	} {
		t.Run("replay crossed "+name, func(t *testing.T) {
			crossed := real
			mutate(&crossed)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				post.Snapshot, fixture.request, fixture.launch, record, &crossed,
			); err == nil {
				t.Fatal("crossed physical manifest fact replayed")
			}
		})
	}
}

func TestAgentEnvironmentLifecycleReplayRejectsCrossedTerminalEffectReceipt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 41, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	quiesced := lifecycleToken(t, fixture.initial.Subject.ExternalRef,
		ports.AgentEnvironmentQuiesced, "revision:terminal-replay")
	outcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token, NextToken: quiesced,
		IdempotencyKey: prepared.Attempt.IdempotencyKey, ReceiptRef: "receipt:quiesce:terminal-replay",
		ConfirmedAt: fixture.now.Add(time.Second),
	}, fixture.now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	record := fixture.record
	appendLifecyclePreparedFacts(&record, prepared)

	for name, mutate := range map[string]func(*EffectReceipt){
		"intent ref":    func(value *EffectReceipt) { value.IntentRef = "effect-intent:crossed" },
		"intent digest": func(value *EffectReceipt) { value.IntentDigest = strings.Repeat("f", 64) },
		"approval ref":  func(value *EffectReceipt) { value.ApprovalRef = "effect-approval:crossed" },
		"idempotency":   func(value *EffectReceipt) { value.IdempotencyKey = "idempotency:crossed" },
		"subject":       func(value *EffectReceipt) { value.Subject.PlanGeneration++ },
	} {
		t.Run(name, func(t *testing.T) {
			crossed := *post.EffectReceipt
			mutate(&crossed)
			candidate := record
			candidate.EffectReceipts = append(candidate.EffectReceipts, crossed)
			candidate.ConsumptionReceipts = append(candidate.ConsumptionReceipts, post.ConsumptionReceipt)
			if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
				post.Snapshot, fixture.request, fixture.launch, candidate, nil,
			); err == nil {
				t.Fatal("crossed terminal effect receipt replayed")
			}
		})
	}

	duplicate := record
	duplicate.EffectAttempts = append(duplicate.EffectAttempts, prepared.Attempt)
	duplicate.EffectReceipts = append(duplicate.EffectReceipts, *post.EffectReceipt)
	duplicate.ConsumptionReceipts = append(duplicate.ConsumptionReceipts, post.ConsumptionReceipt)
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(
		post.Snapshot, fixture.request, fixture.launch, duplicate, nil,
	); err == nil {
		t.Fatal("duplicate terminal effect identity replayed")
	}
}

func TestAgentEnvironmentLifecycleCloseRequiresDerivedApplicationReceipt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	prepared, receipt, record := lifecyclePreparedPreserve(t, fixture)
	real := lifecyclePreservationFact(t, record, receipt, fixture.now.Add(6*time.Second))
	outcome, err := RecordAgentEnvironmentPreserveOutcome(
		prepared, receipt, &real, record, fixture.now.Add(7*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	invalid := post.Snapshot
	invalid.Preservation.ApplicationReceiptRef = ""
	claim := lifecycleClaimFor(t, fixture.base, invalid,
		ActionCloseAgentEnvironment, 51, fixture.now.Add(8*time.Second))
	if _, err := PrepareAgentEnvironmentLifecycleEffect(invalid, claim, fixture.now.Add(8*time.Second)); err == nil {
		t.Fatal("close prepared without derived application receipt")
	}
}
