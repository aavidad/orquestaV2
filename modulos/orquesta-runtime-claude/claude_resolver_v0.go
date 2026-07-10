package orquestaruntimeclaude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ClaudeExecResolverV0 struct {
	profile ClaudeConnectorProfileV0
}

func NewClaudeExecResolverV0(profile ClaudeConnectorProfileV0) ClaudeExecResolverV0 {
	return ClaudeExecResolverV0{profile: profile}
}

func (r ClaudeExecResolverV0) ResolveExternalAgentProcessCommandV0(
	ctx context.Context,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (orquestaruntime.ProcessRuntimeLaunchRequestV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, "context", spec.CorrelationID, err.Error()),
		}
	}
	if issues := ValidateClaudeConnectorProfileV0(r.profile); len(issues) > 0 {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, claudeCorrelateIssuesV0(spec.CorrelationID, issues)
	}
	if specIssues := spec.Validate(); len(specIssues) > 0 {
		evidence := []string{"invalid_spec"}
		for _, issue := range specIssues {
			evidence = append(evidence, string(issue.Code))
			if strings.TrimSpace(issue.Field) != "" {
				evidence = append(evidence, string(issue.Code)+":"+strings.TrimSpace(issue.Field))
			}
			evidence = append(evidence, issue.Evidence...)
		}
		issue := claudeIssueV0(
			ClaudeConnectorProfileInvalidV0,
			"external_agent_launch_spec",
			spec.CorrelationID,
			"",
		)
		issue.Evidence = compactClaudeIssueEvidenceV0(evidence)
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{issue}
	}
	if !r.launchRoutingReceiptV0(spec).validForLaunchV0(r.profile.Model) {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorValueInvalidV0, "model_routing", spec.CorrelationID, "model_routing_invalid"),
		}
	}
	req := r.processRuntimeLaunchRequestV0()
	if issue := claudeProcessRuntimeRequestIssueV0(spec.CorrelationID, req); issue != nil {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{*issue}
	}
	if issues := r.materializeFilesV0(spec); len(issues) > 0 {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, claudeCorrelateIssuesV0(spec.CorrelationID, issues)
	}
	return req, nil
}

func (r ClaudeExecResolverV0) processRuntimeLaunchRequestV0() orquestaruntime.ProcessRuntimeLaunchRequestV0 {
	return orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: filepath.Join(r.profile.RuntimeWorkDir, ClaudeWrapperFileNameV0),
		Env:         []string{},
		WorkingDir:  r.profile.ProjectWorkDir,
	}
}

func claudeProcessRuntimeRequestIssueV0(
	correlationID string,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) *orquestaruntime.ExternalAgentConnectorErrorV0 {
	err := orquestaruntime.ValidateProcessRuntimeLaunchRequestV0(req)
	if err == nil {
		return nil
	}
	issue := claudeIssueV0(
		ClaudeConnectorPathInvalidV0,
		"process_runtime_launch_request",
		correlationID,
		"process_runtime_launch_request_invalid",
	)
	if runtimeErr, ok := err.(orquestaruntime.ProcessRuntimeErrorV0); ok {
		issue.Field = runtimeErr.Field
		issue.Evidence = compactClaudeIssueEvidenceV0([]string{string(runtimeErr.Code)})
	}
	return &issue
}

