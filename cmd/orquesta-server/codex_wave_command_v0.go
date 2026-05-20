package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const codexWaveSummarySchemaVersionV0 = "orquesta_codex_wave_launch.v0"
const codexWaveRegistryFileNameV0 = "codex_wave_registry_v0.json"

type codexWaveLaunchSummaryV0 struct {
	SchemaVersion  string                    `json:"schema_version"`
	WaveRef        string                    `json:"wave_ref"`
	AgentCount     int                       `json:"agent_count"`
	ProjectWorkDir string                    `json:"project_work_dir"`
	RuntimeWorkDir string                    `json:"runtime_work_dir"`
	RegistryPath   string                    `json:"registry_path,omitempty"`
	Sandbox        string                    `json:"sandbox"`
	ApprovalPolicy string                    `json:"approval_policy"`
	CreatedAt      string                    `json:"created_at,omitempty"`
	UpdatedAt      string                    `json:"updated_at,omitempty"`
	DryRun         bool                      `json:"dry_run,omitempty"`
	Agents         []codexWaveAgentSummaryV0 `json:"agents"`
	Errors         []codexWavePublicErrorV0  `json:"errors,omitempty"`
}

type codexWaveAgentSummaryV0 struct {
	AgentRef         string `json:"agent_ref"`
	RuntimeWorkDir   string `json:"runtime_work_dir"`
	PromptPath       string `json:"prompt_path"`
	WrapperPath      string `json:"wrapper_path"`
	StdoutPath       string `json:"stdout_path"`
	StderrPath       string `json:"stderr_path"`
	LastMessagePath  string `json:"last_message_path"`
	HomeDir          string `json:"home_dir,omitempty"`
	CodeHomeDir      string `json:"code_home_dir,omitempty"`
	ProcessRef       string `json:"process_ref,omitempty"`
	SessionRef       string `json:"session_ref,omitempty"`
	LaunchRef        string `json:"launch_ref,omitempty"`
	PID              int    `json:"pid,omitempty"`
	StartedAt        string `json:"started_at,omitempty"`
	StopRequestedAt  string `json:"stop_requested_at,omitempty"`
	StdoutBytes      int64  `json:"stdout_bytes,omitempty"`
	StderrBytes      int64  `json:"stderr_bytes,omitempty"`
	LastMessageBytes int64  `json:"last_message_bytes,omitempty"`
	Status           string `json:"status"`
}

type codexWavePublicErrorV0 struct {
	AgentRef string `json:"agent_ref,omitempty"`
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message,omitempty"`
}

type codexWaveConfigV0 struct {
	Agents          int
	WaveRef         string
	Prompt          string
	ProjectWorkDir  string
	RuntimeWorkDir  string
	CommandPath     string
	SourceCodeHome  string
	PathEnv         string
	Model           string
	ReasoningEffort string
	Profile         string
	Sandbox         string
	ApprovalPolicy  string
	ExtraArgs       []string
	IsolateHome     bool
	DryRun          bool
	PurgeRuntime    bool
	AgentPrompts    []string
}

func codexLaunchWaveCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := codexWaveConfigFromArgsV0(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-wave: %v\n", err)
		return 2
	}
	summary, err := runCodexLaunchWaveV0(context.Background(), config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "codex-launch-wave: %v\n", err)
		return 1
	}
	_ = json.NewEncoder(stdout).Encode(summary)
	if len(summary.Errors) > 0 {
		return 1
	}
	return 0
}

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
	reasoningEffort := flags.String("reasoning-effort", envOrDefaultV0("ORQUESTA_CODEX_WAVE_REASONING_EFFORT", envOrDefaultV0("ORQUESTA_CODEX_REASONING_EFFORT", "xhigh")), "esfuerzo de razonamiento")
	profile := flags.String("profile", firstNonEmptyEnvV0("ORQUESTA_CODEX_WAVE_PROFILE", "ORQUESTA_CODEX_PROFILE"), "perfil Codex opcional")
	sandbox := flags.String("sandbox", envOrDefaultV0("ORQUESTA_CODEX_WAVE_SANDBOX", "danger-full-access"), "sandbox Codex")
	approval := flags.String("approval-policy", envOrDefaultV0("ORQUESTA_CODEX_WAVE_APPROVAL_POLICY", "never"), "politica de aprobacion Codex")
	extraArgs := flags.String("extra-args", strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_WAVE_EXTRA_ARGS")), "argumentos extra para codex exec")
	isolateHome := flags.Bool("isolate-home", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_ISOLATE_HOME", false), "copiar CODEX_HOME por agente bajo el runtime")
	dryRun := flags.Bool("dry-run", false, "materializar prompts/wrappers sin arrancar procesos")
	purgeRuntime := flags.Bool("purge-runtime", boolEnvOrDefaultV0("ORQUESTA_CODEX_WAVE_PURGE_RUNTIME", false), "purgar runtime de esta ola antes de materializar")

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
	rootRuntimeDir, err := codexWaveRuntimeDirV0(*runtimeDir, projectWorkDir, ref)
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

	return codexWaveConfigV0{
		Agents:          *agents,
		WaveRef:         ref,
		Prompt:          promptText,
		ProjectWorkDir:  projectWorkDir,
		RuntimeWorkDir:  rootRuntimeDir,
		CommandPath:     resolvedCommand,
		SourceCodeHome:  sourceHome,
		PathEnv:         envOrDefaultV0("ORQUESTA_CODEX_WAVE_PATH", envOrDefaultV0("ORQUESTA_CODEX_PATH", os.Getenv("PATH"))),
		Model:           strings.TrimSpace(*model),
		ReasoningEffort: strings.TrimSpace(*reasoningEffort),
		Profile:         strings.TrimSpace(*profile),
		Sandbox:         strings.TrimSpace(*sandbox),
		ApprovalPolicy:  strings.TrimSpace(*approval),
		ExtraArgs:       strings.Fields(*extraArgs),
		IsolateHome:     *isolateHome,
		DryRun:          *dryRun,
		PurgeRuntime:    *purgeRuntime,
	}, nil
}

func runCodexLaunchWaveV0(
	ctx context.Context,
	config codexWaveConfigV0,
) (codexWaveLaunchSummaryV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.PurgeRuntime {
		if err := codexWavePurgeRuntimeDirV0(config.RuntimeWorkDir, config.WaveRef); err != nil {
			return codexWaveLaunchSummaryV0{}, err
		}
	}
	if err := os.MkdirAll(config.RuntimeWorkDir, 0o700); err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	summary := codexWaveLaunchSummaryV0{
		SchemaVersion:  codexWaveSummarySchemaVersionV0,
		WaveRef:        config.WaveRef,
		AgentCount:     config.Agents,
		ProjectWorkDir: config.ProjectWorkDir,
		RuntimeWorkDir: config.RuntimeWorkDir,
		RegistryPath:   codexWaveRegistryPathV0(config.RuntimeWorkDir),
		Sandbox:        config.Sandbox,
		ApprovalPolicy: config.ApprovalPolicy,
		CreatedAt:      now,
		UpdatedAt:      now,
		DryRun:         config.DryRun,
		Agents:         make([]codexWaveAgentSummaryV0, 0, config.Agents),
	}
	for i := 1; i <= config.Agents; i++ {
		agent, err := codexWaveMaterializeAgentV0(config, i)
		if err != nil {
			summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
				AgentRef: fmt.Sprintf("%s-agent-%02d", config.WaveRef, i),
				Code:     "materialize_failed",
				Message:  err.Error(),
			})
			continue
		}
		if config.DryRun {
			agent.Status = "dry_run"
			summary.Agents = append(summary.Agents, agent)
			continue
		}
		pid, err := codexWaveStartAgentProcessV0(ctx, agent.WrapperPath, config.ProjectWorkDir)
		if err != nil {
			agent.Status = "launch_failed"
			summary.Errors = append(summary.Errors, codexWavePublicErrorV0{
				AgentRef: agent.AgentRef,
				Code:     "launch_failed",
				Message:  err.Error(),
			})
			summary.Agents = append(summary.Agents, agent)
			continue
		}
		agent.ProcessRef = fmt.Sprintf("%s-process-%02d", config.WaveRef, i)
		agent.SessionRef = fmt.Sprintf("%s-session-%02d", config.WaveRef, i)
		agent.LaunchRef = fmt.Sprintf("%s-launch-%02d", config.WaveRef, i)
		agent.PID = pid
		agent.StartedAt = time.Now().UTC().Format(time.RFC3339)
		agent.Status = "running"
		summary.Agents = append(summary.Agents, agent)
	}
	codexWaveRefreshSummaryV0(&summary)
	if err := codexWaveSaveRegistryV0(summary); err != nil {
		return codexWaveLaunchSummaryV0{}, err
	}
	return summary, nil
}

