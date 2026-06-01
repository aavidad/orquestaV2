package orquestaruntime

import "fmt"

type AgentProgressHeartbeatPolicyV0 struct {
	StalledAfterNoProgressTicks int `json:"stalled_after_no_progress_ticks"`
	LoopAfterRepeatedActions    int `json:"loop_after_repeated_actions"`
}

type AgentProgressHeartbeatV0 struct {
	HeartbeatRef        string   `json:"heartbeat_ref"`
	RunID               string   `json:"run_id"`
	AgentRequestID      string   `json:"agent_request_id"`
	ProcessRef          string   `json:"process_ref"`
	TickCounter         int      `json:"tick_counter"`
	ProgressCounter     int      `json:"progress_counter"`
	RepeatedActionCount int      `json:"repeated_action_count"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

func BuildAgentProgressReportFromHeartbeatV0(
	reportID string,
	snapshot ProcessRuntimeSnapshotV0,
	previous *AgentProgressHeartbeatV0,
	current AgentProgressHeartbeatV0,
	policy AgentProgressHeartbeatPolicyV0,
) (AgentProgressReportV0, []AgentProgressReportErrorV0) {
	if issues := validateAgentProgressHeartbeatInputV0(reportID, snapshot, previous, current); len(issues) > 0 {
		return AgentProgressReportV0{}, issues
	}

	policy = normalizeAgentProgressHeartbeatPolicyV0(policy)
	noProgressTicks := agentProgressNoProgressTicksV0(previous, current)
	status := agentProgressStatusFromHeartbeatV0(snapshot.Status, noProgressTicks, current.RepeatedActionCount, policy)
	report := AgentProgressReportV0{
		ReportID:            reportID,
		RunID:               current.RunID,
		AgentRequestID:      current.AgentRequestID,
		Status:              status,
		NoProgressTicks:     noProgressTicks,
		RepeatedActionCount: current.RepeatedActionCount,
		Summary:             agentProgressHeartbeatSummaryV0(status),
		EvidenceRefs:        agentProgressHeartbeatEvidenceRefsV0(snapshot, current),
	}
	if issues := ValidateAgentProgressReportV0(report); len(issues) > 0 {
		return AgentProgressReportV0{}, issues
	}
	return report, nil
}

func validateAgentProgressHeartbeatInputV0(
	reportID string,
	snapshot ProcessRuntimeSnapshotV0,
	previous *AgentProgressHeartbeatV0,
	current AgentProgressHeartbeatV0,
) []AgentProgressReportErrorV0 {
	v := agentProgressHeartbeatValidatorV0{}
	v.requireOpaque("report_id", reportID)
	v.requireOpaque("heartbeat_ref", current.HeartbeatRef)
	v.requireOpaque("run_id", current.RunID)
	v.requireOpaque("agent_request_id", current.AgentRequestID)
	v.requireOpaque("process_ref", current.ProcessRef)
	v.requireNonNegative("tick_counter", current.TickCounter)
	v.requireNonNegative("progress_counter", current.ProgressCounter)
	v.requireNonNegative("repeated_action_count", current.RepeatedActionCount)
	for i, ref := range current.EvidenceRefs {
		v.requireOpaque(fmt.Sprintf("evidence_refs[%d]", i), ref)
	}
	if snapshot.SchemaVersion != ProcessRuntimeConnectorVersionV0 {
		v.add(AgentProgressReportInvalidoV0, "snapshot.schema_version")
	}
	if snapshot.ProcessRef != current.ProcessRef {
		v.add(AgentProgressReportInvalidoV0, "snapshot.process_ref")
	}
	if !isOneOf(string(snapshot.Status), string(ProcessRuntimeRunningV0), string(ProcessRuntimeStoppingV0), string(ProcessRuntimeStoppedV0)) {
		v.add(AgentProgressReportInvalidoV0, "snapshot.status")
	}
	if previous != nil {
		if previous.RunID != current.RunID {
			v.add(AgentProgressReportInvalidoV0, "previous.run_id")
		}
		if previous.AgentRequestID != current.AgentRequestID {
			v.add(AgentProgressReportInvalidoV0, "previous.agent_request_id")
		}
		if previous.ProcessRef != current.ProcessRef {
			v.add(AgentProgressReportInvalidoV0, "previous.process_ref")
		}
		if current.TickCounter <= previous.TickCounter {
			v.add(AgentProgressReportInvalidoV0, "tick_counter")
		}
		if current.ProgressCounter < previous.ProgressCounter {
			v.add(AgentProgressReportInvalidoV0, "progress_counter")
		}
	}
	return v.errors
}

type agentProgressHeartbeatValidatorV0 struct {
	errors []AgentProgressReportErrorV0
}

func (v *agentProgressHeartbeatValidatorV0) requireOpaque(field, value string) {
	if value == "" {
		v.add(AgentProgressReportInvalidoV0, field)
		return
	}
	if looksLikeSecret(value) {
		v.add(AgentProgressSecretoDetectadoV0, field)
		return
	}
	if !opaqueRefPatternV0.MatchString(value) {
		v.add(AgentProgressReferenciaNoOpacaV0, field)
	}
}

func (v *agentProgressHeartbeatValidatorV0) requireNonNegative(field string, value int) {
	if value < 0 {
		v.add(AgentProgressReportInvalidoV0, field)
	}
}

func (v *agentProgressHeartbeatValidatorV0) add(code AgentProgressReportErrorCodeV0, field string) {
	v.errors = append(v.errors, AgentProgressReportErrorV0{
		Code:       code,
		MessageKey: "orquesta.runtime.agent_progress_report." + string(code),
		Field:      field,
		Retryable:  false,
	})
}

func normalizeAgentProgressHeartbeatPolicyV0(
	policy AgentProgressHeartbeatPolicyV0,
) AgentProgressHeartbeatPolicyV0 {
	if policy.StalledAfterNoProgressTicks < 1 {
		policy.StalledAfterNoProgressTicks = 1
	}
	if policy.LoopAfterRepeatedActions < 1 {
		policy.LoopAfterRepeatedActions = 2
	}
	return policy
}

func agentProgressNoProgressTicksV0(
	previous *AgentProgressHeartbeatV0,
	current AgentProgressHeartbeatV0,
) int {
	if previous == nil || current.ProgressCounter > previous.ProgressCounter {
		return 0
	}
	return current.TickCounter - previous.TickCounter
}

func agentProgressStatusFromHeartbeatV0(
	processStatus ProcessRuntimeStatusV0,
	noProgressTicks int,
	repeatedActionCount int,
	policy AgentProgressHeartbeatPolicyV0,
) AgentProgressStatusV0 {
	if processStatus == ProcessRuntimeStoppedV0 {
		return AgentStoppedV0
	}
	if repeatedActionCount >= policy.LoopAfterRepeatedActions {
		return AgentLoopDetectedV0
	}
	if noProgressTicks >= policy.StalledAfterNoProgressTicks {
		return AgentStalledV0
	}
	return AgentProgressingV0
}

func agentProgressHeartbeatSummaryV0(status AgentProgressStatusV0) string {
	switch status {
	case AgentProgressingV0:
		return "Heartbeat vivo con avance compacto."
	case AgentStalledV0:
		return "Heartbeat vivo sin avance compacto."
	case AgentLoopDetectedV0:
		return "Heartbeat vivo con repeticion compacta."
	case AgentStoppedV0:
		return "Proceso observado como parado."
	default:
		return "Heartbeat compacto observado."
	}
}

func agentProgressHeartbeatEvidenceRefsV0(
	snapshot ProcessRuntimeSnapshotV0,
	current AgentProgressHeartbeatV0,
) []string {
	refs := make([]string, 0, 3+len(current.EvidenceRefs))
	refs = append(refs, current.HeartbeatRef, current.ProcessRef)
	if snapshot.LaunchRef != "" {
		refs = append(refs, snapshot.LaunchRef)
	}
	if snapshot.StopRef != "" {
		refs = append(refs, snapshot.StopRef)
	}
	refs = append(refs, current.EvidenceRefs...)
	return refs
}
