package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
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
		RuntimeWorkDir:    runtimeDir,
		ProjectWorkDir:    projectDir,
		SourceCodeHomeDir: sourceCodeHome,
		Timeout:           time.Second,
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	if !strings.HasSuffix(socketPath, ".sock") ||
		(!strings.HasPrefix(socketPath, filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0)+string(os.PathSeparator)) &&
			socketPath != codexAppServerTmuxShortSocketPathV0(config)) {
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
	if raw, err := os.ReadFile(filepath.Join(isolatedCodeHome, "config.toml")); err != nil ||
		!strings.Contains(string(raw), "model = \"test\"") ||
		!strings.Contains(string(raw), `[projects."`+projectDir+`"]`) {
		t.Fatalf("config proyectada raw=%q err=%v", string(raw), err)
	}
}

func TestCodexAppServerTmuxBackendV0PreparaCodeHomeDesdeCODEXHOMEResueltoV0(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	codeHome := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "codex-home")
	sourceCodeHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("mkdir source code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "auth.json"), []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"redacted"}}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "config.toml"), []byte("model = \"test\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CODEX_HOME", sourceCodeHome)
	t.Setenv(envCodexCodeHomeV0, "")
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:       codeHome,
		RuntimeWorkDir:    runtimeDir,
		ProjectWorkDir:    filepath.Join(root, "project"),
		SourceCodeHomeDir: codeHomeDirV0(),
	}

	if err := backend.prepareTmuxCodeHomeV0(); err != nil {
		t.Fatalf("prepareTmuxCodeHomeV0: %v", err)
	}
	if raw, err := os.ReadFile(filepath.Join(codeHome, "auth.json")); err != nil ||
		!strings.Contains(string(raw), `"auth_mode":"chatgpt"`) {
		t.Fatalf("auth no proyectado desde CODEX_HOME raw=%q err=%v", string(raw), err)
	}
	if raw, err := os.ReadFile(filepath.Join(codeHome, "config.toml")); err != nil ||
		!strings.Contains(string(raw), "model = \"test\"") ||
		!strings.Contains(string(raw), `[projects."`+filepath.Join(root, "project")+`"]`) {
		t.Fatalf("config no proyectada desde CODEX_HOME raw=%q err=%v", string(raw), err)
	}
}

