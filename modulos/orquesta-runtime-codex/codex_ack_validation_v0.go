package orquestaruntimecodex

import (
	"encoding/json"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const codexAgentAckStatusCompletedV0 = "completed"

func ReadAndValidateCodexAgentAckFileV0(
	path string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	data, err := os.ReadFile(path)
	if err != nil {
		issue := codexIssueV0(CodexConnectorAckInvalidV0, CodexAgentAckFileNameV0, spec.CorrelationID, "read_failed")
		if os.IsNotExist(err) {
			issue.Retryable = true
			issue.Evidence = []string{"ack_not_ready"}
		}
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{issue}
	}
	return ValidateCodexAgentAckBytesForSpecV0(data, spec)
}

func ValidateCodexAgentAckBytesForSpecV0(
	data []byte,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	if codexAckBytesContainForbiddenDetailV0(data) {
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckForbiddenV0, CodexAgentAckFileNameV0, spec.CorrelationID, "forbidden_detail"),
		}
	}
	var ack CodexAgentAckV0
	if err := json.Unmarshal(data, &ack); err != nil {
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckInvalidV0, CodexAgentAckFileNameV0, spec.CorrelationID, "json_invalid"),
		}
	}
	issues := ValidateCodexAgentAckForSpecV0(ack, spec)
	if strings.TrimSpace(ack.Status) == codexAgentAckStatusCompletedV0 &&
		codexAckBytesHaveFailedTestEvidenceV0(data) &&
		!codexAckIssuesContainEvidenceV0(issues, "failed_test_evidence") {
		issues = append(issues, codexIssueV0(
			CodexConnectorAckArtifactV0,
			"tests",
			spec.CorrelationID,
			"failed_test_evidence",
		))
	}
	return ack, issues
}

func ValidateCodexAgentAckForSpecV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	v := codexAckValidatorV0{correlationID: spec.CorrelationID}
	v.validateShape(ack)
	v.validateCorrelation(ack, spec)
	if strings.TrimSpace(ack.Status) == codexAgentAckStatusCompletedV0 {
		v.validateCompletedEvidence(ack, spec.AgentPacket)
	}
	return v.issues
}

type codexAckValidatorV0 struct {
	correlationID string
	issues        []orquestaruntime.ExternalAgentConnectorErrorV0
}

func (v *codexAckValidatorV0) validateShape(ack CodexAgentAckV0) {
	if ack.SchemaVersion != CodexAgentAckSchemaVersionV0 {
		v.add(CodexConnectorAckInvalidV0, "schema_version", "schema_version_invalid")
	}
	for field, value := range map[string]string{
		"request_id":     ack.RequestID,
		"correlation_id": ack.CorrelationID,
		"ack_ref":        ack.AckRef,
		"target_module":  ack.TargetModule,
		"task_ref":       ack.TaskRef,
		"status":         ack.Status,
	} {
		if strings.TrimSpace(value) == "" {
			v.add(CodexConnectorAckInvalidV0, field, "required")
		}
	}
	if strings.TrimSpace(ack.Status) != codexAgentAckStatusCompletedV0 &&
		strings.TrimSpace(ack.Status) != "failed" {
		v.add(CodexConnectorAckInvalidV0, "status", "status_invalid")
	}
}

func (v *codexAckValidatorV0) validateCorrelation(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) {
	packet := spec.AgentPacket
	if ack.RequestID != spec.RequestID ||
		ack.RequestID != packet.RequestID ||
		ack.CorrelationID != spec.CorrelationID ||
		ack.CorrelationID != packet.CorrelationID ||
		ack.TargetModule != packet.TargetModule ||
		ack.TaskRef != packet.Task.TaskRef ||
		ack.AckRef != packet.DeliveryRefs.AckRef {
		v.add(CodexConnectorAckCorrelationV0, "agent_ack", "correlation_mismatch")
	}
}

