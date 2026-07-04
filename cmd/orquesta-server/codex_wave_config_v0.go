package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

var codexWaveRefGeneratorV0 = orquestaruntime.NewSystemRefGeneratorV0()

func codexWaveConfigFromArgsV0(args []string, stderr io.Writer) (codexWaveConfigV0, error) {
	projectConfig := codexWaveProjectConfigFromArgsEnvV0(args)
	flags := flag.NewFlagSet("codex-launch-wave", flag.ContinueOnError)
	flags.SetOutput(stderr)

	agents := flags.Int("agents", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveAgentsV0, 1), "numero de agentes Codex a lanzar")
	waveRef := flags.String("wave-ref", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveRefV0, ""), "ref opaca de la ola")
	prompt := flags.String("prompt", "", "instrucciones para los agentes")
	promptFile := flags.String("prompt-file", "", "archivo con instrucciones para los agentes")
	projectDir := flags.String("project-dir", strings.TrimSpace(os.Getenv(envCodexProjectWorkDirV0)), "directorio de trabajo del proyecto")
	runtimeDir := flags.String("runtime-dir", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveRuntimeWorkDirV0, ""), "directorio runtime de esta ola")
	commandPath := flags.String("command", strings.TrimSpace(os.Getenv(envCodexCommandV0)), "ruta al binario codex")
	sourceCodeHome := flags.String("source-code-home", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveSourceCodeHomeV0, ""), "CODEX_HOME fuente para copiar credenciales/config")
	model := flags.String("model", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveModelV0, firstNonEmptyEnvV0(envCodexModelV0)), "modelo Codex opcional")
	reasoningEffort := flags.String("reasoning-effort", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveReasoningEffortV0, envOrDefaultV0(envCodexReasoningEffortV0, "medium")), "esfuerzo de razonamiento")
	profile := flags.String("profile", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveProfileV0, firstNonEmptyEnvV0(envCodexProfileV0)), "perfil Codex opcional")
	sandbox := flags.String("sandbox", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveSandboxV0, "workspace-write"), "sandbox Codex")
	approval := flags.String("approval-policy", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveApprovalPolicyV0, "never"), "politica de aprobacion Codex")
	extraArgs := flags.String("extra-args", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveExtraArgsV0, ""), "argumentos extra para codex exec")
	isolateHome := flags.Bool("isolate-home", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveIsolateHomeV0, false), "copiar CODEX_HOME por agente bajo el runtime")
	strictCredentialProjection := flags.Bool("strict-credential-projection", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveStrictCredentialProjectionV0, false), "exigir proyeccion aislada con auth/config presentes")
	projectMemories := flags.Bool("project-memories", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectMemoriesV0, false), "permitir proyeccion explicita de memories de CODEX_HOME")
	projectionMaxFiles := flags.Int("projection-max-files", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxFilesV0, codexWaveProjectionDefaultMaxFilesV0), "maximo de ficheros a copiar desde CODEX_HOME")
	projectionMaxFileBytes := flags.Int("projection-max-file-bytes", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxFileBytesV0, int(codexWaveProjectionDefaultMaxFileBytesV0)), "maximo de bytes por fichero proyectado")
	projectionMaxTotalBytes := flags.Int("projection-max-total-bytes", codexWaveIntValueFromProjectConfigFileV0(projectConfig, envCodexWaveProjectionMaxTotalBytesV0, int(codexWaveProjectionDefaultMaxTotalBytesV0)), "maximo total de bytes proyectados desde CODEX_HOME")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeV0, false), "purgar runtime de esta ola antes de materializar")
	purgeConfirm := flags.String("confirm-purge-runtime", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeConfirmV0, ""), "confirmacion explicita: debe coincidir con wave-ref")
	purgeReportOnly := flags.Bool("purge-runtime-report", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWavePurgeRuntimeReportV0, false), "reportar purga sin borrar")
	allowUnmanagedLaunch := flags.Bool("allow-unmanaged-launch", codexWaveBoolValueFromProjectConfigFileV0(projectConfig, envCodexWaveAllowUnmanagedLaunchV0, false), "breakglass auditado para lanzar agentes fuera del servidor/cola de Orquesta")
	unmanagedLaunchReason := flags.String("unmanaged-launch-reason", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveUnmanagedLaunchReasonV0, ""), "motivo auditado para el breakglass unmanaged")
	unmanagedLaunchConfirm := flags.String("confirm-unmanaged-launch", codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWaveUnmanagedLaunchConfirmV0, ""), "confirmacion explicita unmanaged: debe coincidir con wave-ref")

	if err := flags.Parse(args); err != nil {
		return codexWaveConfigV0{}, err
	}
	promptText, inputReceipt, err := codexWavePromptTextV0(*prompt, *promptFile, flags.Args())
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
		Agents:                 *agents,
		WaveRef:                ref,
		Prompt:                 promptText,
		ProjectWorkDir:         projectWorkDir,
		RuntimeWorkDir:         rootRuntimeDir,
		CommandPath:            resolvedCommand,
		SourceCodeHome:         sourceHome,
		PathEnv:                codexWaveStringValueFromProjectConfigFileV0(projectConfig, envCodexWavePathV0, envOrDefaultV0(envCodexPathV0, os.Getenv("PATH"))),
		Model:                  strings.TrimSpace(*model),
		ReasoningEffort:        codexReasoningEffortPolicyV0(*reasoningEffort),
		Profile:                strings.TrimSpace(*profile),
		Sandbox:                strings.TrimSpace(*sandbox),
		ApprovalPolicy:         strings.TrimSpace(*approval),
		ExtraArgs:              strings.Fields(*extraArgs),
		IsolateHome:            *isolateHome,
		DryRun:                 *dryRun,
		PurgeRuntime:           *purgeRuntime,
		PurgeConfirm:           strings.TrimSpace(*purgeConfirm),
		PurgeReportOnly:        *purgeReportOnly,
		AllowUnmanagedLaunch:   *allowUnmanagedLaunch,
		UnmanagedLaunchReason:  strings.TrimSpace(*unmanagedLaunchReason),
		UnmanagedLaunchConfirm: strings.TrimSpace(*unmanagedLaunchConfirm),
		CredentialProjectionPolicy: codexWaveCredentialProjectionPolicyWithBoundsV0(
			*strictCredentialProjection,
			*projectMemories,
			*projectionMaxFiles,
			int64(*projectionMaxFileBytes),
			int64(*projectionMaxTotalBytes),
		),
		OperatorInputs: []codexWaveOperatorInputReceiptV0{inputReceipt},
	}
	if err := codexWaveValidateProjectionPolicyV0(config); err != nil {
		return codexWaveConfigV0{}, err
	}
	return config, nil
}

