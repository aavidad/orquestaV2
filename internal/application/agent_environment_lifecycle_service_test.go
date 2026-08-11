package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestAgentEnvironmentLifecycleServiceInitializeReadsBeforeInspectAndReplaysCASWinner(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)

	t.Run("stored", func(t *testing.T) {
		store := newFakeAgentEnvironmentLifecycleStore(fixture.record)
		store.state = AgentEnvironmentLifecycleStoredState{Snapshot: fixture.initial}
		store.exists = true
		physical := newFakeAgentEnvironmentLifecyclePhysical(t, fixture.initial.Subject, fixture.initial.Token,
			&mutableClock{now: fixture.now})
		service := newAgentEnvironmentLifecycleServiceForTest(t, store, physical, &mutableClock{now: fixture.now})

		got, created, err := service.Initialize(context.Background(), InitializeAgentEnvironmentLifecycleRequest{
			LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: fixture.record,
		})
		if err != nil || created || got != fixture.initial {
			t.Fatalf("stored initialize created=%t snapshot=%+v err=%v", created, got, err)
		}
		if physical.inspectCalls != 0 {
			t.Fatalf("stored initialize inspected provider %d times", physical.inspectCalls)
		}
	})

	t.Run("lost insert race", func(t *testing.T) {
		store := newFakeAgentEnvironmentLifecycleStore(fixture.record)
		store.initialCASWinner = &fixture.initial
		clock := &mutableClock{now: fixture.now.Add(2 * time.Second)}
		physical := newFakeAgentEnvironmentLifecyclePhysical(t, fixture.initial.Subject, fixture.initial.Token, clock)
		service := newAgentEnvironmentLifecycleServiceForTest(t, store, physical, clock)

		got, created, err := service.Initialize(context.Background(), InitializeAgentEnvironmentLifecycleRequest{
			LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: fixture.record,
		})
		if err != nil || created || got != fixture.initial {
			t.Fatalf("CAS-loser initialize created=%t snapshot=%+v err=%v", created, got, err)
		}
		if physical.inspectCalls != 1 || got.RecordedAt.Equal(clock.Now()) {
			t.Fatalf("winner not replayed: inspections=%d got_at=%s local_at=%s",
				physical.inspectCalls, got.RecordedAt, clock.Now())
		}
	})
}

