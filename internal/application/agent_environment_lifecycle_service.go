package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/ports"
)

var errAgentEnvironmentLifecycleServiceInvalid = errors.New("application.agent_environment_lifecycle_service_invalid")

type AgentEnvironmentLifecycleActionBuilder func(
	GoalRecord,
	AgentEnvironmentLifecycleSnapshot,
	ActionKind,
	time.Time,
) (ActionRecord, error)

// AgentEnvironmentPreservationBuilder enriches a terminal physical Preserve
// receipt into the existing application fact. It builds but does not persist;
// persistence belongs to the same terminal CAS as receipt and continuation.
type AgentEnvironmentPreservationBuilder func(
	context.Context,
	ports.AgentPreserveReceipt,
	GoalRecord,
	time.Time,
) (ComprobantePreservacionEntornoAgente, error)

type AgentEnvironmentLifecycleServiceDependencies struct {
	Store             AgentEnvironmentLifecycleStore
	Physical          ports.AgentEnvironmentLifecycle
	Reconciler        ports.AgentEnvironmentLifecycleReconciler
	Clock             Clock
	BuildNextAction   AgentEnvironmentLifecycleActionBuilder
	BuildPreservation AgentEnvironmentPreservationBuilder
}

type AgentEnvironmentLifecycleService struct {
	store             AgentEnvironmentLifecycleStore
	physical          ports.AgentEnvironmentLifecycle
	reconciler        ports.AgentEnvironmentLifecycleReconciler
	clock             Clock
	buildNextAction   AgentEnvironmentLifecycleActionBuilder
	buildPreservation AgentEnvironmentPreservationBuilder
}

type InitializeAgentEnvironmentLifecycleRequest struct {
	LaunchRequest ports.AgentLaunchRequest
	LaunchReceipt ports.AgentLaunchReceipt
	Record        GoalRecord
}

type AdvanceAgentEnvironmentLifecycleRequest struct {
	LaunchRequest ports.AgentLaunchRequest
	LaunchReceipt ports.AgentLaunchReceipt
	Record        GoalRecord
	// Claim is consumed only when Snapshot has no unresolved Attempt. If restart
	// finds attempted, service ignores this field and reconciles stored Claim.
	Claim ActionClaim
}

type AgentEnvironmentLifecycleServiceResult struct {
	Snapshot        AgentEnvironmentLifecycleSnapshot
	NextAction      *ActionRecord
	Pending         bool
	ReadyToFinalize bool
}

func NewAgentEnvironmentLifecycleService(
	dependencies AgentEnvironmentLifecycleServiceDependencies,
) (*AgentEnvironmentLifecycleService, error) {
	if dependencies.Store == nil || dependencies.Physical == nil || dependencies.Reconciler == nil ||
		dependencies.Clock == nil || dependencies.BuildNextAction == nil || dependencies.BuildPreservation == nil {
		return nil, errAgentEnvironmentLifecycleServiceInvalid
	}
	return &AgentEnvironmentLifecycleService{
		store: dependencies.Store, physical: dependencies.Physical, reconciler: dependencies.Reconciler,
		clock: dependencies.Clock, buildNextAction: dependencies.BuildNextAction,
		buildPreservation: dependencies.BuildPreservation,
	}, nil
}

