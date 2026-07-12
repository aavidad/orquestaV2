package orquestadirectoragentworkflow

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

const DirectorAgentDecisionBatchTooLargeReasonV0 = "director_decisions_batch_too_large"

type DirectorAgentDecisionBatchBudgetV0 struct {
	MaxSources          int
	MaxDescriptors      int
	MaxDecisions        int
	MaxCreateMicrotasks int
	MaxNewTasks         int
	MaxExpectedOutbox   int
	MaxBytes            int
}

type DirectorAgentDecisionBatchStatsV0 struct {
	Sources            int
	Descriptors        int
	Decisions          int
	CreateMicrotasks   int
	NewTasks           int
	ExpectedOutbox     int
	Bytes              int
	OmittedDecisions   int
	OmittedDescriptors int
	Cause              string
}

type DirectorAgentDecisionBatchBudgetIssueV0 struct {
	ReasonCode string
	Field      string
	Limit      int
	Stats      DirectorAgentDecisionBatchStatsV0
}

func (issue DirectorAgentDecisionBatchBudgetIssueV0) Error() string {
	reason := strings.TrimSpace(issue.ReasonCode)
	if reason == "" {
		reason = DirectorAgentDecisionBatchTooLargeReasonV0
	}
	return fmt.Sprintf(
		"%s: field=%s limit=%d descriptors=%d decisions=%d microtasks=%d tasks=%d outbox=%d bytes=%d omitted_decisions=%d omitted_descriptors=%d cause=%s",
		reason,
		strings.TrimSpace(issue.Field),
		issue.Limit,
		issue.Stats.Descriptors,
		issue.Stats.Decisions,
		issue.Stats.CreateMicrotasks,
		issue.Stats.NewTasks,
		issue.Stats.ExpectedOutbox,
		issue.Stats.Bytes,
		issue.Stats.OmittedDecisions,
		issue.Stats.OmittedDescriptors,
		strings.TrimSpace(issue.Stats.Cause),
	)
}

func NormalizeDirectorAgentDecisionBatchBudgetV0(
	budget DirectorAgentDecisionBatchBudgetV0,
) DirectorAgentDecisionBatchBudgetV0 {
	return budget
}

func ValidateDirectorAgentDecisionBatchBudgetV0(
	budget DirectorAgentDecisionBatchBudgetV0,
	stats DirectorAgentDecisionBatchStatsV0,
) error {
	budget = NormalizeDirectorAgentDecisionBatchBudgetV0(budget)
	limits := []struct {
		field string
		got   int
		max   int
	}{
		{"sources", stats.Sources, budget.MaxSources},
		{"descriptors", stats.Descriptors, budget.MaxDescriptors},
		{"decisions", stats.Decisions, budget.MaxDecisions},
		{"create_microtasks", stats.CreateMicrotasks, budget.MaxCreateMicrotasks},
		{"new_tasks", stats.NewTasks, budget.MaxNewTasks},
		{"expected_outbox", stats.ExpectedOutbox, budget.MaxExpectedOutbox},
		{"bytes", stats.Bytes, budget.MaxBytes},
	}
	for _, limit := range limits {
		if limit.max <= 0 || limit.got <= limit.max {
			continue
		}
		stats.Cause = limit.field
		return DirectorAgentDecisionBatchBudgetIssueV0{
			ReasonCode: DirectorAgentDecisionBatchTooLargeReasonV0,
			Field:      limit.field,
			Limit:      limit.max,
			Stats:      stats,
		}
	}
	return nil
}

func CountDirectorAgentDecisionBatchV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) DirectorAgentDecisionBatchStatsV0 {
	stats := DirectorAgentDecisionBatchStatsV0{
		Decisions:      len(decisions),
		ExpectedOutbox: len(decisions),
	}
	existing := map[string]bool{}
	for _, taskRef := range run.Tasks {
		existing[strings.TrimSpace(taskRef)] = true
	}
	seenNew := map[string]bool{}
	for _, decision := range decisions {
		if decision.CreateMicrotask == nil {
			continue
		}
		stats.CreateMicrotasks++
		taskRef := strings.TrimSpace(decision.CreateMicrotask.Task.TaskID)
		if taskRef == "" || existing[taskRef] || seenNew[taskRef] {
			continue
		}
		seenNew[taskRef] = true
		stats.NewTasks++
	}
	return stats
}

func AddDirectorAgentDecisionBatchStatsV0(
	left DirectorAgentDecisionBatchStatsV0,
	right DirectorAgentDecisionBatchStatsV0,
) DirectorAgentDecisionBatchStatsV0 {
	left.Sources += right.Sources
	left.Descriptors += right.Descriptors
	left.Decisions += right.Decisions
	left.CreateMicrotasks += right.CreateMicrotasks
	left.NewTasks += right.NewTasks
	left.ExpectedOutbox += right.ExpectedOutbox
	left.Bytes += right.Bytes
	left.OmittedDecisions += right.OmittedDecisions
	left.OmittedDescriptors += right.OmittedDescriptors
	return left
}
