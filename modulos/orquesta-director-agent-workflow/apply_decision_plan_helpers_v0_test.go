package orquestadirectoragentworkflow

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func directorAgentWorkflowPlanningRunForTestV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowStartRunCommandV0(t, runRef))
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowOpenVoteCommandV0(t, runRef))
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowRequestVoteCommandV0(t, runRef))
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowAcceptDecisionCommandV0(t, runRef))
	run = directorAgentWorkflowApplyForTestV0(t, run, directorAgentWorkflowOpenPlanningCommandV0(t, runRef))
	return run
}

func directorAgentWorkflowOpenVoteCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-vote-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-open-vote-" + runRef,
			OccurredAt:     "2026-05-09T23:02:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: "votacion_y_decision",
			Reason:  "Preparar decision.",
		},
	)
	if err != nil {
		t.Fatalf("open vote command: %v", err)
	}
	return command
}

func directorAgentWorkflowRequestVoteCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRequestVoteCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-request-vote-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-request-vote-" + runRef,
			OccurredAt:     "2026-05-09T23:03:00Z",
		},
		orquestacoreworkflow.RequestVoteCommandPayloadV0{
			VoteRequestID:              "vote-ref-architecture-001",
			PhaseID:                    "votacion_y_decision",
			DecisionTopicRef:           "topic-ref-architecture-001",
			BrainstormRef:              "brainstorm-ref-architecture-001",
			Summary:                    "Elegir arquitectura hexagonal e i18n obligatorio.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-vote-001"},
		},
	)
	if err != nil {
		t.Fatalf("request vote command: %v", err)
	}
	return command
}

func directorAgentWorkflowAcceptDecisionCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewAcceptDecisionCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-accept-decision-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-accept-decision-" + runRef,
			OccurredAt:     "2026-05-09T23:04:00Z",
		},
		orquestacoreworkflow.AcceptDecisionCommandPayloadV0{
			DecisionRef:       "decision-ref-architecture-001",
			PhaseID:           "votacion_y_decision",
			VoteRef:           "vote-ref-architecture-001",
			AcceptedOptionRef: "option:hexagonal_i18n_connectors",
			Summary:           "Aceptar arquitectura hexagonal con i18n y conectores en periferia.",
			EvidenceRefs:      []string{"evidence-ref-decision-001"},
		},
	)
	if err != nil {
		t.Fatalf("accept decision command: %v", err)
	}
	return command
}

func directorAgentWorkflowOpenPlanningCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-planning-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-open-planning-" + runRef,
			OccurredAt:     "2026-05-09T23:05:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: "planificacion_microtareas",
			Reason:  "Preparar microtareas.",
		},
	)
	if err != nil {
		t.Fatalf("open planning command: %v", err)
	}
	return command
}
