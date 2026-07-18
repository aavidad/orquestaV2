package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

// Existing application tests use memoryRepository directly. These no-op
// governance methods keep that shared fake structurally current without
// adding V15 state to the older helper.
func (*memoryRepository) EffectReplay(context.Context, EffectReplayRequest) (EffectApproval, bool, error) {
	return EffectApproval{}, false, nil
}

func (*memoryRepository) DecideEffect(context.Context, DecideEffectState) (EffectApproval, bool, error) {
	return EffectApproval{}, false, errors.New("test.effect_governance_not_configured")
}

func (*memoryRepository) RecordEffectAttempt(context.Context, RecordEffectAttemptState) (EffectAttempt, bool, error) {
	return EffectAttempt{}, false, errors.New("test.effect_governance_not_configured")
}

type effectDecisionMutation struct {
	request  EffectReplayRequest
	approval EffectApproval
}

type effectDecisionRepository struct {
	*memoryRepository
	governanceMu sync.Mutex
	decisions    map[string]effectDecisionMutation
	decideCalls  int
}

func newEffectDecisionRepository() *effectDecisionRepository {
	return &effectDecisionRepository{
		memoryRepository: newMemoryRepository(),
		decisions:        make(map[string]effectDecisionMutation),
	}
}

func (repository *effectDecisionRepository) EffectReplay(
	_ context.Context,
	request EffectReplayRequest,
) (EffectApproval, bool, error) {
	repository.governanceMu.Lock()
	defer repository.governanceMu.Unlock()
	mutation, found := repository.decisions[effectDecisionKey(request.PrincipalRef, request.ProjectRef, request.RequestRef)]
	if !found {
		return EffectApproval{}, false, nil
	}
	if mutation.request != request {
		return EffectApproval{}, false, &StateError{Code: StateConflict}
	}
	return mutation.approval, true, nil
}

func (repository *effectDecisionRepository) DecideEffect(
	_ context.Context,
	state DecideEffectState,
) (EffectApproval, bool, error) {
	repository.governanceMu.Lock()
	defer repository.governanceMu.Unlock()
	request := EffectReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		PrincipalRef: state.PrincipalRef, ProjectRef: state.ProjectRef, GoalRef: state.GoalRef,
		IntentRef: state.IntentRef, IntentDigest: state.IntentDigest,
	}
	key := effectDecisionKey(state.PrincipalRef, state.ProjectRef, state.RequestRef)
	if mutation, found := repository.decisions[key]; found {
		if mutation.request != request {
			return EffectApproval{}, false, &StateError{Code: StateConflict}
		}
		return mutation.approval, false, nil
	}
	if state.OperationAt.IsZero() || !state.OperationAt.Equal(state.Approval.DecidedAt) ||
		state.AuthorizationReceipt.Ref() != state.Approval.AuthorizationReceipt.Ref() ||
		ValidatePersistedEffectApproval(state.Approval) != nil {
		return EffectApproval{}, false, &StateError{Code: StateInvalid}
	}
	record, err := repository.memoryRepository.GetGoal(context.Background(), state.GoalRef)
	if err != nil {
		return EffectApproval{}, false, err
	}
	intent, found := effectIntentByRef(record.EffectIntents, state.IntentRef)
	if !found || intent.Digest != state.IntentDigest || intent.Subject.ProjectRef != state.ProjectRef ||
		intent.Subject != state.Approval.Subject {
		return EffectApproval{}, false, &StateError{Code: StateConflict}
	}
	repository.memoryRepository.mu.Lock()
	current := repository.memoryRepository.records[state.GoalRef]
	current.EffectApprovals = append(current.EffectApprovals, state.Approval)
	repository.memoryRepository.records[state.GoalRef] = current
	repository.memoryRepository.mu.Unlock()
	repository.decisions[key] = effectDecisionMutation{request: request, approval: state.Approval}
	repository.decideCalls++
	return state.Approval, true, nil
}

