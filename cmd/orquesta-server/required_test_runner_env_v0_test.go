package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestRequiredTestRunnerFromEnvV0RequiereAllowlist(t *testing.T) {
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", "")
	t.Setenv("ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS", "")
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}

	_, err = requiredTestRunnerFromEnvV0(orquestaserver.ConfigV0{
		StateDir:       t.TempDir(),
		ProjectWorkDir: t.TempDir(),
	}, stateStore)
	if err == nil || err.Error() != "required_test_allowed_commands_required" {
		t.Fatalf("err=%v", err)
	}
}

func TestRequiredTestRunnerFromEnvV0GoCommandInyectaEntornoGoAcotado(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "required-test-output")
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", filepath.Join(projectDir, "go-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", outputDir)
	t.Setenv("ORQUESTA_REQUIRED_TEST_ENV", "")
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: filepath.Join(stateDir, "state")})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}

	runnerPort, err := requiredTestRunnerFromEnvV0(orquestaserver.ConfigV0{
		StateDir:       stateDir,
		ProjectWorkDir: projectDir,
	}, stateStore)
	if err != nil {
		t.Fatalf("requiredTestRunnerFromEnvV0: %v", err)
	}
	env := requiredTestExecutorEnvByKeyV0(t, runnerPort)

	expected := map[string]string{
		"GOCACHE":    filepath.Join(outputDir, "go-build-cache"),
		"GOPATH":     filepath.Join(outputDir, "go-path"),
		"GOMODCACHE": filepath.Join(outputDir, "go-mod-cache"),
	}
	for key, want := range expected {
		if got := env[key]; got != want {
			t.Fatalf("%s=%q, want %q; env=%v", key, got, want, env)
		}
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("%s dir no disponible: %v", key, err)
		}
	}
	if _, ok := env["HOME"]; ok {
		t.Fatalf("HOME no debe heredarse: env=%v", env)
	}
	if _, ok := env["ORQUESTA_CODEX_HOME"]; ok {
		t.Fatalf("ORQUESTA_CODEX_HOME no debe inyectarse: env=%v", env)
	}
}

func TestRequiredTestRunnerFromEnvV0PreservaEntornoGoExplicito(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "required-test-output")
	explicit := map[string]string{
		"GOCACHE":    filepath.Join(t.TempDir(), "custom-gocache"),
		"GOPATH":     filepath.Join(t.TempDir(), "custom-gopath"),
		"GOMODCACHE": filepath.Join(t.TempDir(), "custom-gomodcache"),
	}
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", filepath.Join(projectDir, "go-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", outputDir)
	t.Setenv("ORQUESTA_REQUIRED_TEST_ENV",
		"GOCACHE="+explicit["GOCACHE"]+","+
			"GOPATH="+explicit["GOPATH"]+","+
			"GOMODCACHE="+explicit["GOMODCACHE"],
	)
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: filepath.Join(stateDir, "state")})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}

	runnerPort, err := requiredTestRunnerFromEnvV0(orquestaserver.ConfigV0{
		StateDir:       stateDir,
		ProjectWorkDir: projectDir,
	}, stateStore)
	if err != nil {
		t.Fatalf("requiredTestRunnerFromEnvV0: %v", err)
	}
	env := requiredTestExecutorEnvByKeyV0(t, runnerPort)

	for key, want := range explicit {
		if got := env[key]; got != want {
			t.Fatalf("%s=%q, want %q; env=%v", key, got, want, env)
		}
	}
	if got := env["GOCACHE"]; strings.HasPrefix(got, outputDir) {
		t.Fatalf("GOCACHE explicito fue reemplazado: %q", got)
	}
}

func TestRequiredTestRunnerFromEnvV0ConfiguraRetencionDeOutput(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", filepath.Join(projectDir, "go-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_OUTPUT_MAX_ARTIFACTS", "7")
	stateStore, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: filepath.Join(stateDir, "state")})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}

	runnerPort, err := requiredTestRunnerFromEnvV0(orquestaserver.ConfigV0{
		StateDir:       stateDir,
		ProjectWorkDir: projectDir,
	}, stateStore)
	if err != nil {
		t.Fatalf("requiredTestRunnerFromEnvV0: %v", err)
	}
	executor := requiredTestExecutorForEnvTestV0(t, runnerPort)
	if executor.MaxArtifacts != 7 {
		t.Fatalf("MaxArtifacts=%d, want 7", executor.MaxArtifacts)
	}
}

func TestValidateCodexCommandAvailableV0MantieneRequisitoSinExternalOnly(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_COMMAND", "codex-missing-orquesta-test")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")

	if err := validateCodexCommandAvailableV0(); err == nil || err.Error() != "codex_command_unavailable" {
		t.Fatalf("validateCodexCommandAvailableV0 err=%v", err)
	}
}

func TestValidateCodexCommandAvailableV0ExigeCodexAunqueHayaOPESMientrasStackSeaCodex(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_COMMAND", "codex-missing-orquesta-test")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ORQUESTA_OPES_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("OPES_BASE_URL", "")

	if err := validateCodexCommandAvailableV0(); err == nil || err.Error() != "codex_command_unavailable" {
		t.Fatalf("validateCodexCommandAvailableV0 err=%v", err)
	}
}

func requiredTestExecutorEnvByKeyV0(
	t *testing.T,
	runnerPort orquestacionnucleoapp.RequiredTestRunnerPortV0,
) map[string]string {
	t.Helper()
	executor := requiredTestExecutorForEnvTestV0(t, runnerPort)
	env := map[string]string{}
	for _, item := range executor.Env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			t.Fatalf("env invalido en executor: %q", item)
		}
		env[key] = value
	}
	return env
}

func requiredTestExecutorForEnvTestV0(
	t *testing.T,
	runnerPort orquestacionnucleoapp.RequiredTestRunnerPortV0,
) orquestaruntimerequiredtest.LocalCommandExecutorV0 {
	t.Helper()
	runner, ok := runnerPort.(orquestacionnucleoapp.RequiredTestRunnerV0)
	if !ok {
		t.Fatalf("RequiredTestRunner usa %T", runnerPort)
	}
	executor, ok := runner.Executor.(orquestaruntimerequiredtest.LocalCommandExecutorV0)
	if !ok {
		t.Fatalf("RequiredTest executor usa %T", runner.Executor)
	}
	return executor
}
