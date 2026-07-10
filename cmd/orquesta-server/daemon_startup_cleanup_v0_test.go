package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestCleanupCodexGoalBackendAfterStartupFailureV0SoloConDaemonInactivoYGeneracionExactaV0(t *testing.T) {
	config, _, tmuxLog, socketPath := prepareStartupCleanupOwnedGenerationV0(t)

	err := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, stoppedServerDaemonIdentityV0(os.Getpid()))
	if err == nil || !strings.Contains(err.Error(), "daemon_process_identity_changed") {
		t.Fatalf("daemon vivo/identidad incompleta aceptado: %v", err)
	}
	assertStartupCleanupSessionPresentV0(t, tmuxLog, socketPath)

	if err := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, deadServerDaemonIdentityForTestV0(t)); err != nil {
		t.Fatalf("cleanup generacion exacta: %v", err)
	}
	assertStartupCleanupSessionGoneV0(t, tmuxLog, socketPath)
}

func TestCleanupCodexGoalBackendAfterStartupFailureV0PreservaSesionMarkerlessV0(t *testing.T) {
	root := t.TempDir()
	config, tmuxLog, socketPath := prepareStartupCleanupConfigV0(t, root)
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socketPath, []byte("foreign socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmuxLog+".session", []byte("foreign session"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, deadServerDaemonIdentityForTestV0(t)); err != nil {
		t.Fatalf("cleanup markerless debe ser no-op: %v", err)
	}
	assertStartupCleanupSessionPresentV0(t, tmuxLog, socketPath)
}

func TestCleanupCodexGoalBackendAfterStartupFailureV0NoTocaUsoAppAjenaV0(t *testing.T) {
	config, _, _ := prepareStartupCleanupConfigV0(t, t.TempDir())
	foreignRuntime, err := os.MkdirTemp(os.TempDir(), "oq-gsrv-uso-app-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(foreignRuntime)
	process := exec.Command("bash", "-c", "exec -a 'uso-app codex app-server' sleep 30")
	process.Dir = foreignRuntime
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = process.Process.Kill(); _ = process.Wait() }()

	if err := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, deadServerDaemonIdentityForTestV0(t)); err != nil {
		t.Fatalf("cleanup sin marker propio: %v", err)
	}
	if err := syscall.Kill(process.Process.Pid, 0); err != nil {
		t.Fatalf("uso-app ajena tocada: %v", err)
	}
}

func TestCleanupDetachedDaemonAfterStartupFailureV0PropagaErrorDeSignalV0(t *testing.T) {
	config, _, tmuxLog, socketPath := prepareStartupCleanupOwnedGenerationV0(t)
	process, identity := startDetachedCleanupProcessForTestV0(t)
	defer stopDetachedCleanupProcessForTestV0(process, identity)
	originalSignal := serverStartupCleanupSignalProcessV0
	serverStartupCleanupSignalProcessV0 = func(serverDaemonProcessIdentityV0) error {
		return fmt.Errorf("daemon_process_signal_failed")
	}
	defer func() { serverStartupCleanupSignalProcessV0 = originalSignal }()

	err := cleanupDetachedDaemonAfterStartupFailureV0(config, identity)
	if err == nil || !strings.Contains(err.Error(), "daemon_process_signal_failed") {
		t.Fatalf("error de signal ocultado: %v", err)
	}
	assertStartupCleanupSessionPresentV0(t, tmuxLog, socketPath)
}

func TestCleanupDetachedDaemonAfterStartupFailureV0RechazaIdentidadCambiadaV0(t *testing.T) {
	config, _, tmuxLog, socketPath := prepareStartupCleanupOwnedGenerationV0(t)
	process, identity := startDetachedCleanupProcessForTestV0(t)
	defer stopDetachedCleanupProcessForTestV0(process, identity)
	identity.StartRef += "-changed"

	err := cleanupDetachedDaemonAfterStartupFailureV0(config, identity)
	if err == nil || !strings.Contains(err.Error(), "daemon_process_identity_changed") {
		t.Fatalf("identidad cambiada aceptada: %v", err)
	}
	assertStartupCleanupSessionPresentV0(t, tmuxLog, socketPath)
}

