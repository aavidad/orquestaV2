package orquestaobservability

import (
	"fmt"
	"strings"
)

func ValidateDirectorDecisionContextV0(context DirectorDecisionContextV0) error {
	var issues []OperationalStatusValidationIssueV0
	validateDirectorDecisionContextV0(context, "", addOperationalStatusIssueFuncV0(&issues))
	if len(issues) > 0 {
		return OperationalStatusValidationErrorV0{Issues: issues}
	}
	return nil
}

func validateDirectorDecisionContextV0(context DirectorDecisionContextV0, prefix string, add func(string, string)) {
	if strings.TrimSpace(context.SchemaVersion) != DirectorDecisionContextSchemaVersionV0 {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "schema_version"))
	}
	validateRequiredOperationalRefV0(context.RunRef, fieldV0(prefix, "run_ref"), add)
	validateOptionalOccurredAtDirectorDecisionV0(context.ObservedAt, fieldV0(prefix, "observed_at"), add)
	validateOptionalDirectorDecisionTokenV0(context.CurrentPhase, fieldV0(prefix, "current_phase"), add)
	validateDirectorDecisionProgressV0(context.Progress, fieldV0(prefix, "progress"), add)
	validateDirectorDecisionLifecycleV0(context.Lifecycle, fieldV0(prefix, "lifecycle"), add)
	validateDirectorDecisionClosureV0(context.Closure, fieldV0(prefix, "closure"), add)
	validateDirectorDecisionQuietnessV0(context.Quietness, fieldV0(prefix, "quietness"), add)
	validateDirectorDecisionReworkReplanV0(context.ReworkReplan, fieldV0(prefix, "rework_replan"), add)
	validateDirectorDecisionPhasesV0(context.Phases, fieldV0(prefix, "phases"), add)
	validateDirectorDecisionTasksV0(context.Tasks, fieldV0(prefix, "tasks"), add)
	validateDirectorDecisionAgentsV0(context.Agents, fieldV0(prefix, "agents"), add)
	validateDirectorDecisionBlockersV0(context.Blockers, fieldV0(prefix, "blockers"), add)
	validateDirectorDecisionActivityV0(context.Activity, fieldV0(prefix, "activity_recent"), add)
	validateDirectorDecisionWarningsV0(context.Warnings, fieldV0(prefix, "warnings"), add)
	validateDiagnosticoPrivacyV0(context.Privacy, fieldV0(prefix, "privacy"), add)
	validateOperationalJSONSizeV0(context, maxDiagnosticoJSONBytesV0, fieldV0(prefix, ""), add)
}

func validateDirectorDecisionProgressV0(progress DirectorDecisionProgressV0, prefix string, add func(string, string)) {
	validateDirectorDecisionNonNegativeV0(progress.PercentComplete, fieldV0(prefix, "percent_complete"), add)
	if progress.PercentComplete > 100 {
		add(ErrOperationalStatusQueryInvalidaV0, fieldV0(prefix, "percent_complete"))
	}
	validateDirectorDecisionNonNegativeV0(progress.TasksTotal, fieldV0(prefix, "tasks_total"), add)
	validateDirectorDecisionNonNegativeV0(progress.TasksClosed, fieldV0(prefix, "tasks_closed"), add)
	validateDirectorDecisionNonNegativeV0(progress.TasksOpen, fieldV0(prefix, "tasks_open"), add)
	validateDirectorDecisionNonNegativeV0(progress.TasksObserved, fieldV0(prefix, "tasks_observed"), add)
	validateDirectorDecisionNonNegativeV0(progress.ObservedAgents, fieldV0(prefix, "observed_agents"), add)
	validateDirectorDecisionNonNegativeV0(progress.ProgressingAgents, fieldV0(prefix, "progressing_agents"), add)
	validateDirectorDecisionNonNegativeV0(progress.StalledAgents, fieldV0(prefix, "stalled_agents"), add)
	validateDirectorDecisionNonNegativeV0(progress.LoopDetectedAgents, fieldV0(prefix, "loop_detected_agents"), add)
	validateDirectorDecisionNonNegativeV0(progress.StoppedAgents, fieldV0(prefix, "stopped_agents"), add)
	validateOperationalRefListV0(progress.NoSignalAgentRefs, maxOperationalEvidenceRefsV0, fieldV0(prefix, "no_signal_agent_refs"), add)
}

func validateDirectorDecisionLifecycleV0(lifecycle DirectorDecisionLifecycleV0, prefix string, add func(string, string)) {
	values := map[string]int{
		"agents_requested":          lifecycle.AgentsRequested,
		"agents_started":            lifecycle.AgentsStarted,
		"agents_running":            lifecycle.AgentsRunning,
		"agents_failed":             lifecycle.AgentsFailed,
		"agents_stop_requested":     lifecycle.AgentsStopRequested,
		"agents_stopped":            lifecycle.AgentsStopped,
		"agents_in_flight":          lifecycle.AgentsInFlight,
		"agents_control_registered": lifecycle.AgentsControlRegistered,
		"agents_control_missing":    lifecycle.AgentsControlMissing,
		"agents_need_attention":     lifecycle.AgentsNeedAttention,
	}
	for field, value := range values {
		validateDirectorDecisionNonNegativeV0(value, fieldV0(prefix, field), add)
	}
}

