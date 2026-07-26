package networkplan

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func validPolicy(t *testing.T) ports.AgentMicroVMNetworkPolicy {
	t.Helper()
	projectRef, _ := goal.NewProjectRef("project:firecracker-network")
	goalRef, _ := goal.NewGoalRef("goal:firecracker-network")
	workItemRef, _ := goal.NewWorkItemRef("work:firecracker-network")
	executionRef, _ := goal.NewExecutionRef("execution:firecracker-network")
	scope := ports.AgentMicroVMNetworkScope{
		ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		PlanGeneration: 5, AppSpecGeneration: 4, ExecutionAttempt: 2,
		SpecHash: strings.Repeat("1", 64), AgentRef: "agent:codex",
	}
	proxy := ports.AgentMicroVMVsockService{
		Port: 52002, IdentityRef: "service:controlled-egress-proxy",
		IdentityDigest: strings.Repeat("3", 64),
	}
	policy := ports.AgentMicroVMNetworkPolicy{
		Ref: "policy:agent-network:controlled", Scope: scope, GuestCID: 73,
		LaunchIdentityRef: "launch-identity:execution:firecracker-network",
		LaunchCredential: ports.AgentMicroVMLaunchCredentialBinding{
			Ref:        "credential:launch-firecracker-network",
			OwnerRef:   scope.ProjectRef.String(),
			ScopeRef:   ports.AgentMicroVMLaunchCredentialScopeRef(scope),
			PurposeRef: ports.AgentMicroVMLaunchCredentialPurpose,
			Version:    1,
		},
		LaunchAttestationRef: mustAttestationRef(t, "attestation:launch:execution:firecracker-network"),
		EgressMode:           ports.AgentMicroVMEgressControlledProxy,
		Broker: ports.AgentMicroVMVsockService{
			Port: 52001, IdentityRef: "service:orquesta-broker",
			IdentityDigest: strings.Repeat("2", 64),
		},
		ControlledProxy: &proxy, EgressPolicyRef: "policy:egress:research",
		EgressPolicyDigest: strings.Repeat("4", 64),
	}
	policy.LaunchBindingDigest = ports.AgentMicroVMLaunchBindingDigest(
		scope, policy.LaunchIdentityRef, policy.LaunchCredential, policy.LaunchAttestationRef,
	)
	return policy
}

