package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const agentEnvironmentLifecycleSnapshotSchema = "orquesta.agent-environment-lifecycle.snapshot.v1"

var errAgentEnvironmentLifecycleStateInvalid = errors.New("application.agent_environment_lifecycle_state_invalid")

// AgentEnvironmentLifecycleEffectSnapshot binds the current physical frontier
// to the canonical effect ledger. No outcome means the call may have happened
// or returned pending and must be reconciled; only a terminal outcome is
// persistible. An unresolved effect never permits a new EffectAttempt.
type AgentEnvironmentLifecycleEffectSnapshot struct {
	ActionRef          string
	ActionKind         ActionKind
	AttemptRef         string
	ActionFence        uint64
	DeliveryAttempt    uint64
	WorkItemGeneration goal.Revision
	ClaimToken         string
	WorkerRef          string
	IdempotencyKey     string
	ExpectedToken      ports.AgentEnvironmentLifecycleToken
	OutcomeToken       ports.AgentEnvironmentLifecycleToken
	PhysicalReceipt    string
	EffectReceiptRef   string
}

func (effect AgentEnvironmentLifecycleEffectSnapshot) IsEmpty() bool {
	return effect == (AgentEnvironmentLifecycleEffectSnapshot{})
}

func (effect AgentEnvironmentLifecycleEffectSnapshot) NeedsReconciliation() bool {
	if effect.IsEmpty() || effect.EffectReceiptRef != "" {
		return false
	}
	return effect.OutcomeToken == (ports.AgentEnvironmentLifecycleToken{})
}

// AgentEnvironmentLifecycleSnapshot is a state-centric projection for one
// Execution. Physical state remains owned by Token; lifecycle authority stays
// in application and the existing effect/preservation facts. The durable CAS
// snapshot is a trusted authority boundary: Replay detects causal or ledger
// misalignment and rejects independently invented tokens, but does not provide
// cryptographic tamper evidence against coordinated mutation of that same
// trusted snapshot source.
type AgentEnvironmentLifecycleSnapshot struct {
	Schema                 string
	Revision               uint64
	LaunchReceiptRef       string
	Subject                ports.AgentEnvironmentLifecycleSubject
	Token                  ports.AgentEnvironmentLifecycleToken
	Effect                 AgentEnvironmentLifecycleEffectSnapshot
	Preservation           ports.AgentPreservationBinding
	PreservationReceiptRef string
	RecordedAt             time.Time
}

// AgentEnvironmentLifecycleInitialState is the only admissible durable birth of
// the projection. The writer inserts Snapshot iff no lifecycle row exists for
// its exact Execution; an exact replay returns the stored snapshot without
// inventing another physical inspection fact.
type AgentEnvironmentLifecycleInitialState struct {
	Snapshot    AgentEnvironmentLifecycleSnapshot
	OperationAt time.Time
}

// AgentEnvironmentLifecycleStoredState is the restart read model. Claim and
// Attempt are absent only at the initial active frontier. Once an effect was
// attempted they remain the original authority even after its claim lease
// expires; a later reader must never replace them with a new claim.
type AgentEnvironmentLifecycleStoredState struct {
	Snapshot     AgentEnvironmentLifecycleSnapshot
	Claim        ActionClaim
	Attempt      EffectAttempt
	HasAttempt   bool
	Preservation *ComprobantePreservacionEntornoAgente
	NextAction   *ActionRecord
	// ReadyToFinalize is durable response state, not a second authority. It is
	// true exactly when Snapshot is already closed.
	ReadyToFinalize bool
}

// AgentEnvironmentLifecyclePreEffectState carries the data an atomic CAS uses
// to append Attempt and replace Snapshot with its ambiguous frontier.
type AgentEnvironmentLifecyclePreEffectState struct {
	Claim                         ActionClaim
	ExpectedRevision              uint64
	RequiredApplicationReceiptRef string
	Attempt                       EffectAttempt
	Snapshot                      AgentEnvironmentLifecycleSnapshot
	OperationAt                   time.Time
}

