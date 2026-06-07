package orquestaruntimegemini

import (
	"fmt"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	GeminiConnectorProfileSchemaVersionV0 = "gemini_connector_profile.v0"

	GeminiAgentPacketFileNameV0           = "agent_packet.json"
	GeminiAgentPromptFileNameV0           = "agent_prompt.txt"
	GeminiAgentAckFileNameV0              = "agent_ack.json"
	GeminiDirectorDecisionsFileNameV0     = "director_decisions.json"
	GeminiShutdownRequestFileNameV0       = "orquesta_shutdown_request.json"
	GeminiShutdownCheckpointAckFileNameV0 = "agent_shutdown_checkpoint_ack.json"
	GeminiStdoutFileNameV0                = "gemini_stdout.log"
	GeminiStderrFileNameV0                = "gemini_stderr.log"
	GeminiWrapperFileNameV0               = "orquesta_gemini_exec_v0.sh"
)

type GeminiConnectorIssueCodeV0 string

const (
	GeminiConnectorProfileInvalidV0 GeminiConnectorIssueCodeV0 = "gemini_connector_profile_invalido"
	GeminiConnectorOptInRequiredV0  GeminiConnectorIssueCodeV0 = "gemini_connector_opt_in_requerido"
	GeminiConnectorPathInvalidV0    GeminiConnectorIssueCodeV0 = "gemini_connector_path_invalido"
	GeminiConnectorValueInvalidV0   GeminiConnectorIssueCodeV0 = "gemini_connector_value_invalido"
	GeminiConnectorFilesystemV0     GeminiConnectorIssueCodeV0 = "gemini_connector_filesystem_error"
)

const (
	GeminiRuntimeWorkDirInsideProjectV0 = "inside_project_control_dir"
	GeminiRuntimeWorkDirExternalRootV0  = "external_control_root"
)

type GeminiConnectorProfileV0 struct {
	SchemaVersion           string   `json:"schema_version"`
	OptIn                   bool     `json:"opt_in"`
	CommandPath             string   `json:"command_path"`
	ProjectWorkDir          string   `json:"project_work_dir"`
	RuntimeWorkDir          string   `json:"runtime_work_dir"`
	RuntimeWorkDirPlacement string   `json:"runtime_work_dir_placement,omitempty"`
	HomeDir                 string   `json:"home_dir,omitempty"`
	PathEnv                 string   `json:"path_env,omitempty"`
	Model                   string   `json:"model,omitempty"`
	ApprovalMode            string   `json:"approval_mode,omitempty"`
	OutputFormat            string   `json:"output_format,omitempty"`
	ExtraArgs               []string `json:"extra_args,omitempty"`
	PromptHints             []string `json:"prompt_hints,omitempty"`
}

