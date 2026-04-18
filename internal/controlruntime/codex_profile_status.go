package controlruntime

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

var codexProfileStatusCommandTimeout = 1500 * time.Millisecond

type codexProfileStatusEnvelope struct {
	Source       string `json:"source"`
	Profile      string `json:"profile"`
	CodexHome    string `json:"codex_home"`
	CheckedAt    string `json:"checked_at"`
	ObservedAt   string `json:"observed_at"`
	SessionPath  string `json:"session_path"`
	SessionID    string `json:"session_id"`
	FreshestHome string `json:"freshest_home"`
	Account      struct {
		Email       string `json:"email"`
		User        string `json:"user"`
		PlanType    string `json:"plan_type"`
		AccountID   string `json:"account_id"`
		AuthMode    string `json:"auth_mode"`
		AuthSource  string `json:"auth_source"`
		LastRefresh string `json:"last_refresh"`
	} `json:"account"`
	RateLimits struct {
		Primary struct {
			UsedPercent   *float64 `json:"used_percent"`
			LeftPercent   *float64 `json:"left_percent"`
			WindowMinutes int      `json:"window_minutes"`
			ResetsAt      any      `json:"resets_at"`
		} `json:"primary"`
		Secondary struct {
			UsedPercent   *float64 `json:"used_percent"`
			LeftPercent   *float64 `json:"left_percent"`
			WindowMinutes int      `json:"window_minutes"`
			ResetsAt      any      `json:"resets_at"`
		} `json:"secondary"`
		Credits  any    `json:"credits"`
		PlanType string `json:"plan_type"`
	} `json:"rate_limits"`
}

func ObserveCodexProfileStatus(obj ObjetivoProceso) (*CodexObservedArtifacts, error) {
	if !esRuntimeCodexLocal(obj) {
		return nil, nil
	}
	wrapper, profile, ok := codexProfileCommand(obj.MetadataJSON)
	if !ok {
		return nil, nil
	}
	timeout := codexProfileStatusCommandTimeout
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, wrapper, profile, "status-json")
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	var env codexProfileStatusEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, err
	}
	observedAt := firstParsedTime(strings.TrimSpace(env.CheckedAt), strings.TrimSpace(env.ObservedAt))
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	meta := map[string]any{
		"observed_scope": strings.TrimSpace(env.FreshestHome),
		"profile":        strings.TrimSpace(env.Profile),
		"codex_home":     strings.TrimSpace(env.CodexHome),
		"session_path":   strings.TrimSpace(env.SessionPath),
		"session_id":     strings.TrimSpace(env.SessionID),
		"account_id":     strings.TrimSpace(env.Account.AccountID),
		"account_email":  strings.TrimSpace(env.Account.Email),
		"account_user":   strings.TrimSpace(env.Account.User),
		"plan_type":      firstNonEmptyString(strings.TrimSpace(env.RateLimits.PlanType), strings.TrimSpace(env.Account.PlanType)),
		"rate_limits": map[string]any{
			"primary": map[string]any{
				"used_percent":   env.RateLimits.Primary.UsedPercent,
				"left_percent":   env.RateLimits.Primary.LeftPercent,
				"window_minutes": env.RateLimits.Primary.WindowMinutes,
				"resets_at":      env.RateLimits.Primary.ResetsAt,
			},
			"secondary": map[string]any{
				"used_percent":   env.RateLimits.Secondary.UsedPercent,
				"left_percent":   env.RateLimits.Secondary.LeftPercent,
				"window_minutes": env.RateLimits.Secondary.WindowMinutes,
				"resets_at":      env.RateLimits.Secondary.ResetsAt,
			},
			"credits":   env.RateLimits.Credits,
			"plan_type": firstNonEmptyString(strings.TrimSpace(env.RateLimits.PlanType), strings.TrimSpace(env.Account.PlanType)),
		},
	}
	rawSnapshot := "{}"
	if raw, err := json.Marshal(meta); err == nil {
		rawSnapshot = string(raw)
	}
	return &CodexObservedArtifacts{
		ObservationSource: firstNonEmptyString(strings.TrimSpace(env.Source), "codex_profile_status"),
		Profile:           firstNonEmptyString(strings.TrimSpace(env.Profile), profile),
		ExternalID:        strings.TrimSpace(env.SessionID),
		SessionPath:       strings.TrimSpace(env.SessionPath),
		ObservedAt:        observedAt,
		ObservedScope:     "profile_status",
		AccountID:         strings.TrimSpace(env.Account.AccountID),
		AccountEmail:      strings.TrimSpace(env.Account.Email),
		AccountUser:       strings.TrimSpace(env.Account.User),
		AccountSource:     "codex_profile_status",
		Primary: CodexObservedRateLimit{
			UsedPercent:   env.RateLimits.Primary.UsedPercent,
			WindowMinutes: env.RateLimits.Primary.WindowMinutes,
			ResetsAt:      parseCodexResetAt(env.RateLimits.Primary.ResetsAt),
		},
		Secondary: CodexObservedRateLimit{
			UsedPercent:   env.RateLimits.Secondary.UsedPercent,
			WindowMinutes: env.RateLimits.Secondary.WindowMinutes,
			ResetsAt:      parseCodexResetAt(env.RateLimits.Secondary.ResetsAt),
		},
		Credits:     codexObservedCredits(env.RateLimits.Credits),
		PlanType:    firstNonEmptyString(strings.TrimSpace(env.RateLimits.PlanType), strings.TrimSpace(env.Account.PlanType)),
		RawSnapshot: rawSnapshot,
	}, nil
}

func codexProfileCommand(metadataJSON string) (string, string, bool) {
	meta := metadataMap(metadataJSON)
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(metadataJSON); err == nil && snap != nil && snap.Manifest != nil {
		if wrapper := strings.TrimSpace(snap.Manifest.ProfileStatusWrapper); wrapper != "" {
			if profile := strings.TrimSpace(snap.Manifest.Profile); profile != "" {
				return wrapper, profile, true
			}
		}
	}
	if wrapper := strings.TrimSpace(stringValueFromMetadata(meta, "profile_status_wrapper")); wrapper != "" {
		if profile := strings.TrimSpace(stringValueFromMetadata(meta, "profile_name")); profile != "" {
			return wrapper, profile, true
		}
	}
	return codexProfileCommandFromCandidates(
		stringValueFromMetadata(meta, "rendered_command"),
		stringValueFromMetadata(meta, "wrapped_command"),
	)
}

func codexProfileCommandFromCandidates(candidates ...string) (string, string, bool) {
	for _, candidate := range candidates {
		wrapper, profile, ok := codexProfileCommandFromRendered(candidate)
		if ok {
			return wrapper, profile, true
		}
	}
	return "", "", false
}

func codexProfileCommandFromRendered(rendered string) (string, string, bool) {
	tokens, err := splitShellQuotedCommand(strings.TrimSpace(rendered))
	if err != nil || len(tokens) < 2 {
		return "", "", false
	}
	if filepath.Base(tokens[0]) != "codex-perfil" {
		return "", "", false
	}
	profile := strings.TrimSpace(tokens[1])
	if profile == "" {
		return "", "", false
	}
	return tokens[0], profile, true
}
