/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimeagente

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type ConnectorConfig struct {
	Slug         string
	Nombre       string
	Transporte   string
	Comando      string
	ArgsJSON     string
	EnvJSON      string
	MetadataJSON string
	Activo       bool
}

type ResumeContext struct {
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	CWD                string
}

const (
	MailboxDeliveryInteractive        = "interactive"
	MailboxDeliveryBootstrapOnly      = "bootstrap_only"
	MailboxDeliveryCoordinatedRestart = "coordinated_restart"
	MailboxDeliverySessionResume      = "session_resume"
)

type LaunchRequest struct {
	Agente       string
	Rol          string
	ProyectoSlug string
	ProyectoRuta string
	Modelo       string
	Razonamiento string
	PerfilTarea  string
	Conector     ConnectorConfig
	Resume       ResumeContext
	TareaID      *int64
}

type LaunchPlan struct {
	Driver               string            `json:"driver"`
	Transporte           string            `json:"transporte"`
	Modo                 string            `json:"modo"`
	Comando              string            `json:"comando"`
	Args                 []string          `json:"args"`
	Env                  map[string]string `json:"env"`
	WorkingDir           string            `json:"working_dir"`
	Branch               string            `json:"branch,omitempty"`
	WorktreePath         string            `json:"worktree_path,omitempty"`
	Modelo               string            `json:"modelo,omitempty"`
	Razonamiento         string            `json:"razonamiento,omitempty"`
	PerfilTarea          string            `json:"perfil_tarea,omitempty"`
	NativeResume         bool              `json:"native_resume"`
	ContinuityPrompt     string            `json:"continuity_prompt,omitempty"`
	BootstrapPrompt      string            `json:"bootstrap_prompt,omitempty"`
	LaunchPromptEmbedded bool              `json:"launch_prompt_embedded,omitempty"`
	LaunchPromptMode     string            `json:"launch_prompt_mode,omitempty"`
	LaunchPromptDelayMS  int               `json:"launch_prompt_delay_ms,omitempty"`
	CanSendInput         *bool             `json:"can_send_input,omitempty"`
	MailboxDeliveryMode  string            `json:"mailbox_delivery_mode,omitempty"`
	RemoteConfigJSON     string            `json:"remote_config_json,omitempty"`
	Notas                []string          `json:"notas,omitempty"`
	TareaID              *int64            `json:"tarea_id,omitempty"`
}

type Driver interface {
	Transport() string
	Prepare(req LaunchRequest) (*LaunchPlan, error)
}

type Registry struct {
	drivers map[string]Driver
}

func NewRegistry(drivers ...Driver) *Registry {
	r := &Registry{drivers: make(map[string]Driver, len(drivers))}
	for _, driver := range drivers {
		if driver == nil {
			continue
		}
		r.drivers[driver.Transport()] = driver
	}
	return r
}

func DefaultRegistry() *Registry {
	return NewRegistry(
		commandDriver{transport: "cli"},
		commandDriver{transport: "mcp_stdio"},
		remoteDriver{transport: "mcp_http"},
		remoteDriver{transport: "api"},
		remoteDriver{transport: "otro"},
	)
}

func (r *Registry) Prepare(req LaunchRequest) (*LaunchPlan, error) {
	if strings.TrimSpace(req.Conector.Transporte) == "" {
		return nil, fmt.Errorf("el conector '%s' no define transporte", req.Conector.Slug)
	}
	driver, ok := r.drivers[req.Conector.Transporte]
	if !ok {
		return nil, fmt.Errorf("no existe driver para el transporte '%s'", req.Conector.Transporte)
	}
	return driver.Prepare(req)
}

type commandDriver struct {
	transport string
}

func (d commandDriver) Transport() string {
	return d.transport
}

