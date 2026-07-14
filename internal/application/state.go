package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

// StateErrorCode is stable machine-readable repository failure information.
type StateErrorCode string

const (
	StateNotFound       StateErrorCode = "state.not_found"
	StateConflict       StateErrorCode = "state.conflict"
	StateInvalid        StateErrorCode = "state.invalid"
	StateAlreadyClaimed StateErrorCode = "state.already_claimed"
)

type StateError struct {
	Code  StateErrorCode
	Cause error
}

func (err *StateError) Error() string {
	if err == nil {
		return ""
	}
	return string(err.Code)
}

func (err *StateError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func IsStateError(err error, code StateErrorCode) bool {
	var stateErr *StateError
	return errors.As(err, &stateErr) && stateErr.Code == code
}

type ExecutionState string

const (
	ExecutionQueued    ExecutionState = "queued"
	ExecutionRunning   ExecutionState = "running"
	ExecutionSucceeded ExecutionState = "succeeded"
	ExecutionFailed    ExecutionState = "failed"
)

type ExecutionRecord struct {
	Ref                goal.ExecutionRef
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	State              ExecutionState
	ArtifactMediaType  string
	IdempotencyKey     string
	MaxOutputBytes     int64
	MaxAttempts        uint64
	ProviderRef        string
	ExternalRef        string
	CreatedAt          time.Time
	DeadlineAt         time.Time
	StartedAt          time.Time
	ProviderAcceptedAt time.Time
	LastObservedAt     time.Time
	ProviderObservedAt time.Time
	FinishedAt         time.Time
	FailureCode        string
}

type ArtifactRecord struct {
	Stored      ports.StoredArtifact
	GoalRef     goal.GoalRef
	WorkItemRef goal.WorkItemRef
	CreatedAt   time.Time
}

type AttestationRecord struct {
	Ref          goal.AttestationRef
	GoalRef      goal.GoalRef
	WorkItemRef  goal.WorkItemRef
	ExecutionRef goal.ExecutionRef
	ArtifactRef  goal.ArtifactRef
	Policy       string
	AcceptedAt   time.Time
}

type EventRecord struct {
	Ref          string
	Kind         string
	GoalRef      goal.GoalRef
	WorkItemRef  goal.WorkItemRef
	ExecutionRef goal.ExecutionRef
	OccurredAt   time.Time
}

type ActionKind string

const (
	ActionLaunchAgent  ActionKind = "launch_agent"
	ActionObserveAgent ActionKind = "observe_agent"
)

type ActionRecord struct {
	Ref          string
	Kind         ActionKind
	GoalRef      goal.GoalRef
	WorkItemRef  goal.WorkItemRef
	ExecutionRef goal.ExecutionRef
	AvailableAt  time.Time
}

type ActionClaim struct {
	Action     ActionRecord
	Token      string
	WorkerRef  string
	Attempt    uint64
	LeaseUntil time.Time
}

type ClaimRequest struct {
	WorkerRef     string
	Token         string
	Now           time.Time
	LeaseDuration time.Duration
}

type GoalRecord struct {
	RequestRef         string
	RequestFingerprint string
	Intent             goal.IntentManifest
	Goal               goal.Goal
	Execution          ExecutionRecord
	Artifacts          []ArtifactRecord
	Attestations       []AttestationRecord
}

type GoalSummary struct {
	Ref           goal.GoalRef
	IntentRef     goal.IntentRef
	ActorRef      goal.ActorRef
	ProjectRef    goal.ProjectRef
	Statement     string
	State         goal.GoalState
	Revision      goal.Revision
	CreatedAt     time.Time
	ClosedAt      time.Time
	ArtifactCount int
}

type RepositoryStatus struct {
	Goals              int64
	RunningGoals       int64
	PendingActions     int64
	QuarantinedActions int64
}

type CreateGoalState struct {
	RequestRef         string
	RequestFingerprint string
	Intent             goal.IntentManifest
	Goal               goal.Goal
	Execution          ExecutionRecord
	Action             ActionRecord
	Event              EventRecord
}

type LaunchAcceptedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	NextAction           ActionRecord
	Event                EventRecord
}

type ActionRequeuedState struct {
	Claim       ActionClaim
	Execution   ExecutionRecord
	AvailableAt time.Time
	ErrorCode   string
}

type ActionQuarantinedState struct {
	Claim     ActionClaim
	ErrorCode string
	Event     EventRecord
}

type GoalSucceededState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	Artifact             ArtifactRecord
	Attestation          AttestationRecord
	Events               []EventRecord
}

type GoalFailedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	Events               []EventRecord
}

// StateRepository is the durable state port. Every mutation is an atomic
// application-level operation; adapters never choose lifecycle transitions.
type StateRepository interface {
	CreateGoal(context.Context, CreateGoalState) (GoalRecord, bool, error)
	GetGoal(context.Context, goal.GoalRef) (GoalRecord, error)
	ListGoals(context.Context, goal.ActorRef, goal.ProjectRef, int) ([]GoalSummary, error)
	Status(context.Context) (RepositoryStatus, error)
	ClaimNextAction(context.Context, ClaimRequest) (ActionClaim, bool, error)
	RecordLaunchAccepted(context.Context, LaunchAcceptedState) error
	RequeueAction(context.Context, ActionRequeuedState) error
	QuarantineAction(context.Context, ActionQuarantinedState) error
	RecordGoalSucceeded(context.Context, GoalSucceededState) error
	RecordGoalFailed(context.Context, GoalFailedState) error
}