func (repository *effectDecisionRepository) RecordEffectAttempt(
	_ context.Context,
	state RecordEffectAttemptState,
) (EffectAttempt, bool, error) {
	if state.OperationAt.IsZero() || state.Attempt.Ref == "" || state.Attempt.ActionFence == 0 {
		return EffectAttempt{}, false, &StateError{Code: StateInvalid}
	}
	repository.memoryRepository.mu.Lock()
	defer repository.memoryRepository.mu.Unlock()
	record, found := repository.memoryRepository.records[state.Attempt.Subject.GoalRef]
	if !found {
		return EffectAttempt{}, false, &StateError{Code: StateNotFound}
	}
	for _, attempt := range record.EffectAttempts {
		if attempt.Ref == state.Attempt.Ref {
			return attempt, false, nil
		}
	}
	record.EffectAttempts = append(record.EffectAttempts, state.Attempt)
	repository.memoryRepository.records[state.Attempt.Subject.GoalRef] = record
	return state.Attempt, true, nil
}

func TestDecideEffectIsAuthorizedRequestIdempotentAndExact(t *testing.T) {
	fixture := newEffectDecisionFixture(t, governance.SecurityCriticalityNormal, time.Minute)
	request := DecideEffectRequest{
		RequestRef: "request:effect-decision", GoalRef: fixture.intent.Subject.GoalRef,
		IntentRef: fixture.intent.Ref, ExpectedIntentDigest: fixture.intent.Digest,
		Decision: EffectApproved, Reason: "  approved for exact scope  ",
	}
	first, err := fixture.orchestrator.DecideEffect(context.Background(), fixture.access, request)
	if err != nil || !first.Created || first.Approval.Decision != EffectApproved ||
		first.Approval.Source != EffectApprovalSourceExplicitDecision ||
		!first.Approval.ExpiresAt.Equal(fixture.clock.Now().Add(time.Minute)) {
		t.Fatalf("first decision=%+v err=%v", first, err)
	}
	second, err := fixture.orchestrator.DecideEffect(context.Background(), fixture.access, request)
	if err != nil || second.Created || !reflect.DeepEqual(second.Approval, first.Approval) || fixture.repository.decideCalls != 1 {
		t.Fatalf("replay=%+v calls=%d err=%v", second, fixture.repository.decideCalls, err)
	}
	conflict := request
	conflict.Reason = "different semantics"
	if _, err := fixture.orchestrator.DecideEffect(context.Background(), fixture.access, conflict); !IsStateError(err, StateConflict) {
		t.Fatalf("semantic conflict=%v", err)
	}
	stored, err := fixture.repository.GetGoal(context.Background(), fixture.intent.Subject.GoalRef)
	if err != nil || len(stored.EffectApprovals) != 1 || stored.EffectApprovals[0].IntentDigest != fixture.intent.Digest {
		t.Fatalf("stored exact approval=%+v err=%v", stored.EffectApprovals, err)
	}
}

