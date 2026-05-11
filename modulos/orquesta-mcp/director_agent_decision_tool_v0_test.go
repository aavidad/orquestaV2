package orquestamcp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPDirectorAgentDecisionToolExecutorV0AplicaDecision(t *testing.T) {
	run := mcpDirectorDecisionRunForTestV0(t, "run-mcp-director-decision-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	executor := NewMCPDirectorAgentDecisionToolExecutorV0(
		orquestadirectoragentworkflow.ApplyDirectorAgentDecisionPortsV0{
			RunStore:  store,
			EventSink: sink,
		},
	)

	result, err := executor.Execute(
		context.Background(),
		validMCPDirectorDecisionInputForTestV0("run-mcp-director-decision-001"),
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorAgentDecisionEstadoOKV0 ||
		result.RunRef != "run-mcp-director-decision-001" ||
		result.Command.CommandType != orquestacoreworkflow.OrchestrationCommandRequestBrainstormV0 ||
		result.EventsCount != 1 {
		t.Fatalf("result=%+v", result)
	}
	if !mcpDirectorDecisionSinkHasEventV0(sink, orquestacoreworkflow.OrchestrationEventBrainstormRequestedV0) {
		t.Fatalf("sink sin BrainstormRequested: %+v", sink.EventsV0())
	}
}

func TestMCPDirectorAgentDecisionToolExecutorV0DevuelveIssuesPublicos(t *testing.T) {
	input := validMCPDirectorDecisionInputForTestV0("run-mcp-director-decision-002")
	input.OccurredAt = ""

	result, err := NewMCPDirectorAgentDecisionToolExecutorV0(
		orquestadirectoragentworkflow.ApplyDirectorAgentDecisionPortsV0{},
	).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorAgentDecisionEstadoErrorV0 || len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func validMCPDirectorDecisionInputForTestV0(runRef string) MCPDirectorAgentDecisionToolInputV0 {
	return MCPDirectorAgentDecisionToolInputV0{
		RequestID:     "request-ref-director-decision-001",
		CorrelationID: "corr-director-decision-001",
		OccurredAt:    "2026-05-09T23:40:00Z",
		Decision: orquestadirectoragent.DirectorAgentDecisionV0{
			SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
			DecisionRef:   "director-decision-ref-001",
			RunID:         runRef,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			CommandType:   orquestadirectoragent.DirectorAgentCommandRequestBrainstormV0,
			CommandRef:    "command-ref-director-brainstorm-001",
			Summary:       "Proponer siguiente brainstorm compacto.",
			EvidenceRefs:  []string{"evidence-ref-director-decision-001"},
			RequestBrainstorm: &orquestadirectoragent.DirectorAgentBrainstormCommandV0{
				BrainstormRequestID:        "brainstorm-ref-director-extra-001",
				PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
				TopicRef:                   "topic-ref-director-extra-001",
				Summary:                    "Analizar alternativa compacta.",
				MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
				EvidenceRefs:               []string{"evidence-ref-director-extra-001"},
			},
		},
	}
}

func mcpDirectorDecisionRunForTestV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	run = mcpDirectorDecisionApplyForTestV0(t, run, mcpDirectorDecisionStartRunCommandV0(t, runRef))
	run = mcpDirectorDecisionApplyForTestV0(t, run, mcpDirectorDecisionOpenBrainstormCommandV0(t, runRef))
	return run
}

func mcpDirectorDecisionStartRunCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-start-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-start-" + runRef,
			OccurredAt:     "2026-05-09T23:35:00Z",
		},
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-mcp-director-decision",
			AppSpecRef: "spec-ref-mcp-director-decision",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func mcpDirectorDecisionOpenBrainstormCommandV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-brainstorm-" + runRef,
			RunID:          runRef,
			IdempotencyKey: "idem-open-brainstorm-" + runRef,
			OccurredAt:     "2026-05-09T23:36:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
			Reason:  "Preparar brainstorm.",
		},
	)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	return command
}

func mcpDirectorDecisionApplyForTestV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle command: %v", err)
	}
	for _, event := range result.Events {
		run, err = orquestacoreworkflow.ApplyEventV0(run, event)
		if err != nil {
			t.Fatalf("apply event: %v", err)
		}
	}
	return run
}

func mcpDirectorDecisionSinkHasEventV0(
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	eventType string,
) bool {
	for _, event := range sink.EventsV0() {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
