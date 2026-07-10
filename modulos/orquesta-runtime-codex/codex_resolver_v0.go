package orquestaruntimecodex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type CodexExecResolverV0 struct {
	profile CodexConnectorProfileV0
}

func NewCodexExecResolverV0(profile CodexConnectorProfileV0) CodexExecResolverV0 {
	return CodexExecResolverV0{profile: profile}
}

func (r CodexExecResolverV0) ResolveExternalAgentProcessCommandV0(
	ctx context.Context,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (orquestaruntime.ProcessRuntimeLaunchRequestV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, "context", spec.CorrelationID, err.Error()),
		}
	}
	if issues := ValidateCodexConnectorProfileV0(r.profile); len(issues) > 0 {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, codexCorrelateIssuesV0(spec.CorrelationID, issues)
	}
	if !spec.Valid() {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorProfileInvalidV0, "external_agent_launch_spec", spec.CorrelationID, "invalid_spec"),
		}
	}
	if !r.launchRoutingReceiptV0(spec).validForLaunchV0(r.profile.Model) {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorValueInvalidV0, "model_routing", spec.CorrelationID, "model_routing_invalid"),
		}
	}
	req := r.processRuntimeLaunchRequestV0()
	if issue := codexProcessRuntimeRequestIssueV0(spec.CorrelationID, req); issue != nil {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{*issue}
	}
	if issues := r.materializeFilesV0(spec); len(issues) > 0 {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, codexCorrelateIssuesV0(spec.CorrelationID, issues)
	}
	return req, nil
}

func (r CodexExecResolverV0) processRuntimeLaunchRequestV0() orquestaruntime.ProcessRuntimeLaunchRequestV0 {
	return orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: filepath.Join(r.profile.RuntimeWorkDir, CodexWrapperFileNameV0),
		Env:         []string{},
		WorkingDir:  r.profile.ProjectWorkDir,
	}
}

func codexProcessRuntimeRequestIssueV0(
	correlationID string,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) *orquestaruntime.ExternalAgentConnectorErrorV0 {
	err := orquestaruntime.ValidateProcessRuntimeLaunchRequestV0(req)
	if err == nil {
		return nil
	}
	issue := codexIssueV0(
		CodexConnectorPathInvalidV0,
		"process_runtime_launch_request",
		correlationID,
		"process_runtime_launch_request_invalid",
	)
	if runtimeErr, ok := err.(orquestaruntime.ProcessRuntimeErrorV0); ok {
		issue.Field = runtimeErr.Field
		issue.Evidence = compactCodexIssueEvidenceV0([]string{string(runtimeErr.Code)})
	}
	return &issue
}

func (r CodexExecResolverV0) materializeFilesV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	if err := os.MkdirAll(r.profile.RuntimeWorkDir, 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, "runtime_work_dir", spec.CorrelationID, err.Error()),
		}
	}
	if err := os.MkdirAll(r.profile.ProjectWorkDir, 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, "project_work_dir", spec.CorrelationID, err.Error()),
		}
	}
	packetPath := filepath.Join(r.profile.RuntimeWorkDir, CodexAgentPacketFileNameV0)
	promptPath := filepath.Join(r.profile.RuntimeWorkDir, CodexAgentPromptFileNameV0)
	receiptPath := filepath.Join(r.profile.RuntimeWorkDir, CodexModelRoutingReceiptFileNameV0)
	if err := writeJSONFileV0(r.profile.RuntimeWorkDir, packetPath, spec.AgentPacket); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexAgentPacketFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	if err := writeJSONFileV0(r.profile.RuntimeWorkDir, receiptPath, r.launchRoutingReceiptV0(spec)); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexModelRoutingReceiptFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	promptHints := append([]string(nil), r.profile.PromptHints...)
	promptHints = append(promptHints, codexRuntimeWorkDirPromptHintV0(r.profile))
	prompt := BuildCodexAgentPromptWithControlFilesV0(
		spec.AgentPacket,
		promptHints,
		CodexControlFilesV0{
			PacketPath:          packetPath,
			AckPath:             filepath.Join(r.profile.RuntimeWorkDir, CodexAgentAckFileNameV0),
			DecisionPath:        filepath.Join(r.profile.RuntimeWorkDir, CodexDirectorDecisionsFileNameV0),
			ShutdownRequestPath: filepath.Join(r.profile.RuntimeWorkDir, CodexShutdownRequestFileNameV0),
			ShutdownAckPath:     filepath.Join(r.profile.RuntimeWorkDir, CodexShutdownCheckpointAckFileNameV0),
			SkillInstructions:   append([]CodexSkillInstructionV0(nil), r.profile.SkillInstructions...),
			ModelRouting:        CodexModelRoutingReceiptV0{},
		},
	)
	if err := writeControlFileV0(r.profile.RuntimeWorkDir, promptPath, CodexAgentPromptFileNameV0, "agent_prompt", []byte(prompt), 0o600); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexAgentPromptFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	wrapper := BuildCodexWrapperScriptV0(r.profile)
	if err := writeControlFileV0(r.profile.RuntimeWorkDir, filepath.Join(r.profile.RuntimeWorkDir, CodexWrapperFileNameV0), CodexWrapperFileNameV0, "codex_wrapper", []byte(wrapper), 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexWrapperFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	return nil
}

