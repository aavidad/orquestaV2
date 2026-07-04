package orquestaappcodexstack

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	codexStackRealSmokeTimeoutMSKeyV0      = "ORQUESTA_CODEX_SMOKE_TIMEOUT_MS"
	codexStackRealSmokeTimeoutSecondsKeyV0 = "ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS"
)

type codexStackRealSmokeConfigV0 struct {
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
	MaxBatchReady   int
	MaxConcurrency  int
	Timeout         time.Duration

	PromoteMaterializedArtifactWithoutAck bool
}

func codexStackRealSmokeConfigForTestV0(t *testing.T) codexStackRealSmokeConfigV0 {
	t.Helper()
	projectDir := codexStackRealSmokeProjectDirV0(t)
	sandbox := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_SANDBOX"))
	runtimeDir := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_RUNTIME_WORKDIR"))
	if runtimeDir == "" && sandbox == "workspace-write" {
		runtimeDir = filepath.Join(filepath.Dir(projectDir), ".orquesta-runtime", filepath.Base(projectDir))
	}
	if runtimeDir == "" {
		runtimeDir = filepath.Join(t.TempDir(), "runtime")
	}
	return codexStackRealSmokeConfigV0{
		CommandPath:     codexStackRealSmokeCommandPathV0(t),
		ProjectWorkDir:  projectDir,
		RuntimeWorkDir:  codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_CODEX_RUNTIME_WORKDIR", runtimeDir),
		CodeHomeDir:     codexStackRealSmokeRequiredAbsDirEnvV0(t, "ORQUESTA_CODEX_CODE_HOME"),
		HomeDir:         codexStackRealSmokeRequiredAbsDirEnvV0(t, "ORQUESTA_CODEX_HOME"),
		PathEnv:         codexStackRealSmokeRequiredEnvV0(t, "ORQUESTA_CODEX_PATH"),
		Model:           strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_MODEL")),
		ReasoningEffort: codexStackRealSmokeReasoningEffortV0(),
		Profile:         strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROFILE")),
		Sandbox:         sandbox,
		ApprovalPolicy:  codexStackRealSmokeRequiredEnvV0(t, "ORQUESTA_CODEX_APPROVAL_POLICY"),
		ExtraArgs:       strings.Fields(os.Getenv("ORQUESTA_CODEX_EXTRA_ARGS")),
		MaxBatchReady:   1,
		MaxConcurrency:  1,
		Timeout:         codexStackRealSmokeTimeoutV0(t),
	}
}

func codexStackRealSmokeReasoningEffortV0() string {
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REASONING_EFFORT"))
	if value == "xhigh" {
		return value
	}
	return "high"
}

func codexStackRealSmokeProjectDirV0(t *testing.T) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROJECT_WORKDIR"))
	if value == "" {
		value = t.TempDir()
	}
	dir := codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_CODEX_PROJECT_WORKDIR", value)
	codexStackRealSmokeWriteProjectContextV0(t, dir)
	return dir
}

func codexStackRealSmokeCommandPathV0(t *testing.T) string {
	t.Helper()
	raw := codexStackRealSmokeRequiredEnvV0(t, "ORQUESTA_CODEX_COMMAND")
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		t.Fatalf("ORQUESTA_CODEX_COMMAND no encontrado: %s", raw)
	}
	return path
}

func codexStackRealSmokeRequiredAbsDirEnvV0(t *testing.T, key string) string {
	t.Helper()
	return codexStackRealSmokeEnsureDirV0(t, key, codexStackRealSmokeRequiredEnvV0(t, key))
}

func codexStackRealSmokeRequiredEnvV0(t *testing.T, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Fatalf("%s requerido para smoke opt-in", key)
	}
	return value
}

func codexStackRealSmokeEnsureDirV0(t *testing.T, key string, value string) string {
	t.Helper()
	if !filepath.IsAbs(value) {
		t.Fatalf("%s debe ser absoluto", key)
	}
	if err := os.MkdirAll(value, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", key, err)
	}
	return value
}

func codexStackRealSmokeTimeoutV0(t *testing.T) time.Duration {
	t.Helper()
	rawMS := strings.TrimSpace(os.Getenv(codexStackRealSmokeTimeoutMSKeyV0))
	if rawMS != "" {
		milliseconds, err := strconv.Atoi(rawMS)
		if err != nil || milliseconds < 30000 {
			t.Fatalf("%s invalido: %q", codexStackRealSmokeTimeoutMSKeyV0, rawMS)
		}
		return time.Duration(milliseconds) * time.Millisecond
	}
	raw := strings.TrimSpace(os.Getenv(codexStackRealSmokeTimeoutSecondsKeyV0))
	if raw == "" {
		return 240 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 30 {
		t.Fatalf("%s invalido: %q", codexStackRealSmokeTimeoutSecondsKeyV0, raw)
	}
	t.Logf("%s deprecado; usa %s", codexStackRealSmokeTimeoutSecondsKeyV0, codexStackRealSmokeTimeoutMSKeyV0)
	milliseconds := seconds * 1000
	return time.Duration(milliseconds) * time.Millisecond
}

func codexStackRealSmokeMaxExternalWaitsV0(timeout time.Duration, interval time.Duration) int {
	waits := int(timeout/interval) - 2
	if waits < 1 {
		return 1
	}
	return waits
}

func TestCodexStackRealSmokeTimeoutV0UsaMilisegundosCanonicos(t *testing.T) {
	t.Setenv(codexStackRealSmokeTimeoutMSKeyV0, "31500")
	t.Setenv(codexStackRealSmokeTimeoutSecondsKeyV0, "")

	got := codexStackRealSmokeTimeoutV0(t)
	if got != 31500*time.Millisecond {
		t.Fatalf("timeout = %s, want %s", got, 31500*time.Millisecond)
	}
}

func TestCodexStackRealSmokeTimeoutV0AceptaAliasLegacySeconds(t *testing.T) {
	t.Setenv(codexStackRealSmokeTimeoutMSKeyV0, "")
	t.Setenv(codexStackRealSmokeTimeoutSecondsKeyV0, "31")

	got := codexStackRealSmokeTimeoutV0(t)
	if got != 31*time.Second {
		t.Fatalf("timeout = %s, want %s", got, 31*time.Second)
	}
}

func TestCodexStackRealSmokeTimeoutV0PriorizaMilisegundosSobreLegacy(t *testing.T) {
	t.Setenv(codexStackRealSmokeTimeoutMSKeyV0, "32500")
	t.Setenv(codexStackRealSmokeTimeoutSecondsKeyV0, "99")

	got := codexStackRealSmokeTimeoutV0(t)
	if got != 32500*time.Millisecond {
		t.Fatalf("timeout = %s, want %s", got, 32500*time.Millisecond)
	}
}
