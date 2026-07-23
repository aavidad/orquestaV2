package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

type failFirstCommandAuditCompletion struct {
	delegate AuditPort
	failed   bool
}

func (audit *failFirstCommandAuditCompletion) Begin(
	ctx context.Context,
	record CommandAuditRecord,
) (AuditSession, error) {
	return audit.delegate.Begin(ctx, record)
}

func (audit *failFirstCommandAuditCompletion) Complete(
	ctx context.Context,
	request AuditCompletionRequest,
) (AuditCompletion, error) {
	if !audit.failed {
		audit.failed = true
		return AuditCompletion{}, errors.New("simulated.crash_after_handler_before_completion")
	}
	return audit.delegate.Complete(ctx, request)
}

func TestCommandDispatcherCrashAfterMutatingHandlerReplaysApplicationReceiptWithoutSecondEffect(t *testing.T) {
	at := time.Date(2026, 7, 23, 20, 0, 0, 0, time.UTC)
	state := &replayState{}
	orchestrator := newCrashReplayOrchestrator(t, state, at)
	durableAudit := newMemoryAudit()
	crashFrontier := &failFirstCommandAuditCompletion{delegate: durableAudit}

	beforeCrash, err := NewDispatcher(
		orchestrator, crashFrontier, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
	)
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"statement": "durable crash replay", "confirm": true}
	first := invoke(t, beforeCrash, "orquesta.goals.create", "request:crash-replay", payload, false)
	if first.Failure == nil || first.Failure.Code != CodeUnavailable || state.creates != 1 || state.createCalls != 1 {
		t.Fatalf("first=%+v create_calls=%d effects=%d", first, state.createCalls, state.creates)
	}

	// A fresh dispatcher represents process recovery: durable admission and
	// application state survive, while the terminal audit fact was not written.
	afterCrash, err := NewDispatcher(
		orchestrator, durableAudit, APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100},
	)
	if err != nil {
		t.Fatal(err)
	}
	second := invoke(t, afterCrash, "orquesta.goals.create", "request:crash-replay", payload, false)
	third := invoke(t, afterCrash, "orquesta.goals.create", "request:crash-replay", payload, false)
	if second.Failure != nil || third.Failure != nil || second.AuditRef == "" ||
		second.AuditRef != third.AuditRef || !bytes.Equal(second.Data, third.Data) ||
		state.creates != 1 || state.createCalls != 3 {
		t.Fatalf("second=%+v third=%+v create_calls=%d effects=%d", second, third, state.createCalls, state.creates)
	}
}

func newCrashReplayOrchestrator(t *testing.T, state *replayState, at time.Time) *application.Orchestrator {
	t.Helper()
	policy := replayBudgetPolicy(at)
	orchestrator, err := application.New(application.Dependencies{
		State: state, Access: replayAccess{}, Launcher: replayAgent{}, Observer: replayObserver{},
		Artifacts: replayArtifacts{}, Clock: replayClock{at}, IDs: &replayIDs{},
		MaxOutputBytes: 1 << 20, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute, EffectApprovalTTL: policy.EffectApprovalTTL,
		BudgetPolicy: policy, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: ports.AgentCapabilities{
			ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test", Unrestricted: true,
		},
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new crash replay orchestrator: %w", err))
	}
	return orchestrator
}
