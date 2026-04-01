package controlruntime

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ClaudeObservedUsage struct {
	InputTokens              int64
	OutputTokens             int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
	TotalTokens              int64
	EstimatedCostUSD         *float64
	MessageCount             int
	Turns                    int
	UpdatedAt                *time.Time
}

type ClaudeRustObservedArtifacts struct {
	ObservedAt     time.Time
	SessionPath    string
	AccountEmail   string
	AccountUser    string
	AccountSource  string
	OAuthExpiresAt *time.Time
	Usage          ClaudeObservedUsage
	RawSnapshot    string
}

func ObserveClaudeRustArtifacts(obj ObjetivoProceso) (*ClaudeRustObservedArtifacts, error) {
	if estado, observed, err := ConsultarEstadoLocal(obj); err != nil {
		return nil, err
	} else if observed && estado != nil && strings.TrimSpace(estado.MetadataJSON) != "" {
		obj.MetadataJSON = estado.MetadataJSON
	}
	if !esRuntimeClaudeLocal(obj) {
		return nil, nil
	}
	meta := metadataMap(obj.MetadataJSON)
	now := time.Now().UTC()
	workingDir := strings.TrimSpace(stringValueFromMetadata(meta, "working_dir"))
	credentialsPath := claudeCredentialsPath(meta)
	sessionPath, usage, sessionUpdatedAt, err := readClaudeLatestSessionUsage(workingDir)
	if err != nil {
		return nil, err
	}
	authRaw, email, user, oauthExpiresAt, source := readClaudeOAuthIdentity(credentialsPath)
	if strings.TrimSpace(authRaw) == "" && strings.TrimSpace(sessionPath) == "" {
		return nil, nil
	}
	artifacts := &ClaudeRustObservedArtifacts{
		ObservedAt:     now,
		SessionPath:    strings.TrimSpace(sessionPath),
		AccountEmail:   strings.TrimSpace(email),
		AccountUser:    strings.TrimSpace(user),
		AccountSource:  firstNonEmptyString(strings.TrimSpace(source), "claude_rust_credentials"),
		OAuthExpiresAt: oauthExpiresAt,
		Usage:          usage,
	}
	payload := map[string]any{
		"observed_scope": "claude_rust_session",
	}
	if credentialsPath != "" {
		payload["credentials_path"] = credentialsPath
	}
	if artifacts.AccountEmail != "" {
		payload["account_email"] = artifacts.AccountEmail
	}
	if artifacts.AccountUser != "" {
		payload["account_user"] = artifacts.AccountUser
	}
	if artifacts.AccountSource != "" {
		payload["account_source"] = artifacts.AccountSource
	}
	if artifacts.OAuthExpiresAt != nil && !artifacts.OAuthExpiresAt.IsZero() {
		payload["oauth"] = map[string]any{
			"expires_at": artifacts.OAuthExpiresAt.UTC().Format(time.RFC3339Nano),
		}
	}
	if strings.TrimSpace(sessionPath) != "" {
		sessionPayload := map[string]any{
			"session_path": sessionPath,
			"message_count": usage.MessageCount,
			"turns": usage.Turns,
			"input_tokens": usage.InputTokens,
			"output_tokens": usage.OutputTokens,
			"cache_creation_input_tokens": usage.CacheCreationInputTokens,
			"cache_read_input_tokens": usage.CacheReadInputTokens,
			"total_tokens": usage.TotalTokens,
			"pricing_source": "rust_default_sonnet",
		}
		if usage.UpdatedAt != nil && !usage.UpdatedAt.IsZero() {
			sessionPayload["updated_at"] = usage.UpdatedAt.UTC().Format(time.RFC3339Nano)
		} else if sessionUpdatedAt != nil && !sessionUpdatedAt.IsZero() {
			sessionPayload["updated_at"] = sessionUpdatedAt.UTC().Format(time.RFC3339Nano)
		}
		if usage.EstimatedCostUSD != nil {
			sessionPayload["estimated_cost_usd"] = *usage.EstimatedCostUSD
		}
		payload["session_usage"] = sessionPayload
	}
	if authRaw != "" {
		payload["auth_snapshot"] = jsonMap(authRaw)
	}
	if raw, err := json.Marshal(payload); err == nil {
		artifacts.RawSnapshot = string(raw)
	}
	return artifacts, nil
}

func esRuntimeClaudeLocal(obj ObjetivoProceso) bool {
	meta := metadataMap(obj.MetadataJSON)
	for _, raw := range []string{
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
		stringValueFromMetadata(meta, "herramienta"),
		stringValueFromMetadata(meta, "conector"),
		stringValueFromMetadata(meta, "connector"),
	} {
		lower := strings.ToLower(strings.TrimSpace(raw))
		if lower == "" {
			continue
		}
		if strings.Contains(lower, "codex") {
			continue
		}
		if strings.Contains(lower, "claude") || strings.Contains(lower, "anthropic") || strings.Contains(lower, "rusty-claude") {
			return true
		}
	}
	if workdir := strings.TrimSpace(stringValueFromMetadata(meta, "working_dir")); workdir != "" {
		if _, err := os.Stat(filepath.Join(workdir, ".claude", "sessions")); err == nil {
			return true
		}
	}
	return false
}

func claudeCredentialsPath(meta map[string]any) string {
	if home := strings.TrimSpace(stringValueFromMetadata(meta, "claude_config_home")); home != "" {
		return filepath.Join(home, "credentials.json")
	}
	if home := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_HOME")); home != "" {
		return filepath.Join(home, "credentials.json")
	}
	if home := userHomeDir(); home != "" {
		return filepath.Join(home, ".claude", "credentials.json")
	}
	return ""
}

