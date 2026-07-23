package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

// StateErrorCode is stable machine-readable repository failure information.
type StateErrorCode string

const (
	StateNotFound       StateErrorCode = "state.not_found"
	StateConflict       StateErrorCode = "state.conflict"
	StateInvalid        StateErrorCode = "state.invalid"
	StateAlreadyClaimed StateErrorCode = "state.already_claimed"
	// StateRecipientMailboxActive means an execution replacement lost the
	// atomic race against an unresolved mailbox addressed to that exact
	// execution. The application must fail the recipient attempt and retire
	// those inbox records; it must never readdress them implicitly.
	StateRecipientMailboxActive StateErrorCode = "state.recipient_mailbox_active"
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
	ExecutionQueued              ExecutionState = "queued"
	ExecutionDispatching         ExecutionState = "dispatching"
	ExecutionRunning             ExecutionState = "running"
	ExecutionAwaitingCommit      ExecutionState = "awaiting_commit"
	ExecutionAwaitingAttestation ExecutionState = "awaiting_attestation"
	ExecutionAwaitingIntegration ExecutionState = "awaiting_integration"
	ExecutionSucceeded           ExecutionState = "succeeded"
	ExecutionFailed              ExecutionState = "failed"
	ExecutionCanceled            ExecutionState = "canceled"
	ExecutionStopped             ExecutionState = "stopped"
)

type ExecutionPurpose string

const (
	ExecutionPurposeWork              ExecutionPurpose = "work"
	ExecutionPurposeAuthor            ExecutionPurpose = "author"
	ExecutionPurposePrimaryReview     ExecutionPurpose = "primary_review"
	ExecutionPurposeAdversarialReview ExecutionPurpose = "adversarial_review"
	ExecutionPurposeCouncilProposer   ExecutionPurpose = "council_proposer"
	ExecutionPurposeCouncilCritic     ExecutionPurpose = "council_critic"
	ExecutionPurposeCouncilArbiter    ExecutionPurpose = "council_arbiter"
)

type ArtifactKind string

const (
	ArtifactKindAgentOutput         ArtifactKind = "agent_output"
	ArtifactKindTestSubjectManifest ArtifactKind = "test_subject_manifest"
	ArtifactKindTestReport          ArtifactKind = "test_attestation_report"
	ArtifactKindReviewAssessment    ArtifactKind = "review_assessment"
	ArtifactKindReviewDiagnostic    ArtifactKind = "review_diagnostic"
	ArtifactKindCouncilContribution ArtifactKind = "council_contribution"
)

type AttestationKind string

const (
	AttestationKindArtifactProvenance AttestationKind = "artifact_provenance"
	AttestationKindRequiredTests      AttestationKind = "required_tests"
)

type AttestationVerdict string

const (
	AttestationVerdictObserved AttestationVerdict = "observed"
	AttestationVerdictPassed   AttestationVerdict = "passed"
	AttestationVerdictFailed   AttestationVerdict = "failed"
)

type ExecutionRecord struct {
	Ref                   goal.ExecutionRef
	GoalRef               goal.GoalRef
	WorkItemRef           goal.WorkItemRef
	AttemptNo             uint64
	MaxExecutionAttempts  uint64
	ReplacesExecutionRef  goal.ExecutionRef
	PlanGeneration        goal.PlanGeneration
	AppSpecGeneration     goal.AppSpecGeneration
	SpecHash              string
	RepositoryRef         identity.RepositoryRef
	State                 ExecutionState
	ArtifactMediaType     string
	IdempotencyKey        string
	MaxOutputBytes        int64
	ProviderRef           string
	ModelRef              string
	AgentRef              string
	ExternalRef           string
	ExecutionWorkspaceRef ports.ExecutionWorkspaceRef
	BudgetReservationRef  string
	EffectIntentRef       string
	LaunchReceiptRef      string
	CreatedAt             time.Time
	DeadlineAt            time.Time
	StartedAt             time.Time
	ProviderAcceptedAt    time.Time
	LastObservedAt        time.Time
	ProviderObservedAt    time.Time
	FinishedAt            time.Time
	FailureCode           string
	// RecipientMailboxRetired is durable evidence that stopping or canceling
	// this exact recipient retired at least one unresolved V13 envelope. A
	// later execution retry must reject instead of readdressing that evidence.
	RecipientMailboxRetired bool
	Purpose                 ExecutionPurpose
	ReviewSubjectDigest     string
	CouncilSubjectDigest    CouncilSubjectDigest
}

