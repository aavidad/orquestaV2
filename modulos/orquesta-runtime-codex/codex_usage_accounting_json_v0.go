package orquestaruntimecodex

import (
	"encoding/json"
	"strings"
)

func codexUsageSnapshotFromJSONV0(sample string) (CodexUsageAccountingSnapshotV0, bool) {
	var payload any
	if err := json.Unmarshal([]byte(sample), &payload); err != nil {
		return CodexUsageAccountingSnapshotV0{}, false
	}
	snapshot := CodexUsageAccountingSnapshotV0{QuotaStatus: CodexUsageQuotaNotConfiguredV0}
	walkCodexUsageJSONV0(&snapshot, nil, payload)
	if snapshot.TotalTokens == 0 && snapshot.PromptTokens+snapshot.CompletionTokens > 0 {
		snapshot.TotalTokens = snapshot.PromptTokens + snapshot.CompletionTokens
	}
	snapshot.Observed = snapshot.QuotaStatus != CodexUsageQuotaNotConfiguredV0 ||
		snapshot.PromptTokens > 0 ||
		snapshot.CompletionTokens > 0 ||
		snapshot.TotalTokens > 0 ||
		snapshot.QuotaRemaining > 0 ||
		snapshot.QuotaLimit > 0
	return snapshot, true
}

func walkCodexUsageJSONV0(snapshot *CodexUsageAccountingSnapshotV0, path []string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			walkCodexUsageJSONV0(snapshot, append(path, normalizeCodexUsageJSONKeyV0(key)), child)
		}
	case []any:
		for _, child := range typed {
			walkCodexUsageJSONV0(snapshot, path, child)
		}
	case string:
		applyCodexUsageJSONStringV0(snapshot, path, typed)
	case float64:
		applyCodexUsageJSONNumberV0(snapshot, path, int64(typed))
	}
}

func applyCodexUsageJSONStringV0(
	snapshot *CodexUsageAccountingSnapshotV0,
	path []string,
	value string,
) {
	key := lastCodexUsageJSONKeyV0(path)
	if key == "quota_status" || (key == "status" && pathHasCodexUsageJSONKeyV0(path, "quota")) {
		snapshot.QuotaStatus = codexUsageQuotaPrecedenceV0(
			snapshot.QuotaStatus,
			normalizeCodexUsageQuotaTextV0(value),
		)
	}
}

func applyCodexUsageJSONNumberV0(
	snapshot *CodexUsageAccountingSnapshotV0,
	path []string,
	value int64,
) {
	if value <= 0 {
		return
	}
	switch lastCodexUsageJSONKeyV0(path) {
	case "prompt_tokens", "input_tokens":
		snapshot.PromptTokens = value
	case "completion_tokens", "output_tokens":
		snapshot.CompletionTokens = value
	case "total_tokens", "tokens_used":
		snapshot.TotalTokens = value
	case "quota_remaining", "remaining", "remaining_tokens", "remaining_messages":
		if pathHasCodexUsageJSONKeyV0(path, "quota") {
			snapshot.QuotaRemaining = value
		}
	case "quota_limit", "limit", "limit_tokens", "limit_messages":
		if pathHasCodexUsageJSONKeyV0(path, "quota") {
			snapshot.QuotaLimit = value
		}
	}
}

func normalizeCodexUsageQuotaTextV0(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case CodexUsageQuotaAvailableV0, "ok", "active":
		return CodexUsageQuotaAvailableV0
	case CodexUsageQuotaLimitedV0, "rate_limited", "throttled":
		return CodexUsageQuotaLimitedV0
	case CodexUsageQuotaExhaustedV0, "exceeded", "depleted":
		return CodexUsageQuotaExhaustedV0
	case CodexUsageQuotaUnknownV0:
		return CodexUsageQuotaUnknownV0
	default:
		return CodexUsageQuotaNotConfiguredV0
	}
}

func normalizeCodexUsageJSONKeyV0(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func lastCodexUsageJSONKeyV0(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[len(path)-1]
}

func pathHasCodexUsageJSONKeyV0(path []string, key string) bool {
	for _, part := range path {
		if part == key {
			return true
		}
	}
	return false
}
