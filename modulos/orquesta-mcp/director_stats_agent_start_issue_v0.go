package orquestamcp

import (
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	mcpDirectorStatsAgentRequestedNotStartedV0             = "agent_requested_not_started"
	mcpDirectorStatsExternalWorkAgentRequestedNotStartedV0 = "external_work_agent_requested_not_started"
	mcpDirectorStatsExternalWorkStoppedNoDeliveryV0        = "external_work_accepted_stopped_without_delivery"
)

func enrichMCPDirectorStatsRequestedAgentNotStartedV0(
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if stats == nil ||
		stats.Counts.AgentsRequested <= 0 ||
		stats.Counts.AgentsStarted > 0 ||
		stats.Counts.AgentsInFlight > 0 ||
		mcpDirectorStatsHasProcessRefV0(stats.Agents) {
		return
	}
	code := mcpDirectorStatsAgentRequestedNotStartedV0
	if mcpDirectorStatsLooksExternalWorkV0(*stats) {
		code = mcpDirectorStatsExternalWorkAgentRequestedNotStartedV0
	}
	if mcpDirectorStatsProgressIssueExistsV0(stats.Progress.Issues, code) {
		return
	}
	stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
		Code:    code,
		Field:   "agents",
		Message: "cause=unknown action=retry_materialization_or_check_capacity_auth_runtime_queue_outbox_policy",
	})
}

func enrichMCPDirectorStatsExternalWorkStoppedNoDeliveryV0(
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if stats == nil ||
		!mcpDirectorStatsLooksExternalWorkV0(*stats) ||
		strings.TrimSpace(stats.Status) != "stopped" ||
		stats.Counts.AgentsRequested > 0 ||
		stats.Counts.AgentsStarted > 0 ||
		stats.Counts.AgentsInFlight > 0 ||
		stats.Counts.Deliveries > 0 ||
		stats.Counts.AgentsDelivered > 0 {
		return
	}
	if mcpDirectorStatsProgressIssueExistsV0(stats.Progress.Issues, mcpDirectorStatsExternalWorkStoppedNoDeliveryV0) {
		return
	}
	stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
		Code:    mcpDirectorStatsExternalWorkStoppedNoDeliveryV0,
		Field:   "run",
		Message: "external_work accepted and stopped without agents or delivery action=relaunch_or_replan_external_work_with_causal_error",
	})
}

func mcpDirectorStatsLooksExternalWorkV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) bool {
	projectRef := strings.ToLower(strings.TrimSpace(stats.ProjectRef))
	appSpecRef := strings.ToLower(strings.TrimSpace(stats.AppSpecRef))
	return strings.Contains(projectRef, "external-work") ||
		strings.Contains(projectRef, "external_work") ||
		strings.Contains(appSpecRef, "external-work") ||
		strings.Contains(appSpecRef, "external_work") ||
		projectRef == "opes" ||
		strings.HasPrefix(projectRef, "opes-") ||
		strings.HasPrefix(appSpecRef, "app-spec-opes")
}

func mcpDirectorStatsHasProcessRefV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
) bool {
	for _, agent := range agents {
		if agent.Process == nil {
			continue
		}
		if strings.TrimSpace(agent.Process.ProcessRef) != "" ||
			strings.TrimSpace(agent.Process.SessionRef) != "" ||
			strings.TrimSpace(agent.Process.LaunchRef) != "" ||
			strings.TrimSpace(agent.Process.ReadinessRef) != "" {
			return true
		}
	}
	return false
}

func mcpDirectorStatsProgressIssueExistsV0(
	issues []orquestacionnucleoapp.DirectorProgressIssueV0,
	code string,
) bool {
	code = strings.TrimSpace(code)
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}
