package networkplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"orquesta/internal/ports"
)

const (
	PlanSchema       = "orquesta.agent-firecracker-network-plan.v1"
	ReceiptStatus    = "planned_not_applied"
	Transport        = "vsock_only"
	GuestProxyBridge = "guest_loopback_to_vsock"
)

var deniedDestinationClasses = []string{
	"direct_internet",
	"east_west",
	"host_lan",
	"host_loopback",
	"inbound",
	"link_local",
	"metadata",
	"rfc1918",
	"ula",
}

type Config struct {
	AdapterRef string
}

type RenderRequest struct {
	Policy               ports.AgentMicroVMNetworkPolicy
	ExpectedPolicyDigest string
	VsockBackendRef      string
	IdempotencyKey       string
}

type PlanReceipt struct {
	Status                     string
	AdapterRef                 string
	ProjectRef                 string
	GoalRef                    string
	WorkItemRef                string
	ExecutionRef               string
	AgentRef                   string
	PolicyRef                  string
	PolicyDigest               string
	LaunchIdentityRef          string
	LaunchCredentialRef        string
	LaunchCredentialOwnerRef   string
	LaunchCredentialScopeRef   string
	LaunchCredentialPurposeRef string
	LaunchCredentialVersion    uint64
	LaunchAttestationRef       string
	LaunchBindingDigest        string
	VsockBackendRef            string
	BackendBindingDigest       string
	PlanDigest                 string
	IdempotencyKey             string
	ReceiptRef                 string
}

type RenderedPlan struct {
	Document []byte
	Receipt  PlanReceipt
}

type Error struct {
	Code string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var planErr *Error
	if errors.As(err, &planErr) {
		return planErr.Code
	}
	return ""
}

func Render(config Config, request RenderRequest) (RenderedPlan, error) {
	if !validRef(config.AdapterRef) {
		return RenderedPlan{}, planError("config_invalid")
	}
	if err := ports.ValidateAgentMicroVMNetworkPolicy(request.Policy); err != nil {
		return RenderedPlan{}, planError("policy_invalid")
	}
	policyDigest, err := ports.AgentMicroVMNetworkPolicyDigest(request.Policy)
	if err != nil || request.ExpectedPolicyDigest != policyDigest {
		return RenderedPlan{}, planError("policy_digest_mismatch")
	}
	if !validRef(request.VsockBackendRef) {
		return RenderedPlan{}, planError("vsock_backend_ref_invalid")
	}
	if !validRef(request.IdempotencyKey) {
		return RenderedPlan{}, planError("idempotency_key_invalid")
	}

	backendDigest := backendBindingDigest(request.Policy, policyDigest, request.VsockBackendRef)
	document := planProjection(request, policyDigest, backendDigest)
	payload, err := json.Marshal(document)
	if err != nil {
		return RenderedPlan{}, planError("render_failed")
	}
	planDigest := digestBytes(payload)
	receipt := PlanReceipt{
		Status: ReceiptStatus, AdapterRef: config.AdapterRef,
		ProjectRef:   request.Policy.Scope.ProjectRef.String(),
		GoalRef:      request.Policy.Scope.GoalRef.String(),
		WorkItemRef:  request.Policy.Scope.WorkItemRef.String(),
		ExecutionRef: request.Policy.Scope.ExecutionRef.String(),
		AgentRef:     request.Policy.Scope.AgentRef,
		PolicyRef:    request.Policy.Ref, PolicyDigest: policyDigest,
		LaunchIdentityRef:          request.Policy.LaunchIdentityRef,
		LaunchCredentialRef:        request.Policy.LaunchCredential.Ref,
		LaunchCredentialOwnerRef:   request.Policy.LaunchCredential.OwnerRef,
		LaunchCredentialScopeRef:   request.Policy.LaunchCredential.ScopeRef,
		LaunchCredentialPurposeRef: request.Policy.LaunchCredential.PurposeRef,
		LaunchCredentialVersion:    request.Policy.LaunchCredential.Version,
		LaunchAttestationRef:       request.Policy.LaunchAttestationRef.String(),
		LaunchBindingDigest:        request.Policy.LaunchBindingDigest,
		VsockBackendRef:            request.VsockBackendRef, BackendBindingDigest: backendDigest,
		PlanDigest: planDigest, IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "agent-microvm-network-plan-receipt:" + planDigest,
	}
	return RenderedPlan{Document: payload, Receipt: receipt}, nil
}