// AgentEnvironmentLifecyclePostEffectState carries the only authority for a
// terminal-resolution CAS. Claim is the original claim captured by Attempt;
// its lease is checked at Attempt.StartedAt and MUST NOT be required to remain
// live at OperationAt. This state never authorizes another physical call.
// EffectReceipt exists only for a terminal physical result. PreservationFact
// exists only after causal validation of terminal preserve.
type AgentEnvironmentLifecyclePostEffectState struct {
	Claim              ActionClaim
	ExpectedRevision   uint64
	Attempt            EffectAttempt
	Snapshot           AgentEnvironmentLifecycleSnapshot
	EffectReceipt      *EffectReceipt
	ConsumptionReceipt ActionConsumptionReceipt
	PreservationFact   *ComprobantePreservacionEntornoAgente
	// NextAction is application-owned continuation. Quiesce terminal requires
	// Preserve; Preserve terminal requires Close. It is stored atomically with
	// this terminal CAS. Close has no next lifecycle action.
	NextAction *ActionRecord
	// ReadyToFinalize is true only for the durable closed physical frontier.
	// It is a convenience projection; Snapshot.Token remains authority.
	ReadyToFinalize bool
	OperationAt     time.Time
}

// AgentEnvironmentLifecycleStore is one cohesive durable application port.
// Initial, attempted, and terminal writes are separate CAS boundaries because
// the physical call occurs strictly between attempted and terminal. No method
// invokes the physical provider. RecordAttempt atomically appends the exact
// EffectAttempt and attempted Snapshot. RecordTerminal additionally stores the
// application-selected NextAction, when present. A false write result means a
// CAS winner or exact replay exists and the caller must read that durable state;
// it never authorizes executing the caller's candidate effect.
type AgentEnvironmentLifecycleStore interface {
	AgentEnvironmentLifecycleTerminalWriter
	GetAgentEnvironmentLifecycle(
		context.Context,
		goal.ExecutionRef,
	) (AgentEnvironmentLifecycleStoredState, bool, error)
	RecordAgentEnvironmentLifecycleInitial(
		context.Context,
		AgentEnvironmentLifecycleInitialState,
	) (AgentEnvironmentLifecycleSnapshot, bool, error)
	RecordAgentEnvironmentLifecycleAttempt(
		context.Context,
		AgentEnvironmentLifecyclePreEffectState,
	) (AgentEnvironmentLifecycleSnapshot, bool, error)
}

// AgentEnvironmentLifecycleTerminalWriter is the durable, lifecycle-specific
// CAS port. An implementation compares ExpectedRevision with the attempted
// snapshot, validates the exact Attempt, receipts, and resulting snapshot, and
// atomically stores attempted -> terminal. It does not use generic recovery
// claims, re-authorize the effect, or fence the original claim lease at commit.
// A successful call therefore appends one terminal EffectReceipt, one matching
// ActionConsumptionReceipt, an optional preservation fact, the application-
// selected next action, and one snapshot revision while making zero calls to
// the physical lifecycle provider. ReadyToFinalize is accepted only for closed.
type AgentEnvironmentLifecycleTerminalWriter interface {
	RecordAgentEnvironmentLifecycleTerminal(
		context.Context,
		AgentEnvironmentLifecyclePostEffectState,
	) (AgentEnvironmentLifecycleSnapshot, bool, error)
}

// AgentEnvironmentLifecycleOutcome is a closed result. A terminal physical
// receipt yields exactly one persistible PostEffect. A pending provider reply
// yields only read-only reconciliation authority and leaves the durable
// attempted snapshot unchanged.
type AgentEnvironmentLifecycleOutcome struct {
	Terminal       *AgentEnvironmentLifecyclePostEffectState
	Reconciliation *AgentEnvironmentLifecycleReconciliation
}

// AgentEnvironmentLifecycleReconciliation is read-only authority. It points
// Inspect at the same subject and carries exactly one historical-receipt
// request bound to Effect.ExpectedToken. It never authorizes Quiesce,
// Preserve, Close, or a replacement EffectAttempt.
type AgentEnvironmentLifecycleReconciliation struct {
	AttemptRef     string
	ActionRef      string
	ActionKind     ActionKind
	ActionFence    uint64
	Token          ports.AgentEnvironmentLifecycleToken
	InspectRequest ports.AgentEnvironmentInspectRequest
	ReceiptRequest AgentEnvironmentLifecycleReceiptRequest
}

