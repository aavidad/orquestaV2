package ports

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

type AgentStatus string

const (
	AgentPending   AgentStatus = "pending"
	AgentRunning   AgentStatus = "running"
	AgentCompleted AgentStatus = "completed"
	AgentFailed    AgentStatus = "failed"
)

type AgentFailureDisposition string

const (
	// AgentFailureDispositionTerminalSecurity marks a security failure that
	// must not be retried. Empty preserves the legacy retryable behavior.
	AgentFailureDispositionTerminalSecurity AgentFailureDisposition = "terminal_security"
)

// AgentPlacementRef identifica cuenta y colocación sin exponer perfiles, rutas ni credenciales.
type AgentPlacementRef struct{ value string }

func NewAgentPlacementRef(value string) (AgentPlacementRef, error) {
	if !validAgentIdentityRef(value) || strings.ContainsRune(value, '\x00') {
		return AgentPlacementRef{}, &AgentContractError{Code: "agent.placement_ref_invalid"}
	}
	return AgentPlacementRef{value: value}, nil
}
func (ref AgentPlacementRef) String() string { return ref.value }

type AgentCapabilities struct {
	ProviderRef                 string
	ModelRef                    string
	AgentRef                    string
	Unrestricted                bool
	RequierePreservacionEntorno bool
	RoleKeys                    []string
	SkillRefs                   []string
	ToolRefs                    []string
	CapabilityRefs              []string
}

// AgentRequirements describes the neutral capabilities required by one work
// item. Provider/model selection remains an adapter/composition concern.
type AgentRequirements struct {
	RoleKey        string
	SkillRefs      []string
	ToolRefs       []string
	CapabilityRefs []string
}

type AgentLaunchRequest struct {
	ExecutionRef goal.ExecutionRef
	// ReferenciaColocacion fija la cuenta/aislamiento elegido por el claim.
	ReferenciaColocacion AgentPlacementRef
	// SessionRef binds a child runtime to the exact durable execution tuple.
	// Empty preserves compositions that do not expose execution-bound APIs.
	SessionRef ExecutionSessionRef
	// AccessAuthority carries opaque capabilities deterministically bound to
	// SessionRef. Empty preserves adapters without an execution session broker.
	AccessAuthority AgentLaunchAccessAuthority
	// EgressAuthority carries the exact durable, non-secret egress policy
	// authority. Empty preserves launches that do not request egress. It is
	// deliberately excluded from AgentPrompt and interpreted only by an
	// isolation adapter in a later compilation boundary.
	EgressAuthority AgentLaunchEgressAuthority
	// ExecutionWorkspaceRef is an opaque, optional execution workspace binding.
	// When empty the adapter preserves the non-code path.  A physical path is
	// deliberately never carried through this provider-neutral request.
	ExecutionWorkspaceRef       ExecutionWorkspaceRef
	GoalRef                     goal.GoalRef
	WorkItemRef                 goal.WorkItemRef
	PlanGeneration              goal.PlanGeneration
	AppSpecGeneration           goal.AppSpecGeneration
	ExecutionAttempt            uint64
	SpecHash                    string
	ActorRef                    goal.ActorRef
	ProjectRef                  goal.ProjectRef
	Objective                   string
	PhaseRef                    string
	PhaseKey                    string
	PhaseTemplateRef            string
	PhaseInputRefs              []string
	PhaseCriterionRefs          []string
	RoleKey                     string
	SkillRefs                   []string
	ToolRefs                    []string
	CapabilityRefs              []string
	WriteSet                    []string
	OutputContract              string
	ArtifactMediaType           string
	IdempotencyKey              string
	MaxOutputBytes              int64
	BudgetDemand                governance.BudgetDemand
	SecurityCriticality         governance.SecurityCriticality
	ReasoningEffort             governance.ReasoningEffort
	RequierePreservacionEntorno bool
	// EffectAuthority carries the durable facts that authorized this physical
	// launch. Process adapters may ignore it; adapters crossing a stronger
	// isolation boundary can require ValidateAgentLaunchEffectAuthority.
	EffectAuthority AgentLaunchEffectAuthority
}

const (
	maxAgentLaunchEgressPolicyRefBytes        = 512
	maxAgentLaunchEgressCanonicalPayloadBytes = 64 << 10
)

