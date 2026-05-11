package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestMCPCoreWorkflowCommandToolDescriptorV0Compacto(t *testing.T) {
	descriptor := MCPCoreWorkflowCommandToolDescriptorV0Value()
	if descriptor.Name != MCPCoreWorkflowCommandToolNameV0 ||
		descriptor.Version != MCPCoreWorkflowCommandToolVersionV0 ||
		descriptor.ResourceURI != MCPCoreWorkflowContractsResourceURIV0 {
		t.Fatalf("descriptor identidad: %+v", descriptor)
	}
	assertCoreWorkflowCommandToolJSONSaneadoV0(t, descriptor, 900)
}

func TestExecuteMCPCoreWorkflowCommandToolV0StartRunAplicaEstadoCompacto(t *testing.T) {
	command := mustMCPStartRunCommandV0(t, "cmd-core-start", "idem-core-start")
	result := ExecuteMCPCoreWorkflowCommandToolV0(MCPCoreWorkflowCommandToolInputV0{
		RequestID:     "req-core-command",
		CorrelationID: "corr-core-command",
		Command:       command,
	})

	if result.Estado != MCPCoreWorkflowCommandEstadoOKV0 || len(result.Errores) != 0 {
		t.Fatalf("resultado ok esperado: %+v", result)
	}
	if result.Command.CommandType != orquestacoreworkflow.OrchestrationCommandStartRunV0 ||
		len(result.Events) != 1 ||
		result.Events[0] != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		t.Fatalf("eventos inesperados: %+v", result)
	}
	if result.StateAfter.Status != string(orquestacoreworkflow.OrchestrationRunStatusActiveV0) ||
		result.StateAfter.LastSequence != 1 ||
		result.StateAfter.RunID != command.RunID {
		t.Fatalf("state_after compacto inesperado: %+v", result.StateAfter)
	}
	if result.StatePublic != nil {
		t.Fatalf("state_public debe omitirse por defecto")
	}
	assertCoreWorkflowCommandToolJSONSaneadoV0(t, result, 1200)
}

func TestExecuteMCPCoreWorkflowCommandToolV0AskDirectorNoExponePayloads(t *testing.T) {
	current := mustMCPStartedRunV0(t)
	command := mustMCPAskDirectorCommandV0(t, current.RunID)

	result := ExecuteMCPCoreWorkflowCommandToolV0(MCPCoreWorkflowCommandToolInputV0{
		CorrelationID: "corr-director-question",
		Current:       current,
		Command:       command,
	})

	if result.Estado != MCPCoreWorkflowCommandEstadoOKV0 || len(result.Events) != 2 {
		t.Fatalf("resultado pregunta esperado: %+v", result)
	}
	if result.Events[0] != orquestacoreworkflow.OrchestrationEventDirectorQuestionRaisedV0 ||
		result.Events[1] != orquestacoreworkflow.OrchestrationEventRunBlockedV0 {
		t.Fatalf("eventos pregunta inesperados: %+v", result.Events)
	}
	if len(result.Outbox) != 1 ||
		result.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageSendDirectorQuestionV0 ||
		result.Outbox[0].TargetPort != orquestacoreworkflow.OutboxTargetDirectorV0 {
		t.Fatalf("outbox compacto inesperado: %+v", result.Outbox)
	}
	if result.StateAfter.Status != string(orquestacoreworkflow.OrchestrationRunStatusBlockedV0) ||
		result.StateAfter.Blockers != 1 {
		t.Fatalf("state_after no bloqueado: %+v", result.StateAfter)
	}
	assertCoreWorkflowCommandToolJSONSaneadoV0(t, result, 1500)
}

