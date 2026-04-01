package controlruntime

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestObserveClaudeRustArtifactsLeeCredencialesYSesion(t *testing.T) {
	tmp := t.TempDir()
	configHome := filepath.Join(tmp, "claude-home")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"claude6@example.com","name":"Carlos Claude"}`))
	credentials := `{"oauth":{"access_token":"header.` + claims + `.sig","expires_at":"2026-04-01T12:00:00Z"}}`
	if err := os.WriteFile(filepath.Join(configHome, "credentials.json"), []byte(credentials), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	workingDir := filepath.Join(tmp, "repo")
	sessionsDir := filepath.Join(workingDir, ".claude", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	sessionJSON := map[string]any{
		"version": 1,
		"messages": []any{
			map[string]any{"role": "user", "blocks": []any{map[string]any{"type": "text", "text": "hola"}}},
			map[string]any{
				"role": "assistant",
				"blocks": []any{map[string]any{"type": "text", "text": "ok"}},
				"usage": map[string]any{
					"input_tokens":                1200,
					"output_tokens":               300,
					"cache_creation_input_tokens": 50,
					"cache_read_input_tokens":     20,
				},
			},
		},
	}
	rawSession, _ := json.Marshal(sessionJSON)
	sessionPath := filepath.Join(sessionsDir, "session-1.json")
	if err := os.WriteFile(sessionPath, rawSession, 0o600); err != nil {
		t.Fatalf("write session: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"rendered_command":   "claude",
		"working_dir":        workingDir,
		"claude_config_home": configHome,
		"herramienta":        "claude-cli",
	})
	artifacts, err := ObserveClaudeRustArtifacts(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveClaudeRustArtifacts: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.AccountUser != "Carlos Claude" {
		t.Fatalf("account user inesperado: %+v", artifacts)
	}
	if artifacts.SessionPath != sessionPath {
		t.Fatalf("session path inesperado: %s", artifacts.SessionPath)
	}
	if artifacts.Usage.TotalTokens != 1570 {
		t.Fatalf("total tokens inesperado: %+v", artifacts.Usage)
	}
	if artifacts.Usage.EstimatedCostUSD == nil || *artifacts.Usage.EstimatedCostUSD <= 0 {
		t.Fatalf("coste estimado inesperado: %+v", artifacts.Usage)
	}
	if artifacts.OAuthExpiresAt == nil || artifacts.OAuthExpiresAt.UTC().Format(time.RFC3339) != "2026-04-01T12:00:00Z" {
		t.Fatalf("oauth expires_at inesperado: %+v", artifacts)
	}
}
