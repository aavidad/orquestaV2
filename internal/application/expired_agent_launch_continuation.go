package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/ports"
)

var ErrExpiredAgentLaunchContinuationUnsupported = errors.New("application.expired_agent_launch_continuation_unsupported")

const ExpiredAgentLaunchContinuationConfirmationV41 = "emitir_autoridad_v41"

// ExpiredAgentLaunchContinuationCausalBindingV41 is the complete immutable
// Orquesta identity carried into the sibling protocol. It points at the first
// recorded reconciliation attempt and the sole pre-existing physical effect
// attempt; issuing V41 never creates either one.
type ExpiredAgentLaunchContinuationCausalBindingV41 struct {
	ReconciliationAuthorityRef string
	ReconciliationAttemptRef   string
	ProjectRef                 string
	GoalRef                    string
	WorkItemRef                string
	ExecutionRef               string
	ActionRef                  string
	EffectIntentRef            string
	EffectIntentDigest         string
	EffectAttemptRef           string
	PlanGeneration             uint64
	WorkItemGeneration         uint64
	ActionFence                uint64
}

// ExpiredAgentLaunchContinuationSourceV41 is a read-only snapshot of the V40
// authority and its first requeued attempt. Claim is reconstructed exclusively
// from durable history so application can reuse the existing recovery checks.
type ExpiredAgentLaunchContinuationSourceV41 struct {
	Authority             TerminalAgentLaunchReconciliationAuthority
	ReconciliationAttempt TerminalAgentLaunchReconciliationAttempt
	Claim                 ActionClaim
}

type ExpiredAgentLaunchContinuationPreparationV41 struct {
	Binding               ExpiredAgentLaunchContinuationCausalBindingV41
	ManifestSHA256        string
	RequestKeySHA256      string
	OriginalRequestSHA256 string
	AMVLaunchRef          string
	AMVExecutionRef       string
	AMVRunRef             string
	AMVFence              uint64
	AMVGeneration         uint64
	AMVCID                uint32
	AMVIdentitySHA256     string
	PreparedAt            time.Time
}

type PreflightExpiredAgentLaunchContinuationRequestV41 struct {
	RequestRef                 string
	ReconciliationAuthorityRef string
}

type PreflightExpiredAgentLaunchContinuationResultV41 struct {
	Preparation ExpiredAgentLaunchContinuationPreparationV41
}

type ConfirmExpiredAgentLaunchContinuationRequestV41 struct {
	RequestRef                 string
	ReconciliationAuthorityRef string
	ExpectedManifestSHA256     string
	Confirmation               string
}

type ConfirmExpiredAgentLaunchContinuationResultV41 struct {
	Record  ExpiredAgentLaunchContinuationRecordV41
	Created bool
}

type ExpiredAgentLaunchContinuationIssuanceV41 struct {
	ActorRef               string
	ProjectRef             string
	CredentialRequestRef   string
	SubjectRef             string
	AuthorityRef           string
	ExpectedManifestSHA256 string
	IssuedAt               time.Time
}

// ExpiredAgentLaunchContinuationRecordV41 is the opaque durable bridge between
// an explicitly issued one-use authority and the sole historical EffectAttempt.
// Application owns causal refs; the adapter owns the protocol bytes.
type ExpiredAgentLaunchContinuationRecordV41 struct {
	SubjectRef                 string
	ReconciliationAuthorityRef string
	ReconciliationAttemptRef   string
	ProjectRef                 string
	GoalRef                    string
	WorkItemRef                string
	ExecutionRef               string
	ActionRef                  string
	EffectIntentRef            string
	EffectIntentDigest         string
	EffectAttemptRef           string
	PlanGeneration             uint64
	WorkItemGeneration         uint64
	ActionFence                uint64
	RequestKeySHA256           string
	OriginalRequestSHA256      string
	AMVLaunchRef               string
	AMVExecutionRef            string
	AMVRunRef                  string
	AMVFence                   uint64
	AMVGeneration              uint64
	AMVCID                     uint32
	AMVIdentitySHA256          string
	SourceDigest               string
	ProfileDescriptorBytes     []byte
	ProfileDescriptorSHA256    string
	PlanBytes                  []byte
	PlanSHA256                 string
	ConcessionBytes            []byte
	ConcessionSHA256           string
	ManifestBytes              []byte
	ManifestBytesSHA256        string
	ManifestSHA256             string
	PreparedAt                 time.Time
	AuthorityRef               string
	AuthoritySHA256            string
	AuthorityBytes             []byte
	AuthorityBytesSHA256       string
	KeyID                      string
	KeyEpoch                   uint64
	TrustRevision              uint64
	PublicKey                  []byte
	Signature                  []byte
	IssuedUnixMS               uint64
	ExpiresUnixMS              uint64
	AdmittedUnixMS             uint64
}

