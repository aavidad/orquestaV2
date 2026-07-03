package orquestamcp

import (
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func buildMCPDirectorStatsOpsSnapshotV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
	context *orquestaobservability.DirectorDecisionContextV0,
	observedAt string,
) *orquestaobservability.DirectorAutonomousOpsSnapshotV0 {
	run := directorOpsRunFromStatsMCPV0(stats)
	snapshot := orquestaobservability.DirectorAutonomousOpsSnapshotV0{
		SchemaVersion: orquestaobservability.DirectorAutonomousOpsSnapshotSchemaVersionV0,
		SnapshotRef:   "ops-snapshot-" + strings.TrimSpace(stats.RunRef),
		ObservedAt:    firstNonEmptyMCPV0(observedAt, contextObservedAtMCPV0(context)),
		Runs:          []orquestaobservability.DirectorAutonomousOpsRunV0{run},
		Agents:        directorOpsAgentsFromStatsMCPV0(stats),
		Privacy:       orquestaobservability.NewDiagnosticoPrivacyMetadataOnlyV0(),
	}
	snapshot.Decision = directorOpsDecisionFromRunMCPV0(run, stats.Closure.BlockerRefs)
	return &snapshot
}

func buildMCPAutoprogrammingOpsSnapshotV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	operator *MCPAutoprogrammingOperatorV0,
	staleRunning []MCPAutoprogrammingActionableRunV0,
	observedAt string,
) *orquestaobservability.DirectorAutonomousOpsSnapshotV0 {
	snapshot := orquestaobservability.DirectorAutonomousOpsSnapshotV0{
		SchemaVersion: orquestaobservability.DirectorAutonomousOpsSnapshotSchemaVersionV0,
		SnapshotRef:   "ops-snapshot-" + firstNonEmptyMCPV0(opsQueueRefMCPV0(queue), opsRunRefMCPV0(run), "autoprogramming"),
		ObservedAt:    strings.TrimSpace(observedAt),
		Queue:         directorOpsQueueFromMCPV0(queue),
		Privacy:       orquestaobservability.NewDiagnosticoPrivacyMetadataOnlyV0(),
	}
	if run != nil && run.Stats != nil {
		snapshot.Runs = []orquestaobservability.DirectorAutonomousOpsRunV0{
			directorOpsRunFromStatsMCPV0(*run.Stats),
		}
		snapshot.Agents = directorOpsAgentsFromStatsMCPV0(*run.Stats)
	}
	snapshot.Decision = directorOpsDecisionFromAutoprogrammingSafeActionsMCPV0(operator, snapshot, staleRunning)
	return &snapshot
}

func directorOpsQueueFromMCPV0(
	queue *MCPRunQueuePriorityToolResultV0,
) orquestaobservability.DirectorAutonomousOpsQueueV0 {
	if queue == nil {
		return orquestaobservability.DirectorAutonomousOpsQueueV0{}
	}
	refs := make([]string, 0, len(queue.Ranked))
	for _, item := range queue.Ranked {
		if ref := strings.TrimSpace(item.RunRef); ref != "" {
			refs = append(refs, ref)
		}
	}
	return orquestaobservability.DirectorAutonomousOpsQueueV0{
		QueueRef:      strings.TrimSpace(queue.QueueRef),
		Live:          queue.Estado == MCPRunQueuePriorityEstadoOKV0,
		Count:         queue.Count,
		RankedRunRefs: compactStringsMCPV0(refs),
	}
}