func (d commandDriver) Prepare(req LaunchRequest) (*LaunchPlan, error) {
	if strings.TrimSpace(req.Conector.Comando) == "" {
		return nil, fmt.Errorf("el conector '%s' no define comando", req.Conector.Slug)
	}

	args, err := parseStringSliceJSON(req.Conector.ArgsJSON)
	if err != nil {
		return nil, fmt.Errorf("args_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	env, err := parseStringMapJSON(req.Conector.EnvJSON)
	if err != nil {
		return nil, fmt.Errorf("env_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	metadata, err := parseAnyMapJSON(req.Conector.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	metadata = normalizeConnectorMetadata(req.Conector, metadata)
	resume := SanitizarResumeParaConector(req.Conector, req.Resume)
	req.Resume = resume

	workingDir := strings.TrimSpace(resume.CWD)
	if workingDir == "" {
		workingDir = strings.TrimSpace(req.ProyectoRuta)
	}
	plan := &LaunchPlan{
		Driver:       d.transport,
		Transporte:   d.transport,
		Modo:         "launch",
		WorkingDir:   workingDir,
		Branch:       strings.TrimSpace(resume.Branch),
		WorktreePath: inferWorktreePath(workingDir),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		PerfilTarea:  strings.TrimSpace(req.PerfilTarea),
		TareaID:      req.TareaID,
	}
	applyExecutionProfile(plan, metadata)

	vars := connectorTemplateVars(req, workingDir)
	vars["model"] = plan.Modelo
	vars["reasoning"] = plan.Razonamiento
	vars["task_profile"] = plan.PerfilTarea

	comando := resolveConnectorCommandPath(expandConnectorTemplate(strings.TrimSpace(req.Conector.Comando), vars), strings.TrimSpace(req.ProyectoRuta))
	expandedArgs := expandConnectorArgs(args, vars)
	if isCodexCLIRequest(req) {
		comando = canonicalCodexProfileWrapper(req, comando)
		expandedArgs = canonicalCodexProfileArgs(req, expandedArgs)
	}
	plan.Comando = comando
	plan.Args = expandedArgs
	plan.Env = expandConnectorEnv(env, vars)
	asegurarEntornoBaseOrquesta(plan.Env, vars)
	if filepath.Base(strings.TrimSpace(plan.Comando)) == "codex-perfil" {
		ensureCodexProfileWrapperEnv(plan.Env)
	}
	asegurarPathHerramientasBase(plan.Env, "go", "git", "rg")
	asegurarPathParaComando(plan.Env, plan.Comando)
	if applyLocalControlHints(plan, metadata) {
		plan.Notas = append(plan.Notas, "El conector aporta capacidades locales de control desde metadata_json.")
	}

	if applyLaunchHints(plan, metadata, req) {
		plan.Notas = append(plan.Notas, "El conector aporta hints de arranque desde metadata_json.")
	}

	if resumeActivaModoLocal(resume) {
		plan.Modo = "resume"
		plan.NativeResume = applyResumeHints(plan, metadata, resume)
		applyResumeDeliveryHints(plan, req)
		if !plan.NativeResume {
			plan.ContinuityPrompt = construirPromptContinuidad(req)
			plan.Notas = append(plan.Notas, "El conector no declara estrategia nativa de reanudación; se devuelve prompt de continuidad.")
		}
	}

	return plan, nil
}

type remoteDriver struct {
	transport string
}

func (d remoteDriver) Transport() string {
	return d.transport
}

func (d remoteDriver) Prepare(req LaunchRequest) (*LaunchPlan, error) {
	metadata, err := parseAnyMapJSON(req.Conector.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	metadata = normalizeConnectorMetadata(req.Conector, metadata)
	resume := SanitizarResumeParaConector(req.Conector, req.Resume)
	req.Resume = resume
	env, err := parseStringMapJSON(req.Conector.EnvJSON)
	if err != nil {
		return nil, fmt.Errorf("env_json inválido para '%s': %w", req.Conector.Slug, err)
	}

	plan := &LaunchPlan{
		Driver:       d.transport,
		Transporte:   d.transport,
		Modo:         "launch",
		Comando:      req.Conector.Comando,
		Env:          env,
		WorkingDir:   strings.TrimSpace(req.ProyectoRuta),
		Branch:       strings.TrimSpace(resume.Branch),
		WorktreePath: inferWorktreePath(strings.TrimSpace(req.ProyectoRuta)),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		PerfilTarea:  strings.TrimSpace(req.PerfilTarea),
		TareaID:      req.TareaID,
		Notas: []string{
			"El transporte remoto requiere un adaptador externo que consuma este plan.",
		},
	}
	applyExecutionProfile(plan, metadata)
	plan.RemoteConfigJSON = buildRemoteConfigJSON(req, metadata)

	if endpoint := stringMetadata(metadata, "endpoint"); endpoint != "" {
		plan.Notas = append(plan.Notas, "Endpoint: "+endpoint)
	}
	if resumeActivaModoRemoto(resume) {
		plan.Modo = "resume"
		plan.ContinuityPrompt = construirPromptContinuidad(req)
	}
	return plan, nil
}

func inferWorktreePath(workingDir string) string {
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return ""
	}
	normalized := filepath.ToSlash(workingDir)
	if strings.Contains(normalized, "/.orquesta-worktrees/") {
		return workingDir
	}
	return ""
}

func resolveConnectorCommandPath(command, projectPath string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if filepath.IsAbs(command) {
		return filepath.Clean(command)
	}
	if !strings.Contains(command, string(os.PathSeparator)) && !strings.Contains(command, "/") {
		if resolved, err := exec.LookPath(command); err == nil && strings.TrimSpace(resolved) != "" {
			return filepath.Clean(resolved)
		}
		if resolved := resolveConnectorCommandPathFromUserHome(command); resolved != "" {
			return resolved
		}
		return command
	}
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return command
	}
	if !filepath.IsAbs(projectPath) {
		if abs, err := filepath.Abs(projectPath); err == nil {
			projectPath = abs
		}
	}
	if projectPath == "" {
		return command
	}
	return filepath.Clean(filepath.Join(projectPath, command))
}

func resolveConnectorCommandPathFromUserHome(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(home, ".local", "bin", command),
		filepath.Join(home, ".cargo", "bin", command),
	}
	switch command {
	case "go":
		candidates = append(candidates,
			"/usr/local/go/bin/go",
			"/usr/lib/go/bin/go",
		)
	case "git":
		candidates = append(candidates,
			"/usr/bin/git",
			"/usr/local/bin/git",
		)
	case "rg":
		candidates = append(candidates,
			"/usr/bin/rg",
			"/usr/local/bin/rg",
		)
	}
	if matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", command)); err == nil && len(matches) > 0 {
		sort.Strings(matches)
		for i := len(matches) - 1; i >= 0; i-- {
			candidates = append(candidates, matches[i])
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func asegurarPathParaComando(env map[string]string, command string) {
	command = strings.TrimSpace(command)
	if !filepath.IsAbs(command) {
		return
	}
	dir := strings.TrimSpace(filepath.Dir(command))
	if dir == "" || dir == "." || dir == string(os.PathSeparator) {
		return
	}
	current := ""
	if env != nil {
		current = strings.TrimSpace(env["PATH"])
	}
	if current == "" {
		current = strings.TrimSpace(os.Getenv("PATH"))
	}
	parts := strings.Split(current, string(os.PathListSeparator))
	for _, part := range parts {
		if filepath.Clean(strings.TrimSpace(part)) == filepath.Clean(dir) {
			if env != nil {
				env["PATH"] = current
			}
			return
		}
	}
	if env == nil {
		return
	}
	if current == "" {
		env["PATH"] = dir
		return
	}
	env["PATH"] = dir + string(os.PathListSeparator) + current
}

func asegurarPathHerramientasBase(env map[string]string, commands ...string) {
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		if resolved, err := exec.LookPath(command); err == nil && strings.TrimSpace(resolved) != "" {
			asegurarPathParaComando(env, resolved)
			continue
		}
		if resolved := resolveConnectorCommandPathFromUserHome(command); resolved != "" {
			asegurarPathParaComando(env, resolved)
		}
	}
}

func asegurarEntornoBaseOrquesta(env map[string]string, vars map[string]string) {
	if env == nil {
		return
	}
	if strings.TrimSpace(env["ORQUESTA_SERVER_URL"]) == "" {
		if value := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_URL")); value != "" {
			env["ORQUESTA_SERVER_URL"] = value
		}
	}
	if strings.TrimSpace(env["ORQUESTA_BIN"]) == "" {
		if value := strings.TrimSpace(vars["orquesta_executable"]); value != "" {
			env["ORQUESTA_BIN"] = value
		}
	}
}

func canonicalCodexProfileWrapper(req LaunchRequest, resolvedCommand string) string {
	resolvedCommand = strings.TrimSpace(resolvedCommand)
	if filepath.Base(resolvedCommand) == "codex-perfil" {
		return resolvedCommand
	}
	if !codexDebeUsarWrapperPerfiles() {
		return resolvedCommand
	}
	if wrapper := codexProfileWrapperPath(strings.TrimSpace(req.ProyectoRuta)); wrapper != "" {
		return wrapper
	}
	return resolvedCommand
}

func canonicalCodexProfileArgs(req LaunchRequest, args []string) []string {
	if !codexDebeUsarWrapperPerfiles() {
		return args
	}
	profile := strings.TrimSpace(req.Agente)
	if profile == "" {
		return args
	}
	if len(args) > 0 && strings.TrimSpace(args[0]) == profile {
		return args
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, profile)
	out = append(out, args...)
	return out
}

func codexDebeUsarWrapperPerfiles() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_USE_PROFILE_WRAPPER")))
	return raw == "1" || raw == "true" || raw == "yes"
}

func codexProfileWrapperPath(projectPath string) string {
	if base := strings.TrimSpace(os.Getenv("CODEX_MULTI_BASE")); base != "" {
		return filepath.Clean(filepath.Join(base, "bin", "codex-perfil"))
	}
	projectPath = strings.TrimSpace(projectPath)
	if projectPath != "" {
		if !filepath.IsAbs(projectPath) {
			if abs, err := filepath.Abs(projectPath); err == nil {
				projectPath = abs
			}
		}
		current := projectPath
		for current != "" {
			parent := filepath.Dir(current)
			if parent == "" || parent == current {
				break
			}
			candidate := filepath.Join(parent, "codex-perfiles", "bin", "codex-perfil")
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return filepath.Clean(candidate)
			}
			current = parent
		}
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, "Trabajo", "codex-perfiles", "bin", "codex-perfil")
	}
	return "codex-perfil"
}

func ensureCodexProfileWrapperEnv(env map[string]string) {
	if env == nil {
		return
	}
	if strings.TrimSpace(env["ORQUESTA_TERMINAL_BACKEND"]) == "" {
		env["ORQUESTA_TERMINAL_BACKEND"] = "tmux"
	}
	if codexBin := strings.TrimSpace(env["CODEX_BIN"]); codexBin != "" {
		asegurarPathParaComando(env, codexBin)
		return
	}
	codexBin := strings.TrimSpace(resolveCodexBinaryPath())
	if codexBin == "" {
		return
	}
	env["CODEX_BIN"] = codexBin
	asegurarPathParaComando(env, codexBin)
}