// AgentLaunchEgressAuthority is an opaque, provider-neutral copy of authority
// already admitted and persisted by application. The three fields form one
// indivisible tuple; CanonicalPayload is validated as exact bytes, not parsed.
type AgentLaunchEgressAuthority struct {
	PolicyRef        string
	PayloadSHA256    string
	CanonicalPayload []byte
}

func (authority AgentLaunchEgressAuthority) IsEmpty() bool {
	return authority.PolicyRef == "" && authority.PayloadSHA256 == "" && len(authority.CanonicalPayload) == 0
}

func EqualAgentLaunchEgressAuthority(left, right AgentLaunchEgressAuthority) bool {
	return left.PolicyRef == right.PolicyRef && left.PayloadSHA256 == right.PayloadSHA256 &&
		bytes.Equal(left.CanonicalPayload, right.CanonicalPayload)
}

func ValidateAgentLaunchEgressAuthority(authority AgentLaunchEgressAuthority) error {
	if authority.IsEmpty() {
		return nil
	}
	if authority.PolicyRef == "" || authority.PayloadSHA256 == "" || len(authority.CanonicalPayload) == 0 {
		return &AgentContractError{Code: "agent.egress_authority_partial"}
	}
	if len(authority.PolicyRef) > maxAgentLaunchEgressPolicyRefBytes ||
		strings.TrimSpace(authority.PolicyRef) != authority.PolicyRef ||
		strings.ContainsRune(authority.PolicyRef, '\x00') || !utf8.ValidString(authority.PolicyRef) {
		return &AgentContractError{Code: "agent.egress_policy_ref_invalid"}
	}
	if len(authority.CanonicalPayload) > maxAgentLaunchEgressCanonicalPayloadBytes {
		return &AgentContractError{Code: "agent.egress_payload_too_large"}
	}
	digest := sha256.Sum256(authority.CanonicalPayload)
	if authority.PayloadSHA256 != fmt.Sprintf("%x", digest) {
		return &AgentContractError{Code: "agent.egress_payload_digest_invalid"}
	}
	return nil
}

// AgentLaunchAccessAuthority contains no credential material or physical
// endpoints. Session-aware isolation adapters resolve these opaque refs through
// their own composition ports.
type AgentLaunchAccessAuthority struct {
	ArtifactAccessRef  ExecutionArtifactAccessRef
	MCPAccessRef       ExecutionMCPAccessRef
	MailboxEndpointRef ExecutionMailboxEndpointRef
}

func ValidateAgentLaunchAccessAuthority(
	sessionRef ExecutionSessionRef,
	authority AgentLaunchAccessAuthority,
) error {
	empty := authority.ArtifactAccessRef.String() == "" && authority.MCPAccessRef.String() == "" &&
		authority.MailboxEndpointRef.String() == ""
	if empty {
		return nil
	}
	if sessionRef.String() == "" {
		return &AgentContractError{Code: "agent.access_authority_session_required"}
	}
	if _, err := NewExecutionSessionRef(sessionRef.String()); err != nil {
		return &AgentContractError{Code: "agent.execution_session_ref_invalid"}
	}
	if _, err := NewExecutionArtifactAccessRef(authority.ArtifactAccessRef.String()); err != nil {
		return &AgentContractError{Code: "agent.artifact_access_ref_invalid"}
	}
	if _, err := NewExecutionMCPAccessRef(authority.MCPAccessRef.String()); err != nil {
		return &AgentContractError{Code: "agent.mcp_access_ref_invalid"}
	}
	if _, err := NewExecutionMailboxEndpointRef(authority.MailboxEndpointRef.String()); err != nil {
		return &AgentContractError{Code: "agent.mailbox_endpoint_ref_invalid"}
	}
	return nil
}

// AgentLaunchEffectAuthority is provider-neutral and contains no credential
// material. Every value is copied from already persisted governance facts.
type AgentLaunchEffectAuthority struct {
	AuthorizationReceiptRef string
	EffectApprovalRef       string
	EffectAttemptRef        string
	ActionFence             uint64
	StartedAt               time.Time
	ClaimLeaseUntil         time.Time
	// ApprovalExpiresAt is zero only for durable automatic approvals, whose
	// existing contract has no expiry. Isolation-specific adapters may require
	// an explicit approval and therefore a non-zero value.
	ApprovalExpiresAt time.Time
}

