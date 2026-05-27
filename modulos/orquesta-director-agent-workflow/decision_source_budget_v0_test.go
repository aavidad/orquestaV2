package orquestadirectoragentworkflow

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestDirectorAgentDecisionBatchBudgetV0CuentaMicrotareasNuevas(t *testing.T) {
	decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
		decisionSourceBudgetMicrotaskForTestV0("task-ref-existing-001"),
		decisionSourceBudgetMicrotaskForTestV0("task-ref-new-001"),
		decisionSourceBudgetMicrotaskForTestV0("task-ref-new-001"),
	}
	stats := CountDirectorAgentDecisionBatchV0(
		orquestacoreworkflow.OrchestrationRunV0{
			Tasks: []string{"task-ref-existing-001"},
		},
		decisions,
	)

	if stats.Decisions != 3 || stats.CreateMicrotasks != 3 || stats.NewTasks != 1 ||
		stats.ExpectedOutbox != 3 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestDirectorAgentDecisionBatchBudgetV0EmiteReasonPublico(t *testing.T) {
	err := ValidateDirectorAgentDecisionBatchBudgetV0(
		DirectorAgentDecisionBatchBudgetV0{MaxDecisions: 1},
		DirectorAgentDecisionBatchStatsV0{Decisions: 2, OmittedDecisions: 2},
	)

	var issue DirectorAgentDecisionBatchBudgetIssueV0
	if !errors.As(err, &issue) {
		t.Fatalf("err=%T %[1]v", err)
	}
	if issue.ReasonCode != DirectorAgentDecisionBatchTooLargeReasonV0 ||
		!strings.Contains(err.Error(), "omitted_decisions=2") {
		t.Fatalf("issue=%+v err=%v", issue, err)
	}
}

func TestDirectorAgentDecisionBatchBudgetV0DefaultNoLimitaBacklog(t *testing.T) {
	decisions := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, 128)
	for i := 0; i < 128; i++ {
		decisions = append(decisions, decisionSourceBudgetMicrotaskForTestV0(
			"task-ref-resigrx-rx-"+strconv.Itoa(i),
		))
	}
	stats := CountDirectorAgentDecisionBatchV0(
		orquestacoreworkflow.OrchestrationRunV0{},
		decisions,
	)

	if err := ValidateDirectorAgentDecisionBatchBudgetV0(
		DirectorAgentDecisionBatchBudgetV0{},
		stats,
	); err != nil {
		t.Fatalf("default budget rejected backlog: %v", err)
	}
}

func decisionSourceBudgetMicrotaskForTestV0(
	taskRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		CommandType: orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				TaskID: taskRef,
			},
		},
	}
}