func (r ClaudeExecResolverV0) materializeFilesV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	if err := os.MkdirAll(r.profile.RuntimeWorkDir, 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, "runtime_work_dir", spec.CorrelationID, err.Error()),
		}
	}
	if err := os.MkdirAll(r.profile.ProjectWorkDir, 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, "project_work_dir", spec.CorrelationID, err.Error()),
		}
	}
	packetPath := filepath.Join(r.profile.RuntimeWorkDir, ClaudeAgentPacketFileNameV0)
	promptPath := filepath.Join(r.profile.RuntimeWorkDir, ClaudeAgentPromptFileNameV0)
	receiptPath := filepath.Join(r.profile.RuntimeWorkDir, ClaudeModelRoutingReceiptFileNameV0)
	if err := writeClaudeJSONFileV0(r.profile.RuntimeWorkDir, packetPath, spec.AgentPacket); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, ClaudeAgentPacketFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	if err := writeClaudeJSONFileV0(r.profile.RuntimeWorkDir, receiptPath, r.launchRoutingReceiptV0(spec)); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, ClaudeModelRoutingReceiptFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	promptHints := append([]string(nil), r.profile.PromptHints...)
	promptHints = append(promptHints, claudeRuntimeWorkDirPromptHintForLocaleV0(r.profile))
	prompt := BuildClaudeAgentPromptWithLocaleAndControlFilesV0(
		spec.AgentPacket,
		promptHints,
		r.profile.PromptLocale,
		ClaudeControlFilesV0{
			PacketPath:          packetPath,
			AckPath:             filepath.Join(r.profile.RuntimeWorkDir, ClaudeAgentAckFileNameV0),
			DecisionPath:        filepath.Join(r.profile.RuntimeWorkDir, ClaudeDirectorDecisionsFileNameV0),
			ShutdownRequestPath: filepath.Join(r.profile.RuntimeWorkDir, ClaudeShutdownRequestFileNameV0),
			ShutdownAckPath:     filepath.Join(r.profile.RuntimeWorkDir, ClaudeShutdownCheckpointAckFileNameV0),
		},
	)
	if err := writeClaudeControlFileV0(r.profile.RuntimeWorkDir, promptPath, ClaudeAgentPromptFileNameV0, []byte(prompt), 0o600); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, ClaudeAgentPromptFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	wrapper := BuildClaudeWrapperScriptV0(r.profile)
	if err := writeClaudeControlFileV0(r.profile.RuntimeWorkDir, filepath.Join(r.profile.RuntimeWorkDir, ClaudeWrapperFileNameV0), ClaudeWrapperFileNameV0, []byte(wrapper), 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			claudeIssueV0(ClaudeConnectorFilesystemV0, ClaudeWrapperFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	return nil
}

func (r ClaudeExecResolverV0) launchRoutingReceiptV0(spec orquestaruntime.ExternalAgentLaunchSpecV0) ClaudeModelRoutingReceiptV0 {
	receipt := r.profile.ModelRouting
	receipt.SchemaVersion = ClaudeModelRoutingReceiptSchemaVersionV0
	receipt.RequestID = strings.TrimSpace(spec.RequestID)
	receipt.CorrelationID = strings.TrimSpace(spec.CorrelationID)
	receipt.TaskRef = strings.TrimSpace(spec.AgentPacket.Task.TaskRef)
	receipt.ReasoningEffort = strings.TrimSpace(r.profile.Effort)
	return receipt
}

func claudeRuntimeWorkDirPromptHintV0(profile ClaudeConnectorProfileV0) string {
	switch strings.TrimSpace(profile.RuntimeWorkDirPlacement) {
	case ClaudeRuntimeWorkDirExternalRootV0:
		return "runtime_work_dir es writable root externo solo para control; no crees docs/codigo alli ni lo trates como workdir de producto."
	case ClaudeRuntimeWorkDirInsideProjectV0:
		return "runtime_work_dir es directorio de control dentro del proyecto; no incluyas sus ficheros en ACK.files ni en artefactos de producto."
	default:
		return "runtime_work_dir debe usarse solo para ficheros de control del agente."
	}
}

func claudeRuntimeWorkDirPromptHintForLocaleV0(profile ClaudeConnectorProfileV0) string {
	if !claudeGoalPromptEnglishLocaleV0(profile.PromptLocale) {
		return claudeRuntimeWorkDirPromptHintV0(profile)
	}
	switch strings.TrimSpace(profile.RuntimeWorkDirPlacement) {
	case ClaudeRuntimeWorkDirExternalRootV0:
		return "runtime_work_dir is an external writable root for control only; do not create product docs/code there or treat it as the product workdir."
	case ClaudeRuntimeWorkDirInsideProjectV0:
		return "runtime_work_dir is a control directory inside the project; do not include its files in ACK.files or product artifacts."
	default:
		return "runtime_work_dir must be used only for agent control files."
	}
}

func writeClaudeJSONFileV0(rootDir string, path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeClaudeControlFileV0(rootDir, path, filepath.Base(path), data, 0o600)
}

func writeClaudeControlFileV0(
	rootDir string,
	path string,
	fileName string,
	data []byte,
	perm os.FileMode,
) error {
	rootDir = filepath.Clean(strings.TrimSpace(rootDir))
	path = filepath.Clean(strings.TrimSpace(path))
	fileName = strings.TrimSpace(fileName)
	if rootDir == "" || rootDir == "." || !filepath.IsAbs(rootDir) {
		return fmt.Errorf("claude_control_file_write: root_invalid")
	}
	if path == "" || path == "." || !filepath.IsAbs(path) || filepath.Base(path) != fileName {
		return fmt.Errorf("claude_control_file_write: path_invalid")
	}
	rel, err := filepath.Rel(rootDir, path)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("claude_control_file_write: path_outside_root")
	}
	if perm != 0o600 && perm != 0o700 {
		return fmt.Errorf("claude_control_file_write: perm_invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("claude_control_file_write: parent_unavailable")
	}
	return os.WriteFile(path, data, perm)
}

func claudeIssueV0(
	code ClaudeConnectorIssueCodeV0,
	field string,
	correlationID string,
	evidence string,
) orquestaruntime.ExternalAgentConnectorErrorV0 {
	issue := orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:          orquestaruntime.ExternalAgentConnectorErrorCodeV0(code),
		MessageKey:    "orquesta.runtime.claude." + string(code),
		Field:         field,
		Retryable:     false,
		CorrelationID: correlationID,
	}
	if evidence != "" {
		issue.Evidence = []string{evidence}
	}
	return issue
}

func claudeCorrelateIssuesV0(
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

func compactClaudeIssueEvidenceV0(values []string) []string {
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