func TestCodexAppServerTmuxBackendV0FiltraProjectsAjenosDelConfigV0(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	projectDir := filepath.Join(root, "project")
	codeHome := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "codex-home")
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("mkdir source code home: %v", err)
	}
	sourceConfig := strings.Join([]string{
		`model = "test"`,
		``,
		`[projects."/workspace/project"]`,
		`trust_level = "trusted"`,
		``,
		`[projects."/srv/orquesta-self/worktrees/orquesta"]`,
		`trust_level = "trusted"`,
		``,
		`[tui.model_availability_nux]`,
		`"gpt-5.5" = 4`,
		``,
	}, "\n")
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "auth.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "config.toml"), []byte(sourceConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:       codeHome,
		RuntimeWorkDir:    runtimeDir,
		ProjectWorkDir:    projectDir,
		SourceCodeHomeDir: sourceCodeHome,
	}

	if err := backend.prepareTmuxCodeHomeV0(); err != nil {
		t.Fatalf("prepareTmuxCodeHomeV0: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(codeHome, "config.toml"))
	if err != nil {
		t.Fatalf("read config proyectada: %v", err)
	}
	config := string(raw)
	if strings.Contains(config, "/workspace/project") ||
		strings.Contains(config, "/srv/orquesta-self/worktrees/orquesta") {
		t.Fatalf("config conserva proyectos ajenos: %s", config)
	}
	if !strings.Contains(config, `model = "test"`) ||
		!strings.Contains(config, `[tui.model_availability_nux]`) ||
		!strings.Contains(config, `[projects."`+projectDir+`"]`) ||
		!strings.Contains(config, `trust_level = "trusted"`) {
		t.Fatalf("config no conserva globales o proyecto aislado: %s", config)
	}
}

func TestServerCodexGoalBackendFromEnvV0TmuxSinAuthDegradaYConservaShutdownV0(t *testing.T) {
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
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("CODEX_API_KEY", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	backend, err := serverCodexGoalBackendFromEnvForWorkDirV0(config, projectDir)
	if err != nil {
		t.Fatalf("serverCodexGoalBackendFromEnvForWorkDirV0: %v", err)
	}

	if got := serverCodexGoalBackendUnavailableIssueCodeV0(backend); got != "codex_app_server_auth_missing" {
		t.Fatalf("issue=%q", got)
	}
	if backend.ShutdownHook == nil {
		t.Fatalf("backend degradado debe conservar shutdown hook para limpiar app-server tmux")
	}
	isolatedCodeHome := filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0, "codex-home")
	if _, err := os.Stat(filepath.Join(isolatedCodeHome, "auth.json")); !os.IsNotExist(err) {
		t.Fatalf("auth inesperado err=%v", err)
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

func TestCodexAppServerTmuxBackendV0PreparaCodeHomeLimpioV0(t *testing.T) {
	root := t.TempDir()
	sourceCodeHome := filepath.Join(root, "source-codex-home")
	codeHome := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "codex-home")
	if err := os.MkdirAll(sourceCodeHome, 0o700); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(codeHome, "skills", ".system"), 0o700); err != nil {
		t.Fatalf("mkdir dirty skills: %v", err)
	}
	for _, path := range []string{
		filepath.Join(codeHome, ".tmp", "plugins-1", "plugin.json"),
		filepath.Join(codeHome, "cache", "codex_apps_1", "entry.json"),
		filepath.Join(codeHome, "app-server-control", "app-server-startup.lock"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("mkdir dirty path %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("dirty"), 0o600); err != nil {
			t.Fatalf("write dirty path %s: %v", path, err)
		}
	}
	for _, path := range []string{
		filepath.Join(codeHome, "skills", ".system", "SKILL.md"),
		filepath.Join(codeHome, "state_5.sqlite"),
		filepath.Join(codeHome, "goals_1.sqlite"),
		filepath.Join(codeHome, "logs_2.sqlite"),
		filepath.Join(codeHome, "models_cache.json"),
	} {
		if err := os.WriteFile(path, []byte("dirty"), 0o600); err != nil {
			t.Fatalf("write dirty file %s: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "auth.json"), []byte(`{"token":"ok"}`), 0o644); err != nil {
		t.Fatalf("write auth source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "config.toml"), []byte("model = \"test\"\n"), 0o644); err != nil {
		t.Fatalf("write config source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceCodeHome, "models_cache.json"), []byte("source dirty"), 0o600); err != nil {
		t.Fatalf("write source mutable: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:       codeHome,
		RuntimeWorkDir:    filepath.Join(root, "runtime"),
		SourceCodeHomeDir: sourceCodeHome,
	}

	if err := backend.prepareTmuxCodeHomeV0(); err != nil {
		t.Fatalf("prepareTmuxCodeHomeV0: %v", err)
	}

	for _, path := range []string{
		filepath.Join(codeHome, "skills"),
		filepath.Join(codeHome, ".tmp"),
		filepath.Join(codeHome, "cache"),
		filepath.Join(codeHome, "app-server-control"),
		filepath.Join(codeHome, "state_5.sqlite"),
		filepath.Join(codeHome, "goals_1.sqlite"),
		filepath.Join(codeHome, "logs_2.sqlite"),
		filepath.Join(codeHome, "models_cache.json"),
	} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("estado mutable no limpiado path=%s err=%v", path, err)
		}
	}
	assertProjectedModeV0(t, filepath.Join(codeHome, "auth.json"), 0o600)
	assertProjectedModeV0(t, filepath.Join(codeHome, "config.toml"), 0o600)
	if raw, err := os.ReadFile(filepath.Join(codeHome, "auth.json")); err != nil || string(raw) != `{"token":"ok"}` {
		t.Fatalf("auth proyectado raw=%q err=%v", string(raw), err)
	}
	if raw, err := os.ReadFile(filepath.Join(codeHome, "config.toml")); err != nil || string(raw) != "model = \"test\"\n" {
		t.Fatalf("config proyectada raw=%q err=%v", string(raw), err)
	}
}

func TestCodexAppServerTmuxBackendV0PreparaCodeHomeConservaCredencialesActualesSiNoHayFuenteV0(t *testing.T) {
	root := t.TempDir()
	codeHome := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "codex-home")
	if err := os.MkdirAll(filepath.Join(codeHome, "cache"), 0o700); err != nil {
		t.Fatalf("mkdir code home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "auth.json"), []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "config.toml"), []byte("approval_policy = \"never\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codeHome, "cache", "old.json"), []byte("dirty"), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:    codeHome,
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}

	if err := backend.prepareTmuxCodeHomeV0(); err != nil {
		t.Fatalf("prepareTmuxCodeHomeV0: %v", err)
	}

	if raw, err := os.ReadFile(filepath.Join(codeHome, "auth.json")); err != nil || string(raw) != `{"old":true}` {
		t.Fatalf("auth actual no conservado raw=%q err=%v", string(raw), err)
	}
	if raw, err := os.ReadFile(filepath.Join(codeHome, "config.toml")); err != nil || string(raw) != "approval_policy = \"never\"\n" {
		t.Fatalf("config actual no conservada raw=%q err=%v", string(raw), err)
	}
	if _, err := os.Lstat(filepath.Join(codeHome, "cache")); !os.IsNotExist(err) {
		t.Fatalf("cache no limpiada err=%v", err)
	}
}

func TestCodexAppServerTmuxBackendV0PreparaCodeHomeRechazaSymlinkV0(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	goalDir := filepath.Join(root, "runtime", codexAppServerTmuxDirV0)
	if err := os.MkdirAll(goalDir, 0o700); err != nil {
		t.Fatalf("mkdir goal dir: %v", err)
	}
	codeHome := filepath.Join(goalDir, "codex-home")
	if err := os.Symlink(target, codeHome); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:    codeHome,
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}

	err := backend.prepareTmuxCodeHomeV0()

	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_code_home_unsafe") {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Lstat(codeHome); statErr != nil {
		t.Fatalf("symlink destino fue eliminado: %v", statErr)
	}
	if _, statErr := os.Stat(target); statErr != nil {
		t.Fatalf("target externo fue tocado: %v", statErr)
	}
}

func TestCodexAppServerTmuxBackendV0PreparaCodeHomeRechazaRutaNoGoalSrvV0(t *testing.T) {
	root := t.TempDir()
	codeHome := filepath.Join(root, "runtime", "codex-home")
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:    codeHome,
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}

	err := backend.prepareTmuxCodeHomeV0()

	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_code_home_unsafe") {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Lstat(codeHome); !os.IsNotExist(statErr) {
		t.Fatalf("ruta insegura fue creada err=%v", statErr)
	}
}

