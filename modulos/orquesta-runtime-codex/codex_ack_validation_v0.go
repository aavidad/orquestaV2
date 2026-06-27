package orquestaruntimecodex

import (
	"encoding/json"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	codexAgentAckStatusCompletedV0 = "completed"
	codexAgentAckStatusBlockedV0   = "blocked"

	CodexAgentAckInvalidParentSubroleCollisionEvidenceV0   = "invalid_parent_ack_subrole_collision"
	CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0 = "invalid_parent_ack_child_task_collision"
)

func ReadAndValidateCodexAgentAckFileV0(
	path string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	data, err := ReadCodexControlFileBytesV0(path, CodexAgentAckFileNameV0)
	if err != nil {
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			CodexControlFileReadIssueV0(err, CodexAgentAckFileNameV0, spec.CorrelationID, "ack_not_ready"),
		}
	}
	return ValidateCodexAgentAckBytesForSpecV0(data, spec)
}

func ValidateCodexAgentAckBytesForSpecV0(
	data []byte,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) (CodexAgentAckV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	var ack CodexAgentAckV0
	if err := json.Unmarshal(data, &ack); err != nil {
		return CodexAgentAckV0{}, []orquestaruntime.ExternalAgentConnectorErrorV0{
			codexIssueV0(CodexConnectorAckInvalidV0, CodexAgentAckFileNameV0, spec.CorrelationID, "json_invalid"),
		}
	}
	ack = codexAgentAckWithSpecDefaultsV0(ack, spec)
	issues := ValidateCodexAgentAckForSpecV0(ack, spec)
	if strings.TrimSpace(ack.Status) == codexAgentAckStatusCompletedV0 &&
		codexAckBytesHaveFailedTestEvidenceV0(data) &&
		!codexAckIssuesContainEvidenceV0(issues, "failed_test_evidence") {
		return ack, issues
	}
	return ack, issues
}

func ValidateCodexAgentAckForSpecV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestaruntime.ExternalAgentConnectorErrorV0 {
	ack = codexAgentAckWithSpecDefaultsV0(ack, spec)
	v := codexAckValidatorV0{correlationID: spec.CorrelationID}
	v.validateShape(ack)
	v.validateCorrelation(ack, spec)
	v.validateSensitiveDetails(ack)
	if codexAgentAckStatusCarriesReviewableWorkV0(ack.Status) {
		v.validateCompletedEvidence(ack, spec.AgentPacket)
	}
	return v.issues
}

func codexAgentAckWithSpecDefaultsV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) CodexAgentAckV0 {
	ack = codexAgentAckWithMissingSpecDefaultsV0(ack, spec)
	packet := spec.AgentPacket
	if codexAgentAckIdentityMatchesSpecV0(ack, spec) {
		if strings.TrimSpace(ack.CorrelationID) != strings.TrimSpace(packet.CorrelationID) {
			ack.CorrelationID = packet.CorrelationID
		}
		if !codexAgentAckParentTaskCollisionV0(ack, spec) {
			ack.TaskRef = packet.Task.TaskRef
		}
	}
	return ack
}

func codexAgentAckWithMissingSpecDefaultsV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) CodexAgentAckV0 {
	packet := spec.AgentPacket
	if strings.TrimSpace(ack.RequestID) == "" {
		ack.RequestID = firstNonEmptyCodexAckValueV0(spec.RequestID, packet.RequestID)
	}
	if strings.TrimSpace(ack.CorrelationID) == "" {
		ack.CorrelationID = firstNonEmptyCodexAckValueV0(spec.CorrelationID, packet.CorrelationID)
	}
	if strings.TrimSpace(ack.AckRef) == "" {
		ack.AckRef = packet.DeliveryRefs.AckRef
	}
	if strings.TrimSpace(ack.TargetModule) == "" {
		ack.TargetModule = packet.TargetModule
	}
	if strings.TrimSpace(ack.TaskRef) == "" {
		ack.TaskRef = packet.Task.TaskRef
	}
	return ack
}

func codexAgentAckIdentityMatchesSpecV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) bool {
	packet := spec.AgentPacket
	return strings.TrimSpace(ack.RequestID) == strings.TrimSpace(spec.RequestID) &&
		strings.TrimSpace(ack.RequestID) == strings.TrimSpace(packet.RequestID) &&
		strings.TrimSpace(ack.TargetModule) == strings.TrimSpace(packet.TargetModule) &&
		strings.TrimSpace(ack.AckRef) == strings.TrimSpace(packet.DeliveryRefs.AckRef)
}

