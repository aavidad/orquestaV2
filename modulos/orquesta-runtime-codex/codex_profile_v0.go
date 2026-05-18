package orquestaruntimecodex

import (
	"fmt"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	CodexConnectorProfileSchemaVersionV0 = "codex_connector_profile.v0"

	CodexAgentPacketFileNameV0           = "agent_packet.json"
	CodexAgentPromptFileNameV0           = "agent_prompt.txt"
	CodexAgentAckFileNameV0              = "agent_ack.json"
	CodexDirectorDecisionsFileNameV0     = "director_decisions.json"
	CodexShutdownRequestFileNameV0       = "orquesta_shutdown_request.json"
	CodexShutdownCheckpointAckFileNameV0 = "agent_shutdown_checkpoint_ack.json"
	CodexLastMessageFileNameV0           = "codex_last_message.txt"
	CodexStdoutFileNameV0                = "codex_stdout.log"
	CodexStderrFileNameV0                = "codex_stderr.log"
	CodexWrapperFileNameV0               = "orquesta_codex_exec_v0.sh"
)

type CodexConnectorIssueCodeV0 string

const (
	CodexConnectorProfileInvalidV0 CodexConnectorIssueCodeV0 = "codex_connector_profile_invalido"
	CodexConnectorOptInRequiredV0  CodexConnectorIssueCodeV0 = "codex_connector_opt_in_requerido"
	CodexConnectorPathInvalidV0    CodexConnectorIssueCodeV0 = "codex_connector_path_invalido"
	CodexConnectorValueInvalidV0   CodexConnectorIssueCodeV0 = "codex_connector_value_invalido"
	CodexConnectorFilesystemV0     CodexConnectorIssueCodeV0 = "codex_connector_filesystem_error"
	CodexConnectorAckInvalidV0     CodexConnectorIssueCodeV0 = "codex_agent_ack_invalido"
	CodexConnectorAckCorrelationV0 CodexConnectorIssueCodeV0 = "codex_agent_ack_correlacion_invalida"
	CodexConnectorAckArtifactV0    CodexConnectorIssueCodeV0 = "codex_agent_ack_artifact_invalido"
	CodexConnectorAckForbiddenV0   CodexConnectorIssueCodeV0 = "codex_agent_ack_detalle_prohibido"
)

type CodexConnectorProfileV0 struct {
	SchemaVersion   string   `json:"schema_version"`
	OptIn           bool     `json:"opt_in"`
	CommandPath     string   `json:"command_path"`
	ProjectWorkDir  string   `json:"project_work_dir"`
	RuntimeWorkDir  string   `json:"runtime_work_dir"`
	CodeHomeDir     string   `json:"code_home_dir,omitempty"`
	HomeDir         string   `json:"home_dir,omitempty"`
	PathEnv         string   `json:"path_env,omitempty"`
	Model           string   `json:"model,omitempty"`
	ReasoningEffort string   `json:"reasoning_effort,omitempty"`
	Profile         string   `json:"profile,omitempty"`
	Sandbox         string   `json:"sandbox,omitempty"`
	ApprovalPolicy  string   `json:"approval_policy,omitempty"`
	ExtraArgs       []string `json:"extra_args,omitempty"`
	PromptHints     []string `json:"prompt_hints,omitempty"`
}

func ValidateCodexConnectorProfileV0(
	profile CodexConnectorProfileV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	v := codexProfileValidatorV0{}
	if profile.SchemaVersion != CodexConnectorProfileSchemaVersionV0 {
		v.add(CodexConnectorProfileInvalidV0, "schema_version")
	}
	if !profile.OptIn {
		v.add(CodexConnectorOptInRequiredV0, "opt_in")
	}
	v.requireAbsPath("command_path", profile.CommandPath)
	v.requireAbsPath("project_work_dir", profile.ProjectWorkDir)
	v.requireAbsPath("runtime_work_dir", profile.RuntimeWorkDir)
	v.validateWorkspaceWriteRuntimeIsolationV0(profile)
	v.optionalAbsPath("code_home_dir", profile.CodeHomeDir)
	v.optionalAbsPath("home_dir", profile.HomeDir)
	v.optionalSafeValue("path_env", profile.PathEnv)
	v.optionalSafeValue("model", profile.Model)
	v.optionalSafeValue("reasoning_effort", profile.ReasoningEffort)
	v.optionalSafeValue("profile", profile.Profile)
	v.optionalSafeValue("sandbox", profile.Sandbox)
	v.optionalSafeValue("approval_policy", profile.ApprovalPolicy)
	for i, value := range profile.ExtraArgs {
		v.optionalSafeValue(fmt.Sprintf("extra_args[%d]", i), value)
	}
	for i, value := range profile.PromptHints {
		v.optionalSafeValue(fmt.Sprintf("prompt_hints[%d]", i), value)
	}
	return v.issues
}

type codexProfileValidatorV0 struct {
	issues []orquestaruntime.ExternalAgentConnectorErrorV0
}

func (v *codexProfileValidatorV0) requireAbsPath(field, value string) {
	if strings.TrimSpace(value) == "" || !filepath.IsAbs(value) || codexHasControlCharsV0(value) {
		v.add(CodexConnectorPathInvalidV0, field)
	}
}

func (v *codexProfileValidatorV0) optionalAbsPath(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	v.requireAbsPath(field, value)
}

func (v *codexProfileValidatorV0) optionalSafeValue(field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if codexHasControlCharsV0(value) || strings.ContainsAny(value, "\x00\r\n") {
		v.add(CodexConnectorValueInvalidV0, field)
	}
}

func (v *codexProfileValidatorV0) validateWorkspaceWriteRuntimeIsolationV0(
	profile CodexConnectorProfileV0,
) {
	if strings.TrimSpace(profile.Sandbox) != "workspace-write" ||
		strings.TrimSpace(profile.ProjectWorkDir) == "" ||
		strings.TrimSpace(profile.RuntimeWorkDir) == "" {
		return
	}
	if filepath.Clean(profile.RuntimeWorkDir) == filepath.Clean(profile.ProjectWorkDir) ||
		codexPathInsideV0(profile.ProjectWorkDir, profile.RuntimeWorkDir) {
		v.add(CodexConnectorPathInvalidV0, "runtime_work_dir")
	}
}

func (v *codexProfileValidatorV0) add(code CodexConnectorIssueCodeV0, field string) {
	v.issues = append(v.issues, orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:       orquestaruntime.ExternalAgentConnectorErrorCodeV0(code),
		MessageKey: "orquesta.runtime.codex." + string(code),
		Field:      field,
		Retryable:  false,
	})
}

func codexPathInsideV0(child string, parent string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}

func codexHasControlCharsV0(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}
