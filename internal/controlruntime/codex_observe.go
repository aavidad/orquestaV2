package controlruntime

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type CodexObservedRateLimit struct {
	UsedPercent   *float64
	WindowMinutes int
	ResetsAt      *time.Time
}

type CodexObservedArtifacts struct {
	ObservationSource string
	Profile           string
	ExternalID        string
	SessionPath       string
	ObservedAt        time.Time
	ObservedScope     string
	AccountID         string
	AccountEmail      string
	AccountUser       string
	AccountSource     string
	Primary           CodexObservedRateLimit
	Secondary         CodexObservedRateLimit
	Credits           *float64
	PlanType          string
	RawSnapshot       string
}

type codexSessionEvent struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexTokenCountPayload struct {
	Type       string          `json:"type"`
	Info       json.RawMessage `json:"info"`
	RateLimits struct {
		Primary struct {
			UsedPercent   *float64 `json:"used_percent"`
			WindowMinutes int      `json:"window_minutes"`
			ResetsAt      any      `json:"resets_at"`
		} `json:"primary"`
		Secondary struct {
			UsedPercent   *float64 `json:"used_percent"`
			WindowMinutes int      `json:"window_minutes"`
			ResetsAt      any      `json:"resets_at"`
		} `json:"secondary"`
		Credits  any    `json:"credits"`
		PlanType string `json:"plan_type"`
	} `json:"rate_limits"`
}

