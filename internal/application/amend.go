package application

import (
	"context"
	"errors"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type AmendRequest struct {
	RequestRef             string
	SourceGoalRef          goal.GoalRef
	ExpectedSourceRevision goal.Revision
	ExpectedSourceSpecHash string
	Statement              string
	NormalizedObjective    string
	Reason                 string
	Confirm                bool
}

type AmendResult struct {
	Record  GoalRecord
	Created bool
}

// Amend creates a new pending Goal generation. It never mutates the source
// Goal and never copies source executions or evidence into the successor.
func (orchestrator *Orchestrator) Amend(ctx context.Context, access Access, request AmendRequest) (AmendResult, error) {
	if orchestrator == nil {
		return AmendResult{}, errors.New("application.unavailable")
	}
	if !request.Confirm {
		return AmendResult{}, errors.New("application.confirmation_required")
	}
	if err := validateAmendRequest(request); err != nil {
		return AmendResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return AmendResult{}, err
	}
	request.NormalizedObjective = normalizedObjective(request.Statement, request.NormalizedObjective)
	now := orchestrator.clock.Now()
	authorizationReceipt, err := orchestrator.authorize(
		ctx, access, identity.PermissionGoalsAmend, request.SourceGoalRef.String(), now,
	)
	if err != nil {
		return AmendResult{}, err
	}
	now = authorizationCausalFloor(now, authorizationReceipt)

	source, err := orchestrator.state.GetGoal(ctx, request.SourceGoalRef)
	if err != nil {
		return AmendResult{}, err
	}
	if source.Goal.Project() != projectRef {
		return AmendResult{}, &StateError{Code: StateNotFound}
	}
	if source.Goal.Revision() != request.ExpectedSourceRevision ||
		source.Goal.SpecHash() != request.ExpectedSourceSpecHash {
		return AmendResult{}, &StateError{Code: StateConflict}
	}
	if !source.Goal.IsTerminal() {
		return AmendResult{}, &goal.DomainError{Code: goal.ErrorInvalidTransition, Field: "source_goal"}
	}

	successor, goalRef, err := orchestrator.buildSuccessorGoal(ctx, source.Goal, principal.ActorRef, request, now)
	if err != nil {
		return AmendResult{}, err
	}

	fingerprint := amendmentFingerprint(access, request)
	record, created, err := orchestrator.state.AmendGoal(ctx, AmendGoalState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorizationReceipt, RequestedBy: principal.Ref, ProjectRef: projectRef,
		SourceGoalRef:          request.SourceGoalRef,
		ExpectedSourceRevision: request.ExpectedSourceRevision,
		ExpectedSourceSpecHash: request.ExpectedSourceSpecHash,
		Successor:              successor,
		Events: []EventRecord{{
			Ref: "event:goal-amended:" + goalRef.String(), Kind: "goal.amended",
			GoalRef: goalRef, OccurredAt: now,
		}},
	})
	if err != nil {
		return AmendResult{}, err
	}
	if created {
		if err := validatePersistedCandidate(successor, nil, record); err != nil {
			return AmendResult{}, err
		}
	}
	if err := validateAmendedRecord(request, fingerprint, principal, projectRef, source.Goal, record); err != nil {
		return AmendResult{}, err
	}
	return AmendResult{Record: record, Created: created}, nil
}

func (orchestrator *Orchestrator) buildSuccessorGoal(
	ctx context.Context,
	source goal.Goal,
	confirmedBy goal.ActorRef,
	request AmendRequest,
	now time.Time,
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
		Ref: intentRef, Actor: source.Actor(), Project: source.Project(),
		Statement: request.Statement, SubmittedAt: now,
	})
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	appSpec, err := source.AppSpec().Amend(goal.AppSpecInput{
		Ref: appSpecRef, Intent: intent, Objective: request.NormalizedObjective,
		Reason: request.Reason, ConfirmedBy: confirmedBy, ConfirmedAt: now,
	})
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	successor, err := goal.NewSuccessorGoal(goalRef, source, appSpec, now)
	if err != nil {
		return goal.Goal{}, goal.GoalRef{}, err
	}
	return successor, goalRef, nil
}

func amendmentFingerprint(access Access, request AmendRequest) string {
	return fingerprintFields(
		"orquesta.amend.v1", access.principal.Ref.String(), access.projectRef.String(),
		request.SourceGoalRef.String(), strconv.FormatUint(uint64(request.ExpectedSourceRevision), 10),
		request.ExpectedSourceSpecHash, request.Statement,
		normalizedObjective(request.Statement, request.NormalizedObjective), request.Reason,
	)
}
