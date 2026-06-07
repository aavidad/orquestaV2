package orquestaruntimeclaude

import (
	"fmt"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	ClaudeConnectorProfileSchemaVersionV0 = "claude_connector_profile.v0"

	ClaudeAgentPacketFileNameV0           = "agent_packet.json"
	ClaudeAgentPromptFileNameV0           = "agent_prompt.txt"
	ClaudeAgentAckFileNameV0              = "agent_ack.json"
	ClaudeDirectorDecisionsFileNameV0     = "director_decisions.json"
	ClaudeShutdownRequestFileNameV0       = "orquesta_shutdown_request.json"
	ClaudeShutdownCheckpointAckFileNameV0 = "agent_shutdown_checkpoint_ack.json"
	ClaudeStdoutFileNameV0                = "claude_stdout.log"
	ClaudeStderrFileNameV0                = "claude_stderr.log"
	ClaudeWrapperFileNameV0               = "orquesta_claude_exec_v0.sh"
)

type ClaudeConnectorIssueCodeV0 string

const (
	ClaudeConnectorProfileInvalidV0 ClaudeConnectorIssueCodeV0 = "claude_connector_profile_invalido"
	ClaudeConnectorOptInRequiredV0  ClaudeConnectorIssueCodeV0 = "claude_connector_opt_in_requerido"
	ClaudeConnectorPathInvalidV0    ClaudeConnectorIssueCodeV0 = "claude_connector_path_invalido"
	ClaudeConnectorValueInvalidV0   ClaudeConnectorIssueCodeV0 = "claude_connector_value_invalido"
	ClaudeConnectorFilesystemV0     ClaudeConnectorIssueCodeV0 = "claude_connector_filesystem_error"
)

const (
	ClaudeRuntimeWorkDirInsideProjectV0 = "inside_project_control_dir"
	ClaudeRuntimeWorkDirExternalRootV0  = "external_control_root"
)

type ClaudeConnectorProfileV0 struct {
	SchemaVersion           string   `json:"schema_version"`
	OptIn                   bool     `json:"opt_in"`
	CommandPath             string   `json:"command_path"`
	ProjectWorkDir          string   `json:"project_work_dir"`
	RuntimeWorkDir          string   `json:"runtime_work_dir"`
	RuntimeWorkDirPlacement string   `json:"runtime_work_dir_placement,omitempty"`
	HomeDir                 string   `json:"home_dir,omitempty"`
	PathEnv                 string   `json:"path_env,omitempty"`
	Model                   string   `json:"model,omitempty"`
	PermissionMode          string   `json:"permission_mode,omitempty"`
	OutputFormat            string   `json:"output_format,omitempty"`
	Effort                  string   `json:"effort,omitempty"`
	ExtraArgs               []string `json:"extra_args,omitempty"`
	PromptHints             []string `json:"prompt_hints,omitempty"`
}

func ValidateClaudeConnectorProfileV0(
	profile ClaudeConnectorProfileV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	v := claudeProfileValidatorV0{}
	if profile.SchemaVersion != ClaudeConnectorProfileSchemaVersionV0 {
		v.add(ClaudeConnectorProfileInvalidV0, "schema_version")
	}
	if !profile.OptIn {
		v.add(ClaudeConnectorOptInRequiredV0, "opt_in")
	}
	v.requireAbsPath("command_path", profile.CommandPath)
	v.requireAbsPath("project_work_dir", profile.ProjectWorkDir)
	v.requireAbsPath("runtime_work_dir", profile.RuntimeWorkDir)
	v.validateRuntimeWorkDirPlacementV0(profile)
	v.validateRuntimeWorkDirIsolationV0(profile)
	v.optionalAbsPath("home_dir", profile.HomeDir)
	v.optionalSafeValue("path_env", profile.PathEnv)
	v.optionalSafeValue("model", profile.Model)
	v.optionalSafeValue("permission_mode", profile.PermissionMode)
	v.optionalSafeValue("output_format", profile.OutputFormat)
	v.optionalSafeValue("effort", profile.Effort)
	for i, value := range profile.ExtraArgs {
		field := fmt.Sprintf("extra_args[%d]", i)
		v.optionalSafeValue(field, value)
		v.rejectOwnedOrUnsafeExtraArgV0(field, value)
	}
	for i, value := range profile.PromptHints {
		v.optionalSafeValue(fmt.Sprintf("prompt_hints[%d]", i), value)
	}
	return v.issues
}

