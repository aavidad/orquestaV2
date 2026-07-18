package application

import (
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type ControlOperation string

const (
	ControlPause  ControlOperation = "pause"
	ControlResume ControlOperation = "resume"
	ControlCancel ControlOperation = "cancel"
	ControlStop   ControlOperation = "stop"
	ControlRetry  ControlOperation = "retry"
)

type ControlTarget string

const (
	ControlTargetGoal      ControlTarget = "goal"
	ControlTargetWorkItem  ControlTarget = "work_item"
	ControlTargetExecution ControlTarget = "execution"
)

type ControlStatus string

const (
	ControlRequested ControlStatus = "requested"
	ControlConfirmed ControlStatus = "confirmed"
	// ControlSuperseded is terminal lineage, not provider success. It is used
	// only when an exact forced Stop atomically replaces its pending
	// cooperative Stop for the same Execution.
	ControlSuperseded ControlStatus = "superseded"
)

type ControlRequest struct {
	RequestRef                string
	Operation                 ControlOperation
	Target                    ControlTarget
	GoalRef                   goal.GoalRef
	ExpectedGoalRevision      goal.Revision
	ExpectedPlanGeneration    goal.PlanGeneration
	ExpectedAppSpecGeneration goal.AppSpecGeneration
	ExpectedSpecHash          string
	WorkItemRef               goal.WorkItemRef
	ExpectedWorkItemRevision  goal.Revision
	ExecutionRef              goal.ExecutionRef
	ExpectedExecutionAttempt  uint64
	Mode                      ports.AgentStopMode
	Reason                    string
}

type ControlRecord struct {
	Ref                    string
	RequestRef             string
	RequestFingerprint     string
	PrincipalRef           identity.PrincipalRef
	ProjectRef             goal.ProjectRef
	GoalRef                goal.GoalRef
	WorkItemRef            goal.WorkItemRef
	WorkItemRevision       goal.Revision
	ExecutionRef           goal.ExecutionRef
	ExecutionAttempt       uint64
	Operation              ControlOperation
	Target                 ControlTarget
	Mode                   ports.AgentStopMode
	Reason                 string
	GoalRevision           goal.Revision
	PlanGeneration         goal.PlanGeneration
	AppSpecGeneration      goal.AppSpecGeneration
	SpecHash               string
	Status                 ControlStatus
	RequestedAt            time.Time
	ConfirmedAt            time.Time
	ReceiptRef             string
	SupersedesControlRef   string
	SupersededAt           time.Time
	SupersededByControlRef string
	AuthorizationReceipt   identity.AuthorizationReceipt
}

type ControlResult struct {
	Control ControlRecord
	Created bool
}

type ControlReplayRequest struct {
	RequestRef         string
	RequestFingerprint string
	PrincipalRef       identity.PrincipalRef
	ProjectRef         goal.ProjectRef
	GoalRef            goal.GoalRef
}

// ApplyControlState is the only control write. Goal, execution projections,
// existing outbox actions, events and the immutable control receipt change in
// one StateRepository transaction.
type ApplyControlState struct {
	RequestRef                         string
	RequestFingerprint                 string
	AuthorizationReceipt               identity.AuthorizationReceipt
	PrincipalRef                       identity.PrincipalRef
	ProjectRef                         goal.ProjectRef
	GoalRef                            goal.GoalRef
	ExpectedGoalRevision               goal.Revision
	ExpectedPlanGeneration             goal.PlanGeneration
	ExpectedWorkItemRevision           goal.Revision
	ExpectedExecutionState             ExecutionState
	ExpectedControlStatus              ControlStatus
	Claim                              ActionClaim
	ClaimErrorCode                     string
	Goal                               goal.Goal
	Executions                         []ExecutionRecord
	NewActions                         []ActionRecord
	RetireActionRefs                   []string
	RetireMailboxForExecutionRef       goal.ExecutionRef
	RequireMailboxClearForExecutionRef goal.ExecutionRef
	EffectReceipt                      *EffectReceipt
	BudgetSettlement                   *governance.BudgetSettlement
	// SupersededControl carries the complete typed old->new Stop lineage. The
	// repository validates and persists it in the same transaction as the new
	// forced Stop and its replacement outbox action.
	SupersededControl *ControlRecord
	Events            []EventRecord
	Control           ControlRecord
	OperationAt       time.Time
}
