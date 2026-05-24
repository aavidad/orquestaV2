package orquestaruntimecodex

import (
	"encoding/json"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

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
	if issues := codexStrictCompletedAckIssuesV0(ack, spec); len(issues) > 0 {
		return ack, issues
	}
	validated, issues := ValidateCodexAgentAckBytesForSpecV0(data, spec)
	return validated, issues
}

func codexStrictCompletedAckIssuesV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	v := codexAckValidatorV0{correlationID: spec.CorrelationID}
	if strings.TrimSpace(ack.SchemaVersion) != CodexAgentAckSchemaVersionV0 {
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
	if strings.TrimSpace(ack.Status) != codexAgentAckStatusCompletedV0 {
		v.add(CodexConnectorAckInvalidV0, "status", "status_not_completed")
	}
	packet := spec.AgentPacket
	if strings.TrimSpace(ack.RequestID) != strings.TrimSpace(spec.RequestID) ||
		strings.TrimSpace(ack.RequestID) != strings.TrimSpace(packet.RequestID) ||
		strings.TrimSpace(ack.CorrelationID) != strings.TrimSpace(spec.CorrelationID) ||
		strings.TrimSpace(ack.CorrelationID) != strings.TrimSpace(packet.CorrelationID) ||
		strings.TrimSpace(ack.TargetModule) != strings.TrimSpace(packet.TargetModule) ||
		strings.TrimSpace(ack.TaskRef) != strings.TrimSpace(packet.Task.TaskRef) ||
		strings.TrimSpace(ack.AckRef) != strings.TrimSpace(packet.DeliveryRefs.AckRef) {
		v.add(CodexConnectorAckCorrelationV0, "agent_ack", "correlation_mismatch")
	}
	return v.issues
}
