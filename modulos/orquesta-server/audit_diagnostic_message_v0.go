package orquestaserver

import "strings"

func publicAuditDiagnosticMessageV0(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	lower := strings.ToLower(message)
	for _, forbidden := range []string{"/", "\\", "home", "token", "secret", "prompt", "transcript"} {
		if strings.Contains(lower, forbidden) {
			return "diagnostic_message_redacted"
		}
	}
	return message
}
