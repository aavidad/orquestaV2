package main

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestAutoprogrammingPromotionGuardianConsumePromocionBreakglassV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"candidate_promoted_breakglass","promoted":true,"redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","config_effective":{"promote":true,"skip_health":true,"repair_codex":false,"force_after_timeout":false,"shutdown_forced":false,"health_timeout_ms":1000,"command_timeout_ms":1000,"shutdown_timeout_ms":1000,"command_output_max_bytes":1024,"artifact_max_bytes":1024,"shutdown_queue_limit":1,"skip_health_evidence_refs":["evidence-ref-skip-health"]},"reason_codes":["candidate_built","candidate_tests_passed","candidate_not_live_checked","guardian_skip_health_breakglass_authorized"],"counters":{"evidence_refs":2},"evidence_refs":["evidence-ref-skip-health","evidence-ref-guardian-candidate-not-live-checked"]}'`,
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err != nil ||
		result.Status != "candidate_promoted_breakglass" ||
		!result.BreakglassPromoted ||
		!containsStringV0(result.ReasonCodes, "candidate_not_live_checked") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianReceiptPromocionBreakglassV0(t *testing.T) {
	effect := autoprogrammingPromotionEffectWithGuardianV0(
		orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
			Status:       orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
			PromotionRef: "promotion-ref-guardian-breakglass",
			EvidenceRefs: []string{"evidence-ref-staging"},
		},
		orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: "promotion-ref-guardian-breakglass",
			RunRef:       "run-ref-guardian-breakglass",
			WorktreeRef:  "worktree-ref-orquesta-server-idle-self-improvement",
			BranchRef:    "branch-ref-orquesta-server-idle-self-improvement",
		},
		serverAutoprogrammingPromotionGuardianResultV0{
			Status:       "candidate_promoted_breakglass",
			Phase:        "promote",
			Promote:      true,
			Promoted:     true,
			ReasonCodes:  []string{"candidate_built", "candidate_tests_passed", "candidate_not_live_checked"},
			EvidenceRefs: []string{"evidence-ref-skip-health"},
		},
	)
	if effect.Retryable || len(effect.PromotionGuardianReceipts) != 1 {
		t.Fatalf("effect=%+v", effect)
	}
	receipt := effect.PromotionGuardianReceipts[0]
	if receipt.Status != orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidatePromotedBreakglassV0 ||
		!containsStringV0(receipt.ReasonCodes, "candidate_not_live_checked") ||
		!containsStringV0(receipt.EvidenceRefs, "evidence-ref-skip-health") {
		t.Fatalf("receipt=%+v", receipt)
	}
}
