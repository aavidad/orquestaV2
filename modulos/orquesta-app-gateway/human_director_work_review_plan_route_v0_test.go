package orquestaappgateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

func TestHumanDirectorWorkReviewPlanAPIRouteV0OperatorQuestionOptIn(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:       time.Second,
		OperatorQuery: gatewayHumanDirectorOperatorQueryV0{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/director/human-work/review-plan",
		strings.NewReader(`{
			"request_id":"request-ref-human-director-route-001",
			"worktree_isolated":true,
			"raise_operator_question":true,
			"operator_query":{
				"query_connector_ref":"query-connector-ref-human-director",
				"question":"Puede aprobar este plan o indicar ajuste acotado?",
				"evidence_refs":["evidence-ref-human-director-route-001"]
			},
			"work_intake":{
				"schema_version":"human_director_work_intake.v0",
				"request_ref":"request-ref-human-director-route-001",
				"project_ref":"project-ref-orquesta",
				"worktree_ref":"worktree-ref-human-director-route-001",
				"branch_ref":"branch-ref-human-director-route-001",
				"request":{
					"title":"Orden humana amplia para director",
					"objective":"Planificar tarea real reutilizando codigo existente.",
					"acceptance_criteria":["el plan conserva review humano"],
					"required_tests":["go test -count=1 ./modulos/orquesta-mcp"]
				},
				"hints":{"write_set":["modulos/orquesta-mcp"]},
				"context_refs":["evidence-ref-human-director-route-context"]
			}
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPHumanDirectorWorkReviewPlanToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Accepted || result.OperatorQuestion == nil || !result.OperatorQuestion.Accepted {
		t.Fatalf("result=%+v", result)
	}
}

type gatewayHumanDirectorOperatorQueryV0 struct{}

func (gatewayHumanDirectorOperatorQueryV0) RaiseOperatorDirectedQueryV0(
	input operator.OperatorDirectedQueryV0,
) (operator.OperatorMCPDirectedQueryResultV0, error) {
	return operator.OperatorMCPDirectedQueryResultV0{
		Accepted:   true,
		AnswerRef:  "answer-ref-" + input.QueryRef,
		NextAction: "await_external_operator_answer",
		TraceRefs:  []string{"trace-ref-" + input.TargetRef},
	}, nil
}
