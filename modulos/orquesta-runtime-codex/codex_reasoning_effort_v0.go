package orquestaruntimecodex

import "strings"

func codexEffectiveReasoningEffortV0(profileEffort string, packetCapacity string) string {
	profileEffort = strings.TrimSpace(profileEffort)
	packetCapacity = strings.TrimSpace(packetCapacity)
	if profileEffort == "" {
		profileEffort = "medium"
	}
	if packetCapacity == "xhigh" {
		return "xhigh"
	}
	return profileEffort
}
