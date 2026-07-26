package application

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const initialAppSpecReason = "operator.initial_confirmation"

type SubmitRequest struct {
	RequestRef          string
	Statement           string
	NormalizedObjective string
	Confirm             bool
	Plan                *PlanSpec
}

type SubmitResult struct {
	Record  GoalRecord
	Created bool
}

func (orchestrator *Orchestrator) Submit(ctx context.Context, access Access, request SubmitRequest) (SubmitResult, error) {
	if orchestrator == nil {
		return SubmitResult{}, errors.New("application.unavailable")
	}
	if !request.Confirm {
		return SubmitResult{}, errors.New("application.confirmation_required")
	}
	if err := validateSubmitRequest(request); err != nil {
		return SubmitResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return SubmitResult{}, err
	}
	request.NormalizedObjective = normalizedObjective(request.Statement, request.NormalizedObjective)
	now := orchestrator.clock.Now()
	authorizationReceipt, err := orchestrator.authorize(
		ctx, access, identity.PermissionGoalsCreate, projectRef.String(), now,
	)
	if err != nil {
		return SubmitResult{}, err
	}
	now = authorizationCausalFloor(now, authorizationReceipt)
	fingerprint := submissionFingerprint(access, request)
	createState, err := orchestrator.prepareGoalCreation(
		ctx, request, principal, projectRef, authorizationReceipt, now,
		initialAppSpecReason, fingerprint,
	)
	if err != nil {
		return SubmitResult{}, err
	}
	record, created, err := orchestrator.state.CreateGoal(ctx, createState)
	if err != nil {
		return SubmitResult{}, err
	}
	if created {
		if err := validatePersistedCandidate(
			createState.Goal, createState.Executions, record,
		); err != nil {
			return SubmitResult{}, err
		}
	}
	if err := validateCreatedRecord(request, fingerprint, principal, projectRef, record); err != nil {
		return SubmitResult{}, err
	}
	return SubmitResult{Record: record, Created: created}, nil
}

func (orchestrator *Orchestrator) buildSubmittedGoal(
	ctx context.Context,
	request SubmitRequest,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	now time.Time,
	reason string,
) (goal.Goal, goal.GoalRef, error) {
	intentRef, err := newIntentRef(ctx, orchestrator.ids)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	appSpecRef, err := newAppSpecRef(ctx, orchestrator.ids)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	goalRef, err := newGoalRef(ctx, orchestrator.ids)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: intentRef, Actor: principal.ActorRef, Project: projectRef,
		Statement: request.Statement, SubmittedAt: now,
	})
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	appSpec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref: appSpecRef, Intent: intent, Objective: request.NormalizedObjective,
		Reason: reason, ConfirmedBy: principal.ActorRef, ConfirmedAt: now,
	})
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	aggregate, err := goal.NewGoal(goalRef, appSpec, now)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	plan, err := orchestrator.compilePlan(ctx, request, goalRef, principal.ActorRef, projectRef, now)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	aggregate, err = aggregate.ApplyPlan(aggregate.Revision(), plan)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	aggregate, err = aggregate.Start(aggregate.Revision(), now)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	return aggregate, goalRef, nil
}

func (orchestrator *Orchestrator) prepareGoalCreation(
	ctx context.Context,
	request SubmitRequest,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	authorizationReceipt identity.AuthorizationReceipt,
	now time.Time,
	reason string,
	fingerprint string,
) (CreateGoalState, error) {
	aggregate, goalRef, err := orchestrator.buildSubmittedGoal(
		ctx, request, principal, projectRef, now, reason,
	)
	if err != nil {
		return CreateGoalState{}, err
	}
	authorities := workItemAuthorities(
		aggregate.WorkItems(), principal.Ref, identity.PermissionGoalsCreate,
		EffectApprovalSourceGoalConfirmation, authorizationReceipt, now,
	)
	executions, actions, scheduledEvents, err := orchestrator.scheduleReady(
		ctx, aggregate, nil, authorities,
		orchestrator.budgetPolicy.effectPolicy(), now,
	)
	if err != nil {
		return CreateGoalState{}, err
	}
	events := append([]EventRecord{{
		Ref: "event:goal-created:" + goalRef.String(), Kind: "goal.created",
		GoalRef: goalRef, OccurredAt: now,
	}}, scheduledEvents...)
	return CreateGoalState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorizationReceipt, RequestedBy: principal.Ref,
		Goal: aggregate, Executions: executions, Actions: actions, Events: events,
		WorkItemAuthorities: authorities,
		BudgetEnvelopes: orchestrator.budgetPolicy.envelopes(
			projectRef, goalRef, now,
		),
	}, nil
}

func submissionFingerprint(access Access, request SubmitRequest) string {
	digest := fingerprintDigest(
		"orquesta.submit.v2", access.principal.Ref.String(), access.projectRef.String(),
		request.Statement, normalizedObjective(request.Statement, request.NormalizedObjective),
	)
	writePlanFingerprint(digest, request.Plan)
	return fingerprintHex(digest)
}

func normalizedObjective(statement, objective string) string {
	if normalized := strings.TrimSpace(objective); normalized != "" {
		return normalized
	}
	return strings.TrimSpace(statement)
}

func writeFingerprintField(digest hash.Hash, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(value))
}

func fingerprintDigest(version string, fields ...string) hash.Hash {
	digest := sha256.New()
	writeFingerprintField(digest, version)
	for _, field := range fields {
		writeFingerprintField(digest, field)
	}
	return digest
}

func fingerprintFields(version string, fields ...string) string {
	return fingerprintHex(fingerprintDigest(version, fields...))
}

func fingerprintHex(digest hash.Hash) string { return hex.EncodeToString(digest.Sum(nil)) }

func newGoalDomainRef[T any](ctx context.Context, ids IDGenerator, kind string, parse func(string) (T, error)) (T, error) {
	value, err := ids.NewID(ctx, kind)
	if err != nil {
		var zero T
		return zero, err
	}
	return parse(value)
}

func newIntentRef(ctx context.Context, ids IDGenerator) (goal.IntentRef, error) {
	return newGoalDomainRef(ctx, ids, "intent", goal.NewIntentRef)
}
func newAppSpecRef(ctx context.Context, ids IDGenerator) (goal.AppSpecRef, error) {
	return newGoalDomainRef(ctx, ids, "app-spec", goal.NewAppSpecRef)
}
func newGoalRef(ctx context.Context, ids IDGenerator) (goal.GoalRef, error) {
	return newGoalDomainRef(ctx, ids, "goal", goal.NewGoalRef)
}
func newWorkItemRef(ctx context.Context, ids IDGenerator) (goal.WorkItemRef, error) {
	return newGoalDomainRef(ctx, ids, "work-item", goal.NewWorkItemRef)
}
func newExecutionRef(ctx context.Context, ids IDGenerator) (goal.ExecutionRef, error) {
	return newGoalDomainRef(ctx, ids, "execution", goal.NewExecutionRef)
}