func ValidateAgentLaunchEffectAuthority(authority AgentLaunchEffectAuthority) error {
	switch {
	case !validAgentReceiptRef(authority.AuthorizationReceiptRef):
		return &AgentContractError{Code: "agent.authorization_receipt_ref_required"}
	case !validAgentReceiptRef(authority.EffectApprovalRef):
		return &AgentContractError{Code: "agent.effect_approval_ref_required"}
	case !validAgentReceiptRef(authority.EffectAttemptRef):
		return &AgentContractError{Code: "agent.effect_attempt_ref_required"}
	case authority.ActionFence == 0:
		return &AgentContractError{Code: "agent.action_fence_required"}
	case authority.StartedAt.IsZero():
		return &AgentContractError{Code: "agent.effect_started_at_required"}
	case authority.ClaimLeaseUntil.IsZero():
		return &AgentContractError{Code: "agent.claim_lease_until_required"}
	case !authority.ClaimLeaseUntil.After(authority.StartedAt):
		return &AgentContractError{Code: "agent.claim_lease_until_invalid"}
	case !authority.ApprovalExpiresAt.IsZero() && !authority.ApprovalExpiresAt.After(authority.StartedAt):
		return &AgentContractError{Code: "agent.approval_expires_at_invalid"}
	default:
		return nil
	}
}

type AgentLaunchReceipt struct {
	ExecutionRef                goal.ExecutionRef
	GoalRef                     goal.GoalRef
	WorkItemRef                 goal.WorkItemRef
	PlanGeneration              goal.PlanGeneration
	AppSpecGeneration           goal.AppSpecGeneration
	ExecutionAttempt            uint64
	LaunchActionFence           uint64
	SpecHash                    string
	ProviderRef                 string
	ModelRef                    string
	AgentRef                    string
	ExternalRef                 string
	IdempotencyKey              string
	ReceiptRef                  string
	AcceptedAt                  time.Time
	RequierePreservacionEntorno bool
}

// AgentObserveRequest carries the durable identity of one accepted execution.
// Adapters must not need process-local launch state to locate the execution.
type AgentObserveRequest struct {
	ExecutionRef      goal.ExecutionRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	ExecutionAttempt  uint64
	LaunchActionFence uint64
	SpecHash          string
	ProviderRef       string
	ModelRef          string
	AgentRef          string
	ExternalRef       string
	SessionRef        ExecutionSessionRef
	ArtifactMediaType string
	MaxOutputBytes    int64
}

type AgentObservation struct {
	ExecutionRef       goal.ExecutionRef
	SpecHash           string
	Status             AgentStatus
	FailureDisposition AgentFailureDisposition
	MediaType          string
	Content            []byte
	ErrorCode          string
	Usage              governance.ResourceUsage
	ObservedAt         time.Time
}

func ValidateAgentObserveRequest(request AgentObserveRequest) error {
	if request.SessionRef.String() != "" {
		if _, err := NewExecutionSessionRef(request.SessionRef.String()); err != nil {
			return &AgentContractError{Code: "agent.observation_request_session_ref_invalid"}
		}
	}
	switch {
	case request.ExecutionRef.String() == "":
		return &AgentContractError{Code: "agent.observation_request_execution_ref_required"}
	case request.GoalRef.String() == "":
		return &AgentContractError{Code: "agent.observation_request_goal_ref_required"}
	case request.WorkItemRef.String() == "":
		return &AgentContractError{Code: "agent.observation_request_work_item_ref_required"}
	case request.PlanGeneration == 0:
		return &AgentContractError{Code: "agent.observation_request_plan_generation_required"}
	case request.AppSpecGeneration == 0:
		return &AgentContractError{Code: "agent.observation_request_app_spec_generation_required"}
	case request.ExecutionAttempt == 0:
		return &AgentContractError{Code: "agent.observation_request_execution_attempt_required"}
	case request.LaunchActionFence == 0:
		return &AgentContractError{Code: "agent.observation_request_launch_action_fence_required"}
	case request.SpecHash == "":
		return &AgentContractError{Code: "agent.observation_request_spec_hash_required"}
	case !goal.IsCanonicalAppSpecHash(request.SpecHash):
		return &AgentContractError{Code: "agent.observation_request_spec_hash_invalid"}
	case !validAgentIdentityRef(request.ProviderRef):
		return &AgentContractError{Code: "agent.observation_request_provider_ref_required"}
	case !validAgentIdentityRef(request.ModelRef):
		return &AgentContractError{Code: "agent.observation_request_model_ref_required"}
	case !validAgentIdentityRef(request.AgentRef):
		return &AgentContractError{Code: "agent.observation_request_agent_ref_required"}
	case strings.TrimSpace(request.ExternalRef) == "":
		return &AgentContractError{Code: "agent.observation_request_external_ref_required"}
	case strings.TrimSpace(request.ArtifactMediaType) == "":
		return &AgentContractError{Code: "agent.observation_request_artifact_media_type_required"}
	case request.MaxOutputBytes <= 0:
		return &AgentContractError{Code: "agent.observation_request_max_output_bytes_invalid"}
	default:
		return nil
	}
}