func resolveCodexBinaryPath() string {
	if resolved, err := exec.LookPath("codex"); err == nil && strings.TrimSpace(resolved) != "" {
		return resolved
	}
	candidates := make([]string, 0, 8)
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "codex"),
			filepath.Join(home, "bin", "codex"),
			filepath.Join(home, ".npm-global", "bin", "codex"),
		)
		if matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "codex")); err == nil {
			sort.Strings(matches)
			for i := len(matches) - 1; i >= 0; i-- {
				candidates = append(candidates, matches[i])
			}
		}
	}
	candidates = append(candidates,
		"/usr/local/bin/codex",
		"/usr/bin/codex",
	)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		return filepath.Clean(candidate)
	}
	return ""
}

func normalizeConnectorMetadata(conector ConnectorConfig, metadata map[string]any) map[string]any {
	if !isCodexCLIConnector(conector) {
		return metadata
	}
	legacyReasoning := stringMetadata(metadata, "reasoning_flag") != ""
	needsCopy := legacyReasoning
	if !hasMetadataKey(metadata, "can_send_input") {
		needsCopy = true
	}
	for _, key := range []string{"sandbox_flag", "sandbox_mode", "approval_flag", "approval_policy"} {
		if !hasMetadataKey(metadata, key) {
			needsCopy = true
		}
	}
	if !needsCopy {
		return metadata
	}
	out := make(map[string]any, len(metadata)+6)
	for k, v := range metadata {
		if strings.EqualFold(strings.TrimSpace(k), "reasoning_flag") {
			continue
		}
		out[k] = v
	}
	if legacyReasoning {
		if stringMetadata(out, "reasoning_config_key") == "" {
			out["reasoning_config_key"] = "model_reasoning_effort"
		}
		if stringMetadata(out, "config_flag") == "" {
			out["config_flag"] = "-c"
		}
	}
	// Codex TUI is currently unstable under injected stdin. Treat mailbox as a
	// durable queue to be consumed on the next natural restart instead of
	// forcing coordinated restarts for routine nudges.
	if !hasMetadataKey(out, "can_send_input") {
		out["can_send_input"] = false
	}
	if !hasMetadataKey(out, "can_run_slash_commands") {
		out["can_run_slash_commands"] = false
	}
	// Preserve explicit launch-prompt embedding metadata for Codex. Forcing
	// post_start here deadlocks bootstrap_only runtimes: they start without a
	// first task, never create a resumable session, and mailbox delivery stays
	// pending forever.
	if stringMetadata(out, "launch_prompt_transport") == "" &&
		!boolMetadata(out, "launch_prompt_positional") &&
		stringMetadata(out, "launch_prompt_flag") == "" &&
		stringMetadata(out, "launch_prompt_env") == "" &&
		!boolMetadata(out, "launch_prompt_native") {
		out["launch_prompt_transport"] = "post_start"
	}
	if stringMetadata(out, "mailbox_delivery_mode") == "" {
		out["mailbox_delivery_mode"] = MailboxDeliveryBootstrapOnly
	}
	if stringMetadata(out, "sandbox_flag") == "" {
		out["sandbox_flag"] = "--sandbox"
	}
	if stringMetadata(out, "sandbox_mode") == "" {
		out["sandbox_mode"] = "danger-full-access"
	}
	if stringMetadata(out, "approval_flag") == "" {
		out["approval_flag"] = "--ask-for-approval"
	}
	if stringMetadata(out, "approval_policy") == "" {
		out["approval_policy"] = "never"
	}
	return out
}

func NormalizeConnectorMetadataJSON(conector ConnectorConfig) (string, error) {
	metadata, err := parseAnyMapJSON(conector.MetadataJSON)
	if err != nil {
		return "", err
	}
	normalized := normalizeConnectorMetadata(conector, metadata)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func isCodexCLIConnector(conector ConnectorConfig) bool {
	if strings.EqualFold(strings.TrimSpace(conector.Slug), "codex-cli") {
		return true
	}
	comando := strings.ToLower(strings.TrimSpace(filepath.Base(strings.TrimSpace(conector.Comando))))
	return comando == "codex" || comando == "codex-cli" || strings.HasPrefix(comando, "codex-perfil")
}

func isCodexCLIRequest(req LaunchRequest) bool {
	return isCodexCLIConnector(req.Conector)
}

func isOllamaCLIConnector(conector ConnectorConfig) bool {
	comando := strings.ToLower(strings.TrimSpace(filepath.Base(strings.TrimSpace(conector.Comando))))
	return EsConectorFamiliaOllama(conector.Slug, comando)
}

func ModeloCompatibleConConector(conector ConnectorConfig, modelo string) bool {
	modelo = strings.ToLower(strings.TrimSpace(modelo))
	if modelo == "" {
		return true
	}
	if isOllamaCLIConnector(conector) {
		return true
	}
	familia := ""
	metadata, err := parseAnyMapJSON(conector.MetadataJSON)
	if err == nil {
		metadata = normalizeConnectorMetadata(conector, metadata)
		familia = strings.ToLower(strings.TrimSpace(stringMetadata(metadata, "familia")))
	}
	if familia == "" {
		switch {
		case isCodexCLIConnector(conector):
			familia = "openai"
		default:
			comando := strings.ToLower(strings.TrimSpace(filepath.Base(strings.TrimSpace(conector.Comando))))
			switch comando {
			case "claude", "claude-code":
				familia = "anthropic"
			case "gemini":
				familia = "google"
			}
		}
	}
	switch familia {
	case "openai":
		return modeloCompatibleFamiliaOpenAI(modelo)
	case "anthropic":
		return strings.HasPrefix(modelo, "claude")
	case "google":
		return strings.HasPrefix(modelo, "gemini")
	}
	return true
}

func modeloCompatibleFamiliaOpenAI(modelo string) bool {
	for _, prefix := range []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"codex",
	} {
		if strings.HasPrefix(modelo, prefix) {
			return true
		}
	}
	return false
}

func SanitizarResumeParaConector(conector ConnectorConfig, resume ResumeContext) ResumeContext {
	metadata, err := parseAnyMapJSON(conector.MetadataJSON)
	if err != nil {
		return resume
	}
	metadata = normalizeConnectorMetadata(conector, metadata)
	if conectorPermiteResume(metadata) {
		return resume
	}
	preservarResumenContinuidad := boolMetadata(metadata, "pool_compartido")
	payloadTeniaContexto := resumePayloadTieneContextoReanudable(resume.ResumePayloadJSON)
	resume.ExternalSessionID = ""
	resume.ResumePayloadJSON = conservarSoloPerfilEjecucionResumePayload(resume.ResumePayloadJSON)
	if payloadTeniaContexto {
		resume.ResumePayloadJSON = ""
	}
	if !preservarResumenContinuidad {
		resume.ResumenContinuidad = ""
	}
	return resume
}

func conservarSoloPerfilEjecucionResumePayload(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	payload, err := parseAnyMapJSON(raw)
	if err != nil {
		return ""
	}
	perfilRaw, ok := payload["perfil_ejecucion"].(map[string]any)
	if !ok {
		return ""
	}
	perfil := map[string]any{}
	if value := strings.TrimSpace(stringFromAny(perfilRaw["perfil_tarea"])); value != "" {
		perfil["perfil_tarea"] = value
	}
	if value := strings.TrimSpace(stringFromAny(perfilRaw["modelo"])); value != "" {
		perfil["modelo"] = value
	}
	if value := strings.TrimSpace(stringFromAny(perfilRaw["razonamiento"])); value != "" {
		perfil["razonamiento"] = value
	}
	if len(perfil) == 0 {
		return ""
	}
	data, err := json.Marshal(map[string]any{
		"perfil_ejecucion": perfil,
	})
	if err != nil {
		return ""
	}
	return string(data)
}

