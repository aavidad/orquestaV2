package orquestaoutboxdispatch

import (
	"strings"
	"time"
)

const (
	leaseIssueConflictV0 = "lease_conflict"
	leaseIssueExpiredV0  = "lease_expired"
	leaseIssueNotFoundV0 = "lease_not_found"
)

func PlanOutboxDeliveryLeaseClaimV0(
	input OutboxDeliveryLeaseClaimInputV0,
) OutboxDeliveryLeaseClaimResultV0 {
	claim := normalizeLeaseClaimRequestV0(input.Claim)
	now, leaseUntil, issues := validateLeaseClaimRequestV0(claim)
	if len(issues) > 0 {
		return OutboxDeliveryLeaseClaimResultV0{Status: OutboxDeliveryLeaseInvalidRequestV0, Issues: issues}
	}

	attempt := 1
	for _, existing := range normalizeDeliveryLeasesV0(input.ExistingLeases) {
		if existing.ClaimRef == claim.ClaimRef && existing.MessageID != claim.MessageID {
			return leaseClaimConflictResultV0("claim_ref", "claim_ref asociado a otro message_id")
		}
		if existing.MessageID != claim.MessageID {
			continue
		}
		if existing.ClaimRef == claim.ClaimRef {
			if leaseClaimReplayMatchesV0(existing, claim) {
				return OutboxDeliveryLeaseClaimResultV0{Status: OutboxDeliveryLeaseIdempotentV0, Lease: existing}
			}
			return leaseClaimConflictResultV0("claim_ref", "claim_ref no coincide con el lease vigente")
		}
		if !deliveryLeaseExpiredAtV0(existing, now) {
			return leaseClaimConflictResultV0("message_id", "message_id con lease vigente")
		}
		if existing.Attempt >= attempt {
			attempt = existing.Attempt + 1
		}
	}

	return OutboxDeliveryLeaseClaimResultV0{
		Status: OutboxDeliveryLeaseClaimedV0,
		Lease: OutboxDeliveryLeaseV0{
			ClaimRef:     claim.ClaimRef,
			MessageID:    claim.MessageID,
			TargetPort:   claim.TargetPort,
			ClaimedByRef: claim.ClaimedByRef,
			ClaimedAt:    now.Format(time.RFC3339),
			LeaseUntil:   leaseUntil.Format(time.RFC3339),
			Attempt:      attempt,
		},
	}
}

func PlanOutboxDeliveryLeaseRenewalV0(
	input OutboxDeliveryLeaseRenewalInputV0,
) OutboxDeliveryLeaseRenewalResultV0 {
	renewal := normalizeLeaseRenewalV0(input.Renewal)
	current := normalizeDeliveryLeaseV0(input.Current)
	now, leaseUntil, issues := validateLeaseRenewalV0(renewal)
	if len(issues) > 0 {
		return OutboxDeliveryLeaseRenewalResultV0{Status: OutboxDeliveryLeaseInvalidRequestV0, Issues: issues}
	}
	if issue := validateLeaseRefMatchesV0(current, renewal.ClaimRef, renewal.ClaimedByRef); issue.Code != "" {
		return OutboxDeliveryLeaseRenewalResultV0{Status: statusFromLeaseRefIssueV0(issue), Issues: []DispatchIssueV0{issue}}
	}
	if deliveryLeaseExpiredAtV0(current, now) {
		return OutboxDeliveryLeaseRenewalResultV0{
			Status: OutboxDeliveryLeaseExpiredV0,
			Lease:  current,
			Issues: []DispatchIssueV0{issueV0(leaseIssueExpiredV0, "now_observed_at", "lease expirado")},
		}
	}
	current.LeaseUntil = leaseUntil.Format(time.RFC3339)
	return OutboxDeliveryLeaseRenewalResultV0{Status: OutboxDeliveryLeaseRenewedV0, Lease: current}
}

