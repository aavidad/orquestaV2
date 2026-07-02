package orquestaruntimecodex

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexWrapperV0WorkspaceWriteAutorizaRuntimeParaACK(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.Sandbox = "workspace-write"

	wrapper := BuildCodexWrapperScriptV0(profile)
	want := "--add-dir " + shellQuoteV0(profile.RuntimeWorkDir)
	if !strings.Contains(wrapper, want) {
		t.Fatalf("wrapper no autoriza runtime para ACK: falta %q\n%s", want, wrapper)
	}
}

func TestCodexWrapperV0NoAnadeRuntimeWritableSinWorkspaceWrite(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.Sandbox = "danger-full-access"

	wrapper := BuildCodexWrapperScriptV0(profile)
	if strings.Contains(wrapper, "--add-dir ") {
		t.Fatalf("wrapper no debe anadir add-dir fuera de workspace-write:\n%s", wrapper)
	}
}

func TestCodexWrapperV0FijaEsfuerzoDeRazonamiento(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ReasoningEffort = "medium"

	wrapper := BuildCodexWrapperScriptV0(profile)
	want := "-c " + shellQuoteV0(`model_reasoning_effort="medium"`)
	if !strings.Contains(wrapper, want) {
		t.Fatalf("wrapper no fija esfuerzo de razonamiento: falta %q\n%s", want, wrapper)
	}
}

func TestCodexWrapperV0PermiteWorkspacesSinGitPorDefecto(t *testing.T) {
	profile := codexProfileForTestV0(t)

	wrapper := BuildCodexWrapperScriptV0(profile)
	if !strings.Contains(wrapper, "exec --skip-git-repo-check ") {
		t.Fatalf("wrapper no permite workspace sin git:\n%s", wrapper)
	}
}

func TestCodexWrapperV0NoDuplicaSkipGitRepoCheck(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.ExtraArgs = []string{"--skip-git-repo-check"}

	wrapper := BuildCodexWrapperScriptV0(profile)
	if strings.Count(wrapper, "--skip-git-repo-check") != 1 {
		t.Fatalf("wrapper duplica skip git repo check:\n%s", wrapper)
	}
}

func TestCodexWrapperV0SerializaArranqueCompartidoDeCodexHomeV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")

	wrapper := BuildCodexWrapperScriptV0(profile)
	for _, want := range []string{
		".orquesta-codex-startup.lock",
		"ORQUESTA_CODEX_STARTUP_LOCK_SECONDS",
		"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS",
		"ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS",
		"orquesta_codex_reap_stale_startup_lock_v0",
		"orquesta_codex_release_startup_lock_v0",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("wrapper no contiene lock de arranque %q:\n%s", want, wrapper)
		}
	}
	releaseIndex := strings.LastIndex(wrapper, "orquesta_codex_release_startup_lock_v0")
	waitIndex := strings.LastIndex(wrapper, "wait \"$orquesta_codex_child_v0\"")
	if releaseIndex < 0 || waitIndex < 0 || releaseIndex > waitIndex {
		t.Fatalf("wrapper debe liberar lock antes de esperar toda la ejecucion:\n%s", wrapper)
	}
}

func TestCodexWrapperV0RetiraStartupLockObsoletoVacio(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexNoUsageCommandV0(t, profile.RuntimeWorkDir)
	lockPath := filepath.Join(profile.CodeHomeDir, ".orquesta-codex-startup.lock")
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir stale lock: %v", err)
	}
	if err := os.Chtimes(lockPath, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatalf("touch stale lock: %v", err)
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	cmd := exec.Command(wrapperPath)
	cmd.Env = append(os.Environ(), "ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper con lock obsoleto fallo: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "orquesta_codex_startup_lock_timeout") {
		t.Fatalf("wrapper no debe agotar timeout con lock obsoleto:\n%s", out)
	}
}

func TestCodexWrapperV0StaleLockPorDefectoNoSuperaTimeout(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexNoUsageCommandV0(t, profile.RuntimeWorkDir)
	lockPath := filepath.Join(profile.CodeHomeDir, ".orquesta-codex-startup.lock")
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir stale lock: %v", err)
	}
	if err := os.Chtimes(lockPath, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatalf("touch stale lock: %v", err)
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	cmd := exec.Command(wrapperPath)
	cmd.Env = append(os.Environ(), "ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper con stale por defecto fallo: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "orquesta_codex_startup_lock_timeout") {
		t.Fatalf("wrapper no debe agotar timeout con stale por defecto:\n%s", out)
	}
}

