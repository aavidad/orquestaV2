package orquestaserver

import "strings"

const (
	serverOperationalMessageMaxRunesV0          = 192
	serverOperationalMessageRedactedV0          = "operational_message_redacted"
	serverStartupOperationalMessageRedactedV0   = "startup_message_redacted"
	serverOperationalMessageTruncatedSuffixV0   = "...[truncated]"
	serverOperationalMessageSensitiveTokenRefV0 = "[redacted]"
)

func projectServerStartupMessageV0(message string) string {
	return projectServerOperationalMessageV0("startup", message)
}

func projectServerOperationalMessageV0(scope string, message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if serverOperationalMessageLooksSensitiveV0(message) {
		if strings.TrimSpace(scope) == "startup" {
			return serverStartupOperationalMessageRedactedV0
		}
		return serverOperationalMessageRedactedV0
	}
	return truncateServerOperationalMessageV0(message)
}

func projectServerOperationalReasonFieldV0(key string, value string) string {
	key = strings.TrimSpace(key)
	value = projectServerOperationalMessageV0("", value)
	if key == "" || value == "" {
		return ""
	}
	return key + "=" + value
}

func serverOperationalMessageLooksSensitiveV0(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	for _, fragment := range []string{
		"/",
		"\\",
		"home",
		"token",
		"secret",
		"authorization",
		"bearer ",
		"password",
		"prompt",
		"transcript",
		"runtime",
		".codex",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func truncateServerOperationalMessageV0(message string) string {
	runes := []rune(strings.TrimSpace(message))
	if len(runes) <= serverOperationalMessageMaxRunesV0 {
		return string(runes)
	}
	suffix := []rune(serverOperationalMessageTruncatedSuffixV0)
	limit := serverOperationalMessageMaxRunesV0 - len(suffix)
	if limit < 1 {
		return serverOperationalMessageSensitiveTokenRefV0
	}
	return string(runes[:limit]) + serverOperationalMessageTruncatedSuffixV0
}