func TestCodexAppServerTmuxBackendV0PreparaCodeHomeRechazaDestinoFueraDeRuntimeV0(t *testing.T) {
	root := t.TempDir()
	codeHome := filepath.Join(root, "other-runtime", codexAppServerTmuxDirV0, "codex-home")
	backend := serverCodexAppServerTmuxBackendV0{
		CodeHomeDir:    codeHome,
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	}

	err := backend.prepareTmuxCodeHomeV0()

	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_code_home_unsafe") {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Lstat(codeHome); !os.IsNotExist(statErr) {
		t.Fatalf("destino externo fue creado err=%v", statErr)
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

func TestCodexAppServerTmuxBackendV0LimpiaSesionSiSocketNoLlegaV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
set -eu
log="${ORQUESTA_TEST_TMUX_LOG:-}"
if [ -n "$log" ]; then
  printf '%s\n' "$*" >> "$log"
fi
case "${1:-}" in
  has-session)
    [ -n "$log" ] && [ -e "${log}.session" ] && exit 0
    exit 1
    ;;
  new-session)
    [ -n "$log" ] && : > "${log}.session"
    exit 0
    ;;
  kill-session)
    [ -n "$log" ] && rm -f "${log}.session"
    exit 0
    ;;
  display-message)
    exit 0
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-timeout.sock")
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:     socketPath,
		SessionName:    "orquesta-goal-socket-timeout",
		CodeHomeDir:    filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "codex-home"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		Timeout:        80 * time.Millisecond,
	}

	err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{})

	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_socket_timeout") {
		t.Fatalf("err=%v", err)
	}
	logRaw, readErr := os.ReadFile(tmuxLog)
	if readErr != nil {
		t.Fatalf("read tmux log: %v", readErr)
	}
	if !strings.Contains(string(logRaw), "new-session") ||
		!strings.Contains(string(logRaw), "kill-session") {
		t.Fatalf("tmux log sin cleanup: %s", string(logRaw))
	}
	if _, statErr := os.Stat(tmuxLog + ".session"); !os.IsNotExist(statErr) {
		t.Fatalf("session fake no eliminada err=%v", statErr)
	}
	if _, statErr := os.Stat(backend.tmuxOwnerMarkerPathV0()); !os.IsNotExist(statErr) {
		t.Fatalf("owner marker no eliminado err=%v", statErr)
	}
	if _, statErr := os.Stat(socketPath); !os.IsNotExist(statErr) {
		t.Fatalf("socket no eliminado err=%v", statErr)
	}
}