func resumeActivaModoLocal(resume ResumeContext) bool {
	return strings.TrimSpace(resume.ExternalSessionID) != "" ||
		strings.TrimSpace(resume.ResumenContinuidad) != ""
}

func resumeActivaModoRemoto(resume ResumeContext) bool {
	return strings.TrimSpace(resume.ExternalSessionID) != ""
}

func conectorPermiteResume(metadata map[string]any) bool {
	if metadata == nil {
		return true
	}
	if hasMetadataKey(metadata, "reanudable") {
		return boolMetadata(metadata, "reanudable")
	}
	return true
}

func isOllamaCLIRequest(req LaunchRequest) bool {
	return isOllamaCLIConnector(req.Conector)
}

func buildRemoteConfigJSON(req LaunchRequest, metadata map[string]any) string {
	cfg := map[string]any{}
	endpoint := stringMetadata(metadata, "endpoint")
	if endpoint == "" {
		endpoint = endpointFromCommand(req.Conector.Comando)
	}
	if endpoint != "" {
		cfg["endpoint"] = endpoint
	}
	if authHeader := stringMetadata(metadata, "auth_header"); authHeader != "" {
		cfg["auth_header"] = authHeader
	}
	if authTokenEnv := stringMetadata(metadata, "auth_token_env"); authTokenEnv != "" {
		cfg["auth_token_env"] = authTokenEnv
	}
	if authMode := stringMetadata(metadata, "auth_mode"); authMode != "" {
		cfg["auth_mode"] = authMode
	}
	if preserve := boolMetadata(metadata, "preserve_external_session_on_stop"); preserve {
		cfg["preserve_external_session_on_stop"] = true
	}
	if requiresHuman := boolMetadata(metadata, "requires_human_reauth"); requiresHuman {
		cfg["requires_human_reauth"] = true
	}
	if timeoutMS := intMetadata(metadata, "timeout_ms"); timeoutMS > 0 {
		cfg["timeout_ms"] = timeoutMS
	}
	launchPathDefault, resumePathDefault, statusPathDefault, pausePathDefault, continuePathDefault, stopPathDefault, inputPathDefault := resolverRutasRemotasPorDefectoConector(req.Conector, metadata)
	cfg["launch_path"] = metadataStringOrDefault(metadata, "launch_path", launchPathDefault)
	cfg["resume_path"] = metadataStringOrDefault(metadata, "resume_path", resumePathDefault)
	if statusPath, ok := metadataString(metadata, "status_path"); ok && strings.TrimSpace(statusPath) != "" {
		cfg["status_path"] = statusPath
	} else if strings.TrimSpace(statusPathDefault) != "" {
		cfg["status_path"] = statusPathDefault
	}
	cfg["pause_path"] = metadataStringOrDefault(metadata, "pause_path", pausePathDefault)
	cfg["continue_path"] = metadataStringOrDefault(metadata, "continue_path", continuePathDefault)
	cfg["stop_path"] = metadataStringOrDefault(metadata, "stop_path", stopPathDefault)
	cfg["input_path"] = metadataStringOrDefault(metadata, "input_path", inputPathDefault)
	cfg["session_id_field"] = metadataStringOrDefault(metadata, "session_id_field", "external_session_id")
	cfg["handle_ref_field"] = metadataStringOrDefault(metadata, "handle_ref_field", "handle_ref")
	if hasMetadataKey(metadata, "can_send_input") {
		cfg["can_send_input"] = boolMetadata(metadata, "can_send_input")
	}
	if hasMetadataKey(metadata, "can_pause") {
		cfg["can_pause"] = boolMetadata(metadata, "can_pause")
	}
	if hasMetadataKey(metadata, "can_stop") {
		cfg["can_stop"] = boolMetadata(metadata, "can_stop")
	}
	if retryCount := intMetadata(metadata, "control_retry_count"); retryCount > 0 {
		cfg["control_retry_count"] = retryCount
	}
	if retryBackoff := intMetadata(metadata, "control_retry_backoff_ms"); retryBackoff > 0 {
		cfg["control_retry_backoff_ms"] = retryBackoff
	}
	if retryStatuses, ok := metadata["control_retry_statuses"].([]any); ok && len(retryStatuses) > 0 {
		vals := make([]int, 0, len(retryStatuses))
		for _, raw := range retryStatuses {
			switch v := raw.(type) {
			case float64:
				vals = append(vals, int(v))
			case int:
				vals = append(vals, v)
			}
		}
		if len(vals) > 0 {
			cfg["control_retry_statuses"] = vals
		}
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(data)
}

func resolverRutasRemotasPorDefectoConector(conector ConnectorConfig, metadata map[string]any) (launchPath, resumePath, statusPath, pausePath, continuePath, stopPath, inputPath string) {
	launchPath = "/launch"
	resumePath = "/resume"
	statusPath = ""
	pausePath = "/pause"
	continuePath = "/continue"
	stopPath = "/stop"
	inputPath = "/input"

	slug := strings.ToLower(strings.TrimSpace(conector.Slug))
	if slug == "ollama_pool_local" || slug == "ollama-pool-local" || boolMetadata(metadata, "pool_compartido") {
		return "/api/runtime/ollama-pool/launch",
			"/resume",
			"/api/runtime/ollama-pool/status",
			"/pause",
			"/continue",
			"/api/runtime/ollama-pool/stop",
			"/api/runtime/ollama-pool/input"
	}
	return launchPath, resumePath, statusPath, pausePath, continuePath, stopPath, inputPath
}

func applyLocalControlHints(plan *LaunchPlan, metadata map[string]any) bool {
	if plan == nil {
		return false
	}
	applied := false
	if hasMetadataKey(metadata, "can_send_input") {
		value := boolMetadata(metadata, "can_send_input")
		plan.CanSendInput = &value
		applied = true
	}
	if mode := NormalizeMailboxDeliveryMode(stringMetadata(metadata, "mailbox_delivery_mode")); mode != "" {
		plan.MailboxDeliveryMode = mode
		applied = true
	}
	if plan.MailboxDeliveryMode == "" {
		plan.MailboxDeliveryMode = defaultMailboxDeliveryMode(plan.CanSendInput)
	}
	return applied
}

func applyResumeDeliveryHints(plan *LaunchPlan, req LaunchRequest) {
	if plan == nil {
		return
	}
	if !isCodexCLIRequest(req) {
		return
	}
	if strings.TrimSpace(req.Resume.ExternalSessionID) == "" {
		return
	}
	if plan.CanSendInput != nil && *plan.CanSendInput {
		return
	}
	switch NormalizeMailboxDeliveryMode(plan.MailboxDeliveryMode) {
	case "", MailboxDeliveryBootstrapOnly, MailboxDeliveryInteractive, MailboxDeliveryCoordinatedRestart:
		plan.MailboxDeliveryMode = MailboxDeliverySessionResume
	}
}

func NormalizeMailboxDeliveryMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case MailboxDeliveryInteractive:
		return MailboxDeliveryInteractive
	case MailboxDeliveryBootstrapOnly:
		return MailboxDeliveryBootstrapOnly
	case MailboxDeliveryCoordinatedRestart:
		return MailboxDeliveryCoordinatedRestart
	case MailboxDeliverySessionResume:
		return MailboxDeliverySessionResume
	default:
		return ""
	}
}