// Initialize inspects only when no durable snapshot exists. Losing the insert
// response therefore replays stored state and never turns Inspect into another
// lifecycle authority.
func (service *AgentEnvironmentLifecycleService) Initialize(
	ctx context.Context,
	request InitializeAgentEnvironmentLifecycleRequest,
) (AgentEnvironmentLifecycleSnapshot, bool, error) {
	if service == nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, errAgentEnvironmentLifecycleServiceInvalid
	}
	if err := ctx.Err(); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	stored, found, err := service.store.GetAgentEnvironmentLifecycle(ctx, request.LaunchRequest.ExecutionRef)
	if err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if found {
		replayed, replayErr := replayStoredAgentEnvironmentLifecycle(stored, request)
		return replayed, false, replayErr
	}
	subject := agentEnvironmentLifecycleSubject(request.LaunchReceipt)
	inspection, err := service.physical.Inspect(ctx, ports.AgentEnvironmentInspectRequest{Subject: subject})
	if err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	at := service.clock.Now().UTC()
	snapshot, err := NewAgentEnvironmentLifecycleSnapshot(
		request.LaunchRequest, request.LaunchReceipt, inspection, at,
	)
	if err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	state := AgentEnvironmentLifecycleInitialState{Snapshot: snapshot, OperationAt: at}
	if err := ValidateAgentEnvironmentLifecycleInitialState(state); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	persisted, created, err := service.store.RecordAgentEnvironmentLifecycleInitial(ctx, state)
	if err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	if created && persisted != snapshot {
		return AgentEnvironmentLifecycleSnapshot{}, false, errAgentEnvironmentLifecycleStateInvalid
	}
	if !created {
		winner, found, readErr := service.store.GetAgentEnvironmentLifecycle(ctx, request.LaunchRequest.ExecutionRef)
		if readErr != nil {
			return AgentEnvironmentLifecycleSnapshot{}, false, readErr
		}
		if !found || winner.Snapshot != persisted {
			return AgentEnvironmentLifecycleSnapshot{}, false, &StateError{Code: StateConflict}
		}
		replayed, replayErr := replayStoredAgentEnvironmentLifecycle(winner, request)
		return replayed, false, replayErr
	}
	if _, err := ReplayAgentEnvironmentLifecycleSnapshot(persisted,
		request.LaunchRequest, request.LaunchReceipt, request.Record, nil,
	); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, false, err
	}
	return persisted, created, nil
}

// Advance performs at most one physical mutation. Durable attempted state wins
// over any supplied Claim: after restart or lost response, only read-only
// reconciliation of the original attempt is permitted.
func (service *AgentEnvironmentLifecycleService) Advance(
	ctx context.Context,
	request AdvanceAgentEnvironmentLifecycleRequest,
) (AgentEnvironmentLifecycleServiceResult, error) {
	if service == nil {
		return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleServiceInvalid
	}
	if err := ctx.Err(); err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	stored, found, err := service.store.GetAgentEnvironmentLifecycle(ctx, request.LaunchRequest.ExecutionRef)
	if err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	if !found {
		return AgentEnvironmentLifecycleServiceResult{}, &StateError{Code: StateNotFound}
	}
	if _, err := replayStoredAgentEnvironmentLifecycle(stored, InitializeAgentEnvironmentLifecycleRequest{
		LaunchRequest: request.LaunchRequest, LaunchReceipt: request.LaunchReceipt, Record: request.Record,
	}); err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	if stored.Snapshot.Effect.NeedsReconciliation() {
		return service.reconcile(ctx, request, stored)
	}
	// A terminal-write response may be lost. Replaying the exact claim returns
	// the durable terminal result; it never interprets that claim as authority
	// for the next lifecycle effect.
	if !stored.Snapshot.Effect.IsEmpty() && sameAgentEnvironmentLifecycleClaim(stored.Claim, request.Claim) {
		return resultFromStoredAgentEnvironmentLifecycle(stored)
	}
	if stored.Snapshot.Token.State == ports.AgentEnvironmentClosed {
		return resultFromStoredAgentEnvironmentLifecycle(stored)
	}
	at := service.clock.Now().UTC()
	prepared, err := PrepareAgentEnvironmentLifecycleEffect(stored.Snapshot, request.Claim, at)
	if err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	persisted, created, err := service.store.RecordAgentEnvironmentLifecycleAttempt(ctx, prepared)
	if err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	if created {
		if persisted != prepared.Snapshot {
			return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
		}
		return service.executePhysicalOnce(ctx, request, prepared)
	}
	replayed, found, readErr := service.store.GetAgentEnvironmentLifecycle(ctx, request.LaunchRequest.ExecutionRef)
	if readErr != nil {
		return AgentEnvironmentLifecycleServiceResult{}, readErr
	}
	if !found {
		return AgentEnvironmentLifecycleServiceResult{}, &StateError{Code: StateConflict}
	}
	if !sameAgentEnvironmentLifecycleClaim(replayed.Claim, request.Claim) {
		return AgentEnvironmentLifecycleServiceResult{}, &StateError{Code: StateConflict}
	}
	if replayed.Snapshot.Effect.NeedsReconciliation() {
		return service.reconcile(ctx, request, replayed)
	}
	return resultFromStoredAgentEnvironmentLifecycle(replayed)
}

