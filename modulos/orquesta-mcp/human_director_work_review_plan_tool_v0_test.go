package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestMCPHumanDirectorWorkReviewPlanDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPHumanDirectorWorkReviewPlanDescriptorV0()

	if descriptor.Name != MCPHumanDirectorWorkReviewPlanToolNameV0 ||
		descriptor.ResourceURI != MCPHumanDirectorWorkReviewPlanResourceURIV0 ||
		descriptor.InputSchema == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	for _, want := range []string{"acceptance_checks", "criterion_ref", "description?", "command"} {
		if !strings.Contains(descriptor.InputSchema, want) {
			t.Fatalf("input_schema no declara %q: %s", want, descriptor.InputSchema)
		}
	}
	assertTransportPayloadSaneadoMCPTestV0(t, descriptor, 1200)
}

func TestMCPHumanDirectorWorkReviewPlanExecutorV0ProyectaPrepareRunRevisable(t *testing.T) {
	result, err := MCPHumanDirectorWorkReviewPlanToolExecutorV0{}.Execute(
		context.Background(),
		MCPHumanDirectorWorkReviewPlanToolInputV0{
			RequestID:        "request-ref-human-director-mcp-001",
			CorrelationID:    "corr-human-director-mcp-001",
			WorktreeIsolated: true,
			WorkIntake:       validMCPHumanDirectorWorkIntakeV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 ||
		!result.Accepted ||
		result.Plan.Status != orquestaappdirectorintake.HumanDirectorPlanStatusReadyForReviewV0 ||
		result.AutoprogrammingRequest == nil {
		t.Fatalf("result=%+v", result)
	}
	request := result.AutoprogrammingRequest
	if request.RequestRef != "request-ref-human-director-mcp-001" ||
		!request.WorktreeIsolated ||
		request.WorktreeRef != "worktree-ref-human-director-mcp-001" ||
		len(request.Tasks) != 1 ||
		len(request.WriteSet) != 1 ||
		len(request.RequiredTests) != 1 {
		t.Fatalf("autoprogramming_request=%+v", request)
	}
	if !stringsSliceContainsMCPHumanWorkV0(result.NextActions, "post_/api/v0/autoprogramming/prepare-run_with_autoprogramming_request") {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
}

func TestMCPHumanDirectorWorkReviewPlanExecutorV0PidePuenteHumanoSinTirarTrabajo(t *testing.T) {
	intake := validMCPHumanDirectorWorkIntakeV0()
	intake.WorktreeRef = " "
	intake.BranchRef = " "

	result, err := MCPHumanDirectorWorkReviewPlanToolExecutorV0{}.Execute(
		context.Background(),
		MCPHumanDirectorWorkReviewPlanToolInputV0{
			RequestID:  "request-ref-human-director-review-001",
			WorkIntake: intake,
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 ||
		!result.Accepted ||
		result.AutoprogrammingRequest != nil {
		t.Fatalf("result=%+v", result)
	}
	if !stringsSliceContainsMCPHumanWorkV0(result.NextActions, "use_orquesta.operator.directed_query.v0_for_human_bridge_if_needed") {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
	if !stringsSliceContainsMCPHumanWorkV0(result.NextActions, "study_before_prepare_run") {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
}

func TestMCPHumanDirectorWorkReviewPlanExecutorV0ElevaPreguntaOperadorOptIn(t *testing.T) {
	executor := MCPHumanDirectorWorkReviewPlanToolExecutorV0{OperatorQuery: recordingHumanDirectorOperatorQueryV0{}}
	result, err := executor.Execute(
		context.Background(),
		MCPHumanDirectorWorkReviewPlanToolInputV0{
			RequestID:             "request-ref-human-director-query-001",
			WorktreeIsolated:      true,
			RaiseOperatorQuestion: true,
			OperatorQuery:         validHumanDirectorOperatorQueryV0(),
			WorkIntake:            validMCPHumanDirectorWorkIntakeV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 ||
		!result.Accepted ||
		result.Plan.RequestRef == "" ||
		result.OperatorQuestion == nil ||
		!result.OperatorQuestion.Accepted {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPHumanDirectorWorkReviewPlanExecutorV0PreguntaOperadorSinPuertoEsRepair(t *testing.T) {
	result, err := MCPHumanDirectorWorkReviewPlanToolExecutorV0{}.Execute(
		context.Background(),
		MCPHumanDirectorWorkReviewPlanToolInputV0{
			RequestID:             "request-ref-human-director-query-missing-001",
			RaiseOperatorQuestion: true,
			OperatorQuery:         validHumanDirectorOperatorQueryV0(),
			WorkIntake:            validMCPHumanDirectorWorkIntakeV0(),
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 ||
		!result.Accepted ||
		result.Plan.RequestRef == "" ||
		result.OperatorQuestion != nil ||
		len(result.Errores) == 0 ||
		!stringsSliceContainsMCPHumanWorkV0(result.NextActions, "repair_configure_operator_directed_query_port") {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPHumanDirectorWorkReviewPlanTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPHumanDirectorWorkReviewPlanToolNameV0,
		MCPHumanDirectorWorkReviewPlanToolInputV0{
			RequestID:        "request-ref-human-director-transport-001",
			WorktreeIsolated: true,
			WorkIntake:       validMCPHumanDirectorWorkIntakeV0(),
		},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPHumanDirectorWorkReviewPlanToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 ||
		!result.Accepted ||
		result.AutoprogrammingRequest == nil {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 3200)
}

func TestMCPHumanDirectorWorkReviewPlanTransportV0AceptaWorkIntakeTextoAmigable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPHumanDirectorWorkReviewPlanToolNameV0,
		map[string]any{
			"request_id":        "request-ref-human-work-text-001",
			"worktree_isolated": true,
			"work_intake":       "Revisar cola vacia y preparar automejora segura dentro del repo autorizado.",
		},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPHumanDirectorWorkReviewPlanToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 ||
		!result.Accepted ||
		result.Plan.Goal == "" ||
		len(result.Plan.Steps) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

type recordingHumanDirectorOperatorQueryV0 struct{}

func (recordingHumanDirectorOperatorQueryV0) RaiseOperatorDirectedQueryV0(
	input operator.OperatorDirectedQueryV0,
) (operator.OperatorMCPDirectedQueryResultV0, error) {
	return operator.OperatorMCPDirectedQueryResultV0{
		Accepted:   true,
		AnswerRef:  "answer-ref-" + input.QueryRef,
		NextAction: "await_external_operator_answer",
		TraceRefs:  []string{"trace-ref-" + input.TargetRef},
	}, nil
}

func validHumanDirectorOperatorQueryV0() operator.OperatorDirectedQueryV0 {
	return operator.OperatorDirectedQueryV0{
		QueryConnectorRef: "query-connector-ref-human-director",
		Question:          "Puede aprobar este plan o indicar ajuste acotado?",
		EvidenceRefs:      []string{"evidence-ref-human-director-query-001"},
	}
}

func TestMCPHumanDirectorWorkReviewPlanHTTPHandlerV0(t *testing.T) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPHumanDirectorWorkReviewPlanToolInputV0{
		RequestID:        "request-ref-human-director-http-001",
		WorktreeIsolated: true,
		WorkIntake:       validMCPHumanDirectorWorkIntakeV0(),
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPHumanDirectorWorkReviewPlanHTTPPathV0, body)

	NewMCPHumanDirectorWorkReviewPlanHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPHumanDirectorWorkReviewPlanToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPHumanDirectorWorkReviewPlanEstadoOKV0 || result.AutoprogrammingRequest == nil {
		t.Fatalf("result=%+v", result)
	}
}

func validMCPHumanDirectorWorkIntakeV0() orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0 {
	return orquestaappdirectorintake.HumanDirectorWorkIntakeRequestV0{
		SchemaVersion: "human_director_work_intake.v0",
		RequestRef:    "request-ref-human-director-mcp-001",
		ProjectRef:    "project-ref-orquesta",
		WorktreeRef:   "worktree-ref-human-director-mcp-001",
		BranchRef:     "branch-ref-human-director-mcp-001",
		Request: orquestaappdirectorintake.HumanDirectorWorkRequestV0{
			Title:              "Orden humana amplia para director",
			Objective:          "Planificar y preparar una tarea real reutilizando codigo existente.",
			AcceptanceCriteria: []string{"el plan conserva review humano y prepare-run posterior"},
			RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-mcp"},
			CompactRules:       []string{"no tirar todo el trabajo por dudas reparables"},
		},
		Rules: []string{"hexagonal puro", "usar puente humano si surge duda"},
		Limits: orquestaappdirectorintake.HumanDirectorWorkLimitsV0{
			MaxAgents:          6,
			MaxFanout:          6,
			MaxWriteSetEntries: 6,
		},
		Hints: orquestaappdirectorintake.HumanDirectorWorkHintsV0{
			Areas:           []string{"web_application"},
			WriteSet:        []string{"modulos/orquesta-mcp"},
			CanRepairSafely: []string{"normalizar alias razonables"},
		},
		ContextRefs: []string{"doc-ref:principio-orquesta-piensa-director"},
	}
}

func stringsSliceContainsMCPHumanWorkV0(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}
