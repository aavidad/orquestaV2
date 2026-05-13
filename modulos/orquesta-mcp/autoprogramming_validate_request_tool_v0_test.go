package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestMCPAutoprogrammingValidateRequestDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPAutoprogrammingValidateRequestDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingValidateRequestToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingValidateRequestResourceURIV0 ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, descriptor, 1200)
}

func TestMCPAutoprogrammingValidateRequestExecutorV0AceptaSolicitudAislada(t *testing.T) {
	result, err := MCPAutoprogrammingValidateRequestToolExecutorV0{}.Execute(
		context.Background(),
		MCPAutoprogrammingValidateRequestToolInputV0{
			RequestID:              "request-ref-mcp-autoprogramming-001",
			CorrelationID:          "corr-mcp-autoprogramming-001",
			AutoprogrammingRequest: validMCPAutoprogrammingRequestV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingValidateRequestEstadoOKV0 ||
		!result.Accepted ||
		result.RequestID != "request-ref-mcp-autoprogramming-001" ||
		result.CorrelationID != "corr-mcp-autoprogramming-001" {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Groups) != 1 ||
		result.Groups[0].Area != "mcp" ||
		len(result.WriteSet) != 2 ||
		len(result.RequiredTests) != 1 {
		t.Fatalf("proyeccion incompleta=%+v", result)
	}
}

func TestMCPAutoprogrammingValidateRequestExecutorV0DevuelveIssuesPublicos(t *testing.T) {
	request := validMCPAutoprogrammingRequestV0()
	request.BranchRef = " "
	request.WriteSet = []string{"../fuera.go"}

	result, err := MCPAutoprogrammingValidateRequestToolExecutorV0{}.Execute(
		context.Background(),
		MCPAutoprogrammingValidateRequestToolInputV0{
			RequestID:              "request-ref-mcp-autoprogramming-invalid-001",
			AutoprogrammingRequest: request,
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingValidateRequestEstadoErrorV0 || result.Accepted {
		t.Fatalf("result=%+v", result)
	}
	assertMCPAutoprogrammingIssueV0(t, result, "branch_ref_missing")
	assertMCPAutoprogrammingIssueV0(t, result, "write_set_path_invalid")
}

func TestMCPAutoprogrammingValidateRequestTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingValidateRequestToolNameV0,
		MCPAutoprogrammingValidateRequestToolInputV0{
			RequestID:              "request-ref-mcp-autoprogramming-transport-001",
			AutoprogrammingRequest: validMCPAutoprogrammingRequestV0(),
		},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPAutoprogrammingValidateRequestToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingValidateRequestEstadoOKV0 || !result.Accepted {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 1400)
}

func validMCPAutoprogrammingRequestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	return orquestaautoprogramming.AutoprogrammingRequestV0{
		ProjectRef:       "project-ref-orquesta",
		WorktreeRef:      "worktree-ref-orquesta-aislada-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-autoprogramming-orquesta-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-mcp-autoprogramming-a", Area: "MCP"},
			{TaskRef: "task-ref-mcp-autoprogramming-b", Area: "mcp"},
		},
		WriteSet: []string{
			"modulos/orquesta-mcp/autoprogramming_validate_request_tool_v0.go",
			"modulos/orquesta-mcp/autoprogramming_validate_request_tool_v0_test.go",
		},
		RequiredTests: []string{
			"go test -count=1 ./modulos/orquesta-mcp",
		},
	}
}

func assertMCPAutoprogrammingIssueV0(
	t *testing.T,
	result MCPAutoprogrammingValidateRequestToolResultV0,
	code string,
) {
	t.Helper()
	for _, issue := range result.Errores {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrado: %+v", code, result.Errores)
}
