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

// CouncilSkipState is an authorized human fact before any round or fact.
type CouncilSkipState struct {
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	PrincipalRef         identity.PrincipalRef
	RoundSubject         council.Subject
	Skip                 council.Skip
	Record               CouncilSkipRecord
	Events               []EventRecord
	OperationAt          time.Time
}
