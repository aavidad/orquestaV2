package orquestamcp

import (
	"sort"
	"strings"
)

func buildMCPAutoprogrammingProjectsV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingProjectV0 {
	byRef := map[string]*MCPAutoprogrammingProjectV0{}
	if queue != nil {
		for _, item := range queue.Ranked {
			project := projectProjectionMCPAutoprogrammingV0(byRef, item.AppRef)
			project.AppRef = firstNonEmptyMCPV0(project.AppRef, item.AppRef)
			project.QueueCount++
			project.RunRefs = appendCompactUniqueMCPAutoprogrammingV0(project.RunRefs, item.RunRef)
		}
	}
	if run != nil && run.Stats != nil {
		stats := run.Stats
		project := projectProjectionMCPAutoprogrammingV0(byRef, firstNonEmptyMCPV0(
			stats.ProjectRef,
			stats.AppSpecRef,
			queueAppRefForRunMCPAutoprogrammingV0(queue, stats.RunRef),
			run.RunRef,
		))
		project.ProjectRef = firstNonEmptyMCPV0(project.ProjectRef, stats.ProjectRef)
		project.AppRef = firstNonEmptyMCPV0(project.AppRef, stats.AppSpecRef)
		project.RunRefs = appendCompactUniqueMCPAutoprogrammingV0(project.RunRefs, stats.RunRef)
		project.TasksTotal += stats.Counts.TasksTotal
		project.TasksClosed += stats.Counts.TasksClosed
		project.AgentsInFlight += stats.Counts.AgentsInFlight
		project.Blocked = project.Blocked || stats.Closure.Blocked
	}
	keys := make([]string, 0, len(byRef))
	for key := range byRef {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]MCPAutoprogrammingProjectV0, 0, len(keys))
	for _, key := range keys {
		out = append(out, *byRef[key])
	}
	return out
}

func queueAppRefForRunMCPAutoprogrammingV0(
	queue *MCPRunQueuePriorityToolResultV0,
	runRef string,
) string {
	if queue == nil {
		return ""
	}
	runRef = strings.TrimSpace(runRef)
	for _, item := range queue.Ranked {
		if strings.TrimSpace(item.RunRef) == runRef {
			return strings.TrimSpace(item.AppRef)
		}
	}
	return ""
}

