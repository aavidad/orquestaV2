package orquestaruntime

import (
	"fmt"
	"path/filepath"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const ProcessRuntimeConnectorVersionV0 = "process_runtime_connector.v0"

type ProcessRuntimeStatusV0 string

const (
	ProcessRuntimeRunningV0 ProcessRuntimeStatusV0 = "running"
	ProcessRuntimeStoppedV0 ProcessRuntimeStatusV0 = "stopped"
)

type ProcessRuntimeErrorCodeV0 string

const (
	ProcessRuntimeConfigInvalidaV0 ProcessRuntimeErrorCodeV0 = "process_runtime_config_invalida"
	ProcessRuntimeShellProhibidaV0 ProcessRuntimeErrorCodeV0 = "process_runtime_shell_prohibida"
	ProcessRuntimeEnvProhibidoV0   ProcessRuntimeErrorCodeV0 = "process_runtime_env_prohibido"
	ProcessRuntimeRefInvalidaV0    ProcessRuntimeErrorCodeV0 = "process_runtime_ref_invalida"
	ProcessRuntimeNoEncontradoV0   ProcessRuntimeErrorCodeV0 = "process_runtime_no_encontrado"
	ProcessRuntimeLaunchFallidoV0  ProcessRuntimeErrorCodeV0 = "process_runtime_launch_fallido"
	ProcessRuntimeStopFallidoV0    ProcessRuntimeErrorCodeV0 = "process_runtime_stop_fallido"
	ProcessRuntimeContextDoneV0    ProcessRuntimeErrorCodeV0 = "process_runtime_context_done"
)

type ProcessRuntimeLaunchRequestV0 struct {
	CommandPath string   `json:"-"`
	Args        []string `json:"-"`
	Env         []string `json:"-"`
	WorkingDir  string   `json:"-"`
}

type ProcessRuntimeSnapshotV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	ProcessRef    string                 `json:"process_ref"`
	SessionRef    string                 `json:"session_ref,omitempty"`
	LaunchRef     string                 `json:"launch_ref,omitempty"`
	StopRef       string                 `json:"stop_ref,omitempty"`
	Status        ProcessRuntimeStatusV0 `json:"status"`
}

type ProcessRuntimeErrorV0 struct {
	Code       ProcessRuntimeErrorCodeV0 `json:"code"`
	MessageKey string                    `json:"message_key"`
	Field      string                    `json:"field,omitempty"`
	Retryable  bool                      `json:"retryable"`
	Evidence   []string                  `json:"evidence,omitempty"`
}

func (e ProcessRuntimeErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

func ValidateProcessRuntimeLaunchRequestV0(req ProcessRuntimeLaunchRequestV0) error {
	return validateProcessRuntimeLaunchRequestV0(req)
}

func validateProcessRuntimeLaunchRequestV0(req ProcessRuntimeLaunchRequestV0) error {
	if strings.TrimSpace(req.CommandPath) == "" {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "command_path")
	}
	if !filepath.IsAbs(req.CommandPath) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "command_path")
	}
	if processRuntimeOperationalPathUnsafeV0(req.CommandPath) || looksLikeSecret(req.CommandPath) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "command_path")
	}
	if processRuntimeCommandIsShellV0(req.CommandPath) {
		return processRuntimeErrorV0(ProcessRuntimeShellProhibidaV0, "command_path")
	}
	if strings.TrimSpace(req.WorkingDir) == "" {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "working_dir")
	}
	if !filepath.IsAbs(req.WorkingDir) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "working_dir")
	}
	if processRuntimeOperationalPathUnsafeV0(req.WorkingDir) {
		return processRuntimeErrorV0(ProcessRuntimeConfigInvalidaV0, "working_dir")
	}
	if req.Env == nil {
		return processRuntimeErrorV0(ProcessRuntimeEnvProhibidoV0, "env")
	}
	for i, arg := range req.Args {
		if processRuntimeUnsafeValueV0(arg) {
			return processRuntimeErrorV0(
				ProcessRuntimeConfigInvalidaV0,
				fmt.Sprintf("args[%d]", i),
			)
		}
	}
	for i, item := range req.Env {
		if !processRuntimeEnvEntryAllowedV0(item) {
			return processRuntimeErrorV0(
				ProcessRuntimeEnvProhibidoV0,
				fmt.Sprintf("env[%d]", i),
			)
		}
	}
	return nil
}

