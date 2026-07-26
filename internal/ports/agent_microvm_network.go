package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
)

const (
	AgentMicroVMNetworkPolicySchema = "orquesta.agent-microvm-network-policy.v1"

	AgentMicroVMEgressBrokerOnly        AgentMicroVMEgressMode = "broker_only"
	AgentMicroVMEgressControlledProxy   AgentMicroVMEgressMode = "broker_and_controlled_proxy"
	AgentMicroVMVsockHostCID            uint32                 = 2
	AgentMicroVMMinimumGuestCID         uint32                 = 3
	AgentMicroVMReservedAnyCID          uint32                 = ^uint32(0)
	AgentMicroVMServiceBroker           string                 = "orquesta_broker"
	AgentMicroVMServiceControlledProxy  string                 = "controlled_egress_proxy"
	AgentMicroVMLaunchCredentialPurpose credentials.PurposeRef = "purpose:agent-microvm-launch-proof"
)

type AgentMicroVMEgressMode string

// AgentMicroVMNetworkScope binds a network policy to one exact launch. The
// logical agent identity is explicit because a vsock CID is routing metadata,
// never authentication.
type AgentMicroVMNetworkScope struct {
	ProjectRef        goal.ProjectRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ExecutionRef      goal.ExecutionRef
	PlanGeneration    goal.PlanGeneration
	AppSpecGeneration goal.AppSpecGeneration
	ExecutionAttempt  uint64
	SpecHash          string
	AgentRef          string
}

// AgentMicroVMVsockService identifies one host-side service reachable through
// the per-VM vsock backend. IdentityDigest pins the service identity or
// certificate material; it is not a secret.
type AgentMicroVMVsockService struct {
	Port           uint32
	IdentityRef    string
	IdentityDigest string
}

type AgentMicroVMLaunchCredentialBinding struct {
	Ref        credentials.CredentialRef
	OwnerRef   credentials.OwnerRef
	ScopeRef   credentials.ScopeRef
	PurposeRef credentials.PurposeRef
	Version    credentials.Version
}

// AgentMicroVMNetworkPolicy has no representation for a NIC, TAP, bridge, NAT
// or arbitrary destination. Absence is deliberate: adding IP connectivity
// requires a new reviewed contract rather than a permissive field.
type AgentMicroVMNetworkPolicy struct {
	Ref                  string
	Scope                AgentMicroVMNetworkScope
	GuestCID             uint32
	LaunchIdentityRef    string
	LaunchCredential     AgentMicroVMLaunchCredentialBinding
	LaunchAttestationRef goal.AttestationRef
	LaunchBindingDigest  string
	EgressMode           AgentMicroVMEgressMode
	Broker               AgentMicroVMVsockService
	ControlledProxy      *AgentMicroVMVsockService
	EgressPolicyRef      string
	EgressPolicyDigest   string
}

type AgentMicroVMNetworkContractError struct {
	Code string
}

func (err *AgentMicroVMNetworkContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentMicroVMNetworkContractErrorCode(err error) string {
	var contractErr *AgentMicroVMNetworkContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateAgentMicroVMNetworkPolicy(policy AgentMicroVMNetworkPolicy) error {
	switch {
	case !validWorkspaceLogicalRef(policy.Ref):
		return agentMicroVMNetworkError("policy_ref_invalid")
	case !validAgentMicroVMNetworkScope(policy.Scope):
		return agentMicroVMNetworkError("scope_invalid")
	case policy.GuestCID < AgentMicroVMMinimumGuestCID || policy.GuestCID == AgentMicroVMReservedAnyCID:
		return agentMicroVMNetworkError("guest_cid_invalid")
	case !validWorkspaceLogicalRef(policy.LaunchIdentityRef):
		return agentMicroVMNetworkError("launch_identity_ref_invalid")
	case !validAgentMicroVMLaunchCredential(policy.Scope, policy.LaunchCredential):
		return agentMicroVMNetworkError("launch_credential_invalid")
	case policy.LaunchAttestationRef.String() == "":
		return agentMicroVMNetworkError("launch_attestation_ref_invalid")
	case policy.LaunchBindingDigest != AgentMicroVMLaunchBindingDigest(
		policy.Scope,
		policy.LaunchIdentityRef,
		policy.LaunchCredential,
		policy.LaunchAttestationRef,
	):
		return agentMicroVMNetworkError("launch_binding_digest_invalid")
	case !validAgentMicroVMVsockService(policy.Broker):
		return agentMicroVMNetworkError("broker_invalid")
	case policy.EgressMode == AgentMicroVMEgressBrokerOnly:
		if policy.ControlledProxy != nil || policy.EgressPolicyRef != "" || policy.EgressPolicyDigest != "" {
			return agentMicroVMNetworkError("broker_only_scope_invalid")
		}
	case policy.EgressMode == AgentMicroVMEgressControlledProxy:
		if policy.ControlledProxy == nil || !validAgentMicroVMVsockService(*policy.ControlledProxy) {
			return agentMicroVMNetworkError("controlled_proxy_invalid")
		}
		if policy.ControlledProxy.Port == policy.Broker.Port {
			return agentMicroVMNetworkError("service_port_collision")
		}
		if !validWorkspaceLogicalRef(policy.EgressPolicyRef) || !validWorkspaceDigest(policy.EgressPolicyDigest) {
			return agentMicroVMNetworkError("egress_policy_invalid")
		}
	default:
		return agentMicroVMNetworkError("egress_mode_invalid")
	}
	return nil
}

// AgentMicroVMLaunchBindingDigest is a public causal binding, not an
// authenticator. The broker must additionally verify a challenge proof using
// LaunchCredential through CredentialStore and the exact attestation ref.
// Secret or proof bytes never enter this policy or its receipts.
func AgentMicroVMLaunchBindingDigest(
	scope AgentMicroVMNetworkScope,
	identityRef string,
	credential AgentMicroVMLaunchCredentialBinding,
	attestationRef goal.AttestationRef,
) string {
	document := agentMicroVMLaunchIdentityDocument{
		Schema:               "orquesta.agent-microvm-launch-identity.v1",
		ProjectRef:           scope.ProjectRef.String(),
		GoalRef:              scope.GoalRef.String(),
		WorkItemRef:          scope.WorkItemRef.String(),
		ExecutionRef:         scope.ExecutionRef.String(),
		PlanGeneration:       uint64(scope.PlanGeneration),
		AppSpecGeneration:    uint64(scope.AppSpecGeneration),
		ExecutionAttempt:     scope.ExecutionAttempt,
		SpecHash:             scope.SpecHash,
		AgentRef:             scope.AgentRef,
		IdentityRef:          identityRef,
		CredentialRef:        credential.Ref.String(),
		CredentialOwnerRef:   credential.OwnerRef.String(),
		CredentialScopeRef:   credential.ScopeRef.String(),
		CredentialPurposeRef: credential.PurposeRef.String(),
		CredentialVersion:    uint64(credential.Version),
		AttestationRef:       attestationRef.String(),
	}
	return agentMicroVMNetworkDocumentDigest(document)
}

func AgentMicroVMLaunchCredentialScopeRef(scope AgentMicroVMNetworkScope) credentials.ScopeRef {
	return credentials.ScopeRef(
		"scope:agent-microvm-launch:" + agentMicroVMNetworkDocumentDigest(agentMicroVMLaunchScopeDocument(scope)),
	)
}

func AgentMicroVMNetworkPolicyDigest(policy AgentMicroVMNetworkPolicy) (string, error) {
	if err := ValidateAgentMicroVMNetworkPolicy(policy); err != nil {
		return "", err
	}
	document := agentMicroVMNetworkPolicyProjection(policy)
	return agentMicroVMNetworkDocumentDigest(document), nil
}

func validAgentMicroVMNetworkScope(scope AgentMicroVMNetworkScope) bool {
	return scope.ProjectRef.String() != "" &&
		scope.GoalRef.String() != "" &&
		scope.WorkItemRef.String() != "" &&
		scope.ExecutionRef.String() != "" &&
		scope.PlanGeneration > 0 &&
		scope.AppSpecGeneration > 0 &&
		scope.ExecutionAttempt > 0 &&
		goal.IsCanonicalAppSpecHash(scope.SpecHash) &&
		validWorkspaceLogicalRef(scope.AgentRef)
}

func validAgentMicroVMVsockService(service AgentMicroVMVsockService) bool {
	return service.Port > 0 &&
		validWorkspaceLogicalRef(service.IdentityRef) &&
		validWorkspaceDigest(service.IdentityDigest)
}

func validAgentMicroVMLaunchCredential(
	scope AgentMicroVMNetworkScope,
	binding AgentMicroVMLaunchCredentialBinding,
) bool {
	return credentials.ValidateCredentialRef(binding.Ref) == nil &&
		credentials.ValidateOwnerRef(binding.OwnerRef) == nil &&
		binding.OwnerRef.String() == scope.ProjectRef.String() &&
		credentials.ValidateScopeRef(binding.ScopeRef) == nil &&
		binding.ScopeRef == AgentMicroVMLaunchCredentialScopeRef(scope) &&
		credentials.ValidatePurposeRef(binding.PurposeRef) == nil &&
		binding.PurposeRef == AgentMicroVMLaunchCredentialPurpose &&
		binding.Version > 0
}

func agentMicroVMNetworkError(suffix string) error {
	return &AgentMicroVMNetworkContractError{Code: "agent_microvm_network." + suffix}
}

type agentMicroVMLaunchIdentityDocument struct {
	Schema               string `json:"schema"`
	ProjectRef           string `json:"project_ref"`
	GoalRef              string `json:"goal_ref"`
	WorkItemRef          string `json:"work_item_ref"`
	ExecutionRef         string `json:"execution_ref"`
	PlanGeneration       uint64 `json:"plan_generation"`
	AppSpecGeneration    uint64 `json:"app_spec_generation"`
	ExecutionAttempt     uint64 `json:"execution_attempt"`
	SpecHash             string `json:"spec_hash"`
	AgentRef             string `json:"agent_ref"`
	IdentityRef          string `json:"identity_ref"`
	CredentialRef        string `json:"credential_ref"`
	CredentialOwnerRef   string `json:"credential_owner_ref"`
	CredentialScopeRef   string `json:"credential_scope_ref"`
	CredentialPurposeRef string `json:"credential_purpose_ref"`
	CredentialVersion    uint64 `json:"credential_version"`
	AttestationRef       string `json:"attestation_ref"`
}

type agentMicroVMNetworkPolicyDocument struct {
	Schema               string                               `json:"schema"`
	Ref                  string                               `json:"ref"`
	Scope                agentMicroVMLaunchIdentityDocument   `json:"scope"`
	GuestCID             uint32                               `json:"guest_cid"`
	LaunchIdentityRef    string                               `json:"launch_identity_ref"`
	LaunchCredential     agentMicroVMLaunchCredentialDocument `json:"launch_credential"`
	LaunchAttestationRef string                               `json:"launch_attestation_ref"`
	LaunchBindingDigest  string                               `json:"launch_binding_digest"`
	EgressMode           string                               `json:"egress_mode"`
	Broker               agentMicroVMVsockServiceDocument     `json:"broker"`
	ControlledProxy      *agentMicroVMVsockServiceDocument    `json:"controlled_proxy,omitempty"`
	EgressPolicyRef      string                               `json:"egress_policy_ref,omitempty"`
	EgressPolicyDigest   string                               `json:"egress_policy_digest,omitempty"`
}

type agentMicroVMVsockServiceDocument struct {
	Port           uint32 `json:"port"`
	IdentityRef    string `json:"identity_ref"`
	IdentityDigest string `json:"identity_digest"`
}

type agentMicroVMLaunchCredentialDocument struct {
	Ref        string `json:"ref"`
	OwnerRef   string `json:"owner_ref"`
	ScopeRef   string `json:"scope_ref"`
	PurposeRef string `json:"purpose_ref"`
	Version    uint64 `json:"version"`
}

func agentMicroVMNetworkPolicyProjection(policy AgentMicroVMNetworkPolicy) agentMicroVMNetworkPolicyDocument {
	scope := agentMicroVMLaunchScopeDocument(policy.Scope)
	document := agentMicroVMNetworkPolicyDocument{
		Schema: AgentMicroVMNetworkPolicySchema, Ref: policy.Ref, Scope: scope,
		GuestCID: policy.GuestCID, LaunchIdentityRef: policy.LaunchIdentityRef,
		LaunchCredential: agentMicroVMLaunchCredentialDocument{
			Ref: policy.LaunchCredential.Ref.String(), OwnerRef: policy.LaunchCredential.OwnerRef.String(),
			ScopeRef:   policy.LaunchCredential.ScopeRef.String(),
			PurposeRef: policy.LaunchCredential.PurposeRef.String(),
			Version:    uint64(policy.LaunchCredential.Version),
		},
		LaunchAttestationRef: policy.LaunchAttestationRef.String(),
		LaunchBindingDigest:  policy.LaunchBindingDigest, EgressMode: string(policy.EgressMode),
		Broker:          agentMicroVMVsockServiceProjection(policy.Broker),
		EgressPolicyRef: policy.EgressPolicyRef, EgressPolicyDigest: policy.EgressPolicyDigest,
	}
	if policy.ControlledProxy != nil {
		proxy := agentMicroVMVsockServiceProjection(*policy.ControlledProxy)
		document.ControlledProxy = &proxy
	}
	return document
}

func agentMicroVMLaunchScopeDocument(scope AgentMicroVMNetworkScope) agentMicroVMLaunchIdentityDocument {
	return agentMicroVMLaunchIdentityDocument{
		Schema:     "orquesta.agent-microvm-network-scope.v1",
		ProjectRef: scope.ProjectRef.String(), GoalRef: scope.GoalRef.String(),
		WorkItemRef: scope.WorkItemRef.String(), ExecutionRef: scope.ExecutionRef.String(),
		PlanGeneration:    uint64(scope.PlanGeneration),
		AppSpecGeneration: uint64(scope.AppSpecGeneration),
		ExecutionAttempt:  scope.ExecutionAttempt, SpecHash: scope.SpecHash, AgentRef: scope.AgentRef,
	}
}

func agentMicroVMVsockServiceProjection(service AgentMicroVMVsockService) agentMicroVMVsockServiceDocument {
	return agentMicroVMVsockServiceDocument(service)
}

func agentMicroVMNetworkDocumentDigest(document any) string {
	payload, err := json.Marshal(document)
	if err != nil {
		panic("agent microVM network canonical document cannot fail: " + err.Error())
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
