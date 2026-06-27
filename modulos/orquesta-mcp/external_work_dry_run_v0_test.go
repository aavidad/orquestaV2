package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	if !reflect.DeepEqual(result.Spec, want) {
		t.Fatalf("spec dry-run no coincide\nwant=%+v\ngot=%+v", want, result.Spec)
	}
	if result.Spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 {
		t.Fatalf("director_kind=%q", result.Spec.DirectorKind)
	}
	if !mcpExternalWorkRunStringInSetTestV0(result.WriteSet, "deliveries/opes/job-ref-dry-run") ||
		!mcpExternalWorkRunStringInSetTestV0(result.RequiredTests, "go test -count=1 ./modulos/orquesta-mcp") ||
		!mcpExternalWorkRunStringInSetTestV0(result.RequiredTests, "domain-test-ref-qc") {
		t.Fatalf("write_set=%+v required_tests=%+v", result.WriteSet, result.RequiredTests)
	}
	if result.EstModel != "gpt-test" || result.EstTokens <= 0 || result.EstCostUSD <= 0 || result.EstWallClock == "" {
		t.Fatalf("estimate invalida: model=%q tokens=%d cost=%f wall=%q", result.EstModel, result.EstTokens, result.EstCostUSD, result.EstWallClock)
	}
	if !mcpExternalWorkRunStringInSetTestV0(result.EvidenceRefs, MCPExternalWorkDryRunEvidenceRefV0) {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
	for _, ctx := range result.Spec.ContextRefs {
		if strings.Contains(ctx.Ref, "valor-privado") || strings.Contains(ctx.Ref, "no-copiar") {
			t.Fatalf("context_refs contiene payload: %+v", result.Spec.ContextRefs)
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

func TestMCPExternalWorkDryRunHTTPV0DevuelveSpecSinExecutorExternoV0(t *testing.T) {
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
		result.Spec.GoalRef == "" ||
		result.GoalRef != result.Spec.GoalRef {
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
		result.Spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 {
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
