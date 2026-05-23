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

func TestAutoprogrammingSelfImprovementAPIRouteV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/autoprogramming/self-improvement",
		strings.NewReader(`{
			"request_id":"request-ref-self-improvement-route-001",
			"proposal":{
				"project_ref":"project-ref-orquesta",
				"worktree_ref":"worktree-ref-self-improvement-route-001",
				"worktree_isolated":true,
				"branch_ref":"branch-ref-self-improvement-route-001",
				"observed_by":"director",
				"failure_summary":"El director detecto una mejora general reutilizable.",
				"suggested_area":"mcp",
				"suggested_write_set":["modulos/orquesta-mcp/autoprogramming_self_improvement_tool_v0.go"],
				"required_tests":["go test -count=1 ./modulos/orquesta-mcp"]
			}
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingSelfImprovementToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Accepted ||
		result.Estado != orquestamcp.MCPAutoprogrammingSelfImprovementEstadoOKV0 ||
		result.PrepareRun == nil ||
		result.PriorityScore <= 0 {
		t.Fatalf("result=%+v", result)
	}
}