func ValidateGeminiConnectorProfileV0(
	profile GeminiConnectorProfileV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	v := geminiProfileValidatorV0{}
	if profile.SchemaVersion != GeminiConnectorProfileSchemaVersionV0 {
		v.add(GeminiConnectorProfileInvalidV0, "schema_version")
	}
	if !profile.OptIn {
		v.add(GeminiConnectorOptInRequiredV0, "opt_in")
	}
	v.requireAbsPath("command_path", profile.CommandPath)
	v.requireAbsPath("project_work_dir", profile.ProjectWorkDir)
	v.requireAbsPath("runtime_work_dir", profile.RuntimeWorkDir)
	v.validateRuntimeWorkDirPlacementV0(profile)
	v.validateRuntimeWorkDirIsolationV0(profile)
	v.optionalAbsPath("home_dir", profile.HomeDir)
	v.optionalSafeValue("path_env", profile.PathEnv)
	v.optionalSafeValue("model", profile.Model)
	v.optionalSafeValue("approval_mode", profile.ApprovalMode)
	v.optionalSafeValue("output_format", profile.OutputFormat)
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

func InferGeminiRuntimeWorkDirPlacementV0(projectWorkDir string, runtimeWorkDir string) string {
	project := filepath.Clean(strings.TrimSpace(projectWorkDir))
	runtime := filepath.Clean(strings.TrimSpace(runtimeWorkDir))
	if project == "." || runtime == "." {
		return ""
	}
	if runtime == project || geminiPathInsideV0(runtime, project) {
		return GeminiRuntimeWorkDirInsideProjectV0
	}
	return GeminiRuntimeWorkDirExternalRootV0
}

type geminiProfileValidatorV0 struct {
	issues []orquestaruntime.ExternalAgentConnectorErrorV0
}

func (v *geminiProfileValidatorV0) requireAbsPath(field, value string) {
	if strings.TrimSpace(value) == "" || !filepath.IsAbs(value) || geminiHasControlCharsV0(value) {
		v.add(GeminiConnectorPathInvalidV0, field)
	}
}

func (v *geminiProfileValidatorV0) optionalAbsPath(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	v.requireAbsPath(field, value)
}

func (v *geminiProfileValidatorV0) optionalSafeValue(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if geminiHasControlCharsV0(value) || strings.ContainsAny(value, "\x00\r\n") {
		v.add(GeminiConnectorValueInvalidV0, field)
	}
}

func (v *geminiProfileValidatorV0) validateRuntimeWorkDirPlacementV0(
	profile GeminiConnectorProfileV0,
) {
	declared := strings.TrimSpace(profile.RuntimeWorkDirPlacement)
	if declared == "" {
		return
	}
	expected := InferGeminiRuntimeWorkDirPlacementV0(
		profile.ProjectWorkDir,
		profile.RuntimeWorkDir,
	)
	if expected != "" && declared != expected {
		v.add(GeminiConnectorValueInvalidV0, "runtime_work_dir_placement")
	}
	v.optionalSafeValue("runtime_work_dir_placement", declared)
}

func (v *geminiProfileValidatorV0) validateRuntimeWorkDirIsolationV0(
	profile GeminiConnectorProfileV0,
) {
	if strings.TrimSpace(profile.ProjectWorkDir) == "" ||
		strings.TrimSpace(profile.RuntimeWorkDir) == "" {
		return
	}
	if filepath.Clean(profile.RuntimeWorkDir) == filepath.Clean(profile.ProjectWorkDir) ||
		geminiPathInsideV0(profile.ProjectWorkDir, profile.RuntimeWorkDir) {
		v.add(GeminiConnectorPathInvalidV0, "runtime_work_dir")
	}
}

func (v *geminiProfileValidatorV0) rejectOwnedOrUnsafeExtraArgV0(field, value string) {
	trimmed := strings.TrimSpace(value)
	switch {
	case trimmed == "--prompt",
		strings.HasPrefix(trimmed, "--prompt="),
		trimmed == "-p",
		trimmed == "--prompt-interactive",
		strings.HasPrefix(trimmed, "--prompt-interactive="),
		trimmed == "-i",
		trimmed == "--model",
		strings.HasPrefix(trimmed, "--model="),
		trimmed == "-m",
		trimmed == "--approval-mode",
		strings.HasPrefix(trimmed, "--approval-mode="),
		trimmed == "--output-format",
		strings.HasPrefix(trimmed, "--output-format="),
		trimmed == "--include-directories",
		strings.HasPrefix(trimmed, "--include-directories="),
		trimmed == "--worktree",
		strings.HasPrefix(trimmed, "--worktree="),
		trimmed == "-w",
		trimmed == "--allowed-tools",
		strings.HasPrefix(trimmed, "--allowed-tools="):
		v.add(GeminiConnectorValueInvalidV0, field)
	}
}

func (v *geminiProfileValidatorV0) add(code GeminiConnectorIssueCodeV0, field string) {
	v.issues = append(v.issues, orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:       orquestaruntime.ExternalAgentConnectorErrorCodeV0(code),
		MessageKey: "orquesta.runtime.gemini." + string(code),
		Field:      field,
		Retryable:  false,
	})
}

func geminiPathInsideV0(child string, parent string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

func geminiHasControlCharsV0(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}