// ExpiredAgentLaunchContinuationStoreV41 accepts subject and authority in one
// transaction. A partial durable authority is never a recoverable state.
type ExpiredAgentLaunchContinuationStoreV41 interface {
	RecordExpiredAgentLaunchContinuationV41(
		context.Context,
		ExpiredAgentLaunchContinuationRecordV41,
	) (ExpiredAgentLaunchContinuationRecordV41, bool, error)
	ExpiredAgentLaunchContinuationV41(
		context.Context,
		string,
	) (ExpiredAgentLaunchContinuationRecordV41, bool, error)
}

// ExpiredAgentLaunchContinuationSourceStoreV41 reads the exact V40 source. It
// never claims a job or advances the scheduler.
type ExpiredAgentLaunchContinuationSourceStoreV41 interface {
	ExpiredAgentLaunchContinuationSourceV41(
		context.Context,
		string,
	) (ExpiredAgentLaunchContinuationSourceV41, bool, error)
}

// ExpiredAgentLaunchContinuationWriterV41 is deliberately preparation and
// issuance only. It has no Continue, Launch or reconciliation operation.
type ExpiredAgentLaunchContinuationWriterV41 interface {
	PrepareExpiredAgentLaunchContinuationV41(
		context.Context,
		ports.AgentLaunchRequest,
		ExpiredAgentLaunchContinuationCausalBindingV41,
	) (ExpiredAgentLaunchContinuationPreparationV41, error)
	IssueExpiredAgentLaunchContinuationV41(
		context.Context,
		ports.AgentLaunchRequest,
		ExpiredAgentLaunchContinuationCausalBindingV41,
		ExpiredAgentLaunchContinuationIssuanceV41,
	) (ExpiredAgentLaunchContinuationRecordV41, error)
}

// ExpiredAgentLaunchContinuerV41 is the only terminal recovery effect port.
// It receives already admitted bytes and cannot synthesize an ordinary Launch.
type ExpiredAgentLaunchContinuerV41 interface {
	ContinueExpiredAgentLaunchV41(
		context.Context,
		ports.AgentLaunchRequest,
		ExpiredAgentLaunchContinuationRecordV41,
	) (ports.AgentLaunchReceipt, error)
}

func ExpiredAgentLaunchContinuerV41From(
	launcher AgentLauncher,
) (ExpiredAgentLaunchContinuerV41, error) {
	continuer, ok := launcher.(ExpiredAgentLaunchContinuerV41)
	if launcher == nil || !ok {
		return nil, ErrExpiredAgentLaunchContinuationUnsupported
	}
	return continuer, nil
}

func (orchestrator *Orchestrator) PreflightExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	access Access,
	request PreflightExpiredAgentLaunchContinuationRequestV41,
) (PreflightExpiredAgentLaunchContinuationResultV41, error) {
	if orchestrator == nil || orchestrator.expiredLaunchContinuationWriter == nil ||
		orchestrator.expiredLaunchContinuationSource == nil {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, ErrExpiredAgentLaunchContinuationUnsupported
	}
	if !validApplicationRef(request.RequestRef) || !validApplicationRef(request.ReconciliationAuthorityRef) {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, ErrAgentLaunchRecoveryInvalid
	}
	_, projectRef, err := access.values()
	if err != nil {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, err
	}
	if _, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, "effects.approve",
		"agent-launch-expired-continuation-preflight:"+request.ReconciliationAuthorityRef,
		orchestrator.clock.Now().UTC(),
		"authorization-request:agent-launch-expired-continuation-preflight:"+request.RequestRef,
	); err != nil {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, err
	}
	launch, binding, err := orchestrator.buildExpiredAgentLaunchContinuationRequestV41(
		ctx, projectRef.String(), request.ReconciliationAuthorityRef,
	)
	if err != nil {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, err
	}
	preparation, err := orchestrator.expiredLaunchContinuationWriter.PrepareExpiredAgentLaunchContinuationV41(
		ctx, launch, binding,
	)
	if err != nil {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, err
	}
	preparation.PreparedAt = orchestrator.clock.Now().UTC()
	if validateExpiredAgentLaunchContinuationPreparationV41(preparation, binding) != nil {
		return PreflightExpiredAgentLaunchContinuationResultV41{}, ErrAgentLaunchRecoveryInvalid
	}
	return PreflightExpiredAgentLaunchContinuationResultV41{Preparation: preparation}, nil
}

