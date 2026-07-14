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
	ExecutionQueued      ExecutionState = "queued"
	ExecutionDispatching ExecutionState = "dispatching"
	ExecutionRunning     ExecutionState = "running"
	ExecutionSucceeded   ExecutionState = "succeeded"
	ExecutionFailed      ExecutionState = "failed"
)

type ExecutionRecord struct {
	Ref                  goal.ExecutionRef
	GoalRef              goal.GoalRef
	WorkItemRef          goal.WorkItemRef
	AttemptNo            uint64
	MaxExecutionAttempts uint64
	ReplacesExecutionRef goal.ExecutionRef
	PlanGeneration       goal.PlanGeneration
	AppSpecGeneration    goal.AppSpecGeneration
	SpecHash             string
	State                ExecutionState
	ArtifactMediaType    string
	IdempotencyKey       string
	MaxOutputBytes       int64
	ProviderRef          string
	ModelRef             string
	AgentRef             string
	ExternalRef          string
	CreatedAt            time.Time
	DeadlineAt           time.Time
	StartedAt            time.Time
	ProviderAcceptedAt   time.Time
	LastObservedAt       time.Time
	ProviderObservedAt   time.Time
	FinishedAt           time.Time
	FailureCode          string
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
	Ref                string
	Kind               ActionKind
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	ExecutionRef       goal.ExecutionRef
	PlanGeneration     goal.PlanGeneration
	WorkItemGeneration goal.Revision
	AvailableAt        time.Time
}

type ActionClaim struct {
	Action          ActionRecord
	Token           string
	WorkerRef       string
	DeliveryAttempt uint64
	Fence           uint64
	LeaseUntil      time.Time
}

type ClaimRequest struct {
	WorkerRef     string
	Token         string
	LeaseDuration time.Duration
	Capabilities  ports.AgentCapabilities
}

type ActionConsumptionOutcome string

const (
	ActionConsumedCompleted   ActionConsumptionOutcome = "completed"
	ActionConsumedQuarantined ActionConsumptionOutcome = "quarantined"
)

// ActionConsumptionReceipt is the immutable proof that one fenced outbox
// delivery was consumed. Requeue deliberately creates no receipt.
type ActionConsumptionReceipt struct {
	ActionRef          string
	Kind               ActionKind
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	ExecutionRef       goal.ExecutionRef
	PlanGeneration     goal.PlanGeneration
	WorkItemGeneration goal.Revision
	Fence              uint64
	DeliveryAttempt    uint64
	ClaimToken         string
	WorkerRef          string
	Outcome            ActionConsumptionOutcome
	ErrorCode          string
	ConsumedAt         time.Time
}

type GoalRecord struct {
	RequestRef          string
	RequestFingerprint  string
	Goal                goal.Goal
	Executions          []ExecutionRecord
	Artifacts           []ArtifactRecord
	Attestations        []AttestationRecord
	ConsumptionReceipts []ActionConsumptionReceipt
}

type GoalSummary struct {
	Ref               goal.GoalRef
	IntentRef         goal.IntentRef
	AppSpecRef        goal.AppSpecRef
	AppSpecGeneration goal.AppSpecGeneration
	SpecHash          string
	ActorRef          goal.ActorRef
	ProjectRef        goal.ProjectRef
	Statement         string
	State             goal.GoalState
	Revision          goal.Revision
	CreatedAt         time.Time
	ClosedAt          time.Time
	ArtifactCount     int
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
	Goal               goal.Goal
	Executions         []ExecutionRecord
	Actions            []ActionRecord
	Events             []EventRecord
}

// AmendGoalState carries a fully constructed successor plus the source fence
// that the repository must revalidate atomically. Generated refs are not part
// of idempotency: an equal replay returns the already persisted successor.
type AmendGoalState struct {
	RequestRef             string
	RequestFingerprint     string
	ActorRef               goal.ActorRef
	ProjectRef             goal.ProjectRef
	SourceGoalRef          goal.GoalRef
	ExpectedSourceRevision goal.Revision
	ExpectedSourceSpecHash string
	Successor              goal.Goal
	Events                 []EventRecord
}

// LaunchPreparedState reserves the WorkItem in the aggregate before the
// external provider effect. The launch claim deliberately remains open.
type LaunchPreparedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	Event                EventRecord
	OperationAt          time.Time
}

type LaunchAcceptedState struct {
	Claim       ActionClaim
	Execution   ExecutionRecord
	NextAction  ActionRecord
	Event       EventRecord
	OperationAt time.Time
}

type ActionRequeuedState struct {
	Claim       ActionClaim
	Execution   ExecutionRecord
	AvailableAt time.Time
	ErrorCode   string
	OperationAt time.Time
}

type ActionQuarantinedState struct {
	Claim       ActionClaim
	ErrorCode   string
	Event       EventRecord
	OperationAt time.Time
}

// ExecutionReplacedState atomically consumes the failed attempt's action,
// rebinds the authoritative WorkItem and queues the next provider attempt.
type ExecutionReplacedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	FailedExecution      ExecutionRecord
	ReplacementExecution ExecutionRecord
	NextAction           ActionRecord
	Events               []EventRecord
	ErrorCode            string
	OperationAt          time.Time
}

type GoalSucceededState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	Artifact             ArtifactRecord
	Attestation          AttestationRecord
	NewExecutions        []ExecutionRecord
	NewActions           []ActionRecord
	Events               []EventRecord
	OperationAt          time.Time
}

type GoalFailedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	NewExecutions        []ExecutionRecord
	NewActions           []ActionRecord
	Events               []EventRecord
	OperationAt          time.Time
}

// StateRepository is the durable state port. Every mutation is an atomic
// application-level operation; adapters never choose lifecycle transitions.
// Mutations carrying ActionClaim must also fence the lease with the adapter's
// trusted transaction clock immediately before commit. OperationAt is causal
// lifecycle data and must never be accepted as proof that a lease is still live.
type StateRepository interface {
	CreateGoal(context.Context, CreateGoalState) (GoalRecord, bool, error)
	AmendGoal(context.Context, AmendGoalState) (GoalRecord, bool, error)
	GetGoal(context.Context, goal.GoalRef) (GoalRecord, error)
	ListGoals(context.Context, goal.ActorRef, goal.ProjectRef, int) ([]GoalSummary, error)
	Status(context.Context) (RepositoryStatus, error)
	ClaimNextAction(context.Context, ClaimRequest) (ActionClaim, bool, error)
	RecordLaunchPrepared(context.Context, LaunchPreparedState) error
	RecordLaunchAccepted(context.Context, LaunchAcceptedState) error
	RequeueAction(context.Context, ActionRequeuedState) error
	QuarantineAction(context.Context, ActionQuarantinedState) error
	RecordExecutionReplaced(context.Context, ExecutionReplacedState) error
	RecordGoalSucceeded(context.Context, GoalSucceededState) error
	RecordGoalFailed(context.Context, GoalFailedState) error
}
