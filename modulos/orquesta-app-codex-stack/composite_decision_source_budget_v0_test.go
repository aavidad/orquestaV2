package orquestaappcodexstack

import (
	"context"
	"errors"
	"testing"

	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestCompositeDirectorDecisionSourceV0LimitaDecisionsAgregadas(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-budget-001",
		"brainstorm-ref-stack-budget-001",
	)
	source := compositeDirectorDecisionSourceV0{
		Budget: orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetV0{
			MaxDecisions: len(decisions) - 1,
		},
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	var issue orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetIssueV0
	if !errors.As(err, &issue) {
		t.Fatalf("err=%T %[1]v", err)
	}
	if issue.Field != "decisions" || issue.Stats.OmittedDecisions != len(decisions) {
		t.Fatalf("issue=%+v", issue)
	}
}

func TestCompositeDirectorDecisionSourceV0LimitaSources(t *testing.T) {
	source := compositeDirectorDecisionSourceV0{
		Budget: orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetV0{
			MaxSources: 1,
		},
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{},
			codexStackStaticDecisionSourceForTestV0{},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	var issue orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetIssueV0
	if !errors.As(err, &issue) || issue.Field != "sources" {
		t.Fatalf("issue=%+v err=%v", issue, err)
	}
}
