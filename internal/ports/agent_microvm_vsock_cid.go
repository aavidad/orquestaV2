package ports

import (
	"context"
	"errors"
	"time"
)

const (
	AgentMicroVMVsockCIDLeaseSchema   = "orquesta.agent-microvm-vsock-cid-lease.v1"
	AgentMicroVMVsockCIDReleaseSchema = "orquesta.agent-microvm-vsock-cid-release.v1"
)

// AgentMicroVMVsockCIDReservationRequest asks for one exclusive guest CID.
// OwnerRef identifies the launch transaction that owns the reservation. A
// retry must preserve IdempotencyKey and every semantic field.
type AgentMicroVMVsockCIDReservationRequest struct {
	PoolRef        string
	Scope          AgentMicroVMNetworkScope
	OwnerRef       string
	LeaseDuration  time.Duration
	IdempotencyKey string
}

// AgentMicroVMVsockCIDLease is evidence returned by the allocator, not
// authority by possession. Consumers that can expose a vsock backend must
// recover the current lease from the allocator and compare fencing token and
// revision immediately before the effect.
type AgentMicroVMVsockCIDLease struct {
	PoolRef         string
	ScopeDigest     string
	ExecutionRef    string
	AgentRef        string
	OwnerRef        string
	RequestDigest   string
	IdempotencyKey  string
	GuestCID        uint32
	FencingToken    uint64
	Revision        uint64
	AcquiredAt      time.Time
	ExpiresAt       time.Time
	LeaseRef        string
	VsockBackendRef string
	AdapterRef      string
	ReceiptRef      string
}

// AgentMicroVMVsockCIDRenewalRequest extends an active lease. ExpectedRevision
// prevents a stale owner view from overwriting a newer renewal.
type AgentMicroVMVsockCIDRenewalRequest struct {
	PoolRef          string
	ScopeDigest      string
	LeaseRef         string
	OwnerRef         string
	FencingToken     uint64
	ExpectedRevision uint64
	LeaseDuration    time.Duration
	IdempotencyKey   string
}

type AgentMicroVMVsockCIDRecoveryRequest struct {
	PoolRef      string
	ScopeDigest  string
	LeaseRef     string
	OwnerRef     string
	FencingToken uint64
}

type AgentMicroVMVsockCIDReleaseRequest struct {
	PoolRef          string
	ScopeDigest      string
	LeaseRef         string
	OwnerRef         string
	FencingToken     uint64
	ExpectedRevision uint64
	IdempotencyKey   string
}

type AgentMicroVMVsockCIDReleaseReceipt struct {
	PoolRef        string
	ScopeDigest    string
	LeaseRef       string
	OwnerRef       string
	GuestCID       uint32
	FencingToken   uint64
	FinalRevision  uint64
	ReleasedAt     time.Time
	IdempotencyKey string
	AdapterRef     string
	ReceiptRef     string
}

// AgentMicroVMVsockCIDAllocator owns the durable uniqueness and fencing
// boundary. Reserve, Renew and Release are atomic and idempotent. Recover is a
// read of current authority; it never revives an expired or released lease.
type AgentMicroVMVsockCIDAllocator interface {
	Reserve(context.Context, AgentMicroVMVsockCIDReservationRequest) (AgentMicroVMVsockCIDLease, error)
	Renew(context.Context, AgentMicroVMVsockCIDRenewalRequest) (AgentMicroVMVsockCIDLease, error)
	Recover(context.Context, AgentMicroVMVsockCIDRecoveryRequest) (AgentMicroVMVsockCIDLease, error)
	Release(context.Context, AgentMicroVMVsockCIDReleaseRequest) (AgentMicroVMVsockCIDReleaseReceipt, error)
}

type AgentMicroVMVsockCIDContractError struct {
	Code string
}

func (err *AgentMicroVMVsockCIDContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentMicroVMVsockCIDContractErrorCode(err error) string {
	var contractErr *AgentMicroVMVsockCIDContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateAgentMicroVMVsockCIDReservationRequest(
	request AgentMicroVMVsockCIDReservationRequest,
) error {
	if !validWorkspaceLogicalRef(request.PoolRef) ||
		!validAgentMicroVMNetworkScope(request.Scope) ||
		!validWorkspaceLogicalRef(request.OwnerRef) ||
		request.LeaseDuration <= 0 ||
		!validWorkspaceLogicalRef(request.IdempotencyKey) {
		return agentMicroVMVsockCIDError("reservation_request_invalid")
	}
	return nil
}

func AgentMicroVMVsockCIDReservationRequestDigest(
	request AgentMicroVMVsockCIDReservationRequest,
) (string, error) {
	if err := ValidateAgentMicroVMVsockCIDReservationRequest(request); err != nil {
		return "", err
	}
	document := struct {
		Schema           string                             `json:"schema"`
		PoolRef          string                             `json:"pool_ref"`
		Scope            agentMicroVMLaunchIdentityDocument `json:"scope"`
		OwnerRef         string                             `json:"owner_ref"`
		LeaseNanoseconds int64                              `json:"lease_nanoseconds"`
		IdempotencyKey   string                             `json:"idempotency_key"`
	}{
		Schema: AgentMicroVMVsockCIDLeaseSchema, PoolRef: request.PoolRef,
		Scope: agentMicroVMLaunchScopeDocument(request.Scope), OwnerRef: request.OwnerRef,
		LeaseNanoseconds: int64(request.LeaseDuration), IdempotencyKey: request.IdempotencyKey,
	}
	return agentMicroVMNetworkDocumentDigest(document), nil
}

func AgentMicroVMNetworkScopeDigest(scope AgentMicroVMNetworkScope) (string, error) {
	if !validAgentMicroVMNetworkScope(scope) {
		return "", agentMicroVMVsockCIDError("scope_invalid")
	}
	return agentMicroVMNetworkDocumentDigest(agentMicroVMLaunchScopeDocument(scope)), nil
}

func ValidateAgentMicroVMVsockCIDLease(lease AgentMicroVMVsockCIDLease) error {
	if !validWorkspaceLogicalRef(lease.PoolRef) ||
		!validWorkspaceDigest(lease.ScopeDigest) ||
		!validWorkspaceLogicalRef(lease.ExecutionRef) ||
		!validWorkspaceLogicalRef(lease.AgentRef) ||
		!validWorkspaceLogicalRef(lease.OwnerRef) ||
		!validWorkspaceDigest(lease.RequestDigest) ||
		!validWorkspaceLogicalRef(lease.IdempotencyKey) ||
		lease.GuestCID < AgentMicroVMMinimumGuestCID ||
		lease.GuestCID == AgentMicroVMReservedAnyCID ||
		lease.FencingToken == 0 || lease.Revision == 0 ||
		lease.AcquiredAt.IsZero() || !lease.ExpiresAt.After(lease.AcquiredAt) ||
		!validWorkspaceLogicalRef(lease.AdapterRef) {
		return agentMicroVMVsockCIDError("lease_invalid")
	}
	if lease.LeaseRef != AgentMicroVMVsockCIDLeaseRef(lease) ||
		lease.VsockBackendRef != AgentMicroVMVsockBackendRef(lease.LeaseRef) ||
		lease.ReceiptRef != AgentMicroVMVsockCIDLeaseReceiptRef(lease) {
		return agentMicroVMVsockCIDError("lease_binding_invalid")
	}
	return nil
}

func AgentMicroVMVsockCIDLeaseRef(lease AgentMicroVMVsockCIDLease) string {
	document := struct {
		Schema        string `json:"schema"`
		PoolRef       string `json:"pool_ref"`
		ScopeDigest   string `json:"scope_digest"`
		OwnerRef      string `json:"owner_ref"`
		RequestDigest string `json:"request_digest"`
		GuestCID      uint32 `json:"guest_cid"`
		FencingToken  uint64 `json:"fencing_token"`
		AcquiredAt    string `json:"acquired_at"`
	}{
		Schema: AgentMicroVMVsockCIDLeaseSchema, PoolRef: lease.PoolRef,
		ScopeDigest: lease.ScopeDigest, OwnerRef: lease.OwnerRef,
		RequestDigest: lease.RequestDigest, GuestCID: lease.GuestCID,
		FencingToken: lease.FencingToken, AcquiredAt: lease.AcquiredAt.UTC().Format(time.RFC3339Nano),
	}
	return "vsock-cid-lease:" + agentMicroVMNetworkDocumentDigest(document)
}

func AgentMicroVMVsockBackendRef(leaseRef string) string {
	document := struct {
		Schema   string `json:"schema"`
		LeaseRef string `json:"lease_ref"`
	}{
		Schema: "orquesta.agent-microvm-vsock-backend.v1", LeaseRef: leaseRef,
	}
	return "vsock-backend:" + agentMicroVMNetworkDocumentDigest(document)
}

func AgentMicroVMVsockCIDLeaseReceiptRef(lease AgentMicroVMVsockCIDLease) string {
	document := struct {
		Schema     string `json:"schema"`
		LeaseRef   string `json:"lease_ref"`
		Revision   uint64 `json:"revision"`
		ExpiresAt  string `json:"expires_at"`
		AdapterRef string `json:"adapter_ref"`
	}{
		Schema: AgentMicroVMVsockCIDLeaseSchema, LeaseRef: lease.LeaseRef,
		Revision: lease.Revision, ExpiresAt: lease.ExpiresAt.UTC().Format(time.RFC3339Nano),
		AdapterRef: lease.AdapterRef,
	}
	return "vsock-cid-lease-receipt:" + agentMicroVMNetworkDocumentDigest(document)
}

func ValidateAgentMicroVMVsockCIDRenewalRequest(request AgentMicroVMVsockCIDRenewalRequest) error {
	if !validCIDLeaseOperation(
		request.PoolRef, request.ScopeDigest, request.LeaseRef, request.OwnerRef,
		request.FencingToken, request.ExpectedRevision, request.IdempotencyKey,
	) || request.LeaseDuration <= 0 {
		return agentMicroVMVsockCIDError("renewal_request_invalid")
	}
	return nil
}

func AgentMicroVMVsockCIDRenewalRequestDigest(request AgentMicroVMVsockCIDRenewalRequest) (string, error) {
	if err := ValidateAgentMicroVMVsockCIDRenewalRequest(request); err != nil {
		return "", err
	}
	return agentMicroVMVsockCIDOperationDigest("renew", request.PoolRef, request.ScopeDigest,
		request.LeaseRef, request.OwnerRef, request.FencingToken, request.ExpectedRevision,
		int64(request.LeaseDuration), request.IdempotencyKey), nil
}

func ValidateAgentMicroVMVsockCIDRecoveryRequest(request AgentMicroVMVsockCIDRecoveryRequest) error {
	if !validWorkspaceLogicalRef(request.PoolRef) ||
		!validWorkspaceDigest(request.ScopeDigest) ||
		!validWorkspaceLogicalRef(request.LeaseRef) ||
		!validWorkspaceLogicalRef(request.OwnerRef) ||
		request.FencingToken == 0 {
		return agentMicroVMVsockCIDError("recovery_request_invalid")
	}
	return nil
}

func ValidateAgentMicroVMVsockCIDReleaseRequest(request AgentMicroVMVsockCIDReleaseRequest) error {
	if !validCIDLeaseOperation(
		request.PoolRef, request.ScopeDigest, request.LeaseRef, request.OwnerRef,
		request.FencingToken, request.ExpectedRevision, request.IdempotencyKey,
	) {
		return agentMicroVMVsockCIDError("release_request_invalid")
	}
	return nil
}

func AgentMicroVMVsockCIDReleaseRequestDigest(request AgentMicroVMVsockCIDReleaseRequest) (string, error) {
	if err := ValidateAgentMicroVMVsockCIDReleaseRequest(request); err != nil {
		return "", err
	}
	return agentMicroVMVsockCIDOperationDigest("release", request.PoolRef, request.ScopeDigest,
		request.LeaseRef, request.OwnerRef, request.FencingToken, request.ExpectedRevision,
		0, request.IdempotencyKey), nil
}

func ValidateAgentMicroVMVsockCIDReleaseReceipt(
	request AgentMicroVMVsockCIDReleaseRequest,
	receipt AgentMicroVMVsockCIDReleaseReceipt,
) error {
	if err := ValidateAgentMicroVMVsockCIDReleaseRequest(request); err != nil {
		return err
	}
	if receipt.PoolRef != request.PoolRef || receipt.ScopeDigest != request.ScopeDigest ||
		receipt.LeaseRef != request.LeaseRef || receipt.OwnerRef != request.OwnerRef ||
		receipt.FencingToken != request.FencingToken ||
		receipt.FinalRevision != request.ExpectedRevision ||
		receipt.GuestCID < AgentMicroVMMinimumGuestCID ||
		receipt.GuestCID == AgentMicroVMReservedAnyCID ||
		receipt.ReleasedAt.IsZero() ||
		receipt.IdempotencyKey != request.IdempotencyKey ||
		!validWorkspaceLogicalRef(receipt.AdapterRef) ||
		receipt.ReceiptRef != AgentMicroVMVsockCIDReleaseReceiptRef(receipt) {
		return agentMicroVMVsockCIDError("release_receipt_invalid")
	}
	return nil
}

func AgentMicroVMVsockCIDReleaseReceiptRef(receipt AgentMicroVMVsockCIDReleaseReceipt) string {
	document := struct {
		Schema         string `json:"schema"`
		LeaseRef       string `json:"lease_ref"`
		FencingToken   uint64 `json:"fencing_token"`
		FinalRevision  uint64 `json:"final_revision"`
		ReleasedAt     string `json:"released_at"`
		IdempotencyKey string `json:"idempotency_key"`
		AdapterRef     string `json:"adapter_ref"`
	}{
		Schema: AgentMicroVMVsockCIDReleaseSchema, LeaseRef: receipt.LeaseRef,
		FencingToken: receipt.FencingToken, FinalRevision: receipt.FinalRevision,
		ReleasedAt:     receipt.ReleasedAt.UTC().Format(time.RFC3339Nano),
		IdempotencyKey: receipt.IdempotencyKey, AdapterRef: receipt.AdapterRef,
	}
	return "vsock-cid-release-receipt:" + agentMicroVMNetworkDocumentDigest(document)
}

func validCIDLeaseOperation(
	poolRef string,
	scopeDigest string,
	leaseRef string,
	ownerRef string,
	fencingToken uint64,
	revision uint64,
	idempotencyKey string,
) bool {
	return validWorkspaceLogicalRef(poolRef) &&
		validWorkspaceDigest(scopeDigest) &&
		validWorkspaceLogicalRef(leaseRef) &&
		validWorkspaceLogicalRef(ownerRef) &&
		fencingToken > 0 && revision > 0 &&
		validWorkspaceLogicalRef(idempotencyKey)
}

func agentMicroVMVsockCIDOperationDigest(
	kind string,
	poolRef string,
	scopeDigest string,
	leaseRef string,
	ownerRef string,
	fencingToken uint64,
	revision uint64,
	leaseNanoseconds int64,
	idempotencyKey string,
) string {
	document := struct {
		Schema           string `json:"schema"`
		Kind             string `json:"kind"`
		PoolRef          string `json:"pool_ref"`
		ScopeDigest      string `json:"scope_digest"`
		LeaseRef         string `json:"lease_ref"`
		OwnerRef         string `json:"owner_ref"`
		FencingToken     uint64 `json:"fencing_token"`
		ExpectedRevision uint64 `json:"expected_revision"`
		LeaseNanoseconds int64  `json:"lease_nanoseconds,omitempty"`
		IdempotencyKey   string `json:"idempotency_key"`
	}{
		Schema: AgentMicroVMVsockCIDLeaseSchema, Kind: kind, PoolRef: poolRef,
		ScopeDigest: scopeDigest, LeaseRef: leaseRef, OwnerRef: ownerRef,
		FencingToken: fencingToken, ExpectedRevision: revision,
		LeaseNanoseconds: leaseNanoseconds, IdempotencyKey: idempotencyKey,
	}
	return agentMicroVMNetworkDocumentDigest(document)
}

func agentMicroVMVsockCIDError(suffix string) error {
	return &AgentMicroVMVsockCIDContractError{Code: "agent_microvm_vsock_cid." + suffix}
}
