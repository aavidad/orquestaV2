package orquestaruntimecodex

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func codexTextContainsSensitiveDetailIgnoringRailEnvV0(value string) bool {
	lower := strings.ToLower(value)
	for _, fragment := range orquestarails.OperationalSensitiveFragmentsV0 {
		if orquestarails.ContainsFragmentWithBoundaryV0(lower, fragment) {
			return true
		}
	}
	return false
}
