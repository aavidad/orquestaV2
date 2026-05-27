package orquestaruntimecodex

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	CodexUsageQuotaUnknownV0       = "unknown"
	CodexUsageQuotaNotConfiguredV0 = "not_configured"
	CodexUsageQuotaAvailableV0     = "available"
	CodexUsageQuotaLimitedV0       = "limited"
	CodexUsageQuotaExhaustedV0     = "exhausted"

	CodexUsageQuotaReasonNotReportedV0         = "quota_not_reported_in_redacted_usage_report"
	CodexUsageQuotaReasonUnknownReportedV0     = "quota_status_unknown_in_redacted_usage_report"
	CodexUsageQuotaReasonObservedUnavailableV0 = "quota_observed_unavailable"
)

type CodexUsageAccountingSnapshotV0 struct {
	Observed         bool
	QuotaStatus      string
	QuotaRemaining   int64
	QuotaLimit       int64
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	QuotaReason      string
}

func BuildCodexUsageAccountingSnapshotV0(
	samples []string,
) CodexUsageAccountingSnapshotV0 {
	snapshot := CodexUsageAccountingSnapshotV0{
		QuotaStatus: CodexUsageQuotaNotConfiguredV0,
	}
	for _, sample := range samples {
		mergeCodexUsageSnapshotV0(&snapshot, parseCodexUsageSampleV0(sample))
	}
	if snapshot.TotalTokens == 0 && snapshot.PromptTokens+snapshot.CompletionTokens > 0 {
		snapshot.TotalTokens = snapshot.PromptTokens + snapshot.CompletionTokens
	}
	finalizeCodexUsageAccountingSnapshotV0(&snapshot)
	return snapshot
}

func parseCodexUsageSampleV0(sample string) CodexUsageAccountingSnapshotV0 {
	text := strings.TrimSpace(sample)
	if text == "" {
		return CodexUsageAccountingSnapshotV0{QuotaStatus: CodexUsageQuotaNotConfiguredV0}
	}
	if snapshot, ok := codexUsageSnapshotFromJSONV0(text); ok {
		return snapshot
	}
	snapshot := CodexUsageAccountingSnapshotV0{
		QuotaStatus: codexUsageQuotaStatusFromTextV0(text),
	}
	if snapshot.QuotaStatus == CodexUsageQuotaUnknownV0 {
		snapshot.QuotaReason = codexUsageUnknownReasonFromTextV0(text)
	}
	snapshot.PromptTokens = codexUsageFirstIntV0(text, codexUsagePromptTokenPatternsV0())
	snapshot.CompletionTokens = codexUsageFirstIntV0(text, codexUsageCompletionTokenPatternsV0())
	snapshot.TotalTokens = codexUsageFirstIntV0(text, codexUsageTotalTokenPatternsV0())
	snapshot.QuotaRemaining = codexUsageFirstIntV0(text, codexUsageQuotaRemainingPatternsV0())
	snapshot.QuotaLimit = codexUsageFirstIntV0(text, codexUsageQuotaLimitPatternsV0())
	snapshot.Observed = snapshot.QuotaStatus != CodexUsageQuotaNotConfiguredV0 ||
		snapshot.PromptTokens > 0 ||
		snapshot.CompletionTokens > 0 ||
		snapshot.TotalTokens > 0 ||
		snapshot.QuotaRemaining > 0 ||
		snapshot.QuotaLimit > 0
	finalizeCodexUsageAccountingSnapshotV0(&snapshot)
	return snapshot
}

func finalizeCodexUsageAccountingSnapshotV0(snapshot *CodexUsageAccountingSnapshotV0) {
	if snapshot == nil || !snapshot.Observed {
		return
	}
	if snapshot.QuotaStatus == "" || snapshot.QuotaStatus == CodexUsageQuotaNotConfiguredV0 {
		snapshot.QuotaStatus = CodexUsageQuotaUnknownV0
		snapshot.QuotaReason = CodexUsageQuotaReasonNotReportedV0
		return
	}
	if snapshot.QuotaStatus == CodexUsageQuotaUnknownV0 && strings.TrimSpace(snapshot.QuotaReason) == "" {
		snapshot.QuotaReason = CodexUsageQuotaReasonNotReportedV0
		return
	}
	if snapshot.QuotaStatus == CodexUsageQuotaAvailableV0 ||
		snapshot.QuotaStatus == CodexUsageQuotaLimitedV0 ||
		snapshot.QuotaStatus == CodexUsageQuotaExhaustedV0 {
		snapshot.QuotaReason = ""
	}
}

