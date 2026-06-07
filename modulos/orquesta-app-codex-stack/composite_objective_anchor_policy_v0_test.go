package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestCompositeDirectorDecisionSourceV0NoBloqueaMicrotareaSinObjetivoActual(t *testing.T) {
	source := codexStackAnchoredSourceForTestV0(
		codexStackDirectorDecisionsForTestV0(
			"run-ref-anchor-missing-001",
			"brainstorm-ref-anchor-missing-001",
		),
	)

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		codexStackAnchorRequestForTestV0("validacion autoprogramming gateway"),
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(decisions) == 0 {
		t.Fatalf("decisions vacias")
	}
}

func TestCompositeDirectorDecisionSourceV0NoBloqueaObjetivoActualDesviado(t *testing.T) {
	decisions := codexStackMutateFirstMicrotaskForPolicyTestV0(
		codexStackDirectorDecisionsForTestV0(
			"run-ref-anchor-token-001",
			"brainstorm-ref-anchor-token-001",
		),
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.AcceptanceCriteria = append(task.AcceptanceCriteria, "objetivo_actual: agenda rest")
		},
	)
	source := codexStackAnchoredSourceForTestV0(decisions)

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		codexStackAnchorRequestForTestV0("validacion autoprogramming gateway"),
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) == 0 {
		t.Fatalf("decisions vacias")
	}
}

func TestCompositeDirectorDecisionSourceV0AceptaObjetivoActualAnclado(t *testing.T) {
	decisions := codexStackMutateFirstMicrotaskForPolicyTestV0(
		codexStackDirectorDecisionsForTestV0(
			"run-ref-anchor-ok-001",
			"brainstorm-ref-anchor-ok-001",
		),
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.AcceptanceCriteria = append(
				task.AcceptanceCriteria,
				"objetivo_actual: exponer validacion autoprogramming por gateway",
			)
		},
	)
	source := codexStackAnchoredSourceForTestV0(decisions)

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		codexStackAnchorRequestForTestV0("validacion autoprogramming gateway"),
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(decisions) == 0 {
		t.Fatalf("decisions vacias")
	}
}

func codexStackAnchoredSourceForTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) compositeDirectorDecisionSourceV0 {
	return compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}
}

func codexStackAnchorRequestForTestV0(
	objective string,
) orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0 {
	return orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
		RequestKind:    "modificar_app_existente",
		ExecutionMode:  "normal",
		ObjectiveHints: []string{objective},
	}
}
