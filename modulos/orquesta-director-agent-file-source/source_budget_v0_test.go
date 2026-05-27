package orquestadirectoragentfilesource

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestDirectorAgentDecisionFileSourceV0LimitaDescriptorsPorRequest(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-budget-001"),
	})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{
				{RunID: "run-ref-budget-001", Path: "one.json"},
				{RunID: "run-ref-budget-001", Path: "two.json"},
			},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"one.json": data, "two.json": data},
		},
		BatchBudget: orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetV0{
			MaxDescriptors: 1,
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-budget-001"),
	)

	assertDirectorDecisionBatchTooLargeForTestV0(t, err, "descriptors")
}

func TestDirectorAgentDecisionFileSourceV0LimitaDecisionsAcumuladas(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-budget-002"),
		validOpenVoteDecisionForTestV0("run-ref-budget-002"),
	})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-budget-002",
				Path:  "decisions.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"decisions.json": data},
		},
		BatchBudget: orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetV0{
			MaxDecisions: 1,
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-budget-002"),
	)

	assertDirectorDecisionBatchTooLargeForTestV0(t, err, "decisions")
}

func assertDirectorDecisionBatchTooLargeForTestV0(t *testing.T, err error, field string) {
	t.Helper()
	var issue orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetIssueV0
	if !errors.As(err, &issue) {
		t.Fatalf("err=%T %[1]v", err)
	}
	if issue.Field != field ||
		!strings.Contains(err.Error(), orquestadirectoragentworkflow.DirectorAgentDecisionBatchTooLargeReasonV0) {
		t.Fatalf("issue=%+v err=%v", issue, err)
	}
}
