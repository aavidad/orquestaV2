package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestMain(m *testing.M) {
	cleanup := configureServerPackageTestEnvV0()
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func configureServerPackageTestEnvV0() func() {
	tmpParent := "."
	for _, candidate := range []string{"/workspace/runtime", "/var/tmp"} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			tmpParent = candidate
			break
		}
	}
	tmpDir, err := os.MkdirTemp(tmpParent, "orquesta-server-test-tmp-")
	if err == nil {
		if abs, absErr := filepath.Abs(tmpDir); absErr == nil {
			_ = os.Setenv("TMPDIR", abs)
			_ = os.Setenv("GOTMPDIR", abs)
			_ = os.Setenv("GOCACHE", filepath.Join(abs, "go-cache"))
			_ = os.Setenv("GOPATH", filepath.Join(abs, "go"))
			_ = os.Setenv("GOMODCACHE", filepath.Join(abs, "go", "pkg", "mod"))
		}
	}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "ORQUESTA_") || strings.HasPrefix(key, "OPES_") {
			_ = os.Unsetenv(key)
		}
	}
	_ = os.Unsetenv("CODEX_HOME")
	_ = os.Setenv(envCodexSandboxV0, "danger-full-access")
	if executable, executableErr := os.Executable(); executableErr == nil {
		_ = os.Setenv(envCodexCommandV0, executable)
		_ = os.Setenv("TEST_CODEX_APP_SERVER_BINARY", executable)
	}
	return func() {
		if tmpDir != "" {
			cleanupServerPackageTestTempDirV0(tmpDir)
		}
	}
}

func cleanupServerPackageTestTempDirV0(path string) {
	_ = filepath.WalkDir(path, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		mode := os.FileMode(0o700)
		if entry != nil && !entry.IsDir() {
			mode = 0o600
		}
		_ = os.Chmod(current, mode)
		return nil
	})
	_ = os.RemoveAll(path)
}

func TestCleanupServerPackageTestTempDirV0EliminaModuloGoReadOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "orquesta-server-test-tmp")
	moduleDir := filepath.Join(root, "go", "pkg", "mod", "golang.org", "x", "text@v0.38.0")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module: %v", err)
	}
	filePath := filepath.Join(moduleDir, "go.mod")
	if err := os.WriteFile(filePath, []byte("module golang.org/x/text\n"), 0o400); err != nil {
		t.Fatalf("write module file: %v", err)
	}
	if err := os.Chmod(moduleDir, 0o500); err != nil {
		t.Fatalf("chmod module dir: %v", err)
	}

	cleanupServerPackageTestTempDirV0(root)

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("root no eliminado, err=%v", err)
	}
}

func TestCodexRuntimeConfigV0UsaSandboxWorkspaceWritePorDefecto(t *testing.T) {
	t.Setenv(envCodexSandboxV0, "")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "workspace-write" {
		t.Fatalf("sandbox=%q want workspace-write", config.Sandbox)
	}
}

func TestCodeHomeDirV0UsaOrquestaCodexHomeComoAliasDirectoV0(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex-home")
	processHome := filepath.Join(root, "home")
	t.Setenv(envCodexCodeHomeV0, "")
	t.Setenv(envCodexHomeV0, codexHome)
	t.Setenv(envCodexCodeHomeLegacyV0, processHome)

	if got := codeHomeDirV0(); got != codexHome {
		t.Fatalf("code_home=%q want %q", got, codexHome)
	}
}

func TestCodeHomeDirV0UsaCodexHomeLegacyAliasSiNoHayCanonicaNiOrquestaAliasV0(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "codex-home")
	t.Setenv(envCodexCodeHomeV0, "")
	t.Setenv(envCodexHomeV0, "")
	t.Setenv(envCodexCodeHomeLegacyV0, codexHome)

	if got := codeHomeDirV0(); got != codexHome {
		t.Fatalf("code_home=%q want %q", got, codexHome)
	}
}

func TestCodeHomeDirV0NoInventaRutaRelativaSinHomeV0(t *testing.T) {
	t.Setenv(envCodexCodeHomeV0, "")
	t.Setenv(envCodexHomeV0, "")
	t.Setenv(envCodexCodeHomeLegacyV0, "")
	t.Setenv("HOME", "")

	if got := codeHomeDirV0(); got != "" {
		t.Fatalf("code_home=%q want vacio", got)
	}
}

func TestCodexCommandPathV0RechazaRutaRelativaAunqueEsteEnPath(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	codexBin := filepath.Join(binDir, "codex")
	if err := os.WriteFile(codexBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write codex: %v", err)
	}
	t.Setenv(envCodexCommandV0, "codex")
	t.Setenv(envCodexPathV0, binDir)
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty-path"))

	if got := codexCommandPathV0(); got != "" {
		t.Fatalf("codex_command=%q want vacio", got)
	}
}

func TestValidateCodexCommandAvailableV0PreflightVersion(t *testing.T) {
	root := t.TempDir()
	codexBin := filepath.Join(root, "codex")
	if err := os.WriteFile(codexBin, []byte("#!/bin/sh\nprintf 'codex-cli 1.2.3\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCodexCommandV0, codexBin)
	t.Setenv(envCodexPathV0, "/bin:/usr/bin")
	if err := validateCodexCommandAvailableV0(); err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if err := os.WriteFile(codexBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateCodexCommandAvailableV0(); err == nil || err.Error() != "codex_command_version_unavailable" {
		t.Fatalf("preflight vacio err=%v", err)
	}
}

func TestCodexRuntimeConfigV0DangerFullAccessSoloOptInExplicito(t *testing.T) {
	t.Setenv(envCodexSandboxV0, "danger-full-access")

	config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		RuntimeWorkDir: t.TempDir(),
	}, nil)

	if config.Sandbox != "danger-full-access" {
		t.Fatalf("sandbox=%q want danger-full-access", config.Sandbox)
	}
}

func TestCodexRuntimeConfigV0NoHeredaReasoningGlobal(t *testing.T) {
	for _, effort := range []string{"high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			t.Setenv(envCodexReasoningEffortV0, effort)

			config := codexRuntimeConfigV0(orquestaserver.ConfigV0{
				ProjectWorkDir: t.TempDir(),
				RuntimeWorkDir: t.TempDir(),
			}, nil)

			if config.ReasoningEffort != "" {
				t.Fatalf("reasoning_effort heredado=%q", config.ReasoningEffort)
			}
		})
	}
}

func TestCodexStackCapacityConfigV0ConservaReasoningHighYXHigh(t *testing.T) {
	for _, effort := range []orquestacoreworkflow.OrchestrationCapacityRecommendationV0{
		orquestacoreworkflow.OrchestrationCapacityHighV0,
		orquestacoreworkflow.OrchestrationCapacityXHighV0,
	} {
		t.Run(string(effort), func(t *testing.T) {
			t.Setenv(envCapacityReasoningEffortV0, string(effort))

			config := codexStackCapacityEnvConfigFromEnvV0()

			if config.ReasoningEffort != effort {
				t.Fatalf("capacity_reasoning=%q want %q", config.ReasoningEffort, effort)
			}
		})
	}
}
