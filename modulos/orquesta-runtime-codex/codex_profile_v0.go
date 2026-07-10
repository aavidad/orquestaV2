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
	CodexProcessDoneFileNameV0           = "codex_process_done_v0"
	CodexStdoutFileNameV0                = "codex_stdout.log"
	CodexStderrFileNameV0                = "codex_stderr.log"
	CodexUsageAccountingFileNameV0       = "codex_usage_accounting.json"
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
	CodexConnectorContextMissingV0 CodexConnectorIssueCodeV0 = "codex_agent_contexto_requerido_no_materializado"
)

const (
	CodexRuntimeWorkDirInsideProjectV0 = "inside_project_control_dir"
	CodexRuntimeWorkDirExternalRootV0  = "external_control_root"
	CodexApprovalPolicyNeverV0         = "never"
	CodexApprovalPolicyOnRequestV0     = "on-request"
	CodexApprovalPolicyOnFailureV0     = "on-failure"
	CodexApprovalPolicyUnlessTrustedV0 = "unless-trusted"
	CodexApprovalPolicyUntrustedV0     = "untrusted"
	CodexApprovalPolicyOnWorkspaceV0   = "on-workspace-write"
)

type CodexConnectorProfileV0 struct {
	SchemaVersion            string                     `json:"schema_version"`
	OptIn                    bool                       `json:"opt_in"`
	CommandPath              string                     `json:"command_path"`
	ProjectWorkDir           string                     `json:"project_work_dir"`
	RuntimeWorkDir           string                     `json:"runtime_work_dir"`
	RuntimeWorkDirPlacement  string                     `json:"runtime_work_dir_placement,omitempty"`
	CodeHomeDir              string                     `json:"code_home_dir,omitempty"`
	HomeDir                  string                     `json:"home_dir,omitempty"`
	PathEnv                  string                     `json:"path_env,omitempty"`
	Model                    string                     `json:"model,omitempty"`
	ModelRouting             CodexModelRoutingReceiptV0 `json:"model_routing,omitempty"`
	ReasoningEffort          string                     `json:"reasoning_effort,omitempty"`
	Profile                  string                     `json:"profile,omitempty"`
	Sandbox                  string                     `json:"sandbox,omitempty"`
	ApprovalPolicy           string                     `json:"approval_policy,omitempty"`
	InteractiveApprovalOptIn bool                       `json:"interactive_approval_opt_in,omitempty"`
	ExtraArgs                []string                   `json:"extra_args,omitempty"`
	PromptHints              []string                   `json:"prompt_hints,omitempty"`
	SkillInstructions        []CodexSkillInstructionV0  `json:"skill_instructions,omitempty"`
}

type CodexSkillInstructionV0 struct {
	SkillRef string `json:"skill_ref"`
	Text     string `json:"text"`
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
	v.validateRuntimeWorkDirPlacementV0(profile)
	v.validateWorkspaceWriteRuntimeIsolationV0(profile)
	v.optionalAbsPath("code_home_dir", profile.CodeHomeDir)
	v.optionalAbsPath("home_dir", profile.HomeDir)
	v.optionalSafeValue("path_env", profile.PathEnv)
	v.optionalSafeValue("model", profile.Model)
	v.optionalSafeValue("reasoning_effort", profile.ReasoningEffort)
	v.optionalSafeValue("profile", profile.Profile)
	v.requireSupportedSandboxV0(profile.Sandbox)
	v.validateApprovalPolicyV0(profile)
	for i, value := range profile.ExtraArgs {
		v.optionalSafeValue(fmt.Sprintf("extra_args[%d]", i), value)
		v.rejectWorkspaceEscapeExtraArgV0(fmt.Sprintf("extra_args[%d]", i), value)
	}
	for i, value := range profile.PromptHints {
		v.optionalSafeValue(fmt.Sprintf("prompt_hints[%d]", i), value)
	}
	for i, instruction := range profile.SkillInstructions {
		v.validateSkillInstructionV0(i, instruction)
	}
	return v.issues
}

