package orquestaruntimecodex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	CodexShutdownRequestSchemaVersionV0       = "codex_shutdown_request.v0"
	CodexShutdownCheckpointAckSchemaVersionV0 = "codex_shutdown_checkpoint_ack.v0"
	CodexShutdownCheckpointStatusReadyV0      = "checkpoint_ready"
)

type CodexShutdownRequestV0 struct {
	SchemaVersion      string   `json:"schema_version"`
	RunRef             string   `json:"run_ref"`
	AgentRef           string   `json:"agent_ref"`
	CorrelationID      string   `json:"correlation_id,omitempty"`
	RequestedBy        string   `json:"requested_by,omitempty"`
	Reason             string   `json:"reason,omitempty"`
	ShutdownAttemptRef string   `json:"shutdown_attempt_ref"`
	CheckpointRef      string   `json:"checkpoint_ref"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type CodexShutdownCheckpointAckV0 struct {
	SchemaVersion      string   `json:"schema_version"`
	RunRef             string   `json:"run_ref"`
	AgentRef           string   `json:"agent_ref"`
	ShutdownAttemptRef string   `json:"shutdown_attempt_ref"`
	CheckpointRef      string   `json:"checkpoint_ref"`
	Status             string   `json:"status"`
	Summary            string   `json:"summary,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

func WriteCodexShutdownRequestFileV0(
	path string,
	request CodexShutdownRequestV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	path = strings.TrimSpace(path)
	if path == "" || filepath.Base(path) != CodexShutdownRequestFileNameV0 {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorPathInvalidV0, CodexShutdownRequestFileNameV0, request.CorrelationID, "path_invalid"),
		}
	}
	request = normalizeCodexShutdownRequestV0(request)
	if issues := ValidateCodexShutdownRequestV0(request); len(issues) > 0 {
		return issues
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexShutdownRequestFileNameV0, request.CorrelationID, "mkdir_failed"),
		}
	}
	if err := writeJSONFileV0(filepath.Dir(path), path, request); err != nil {
		return []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorFilesystemV0, CodexShutdownRequestFileNameV0, request.CorrelationID, "write_failed"),
		}
	}
	return nil
}

func ReadCodexShutdownCheckpointAckFileV0(
	path string,
	request CodexShutdownRequestV0,
) (CodexShutdownCheckpointAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	data, err := ReadCodexControlFileBytesV0(strings.TrimSpace(path), CodexShutdownCheckpointAckFileNameV0)
	if err != nil {
		return CodexShutdownCheckpointAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			CodexControlFileReadIssueV0(
				err,
				CodexShutdownCheckpointAckFileNameV0,
				request.CorrelationID,
				"checkpoint_ack_not_ready",
			),
		}
	}
	if codexShutdownBytesForbiddenV0(data) {
		return CodexShutdownCheckpointAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckForbiddenV0, CodexShutdownCheckpointAckFileNameV0, request.CorrelationID, "forbidden_detail"),
		}
	}
	var ack CodexShutdownCheckpointAckV0
	if err := json.Unmarshal(data, &ack); err != nil {
		return CodexShutdownCheckpointAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckInvalidV0, CodexShutdownCheckpointAckFileNameV0, request.CorrelationID, "json_invalid"),
		}
	}
	return ack, ValidateCodexShutdownCheckpointAckV0(ack, request)
}

func ValidateCodexShutdownRequestV0(
	request CodexShutdownRequestV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	request = normalizeCodexShutdownRequestV0(request)
	v := codexShutdownValidatorV0{correlationID: request.CorrelationID}
	if request.SchemaVersion != CodexShutdownRequestSchemaVersionV0 {
		v.add(CodexConnectorAckInvalidV0, "schema_version", "schema_version_invalid")
	}
	v.required("run_ref", request.RunRef)
	v.required("agent_ref", request.AgentRef)
	v.required("shutdown_attempt_ref", request.ShutdownAttemptRef)
	v.required("checkpoint_ref", request.CheckpointRef)
	v.noForbidden("requested_by", request.RequestedBy)
	v.noForbidden("reason", request.Reason)
	v.noForbiddenList("evidence_refs", request.EvidenceRefs)
	return v.issues
}

