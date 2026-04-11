package controlruntime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ClaudeStatusLive struct {
	Turns          *int64
	Model          string
	PermissionMode string
	LastUsageIn    *int64
	LastUsageOut   *int64
	ConfigPath     string
	RawOutput      string
}

var claudeStatusPattern = regexp.MustCompile(`(?m)^status:\s+turns=(\d+)\s+model=([^\s]+)\s+permission-mode=([^\s]+)\s+output-format=([^\s]+)\s+last-usage=(\d+)\s+in/(\d+)\s+out\s+config=(.+)$`)

func ObserveClaudeStatusLive(obj ObjetivoProceso) (*ClaudeRustObservedArtifacts, error) {
	if !esRuntimeClaudeLocal(obj) {
		return nil, nil
	}
	result, err := RunSlashCommandLive(obj, "/status", SlashCommandOptions{})
	if err != nil {
		return nil, err
	}
	if result == nil || strings.TrimSpace(result.RawOutput) == "" {
		return nil, nil
	}
	status, err := ParseClaudeStatusLive(result.RawOutput)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, nil
	}
	payload := map[string]any{
		"observed_scope":  "status_live",
		"model":           strings.TrimSpace(status.Model),
		"permission_mode": strings.TrimSpace(status.PermissionMode),
		"config_path":     strings.TrimSpace(status.ConfigPath),
		"raw_output":      strings.TrimSpace(status.RawOutput),
	}
	if status.Turns != nil {
		payload["turns"] = *status.Turns
	}
	usage := ClaudeObservedUsage{}
	if status.LastUsageIn != nil {
		usage.InputTokens = *status.LastUsageIn
		payload["input_tokens"] = *status.LastUsageIn
	}
	if status.LastUsageOut != nil {
		usage.OutputTokens = *status.LastUsageOut
	}
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	payload["session_usage"] = map[string]any{
		"turns":          valueInt64OrZero(status.Turns),
		"input_tokens":   usage.InputTokens,
		"output_tokens":  usage.OutputTokens,
		"total_tokens":   usage.TotalTokens,
		"pricing_source": "claude_status_live",
	}
	rawSnapshot := "{}"
	if raw, err := json.Marshal(payload); err == nil {
		rawSnapshot = string(raw)
	}
	return &ClaudeRustObservedArtifacts{
		ObservedAt:    result.FinishedAt,
		AccountSource: "claude_status_live",
		Usage:         usage,
		RawSnapshot:   rawSnapshot,
	}, nil
}

func ParseClaudeStatusLive(raw string) (*ClaudeStatusLive, error) {
	text := normalizeSlashCommandOutput(raw)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	match := claudeStatusPattern.FindStringSubmatch(text)
	if len(match) != 8 {
		return nil, fmt.Errorf("salida de /status de claude no parseable")
	}
	turns, err := parseInt64StatusField(match[1])
	if err != nil {
		return nil, err
	}
	inTokens, err := parseInt64StatusField(match[5])
	if err != nil {
		return nil, err
	}
	outTokens, err := parseInt64StatusField(match[6])
	if err != nil {
		return nil, err
	}
	return &ClaudeStatusLive{
		Turns:          turns,
		Model:          strings.TrimSpace(match[2]),
		PermissionMode: strings.TrimSpace(match[3]),
		LastUsageIn:    inTokens,
		LastUsageOut:   outTokens,
		ConfigPath:     strings.TrimSpace(match[7]),
		RawOutput:      text,
	}, nil
}

func parseInt64StatusField(raw string) (*int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func valueInt64OrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