func TestCodexWrapperV0NoEsperaStartupLockSiCodexHomeEstaAisladoEnRuntimeV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CodeHomeDir = filepath.Join(profile.RuntimeWorkDir, "codex-home")

	wrapper := BuildCodexWrapperScriptV0(profile)
	if !strings.Contains(wrapper, `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS:-0`) {
		t.Fatalf("wrapper debe usar espera cero con CODEX_HOME aislado por agente:\n%s", wrapper)
	}
}

func TestCodexWrapperV0NoEsperaStartupLockConRuntimeFakeV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CommandPath = filepath.Join(t.TempDir(), "codex-fake")
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")

	wrapper := BuildCodexWrapperScriptV0(profile)
	if !strings.Contains(wrapper, `ORQUESTA_CODEX_STARTUP_LOCK_SECONDS:-0`) {
		t.Fatalf("wrapper debe usar espera cero con runtime fake:\n%s", wrapper)
	}
}

func TestCodexWrapperV0CualquierVariableStartupLockCuentaComoExplicitaV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")

	wrapper := BuildCodexWrapperScriptV0(profile)
	for _, want := range []string{
		"${ORQUESTA_CODEX_STARTUP_LOCK_SECONDS+x}",
		"${ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS+x}",
		"${ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS+x}",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("wrapper debe tratar la presencia de cualquier variable de startup-lock como explicita: falta %q\n%s", want, wrapper)
		}
	}
}

func TestCodexWrapperV0RespetaCadaVariableStartupLockExplicitaAunqueDefaultSeaCeroV0(t *testing.T) {
	t.Run("lock_seconds_solo", func(t *testing.T) {
		profile, wrapperPath := prepareCodexStartupLockWrapperV0(t, writeFakeCodexStartupLockProbeCommandV0)

		cmd := exec.Command(wrapperPath)
		cmd.Env = append(os.Environ(), "ORQUESTA_CODEX_STARTUP_LOCK_SECONDS=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			stderr := readCodexWrapperTestFileV0(t, filepath.Join(profile.RuntimeWorkDir, CodexStderrFileNameV0))
			t.Fatalf("wrapper debe activar lock por STARTUP_LOCK_SECONDS explicito, err=%v out=%s stderr=%s", err, out, stderr)
		}
		stderr := readCodexWrapperTestFileV0(t, filepath.Join(profile.RuntimeWorkDir, CodexStderrFileNameV0))
		if !strings.Contains(stderr, "startup_lock_present") {
			t.Fatalf("STARTUP_LOCK_SECONDS explicito debe crear lock aunque el default sea cero; stderr=%s out=%s", stderr, out)
		}
	})

	t.Run("stale_solo", func(t *testing.T) {
		profile, wrapperPath := prepareCodexStartupLockWrapperV0(t, writeFakeCodexNoUsageCommandV0)
		lockPath := mkdirCodexStartupLockForTestV0(t, profile.CodeHomeDir)
		touchCodexStartupLockForTestV0(t, lockPath, time.Unix(1, 0))

		cmd := exec.Command(wrapperPath)
		cmd.Env = append(os.Environ(), "ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("wrapper debe respetar STARTUP_LOCK_STALE_SECONDS explicito, err=%v out=%s", err, out)
		}
		if !strings.Contains(string(out), "orquesta_codex_startup_lock_stale_reaped") {
			t.Fatalf("STARTUP_LOCK_STALE_SECONDS explicito debe activar lock/reap aunque el default sea cero:\n%s", out)
		}
	})

	t.Run("stale_solo_vacio_presente", func(t *testing.T) {
		profile, wrapperPath := prepareCodexStartupLockWrapperV0(t, writeFakeCodexNoUsageCommandV0)
		lockPath := mkdirCodexStartupLockForTestV0(t, profile.CodeHomeDir)
		touchCodexStartupLockForTestV0(t, lockPath, time.Unix(1, 0))

		cmd := exec.Command(wrapperPath)
		cmd.Env = append(os.Environ(), "ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS=")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("wrapper debe respetar STARTUP_LOCK_STALE_SECONDS presente aunque este vacio, err=%v out=%s", err, out)
		}
		if !strings.Contains(string(out), "orquesta_codex_startup_lock_stale_reaped") {
			t.Fatalf("STARTUP_LOCK_STALE_SECONDS presente debe activar lock/reap aunque el default sea cero:\n%s", out)
		}
	})
}

