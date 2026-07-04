package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0DelegaEnExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		result: MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:        MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted:      true,
			RunRef:        "run-autoprogramming-001",
			WaitAgentRefs: []string{"agent-request-001"},
			GoalSpecs: []orquestagoal.GoalWorkSpecV0{{
				SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
				GoalRef:       "goal-ref-autoprogramming-001",
				RequestRef:    "request-autoprogramming-001",
				RunRef:        "run-autoprogramming-001",
				WorkKind:      "autoprogramming",
				Objective:     "Preparar goal desde prepare-run.",
				DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-mcp"}},
			}},
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-autoprogramming-001",
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.input.RequestID != "request-autoprogramming-001" {
		t.Fatalf("input=%+v", executor.input)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, `"goal_specs"`) ||
		strings.Contains(raw, `"objective"`) ||
		strings.Contains(raw, `"write_set"`) ||
		strings.Contains(raw, "Preparar goal desde prepare-run.") ||
		strings.Contains(raw, "modulos/orquesta-mcp") {
		t.Fatalf("respuesta publica filtra GoalWorkSpec completo: %s", raw)
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.RunRef != "run-autoprogramming-001" ||
		len(result.WaitAgentRefs) != 1 ||
		len(result.GoalSpecSummaries) != 1 ||
		result.GoalSpecSummaries[0].RunRef != "run-autoprogramming-001" ||
		result.GoalSpecSummaries[0].SpecHash == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0SerializaGoalsBatch(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		result: MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:   MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted: true,
			RunRef:   "run-autoprogramming-batch-001-goal-01",
			Goals: []MCPAutoprogrammingGoalRunV0{
				{RunRef: "run-autoprogramming-batch-001-goal-01", GoalRef: "goal-ref-batch-001", GoalStatus: orquestagoal.GoalStatusRunningV0},
				{RunRef: "run-autoprogramming-batch-001-goal-02", GoalRef: "goal-ref-batch-002", GoalStatus: orquestagoal.GoalStatusRunningV0},
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewBufferString(`{"request_id":"request-autoprogramming-batch-001"}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Goal != nil ||
		len(result.Goals) != 2 ||
		result.Goals[0].RunRef != "run-autoprogramming-batch-001-goal-01" ||
		result.Goals[1].RunRef != "run-autoprogramming-batch-001-goal-02" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0DeclaraEvidenciaConfigProjectionVerificada(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		result: MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:       MCPAutoprogrammingPrepareRunEstadoOKV0,
			Accepted:     true,
			RunRef:       "run-autoprogramming-config-projection-ok-001",
			EvidenceRefs: []string{"evidence-ref-pre-existing"},
		},
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewBufferString(`{
		"request_id":"request-autoprogramming-config-projection-ok-001",
		"required_settings":[{"key":"ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS","value":"450000"}]
	}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !containsMCPStringPartForTestV0(result.EvidenceRefs, MCPConfigProjectionVerifiedEvidenceRefV0) ||
		!containsMCPStringPartForTestV0(result.EvidenceRefs, "evidence-ref-pre-existing") {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0ExecutorNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0SoloPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPAutoprogrammingPrepareRunHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(&fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Allow"); got != mcpPublicHTTPAllowHeaderV0(http.MethodPost) {
		t.Fatalf("allow=%q", got)
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0OptionsSinEfectos(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{}
	req := httptest.NewRequest(http.MethodOptions, MCPAutoprogrammingPrepareRunHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Allow"); got != mcpPublicHTTPAllowHeaderV0(http.MethodPost) {
		t.Fatalf("allow=%q", got)
	}
	if rec.Body.Len() != 0 || executor.input.RequestID != "" {
		t.Fatalf("options con efectos body=%q input=%+v", rec.Body.String(), executor.input)
	}
}

func TestMCPAutoprogrammingPrepareRunHTTPHandlerV0NoPropagaErrorNoCatalogado(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		err: errors.New("prepare failed at /root/Trabajo/orquesta token=secret123456 request-ref-prepare-error-001"),
	}
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewBufferString(`{"request_id":"request-ref-prepare-error-001"}`))
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPAutoprogrammingPrepareRunHTTPExecutorErrorCodeV0 ||
		result.Errores[0].Message != MCPAutoprogrammingPrepareRunHTTPExecutorErrorCodeV0 {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(rec.Body.String(), "/root/Trabajo") ||
		strings.Contains(rec.Body.String(), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", rec.Body.String())
	}
}

func TestMCPAutoprogrammingPrepareRunTransportV0DevuelvePayloadPublicoSiExecutorFalla(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0{
		err: errors.New("prepare failed at /root/Trabajo/orquesta token=secret123456 request-ref-prepare-transport-error-001"),
	}
	raw, err := json.Marshal(MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-ref-prepare-transport-error-001",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	output, err := mcpAutoprogrammingPrepareRunTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("transport no debe romper JSON-RPC: %v", err)
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, string(output))
	}
	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_prepare_run_executor_error" ||
		result.Errores[0].Message != "autoprogramming_prepare_run_executor_error" {
		t.Fatalf("payload publico incompleto: %+v", result)
	}
	if strings.Contains(string(output), "/root/Trabajo") ||
		strings.Contains(string(output), "secret123456") {
		t.Fatalf("payload filtra datos operativos: %s", string(output))
	}
}

type fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0 struct {
	input  MCPAutoprogrammingPrepareRunToolInputV0
	result MCPAutoprogrammingPrepareRunToolResultV0
	err    error
}

func (executor *fakeMCPAutoprogrammingPrepareRunHTTPExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingPrepareRunToolInputV0,
) (MCPAutoprogrammingPrepareRunToolResultV0, error) {
	_ = ctx
	executor.input = input
	if executor.result.Estado == "" {
		executor.result = MCPAutoprogrammingPrepareRunToolResultV0{Estado: MCPAutoprogrammingPrepareRunEstadoOKV0}
	}
	return executor.result, executor.err
}
