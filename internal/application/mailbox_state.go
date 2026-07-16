package application

import (
	"errors"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// MailboxMessageRef identifies one immutable message admitted to the durable
// control-plane outbox. It carries no provider, process or filesystem detail.
type MailboxMessageRef struct{ value string }

func NewMailboxMessageRef(value string) (MailboxMessageRef, error) {
	if !validApplicationRef(value) || strings.ContainsRune(value, '\x00') {
		return MailboxMessageRef{}, errors.New("application.mailbox_message_ref_invalid")
	}
	return MailboxMessageRef{value: value}, nil
}

func (ref MailboxMessageRef) String() string { return ref.value }

type MailboxKind string

const (
	MailboxKindChildDelivery MailboxKind = "child_delivery"
)

type MailboxState string

const (
	MailboxStateAdmitted     MailboxState = "admitted"
	MailboxStateClaimed      MailboxState = "claimed"
	MailboxStateDelivered    MailboxState = "delivered"
	MailboxStateConsumed     MailboxState = "consumed"
	MailboxStateAcknowledged MailboxState = "acknowledged"
	MailboxStateBlocked      MailboxState = "blocked"
	MailboxStateRetired      MailboxState = "retired"
)

type MailboxOutcome string

const (
	MailboxOutcomeAcknowledged MailboxOutcome = "acknowledged"
	MailboxOutcomeBlocked      MailboxOutcome = "blocked"
)

// MailboxEndpoint binds a project principal to one exact WorkItem execution.
// A successor execution is deliberately a different recipient even when the
// principal and WorkItem remain equal.
type MailboxEndpoint struct {
	PrincipalRef identity.PrincipalRef
	WorkItemRef  goal.WorkItemRef
	ExecutionRef goal.ExecutionRef
}

// MailboxEnvelope is immutable message identity and compact context. Large
// results remain content-addressed artifacts and travel only by ref.
type MailboxEnvelope struct {
	Ref                  MailboxMessageRef
	RequestRef           string
	RequestFingerprint   string
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	TargetPlanGeneration goal.PlanGeneration
	Kind                 MailboxKind
	ParentWorkItemRef    goal.WorkItemRef
	ChildWorkItemRef     goal.WorkItemRef
	Source               MailboxEndpoint
	Recipient            MailboxEndpoint
	Summary              string
	ArtifactRefs         []goal.ArtifactRef
	ContentHash          string
	AdmittedAt           time.Time
}

type MailboxAdmissionReceipt struct {
	Ref                  string
	MessageRef           MailboxMessageRef
	RequestRef           string
	RequestFingerprint   string
	PrincipalRef         identity.PrincipalRef
	AuthorizationReceipt identity.AuthorizationReceipt
	AdmittedAt           time.Time
}

// MailboxDeliveryAttempt records one fenced at-least-once delivery. Prior
// attempts are immutable evidence and are never overwritten by reclaim.
type MailboxDeliveryAttempt struct {
	MessageRef      MailboxMessageRef
	ActionRef       string
	Recipient       MailboxEndpoint
	ClaimRequestRef string
	ClaimToken      string
	Fence           uint64
	ClaimedAt       time.Time
	LeaseUntil      time.Time
	DeliveryRef     string
	DeliveredAt     time.Time
	ConsumptionRef  string
	ConsumedAt      time.Time
}

type MailboxAcknowledgement struct {
	Ref                  string
	MessageRef           MailboxMessageRef
	ActionRef            string
	RequestRef           string
	RequestFingerprint   string
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	TargetPlanGeneration goal.PlanGeneration
	ParentWorkItemRef    goal.WorkItemRef
	ChildWorkItemRef     goal.WorkItemRef
	Recipient            MailboxEndpoint
	Fence                uint64
	Outcome              MailboxOutcome
	EffectOrReworkRef    string
	AcknowledgedAt       time.Time
	AuthorizationReceipt identity.AuthorizationReceipt
}

// MailboxRetirement is the system-owned terminal fact produced when the exact
// recipient execution fails. It is deliberately not a recipient ACK, an
// authorization receipt or a Goal child-handoff resolution.
type MailboxRetirement struct {
	MessageRef            MailboxMessageRef
	ActionRef             string
	RecipientExecutionRef goal.ExecutionRef
	FailureCode           string
	RetiredAt             time.Time
}

type MailboxRecord struct {
	Envelope        MailboxEnvelope
	Admission       MailboxAdmissionReceipt
	Action          ActionRecord
	Attempts        []MailboxDeliveryAttempt
	Acknowledgement *MailboxAcknowledgement
	Retirement      *MailboxRetirement
	State           MailboxState
}

type MailboxClaim struct {
	Record  MailboxRecord
	Attempt MailboxDeliveryAttempt
}

type MailboxMutationKind string

const (
	MailboxMutationAdmit       MailboxMutationKind = "admit"
	MailboxMutationClaim       MailboxMutationKind = "claim"
	MailboxMutationDeliver     MailboxMutationKind = "deliver"
	MailboxMutationConsume     MailboxMutationKind = "consume"
	MailboxMutationAcknowledge MailboxMutationKind = "acknowledge"
	MailboxMutationBlock       MailboxMutationKind = "block"
)

type MailboxReplayRequest struct {
	Kind               MailboxMutationKind
	RequestRef         string
	RequestFingerprint string
	PrincipalRef       identity.PrincipalRef
	ProjectRef         goal.ProjectRef
	GoalRef            goal.GoalRef
	MessageRef         MailboxMessageRef
}

type MailboxReplayRecord struct {
	Record          MailboxRecord
	Claim           MailboxClaim
	Acknowledgement MailboxAcknowledgement
}

type AdmitMailboxState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	Envelope             MailboxEnvelope
	Admission            MailboxAdmissionReceipt
	Action               ActionRecord
	Event                EventRecord
	OperationAt          time.Time
}

