package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerCodexGoalBackendFromEnvV0TmuxPreflightOKV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(fakeCodexAppServerTmuxCommandForTestV0()), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	fakeCodex := filepath.Join(root, "codex-fake-tmux-ok")
	if err := os.WriteFile(fakeCodex, []byte(fakeCodexAppServerTmuxCodexForTestV0()), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	projectDir := filepath.Join(root, "project")
	runtimeDir := filepath.Join(root, "runtime")
	homeDir := filepath.Join(root, "home")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("mkdir source code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "config.toml"), []byte("model = \"test\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexHomeV0, homeDir)
	t.Setenv(envCodexCodeHomeV0, sourceCodeHome)
	t.Setenv(envCodexCommandV0, fakeCodex)
	t.Setenv(envCodexPathV0, binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
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
		CommandPath:       fakeCodex,
		PathEnv:           binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:        socketPath,
		SessionName:       codexAppServerTmuxSessionNameV0(config),
		HomeDir:           homeDir,
		CodeHomeDir:       codeHomePath,
		SourceCodeHomeDir: sourceCodeHome,
		Timeout:           time.Second,
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	if !strings.HasPrefix(socketPath, filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0)+string(os.PathSeparator)) ||
		!strings.HasSuffix(socketPath, ".sock") {
		t.Fatalf("socket path fuera de runtime: %q", socketPath)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(socketPath), codexAppServerTmuxMarkerFileV0)); err != nil {
		t.Fatalf("owner marker ausente: %v", err)
	}
	logRaw, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	logText := string(logRaw)
	if !strings.Contains(logText, "new-session") ||
		!strings.Contains(logText, "app-server --listen") ||
		!strings.Contains(logText, "unix://") ||
		!strings.Contains(logText, "CODEX_HOME=") ||
		strings.Contains(logText, "--stdio") {
		t.Fatalf("tmux log inesperado: %s", logText)
	}
	isolatedCodeHome := filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0, "codex-home")
	if raw, err := os.ReadFile(filepath.Join(isolatedCodeHome, "auth.json")); err != nil || string(raw) != `{"ok":true}` {
		t.Fatalf("auth proyectado raw=%q err=%v", string(raw), err)
	}
	if raw, err := os.ReadFile(filepath.Join(isolatedCodeHome, "config.toml")); err != nil || string(raw) != "model = \"test\"\n" {
		t.Fatalf("config proyectada raw=%q err=%v", string(raw), err)
	}
}

func TestCodexAppServerCommandProtocolV0RespetaTimeoutSiProxyNoRespondeV0(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "codex-hang")
	if err := os.WriteFile(script, []byte(`#!/bin/sh
trap 'exit 0' TERM INT
sleep 30
`), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: script,
		Args:        []string{"app-server", "proxy", "--sock", filepath.Join(root, "missing.sock")},
		Timeout:     50 * time.Millisecond,
	}
	started := time.Now()
	err := protocol.ProbeV0(context.Background())
	if err == nil || !strings.Contains(err.Error(), "codex_app_server_timeout") {
		t.Fatalf("err=%v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("timeout no acotado: %s", elapsed)
	}
}

func TestCodexAppServerTmuxBackendV0DetectaSesionMuertaSinEsperarSocketTimeoutV0(t *testing.T) {
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
  new-session)
    exit 0
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		CommandPath: filepath.Join(root, "codex-no-usado"),
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  filepath.Join(root, "runtime", "server.sock"),
		SessionName: "orquesta-goal-dead-session",
		Timeout:     time.Second,
	}
	err := backend.EnsureV0(context.Background(), serverCodexAppServerCommandProtocolV0{
		CommandPath: filepath.Join(root, "codex-no-usado"),
		Args:        []string{"app-server", "proxy", "--sock", filepath.Join(root, "runtime", "server.sock")},
		Timeout:     50 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_session_exited") {
		t.Fatalf("err=%v", err)
	}
}

func TestCodexAppServerTmuxSocketPathV0SeMantieneCortoV0(t *testing.T) {
	config := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		RuntimeWorkDir: "/home/alberto/Trabajo/orquesta/.orquesta-runtime-selfrepair-19038",
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta/.orquesta-selfrepair/worktree-19038",
	})
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatalf("socket path: %v", err)
	}
	if len(socketPath) >= 100 {
		t.Fatalf("socket path demasiado largo: len=%d path=%s", len(socketPath), socketPath)
	}
	if !strings.Contains(socketPath, string(os.PathSeparator)+codexAppServerTmuxDirV0+string(os.PathSeparator)+"g-") {
		t.Fatalf("socket path inesperado: %s", socketPath)
	}
}

func TestCodexAppServerTmuxStartupTimeoutV0DaMargenAlPrimerArranqueV0(t *testing.T) {
	if got := codexAppServerTmuxStartupTimeoutV0(time.Second); got != 60*time.Second {
		t.Fatalf("startup timeout=%s", got)
	}
	if got := codexAppServerTmuxStartupTimeoutV0(90 * time.Second); got != 90*time.Second {
		t.Fatalf("startup timeout override=%s", got)
	}
}

func fakeCodexAppServerTmuxCodexForTestV0() string {
	return `#!/bin/sh
if [ "$1" != "app-server" ] || [ "$2" != "proxy" ] || [ "$3" != "--sock" ] || [ -z "$4" ]; then
  echo "args inesperados: $*" >&2
  exit 1
fi
if [ ! -e "$4" ]; then
  echo "Error: failed to connect to socket at $4" >&2
  exit 1
fi
while IFS= read -r line; do
  case "$line" in
    *'"id":1'*) printf '{"id":1,"result":{}}\n' ;;
    *'"id":2'*) printf '{"id":2,"result":{"data":[]}}\n'; exit 0 ;;
  esac
done
exit 1
`
}

type fakeCodexAppServerProbeV0 struct{}

func (fakeCodexAppServerProbeV0) ProbeV0(context.Context) error {
	return nil
}

func fakeCodexAppServerTmuxCommandForTestV0() string {
	return `#!/bin/sh
set -eu
if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$ORQUESTA_TEST_TMUX_LOG"
fi
case "${1:-}" in
  has-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ] && [ -e "${ORQUESTA_TEST_TMUX_LOG}.session" ]; then
      exit 0
    fi
    exit 1
    ;;
  kill-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
      rm -f "${ORQUESTA_TEST_TMUX_LOG}.session"
    fi
    exit 0
    ;;
  new-session)
    if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
      : > "${ORQUESTA_TEST_TMUX_LOG}.session"
    fi
    sock=""
    for arg in "$@"; do
      case "$arg" in
        unix://*)
          sock="${arg#unix://}"
          ;;
        *unix://*)
          sock="${arg#*unix://}"
          sock="${sock%%\'*}"
          sock="${sock%%\"*}"
          sock="${sock%% *}"
          ;;
      esac
    done
    if [ -z "$sock" ]; then
      echo "socket missing" >&2
      exit 2
    fi
    mkdir -p "$(dirname "$sock")"
    : > "$sock"
    exit 0
    ;;
esac
echo "tmux args inesperados: $*" >&2
exit 2
`
}
