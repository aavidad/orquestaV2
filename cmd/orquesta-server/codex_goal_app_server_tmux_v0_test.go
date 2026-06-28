package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, fakeCodex)
	t.Setenv(envCodexPathV0, binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(envCodexGoalBackendV0, codexGoalBackendAppServerTmuxV0)
	t.Setenv(envCodexGoalPreflightTimeoutMSV0, "1000")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvV0(config)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvV0: %v", err)
	}
	starter, ok := backend.Starter.(serverCodexAppServerGoalBackendV0)
	if !ok {
		t.Fatalf("starter real=%T %+v", backend.Starter, backend.Starter)
	}
	protocol, ok := starter.Protocol.(serverCodexAppServerCommandProtocolV0)
	if !ok {
		t.Fatalf("protocol=%T", starter.Protocol)
	}
	if len(protocol.Args) != 4 ||
		!reflect.DeepEqual(protocol.Args[:3], []string{"app-server", "proxy", "--sock"}) {
		t.Fatalf("args=%v", protocol.Args)
	}
	socketPath := protocol.Args[3]
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
		!strings.Contains(logText, "app-server --listen unix://") ||
		strings.Contains(logText, "--stdio") {
		t.Fatalf("tmux log inesperado: %s", logText)
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

func fakeCodexAppServerTmuxCommandForTestV0() string {
	return `#!/bin/sh
set -eu
if [ -n "${ORQUESTA_TEST_TMUX_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$ORQUESTA_TEST_TMUX_LOG"
fi
case "${1:-}" in
  has-session)
    exit 1
    ;;
  new-session)
    sock=""
    for arg in "$@"; do
      case "$arg" in
        unix://*)
          sock="${arg#unix://}"
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