type ClaimMailboxState struct {
	RequestRef            string
	RequestFingerprint    string
	AuthorizationReceipt  identity.AuthorizationReceipt
	PrincipalRef          identity.PrincipalRef
	ProjectRef            goal.ProjectRef
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientExecutionRef goal.ExecutionRef
	Token                 string
	LeaseDuration         time.Duration
	RequestedAt           time.Time
}

type MarkMailboxDeliveredState struct {
	RequestRef            string
	RequestFingerprint    string
	AuthorizationReceipt  identity.AuthorizationReceipt
	PrincipalRef          identity.PrincipalRef
	ProjectRef            goal.ProjectRef
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientExecutionRef goal.ExecutionRef
	ClaimToken            string
	Fence                 uint64
	DeliveryRef           string
	OperationAt           time.Time
}

type ConsumeMailboxState struct {
	RequestRef            string
	RequestFingerprint    string
	AuthorizationReceipt  identity.AuthorizationReceipt
	PrincipalRef          identity.PrincipalRef
	ProjectRef            goal.ProjectRef
	GoalRef               goal.GoalRef
	MessageRef            MailboxMessageRef
	RecipientExecutionRef goal.ExecutionRef
	ClaimToken            string
	Fence                 uint64
	ConsumptionRef        string
	ConsumptionReceipt    ActionConsumptionReceipt
	OperationAt           time.Time
}

type ResolveMailboxState struct {
	RequestRef             string
	RequestFingerprint     string
	AuthorizationReceipt   identity.AuthorizationReceipt
	PrincipalRef           identity.PrincipalRef
	ProjectRef             goal.ProjectRef
	GoalRef                goal.GoalRef
	MessageRef             MailboxMessageRef
	RecipientExecutionRef  goal.ExecutionRef
	ClaimToken             string
	Fence                  uint64
	ExpectedGoalRevision   goal.Revision
	ExpectedPlanGeneration goal.PlanGeneration
	Goal                   goal.Goal
	Acknowledgement        MailboxAcknowledgement
	Events                 []EventRecord
	OperationAt            time.Time
}