func defaultMailboxDeliveryMode(canSendInput *bool) string {
	if canSendInput != nil && !*canSendInput {
		return MailboxDeliveryBootstrapOnly
	}
	return MailboxDeliveryInteractive
}

func boolMetadata(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	switch v := metadata[key].(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "si", "sí", "on":
			return true
		}
	}
	return false
}

func hasMetadataKey(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	_, ok := metadata[key]
	return ok
}

func metadataString(metadata map[string]any, key string) (string, bool) {
	if metadata == nil {
		return "", false
	}
	raw, ok := metadata[key]
	if !ok {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v), true
	default:
		return "", true
	}
}

func metadataStringOrDefault(metadata map[string]any, key string, fallback string) string {
	if value, ok := metadataString(metadata, key); ok {
		return value
	}
	return fallback
}

func endpointFromCommand(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.EqualFold(raw, "ollama") {
		if value := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_URL")); value != "" {
			return strings.TrimRight(value, "/")
		}
		return "http://127.0.0.1:16543"
	}
	return ""
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func connectorTemplateVars(req LaunchRequest, workingDir string) map[string]string {
	orquestaExecutable, _ := os.Executable()
	orquestaExecutable = strings.TrimSpace(orquestaExecutable)
	orquestaBinDir := ""
	if orquestaExecutable != "" {
		orquestaBinDir = filepath.Dir(orquestaExecutable)
	}
	sessionUUID := generateConnectorSessionUUID()
	return map[string]string{
		"agent":               strings.TrimSpace(req.Agente),
		"role":                strings.TrimSpace(req.Rol),
		"project_slug":        strings.TrimSpace(req.ProyectoSlug),
		"project_path":        strings.TrimSpace(req.ProyectoRuta),
		"working_dir":         strings.TrimSpace(workingDir),
		"model":               strings.TrimSpace(req.Modelo),
		"reasoning":           strings.TrimSpace(req.Razonamiento),
		"task_profile":        strings.TrimSpace(req.PerfilTarea),
		"connector_slug":      strings.TrimSpace(req.Conector.Slug),
		"branch":              strings.TrimSpace(req.Resume.Branch),
		"external_session_id": strings.TrimSpace(req.Resume.ExternalSessionID),
		"orquesta_executable": orquestaExecutable,
		"orquesta_bin_dir":    strings.TrimSpace(orquestaBinDir),
		"host_path":           strings.TrimSpace(os.Getenv("PATH")),
		"home":                strings.TrimSpace(os.Getenv("HOME")),
		"session_uuid":        sessionUUID,
	}
}

func generateConnectorSessionUUID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return ""
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		raw[0:4],
		raw[4:6],
		raw[6:8],
		raw[8:10],
		raw[10:16],
	)
}

func expandConnectorTemplate(raw string, vars map[string]string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(vars) == 0 {
		return raw
	}
	replacements := make([]string, 0, len(vars)*2)
	for key, value := range vars {
		replacements = append(replacements, "{{"+key+"}}", strings.TrimSpace(value))
	}
	return strings.NewReplacer(replacements...).Replace(raw)
}

func expandConnectorArgs(args []string, vars map[string]string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, expandConnectorTemplate(arg, vars))
	}
	return out
}

func expandConnectorEnv(env map[string]string, vars map[string]string) map[string]string {
	if len(env) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		out[key] = expandConnectorTemplate(value, vars)
	}
	return out
}

func RenderCommand(plan *LaunchPlan) string {
	if plan == nil {
		return ""
	}
	partes := []string{}
	if cmd := strings.TrimSpace(plan.Comando); cmd != "" {
		partes = append(partes, shellQuote(cmd))
	}
	for _, arg := range plan.Args {
		partes = append(partes, shellQuote(arg))
	}
	return strings.TrimSpace(strings.Join(partes, " "))
}

func LaunchPromptText(plan *LaunchPlan) string {
	if plan == nil {
		return ""
	}
	partes := make([]string, 0, 2)
	if bootstrap := strings.TrimSpace(plan.BootstrapPrompt); bootstrap != "" {
		partes = append(partes, bootstrap)
	}
	if continuity := strings.TrimSpace(plan.ContinuityPrompt); continuity != "" {
		if len(partes) == 0 || partes[len(partes)-1] != continuity {
			partes = append(partes, continuity)
		}
	}
	return strings.TrimSpace(strings.Join(partes, "\n\n"))
}

func ApplyLaunchPromptMetadata(plan *LaunchPlan, metadataJSON string) error {
	metadata, err := parseAnyMapJSON(metadataJSON)
	if err != nil {
		return err
	}
	applyLaunchPromptHints(plan, metadata)
	return nil
}

func shellQuote(raw string) string {
	if raw == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(raw, "'", `'\''`) + "'"
}

func parseStringSliceJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseStringMapJSON(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	out := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseAnyMapJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	out := make(map[string]any)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func tieneContextoReanudable(r ResumeContext) bool {
	return strings.TrimSpace(r.ExternalSessionID) != "" ||
		resumePayloadTieneContextoReanudable(r.ResumePayloadJSON) ||
		strings.TrimSpace(r.ResumenContinuidad) != ""
}

func resumePayloadTieneContextoReanudable(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	payload, err := parseAnyMapJSON(raw)
	if err != nil {
		return true
	}
	if len(payload) == 0 {
		return false
	}
	for key, value := range payload {
		if strings.EqualFold(strings.TrimSpace(key), "perfil_ejecucion") {
			continue
		}
		if resumePayloadValorNoVacio(value) {
			return true
		}
	}
	return false
}

func resumePayloadValorNoVacio(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func applyLaunchHints(plan *LaunchPlan, metadata map[string]any, req LaunchRequest) bool {
	aplicado := false
	if subcmd := stringMetadata(metadata, "launch_subcommand"); subcmd != "" {
		plan.Args = append([]string{subcmd}, plan.Args...)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "sandbox_flag"); flag != "" {
		if mode := stringMetadata(metadata, "sandbox_mode"); mode != "" && !argsContainOption(plan.Args, flag) {
			plan.Args = append(plan.Args, flag, mode)
			aplicado = true
		}
	}
	if flag := stringMetadata(metadata, "approval_flag"); flag != "" {
		if policy := stringMetadata(metadata, "approval_policy"); policy != "" && !argsContainOption(plan.Args, flag) {
			plan.Args = append(plan.Args, flag, policy)
			aplicado = true
		}
	}
	if flag := stringMetadata(metadata, "cwd_flag"); flag != "" && strings.TrimSpace(plan.WorkingDir) != "" {
		plan.Args = append(plan.Args, flag, plan.WorkingDir)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "branch_flag"); flag != "" && strings.TrimSpace(req.Resume.Branch) != "" {
		plan.Args = append(plan.Args, flag, req.Resume.Branch)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "model_flag"); flag != "" && strings.TrimSpace(plan.Modelo) != "" {
		plan.Args = append(plan.Args, flag, plan.Modelo)
		aplicado = true
	} else if key := stringMetadata(metadata, "model_config_key"); key != "" && strings.TrimSpace(plan.Modelo) != "" {
		plan.Args = appendConnectorConfigOverride(plan.Args, metadata, key, plan.Modelo)
		aplicado = true
	} else if boolMetadata(metadata, "model_positional") && strings.TrimSpace(plan.Modelo) != "" {
		plan.Args = append(plan.Args, plan.Modelo)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "reasoning_flag"); flag != "" && strings.TrimSpace(plan.Razonamiento) != "" {
		plan.Args = append(plan.Args, flag, plan.Razonamiento)
		aplicado = true
	} else if key := stringMetadata(metadata, "reasoning_config_key"); key != "" && strings.TrimSpace(plan.Razonamiento) != "" {
		plan.Args = appendConnectorConfigOverride(plan.Args, metadata, key, plan.Razonamiento)
		aplicado = true
	}
	return aplicado
}

func argsContainOption(args []string, flag string) bool {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return false
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == flag {
			return true
		}
	}
	return false
}

func appendConnectorConfigOverride(args []string, metadata map[string]any, key, value string) []string {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return args
	}
	flag := stringMetadata(metadata, "config_flag")
	if flag == "" {
		flag = "-c"
	}
	return append(args, flag, key+"="+strconv.Quote(value))
}