func directorOpsRunFromStatsMCPV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) orquestaobservability.DirectorAutonomousOpsRunV0 {
	tasksTotal := firstNonZeroIntMCPV0(stats.Progress.TasksTotal, stats.Counts.TasksTotal)
	tasksClosed := firstNonZeroIntMCPV0(stats.Progress.TasksClosed, stats.Counts.TasksClosed)
	run := orquestaobservability.DirectorAutonomousOpsRunV0{
		RunRef:              strings.TrimSpace(stats.RunRef),
		AppRef:              strings.TrimSpace(stats.ProjectRef),
		Status:              strings.TrimSpace(stats.Status),
		CurrentPhase:        strings.TrimSpace(stats.CurrentPhase),
		ClosureStatus:       strings.TrimSpace(stats.Closure.Status),
		Blocked:             stats.Closure.Blocked,
		PercentComplete:     stats.Progress.PercentComplete,
		TasksTotal:          tasksTotal,
		TasksClosed:         tasksClosed,
		TasksOpen:           stats.Counts.TasksOpen,
		AgentsInFlight:      stats.Counts.AgentsInFlight,
		AgentsNeedAttention: directorOpsAgentsNeedAttentionMCPV0(stats),
		AgentsFailed:        stats.Counts.AgentsFailed,
		ReworkRequests:      stats.Counts.ReworkRequests,
		ReplanDecisions:     stats.Counts.ReplanDecisions,
	}
	if stats.UsageSummary != nil {
		run.QuotaStatus = strings.TrimSpace(stats.UsageSummary.QuotaStatus)
		run.QuotaRemaining = stats.UsageSummary.QuotaRemaining
		run.QuotaLimit = stats.UsageSummary.QuotaLimit
		run.TotalTokens = stats.UsageSummary.TotalTokens
	}
	return run
}

func directorOpsAgentsFromStatsMCPV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []orquestaobservability.DirectorAutonomousOpsAgentV0 {
	out := make([]orquestaobservability.DirectorAutonomousOpsAgentV0, 0, len(stats.Agents))
	for _, agent := range stats.Agents {
		item := orquestaobservability.DirectorAutonomousOpsAgentV0{
			RunRef:         strings.TrimSpace(stats.RunRef),
			AgentRef:       strings.TrimSpace(agent.AgentRequestID),
			Status:         strings.TrimSpace(agent.Status),
			InFlight:       agent.InFlight,
			NeedsAttention: agent.NeedsAttention,
			CanStop:        agent.CanStop,
		}
		if agent.LastProgress != nil {
			item.TaskRef = strings.TrimSpace(agent.LastProgress.TaskRef)
			item.ProgressStatus = strings.TrimSpace(agent.LastProgress.Status)
		}
		if agent.Usage != nil {
			item.RuntimeKind = strings.TrimSpace(agent.Usage.RuntimeKind)
			item.CapacityLevel = strings.TrimSpace(agent.Usage.CapacityLevel)
			item.QuotaStatus = strings.TrimSpace(agent.Usage.QuotaStatus)
			item.TotalTokens = agent.Usage.TotalTokens
		}
		if item.AgentRef != "" {
			out = append(out, item)
		}
	}
	return out
}

func directorOpsDecisionFromSnapshotMCPV0(
	snapshot orquestaobservability.DirectorAutonomousOpsSnapshotV0,
) orquestaobservability.DirectorAutonomousOpsDecisionV0 {
	for _, run := range snapshot.Runs {
		decision := directorOpsDecisionFromRunMCPV0(run, nil)
		if decision.Action != orquestaobservability.DirectorAutonomousOpsActionIdleV0 &&
			decision.Action != orquestaobservability.DirectorAutonomousOpsActionClosedV0 {
			return decision
		}
	}
	if snapshot.Queue.Live && snapshot.Queue.Count > 0 {
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionSuperviseQueueV0,
			Scope:      "queue",
			ReasonCode: "queue_has_candidates",
			SummaryKey: "director.ops.decision.supervise_queue",
		}
	}
	return orquestaobservability.DirectorAutonomousOpsDecisionV0{
		Action:     orquestaobservability.DirectorAutonomousOpsActionIdleV0,
		Scope:      "workspace",
		ReasonCode: "no_active_work",
		SummaryKey: "director.ops.decision.idle",
	}
}