func ObserveCodexArtifacts(obj ObjetivoProceso) (*CodexObservedArtifacts, error) {
	if estado, observed, err := ConsultarEstadoLocal(obj); err != nil {
		return nil, err
	} else if observed && estado != nil && strings.TrimSpace(estado.MetadataJSON) != "" {
		obj.MetadataJSON = estado.MetadataJSON
	}
	meta := metadataMap(obj.MetadataJSON)
	if !esRuntimeCodexLocal(obj) {
		return nil, nil
	}
	rendered := firstNonEmptyString(
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
	)
	startedAt := metadataTime(meta, "started_at")
	externalID := firstNonEmptyString(
		stringValueFromMetadata(meta, "external_session_id"),
	)
	if externalID == "" {
		detected, err := detectCodexSessionID(rendered, strings.TrimSpace(stringValueFromMetadata(meta, "working_dir")), startedAt, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		externalID = strings.TrimSpace(detected)
	}
	sessionPath, observedAt, raw, err := readCodexLatestTokenCount(rendered, externalID, startedAt)
	if err != nil {
		return nil, err
	}
	scope := "session"
	searchAnchor := startedAt
	if !observedAt.IsZero() {
		searchAnchor = observedAt
	}
	if latestPath, latestObservedAt, latestRaw, err := readCodexLatestTokenCount(rendered, "", searchAnchor); err != nil {
		return nil, err
	} else if latestObservedAt.After(observedAt) {
		sessionPath = latestPath
		observedAt = latestObservedAt
		raw = latestRaw
		scope = "profile"
	}
	if strings.TrimSpace(sessionPath) == "" {
		return nil, nil
	}
	artifacts := &CodexObservedArtifacts{
		ObservationSource: "codex_token_count_observed",
		ExternalID:        externalID,
		SessionPath:       sessionPath,
		ObservedAt:        observedAt,
		ObservedScope:     scope,
		Credits:           codexObservedCredits(raw.RateLimits.Credits),
		PlanType:          strings.TrimSpace(raw.RateLimits.PlanType),
	}
	if snapshot, err := json.Marshal(raw); err == nil {
		artifacts.RawSnapshot = string(snapshot)
	}
	artifacts.Primary = CodexObservedRateLimit{
		UsedPercent:   raw.RateLimits.Primary.UsedPercent,
		WindowMinutes: raw.RateLimits.Primary.WindowMinutes,
		ResetsAt:      parseCodexResetAt(raw.RateLimits.Primary.ResetsAt),
	}
	artifacts.Secondary = CodexObservedRateLimit{
		UsedPercent:   raw.RateLimits.Secondary.UsedPercent,
		WindowMinutes: raw.RateLimits.Secondary.WindowMinutes,
		ResetsAt:      parseCodexResetAt(raw.RateLimits.Secondary.ResetsAt),
	}
	artifacts.Profile = perfilDesdeRenderedCommand(rendered)
	authRaw, email, user, accountID, source := readCodexAuthIdentity(rendered, artifacts.Profile)
	if email != "" || accountID != "" {
		accountAnchor := startedAt
		if !artifacts.ObservedAt.IsZero() {
			accountAnchor = artifacts.ObservedAt
		}
		if accountPath, accountObservedAt, accountRaw, accountProfile, err := readCodexLatestTokenCountForAccount(rendered, accountID, email, accountAnchor); err != nil {
			return nil, err
		} else if accountObservedAt.After(observedAt) {
			sessionPath = accountPath
			observedAt = accountObservedAt
			raw = accountRaw
			scope = "account"
			artifacts.SessionPath = sessionPath
			artifacts.ObservedAt = observedAt
			artifacts.ObservedScope = scope
			artifacts.Credits = codexObservedCredits(raw.RateLimits.Credits)
			artifacts.PlanType = strings.TrimSpace(raw.RateLimits.PlanType)
			artifacts.Primary = CodexObservedRateLimit{
				UsedPercent:   raw.RateLimits.Primary.UsedPercent,
				WindowMinutes: raw.RateLimits.Primary.WindowMinutes,
				ResetsAt:      parseCodexResetAt(raw.RateLimits.Primary.ResetsAt),
			}
			artifacts.Secondary = CodexObservedRateLimit{
				UsedPercent:   raw.RateLimits.Secondary.UsedPercent,
				WindowMinutes: raw.RateLimits.Secondary.WindowMinutes,
				ResetsAt:      parseCodexResetAt(raw.RateLimits.Secondary.ResetsAt),
			}
			if snapshot, err := json.Marshal(raw); err == nil {
				artifacts.RawSnapshot = string(snapshot)
			}
			if strings.TrimSpace(accountProfile) != "" {
				artifacts.Profile = strings.TrimSpace(accountProfile)
			}
		}
	}
	if authRaw != "" && artifacts.RawSnapshot != "" {
		var infoMap map[string]any
		if err := json.Unmarshal([]byte(artifacts.RawSnapshot), &infoMap); err == nil {
			infoMap["observed_scope"] = artifacts.ObservedScope
			infoMap["account_id"] = accountID
			infoMap["account_email"] = email
			infoMap["account_user"] = user
			infoMap["account_source"] = source
			infoMap["auth_snapshot"] = jsonMap(authRaw)
			if merged, err := json.Marshal(infoMap); err == nil {
				artifacts.RawSnapshot = string(merged)
			}
		}
	}
	if artifacts.RawSnapshot != "" && authRaw == "" {
		var infoMap map[string]any
		if err := json.Unmarshal([]byte(artifacts.RawSnapshot), &infoMap); err == nil {
			infoMap["observed_scope"] = artifacts.ObservedScope
			if merged, err := json.Marshal(infoMap); err == nil {
				artifacts.RawSnapshot = string(merged)
			}
		}
	}
	artifacts.AccountID = accountID
	artifacts.AccountEmail = email
	artifacts.AccountUser = user
	artifacts.AccountSource = source
	return artifacts, nil
}

func readCodexLatestTokenCountForAccount(renderedCommand, accountID, email string, startedAt time.Time) (string, time.Time, codexTokenCountPayload, string, error) {
	baseRoot := codexProfilesBase(renderedCommand)
	if strings.TrimSpace(baseRoot) == "" {
		return "", time.Time{}, codexTokenCountPayload{}, "", nil
	}
	homesDir := filepath.Join(baseRoot, "homes")
	entries, err := os.ReadDir(homesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", time.Time{}, codexTokenCountPayload{}, "", nil
		}
		return "", time.Time{}, codexTokenCountPayload{}, "", err
	}
	wantAccountID := strings.ToLower(strings.TrimSpace(accountID))
	wantEmail := strings.ToLower(strings.TrimSpace(email))
	var bestPath string
	var bestObserved time.Time
	var bestPayload codexTokenCountPayload
	var bestProfile string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		profile := strings.TrimSpace(entry.Name())
		if profile == "" {
			continue
		}
		authPath := filepath.Join(homesDir, profile, "auth.json")
		raw, gotEmail, _, gotAccountID, _ := readCodexAuthIdentityFromPath(authPath)
		_ = raw
		switch {
		case wantAccountID != "":
			if strings.ToLower(strings.TrimSpace(gotAccountID)) != wantAccountID {
				continue
			}
		case wantEmail != "":
			if strings.ToLower(strings.TrimSpace(gotEmail)) != wantEmail {
				continue
			}
		default:
			continue
		}
		rendered := filepath.Join(baseRoot, "bin", "codex-perfil") + " " + profile
		path, observedAt, payload, err := readCodexLatestTokenCount(rendered, "", startedAt)
		if err != nil {
			return "", time.Time{}, codexTokenCountPayload{}, "", err
		}
		if observedAt.IsZero() {
			continue
		}
		if bestPath == "" || observedAt.After(bestObserved) {
			bestPath = path
			bestObserved = observedAt
			bestPayload = payload
			bestProfile = profile
		}
	}
	return bestPath, bestObserved, bestPayload, bestProfile, nil
}

