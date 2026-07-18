package application

import (
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type DirectorMutationKind string

const (
	DirectorMutationClaim DirectorMutationKind = "claim"
	DirectorMutationRenew DirectorMutationKind = "renew"
	DirectorMutationPlan  DirectorMutationKind = "plan"
)

// DirectorLeaseRecord is the sole transferable authority to propose changes
// to one Goal. Lease validity is decided with the repository transaction clock.
type DirectorLeaseRecord struct {
	GoalRef      goal.GoalRef
	PrincipalRef identity.PrincipalRef
	Token        string
	Fence        uint64
	LeaseUntil   time.Time
}

// DirectorDecisionRecord is immutable causal audit for one accepted plan
// proposal. Lease tokens are deliberately absent from durable audit.
type DirectorDecisionRecord struct {
	Ref                    string
	RequestRef             string
	RequestFingerprint     string
	GoalRef                goal.GoalRef
	PrincipalRef           identity.PrincipalRef
	LeaseFence             uint64
	SourceGoalRevision     goal.Revision
	SourcePlanGeneration   goal.PlanGeneration
	Cause                  goal.ReplanCause
	SourceWorkItemRef      goal.WorkItemRef
	SourceWorkItemRevision goal.Revision
	SourceExecutionRef     goal.ExecutionRef
	SourceExecutionAttempt uint64
	AppliedGoalRevision    goal.Revision
	AppliedPlanGeneration  goal.PlanGeneration
	Reason                 string
	DecidedAt              time.Time
	AuthorizationReceipt   identity.AuthorizationReceipt
}

// DirectorReplayRequest performs a read-only idempotency lookup before a new
// authorization receipt, ID allocation or aggregate read. Access has already
// bound the authenticated principal and explicit project in application.
type DirectorReplayRequest struct {
	Kind               DirectorMutationKind
	RequestRef         string
	RequestFingerprint string
	PrincipalRef       identity.PrincipalRef
	ProjectRef         goal.ProjectRef
	GoalRef            goal.GoalRef
}

type DirectorReplayRecord struct {
	Lease    DirectorLeaseRecord
	Decision DirectorDecisionRecord
}

// ClaimDirectorState asks the active StateRepository to claim or take over a
// lease atomically. The repository derives lease_until from its trusted clock.
type ClaimDirectorState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	PrincipalRef         identity.PrincipalRef
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	Token                string
	LeaseDuration        time.Duration
	RequestedAt          time.Time
}

// RenewDirectorState renews the exact live token/fence pair. The repository
// must reject expired, replaced, wrong-project and wrong-principal leases.
type RenewDirectorState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	PrincipalRef         identity.PrincipalRef
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	Token                string
	Fence                uint64
	LeaseDuration        time.Duration
	RequestedAt          time.Time
}

// ApplyDirectorPlanState is one atomic Goal mutation, event append, execution
// schedule, outbox write and immutable Director decision. The compact decision
// is the idempotent response; GetGoal remains the only full-state projection.
type ApplyDirectorPlanState struct {
	RequestRef               string
	RequestFingerprint       string
	AuthorizationReceipt     identity.AuthorizationReceipt
	PrincipalRef             identity.PrincipalRef
	ProjectRef               goal.ProjectRef
	GoalRef                  goal.GoalRef
	LeaseToken               string
	LeaseFence               uint64
	ExpectedGoalRevision     goal.Revision
	ExpectedPlanGeneration   goal.PlanGeneration
	ExpectedWorkItemRevision goal.Revision
	Goal                     goal.Goal
	UpdatedExecutions        []ExecutionRecord
	NewExecutions            []ExecutionRecord
	NewActions               []ActionRecord
	NewWorkItemAuthorities   []WorkItemAuthority
	RetireActionRefs         []string
	Events                   []EventRecord
	Decision                 DirectorDecisionRecord
	OperationAt              time.Time
}
