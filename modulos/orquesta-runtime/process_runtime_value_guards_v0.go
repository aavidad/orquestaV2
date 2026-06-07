package orquestaruntime

import (
	"strings"
)

func processRuntimeUnsafeValueV0(value string) bool {
	return looksLikeSecret(value) ||
		processRuntimeContainsForbiddenMarkerV0(value) ||
		looksLikeConcreteProviderValueV0(value) ||
		looksLikeConcreteModelValueV0(value) ||
		processRuntimeContainsRuntimePayloadMarkerV0(value)
}

func processRuntimeContainsForbiddenMarkerV0(value string) bool {
	return processRuntimeContainsHomeMarkerV0(value) ||
		processRuntimeContainsCredentialMarkerV0(value)
}

func processRuntimeOperationalPathUnsafeV0(value string) bool {
	return processRuntimeContainsExplicitHomeMarkerV0(value) ||
		processRuntimeContainsCredentialMarkerV0(value)
}

func processRuntimeContainsHomeMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return processRuntimeContainsExplicitHomeMarkerV0(low) ||
		strings.Contains(low, "/home/") ||
		strings.Contains(low, `\users\`) ||
		strings.HasPrefix(low, "~")
}

func processRuntimeContainsExplicitHomeMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "$home") ||
		strings.Contains(low, "${home}") ||
		strings.Contains(low, "%userprofile%") ||
		strings.Contains(low, "%homepath%") ||
		strings.HasPrefix(low, "~")
}

func processRuntimeContainsCredentialMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "oauth") ||
		strings.Contains(low, "token") ||
		strings.Contains(low, "secret") ||
		strings.Contains(low, "api_key") ||
		strings.Contains(low, "credential")
}

func processRuntimeContainsRuntimePayloadMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "://") ||
		strings.Contains(low, "git@") ||
		strings.Contains(low, "prompt") ||
		strings.Contains(low, "transcript")
}