func InferCodexRuntimeWorkDirPlacementV0(projectWorkDir string, runtimeWorkDir string) string {
	project := filepath.Clean(strings.TrimSpace(projectWorkDir))
	runtime := filepath.Clean(strings.TrimSpace(runtimeWorkDir))
	if project == "." || runtime == "." {
		return ""
	}
	if runtime == project || codexPathInsideV0(runtime, project) {
		return CodexRuntimeWorkDirInsideProjectV0
	}
	return CodexRuntimeWorkDirExternalRootV0
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

func (v *codexProfileValidatorV0) requireSupportedSandboxV0(value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "workspace-write" && trimmed != "danger-full-access" {
		v.add(CodexConnectorValueInvalidV0, "sandbox")
		return
	}
	v.optionalSafeValue("sandbox", trimmed)
}

func (v *codexProfileValidatorV0) validateApprovalPolicyV0(profile CodexConnectorProfileV0) {
	trimmed := strings.TrimSpace(profile.ApprovalPolicy)
	if trimmed == "" {
		v.add(CodexConnectorValueInvalidV0, "approval_policy")
		return
	}
	switch trimmed {
	case CodexApprovalPolicyNeverV0:
	case CodexApprovalPolicyOnFailureV0,
		CodexApprovalPolicyOnRequestV0,
		CodexApprovalPolicyUnlessTrustedV0,
		CodexApprovalPolicyUntrustedV0,
		CodexApprovalPolicyOnWorkspaceV0:
		if !profile.InteractiveApprovalOptIn {
			v.add(CodexConnectorValueInvalidV0, "approval_policy")
			return
		}
	default:
		v.add(CodexConnectorValueInvalidV0, "approval_policy")
		return
	}
	v.optionalSafeValue("approval_policy", trimmed)
}

func (v *codexProfileValidatorV0) validateRuntimeWorkDirPlacementV0(
	profile CodexConnectorProfileV0,
) {
	declared := strings.TrimSpace(profile.RuntimeWorkDirPlacement)
	if declared == "" {
		return
	}
	expected := InferCodexRuntimeWorkDirPlacementV0(
		profile.ProjectWorkDir,
		profile.RuntimeWorkDir,
	)
	if expected != "" && declared != expected {
		v.add(CodexConnectorValueInvalidV0, "runtime_work_dir_placement")
	}
	v.optionalSafeValue("runtime_work_dir_placement", declared)
}

func (v *codexProfileValidatorV0) rejectWorkspaceEscapeExtraArgV0(field, value string) {
	trimmed := strings.TrimSpace(value)
	switch {
	case trimmed == "--add-dir",
		strings.HasPrefix(trimmed, "--add-dir="),
		trimmed == "-C",
		trimmed == "--cd",
		strings.HasPrefix(trimmed, "--cd="),
		trimmed == "--sandbox",
		strings.HasPrefix(trimmed, "--sandbox="):
		v.add(CodexConnectorValueInvalidV0, field)
	}
}

func (v *codexProfileValidatorV0) validateSkillInstructionV0(index int, instruction CodexSkillInstructionV0) {
	refField := fmt.Sprintf("skill_instructions[%d].skill_ref", index)
	textField := fmt.Sprintf("skill_instructions[%d].text", index)
	ref := strings.TrimSpace(instruction.SkillRef)
	text := strings.TrimSpace(instruction.Text)
	if ref == "" || strings.ContainsAny(ref, " /\\\t\r\n") || codexHasControlCharsV0(ref) {
		v.add(CodexConnectorValueInvalidV0, refField)
	}
	if text == "" || len(text) > 2000 {
		v.add(CodexConnectorValueInvalidV0, textField)
		return
	}
	v.optionalSafeValue(textField, text)
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
