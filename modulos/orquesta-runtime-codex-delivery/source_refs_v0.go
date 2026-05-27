package orquestaruntimecodexdelivery

import "strings"

func codexDeliveryObservationCoreEvidenceRefsV0(values []string) []string {
	const maxCoreEvidenceRefs = 18
	const maxCoreEvidenceRefLen = 240
	seen := map[string]bool{}
	result := make([]string, 0, maxCoreEvidenceRefs+1)
	truncated := false
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len([]rune(value)) > maxCoreEvidenceRefLen {
			value = string([]rune(value)[:maxCoreEvidenceRefLen]) + "-truncated"
			truncated = true
		}
		if seen[value] {
			continue
		}
		if len(result) >= maxCoreEvidenceRefs {
			truncated = true
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	if truncated && !seen["evidence-ref-codex-delivery-evidence-truncated"] {
		result = append(result, "evidence-ref-codex-delivery-evidence-truncated")
	}
	return result
}

func compactCodexDeliveryRefsV0(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func stringInCodexDeliverySetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexReceiptArtifactRefRegisteredV0(values []string, artifactRef string) bool {
	artifactRef = strings.TrimSpace(artifactRef)
	if artifactRef == "" {
		return false
	}
	for _, value := range values {
		if codexReceiptProjectionArtifactRefV0(value) == artifactRef {
			return true
		}
	}
	return false
}

func codexReceiptProjectionArtifactRefV0(value string) string {
	value = strings.TrimSpace(value)
	artifactRef, _, ok := strings.Cut(value, "#phase:")
	if ok {
		return strings.TrimSpace(artifactRef)
	}
	return value
}