func firstNonEmptyCodexAckValueV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type codexAckValidatorV0 struct {
	correlationID string
	issues        []orquestaruntime.ExternalAgentConnectorErrorV0
}

func (v *codexAckValidatorV0) validateShape(ack CodexAgentAckV0) {
	if !codexSchemaVersionCompatibleV0(ack.SchemaVersion, CodexAgentAckSchemaVersionV0) {
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
		strings.TrimSpace(ack.Status) != codexAgentAckStatusBlockedV0 &&
		strings.TrimSpace(ack.Status) != "failed" {
		v.add(CodexConnectorAckInvalidV0, "status", "status_invalid")
	}
}

func codexAgentAckStatusCarriesReviewableWorkV0(status string) bool {
	switch strings.TrimSpace(status) {
	case codexAgentAckStatusCompletedV0, codexAgentAckStatusBlockedV0:
		return true
	default:
		return false
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
		evidence := "correlation_mismatch"
		if codexAgentAckParentChildTaskCollisionV0(ack, spec) {
			evidence = CodexAgentAckInvalidParentChildTaskCollisionEvidenceV0
		} else if codexAgentAckParentSubroleCollisionV0(ack, spec) {
			evidence = CodexAgentAckInvalidParentSubroleCollisionEvidenceV0
		}
		v.add(CodexConnectorAckCorrelationV0, "agent_ack", evidence)
	}
}

func codexAgentAckParentTaskCollisionV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) bool {
	return codexAgentAckParentChildTaskCollisionV0(ack, spec) ||
		codexAgentAckParentSubroleCollisionV0(ack, spec)
}

func codexAgentAckParentSubroleCollisionV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) bool {
	taskRef := strings.TrimSpace(ack.TaskRef)
	return codexAgentAckParentTaskRefMismatchWithMatchingIdentityV0(ack, spec) &&
		codexAgentAckTaskRefLooksSubroleV0(taskRef)
}

func codexAgentAckParentChildTaskCollisionV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) bool {
	if !codexAgentAckParentTaskRefMismatchWithMatchingIdentityV0(ack, spec) {
		return false
	}
	taskRef := strings.TrimSpace(ack.TaskRef)
	for _, childTaskRef := range spec.AgentPacket.Task.ChildTaskRefs {
		if taskRef == strings.TrimSpace(childTaskRef) {
			return true
		}
	}
	return false
}

func codexAgentAckParentTaskRefMismatchWithMatchingIdentityV0(
	ack CodexAgentAckV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) bool {
	packet := spec.AgentPacket
	taskRef := strings.TrimSpace(ack.TaskRef)
	parentTaskRef := strings.TrimSpace(packet.Task.TaskRef)
	return strings.TrimSpace(ack.RequestID) == strings.TrimSpace(spec.RequestID) &&
		strings.TrimSpace(ack.RequestID) == strings.TrimSpace(packet.RequestID) &&
		strings.TrimSpace(ack.AckRef) == strings.TrimSpace(packet.DeliveryRefs.AckRef) &&
		strings.TrimSpace(ack.TargetModule) == strings.TrimSpace(packet.TargetModule) &&
		taskRef != "" &&
		parentTaskRef != "" &&
		taskRef != parentTaskRef
}

func codexAgentAckTaskRefLooksSubroleV0(taskRef string) bool {
	normalized := strings.ToLower(strings.TrimSpace(taskRef))
	return strings.Contains(normalized, "subrole") ||
		strings.Contains(normalized, "subrol")
}

func (v *codexAckValidatorV0) validateSensitiveDetails(ack CodexAgentAckV0) {
	// Operational/local-detail markers are advisory. The director/orchestrator
	// decides whether to ignore, remove or turn them into improvement work.
	// Effective secrets are different: they are a hard security boundary even
	// while legacy detail rails are offline.
	if codexAckContainsSensitiveDetailV0(ack) {
		v.add(CodexConnectorAckForbiddenV0, "agent_ack", "forbidden_sensitive_detail")
	}
}

func (v *codexAckValidatorV0) validateCompletedEvidence(
	ack CodexAgentAckV0,
	_ orquestaruntime.AgentStartPacketV0,
) {
	files := normalizeCodexAckPathsV0(ack.Files)
	if ack.Files != nil {
		if codexAckHasInvalidPathV0(ack.Files) {
			v.add(CodexConnectorAckArtifactV0, "files", "artifact_path_invalid")
			return
		}
		if codexAckHasForbiddenArtifactPathV0(files) {
			v.add(CodexConnectorAckArtifactV0, "files", codexAckForbiddenArtifactEvidenceV0(files))
			return
		}
	}
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