func applyLaunchPromptHints(plan *LaunchPlan, metadata map[string]any) bool {
	if plan == nil || plan.LaunchPromptEmbedded || plan.NativeResume {
		return false
	}
	prompt := LaunchPromptText(plan)
	if strings.TrimSpace(prompt) == "" {
		return false
	}
	if delayMS := intMetadata(metadata, "launch_prompt_delay_ms"); delayMS > 0 {
		plan.LaunchPromptDelayMS = delayMS
	}
	switch strings.ToLower(strings.TrimSpace(stringMetadata(metadata, "launch_prompt_transport"))) {
	case "post_start", "post-start", "pty", "stdin":
		plan.LaunchPromptMode = "post_start"
		return false
	}
	if maxBytes := intMetadata(metadata, "launch_prompt_max_bytes"); maxBytes > 0 && len([]byte(prompt)) > maxBytes {
		plan.LaunchPromptMode = "post_start"
		return false
	}

	aplicado := false
	if flag := stringMetadata(metadata, "launch_prompt_flag"); flag != "" {
		plan.Args = append(plan.Args, flag, prompt)
		aplicado = true
	}
	if boolMetadata(metadata, "launch_prompt_positional") {
		plan.Args = append(plan.Args, prompt)
		aplicado = true
	}
	if envKey := stringMetadata(metadata, "launch_prompt_env"); envKey != "" {
		if plan.Env == nil {
			plan.Env = map[string]string{}
		}
		plan.Env[envKey] = prompt
		aplicado = true
	}
	if boolMetadata(metadata, "launch_prompt_native") {
		aplicado = true
	}
	if aplicado {
		plan.LaunchPromptEmbedded = true
		if plan.LaunchPromptMode == "" {
			plan.LaunchPromptMode = "embedded"
		}
	} else if plan.LaunchPromptMode == "" {
		plan.LaunchPromptMode = "post_start"
	}
	return aplicado
}

func applyExecutionProfile(plan *LaunchPlan, metadata map[string]any) {
	if plan == nil {
		return
	}
	if plan.Modelo == "" {
		plan.Modelo = stringMetadata(metadata, "default_model")
	}
	if plan.Razonamiento == "" {
		plan.Razonamiento = stringMetadata(metadata, "default_reasoning_effort")
	}
	if plan.PerfilTarea == "" {
		plan.PerfilTarea = stringMetadata(metadata, "default_task_profile")
	}
}

func applyResumeHints(plan *LaunchPlan, metadata map[string]any, resume ResumeContext) bool {
	aplicado := false
	subcmd := stringMetadata(metadata, "resume_subcommand")
	if subcmd != "" {
		plan.Args = append([]string{subcmd}, plan.Args...)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "session_id_flag"); flag != "" && strings.TrimSpace(resume.ExternalSessionID) != "" {
		plan.Args = append(plan.Args, flag, resume.ExternalSessionID)
		aplicado = true
	} else if subcmd != "" && strings.TrimSpace(resume.ExternalSessionID) != "" {
		plan.Args = append(plan.Args, resume.ExternalSessionID)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "resume_payload_flag"); flag != "" && strings.TrimSpace(resume.ResumePayloadJSON) != "" {
		plan.Args = append(plan.Args, flag, resume.ResumePayloadJSON)
		aplicado = true
	}
	if !aplicado {
		return false
	}
	if flag := stringMetadata(metadata, "branch_flag"); flag != "" && strings.TrimSpace(resume.Branch) != "" {
		plan.Args = append(plan.Args, flag, resume.Branch)
	}
	if flag := stringMetadata(metadata, "cwd_flag"); flag != "" && strings.TrimSpace(plan.WorkingDir) != "" {
		plan.Args = append(plan.Args, flag, plan.WorkingDir)
	}
	return aplicado
}

func construirPromptContinuidad(req LaunchRequest) string {
	if isOllamaCLIRequest(req) {
		partes := []string{
			"PROTOCOLO_ORQUESTA_MICRO",
			"NO_INTERPRETAR_COMO_PREGUNTA",
			"SI_NO_HAY_MICROTAREA=ACK-ESPERA",
			"ESPERA_MICROTAREA_CERRADA",
		}
		return strings.Join(partes, " ")
	}
	if isCodexCLIRequest(req) {
		partes := []string{"Retoma el trabajo actual desde Orquesta."}
		pipelineLocalPriorizado := resumePayloadCodexEsPipelineLocalPriorizado(req.Resume.ResumePayloadJSON)
		if strings.TrimSpace(req.ProyectoSlug) != "" && !pipelineLocalPriorizado {
			partes = append(partes, "Proyecto: "+strings.TrimSpace(req.ProyectoSlug)+".")
		}
		if strings.TrimSpace(req.Resume.Branch) != "" && !pipelineLocalPriorizado {
			partes = append(partes, "Rama: "+strings.TrimSpace(req.Resume.Branch)+".")
		}
		if resumen := resumirContinuidadCodex(req.Resume.ResumenContinuidad); resumen != "" {
			partes = append(partes, "Continuidad: "+resumen+".")
		}
		if resumenPayload := resumirResumePayloadCodex(req.Resume.ResumePayloadJSON); resumenPayload != "" {
			partes = append(partes, resumenPayload+".")
		}
		partes = append(partes, "Activa $caveman. Salida minima: hecho, tests, riesgos/bloqueos; nada mas.")
		if !pipelineLocalPriorizado {
			partes = append(partes, "Ignora cualquier conversación vieja que no coincida con la tarea activa o el mailbox actual.")
			partes = append(partes, "No abras frentes nuevos ni reescribas código fuera del alcance inmediato.")
			partes = append(partes, "No hagas broad scans ni releas contexto global si mailbox/tarea ya acotan el slice.")
			partes = append(partes, "Empieza por la tarea asignada y consulta Orquesta antes de desviarte.")
		}
		return strings.Join(partes, " ")
	}

	partes := []string{
		fmt.Sprintf("Retoma el trabajo del agente %s", req.Agente),
	}
	if strings.TrimSpace(req.ProyectoSlug) != "" {
		partes = append(partes, "en el proyecto "+req.ProyectoSlug)
	}
	if strings.TrimSpace(req.Resume.Branch) != "" {
		partes = append(partes, "sobre la rama "+req.Resume.Branch)
	}

	var detalle []string
	if strings.TrimSpace(req.Resume.ResumenContinuidad) != "" {
		detalle = append(detalle, "Resumen previo: "+req.Resume.ResumenContinuidad)
	}
	if strings.TrimSpace(req.Resume.ExternalSessionID) != "" {
		detalle = append(detalle, "External session id: "+req.Resume.ExternalSessionID)
	}
	if resumenPayload := resumirResumePayload(req.Resume.ResumePayloadJSON); resumenPayload != "" {
		detalle = append(detalle, resumenPayload)
	}

	base := strings.Join(partes, ". ")
	if len(detalle) == 0 {
		return base + "."
	}
	return base + ". " + strings.Join(detalle, ". ")
}