func TestCodexAppServerTmuxBackendV0ReparaSocketExistenteAntesDeReusarSesionV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
case "${1:-}" in
  has-session)
    exit 0
    ;;
  kill-session|new-session)
    echo "no debe reiniciar sesion" >&2
    exit 2
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	socketDir := filepath.Join(root, "runtime", codexAppServerTmuxDirV0)
	socketPath := filepath.Join(socketDir, "g-existing.sock")
	if err := os.MkdirAll(socketDir, 0o777); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("fake socket"), 0o666); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	if err := os.Chmod(socketPath, 0o666); err != nil {
		t.Fatalf("chmod socket: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-existing",
		Timeout:     time.Second,
	}
	if err := backend.writeTmuxOwnerMarkerV0(); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	if err := os.Chmod(socketDir, 0o777); err != nil {
		t.Fatalf("chmod socket dir: %v", err)
	}
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	socketInfo, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if got := socketInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("socket mode=%#o", got)
	}
	dirInfo, err := os.Stat(socketDir)
	if err != nil {
		t.Fatalf("stat socket dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("socket dir mode=%#o", got)
	}
}

func TestCodexAppServerTmuxBackendV0ShutdownMataSesionPropiaV0(t *testing.T) {
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
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-shutdown.sock")
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-shutdown",
		Timeout:     time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}
	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0: %v", err)
	}
	logRaw, err := os.ReadFile(tmuxLog)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	if !strings.Contains(string(logRaw), "kill-session") {
		t.Fatalf("tmux shutdown no mato sesion: %s", string(logRaw))
	}
	if !strings.Contains(string(logRaw), "display-message") {
		t.Fatalf("tmux shutdown no consulto pane pid: %s", string(logRaw))
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
	if _, err := os.Stat(backend.tmuxOwnerMarkerPathV0()); !os.IsNotExist(err) {
		t.Fatalf("owner marker no eliminado err=%v", err)
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake no eliminada err=%v", err)
	}
}

func TestCodexAppServerTmuxBackendV0ShutdownNormalNoMataSesionSinOwnerMarkerV0(t *testing.T) {
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
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-unowned.sock")
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
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-unowned-normal",
		Timeout:     time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)

	if err := backend.ShutdownV0(context.Background()); err != nil {
		t.Fatalf("ShutdownV0: %v", err)
	}
	if rawLog, err := os.ReadFile(tmuxLog); err == nil && strings.Contains(string(rawLog), "kill-session") {
		t.Fatalf("shutdown normal mato sesion sin owner marker: %s", string(rawLog))
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("session fake no debe eliminarse sin owner marker: %v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket no debe eliminarse sin owner marker: %v", err)
	}
}

func TestCodexAppServerTmuxBackendV0CleanupActiveWorkLimpiaSesionPropiaV0(t *testing.T) {
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
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-cleanup.sock")
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-cleanup-1234567890",
		Timeout:     time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	if err := backend.EnsureV0(context.Background(), fakeCodexAppServerProbeV0{}); err != nil {
		t.Fatalf("EnsureV0: %v", err)
	}

	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
		ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
			Kind:    "goal_backend",
			WorkRef: backend.SessionName,
			Status:  orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
		}},
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
	if result.CleanedWorkCount != 1 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned") {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
	if _, err := os.Stat(backend.tmuxOwnerMarkerPathV0()); !os.IsNotExist(err) {
		t.Fatalf("owner marker no eliminado err=%v", err)
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake no eliminada err=%v", err)
	}
}

func TestCodexAppServerTmuxBackendV0CleanupActiveWorkMataProcesoSocketSinSesionV0(t *testing.T) {
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
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-cleanup-process.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("stale socket"), 0o600); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	process := exec.Command("bash", "-c", "trap '' TERM; exec -a 'codex app-server --listen unix://"+socketPath+"' sleep 30")
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for len(codexAppServerTmuxSocketProcessPIDsV0(ctx, socketPath)) == 0 {
		select {
		case <-ctx.Done():
			t.Fatalf("fake app-server process no detectado por socket")
		case <-time.After(20 * time.Millisecond):
		}
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:                binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:             socketPath,
		SessionName:            "orquesta-goal-cleanup-process-1234567890",
		Timeout:                50 * time.Millisecond,
		ShutdownCleanupTimeout: 250 * time.Millisecond,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)

	before, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0 before: %v", err)
	}
	if len(before.ActiveWorks) != 1 ||
		!containsStringForTestV0(before.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-cmdline") {
		t.Fatalf("before=%+v", before)
	}
	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
		ActiveWorks:         before.ActiveWorks,
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
	if result.CleanedWorkCount != 1 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-processDone:
	case <-time.After(time.Second):
		t.Fatalf("fake app-server process sigue vivo tras cleanup")
	}
	after, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0 after: %v", err)
	}
	if len(after.ActiveWorks) != 0 {
		t.Fatalf("after=%+v", after)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
}

