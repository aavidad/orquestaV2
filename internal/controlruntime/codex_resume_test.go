package controlruntime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDetectExternalSessionIDFromCodexPerfilSessionStore(t *testing.T) {
	tmp := t.TempDir()
	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	startedAt := time.Now().UTC()
	workingDir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	sessionDir := filepath.Join(filepath.Dir(filepath.Dir(wrapper)), "homes", "Codex2", "sessions", startedAt.In(time.Local).Format("2006"), startedAt.In(time.Local).Format("01"), startedAt.In(time.Local).Format("02"))
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	sessionFile := filepath.Join(sessionDir, "rollout-test.jsonl")
	sessionMeta := `{"timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","type":"session_meta","payload":{"id":"sess-codex-123","timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","cwd":"` + workingDir + `"}}` + "\n"
	if err := os.WriteFile(sessionFile, []byte(sessionMeta), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	obj := ObjetivoProceso{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"'` + wrapper + `' 'Codex2'","working_dir":"` + workingDir + `","started_at":"` + startedAt.Format(time.RFC3339Nano) + `"}`,
	}
	got, err := DetectExternalSessionID(obj)
	if err != nil {
		t.Fatalf("DetectExternalSessionID: %v", err)
	}
	if got != "sess-codex-123" {
		t.Fatalf("session id inesperado: %q", got)
	}
}

func TestEnviarInstruccionSesionResumeUsaCodexPerfil(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "resume.out")
	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" > " + shellQuoteForTest(outPath) + "\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	obj := ObjetivoProceso{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"'` + wrapper + `' 'Codex5'","working_dir":"` + tmp + `","external_session_id":"sess-resume-456"}`,
	}
	aplicado, _, err := EnviarInstruccionSesionResume(obj, "", "hola session resume")
	if err != nil {
		t.Fatalf("EnviarInstruccionSesionResume: %v", err)
	}
	if !aplicado {
		t.Fatal("session_resume deberia aplicar control real")
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read resume output: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"Codex5", "exec", "resume", "sess-resume-456", "hola session resume"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("argv inesperado: got=%q want=%q", got, want)
	}
}

func TestEnviarInstruccionSesionResumeAceptaMetadataDeSupervisorLocal(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "resume-supervisor.out")
	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" > " + shellQuoteForTest(outPath) + "\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	obj := ObjetivoProceso{
		MetadataJSON: `{"supervisor_driver":"local_runtime_supervisor","wrapped_command":"` + wrapper + ` Codex5","working_dir":"` + tmp + `","external_session_id":"sess-supervisor-456","herramienta":"codex-cli"}`,
	}
	aplicado, _, err := EnviarInstruccionSesionResume(obj, "", "hola supervisor resume")
	if err != nil {
		t.Fatalf("EnviarInstruccionSesionResume supervisor: %v", err)
	}
	if !aplicado {
		t.Fatal("session_resume deberia aplicar control real desde metadata de supervisor")
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read resume output: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"Codex5", "exec", "resume", "sess-supervisor-456", "hola supervisor resume"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("argv inesperado desde supervisor: got=%q want=%q", got, want)
	}
}

func TestCodexResumeTimeoutDefaultYOverride(t *testing.T) {
	if got := codexResumeTimeout(nil); got != 20*time.Second {
		t.Fatalf("timeout por defecto inesperado: %s", got)
	}
	if got := codexResumeTimeout(map[string]any{"session_resume_timeout_ms": float64(3500)}); got != 3500*time.Millisecond {
		t.Fatalf("timeout override inesperado: %s", got)
	}
}

