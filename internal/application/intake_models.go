package application

import (
	"context"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

type IntakeOperation string

const (
	IntakeOperationCreate IntakeOperation = "create"
	IntakeOperationApply  IntakeOperation = "apply"
)

// IntakeReceipt is the immutable result of one accepted intake mutation.
// Ref, RequestFingerprint and StateDigest are deterministic lower-case SHA-256
// values scoped by actor, project and request.
type IntakeReceipt struct {
	Ref                     string
	RequestRef              string
	RequestFingerprint      string
	Operation               IntakeOperation
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	PreviousRevision        intake.Revision
	Revision                intake.Revision
	StateDigest             string
	AuthorizationReceiptRef string
}

// IntakeRecord couples the immutable mutation result with its stable receipt.
// Replay returns the exact historical record; GetIntake returns the current
// record for the actor/project/state identity.
type IntakeRecord struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	State      intake.State
	Receipt    IntakeReceipt
}

type IntakeReplayRequest struct {
	RequestRef              string
	RequestFingerprint      string
	Operation               IntakeOperation
	ActorRef                goal.ActorRef
	ProjectRef              goal.ProjectRef
	StateRef                intake.Ref
	AuthorizationReceiptRef string
}

type IntakeCreateState struct {
	RequestRef           string
	RequestFingerprint   string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	AuthorizationReceipt identity.AuthorizationReceipt
	State                intake.State
	Receipt              IntakeReceipt
}

type IntakeApplyState struct {
	RequestRef           string
	RequestFingerprint   string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	AuthorizationReceipt identity.AuthorizationReceipt
	ExpectedRevision     intake.Revision
	State                intake.State
	Receipt              IntakeReceipt
}

// IntakeStore is the durable state port consumed by IntakeService. Mutations
// atomically enforce actor/project/request idempotency and state revision CAS.
// ReplayIntake's bool means found. CreateIntake and ApplyIntake return true
// only for the first committed mutation; an equal replay returns the original
// record and false. Divergent replay or stale CAS returns StateConflict.
type IntakeStore interface {
	ReplayIntake(context.Context, IntakeReplayRequest) (IntakeRecord, bool, error)
	CreateIntake(context.Context, IntakeCreateState) (IntakeRecord, bool, error)
	GetIntake(context.Context, goal.ActorRef, goal.ProjectRef, intake.Ref) (IntakeRecord, error)
	ApplyIntake(context.Context, IntakeApplyState) (IntakeRecord, bool, error)
}

type CreateIntakeRequest struct {
	RequestRef           string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	StateRef             intake.Ref
	Policy               intake.Policy
	AuthorizationReceipt identity.AuthorizationReceipt
}

type GetIntakeRequest struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	StateRef   intake.Ref
}

type ApplyIntakeRequest struct {
	RequestRef           string
	ActorRef             goal.ActorRef
	ProjectRef           goal.ProjectRef
	Change               intake.Change
	AuthorizationReceipt identity.AuthorizationReceipt
}

type IntakeResult struct {
	Record  IntakeRecord
	Changed bool
}

// IntakeSnapshot is the adapter-safe state-centric representation of
// intake.State. It contains no independent lifecycle or mutation authority.
type IntakeSnapshot struct {
	Schema         string            `json:"schema"`
	Ref            intake.Ref        `json:"ref"`
	Revision       intake.Revision   `json:"revision"`
	Policy         intake.Policy     `json:"policy"`
	QuestionRounds uint32            `json:"question_rounds"`
	Issues         []intake.Issue    `json:"issues,omitempty"`
	Questions      []intake.Question `json:"questions,omitempty"`
	Decisions      []intake.Decision `json:"decisions,omitempty"`
	History        []intake.Mutation `json:"history,omitempty"`
}
