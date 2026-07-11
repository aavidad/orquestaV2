package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMainV0WithoutCommandReturnsUsageError(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	if code := runMain(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "comando requerido") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRunServerCommandV0DegradedIdentityPrecedesCodexCommandPreflightV0(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	runtimeDir := filepath.Join(root, "runtime")
	t.Setenv(envCodexProjectWorkDirV0, root)
	t.Setenv(envServerWorktreeV0, root)
	t.Setenv(envServerAddrV0, "127.0.0.1:not-a-port")
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(root, "missing-codex"))

	var stderr strings.Builder
	if code := runServerCommandV0(nil, nil, &stderr); code != 1 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "reason_code=degraded_identity_listen_failed") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	if strings.Contains(stderr.String(), "codex_command_") {
		t.Fatalf("identidad degradada quedo bloqueada por Codex: stderr=%q", stderr.String())
	}
	for _, path := range []string{stateDir, runtimeDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("run creo %s antes de servir identidad degradada: %v", path, err)
		}
	}
}