// ReviewRecord is a fact owned by the existing Goal state transaction, not a review lifecycle.
type ReviewRecord struct {
	Ref                      string
	GoalRef                  goal.GoalRef
	WorkItemRef              goal.WorkItemRef
	ChangeSetRef             ports.ChangeSetRef
	SubjectDigest            string
	Role                     review.Role
	Verdict                  review.Verdict
	ReviewerExecutionRef     goal.ExecutionRef
	ReviewerExecutionAttempt uint64
	LaunchReceiptRef         string
	PrincipalRef             identity.PrincipalRef
	AgentRef                 string
	ExternalRef              string
	AssessmentArtifactRef    string
	AssessmentDigest         string
	RecordedAt               time.Time
}

func (record ReviewRecord) Assessment() (review.Assessment, error) {
	return review.NewAssessment(review.Assessment{
		SubjectDigest: record.SubjectDigest, Role: record.Role, Verdict: record.Verdict,
		ReviewerExecutionRef: record.ReviewerExecutionRef.String(), ReviewerExecutionAttempt: record.ReviewerExecutionAttempt,
		LaunchReceiptRef: record.LaunchReceiptRef, ReviewerExternalRef: record.ExternalRef,
		AssessmentArtifactRef: record.AssessmentArtifactRef, AssessmentDigest: record.AssessmentDigest,
		RecordedAt: record.RecordedAt,
	})
}

type ArtifactRecord struct {
	OccurrenceRef      string
	Kind               ArtifactKind
	Stored             ports.StoredArtifact
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	ExecutionRef       goal.ExecutionRef
	ExecutionAttempt   uint64
	PlanGeneration     goal.PlanGeneration
	WorkItemGeneration goal.Revision
	AppSpecGeneration  goal.AppSpecGeneration
	SpecHash           string
	CreatedAt          time.Time
}

type AttestationRecord struct {
	Ref                    goal.AttestationRef
	Kind                   AttestationKind
	Verdict                AttestationVerdict
	GoalRef                goal.GoalRef
	WorkItemRef            goal.WorkItemRef
	ExecutionRef           goal.ExecutionRef
	ExecutionAttempt       uint64
	PlanGeneration         goal.PlanGeneration
	WorkItemGeneration     goal.Revision
	AppSpecGeneration      goal.AppSpecGeneration
	SpecHash               string
	ArtifactRef            goal.ArtifactRef
	SubjectDigest          string
	WorkspaceBindingDigest string
	ChangeSetRef           ports.ChangeSetRef
	ChangeSetDigest        string
	ManifestArtifactRef    goal.ArtifactRef
	ReportArtifactRef      goal.ArtifactRef
	AttestorRef            string
	ReceiptRef             string
	Tests                  []ports.RequiredTestOutcome
	PolicyRef              string
	RequiredTestsDigest    string
	PolicyDigest           string
	EffectIntentRef        string
	EffectAttemptRef       string
	EffectFence            uint64
	EffectReceiptRef       string
	StartedAt              time.Time
	FinishedAt             time.Time

	// Policy and AcceptedAt keep the V16 read model compatible while adapters
	// migrate to the typed fields above. They carry no independent authority.
	Policy     string
	AcceptedAt time.Time
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
	ActionLaunchAgent      ActionKind = "launch_agent"
	ActionObserveAgent     ActionKind = "observe_agent"
	ActionStopAgent        ActionKind = "stop_agent"
	ActionDeliverMailbox   ActionKind = "deliver_mailbox"
	ActionAdmitMailbox     ActionKind = "admit_mailbox"
	ActionPrepareWorkspace ActionKind = "prepare_workspace"
	ActionCommitChange     ActionKind = "commit_change"
	ActionAttestTest       ActionKind = "attest_test"
	ActionIntegrateChange  ActionKind = "integrate_change"
)

