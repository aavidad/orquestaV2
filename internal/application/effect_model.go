package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

type EffectKind string

const (
	EffectKindAgentLaunch EffectKind = "agent_launch"
	EffectKindAgentStop   EffectKind = "agent_stop"
)

type EffectApprovalSource string

const (
	EffectApprovalSourceGoalConfirmation EffectApprovalSource = "goal_confirmation"
	EffectApprovalSourceDirectorDecision EffectApprovalSource = "director_decision"
	EffectApprovalSourceExplicitDecision EffectApprovalSource = "explicit_decision"
)

type EffectDecision string

const (
	EffectApproved EffectDecision = "approved"
	EffectDenied   EffectDecision = "denied"
)

// EffectSubject is the immutable causal scope copied into every effect fact.
// It does not own lifecycle; it binds evidence to the authoritative Goal.
type EffectSubject struct {
	ProjectRef        goal.ProjectRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ExecutionRef      goal.ExecutionRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	SpecHash          string
	ActorRef          goal.ActorRef
}

// EffectIntent is admission evidence, not proof that an external effect ran.
type EffectIntent struct {
	Ref                 string
	RequestRef          string
	RequestFingerprint  string
	ActionRef           string
	ActionKind          ActionKind
	Kind                EffectKind
	Subject             EffectSubject
	ProposedBy          identity.PrincipalRef
	Permission          identity.Permission
	Authority           identity.AuthorizationReceipt
	Demand              governance.BudgetDemand
	SecurityCriticality governance.SecurityCriticality
	ReasoningEffort     governance.ReasoningEffort
	IdempotencyKey      string
	CreatedAt           time.Time
	Digest              string
}

// EffectApproval is a distinct approved/denied decision over one exact intent.
type EffectApproval struct {
	Ref                  string
	RequestRef           string
	RequestFingerprint   string
	IntentRef            string
	IntentDigest         string
	Subject              EffectSubject
	ProposedBy           identity.PrincipalRef
	DecidedBy            identity.PrincipalRef
	Decision             EffectDecision
	Source               EffectApprovalSource
	SecurityCriticality  governance.SecurityCriticality
	Reason               string
	IdempotencyKey       string
	AuthorizationReceipt identity.AuthorizationReceipt
	DecidedAt            time.Time
	ExpiresAt            time.Time
}

// EffectAttempt is persisted before an adapter crosses the external boundary.
// ActionFence is the existing outbox attempt ordinal; effects add no scheduler.
type EffectAttempt struct {
	Ref            string
	IntentRef      string
	IntentDigest   string
	ApprovalRef    string
	Subject        EffectSubject
	ActionRef      string
	ActionFence    uint64
	WorkerRef      string
	IdempotencyKey string
	StartedAt      time.Time
	FinishedAt     time.Time
	FailureCode    string
}

// EffectReceipt is external confirmation and is never interchangeable with an
// admission ACK, approval, attempt, or ActionConsumptionReceipt.
type EffectReceipt struct {
	Ref            string
	IntentRef      string
	IntentDigest   string
	ApprovalRef    string
	AttemptRef     string
	Subject        EffectSubject
	ActionRef      string
	ActionFence    uint64
	IdempotencyKey string
	ExternalRef    string
	Status         string
	Usage          governance.ResourceUsage
	ConfirmedAt    time.Time
}

type EffectReplayRequest struct {
	RequestRef         string
	RequestFingerprint string
	PrincipalRef       identity.PrincipalRef
	ProjectRef         goal.ProjectRef
	GoalRef            goal.GoalRef
	IntentRef          string
	IntentDigest       string
}

type DecideEffectState struct {
	RequestRef           string
	RequestFingerprint   string
	AuthorizationReceipt identity.AuthorizationReceipt
	PrincipalRef         identity.PrincipalRef
	ProjectRef           goal.ProjectRef
	GoalRef              goal.GoalRef
	IntentRef            string
	IntentDigest         string
	Approval             EffectApproval
	OperationAt          time.Time
}

type RecordEffectAttemptState struct {
	Claim       ActionClaim
	Attempt     EffectAttempt
	OperationAt time.Time
}

