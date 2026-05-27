package orquestaruntimecodexdelivery

import (
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexProgressFailedAckObservationV0(
	descriptor CodexReceiptDescriptorV0,
) (orquestacionnucleoapp.AgentProgressObservationV0, bool, error) {
	data, err := orquestaruntimecodex.ReadCodexControlFileBytesV0(
		strings.TrimSpace(descriptor.AckPath),
		orquestaruntimecodex.CodexAgentAckFileNameV0,
	)
	if err != nil {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	ack, issues := orquestaruntimecodex.ValidateCodexAgentAckBytesForSpecV0(data, descriptor.Spec)
	if len(issues) > 0 || strings.TrimSpace(ack.Status) != "failed" {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	packet := descriptor.Spec.AgentPacket
	agentRef := firstCodexProgressFailedAckValueV0(
		ack.RequestID,
		descriptor.AgentRef,
		descriptor.Spec.RequestID,
		packet.RequestID,
	)
	taskRef := firstCodexProgressFailedAckValueV0(ack.TaskRef, packet.Task.TaskRef)
	reportID := "agent-progress-report-ref-" + codexProgressFailedAckSafeRefV0(ack.AckRef) + "-failed-ack"
	report := orquestaruntime.AgentProgressReportV0{
		ReportID:         reportID,
		RunID:            strings.TrimSpace(descriptor.RunID),
		AgentRequestID:   agentRef,
		Status:           orquestaruntime.AgentStoppedV0,
		DecisionRequired: true,
		Summary:          "ACK terminal fallido registrado por agente; cerrar agente logico y replanificar.",
		EvidenceRefs: codexProgressFailedAckEvidenceRefsV0(
			ack,
			packet,
		),
	}
	observation := orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + reportID,
		Report:           report,
		PhaseID:          packet.Phase,
		TaskRef:          taskRef,
		AssessmentRef:    "assessment-ref-" + reportID,
		DecisionRequired: true,
		EvidenceRefs:     report.EvidenceRefs,
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(report); len(issues) > 0 {
		return orquestacionnucleoapp.AgentProgressObservationV0{}, false, nil
	}
	return observation, true, nil
}

func codexProgressFailedAckEvidenceRefsV0(
	ack orquestaruntimecodex.CodexAgentAckV0,
	packet orquestaruntime.AgentStartPacketV0,
) []string {
	refs := []string{
		ack.AckRef,
		packet.DeliveryRefs.MailboxRef,
		packet.DeliveryRefs.ReadinessRef,
		packet.DeliveryRefs.CheckpointRef,
		"evidence-ref-codex-ack-failed",
	}
	refs = append(refs, orquestaruntimecodex.CodexRequiredTestReceiptEvidenceRefsV0(ack.TestReceipts)...)
	return compactCodexDeliveryRefsV0(refs)
}

func firstCodexProgressFailedAckValueV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func codexProgressFailedAckSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", " ", "-", "#", "-", ":", "-", ".", "-")
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "ack-ref-failed"
	}
	if len(value) > 96 {
		value = strings.Trim(value[:96], "-")
	}
	if value == "" {
		return "ack-ref-failed"
	}
	return value
}
