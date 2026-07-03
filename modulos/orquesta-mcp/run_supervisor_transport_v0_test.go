package orquestamcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestMCPRunSupervisorDescriptorV0DeclaraEvidenciaEnErrores(t *testing.T) {
	descriptor := MCPRunSupervisorDescriptorV0()
	if !strings.Contains(descriptor.Output, "error:{errores_publicos,evidence_refs?") ||
		!strings.Contains(descriptor.Output, "diagnostics?") ||
		!strings.Contains(descriptor.Output, "operation_ref?") ||
		!strings.Contains(descriptor.Output, "repair_run_refs?") {
		t.Fatalf("descriptor runs.supervisor debe declarar evidencia/diagnosticos en errores: %+v", descriptor)
	}
}

func TestMCPRunSupervisorTransportHandlerV0PropagaErrorPublicoDelExecutor(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-transport-error-001",
		CorrelationID: "corr-run-supervisor-transport-error-001",
		RunRef:        "run-ref-supervisor-transport-error-001",
	}
	executor := fakeMCPRunSupervisorTransportExecutorV0{
		result: NewMCPRunSupervisorErrorResultV0(
			input,
			"run_supervisor_execute_error",
			"executor",
			"delivery_ack_ingestion_failed",
		),
		err: errors.New("internal stack error must not become mcp_tool_handler_error"),
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	payload, err := mcpRunSupervisorTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler devolvio error de transporte: %v", err)
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal result: %v payload=%s", err, string(payload))
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_supervisor_execute_error" ||
		result.Errores[0].Message != "delivery_ack_ingestion_failed" {
		t.Fatalf("error publico perdido: %+v", result)
	}
}

func TestMCPRunSupervisorTransportHandlerV0DevuelvePayloadPublicoSiExecutorNoDaResultado(t *testing.T) {
	input := MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-transport-generic-error-001",
		CorrelationID: "corr-run-supervisor-transport-generic-error-001",
		RunRef:        "run-ref-supervisor-transport-generic-error-001",
	}
	executor := fakeMCPRunSupervisorTransportExecutorV0{
		err: errors.New("codex failed in /root/Trabajo/orquesta with token=secret123456"),
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}

	payload, err := mcpRunSupervisorTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler no debe romper transporte: %v", err)
	}
	var result MCPRunSupervisorToolResultV0
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal result: %v payload=%s", err, string(payload))
	}
	if result.Estado != MCPRunSupervisorEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_supervisor_execute_error" ||
		result.Errores[0].Message != "run_supervisor_execute_error" {
		t.Fatalf("payload publico perdido: %+v", result)
	}
	if result.OperationRef == "" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-supervisor-execute-error") ||
		!containsStringMCPTestV0(result.EvidenceRefs, result.OperationRef) ||
		!hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_supervisor_execute_error") ||
		!containsStringMCPTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-run-supervisor-execute-error") {
		t.Fatalf("payload publico debe conservar evidencia/diagnostico: %+v", result)
	}
	if strings.Contains(string(payload), "/root/Trabajo") ||
		strings.Contains(string(payload), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", string(payload))
	}
}

type fakeMCPRunSupervisorTransportExecutorV0 struct {
	result MCPRunSupervisorToolResultV0
	err    error
}

func (executor fakeMCPRunSupervisorTransportExecutorV0) Execute(
	_ context.Context,
	_ MCPRunSupervisorToolInputV0,
) (MCPRunSupervisorToolResultV0, error) {
	return executor.result, executor.err
}