func mustAttestationRef(t *testing.T, value string) goal.AttestationRef {
	t.Helper()
	ref, err := goal.NewAttestationRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func validRequest(t *testing.T) RenderRequest {
	t.Helper()
	policy := validPolicy(t)
	policyDigest, err := ports.AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	reservationRequest := ports.AgentMicroVMVsockCIDReservationRequest{
		PoolRef: "vsock-pool:host-128g-16", Scope: policy.Scope,
		OwnerRef: policy.LaunchIdentityRef, LeaseDuration: 10 * time.Minute,
		IdempotencyKey: "reserve-vsock-cid:execution:firecracker-network:2",
	}
	requestDigest, err := ports.AgentMicroVMVsockCIDReservationRequestDigest(reservationRequest)
	if err != nil {
		t.Fatal(err)
	}
	scopeDigest, err := ports.AgentMicroVMNetworkScopeDigest(policy.Scope)
	if err != nil {
		t.Fatal(err)
	}
	acquiredAt := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	lease := ports.AgentMicroVMVsockCIDLease{
		PoolRef: reservationRequest.PoolRef, ScopeDigest: scopeDigest,
		ExecutionRef: policy.Scope.ExecutionRef.String(), AgentRef: policy.Scope.AgentRef,
		OwnerRef: policy.LaunchIdentityRef, RequestDigest: requestDigest,
		IdempotencyKey: reservationRequest.IdempotencyKey, GuestCID: policy.GuestCID,
		FencingToken: 7, Revision: 1, AcquiredAt: acquiredAt,
		ExpiresAt:  acquiredAt.Add(reservationRequest.LeaseDuration),
		AdapterRef: "adapter:firecracker-vsock-cid",
	}
	lease.LeaseRef = ports.AgentMicroVMVsockCIDLeaseRef(lease)
	lease.VsockBackendRef = ports.AgentMicroVMVsockBackendRef(lease.LeaseRef)
	lease.ReceiptRef = ports.AgentMicroVMVsockCIDLeaseReceiptRef(lease)
	return RenderRequest{
		Policy: policy, ExpectedPolicyDigest: policyDigest,
		CIDLease: lease, IdempotencyKey: "network-plan:execution:firecracker-network:2",
	}
}

func TestRenderProducesDeterministicVsockOnlyPlan(t *testing.T) {
	config := Config{AdapterRef: "adapter:agent-firecracker-network-plan"}
	request := validRequest(t)
	first, err := Render(config, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(config, request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Document, second.Document) || first.Receipt != second.Receipt {
		t.Fatalf("render is not deterministic:\nfirst=%s\nsecond=%s", first.Document, second.Document)
	}
	if err := ValidateReceipt(request, first); err != nil {
		t.Fatalf("receipt rejected: %v", err)
	}

	var document planDocument
	if err := json.Unmarshal(first.Document, &document); err != nil {
		t.Fatal(err)
	}
	if document.Schema != PlanSchema || document.Status != ReceiptStatus ||
		document.Transport != Transport || document.HostCID != ports.AgentMicroVMVsockHostCID ||
		document.GuestCID != request.Policy.GuestCID {
		t.Fatalf("unexpected plan identity: %+v", document)
	}
	if len(document.NetworkInterfaces) != 0 || document.TAP || document.Bridge || document.NAT ||
		document.Inbound || document.EastWest || document.DirectInternet {
		t.Fatalf("IP or peer connectivity enabled: %+v", document)
	}
	if len(document.AllowedServices) != 2 ||
		document.AllowedServices[0].Service != ports.AgentMicroVMServiceBroker ||
		document.AllowedServices[1].Service != ports.AgentMicroVMServiceControlledProxy {
		t.Fatalf("unexpected allowlist: %+v", document.AllowedServices)
	}
	for _, service := range document.AllowedServices {
		if service.CID != ports.AgentMicroVMVsockHostCID || service.Port == 0 {
			t.Fatalf("service escaped exact host vsock: %+v", service)
		}
	}
	if document.GuestHTTPProxyBridge != GuestProxyBridge ||
		document.LaunchAuthentication != "credential_store_challenge_and_attestation_required" ||
		!document.ServicesBlockedUntilAuth {
		t.Fatalf("proxy/authentication contract missing: %+v", document)
	}
	if strings.Join(document.DeniedDestinationClasses, ",") != strings.Join(deniedDestinationClasses, ",") {
		t.Fatalf("denials drifted: %v", document.DeniedDestinationClasses)
	}
	if first.Receipt.Status != ReceiptStatus ||
		first.Receipt.PolicyDigest != request.ExpectedPolicyDigest ||
		first.Receipt.LaunchCredentialRef != request.Policy.LaunchCredential.Ref ||
		first.Receipt.VsockCIDLeaseRef != request.CIDLease.LeaseRef ||
		first.Receipt.VsockCIDFencingToken != request.CIDLease.FencingToken ||
		first.Receipt.VsockCIDLeaseRevision != request.CIDLease.Revision ||
		first.Receipt.VsockCIDLeaseReceiptRef != request.CIDLease.ReceiptRef ||
		!strings.HasPrefix(first.Receipt.ReceiptRef, "agent-microvm-network-plan-receipt:") {
		t.Fatalf("receipt does not bind plan: %+v", first.Receipt)
	}
}

func TestRenderRejectsPolicyTamperAndAmbiguousRouting(t *testing.T) {
	config := Config{AdapterRef: "adapter:agent-firecracker-network-plan"}
	tests := map[string]struct {
		mutate   func(*RenderRequest)
		wantCode string
	}{
		"policy digest tamper": {
			func(request *RenderRequest) { request.ExpectedPolicyDigest = strings.Repeat("0", 64) },
			"agent_firecracker_network_plan.policy_digest_mismatch",
		},
		"scope replay": {
			func(request *RenderRequest) { request.Policy.Scope.ExecutionAttempt++ },
			"agent_firecracker_network_plan.policy_invalid",
		},
		"goal spoofing": {
			func(request *RenderRequest) {
				request.Policy.Scope.GoalRef, _ = goal.NewGoalRef("goal:other")
			},
			"agent_firecracker_network_plan.policy_invalid",
		},
		"credential substitution": {
			func(request *RenderRequest) { request.Policy.LaunchCredential.Ref = "credential:other" },
			"agent_firecracker_network_plan.policy_invalid",
		},
		"lease missing": {
			func(request *RenderRequest) { request.CIDLease = ports.AgentMicroVMVsockCIDLease{} },
			"agent_firecracker_network_plan.cid_lease_invalid",
		},
		"lease from other owner": {
			func(request *RenderRequest) {
				request.CIDLease.OwnerRef = "launch-identity:other"
				request.CIDLease.LeaseRef = ports.AgentMicroVMVsockCIDLeaseRef(request.CIDLease)
				request.CIDLease.VsockBackendRef = ports.AgentMicroVMVsockBackendRef(request.CIDLease.LeaseRef)
				request.CIDLease.ReceiptRef = ports.AgentMicroVMVsockCIDLeaseReceiptRef(request.CIDLease)
			},
			"agent_firecracker_network_plan.cid_lease_binding_mismatch",
		},
		"idempotency missing": {
			func(request *RenderRequest) { request.IdempotencyKey = "" },
			"agent_firecracker_network_plan.idempotency_key_invalid",
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			request := validRequest(t)
			testCase.mutate(&request)
			_, err := Render(config, request)
			if code := ErrorCode(err); code != testCase.wantCode {
				t.Fatalf("error code = %q, want %q (err=%v)", code, testCase.wantCode, err)
			}
		})
	}
}

func TestRenderBrokerOnlyHasNoProxyOrEgressPolicy(t *testing.T) {
	request := validRequest(t)
	request.Policy.EgressMode = ports.AgentMicroVMEgressBrokerOnly
	request.Policy.ControlledProxy = nil
	request.Policy.EgressPolicyRef = ""
	request.Policy.EgressPolicyDigest = ""
	request.ExpectedPolicyDigest, _ = ports.AgentMicroVMNetworkPolicyDigest(request.Policy)
	rendered, err := Render(Config{AdapterRef: "adapter:agent-firecracker-network-plan"}, request)
	if err != nil {
		t.Fatal(err)
	}
	var document planDocument
	if err := json.Unmarshal(rendered.Document, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.AllowedServices) != 1 ||
		document.AllowedServices[0].Service != ports.AgentMicroVMServiceBroker ||
		document.GuestHTTPProxyBridge != "" ||
		document.EgressPolicyRef != "" || document.EgressPolicyDigest != "" {
		t.Fatalf("broker-only plan leaked proxy access: %+v", document)
	}
}

func TestRenderReceiptBindsBackendAndRejectsMutation(t *testing.T) {
	request := validRequest(t)
	rendered, err := Render(Config{AdapterRef: "adapter:agent-firecracker-network-plan"}, request)
	if err != nil {
		t.Fatal(err)
	}
	changedLease := request
	changedLease.CIDLease.Revision++
	changedLease.CIDLease.ReceiptRef = ports.AgentMicroVMVsockCIDLeaseReceiptRef(changedLease.CIDLease)
	other, err := Render(Config{AdapterRef: "adapter:agent-firecracker-network-plan"}, changedLease)
	if err != nil {
		t.Fatal(err)
	}
	if other.Receipt.PlanDigest == rendered.Receipt.PlanDigest ||
		other.Receipt.BackendBindingDigest == rendered.Receipt.BackendBindingDigest {
		t.Fatal("backend mutation reused plan or binding digest")
	}
	tampered := rendered
	tampered.Receipt.AgentRef = "agent:other"
	if code := ErrorCode(ValidateReceipt(request, tampered)); code != "agent_firecracker_network_plan.receipt_mismatch" {
		t.Fatalf("tampered receipt code = %q", code)
	}
	tampered = rendered
	tampered.Document = append([]byte(nil), rendered.Document...)
	tampered.Document[len(tampered.Document)-1] ^= 1
	if code := ErrorCode(ValidateReceipt(request, tampered)); code !=
		"agent_firecracker_network_plan.document_digest_mismatch" {
		t.Fatalf("tampered document code = %q", code)
	}
}
