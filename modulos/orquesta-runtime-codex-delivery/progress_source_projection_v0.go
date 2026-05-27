package orquestaruntimecodexdelivery

import (
	"fmt"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func codexProgressObservationV0(
	descriptor CodexReceiptDescriptorV0,
	report orquestaruntime.AgentProgressReportV0,
	decisionRequired bool,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	task := descriptor.Spec.AgentPacket.Task
	return orquestacionnucleoapp.AgentProgressObservationV0{
		CandidateRef:     "progress-candidate-ref-" + report.ReportID,
		Report:           report,
		PhaseID:          descriptor.Spec.AgentPacket.Phase,
		TaskRef:          strings.TrimSpace(task.TaskRef),
		DecisionRequired: decisionRequired,
		EvidenceRefs:     compactCodexDeliveryRefsV0(report.EvidenceRefs),
	}
}

func codexProgressObservationForCurrentRunPhaseV0(
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	currentPhase := strings.TrimSpace(string(request.Run.CurrentPhase))
	if currentPhase != "" {
		observation.PhaseID = currentPhase
	}
	return observation
}

func codexProgressReportDecisionRequiredV0(report orquestaruntime.AgentProgressReportV0) bool {
	if report.DecisionRequired {
		return true
	}
	return report.Status == orquestaruntime.AgentLoopDetectedV0 ||
		report.Status == orquestaruntime.AgentStoppedV0
}

func codexProgressReportWithRepeatedActionEvidenceV0(
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	if report.RepeatedActionCount <= 0 {
		return report
	}
	report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
		report.EvidenceRefs,
		"evidence-ref-repeated-action",
	))
	return report
}

func codexProgressEvidenceRefsV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) []string {
	return compactCodexDeliveryRefsV0([]string{
		"progress-evidence-ref-" + strings.TrimSpace(record.AgentRequestID),
		strings.TrimSpace(record.SessionRef),
	})
}

func codexProgressReportIDV0(
	current orquestaruntime.AgentProgressHeartbeatV0,
) string {
	return fmt.Sprintf("agent-progress-report-ref-%s-%06d", current.AgentRequestID, current.TickCounter)
}