func readCodexLatestTokenCount(renderedCommand, externalID string, startedAt time.Time) (string, time.Time, codexTokenCountPayload, error) {
	var bestPath string
	var bestObserved time.Time
	var bestPayload codexTokenCountPayload
	for _, root := range codexSessionRoots(renderedCommand) {
		for _, dir := range codexSessionDayDirs(root, startedAt, time.Now().UTC()) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", time.Time{}, codexTokenCountPayload{}, err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				match, observedAt, payload, err := readCodexTokenCountFromSession(path, externalID)
				if err != nil {
					return "", time.Time{}, codexTokenCountPayload{}, err
				}
				if !match || observedAt.IsZero() {
					continue
				}
				if bestPath == "" || observedAt.After(bestObserved) {
					bestPath = path
					bestObserved = observedAt
					bestPayload = payload
				}
			}
		}
	}
	return bestPath, bestObserved, bestPayload, nil
}

func readCodexTokenCountFromSession(path, externalID string) (bool, time.Time, codexTokenCountPayload, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, time.Time{}, codexTokenCountPayload{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	match := strings.TrimSpace(externalID) == ""
	var observedAt time.Time
	var payload codexTokenCountPayload
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev codexSessionEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if !match && ev.Type == "session_meta" {
			var meta codexSessionMetaLine
			if err := json.Unmarshal([]byte(line), &meta); err == nil && strings.TrimSpace(meta.Payload.ID) == strings.TrimSpace(externalID) {
				match = true
			}
		}
		if !match || ev.Type != "event_msg" {
			continue
		}
		var tokenPayload codexTokenCountPayload
		if err := json.Unmarshal(ev.Payload, &tokenPayload); err != nil {
			continue
		}
		if strings.TrimSpace(tokenPayload.Type) != "token_count" {
			continue
		}
		ts := firstParsedTime(ev.Timestamp)
		if ts.IsZero() {
			continue
		}
		observedAt = ts.UTC()
		payload = tokenPayload
	}
	if err := scanner.Err(); err != nil {
		return false, time.Time{}, codexTokenCountPayload{}, err
	}
	return match, observedAt, payload, nil
}

func parseCodexResetAt(raw any) *time.Time {
	switch v := raw.(type) {
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
		if err != nil || n <= 0 {
			return nil
		}
		ts := time.Unix(n, 0).UTC()
		return &ts
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
			ts = ts.UTC()
			return &ts
		}
	}
	return nil
}

