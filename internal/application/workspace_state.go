package application

import (
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

// WorkspacePreparedState consumes one fenced prepare action and publishes the
// exact binding, its external-effect receipt and the next launch atomically.
type WorkspacePreparedState struct {
	Claim         ActionClaim
	Execution     ExecutionRecord
	Binding       WorkspaceBinding
	NextAction    ActionRecord
	EffectReceipt EffectReceipt
	Event         EventRecord
	OperationAt   time.Time
}

// ExecutionOutputReadyState preserves agent output before Git commit. Goal and
// WorkItem remain running; only integration may accept their evidence.
type ExecutionOutputReadyState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Execution            ExecutionRecord
	Artifact             ArtifactRecord
	Attestation          AttestationRecord
	NextAction           ActionRecord
	BudgetSettlement     *governance.BudgetSettlement
	Event                EventRecord
	OperationAt          time.Time
}

// ChangeCommittedState consumes commit_change and records one immutable local
// commit. New write work schedules attest_test; legacy work without declared
// tests remains pending with no invented evidence or next action.
type ChangeCommittedState struct {
	Claim         ActionClaim
	Execution     ExecutionRecord
	ChangeSet     ChangeSet
	NextAction    *ActionRecord
	EffectReceipt EffectReceipt
	Event         EventRecord
	OperationAt   time.Time
}

// TestAttestedState publishes both immutable CAS occurrences and one typed
// verdict in the same state transaction that consumes attest_test. PASS leaves
// integration to the separately authorized IntegrateChange use case.
type TestAttestedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	ManifestArtifact     ArtifactRecord
	ReportArtifact       ArtifactRecord
	Attestation          AttestationRecord
	EffectReceipt        EffectReceipt
	NewExecutions        []ExecutionRecord
	NewActions           []ActionRecord
	Events               []EventRecord
	OperationAt          time.Time
}

type AdmitIntegrationState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	PrincipalRef         identity.PrincipalRef
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	ChangeRef            ports.ChangeSetRef
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Action               ActionRecord
	OperationAt          time.Time
}

// IntegrationResultState consumes integrate_change. Integrated results may
// complete the WorkItem; conflict/stale results preserve pending ChangeSet and
// use the existing interrupted-work path rather than a Git lifecycle.
type IntegrationResultState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	Observation          MergeObservation
	Integration          IntegrationReceipt
	EffectReceipt        EffectReceipt
	NewExecutions        []ExecutionRecord
	NewActions           []ActionRecord
	Events               []EventRecord
	OperationAt          time.Time
}

type PendingChangeQuery struct {
	ProjectRef    goal.ProjectRef
	RepositoryRef identity.RepositoryRef
	ActorRef      goal.ActorRef
	Limit         int
}

type PendingChange struct {
	Binding     WorkspaceBinding
	ChangeSet   ChangeSet
	Observation MergeObservation
	Integration IntegrationReceipt
}