func directorOpsDecisionFromAutoprogrammingSafeActionsMCPV0(
	operator *MCPAutoprogrammingOperatorV0,
	snapshot orquestaobservability.DirectorAutonomousOpsSnapshotV0,
	staleRunning []MCPAutoprogrammingActionableRunV0,
) orquestaobservability.DirectorAutonomousOpsDecisionV0 {
	for _, action := range staleRunning {
		if decision, ok := directorOpsDecisionFromAutoprogrammingActionableRunMCPV0(action); ok {
			return decision
		}
		code := strings.TrimSpace(action.Code)
		if code != mcpAutoprogrammingActionGoalFirstStateMissingV0 &&
			code != mcpAutoprogrammingActionGoalFirstBlockedV0 {
			continue
		}
		if code == mcpAutoprogrammingActionGoalFirstBlockedV0 {
			return orquestaobservability.DirectorAutonomousOpsDecisionV0{
				Action:       orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0,
				Scope:        "run",
				RunRef:       strings.TrimSpace(action.RunRef),
				Attention:    true,
				ReasonCode:   "goal_first_blocked",
				SummaryKey:   "director.ops.decision.review_replan",
				EvidenceRefs: compactStringsMCPV0(action.EvidenceRefs),
			}
		}
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:       orquestaobservability.DirectorAutonomousOpsActionRepairGoalStateV0,
			Scope:        "run",
			RunRef:       strings.TrimSpace(action.RunRef),
			Attention:    true,
			ReasonCode:   "goal_first_state_missing",
			SummaryKey:   "director.ops.decision.repair_goal_state",
			EvidenceRefs: compactStringsMCPV0(action.EvidenceRefs),
		}
	}
	safeActions := directorOpsAutoprogrammingSafeActionsMCPV0(operator)
	for _, action := range safeActions {
		if strings.TrimSpace(action.Action) != "observe_goal" {
			continue
		}
		runRef := strings.TrimSpace(action.RunRef)
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionObserveGoalV0,
			Scope:      firstNonEmptyMCPV0(strings.TrimSpace(action.Scope), "run"),
			RunRef:     runRef,
			ReasonCode: "goal_first_observe_required",
			SummaryKey: "director.ops.decision.observe_goal",
		}
	}
	if operator != nil && len(safeActions) == 0 {
		return directorOpsDecisionFromAutoprogrammingSnapshotWithoutQueueSuperviseMCPV0(snapshot)
	}
	return directorOpsDecisionFromSnapshotMCPV0(snapshot)
}

func directorOpsDecisionFromAutoprogrammingActionableRunMCPV0(
	action MCPAutoprogrammingActionableRunV0,
) (orquestaobservability.DirectorAutonomousOpsDecisionV0, bool) {
	recommendedAction := strings.TrimSpace(action.RecommendedAction)
	if recommendedAction == "" {
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{}, false
	}
	switch strings.TrimSpace(action.Severity) {
	case "blocked", "warning":
	default:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{}, false
	}
	code := strings.TrimSpace(action.Code)
	if code == "" || code == mcpAutoprogrammingActionGoalFirstBlockedV0 ||
		code == mcpAutoprogrammingActionGoalFirstStateMissingV0 {
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{}, false
	}
	return orquestaobservability.DirectorAutonomousOpsDecisionV0{
		Action:       orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0,
		Scope:        "run",
		RunRef:       strings.TrimSpace(action.RunRef),
		Attention:    true,
		ReasonCode:   code,
		SummaryKey:   "director.ops.decision." + directorOpsSummaryActionKeyMCPV0(recommendedAction),
		EvidenceRefs: compactStringsMCPV0(action.EvidenceRefs),
	}, true
}

func directorOpsSummaryActionKeyMCPV0(action string) string {
	action = strings.TrimSpace(action)
	prefix, _, found := strings.Cut(action, ":")
	if found && strings.TrimSpace(prefix) != "" {
		return strings.TrimSpace(prefix)
	}
	return action
}

func directorOpsDecisionFromAutoprogrammingSnapshotWithoutQueueSuperviseMCPV0(
	snapshot orquestaobservability.DirectorAutonomousOpsSnapshotV0,
) orquestaobservability.DirectorAutonomousOpsDecisionV0 {
	for _, run := range snapshot.Runs {
		decision := directorOpsDecisionFromRunMCPV0(run, nil)
		if decision.Action != orquestaobservability.DirectorAutonomousOpsActionIdleV0 &&
			decision.Action != orquestaobservability.DirectorAutonomousOpsActionClosedV0 {
			return decision
		}
	}
	return orquestaobservability.DirectorAutonomousOpsDecisionV0{
		Action:     orquestaobservability.DirectorAutonomousOpsActionIdleV0,
		Scope:      "workspace",
		ReasonCode: "no_safe_autoprogramming_action",
		SummaryKey: "director.ops.decision.idle",
	}
}