func TestCodexWrapperV0TimeoutSoloYTimeoutStaleExplicitosNoSeSustituyenV0(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{
			name: "timeout_solo",
			env:  []string{"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS=0"},
		},
		{
			name: "timeout_mas_stale",
			env: []string{
				"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS=0",
				"ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS=0",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile, wrapperPath := prepareCodexStartupLockWrapperV0(t, writeFakeCodexNoUsageCommandV0)
			mkdirCodexStartupLockForTestV0(t, profile.CodeHomeDir)

			cmd := exec.Command(wrapperPath)
			cmd.Env = append(os.Environ(), tc.env...)
			out, err := cmd.CombinedOutput()
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 75 {
				t.Fatalf("wrapper debe cortar por timeout explicito %s aunque el default sea cero, err=%v out=%s", tc.name, err, out)
			}
			if !strings.Contains(string(out), "orquesta_codex_startup_lock_timeout") {
				t.Fatalf("wrapper no publico timeout de lock explicito %s:\n%s", tc.name, out)
			}
		})
	}
}

func TestCodexWrapperV0RespetaStartupLockExplicitoAunqueDefaultSeaCeroV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexNoUsageCommandV0(t, profile.RuntimeWorkDir)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	lockPath := filepath.Join(profile.CodeHomeDir, ".orquesta-codex-startup.lock")
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	cmd := exec.Command(wrapperPath)
	cmd.Env = append(
		os.Environ(),
		"ORQUESTA_CODEX_STARTUP_LOCK_SECONDS=1",
		"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS=0",
		"ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS=0",
	)
	out, err := cmd.CombinedOutput()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 75 {
		t.Fatalf("wrapper debe respetar lock explicito y cortar por timeout, err=%v out=%s", err, out)
	}
	if !strings.Contains(string(out), "orquesta_codex_startup_lock_timeout") {
		t.Fatalf("wrapper no publico timeout de lock explicito:\n%s", out)
	}
}

func TestCodexWrapperV0RespetaStartupLockTimeoutSoloAunqueDefaultSeaCeroV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexNoUsageCommandV0(t, profile.RuntimeWorkDir)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	lockPath := filepath.Join(profile.CodeHomeDir, ".orquesta-codex-startup.lock")
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	cmd := exec.Command(wrapperPath)
	cmd.Env = append(
		os.Environ(),
		"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS=0",
	)
	out, err := cmd.CombinedOutput()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 75 {
		t.Fatalf("wrapper debe activar lock si timeout es explicito aunque stale no exista, err=%v out=%s", err, out)
	}
	if !strings.Contains(string(out), "orquesta_codex_startup_lock_timeout") {
		t.Fatalf("wrapper no publico timeout de lock explicito por timeout solo:\n%s", out)
	}
}

func TestCodexWrapperV0RespetaStartupLockTimeoutExplicitoAunqueDefaultSeaCeroV0(t *testing.T) {
	profile := codexProfileForTestV0(t)
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexNoUsageCommandV0(t, profile.RuntimeWorkDir)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	lockPath := filepath.Join(profile.CodeHomeDir, ".orquesta-codex-startup.lock")
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	cmd := exec.Command(wrapperPath)
	cmd.Env = append(
		os.Environ(),
		"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS=0",
		"ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS=0",
	)
	out, err := cmd.CombinedOutput()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 75 {
		t.Fatalf("wrapper debe activar lock si timeout/stale son explicitos, err=%v out=%s", err, out)
	}
	if !strings.Contains(string(out), "orquesta_codex_startup_lock_timeout") {
		t.Fatalf("wrapper no publico timeout de lock explicito por timeout/stale:\n%s", out)
	}
}

func TestCodexWrapperV0GeneraReporteUsoRedactadoYConservaExitCode(t *testing.T) {
	profile := codexProfileForTestV0(t)
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexUsageCommandV0(t, profile.RuntimeWorkDir)
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	err := exec.Command(wrapperPath).Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 7 {
		t.Fatalf("exit=%v", err)
	}
	report, err := os.ReadFile(filepath.Join(profile.RuntimeWorkDir, CodexUsageAccountingFileNameV0))
	if err != nil {
		t.Fatalf("read usage report: %v", err)
	}
	if !CodexUsageAccountingReportRedactedV0(string(report)) {
		t.Fatalf("usage report no redactado: %s", report)
	}
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{string(report)})
	if snapshot.QuotaStatus != CodexUsageQuotaExhaustedV0 ||
		snapshot.PromptTokens != 123 ||
		snapshot.CompletionTokens != 45 ||
		snapshot.TotalTokens != 168 {
		t.Fatalf("snapshot=%+v report=%s", snapshot, report)
	}
}

