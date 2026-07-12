package orquestamcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
)

func TestMCPAutoprogrammingStatusTransportV0CompactaSalidaBajoLimiteMCP(t *testing.T) {
	diagnostics := make([]MCPAutoprogrammingDiagnosticV0, 60)
	for index := range diagnostics {
		diagnostics[index] = MCPAutoprogrammingDiagnosticV0{
			Code:    fmt.Sprintf("diagnostic-%02d", index),
			Scope:   "transport-test",
			Message: strings.Repeat(fmt.Sprintf("detalle-%02d-", index), 300),
			EvidenceRefs: []string{
				fmt.Sprintf("evidence-%02d-a", index),
				fmt.Sprintf("evidence-%02d-b", index),
				fmt.Sprintf("evidence-%02d-c", index),
			},
		}
	}
	result := MCPAutoprogrammingStatusToolResultV0{
		Estado:      MCPAutoprogrammingStatusEstadoOKV0,
		RequestID:   "request-compact-001",
		Diagnostics: diagnostics,
		EvidenceRefs: []string{
			strings.Repeat("evidence-large-", 5000),
		},
	}
	payload, err := marshalMCPAutoprogrammingStatusTransportV0(result, []MCPAutoprogrammingOperatorAdviceV0{{
		AdviceRef: strings.Repeat("advice-ref-large-", 5000),
		Message:   strings.Repeat("operator-message-large-", 5000),
	}})
	if err != nil {
		t.Fatalf("marshal compact status: %v", err)
	}
	if len(payload) > mcpAutoprogrammingStatusTransportTargetBytesV0 {
		t.Fatalf("payload=%d target=%d", len(payload), mcpAutoprogrammingStatusTransportTargetBytesV0)
	}
	var projected mcpAutoprogrammingStatusTransportProjectedResultV0
	if err := json.Unmarshal(payload, &projected); err != nil {
		t.Fatalf("decode projected status: %v", err)
	}
	projection := projected.OutputProjection
	if projected.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		projection == nil || projection.Mode != "compact" || !projection.Truncated ||
		projection.ObservedBytes <= projection.TargetBytes ||
		projection.ReturnedBytes > projection.TargetBytes ||
		projection.DiagnosticsTotal != len(diagnostics) {
		t.Fatalf("projected=%+v projection=%+v", projected, projection)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(projected.Diagnostics, "mcp_status_output_compacted") {
		t.Fatalf("diagnostics=%+v", projected.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusTransportV0ConservaSalidaPequena(t *testing.T) {
	result := MCPAutoprogrammingStatusToolResultV0{
		Estado:    MCPAutoprogrammingStatusEstadoOKV0,
		RequestID: "request-small-001",
	}
	payload, err := marshalMCPAutoprogrammingStatusTransportV0(result, nil)
	if err != nil {
		t.Fatalf("marshal small status: %v", err)
	}
	var projected mcpAutoprogrammingStatusTransportProjectedResultV0
	if err := json.Unmarshal(payload, &projected); err != nil {
		t.Fatalf("decode small status: %v", err)
	}
	if projected.RequestID != result.RequestID || projected.OutputProjection != nil {
		t.Fatalf("projected=%+v", projected)
	}
}

func TestMCPTransportV0AutoprogrammingStatusQuedaOptInSinPuertos(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.Tool != MCPAutoprogrammingStatusToolNameV0 ||
		result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("autoprogramming status debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0AutoprogrammingStatusPublicaDiagnosticosConfiguradosDesdeBindings(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		AutoprogrammingStatusDiagnostics: []MCPAutoprogrammingDiagnosticV0{{
			Code:         "codex_goal_backend_degraded",
			Scope:        "app_goal",
			Message:      "codex goal backend degradado: codex_app_server_auth_missing",
			EvidenceRefs: []string{"evidence-ref-server-codex-goal-backend-degraded-app_goal"},
		}},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "codex_goal_backend_degraded") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}

func TestMCPTransportV0AutoprogrammingStatusPublicaGoalProgressPolicyDesdeBindings(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		AutoprogrammingGoalProgressPolicy: MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: 33000,
			CheckpointOnlyMaxWaitSeconds:        222,
			NoCheckpointWarningMaxWaitSeconds:   111,
		},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.GoalProgressPolicy == nil ||
		result.GoalProgressPolicy.CheckpointOnlyHighConsumptionTokens != 33000 ||
		result.GoalProgressPolicy.CheckpointOnlyMaxWaitSeconds != 222 ||
		result.GoalProgressPolicy.NoCheckpointWarningMaxWaitSeconds != 111 {
		t.Fatalf("goal_progress_policy=%+v", result.GoalProgressPolicy)
	}
}

func TestMCPTransportV0AutoprogrammingStatusUsaEstadoVivoDesdeBindings(t *testing.T) {
	runRef := "run-ref-estado-vivo-transport-001"
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		AutoprogrammingEstadoVivoSource: &fakeMCPAutoprogrammingEstadoVivoSourceV0{
			evidencias: []orquestaestadovivo.EvidenciaEstadoV0{{
				RunRef:                      runRef,
				Fuente:                      "process_snapshot",
				Estado:                      "running",
				Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
				RuntimeIdentityRef:          "runtime-transport-live",
				RuntimeObservationAttempted: true,
				RuntimeObservado:            true,
				ProcesoVivo:                 true,
				EvidenceRefs:                []string{"evidence-ref-estado-vivo-transport-live"},
			}},
		},
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("call autoprogramming status: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.QueueHealth == nil ||
		result.QueueHealth.RunningLive != 1 {
		t.Fatalf("result=%+v", result)
	}
}