// AgentEnvironmentLifecycleReceiptRequest is a closed sum: a valid
// reconciliation contains exactly one non-nil request matching ActionKind.
type AgentEnvironmentLifecycleReceiptRequest struct {
	Quiesce  *ports.AgentQuiesceRequest
	Preserve *ports.AgentPreserveRequest
	Close    *ports.AgentCloseRequest
}

func NewAgentEnvironmentLifecycleSnapshot(
	launchRequest ports.AgentLaunchRequest,
	launchReceipt ports.AgentLaunchReceipt,
	inspectReceipt ports.AgentEnvironmentInspectReceipt,
	recordedAt time.Time,
) (AgentEnvironmentLifecycleSnapshot, error) {
	subject := inspectReceipt.Subject
	if ports.ValidateAgentLaunchReceipt(launchRequest, launchReceipt) != nil ||
		ports.ValidateAgentEnvironmentLifecycleTarget(launchReceipt, subject) != nil ||
		ports.ValidateAgentEnvironmentInspectReceipt(
			ports.AgentEnvironmentInspectRequest{Subject: subject}, inspectReceipt,
		) != nil || inspectReceipt.Token.State != ports.AgentEnvironmentActive ||
		recordedAt.IsZero() || recordedAt.Before(launchReceipt.AcceptedAt) {
		return AgentEnvironmentLifecycleSnapshot{}, errAgentEnvironmentLifecycleStateInvalid
	}
	snapshot := AgentEnvironmentLifecycleSnapshot{
		Schema: agentEnvironmentLifecycleSnapshotSchema, Revision: 1,
		LaunchReceiptRef: launchReceipt.ReceiptRef, Subject: subject,
		Token: inspectReceipt.Token, RecordedAt: recordedAt.UTC(),
	}
	return snapshot, nil
}

