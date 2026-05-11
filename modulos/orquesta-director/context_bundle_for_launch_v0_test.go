package orquestadirector

import (
	"encoding/json"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildLaunchContextBundleV0DesdeOutboxLaunch(t *testing.T) {
	result, err := BuildLaunchContextBundleV0(validLaunchContextBundleInputV0(t))
	if err != nil {
		t.Fatalf("BuildLaunchContextBundleV0: %v %+v", err, result.Issues)
	}
	bundle := result.Bundle
	if !bundle.Valid() {
		t.Fatalf("bundle invalido: %+v", bundle.Issues)
	}
	if bundle.WorkOrderRef != "agent-request-context-001" || bundle.TargetModule != "orquesta-runtime" {
		t.Fatalf("bundle inesperado: %+v", bundle)
	}
	requireLaunchContextEntryV0(t, bundle, "modulos/orquesta-runtime/AGENTS.md")
	requireLaunchContextEntryV0(t, bundle, "modulos/orquesta-runtime/docs/pruebas.md")
	requireLaunchContextEntryV0(t, bundle, "process_runtime_connector_v0.go")
	requireLaunchContextEntryV0(t, bundle, "ProcessRuntimeConnectorV0")
	requireLaunchContextEntryV0(t, bundle, "orquesta-capacity:CapacityDecisionV0")
	if !strings.Contains(bundle.Summary.DirectorQuestionPolicy, "CONSULTA_AL_DIRECTOR") {
		t.Fatalf("politica director ausente: %+v", bundle.Summary)
	}
}

func TestBuildLaunchContextBundleV0RechazaOutboxNoLaunch(t *testing.T) {
	input := validLaunchContextBundleInputV0(t)
	input.Message.MessageType = orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0

	result, err := BuildLaunchContextBundleV0(input)
	if err == nil {
		t.Fatalf("expected error, result=%+v", result)
	}
	requireLaunchContextIssueV0(t, result.Issues, ErrDirectorContextBundleOutboxV0)
}

func TestBuildLaunchContextBundleV0RechazaSinWriteSet(t *testing.T) {
	input := validLaunchContextBundleInputV0(t)
	input.WriteSet = nil

	result, err := BuildLaunchContextBundleV0(input)
	if err == nil {
		t.Fatalf("expected error, result=%+v", result)
	}
	requireLaunchContextIssueV0(t, result.Issues, string(orquestacontext.ErrContextBundleCampoRequeridoV0))
}

func validLaunchContextBundleInputV0(t *testing.T) LaunchContextBundleInputV0 {
	t.Helper()
	payload := orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-context-001",
		RunID:              "run-ref-context-001",
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ref-context-001",
		CapacityRequestRef: "capacity-ref-context-001",
		Role:               "builder",
		Summary:            "Preparar lanzamiento con contexto pequeno.",
		EvidenceRefs:       []string{"evidence-ref-context-001"},
	}
	return LaunchContextBundleInputV0{
		Message: orquestacoreworkflow.OutboxMessageV0{
			MessageID:        "outbox-context-001",
			MessageType:      orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
			RunID:            "run-ref-context-001",
			IdempotencyKey:   "idem-context-001",
			CorrelationID:    "corr-context-001",
			CausationEventID: "event-context-001",
			TargetPort:       orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			PayloadVersion:   orquestacoreworkflow.OutboxPayloadVersionV0,
			Payload:          mustLaunchContextPayloadV0(t, payload),
		},
		TargetModule:    "orquesta-runtime",
		TaskKind:        "microtarea_codigo",
		Objective:       "Construir request runtime con contexto resuelto.",
		CapacityLevel:   "high",
		ReadSet:         []string{"runtime_launch_request_types_v0.go"},
		WriteSet:        []string{"process_runtime_connector_v0.go", "process_runtime_connector_v0_test.go"},
		ContractRefs:    []string{"ProcessRuntimeConnectorV0"},
		CrossModuleRefs: []string{"orquesta-capacity:CapacityDecisionV0"},
	}
}

func mustLaunchContextPayloadV0(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func requireLaunchContextEntryV0(t *testing.T, bundle orquestacontext.ContextBundleV0, source string) {
	t.Helper()
	for _, entry := range bundle.Entries {
		if entry.SourceRef == source {
			return
		}
	}
	t.Fatalf("source %q no encontrada en %+v", source, bundle.Entries)
}

func requireLaunchContextIssueV0(t *testing.T, issues []LaunchContextBundleIssueV0, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrada en %+v", code, issues)
}