func (service *AgentEnvironmentLifecycleService) executePhysicalOnce(
	ctx context.Context,
	request AdvanceAgentEnvironmentLifecycleRequest,
	prepared AgentEnvironmentLifecyclePreEffectState,
) (AgentEnvironmentLifecycleServiceResult, error) {
	var (
		outcome AgentEnvironmentLifecycleOutcome
		err     error
	)
	subject := prepared.Snapshot.Subject
	expected := prepared.Snapshot.Effect.ExpectedToken
	key := prepared.Attempt.IdempotencyKey
	switch prepared.Claim.Action.Kind {
	case ActionQuiesceAgent:
		var receipt ports.AgentQuiesceReceipt
		receipt, err = service.physical.Quiesce(ctx, ports.AgentQuiesceRequest{
			Subject: subject, ExpectedToken: expected, IdempotencyKey: key,
		})
		if err == nil {
			outcome, err = RecordAgentEnvironmentQuiesceOutcome(prepared, receipt, service.clock.Now().UTC())
		}
	case ActionPreserveAgentEnvironment:
		var receipt ports.AgentPreserveReceipt
		receipt, err = service.physical.Preserve(ctx, ports.AgentPreserveRequest{
			Subject: subject, ExpectedToken: expected, IdempotencyKey: key,
		})
		if err == nil {
			at := service.clock.Now().UTC()
			var preservation *ComprobantePreservacionEntornoAgente
			if receipt.NextToken.State == ports.AgentEnvironmentPreserved {
				fact, buildErr := service.buildPreservation(ctx, receipt, request.Record, at)
				if buildErr != nil {
					return agentEnvironmentLifecycleResult(prepared.Snapshot, nil, true, false), buildErr
				}
				preservation = &fact
			}
			outcome, err = RecordAgentEnvironmentPreserveOutcome(
				prepared, receipt, preservation, request.Record, at,
			)
		}
	case ActionCloseAgentEnvironment:
		var receipt ports.AgentCloseReceipt
		receipt, err = service.physical.Close(ctx, ports.AgentCloseRequest{
			Subject: subject, ExpectedToken: expected, Preservation: prepared.Snapshot.Preservation,
			IdempotencyKey: key,
		})
		if err == nil {
			outcome, err = RecordAgentEnvironmentCloseOutcome(prepared, receipt, service.clock.Now().UTC())
		}
	default:
		return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
	}
	if err != nil {
		return agentEnvironmentLifecycleResult(prepared.Snapshot, nil, true, false), err
	}
	if outcome.Reconciliation != nil {
		return agentEnvironmentLifecycleResult(prepared.Snapshot, nil, true, false), nil
	}
	if outcome.Terminal == nil {
		return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return service.persistTerminal(ctx, request.Record, *outcome.Terminal)
}

func (service *AgentEnvironmentLifecycleService) reconcile(
	ctx context.Context,
	request AdvanceAgentEnvironmentLifecycleRequest,
	stored AgentEnvironmentLifecycleStoredState,
) (AgentEnvironmentLifecycleServiceResult, error) {
	if err := validateStoredAgentEnvironmentLifecycleAttempt(stored); err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	reconciliation, err := AgentEnvironmentLifecycleReconciliationRequest(stored.Snapshot)
	if err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	inspection, err := service.physical.Inspect(ctx, reconciliation.InspectRequest)
	if err != nil {
		return agentEnvironmentLifecycleResult(stored.Snapshot, nil, true, false), err
	}
	if ports.ValidateAgentEnvironmentInspectReceipt(reconciliation.InspectRequest, inspection) != nil {
		return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
	}
	_, terminal := agentEnvironmentLifecycleOutcomeStates(reconciliation.ActionKind)
	if inspection.Token.State != terminal {
		return agentEnvironmentLifecycleResult(stored.Snapshot, nil, true, false), nil
	}
	at := service.clock.Now().UTC()
	var post AgentEnvironmentLifecyclePostEffectState
	switch reconciliation.ActionKind {
	case ActionQuiesceAgent:
		receipt, reconcileErr := service.reconciler.ReconcileQuiesce(ctx, *reconciliation.ReceiptRequest.Quiesce)
		if reconcileErr != nil {
			return agentEnvironmentLifecycleResult(stored.Snapshot, nil, true, false), reconcileErr
		}
		post, err = RecordAgentEnvironmentQuiesceReconciliationOutcome(
			stored.Snapshot, stored.Claim, stored.Attempt, inspection, receipt, at,
		)
	case ActionPreserveAgentEnvironment:
		receipt, reconcileErr := service.reconciler.ReconcilePreserve(ctx, *reconciliation.ReceiptRequest.Preserve)
		if reconcileErr != nil {
			return agentEnvironmentLifecycleResult(stored.Snapshot, nil, true, false), reconcileErr
		}
		fact, buildErr := service.buildPreservation(ctx, receipt, request.Record, at)
		if buildErr != nil {
			return agentEnvironmentLifecycleResult(stored.Snapshot, nil, true, false), buildErr
		}
		post, err = RecordAgentEnvironmentPreserveReconciliationOutcome(
			stored.Snapshot, stored.Claim, stored.Attempt, inspection, receipt, &fact, request.Record, at,
		)
	case ActionCloseAgentEnvironment:
		receipt, reconcileErr := service.reconciler.ReconcileClose(ctx, *reconciliation.ReceiptRequest.Close)
		if reconcileErr != nil {
			return agentEnvironmentLifecycleResult(stored.Snapshot, nil, true, false), reconcileErr
		}
		post, err = RecordAgentEnvironmentCloseReconciliationOutcome(
			stored.Snapshot, stored.Claim, stored.Attempt, inspection, receipt, at,
		)
	default:
		err = errAgentEnvironmentLifecycleStateInvalid
	}
	if err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	return service.persistTerminal(ctx, request.Record, post)
}

func (service *AgentEnvironmentLifecycleService) persistTerminal(
	ctx context.Context,
	record GoalRecord,
	post AgentEnvironmentLifecyclePostEffectState,
) (AgentEnvironmentLifecycleServiceResult, error) {
	wantNext, hasNext := agentEnvironmentLifecycleAction(post.Snapshot.Token.State)
	if hasNext {
		next, err := service.buildNextAction(record, post.Snapshot, wantNext, post.OperationAt)
		if err != nil {
			return AgentEnvironmentLifecycleServiceResult{}, err
		}
		post.NextAction = &next
	} else if post.Snapshot.Token.State == ports.AgentEnvironmentClosed {
		post.ReadyToFinalize = true
	} else {
		return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
	}
	if err := ValidateAgentEnvironmentLifecycleTerminalState(post); err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	persisted, written, err := service.store.RecordAgentEnvironmentLifecycleTerminal(ctx, post)
	if err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	if written {
		if persisted != post.Snapshot {
			return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
		}
		return agentEnvironmentLifecycleResult(persisted, post.NextAction, false, post.ReadyToFinalize), nil
	}
	winner, found, readErr := service.store.GetAgentEnvironmentLifecycle(ctx, post.Snapshot.Subject.ExecutionRef)
	if readErr != nil {
		return AgentEnvironmentLifecycleServiceResult{}, readErr
	}
	if !found || winner.Snapshot != persisted || winner.Snapshot.Effect.NeedsReconciliation() ||
		!sameAgentEnvironmentLifecycleClaim(winner.Claim, post.Claim) {
		return AgentEnvironmentLifecycleServiceResult{}, &StateError{Code: StateConflict}
	}
	result, resultErr := resultFromStoredAgentEnvironmentLifecycle(winner)
	if resultErr != nil {
		return AgentEnvironmentLifecycleServiceResult{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return result, nil
}

func replayStoredAgentEnvironmentLifecycle(
	stored AgentEnvironmentLifecycleStoredState,
	request InitializeAgentEnvironmentLifecycleRequest,
) (AgentEnvironmentLifecycleSnapshot, error) {
	if err := validateStoredAgentEnvironmentLifecycle(stored); err != nil {
		return AgentEnvironmentLifecycleSnapshot{}, err
	}
	return ReplayAgentEnvironmentLifecycleSnapshot(
		stored.Snapshot, request.LaunchRequest, request.LaunchReceipt, request.Record, stored.Preservation,
	)
}

func validateStoredAgentEnvironmentLifecycle(stored AgentEnvironmentLifecycleStoredState) error {
	if validateAgentEnvironmentLifecycleSnapshot(stored.Snapshot) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if stored.Snapshot.Effect.IsEmpty() {
		if stored.HasAttempt || stored.Claim != (ActionClaim{}) || stored.Attempt != (EffectAttempt{}) ||
			stored.Preservation != nil || stored.NextAction != nil || stored.ReadyToFinalize {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	if err := validateStoredAgentEnvironmentLifecycleAttempt(stored); err != nil {
		return err
	}
	requiresPreservation := agentEnvironmentLifecycleRequiresPreservation(stored.Snapshot.Token.State)
	if requiresPreservation != (stored.Preservation != nil) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if stored.Preservation != nil && stored.Preservation.Ref != stored.Snapshot.Preservation.ApplicationReceiptRef {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if stored.Snapshot.Effect.NeedsReconciliation() {
		if stored.NextAction != nil || stored.ReadyToFinalize {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	wantNext, hasNext := agentEnvironmentLifecycleAction(stored.Snapshot.Token.State)
	if hasNext {
		if stored.NextAction == nil || stored.ReadyToFinalize ||
			validateAgentEnvironmentLifecycleNextAction(
				*stored.NextAction, stored.Snapshot, wantNext, stored.Snapshot.RecordedAt,
			) != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	if stored.Snapshot.Token.State != ports.AgentEnvironmentClosed || stored.NextAction != nil || !stored.ReadyToFinalize {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateStoredAgentEnvironmentLifecycleAttempt(stored AgentEnvironmentLifecycleStoredState) error {
	effect := stored.Snapshot.Effect
	authority := stored.Snapshot
	authority.Token = effect.ExpectedToken
	if !stored.HasAttempt || effect.IsEmpty() || stored.Claim == (ActionClaim{}) || stored.Attempt == (EffectAttempt{}) ||
		validateAgentEnvironmentLifecycleClaim(authority, stored.Claim, effect.ActionKind, stored.Attempt.StartedAt) != nil ||
		validateEffectAttempt(stored.Claim, stored.Attempt) != nil || effect.ActionRef != stored.Claim.Action.Ref ||
		effect.AttemptRef != stored.Attempt.Ref || effect.ActionFence != stored.Attempt.ActionFence ||
		effect.DeliveryAttempt != stored.Claim.DeliveryAttempt || effect.WorkItemGeneration != stored.Claim.Action.WorkItemGeneration ||
		effect.ClaimToken != stored.Claim.Token || effect.WorkerRef != stored.Claim.WorkerRef ||
		effect.IdempotencyKey != stored.Attempt.IdempotencyKey {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func ValidateAgentEnvironmentLifecycleInitialState(state AgentEnvironmentLifecycleInitialState) error {
	if validateAgentEnvironmentLifecycleSnapshot(state.Snapshot) != nil || state.Snapshot.Revision != 1 ||
		state.Snapshot.Token.State != ports.AgentEnvironmentActive || !state.Snapshot.Effect.IsEmpty() ||
		state.OperationAt.IsZero() || !state.Snapshot.RecordedAt.Equal(state.OperationAt) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func ValidateAgentEnvironmentLifecycleAttemptState(state AgentEnvironmentLifecyclePreEffectState) error {
	return validateAgentEnvironmentLifecyclePreEffect(state)
}

func ValidateAgentEnvironmentLifecycleTerminalState(state AgentEnvironmentLifecyclePostEffectState) error {
	if validateAgentEnvironmentLifecyclePostEffect(state) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	wantNext, hasNext := agentEnvironmentLifecycleAction(state.Snapshot.Token.State)
	if hasNext {
		if state.ReadyToFinalize || state.NextAction == nil ||
			validateAgentEnvironmentLifecycleNextAction(*state.NextAction, state.Snapshot, wantNext, state.OperationAt) != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	if state.Snapshot.Token.State != ports.AgentEnvironmentClosed || !state.ReadyToFinalize || state.NextAction != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentLifecycleNextAction(
	action ActionRecord,
	snapshot AgentEnvironmentLifecycleSnapshot,
	want ActionKind,
	at time.Time,
) error {
	intent := action.EffectIntent
	if action.Kind != want || want == ActionStopAgent || action.Ref != agentEnvironmentLifecycleActionRef(want, snapshot.Subject.ExecutionRef) ||
		action.GoalRef != snapshot.Subject.GoalRef || action.WorkItemRef != snapshot.Subject.WorkItemRef ||
		action.ExecutionRef != snapshot.Subject.ExecutionRef || action.PlanGeneration != snapshot.Subject.PlanGeneration ||
		action.WorkItemGeneration == 0 || !action.AvailableAt.Equal(at) || action.ControlRef != "" ||
		action.ChangeRef.String() != "" || action.ExpectedTargetOID != "" || action.ReviewGateDigest != "" ||
		action.CouncilResolution != nil || action.EffectIntentRef == "" || action.EffectIntentRef != intent.Ref ||
		intent.ActionRef != action.Ref || intent.ActionKind != action.Kind ||
		intent.Kind != lifecycleEffectKindForAction(action.Kind) || intent.Subject.GoalRef != snapshot.Subject.GoalRef ||
		intent.Subject.WorkItemRef != snapshot.Subject.WorkItemRef || intent.Subject.ExecutionRef != snapshot.Subject.ExecutionRef ||
		intent.Subject.PlanGeneration != snapshot.Subject.PlanGeneration ||
		intent.Subject.AppSpecGeneration != snapshot.Subject.AppSpecGeneration || intent.Subject.SpecHash != snapshot.Subject.SpecHash ||
		intent.Ref != "effect-intent:"+action.Ref ||
		intent.TargetDigest != agentEnvironmentLifecycleTargetDigest(snapshot, want) ||
		ValidateEffectIntent(intent) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if action.EffectApproval != nil && ValidateEffectApproval(intent, *action.EffectApproval) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func agentEnvironmentLifecycleSubject(receipt ports.AgentLaunchReceipt) ports.AgentEnvironmentLifecycleSubject {
	return ports.AgentEnvironmentLifecycleSubject{
		ExecutionRef: receipt.ExecutionRef, GoalRef: receipt.GoalRef, WorkItemRef: receipt.WorkItemRef,
		PlanGeneration: receipt.PlanGeneration, AppSpecGeneration: receipt.AppSpecGeneration,
		ExecutionAttempt: receipt.ExecutionAttempt, SpecHash: receipt.SpecHash,
		ProviderRef: receipt.ProviderRef, ModelRef: receipt.ModelRef, AgentRef: receipt.AgentRef,
		ExternalRef: receipt.ExternalRef,
	}
}

func agentEnvironmentLifecycleResult(
	snapshot AgentEnvironmentLifecycleSnapshot,
	next *ActionRecord,
	pending bool,
	ready bool,
) AgentEnvironmentLifecycleServiceResult {
	var copy *ActionRecord
	if next != nil {
		value := *next
		copy = &value
	}
	return AgentEnvironmentLifecycleServiceResult{
		Snapshot: snapshot, NextAction: copy, Pending: pending, ReadyToFinalize: ready,
	}
}

func resultFromStoredAgentEnvironmentLifecycle(
	stored AgentEnvironmentLifecycleStoredState,
) (AgentEnvironmentLifecycleServiceResult, error) {
	if err := validateStoredAgentEnvironmentLifecycle(stored); err != nil {
		return AgentEnvironmentLifecycleServiceResult{}, err
	}
	return agentEnvironmentLifecycleResult(
		stored.Snapshot, stored.NextAction, stored.Snapshot.Effect.NeedsReconciliation(), stored.ReadyToFinalize,
	), nil
}

func sameAgentEnvironmentLifecycleClaim(left ActionClaim, right ActionClaim) bool {
	leftApproval, rightApproval := left.Action.EffectApproval, right.Action.EffectApproval
	left.Action.EffectApproval, right.Action.EffectApproval = nil, nil
	if left != right || (leftApproval == nil) != (rightApproval == nil) {
		return false
	}
	return leftApproval == nil || *leftApproval == *rightApproval
}