func validateDirectorDecisionClosureV0(closure DirectorDecisionClosureV0, prefix string, add func(string, string)) {
	validateOptionalDirectorDecisionTokenV0(closure.Status, fieldV0(prefix, "status"), add)
	for index, cause := range closure.BlockedBy {
		validateOperationalTokenTextV0(cause, fmt.Sprintf("%s[%d]", fieldV0(prefix, "blocked_by"), index), add)
	}
	validateOperationalRefListV0(closure.BlockerRefs, maxOperationalEvidenceRefsV0, fieldV0(prefix, "blocker_refs"), add)
}

func validateDirectorDecisionQuietnessV0(quietness DirectorDecisionQuietnessV0, prefix string, add func(string, string)) {
	validateOperationalRefListV0(quietness.NoSignalAgentRefs, maxOperationalEvidenceRefsV0, fieldV0(prefix, "no_signal_agent_refs"), add)
	validateDirectorDecisionNonNegativeV0(quietness.TasksWithoutSignal, fieldV0(prefix, "tasks_without_signal"), add)
	validateDirectorDecisionNonNegativeV0(quietness.StalledAgents, fieldV0(prefix, "stalled_agents"), add)
	validateDirectorDecisionNonNegativeV0(quietness.LoopDetectedAgents, fieldV0(prefix, "loop_detected_agents"), add)
	validateDirectorDecisionNonNegativeV0(quietness.StoppedAgents, fieldV0(prefix, "stopped_agents"), add)
	validateDirectorDecisionNonNegativeV0(quietness.MaxNoProgressTicks, fieldV0(prefix, "max_no_progress_ticks"), add)
}

func validateDirectorDecisionReworkReplanV0(value DirectorDecisionReworkReplanV0, prefix string, add func(string, string)) {
	validateDirectorDecisionNonNegativeV0(value.ReworkRequests, fieldV0(prefix, "rework_requests"), add)
	validateOperationalRefListV0(value.ReworkRequestRefs, maxOperationalReferenceItemsV0, fieldV0(prefix, "rework_request_refs"), add)
	validateDirectorDecisionNonNegativeV0(value.ReplanDecisions, fieldV0(prefix, "replan_decisions"), add)
	validateOperationalRefListV0(value.ReplanDecisionRefs, maxOperationalReferenceItemsV0, fieldV0(prefix, "replan_decision_refs"), add)
}

func validateDirectorDecisionPhasesV0(phases []DirectorDecisionPhaseV0, prefix string, add func(string, string)) {
	if len(phases) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, phase := range phases {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateOperationalTokenTextV0(phase.PhaseID, fieldV0(itemPrefix, "phase_id"), add)
		validateOptionalDirectorDecisionTokenV0(phase.Status, fieldV0(itemPrefix, "status"), add)
		validateOptionalOccurredAtDirectorDecisionV0(phase.OpenedAt, fieldV0(itemPrefix, "opened_at"), add)
		validateOptionalOccurredAtDirectorDecisionV0(phase.ClosedAt, fieldV0(itemPrefix, "closed_at"), add)
		if phase.DurationSeconds < 0 {
			add(ErrOperationalStatusQueryInvalidaV0, fieldV0(itemPrefix, "duration_seconds"))
		}
		validateOptionalDirectorDecisionTokenV0(phase.RecommendedCapacity, fieldV0(itemPrefix, "recommended_capacity"), add)
		validateDirectorDecisionNonNegativeV0(phase.EntryCriteria, fieldV0(itemPrefix, "entry_criteria"), add)
		validateDirectorDecisionNonNegativeV0(phase.ExitCriteria, fieldV0(itemPrefix, "exit_criteria"), add)
		validateDirectorDecisionNonNegativeV0(phase.EvidenceRequired, fieldV0(itemPrefix, "evidence_required"), add)
	}
}

func validateDirectorDecisionTasksV0(tasks []DirectorDecisionTaskV0, prefix string, add func(string, string)) {
	if len(tasks) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, task := range tasks {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(task.TaskRef, fieldV0(itemPrefix, "task_ref"), add)
		validateOperationalTokenTextV0(task.Status, fieldV0(itemPrefix, "status"), add)
		validateOptionalOperationalRefV0(task.AgentRequestID, fieldV0(itemPrefix, "agent_request_id"), add)
		validateOptionalOperationalRefV0(task.DeliveryRef, fieldV0(itemPrefix, "delivery_ref"), add)
		validateOptionalOperationalRefV0(task.LastReportRef, fieldV0(itemPrefix, "last_report_ref"), add)
		validateOptionalDirectorDecisionTokenV0(task.ProgressStatus, fieldV0(itemPrefix, "progress_status"), add)
		validateDirectorDecisionNonNegativeV0(task.NoProgressTicks, fieldV0(itemPrefix, "no_progress_ticks"), add)
		validateDirectorDecisionNonNegativeV0(task.RepeatedActionCount, fieldV0(itemPrefix, "repeated_action_count"), add)
		validateOperationalRefListV0(task.EvidenceRefs, maxOperationalEvidenceRefsV0, fieldV0(itemPrefix, "evidence_refs"), add)
	}
}