type ActionRecord struct {
	Ref                string
	Kind               ActionKind
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	ExecutionRef       goal.ExecutionRef
	ChangeRef          ports.ChangeSetRef
	ControlRef         string
	ExpectedTargetOID  string
	ReviewGateDigest   string
	CouncilResolution  *CouncilResolution
	EffectIntentRef    string
	EffectIntent       EffectIntent
	EffectApproval     *EffectApproval
	PlanGeneration     goal.PlanGeneration
	WorkItemGeneration goal.Revision
	AvailableAt        time.Time
}

type ActionClaim struct {
	Action               ActionRecord
	Token                string
	WorkerRef            string
	DeliveryAttempt      uint64
	Fence                uint64
	BudgetReservationRef string
	BudgetReservation    governance.BudgetReservation
	EffectApproval       EffectApproval
	LeaseUntil           time.Time
}

type ClaimRequest struct {
	WorkerRef               string
	Token                   string
	LeaseDuration           time.Duration
	AttestTestLeaseDuration time.Duration
	Capabilities            ports.AgentCapabilities
	BudgetPolicy            BudgetPolicy
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
	ChangeRef          ports.ChangeSetRef
	MailboxMessageRef  MailboxMessageRef
	PlanGeneration     goal.PlanGeneration
	WorkItemGeneration goal.Revision
	Fence              uint64
	DeliveryAttempt    uint64
	ClaimToken         string
	WorkerRef          string
	Outcome            ActionConsumptionOutcome
	ErrorCode          string
	// EffectReceiptRef is the only link to external effect evidence. Status and
	// timestamps live exclusively in the immutable EffectReceipt fact.
	EffectReceiptRef string
	ConsumedAt       time.Time
}

// WorkItemAuthority preserves the exact create/direct decision that admitted
// a WorkItem. It lets later readiness scheduling create effects without
// inventing authority or consulting a second control loop.
type WorkItemAuthority struct {
	WorkItemRef          goal.WorkItemRef
	PrincipalRef         identity.PrincipalRef
	Permission           identity.Permission
	Source               EffectApprovalSource
	AuthorizationReceipt identity.AuthorizationReceipt
	RecordedAt           time.Time
}

type GoalRecord struct {
	RequestRef          string
	RequestFingerprint  string
	RequestedBy         identity.PrincipalRef
	Goal                goal.Goal
	Executions          []ExecutionRecord
	Artifacts           []ArtifactRecord
	Attestations        []AttestationRecord
	Controls            []ControlRecord
	BudgetEnvelopes     []governance.BudgetEnvelope
	BudgetReservations  []governance.BudgetReservation
	BudgetSettlements   []governance.BudgetSettlement
	WorkItemAuthorities []WorkItemAuthority
	EffectIntents       []EffectIntent
	EffectApprovals     []EffectApproval
	EffectAttempts      []EffectAttempt
	EffectReceipts      []EffectReceipt
	WorkspaceBindings   []WorkspaceBinding
	ChangeSets          []ChangeSet
	MergeObservations   []MergeObservation
	IntegrationReceipts []IntegrationReceipt
	Reviews             []ReviewRecord
	CouncilRounds       []CouncilRoundRecord
	CouncilFacts        []council.ContributionFact
	CouncilDecisions    []CouncilDecisionRecord
	CouncilSkips        []CouncilSkipRecord
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
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	RequestedBy          identity.PrincipalRef
	Goal                 goal.Goal
	Executions           []ExecutionRecord
	Actions              []ActionRecord
	Events               []EventRecord
	WorkItemAuthorities  []WorkItemAuthority
	BudgetEnvelopes      []governance.BudgetEnvelope
}

// AmendGoalState carries a fully constructed successor plus the source fence
// that the repository must revalidate atomically. Generated refs are not part
// of idempotency: an equal replay returns the already persisted successor.
type AmendGoalState struct {
	RequestRef             string
	RequestFingerprint     string
	AuthorizationReceipt   identity.AuthorizationReceipt
	RequestedBy            identity.PrincipalRef
	ProjectRef             goal.ProjectRef
	SourceGoalRef          goal.GoalRef
	ExpectedSourceRevision goal.Revision
	ExpectedSourceSpecHash string
	Successor              goal.Goal
	Events                 []EventRecord
}

