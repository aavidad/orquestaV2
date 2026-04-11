package controlruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunSlashCommandLiveLeeDeltaDelLog(t *testing.T) {
	tmp := t.TempDir()
	stdinPath := filepath.Join(tmp, "stdin.txt")
	logPath := filepath.Join(tmp, "pty.log")
	if err := os.WriteFile(stdinPath, nil, 0o600); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("previo\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	pid := int64(os.Getpid())
	obj := ObjetivoProceso{
		PID:          &pid,
		MetadataJSON: `{"stdin_path":"` + stdinPath + `","log_path":"` + logPath + `"}`,
	}
	go func() {
		time.Sleep(250 * time.Millisecond)
		f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600)
		if f != nil {
			_, _ = f.WriteString("Account: test@example.com (Plus)\n")
			_, _ = f.WriteString("5h limit: [████] 89% left (resets 20:31)\n")
			_ = f.Close()
		}
	}()
	got, err := RunSlashCommandLive(obj, "/status", SlashCommandOptions{
		Timeout: 2 * time.Second,
		Settle:  250 * time.Millisecond,
		Poll:    50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RunSlashCommandLive: %v", err)
	}
	if got == nil || !strings.Contains(got.RawOutput, "test@example.com") {
		t.Fatalf("salida inesperada: %+v", got)
	}
	stdinRaw, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	if !strings.Contains(string(stdinRaw), "/status") {
		t.Fatalf("el wrapper no entrego /status al stdin: %q", string(stdinRaw))
	}
}

func TestParseCodexStatusLiveExtraeCuentaYCuotas(t *testing.T) {
	raw := `
╭───────────────────────────────────────────────────────────────────────────────╮
│  >_ OpenAI Codex (v0.118.0)                                                    │
│  Model:                gpt-5.4 (reasoning medium, summaries auto)              │
│  Account:              sonia@avidad.com (Plus)                                 │
│  Session:              019d3e99-e436-7b40-ae33-db24c109a777                    │
│  5h limit:             [██████████████████░░] 89% left (resets 20:31)          │
│  Weekly limit:         [███████████████████░] 97% left (resets 15:31 on 9 Apr) │
╰───────────────────────────────────────────────────────────────────────────────╯`
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.FixedZone("CEST", 2*3600))
	got, err := ParseCodexStatusLive(raw, now)
	if err != nil {
		t.Fatalf("ParseCodexStatusLive: %v", err)
	}
	if got == nil {
		t.Fatalf("status nil")
	}
	if got.AccountEmail != "sonia@avidad.com" || got.PlanType != "Plus" {
		t.Fatalf("cuenta inesperada: %+v", got)
	}
	if got.FiveHourLeft == nil || *got.FiveHourLeft != 89 {
		t.Fatalf("5h inesperado: %+v", got)
	}
	if got.WeeklyLeft == nil || *got.WeeklyLeft != 97 {
		t.Fatalf("weekly inesperado: %+v", got)
	}
	if got.SessionID != "019d3e99-e436-7b40-ae33-db24c109a777" {
		t.Fatalf("session inesperada: %+v", got)
	}
	if got.FiveHourReset == nil || got.WeeklyReset == nil {
		t.Fatalf("resets no parseados: %+v", got)
	}
}

func TestSupportedSlashCommandsSegunRuntime(t *testing.T) {
	codex := SupportedSlashCommands(ObjetivoProceso{MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex7"}`})
	if len(codex) == 0 || codex[0].Command != "/status" {
		t.Fatalf("registry codex inesperado: %+v", codex)
	}
	claude := SupportedSlashCommands(ObjetivoProceso{MetadataJSON: `{"rendered_command":"claude","working_dir":"/tmp/x"}`})
	if len(claude) == 0 || claude[0].Command != "/status" {
		t.Fatalf("registry claude inesperado: %+v", claude)
	}
}