func TestCodexAppServerTmuxBackendV0CleanupActiveWorkContinuaTrasPaneTimeoutV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	tmuxLog := filepath.Join(root, "tmux.log")
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
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
  display-message)
    printf '%s\n' "${ORQUESTA_TEST_PANE_PID:-}"
    exit 0
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-cleanup-pane.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("stale socket"), 0o600); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	process := exec.Command("bash", "-c", "trap '' TERM; exec -a 'codex app-server --listen unix://"+socketPath+"' sleep 30")
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
	if err := os.WriteFile(tmuxLog+".session", []byte("stale session"), 0o600); err != nil {
		t.Fatalf("write fake session: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:                binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:             socketPath,
		SessionName:            "orquesta-goal-cleanup-pane-1234567890",
		Timeout:                50 * time.Millisecond,
		ShutdownCleanupTimeout: 200 * time.Millisecond,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)
	t.Setenv("ORQUESTA_TEST_PANE_PID", strconv.Itoa(process.Process.Pid))
	if err := backend.writeTmuxOwnerMarkerV0(); err != nil {
		t.Fatalf("write owner marker: %v", err)
	}

	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
		ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
			Kind:    "goal_backend",
			WorkRef: backend.SessionName,
			Status:  orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
		}},
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
	if result.CleanedWorkCount != 1 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned") {
		t.Fatalf("result=%+v", result)
	}
	select {
	case <-processDone:
	case <-time.After(time.Second):
		t.Fatalf("fake app-server process sigue vivo tras cleanup")
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket no eliminado err=%v", err)
	}
	if _, err := os.Stat(backend.tmuxOwnerMarkerPathV0()); !os.IsNotExist(err) {
		t.Fatalf("owner marker no eliminado err=%v", err)
	}
}

func TestCodexAppServerTmuxBackendV0CleanupActiveWorkNoMataSesionSinOwnerNiConfigSeguraV0(t *testing.T) {
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
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-unowned.sock")
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
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  socketPath,
		SessionName: "external-session",
		Timeout:     time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)

	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
		ActiveWorks: []orquestaservershutdown.ActiveShutdownWorkV0{{
			Kind:    "goal_backend",
			WorkRef: backend.SessionName,
			Status:  orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
		}},
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
	if result.CleanedWorkCount != 0 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(tmuxLog + ".session"); err != nil {
		t.Fatalf("session fake no debe eliminarse sin owner/config segura: %v", err)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket no debe eliminarse sin owner/config segura: %v", err)
	}
}