func ReplayAgentEnvironmentLifecycleSnapshot(
	snapshot AgentEnvironmentLifecycleSnapshot,
	launchRequest ports.AgentLaunchRequest,
	launchReceipt ports.AgentLaunchReceipt,
	record GoalRecord,
	preservation *ComprobantePreservacionEntornoAgente,
) (AgentEnvironmentLifecycleSnapshot, error) {
	if validateAgentEnvironmentLifecycleSnapshot(snapshot) != nil ||
		ports.ValidateAgentLaunchReceipt(launchRequest, launchReceipt) != nil ||
		ports.ValidateAgentEnvironmentLifecycleTarget(launchReceipt, snapshot.Subject) != nil ||
		snapshot.LaunchReceiptRef != launchReceipt.ReceiptRef ||
		validateAgentEnvironmentLifecycleRecord(snapshot, launchReceipt, record, preservation) != nil {
		return AgentEnvironmentLifecycleSnapshot{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return snapshot, nil
}

func PrepareAgentEnvironmentLifecycleEffect(
	current AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	operationAt time.Time,
) (AgentEnvironmentLifecyclePreEffectState, error) {
	if validateAgentEnvironmentLifecycleSnapshot(current) != nil || current.Effect.NeedsReconciliation() ||
		operationAt.Before(current.RecordedAt) {
		return AgentEnvironmentLifecyclePreEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	wantKind, ok := agentEnvironmentLifecycleAction(current.Token.State)
	if !ok || validateAgentEnvironmentLifecycleClaim(current, claim, wantKind, operationAt) != nil {
		return AgentEnvironmentLifecyclePreEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	intent := claim.Action.EffectIntent
	attempt := EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: intent.Ref, IntentDigest: intent.Digest, ApprovalRef: claim.EffectApproval.Ref,
		Subject: intent.Subject, ActionRef: claim.Action.Ref, ActionFence: claim.Fence,
		WorkerRef: claim.WorkerRef, IdempotencyKey: intent.IdempotencyKey,
		StartedAt: operationAt.UTC(), ClaimLeaseUntil: claim.LeaseUntil.UTC(),
	}
	if validateEffectAttempt(claim, attempt) != nil {
		return AgentEnvironmentLifecyclePreEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	next := current
	next.Revision++
	next.Effect = AgentEnvironmentLifecycleEffectSnapshot{
		ActionRef: claim.Action.Ref, ActionKind: claim.Action.Kind, AttemptRef: attempt.Ref,
		ActionFence: attempt.ActionFence, DeliveryAttempt: claim.DeliveryAttempt,
		WorkItemGeneration: claim.Action.WorkItemGeneration,
		ClaimToken:         claim.Token, WorkerRef: claim.WorkerRef, IdempotencyKey: attempt.IdempotencyKey,
		ExpectedToken: current.Token,
	}
	next.RecordedAt = operationAt.UTC()
	state := AgentEnvironmentLifecyclePreEffectState{
		Claim: claim, ExpectedRevision: current.Revision, Attempt: attempt,
		Snapshot: next, OperationAt: operationAt.UTC(),
	}
	if wantKind == ActionCloseAgentEnvironment {
		state.RequiredApplicationReceiptRef = current.Preservation.ApplicationReceiptRef
	}
	if validateAgentEnvironmentLifecyclePreEffect(state) != nil {
		return AgentEnvironmentLifecyclePreEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return state, nil
}

func AgentEnvironmentLifecycleReconciliationRequest(
	snapshot AgentEnvironmentLifecycleSnapshot,
) (AgentEnvironmentLifecycleReconciliation, error) {
	if validateAgentEnvironmentLifecycleSnapshot(snapshot) != nil || !snapshot.Effect.NeedsReconciliation() {
		return AgentEnvironmentLifecycleReconciliation{}, errAgentEnvironmentLifecycleStateInvalid
	}
	reconciliation := AgentEnvironmentLifecycleReconciliation{
		AttemptRef: snapshot.Effect.AttemptRef, ActionRef: snapshot.Effect.ActionRef,
		ActionKind: snapshot.Effect.ActionKind, ActionFence: snapshot.Effect.ActionFence,
		Token: snapshot.Token, InspectRequest: ports.AgentEnvironmentInspectRequest{Subject: snapshot.Subject},
	}
	switch snapshot.Effect.ActionKind {
	case ActionQuiesceAgent:
		request := ports.AgentQuiesceRequest{
			Subject: snapshot.Subject, ExpectedToken: snapshot.Effect.ExpectedToken,
			IdempotencyKey: snapshot.Effect.IdempotencyKey,
		}
		if ports.ValidateAgentQuiesceRequest(request) != nil {
			return AgentEnvironmentLifecycleReconciliation{}, errAgentEnvironmentLifecycleStateInvalid
		}
		reconciliation.ReceiptRequest.Quiesce = &request
	case ActionPreserveAgentEnvironment:
		request := ports.AgentPreserveRequest{
			Subject: snapshot.Subject, ExpectedToken: snapshot.Effect.ExpectedToken,
			IdempotencyKey: snapshot.Effect.IdempotencyKey,
		}
		if ports.ValidateAgentPreserveRequest(request) != nil {
			return AgentEnvironmentLifecycleReconciliation{}, errAgentEnvironmentLifecycleStateInvalid
		}
		reconciliation.ReceiptRequest.Preserve = &request
	case ActionCloseAgentEnvironment:
		request := ports.AgentCloseRequest{
			Subject: snapshot.Subject, ExpectedToken: snapshot.Effect.ExpectedToken,
			Preservation: snapshot.Preservation, IdempotencyKey: snapshot.Effect.IdempotencyKey,
		}
		if ports.ValidateAgentCloseRequest(request) != nil {
			return AgentEnvironmentLifecycleReconciliation{}, errAgentEnvironmentLifecycleStateInvalid
		}
		reconciliation.ReceiptRequest.Close = &request
	default:
		return AgentEnvironmentLifecycleReconciliation{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return reconciliation, nil
}
