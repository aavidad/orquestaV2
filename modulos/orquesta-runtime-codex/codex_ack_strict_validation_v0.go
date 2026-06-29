package orquestaruntimecodex

import (
	"encoding/json"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func ReadAndValidateStrictCompletedCodexAgentAckFileV0(
	path string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	data, err := ReadCodexControlFileBytesV0(path, CodexAgentAckFileNameV0)
	if err != nil {
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			CodexControlFileReadIssueV0(err, CodexAgentAckFileNameV0, spec.CorrelationID, "ack_not_ready"),
		}
	}
	return ValidateStrictCompletedCodexAgentAckBytesForSpecV0(data, spec)
}

func ValidateStrictCompletedCodexAgentAckBytesForSpecV0(
	data []byte,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	var ack CodexAgentAckV0
	if err := json.Unmarshal(data, &ack); err != nil {
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckInvalidV0, CodexAgentAckFileNameV0, spec.CorrelationID, "json_invalid"),
		}
	}
	if codexStrictAckHasRecoverableCompletionEvidenceV0(ack) {
		ack = codexAgentAckWithMissingSpecDefaultsV0(ack, spec)
	}
	issues := codexStrictCompletedAckIssuesV0(ack, spec)
	issues = append(issues, codexStrictCompletedAckRawTestReceiptIssuesV0(data, spec.CorrelationID)...)
	if len(issues) > 0 {
		return ack, issues
	}
	validated, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	return validated, issues
}

func codexStrictAckHasRecoverableCompletionEvidenceV0(ack CodexAgentAckV0) bool {
	return ack.Files != nil || codexAckHasNonFileCompletionEvidenceV0(ack)
}

func CodexAgentPacketRequiresStrictTerminalAckV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	return codexAckStringInSetV0(packet.Policies, "ack_terminal_strict") ||
		codexAckStringInSetV0(packet.Policies, "orquestav2_strict")
}

func codexStrictCompletedAckIssuesV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	v := codexAckValidatorV0{correlationID: spec.CorrelationID}
	if !codexAgentAckSchemaVersionCompatibleV0(ack.SchemaVersion) {
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
	if !codexAgentAckStatusCarriesReviewableWorkV0(ack.Status) {
		v.add(CodexConnectorAckInvalidV0, "status", "status_not_completed")
	}
	if ack.Files == nil && !codexAckHasNonFileCompletionEvidenceV0(ack) {
		v.add(CodexConnectorAckArtifactV0, "files", "required")
	}
	if ack.Tests == nil {
		v.add(CodexConnectorAckArtifactV0, "tests", "required")
	}
	v.validateStrictFiles(ack, spec.AgentPacket)
	v.validateStrictTests(ack, spec.AgentPacket)
	v.validateStrictTestReceipts(ack, spec.AgentPacket)
	if !codexAgentAckCorrelationMatchesSpecV0(ack, spec) {
		evidence := "correlation_mismatch"
		if codexAgentAckParentChildTaskCollisionV0(ack, spec) {
			evidence = CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0
		} else if codexAgentAckParentSubroleCollisionV0(ack, spec) {
			evidence = CodexAgentAckInvalidParentSubroleCollisionEvidenceV0
		}
		v.add(CodexConnectorAckCorrelationV0, "agent_ack", evidence)
	}
	return v.issues
}

func (v *codexAckValidatorV0) validateStrictFiles(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) {
	if ack.Files == nil {
		return
	}
	if codexAckHasInvalidPathV0(ack.Files) {
		v.add(CodexConnectorAckArtifactV0, "files", "artifact_path_invalid")
		return
	}
	files := normalizeCodexAckPathsV0(ack.Files)
	if len(files) == 0 {
		if codexAckHasNonFileCompletionEvidenceV0(ack) {
			return
		}
		v.add(CodexConnectorAckArtifactV0, "files", "required")
		return
	}
	if codexAckHasForbiddenArtifactPathV0(files) {
		v.add(CodexConnectorAckArtifactV0, "files", codexAckForbiddenArtifactEvidenceV0(files))
		return
	}
	writeSet := normalizeCodexAckWriteSetPathsV0(packet.Task.WriteSet)
	if len(writeSet) == 0 {
		v.add(CodexConnectorAckArtifactV0, "write_set", "required")
		return
	}
	// Write-set drift is evaluated by review/worktree gates as a soft rail.
	// Strict ACK validation only owns receipt shape, identity, safe paths and
	// required test evidence, so useful completed work can reach review.
}

func codexAckHasNonFileCompletionEvidenceV0(ack CodexAgentAckV0) bool {
	return len(compactCodexAckStringsV0(ack.Tests)) > 0 ||
		len(compactCodexAckStringsV0(ack.Notes)) > 0 ||
		len(ack.TestReceipts) > 0
}

func (v *codexAckValidatorV0) validateStrictTests(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) {
	if ack.Tests == nil {
		return
	}
	required := compactCodexAckStringsV0(packet.Task.RequiredTests)
	if len(required) == 0 {
		return
	}
	tests := compactCodexAckStringsV0(ack.Tests)
	for _, command := range required {
		if !codexTestCommandInSetV0(tests, command) {
			v.add(CodexConnectorAckArtifactV0, "tests", "required_tests_mismatch")
			return
		}
	}
}

func codexAckStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func compactCodexAckStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