func TestCodexAppServerTmuxBackendV0CleanupActiveWorkLimpiaOwnerMarkerRecuperableEscaneadoV0(t *testing.T) {
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
	runtimeDir := filepath.Join(root, "runtime")
	ownerDir := filepath.Join(runtimeDir, codexAppServerTmuxDirV0)
	if err := os.MkdirAll(ownerDir, 0o700); err != nil {
		t.Fatalf("mkdir owner dir: %v", err)
	}
	scannedSession := "orquesta-goal-scanned-cleanup-1234567890"
	markerPath := filepath.Join(ownerDir, codexAppServerTmuxMarkerFileV0)
	marker := `{
  "schema_version": "orquesta_codex_app_server_tmux_owner.v0",
  "owner_ref": "orquesta-codex-goal-app-server-tmux-v0",
  "session_name": "` + scannedSession + `",
  "socket_ref": "socket-ref-codex-goal-app-server-tmux"
}
`
	if err := os.WriteFile(markerPath, []byte(marker), 0o600); err != nil {
		t.Fatalf("write owner marker: %v", err)
	}
	scannedSocket := filepath.Join(ownerDir, "g-scanned.sock")
	if err := os.WriteFile(scannedSocket, []byte("stale socket"), 0o600); err != nil {
		t.Fatalf("write scanned socket: %v", err)
	}
	if err := os.WriteFile(tmuxLog+".session", []byte("stale session"), 0o600); err != nil {
		t.Fatalf("write fake session: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:     filepath.Join(root, "other", "s.sock"),
		SessionName:    "external-session",
		RuntimeWorkDir: runtimeDir,
		Timeout:        time.Second,
	}
	t.Setenv("ORQUESTA_TEST_TMUX_LOG", tmuxLog)

	before, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0 before: %v", err)
	}
	if len(before.ActiveWorks) != 1 ||
		before.ActiveWorks[0].WorkRef != scannedSession ||
		!containsStringForTestV0(before.EvidenceRefs, "evidence-ref-codex-app-server-tmux-owner-scan") {
		t.Fatalf("before=%+v", before)
	}

	result, err := backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
		ActiveWorks:         before.ActiveWorks,
	})
	if err != nil {
		t.Fatalf("CleanupActiveShutdownWorkV0: %v", err)
	}
	if result.CleanedWorkCount != 1 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-owner-cleaned") {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("owner marker escaneado no eliminado err=%v", err)
	}
	if _, err := os.Stat(scannedSocket); !os.IsNotExist(err) {
		t.Fatalf("socket escaneado no eliminado err=%v", err)
	}
	if _, err := os.Stat(tmuxLog + ".session"); !os.IsNotExist(err) {
		t.Fatalf("session fake escaneada no eliminada err=%v", err)
	}
	after, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0 after: %v", err)
	}
	if len(after.ActiveWorks) != 0 {
		t.Fatalf("after=%+v", after)
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaRestosPropiosV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
case "${1:-}" in
  has-session)
    exit 0
    ;;
  display-message)
    printf '%s\n' "${ORQUESTA_TEST_PANE_PID:-}"
    exit 0
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	sleep := exec.Command("sleep", "30")
	if err := sleep.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = sleep.Process.Kill()
		_, _ = sleep.Process.Wait()
	}()
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "g-residue.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("fake socket"), 0o600); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:     binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-residue-1234567890",
		Timeout:     time.Second,
	}
	if err := backend.writeTmuxOwnerMarkerV0(); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	t.Setenv("ORQUESTA_TEST_PANE_PID", strconv.Itoa(sleep.Process.Pid))

	result, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{
		EvidenceRefs: []string{"evidence-ref-shutdown-request"},
	})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_backend" ||
		result.ActiveWorks[0].WorkRef != backend.SessionName ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-residue") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-owner-marker") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-socket") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-session") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process") {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkIgnoraSocketNoPropioV0(t *testing.T) {
	root := t.TempDir()
	socketPath := filepath.Join(root, "runtime", "external.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir socket dir: %v", err)
	}
	if err := os.WriteFile(socketPath, []byte("external socket"), 0o600); err != nil {
		t.Fatalf("write socket: %v", err)
	}
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  socketPath,
		SessionName: "external-session",
		Timeout:     time.Second,
	}

	result, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 0 || len(result.EvidenceRefs) != 0 {
		t.Fatalf("socket no propio no debe bloquear: %+v", result)
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaOwnerMarkerRecuperableFueraDeSesionConfiguradaV0(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeTmux := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
case "${1:-}" in
  has-session)
    exit 0
    ;;
  display-message)
    printf '%s\n' "${ORQUESTA_TEST_PANE_PID:-}"
    exit 0
    ;;