func codexWavePromptTextV0(prompt string, promptFile string, trailing []string) (string, codexWaveOperatorInputReceiptV0, error) {
	if strings.TrimSpace(promptFile) != "" {
		text, receipt, err := codexWaveOperatorTextFromFileV0("prompt", promptFile)
		if err != nil {
			return "", codexWaveOperatorInputReceiptV0{}, err
		}
		return text, receipt, nil
	}
	if strings.TrimSpace(prompt) != "" {
		text := strings.TrimSpace(prompt)
		return text, codexWaveOperatorInputReceiptFromTextV0("prompt", "operator_inline", text), nil
	}
	if len(trailing) > 0 {
		text := strings.TrimSpace(strings.Join(trailing, " "))
		if text != "" {
			return text, codexWaveOperatorInputReceiptFromTextV0("prompt", "operator_trailing", text), nil
		}
	}
	return "", codexWaveOperatorInputReceiptV0{}, errors.New("prompt_required")
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
		base := strings.TrimSpace(os.Getenv(envCodexRuntimeWorkDirV0))
		if base == "" {
			base = filepath.Join(projectWorkDir, ".orquesta-runtime")
		}
		value = filepath.Join(base, "codex-waves", waveRef)
	}
	return filepath.Abs(value)
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
	for _, candidate := range []string{raw, strings.TrimSpace(os.Getenv(envCodexCodeHomeV0)), strings.TrimSpace(os.Getenv(envCodexCodeHomeLegacyV0))} {
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
		ref, _ := codexWaveRefGeneratorV0.NextRefV0("codex-wave-v0", "wave")
		return ref, nil
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