func TestAgentEnvironmentLifecycleServiceComposesQuiescePreserveClose(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	clock := &mutableClock{now: fixture.now.Add(time.Second)}
	order := []string{}
	store := newFakeAgentEnvironmentLifecycleStore(fixture.record)
	store.state = AgentEnvironmentLifecycleStoredState{Snapshot: fixture.initial}
	store.exists, store.order = true, &order
	physical := newFakeAgentEnvironmentLifecyclePhysical(t, fixture.initial.Subject, fixture.initial.Token, clock)
	physical.order = &order
	service := newAgentEnvironmentLifecycleServiceForTest(t, store, physical, clock)

	quiesceAction, err := (&Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		store.record, fixture.initial, ActionQuiesceAgent, clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := lifecycleServiceClaim(t, fixture.base, quiesceAction, 71, clock.Now())
	result, err := service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: claim,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleServiceContinuation(t, result, ports.AgentEnvironmentQuiesced, ActionPreserveAgentEnvironment)
	if result.NextAction.Kind == ActionStopAgent {
		t.Fatal("successful quiesce composed Stop")
	}
	// Simulate losing the terminal Store response: exact Q replay must return
	// the durable P continuation without another physical or store write.
	replayed, err := service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: claim,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleServiceContinuation(t, replayed, ports.AgentEnvironmentQuiesced, ActionPreserveAgentEnvironment)
	if physical.quiesceCalls != 1 || store.attemptWrites != 1 || store.terminalWrites != 1 {
		t.Fatalf("terminal response replay repeated write/effect: Q=%d attempts=%d terminals=%d",
			physical.quiesceCalls, store.attemptWrites, store.terminalWrites)
	}

	clock.Advance(time.Second)
	claim = lifecycleServiceClaim(t, fixture.base, *result.NextAction, 72, clock.Now())
	result, err = service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: claim,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleServiceContinuation(t, result, ports.AgentEnvironmentPreserved, ActionCloseAgentEnvironment)
	if store.state.Preservation == nil || store.state.Snapshot.Preservation.ApplicationReceiptRef == "" {
		t.Fatal("preserve terminal CAS omitted application preservation fact")
	}

	clock.Advance(time.Second)
	claim = lifecycleServiceClaim(t, fixture.base, *result.NextAction, 73, clock.Now())
	result, err = service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: claim,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Snapshot.Token.State != ports.AgentEnvironmentClosed || result.Pending || !result.ReadyToFinalize ||
		result.NextAction != nil {
		t.Fatalf("closed result invalid: %+v", result)
	}
	if physical.quiesceCalls != 1 || physical.preserveCalls != 1 || physical.closeCalls != 1 ||
		physical.reconcileQuiesceCalls+physical.reconcilePreserveCalls+physical.reconcileCloseCalls != 0 {
		t.Fatalf("physical calls Q=%d P=%d C=%d reconcile=%d/%d/%d", physical.quiesceCalls,
			physical.preserveCalls, physical.closeCalls, physical.reconcileQuiesceCalls,
			physical.reconcilePreserveCalls, physical.reconcileCloseCalls)
	}
	wantOrder := []string{
		"store.attempt:quiesce_agent", "physical.quiesce", "store.terminal:quiesce_agent",
		"store.attempt:preserve_agent_environment", "physical.preserve", "store.terminal:preserve_agent_environment",
		"store.attempt:close_agent_environment", "physical.close", "store.terminal:close_agent_environment",
	}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("causal order\n got: %v\nwant: %v", order, wantOrder)
	}
}

func TestBuildAgentEnvironmentLifecycleActionKeepsTransitionAuthorityInApplication(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	builder := (&Orchestrator{}).BuildAgentEnvironmentLifecycleAction
	action, err := builder(fixture.record, fixture.initial, ActionQuiesceAgent, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if action.Kind != ActionQuiesceAgent || action.Kind == ActionStopAgent ||
		action.EffectIntent.TargetDigest != agentEnvironmentLifecycleTargetDigest(fixture.initial, ActionQuiesceAgent) {
		t.Fatalf("application built invalid initial transition: %+v", action)
	}
	if _, err := builder(fixture.record, fixture.initial, ActionStopAgent, fixture.now); err == nil {
		t.Fatal("application lifecycle builder accepted Stop")
	}
	if _, err := builder(fixture.record, fixture.initial, ActionPreserveAgentEnvironment, fixture.now); err == nil {
		t.Fatal("application lifecycle builder skipped Quiesce")
	}
}

func TestAgentEnvironmentLifecycleServicePendingNeverReeffectsAndUsesOriginalAttempt(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	clock := &mutableClock{now: fixture.now.Add(time.Second)}
	store := newFakeAgentEnvironmentLifecycleStore(fixture.record)
	store.state, store.exists = AgentEnvironmentLifecycleStoredState{Snapshot: fixture.initial}, true
	physical := newFakeAgentEnvironmentLifecyclePhysical(t, fixture.initial.Subject, fixture.initial.Token, clock)
	physical.pendingQuiesceOnce = true
	service := newAgentEnvironmentLifecycleServiceForTest(t, store, physical, clock)
	action, err := (&Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		store.record, fixture.initial, ActionQuiesceAgent, clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	original := lifecycleServiceClaim(t, fixture.base, action, 81, clock.Now())

	result, err := service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: original,
	})
	if err != nil || !result.Pending || !store.state.Snapshot.Effect.NeedsReconciliation() {
		t.Fatalf("pending result=%+v err=%v", result, err)
	}
	physical.completePendingQuiesce()
	clock.Advance(time.Second)
	hostile := original
	hostile.Action.Kind, hostile.Action.Ref = ActionStopAgent, "action:stop:hostile"
	hostile.Token, hostile.Fence = "claim:hostile", 999
	result, err = service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: hostile,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleServiceContinuation(t, result, ports.AgentEnvironmentQuiesced, ActionPreserveAgentEnvironment)
	if physical.quiesceCalls != 1 || physical.reconcileQuiesceCalls != 1 || store.attemptWrites != 1 ||
		physical.lastReconcileQuiesce != physical.lastQuiesce {
		t.Fatalf("pending re-effect/crossed attempt: Q=%d R=%d attempts=%d request=%+v original=%+v",
			physical.quiesceCalls, physical.reconcileQuiesceCalls, store.attemptWrites,
			physical.lastReconcileQuiesce, physical.lastQuiesce)
	}
}

