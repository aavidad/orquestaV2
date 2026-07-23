package commands

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type replayState struct {
	application.StateRepository
	mu          sync.Mutex
	record      application.GoalRecord
	createCalls int
	creates     int
}

func (state *replayState) CreateGoal(_ context.Context, input application.CreateGoalState) (application.GoalRecord, bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.createCalls++
	if state.record.RequestRef != "" {
		if state.record.RequestRef != input.RequestRef || state.record.RequestFingerprint != input.RequestFingerprint {
			return application.GoalRecord{}, false, &application.StateError{Code: application.StateConflict}
		}
		return state.record, false, nil
	}
	state.creates++
	state.record = application.GoalRecord{
		RequestRef: input.RequestRef, RequestFingerprint: input.RequestFingerprint, RequestedBy: input.RequestedBy,
		Goal: input.Goal, Executions: append([]application.ExecutionRecord(nil), input.Executions...),
		BudgetEnvelopes:     append([]governance.BudgetEnvelope(nil), input.BudgetEnvelopes...),
		WorkItemAuthorities: append([]application.WorkItemAuthority(nil), input.WorkItemAuthorities...),
	}
	for _, action := range input.Actions {
		if action.EffectIntent.Ref != "" {
			state.record.EffectIntents = append(state.record.EffectIntents, action.EffectIntent)
		}
		if action.EffectApproval != nil {
			state.record.EffectApprovals = append(state.record.EffectApprovals, *action.EffectApproval)
		}
	}
	return state.record, true, nil
}

func (state *replayState) progress(at time.Time) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	updated, err := state.record.Goal.SetPaused(state.record.Goal.Revision(), true, at)
	if err != nil {
		return err
	}
	state.record.Goal = updated
	return nil
}

type replayAccess struct{ application.AccessRepository }

func (replayAccess) Authorize(_ context.Context, request identity.AuthorizationRequest) (identity.AuthorizationReceipt, error) {
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationAllowed, Role: identity.RoleProjectOwner,
		MembershipRevision: 1, ReasonCode: "policy.allowed", DecidedAt: request.RequestedAt(),
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	return identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: "authorization-receipt:" + request.RequestRef(), Decision: decision, RecordedAt: request.RequestedAt(),
	})
}

type replayIDs struct {
	mu   sync.Mutex
	next int
}

func (ids *replayIDs) NewID(_ context.Context, kind string) (string, error) {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:real-%d", kind, ids.next), nil
}

type replayClock struct{ at time.Time }

func (clock replayClock) Now() time.Time { return clock.at }

type replayAgent struct{ application.AgentLauncher }

func (replayAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test", Unrestricted: true}, nil
}
func (replayAgent) Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	return ports.AgentLaunchReceipt{}, fmt.Errorf("unexpected launch")
}

type replayObserver struct{ application.AgentObserver }
type replayArtifacts struct{ application.ArtifactStore }

func TestRealOrchestratorCommandReplaySurvivesAggregateProgressWithoutSecondEffect(t *testing.T) {
	at := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
	state := &replayState{}
	policy := replayBudgetPolicy(at)
	orchestrator, err := application.New(application.Dependencies{
		State: state, Access: replayAccess{}, Launcher: replayAgent{}, Observer: replayObserver{}, Artifacts: replayArtifacts{},
		Clock: replayClock{at}, IDs: &replayIDs{}, MaxOutputBytes: 1 << 20, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute, EffectApprovalTTL: policy.EffectApprovalTTL,
		BudgetPolicy: policy, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: ports.AgentCapabilities{ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test", Unrestricted: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewDispatcher(orchestrator, newMemoryAudit(), APILimits{MaxRequestBytes: 1 << 20, MaxListLimit: 100})
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"statement": "build a durable service", "confirm": true}
	first := invoke(t, dispatcher, "orquesta.goals.create", "request:real-replay", payload, false)
	if first.Failure != nil {
		t.Fatalf("first=%+v", first)
	}
	before := state.record.Goal.Revision()
	if err := state.progress(at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if state.record.Goal.Revision() == before {
		t.Fatal("aggregate did not progress")
	}
	second := invoke(t, dispatcher, "orquesta.goals.create", "request:real-replay", payload, false)
	if second.Failure != nil || first.AuditRef != second.AuditRef || string(first.Data) != string(second.Data) {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if state.createCalls != 2 || state.creates != 1 {
		t.Fatalf("create calls=%d effects=%d", state.createCalls, state.creates)
	}
}

func replayBudgetPolicy(at time.Time) application.BudgetPolicy {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	limit := governance.ResourceVector{Tokens: 1_000_000, MoneyMicros: 1_000_000, Currency: "EUR", ActiveTimeNS: int64(time.Hour), ProcessSlots: 70, DiskBytes: 1 << 30}
	envelope := func(ref, subject string, scope governance.BudgetScope) governance.BudgetEnvelope {
		return governance.BudgetEnvelope{Ref: ref, SubjectRef: subject, Scope: scope, Limit: limit, Revision: 1, PolicyHash: digest, CreatedAt: at}
	}
	return application.BudgetPolicy{
		DeploymentEnvelope:      envelope("budget-envelope:deployment:test", "deployment:test", governance.BudgetScopeDeployment),
		ProjectEnvelopeTemplate: envelope("budget-envelope:project:test", "project:test", governance.BudgetScopeProject),
		GoalEnvelopeTemplate:    envelope("budget-envelope:goal:test", "goal:test", governance.BudgetScopeGoal),
		DefaultWorkItemDemand:   governance.ResourceVector{Tokens: 100, MoneyMicros: 100, Currency: "EUR", ActiveTimeNS: int64(time.Minute), ProcessSlots: 1, DiskBytes: 1 << 10},
		QuotaRetryDelay:         time.Second, EffectApprovalTTL: time.Hour, PolicyHash: digest,
	}
}