// ValidateAgentObserveTarget binds an observation query to the exact accepted
// launch. Launch-only prompt and governance fields remain outside this check.
func ValidateAgentObserveTarget(
	launch AgentLaunchRequest,
	receipt AgentLaunchReceipt,
	request AgentObserveRequest,
) error {
	if err := ValidateAgentObserveRequest(request); err != nil {
		return err
	}
	checks := []struct {
		matches bool
		field   string
	}{
		{request.ExecutionRef == receipt.ExecutionRef, "execution_ref"},
		{request.GoalRef == receipt.GoalRef, "goal_ref"},
		{request.WorkItemRef == receipt.WorkItemRef, "work_item_ref"},
		{request.PlanGeneration == receipt.PlanGeneration, "plan_generation"},
		{request.AppSpecGeneration == receipt.AppSpecGeneration, "app_spec_generation"},
		{request.ExecutionAttempt == receipt.ExecutionAttempt, "execution_attempt"},
		{request.LaunchActionFence == launch.EffectAuthority.ActionFence, "launch_action_fence"},
		{request.SpecHash == receipt.SpecHash, "spec_hash"},
		{request.ProviderRef == receipt.ProviderRef, "provider_ref"},
		{request.ModelRef == receipt.ModelRef, "model_ref"},
		{request.AgentRef == receipt.AgentRef, "agent_ref"},
		{request.ExternalRef == receipt.ExternalRef, "external_ref"},
		{request.SessionRef == launch.SessionRef, "session_ref"},
		{request.ArtifactMediaType == launch.ArtifactMediaType, "artifact_media_type"},
		{request.MaxOutputBytes == launch.MaxOutputBytes, "max_output_bytes"},
	}
	for _, check := range checks {
		if !check.matches {
			return &AgentContractError{Code: "agent.observation_request_" + check.field + "_mismatch"}
		}
	}
	return nil
}

type AgentContractError struct {
	Code string
}

