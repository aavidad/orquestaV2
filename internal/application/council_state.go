package application

import (
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

// OpenCouncilRoundState appends one immutable round and its three prebuilt
// launches in same state transaction. Caller supplies V18-approved subject.
type OpenCouncilRoundState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	PrincipalRef         identity.PrincipalRef
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	LeaseToken           string
	LeaseFence           uint64
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Round                CouncilRoundRecord
	Executions           []ExecutionRecord
	Actions              []ActionRecord
	Events               []EventRecord
	OperationAt          time.Time
}

// CouncilContributionState consumes exactly one Council observation. Decision
// is absent until all three role facts exist.
type CouncilContributionState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Execution            ExecutionRecord
	Artifact             ArtifactRecord
	Fact                 council.ContributionFact
	Decision             *CouncilDecisionRecord
	BudgetSettlement     *governance.BudgetSettlement
	Events               []EventRecord
	OperationAt          time.Time
}

// CouncilExecutionReplaced retries one Council role without rebinding the
// WorkItem authority or touching its two peer roles.
type CouncilExecutionReplacedState struct {
	Claim                  ActionClaim
	ExpectedGoalRevision   goal.Revision
	ExpectedItemRevision   goal.Revision
	ExpectedExecutionState ExecutionState
	FailedExecution        ExecutionRecord
	ReplacementExecution   ExecutionRecord
	NextAction             ActionRecord
	BudgetSettlement       *governance.BudgetSettlement
	Events                 []EventRecord
	OperationAt            time.Time
}

type CouncilParticipantRetirement struct {
	Execution     ExecutionRecord
	ExpectedState ExecutionState
}

// CouncilExecutionFailed preserves a failed/exhausted role as a durable fact
// and atomically begins causal cleanup of the remaining live round.
type CouncilExecutionFailedState struct {
	Claim                  ActionClaim
	ExpectedGoalRevision   goal.Revision
	ExpectedItemRevision   goal.Revision
	ExpectedExecutionState ExecutionState
	Execution              ExecutionRecord
	Goal                   goal.Goal
	AuthorExecution        ExecutionRecord
	RetiredPeers           []CouncilParticipantRetirement
	RetireActionRefs       []string
	CleanupControls        []ControlRecord
	CleanupActions         []ActionRecord
	ResolvedCleanup        *ControlRecord
	BudgetSettlement       *governance.BudgetSettlement
	Events                 []EventRecord
	OperationAt            time.Time
}

// CouncilSkipState is an authorized human fact before any round or fact.
type CouncilSkipState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	PrincipalRef         identity.PrincipalRef
	RoundSubject         council.Subject
	Skip                 council.Skip
	Record               CouncilSkipRecord
	Events               []EventRecord
	OperationAt          time.Time
}