func (orchestrator *Orchestrator) ConfirmExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	access Access,
	request ConfirmExpiredAgentLaunchContinuationRequestV41,
) (ConfirmExpiredAgentLaunchContinuationResultV41, error) {
	if orchestrator == nil || orchestrator.expiredLaunchContinuationWriter == nil ||
		orchestrator.expiredLaunchContinuationSource == nil || orchestrator.expiredLaunchContinuationStore == nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, ErrExpiredAgentLaunchContinuationUnsupported
	}
	if !validApplicationRef(request.RequestRef) || !validApplicationRef(request.ReconciliationAuthorityRef) ||
		!validExpiredAgentLaunchContinuationDigest(request.ExpectedManifestSHA256) ||
		request.Confirmation != ExpiredAgentLaunchContinuationConfirmationV41 {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, ErrAgentLaunchRecoveryInvalid
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, err
	}
	now := orchestrator.clock.Now().UTC()
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, "effects.approve",
		"agent-launch-expired-continuation-confirm:"+request.ReconciliationAuthorityRef+":"+request.ExpectedManifestSHA256,
		now,
		"authorization-request:agent-launch-expired-continuation-confirm:"+request.RequestRef,
	)
	if err != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, err
	}
	if replay, found, readErr := orchestrator.expiredLaunchContinuationStore.ExpiredAgentLaunchContinuationV41(
		ctx, request.ReconciliationAuthorityRef,
	); readErr != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, readErr
	} else if found {
		if replay.ProjectRef != projectRef.String() || replay.ManifestSHA256 != request.ExpectedManifestSHA256 {
			return ConfirmExpiredAgentLaunchContinuationResultV41{}, &StateError{Code: StateConflict}
		}
		return ConfirmExpiredAgentLaunchContinuationResultV41{Record: replay}, nil
	}
	launch, binding, err := orchestrator.buildExpiredAgentLaunchContinuationRequestV41(
		ctx, projectRef.String(), request.ReconciliationAuthorityRef,
	)
	if err != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, err
	}
	issuedAt := authorizationCausalFloor(now, authorization)
	identity := fingerprintFields(
		"orquesta.agent-launch-expired-continuation-authority.v1",
		binding.ReconciliationAuthorityRef, binding.ReconciliationAttemptRef,
		binding.EffectAttemptRef, request.ExpectedManifestSHA256, authorization.Ref(),
	)
	issued, err := orchestrator.expiredLaunchContinuationWriter.IssueExpiredAgentLaunchContinuationV41(
		ctx, launch, binding, ExpiredAgentLaunchContinuationIssuanceV41{
			ActorRef: principal.ActorRef.String(), ProjectRef: projectRef.String(),
			CredentialRequestRef:   "request:agent-launch-expired-continuation-sign:" + identity,
			SubjectRef:             "expired-launch-continuation-subject:" + identity,
			AuthorityRef:           "continuation:" + identity,
			ExpectedManifestSHA256: request.ExpectedManifestSHA256, IssuedAt: issuedAt,
		},
	)
	if err != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, err
	}
	if validateExpiredAgentLaunchContinuationRecordBindingV41(issued, binding, request.ExpectedManifestSHA256) != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, ErrAgentLaunchRecoveryInvalid
	}
	persisted, created, err := orchestrator.expiredLaunchContinuationStore.RecordExpiredAgentLaunchContinuationV41(ctx, issued)
	if err != nil {
		return ConfirmExpiredAgentLaunchContinuationResultV41{}, err
	}
	return ConfirmExpiredAgentLaunchContinuationResultV41{Record: persisted, Created: created}, nil
}

