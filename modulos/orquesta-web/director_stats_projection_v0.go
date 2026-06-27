package orquestaweb

func directorStatsAgentsV0(
	agents []WebDirectorAgentStatsContractV0,
) []WebDirectorStatsAgentV0 {
	out := make([]WebDirectorStatsAgentV0, 0, len(agents))
	for _, agent := range agents {
		item := WebDirectorStatsAgentV0{
			AgentRequestID:   trimDirectorStatsV0(agent.AgentRequestID),
			Status:           trimDirectorStatsV0(agent.Status),
			ControlState:     trimDirectorStatsV0(agent.ControlState),
			InFlight:         agent.InFlight,
			CanStop:          agent.CanStop,
			NeedsAttention:   agent.NeedsAttention,
			StopReasonCode:   trimDirectorStatsV0(agent.StopReasonCode),
			StopReasonSource: trimDirectorStatsV0(agent.StopReasonSource),
			StopReasonRef:    trimDirectorStatsV0(agent.StopReasonRef),
		}
		if agent.LastProgress != nil {
			progress := agent.LastProgress
			item.ProgressStatus = trimDirectorStatsV0(progress.Status)
			item.TaskRef = trimDirectorStatsV0(progress.TaskRef)
			item.NoProgressTicks = progress.NoProgressTicks
			item.RepeatedActionCount = progress.RepeatedActionCount
		}
		if agent.Usage != nil {
			item.CapacityLevel = trimDirectorStatsV0(agent.Usage.CapacityLevel)
			item.QuotaStatus = trimDirectorStatsV0(agent.Usage.QuotaStatus)
			item.QuotaRemaining = agent.Usage.QuotaRemaining
			item.QuotaLimit = agent.Usage.QuotaLimit
			item.TotalTokens = agent.Usage.TotalTokens
		}
		out = append(out, item)
	}
	if out == nil {
		return []WebDirectorStatsAgentV0{}
	}
	return out
}

func directorStatsTasksV0(
	tasks []WebDirectorTaskProgressV0,
) []WebDirectorStatsTaskV0 {
	out := make([]WebDirectorStatsTaskV0, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, WebDirectorStatsTaskV0{
			TaskRef:        trimDirectorStatsV0(task.TaskRef),
			Status:         trimDirectorStatsV0(task.Status),
			AgentRequestID: trimDirectorStatsV0(task.AgentRequestID),
			ProgressStatus: trimDirectorStatsV0(task.ProgressStatus),
		})
	}
	if out == nil {
		return []WebDirectorStatsTaskV0{}
	}
	return out
}

func directorStatsExternalJobV0(
	externalJob *WebDirectorExternalJobV0,
) WebDirectorExternalJobV0 {
	if externalJob == nil || trimDirectorStatsV0(externalJob.JobRef) == "" {
		return WebDirectorExternalJobV0{}
	}
	out := WebDirectorExternalJobV0{
		Available:    true,
		AppRef:       trimDirectorStatsV0(externalJob.AppRef),
		JobRef:       trimDirectorStatsV0(externalJob.JobRef),
		WorkKind:     trimDirectorStatsV0(externalJob.WorkKind),
		ChangeRef:    trimDirectorStatsV0(externalJob.ChangeRef),
		RunRef:       trimDirectorStatsV0(externalJob.RunRef),
		TaskRef:      trimDirectorStatsV0(externalJob.TaskRef),
		AgentRef:     trimDirectorStatsV0(externalJob.AgentRef),
		Status:       trimDirectorStatsV0(externalJob.Status),
		StatusReason: trimDirectorStatsV0(externalJob.StatusReason),
		DeliveryRefs: compactOperationalStringsV0(externalJob.DeliveryRefs),
		IssueRefs:    compactOperationalStringsV0(externalJob.IssueRefs),
		EvidenceRefs: compactOperationalStringsV0(externalJob.EvidenceRefs),
		Diagnostics:  directorStatsExternalJobDiagnosticsV0(externalJob.Diagnostics),
	}
	return out
}

func directorStatsExternalJobDiagnosticsV0(
	diagnostics []WebDirectorExternalJobDiagnosticV0,
) []WebDirectorExternalJobDiagnosticV0 {
	out := make([]WebDirectorExternalJobDiagnosticV0, 0, len(diagnostics))
	for _, item := range diagnostics {
		code := trimDirectorStatsV0(item.Code)
		if code == "" {
			continue
		}
		out = append(out, WebDirectorExternalJobDiagnosticV0{
			Code:         code,
			Scope:        trimDirectorStatsV0(item.Scope),
			Message:      trimDirectorStatsV0(item.Message),
			EvidenceRefs: compactOperationalStringsV0(item.EvidenceRefs),
		})
	}
	if out == nil {
		return []WebDirectorExternalJobDiagnosticV0{}
	}
	return out
}

func directorStatsExternalJobNeedsAttentionV0(externalJob WebDirectorExternalJobV0) bool {
	if !externalJob.Available {
		return false
	}
	if len(externalJob.IssueRefs) > 0 || len(externalJob.Diagnostics) > 0 {
		return true
	}
	switch trimDirectorStatsV0(externalJob.Status) {
	case "integration_required", "parent_ack_received", "blocked", "failed":
		return true
	default:
		return false
	}
}

func directorStatsCheckpointV0(
	stats WebDirectorRunStatsContractV0,
) WebDirectorStatsCheckpointV0 {
	pendingRefs := compactOperationalStringsV0(stats.PendingCheckpointAgentRefs)
	evidenceRefs := compactOperationalStringsV0(stats.CheckpointEvidenceRefs)
	agentsPending := stats.CheckpointAgentsPending
	if agentsPending == 0 && len(pendingRefs) > 0 {
		agentsPending = len(pendingRefs)
	}
	return WebDirectorStatsCheckpointV0{
		CheckpointAgentsPending:    agentsPending,
		PendingCheckpointAgentRefs: pendingRefs,
		CheckpointEvidenceRefs:     evidenceRefs,
		RequiresAttention:          agentsPending > 0 || len(pendingRefs) > 0,
	}
}

func directorStatsIssuesV0(
	first []WebDirectorStatsPublicIssueV0,
	second []WebDirectorStatsPublicIssueV0,
) []WebDirectorStatsPublicIssueV0 {
	out := make([]WebDirectorStatsPublicIssueV0, 0, len(first)+len(second))
	for _, value := range append(first, second...) {
		code := trimDirectorStatsV0(value.Code)
		if code == "" {
			code = "director_stats_error_publico"
		}
		out = append(out, WebDirectorStatsPublicIssueV0{
			Code:    code,
			Field:   trimDirectorStatsV0(value.Field),
			Message: trimDirectorStatsV0(value.Message),
		})
	}
	if out == nil {
		return []WebDirectorStatsPublicIssueV0{}
	}
	return out
}

func webDirectorStatsAttentionCountV0(
	agents []WebDirectorAgentStatsContractV0,
) int {
	count := 0
	for _, agent := range agents {
		if agent.NeedsAttention {
			count++
		}
	}
	return count
}
