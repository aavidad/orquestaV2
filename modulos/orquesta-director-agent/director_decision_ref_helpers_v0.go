package orquestadirectoragent

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func directorAgentHasForbiddenDetailV0(values ...string) bool {
	return orquestarails.ValuesContainOperationalSensitiveDetailV0(values)
}

func directorAgentContainsForbiddenFragmentV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	if directorAgentFragmentHasSeparatorV0(fragment) {
		return strings.Contains(lowerValue, fragment)
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if directorAgentHasTokenBoundaryV0(lowerValue, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func directorAgentFragmentHasSeparatorV0(fragment string) bool {
	for i := 0; i < len(fragment); i++ {
		if !directorAgentIsAsciiLetterOrDigitV0(fragment[i]) {
			return true
		}
	}
	return false
}

func directorAgentHasTokenBoundaryV0(value string, start int, end int) bool {
	before := start == 0 || !directorAgentIsAsciiLetterOrDigitV0(value[start-1])
	after := end >= len(value) || !directorAgentIsAsciiLetterOrDigitV0(value[end])
	return before && after
}

func directorAgentIsAsciiLetterOrDigitV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func directorAgentRefCompactV0(value string) bool {
	if len(value) < 2 || len(value) > 512 {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			continue
		}
		if b == '.' || b == '_' || b == ':' || b == '-' {
			continue
		}
		return false
	}
	return true
}

func directorAgentContextRefCompactV0(value string) bool {
	if strings.TrimSpace(value) == "" || len(value) > maxDirectorAgentStringV0 {
		return false
	}
	return !strings.ContainsAny(value, " /\\\t\n\r")
}
