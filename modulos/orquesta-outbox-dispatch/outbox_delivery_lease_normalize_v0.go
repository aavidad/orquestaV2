package orquestaoutboxdispatch

import "strings"

func normalizeDeliveryLeasesV0(leases []OutboxDeliveryLeaseV0) []OutboxDeliveryLeaseV0 {
	if len(leases) == 0 {
		return nil
	}
	normalized := make([]OutboxDeliveryLeaseV0, 0, len(leases))
	for _, lease := range leases {
		normalized = append(normalized, normalizeDeliveryLeaseV0(lease))
	}
	return normalized
}

func normalizeDeliveryLeaseV0(lease OutboxDeliveryLeaseV0) OutboxDeliveryLeaseV0 {
	return OutboxDeliveryLeaseV0{
		ClaimRef:     strings.TrimSpace(lease.ClaimRef),
		MessageID:    strings.TrimSpace(lease.MessageID),
		TargetPort:   strings.TrimSpace(lease.TargetPort),
		ClaimedByRef: strings.TrimSpace(lease.ClaimedByRef),
		ClaimedAt:    strings.TrimSpace(lease.ClaimedAt),
		LeaseUntil:   strings.TrimSpace(lease.LeaseUntil),
		Attempt:      lease.Attempt,
	}
}

func normalizeLeaseClaimRequestV0(
	claim OutboxDeliveryLeaseClaimRequestV0,
) OutboxDeliveryLeaseClaimRequestV0 {
	return OutboxDeliveryLeaseClaimRequestV0{
		ClaimRef:      strings.TrimSpace(claim.ClaimRef),
		MessageID:     strings.TrimSpace(claim.MessageID),
		TargetPort:    strings.TrimSpace(claim.TargetPort),
		ClaimedByRef:  strings.TrimSpace(claim.ClaimedByRef),
		NowObservedAt: strings.TrimSpace(claim.NowObservedAt),
		LeaseUntil:    strings.TrimSpace(claim.LeaseUntil),
	}
}

func normalizeLeaseRenewalV0(renewal OutboxDeliveryLeaseRenewalV0) OutboxDeliveryLeaseRenewalV0 {
	return OutboxDeliveryLeaseRenewalV0{
		ClaimRef:      strings.TrimSpace(renewal.ClaimRef),
		ClaimedByRef:  strings.TrimSpace(renewal.ClaimedByRef),
		NowObservedAt: strings.TrimSpace(renewal.NowObservedAt),
		LeaseUntil:    strings.TrimSpace(renewal.LeaseUntil),
	}
}

func normalizeLeaseReleaseV0(release OutboxDeliveryLeaseReleaseV0) OutboxDeliveryLeaseReleaseV0 {
	return OutboxDeliveryLeaseReleaseV0{
		ClaimRef:      strings.TrimSpace(release.ClaimRef),
		ClaimedByRef:  strings.TrimSpace(release.ClaimedByRef),
		NowObservedAt: strings.TrimSpace(release.NowObservedAt),
	}
}

func normalizeAckByLeaseV0(ack OutboxDeliveryAckByLeaseV0) OutboxDeliveryAckByLeaseV0 {
	return OutboxDeliveryAckByLeaseV0{
		ClaimRef:      strings.TrimSpace(ack.ClaimRef),
		ClaimedByRef:  strings.TrimSpace(ack.ClaimedByRef),
		NowObservedAt: strings.TrimSpace(ack.NowObservedAt),
		DispatchRef:   strings.TrimSpace(ack.DispatchRef),
		EvidenceRefs:  compactStringsV0(ack.EvidenceRefs),
	}
}
