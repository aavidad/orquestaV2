package main

import (
	"context"
	"os"
	"os/exec"
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

func TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0MataSesionConfiguradaSinOwnerMarker(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("stale socket"), 0o600); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	if err := os.WriteFile(tmuxLog+".session", []byte("stale session"), 0o600); err != nil {
		t.Fatalf("write fake session: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  socketPath,
		SessionName: codexAppServerTmuxSessionNameV0(config),
	}
	if _, err := os.Stat(backend.tmuxOwnerMarkerPathV0()); !os.IsNotExist(err) {
		t.Fatalf("owner marker debe estar ausente err=%v", err)
	}

	cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, -1)

	rawLog, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("tmux no recibio kill-session para huerfano configurado: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake no eliminada err=%v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
}

func TestCleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0MataProcesoPropioSinSocketV0(t *testing.T) {
	if _, err := os.Stat("/proc/self/cwd"); err != nil {
		t.Skip("procfs no disponible")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
case "${1:-}" in
  has-session)
    exit 1
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	ownedWorkdir := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "owned-cwd")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(ownedWorkdir, 0o700); err != nil {
		t.Fatalf("mkdir owned workdir: %v", err)
	}
	pathEnv := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexPathV0, pathEnv)
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "200")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	process := exec.Command("bash", "-c", "trap '' TERM; exec -a 'codex app-server' sleep 30")
	process.Dir = ownedWorkdir
	if err := process.Start(); err != nil {
		t.Fatalf("start fake app-server process: %v", err)
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

	cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, os.Getpid())
	select {
	case <-processDone:
		t.Fatalf("cleanup no debe matar proceso propio mientras daemon sigue vivo")
	default:
	}

	cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, -1)
	select {
	case <-processDone:
	case <-time.After(time.Second):
		t.Fatalf("fake app-server runtime-owned sigue vivo tras cleanup de startup")
	}
}
