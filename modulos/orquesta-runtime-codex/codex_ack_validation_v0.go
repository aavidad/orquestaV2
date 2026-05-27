package orquestaruntimecodex

import (
	"encoding/json"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const codexAgentAckStatusCompletedV0 = "completed"

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
	ack = codexAgentAckWithSpecDefaultsV0(ack, spec)
	v := codexAckValidatorV0{correlationID: spec.CorrelationID}
	v.validateShape(ack)
	v.validateCorrelation(ack, spec)
	v.validateSensitiveDetails(ack)
	if strings.TrimSpace(ack.Status) == codexAgentAckStatusCompletedV0 {
		v.validateCompletedEvidence(ack, spec.AgentPacket)
	}
	return v.issues
}

func codexAgentAckWithSpecDefaultsV0(
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
	if codexAgentAckIdentityMatchesSpecV0(ack, spec) {
		if strings.TrimSpace(ack.CorrelationID) != strings.TrimSpace(packet.CorrelationID) {
			ack.CorrelationID = packet.CorrelationID
		}
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

func (v *codexAckValidatorV0) validateSensitiveDetails(ack CodexAgentAckV0) {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return
	}
	if codexAckContainsSensitiveDetailV0(ack) || codexAckContainsLocalProductDetailV0(ack) {
		v.add(CodexConnectorAckForbiddenV0, "agent_ack", "forbidden_sensitive_detail")
	}
}

func (v *codexAckValidatorV0) validateCompletedEvidence(
	ack CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) {
	writeSet := normalizeCodexAckWriteSetPathsV0(packet.Task.WriteSet)
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
		if len(writeSet) > 0 && len(files) == 0 && len(ack.Tests) == 0 && len(ack.Notes) == 0 {
			v.add(CodexConnectorAckArtifactV0, "files", "missing_required_artifact")
		}
	}
	if codexPacketHasRequiredTruncatedContextV0(packet) &&
		!codexAckContainsNotePrefixV0(ack.Notes, "contexto_truncado_resuelto") {
		v.add(CodexConnectorAckArtifactV0, "notes", "required_context_truncated")
		return
	}
	if codexPacketRequiresRefOnlyAckEvidenceV0(packet) &&
		!codexAckContainsNotePrefixV0(ack.Notes, "contexto_ref_only_resuelto") {
		v.add(CodexConnectorAckArtifactV0, "notes", "required_context_ref_only")
		return
	}
	if codexAckHasFailedTestEvidenceV0(ack) {
		v.add(CodexConnectorAckArtifactV0, "tests", "failed_test_evidence")
		return
	}
	for _, required := range packet.Task.RequiredTests {
		if ack.Tests == nil {
			v.add(CodexConnectorAckArtifactV0, "tests", "missing_required_test")
			return
		}
		if !codexAckContainsTrimmedV0(ack.Tests, required) {
			v.add(CodexConnectorAckArtifactV0, "tests", "missing_required_test")
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