func readClaudeOAuthIdentity(path string) (string, string, string, *time.Time, string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", "", nil, ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", nil, ""
	}
	data := jsonMap(string(raw))
	if data == nil {
		return "", "", "", nil, ""
	}
	oauth, _ := data["oauth"].(map[string]any)
	root := data
	if oauth != nil {
		root = oauth
	}
	email := strings.TrimSpace(recursiveString(root,
		"email",
		"account_email",
		"user_email",
		"login_email",
	))
	user := strings.TrimSpace(recursiveString(root,
		"name",
		"preferred_username",
		"username",
		"user_name",
		"account_user",
	))
	if claims := tokenClaimsFromClaudeCredentials(root); claims != nil {
		if email == "" {
			email = strings.TrimSpace(recursiveString(claims, "email"))
		}
		if user == "" {
			user = strings.TrimSpace(recursiveString(claims, "name", "preferred_username"))
		}
	}
	var expiresAt *time.Time
	for _, candidate := range []any{root["expires_at"], data["expires_at"]} {
		if ts := parseClaudeObservedTime(candidate); ts != nil {
			expiresAt = ts
			break
		}
	}
	return string(raw), email, user, expiresAt, "claude_rust_credentials"
}

func tokenClaimsFromClaudeCredentials(data map[string]any) map[string]any {
	for _, key := range []string{"id_token", "access_token"} {
		raw := strings.TrimSpace(stringValueFromMetadata(data, key))
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ".")
		if len(parts) != 3 {
			continue
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}
		var claims map[string]any
		if err := json.Unmarshal(payload, &claims); err != nil {
			continue
		}
		return claims
	}
	return nil
}

func readClaudeLatestSessionUsage(workingDir string) (string, ClaudeObservedUsage, *time.Time, error) {
	var empty ClaudeObservedUsage
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return "", empty, nil, nil
	}
	sessionsDir := filepath.Join(workingDir, ".claude", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", empty, nil, nil
		}
		return "", empty, nil, err
	}
	var bestPath string
	var bestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", empty, nil, err
		}
		modified := info.ModTime().UTC()
		if bestPath == "" || modified.After(bestTime) {
			bestPath = filepath.Join(sessionsDir, entry.Name())
			bestTime = modified
		}
	}
	if bestPath == "" {
		return "", empty, nil, nil
	}
	raw, err := os.ReadFile(bestPath)
	if err != nil {
		return "", empty, nil, err
	}
	root := jsonMap(string(raw))
	if root == nil {
		return "", empty, nil, nil
	}
	messages, _ := root["messages"].([]any)
	var usage ClaudeObservedUsage
	usage.MessageCount = len(messages)
	var latestUsageAt *time.Time
	for _, item := range messages {
		msg, _ := item.(map[string]any)
		if msg == nil {
			continue
		}
		if _, ok := msg["usage"]; ok {
			usage.Turns++
		}
		usage.InputTokens += int64(intFromAny(recursiveAny(msg, "usage", "input_tokens")))
		usage.OutputTokens += int64(intFromAny(recursiveAny(msg, "usage", "output_tokens")))
		usage.CacheCreationInputTokens += int64(intFromAny(recursiveAny(msg, "usage", "cache_creation_input_tokens")))
		usage.CacheReadInputTokens += int64(intFromAny(recursiveAny(msg, "usage", "cache_read_input_tokens")))
		if ts := parseClaudeObservedTime(recursiveAny(msg, "updated_at")); ts != nil {
			if latestUsageAt == nil || ts.After(*latestUsageAt) {
				tmp := ts.UTC()
				latestUsageAt = &tmp
			}
		}
	}
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
	if usage.TotalTokens > 0 {
		estimated := estimateClaudeUsageCostUSD(usage)
		usage.EstimatedCostUSD = &estimated
	}
	if latestUsageAt == nil && !bestTime.IsZero() {
		tmp := bestTime.UTC()
		latestUsageAt = &tmp
	}
	usage.UpdatedAt = latestUsageAt
	return bestPath, usage, latestUsageAt, nil
}

func recursiveAny(raw any, path ...string) any {
	current := raw
	for _, key := range path {
		value, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = value[key]
	}
	return current
}

func intFromAny(raw any) int {
	switch v := raw.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		out, err := v.Int64()
		if err == nil {
			return int(out)
		}
	case string:
		out, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return out
		}
	}
	return 0
}

func parseClaudeObservedTime(raw any) *time.Time {
	switch v := raw.(type) {
	case nil:
		return nil
	case float64:
		if v <= 0 {
			return nil
		}
		ts := time.Unix(int64(v), 0).UTC()
		return &ts
	case int64:
		if v <= 0 {
			return nil
		}
		ts := time.Unix(v, 0).UTC()
		return &ts
	case json.Number:
		n, err := v.Int64()
		if err == nil && n > 0 {
			ts := time.Unix(n, 0).UTC()
			return &ts
		}
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			ts := time.Unix(n, 0).UTC()
			return &ts
		}
		if ts := firstParsedTime(v); !ts.IsZero() {
			tmp := ts.UTC()
			return &tmp
		}
	}
	return nil
}

func estimateClaudeUsageCostUSD(usage ClaudeObservedUsage) float64 {
	const (
		inputCostPerMillion         = 15.0
		outputCostPerMillion        = 75.0
		cacheCreationCostPerMillion = 18.75
		cacheReadCostPerMillion     = 1.5
	)
	total := 0.0
	total += float64(usage.InputTokens) / 1_000_000.0 * inputCostPerMillion
	total += float64(usage.OutputTokens) / 1_000_000.0 * outputCostPerMillion
	total += float64(usage.CacheCreationInputTokens) / 1_000_000.0 * cacheCreationCostPerMillion
	total += float64(usage.CacheReadInputTokens) / 1_000_000.0 * cacheReadCostPerMillion
	return total
}
