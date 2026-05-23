package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestMCPAutoprogrammingSelfImprovementDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPAutoprogrammingSelfImprovementDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingSelfImprovementToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingSelfImprovementResourceURIV0 ||
		descriptor.InputSchema == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, descriptor, 1200)
}

func TestMCPAutoprogrammingSelfImprovementExecutorV0DevuelvePrepareRunBajaPrioridad(t *testing.T) {
	result, err := MCPAutoprogrammingSelfImprovementToolExecutorV0{}.Execute(
		context.Background(),
		MCPAutoprogrammingSelfImprovementToolInputV0{
			RequestID:     "request-ref-self-improvement-mcp-001",
			CorrelationID: "corr-self-improvement-mcp-001",
			Proposal:      validMCPSelfImprovementProposalV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingSelfImprovementEstadoOKV0 ||
		!result.Accepted ||
		!result.Background ||
		result.PriorityScore != orquestaautoprogramming.AutoprogrammingSelfImprovementDefaultPriorityScoreV0 ||
		result.AutoprogrammingRequest == nil ||
		result.PrepareRun == nil {
		t.Fatalf("result=%+v", result)
	}
	if result.PrepareRun.PriorityScore != result.PriorityScore ||
		result.PrepareRun.AutoprogrammingRequest.RequestRef != result.AutoprogrammingRequest.RequestRef {
		t.Fatalf("prepare_run=%+v request=%+v", result.PrepareRun, result.AutoprogrammingRequest)
	}
}

func TestMCPAutoprogrammingSelfImprovementExecutorV0AutoPrepareRunOptIn(t *testing.T) {
	executor := NewMCPAutoprogrammingSelfImprovementToolExecutorV0(
		recordingSelfImprovementPrepareRunExecutorV0{},
	)
	result, err := executor.Execute(
		context.Background(),
		MCPAutoprogrammingSelfImprovementToolInputV0{
			RequestID:      "request-ref-self-improvement-auto-001",
			AutoPrepareRun: true,
			Proposal:       validMCPSelfImprovementProposalV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingSelfImprovementEstadoOKV0 ||
		result.PreparedRun == nil ||
		!result.PreparedRun.Accepted ||
		result.PreparedRun.RunRef == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSelfImprovementExecutorV0AutoPrepareRunSinPuertoReparable(t *testing.T) {
	result, err := MCPAutoprogrammingSelfImprovementToolExecutorV0{}.Execute(
		context.Background(),
		MCPAutoprogrammingSelfImprovementToolInputV0{
			RequestID:      "request-ref-self-improvement-auto-missing-001",
			AutoPrepareRun: true,
			Proposal:       validMCPSelfImprovementProposalV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingSelfImprovementEstadoOKV0 ||
		!result.Accepted ||
		result.PrepareRun == nil ||
		result.PreparedRun != nil ||
		len(result.Errores) == 0 ||
		!stringsSliceContainsMCPHumanWorkV0(result.NextActions, "repair_configure_autoprogramming_prepare_run_executor") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingSelfImprovementExecutorV0ConservaEvidenciaSiFaltanRefs(t *testing.T) {
	proposal := validMCPSelfImprovementProposalV0()
	proposal.WorktreeRef = " "
	proposal.BranchRef = " "
	proposal.WorktreeIsolated = false

	result, err := MCPAutoprogrammingSelfImprovementToolExecutorV0{}.Execute(
		context.Background(),
		MCPAutoprogrammingSelfImprovementToolInputV0{
			RequestID: "request-ref-self-improvement-mcp-invalid-001",
			Proposal:  proposal,
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingSelfImprovementEstadoErrorV0 ||
		result.Accepted ||
		!result.Background ||
		result.PrepareRun != nil ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if !stringsSliceContainsMCPHumanWorkV0(result.NextActions, "preserve_failure_evidence") {
		t.Fatalf("next_actions=%v", result.NextActions)
	}
}

type recordingSelfImprovementPrepareRunExecutorV0 struct{}

func (recordingSelfImprovementPrepareRunExecutorV0) Execute(
	_ context.Context,
	input MCPAutoprogrammingPrepareRunToolInputV0,
) (MCPAutoprogrammingPrepareRunToolResultV0, error) {
	return MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:        MCPAutoprogrammingPrepareRunEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Accepted:      true,
		RunRef:        "run-ref-" + input.AutoprogrammingRequest.RequestRef,
		ProjectRef:    input.AutoprogrammingRequest.ProjectRef,
		WorktreeRef:   input.AutoprogrammingRequest.WorktreeRef,
		BranchRef:     input.AutoprogrammingRequest.BranchRef,
	}, nil
}

func TestMCPAutoprogrammingSelfImprovementTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingSelfImprovementToolNameV0,
		MCPAutoprogrammingSelfImprovementToolInputV0{
			RequestID: "request-ref-self-improvement-transport-001",
			Proposal:  validMCPSelfImprovementProposalV0(),
		},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPAutoprogrammingSelfImprovementToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingSelfImprovementEstadoOKV0 || result.PrepareRun == nil {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 3200)
}

func TestMCPAutoprogrammingSelfImprovementHTTPHandlerV0(t *testing.T) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID: "request-ref-self-improvement-http-001",
		Proposal:  validMCPSelfImprovementProposalV0(),
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingSelfImprovementHTTPPathV0, body)

	NewMCPAutoprogrammingSelfImprovementHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingSelfImprovementToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingSelfImprovementEstadoOKV0 || result.PrepareRun == nil {
		t.Fatalf("result=%+v", result)
	}
}

func validMCPSelfImprovementProposalV0() orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 {
	return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
		ProjectRef:         "project-ref-orquesta",
		WorktreeRef:        "worktree-ref-self-improvement-mcp-001",
		WorktreeIsolated:   true,
		BranchRef:          "branch-ref-self-improvement-mcp-001",
		ObservedBy:         "director",
		SourceRunRef:       "run-ref-primary-mcp-001",
		SourceTaskRef:      "task-ref-primary-mcp-001",
		FailureKind:        "mcp",
		FailureSummary:     "El agente detecto una correccion reutilizable.",
		SuggestedArea:      "MCP",
		SuggestedWriteSet:  []string{"modulos/orquesta-mcp/autoprogramming_self_improvement_tool_v0.go"},
		RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-mcp"},
		AcceptanceCriteria: []string{"la automejora queda como trabajo secundario"},
		CompactRules:       []string{"no bloquear el trabajo principal"},
		EvidenceRefs:       []string{"evidence-ref-self-improvement-mcp-001"},
	}
}
