/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package runtimeagente

import (
	"encoding/json"
	"fmt"
	"os"
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
}

type LaunchPlan struct {
	Driver               string            `json:"driver"`
	Transporte           string            `json:"transporte"`
	Modo                 string            `json:"modo"`
	Comando              string            `json:"comando"`
	Args                 []string          `json:"args"`
	Env                  map[string]string `json:"env"`
	WorkingDir           string            `json:"working_dir"`
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

	workingDir := strings.TrimSpace(req.Resume.CWD)
	if workingDir == "" {
		workingDir = strings.TrimSpace(req.ProyectoRuta)
	}
	vars := connectorTemplateVars(req, workingDir)
	plan := &LaunchPlan{
		Driver:       d.transport,
		Transporte:   d.transport,
		Modo:         "launch",
		Comando:      expandConnectorTemplate(strings.TrimSpace(req.Conector.Comando), vars),
		Args:         expandConnectorArgs(args, vars),
		Env:          expandConnectorEnv(env, vars),
		WorkingDir:   workingDir,
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		PerfilTarea:  strings.TrimSpace(req.PerfilTarea),
	}
	applyExecutionProfile(plan, metadata)
	if applyLocalControlHints(plan, metadata) {
		plan.Notas = append(plan.Notas, "El conector aporta capacidades locales de control desde metadata_json.")
	}

	if applyLaunchHints(plan, metadata, req) {
		plan.Notas = append(plan.Notas, "El conector aporta hints de arranque desde metadata_json.")
	}

	if tieneContextoReanudable(req.Resume) {
		plan.Modo = "resume"
		plan.NativeResume = applyResumeHints(plan, metadata, req.Resume)
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
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		PerfilTarea:  strings.TrimSpace(req.PerfilTarea),
		Notas: []string{
			"El transporte remoto requiere un adaptador externo que consuma este plan.",
		},
	}
	applyExecutionProfile(plan, metadata)
	plan.RemoteConfigJSON = buildRemoteConfigJSON(req, metadata)

	if endpoint := stringMetadata(metadata, "endpoint"); endpoint != "" {
		plan.Notas = append(plan.Notas, "Endpoint: "+endpoint)
	}
	if tieneContextoReanudable(req.Resume) {
		plan.Modo = "resume"
		plan.ContinuityPrompt = construirPromptContinuidad(req)
	}
	return plan, nil
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
	if !needsCopy {
		return metadata
	}
	out := make(map[string]any, len(metadata)+2)
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
	if stringMetadata(out, "mailbox_delivery_mode") == "" {
		out["mailbox_delivery_mode"] = MailboxDeliveryBootstrapOnly
	}
	return out
}

func isCodexCLIConnector(conector ConnectorConfig) bool {
	if strings.EqualFold(strings.TrimSpace(conector.Slug), "codex-cli") {
		return true
	}
	comando := filepath.Base(strings.TrimSpace(conector.Comando))
	return strings.EqualFold(comando, "codex")
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
	cfg["launch_path"] = metadataStringOrDefault(metadata, "launch_path", "/launch")
	cfg["resume_path"] = metadataStringOrDefault(metadata, "resume_path", "/resume")
	if statusPath, ok := metadataString(metadata, "status_path"); ok && strings.TrimSpace(statusPath) != "" {
		cfg["status_path"] = statusPath
	}
	cfg["pause_path"] = metadataStringOrDefault(metadata, "pause_path", "/pause")
	cfg["continue_path"] = metadataStringOrDefault(metadata, "continue_path", "/continue")
	cfg["stop_path"] = metadataStringOrDefault(metadata, "stop_path", "/stop")
	cfg["input_path"] = metadataStringOrDefault(metadata, "input_path", "/input")
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
	}
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
		strings.TrimSpace(r.ResumePayloadJSON) != "" ||
		strings.TrimSpace(r.ResumenContinuidad) != ""
}

func applyLaunchHints(plan *LaunchPlan, metadata map[string]any, req LaunchRequest) bool {
	aplicado := false
	if subcmd := stringMetadata(metadata, "launch_subcommand"); subcmd != "" {
		plan.Args = append([]string{subcmd}, plan.Args...)
		aplicado = true
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
		partes = append(partes, fmt.Sprintf("mailbox=%d", len(mailbox)))
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