func (orchestrator *Orchestrator) buildExpiredAgentLaunchContinuationRequestV41(
	ctx context.Context,
	projectRef string,
	reconciliationAuthorityRef string,
) (ports.AgentLaunchRequest, ExpiredAgentLaunchContinuationCausalBindingV41, error) {
	source, found, err := orchestrator.expiredLaunchContinuationSource.ExpiredAgentLaunchContinuationSourceV41(
		ctx, reconciliationAuthorityRef,
	)
	if err != nil || !found {
		if err != nil {
			return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, err
		}
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, &StateError{Code: StateNotFound}
	}
	authority, claim := source.Authority, source.Claim
	if authority.Ref != reconciliationAuthorityRef || authority.ProjectRef.String() != projectRef ||
		claim.TerminalReconciliationRef != authority.Ref || claim.RecoveryEffectAttemptRef != authority.EffectAttemptRef ||
		source.ReconciliationAttempt.AuthorityRef != authority.Ref ||
		source.ReconciliationAttempt.OriginalEffectAttemptRef != authority.EffectAttemptRef {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, &StateError{Code: StateConflict}
	}
	record, err := orchestrator.state.GetGoal(ctx, authority.GoalRef)
	if err != nil {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, err
	}
	_, err = validateTerminalAgentLaunchHistory(record, authority.ProjectRef, ReconcileTerminalAgentLaunchRequest{
		RequestRef: authority.RequestRef, GoalRef: authority.GoalRef, WorkItemRef: authority.WorkItemRef,
		ExecutionRef: authority.ExecutionRef, ActionRef: authority.ActionRef,
		EffectIntentRef: authority.EffectIntentRef, EffectIntentDigest: authority.EffectIntentDigest,
		EffectAttemptRef: authority.EffectAttemptRef, PlanGeneration: authority.PlanGeneration,
		WorkItemGeneration: authority.WorkItemGeneration, ActionFence: authority.ActionFence,
	})
	if err != nil {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, err
	}
	item, itemFound := record.Goal.WorkItem(authority.WorkItemRef)
	if !itemFound {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, ErrAgentLaunchRecoveryInvalid
	}
	execution, executionFound := executionForAction(record, claim.Action)
	phase, phaseFound := phaseForWorkItem(record.Goal, item)
	if !executionFound || !phaseFound || execution.Ref != authority.ExecutionRef {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, ErrAgentLaunchRecoveryInvalid
	}
	launch, err := orchestrator.reconstructAgentLaunchRecoveryRequest(record, item, execution, phase)
	if err != nil {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, err
	}
	launch.ReferenciaColocacion = claim.ReferenciaColocacion
	launch.RequierePreservacionEntorno = execution.RequierePreservacionEntorno
	launch.SessionRef = execution.ExecutionSessionRef
	if execution.ExecutionSessionRef.String() != "" {
		if orchestrator.expiredLaunchContinuationSessionSource == nil ||
			orchestrator.expiredLaunchContinuationSessionMethod == "" {
			return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, ErrExpiredAgentLaunchContinuationUnsupported
		}
		session, sessionErr := orchestrator.expiredLaunchContinuationSessionSource.ExecutionSessionAuthority(
			ctx, execution.Ref, orchestrator.expiredLaunchContinuationSessionMethod,
		)
		expected, deriveErr := DeriveExecutionSessionAuthority(
			ExecutionSessionRequest(record.Goal, execution), orchestrator.expiredLaunchContinuationSessionMethod,
		)
		if sessionErr != nil || deriveErr != nil || session != expected || session.SessionRef != execution.ExecutionSessionRef {
			return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, ErrAgentLaunchRecoveryInvalid
		}
		launch.AccessAuthority = ports.AgentLaunchAccessAuthority{
			ArtifactAccessRef: session.ArtifactAccessRef, MCPAccessRef: session.MCPAccessRef,
			MailboxEndpointRef: session.MailboxEndpointRef,
		}
	}
	launch, attempt, err := BuildAgentLaunchRecoveryRequest(record, claim, launch)
	if err != nil || attempt.Ref != authority.EffectAttemptRef {
		return ports.AgentLaunchRequest{}, ExpiredAgentLaunchContinuationCausalBindingV41{}, ErrAgentLaunchRecoveryInvalid
	}
	binding := ExpiredAgentLaunchContinuationCausalBindingV41{
		ReconciliationAuthorityRef: authority.Ref,
		ReconciliationAttemptRef:   source.ReconciliationAttempt.Ref,
		ProjectRef:                 authority.ProjectRef.String(), GoalRef: authority.GoalRef.String(),
		WorkItemRef: authority.WorkItemRef.String(), ExecutionRef: authority.ExecutionRef.String(),
		ActionRef: authority.ActionRef, EffectIntentRef: authority.EffectIntentRef,
		EffectIntentDigest: authority.EffectIntentDigest, EffectAttemptRef: authority.EffectAttemptRef,
		PlanGeneration: uint64(authority.PlanGeneration), WorkItemGeneration: uint64(authority.WorkItemGeneration),
		ActionFence: authority.ActionFence,
	}
	return launch, binding, nil
}