esac
exit 2
`), 0o700); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	sleep := exec.Command("sleep", "30")
	if err := sleep.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = sleep.Process.Kill()
		_, _ = sleep.Process.Wait()
	}()
	runtimeDir := filepath.Join(root, "runtime")
	ownerDir := filepath.Join(runtimeDir, codexAppServerTmuxDirV0)
	if err := os.MkdirAll(ownerDir, 0o700); err != nil {
		t.Fatalf("mkdir owner dir: %v", err)
	}
	extraSession := "orquesta-goal-extra-owner-1234567890"
	marker := `{
  "schema_version": "orquesta_codex_app_server_tmux_owner.v0",
  "owner_ref": "orquesta-codex-goal-app-server-tmux-v0",
  "session_name": "` + extraSession + `",
  "socket_ref": "socket-ref-codex-goal-app-server-tmux"
}
`
	if err := os.WriteFile(filepath.Join(ownerDir, codexAppServerTmuxMarkerFileV0), []byte(marker), 0o600); err != nil {
		t.Fatalf("write owner marker: %v", err)
	}
	t.Setenv("ORQUESTA_TEST_PANE_PID", strconv.Itoa(sleep.Process.Pid))
	backend := serverCodexAppServerTmuxBackendV0{
		PathEnv:        binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		SocketPath:     filepath.Join(root, "other", "s.sock"),
		SessionName:    "external-session",
		RuntimeWorkDir: runtimeDir,
		Timeout:        time.Second,
	}

	result, err := backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
	if err != nil {
		t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].WorkRef != extraSession ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-owner-scan") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-session") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process") {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaProcesoConSocketConfiguradoSinMarkerV0(t *testing.T) {
	root := t.TempDir()
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "s.sock")
	command := exec.Command(
		"bash",
		"-c",
		"exec -a codex sh -c 'sleep 30' app-server --listen unix://"+socketPath,
	)
	if err := command.Start(); err != nil {
		t.Fatalf("start fake app-server cmdline: %v", err)
	}
	defer func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}()
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-process-only-1234567890",
		Timeout:     50 * time.Millisecond,
	}

	deadline := time.Now().Add(2 * time.Second)
	var result orquestaservershutdown.ActiveShutdownWorkResultV0
	var err error
	for time.Now().Before(deadline) {
		result, err = backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
		if err != nil {
			t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
		}
		if containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-cmdline") {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].WorkRef != backend.SessionName ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-cmdline") {
		t.Fatalf("result=%+v", result)
	}
	for _, ref := range result.EvidenceRefs {
		if strings.Contains(ref, socketPath) || strings.Contains(ref, strconv.Itoa(command.Process.Pid)) {
			t.Fatalf("evidence filtro ruta/pid: %+v", result.EvidenceRefs)
		}
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaProcesoEnRuntimeWorkdirSinSocketConfiguradoV0(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "g-extra.sock")
	command := exec.Command(
		"bash",
		"-c",
		"exec -a codex sh -c 'sleep 30' app-server --listen unix://"+socketPath,
	)
	if err := command.Start(); err != nil {
		t.Fatalf("start fake app-server cmdline: %v", err)
	}
	defer func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}()
	backend := serverCodexAppServerTmuxBackendV0{
		RuntimeWorkDir: runtimeDir,
		SocketPath:     filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "configured.sock"),
		SessionName:    "orquesta-goal-runtime-process-1234567890",
		Timeout:        50 * time.Millisecond,
	}

	deadline := time.Now().Add(2 * time.Second)
	var result orquestaservershutdown.ActiveShutdownWorkResultV0
	var err error
	for time.Now().Before(deadline) {
		result, err = backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
		if err != nil {
			t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
		}
		if containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-runtime-workdir") {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].WorkRef != backend.SessionName ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-runtime-workdir") {
		t.Fatalf("result=%+v", result)
	}
	for _, ref := range result.EvidenceRefs {
		if strings.Contains(ref, socketPath) || strings.Contains(ref, strconv.Itoa(command.Process.Pid)) {
			t.Fatalf("evidence filtro ruta/pid: %+v", result.EvidenceRefs)
		}
	}
}

func TestCodexAppServerTmuxBackendV0ReadActiveShutdownWorkDetectaProcesoPropioSinSocketPorRuntimeWorkdirV0(t *testing.T) {
	if _, err := os.Stat("/proc/self/cwd"); err != nil {
		t.Skipf("procfs no disponible: %v", err)
	}
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	ownedWorkdir := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "owned-cwd")
	if err := os.MkdirAll(ownedWorkdir, 0o700); err != nil {
		t.Fatalf("mkdir owned workdir: %v", err)
	}
	command := exec.Command(
		"bash",
		"-c",
		"exec -a codex sh -c 'sleep 30' app-server",
	)
	command.Dir = ownedWorkdir
	if err := command.Start(); err != nil {
		t.Fatalf("start fake app-server runtime-owned: %v", err)
	}
	defer func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
	}()
	backend := serverCodexAppServerTmuxBackendV0{
		RuntimeWorkDir: runtimeDir,
		SocketPath:     filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "configured.sock"),
		SessionName:    "orquesta-goal-runtime-owned-1234567890",
		Timeout:        50 * time.Millisecond,
	}

	deadline := time.Now().Add(2 * time.Second)
	var result orquestaservershutdown.ActiveShutdownWorkResultV0
	var err error
	for time.Now().Before(deadline) {
		result, err = backend.ReadActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkRequestV0{})
		if err != nil {
			t.Fatalf("ReadActiveShutdownWorkV0: %v", err)
		}
		if containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-runtime-owned") {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].WorkRef != backend.SessionName ||
		result.ActiveWorks[0].Status != orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process-runtime-owned") {
		t.Fatalf("result=%+v", result)
	}
	for _, ref := range result.EvidenceRefs {
		if strings.Contains(ref, ownedWorkdir) || strings.Contains(ref, strconv.Itoa(command.Process.Pid)) {
			t.Fatalf("evidence filtro ruta/pid: %+v", result.EvidenceRefs)
		}
	}
}

func TestCodexAppServerTmuxCommandsContainRuntimeWorkdirSocketV0IgnoraSocketExternoV0(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	raw := "codex app-server --listen unix://" + filepath.Join(root, "otro-runtime", codexAppServerTmuxDirV0, "g.sock")
	if codexAppServerTmuxCommandsContainRuntimeWorkdirSocketV0(raw, runtimeDir) {
		t.Fatalf("socket fuera de runtime propio no debe bloquear")
	}
}

func TestCodexAppServerTmuxBackendV0EsperaPanePIDAntesDeReadyV0(t *testing.T) {
	command := exec.Command("sleep", "30")
	if err := command.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	done := make(chan struct{})
	defer func() {
		_ = command.Process.Kill()
		<-done
	}()
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
		close(done)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	backend := serverCodexAppServerTmuxBackendV0{}
	if err := backend.waitTmuxPaneExitedV0(ctx, strconv.Itoa(command.Process.Pid)); err != nil {
		t.Fatalf("waitTmuxPaneExitedV0: %v", err)
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

func TestCodexAppServerTmuxSocketPathV0UsaFallbackCortoSiRuntimeEsLargoV0(t *testing.T) {
	config := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		RuntimeWorkDir: "/" + strings.Repeat("runtime-largo-", 12),
		ProjectWorkDir: "/workspace/project",
	})
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatalf("socket path: %v", err)
	}
	if len(socketPath) > codexAppServerTmuxMaxSocketPathV0 {
		t.Fatalf("socket fallback demasiado largo: len=%d path=%s", len(socketPath), socketPath)
	}
	wantPrefix := filepath.Join(os.TempDir(), "oq-gsrv-")
	if len(filepath.Join(os.TempDir(), "oq-gsrv-0000000000000000000000000", "s.sock")) > codexAppServerTmuxMaxSocketPathV0 {
		wantPrefix = filepath.Join("/tmp", "oq-gsrv-")
	}
	if !strings.HasPrefix(socketPath, wantPrefix) || !strings.HasSuffix(socketPath, string(os.PathSeparator)+"s.sock") {
		t.Fatalf("socket fallback inesperado: %s", socketPath)
	}
}

func TestCodexAppServerTmuxRuntimeDirV0RechazaSymlinkV0(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	link := filepath.Join(root, "runtime-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink no disponible: %v", err)
	}
	err := ensureCodexAppServerTmuxRuntimeDirV0(filepath.Join(link, "s.sock"))
	if err == nil || !strings.Contains(err.Error(), "codex_app_server_tmux_runtime_dir") {
		t.Fatalf("err=%v", err)
	}
}

func TestCodexAppServerTmuxSocketPathV0FallbackEsDeterministaYAisladoV0(t *testing.T) {
	baseRuntime := "/" + strings.Repeat("runtime-largo-", 12)
	config := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		RuntimeWorkDir: baseRuntime,
		ProjectWorkDir: "/workspace/project-a",
	})
	sameConfig := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		RuntimeWorkDir: baseRuntime,
		ProjectWorkDir: "/workspace/project-a",
	})
	otherProject := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		RuntimeWorkDir: baseRuntime,
		ProjectWorkDir: "/workspace/project-b",
	})
	otherRuntime := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		RuntimeWorkDir: "/" + strings.Repeat("runtime-distinto-", 12),
		ProjectWorkDir: "/workspace/project-a",
	})
	first, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		t.Fatalf("first socket path: %v", err)
	}
	second, err := codexAppServerTmuxSocketPathV0(sameConfig)
	if err != nil {
		t.Fatalf("second socket path: %v", err)
	}
	projectSocket, err := codexAppServerTmuxSocketPathV0(otherProject)
	if err != nil {
		t.Fatalf("project socket path: %v", err)
	}
	runtimeSocket, err := codexAppServerTmuxSocketPathV0(otherRuntime)
	if err != nil {
		t.Fatalf("runtime socket path: %v", err)
	}
	if first != second {
		t.Fatalf("fallback no determinista: first=%s second=%s", first, second)
	}
	if first == projectSocket || first == runtimeSocket {
		t.Fatalf("fallback no aisla runtime/project: first=%s project=%s runtime=%s", first, projectSocket, runtimeSocket)
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
  display-message)
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