func resumirContinuidadCodex(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	clauses := splitContinuityClausesCodex(raw)
	for _, clause := range clauses {
		lower := strings.ToLower(clause)
		if strings.Contains(lower, "handoff") {
			if len(clause) <= 96 && len(strings.Fields(clause)) <= 12 {
				return clause
			}
			return ""
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	first := clauses[0]
	if len(first) > 72 || len(strings.Fields(first)) > 8 {
		return ""
	}
	return first
}

func splitContinuityClausesCodex(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "\n", ". ")
	parts := strings.Split(raw, ". ")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(strings.TrimSuffix(item, "."))
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func resumirResumePayloadCodex(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return ""
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return ""
	}
	partes := make([]string, 0, 8)
	pipelineLocalPriorizado := false
	if order, ok := obj["runtime_order"].(map[string]any); ok {
		label := "runtime_order"
		if tipo := strings.TrimSpace(stringFromAny(order["tipo"])); tipo != "" {
			label += "=" + tipo
		}
		partes = append(partes, label)
		delete(obj, "runtime_order")
	}
	if checkpoint, ok := obj["checkpoint"].(map[string]any); ok {
		label := "checkpoint"
		if id := int64FromAny(checkpoint["id"]); id > 0 {
			label += fmt.Sprintf("#%d", id)
		}
		if kind := strings.TrimSpace(stringFromAny(checkpoint["kind"])); kind != "" {
			label += "(" + kind + ")"
		}
		partes = append(partes, label)
		delete(obj, "checkpoint")
	}
	if mailbox, ok := obj["mailbox"].([]any); ok {
		label := fmt.Sprintf("mailbox=%d", len(mailbox))
		if mailboxPayloadEsPipelineLocalCodex(mailbox) {
			pipelineLocalPriorizado = true
		}
		if resumen := resumirMailboxPayloadCodex(mailbox); resumen != "" {
			label += " " + resumen
		}
		partes = append(partes, label)
		delete(obj, "mailbox")
	}
	if projectContext, ok := obj["project_context"]; ok {
		if !pipelineLocalPriorizado {
			partes = append(partes, resumirProjectContextCodex(projectContext))
		}
		delete(obj, "project_context")
	}
	if governanceCatalog, ok := obj["governance_catalog"]; ok {
		if !pipelineLocalPriorizado {
			partes = append(partes, resumirGovernanceCatalogCodex(governanceCatalog))
		}
		delete(obj, "governance_catalog")
	}
	if _, ok := obj["adopted_context"]; ok {
		if !pipelineLocalPriorizado {
			partes = append(partes, "adopted_context")
		}
		delete(obj, "adopted_context")
	}
	ignored := map[string]struct{}{
		"bootstrap_prompt":         {},
		"bootstrap_prompt_compact": {},
	}
	resto := make([]string, 0, len(obj))
	for key := range obj {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, skip := ignored[key]; skip {
			continue
		}
		resto = append(resto, key)
	}
	sort.Strings(resto)
	partes = append(partes, resto...)
	if len(partes) == 0 {
		return ""
	}
	return strings.Join(partes, ", ")
}

func resumePayloadCodexEsPipelineLocalPriorizado(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return false
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return false
	}
	mailbox, ok := obj["mailbox"].([]any)
	if !ok {
		return false
	}
	return mailboxPayloadEsPipelineLocalCodex(mailbox)
}

func mailboxPayloadEsPipelineLocalCodex(items []any) bool {
	if len(items) == 0 {
		return false
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return false
	}
	kind := strings.TrimSpace(stringFromAny(first["kind"]))
	payload, _ := first["payload"].(map[string]any)
	if payload == nil {
		return false
	}
	source := strings.TrimSpace(stringFromAny(payload["source"]))
	return strings.EqualFold(kind, "pipeline_local") || strings.EqualFold(source, "pipeline_local")
}

func resumirMailboxPayloadCodex(items []any) string {
	if len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	kind := strings.TrimSpace(stringFromAny(first["kind"]))
	payload, _ := first["payload"].(map[string]any)
	if payload == nil {
		return ""
	}
	source := strings.TrimSpace(stringFromAny(payload["source"]))
	if !strings.EqualFold(kind, "pipeline_local") && !strings.EqualFold(source, "pipeline_local") {
		return resumirMailboxPayload(items)
	}
	return resumirMailboxPipelineLocalCodex(kind, payload)
}

func resumirMailboxPipelineLocalCodex(kind string, payload map[string]any) string {
	partes := make([]string, 0, 8)
	if kind = strings.TrimSpace(kind); kind != "" {
		partes = append(partes, kind)
	}
	if accion := strings.TrimSpace(stringFromAny(payload["accion"])); accion != "" {
		partes = append(partes, "accion="+accion)
	}
	tareaID := int64FromAny(payload["tarea_id"])
	if tareaID <= 0 {
		tareaID = int64FromAny(payload["tarea_objetivo_id"])
	}
	if tareaID > 0 {
		partes = append(partes, fmt.Sprintf("tarea#%d", tareaID))
	}
	if carril := strings.TrimSpace(stringFromAny(payload["carril"])); carril != "" {
		partes = append(partes, "carril="+carril)
	}
	writeSetSummary := resumirWriteSetMailboxCodex(payload["write_set"], 3)
	if writeSetSummary != "" {
		partes = append(partes, "write_set="+writeSetSummary)
	}
	texto := strings.TrimSpace(stringFromAny(payload["instruction"]))
	if texto == "" {
		texto = strings.TrimSpace(stringFromAny(payload["texto"]))
	}
	if resumen := resumirInstructionMailboxCodex(texto, writeSetSummary == ""); resumen != "" {
		partes = append(partes, resumen)
	}
	if len(partes) == 0 {
		return ""
	}
	return "[" + strings.Join(partes, " | ") + "]"
}

func resumirWriteSetMailboxCodex(raw any, maxItems int) string {
	items := stringSliceFromAny(raw)
	if len(items) == 0 {
		return ""
	}
	if maxItems <= 0 || len(items) <= maxItems {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:maxItems], ", ") + "…"
}

func resumirInstructionMailboxCodex(raw string, includeWriteSet bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	clauses := splitOperationalClausesCodex(raw)
	if len(clauses) == 0 {
		return truncarResumenMailbox(raw, 120)
	}
	matchers := []string{
		"simbolos foco:",
	}
	if includeWriteSet {
		matchers = append(matchers, "write-set", "write_set", "write set:", "write_set:")
	}
	matchers = append(matchers,
		"tests minimos",
		"tests mínimos",
		"tests:",
		"tarea:",
		"fase:",
		"accion:",
		"carril:",
		"entrega:",
	)
	selected := make([]string, 0, len(matchers))
	seen := map[string]struct{}{}
	appendIfFits := func(clause string) bool {
		clause = compactarOperationalClauseCodex(clause)
		if clause == "" {
			return false
		}
		if _, ok := seen[clause]; ok {
			return true
		}
		candidate := clause
		if len(selected) > 0 {
			candidate = strings.Join(append(append([]string(nil), selected...), clause), ". ")
		}
		const maxRunes = 260
		if len([]rune(candidate)) > maxRunes {
			return false
		}
		selected = append(selected, clause)
		seen[clause] = struct{}{}
		return true
	}
	for _, matcher := range matchers {
		for _, clause := range clauses {
			lower := strings.ToLower(strings.TrimSpace(clause))
			if !strings.Contains(lower, matcher) {
				continue
			}
			if !appendIfFits(clause) {
				break
			}
		}
	}
	if len(selected) == 0 {
		return truncarResumenMailbox(raw, 120)
	}
	return strings.Join(selected, ". ")
}