func TestAgentEnvironmentLifecycleServicePhysicalErrorAfterAttemptRetriesReadOnly(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	clock := &mutableClock{now: fixture.now.Add(time.Second)}
	store := newFakeAgentEnvironmentLifecycleStore(fixture.record)
	store.state, store.exists = AgentEnvironmentLifecycleStoredState{Snapshot: fixture.initial}, true
	physical := newFakeAgentEnvironmentLifecyclePhysical(t, fixture.initial.Subject, fixture.initial.Token, clock)
	physical.failQuiesceAfterApplyOnce = true
	service := newAgentEnvironmentLifecycleServiceForTest(t, store, physical, clock)
	action, err := (&Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		store.record, fixture.initial, ActionQuiesceAgent, clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	original := lifecycleServiceClaim(t, fixture.base, action, 91, clock.Now())

	result, err := service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: original,
	})
	if err == nil || !result.Pending || !store.state.Snapshot.Effect.NeedsReconciliation() {
		t.Fatalf("post-apply provider error lost Attempt: result=%+v err=%v", result, err)
	}
	clock.Advance(time.Second)
	result, err = service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record,
		Claim: ActionClaim{},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleServiceContinuation(t, result, ports.AgentEnvironmentQuiesced, ActionPreserveAgentEnvironment)
	if physical.quiesceCalls != 1 || physical.inspectCalls != 1 || physical.reconcileQuiesceCalls != 1 ||
		store.attemptWrites != 1 {
		t.Fatalf("retry was not read-only: Q=%d inspect=%d reconcile=%d attempts=%d",
			physical.quiesceCalls, physical.inspectCalls, physical.reconcileQuiesceCalls, store.attemptWrites)
	}
}