func validateExpiredAgentLaunchContinuationPreparationV41(
	value ExpiredAgentLaunchContinuationPreparationV41,
	binding ExpiredAgentLaunchContinuationCausalBindingV41,
) error {
	if value.Binding != binding || !validExpiredAgentLaunchContinuationDigest(value.ManifestSHA256) ||
		!validExpiredAgentLaunchContinuationDigest(value.RequestKeySHA256) ||
		!validExpiredAgentLaunchContinuationDigest(value.OriginalRequestSHA256) ||
		!validExpiredAgentLaunchContinuationDigest(value.AMVIdentitySHA256) ||
		value.AMVLaunchRef == "" || value.AMVExecutionRef == "" || value.AMVRunRef == "" ||
		value.AMVFence != binding.ActionFence || value.AMVGeneration != binding.PlanGeneration ||
		value.AMVCID < 3 || value.PreparedAt.IsZero() {
		return ErrAgentLaunchRecoveryInvalid
	}
	return nil
}

func validateExpiredAgentLaunchContinuationRecordBindingV41(
	record ExpiredAgentLaunchContinuationRecordV41,
	binding ExpiredAgentLaunchContinuationCausalBindingV41,
	manifestSHA256 string,
) error {
	if record.ReconciliationAuthorityRef != binding.ReconciliationAuthorityRef ||
		record.ReconciliationAttemptRef != binding.ReconciliationAttemptRef ||
		record.ProjectRef != binding.ProjectRef || record.GoalRef != binding.GoalRef ||
		record.WorkItemRef != binding.WorkItemRef || record.ExecutionRef != binding.ExecutionRef ||
		record.ActionRef != binding.ActionRef || record.EffectIntentRef != binding.EffectIntentRef ||
		record.EffectIntentDigest != binding.EffectIntentDigest || record.EffectAttemptRef != binding.EffectAttemptRef ||
		record.PlanGeneration != binding.PlanGeneration || record.WorkItemGeneration != binding.WorkItemGeneration ||
		record.ActionFence != binding.ActionFence || record.ManifestSHA256 != manifestSHA256 {
		return ErrAgentLaunchRecoveryInvalid
	}
	return nil
}

func validExpiredAgentLaunchContinuationDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func expiredAgentLaunchContinuationBindingFingerprintV41(
	binding ExpiredAgentLaunchContinuationCausalBindingV41,
) string {
	return fingerprintFields(
		"orquesta.agent-launch-expired-continuation-binding.v1",
		binding.ReconciliationAuthorityRef, binding.ReconciliationAttemptRef,
		binding.ProjectRef, binding.GoalRef, binding.WorkItemRef, binding.ExecutionRef,
		binding.ActionRef, binding.EffectIntentRef, binding.EffectIntentDigest, binding.EffectAttemptRef,
		strconv.FormatUint(binding.PlanGeneration, 10), strconv.FormatUint(binding.WorkItemGeneration, 10),
		strconv.FormatUint(binding.ActionFence, 10),
	)
}
