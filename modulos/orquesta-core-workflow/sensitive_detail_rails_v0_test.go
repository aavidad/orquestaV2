package orquestacoreworkflow

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func operationalSensitiveFragmentsForTestV0() []string {
	return orquestarails.OperationalSensitiveFragmentsV0
}

func containsForbiddenFragmentV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if hasTokenBoundaryForFragmentV0(lowerValue, fragment, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func hasTokenBoundaryForFragmentV0(value string, fragment string, start int, end int) bool {
	before := true
	if len(fragment) > 0 && isAsciiLetterOrDigitV0(fragment[0]) {
		before = start == 0 || !isAsciiLetterOrDigitV0(value[start-1])
	}
	after := true
	if len(fragment) > 0 && isAsciiLetterOrDigitV0(fragment[len(fragment)-1]) {
		after = end >= len(value) || !isAsciiLetterOrDigitV0(value[end])
	}
	return before && after
}

func isAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}
