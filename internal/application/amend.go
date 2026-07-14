package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"

	"orquesta/internal/goal"
)

type AmendRequest struct {
	RequestRef             string
	ActorRef               goal.ActorRef
	ProjectRef             goal.ProjectRef
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
func (orchestrator *Orchestrator) Amend(ctx context.Context, request AmendRequest) (AmendResult, error) {
	if orchestrator == nil {
		return AmendResult{}, errors.New("application.unavailable")
	}
	if !request.Confirm {
		return AmendResult{}, errors.New("application.confirmation_required")
	}
	if err := validateAmendRequest(request); err != nil {
		return AmendResult{}, err
	}
	request.NormalizedObjective = normalizedObjective(request.Statement, request.NormalizedObjective)

	source, err := orchestrator.state.GetGoal(ctx, request.SourceGoalRef)
	if err != nil {
		return AmendResult{}, err
	}
	if source.Goal.Actor() != request.ActorRef || source.Goal.Project() != request.ProjectRef {
		return AmendResult{}, &StateError{Code: StateNotFound}
	}
	if source.Goal.Revision() != request.ExpectedSourceRevision ||
		source.Goal.SpecHash() != request.ExpectedSourceSpecHash {
		return AmendResult{}, &StateError{Code: StateConflict}
	}
	if !source.Goal.IsTerminal() {
		return AmendResult{}, &goal.DomainError{Code: goal.ErrorInvalidTransition, Field: "source_goal"}
	}

	now := orchestrator.clock.Now()
	intentRef, err := newIntentRef(ctx, orchestrator.ids)
	if err != nil {
		return AmendResult{}, err
	}
	appSpecRef, err := newAppSpecRef(ctx, orchestrator.ids)
	if err != nil {
		return AmendResult{}, err
	}
	goalRef, err := newGoalRef(ctx, orchestrator.ids)
	if err != nil {
		return AmendResult{}, err
	}
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: intentRef, Actor: request.ActorRef, Project: request.ProjectRef,
		Statement: request.Statement, SubmittedAt: now,
	})
	if err != nil {
		return AmendResult{}, err
	}
	appSpec, err := source.Goal.AppSpec().Amend(goal.AppSpecInput{
		Ref: appSpecRef, Intent: intent, Objective: request.NormalizedObjective,
		Reason: request.Reason, ConfirmedBy: request.ActorRef, ConfirmedAt: now,
	})
	if err != nil {
		return AmendResult{}, err
	}
	successor, err := goal.NewSuccessorGoal(goalRef, source.Goal, appSpec, now)
	if err != nil {
		return AmendResult{}, err
	}

	fingerprint := amendmentFingerprint(request)
	record, created, err := orchestrator.state.AmendGoal(ctx, AmendGoalState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
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
	if err := validateAmendedRecord(request, fingerprint, source.Goal, record); err != nil {
		return AmendResult{}, err
	}
	return AmendResult{Record: record, Created: created}, nil
}

func amendmentFingerprint(request AmendRequest) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.amend.v1")
	writeFingerprintField(digest, request.ActorRef.String())
	writeFingerprintField(digest, request.ProjectRef.String())
	writeFingerprintField(digest, request.SourceGoalRef.String())
	writeFingerprintField(digest, strconv.FormatUint(uint64(request.ExpectedSourceRevision), 10))
	writeFingerprintField(digest, request.ExpectedSourceSpecHash)
	writeFingerprintField(digest, request.Statement)
	writeFingerprintField(digest, normalizedObjective(request.Statement, request.NormalizedObjective))
	writeFingerprintField(digest, request.Reason)
	return hex.EncodeToString(digest.Sum(nil))
}