func ValidateCodexShutdownCheckpointAckV0(
	ack CodexShutdownCheckpointAckV0,
	request CodexShutdownRequestV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	ack = normalizeCodexShutdownCheckpointAckV0(ack)
	request = normalizeCodexShutdownRequestV0(request)
	v := codexShutdownValidatorV0{correlationID: request.CorrelationID}
	if ack.SchemaVersion != CodexShutdownCheckpointAckSchemaVersionV0 {
		v.add(CodexConnectorAckInvalidV0, "schema_version", "schema_version_invalid")
	}
	v.required("run_ref", ack.RunRef)
	v.required("agent_ref", ack.AgentRef)
	v.required("checkpoint_ref", ack.CheckpointRef)
	if strings.TrimSpace(ack.Status) != CodexShutdownCheckpointStatusReadyV0 {
		v.add(CodexConnectorAckInvalidV0, "status", "status_invalid")
	}
	if strings.TrimSpace(ack.RunRef) != strings.TrimSpace(request.RunRef) ||
		strings.TrimSpace(ack.AgentRef) != strings.TrimSpace(request.AgentRef) ||
		strings.TrimSpace(ack.CheckpointRef) != strings.TrimSpace(request.CheckpointRef) {
		v.add(CodexConnectorAckCorrelationV0, "shutdown_checkpoint_ack", "correlation_mismatch")
	}
	if strings.TrimSpace(request.ShutdownAttemptRef) != "" &&
		strings.TrimSpace(ack.ShutdownAttemptRef) != strings.TrimSpace(request.ShutdownAttemptRef) {
		v.add(CodexConnectorAckCorrelationV0, "shutdown_attempt_ref", "shutdown_checkpoint_attempt_mismatch")
	}
	v.noForbidden("summary", ack.Summary)
	v.noForbiddenList("evidence_refs", ack.EvidenceRefs)
	return v.issues
}

func normalizeCodexShutdownRequestV0(request CodexShutdownRequestV0) CodexShutdownRequestV0 {
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.AgentRef = strings.TrimSpace(request.AgentRef)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Reason = strings.TrimSpace(request.Reason)
	request.ShutdownAttemptRef = strings.TrimSpace(request.ShutdownAttemptRef)
	request.CheckpointRef = strings.TrimSpace(request.CheckpointRef)
	request.EvidenceRefs = compactCodexIssueEvidenceV0(request.EvidenceRefs)
	return request
}

func normalizeCodexShutdownCheckpointAckV0(
	ack CodexShutdownCheckpointAckV0,
) CodexShutdownCheckpointAckV0 {
	ack.SchemaVersion = strings.TrimSpace(ack.SchemaVersion)
	ack.RunRef = strings.TrimSpace(ack.RunRef)
	ack.AgentRef = strings.TrimSpace(ack.AgentRef)
	ack.ShutdownAttemptRef = strings.TrimSpace(ack.ShutdownAttemptRef)
	ack.CheckpointRef = strings.TrimSpace(ack.CheckpointRef)
	ack.Status = strings.TrimSpace(ack.Status)
	ack.Summary = strings.TrimSpace(ack.Summary)
	ack.EvidenceRefs = compactCodexIssueEvidenceV0(ack.EvidenceRefs)
	return ack
}

type codexShutdownValidatorV0 struct {
	correlationID string
	issues        []orquestaruntime.ExternalAgentConnectorErrorV0
}

func (v *codexShutdownValidatorV0) required(field string, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(CodexConnectorAckInvalidV0, field, "required")
		return
	}
	v.noForbidden(field, value)
}

func (v *codexShutdownValidatorV0) noForbidden(field string, value string) {
	if codexShutdownValueForbiddenV0(value) {
		v.add(CodexConnectorAckForbiddenV0, field, "forbidden_detail")
	}
}

func (v *codexShutdownValidatorV0) noForbiddenList(field string, values []string) {
	for _, value := range values {
		v.noForbidden(field, value)
	}
}

func (v *codexShutdownValidatorV0) add(
	code CodexConnectorIssueCodeV0,
	field string,
	evidence string,
) {
	v.issues = append(v.issues, codexIssueV0(code, field, v.correlationID, evidence))
}

func codexShutdownBytesForbiddenV0(data []byte) bool {
	return codexShutdownValueForbiddenV0(string(data))
}

func codexShutdownValueForbiddenV0(value string) bool {
	if codexHasControlCharsV0(value) {
		return true
	}
	return codexTextContainsSensitiveDetailIgnoringRailEnvV0(value)
}