func (v *codexAckValidatorV0) validateCompletedEvidence(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) {
	writeSet := normalizeCodexAckPathsV0(packet.Task.WriteSet)
	files := normalizeCodexAckPathsV0(ack.Files)
	if codexAckHasInvalidPathV0(ack.Files) {
		v.add(CodexConnectorAckArtifactV0, "files", "artifact_path_invalid")
		return
	}
	if missing := codexAckMissingWriteSetV0(writeSet, files); len(missing) > 0 {
		v.add(CodexConnectorAckArtifactV0, "files", "missing_required_artifact")
	}
	for _, file := range files {
		if !codexAckPathAllowedV0(file, writeSet) {
			v.add(CodexConnectorAckArtifactV0, "files", "artifact_outside_write_set")
			return
		}
	}
	if codexAckHasFailedTestEvidenceV0(ack) {
		v.add(CodexConnectorAckArtifactV0, "tests", "failed_test_evidence")
		return
	}
	if codexPacketHasRequiredTruncatedContextV0(packet) &&
		!codexAckContainsNotePrefixV0(ack.Notes, "contexto_truncado_resuelto") {
		v.add(CodexConnectorAckArtifactV0, "notes", "required_context_truncated")
		return
	}
	for _, required := range packet.Task.RequiredTests {
		if !codexAckContainsTrimmedV0(ack.Tests, required) {
			v.add(CodexConnectorAckArtifactV0, "tests", "missing_required_test")
			return
		}
	}
}

func codexAckHasInvalidPathV0(values []string) bool {
	for _, value := range values {
		if _, ok := normalizeCodexAckPathV0(value); !ok {
			return true
		}
	}
	return false
}

func normalizeCodexAckPathsV0(values []string) []string {
	paths := make([]string, 0, len(values))
	for _, value := range values {
		path, ok := normalizeCodexAckPathV0(value)
		if ok {
			paths = append(paths, path)
		}
	}
	return paths
}

func normalizeCodexAckPathV0(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" ||
		strings.Contains(trimmed, "://") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.Contains(trimmed, "$HOME") ||
		filepath.IsAbs(trimmed) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func codexAckMissingWriteSetV0(writeSet []string, files []string) []string {
	missing := make([]string, 0)
	for _, required := range writeSet {
		found := false
		for _, file := range files {
			if codexAckPathMatchesWriteSetEntryV0(file, required) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, required)
		}
	}
	return missing
}

func codexAckPathAllowedV0(path string, writeSet []string) bool {
	for _, allowed := range writeSet {
		if codexAckPathMatchesWriteSetEntryV0(path, allowed) {
			return true
		}
	}
	return false
}

func codexAckPathMatchesWriteSetEntryV0(path string, entry string) bool {
	if path == entry || strings.HasPrefix(path, entry+"/") {
		return true
	}
	if strings.HasSuffix(entry, "/**") {
		prefix := strings.TrimSuffix(entry, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if !strings.ContainsAny(entry, "*?[") {
		return false
	}
	matched, err := pathpkg.Match(entry, path)
	return err == nil && matched
}

func codexAckContainsTrimmedV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexAckContainsNotePrefixV0(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == want ||
			strings.HasPrefix(normalized, want+":") ||
			strings.HasPrefix(normalized, want+" ") {
			return true
		}
	}
	return false
}

func codexAckBytesContainForbiddenDetailV0(data []byte) bool {
	normalized := strings.ToLower(strings.ReplaceAll(string(data), `\/`, "/"))
	for _, marker := range []string{
		"/home/",
		"/users/",
		`c:\users\`,
		"$home",
		"~/",
		"access_token",
		"refresh_token",
		"bearer ",
		"oauth",
		"token=",
		"secret=",
		"secreto=",
		"begin private key",
		"transcript",
		"prompt=",
		"completion=",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func (v *codexAckValidatorV0) add(code CodexConnectorIssueCodeV0, field string, evidence string) {
	v.issues = append(v.issues, codexIssueV0(code, field, v.correlationID, evidence))
}

func codexAckIssuesContainEvidenceV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
	evidence string,
) bool {
	for _, issue := range issues {
		for _, item := range issue.Evidence {
			if item == evidence {
				return true
			}
		}
	}
	return false
}