// GovernanceRepository remains part of the one StateRepository authority.
// Implementations must persist each operation atomically with the same state.
type GovernanceRepository interface {
	EffectReplay(context.Context, EffectReplayRequest) (EffectApproval, bool, error)
	DecideEffect(context.Context, DecideEffectState) (EffectApproval, bool, error)
	RecordEffectAttempt(context.Context, RecordEffectAttemptState) (EffectAttempt, bool, error)
}

func EffectIntentDigest(intent EffectIntent) string {
	digest := sha256.New()
	resources := intent.Demand.Resources
	for _, field := range []string{
		"orquesta.effect.intent.v1", intent.Ref, intent.RequestRef, intent.RequestFingerprint,
		intent.ActionRef, string(intent.ActionKind), string(intent.Kind),
		intent.Subject.ProjectRef.String(), intent.Subject.GoalRef.String(), intent.Subject.WorkItemRef.String(),
		intent.Subject.ExecutionRef.String(), strconv.FormatUint(uint64(intent.Subject.PlanGeneration), 10),
		strconv.FormatUint(uint64(intent.Subject.AppSpecGeneration), 10), intent.Subject.SpecHash,
		intent.Subject.ActorRef.String(), intent.ProposedBy.String(), string(intent.Permission),
		intent.Authority.Ref(), intent.Demand.Ref, strconv.FormatInt(resources.Tokens, 10),
		strconv.FormatInt(resources.MoneyMicros, 10), string(resources.Currency),
		strconv.FormatInt(resources.ActiveTimeNS, 10), strconv.FormatInt(resources.ProcessSlots, 10),
		strconv.FormatInt(resources.DiskBytes, 10), string(intent.SecurityCriticality),
		string(intent.ReasoningEffort), intent.IdempotencyKey, intent.CreatedAt.UTC().Format(time.RFC3339Nano),
	} {
		writeFingerprintField(digest, field)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func ValidateEffectIntent(intent EffectIntent) error {
	subject := intent.Subject
	switch {
	case !validApplicationRef(intent.Ref) || !validApplicationRef(intent.RequestRef) ||
		!validEffectDigest(intent.RequestFingerprint) || !validApplicationRef(intent.ActionRef) ||
		!validApplicationRef(intent.IdempotencyKey):
		return errors.New("application.effect_intent_ref_invalid")
	case intent.Kind != EffectKindAgentLaunch && intent.Kind != EffectKindAgentStop:
		return errors.New("application.effect_kind_invalid")
	case (intent.Kind == EffectKindAgentLaunch && intent.ActionKind != ActionLaunchAgent) ||
		(intent.Kind == EffectKindAgentStop && intent.ActionKind != ActionStopAgent):
		return errors.New("application.effect_action_kind_mismatch")
	case subject.ProjectRef.String() == "" || subject.GoalRef.String() == "" ||
		subject.WorkItemRef.String() == "" || subject.ExecutionRef.String() == "" ||
		subject.PlanGeneration == 0 || subject.AppSpecGeneration == 0 ||
		!goal.IsCanonicalAppSpecHash(subject.SpecHash) || subject.ActorRef.String() == "" ||
		intent.ProposedBy.String() == "" || intent.CreatedAt.IsZero():
		return errors.New("application.effect_subject_invalid")
	case governance.ValidateBudgetDemand(intent.Demand) != nil:
		return errors.New("application.effect_demand_invalid")
	case governance.ValidateSecurityCriticality(intent.SecurityCriticality) != nil:
		return errors.New("application.effect_criticality_invalid")
	case governance.ValidateReasoningEffort(intent.ReasoningEffort) != nil:
		return errors.New("application.effect_effort_invalid")
	case identity.ValidatePermission(intent.Permission) != nil:
		return errors.New("application.effect_permission_invalid")
	case !effectIntentAuthorityValid(intent):
		return errors.New("application.effect_authority_invalid")
	case !validEffectDigest(intent.Digest) || intent.Digest != EffectIntentDigest(intent):
		return errors.New("application.effect_digest_invalid")
	default:
		return nil
	}
}

func effectIntentAuthorityValid(intent EffectIntent) bool {
	request := intent.Authority.Decision().Request()
	expectedResource := ""
	switch intent.Permission {
	case identity.PermissionGoalsCreate:
		expectedResource = intent.Subject.ProjectRef.String()
	case identity.PermissionGoalsDirect:
		expectedResource = intent.Subject.GoalRef.String()
	default:
		return false
	}
	return intent.Authority.Decision().Outcome() == identity.AuthorizationAllowed &&
		identity.RoleAllows(intent.Authority.Decision().Role(), intent.Permission) &&
		request.Principal().Ref == intent.ProposedBy && request.ProjectRef() == intent.Subject.ProjectRef &&
		request.Permission() == intent.Permission && request.ResourceRef() == expectedResource &&
		!intent.Authority.RecordedAt().After(intent.CreatedAt)
}

// ValidateEffectApproval binds automatic approvals to the original causal
// authority and reserves effects.approve for an explicit decision. Only normal
// risk may be auto-approved; critical effects also require proposer/approver
// separation.
func ValidateEffectApproval(intent EffectIntent, approval EffectApproval) error {
	if err := ValidateEffectIntent(intent); err != nil {
		return err
	}
	if !validApplicationRef(approval.Ref) || !validApplicationRef(approval.RequestRef) ||
		!validEffectDigest(approval.RequestFingerprint) ||
		approval.IntentRef != intent.Ref || approval.IntentDigest != intent.Digest ||
		approval.Subject != intent.Subject || approval.ProposedBy != intent.ProposedBy ||
		approval.IdempotencyKey != intent.IdempotencyKey ||
		approval.SecurityCriticality != intent.SecurityCriticality {
		return errors.New("application.effect_approval_intent_mismatch")
	}
	if approval.Decision != EffectApproved && approval.Decision != EffectDenied {
		return errors.New("application.effect_approval_decision_invalid")
	}
	if approval.Reason == "" || approval.Reason != strings.TrimSpace(approval.Reason) ||
		approval.DecidedBy.String() == "" || approval.DecidedAt.IsZero() ||
		approval.DecidedAt.Before(intent.CreatedAt) {
		return errors.New("application.effect_approval_fact_invalid")
	}
	if approval.Decision == EffectApproved {
		if !approval.ExpiresAt.After(approval.DecidedAt) {
			return errors.New("application.effect_approval_expiry_invalid")
		}
	} else if !approval.ExpiresAt.IsZero() || approval.Source != EffectApprovalSourceExplicitDecision {
		return errors.New("application.effect_denial_invalid")
	}
	if approval.Decision == EffectApproved && intent.SecurityCriticality == governance.SecurityCriticalityCritical &&
		approval.DecidedBy == intent.ProposedBy {
		return errors.New("application.effect_critical_separation_required")
	}
	switch approval.Source {
	case EffectApprovalSourceGoalConfirmation:
		if intent.SecurityCriticality != governance.SecurityCriticalityNormal ||
			intent.Permission != identity.PermissionGoalsCreate || approval.DecidedBy != intent.ProposedBy ||
			!sameAuthorizationReceipt(approval.AuthorizationReceipt, intent.Authority) {
			return errors.New("application.effect_goal_confirmation_authority_invalid")
		}
	case EffectApprovalSourceDirectorDecision:
		if intent.SecurityCriticality != governance.SecurityCriticalityNormal ||
			intent.Permission != identity.PermissionGoalsDirect || approval.DecidedBy != intent.ProposedBy ||
			!sameAuthorizationReceipt(approval.AuthorizationReceipt, intent.Authority) {
			return errors.New("application.effect_director_authority_invalid")
		}
	case EffectApprovalSourceExplicitDecision:
		if !effectApprovalAuthorizationValid(approval) {
			return errors.New("application.effect_explicit_authority_invalid")
		}
	default:
		return errors.New("application.effect_approval_source_invalid")
	}
	return nil
}

func sameAuthorizationReceipt(left, right identity.AuthorizationReceipt) bool {
	leftRequest, rightRequest := left.Decision().Request(), right.Decision().Request()
	return left.Ref() != "" && left.Ref() == right.Ref() && left.Decision().Outcome() == right.Decision().Outcome() &&
		left.Decision().Role() == right.Decision().Role() &&
		left.Decision().MembershipRevision() == right.Decision().MembershipRevision() &&
		leftRequest.RequestRef() == rightRequest.RequestRef() && leftRequest.Principal().Ref == rightRequest.Principal().Ref &&
		leftRequest.ProjectRef() == rightRequest.ProjectRef() && leftRequest.Permission() == rightRequest.Permission() &&
		leftRequest.ResourceRef() == rightRequest.ResourceRef() && left.RecordedAt().Equal(right.RecordedAt())
}

func validEffectDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