func codexWaveStartAgentProcessV0(ctx context.Context, wrapperPath string, projectWorkDir string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cmd := exec.Command(wrapperPath)
	cmd.Dir = projectWorkDir
	cmd.Env = []string{}
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	configureDetachedProcessV0(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	go func() {
		_ = cmd.Wait()
	}()
	return pid, nil
}

func codexWaveMaterializeAgentV0(
	config codexWaveConfigV0,
	index int,
) (codexWaveAgentSummaryV0, error) {
	agentRef := fmt.Sprintf("%s-agent-%02d", config.WaveRef, index)
	agentRuntimeDir := filepath.Join(config.RuntimeWorkDir, fmt.Sprintf("agent-%02d", index))
	if err := os.MkdirAll(agentRuntimeDir, 0o700); err != nil {
		return codexWaveAgentSummaryV0{}, err
	}

	homeDir := ""
	codeHomeDir := ""
	var err error
	if config.IsolateHome {
		homeDir, codeHomeDir, err = codexWaveCopyCodeHomeV0(config.SourceCodeHome, agentRuntimeDir)
		if err != nil {
			return codexWaveAgentSummaryV0{}, err
		}
	} else {
		homeDir = homeDirV0()
		codeHomeDir = config.SourceCodeHome
	}

	profile := orquestaruntimecodex.CodexConnectorProfileV0{
		SchemaVersion:   orquestaruntimecodex.CodexConnectorProfileSchemaVersionV0,
		OptIn:           true,
		CommandPath:     config.CommandPath,
		ProjectWorkDir:  config.ProjectWorkDir,
		RuntimeWorkDir:  agentRuntimeDir,
		CodeHomeDir:     codeHomeDir,
		HomeDir:         homeDir,
		PathEnv:         config.PathEnv,
		Model:           config.Model,
		ReasoningEffort: config.ReasoningEffort,
		Profile:         config.Profile,
		Sandbox:         config.Sandbox,
		ApprovalPolicy:  config.ApprovalPolicy,
		ExtraArgs:       append([]string(nil), config.ExtraArgs...),
		PromptHints: []string{
			"codex-launch-wave",
			fmt.Sprintf("wave=%s", config.WaveRef),
			fmt.Sprintf("agent=%d/%d", index, config.Agents),
		},
	}
	if issues := orquestaruntimecodex.ValidateCodexConnectorProfileV0(profile); len(issues) > 0 {
		return codexWaveAgentSummaryV0{}, fmt.Errorf("codex_profile_invalid:%s", issues[0].Field)
	}

	prompt := codexWaveAgentPromptV0(config, agentRef, index)
	promptPath := filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexAgentPromptFileNameV0)
	wrapperPath := filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexWrapperFileNameV0)
	if err := os.WriteFile(promptPath, []byte(prompt), 0o600); err != nil {
		return codexWaveAgentSummaryV0{}, err
	}
	if err := os.WriteFile(wrapperPath, []byte(orquestaruntimecodex.BuildCodexWrapperScriptV0(profile)), 0o700); err != nil {
		return codexWaveAgentSummaryV0{}, err
	}
	return codexWaveAgentSummaryV0{
		AgentRef:        agentRef,
		RuntimeWorkDir:  agentRuntimeDir,
		PromptPath:      promptPath,
		WrapperPath:     wrapperPath,
		StdoutPath:      filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexStdoutFileNameV0),
		StderrPath:      filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		LastMessagePath: filepath.Join(agentRuntimeDir, orquestaruntimecodex.CodexLastMessageFileNameV0),
		HomeDir:         homeDir,
		CodeHomeDir:     codeHomeDir,
		Status:          "materialized",
	}, nil
}

