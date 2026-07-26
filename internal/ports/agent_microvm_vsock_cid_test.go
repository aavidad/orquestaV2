package ports

import (
	"testing"
	"time"
)

func validAgentMicroVMVsockCIDReservationRequest(
	t *testing.T,
) AgentMicroVMVsockCIDReservationRequest {
	t.Helper()
	policy := validAgentMicroVMNetworkPolicy(t)
	return AgentMicroVMVsockCIDReservationRequest{
		PoolRef:        "vsock-pool:host-128g-16",
		Scope:          policy.Scope,
		OwnerRef:       policy.LaunchIdentityRef,
		LeaseDuration:  5 * time.Minute,
		IdempotencyKey: "reserve-vsock-cid:execution:network-test:1",
	}
}

func validAgentMicroVMVsockCIDLease(t *testing.T) AgentMicroVMVsockCIDLease {
	t.Helper()
	request := validAgentMicroVMVsockCIDReservationRequest(t)
	requestDigest, err := AgentMicroVMVsockCIDReservationRequestDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	scopeDigest, err := AgentMicroVMNetworkScopeDigest(request.Scope)
	if err != nil {
		t.Fatal(err)
	}
	acquiredAt := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	lease := AgentMicroVMVsockCIDLease{
		PoolRef: request.PoolRef, ScopeDigest: scopeDigest,
		ExecutionRef: request.Scope.ExecutionRef.String(), AgentRef: request.Scope.AgentRef,
		OwnerRef: request.OwnerRef, RequestDigest: requestDigest,
		IdempotencyKey: request.IdempotencyKey, GuestCID: 73,
		FencingToken: 4, Revision: 2, AcquiredAt: acquiredAt,
		ExpiresAt:  acquiredAt.Add(5 * time.Minute),
		AdapterRef: "adapter:firecracker-vsock-cid",
	}
	lease.LeaseRef = AgentMicroVMVsockCIDLeaseRef(lease)
	lease.VsockBackendRef = AgentMicroVMVsockBackendRef(lease.LeaseRef)
	lease.ReceiptRef = AgentMicroVMVsockCIDLeaseReceiptRef(lease)
	return lease
}

func TestAgentMicroVMVsockCIDLeaseBindsScopeOwnerFenceAndRevision(t *testing.T) {
	lease := validAgentMicroVMVsockCIDLease(t)
	if err := ValidateAgentMicroVMVsockCIDLease(lease); err != nil {
		t.Fatal(err)
	}
	if lease.GuestCID < AgentMicroVMMinimumGuestCID ||
		lease.GuestCID == AgentMicroVMReservedAnyCID {
		t.Fatalf("unsafe guest CID accepted: %d", lease.GuestCID)
	}

	mutations := map[string]func(*AgentMicroVMVsockCIDLease){
		"scope": func(candidate *AgentMicroVMVsockCIDLease) {
			candidate.ScopeDigest = agentMicroVMNetworkDocumentDigest("other")
		},
		"owner":    func(candidate *AgentMicroVMVsockCIDLease) { candidate.OwnerRef = "launch-identity:other" },
		"fence":    func(candidate *AgentMicroVMVsockCIDLease) { candidate.FencingToken++ },
		"revision": func(candidate *AgentMicroVMVsockCIDLease) { candidate.Revision++ },
		"expiry":   func(candidate *AgentMicroVMVsockCIDLease) { candidate.ExpiresAt = candidate.ExpiresAt.Add(time.Second) },
		"backend":  func(candidate *AgentMicroVMVsockCIDLease) { candidate.VsockBackendRef = "vsock-backend:other" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := lease
			mutate(&candidate)
			if err := ValidateAgentMicroVMVsockCIDLease(candidate); err == nil {
				t.Fatal("tampered lease accepted")
			}
		})
	}
}

func TestAgentMicroVMVsockCIDContractRejectsReservedCIDsAndForeignRelease(t *testing.T) {
	for _, cid := range []uint32{0, 1, AgentMicroVMVsockHostCID, AgentMicroVMReservedAnyCID} {
		lease := validAgentMicroVMVsockCIDLease(t)
		lease.GuestCID = cid
		lease.LeaseRef = AgentMicroVMVsockCIDLeaseRef(lease)
		lease.VsockBackendRef = AgentMicroVMVsockBackendRef(lease.LeaseRef)
		lease.ReceiptRef = AgentMicroVMVsockCIDLeaseReceiptRef(lease)
		if err := ValidateAgentMicroVMVsockCIDLease(lease); err == nil {
			t.Fatalf("reserved CID %d accepted", cid)
		}
	}

	lease := validAgentMicroVMVsockCIDLease(t)
	request := AgentMicroVMVsockCIDReleaseRequest{
		PoolRef: lease.PoolRef, ScopeDigest: lease.ScopeDigest,
		LeaseRef: lease.LeaseRef, OwnerRef: lease.OwnerRef,
		FencingToken: lease.FencingToken, ExpectedRevision: lease.Revision,
		IdempotencyKey: "release-vsock-cid:execution:network-test:1",
	}
	releasedAt := lease.AcquiredAt.Add(time.Minute)
	receipt := AgentMicroVMVsockCIDReleaseReceipt{
		PoolRef: request.PoolRef, ScopeDigest: request.ScopeDigest,
		LeaseRef: request.LeaseRef, OwnerRef: request.OwnerRef,
		GuestCID: lease.GuestCID, FencingToken: request.FencingToken,
		FinalRevision: request.ExpectedRevision, ReleasedAt: releasedAt,
		IdempotencyKey: request.IdempotencyKey, AdapterRef: lease.AdapterRef,
	}
	receipt.ReceiptRef = AgentMicroVMVsockCIDReleaseReceiptRef(receipt)
	if err := ValidateAgentMicroVMVsockCIDReleaseReceipt(request, receipt); err != nil {
		t.Fatal(err)
	}

	foreign := request
	foreign.OwnerRef = "launch-identity:other"
	if err := ValidateAgentMicroVMVsockCIDReleaseReceipt(foreign, receipt); err == nil {
		t.Fatal("foreign owner accepted release receipt")
	}
}

func TestAgentMicroVMVsockCIDRequestDigestIsCausalAndIdempotent(t *testing.T) {
	request := validAgentMicroVMVsockCIDReservationRequest(t)
	first, err := AgentMicroVMVsockCIDReservationRequestDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := AgentMicroVMVsockCIDReservationRequestDigest(request)
	if err != nil || second != first {
		t.Fatalf("digest drifted: %q != %q (%v)", second, first, err)
	}
	mutations := map[string]func(*AgentMicroVMVsockCIDReservationRequest){
		"owner":       func(candidate *AgentMicroVMVsockCIDReservationRequest) { candidate.OwnerRef = "launch-identity:other" },
		"attempt":     func(candidate *AgentMicroVMVsockCIDReservationRequest) { candidate.Scope.ExecutionAttempt++ },
		"duration":    func(candidate *AgentMicroVMVsockCIDReservationRequest) { candidate.LeaseDuration++ },
		"idempotency": func(candidate *AgentMicroVMVsockCIDReservationRequest) { candidate.IdempotencyKey += ":other" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			digest, err := AgentMicroVMVsockCIDReservationRequestDigest(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if digest == first {
				t.Fatal("causal mutation preserved request digest")
			}
		})
	}
}