func ValidateReceipt(request RenderRequest, candidate RenderedPlan) error {
	if digestBytes(candidate.Document) != candidate.Receipt.PlanDigest {
		return planError("document_digest_mismatch")
	}
	rendered, err := Render(Config{AdapterRef: candidate.Receipt.AdapterRef}, request)
	if err != nil {
		return err
	}
	if !bytes.Equal(candidate.Document, rendered.Document) || candidate.Receipt != rendered.Receipt {
		return planError("receipt_mismatch")
	}
	return nil
}

type planDocument struct {
	Schema                     string             `json:"schema"`
	Status                     string             `json:"status"`
	Scope                      planScope          `json:"scope"`
	PolicyRef                  string             `json:"policy_ref"`
	PolicyDigest               string             `json:"policy_digest"`
	LaunchIdentityRef          string             `json:"launch_identity_ref"`
	LaunchCredentialRef        string             `json:"launch_credential_ref"`
	LaunchCredentialOwnerRef   string             `json:"launch_credential_owner_ref"`
	LaunchCredentialScopeRef   string             `json:"launch_credential_scope_ref"`
	LaunchCredentialPurposeRef string             `json:"launch_credential_purpose_ref"`
	LaunchCredentialVersion    uint64             `json:"launch_credential_version"`
	LaunchAttestationRef       string             `json:"launch_attestation_ref"`
	LaunchBindingDigest        string             `json:"launch_binding_digest"`
	LaunchAuthentication       string             `json:"launch_authentication"`
	ServicesBlockedUntilAuth   bool               `json:"services_blocked_until_auth"`
	Transport                  string             `json:"transport"`
	GuestCID                   uint32             `json:"guest_cid"`
	HostCID                    uint32             `json:"host_cid"`
	VsockBackendRef            string             `json:"vsock_backend_ref"`
	BackendBindingDigest       string             `json:"backend_binding_digest"`
	NetworkInterfaces          []networkInterface `json:"network_interfaces"`
	TAP                        bool               `json:"tap"`
	Bridge                     bool               `json:"bridge"`
	NAT                        bool               `json:"nat"`
	Inbound                    bool               `json:"inbound"`
	EastWest                   bool               `json:"east_west"`
	DirectInternet             bool               `json:"direct_internet"`
	AllowedServices            []allowedService   `json:"allowed_vsock_services"`
	DeniedDestinationClasses   []string           `json:"denied_destination_classes"`
	GuestHTTPProxyBridge       string             `json:"guest_http_proxy_bridge,omitempty"`
	EgressPolicyRef            string             `json:"egress_policy_ref,omitempty"`
	EgressPolicyDigest         string             `json:"egress_policy_digest,omitempty"`
	IdempotencyKey             string             `json:"idempotency_key"`
}

type planScope struct {
	ProjectRef        string `json:"project_ref"`
	GoalRef           string `json:"goal_ref"`
	WorkItemRef       string `json:"work_item_ref"`
	ExecutionRef      string `json:"execution_ref"`
	PlanGeneration    uint64 `json:"plan_generation"`
	AppSpecGeneration uint64 `json:"app_spec_generation"`
	ExecutionAttempt  uint64 `json:"execution_attempt"`
	SpecHash          string `json:"spec_hash"`
	AgentRef          string `json:"agent_ref"`
}

type networkInterface struct{}

type allowedService struct {
	Service        string `json:"service"`
	CID            uint32 `json:"cid"`
	Port           uint32 `json:"port"`
	IdentityRef    string `json:"identity_ref"`
	IdentityDigest string `json:"identity_digest"`
}