// LaunchPreparedState is the only durable transition from queued to
// dispatching before an external provider effect. For an initial attempt it
// also starts the pending WorkItem; retry/replacement attempts keep their
// already-running WorkItem and only cross the execution frontier. The launch
// claim deliberately remains open.
type LaunchPreparedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	Event                EventRecord
	OperationAt          time.Time
}

type LaunchAcceptedState struct {
	Claim         ActionClaim
	Execution     ExecutionRecord
	NextAction    ActionRecord
	Event         EventRecord
	EffectReceipt EffectReceipt
	OperationAt   time.Time
}

type ActionRequeuedState struct {
	Claim              ActionClaim
	Execution          ExecutionRecord
	AvailableAt        time.Time
	ErrorCode          string
	OperationAt        time.Time
	BudgetSettlement   *governance.BudgetSettlement
	ClearEffectBinding bool
}

type ActionQuarantinedState struct {
	Claim              ActionClaim
	ErrorCode          string
	Event              EventRecord
	BudgetSettlement   *governance.BudgetSettlement
	ClearEffectBinding bool
	OperationAt        time.Time
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
	BudgetSettlement     *governance.BudgetSettlement
	OperationAt          time.Time
}

// ExecutionInterruptedState consumes an exhausted provider attempt without
// closing the Goal. The Director may causally replan the interrupted WorkItem.
type ExecutionInterruptedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	Execution            ExecutionRecord
	NewExecutions        []ExecutionRecord
	NewActions           []ActionRecord
	Events               []EventRecord
	BudgetSettlement     *governance.BudgetSettlement
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
	PostArtifactAction   *ActionRecord
	Events               []EventRecord
	BudgetSettlement     *governance.BudgetSettlement
	OperationAt          time.Time
}

type PostArtifactMailboxAdmittedState struct {
	Claim        ActionClaim
	MessageRef   MailboxMessageRef
	AdmissionRef string
	OperationAt  time.Time
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
	BudgetSettlement     *governance.BudgetSettlement
	OperationAt          time.Time
}

// ReviewAssessedState consumes one reviewer observation and publishes its CAS
// occurrence and decision fact atomically. Goal/AuthorExecution change only
// when a complete round derives changes_requested.
type ReviewAssessedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Goal                 goal.Goal
	ReviewerExecution    ExecutionRecord
	AuthorExecution      ExecutionRecord
	Artifact             ArtifactRecord
	Review               ReviewRecord
	BudgetSettlement     *governance.BudgetSettlement
	Events               []EventRecord
	AutoOpenCouncil      *OpenCouncilRoundState
	OperationAt          time.Time
}

// ReviewExecutionReplacedState retries a reviewer without rebinding the
// WorkItem's authoritative author execution.
type ReviewExecutionReplacedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	FailedExecution      ExecutionRecord
	ReplacementExecution ExecutionRecord
	NextAction           ActionRecord
	DiagnosticArtifact   *ArtifactRecord
	BudgetSettlement     *governance.BudgetSettlement
	Events               []EventRecord
	OperationAt          time.Time
}

// ReviewParticipantRetirement terminalizes another live participant of the
// exact same immutable review round without rebinding WorkItem authority.
type ReviewParticipantRetirement struct {
	Execution     ExecutionRecord
	ExpectedState ExecutionState
}

type ReviewExecutionFailedState struct {
	Claim                ActionClaim
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Execution            ExecutionRecord
	Goal                 goal.Goal
	AuthorExecution      ExecutionRecord
	RetiredReviewers     []ReviewParticipantRetirement
	RetireActionRefs     []string
	CleanupControls      []ControlRecord
	CleanupActions       []ActionRecord
	ResolvedCleanup      *ControlRecord
	DiagnosticArtifact   *ArtifactRecord
	EffectReceipt        *EffectReceipt
	QuarantineClaim      bool
	BudgetSettlement     *governance.BudgetSettlement
	Events               []EventRecord
	OperationAt          time.Time
}

