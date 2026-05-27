package main

import (
	"context"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestAutoprogrammingPromotionGuardianPropagaLeaseBusyV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"guardian_promotion_lease_busy","phase":"lease","redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","counters":{"evidence_refs":1},"evidence_refs":["evidence-ref-guardian-promotion-lease-busy"],"lease":{"schema_version":"orquesta_guardian_promotion_lease.v0","lease_ref":"lease-ref-1","owner_ref":"promotion-ref-1","operation":"check-promote","phase":"lease","status":"guardian_promotion_lease_busy","freshness":"deadline_controlled","deadline_at":"2026-05-27T10:30:00Z","payload_hash":"hash","state_ref":"guardian-state-ref-1","reason_code":"guardian_promotion_lease_busy"}}'; exit 1`,
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "guardian_promotion_lease_busy") ||
		result.Status != "guardian_promotion_lease_busy" ||
		len(result.EvidenceRefs) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianLeaseBusyBloqueaRetryableV0(t *testing.T) {
	effect := autoprogrammingPromotionGuardianBlockedEffectV0(
		orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
			Status: "blocked",
		},
		orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: "promotion-ref-lease-busy",
		},
		serverAutoprogrammingPromotionGuardianResultV0{
			Status:       "guardian_promotion_lease_busy",
			Phase:        "lease",
			EvidenceRefs: []string{"evidence-ref-guardian-promotion-lease-busy"},
		},
		assertErrV0("guardian_promotion_lease_busy"),
	)
	if !effect.Retryable ||
		len(effect.PromotionGuardianReceipts) != 1 ||
		effect.PromotionGuardianReceipts[0].Status !=
			orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptPromotionBlockedV0 ||
		effect.Issues[0].Code != "guardian_promotion_lease_busy" {
		t.Fatalf("effect=%+v", effect)
	}
}

type assertErrV0 string

func (err assertErrV0) Error() string { return string(err) }
