package application

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"strings"

	"orquesta/internal/goal"
)

const initialAppSpecReason = "operator.initial_confirmation"

type SubmitRequest struct {
	RequestRef          string
	ActorRef            goal.ActorRef
	ProjectRef          goal.ProjectRef
	Statement           string
	NormalizedObjective string
	Confirm             bool
	Plan                *PlanSpec
}

type SubmitResult struct {
	Record  GoalRecord
	Created bool
}

func (orchestrator *Orchestrator) Submit(ctx context.Context, request SubmitRequest) (SubmitResult, error) {
	if orchestrator == nil {
		return SubmitResult{}, errors.New("application.unavailable")
	}
	if !request.Confirm {
		return SubmitResult{}, errors.New("application.confirmation_required")
	}
	if err := validateSubmitRequest(request); err != nil {
		return SubmitResult{}, err
	}
	request.NormalizedObjective = normalizedObjective(request.Statement, request.NormalizedObjective)
	now := orchestrator.clock.Now()
	intentRef, err := newIntentRef(ctx, orchestrator.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	appSpecRef, err := newAppSpecRef(ctx, orchestrator.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	goalRef, err := newGoalRef(ctx, orchestrator.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: intentRef, Actor: request.ActorRef, Project: request.ProjectRef,
		Statement: request.Statement, SubmittedAt: now,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	appSpec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref: appSpecRef, Intent: intent, Objective: request.NormalizedObjective,
		Reason: initialAppSpecReason, ConfirmedBy: request.ActorRef, ConfirmedAt: now,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	aggregate, err := goal.NewGoal(goalRef, appSpec, now)
	if err != nil {
		return SubmitResult{}, err
	}
	plan, err := orchestrator.compilePlan(ctx, request, goalRef, now)
	if err != nil {
		return SubmitResult{}, err
	}
	aggregate, err = aggregate.ApplyPlan(aggregate.Revision(), plan)
	if err != nil {
		return SubmitResult{}, err
	}
	aggregate, err = aggregate.Start(aggregate.Revision(), now)
	if err != nil {
		return SubmitResult{}, err
	}

	executions, actions, scheduledEvents, err := orchestrator.scheduleReady(ctx, aggregate, nil, now)
	if err != nil {
		return SubmitResult{}, err
	}
	events := append([]EventRecord{{
		Ref: "event:goal-created:" + goalRef.String(), Kind: "goal.created",
		GoalRef: goalRef, OccurredAt: now,
	}}, scheduledEvents...)
	fingerprint := submissionFingerprint(request)
	record, created, err := orchestrator.state.CreateGoal(ctx, CreateGoalState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		Goal:       aggregate,
		Executions: executions, Actions: actions, Events: events,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	if created {
		if err := validatePersistedCandidate(aggregate, executions, record); err != nil {
			return SubmitResult{}, err
		}
	}
	if err := validateCreatedRecord(request, fingerprint, record); err != nil {
		return SubmitResult{}, err
	}
	return SubmitResult{Record: record, Created: created}, nil
}

func submissionFingerprint(request SubmitRequest) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.submit.v2")
	writeFingerprintField(digest, request.ActorRef.String())
	writeFingerprintField(digest, request.ProjectRef.String())
	writeFingerprintField(digest, request.Statement)
	writeFingerprintField(digest, normalizedObjective(request.Statement, request.NormalizedObjective))
	writePlanFingerprint(digest, request.Plan)
	return hex.EncodeToString(digest.Sum(nil))
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

func newIntentRef(ctx context.Context, ids IDGenerator) (goal.IntentRef, error) {
	value, err := ids.NewID(ctx, "intent")
	if err != nil {
		return goal.IntentRef{}, err
	}
	return goal.NewIntentRef(value)
}

func newAppSpecRef(ctx context.Context, ids IDGenerator) (goal.AppSpecRef, error) {
	value, err := ids.NewID(ctx, "app-spec")
	if err != nil {
		return goal.AppSpecRef{}, err
	}
	return goal.NewAppSpecRef(value)
}

func newGoalRef(ctx context.Context, ids IDGenerator) (goal.GoalRef, error) {
	value, err := ids.NewID(ctx, "goal")
	if err != nil {
		return goal.GoalRef{}, err
	}
	return goal.NewGoalRef(value)
}

func newWorkItemRef(ctx context.Context, ids IDGenerator) (goal.WorkItemRef, error) {
	value, err := ids.NewID(ctx, "work-item")
	if err != nil {
		return goal.WorkItemRef{}, err
	}
	return goal.NewWorkItemRef(value)
}

func newExecutionRef(ctx context.Context, ids IDGenerator) (goal.ExecutionRef, error) {
	value, err := ids.NewID(ctx, "execution")
	if err != nil {
		return goal.ExecutionRef{}, err
	}
	return goal.NewExecutionRef(value)
}
