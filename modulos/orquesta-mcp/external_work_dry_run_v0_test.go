package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestBuildExternalWorkDryRunV0CompilaMismoSpecGoalFirstSinLanzarV0(t *testing.T) {
	config := orquestaexternalworkrun.StartExternalWorkRunConfigV0{
		OccurredAt:  "2026-06-27T10:00:00Z",
		RequestedBy: "test",
	}
	input := validExternalWorkDryRunInputMCPTestV0()
	input.Model = "gpt-test"

	result := BuildExternalWorkDryRunV0(input, config)

	if result.Estado != MCPExternalWorkRunEstadoOKV0 ||
		result.RoutePolicy != MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.DirectorExecutionMode != MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 {
		t.Fatalf("result=%+v", result)
	}
	request := orquestaexternalworkrun.PrepareStartExternalWorkRunRequestV0(
		externalWorkRunRequestFromMCPV0(input.MCPExternalWorkRunToolInputV0),
		config,
	)
	want, issues := orquestaexternalworkrun.BuildExternalWorkGoalWorkSpecV0(request, config)
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	want.DirectorKind = orquestagoal.GoalDirectorKindCodexGoalV0
	want = orquestagoal.NormalizeGoalWorkSpecV0(want)
	wantSummary := mcpGoalWorkSpecSummaryV0(want)
	if result.SpecSummary.GoalRef != want.GoalRef ||
		result.SpecSummary.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		result.SpecSummary.SpecHash != wantSummary.SpecHash ||
		result.SpecSummary.WriteSetCount != len(want.WriteSet) ||
		result.SpecSummary.RequiredTestCount != len(want.RequiredTests) ||
		!mcpExternalWorkRunStringInSetTestV0(result.SpecSummary.RequiredTestRefs, "domain-test-ref-qc") {
		t.Fatalf("spec_summary dry-run no coincide\nwant=%+v\ngot=%+v", wantSummary, result.SpecSummary)
	}
	if result.EstModel != "gpt-test" || result.EstTokens <= 0 || result.EstCostUSD <= 0 || result.EstWallClock == "" {
		t.Fatalf("estimate invalida: model=%q tokens=%d cost=%f wall=%q", result.EstModel, result.EstTokens, result.EstCostUSD, result.EstWallClock)
	}
	if !mcpExternalWorkRunStringInSetTestV0(result.EvidenceRefs, MCPExternalWorkDryRunEvidenceRefV0) {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	for _, forbidden := range []string{`"spec":`, `"objective":`, `"write_set":[`, `"required_tests":[`, "go test -count=1 ./modulos/orquesta-mcp", "valor-privado", "no-copiar"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("dry-run publico filtra %q en %s", forbidden, string(encoded))
		}
	}
}

func TestBuildExternalWorkDryRunV0RechazaEntradaAmbiguaV0(t *testing.T) {
	input := validExternalWorkDryRunInputMCPTestV0()
	input.ExternalWorkRunRequest.AppChangeRequest = input.AppChangeRequest

	result := BuildExternalWorkDryRunV0(input, orquestaexternalworkrun.StartExternalWorkRunConfigV0{
		OccurredAt: "2026-06-27T10:00:00Z",
	})

	if result.Estado != MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPExternalWorkRunInputAmbiguousV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPExternalWorkDryRunHTTPV0DevuelveResumenSinExecutorExternoV0(t *testing.T) {
	input := validExternalWorkDryRunInputMCPTestV0()
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(input); err != nil {
		t.Fatalf("encode: %v", err)
	}
	handler := NewMCPExternalWorkDryRunHTTPHandlerWithConfigV0(
		orquestaexternalworkrun.StartExternalWorkRunConfigV0{OccurredAt: "2026-06-27T10:00:00Z"},
	)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, MCPExternalWorkDryRunHTTPPathV0, &body)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result MCPExternalWorkDryRunToolResultV0
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPExternalWorkRunEstadoOKV0 ||
		result.SpecSummary.GoalRef == "" ||
		result.GoalRef != result.SpecSummary.GoalRef ||
		result.SpecSummary.SpecHash == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPExternalWorkDryRunTransportV0RegistradoYCallableSinBindingV0(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	if _, ok := transport.tools[MCPExternalWorkDryRunToolNameV0]; !ok {
		t.Fatalf("tool no registrado: %s", MCPExternalWorkDryRunToolNameV0)
	}
	output, err := transport.CallToolV0(context.Background(), MCPExternalWorkDryRunToolNameV0, validExternalWorkDryRunInputMCPTestV0())
	if err != nil {
		t.Fatalf("call dry-run: %v", err)
	}
	var result MCPExternalWorkDryRunToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPExternalWorkRunEstadoOKV0 ||
		result.RoutePolicy != MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		result.SpecSummary.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 {
		t.Fatalf("result=%+v", result)
	}
}

func validExternalWorkDryRunInputMCPTestV0() MCPExternalWorkDryRunToolInputV0 {
	return MCPExternalWorkDryRunToolInputV0{
		MCPExternalWorkRunToolInputV0: MCPExternalWorkRunToolInputV0{
			RequestID:     "req-external-dry-run-001",
			CorrelationID: "corr-external-dry-run-001",
			ExternalWorkRunRequest: orquestaexternalworkrun.StartExternalWorkRunRequestV0{
				OccurredAt: "2026-06-27T10:00:00Z",
			},
			AppChangeRequest: orquestaappchange.AppChangeRequestV0{
				ChangeRef:       "change-ref-dry-run-001",
				AppRef:          "opes",
				UserIntent:      "Crear borrador OPES con evidencias.",
				AllowedWriteSet: []string{"deliveries/opes/job-ref-dry-run"},
				RequiredTests:   []string{"go test -count=1 ./modulos/orquesta-mcp"},
				AcceptanceCriteria: []string{
					"markdown valido",
				},
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-dry-run",
					WorkKind:   "draft_content_block",
					InputFields: []orquestadomainwork.DomainWorkFieldV0{{
						Name:  "topic_ref",
						Value: "valor-privado-no-copiar",
					}},
					RequiredTests: []orquestadomainwork.DomainWorkRequiredTestV0{{
						TestRef:                "domain-test-ref-qc",
						AcceptanceCriteriaRefs: []string{"criteria-ref-qc"},
					}},
				},
			},
		},
	}
}
