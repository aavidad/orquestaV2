package orquestaruntimegemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type GeminiExecResolverV0 struct {
	profile GeminiConnectorProfileV0
}

func NewGeminiExecResolverV0(profile GeminiConnectorProfileV0) GeminiExecResolverV0 {
	return GeminiExecResolverV0{profile: profile}
}

func (r GeminiExecResolverV0) ResolveExternalAgentProcessCommandV0(
	ctx context.Context,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (orquestaruntime.ProcessRuntimeLaunchRequestV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			geminiIssueV0(GeminiConnectorFilesystemV0, "context", spec.CorrelationID, err.Error()),
		}
	}
	if issues := ValidateGeminiConnectorProfileV0(r.profile); len(issues) > 0 {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, geminiCorrelateIssuesV0(spec.CorrelationID, issues)
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
		issue := geminiIssueV0(
			GeminiConnectorProfileInvalidV0,
			"external_agent_launch_spec",
			spec.CorrelationID,
			"",
		)
		issue.Evidence = compactGeminiIssueEvidenceV0(evidence)
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			issue,
		}
	}
	req := r.processRuntimeLaunchRequestV0()
	if issue := geminiProcessRuntimeRequestIssueV0(spec.CorrelationID, req); issue != nil {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{*issue}
	}
	if issues := r.materializeFilesV0(spec); len(issues) > 0 {
		return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, geminiCorrelateIssuesV0(spec.CorrelationID, issues)
	}
	return req, nil
}

func (r GeminiExecResolverV0) processRuntimeLaunchRequestV0() orquestaruntime.ProcessRuntimeLaunchRequestV0 {
	return orquestaruntime.ProcessRuntimeLaunchRequestV0{
		CommandPath: filepath.Join(r.profile.RuntimeWorkDir, GeminiWrapperFileNameV0),
		Env:         []string{},
		WorkingDir:  r.profile.ProjectWorkDir,
	}
}

func geminiProcessRuntimeRequestIssueV0(
	correlationID string,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) *orquestaruntime.ExternalAgentConnectorErrorV0 {
	err := orquestaruntime.ValidateProcessRuntimeLaunchRequestV0(req)
	if err == nil {
		return nil
	}
	issue := geminiIssueV0(
		GeminiConnectorPathInvalidV0,
		"process_runtime_launch_request",
		correlationID,
		"process_runtime_launch_request_invalid",
	)
	if runtimeErr, ok := err.(orquestaruntime.ProcessRuntimeErrorV0); ok {
		issue.Field = runtimeErr.Field
		issue.Evidence = compactGeminiIssueEvidenceV0([]string{string(runtimeErr.Code)})
	}
	return &issue
}

func (r GeminiExecResolverV0) materializeFilesV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	if err := os.MkdirAll(r.profile.RuntimeWorkDir, 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			geminiIssueV0(GeminiConnectorFilesystemV0, "runtime_work_dir", spec.CorrelationID, err.Error()),
		}
	}
	if err := os.MkdirAll(r.profile.ProjectWorkDir, 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			geminiIssueV0(GeminiConnectorFilesystemV0, "project_work_dir", spec.CorrelationID, err.Error()),
		}
	}
	packetPath := filepath.Join(r.profile.RuntimeWorkDir, GeminiAgentPacketFileNameV0)
	promptPath := filepath.Join(r.profile.RuntimeWorkDir, GeminiAgentPromptFileNameV0)
	if err := writeGeminiJSONFileV0(r.profile.RuntimeWorkDir, packetPath, spec.AgentPacket); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			geminiIssueV0(GeminiConnectorFilesystemV0, GeminiAgentPacketFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	promptHints := append([]string(nil), r.profile.PromptHints...)
	promptHints = append(promptHints, geminiRuntimeWorkDirPromptHintForLocaleV0(r.profile))
	prompt := BuildGeminiAgentPromptWithLocaleAndControlFilesV0(
		spec.AgentPacket,
		promptHints,
		r.profile.PromptLocale,
		GeminiControlFilesV0{
			PacketPath:          packetPath,
			AckPath:             filepath.Join(r.profile.RuntimeWorkDir, GeminiAgentAckFileNameV0),
			DecisionPath:        filepath.Join(r.profile.RuntimeWorkDir, GeminiDirectorDecisionsFileNameV0),
			ShutdownRequestPath: filepath.Join(r.profile.RuntimeWorkDir, GeminiShutdownRequestFileNameV0),
			ShutdownAckPath:     filepath.Join(r.profile.RuntimeWorkDir, GeminiShutdownCheckpointAckFileNameV0),
		},
	)
	if err := writeGeminiControlFileV0(r.profile.RuntimeWorkDir, promptPath, GeminiAgentPromptFileNameV0, []byte(prompt), 0o600); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			geminiIssueV0(GeminiConnectorFilesystemV0, GeminiAgentPromptFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	wrapper := BuildGeminiWrapperScriptV0(r.profile)
	if err := writeGeminiControlFileV0(r.profile.RuntimeWorkDir, filepath.Join(r.profile.RuntimeWorkDir, GeminiWrapperFileNameV0), GeminiWrapperFileNameV0, []byte(wrapper), 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			geminiIssueV0(GeminiConnectorFilesystemV0, GeminiWrapperFileNameV0, spec.CorrelationID, err.Error()),
		}
	}
	return nil
}

