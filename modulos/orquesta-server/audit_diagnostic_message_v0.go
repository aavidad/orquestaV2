package orquestaserver

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func publicAuditDiagnosticMessageV0(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if orquestarails.SecurityModeProgrammingEnabledV0() {
		return message
	}
	lower := strings.ToLower(message)
	for _, forbidden := range []string{"/", "\\", "home", "token", "secret", "prompt", "transcript"} {
		if strings.Contains(lower, forbidden) {
			return "diagnostic_message_redacted"
		}
	}
	return message
}