func directorOpsAutoprogrammingSafeActionsMCPV0(
	operator *MCPAutoprogrammingOperatorV0,
) []MCPAutoprogrammingSafeActionV0 {
	if operator == nil {
		return nil
	}
	return operator.SafeActions
}

func directorOpsDecisionFromRunMCPV0(
	run orquestaobservability.DirectorAutonomousOpsRunV0,
	evidenceRefs []string,
) orquestaobservability.DirectorAutonomousOpsDecisionV0 {
	switch {
	case strings.TrimSpace(run.Status) == "goal_first_state_missing":
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:       orquestaobservability.DirectorAutonomousOpsActionRepairGoalStateV0,
			Scope:        "run",
			RunRef:       run.RunRef,
			Attention:    true,
			ReasonCode:   "goal_first_state_missing",
			SummaryKey:   "director.ops.decision.repair_goal_state",
			EvidenceRefs: compactStringsMCPV0(evidenceRefs),
		}
	case run.Blocked || run.AgentsNeedAttention > 0 || run.AgentsFailed > 0:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:       orquestaobservability.DirectorAutonomousOpsActionReviewReplanV0,
			Scope:        "run",
			RunRef:       run.RunRef,
			Attention:    true,
			ReasonCode:   "attention_required",
			SummaryKey:   "director.ops.decision.review_replan",
			EvidenceRefs: compactStringsMCPV0(evidenceRefs),
		}
	case run.AgentsInFlight > 0:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionWaitDeliveriesV0,
			Scope:      "run",
			RunRef:     run.RunRef,
			ReasonCode: "agents_in_flight",
			SummaryKey: "director.ops.decision.wait_deliveries",
		}
	case run.ClosureStatus == orquestacionnucleoapp.DirectorClosureStatusReadyV0:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionCloseOrValidateV0,
			Scope:      "run",
			RunRef:     run.RunRef,
			ReasonCode: "closure_ready",
			SummaryKey: "director.ops.decision.close_or_validate",
		}
	case run.ClosureStatus == orquestacionnucleoapp.DirectorClosureStatusClosedV0:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionClosedV0,
			Scope:      "run",
			RunRef:     run.RunRef,
			ReasonCode: "closure_closed",
			SummaryKey: "director.ops.decision.closed",
		}
	case run.TasksOpen > 0:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionContinueRunV0,
			Scope:      "run",
			RunRef:     run.RunRef,
			ReasonCode: "open_tasks",
			SummaryKey: "director.ops.decision.continue_run",
		}
	default:
		return orquestaobservability.DirectorAutonomousOpsDecisionV0{
			Action:     orquestaobservability.DirectorAutonomousOpsActionIdleV0,
			Scope:      "run",
			RunRef:     run.RunRef,
			ReasonCode: "run_quiescent",
			SummaryKey: "director.ops.decision.idle",
		}
	}
}

func directorOpsAgentsNeedAttentionMCPV0(stats orquestacionnucleoapp.DirectorRunStatsV0) int {
	total := 0
	for _, agent := range stats.Agents {
		if agent.NeedsAttention {
			total++
		}
	}
	return total
}

func opsQueueRefMCPV0(queue *MCPRunQueuePriorityToolResultV0) string {
	if queue == nil {
		return ""
	}
	return strings.TrimSpace(queue.QueueRef)
}

func opsRunRefMCPV0(run *MCPDirectorStatsToolResultV0) string {
	if run == nil {
		return ""
	}
	return strings.TrimSpace(run.RunRef)
}

func contextObservedAtMCPV0(context *orquestaobservability.DirectorDecisionContextV0) string {
	if context == nil {
		return ""
	}
	return strings.TrimSpace(context.ObservedAt)
}

func firstNonZeroIntMCPV0(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