func validateDirectorDecisionAgentsV0(agents []DirectorDecisionAgentV0, prefix string, add func(string, string)) {
	if len(agents) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, agent := range agents {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(agent.AgentRequestID, fieldV0(itemPrefix, "agent_request_id"), add)
		validateOperationalTokenTextV0(agent.Status, fieldV0(itemPrefix, "status"), add)
		validateOptionalDirectorDecisionTokenV0(agent.ControlState, fieldV0(itemPrefix, "control_state"), add)
		validateOptionalOperationalRefV0(agent.ProcessRef, fieldV0(itemPrefix, "process_ref"), add)
		validateOptionalOperationalRefV0(agent.SessionRef, fieldV0(itemPrefix, "session_ref"), add)
		validateOptionalOperationalRefV0(agent.LaunchRef, fieldV0(itemPrefix, "launch_ref"), add)
		validateOptionalOperationalRefV0(agent.ReadinessRef, fieldV0(itemPrefix, "readiness_ref"), add)
		validateOptionalOperationalRefV0(agent.ProgressTaskRef, fieldV0(itemPrefix, "progress_task_ref"), add)
		validateOptionalDirectorDecisionTokenV0(agent.ProgressStatus, fieldV0(itemPrefix, "progress_status"), add)
		validateDirectorDecisionNonNegativeV0(agent.NoProgressTicks, fieldV0(itemPrefix, "no_progress_ticks"), add)
		validateDirectorDecisionNonNegativeV0(agent.RepeatedActionCount, fieldV0(itemPrefix, "repeated_action_count"), add)
		validateOperationalRefListV0(agent.EvidenceRefs, maxOperationalEvidenceRefsV0, fieldV0(itemPrefix, "evidence_refs"), add)
	}
}

func validateDirectorDecisionBlockersV0(blockers []DirectorDecisionBlockerV0, prefix string, add func(string, string)) {
	if len(blockers) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, blocker := range blockers {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(blocker.BlockerRef, fieldV0(itemPrefix, "blocker_ref"), add)
		validateOptionalDirectorDecisionTokenV0(blocker.Cause, fieldV0(itemPrefix, "cause"), add)
		validateOperationalTokenTextV0(blocker.Source, fieldV0(itemPrefix, "source"), add)
		validateOperationalI18nKeyV0(blocker.SummaryKey, fieldV0(itemPrefix, "summary_key"), add)
	}
}

func validateDirectorDecisionActivityV0(items []DirectorDecisionActivityV0, prefix string, add func(string, string)) {
	if len(items) > maxOperationalListItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, item := range items {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateRequiredOperationalRefV0(item.ActivityRef, fieldV0(itemPrefix, "activity_ref"), add)
		validateOperationalTokenTextV0(item.Kind, fieldV0(itemPrefix, "kind"), add)
		validateOptionalOccurredAtDirectorDecisionV0(item.OccurredAt, fieldV0(itemPrefix, "occurred_at"), add)
		validateOptionalOperationalRefV0(item.SourceRef, fieldV0(itemPrefix, "source_ref"), add)
		validateOptionalOperationalRefV0(item.AgentRequestID, fieldV0(itemPrefix, "agent_request_id"), add)
		validateOptionalOperationalRefV0(item.TaskRef, fieldV0(itemPrefix, "task_ref"), add)
		validateOperationalI18nKeyV0(item.SummaryKey, fieldV0(itemPrefix, "summary_key"), add)
	}
}

func validateDirectorDecisionWarningsV0(warnings []DirectorDecisionWarningV0, prefix string, add func(string, string)) {
	if len(warnings) > maxOperationalWarningItemsV0 {
		add(ErrConsultaDemasiadoAmpliaV0, prefix)
	}
	for index, warning := range warnings {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, index)
		validateOperationalTokenTextV0(warning.Code, fieldV0(itemPrefix, "code"), add)
		validateOperationalI18nKeyV0(warning.SummaryKey, fieldV0(itemPrefix, "summary_key"), add)
	}
}

func validateOptionalDirectorDecisionTokenV0(value string, field string, add func(string, string)) {
	if strings.TrimSpace(value) == "" {
		return
	}
	validateOperationalTokenTextV0(value, field, add)
}

func validateOptionalOccurredAtDirectorDecisionV0(value string, field string, add func(string, string)) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if !isOccurredAtV0(value) {
		add(ErrOperationalStatusQueryInvalidaV0, field)
	}
}

func validateDirectorDecisionNonNegativeV0(value int, field string, add func(string, string)) {
	if value < 0 {
		add(ErrOperationalStatusQueryInvalidaV0, field)
	}
}