func TestCleanupCodexGoalBackendAfterStartupFailureV0NoEntraConGrupoVivoV0(t *testing.T) {
	config, _, tmuxLog, socketPath := prepareStartupCleanupOwnedGenerationV0(t)
	process, identity := startDetachedCleanupProcessForTestV0(t)
	defer stopDetachedCleanupProcessForTestV0(process, identity)

	err := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, identity)
	if err == nil || !strings.Contains(err.Error(), "daemon_process_group_still_active") {
		t.Fatalf("grupo vivo aceptado: %v", err)
	}
	assertStartupCleanupSessionPresentV0(t, tmuxLog, socketPath)
}

func TestCleanupCodexGoalBackendAfterStartupFailureV0PropagaErrorCleanupR8V0(t *testing.T) {
	config, _, tmuxLog, socketPath := prepareStartupCleanupOwnedGenerationV0(t)
	if err := os.WriteFile(tmuxLog+".kill-fail", []byte("fail"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, deadServerDaemonIdentityForTestV0(t))
	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_kill_failed") {
		t.Fatalf("error cleanup R8 ocultado: %v", err)
	}
	assertStartupCleanupSessionPresentV0(t, tmuxLog, socketPath)
}

func prepareStartupCleanupOwnedGenerationV0(
	t *testing.T,
) (orquestaserver.ConfigV0, serverCodexAppServerTmuxBackendV0, string, string) {
	t.Helper()
	config, tmuxLog, socketPath := prepareStartupCleanupConfigV0(t, t.TempDir())
	codeHomePath, err := codexAppServerTmuxCodeHomePathV0(config)
	if err != nil {
		t.Fatal(err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv: os.Getenv(envCodexPathV0), SocketPath: socketPath,
		SessionName: codexAppServerTmuxSessionNameV0(config), CodeHomeDir: codeHomePath,
		RuntimeWorkDir: config.RuntimeWorkDir, Timeout: 2 * time.Second,
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		log, _ := os.ReadFile(tmuxLog)
		helperLog, _ := os.ReadFile(tmuxLog + ".helper.log")
		t.Fatalf("EnsureV0 R8: %v\ntmux=%s\nhelper=%s", err, log, helperLog)
	}
	t.Cleanup(func() {
		_ = os.Remove(tmuxLog + ".kill-fail")
		_ = backend.ShutdownForcedStopV0(context.Background())
	})
	return config, backend, tmuxLog, socketPath
}

func prepareStartupCleanupConfigV0(
	t *testing.T,
	root string,
) (orquestaserver.ConfigV0, string, string) {
	t.Helper()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(fakeCodexAppServerTmuxCommandForTestV0()), 0o700); err != nil {
		t.Fatal(err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv("ORQUESTA_TEST_BINARY", os.Args[0])
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexPathV0, binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "2000")
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatal(err)
	}
	return config, tmuxLog, socketPath
}

func deadServerDaemonIdentityForTestV0(t *testing.T) serverDaemonProcessIdentityV0 {
	t.Helper()
	for _, pid := range []int{99999999, 99999998, 99999997} {
		identity := stoppedServerDaemonIdentityV0(pid)
		if active, err := serverDaemonProcessGroupActiveV0(identity); err == nil && !active {
			return identity
		}
	}
	t.Skip("no hay identidad daemon inactiva verificable")
	return serverDaemonProcessIdentityV0{}
}

func startDetachedCleanupProcessForTestV0(t *testing.T) (*exec.Cmd, serverDaemonProcessIdentityV0) {
	t.Helper()
	process := exec.Command("bash", "-c", "trap '' INT TERM; while :; do sleep 1; done")
	process.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	identity, err := captureServerDaemonProcessIdentityV0(process.Process.Pid)
	if err != nil {
		_ = process.Process.Kill()
		_ = process.Wait()
		t.Fatal(err)
	}
	return process, identity
}

func stopDetachedCleanupProcessForTestV0(process *exec.Cmd, identity serverDaemonProcessIdentityV0) {
	_ = syscall.Kill(-identity.GroupID, syscall.SIGKILL)
	_ = process.Wait()
}

func assertStartupCleanupSessionPresentV0(t *testing.T, tmuxLog string, socketPath string) {
	t.Helper()
	if _, err := os.Lstat(tmuxLog + ".session"); err != nil {
		t.Fatalf("sesion propia ausente: %v", err)
	}
	if _, err := os.Lstat(socketPath); err != nil {
		t.Fatalf("socket propio ausente: %v", err)
	}
}

func assertStartupCleanupSessionGoneV0(t *testing.T, tmuxLog string, socketPath string) {
	t.Helper()
	if _, err := os.Lstat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("sesion propia no eliminada: %v", err)
	}
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket propio no eliminado: %v", err)
	}
}
