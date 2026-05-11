package orquestaoutboxdispatch

import (
	"reflect"
	"strings"
	"testing"
)

func TestPlanOutboxDeliveryLeaseClaimV0CreatesOpaqueLease(t *testing.T) {
	result := PlanOutboxDeliveryLeaseClaimV0(OutboxDeliveryLeaseClaimInputV0{
		Claim: validLeaseClaimV0("claim-001", "outbox-001", observedAtV0(), leaseUntilV0()),
	})

	if result.Status != OutboxDeliveryLeaseClaimedV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseClaimedV0)
	}
	if result.Lease.ClaimRef != "claim-001" || result.Lease.MessageID != "outbox-001" {
		t.Fatalf("lease=%+v", result.Lease)
	}
	if result.Lease.Attempt != 1 {
		t.Fatalf("attempt=%d, want 1", result.Lease.Attempt)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("issues=%+v, want none", result.Issues)
	}
}

func TestPlanOutboxDeliveryLeaseClaimV0IdempotentByMessageAndClaimRef(t *testing.T) {
	existing := validDeliveryLeaseV0("claim-002", "outbox-002", observedAtV0(), leaseUntilV0(), 3)

	result := PlanOutboxDeliveryLeaseClaimV0(OutboxDeliveryLeaseClaimInputV0{
		Claim:          validLeaseClaimV0("claim-002", "outbox-002", observedAtV0(), leaseUntilV0()),
		ExistingLeases: []OutboxDeliveryLeaseV0{existing},
	})

	if result.Status != OutboxDeliveryLeaseIdempotentV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseIdempotentV0)
	}
	if result.Lease.Attempt != 3 || result.Lease.ClaimRef != existing.ClaimRef {
		t.Fatalf("lease=%+v, want existing=%+v", result.Lease, existing)
	}
}

func TestPlanOutboxDeliveryLeaseClaimV0RejectsMessageConflict(t *testing.T) {
	existing := validDeliveryLeaseV0("claim-003", "outbox-003", observedAtV0(), leaseUntilV0(), 1)

	result := PlanOutboxDeliveryLeaseClaimV0(OutboxDeliveryLeaseClaimInputV0{
		Claim:          validLeaseClaimV0("claim-other", "outbox-003", observedAtV0(), leaseUntilV0()),
		ExistingLeases: []OutboxDeliveryLeaseV0{existing},
	})

	if result.Status != OutboxDeliveryLeaseConflictV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseConflictV0)
	}
	if len(result.Issues) != 1 || result.Issues[0].Field != "message_id" {
		t.Fatalf("issues=%+v", result.Issues)
	}
}

func TestPlanOutboxDeliveryLeaseClaimV0RejectsClaimRefConflict(t *testing.T) {
	existing := validDeliveryLeaseV0("claim-004", "outbox-004", observedAtV0(), leaseUntilV0(), 1)

	result := PlanOutboxDeliveryLeaseClaimV0(OutboxDeliveryLeaseClaimInputV0{
		Claim:          validLeaseClaimV0("claim-004", "outbox-other", observedAtV0(), leaseUntilV0()),
		ExistingLeases: []OutboxDeliveryLeaseV0{existing},
	})

	if result.Status != OutboxDeliveryLeaseConflictV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseConflictV0)
	}
	if len(result.Issues) != 1 || result.Issues[0].Field != "claim_ref" {
		t.Fatalf("issues=%+v", result.Issues)
	}
}

func TestPlanOutboxDeliveryLeaseClaimV0AllowsNewClaimAfterObservedExpiration(t *testing.T) {
	existing := validDeliveryLeaseV0("claim-005", "outbox-005",
		"2026-05-06T10:00:00Z", "2026-05-06T10:10:00Z", 2)

	result := PlanOutboxDeliveryLeaseClaimV0(OutboxDeliveryLeaseClaimInputV0{
		Claim: validLeaseClaimV0("claim-new", "outbox-005",
			"2026-05-06T10:11:00Z", "2026-05-06T10:20:00Z"),
		ExistingLeases: []OutboxDeliveryLeaseV0{existing},
	})

	if result.Status != OutboxDeliveryLeaseClaimedV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseClaimedV0)
	}
	if result.Lease.Attempt != 3 || result.Lease.ClaimRef != "claim-new" {
		t.Fatalf("lease=%+v", result.Lease)
	}
}