func splitOperationalClausesCodex(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "\n", ". ")
	parts := strings.Split(raw, ". ")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.TrimRight(part, ".;"))
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func compactarOperationalClauseCodex(clause string) string {
	clause = strings.TrimSpace(strings.TrimRight(clause, ".;"))
	if clause == "" {
		return ""
	}
	lower := strings.ToLower(clause)
	if strings.HasPrefix(lower, "tests minimos") || strings.HasPrefix(lower, "tests mínimos") {
		clause = strings.Replace(clause, "Tests minimos del slice:", "Tests minimos:", 1)
		clause = strings.Replace(clause, "Tests mínimos del slice:", "Tests mínimos:", 1)
		clause = strings.ReplaceAll(clause, " -count=1", "")
		clause = strings.ReplaceAll(clause, " y go test ", " ; go test ")
	}
	if len([]rune(clause)) > 120 {
		clause = truncarResumenMailbox(clause, 120)
	}
	return strings.Join(strings.Fields(clause), " ")
}

func resumirProjectContextCodex(raw any) string {
	ctx, ok := raw.(map[string]any)
	if !ok {
		return "project_context"
	}
	resumen := make([]string, 0, 3)
	if worktree, ok := ctx["worktree_activa"].(map[string]any); ok {
		if branch := strings.TrimSpace(stringFromAny(worktree["branch"])); branch != "" {
			resumen = append(resumen, "Worktree activa en "+branch)
		}
	}
	if tareas, ok := ctx["tareas_activas"].([]any); ok && len(tareas) > 0 {
		resumen = append(resumen, fmt.Sprintf("%d tarea(s) activas del agente", len(tareas)))
	}
	if propuestas, ok := ctx["propuestas_abiertas"].([]any); ok && len(propuestas) > 0 {
		resumen = append(resumen, fmt.Sprintf("%d propuesta(s) abiertas", len(propuestas)))
	}
	if len(resumen) == 0 {
		return "project_context"
	}
	return "project_context: " + strings.Join(resumen, ". ")
}

func resumirGovernanceCatalogCodex(raw any) string {
	ctx, ok := raw.(map[string]any)
	if !ok {
		return "governance_catalog"
	}
	resumen := "Catálogo efectivo"
	if hash := strings.TrimSpace(stringFromAny(ctx["hash"])); hash != "" {
		resumen += " " + hash
	}
	counts := make([]string, 0, 3)
	if reglas := int(int64FromAny(ctx["reglas"])); reglas > 0 {
		counts = append(counts, fmt.Sprintf("%d reglas", reglas))
	}
	if skills := int(int64FromAny(ctx["skills"])); skills > 0 {
		counts = append(counts, fmt.Sprintf("%d skills", skills))
	}
	if workflows := int(int64FromAny(ctx["workflows"])); workflows > 0 {
		counts = append(counts, fmt.Sprintf("%d workflows", workflows))
	}
	if len(counts) > 0 {
		resumen += " (" + strings.Join(counts, ", ") + ")"
	}
	return "governance_catalog: " + resumen
}

func resumirResumePayload(raw string) string {

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "Resume payload disponible"
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return "Resume payload disponible"
	}
	partes := make([]string, 0, len(obj))
	if order, ok := obj["runtime_order"].(map[string]any); ok {
		label := "runtime_order"
		if tipo := strings.TrimSpace(stringFromAny(order["tipo"])); tipo != "" {
			label += "=" + tipo
		}
		partes = append(partes, label)
		delete(obj, "runtime_order")
	}
	if checkpoint, ok := obj["checkpoint"].(map[string]any); ok {
		label := "checkpoint"
		if id := int64FromAny(checkpoint["id"]); id > 0 {
			label += fmt.Sprintf("#%d", id)
		}
		if kind := strings.TrimSpace(stringFromAny(checkpoint["kind"])); kind != "" {
			label += "(" + kind + ")"
		}
		partes = append(partes, label)
		delete(obj, "checkpoint")
	}
	if mailbox, ok := obj["mailbox"].([]any); ok {
		label := fmt.Sprintf("mailbox=%d", len(mailbox))
		if resumen := resumirMailboxPayload(mailbox); resumen != "" {
			label += " " + resumen
		}
		partes = append(partes, label)
		delete(obj, "mailbox")
	}
	for _, key := range []string{"project_context", "governance_catalog", "adopted_context"} {
		if _, ok := obj[key]; ok {
			partes = append(partes, key)
			delete(obj, key)
		}
	}
	resto := make([]string, 0, len(obj))
	for key := range obj {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		resto = append(resto, key)
	}
	sort.Strings(resto)
	partes = append(partes, resto...)
	if len(partes) == 0 {
		return "Resume payload disponible"
	}
	return "Resume payload: " + strings.Join(partes, ", ")
}

func resumirMailboxPayload(items []any) string {
	if len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	kind := strings.TrimSpace(stringFromAny(first["kind"]))
	payload, _ := first["payload"].(map[string]any)
	if payload == nil {
		return ""
	}
	partes := make([]string, 0, 4)
	if kind != "" {
		partes = append(partes, kind)
	}
	if accion := strings.TrimSpace(stringFromAny(payload["accion"])); accion != "" {
		partes = append(partes, "accion="+accion)
	}
	if tareaID := int64FromAny(payload["tarea_id"]); tareaID > 0 {
		partes = append(partes, fmt.Sprintf("tarea#%d", tareaID))
	}
	if micro, ok := payload["microprogramacion"].(map[string]any); ok {
		if especID := int64FromAny(micro["especificacion_id"]); especID > 0 {
			partes = append(partes, fmt.Sprintf("micro#%d", especID))
		}
	}
	texto := strings.TrimSpace(stringFromAny(payload["instruction"]))
	if texto == "" {
		texto = strings.TrimSpace(stringFromAny(payload["texto"]))
	}
	if texto != "" {
		partes = append(partes, truncarResumenMailbox(texto, 72))
	}
	if len(partes) == 0 {
		return ""
	}
	return "[" + strings.Join(partes, " | ") + "]"
}

func stringSliceFromAny(raw any) []string {
	switch items := raw.(type) {
	case []string:
		out := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			out = append(out, item)
		}
		return out
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			texto := strings.TrimSpace(stringFromAny(item))
			if texto == "" {
				continue
			}
			out = append(out, texto)
		}
		return out
	default:
		return nil
	}
}

func truncarResumenMailbox(raw string, max int) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if raw == "" || max <= 0 {
		return ""
	}
	runes := []rune(raw)
	if len(runes) <= max {
		return raw
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func int64FromAny(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		out, _ := n.Int64()
		return out
	default:
		return 0
	}
}

func stringMetadata(metadata map[string]any, key string) string {
	v, ok := metadata[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func intMetadata(metadata map[string]any, key string) int {
	v, ok := metadata[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}
