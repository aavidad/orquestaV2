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
	EffectKindAgentLaunch      EffectKind = "agent_launch"
	EffectKindAgentStop        EffectKind = "agent_stop"
	EffectKindPrepareWorkspace EffectKind = "prepare_workspace"
	EffectKindCommitChange     EffectKind = "commit_change"
	EffectKindIntegrateChange  EffectKind = "integrate_change"
	EffectRiskPolicyV1                    = "orquesta.effect-risk.v1"
)

type EffectStatus string

const (
	EffectStatusAccepted         EffectStatus = "accepted"
	EffectStatusStopped          EffectStatus = "stopped"
	EffectStatusAlreadyStopped   EffectStatus = "already_stopped"
	EffectStatusAlreadyCompleted EffectStatus = "already_completed"
	EffectStatusAlreadyFailed    EffectStatus = "already_failed"
	EffectStatusPrepared         EffectStatus = "prepared"
	EffectStatusCommitted        EffectStatus = "committed"
	EffectStatusIntegrated       EffectStatus = "integrated"
	EffectStatusConflicted       EffectStatus = "conflicted"
	EffectStatusStale            EffectStatus = "stale"
)

type EffectApprovalSource string

const (
	EffectApprovalSourceGoalConfirmation    EffectApprovalSource = "goal_confirmation"
	EffectApprovalSourceDirectorDecision    EffectApprovalSource = "director_decision"
	EffectApprovalSourceExplicitDecision    EffectApprovalSource = "explicit_decision"
	EffectApprovalSourceIntegrationDecision EffectApprovalSource = "integration_decision"
)

type EffectDecision string

const (
	EffectApproved EffectDecision = "approved"
	EffectDenied   EffectDecision = "denied"
)

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
	PolicyHash          string
	PolicyRevision      uint64
	QuotaRetryDelay     time.Duration
	ApprovalTTL         time.Duration
	TargetDigest        string
	IdempotencyKey      string
	CreatedAt           time.Time
	Digest              string
}

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
	PolicyHash           string
	PolicyRevision       uint64
	TargetDigest         string
	Reason               string
	IdempotencyKey       string
	AuthorizationReceipt identity.AuthorizationReceipt
	DecidedAt            time.Time
	ExpiresAt            time.Time
}

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
}

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
	Status         EffectStatus
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
		string(intent.ReasoningEffort), intent.PolicyHash, strconv.FormatUint(intent.PolicyRevision, 10),
		strconv.FormatInt(int64(intent.QuotaRetryDelay), 10), strconv.FormatInt(int64(intent.ApprovalTTL), 10),
		intent.TargetDigest, intent.IdempotencyKey,
		intent.CreatedAt.UTC().Format(time.RFC3339Nano),
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
	case !validEffectKind(intent.Kind):
		return errors.New("application.effect_kind_invalid")
	case !effectActionKindMatches(intent.Kind, intent.ActionKind):
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
	case !validEffectDigest(intent.PolicyHash) || intent.PolicyRevision == 0 || intent.QuotaRetryDelay <= 0 ||
		intent.ApprovalTTL <= 0 || !validEffectDigest(intent.TargetDigest):
		return errors.New("application.effect_policy_hash_invalid")
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
	case identity.PermissionChangesIntegrate:
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

func ValidateEffectApproval(intent EffectIntent, approval EffectApproval) error {
	if err := ValidateEffectIntent(intent); err != nil {
		return err
	}
	if !validApplicationRef(approval.Ref) || !validApplicationRef(approval.RequestRef) ||
		!validEffectDigest(approval.RequestFingerprint) ||
		approval.IntentRef != intent.Ref || approval.IntentDigest != intent.Digest ||
		approval.Subject != intent.Subject || approval.ProposedBy != intent.ProposedBy ||
		approval.IdempotencyKey != intent.IdempotencyKey ||
		approval.SecurityCriticality != intent.SecurityCriticality || approval.PolicyHash != intent.PolicyHash ||
		approval.PolicyRevision != intent.PolicyRevision || approval.TargetDigest != intent.TargetDigest {
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
		if approval.Source == EffectApprovalSourceExplicitDecision &&
			!approval.ExpiresAt.Equal(approval.DecidedAt.Add(intent.ApprovalTTL)) {
			return errors.New("application.effect_approval_expiry_invalid")
		}
		if approval.Source != EffectApprovalSourceExplicitDecision && !approval.ExpiresAt.IsZero() {
			return errors.New("application.effect_automatic_approval_expiry_forbidden")
		}
	} else if !approval.ExpiresAt.IsZero() || approval.Source != EffectApprovalSourceExplicitDecision {
		return errors.New("application.effect_denial_invalid")
	}
	if approval.Decision == EffectApproved && intent.SecurityCriticality == governance.SecurityCriticalityCritical &&
		approval.DecidedBy == intent.ProposedBy {
		return errors.New("application.effect_critical_separation_required")
	}
	if approval.Decision == EffectApproved && intent.SecurityCriticality == governance.SecurityCriticalityCritical &&
		!identity.IsProjectAuthority(approval.AuthorizationReceipt.Decision().Role()) {
		return errors.New("application.effect_critical_project_authority_required")
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
	case EffectApprovalSourceIntegrationDecision:
		if intent.SecurityCriticality != governance.SecurityCriticalityNormal ||
			intent.Permission != identity.PermissionChangesIntegrate || approval.DecidedBy != intent.ProposedBy ||
			!sameAuthorizationReceipt(approval.AuthorizationReceipt, intent.Authority) {
			return errors.New("application.effect_integration_authority_invalid")
		}
	default:
		return errors.New("application.effect_approval_source_invalid")
	}
	return nil
}

func validEffectKind(kind EffectKind) bool {
	switch kind {
	case EffectKindAgentLaunch, EffectKindAgentStop, EffectKindPrepareWorkspace,
		EffectKindCommitChange, EffectKindIntegrateChange:
		return true
	default:
		return false
	}
}

func effectActionKindMatches(kind EffectKind, action ActionKind) bool {
	switch kind {
	case EffectKindAgentLaunch:
		return action == ActionLaunchAgent
	case EffectKindAgentStop:
		return action == ActionStopAgent
	case EffectKindPrepareWorkspace:
		return action == ActionPrepareWorkspace
	case EffectKindCommitChange:
		return action == ActionCommitChange
	case EffectKindIntegrateChange:
		return action == ActionIntegrateChange
	default:
		return false
	}
}

func sameAuthorizationReceipt(left, right identity.AuthorizationReceipt) bool {
	return left.Ref() != "" && left == right
}

func validEffectDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