func geminiRuntimeWorkDirPromptHintV0(profile GeminiConnectorProfileV0) string {
	switch strings.TrimSpace(profile.RuntimeWorkDirPlacement) {
	case GeminiRuntimeWorkDirExternalRootV0:
		return "runtime_work_dir es writable root externo solo para control; no crees docs/codigo alli ni lo trates como workdir de producto."
	case GeminiRuntimeWorkDirInsideProjectV0:
		return "runtime_work_dir es directorio de control dentro del proyecto; no incluyas sus ficheros en ACK.files ni en artefactos de producto."
	default:
		return "runtime_work_dir debe usarse solo para ficheros de control del agente."
	}
}

func geminiRuntimeWorkDirPromptHintForLocaleV0(profile GeminiConnectorProfileV0) string {
	if !geminiGoalPromptEnglishLocaleV0(profile.PromptLocale) {
		return geminiRuntimeWorkDirPromptHintV0(profile)
	}
	switch strings.TrimSpace(profile.RuntimeWorkDirPlacement) {
	case GeminiRuntimeWorkDirExternalRootV0:
		return "runtime_work_dir is an external writable root for control only; do not create product docs/code there or treat it as the product workdir."
	case GeminiRuntimeWorkDirInsideProjectV0:
		return "runtime_work_dir is a control directory inside the project; do not include its files in ACK.files or product artifacts."
	default:
		return "runtime_work_dir must be used only for agent control files."
	}
}

func writeGeminiJSONFileV0(rootDir string, path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeGeminiControlFileV0(rootDir, path, filepath.Base(path), data, 0o600)
}

func writeGeminiControlFileV0(
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
		return fmt.Errorf("gemini_control_file_write: root_invalid")
	}
	if path == "" || path == "." || !filepath.IsAbs(path) || filepath.Base(path) != fileName {
		return fmt.Errorf("gemini_control_file_write: path_invalid")
	}
	rel, err := filepath.Rel(rootDir, path)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("gemini_control_file_write: path_outside_root")
	}
	if perm != 0o600 && perm != 0o700 {
		return fmt.Errorf("gemini_control_file_write: perm_invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("gemini_control_file_write: parent_unavailable")
	}
	return os.WriteFile(path, data, perm)
}

func geminiIssueV0(
	code GeminiConnectorIssueCodeV0,
	field string,
	correlationID string,
	evidence string,
) orquestaruntime.ExternalAgentConnectorErrorV0 {
	issue := orquestaruntime.ExternalAgentConnectorErrorV0{
		Code:          orquestaruntime.ExternalAgentConnectorErrorCodeV0(code),
		MessageKey:    "orquesta.runtime.gemini." + string(code),
		Field:         field,
		Retryable:     false,
		CorrelationID: correlationID,
	}
	if evidence != "" {
		issue.Evidence = []string{evidence}
	}
	return issue
}

func geminiCorrelateIssuesV0(
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

func compactGeminiIssueEvidenceV0(values []string) []string {
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
