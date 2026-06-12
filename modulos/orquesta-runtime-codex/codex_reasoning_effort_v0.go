package orquestaruntimecodex

import "strings"

func codexEffectiveReasoningEffortV0(profileEffort string, packetCapacity string) string {
	profileEffort = strings.TrimSpace(profileEffort)
	packetCapacity = strings.TrimSpace(packetCapacity)
	if packetCapacity == "xhigh" {
		return "xhigh"
	}
	if packetCapacity == "high" && codexReasoningRankV0(profileEffort) < codexReasoningRankV0("high") {
		return "high"
	}
	return profileEffort
}

func codexReasoningRankV0(value string) int {
	switch strings.TrimSpace(value) {
	case "xhigh":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
