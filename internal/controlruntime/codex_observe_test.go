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
	authJSON := `{"profile":{"email":"codex1@example.com","name":"Cuenta Codex1","account_id":"acc-codex1"}}`
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
	if artifacts.AccountID != "acc-codex1" {
		t.Fatalf("account_id inesperado: %+v", artifacts)
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

func TestObserveCodexArtifactsUsaSnapshotMasFrescoDelPerfil(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	sessionsDir := filepath.Join(base, "homes", "Codex1", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	oldFile := filepath.Join(sessionsDir, "rollout-old.jsonl")
	newFile := filepath.Join(sessionsDir, "rollout-new.jsonl")
	if err := os.WriteFile(oldFile, []byte(
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-123","timestamp":"2026-03-31T10:00:00Z","cwd":"/tmp/orquesta"}}`+"\n"+
			`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":1774958400},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":1775300000}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write old session: %v", err)
	}
	if err := os.WriteFile(newFile, []byte(
		`{"timestamp":"2026-03-31T11:00:00Z","type":"session_meta","payload":{"id":"sess-999","timestamp":"2026-03-31T11:00:00Z","cwd":"/tmp/orquesta"}}`+"\n"+
			`{"timestamp":"2026-03-31T11:20:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":30,"window_minutes":300,"resets_at":1774962000},"secondary":{"used_percent":55,"window_minutes":10080,"resets_at":1775303600}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write new session: %v", err)
	}
	rendered := filepath.Join(base, "bin", "codex-perfil") + " Codex1"
	meta := map[string]any{
		"rendered_command":    rendered,
		"working_dir":         "/tmp/orquesta",
		"external_session_id": "sess-123",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexArtifacts(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexArtifacts: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.ObservedScope != "profile" {
		t.Fatalf("scope inesperado: %+v", artifacts)
	}
	if artifacts.SessionPath != newFile {
		t.Fatalf("deberia elegir el snapshot mas fresco del perfil: %s", artifacts.SessionPath)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 30 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
}

func TestObserveCodexArtifactsUsaSnapshotMasFrescoDeLaCuenta(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	for _, profile := range []string{"Codex1", "Codex5"} {
		if err := os.MkdirAll(filepath.Join(base, "homes", profile, "sessions", "2026", "03", "31"), 0o755); err != nil {
			t.Fatalf("mkdir sessions %s: %v", profile, err)
		}
		authPath := filepath.Join(base, "homes", profile, "auth.json")
		if err := os.WriteFile(authPath, []byte(`{"profile":{"email":"shared@example.com","name":"Cuenta Compartida"}}`), 0o600); err != nil {
			t.Fatalf("write auth %s: %v", profile, err)
		}
	}
	oldFile := filepath.Join(base, "homes", "Codex1", "sessions", "2026", "03", "31", "old.jsonl")
	if err := os.WriteFile(oldFile, []byte(
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-123","timestamp":"2026-03-31T10:00:00Z","cwd":"/tmp/orquesta"}}`+"\n"+
			`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":1774958400},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":1775300000}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write old session: %v", err)
	}
	newFile := filepath.Join(base, "homes", "Codex5", "sessions", "2026", "03", "31", "new.jsonl")
	if err := os.WriteFile(newFile, []byte(
		`{"timestamp":"2026-03-31T11:00:00Z","type":"session_meta","payload":{"id":"sess-555","timestamp":"2026-03-31T11:00:00Z","cwd":"/tmp/otro"}}`+"\n"+
			`{"timestamp":"2026-03-31T11:25:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":28,"window_minutes":300,"resets_at":1774962000},"secondary":{"used_percent":41,"window_minutes":10080,"resets_at":1775303600}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write new session: %v", err)
	}
	rendered := filepath.Join(base, "bin", "codex-perfil") + " Codex1"
	meta := map[string]any{
		"rendered_command":    rendered,
		"working_dir":         "/tmp/orquesta",
		"external_session_id": "sess-123",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexArtifacts(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexArtifacts: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.ObservedScope != "account" {
		t.Fatalf("scope inesperado: %+v", artifacts)
	}
	if artifacts.SessionPath != newFile {
		t.Fatalf("deberia elegir el snapshot mas fresco de la cuenta: %s", artifacts.SessionPath)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 28 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
}

func TestObserveCodexArtifactsNoMezclaPerfilesConMismoEmailSiTienenAccountIDDistinto(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	for profile, accountID := range map[string]string{
		"Codex1": "acc-uno",
		"Codex5": "acc-cinco",
	} {
		if err := os.MkdirAll(filepath.Join(base, "homes", profile, "sessions", "2026", "03", "31"), 0o755); err != nil {
			t.Fatalf("mkdir sessions %s: %v", profile, err)
		}
		authPath := filepath.Join(base, "homes", profile, "auth.json")
		authJSON := `{"profile":{"email":"shared@example.com","name":"Cuenta Compartida","account_id":"` + accountID + `"}}`
		if err := os.WriteFile(authPath, []byte(authJSON), 0o600); err != nil {
			t.Fatalf("write auth %s: %v", profile, err)
		}
	}
	oldFile := filepath.Join(base, "homes", "Codex1", "sessions", "2026", "03", "31", "old.jsonl")
	if err := os.WriteFile(oldFile, []byte(
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-123","timestamp":"2026-03-31T10:00:00Z","cwd":"/tmp/orquesta"}}`+"\n"+
			`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":1774958400},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":1775300000}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write old session: %v", err)
	}
	newFile := filepath.Join(base, "homes", "Codex5", "sessions", "2026", "03", "31", "new.jsonl")
	if err := os.WriteFile(newFile, []byte(
		`{"timestamp":"2026-03-31T11:00:00Z","type":"session_meta","payload":{"id":"sess-555","timestamp":"2026-03-31T11:00:00Z","cwd":"/tmp/otro"}}`+"\n"+
			`{"timestamp":"2026-03-31T11:25:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":28,"window_minutes":300,"resets_at":1774962000},"secondary":{"used_percent":41,"window_minutes":10080,"resets_at":1775303600}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write new session: %v", err)
	}
	rendered := filepath.Join(base, "bin", "codex-perfil") + " Codex1"
	meta := map[string]any{
		"rendered_command":    rendered,
		"working_dir":         "/tmp/orquesta",
		"external_session_id": "sess-123",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexArtifacts(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexArtifacts: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.AccountID != "acc-uno" {
		t.Fatalf("deberia conservar el account_id del perfil lanzado: %+v", artifacts)
	}
	if artifacts.SessionPath != oldFile {
		t.Fatalf("no deberia mezclar otra cuenta solo por compartir email: %s", artifacts.SessionPath)
	}
	if artifacts.SessionPath == newFile {
		t.Fatalf("no deberia saltar al snapshot de otro account_id")
	}
}

func TestObserveCodexArtifactsAceptaCreditsObjetoEnTokenCount(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	sessionsDir := filepath.Join(base, "homes", "Codex2", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	sessionPath := filepath.Join(sessionsDir, "rollout-test.jsonl")
	if err := os.WriteFile(sessionPath, []byte(
		`{"timestamp":"2026-03-31T19:31:40Z","type":"session_meta","payload":{"id":"sess-credits","timestamp":"2026-03-31T19:31:40Z","cwd":"/tmp/orquesta"}}`+"\n"+
			`{"timestamp":"2026-03-31T19:31:43Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":0,"window_minutes":300,"resets_at":1775003503},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":1775294400},"credits":{"has_credits":false,"unlimited":false,"balance":null},"plan_type":null}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write session: %v", err)
	}
	rendered := filepath.Join(base, "bin", "codex-perfil") + " Codex2"
	meta := map[string]any{
		"rendered_command":    rendered,
		"working_dir":         "/tmp/orquesta",
		"external_session_id": "sess-credits",
		"started_at":          time.Date(2026, 3, 31, 19, 31, 40, 0, time.UTC).Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexArtifacts(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexArtifacts: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.SessionPath != sessionPath {
		t.Fatalf("session path inesperado: %s", artifacts.SessionPath)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 0 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
	if artifacts.Secondary.UsedPercent == nil || *artifacts.Secondary.UsedPercent != 100 {
		t.Fatalf("secondary inesperado: %+v", artifacts.Secondary)
	}
	if artifacts.Credits != nil {
		t.Fatalf("credits deberia quedar nil cuando el proveedor expone objeto sin balance: %+v", artifacts.Credits)
	}
}

func jsonInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

func TestObserveCodexProfileStatusLeeHelperDelPerfil(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${2:-}" == "status-json" ]]; then
cat <<'JSON'
{"source":"codex_profile_status","profile":"Codex7","checked_at":"2026-04-02T18:00:00Z","observed_at":"2026-04-02T17:59:00Z","session_path":"/tmp/sess.jsonl","session_id":"sess-777","account":{"email":"alberto@avidad.com","user":"Alberto","account_id":"acc-profile-7","plan_type":"team"},"rate_limits":{"primary":{"used_percent":11,"left_percent":89,"window_minutes":300,"resets_at":"2026-04-02T20:31:00Z"},"secondary":{"used_percent":3,"left_percent":97,"window_minutes":10080,"resets_at":"2026-04-09T15:31:00Z"},"credits":null,"plan_type":"team"}}
JSON
exit 0
fi
exit 1
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	meta := map[string]any{
		"rendered_command": wrapper + " Codex7",
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexProfileStatus(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexProfileStatus: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.AccountEmail != "alberto@avidad.com" {
		t.Fatalf("email inesperado: %+v", artifacts)
	}
	if artifacts.AccountID != "acc-profile-7" {
		t.Fatalf("account_id inesperado: %+v", artifacts)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 11 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
	if artifacts.Secondary.UsedPercent == nil || *artifacts.Secondary.UsedPercent != 3 {
		t.Fatalf("secondary inesperado: %+v", artifacts.Secondary)
	}
	if artifacts.ObservationSource != "codex_profile_status" {
		t.Fatalf("fuente inesperada: %+v", artifacts)
	}
}

func TestObserveCodexProfileStatusPrefiereMetadataExplicitaDelPerfil(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "Codex8" || "${2:-}" != "status-json" ]]; then
  exit 7
fi
cat <<'JSON'
{"source":"codex_profile_status","profile":"Codex8","checked_at":"2026-04-02T18:00:00Z","observed_at":"2026-04-02T17:59:00Z","session_path":"/tmp/sess-8.jsonl","session_id":"sess-888","account":{"email":"berserk@avidad.com","user":"Berserk","plan_type":"team"},"rate_limits":{"primary":{"used_percent":22,"left_percent":78,"window_minutes":300,"resets_at":"2026-04-02T20:31:00Z"},"secondary":{"used_percent":9,"left_percent":91,"window_minutes":10080,"resets_at":"2026-04-09T15:31:00Z"},"credits":null,"plan_type":"team"}}
JSON
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	meta := map[string]any{
		"rendered_command":       "exec wrapper opaco",
		"profile_status_wrapper": wrapper,
		"profile_name":           "Codex8",
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexProfileStatus(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexProfileStatus: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.Profile != "Codex8" {
		t.Fatalf("profile inesperado: %+v", artifacts)
	}
	if artifacts.AccountEmail != "berserk@avidad.com" {
		t.Fatalf("email inesperado: %+v", artifacts)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 22 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
}

func TestObserveCodexProfileStatusLeeWrapperDesdeWorkerManifest(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "Codex11" || "${2:-}" != "status-json" ]]; then
  exit 9
fi
cat <<'JSON'
{"source":"codex_profile_status","profile":"Codex11","checked_at":"2026-04-07T08:00:00Z","observed_at":"2026-04-07T07:59:30Z","session_path":"/tmp/sess-11.jsonl","session_id":"sess-111","account":{"email":"codex11@avidad.com","user":"Codex11","plan_type":"team"},"rate_limits":{"primary":{"used_percent":31,"left_percent":69,"window_minutes":300,"resets_at":"2026-04-07T11:00:00Z"},"secondary":{"used_percent":12,"left_percent":88,"window_minutes":10080,"resets_at":"2026-04-14T08:00:00Z"},"credits":null,"plan_type":"team"}}
JSON
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	runDir := filepath.Join(tmp, "runtime", "Codex11", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	manifest := map[string]any{
		"version":                1,
		"agent":                  "Codex11",
		"driver":                 "tmux_cli_session",
		"transport":              "tmux",
		"profile":                "Codex11",
		"profile_status_wrapper": wrapper,
		"status_path":            statusPath,
		"heartbeat_path":         heartbeatPath,
	}
	status := map[string]any{
		"state":      "running",
		"updated_at": "2026-04-07T08:00:00Z",
		"alive":      true,
		"child_pid":  111,
	}
	heartbeat := map[string]any{
		"alive":        true,
		"heartbeat_at": "2026-04-07T08:00:00Z",
		"child_pid":    111,
	}
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, manifest)
	writeJSON(statusPath, status)
	writeJSON(heartbeatPath, heartbeat)

	meta := map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"rendered_command":      "wrapper opaco sin perfil",
	}
	metaJSON, _ := json.Marshal(meta)
	artifacts, err := ObserveCodexProfileStatus(ObjetivoProceso{MetadataJSON: string(metaJSON)})
	if err != nil {
		t.Fatalf("ObserveCodexProfileStatus: %v", err)
	}
	if artifacts == nil {
		t.Fatalf("artifacts nil")
	}
	if artifacts.Profile != "Codex11" {
		t.Fatalf("profile inesperado: %+v", artifacts)
	}
	if artifacts.AccountEmail != "codex11@avidad.com" {
		t.Fatalf("email inesperado: %+v", artifacts)
	}
	if artifacts.Primary.UsedPercent == nil || *artifacts.Primary.UsedPercent != 31 {
		t.Fatalf("primary inesperado: %+v", artifacts.Primary)
	}
}