func TestCodexWrapperV0GeneraReporteUnavailableSinUsoObservado(t *testing.T) {
	profile := codexProfileForTestV0(t)
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	profile.CommandPath = writeFakeCodexNoUsageCommandV0(t, profile.RuntimeWorkDir)
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	if err := exec.Command(wrapperPath).Run(); err != nil {
		t.Fatalf("wrapper: %v", err)
	}
	report, err := os.ReadFile(filepath.Join(profile.RuntimeWorkDir, CodexUsageAccountingFileNameV0))
	if err != nil {
		t.Fatalf("read usage report: %v", err)
	}
	snapshot := BuildCodexUsageAccountingSnapshotV0([]string{string(report)})
	if snapshot.QuotaStatus != CodexUsageQuotaUnknownV0 ||
		snapshot.QuotaReason != CodexUsageQuotaReasonObservedUnavailableV0 ||
		snapshot.TotalTokens != 0 {
		t.Fatalf("snapshot=%+v report=%s", snapshot, report)
	}
}

func TestCodexProfileV0WorkspaceWriteAceptaRuntimeAisladoFueraDelProyecto(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.RuntimeWorkDir = t.TempDir()

	issues := ValidateCodexConnectorProfileV0(profile)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexProfileV0WorkspaceWriteAceptaRuntimeDentroDelProyecto(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.RuntimeWorkDir = profile.ProjectWorkDir + "/.orquesta-runtime"

	issues := ValidateCodexConnectorProfileV0(profile)

	if len(issues) != 0 {
		t.Fatalf("issues inesperadas: %+v", issues)
	}
}

func TestCodexProfileV0WorkspaceWriteRechazaRuntimeQueContieneProyecto(t *testing.T) {
	profile := codexProfileForTestV0(t)
	profile.RuntimeWorkDir = filepath.Dir(profile.ProjectWorkDir)

	issues := ValidateCodexConnectorProfileV0(profile)

	requireCodexIssueV0(t, issues, CodexConnectorPathInvalidV0)
}

func prepareCodexStartupLockWrapperV0(
	t *testing.T,
	writeCommand func(*testing.T, string) string,
) (CodexConnectorProfileV0, string) {
	t.Helper()
	profile := codexProfileForTestV0(t)
	if err := os.MkdirAll(profile.ProjectWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.MkdirAll(profile.RuntimeWorkDir, 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if writeCommand == nil {
		writeCommand = writeFakeCodexNoUsageCommandV0
	}
	profile.CommandPath = writeCommand(t, profile.RuntimeWorkDir)
	profile.CodeHomeDir = filepath.Join(t.TempDir(), "codex-home")
	if err := os.MkdirAll(profile.CodeHomeDir, 0o700); err != nil {
		t.Fatalf("mkdir code home: %v", err)
	}
	wrapperPath := filepath.Join(profile.RuntimeWorkDir, CodexWrapperFileNameV0)
	if err := os.WriteFile(
		filepath.Join(profile.RuntimeWorkDir, CodexAgentPromptFileNameV0),
		[]byte("prompt"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(wrapperPath, []byte(BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	return profile, wrapperPath
}

func mkdirCodexStartupLockForTestV0(t *testing.T, codeHomeDir string) string {
	t.Helper()
	lockPath := filepath.Join(codeHomeDir, ".orquesta-codex-startup.lock")
	if err := os.MkdirAll(lockPath, 0o700); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}
	return lockPath
}

func touchCodexStartupLockForTestV0(t *testing.T, lockPath string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(lockPath, at, at); err != nil {
		t.Fatalf("touch lock: %v", err)
	}
}

func readCodexWrapperTestFileV0(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func writeFakeCodexUsageCommandV0(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-codex.sh")
	script := "#!/bin/sh\n" +
		"cat >/dev/null\n" +
		"echo 'input tokens: 123' >&2\n" +
		"echo 'output tokens: 45' >&2\n" +
		"echo \"ERROR: You've hit your usage limit. Upgrade to Pro or try again at 1:57 PM.\" >&2\n" +
		"exit 7\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return path
}

func writeFakeCodexStartupLockProbeCommandV0(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-codex-startup-lock-probe.sh")
	script := "#!/bin/sh\n" +
		"cat >/dev/null\n" +
		"if [ -d \"${CODEX_HOME:-}/.orquesta-codex-startup.lock\" ]; then\n" +
		"  echo 'startup_lock_present' >&2\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo 'startup_lock_missing' >&2\n" +
		"exit 3\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return path
}

func writeFakeCodexNoUsageCommandV0(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-codex-no-usage.sh")
	script := "#!/bin/sh\n" +
		"cat >/dev/null\n" +
		"echo 'done without accounting' >&2\n" +
		"exit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return path
}
