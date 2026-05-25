package orquestaruntimecodex

import (
	"encoding/json"
	"strings"
)

func CodexUsageAccountingReportRedactedV0(sample string) bool {
	text := strings.TrimSpace(sample)
	if text == "" {
		return false
	}
	var payload any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return false
	}
	return codexUsageJSONRedactedV0(nil, payload)
}

func codexUsageJSONRedactedV0(path []string, value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			part := normalizeCodexUsageJSONKeyV0(key)
			if codexUsageJSONKeySensitiveV0(part) {
				return false
			}
			if !codexUsageJSONRedactedV0(append(path, part), child) {
				return false
			}
		}
	case []any:
		for _, child := range typed {
			if !codexUsageJSONRedactedV0(path, child) {
				return false
			}
		}
	case string:
		if codexUsageJSONStringSensitiveV0(typed) {
			return false
		}
	}
	return true
}

func codexUsageJSONKeySensitiveV0(key string) bool {
	switch key {
	case "prompt", "prompts", "completion", "completions", "transcript", "transcripts",
		"message", "messages", "content", "raw", "payload", "account", "email",
		"home", "home_dir", "path", "token", "access_token", "refresh_token",
		"secret", "credential", "api_key", "provider", "model", "model_name",
		"cost", "cost_micros", "price", "billing", "organization", "org",
		"user", "username", "license", "subscription":
		return true
	default:
		return false
	}
}

func codexUsageJSONStringSensitiveV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "access_token=") ||
		strings.Contains(low, "authorization: bearer") ||
		strings.Contains(low, "bearer ") ||
		strings.Contains(low, "api_key=") ||
		strings.Contains(low, "client_secret=") ||
		strings.Contains(low, "@") ||
		strings.Contains(low, "/users/") ||
		strings.Contains(low, "\\users\\") ||
		strings.Contains(low, "-----begin ")
}
