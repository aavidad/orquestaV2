package orquestaruntimecodexdelivery

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type codexRealSmokeConfigV0 struct {
	CommandPath     string
	ProjectWorkDir  string
	RuntimeWorkDir  string
	CodeHomeDir     string
	HomeDir         string
	PathEnv         string
	Model           string
	ReasoningEffort string
	Profile         string
	Sandbox         string
	ApprovalPolicy  string
	ExtraArgs       []string
	Timeout         time.Duration
}

func codexRealSmokeConfigForTestV0(t *testing.T) codexRealSmokeConfigV0 {
	t.Helper()
	projectDir := codexRealSmokeDirFromEnvOrTempV0(t, "ORQUESTA_CODEX_PROJECT_WORKDIR")
	sandbox := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_SANDBOX"))
	runtimeDir := codexRealSmokeRuntimeDirForTestV0(t, projectDir, sandbox)
	return codexRealSmokeConfigV0{
		CommandPath:     codexRealSmokeCommandPathForTestV0(t),
		ProjectWorkDir:  projectDir,
		RuntimeWorkDir:  runtimeDir,
		CodeHomeDir:     strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_CODE_HOME")),
		HomeDir:         strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_HOME")),
		PathEnv:         strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PATH")),
		Model:           strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_MODEL")),
		ReasoningEffort: codexRealSmokeReasoningEffortV0(),
		Profile:         strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROFILE")),
		Sandbox:         sandbox,
		ApprovalPolicy:  strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_APPROVAL_POLICY")),
		ExtraArgs:       strings.Fields(os.Getenv("ORQUESTA_CODEX_EXTRA_ARGS")),
		Timeout:         codexRealSmokeTimeoutFromEnvV0(t),
	}
}

func codexRealSmokeReasoningEffortV0() string {
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REASONING_EFFORT"))
	if value == "" {
		return "medium"
	}
	return value
}

func codexRealSmokeRuntimeDirForTestV0(t *testing.T, projectDir string, sandbox string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_RUNTIME_WORKDIR"))
	if value == "" && sandbox == "workspace-write" {
		value = filepath.Join(projectDir, ".orquesta-codex-runtime")
	}
	if value != "" {
		return codexRealSmokeEnsureDirV0(t, "ORQUESTA_CODEX_RUNTIME_WORKDIR", value)
	}
	return codexRealSmokeDirFromEnvOrTempV0(t, "ORQUESTA_CODEX_RUNTIME_WORKDIR")
}

func codexRealSmokeDirFromEnvOrTempV0(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = t.TempDir()
	}
	return codexRealSmokeEnsureDirV0(t, key, value)
}

func codexRealSmokeEnsureDirV0(t *testing.T, key string, value string) string {
	t.Helper()
	if !filepath.IsAbs(value) {
		t.Fatalf("%s debe ser absoluto", key)
	}
	if err := os.MkdirAll(value, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", key, err)
	}
	return value
}

func codexRealSmokeCommandPathForTestV0(t *testing.T) string {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_COMMAND"))
	if raw == "" {
		raw = "codex"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		t.Fatalf("ORQUESTA_CODEX_COMMAND no encontrado: %s", raw)
	}
	return path
}

func codexRealSmokeTimeoutFromEnvV0(t *testing.T) time.Duration {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS"))
	if raw == "" {
		return 120 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 10 {
		t.Fatalf("ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS invalido: %q", raw)
	}
	return time.Duration(seconds) * time.Second
}
