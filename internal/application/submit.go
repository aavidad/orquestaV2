package application

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"

	"orquesta/internal/goal"
)

type SubmitRequest struct {
	RequestRef string
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	Statement  string
}

type SubmitResult struct {
	Record  GoalRecord
	Created bool
}

func (orchestrator *Orchestrator) Submit(ctx context.Context, request SubmitRequest) (SubmitResult, error) {
	if orchestrator == nil {
		return SubmitResult{}, errors.New("application.unavailable")
	}
	if err := validateSubmitRequest(request); err != nil {
		return SubmitResult{}, err
	}
	now := orchestrator.clock.Now()
	intentRef, err := newIntentRef(ctx, orchestrator.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	goalRef, err := newGoalRef(ctx, orchestrator.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	workItemRef, err := newWorkItemRef(ctx, orchestrator.ids)
	if err != nil {
		return SubmitResult{}, err
	}
	executionRef, err := newExecutionRef(ctx, orchestrator.ids)
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
	aggregate, err := goal.NewGoal(goalRef, intent, now)
	if err != nil {
		return SubmitResult{}, err
	}
	item, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: workItemRef, Goal: goalRef, Actor: request.ActorRef,
		Project: request.ProjectRef, Objective: request.Statement, CreatedAt: now,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	aggregate, err = aggregate.AddWorkItem(aggregate.Revision(), item)
	if err != nil {
		return SubmitResult{}, err
	}
	aggregate, err = aggregate.Start(aggregate.Revision(), now)
	if err != nil {
		return SubmitResult{}, err
	}

	execution := ExecutionRecord{
		Ref: executionRef, GoalRef: goalRef, WorkItemRef: workItemRef,
		State: ExecutionQueued, ArtifactMediaType: agentArtifactMediaType,
		IdempotencyKey: "execution:" + executionRef.String(),
		MaxOutputBytes: orchestrator.maxOutputBytes,
		MaxAttempts:    orchestrator.maxActionAttempts,
		CreatedAt:      now, DeadlineAt: now.Add(orchestrator.executionTimeout),
	}
	action := ActionRecord{
		Ref: "action:launch:" + executionRef.String(), Kind: ActionLaunchAgent,
		GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		AvailableAt: now,
	}
	event := EventRecord{
		Ref: "event:goal-created:" + goalRef.String(), Kind: "goal.created",
		GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		OccurredAt: now,
	}
	fingerprint := submissionFingerprint(request)
	record, created, err := orchestrator.state.CreateGoal(ctx, CreateGoalState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		Intent: intent, Goal: aggregate,
		Execution: execution, Action: action, Event: event,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	if err := validateCreatedRecord(request, fingerprint, record); err != nil {
		return SubmitResult{}, err
	}
	return SubmitResult{Record: record, Created: created}, nil
}

func submissionFingerprint(request SubmitRequest) string {
	digest := sha256.New()
	writeFingerprintField(digest, request.ActorRef.String())
	writeFingerprintField(digest, request.ProjectRef.String())
	writeFingerprintField(digest, request.Statement)
	return hex.EncodeToString(digest.Sum(nil))
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