func mergeCodexUsageSnapshotV0(
	current *CodexUsageAccountingSnapshotV0,
	next CodexUsageAccountingSnapshotV0,
) {
	if current == nil || !next.Observed {
		return
	}
	current.Observed = true
	current.QuotaStatus = codexUsageQuotaPrecedenceV0(current.QuotaStatus, next.QuotaStatus)
	if next.PromptTokens > 0 {
		current.PromptTokens = next.PromptTokens
	}
	if next.CompletionTokens > 0 {
		current.CompletionTokens = next.CompletionTokens
	}
	if next.TotalTokens > 0 {
		current.TotalTokens = next.TotalTokens
	}
	if strings.TrimSpace(next.QuotaReason) != "" {
		current.QuotaReason = strings.TrimSpace(next.QuotaReason)
	}
	if next.QuotaRemaining > 0 {
		current.QuotaRemaining = next.QuotaRemaining
	}
	if next.QuotaLimit > 0 {
		current.QuotaLimit = next.QuotaLimit
	}
}

func codexUsageQuotaStatusFromTextV0(text string) string {
	low := strings.ToLower(text)
	for _, pattern := range []struct {
		needle string
		status string
	}{
		{"quota status: exhausted", CodexUsageQuotaExhaustedV0},
		{"quota_status\":\"exhausted", CodexUsageQuotaExhaustedV0},
		{"quota_status\": \"exhausted", CodexUsageQuotaExhaustedV0},
		{"usage limit reached", CodexUsageQuotaExhaustedV0},
		{"quota exceeded", CodexUsageQuotaExhaustedV0},
		{"rate limit exceeded", CodexUsageQuotaLimitedV0},
		{"quota status: limited", CodexUsageQuotaLimitedV0},
		{"quota_status\":\"limited", CodexUsageQuotaLimitedV0},
		{"quota_status\": \"limited", CodexUsageQuotaLimitedV0},
		{"rate limit", CodexUsageQuotaLimitedV0},
		{"quota status: available", CodexUsageQuotaAvailableV0},
		{"quota_status\":\"available", CodexUsageQuotaAvailableV0},
		{"quota_status\": \"available", CodexUsageQuotaAvailableV0},
		{"quota status: unknown", CodexUsageQuotaUnknownV0},
		{"quota_status\":\"unknown", CodexUsageQuotaUnknownV0},
		{"quota_status\": \"unknown", CodexUsageQuotaUnknownV0},
		{"quota status: unavailable", CodexUsageQuotaUnknownV0},
		{"quota_status\":\"unavailable", CodexUsageQuotaUnknownV0},
		{"quota_status\": \"unavailable", CodexUsageQuotaUnknownV0},
	} {
		if strings.Contains(low, pattern.needle) {
			return pattern.status
		}
	}
	return CodexUsageQuotaNotConfiguredV0
}

func codexUsageUnknownReasonFromTextV0(text string) string {
	if strings.Contains(strings.ToLower(text), "unavailable") {
		return CodexUsageQuotaReasonObservedUnavailableV0
	}
	return CodexUsageQuotaReasonUnknownReportedV0
}

func codexUsageQuotaPrecedenceV0(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if next == "" || next == CodexUsageQuotaNotConfiguredV0 {
		return current
	}
	if current == "" || current == CodexUsageQuotaNotConfiguredV0 {
		return next
	}
	if current == CodexUsageQuotaExhaustedV0 || next == CodexUsageQuotaExhaustedV0 {
		return CodexUsageQuotaExhaustedV0
	}
	if current == CodexUsageQuotaLimitedV0 || next == CodexUsageQuotaLimitedV0 {
		return CodexUsageQuotaLimitedV0
	}
	if current == CodexUsageQuotaAvailableV0 || next == CodexUsageQuotaAvailableV0 {
		return CodexUsageQuotaAvailableV0
	}
	return CodexUsageQuotaUnknownV0
}

func codexUsageFirstIntV0(text string, patterns []*regexp.Regexp) int64 {
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		if value := codexUsageParseIntV0(matches[len(matches)-1]); value > 0 {
			return value
		}
	}
	return 0
}

func codexUsageParseIntV0(raw string) int64 {
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, raw)
	value, err := strconv.ParseInt(clean, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func codexUsagePromptTokenPatternsV0() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)"prompt_tokens"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)"input_tokens"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)\b(prompt|input)\s+tokens?\b\D{0,32}([0-9][0-9,._ ]*)`),
	}
}

func codexUsageCompletionTokenPatternsV0() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)"completion_tokens"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)"output_tokens"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)\b(completion|output)\s+tokens?\b\D{0,32}([0-9][0-9,._ ]*)`),
	}
}

func codexUsageTotalTokenPatternsV0() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)"total_tokens"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)\btotal\s+tokens?\b\D{0,32}([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)\btokens?\s+used\b\D{0,32}([0-9][0-9,._ ]*)`),
	}
}

func codexUsageQuotaRemainingPatternsV0() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)"quota_remaining"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)\bquota\s+remaining\b\D{0,32}([0-9][0-9,._ ]*)`),
	}
}

func codexUsageQuotaLimitPatternsV0() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)"quota_limit"\s*:\s*([0-9][0-9,._ ]*)`),
		regexp.MustCompile(`(?i)\bquota\s+limit\b\D{0,32}([0-9][0-9,._ ]*)`),
	}
}