func InferClaudeRuntimeWorkDirPlacementV0(projectWorkDir string, runtimeWorkDir string) string {
	project := filepath.Clean(strings.TrimSpace(projectWorkDir))
	runtime := filepath.Clean(strings.TrimSpace(runtimeWorkDir))
	if project == "." || runtime == "." {
		return ""
	}
	if runtime == project || claudePathInsideV0(runtime, project) {
		return ClaudeRuntimeWorkDirInsideProjectV0
	}
	return ClaudeRuntimeWorkDirExternalRootV0
}

type claudeProfileValidatorV0 struct {
	issues []orquestaruntime.ExternalAgentConnectorErrorV0
}

func (v *claudeProfileValidatorV0) requireAbsPath(field, value string) {
	if strings.TrimSpace(value) == "" || !filepath.IsAbs(value) || claudeHasControlCharsV0(value) {
		v.add(ClaudeConnectorPathInvalidV0, field)
	}
}

func (v *claudeProfileValidatorV0) optionalAbsPath(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	v.requireAbsPath(field, value)
}

func (v *claudeProfileValidatorV0) optionalSafeValue(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if claudeHasControlCharsV0(value) || strings.ContainsAny(value, "\x00\r\n") {
		v.add(ClaudeConnectorValueInvalidV0, field)
	}
}

func (v *claudeProfileValidatorV0) validateRuntimeWorkDirPlacementV0(
	profile ClaudeConnectorProfileV0,
) {
	declared := strings.TrimSpace(profile.RuntimeWorkDirPlacement)
	if declared == "" {
		return
	}
	expected := InferClaudeRuntimeWorkDirPlacementV0(
		profile.ProjectWorkDir,
		profile.RuntimeWorkDir,
	)
	if expected != "" && declared != expected {
		v.add(ClaudeConnectorValueInvalidV0, "runtime_work_dir_placement")
	}
	v.optionalSafeValue("runtime_work_dir_placement", declared)
}

func (v *claudeProfileValidatorV0) validateRuntimeWorkDirIsolationV0(
	profile ClaudeConnectorProfileV0,
) {
	if strings.TrimSpace(profile.ProjectWorkDir) == "" ||
		strings.TrimSpace(profile.RuntimeWorkDir) == "" {
		return
	}
	if filepath.Clean(profile.RuntimeWorkDir) == filepath.Clean(profile.ProjectWorkDir) ||
		claudePathInsideV0(profile.ProjectWorkDir, profile.RuntimeWorkDir) {
		v.add(ClaudeConnectorPathInvalidV0, "runtime_work_dir")
	}
}

func (v *claudeProfileValidatorV0) rejectOwnedOrUnsafeExtraArgV0(field, value string) {
	trimmed := strings.TrimSpace(value)
	switch {
	case trimmed == "-p",
		trimmed == "--print",
		trimmed == "--model",
		strings.HasPrefix(trimmed, "--model="),
		trimmed == "--permission-mode",
		strings.HasPrefix(trimmed, "--permission-mode="),
		trimmed == "--output-format",
		strings.HasPrefix(trimmed, "--output-format="),
		trimmed == "--effort",
		strings.HasPrefix(trimmed, "--effort="),
		trimmed == "--add-dir",
		strings.HasPrefix(trimmed, "--add-dir="),
		trimmed == "--tools",
		strings.HasPrefix(trimmed, "--tools="),
		trimmed == "--allowedTools",
		strings.HasPrefix(trimmed, "--allowedTools="),
		trimmed == "--allowed-tools",
		strings.HasPrefix(trimmed, "--allowed-tools="):
		v.add(ClaudeConnectorValueInvalidV0, field)
	}
}

func (v *claudeProfileValidatorV0) add(code ClaudeConnectorIssueCodeV0, field string) {
	v.issues = append(v.issues, orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:       orquestaruntime.ExternalAgentConnectorErrorCodeV0(code),
		MessageKey: "orquesta.runtime.claude." + string(code),
		Field:      field,
		Retryable:  false,
	})
}

func claudePathInsideV0(child string, parent string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

func claudeHasControlCharsV0(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}