func validateProcessRuntimeRefV0(processRef string) error {
	if processRef == "" || !opaqueRefPatternV0.MatchString(processRef) {
		return processRuntimeErrorV0(ProcessRuntimeRefInvalidaV0, "process_ref")
	}
	return nil
}

func processRuntimeErrorV0(code ProcessRuntimeErrorCodeV0, field string) ProcessRuntimeErrorV0 {
	return ProcessRuntimeErrorV0{
		Code:       code,
		MessageKey: "orquesta.runtime.process." + string(code),
		Field:      field,
		Retryable:  code == ProcessRuntimeLaunchFallidoV0 || code == ProcessRuntimeStopFallidoV0,
	}
}

func processRuntimeCommandIsShellV0(commandPath string) bool {
	base := strings.ToLower(filepath.Base(commandPath))
	switch base {
	case "sh", "bash", "dash", "zsh", "fish",
		"cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return false
	}
}

func processRuntimeEnvEntryAllowedV0(item string) bool {
	key, value, ok := strings.Cut(item, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(key)) == "PATH" {
		return processRuntimePathEnvValueAllowedV0(value)
	}
	if processRuntimeEnvKeyForbiddenV0(key) {
		return false
	}
	if strings.ContainsAny(value, `/\`) {
		return false
	}
	return !processRuntimeUnsafeValueV0(value)
}

func processRuntimeEnvKeyForbiddenV0(key string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	upper := strings.ToUpper(strings.TrimSpace(key))
	for _, marker := range []string{
		"HOME",
		"PATH",
		"OAUTH",
		"TOKEN",
		"SECRET",
		"API_KEY",
		"CREDENTIAL",
		"PASSWORD",
	} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func processRuntimePathEnvValueAllowedV0(value string) bool {
	if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
		return false
	}
	for _, dir := range filepath.SplitList(value) {
		if strings.TrimSpace(dir) == "" {
			return false
		}
		if !filepath.IsAbs(dir) {
			return false
		}
		if looksLikeSecret(dir) || processRuntimeContainsCredentialMarkerV0(dir) {
			return false
		}
	}
	return true
}

func processRuntimeUnsafeValueV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	return looksLikeSecret(value) ||
		processRuntimeContainsForbiddenMarkerV0(value) ||
		looksLikeConcreteProviderValueV0(value) ||
		looksLikeConcreteModelValueV0(value)
}

func processRuntimeContainsForbiddenMarkerV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	return processRuntimeContainsHomeMarkerV0(value) ||
		processRuntimeContainsCredentialMarkerV0(value)
}

func processRuntimeOperationalPathUnsafeV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	return processRuntimeContainsExplicitHomeMarkerV0(value) ||
		processRuntimeContainsCredentialMarkerV0(value)
}

func processRuntimeContainsHomeMarkerV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return processRuntimeContainsExplicitHomeMarkerV0(low) ||
		strings.Contains(low, "/home/") ||
		strings.Contains(low, `\users\`) ||
		strings.HasPrefix(low, "~")
}

func processRuntimeContainsExplicitHomeMarkerV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "$home") ||
		strings.Contains(low, "${home}") ||
		strings.Contains(low, "%userprofile%") ||
		strings.Contains(low, "%homepath%") ||
		strings.HasPrefix(low, "~")
}

func processRuntimeContainsCredentialMarkerV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "oauth") ||
		strings.Contains(low, "token") ||
		strings.Contains(low, "secret") ||
		strings.Contains(low, "api_key") ||
		strings.Contains(low, "credential")
}
