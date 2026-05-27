package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func autoprogrammingPromotionGuardianReceiptV0(
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
	effect orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
	guarded serverAutoprogrammingPromotionGuardianResultV0,
) orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptV0 {
	status := autoprogrammingPromotionGuardianReceiptStatusV0(effect, guarded)
	attemptRef := autoprogrammingPromotionGuardianAttemptRefV0(command, effect, guarded)
	receipt := orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptV0{
		SchemaVersion:         orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptSchemaVersionV0,
		Status:                status,
		PromotionRef:          firstNonEmptyV0(guarded.PromotionRef, command.PromotionRef, effect.PromotionRef),
		RunRef:                firstNonEmptyV0(guarded.RunRef, command.RunRef, effect.RunRef),
		ProjectRef:            firstNonEmptyV0(guarded.ProjectRef, command.ProjectRef, effect.ProjectRef),
		AppRef:                guarded.AppRef,
		RepoRef:               guarded.RepoRef,
		WorktreeRef:           firstNonEmptyV0(guarded.WorktreeRef, command.WorktreeRef, effect.WorktreeRef),
		BranchRef:             firstNonEmptyV0(guarded.BranchRef, command.BranchRef, effect.BranchRef),
		GuardianAttemptRef:    attemptRef,
		GuardianResultStatus:  strings.TrimSpace(guarded.Status),
		GuardianResultPhase:   strings.TrimSpace(guarded.Phase),
		GuardianPromote:       guarded.Promote,
		GuardianPromoted:      guarded.Promoted,
		GuardianRestored:      guarded.Restored,
		ManifestRef:           strings.TrimSpace(guarded.ManifestRef),
		RepairPacketRef:       strings.TrimSpace(guarded.RepairPacketRef),
		CandidateHash:         strings.TrimSpace(guarded.CandidateHash),
		CandidateSizeBytes:    guarded.CandidateSizeBytes,
		StagingEffectStatus:   strings.TrimSpace(effect.Status),
		StagingCommitRef:      strings.TrimSpace(effect.CommitRef),
		StagingCommitShortRef: strings.TrimSpace(effect.CommitShortRef),
		StagingChangedPaths:   compactStringsV0(effect.ChangedPaths),
		Retryable:             effect.Retryable || !autoprogrammingPromotionGuardianReceiptPromotedV0(status),
		ReasonCodes:           compactStringsV0(guarded.ReasonCodes),
		EvidenceRefs:          compactStringsV0(append(append([]string(nil), effect.EvidenceRefs...), guarded.EvidenceRefs...)),
	}
	receipt.ReceiptRef = autoprogrammingPromotionGuardianReceiptRefV0(receipt)
	return receipt
}

func autoprogrammingPromotionGuardianBlockedEffectV0(
	effect orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
	guarded serverAutoprogrammingPromotionGuardianResultV0,
	err error,
) orquestaautoprogramming.AutoprogrammingStagingEffectResultV0 {
	receipt := autoprogrammingPromotionGuardianReceiptV0(command, effect, guarded)
	effect.Status = orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0
	effect.Retryable = true
	effect.EvidenceRefs = append(effect.EvidenceRefs, guarded.EvidenceRefs...)
	effect.EvidenceRefs = append(effect.EvidenceRefs, receipt.ReceiptRef)
	effect.PromotionGuardianReceipts = append(effect.PromotionGuardianReceipts, receipt)
	effect.Issues = append(effect.Issues, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
		Code:    autoprogrammingPromotionGuardianErrorCodeV0(err),
		Field:   "guardian",
		Message: err.Error(),
	})
	return effect
}

func autoprogrammingPromotionEffectWithGuardianV0(
	effect orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
	guarded serverAutoprogrammingPromotionGuardianResultV0,
) orquestaautoprogramming.AutoprogrammingStagingEffectResultV0 {
	receipt := autoprogrammingPromotionGuardianReceiptV0(command, effect, guarded)
	effect.EvidenceRefs = append(effect.EvidenceRefs, guarded.EvidenceRefs...)
	effect.EvidenceRefs = append(effect.EvidenceRefs, receipt.ReceiptRef)
	effect.PromotionGuardianReceipts = append(effect.PromotionGuardianReceipts, receipt)
	if receipt.Status == orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidateVerifiedV0 {
		effect.Status = orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0
		effect.Retryable = true
		effect.Issues = append(effect.Issues, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
			Code:    "guardian_candidate_verified_without_promotion",
			Field:   "guardian",
			Message: "guardian verified candidate without binary promotion",
		})
	}
	return effect
}

func autoprogrammingPromotionGuardianReceiptStatusV0(
	effect orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
	guarded serverAutoprogrammingPromotionGuardianResultV0,
) string {
	switch {
	case strings.TrimSpace(guarded.Status) == "candidate_promoted" && guarded.Promoted:
		return orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidatePromotedV0
	case strings.TrimSpace(guarded.Status) == "candidate_promoted_breakglass" && guarded.Promoted:
		return orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidatePromotedBreakglassV0
	case strings.TrimSpace(guarded.Status) == "candidate_promoted" && guarded.VerifiedOnly:
		return orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidateVerifiedV0
	case strings.TrimSpace(guarded.Status) == "last_good_restored" || guarded.Restored:
		return orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptLastGoodRestoredV0
	case strings.TrimSpace(effect.Status) == orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0 ||
		strings.TrimSpace(guarded.Status) == "candidate_failed" ||
		strings.TrimSpace(guarded.Status) == "promotion_incomplete" ||
		strings.TrimSpace(guarded.Status) == "last_good_unverified" ||
		strings.TrimSpace(guarded.Status) == "guardian_promotion_lease_busy" ||
		strings.TrimSpace(guarded.Status) == "guardian_promotion_lease_lost":
		return orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptPromotionBlockedV0
	default:
		return orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptResultInvalidV0
	}
}

func autoprogrammingPromotionGuardianReceiptPromotedV0(status string) bool {
	return status == orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidatePromotedV0 ||
		status == orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidatePromotedBreakglassV0
}

func autoprogrammingPromotionGuardianAttemptRefV0(
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
	effect orquestaautoprogramming.AutoprogrammingStagingEffectResultV0,
	guarded serverAutoprogrammingPromotionGuardianResultV0,
) string {
	return "guardian-attempt-ref-" + autoprogrammingPromotionGuardianHashV0(strings.Join([]string{
		command.PromotionRef,
		command.RunRef,
		command.WorktreeRef,
		command.BranchRef,
		effect.Status,
		effect.CommitRef,
		guarded.Status,
		guarded.Phase,
		guarded.ManifestRef,
		guarded.CandidateHash,
	}, "|"))
}

func autoprogrammingPromotionGuardianReceiptRefV0(
	receipt orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptV0,
) string {
	return "promotion-guardian-receipt-ref-" + autoprogrammingPromotionGuardianHashV0(strings.Join([]string{
		receipt.PromotionRef,
		receipt.RunRef,
		receipt.WorktreeRef,
		receipt.BranchRef,
		receipt.GuardianAttemptRef,
		receipt.Status,
		receipt.StagingEffectStatus,
		receipt.CandidateHash,
	}, "|"))
}

func autoprogrammingPromotionGuardianHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:16]
}
