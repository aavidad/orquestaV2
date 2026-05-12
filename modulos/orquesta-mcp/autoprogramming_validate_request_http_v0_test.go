package orquestamcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPAutoprogrammingValidateRequestHTTPV0AcceptsSnakeCaseJSON(t *testing.T) {
	body := strings.NewReader(`{
		"request_id":"request-ref-autoprogramming-http-001",
		"correlation_id":"corr-autoprogramming-http-001",
		"autoprogramming_request":{
			"project_ref":"project-ref-orquesta",
			"worktree_ref":"worktree-ref-isolated-001",
			"worktree_isolated":true,
			"branch_ref":"branch-ref-autoprogramming-001",
			"tasks":[{"task_ref":"task-ref-rest-validate","area":"mcp"}],
			"write_set":["modulos/orquesta-mcp/autoprogramming_validate_request_http_v0.go"],
			"required_tests":["go test -count=1 ./modulos/orquesta-mcp"]
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPAutoprogrammingValidateRequestHTTPPathV0, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingValidateRequestHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingValidateRequestToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingValidateRequestEstadoOKV0 ||
		!result.Accepted ||
		result.RequestID != "request-ref-autoprogramming-http-001" ||
		len(result.Groups) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingValidateRequestHTTPV0ReturnsPublicErrors(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		MCPAutoprogrammingValidateRequestHTTPPathV0,
		strings.NewReader(`{"request_id":"request-ref-invalid"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	NewMCPAutoprogrammingValidateRequestHTTPHandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPAutoprogrammingValidateRequestToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingValidateRequestEstadoErrorV0 ||
		result.Accepted ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}
