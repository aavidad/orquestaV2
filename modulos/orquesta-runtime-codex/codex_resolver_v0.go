package orquestaruntimecodex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

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
	if err := writeJSONFileV0(packetPath, spec.AgentPacket); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexAgentPacketFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	prompt := BuildCodexAgentPromptWithControlFilesV0(
		spec.AgentPacket,
		r.profile.PromptHints,
		CodexControlFilesV0{
			PacketPath:          packetPath,
			AckPath:             filepath.Join(r.profile.RuntimeWorkDir, CodexAgentAckFileNameV0),
			DecisionPath:        filepath.Join(r.profile.RuntimeWorkDir, CodexDirectorDecisionsFileNameV0),
			ShutdownRequestPath: filepath.Join(r.profile.RuntimeWorkDir, CodexShutdownRequestFileNameV0),
			ShutdownAckPath:     filepath.Join(r.profile.RuntimeWorkDir, CodexShutdownCheckpointAckFileNameV0),
		},
	)
	if err := os.WriteFile(promptPath, []byte(prompt), 0o600); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexAgentPromptFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	wrapper := BuildCodexWrapperScriptV0(r.profile)
	if err := os.WriteFile(filepath.Join(r.profile.RuntimeWorkDir, CodexWrapperFileNameV0), []byte(wrapper), 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexWrapperFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	return nil
}

func writeJSONFileV0(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
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
