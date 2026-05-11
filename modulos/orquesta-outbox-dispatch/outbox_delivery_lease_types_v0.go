package orquestaoutboxdispatch

type OutboxDeliveryLeaseOperationStatusV0 string

const (
	OutboxDeliveryLeaseClaimedV0        OutboxDeliveryLeaseOperationStatusV0 = "claimed"
	OutboxDeliveryLeaseRenewedV0        OutboxDeliveryLeaseOperationStatusV0 = "renewed"
	OutboxDeliveryLeaseReleasedV0       OutboxDeliveryLeaseOperationStatusV0 = "released"
	OutboxDeliveryLeaseAckedV0          OutboxDeliveryLeaseOperationStatusV0 = "acked"
	OutboxDeliveryLeaseIdempotentV0     OutboxDeliveryLeaseOperationStatusV0 = "idempotent"
	OutboxDeliveryLeaseConflictV0       OutboxDeliveryLeaseOperationStatusV0 = "conflict"
	OutboxDeliveryLeaseExpiredV0        OutboxDeliveryLeaseOperationStatusV0 = "expired"
	OutboxDeliveryLeaseNotFoundV0       OutboxDeliveryLeaseOperationStatusV0 = "not_found"
	OutboxDeliveryLeaseInvalidRequestV0 OutboxDeliveryLeaseOperationStatusV0 = "invalid_request"
)

type OutboxDeliveryLeaseV0 struct {
	ClaimRef     string
	MessageID    string
	TargetPort   string
	ClaimedByRef string
	ClaimedAt    string
	LeaseUntil   string
	Attempt      int
}

type OutboxDeliveryLeaseClaimRequestV0 struct {
	ClaimRef      string
	MessageID     string
	TargetPort    string
	ClaimedByRef  string
	NowObservedAt string
	LeaseUntil    string
}

type OutboxDeliveryLeaseClaimInputV0 struct {
	Claim          OutboxDeliveryLeaseClaimRequestV0
	ExistingLeases []OutboxDeliveryLeaseV0
}

type OutboxDeliveryLeaseClaimResultV0 struct {
	Status OutboxDeliveryLeaseOperationStatusV0
	Lease  OutboxDeliveryLeaseV0
	Issues []DispatchIssueV0
}

type OutboxDeliveryLeaseRenewalV0 struct {
	ClaimRef      string
	ClaimedByRef  string
	NowObservedAt string
	LeaseUntil    string
}

type OutboxDeliveryLeaseRenewalInputV0 struct {
	Renewal OutboxDeliveryLeaseRenewalV0
	Current OutboxDeliveryLeaseV0
}

type OutboxDeliveryLeaseRenewalResultV0 struct {
	Status OutboxDeliveryLeaseOperationStatusV0
	Lease  OutboxDeliveryLeaseV0
	Issues []DispatchIssueV0
}

type OutboxDeliveryLeaseReleaseV0 struct {
	ClaimRef      string
	ClaimedByRef  string
	NowObservedAt string
}

type OutboxDeliveryLeaseReleaseInputV0 struct {
	Release OutboxDeliveryLeaseReleaseV0
	Current OutboxDeliveryLeaseV0
}

type OutboxDeliveryLeaseReleaseResultV0 struct {
	Status     OutboxDeliveryLeaseOperationStatusV0
	ClaimRef   string
	ReleasedAt string
	Issues     []DispatchIssueV0
}

type OutboxDeliveryAckByLeaseV0 struct {
	ClaimRef      string
	ClaimedByRef  string
	NowObservedAt string
	DispatchRef   string
	EvidenceRefs  []string
}

type OutboxDeliveryAckByLeaseInputV0 struct {
	Ack     OutboxDeliveryAckByLeaseV0
	Current OutboxDeliveryLeaseV0
}

type OutboxDeliveryAckByLeaseResultV0 struct {
	Status       OutboxDeliveryLeaseOperationStatusV0
	ClaimRef     string
	AckedAt      string
	DispatchRef  string
	EvidenceRefs []string
	Issues       []DispatchIssueV0
}

type OutboxDeliveryLeaseClaimPortV0 interface {
	ClaimOutboxDeliveryLeaseV0(
		claim OutboxDeliveryLeaseClaimRequestV0,
	) (OutboxDeliveryLeaseClaimResultV0, []DispatchIssueV0)
}

type OutboxDeliveryLeaseRenewalPortV0 interface {
	RenewOutboxDeliveryLeaseV0(
		renewal OutboxDeliveryLeaseRenewalV0,
	) (OutboxDeliveryLeaseRenewalResultV0, []DispatchIssueV0)
}

type OutboxDeliveryLeaseReleasePortV0 interface {
	ReleaseOutboxDeliveryLeaseV0(
		release OutboxDeliveryLeaseReleaseV0,
	) (OutboxDeliveryLeaseReleaseResultV0, []DispatchIssueV0)
}

type OutboxDeliveryAckByLeasePortV0 interface {
	AckOutboxDeliveryByLeaseV0(
		ack OutboxDeliveryAckByLeaseV0,
	) (OutboxDeliveryAckByLeaseResultV0, []DispatchIssueV0)
}