func TestExecuteMCPCoreWorkflowCommandToolV0RecordQualityGate(t *testing.T) {
	current := mustMCPRunWithOpenPhaseV0(t, orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	command := mustMCPRecordQualityGateCommandV0(t, current)

	result := ExecuteMCPCoreWorkflowCommandToolV0(MCPCoreWorkflowCommandToolInputV0{
		CorrelationID: "corr-quality-gate",
		Current:       current,
		Command:       command,
	})

	if result.Estado != MCPCoreWorkflowCommandEstadoOKV0 || len(result.Errores) != 0 {
		t.Fatalf("resultado quality gate esperado: %+v", result)
	}
	if result.Command.CommandType != orquestacoreworkflow.OrchestrationCommandRecordQualityGateV0 ||
		len(result.Events) != 1 ||
		result.Events[0] != orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0 {
		t.Fatalf("evento quality gate inesperado: %+v", result)
	}
	if len(result.Outbox) != 0 || result.StateAfter.QualityGates != 1 {
		t.Fatalf("quality gate debe ser durable y sin outbox: %+v", result)
	}
	assertCoreWorkflowCommandToolJSONSaneadoV0(t, result, 1500)
}

func TestExecuteMCPCoreWorkflowCommandToolV0ErrorPublico(t *testing.T) {
	command := mustMCPStartRunCommandV0(t, "cmd-core-bad", "idem-core-bad")
	command.IdempotencyKey = " "

	result := ExecuteMCPCoreWorkflowCommandToolV0(MCPCoreWorkflowCommandToolInputV0{
		CorrelationID: "corr-bad-command",
		Command:       command,
	})

	if result.Estado != MCPCoreWorkflowCommandEstadoErrorV0 || len(result.Errores) != 1 {
		t.Fatalf("error esperado: %+v", result)
	}
	if result.Errores[0].Code != orquestacoreworkflow.ErrIdempotencyKeyRequeridaV0 ||
		result.Errores[0].Field != "idempotency_key" ||
		result.StateAfter.RunID != "" {
		t.Fatalf("error publico incorrecto: %+v", result)
	}
}

func mustMCPStartedRunV0(t *testing.T) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	command := mustMCPStartRunCommandV0(t, "cmd-core-start-current", "idem-core-start-current")
	result := ExecuteMCPCoreWorkflowCommandToolV0(MCPCoreWorkflowCommandToolInputV0{Command: command})
	if result.Estado != MCPCoreWorkflowCommandEstadoOKV0 {
		t.Fatalf("start result: %+v", result)
	}
	event, err := orquestacoreworkflow.HandleCommandV0(orquestacoreworkflow.OrchestrationRunV0{}, command)
	if err != nil || len(event.Events) != 1 {
		t.Fatalf("handle start: events=%+v err=%v", event.Events, err)
	}
	state, err := orquestacoreworkflow.ApplyEventV0(orquestacoreworkflow.OrchestrationRunV0{}, event.Events[0])
	if err != nil {
		t.Fatalf("apply start: %v", err)
	}
	return state
}

func mustMCPRunWithOpenPhaseV0(
	t *testing.T,
	phaseID orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mustMCPStartedRunV0(t)
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		validMCPCommandMetaV0("cmd-core-open-"+string(phaseID), "idem-core-open-"+string(phaseID), run.RunID),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{PhaseID: string(phaseID), Reason: "test"},
	)
	if err != nil {
		t.Fatalf("open phase command: %v", err)
	}
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle open phase: %v", err)
	}
	state, err := applyCoreWorkflowEventsForMCPV0(run, result.Events)
	if err != nil {
		t.Fatalf("apply open phase: %v", err)
	}
	return state
}

func mustMCPStartRunCommandV0(t *testing.T, commandID string, idem string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewStartRunCommandV0(
		validMCPCommandMetaV0(commandID, idem, "run-core-command"),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-core",
			AppSpecRef: "app-spec-ref-core",
		},
	)
	if err != nil {
		t.Fatalf("start command: %v", err)
	}
	return command
}

func mustMCPRecordQualityGateCommandV0(
	t *testing.T,
	current orquestacoreworkflow.OrchestrationRunV0,
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewRecordQualityGateCommandV0(
		validMCPCommandMetaV0("cmd-core-quality", "idem-core-quality", current.RunID),
		orquestacoreworkflow.RecordQualityGateCommandPayloadV0{
			RunRef:       current.RunID,
			GateRef:      "quality-gate-core",
			PhaseID:      string(current.CurrentPhase),
			SubjectRef:   "subject-core-quality",
			Decision:     orquestacoreworkflow.QualityGateDecisionAcceptedV0,
			Summary:      "Quality gate compacto.",
			EvidenceRefs: []string{"evidence-quality-core"},
		},
	)
	if err != nil {
		t.Fatalf("quality gate command: %v", err)
	}
	return command
}

func mustMCPAskDirectorCommandV0(t *testing.T, runID string) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := orquestacoreworkflow.NewAskDirectorCommandV0(
		validMCPCommandMetaV0("cmd-core-ask", "idem-core-ask", runID),
		orquestacoreworkflow.AskDirectorCommandPayloadV0{
			QuestionID:   "q-core-001",
			SourceGroup:  "grupo-control-test",
			Summary:      "Necesitamos una decision compacta del director.",
			Options:      []string{"continue", "replan"},
			EvidenceRefs: []string{"evidence-ref-001"},
			Blocking:     true,
		},
	)
	if err != nil {
		t.Fatalf("ask command: %v", err)
	}
	return command
}

func validMCPCommandMetaV0(commandID string, idem string, runID string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runID,
		IdempotencyKey: idem,
		CorrelationID:  "corr-core-command",
		RequestedBy:    "director",
		OccurredAt:     "2026-05-06T10:00:00Z",
	}
}

func assertCoreWorkflowCommandToolJSONSaneadoV0(t *testing.T, value any, maxBytes int) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := strings.ToLower(string(payload))
	if len(text) > maxBytes {
		t.Fatalf("payload demasiado largo: %d bytes: %s", len(text), text)
	}
	for _, forbidden := range []string{
		"payload\":",
		"project-ref-core",
		"app-spec-ref-core",
		"necesitamos una decision",
		"evidence-ref-001",
		"sql",
		"dsn",
		"oauth",
		"password",
		"transcript",
	} {
		if strings.Contains(text, strings.ToLower(forbidden)) {
			t.Fatalf("payload contiene fragmento prohibido %q: %s", forbidden, text)
		}
	}
}