func PlanOutboxDeliveryLeaseReleaseV0(
	input OutboxDeliveryLeaseReleaseInputV0,
) OutboxDeliveryLeaseReleaseResultV0 {
	release := normalizeLeaseReleaseV0(input.Release)
	current := normalizeDeliveryLeaseV0(input.Current)
	if issues := validateObservedAtV0(release.NowObservedAt); len(issues) > 0 {
		return OutboxDeliveryLeaseReleaseResultV0{Status: OutboxDeliveryLeaseInvalidRequestV0, Issues: issues}
	}
	if issue := validateLeaseRefMatchesV0(current, release.ClaimRef, release.ClaimedByRef); issue.Code != "" {
		return OutboxDeliveryLeaseReleaseResultV0{Status: statusFromLeaseRefIssueV0(issue), Issues: []DispatchIssueV0{issue}}
	}
	return OutboxDeliveryLeaseReleaseResultV0{
		Status:     OutboxDeliveryLeaseReleasedV0,
		ClaimRef:   release.ClaimRef,
		ReleasedAt: release.NowObservedAt,
	}
}

func PlanOutboxDeliveryAckByLeaseV0(
	input OutboxDeliveryAckByLeaseInputV0,
) OutboxDeliveryAckByLeaseResultV0 {
	ack := normalizeAckByLeaseV0(input.Ack)
	current := normalizeDeliveryLeaseV0(input.Current)
	now, issues := validateAckByLeaseV0(ack)
	if len(issues) > 0 {
		return OutboxDeliveryAckByLeaseResultV0{Status: OutboxDeliveryLeaseInvalidRequestV0, Issues: issues}
	}
	if issue := validateLeaseRefMatchesV0(current, ack.ClaimRef, ack.ClaimedByRef); issue.Code != "" {
		return OutboxDeliveryAckByLeaseResultV0{Status: statusFromLeaseRefIssueV0(issue), Issues: []DispatchIssueV0{issue}}
	}
	if deliveryLeaseExpiredAtV0(current, now) {
		return OutboxDeliveryAckByLeaseResultV0{
			Status: OutboxDeliveryLeaseExpiredV0,
			Issues: []DispatchIssueV0{issueV0(leaseIssueExpiredV0, "now_observed_at", "lease expirado")},
		}
	}
	return OutboxDeliveryAckByLeaseResultV0{
		Status:       OutboxDeliveryLeaseAckedV0,
		ClaimRef:     ack.ClaimRef,
		AckedAt:      now.Format(time.RFC3339),
		DispatchRef:  ack.DispatchRef,
		EvidenceRefs: compactStringsV0(ack.EvidenceRefs),
	}
}

func validateLeaseClaimRequestV0(
	claim OutboxDeliveryLeaseClaimRequestV0,
) (time.Time, time.Time, []DispatchIssueV0) {
	var issues []DispatchIssueV0
	requireLeaseFieldV0(&issues, claim.ClaimRef, "claim_ref")
	requireLeaseFieldV0(&issues, claim.MessageID, "message_id")
	requireLeaseFieldV0(&issues, claim.TargetPort, "target_port")
	requireLeaseFieldV0(&issues, claim.ClaimedByRef, "claimed_by_ref")
	now, timeIssues := parseObservedAtV0(claim.NowObservedAt)
	issues = append(issues, timeIssues...)
	leaseUntil, untilIssues := parseLeaseUntilV0(claim.LeaseUntil)
	issues = append(issues, untilIssues...)
	if len(issues) == 0 && !leaseUntil.After(now) {
		issues = append(issues, issueV0(issueInvalidRequestV0, "lease_until", "lease_until debe ser posterior"))
	}
	return now, leaseUntil, issues
}

