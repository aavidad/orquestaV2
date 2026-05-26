package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func codexWaveConfigFromArgsV0(args []string, stderr io.Writer) (codexWaveConfigV0, error) {
	flags := flag.NewFlagSet("codex-launch-wave", flag.ContinueOnError)
	flags.SetOutput(stderr)

	agents := flags.Int("agents", intEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_AGENTS", 1), "numero de agentes Codex a lanzar")
	waveRef := flags.String("wave-ref", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_REF")), "ref opaca de la ola")
	prompt := flags.String("prompt", "", "instrucciones para los agentes")
	promptFile := flags.String("prompt-file", "", "archivo con instrucciones para los agentes")
	projectDir := flags.String("project-dir", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_PROJECT_WORKDIR")), "directorio de trabajo del proyecto")
	runtimeDir := flags.String("runtime-dir", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_RUNTIME_WORKDIR")), "directorio runtime de esta ola")
	commandPath := flags.String("command", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_COMMAND")), "ruta al binario codex")
	sourceCodeHome := flags.String("source-code-home", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_SOURCE_CODEX_HOME")), "CODEX_HOME fuente para copiar credenciales/config")
	model := flags.String("model", firstNonEmptyEnvV0("ORQUESTA_CODEX_WAVE_MODEL", "ORQUESTA_CODEX_MODEL"), "modelo Codex opcional")
	reasoningEffort := flags.String("reasoning-effort", envOrDefaultV0("ORQUESTA_CODEX_WAVE_REASONING_EFFORT", envOrDefaultV0("ORQUESTA_CODEX_REASONING_EFFORT", "medium")), "esfuerzo de razonamiento")
	profile := flags.String("profile", firstNonEmptyEnvV0("ORQUESTA_CODEX_WAVE_PROFILE", "ORQUESTA_CODEX_PROFILE"), "perfil Codex opcional")
	sandbox := flags.String("sandbox", envOrDefaultV0("ORQUESTA_CODEX_WAVE_SANDBOX", "workspace-write"), "sandbox Codex")
	approval := flags.String("approval-policy", envOrDefaultV0("ORQUESTA_CODEX_WAVE_APPROVAL_POLICY", "never"), "politica de aprobacion Codex")
	extraArgs := flags.String("extra-args", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_EXTRA_ARGS")), "argumentos extra para codex exec")
	isolateHome := flags.Bool("isolate-home", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_ISOLATE_HOME", false), "copiar CODEX_HOME por agente bajo el runtime")
	strictCredentialProjection := flags.Bool("strict-credential-projection", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_STRICT_CREDENTIAL_PROJECTION", false), "exigir proyeccion aislada con auth/config presentes")
	projectMemories := flags.Bool("project-memories", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PROJECT_MEMORIES", false), "permitir proyeccion explicita de memories de CODEX_HOME")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME", false), "purgar runtime de esta ola antes de materializar")
	purgeConfirm := flags.String("confirm-purge-runtime", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME_CONFIRM")), "confirmacion explicita: debe coincidir con wave-ref")
	purgeReportOnly := flags.Bool("purge-runtime-report", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME_REPORT", false), "reportar purga sin borrar")
	allowUnmanagedLaunch := flags.Bool("allow-unmanaged-launch", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_ALLOW_UNMANAGED_LAUNCH", false), "breakglass auditado para lanzar agentes fuera del servidor/cola de Orquesta")
	unmanagedLaunchReason := flags.String("unmanaged-launch-reason", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_UNMANAGED_LAUNCH_REASON")), "motivo auditado para el breakglass unmanaged")
	unmanagedLaunchConfirm := flags.String("confirm-unmanaged-launch", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_UNMANAGED_LAUNCH_CONFIRM")), "confirmacion explicita unmanaged: debe coincidir con wave-ref")

	if err := flags.Parse(args); err != nil {
		return codexWaveConfigV0{}, err
	}
	promptText, err := codexWavePromptTextV0(*prompt, *promptFile, flags.Args())
	if err != nil {
		return codexWaveConfigV0{}, err
	}
	projectWorkDir, err := codexWaveProjectDirV0(*projectDir)
	if err != nil {
		return codexWaveConfigV0{}, err
	}
	ref, err := codexWaveRefV0(*waveRef)
	if err != nil {
		return codexWaveConfigV0{}, err
	}
	rootRuntimeDir, err := codexWaveRuntimeDirForLaunchV0(*runtimeDir, projectWorkDir, ref, *purgeRuntime)
	if err != nil {
		return codexWaveConfigV0{}, err
	}
	resolvedCommand, err := codexWaveCommandPathV0(*commandPath)
	if err != nil {
		return codexWaveConfigV0{}, err
	}
	sourceHome := codexWaveSourceCodeHomeV0(*sourceCodeHome)
	if *isolateHome && sourceHome == "" {
		return codexWaveConfigV0{}, errors.New("source_code_home_unavailable")
	}
	if *agents <= 0 || *agents > 64 {
		return codexWaveConfigV0{}, errors.New("agents_out_of_range")
	}

	config := codexWaveConfigV0{
		Agents:                     *agents,
		WaveRef:                    ref,
		Prompt:                     promptText,
		ProjectWorkDir:             projectWorkDir,
		RuntimeWorkDir:             rootRuntimeDir,
		CommandPath:                resolvedCommand,
		SourceCodeHome:             sourceHome,
		PathEnv:                    envOrDefaultV0("ORQUESTA_CODEX_WAVE_PATH", envOrDefaultV0("ORQUESTA_CODEX_PATH", os.Getenv("PATH"))),
		Model:                      strings.TrimSpace(*model),
		ReasoningEffort:            codexReasoningEffortPolicyV0(*reasoningEffort),
		Profile:                    strings.TrimSpace(*profile),
		Sandbox:                    strings.TrimSpace(*sandbox),
		ApprovalPolicy:             strings.TrimSpace(*approval),
		ExtraArgs:                  strings.Fields(*extraArgs),
		IsolateHome:                *isolateHome,
		DryRun:                     *dryRun,
		PurgeRuntime:               *purgeRuntime,
		PurgeConfirm:               strings.TrimSpace(*purgeConfirm),
		PurgeReportOnly:            *purgeReportOnly,
		AllowUnmanagedLaunch:       *allowUnmanagedLaunch,
		UnmanagedLaunchReason:      strings.TrimSpace(*unmanagedLaunchReason),
		UnmanagedLaunchConfirm:     strings.TrimSpace(*unmanagedLaunchConfirm),
		CredentialProjectionPolicy: codexWaveCredentialProjectionPolicyV0FromFlags(*strictCredentialProjection, *projectMemories),
	}
	if err := codexWaveValidateProjectionPolicyV0(config); err != nil {
		return codexWaveConfigV0{}, err
	}
	return config, nil
}

func codexWavePromptTextV0(prompt string, promptFile string, trailing []string) (string, error) {
	if strings.TrimSpace(promptFile) != "" {
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return "", err
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return "", errors.New("prompt_empty")
		}
		return text, nil
	}
	if strings.TrimSpace(prompt) != "" {
		return strings.TrimSpace(prompt), nil
	}
	if len(trailing) > 0 {
		text := strings.TrimSpace(strings.Join(trailing, " "))
		if text != "" {
			return text, nil
		}
	}
	return "", errors.New("prompt_required")
}

func codexReasoningEffortPolicyV0(value string) string {
	switch strings.TrimSpace(value) {
	case "low", "medium", "high", "xhigh":
		return strings.TrimSpace(value)
	default:
		return "medium"
	}
}

func codexWaveProjectDirV0(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return projectDirFromEnvV0()
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return "", err
	}
	return abs, nil
}

func codexWaveRuntimeDirV0(raw string, projectWorkDir string, waveRef string) (string, error) {
	abs, err := codexWaveRuntimeDirPathV0(raw, projectWorkDir, waveRef)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return "", err
	}
	return abs, nil
}

func codexWaveRuntimeDirForLaunchV0(raw string, projectWorkDir string, waveRef string, purgeRuntime bool) (string, error) {
	if purgeRuntime {
		return codexWaveRuntimeDirPathV0(raw, projectWorkDir, waveRef)
	}
	return codexWaveRuntimeDirV0(raw, projectWorkDir, waveRef)
}

func codexWaveRuntimeDirPathV0(raw string, projectWorkDir string, waveRef string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		base := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_RUNTIME_WORKDIR"))
		if base == "" {
			base = filepath.Join(projectWorkDir, ".orquesta-runtime")
		}
		value = filepath.Join(base, "codex-waves", waveRef)
	}
	return filepath.Abs(value)
}

func codexWavePurgeRuntimeDirV0(runtimeDir string, waveRef string) error {
	return errors.New("runtime_purge_requires_confirmed_report")
}

func codexWaveCommandPathV0(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "codex"
	}
	if filepath.IsAbs(value) {
		return value, nil
	}
	resolved, err := exec.LookPath(value)
	if err != nil {
		return "", errors.New("codex_command_unavailable")
	}
	return resolved, nil
}

func codexWaveSourceCodeHomeV0(raw string) string {
	for _, candidate := range []string{raw, strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_CODE_HOME")), strings.TrimSpace(os.Getenv("CODEX_HOME"))} {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err == nil {
			return abs
		}
	}
	if home := homeDirV0(); home != "" {
		return filepath.Join(home, ".codex")
	}
	return ""
}

func codexWaveRefV0(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "codex-wave-v0-" + time.Now().UTC().Format("20060102T150405Z"), nil
	}
	if strings.ContainsAny(value, "\x00\r\n/\\") || strings.Contains(value, "..") {
		return "", errors.New("wave_ref_invalid")
	}
	return value, nil
}

func boolEnvOrDefaultV0(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