func (err *AgentContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentContractErrorCode(err error) string {
	var contractErr *AgentContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateAgentCapabilities(capabilities AgentCapabilities) error {
	switch {
	case !validAgentIdentityRef(capabilities.ProviderRef):
		return &AgentContractError{Code: "agent.provider_ref_required"}
	case !validAgentIdentityRef(capabilities.ModelRef):
		return &AgentContractError{Code: "agent.model_ref_required"}
	case !validAgentIdentityRef(capabilities.AgentRef):
		return &AgentContractError{Code: "agent.agent_ref_required"}
	case !validUniqueAgentRefs(capabilities.RoleKeys, validRoleKey):
		return &AgentContractError{Code: "agent.role_keys_invalid"}
	case !validSkillRefs(capabilities.SkillRefs):
		return &AgentContractError{Code: "agent.skill_refs_invalid"}
	case !validToolRefs(capabilities.ToolRefs):
		return &AgentContractError{Code: "agent.tool_refs_invalid"}
	case !validCapabilityRefs(capabilities.CapabilityRefs):
		return &AgentContractError{Code: "agent.capability_refs_invalid"}
	default:
		return nil
	}
}

// MatchAgentCapabilities applies subset matching without knowing any provider.
// An unrestricted adapter accepts every valid requirement; restricted adapters
// must explicitly advertise the requested role, skills, tools and capabilities.
func MatchAgentCapabilities(capabilities AgentCapabilities, requirements AgentRequirements) bool {
	if ValidateAgentCapabilities(capabilities) != nil || !validAgentRequirements(requirements) {
		return false
	}
	if capabilities.Unrestricted {
		return true
	}
	return containsAgentRef(capabilities.RoleKeys, requirements.RoleKey) &&
		containsAllAgentRefs(capabilities.SkillRefs, requirements.SkillRefs) &&
		containsAllAgentRefs(capabilities.ToolRefs, requirements.ToolRefs) &&
		containsAllAgentRefs(capabilities.CapabilityRefs, requirements.CapabilityRefs)
}

func ValidateAgentLaunchRequest(request AgentLaunchRequest) error {
	if request.SessionRef.String() != "" {
		if _, err := NewExecutionSessionRef(request.SessionRef.String()); err != nil {
			return &AgentContractError{Code: "agent.execution_session_ref_invalid"}
		}
	}
	if err := ValidateAgentLaunchAccessAuthority(request.SessionRef, request.AccessAuthority); err != nil {
		return err
	}
	if err := ValidateAgentLaunchEgressAuthority(request.EgressAuthority); err != nil {
		return err
	}
	switch {
	case request.ReferenciaColocacion.String() == "":
		return &AgentContractError{Code: "agent.placement_ref_required"}
	case request.ExecutionRef.String() == "":
		return &AgentContractError{Code: "agent.execution_ref_required"}
	case request.GoalRef.String() == "":
		return &AgentContractError{Code: "agent.goal_ref_required"}
	case request.WorkItemRef.String() == "":
		return &AgentContractError{Code: "agent.work_item_ref_required"}
	case request.PlanGeneration == 0:
		return &AgentContractError{Code: "agent.plan_generation_required"}
	case request.AppSpecGeneration == 0:
		return &AgentContractError{Code: "agent.app_spec_generation_required"}
	case request.ExecutionAttempt == 0:
		return &AgentContractError{Code: "agent.execution_attempt_required"}
	case request.SpecHash == "":
		return &AgentContractError{Code: "agent.spec_hash_required"}
	case !goal.IsCanonicalAppSpecHash(request.SpecHash):
		return &AgentContractError{Code: "agent.spec_hash_invalid"}
	case request.ActorRef.String() == "":
		return &AgentContractError{Code: "agent.actor_ref_required"}
	case request.ProjectRef.String() == "":
		return &AgentContractError{Code: "agent.project_ref_required"}
	case strings.TrimSpace(request.Objective) == "":
		return &AgentContractError{Code: "agent.objective_required"}
	case !validPhaseRefValue(request.PhaseRef):
		return &AgentContractError{Code: "agent.phase_ref_invalid"}
	case !validPhaseKey(request.PhaseKey):
		return &AgentContractError{Code: "agent.phase_key_invalid"}
	case !validPhaseTemplateRefValue(request.PhaseTemplateRef):
		return &AgentContractError{Code: "agent.phase_template_ref_invalid"}
	case !validInputRefs(request.PhaseInputRefs):
		return &AgentContractError{Code: "agent.phase_input_refs_invalid"}
	case !validCriterionRefs(request.PhaseCriterionRefs):
		return &AgentContractError{Code: "agent.phase_criterion_refs_invalid"}
	case !validRoleKey(request.RoleKey):
		return &AgentContractError{Code: "agent.role_key_invalid"}
	case !validSkillRefs(request.SkillRefs):
		return &AgentContractError{Code: "agent.skill_refs_invalid"}
	case !validToolRefs(request.ToolRefs):
		return &AgentContractError{Code: "agent.tool_refs_invalid"}
	case !validCapabilityRefs(request.CapabilityRefs):
		return &AgentContractError{Code: "agent.capability_refs_invalid"}
	case !validOutputContract(request.OutputContract):
		return &AgentContractError{Code: "agent.output_contract_invalid"}
	case !validWriteSet(request.WriteSet):
		return &AgentContractError{Code: "agent.write_set_invalid"}
	case strings.TrimSpace(request.ArtifactMediaType) == "":
		return &AgentContractError{Code: "agent.artifact_media_type_required"}
	case strings.TrimSpace(request.IdempotencyKey) == "":
		return &AgentContractError{Code: "agent.idempotency_key_required"}
	case request.MaxOutputBytes <= 0:
		return &AgentContractError{Code: "agent.max_output_bytes_invalid"}
	default:
		return validateAgentGovernanceRequest(request)
	}
}

func validPhaseKey(value string) bool {
	key, err := goal.NewPhaseKey(value)
	return err == nil && key.String() == value
}

func validPhaseRefValue(value string) bool {
	ref, err := goal.NewPhaseRef(value)
	return err == nil && ref.String() == value
}

func validPhaseTemplateRefValue(value string) bool {
	ref, err := goal.NewPhaseTemplateRef(value)
	return err == nil && ref.String() == value
}

func validInputRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewInputRef(value)
		return err == nil && ref.String() == value
	})
}

func validCriterionRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewCriterionRef(value)
		return err == nil && ref.String() == value
	})
}

func validRoleKey(value string) bool {
	key, err := goal.NewRoleKey(value)
	return err == nil && key.String() == value
}

func validSkillRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewSkillRef(value)
		return err == nil && ref.String() == value
	})
}

func validToolRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewToolRef(value)
		return err == nil && ref.String() == value
	})
}

func validCapabilityRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewCapabilityRef(value)
		return err == nil && ref.String() == value
	})
}

func validUniqueAgentRefs(values []string, valid func(string) bool) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validAgentIdentityRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}

func validAgentRequirements(requirements AgentRequirements) bool {
	return validRoleKey(requirements.RoleKey) &&
		validSkillRefs(requirements.SkillRefs) &&
		validToolRefs(requirements.ToolRefs) &&
		validCapabilityRefs(requirements.CapabilityRefs)
}

func containsAgentRef(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsAllAgentRefs(available, required []string) bool {
	for _, wanted := range required {
		if !containsAgentRef(available, wanted) {
			return false
		}
	}
	return true
}

func validOutputContract(value string) bool {
	contract, err := goal.NewOutputContract(goal.OutputContractKind(value))
	return err == nil && string(contract.Kind()) == value
}

func validWriteSet(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		scope, err := goal.NewWriteScope(value)
		if err != nil || scope.String() != value {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func ValidateAgentLaunchReceipt(request AgentLaunchRequest, receipt AgentLaunchReceipt) error {
	if receipt.ExecutionRef.String() == "" {
		return &AgentContractError{Code: "agent.receipt_execution_ref_required"}
	}
	if receipt.ExecutionRef != request.ExecutionRef {
		return &AgentContractError{Code: "agent.receipt_execution_mismatch"}
	}
	if receipt.GoalRef.String() == "" {
		return &AgentContractError{Code: "agent.receipt_goal_ref_required"}
	}
	if receipt.GoalRef != request.GoalRef {
		return &AgentContractError{Code: "agent.receipt_goal_mismatch"}
	}
	if receipt.WorkItemRef.String() == "" {
		return &AgentContractError{Code: "agent.receipt_work_item_ref_required"}
	}
	if receipt.WorkItemRef != request.WorkItemRef {
		return &AgentContractError{Code: "agent.receipt_work_item_mismatch"}
	}
	if receipt.PlanGeneration == 0 {
		return &AgentContractError{Code: "agent.receipt_plan_generation_required"}
	}
	if receipt.PlanGeneration != request.PlanGeneration {
		return &AgentContractError{Code: "agent.receipt_plan_generation_mismatch"}
	}
	if receipt.AppSpecGeneration == 0 {
		return &AgentContractError{Code: "agent.receipt_app_spec_generation_required"}
	}
	if receipt.AppSpecGeneration != request.AppSpecGeneration {
		return &AgentContractError{Code: "agent.receipt_app_spec_generation_mismatch"}
	}
	if receipt.ExecutionAttempt == 0 {
		return &AgentContractError{Code: "agent.receipt_execution_attempt_required"}
	}
	if receipt.ExecutionAttempt != request.ExecutionAttempt {
		return &AgentContractError{Code: "agent.receipt_execution_attempt_mismatch"}
	}
	if receipt.LaunchActionFence != request.EffectAuthority.ActionFence {
		return &AgentContractError{Code: "agent.receipt_launch_action_fence_mismatch"}
	}
	if receipt.RequierePreservacionEntorno != request.RequierePreservacionEntorno {
		return &AgentContractError{Code: "agent.receipt_environment_preservation_mismatch"}
	}
	if receipt.SpecHash == "" {
		return &AgentContractError{Code: "agent.receipt_spec_hash_required"}
	}
	if !goal.IsCanonicalAppSpecHash(receipt.SpecHash) {
		return &AgentContractError{Code: "agent.receipt_spec_hash_invalid"}
	}
	if receipt.SpecHash != request.SpecHash {
		return &AgentContractError{Code: "agent.receipt_spec_hash_mismatch"}
	}
	if strings.TrimSpace(receipt.IdempotencyKey) == "" {
		return &AgentContractError{Code: "agent.receipt_idempotency_key_required"}
	}
	if receipt.IdempotencyKey != request.IdempotencyKey {
		return &AgentContractError{Code: "agent.receipt_idempotency_mismatch"}
	}
	if !validAgentReceiptRef(receipt.ReceiptRef) {
		return &AgentContractError{Code: "agent.receipt_ref_required"}
	}
	if !validAgentIdentityRef(receipt.ProviderRef) {
		return &AgentContractError{Code: "agent.receipt_provider_ref_required"}
	}
	if !validAgentIdentityRef(receipt.ModelRef) {
		return &AgentContractError{Code: "agent.receipt_model_ref_required"}
	}
	if !validAgentIdentityRef(receipt.AgentRef) {
		return &AgentContractError{Code: "agent.receipt_agent_ref_required"}
	}
	if strings.TrimSpace(receipt.ExternalRef) == "" {
		return &AgentContractError{Code: "agent.receipt_external_ref_required"}
	}
	if receipt.AcceptedAt.IsZero() {
		return &AgentContractError{Code: "agent.receipt_accepted_at_required"}
	}
	return nil
}

func ValidateAgentObservation(observation AgentObservation, maxOutputBytes int64) error {
	if observation.ExecutionRef.String() == "" {
		return &AgentContractError{Code: "agent.observation_execution_ref_required"}
	}
	if observation.SpecHash == "" {
		return &AgentContractError{Code: "agent.observation_spec_hash_required"}
	}
	if !goal.IsCanonicalAppSpecHash(observation.SpecHash) {
		return &AgentContractError{Code: "agent.observation_spec_hash_invalid"}
	}
	if observation.ObservedAt.IsZero() {
		return &AgentContractError{Code: "agent.observation_observed_at_required"}
	}
	if int64(len(observation.Content)) > maxOutputBytes {
		return &AgentContractError{Code: "agent.observation_output_too_large"}
	}
	if governance.ValidateResourceUsage(observation.Usage) != nil {
		return &AgentContractError{Code: "agent.observation_usage_invalid"}
	}
	switch observation.FailureDisposition {
	case "":
	case AgentFailureDispositionTerminalSecurity:
		if observation.Status != AgentFailed {
			return &AgentContractError{Code: "agent.observation_failure_disposition_conflict"}
		}
	default:
		return &AgentContractError{Code: "agent.observation_failure_disposition_invalid"}
	}
	switch observation.Status {
	case AgentPending, AgentRunning:
		if len(observation.Content) > 0 || observation.ErrorCode != "" {
			return &AgentContractError{Code: "agent.observation_nonterminal_payload"}
		}
	case AgentCompleted:
		if len(observation.Content) == 0 {
			return &AgentContractError{Code: "agent.observation_content_required"}
		}
		if strings.TrimSpace(observation.MediaType) == "" {
			return &AgentContractError{Code: "agent.observation_media_type_required"}
		}
		if observation.ErrorCode != "" {
			return &AgentContractError{Code: "agent.observation_terminal_conflict"}
		}
	case AgentFailed:
		if strings.TrimSpace(observation.ErrorCode) == "" {
			return &AgentContractError{Code: "agent.observation_error_code_required"}
		}
		if len(observation.Content) > 0 {
			return &AgentContractError{Code: "agent.observation_terminal_conflict"}
		}
	default:
		return &AgentContractError{Code: fmt.Sprintf("agent.observation_status_invalid:%s", observation.Status)}
	}
	return nil
}