func (r CodexExecResolverV0) launchRoutingReceiptV0(spec orquestaruntime.ExternalAgentLaunchSpecV0) CodexModelRoutingReceiptV0 {
	receipt := r.profile.ModelRouting
	receipt.SchemaVersion = CodexModelRoutingReceiptSchemaVersionV0
	receipt.RequestID = strings.TrimSpace(spec.RequestID)
	receipt.CorrelationID = strings.TrimSpace(spec.CorrelationID)
	receipt.TaskRef = strings.TrimSpace(spec.AgentPacket.Task.TaskRef)
	receipt.ReasoningEffort = strings.TrimSpace(r.profile.ReasoningEffort)
	return receipt
}

func codexRuntimeWorkDirPromptHintV0(profile CodexConnectorProfileV0) string {
	switch strings.TrimSpace(profile.RuntimeWorkDirPlacement) {
	case CodexRuntimeWorkDirExternalRootV0:
		return "runtime_work_dir es writable root externo solo para control; no crees docs/codigo alli ni lo trates como workdir de producto."
	case CodexRuntimeWorkDirInsideProjectV0:
		return "runtime_work_dir es directorio de control dentro del proyecto; no incluyas sus ficheros en ACK.files ni en artefactos de producto."
	default:
		return "runtime_work_dir debe usarse solo para ficheros de control del agente."
	}
}

func writeJSONFileV0(rootDir string, path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeControlFileV0(rootDir, path, filepath.Base(path), filepath.Base(path), data, 0o600)
}

func writeControlFileV0(
	rootDir string,
	path string,
	fileName string,
	controlKind string,
	data []byte,
	perm os.FileMode,
) error {
	_, err := WriteCodexControlFileBytesV0(CodexControlFileWriteRequestV0{
		RootDir:     rootDir,
		Path:        path,
		FileName:    fileName,
		ControlKind: controlKind,
		Data:        data,
		Mode:        CodexControlFileWriteCreateOrReplaceV0,
		Perm:        perm,
	})
	return err
}

func codexIssueV0(
	code CodexConnectorIssueCodeV0,
	field string,
	correlationID string,
	evidence string,
) orquestaruntime.ExternalAgentConnectorErrorV0 {
	issue := orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:          orquestaruntime.ExternalAgentConnectorErrorCodeV0(code),
		MessageKey:    "orquesta.runtime.codex." + string(code),
		Field:         field,
		Retryable:     false,
		CorrelationID: correlationID,
	}
	if evidence != "" {
		issue.Evidence = []string{evidence}
	}
	return issue
}

func codexCorrelateIssuesV0(
	correlationID string,
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	for i := range issues {
		if issues[i].CorrelationID == "" {
			issues[i].CorrelationID = correlationID
		}
	}
	return issues
}

func compactCodexIssueEvidenceV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
