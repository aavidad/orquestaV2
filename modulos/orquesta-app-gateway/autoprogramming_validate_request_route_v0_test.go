package orquestaappgateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingValidateRequestAPIRouteV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/validate-request",
		strings.NewReader(`{
			"request_id":"request-ref-app-gateway-autoprogramming-001",
			"autoprogramming_request":{
				"project_ref":"project-ref-orquesta",
				"worktree_ref":"worktree-ref-isolated-001",
				"worktree_isolated":true,
				"branch_ref":"branch-ref-autoprogramming-001",
				"tasks":[{"task_ref":"task-ref-api","area":"mcp"}],
				"write_set":["modulos/orquesta-mcp/autoprogramming_validate_request_http_v0.go"],
				"required_tests":["go test -count=1 ./modulos/orquesta-mcp"]
			}
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingValidateRequestToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Accepted || result.Estado != orquestamcp.MCPAutoprogrammingValidateRequestEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
}