func codexWaveAgentPromptV0(config codexWaveConfigV0, agentRef string, index int) string {
	var b strings.Builder
	b.WriteString("Eres un agente Codex lanzado por Orquesta en una ola operativa opt-in.\n\n")
	b.WriteString("Identidad:\n")
	b.WriteString("- wave_ref: ")
	b.WriteString(config.WaveRef)
	b.WriteString("\n- agent_ref: ")
	b.WriteString(agentRef)
	b.WriteString("\n- agente: ")
	b.WriteString(strconv.Itoa(index))
	b.WriteString(" de ")
	b.WriteString(strconv.Itoa(config.Agents))
	b.WriteString("\n\n")
	b.WriteString("Reglas operativas:\n")
	b.WriteString("- Trabaja en el repositorio indicado por Orquesta y respeta AGENTS.md locales antes de editar.\n")
	b.WriteString("- No borres archivos ni codigo existente sin revisar primero su uso y dejar evidencia clara.\n")
	b.WriteString("- Manten el write-set estrecho y coordina mentalmente tu parte con el resto de la ola.\n")
	b.WriteString("- Al terminar, resume cambios, rutas tocadas, pruebas ejecutadas y bloqueos.\n\n")
	b.WriteString("Instrucciones del operador:\n")
	b.WriteString(codexWavePromptForAgentV0(config, index))
	b.WriteString("\n")
	return b.String()
}

func codexWavePromptForAgentV0(config codexWaveConfigV0, index int) string {
	if index > 0 && index <= len(config.AgentPrompts) {
		if prompt := strings.TrimSpace(config.AgentPrompts[index-1]); prompt != "" {
			return prompt
		}
	}
	return config.Prompt
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

func codexWaveRuntimeDirPathV0(raw string, projectWorkDir string, waveRef string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		base := strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_RUNTIME_WORKDIR"))
		if base == "" {
			base = filepath.Join(projectWorkDir, ".orquesta-runtime")
		}
		value = filepath.Join(base, "codex-waves", waveRef)
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func codexWavePurgeRuntimeDirV0(runtimeDir string, waveRef string) error {
	cleanDir := filepath.Clean(strings.TrimSpace(runtimeDir))
	cleanWaveRef := strings.TrimSpace(waveRef)
	if cleanDir == "" || cleanDir == "." || cleanDir == string(filepath.Separator) {
		return errors.New("runtime_dir_purge_unsafe")
	}
	if cleanWaveRef == "" {
		return errors.New("wave_ref_required_for_purge")
	}
	base := filepath.Base(cleanDir)
	if base != cleanWaveRef && !strings.Contains(cleanDir, string(filepath.Separator)+"codex-waves"+string(filepath.Separator)) {
		return errors.New("runtime_dir_purge_ref_mismatch")
	}
	if err := os.RemoveAll(cleanDir); err != nil {
		return err
	}
	return nil
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
	for _, candidate := range []string{
		raw,
		strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_CODE_HOME")),
		strings.TrimSpace(os.Getenv("CODEX_HOME")),
	} {
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

func codexWaveCopyCodeHomeV0(sourceCodeHome string, agentRuntimeDir string) (string, string, error) {
	source := filepath.Clean(sourceCodeHome)
	if source == "" || !filepath.IsAbs(source) {
		return "", "", errors.New("source_code_home_invalid")
	}
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return "", "", errors.New("source_code_home_unavailable")
	}
	homeDir := filepath.Join(agentRuntimeDir, "home")
	codeHomeDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(codeHomeDir, 0o700); err != nil {
		return "", "", err
	}
	fileNames := []string{
		"auth.json",
		"config.toml",
		"instructions.md",
		"AGENTS.md",
		"installation_id",
		"version.json",
		"models_cache.json",
		".personality_migration",
	}
	for _, name := range fileNames {
		if err := codexWaveCopyFileIfExistsV0(filepath.Join(source, name), filepath.Join(codeHomeDir, name)); err != nil {
			return "", "", err
		}
	}
	for _, name := range []string{"skills", "plugins", "rules", "memories"} {
		if err := codexWaveCopyDirIfExistsV0(filepath.Join(source, name), filepath.Join(codeHomeDir, name)); err != nil {
			return "", "", err
		}
	}
	return homeDir, codeHomeDir, nil
}

func codexWaveCopyFileIfExistsV0(source string, target string) error {
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o600)
}

func codexWaveCopyDirIfExistsV0(source string, target string) error {
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o600
		}
		return os.WriteFile(targetPath, data, mode)
	})
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
