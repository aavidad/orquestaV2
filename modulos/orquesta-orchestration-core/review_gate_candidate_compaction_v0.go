package orquestacionnucleoapp

import "strings"

const (
	reviewGateObservationSummaryMaxRunesV0     = 280
	reviewGateObservationEvidenceRefMaxRunesV0 = 180
	reviewGateObservationEvidenceRefsMaxV0     = 12

	reviewGateObservationCompactedEvidenceRefV0 = "evidence-ref-review-gate-payload-compacted"
)

func compactReviewGateObservationPayloadV0(
	observation ReviewGateObservationV0,
) ReviewGateObservationV0 {
	compacted := false
	observation.Summary, compacted = compactReviewGateSummaryV0(observation.Summary, compacted)
	observation.EvidenceRefs, compacted = compactReviewGateEvidenceRefsV0(
		observation.EvidenceRefs,
		compacted,
	)
	if compacted {
		observation.EvidenceRefs = compactStringsV0(append(
			observation.EvidenceRefs,
			reviewGateObservationCompactedEvidenceRefV0,
		))
	}
	return observation
}

func compactReviewGateSummaryV0(value string, compacted bool) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Revision compacta de entrega registrada.", compacted
	}
	if len([]rune(value)) <= reviewGateObservationSummaryMaxRunesV0 {
		return value, compacted
	}
	return strings.TrimSpace(string([]rune(value)[:reviewGateObservationSummaryMaxRunesV0])), true
}

func compactReviewGateEvidenceRefsV0(values []string, compacted bool) ([]string, bool) {
	out := make([]string, 0, reviewGateObservationEvidenceRefsMaxV0)
	for _, value := range compactStringsV0(values) {
		ref, ok := compactReviewGateEvidenceRefV0(value)
		if !ok {
			compacted = true
			continue
		}
		if len(out) >= reviewGateObservationEvidenceRefsMaxV0 {
			compacted = true
			continue
		}
		out = append(out, ref)
	}
	return out, compacted
}

func compactReviewGateEvidenceRefV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\t") {
		return "", false
	}
	if len([]rune(value)) > reviewGateObservationEvidenceRefMaxRunesV0 {
		return "", false
	}
	return value, true
}
