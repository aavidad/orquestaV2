package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runtimeConnectorScriptPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Join(filepath.Dir(wd), "scripts", "runtime_connector.sh")
}

func TestRuntimeConnectorDetectExternalSessionIDIgnoraSQLiteLocal(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := runtimeConnectorScriptPath(t)
	fakeSQLite := filepath.Join(tmp, "sqlite3")
	stateDB := filepath.Join(tmp, "state_5.sqlite")

	if err := os.WriteFile(fakeSQLite, []byte("#!/usr/bin/env bash\necho sess-from-sqlite\n"), 0o755); err != nil {
		t.Fatalf("write fake sqlite3: %v", err)
	}
	if err := os.WriteFile(stateDB, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write fake state db: %v", err)
	}

	cmd := exec.Command("bash", "-lc", `. "`+scriptPath+`"; runtime_detect_external_session_id codex "/tmp/work" "123456" "BOOT" "main" "fallback-id"`)
	cmd.Env = append(os.Environ(),
		"PATH="+tmp+string(os.PathListSeparator)+os.Getenv("PATH"),
		"CODEX_STATE_DB="+stateDB,
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("runtime_detect_external_session_id: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "fallback-id" {
		t.Fatalf("detect deberia ignorar sqlite local, got %q", got)
	}
}

func TestRuntimeConnectorDetectExternalSessionIDUsaHook(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := runtimeConnectorScriptPath(t)
	hookPath := filepath.Join(tmp, "hook.sh")

	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env bash\nif [[ \"$1\" == \"detect\" ]]; then\n  echo sess-from-hook\nfi\n"), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	cmd := exec.Command("bash", "-lc", `. "`+scriptPath+`"; runtime_detect_external_session_id codex "/tmp/work" "123456" "BOOT" "main" "fallback-id"`)
	cmd.Env = append(os.Environ(), "ORQUESTA_RUNTIME_CONNECTOR_SCRIPT="+hookPath)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("runtime_detect_external_session_id hook: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "sess-from-hook" {
		t.Fatalf("detect con hook inesperado: %q", got)
	}
}