func validateLeaseRenewalV0(
	renewal OutboxDeliveryLeaseRenewalV0,
) (time.Time, time.Time, []DispatchIssueV0) {
	var issues []DispatchIssueV0
	requireLeaseFieldV0(&issues, renewal.ClaimRef, "claim_ref")
	requireLeaseFieldV0(&issues, renewal.ClaimedByRef, "claimed_by_ref")
	now, timeIssues := parseObservedAtV0(renewal.NowObservedAt)
	issues = append(issues, timeIssues...)
	leaseUntil, untilIssues := parseLeaseUntilV0(renewal.LeaseUntil)
	issues = append(issues, untilIssues...)
	if len(issues) == 0 && !leaseUntil.After(now) {
		issues = append(issues, issueV0(issueInvalidRequestV0, "lease_until", "lease_until debe ser posterior"))
	}
	return now, leaseUntil, issues
}

func validateAckByLeaseV0(ack OutboxDeliveryAckByLeaseV0) (time.Time, []DispatchIssueV0) {
	var issues []DispatchIssueV0
	requireLeaseFieldV0(&issues, ack.ClaimRef, "claim_ref")
	requireLeaseFieldV0(&issues, ack.ClaimedByRef, "claimed_by_ref")
	requireLeaseFieldV0(&issues, ack.DispatchRef, "dispatch_ref")
	now, timeIssues := parseObservedAtV0(ack.NowObservedAt)
	issues = append(issues, timeIssues...)
	return now, issues
}

func validateObservedAtV0(value string) []DispatchIssueV0 {
	_, issues := parseObservedAtV0(value)
	return issues
}

func validateLeaseRefMatchesV0(
	current OutboxDeliveryLeaseV0,
	claimRef string,
	claimedByRef string,
) DispatchIssueV0 {
	if current.ClaimRef == "" {
		return issueV0(leaseIssueNotFoundV0, "claim_ref", "claim_ref no encontrado")
	}
	if current.ClaimRef != claimRef {
		return issueV0(leaseIssueConflictV0, "claim_ref", "claim_ref no coincide")
	}
	if current.ClaimedByRef != claimedByRef {
		return issueV0(leaseIssueConflictV0, "claimed_by_ref", "claimed_by_ref no coincide")
	}
	return DispatchIssueV0{}
}

func statusFromLeaseRefIssueV0(issue DispatchIssueV0) OutboxDeliveryLeaseOperationStatusV0 {
	if issue.Code == leaseIssueNotFoundV0 {
		return OutboxDeliveryLeaseNotFoundV0
	}
	return OutboxDeliveryLeaseConflictV0
}

func leaseClaimConflictResultV0(field string, message string) OutboxDeliveryLeaseClaimResultV0 {
	return OutboxDeliveryLeaseClaimResultV0{
		Status: OutboxDeliveryLeaseConflictV0,
		Issues: []DispatchIssueV0{issueV0(leaseIssueConflictV0, field, message)},
	}
}

func leaseClaimReplayMatchesV0(
	lease OutboxDeliveryLeaseV0,
	claim OutboxDeliveryLeaseClaimRequestV0,
) bool {
	return lease.TargetPort == claim.TargetPort && lease.ClaimedByRef == claim.ClaimedByRef
}

func deliveryLeaseExpiredAtV0(lease OutboxDeliveryLeaseV0, now time.Time) bool {
	leaseUntil, err := time.Parse(time.RFC3339, lease.LeaseUntil)
	if err != nil {
		return true
	}
	return !leaseUntil.After(now)
}

func parseObservedAtV0(value string) (time.Time, []DispatchIssueV0) {
	return parseLeaseTimeV0(value, "now_observed_at")
}

func parseLeaseUntilV0(value string) (time.Time, []DispatchIssueV0) {
	return parseLeaseTimeV0(value, "lease_until")
}

func parseLeaseTimeV0(value string, field string) (time.Time, []DispatchIssueV0) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, []DispatchIssueV0{issueV0(issueInvalidRequestV0, field, field+" requerido")}
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, []DispatchIssueV0{issueV0(issueInvalidRequestV0, field, field+" invalido")}
	}
	return parsed, nil
}

func requireLeaseFieldV0(issues *[]DispatchIssueV0, value string, field string) {
	if strings.TrimSpace(value) == "" {
		*issues = append(*issues, issueV0(issueInvalidRequestV0, field, field+" requerido"))
	}
}
