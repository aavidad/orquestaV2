package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPAutoprogrammingPrepareRunDescriptorV0EsAdaptadorOptIn(t *testing.T) {
	descriptor := MCPAutoprogrammingPrepareRunDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingPrepareRunToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingPrepareRunResourceURIV0 ||
		descriptor.InputSchema == "" ||
		descriptor.Output == "" {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if len(descriptor.Invariantes) == 0 {
		t.Fatalf("invariantes vacias")
	}
	if !strings.Contains(descriptor.InputSchema, "autoprogramming_request:AutoprogrammingRequestV0") {
		t.Fatalf("descriptor debe publicar autoprogramming_request como objeto tipado, no string generico: %q", descriptor.InputSchema)
	}
}

func TestNewMCPAutoprogrammingPrepareRunErrorResultV0NormalizaErrorPublico(t *testing.T) {
	result := NewMCPAutoprogrammingPrepareRunErrorResultV0(
		MCPAutoprogrammingPrepareRunToolInputV0{
			RequestID:     "request-prepare-run-001",
			CorrelationID: "corr-prepare-run-001",
		},
		"",
		"executor",
		"",
	)

	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		result.RequestID != "request-prepare-run-001" ||
		result.CorrelationID != "corr-prepare-run-001" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code == "" ||
		result.Errores[0].Message == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingPrepareRunTransportV0BoundInvocaExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunTransportExecutorV0{
		result: MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:           MCPAutoprogrammingPrepareRunEstadoOKV0,
			RequestID:        "request-ref-prepare-run-bound-001",
			CorrelationID:    "corr-prepare-run-bound-001",
			Accepted:         true,
			RunRef:           "run-ref-prepare-run-bound-001",
			WorkflowTaskRefs: []string{"workflow-task-ref-001"},
			WaitAgentRefs:    []string{"agent-request-ref-001"},
		},
	}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		AutoprogrammingPrepareRun: executor,
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	tool, ok := transport.tools[MCPAutoprogrammingPrepareRunToolNameV0]
	if !ok {
		t.Fatalf("tool no registrado: %s", MCPAutoprogrammingPrepareRunToolNameV0)
	}
	descriptor := MCPAutoprogrammingPrepareRunDescriptorV0()
	if tool.ResourceURI != descriptor.ResourceURI ||
		tool.InputShape != descriptor.InputSchema ||
		tool.OutputShape != descriptor.Output ||
		tool.Mode != MCPTransportModeOptInV0 {
		t.Fatalf("envelope inesperado: %+v", tool)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingPrepareRunToolNameV0,
		MCPAutoprogrammingPrepareRunToolInputV0{
			RequestID:              "request-ref-prepare-run-bound-001",
			CorrelationID:          "corr-prepare-run-bound-001",
			AutoprogrammingRequest: validMCPAutoprogrammingRequestV0(),
		},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if executor.called != 1 ||
		executor.input.RequestID != "request-ref-prepare-run-bound-001" ||
		executor.input.CorrelationID != "corr-prepare-run-bound-001" {
		t.Fatalf("executor no invocado correctamente: called=%d input=%+v", executor.called, executor.input)
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingPrepareRunEstadoOKV0 ||
		!result.Accepted ||
		result.RunRef != "run-ref-prepare-run-bound-001" ||
		len(result.WaitAgentRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 1400)
}

type fakeMCPAutoprogrammingPrepareRunTransportExecutorV0 struct {
	called int
	input  MCPAutoprogrammingPrepareRunToolInputV0
	result MCPAutoprogrammingPrepareRunToolResultV0
}

func (executor *fakeMCPAutoprogrammingPrepareRunTransportExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingPrepareRunToolInputV0,
) (MCPAutoprogrammingPrepareRunToolResultV0, error) {
	_ = ctx
	executor.called++
	executor.input = input
	return executor.result, nil
}