func TestPlanOutboxDeliveryLeaseRenewalV0RejectsExpiredLeaseByObservedNow(t *testing.T) {
	current := validDeliveryLeaseV0("claim-006", "outbox-006",
		"2026-05-06T10:00:00Z", "2026-05-06T10:10:00Z", 1)

	result := PlanOutboxDeliveryLeaseRenewalV0(OutboxDeliveryLeaseRenewalInputV0{
		Current: current,
		Renewal: OutboxDeliveryLeaseRenewalV0{
			ClaimRef:      "claim-006",
			ClaimedByRef:  "dispatcher-alpha",
			NowObservedAt: "2026-05-06T10:10:00Z",
			LeaseUntil:    "2026-05-06T10:30:00Z",
		},
	})

	if result.Status != OutboxDeliveryLeaseExpiredV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseExpiredV0)
	}
	if len(result.Issues) != 1 || result.Issues[0].Field != "now_observed_at" {
		t.Fatalf("issues=%+v", result.Issues)
	}
}

func TestPlanOutboxDeliveryAckByLeaseV0UsesClaimRefOnly(t *testing.T) {
	current := validDeliveryLeaseV0("claim-007", "outbox-007", observedAtV0(), leaseUntilV0(), 1)

	result := PlanOutboxDeliveryAckByLeaseV0(OutboxDeliveryAckByLeaseInputV0{
		Current: current,
		Ack: OutboxDeliveryAckByLeaseV0{
			ClaimRef:      "claim-007",
			ClaimedByRef:  "dispatcher-alpha",
			NowObservedAt: observedAtV0(),
			DispatchRef:   "dispatch-007",
			EvidenceRefs:  []string{" evidence-1 ", "", "evidence-2"},
		},
	})

	if result.Status != OutboxDeliveryLeaseAckedV0 {
		t.Fatalf("status=%s, want %s", result.Status, OutboxDeliveryLeaseAckedV0)
	}
	if result.ClaimRef != "claim-007" || result.DispatchRef != "dispatch-007" {
		t.Fatalf("result=%+v", result)
	}
	if strings.Join(result.EvidenceRefs, ",") != "evidence-1,evidence-2" {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
	if _, ok := reflect.TypeOf(OutboxDeliveryAckByLeaseV0{}).FieldByName("MessageID"); ok {
		t.Fatal("ack by lease must not expose message_id")
	}
}

func TestOutboxDeliveryLeaseContractsUseOnlyOpaqueRefsV0(t *testing.T) {
	for _, value := range []any{
		OutboxDeliveryLeaseV0{},
		OutboxDeliveryLeaseClaimRequestV0{},
		OutboxDeliveryLeaseRenewalV0{},
		OutboxDeliveryLeaseReleaseV0{},
		OutboxDeliveryAckByLeaseV0{},
	} {
		typ := reflect.TypeOf(value)
		for index := 0; index < typ.NumField(); index++ {
			assertNoConcreteInfraFieldV0(t, typ.Field(index).Name)
		}
	}
}

func assertNoConcreteInfraFieldV0(t *testing.T, field string) {
	t.Helper()
	lower := strings.ToLower(field)
	for _, forbidden := range []string{
		"database", "sql", "redis", "backend", "runtime", "provider", "home", "oauth",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("field %q contains concrete infrastructure token %q", field, forbidden)
		}
	}
}

func validLeaseClaimV0(
	claimRef string,
	messageID string,
	nowObservedAt string,
	leaseUntil string,
) OutboxDeliveryLeaseClaimRequestV0 {
	return OutboxDeliveryLeaseClaimRequestV0{
		ClaimRef:      claimRef,
		MessageID:     messageID,
		TargetPort:    "agent_launcher",
		ClaimedByRef:  "dispatcher-alpha",
		NowObservedAt: nowObservedAt,
		LeaseUntil:    leaseUntil,
	}
}

func validDeliveryLeaseV0(
	claimRef string,
	messageID string,
	claimedAt string,
	leaseUntil string,
	attempt int,
) OutboxDeliveryLeaseV0 {
	return OutboxDeliveryLeaseV0{
		ClaimRef:     claimRef,
		MessageID:    messageID,
		TargetPort:   "agent_launcher",
		ClaimedByRef: "dispatcher-alpha",
		ClaimedAt:    claimedAt,
		LeaseUntil:   leaseUntil,
		Attempt:      attempt,
	}
}

func observedAtV0() string {
	return "2026-05-06T10:00:00Z"
}

func leaseUntilV0() string {
	return "2026-05-06T10:15:00Z"
}
