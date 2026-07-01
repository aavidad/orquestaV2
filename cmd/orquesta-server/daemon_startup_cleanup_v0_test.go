package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0SoloMataSiProcesoCayo(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(fakeCodexAppServerTmuxCommandForTestV0()), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	pathEnv := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexPathV0, pathEnv)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatalf("socket path: %v", err)
	}
	codeHomePath, err := codexAppServerTmuxCodeHomePathV0(config)
	if err != nil {
		t.Fatalf("code home path: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:        pathEnv,
		SocketPath:     socketPath,
		SessionName:    codexAppServerTmuxSessionNameV0(config),
		CodeHomeDir:    codeHomePath,
		RuntimeWorkDir: runtimeDir,
		Timeout:        time.Second,
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}

	cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, os.Getpid())
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("sesion no debe limpiarse mientras el daemon sigue vivo: %v", err)
	}

	cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, -1)
	rawLog, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("tmux no recibio kill-session tras caida de daemon: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake no eliminada err=%v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
	if _, err := os.Stat(backend.tmuxOwnerMarkerPathV0()); !os.IsNotExist(err) {
		t.Fatalf("owner marker no eliminado err=%v", err)
	}
}