func TestAgentEnvironmentLifecycleServiceAttemptCASLoserReconcilesWinnerWithDifferentClock(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	actionAt := fixture.now.Add(time.Second)
	clock := &mutableClock{now: actionAt.Add(time.Second)}
	store := newFakeAgentEnvironmentLifecycleStore(fixture.record)
	store.state, store.exists = AgentEnvironmentLifecycleStoredState{Snapshot: fixture.initial}, true
	store.attemptCASWinnerAt = &actionAt
	physical := newFakeAgentEnvironmentLifecyclePhysical(t, fixture.initial.Subject, fixture.initial.Token, clock)
	service := newAgentEnvironmentLifecycleServiceForTest(t, store, physical, clock)
	action, err := (&Orchestrator{}).BuildAgentEnvironmentLifecycleAction(
		store.record, fixture.initial, ActionQuiesceAgent, actionAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	claim := lifecycleServiceClaim(t, fixture.base, action, 96, actionAt)

	result, err := service.Advance(context.Background(), AdvanceAgentEnvironmentLifecycleRequest{
		LaunchRequest: fixture.request, LaunchReceipt: fixture.launch, Record: store.record, Claim: claim,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pending || result.Snapshot != store.state.Snapshot || !result.Snapshot.RecordedAt.Equal(actionAt) ||
		result.Snapshot.RecordedAt.Equal(clock.Now()) {
		t.Fatalf("CAS winner not replayed: result=%+v winner=%+v", result, store.state.Snapshot)
	}
	if physical.quiesceCalls != 0 || physical.inspectCalls != 1 || store.attemptWrites != 1 {
		t.Fatalf("CAS loser mutated provider: Q=%d inspect=%d attempts=%d",
			physical.quiesceCalls, physical.inspectCalls, store.attemptWrites)
	}
}

func TestValidateAgentEnvironmentLifecycleTerminalStateRequiresClosedBeforeFinalize(t *testing.T) {
	fixture := newAgentEnvironmentLifecycleFixture(t)
	claim := lifecycleClaimFor(t, fixture.base, fixture.initial, ActionQuiesceAgent, 101, fixture.now)
	prepared := mustPrepareLifecycle(t, fixture.initial, claim, fixture.now)
	outcome, err := RecordAgentEnvironmentQuiesceOutcome(prepared, ports.AgentQuiesceReceipt{
		Subject: fixture.initial.Subject, PreviousToken: fixture.initial.Token,
		NextToken:      lifecycleToken(t, fixture.initial.Subject.ExternalRef, ports.AgentEnvironmentQuiesced, "revision:terminal-guard"),
		IdempotencyKey: prepared.Attempt.IdempotencyKey, ReceiptRef: "receipt:quiesce:terminal-guard",
		ConfirmedAt: fixture.now.Add(time.Second),
	}, fixture.now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	post := mustLifecycleTerminalOutcome(t, outcome)
	stop := claim.Action
	stop.Kind = ActionStopAgent
	post.NextAction, post.ReadyToFinalize = &stop, true
	if ValidateAgentEnvironmentLifecycleTerminalState(post) == nil {
		t.Fatal("quiesced frontier accepted Stop and finalization")
	}
}

func newAgentEnvironmentLifecycleServiceForTest(
	t *testing.T,
	store AgentEnvironmentLifecycleStore,
	physical *fakeAgentEnvironmentLifecyclePhysical,
	clock Clock,
) *AgentEnvironmentLifecycleService {
	t.Helper()
	service, err := NewAgentEnvironmentLifecycleService(AgentEnvironmentLifecycleServiceDependencies{
		Store: store, Physical: physical, Reconciler: physical, Clock: clock,
		BuildNextAction: (&Orchestrator{}).BuildAgentEnvironmentLifecycleAction,
		BuildPreservation: func(_ context.Context, receipt ports.AgentPreserveReceipt,
			record GoalRecord, at time.Time,
		) (ComprobantePreservacionEntornoAgente, error) {
			return lifecyclePreservationFact(t, record, receipt, at), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func lifecycleServiceClaim(t *testing.T, base ActionClaim, action ActionRecord, fence uint64, at time.Time) ActionClaim {
	t.Helper()
	if action.EffectApproval == nil {
		t.Fatal("lifecycle action lacks automatic approval")
	}
	claim := base
	claim.Action, claim.EffectApproval = action, *action.EffectApproval
	claim.Token, claim.WorkerRef = "claim:service:"+string(action.Kind), "worker:lifecycle-service"
	claim.DeliveryAttempt, claim.Fence, claim.LeaseUntil = 1, fence, at.Add(time.Hour)
	claim.Disposition, claim.RecoveryEffectAttemptRef = ActionClaimDispositionNormal, ""
	return claim
}

func assertLifecycleServiceContinuation(t *testing.T, result AgentEnvironmentLifecycleServiceResult,
	wantState ports.AgentEnvironmentLifecycleState, wantNext ActionKind,
) {
	t.Helper()
	if result.Snapshot.Token.State != wantState || result.Pending || result.ReadyToFinalize ||
		result.NextAction == nil || result.NextAction.Kind != wantNext || result.NextAction.Kind == ActionStopAgent {
		t.Fatalf("lifecycle continuation invalid: %+v", result)
	}
}

type fakeAgentEnvironmentLifecycleStore struct {
	state              AgentEnvironmentLifecycleStoredState
	exists             bool
	record             GoalRecord
	initialCASWinner   *AgentEnvironmentLifecycleSnapshot
	attemptCASWinnerAt *time.Time
	attemptWrites      int
	terminalWrites     int
	order              *[]string
}

func newFakeAgentEnvironmentLifecycleStore(record GoalRecord) *fakeAgentEnvironmentLifecycleStore {
	return &fakeAgentEnvironmentLifecycleStore{record: record}
}

func (store *fakeAgentEnvironmentLifecycleStore) GetAgentEnvironmentLifecycle(
	_ context.Context,
	executionRef goal.ExecutionRef,
) (AgentEnvironmentLifecycleStoredState, bool, error) {
	if store.exists && store.state.Snapshot.Subject.ExecutionRef != executionRef {
		return AgentEnvironmentLifecycleStoredState{}, false, nil
	}
	return copyAgentEnvironmentLifecycleStoredState(store.state), store.exists, nil
}

func (store *fakeAgentEnvironmentLifecycleStore) RecordAgentEnvironmentLifecycleInitial(
	_ context.Context,
	initial AgentEnvironmentLifecycleInitialState,
) (AgentEnvironmentLifecycleSnapshot, bool, error) {
	if err := ValidateAgentEnvironmentLifecycleInitialState(initial); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if store.exists {
		return store.state.Snapshot, false, nil
	}
	if store.initialCASWinner != nil {
		store.state = AgentEnvironmentLifecycleStoredState{Snapshot: *store.initialCASWinner}
		store.exists = true
		return store.state.Snapshot, false, nil
	}
	store.state, store.exists = AgentEnvironmentLifecycleStoredState{Snapshot: initial.Snapshot}, true
	return initial.Snapshot, true, nil
}

func (store *fakeAgentEnvironmentLifecycleStore) RecordAgentEnvironmentLifecycleAttempt(
	_ context.Context,
	prepared AgentEnvironmentLifecyclePreEffectState,
) (AgentEnvironmentLifecycleSnapshot, bool, error) {
	if err := ValidateAgentEnvironmentLifecycleAttemptState(prepared); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if store.exists && store.state.HasAttempt && store.state.Snapshot == prepared.Snapshot &&
		reflect.DeepEqual(store.state.Claim, prepared.Claim) && store.state.Attempt == prepared.Attempt {
		return store.state.Snapshot, false, nil
	}
	if store.attemptCASWinnerAt != nil && store.exists && !store.state.Snapshot.Effect.NeedsReconciliation() {
		winner, err := PrepareAgentEnvironmentLifecycleEffect(
			store.state.Snapshot, prepared.Claim, *store.attemptCASWinnerAt,
		)
		if err != nil {
			return AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		store.attemptCASWinnerAt = nil
		if err := appendLifecycleServiceAction(&store.record, winner.Claim.Action); err != nil {
			return AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		store.record.EffectAttempts = append(store.record.EffectAttempts, winner.Attempt)
		store.state.Snapshot, store.state.Claim, store.state.Attempt = winner.Snapshot, winner.Claim, winner.Attempt
		store.state.HasAttempt, store.state.NextAction, store.state.ReadyToFinalize = true, nil, false
		store.attemptWrites++
		return winner.Snapshot, false, nil
	}
	if !store.exists || store.state.Snapshot.Revision != prepared.ExpectedRevision ||
		store.state.Snapshot.Effect.NeedsReconciliation() ||
		(store.state.NextAction != nil && !reflect.DeepEqual(*store.state.NextAction, prepared.Claim.Action)) {
		return AgentEnvironmentLifecycleSnapshot{}, false, errors.New("test.lifecycle_attempt_conflict")
	}
	if err := appendLifecycleServiceAction(&store.record, prepared.Claim.Action); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	store.record.EffectAttempts = append(store.record.EffectAttempts, prepared.Attempt)
	store.state.Snapshot, store.state.Claim, store.state.Attempt = prepared.Snapshot, prepared.Claim, prepared.Attempt
	store.state.HasAttempt, store.state.NextAction, store.state.ReadyToFinalize = true, nil, false
	store.attemptWrites++
	store.appendOrder("store.attempt:" + string(prepared.Claim.Action.Kind))
	return prepared.Snapshot, true, nil
}

func (store *fakeAgentEnvironmentLifecycleStore) RecordAgentEnvironmentLifecycleTerminal(
	_ context.Context,
	terminal AgentEnvironmentLifecyclePostEffectState,
) (AgentEnvironmentLifecycleSnapshot, bool, error) {
	if err := ValidateAgentEnvironmentLifecycleTerminalState(terminal); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if store.exists && store.state.Snapshot == terminal.Snapshot &&
		reflect.DeepEqual(store.state.NextAction, terminal.NextAction) &&
		store.state.ReadyToFinalize == terminal.ReadyToFinalize {
		return store.state.Snapshot, false, nil
	}
	if !store.exists || !store.state.Snapshot.Effect.NeedsReconciliation() ||
		store.state.Snapshot.Revision != terminal.ExpectedRevision ||
		!reflect.DeepEqual(store.state.Claim, terminal.Claim) || store.state.Attempt != terminal.Attempt {
		return AgentEnvironmentLifecycleSnapshot{}, false, errors.New("test.lifecycle_terminal_conflict")
	}
	store.record.EffectReceipts = append(store.record.EffectReceipts, *terminal.EffectReceipt)
	store.record.ConsumptionReceipts = append(store.record.ConsumptionReceipts, terminal.ConsumptionReceipt)
	if terminal.PreservationFact != nil {
		value := *terminal.PreservationFact
		store.state.Preservation = &value
	}
	if terminal.NextAction != nil {
		if err := appendLifecycleServiceAction(&store.record, *terminal.NextAction); err != nil {
			return AgentEnvironmentLifecycleSnapshot{}, false, err
		}
		value := *terminal.NextAction
		store.state.NextAction = &value
	} else {
		store.state.NextAction = nil
	}
	store.state.Snapshot, store.state.Claim, store.state.Attempt = terminal.Snapshot, terminal.Claim, terminal.Attempt
	store.state.HasAttempt, store.state.ReadyToFinalize = true, terminal.ReadyToFinalize
	store.terminalWrites++
	store.appendOrder("store.terminal:" + string(terminal.Claim.Action.Kind))
	return terminal.Snapshot, true, nil
}

func (store *fakeAgentEnvironmentLifecycleStore) appendOrder(value string) {
	if store.order != nil {
		*store.order = append(*store.order, value)
	}
}

func appendLifecycleServiceAction(record *GoalRecord, action ActionRecord) error {
	foundIntent := false
	for _, candidate := range record.EffectIntents {
		if candidate.Ref == action.EffectIntent.Ref {
			if candidate != action.EffectIntent {
				return errors.New("test.lifecycle_intent_conflict")
			}
			foundIntent = true
		}
	}
	if !foundIntent {
		record.EffectIntents = append(record.EffectIntents, action.EffectIntent)
	}
	if action.EffectApproval == nil {
		return errors.New("test.lifecycle_approval_missing")
	}
	foundApproval := false
	for _, candidate := range record.EffectApprovals {
		if candidate.Ref == action.EffectApproval.Ref {
			if candidate != *action.EffectApproval {
				return errors.New("test.lifecycle_approval_conflict")
			}
			foundApproval = true
		}
	}
	if !foundApproval {
		record.EffectApprovals = append(record.EffectApprovals, *action.EffectApproval)
	}
	return nil
}

func copyAgentEnvironmentLifecycleStoredState(
	state AgentEnvironmentLifecycleStoredState,
) AgentEnvironmentLifecycleStoredState {
	copy := state
	if state.Preservation != nil {
		value := *state.Preservation
		copy.Preservation = &value
	}
	if state.NextAction != nil {
		value := *state.NextAction
		copy.NextAction = &value
	}
	return copy
}

type fakeAgentEnvironmentLifecyclePhysical struct {
	t       *testing.T
	subject ports.AgentEnvironmentLifecycleSubject
	token   ports.AgentEnvironmentLifecycleToken
	clock   Clock
	order   *[]string

	pendingQuiesceOnce        bool
	failQuiesceAfterApplyOnce bool
	inspectCalls              int
	quiesceCalls              int
	preserveCalls             int
	closeCalls                int
	reconcileQuiesceCalls     int
	reconcilePreserveCalls    int
	reconcileCloseCalls       int
	lastQuiesce               ports.AgentQuiesceRequest
	lastPreserve              ports.AgentPreserveRequest
	lastClose                 ports.AgentCloseRequest
	lastReconcileQuiesce      ports.AgentQuiesceRequest
	quiesceReceipt            ports.AgentQuiesceReceipt
	preserveReceipt           ports.AgentPreserveReceipt
	closeReceipt              ports.AgentCloseReceipt
}

func newFakeAgentEnvironmentLifecyclePhysical(t *testing.T, subject ports.AgentEnvironmentLifecycleSubject,
	token ports.AgentEnvironmentLifecycleToken, clock Clock,
) *fakeAgentEnvironmentLifecyclePhysical {
	return &fakeAgentEnvironmentLifecyclePhysical{t: t, subject: subject, token: token, clock: clock}
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) Inspect(
	_ context.Context,
	request ports.AgentEnvironmentInspectRequest,
) (ports.AgentEnvironmentInspectReceipt, error) {
	physical.inspectCalls++
	if request.Subject != physical.subject {
		return ports.AgentEnvironmentInspectReceipt{}, errors.New("test.inspect_subject_mismatch")
	}
	return ports.AgentEnvironmentInspectReceipt{Subject: physical.subject, Token: physical.token}, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) Quiesce(
	_ context.Context,
	request ports.AgentQuiesceRequest,
) (ports.AgentQuiesceReceipt, error) {
	physical.quiesceCalls++
	physical.appendOrder("physical.quiesce")
	physical.lastQuiesce = request
	if ports.ValidateAgentQuiesceRequest(request) != nil || request.Subject != physical.subject ||
		request.ExpectedToken != physical.token {
		return ports.AgentQuiesceReceipt{}, errors.New("test.quiesce_request_invalid")
	}
	if physical.pendingQuiesceOnce {
		physical.pendingQuiesceOnce = false
		physical.token = lifecycleToken(physical.t, physical.subject.ExternalRef,
			ports.AgentEnvironmentQuiescing, "revision:quiescing")
		return ports.AgentQuiesceReceipt{
			Subject: physical.subject, PreviousToken: request.ExpectedToken, NextToken: physical.token,
			IdempotencyKey: request.IdempotencyKey,
		}, nil
	}
	receipt := physical.terminalQuiesceReceipt(request, "revision:quiesced")
	physical.token, physical.quiesceReceipt = receipt.NextToken, receipt
	if physical.failQuiesceAfterApplyOnce {
		physical.failQuiesceAfterApplyOnce = false
		return ports.AgentQuiesceReceipt{}, errors.New("test.provider_response_lost")
	}
	return receipt, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) Preserve(
	_ context.Context,
	request ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	physical.preserveCalls++
	physical.appendOrder("physical.preserve")
	physical.lastPreserve = request
	if ports.ValidateAgentPreserveRequest(request) != nil || request.Subject != physical.subject ||
		request.ExpectedToken != physical.token {
		return ports.AgentPreserveReceipt{}, errors.New("test.preserve_request_invalid")
	}
	at := physical.clock.Now().UTC()
	receipt := ports.AgentPreserveReceipt{
		Subject: physical.subject, PreviousToken: request.ExpectedToken,
		NextToken: lifecycleToken(physical.t, physical.subject.ExternalRef,
			ports.AgentEnvironmentPreserved, "revision:preserved"),
		IdempotencyKey: request.IdempotencyKey, Manifest: lifecycleManifest(physical.t, physical.subject, at),
		ReceiptRef: "receipt:preserve:service", ConfirmedAt: at,
	}
	physical.token, physical.preserveReceipt = receipt.NextToken, receipt
	return receipt, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) Close(
	_ context.Context,
	request ports.AgentCloseRequest,
) (ports.AgentCloseReceipt, error) {
	physical.closeCalls++
	physical.appendOrder("physical.close")
	physical.lastClose = request
	if ports.ValidateAgentCloseRequest(request) != nil || request.Subject != physical.subject ||
		request.ExpectedToken != physical.token {
		return ports.AgentCloseReceipt{}, errors.New("test.close_request_invalid")
	}
	receipt := ports.AgentCloseReceipt{
		Subject: physical.subject, PreviousToken: request.ExpectedToken,
		NextToken: lifecycleToken(physical.t, physical.subject.ExternalRef,
			ports.AgentEnvironmentClosed, "revision:closed"),
		Preservation: request.Preservation, IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:close:service", ConfirmedAt: physical.clock.Now().UTC(),
	}
	physical.token, physical.closeReceipt = receipt.NextToken, receipt
	return receipt, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) ReconcileQuiesce(
	_ context.Context,
	request ports.AgentQuiesceRequest,
) (ports.AgentQuiesceReceipt, error) {
	physical.reconcileQuiesceCalls++
	physical.lastReconcileQuiesce = request
	if request != physical.lastQuiesce || physical.quiesceReceipt.ReceiptRef == "" {
		return ports.AgentQuiesceReceipt{}, errors.New("test.quiesce_receipt_missing")
	}
	return physical.quiesceReceipt, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) ReconcilePreserve(
	_ context.Context,
	request ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	physical.reconcilePreserveCalls++
	if request != physical.lastPreserve || physical.preserveReceipt.ReceiptRef == "" {
		return ports.AgentPreserveReceipt{}, errors.New("test.preserve_receipt_missing")
	}
	return physical.preserveReceipt, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) ReconcileClose(
	_ context.Context,
	request ports.AgentCloseRequest,
) (ports.AgentCloseReceipt, error) {
	physical.reconcileCloseCalls++
	if request != physical.lastClose || physical.closeReceipt.ReceiptRef == "" {
		return ports.AgentCloseReceipt{}, errors.New("test.close_receipt_missing")
	}
	return physical.closeReceipt, nil
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) terminalQuiesceReceipt(
	request ports.AgentQuiesceRequest,
	revision string,
) ports.AgentQuiesceReceipt {
	return ports.AgentQuiesceReceipt{
		Subject: physical.subject, PreviousToken: request.ExpectedToken,
		NextToken:      lifecycleToken(physical.t, physical.subject.ExternalRef, ports.AgentEnvironmentQuiesced, revision),
		IdempotencyKey: request.IdempotencyKey, ReceiptRef: "receipt:quiesce:service",
		ConfirmedAt: physical.clock.Now().UTC(),
	}
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) completePendingQuiesce() {
	receipt := physical.terminalQuiesceReceipt(physical.lastQuiesce, "revision:quiesced-after-pending")
	physical.token, physical.quiesceReceipt = receipt.NextToken, receipt
}

func (physical *fakeAgentEnvironmentLifecyclePhysical) appendOrder(value string) {
	if physical.order != nil {
		*physical.order = append(*physical.order, value)
	}
}