func TestEffectApprovalSourcesSeparateCausalAutoApprovalFromExplicitAuthority(t *testing.T) {
	fixture := newEffectDecisionFixture(t, governance.SecurityCriticalityNormal, time.Minute)
	principal, _, err := fixture.access.values()
	if err != nil {
		t.Fatal(err)
	}
	automatic := EffectApproval{
		Ref: "effect-approval:auto", RequestRef: "request:auto", RequestFingerprint: strings.Repeat("b", 64),
		IntentRef: fixture.intent.Ref, IntentDigest: fixture.intent.Digest, Subject: fixture.intent.Subject,
		ProposedBy: fixture.intent.ProposedBy, DecidedBy: fixture.intent.ProposedBy,
		Decision: EffectApproved, Source: EffectApprovalSourceDirectorDecision,
		SecurityCriticality: fixture.intent.SecurityCriticality, Reason: "causal director decision",
		IdempotencyKey: fixture.intent.IdempotencyKey, AuthorizationReceipt: fixture.intent.Authority,
		DecidedAt: fixture.intent.CreatedAt, ExpiresAt: fixture.intent.CreatedAt.Add(time.Minute),
	}
	if err := ValidateEffectApproval(fixture.intent, automatic); err != nil {
		t.Fatalf("director causal approval: %v", err)
	}

	goalIntent := fixture.intent
	goalIntent.Permission = identity.PermissionGoalsCreate
	goalIntent.Authority = effectTestAuthorization(
		t, principal, goalIntent.Subject.ProjectRef, identity.PermissionGoalsCreate,
		goalIntent.Subject.ProjectRef.String(), goalIntent.CreatedAt,
	)
	goalIntent.Digest = EffectIntentDigest(goalIntent)
	goalAutomatic := automatic
	goalAutomatic.IntentDigest = goalIntent.Digest
	goalAutomatic.Source = EffectApprovalSourceGoalConfirmation
	goalAutomatic.AuthorizationReceipt = goalIntent.Authority
	if err := ValidateEffectApproval(goalIntent, goalAutomatic); err != nil {
		t.Fatalf("Goal confirmation causal approval: %v", err)
	}

	tampered := automatic
	tampered.AuthorizationReceipt = goalIntent.Authority
	if err := ValidateEffectApproval(fixture.intent, tampered); err == nil {
		t.Fatal("automatic approval accepted unrelated authority")
	}
	critical := fixture.intent
	critical.SecurityCriticality = governance.SecurityCriticalityCritical
	critical.Digest = EffectIntentDigest(critical)
	automatic.IntentDigest = critical.Digest
	automatic.SecurityCriticality = critical.SecurityCriticality
	if err := ValidateEffectApproval(critical, automatic); err == nil {
		t.Fatal("critical effect was auto-approved")
	}
}

func TestCriticalEffectRequiresIndependentApproverAndTTL(t *testing.T) {
	fixture := newEffectDecisionFixture(t, governance.SecurityCriticalityCritical, time.Minute)
	request := DecideEffectRequest{
		RequestRef: "request:critical-effect", GoalRef: fixture.intent.Subject.GoalRef,
		IntentRef: fixture.intent.Ref, ExpectedIntentDigest: fixture.intent.Digest,
		Decision: EffectApproved, Reason: "critical approval",
	}
	if _, err := fixture.orchestrator.DecideEffect(context.Background(), fixture.access, request); err == nil ||
		err.Error() != "application.effect_critical_separation_required" {
		t.Fatalf("same proposer approved critical effect: %v", err)
	}
	reviewer := testPrincipal(t, "principal:effect-reviewer", "actor:effect-reviewer", identity.PrincipalKindHuman)
	reviewerAccess, err := NewAccess(reviewer, fixture.intent.Subject.ProjectRef)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := fixture.orchestrator.DecideEffect(context.Background(), reviewerAccess, request)
	if err != nil || !approved.Created || approved.Approval.DecidedBy != reviewer.Ref {
		t.Fatalf("independent approval=%+v err=%v", approved, err)
	}

	invalidTTL := newEffectDecisionFixture(t, governance.SecurityCriticalityNormal, 0)
	request.GoalRef, request.IntentRef, request.ExpectedIntentDigest = invalidTTL.intent.Subject.GoalRef, invalidTTL.intent.Ref, invalidTTL.intent.Digest
	request.RequestRef = "request:invalid-ttl"
	if _, err := invalidTTL.orchestrator.DecideEffect(context.Background(), invalidTTL.access, request); err == nil ||
		err.Error() != "application.effect_approval_ttl_invalid" {
		t.Fatalf("invalid TTL did not fail closed: %v", err)
	}
}

type effectDecisionFixture struct {
	orchestrator *Orchestrator
	repository   *effectDecisionRepository
	access       Access
	intent       EffectIntent
	clock        *mutableClock
}