// StateRepository is the durable state port. Every mutation is an atomic
// application-level operation; adapters never choose lifecycle transitions.
// Mutations carrying ActionClaim must also fence the lease with the adapter's
// trusted transaction clock immediately before commit. OperationAt is causal
// lifecycle data and must never be accepted as proof that a lease is still live.
type StateRepository interface {
	GovernanceRepository
	CreateGoal(context.Context, CreateGoalState) (GoalRecord, bool, error)
	AmendGoal(context.Context, AmendGoalState) (GoalRecord, bool, error)
	GetGoal(context.Context, goal.GoalRef) (GoalRecord, error)
	ListGoals(context.Context, goal.ProjectRef, int) ([]GoalSummary, error)
	Status(context.Context, goal.ProjectRef) (RepositoryStatus, error)
	ProjectRepository(context.Context, goal.ProjectRef) (identity.RepositoryRef, error)
	ListPendingChanges(context.Context, PendingChangeQuery) ([]PendingChange, error)
	DirectorReplay(context.Context, DirectorReplayRequest) (DirectorReplayRecord, bool, error)
	ClaimDirector(context.Context, ClaimDirectorState) (DirectorLeaseRecord, bool, error)
	RenewDirector(context.Context, RenewDirectorState) (DirectorLeaseRecord, bool, error)
	ApplyDirectorPlan(context.Context, ApplyDirectorPlanState) (DirectorDecisionRecord, bool, error)
	ControlReplay(context.Context, ControlReplayRequest) (ControlRecord, bool, error)
	ApplyControl(context.Context, ApplyControlState) (ControlRecord, bool, error)
	MailboxReplay(context.Context, MailboxReplayRequest) (MailboxReplayRecord, bool, error)
	AdmitMailbox(context.Context, AdmitMailboxState) (MailboxRecord, bool, error)
	ClaimMailbox(context.Context, ClaimMailboxState) (MailboxClaim, bool, error)
	MarkMailboxDelivered(context.Context, MarkMailboxDeliveredState) (MailboxRecord, bool, error)
	ConsumeMailbox(context.Context, ConsumeMailboxState) (MailboxRecord, bool, error)
	AcknowledgeMailbox(context.Context, ResolveMailboxState) (MailboxAcknowledgement, bool, error)
	BlockMailbox(context.Context, ResolveMailboxState) (MailboxAcknowledgement, bool, error)
	GetMailbox(context.Context, goal.ProjectRef, goal.GoalRef, MailboxMessageRef, MailboxEndpoint) (MailboxRecord, error)
	ListMailbox(context.Context, goal.ProjectRef, goal.GoalRef, MailboxEndpoint, int) ([]MailboxRecord, error)
	ClaimNextAction(context.Context, ClaimRequest) (ActionClaim, bool, error)
	RecordLaunchPrepared(context.Context, LaunchPreparedState) error
	RecordLaunchAccepted(context.Context, LaunchAcceptedState) error
	RequeueAction(context.Context, ActionRequeuedState) error
	QuarantineAction(context.Context, ActionQuarantinedState) error
	RecordExecutionReplaced(context.Context, ExecutionReplacedState) error
	RecordExecutionInterrupted(context.Context, ExecutionInterruptedState) error
	RecordGoalSucceeded(context.Context, GoalSucceededState) error
	RecordPostArtifactMailboxAdmitted(context.Context, PostArtifactMailboxAdmittedState) error
	RecordGoalFailed(context.Context, GoalFailedState) error
	RecordReviewAssessed(context.Context, ReviewAssessedState) error
	OpenCouncilRound(context.Context, OpenCouncilRoundState) (CouncilRoundRecord, bool, error)
	RecordCouncilContribution(context.Context, CouncilContributionState) error
	RecordCouncilSkip(context.Context, CouncilSkipState) (CouncilSkipRecord, bool, error)
	RecordCouncilExecutionReplaced(context.Context, CouncilExecutionReplacedState) error
	RecordCouncilExecutionFailed(context.Context, CouncilExecutionFailedState) error
	RecordReviewExecutionReplaced(context.Context, ReviewExecutionReplacedState) error
	RecordReviewExecutionFailed(context.Context, ReviewExecutionFailedState) error
	RecordWorkspacePrepared(context.Context, WorkspacePreparedState) error
	RecordExecutionOutputReady(context.Context, ExecutionOutputReadyState) error
	RecordChangeCommitted(context.Context, ChangeCommittedState) error
	RecordTestAttested(context.Context, TestAttestedState) error
	AdmitIntegration(context.Context, AdmitIntegrationState) (ActionRecord, bool, error)
	RecordIntegrationResult(context.Context, IntegrationResultState) error
}
