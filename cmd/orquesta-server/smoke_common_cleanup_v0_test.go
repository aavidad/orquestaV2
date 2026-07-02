package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSmokeCommonShutdownCleanupMataAppServerPropioSinBaseURLV0(t *testing.T) {
	if _, err := os.Stat("/proc/self/cwd"); err != nil {
		t.Skip("procfs no disponible")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash no disponible")
	}
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	runtimeDir := t.TempDir()
	goalDir := filepath.Join(runtimeDir, "goal-srv", "owned-cwd")
	if err := os.MkdirAll(goalDir, 0o700); err != nil {
		t.Fatalf("mkdir goal dir: %v", err)
	}
	process := exec.Command("bash", "-c", "trap '' TERM; exec -a 'codex app-server' sleep 30")
	process.Dir = goalDir
	if err := process.Start(); err != nil {
		t.Fatalf("start fake app-server: %v", err)
	}
	processDone := make(chan error, 1)
	go func() {
		processDone <- process.Wait()
	}()
	defer func() {
		_ = process.Process.Kill()
		select {
		case <-processDone:
		case <-time.After(time.Second):
		}
	}()

	cleanup := exec.Command(
		"bash",
		"-c",
		`source scripts/lib/smoke_common.sh; smoke_shutdown_orquesta_server "" "" 1 1 "$ORQUESTA_TEST_RUNTIME_DIR"`,
	)
	cleanup.Dir = root
	cleanup.Env = append(os.Environ(), "ORQUESTA_TEST_RUNTIME_DIR="+runtimeDir)
	if output, err := cleanup.CombinedOutput(); err != nil {
		t.Fatalf("cleanup failed: %v output=%s", err, string(output))
	}
	select {
	case <-processDone:
	case <-time.After(time.Second):
		t.Fatalf("fake app-server propio sigue vivo tras cleanup sin base_url")
	}
}