func newEffectDecisionFixture(
	t *testing.T,
	criticality governance.SecurityCriticality,
	ttl time.Duration,
) effectDecisionFixture {
	t.Helper()
	base := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: base}
	repository := newEffectDecisionRepository()
	accessRepository := newMemoryAccessRepository()
	agent := &scriptedAgent{now: clock.Now}
	artifacts := newMemoryArtifactStore()
	orchestrator, err := New(Dependencies{
		State: repository, Access: accessRepository, Launcher: agent, Observer: agent, Controller: agent,
		Artifacts: artifacts, Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3, ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute, EffectApprovalTTL: ttl,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour, AgentCapabilities: testAgentCapabilities(),
	})
	if err != nil {
		t.Fatal(err)
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:effect-goal", Statement: "govern one effect", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item := record.Goal.WorkItems()[0]
	execution := record.Executions[0]
	repository.memoryRepository.mu.Lock()
	action := repository.memoryRepository.actions["action:launch:"+execution.Ref.String()].record
	repository.memoryRepository.mu.Unlock()
	principal, _, err := access.values()
	if err != nil {
		t.Fatal(err)
	}
	authorityRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:effect-intent", Principal: principal, ProjectRef: project,
		Permission: identity.PermissionGoalsDirect, ResourceRef: record.Goal.Ref().String(), RequestedAt: base,
	})
	if err != nil {
		t.Fatal(err)
	}
	authority, err := accessRepository.Authorize(context.Background(), authorityRequest)
	if err != nil {
		t.Fatal(err)
	}
	intent := EffectIntent{
		Ref: "effect-intent:launch", RequestRef: "request:effect-intent",
		RequestFingerprint: strings.Repeat("a", 64), ActionRef: action.Ref, ActionKind: ActionLaunchAgent,
		Kind: EffectKindAgentLaunch,
		Subject: EffectSubject{
			ProjectRef: project, GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
			SpecHash: execution.SpecHash, ActorRef: actor,
		},
		ProposedBy: principal.Ref, Permission: identity.PermissionGoalsDirect, Authority: authority,
		Demand: governance.BudgetDemand{Ref: "budget-demand:launch", Resources: governance.ResourceVector{
			Tokens: 100, ActiveTimeNS: int64(time.Minute), ProcessSlots: 1, DiskBytes: 1024,
		}},
		SecurityCriticality: criticality, ReasoningEffort: governance.ReasoningEffortMedium,
		IdempotencyKey: execution.IdempotencyKey, CreatedAt: base,
	}
	intent.Digest = EffectIntentDigest(intent)
	if err := ValidateEffectIntent(intent); err != nil {
		t.Fatal(err)
	}
	repository.memoryRepository.mu.Lock()
	current := repository.memoryRepository.records[record.Goal.Ref()]
	current.EffectIntents = append(current.EffectIntents, intent)
	actionState := repository.memoryRepository.actions[action.Ref]
	actionState.record.EffectIntentRef = intent.Ref
	repository.memoryRepository.actions[action.Ref] = actionState
	repository.memoryRepository.records[record.Goal.Ref()] = current
	repository.memoryRepository.mu.Unlock()
	return effectDecisionFixture{
		orchestrator: orchestrator, repository: repository, access: access, intent: intent, clock: clock,
	}
}

func effectDecisionKey(principal identity.PrincipalRef, project goal.ProjectRef, requestRef string) string {
	return principal.String() + "\x00" + project.String() + "\x00" + requestRef
}

func effectTestAuthorization(
	t *testing.T,
	principal identity.Principal,
	project goal.ProjectRef,
	permission identity.Permission,
	resource string,
	at time.Time,
) identity.AuthorizationReceipt {
	t.Helper()
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:test:" + string(permission), Principal: principal,
		ProjectRef: project, Permission: permission, ResourceRef: resource, RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationAllowed, Role: identity.RolePlatformAdmin,
		ReasonCode: "access.allowed", DecidedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: "authorization-receipt:test:" + string(permission), Decision: decision, RecordedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}