func TestCodexResumeCommandEnvInyectaCODEXBINYPATH(t *testing.T) {
	tmp := t.TempDir()
	codexBin := filepath.Join(tmp, "bin", "codex")
	if err := os.MkdirAll(filepath.Dir(codexBin), 0o755); err != nil {
		t.Fatalf("mkdir codex bin: %v", err)
	}
	if err := os.WriteFile(codexBin, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write codex bin: %v", err)
	}

	env := codexResumeCommandEnv(map[string]any{"codex_bin": codexBin})
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "CODEX_BIN="+codexBin) {
		t.Fatalf("faltaba CODEX_BIN en env: %s", joined)
	}
	if !strings.Contains(joined, "PATH="+filepath.Dir(codexBin)) && !strings.Contains(joined, "PATH="+filepath.Dir(codexBin)+string(os.PathListSeparator)) {
		t.Fatalf("faltaba PATH enriquecido en env: %s", joined)
	}
}

func TestTrimmedCommandOutputPrefiereErrorRelevanteAlBanner(t *testing.T) {
	raw := []byte("Perfil activo: Codex2\nCODEX_HOME: /tmp/codex\nConsejo: login\nERROR: You've hit your usage limit. Try again at Apr 4th, 2026 11:20 AM.\n")
	got := trimmedCommandOutput(raw)
	if !strings.Contains(strings.ToLower(got), "usage limit") {
		t.Fatalf("faltaba usage limit en salida resumida: %q", got)
	}
	if strings.Contains(got, "Perfil activo") {
		t.Fatalf("el banner no deberia tapar el error relevante: %q", got)
	}
}

func TestTrimmedCommandOutputPrefierePromptDeApproachingRateLimits(t *testing.T) {
	raw := []byte("Perfil activo: Codex2\nTests verdes\nApproaching rate limits\nSwitch to gpt-5.1-codex-mini for lower credit usage?\n")
	got := strings.ToLower(trimmedCommandOutput(raw))
	if !strings.Contains(got, "approaching rate limits") {
		t.Fatalf("faltaba approaching rate limits en salida resumida: %q", got)
	}
	if strings.Contains(got, "perfil activo") {
		t.Fatalf("el banner no deberia tapar el prompt relevante: %q", got)
	}
}

func TestDetectExternalSessionIDUsaSupervisorLocalResidente(t *testing.T) {
	tmp := t.TempDir()
	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	startedAt := time.Now().UTC()
	workingDir := filepath.Join(tmp, "work")
	traceDir := filepath.Join(tmp, "trace")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	sessionDir := filepath.Join(filepath.Dir(filepath.Dir(wrapper)), "homes", "Codex7", "sessions", startedAt.In(time.Local).Format("2006"), startedAt.In(time.Local).Format("01"), startedAt.In(time.Local).Format("02"))
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}

	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	ref := registrarSupervisorLocalResidente(descriptorSupervisorLocal{
		Ref:             traceDir,
		PID:             cmd.Process.Pid,
		StartedAt:       startedAt,
		TraceDir:        traceDir,
		WorkingDir:      workingDir,
		RenderedCommand: "'" + wrapper + "' 'Codex7'",
	}, cmd)
	if ref == "" {
		t.Fatal("faltaba ref de supervisor")
	}

	sessionMeta := `{"timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","type":"session_meta","payload":{"id":"sess-supervisor-789","timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","cwd":"` + workingDir + `"}}` + "\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "rollout-supervisor.jsonl"), []byte(sessionMeta), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	obj := ObjetivoProceso{
		PID:          intPtr64(int64(cmd.Process.Pid)),
		HandleKind:   "process",
		HandleRef:    "process",
		MetadataJSON: `{"driver":"process_pty_cli","supervisor_ref":"` + ref + `","rendered_command":"'` + wrapper + `' 'Codex7'","working_dir":"` + workingDir + `","started_at":"` + startedAt.Format(time.RFC3339Nano) + `"}`,
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, err := DetectExternalSessionID(obj)
		if err != nil {
			t.Fatalf("DetectExternalSessionID: %v", err)
		}
		if got == "sess-supervisor-789" {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("el supervisor local no detecto external_session_id a tiempo")
}

func shellQuoteForTest(raw string) string {
	return "'" + strings.ReplaceAll(raw, "'", `'\''`) + "'"
}