func buildMCPAutoprogrammingTasksV0(
	run *MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingTaskV0 {
	if run == nil || run.Stats == nil {
		return []MCPAutoprogrammingTaskV0{}
	}
	out := make([]MCPAutoprogrammingTaskV0, 0, len(run.Stats.Progress.Tasks))
	for _, task := range run.Stats.Progress.Tasks {
		out = append(out, MCPAutoprogrammingTaskV0{
			RunRef:              strings.TrimSpace(run.Stats.RunRef),
			TaskRef:             strings.TrimSpace(task.TaskRef),
			Status:              strings.TrimSpace(task.Status),
			AgentRequestID:      strings.TrimSpace(task.AgentRequestID),
			DeliveryRef:         strings.TrimSpace(task.DeliveryRef),
			LastReportRef:       strings.TrimSpace(task.LastReportRef),
			ProgressStatus:      strings.TrimSpace(task.ProgressStatus),
			Classification:      strings.TrimSpace(task.Classification),
			BudgetReason:        strings.TrimSpace(task.BudgetReason),
			AgeSeconds:          task.AgeSeconds,
			SecondsSinceAck:     task.SecondsSinceAck,
			MaxExpectedSeconds:  task.MaxExpectedSeconds,
			NoActivityLimit:     task.NoActivityLimitSeconds,
			NoProgressTicks:     task.NoProgressTicks,
			RepeatedActionCount: task.RepeatedActionCount,
			DecisionRequired:    task.DecisionRequired,
			Summary:             strings.TrimSpace(task.Summary),
			EvidenceRefs:        compactStringsMCPV0(task.EvidenceRefs),
		})
	}
	return out
}

func buildMCPAutoprogrammingAgentsV0(
	run *MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingAgentV0 {
	if run == nil || run.Stats == nil {
		return []MCPAutoprogrammingAgentV0{}
	}
	out := make([]MCPAutoprogrammingAgentV0, 0, len(run.Stats.Agents))
	for _, agent := range run.Stats.Agents {
		item := MCPAutoprogrammingAgentV0{
			RunRef:            strings.TrimSpace(run.Stats.RunRef),
			AgentRequestID:    strings.TrimSpace(agent.AgentRequestID),
			Status:            strings.TrimSpace(agent.Status),
			InFlight:          agent.InFlight,
			NeedsAttention:    agent.NeedsAttention,
			Completed:         agent.Completed,
			Failed:            agent.Failed,
			Lost:              agent.Lost,
			StopRequested:     agent.StopRequested,
			StopConfirmed:     agent.StopConfirmed,
			StopReasonCode:    strings.TrimSpace(agent.StopReasonCode),
			StopReasonSource:  strings.TrimSpace(agent.StopReasonSource),
			StopReasonRef:     strings.TrimSpace(agent.StopReasonRef),
			ControlRegistered: agent.ControlRegistered,
			ControlState:      strings.TrimSpace(agent.ControlState),
			CanStop:           agent.CanStop,
		}
		if agent.LastProgress != nil {
			item.TaskRef = strings.TrimSpace(agent.LastProgress.TaskRef)
			item.DeliveryRef = strings.TrimSpace(agent.LastProgress.DeliveryRef)
			item.ReportRef = strings.TrimSpace(agent.LastProgress.ReportRef)
			item.ProgressStatus = strings.TrimSpace(agent.LastProgress.Status)
			item.Classification = strings.TrimSpace(agent.LastProgress.Classification)
			item.BudgetReason = strings.TrimSpace(agent.LastProgress.BudgetReason)
			item.AgeSeconds = agent.LastProgress.AgeSeconds
			item.SecondsSinceAck = agent.LastProgress.SecondsSinceAck
			item.MaxExpectedSeconds = agent.LastProgress.MaxExpectedSeconds
			item.NoActivityLimit = agent.LastProgress.NoActivityLimitSeconds
			item.NoProgressTicks = agent.LastProgress.NoProgressTicks
			item.RepeatedActionCount = agent.LastProgress.RepeatedActionCount
			item.DecisionRequired = agent.LastProgress.DecisionRequired
			item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, agent.LastProgress.EvidenceRefs...))
		}
		if agent.Process != nil {
			item.ProcessRef = strings.TrimSpace(agent.Process.ProcessRef)
			item.SessionRef = strings.TrimSpace(agent.Process.SessionRef)
			item.LaunchRef = strings.TrimSpace(agent.Process.LaunchRef)
			item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, agent.Process.EvidenceRefs...))
		}
		if agent.Usage != nil {
			item.RuntimeKind = strings.TrimSpace(agent.Usage.RuntimeKind)
			item.ConnectorRef = strings.TrimSpace(agent.Usage.ConnectorRef)
			item.ProfileRef = strings.TrimSpace(agent.Usage.ProfileRef)
			item.CapacityLevel = strings.TrimSpace(agent.Usage.CapacityLevel)
			item.QuotaStatus = strings.TrimSpace(agent.Usage.QuotaStatus)
			item.TotalTokens = agent.Usage.TotalTokens
			item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, agent.Usage.EvidenceRefs...))
		}
		out = append(out, item)
	}
	return out
}

func projectProjectionMCPAutoprogrammingV0(
	byRef map[string]*MCPAutoprogrammingProjectV0,
	ref string,
) *MCPAutoprogrammingProjectV0 {
	key := strings.TrimSpace(ref)
	if key == "" {
		key = "project-ref-unknown"
	}
	if byRef[key] == nil {
		byRef[key] = &MCPAutoprogrammingProjectV0{ProjectRef: key}
	}
	return byRef[key]
}

func appendCompactUniqueMCPAutoprogrammingV0(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}
