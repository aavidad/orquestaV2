package orquestaruntimecodex

import "strings"

func codexTextHasExplicitSoftRailMarkerV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{
		"rail pendiente",
		"rail-pendiente",
		"rail_pendiente",
		"rail blando",
		"rail-blando",
		"rail_blando",
		"soft rail",
		"soft-rail",
		"soft_rail",
		"pending rail",
		"pending-rail",
		"pending_rail",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
