package controlruntime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CodexStatusLive struct {
	AccountEmail  string
	PlanType      string
	Model         string
	SessionID     string
	FiveHourLeft  *float64
	FiveHourReset *time.Time
	WeeklyLeft    *float64
	WeeklyReset   *time.Time
	RawOutput     string
}

var (
	codexStatusAccountPattern = regexp.MustCompile(`(?m)^Account:\s+([^\s]+)\s+\(([^)]+)\)\s*$`)
	codexStatusModelPattern   = regexp.MustCompile(`(?m)^Model:\s+(.+?)\s*$`)
	codexStatusSessionPattern = regexp.MustCompile(`(?m)^Session:\s+([0-9a-fA-F-]+)\s*$`)
	codexStatus5hPattern      = regexp.MustCompile(`(?m)^5h limit:\s+\[[^\]]*\]\s+(\d+)% left \(resets ([^)]+)\)\s*$`)
	codexStatusWeekPattern    = regexp.MustCompile(`(?m)^Weekly limit:\s+\[[^\]]*\]\s+(\d+)% left \(resets ([^)]+)\)\s*$`)
)

func ObserveCodexStatusLive(obj ObjetivoProceso) (*CodexObservedArtifacts, error) {
	if !esRuntimeCodexLocal(obj) {
		return nil, nil
	}
	result, err := RunSlashCommandLive(obj, "/status", SlashCommandOptions{})
	if err != nil {
		return nil, err
	}
	if result == nil || strings.TrimSpace(result.RawOutput) == "" {
		return nil, nil
	}
	status, err := ParseCodexStatusLive(result.RawOutput, result.FinishedAt)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, nil
	}
	meta := map[string]any{
		"observed_scope": "status_live",
		"account_email":  strings.TrimSpace(status.AccountEmail),
		"plan_type":      strings.TrimSpace(status.PlanType),
		"model":          strings.TrimSpace(status.Model),
		"session_id":     strings.TrimSpace(status.SessionID),
		"raw_output":     strings.TrimSpace(status.RawOutput),
	}
	if status.FiveHourLeft != nil || status.WeeklyLeft != nil {
		rateLimits := map[string]any{}
		if status.FiveHourLeft != nil {
			rateLimits["primary"] = map[string]any{
				"left_percent":   *status.FiveHourLeft,
				"used_percent":   100 - *status.FiveHourLeft,
				"window_minutes": 300,
				"resets_at":      timePtrRFC3339(status.FiveHourReset),
			}
		}
		if status.WeeklyLeft != nil {
			rateLimits["secondary"] = map[string]any{
				"left_percent":   *status.WeeklyLeft,
				"used_percent":   100 - *status.WeeklyLeft,
				"window_minutes": 10080,
				"resets_at":      timePtrRFC3339(status.WeeklyReset),
			}
		}
		meta["rate_limits"] = rateLimits
	}
	rawSnapshot := "{}"
	if raw, err := json.Marshal(meta); err == nil {
		rawSnapshot = string(raw)
	}
	artifacts := &CodexObservedArtifacts{
		ObservationSource: "codex_status_live",
		ObservedAt:        result.FinishedAt,
		ObservedScope:     "status_live",
		AccountEmail:      strings.TrimSpace(status.AccountEmail),
		AccountUser:       strings.TrimSpace(status.AccountEmail),
		AccountSource:     "codex_status_live",
		PlanType:          strings.TrimSpace(status.PlanType),
		ExternalID:        strings.TrimSpace(status.SessionID),
		RawSnapshot:       rawSnapshot,
		Primary: CodexObservedRateLimit{
			UsedPercent:   leftPercentToUsed(status.FiveHourLeft),
			WindowMinutes: 300,
			ResetsAt:      status.FiveHourReset,
		},
		Secondary: CodexObservedRateLimit{
			UsedPercent:   leftPercentToUsed(status.WeeklyLeft),
			WindowMinutes: 10080,
			ResetsAt:      status.WeeklyReset,
		},
	}
	return artifacts, nil
}

func ParseCodexStatusLive(raw string, now time.Time) (*CodexStatusLive, error) {
	text := normalizeSlashCommandOutput(raw)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	status := &CodexStatusLive{
		RawOutput: text,
	}
	if match := codexStatusAccountPattern.FindStringSubmatch(text); len(match) == 3 {
		status.AccountEmail = strings.TrimSpace(match[1])
		status.PlanType = strings.TrimSpace(match[2])
	}
	if match := codexStatusModelPattern.FindStringSubmatch(text); len(match) == 2 {
		status.Model = strings.TrimSpace(match[1])
	}
	if match := codexStatusSessionPattern.FindStringSubmatch(text); len(match) == 2 {
		status.SessionID = strings.TrimSpace(match[1])
	}
	if match := codexStatus5hPattern.FindStringSubmatch(text); len(match) == 3 {
		left, err := parseLeftPercent(match[1])
		if err != nil {
			return nil, err
		}
		status.FiveHourLeft = left
		status.FiveHourReset = parseCodexStatusShortReset(match[2], now)
	}
	if match := codexStatusWeekPattern.FindStringSubmatch(text); len(match) == 3 {
		left, err := parseLeftPercent(match[1])
		if err != nil {
			return nil, err
		}
		status.WeeklyLeft = left
		status.WeeklyReset = parseCodexStatusWeeklyReset(match[2], now)
	}
	if strings.TrimSpace(status.AccountEmail) == "" &&
		status.FiveHourLeft == nil &&
		status.WeeklyLeft == nil &&
		strings.TrimSpace(status.SessionID) == "" {
		return nil, fmt.Errorf("salida de /status no parseable")
	}
	return status, nil
}

func parseLeftPercent(raw string) (*float64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func leftPercentToUsed(left *float64) *float64 {
	if left == nil {
		return nil
	}
	used := 100 - *left
	return &used
}

func parseCodexStatusShortReset(raw string, now time.Time) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	loc := now.Location()
	ts, err := time.ParseInLocation("15:04", raw, loc)
	if err != nil {
		return nil
	}
	candidate := time.Date(now.Year(), now.Month(), now.Day(), ts.Hour(), ts.Minute(), 0, 0, loc)
	if !candidate.After(now.In(loc).Add(-1 * time.Minute)) {
		candidate = candidate.Add(24 * time.Hour)
	}
	utc := candidate.UTC()
	return &utc
}

func parseCodexStatusWeeklyReset(raw string, now time.Time) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	loc := now.Location()
	withYear := fmt.Sprintf("%s %d", raw, now.In(loc).Year())
	ts, err := time.ParseInLocation("15:04 on 2 Jan 2006", withYear, loc)
	if err != nil {
		return nil
	}
	if !ts.After(now.In(loc).Add(-1 * time.Minute)) {
		ts = ts.AddDate(1, 0, 0)
	}
	utc := ts.UTC()
	return &utc
}

func timePtrRFC3339(ts *time.Time) any {
	if ts == nil || ts.IsZero() {
		return nil
	}
	return ts.UTC().Format(time.RFC3339Nano)
}