func perfilDesdeRenderedCommand(rendered string) string {
	tokens, err := splitShellQuotedCommand(rendered)
	if err != nil || len(tokens) < 2 {
		return ""
	}
	if filepath.Base(tokens[0]) != "codex-perfil" {
		return ""
	}
	return strings.TrimSpace(tokens[1])
}

func readCodexAuthIdentity(rendered, perfil string) (string, string, string, string, string) {
	authPath := codexAuthPath(rendered, perfil)
	if authPath == "" {
		return "", "", "", "", ""
	}
	return readCodexAuthIdentityFromPath(authPath)
}

func readCodexAuthIdentityFromPath(authPath string) (string, string, string, string, string) {
	raw, err := os.ReadFile(authPath)
	if err != nil {
		return "", "", "", "", ""
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", "", "", "", ""
	}
	accountID := strings.TrimSpace(recursiveString(data,
		"account_id",
		"accountId",
	))
	email := strings.TrimSpace(recursiveString(data,
		"email",
		"account_email",
		"user_email",
	))
	user := strings.TrimSpace(recursiveString(data,
		"name",
		"preferred_username",
		"username",
		"user_name",
		"account_user",
	))
	if email == "" || user == "" {
		if jwt := tokenClaimsFromAuth(data); jwt != nil {
			if email == "" {
				email = strings.TrimSpace(recursiveString(jwt, "email"))
			}
			if user == "" {
				user = strings.TrimSpace(recursiveString(jwt, "name", "preferred_username"))
			}
		}
	}
	if accountID == "" && email == "" && user == "" {
		return string(raw), "", "", "", ""
	}
	return string(raw), email, user, accountID, "codex_auth"
}

func codexAuthPath(rendered, perfil string) string {
	baseRoot := codexProfilesBase(rendered)
	for _, root := range codexSessionRoots(rendered) {
		if strings.HasSuffix(root, string(os.PathSeparator)+"sessions") {
			return filepath.Join(filepath.Dir(root), "auth.json")
		}
	}
	if perfil != "" && baseRoot != "" {
		return filepath.Join(baseRoot, "homes", perfil, "auth.json")
	}
	if perfil != "" {
		return filepath.Join(userHomeDir(), "Trabajo", "codex-perfiles", "homes", perfil, "auth.json")
	}
	return ""
}

func codexProfilesBase(rendered string) string {
	tokens, err := splitShellQuotedCommand(rendered)
	if err != nil || len(tokens) == 0 {
		return ""
	}
	if filepath.Base(tokens[0]) != "codex-perfil" {
		return ""
	}
	if strings.Contains(tokens[0], string(os.PathSeparator)) {
		return filepath.Dir(filepath.Dir(tokens[0]))
	}
	return filepath.Join(userHomeDir(), "Trabajo", "codex-perfiles")
}

func tokenClaimsFromAuth(data map[string]any) map[string]any {
	tokens, _ := data["tokens"].(map[string]any)
	for _, key := range []string{"id_token", "access_token"} {
		raw := strings.TrimSpace(stringValueFromMetadata(tokens, key))
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

func jsonMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func codexObservedCredits(raw any) *float64 {
	switch value := raw.(type) {
	case nil:
		return nil
	case float64:
		out := value
		return &out
	case json.Number:
		n, err := value.Float64()
		if err != nil {
			return nil
		}
		return &n
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil
		}
		return &n
	case map[string]any:
		if unlimited, ok := value["unlimited"].(bool); ok && unlimited {
			return nil
		}
		if balance, ok := value["balance"]; ok {
			return codexObservedCredits(balance)
		}
	}
	return nil
}

func recursiveString(raw any, keys ...string) string {
	switch value := raw.(type) {
	case map[string]any:
		for _, key := range keys {
			if text := strings.TrimSpace(stringValueFromMetadata(value, key)); text != "" {
				return text
			}
		}
		for _, nested := range value {
			if text := recursiveString(nested, keys...); text != "" {
				return text
			}
		}
	case []any:
		for _, nested := range value {
			if text := recursiveString(nested, keys...); text != "" {
				return text
			}
		}
	}
	return ""
}