func planProjection(
	request RenderRequest,
	policyDigest string,
	backendDigest string,
) planDocument {
	policy := request.Policy
	document := planDocument{
		Schema: PlanSchema, Status: ReceiptStatus,
		Scope: planScope{
			ProjectRef: policy.Scope.ProjectRef.String(), GoalRef: policy.Scope.GoalRef.String(),
			WorkItemRef: policy.Scope.WorkItemRef.String(), ExecutionRef: policy.Scope.ExecutionRef.String(),
			PlanGeneration:    uint64(policy.Scope.PlanGeneration),
			AppSpecGeneration: uint64(policy.Scope.AppSpecGeneration),
			ExecutionAttempt:  policy.Scope.ExecutionAttempt, SpecHash: policy.Scope.SpecHash,
			AgentRef: policy.Scope.AgentRef,
		},
		PolicyRef: policy.Ref, PolicyDigest: policyDigest,
		LaunchIdentityRef:          policy.LaunchIdentityRef,
		LaunchCredentialRef:        policy.LaunchCredential.Ref,
		LaunchCredentialOwnerRef:   policy.LaunchCredential.OwnerRef,
		LaunchCredentialScopeRef:   policy.LaunchCredential.ScopeRef,
		LaunchCredentialPurposeRef: policy.LaunchCredential.PurposeRef,
		LaunchCredentialVersion:    policy.LaunchCredential.Version,
		LaunchAttestationRef:       policy.LaunchAttestationRef.String(),
		LaunchBindingDigest:        policy.LaunchBindingDigest,
		LaunchAuthentication:       "credential_store_challenge_and_attestation_required",
		ServicesBlockedUntilAuth:   true,
		Transport:                  Transport, GuestCID: policy.GuestCID, HostCID: ports.AgentMicroVMVsockHostCID,
		VsockBackendRef: request.VsockBackendRef, BackendBindingDigest: backendDigest,
		NetworkInterfaces: make([]networkInterface, 0),
		AllowedServices: []allowedService{
			allowedServiceProjection(ports.AgentMicroVMServiceBroker, policy.Broker),
		},
		DeniedDestinationClasses: append([]string(nil), deniedDestinationClasses...),
		IdempotencyKey:           request.IdempotencyKey,
	}
	if policy.ControlledProxy != nil {
		document.AllowedServices = append(document.AllowedServices,
			allowedServiceProjection(ports.AgentMicroVMServiceControlledProxy, *policy.ControlledProxy))
		document.GuestHTTPProxyBridge = GuestProxyBridge
		document.EgressPolicyRef = policy.EgressPolicyRef
		document.EgressPolicyDigest = policy.EgressPolicyDigest
	}
	return document
}

func allowedServiceProjection(service string, endpoint ports.AgentMicroVMVsockService) allowedService {
	return allowedService{
		Service: service, CID: ports.AgentMicroVMVsockHostCID, Port: endpoint.Port,
		IdentityRef: endpoint.IdentityRef, IdentityDigest: endpoint.IdentityDigest,
	}
}

func backendBindingDigest(
	policy ports.AgentMicroVMNetworkPolicy,
	policyDigest string,
	backendRef string,
) string {
	document := struct {
		Schema       string `json:"schema"`
		ExecutionRef string `json:"execution_ref"`
		AgentRef     string `json:"agent_ref"`
		GuestCID     uint32 `json:"guest_cid"`
		PolicyDigest string `json:"policy_digest"`
		BackendRef   string `json:"backend_ref"`
	}{
		Schema:       "orquesta.agent-firecracker-vsock-backend-binding.v1",
		ExecutionRef: policy.Scope.ExecutionRef.String(), AgentRef: policy.Scope.AgentRef,
		GuestCID: policy.GuestCID, PolicyDigest: policyDigest, BackendRef: backendRef,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		panic("agent Firecracker backend binding cannot fail: " + err.Error())
	}
	return digestBytes(payload)
}

func digestBytes(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func validRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func planError(suffix string) error {
	return &Error{Code: "agent_firecracker_network_plan." + suffix}
}
