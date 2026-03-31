package controlruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestObserveCodexArtifactsLeeTokenCountYAuth(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	sessionsDir := filepath.Join(base, "homes", "Codex1", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	authPath := filepath.Join(base, "homes", "Codex1", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatalf("mkdir auth dir: %v", err)
	}
	authJSON := `{"profile":{"email":"codex1@example.com","name":"Cuenta Codex1"}}`
	if err := os.WriteFile(authPath, []byte(authJSON), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	resetPrimary := time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC)
	resetSecondary := time.Date(2026, 4, 6, 7, 26, 0, 0, time.UTC)
	sessionPath := filepath.Join(sessionsDir, "rollout-test.jsonl")
	lines := []string{
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-123","timestamp":"2026-03-31T10:00:00Z","cwd":"/tmp/orquesta"}}`,
		`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":123}},"rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":` + jsonInt(resetPrimary.Unix()) + `},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":` + jsonInt(resetSecondary.Unix()) + `},"credits":null,"plan_type":"plus"}}}`,
	}
	if err := os.WriteFile(sessionPath, []byte(lines[0]+"\n"+lines[1]+"\n"), 0o600); err != nil {
		t.Fatalf("write session: %v", err)
	}
	rendered := filepath.Join(base, "bin", "codex-perfil") + " Codex1"
	startedAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	meta := map[string]any{
		"rendered_command":    rendered,
		"working_dir":         "/tmp/orquesta",
		"external_session_id": "sess-123",
		"started_at":          startedAt.Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexArtifacts(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexArtifacts: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.AccountEmail != "codex1@example.com" {
		t.Fatalf("email inesperado: %+v", artifacts)
	}
	if artifacts.AccountUser != "Cuenta Codex1" {
		t.Fatalf("usuario inesperado: %+v", artifacts)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 12 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
	if artifacts.Secondary.UsedPercent == nil || *artifacts.Secondary.UsedPercent != 43 {
		t.Fatalf("secondary inesperado: %+v", artifacts.Secondary)
	}
	if artifacts.PlanType != "plus" {
		t.Fatalf("plan_type inesperado: %+v", artifacts)
	}
	if artifacts.SessionPath != sessionPath {
		t.Fatalf("session path inesperado: %s", artifacts.SessionPath)
	}
}

func jsonInt(v int64) string {
	return strconv.FormatInt(v, 10)
}
