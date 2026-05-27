package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

const MCPHumanDirectorWorkReviewPlanHTTPPathV0 = "/api/v0/director/human-work/review-plan"

func NewMCPHumanDirectorWorkReviewPlanHTTPHandlerV0() http.Handler {
	return NewMCPHumanDirectorWorkReviewPlanHTTPHandlerWithOperatorQueryV0(nil)
}

func NewMCPHumanDirectorWorkReviewPlanHTTPHandlerWithOperatorQueryV0(
	operatorQuery operator.OperatorMCPDirectedQueryPortV0,
) http.Handler {
	return mcpHumanDirectorWorkReviewPlanHTTPHandlerV0{
		executor: MCPHumanDirectorWorkReviewPlanToolExecutorV0{OperatorQuery: operatorQuery},
	}
}

type mcpHumanDirectorWorkReviewPlanHTTPHandlerV0 struct {
	executor MCPHumanDirectorWorkReviewPlanToolExecutorV0
}

func (handler mcpHumanDirectorWorkReviewPlanHTTPHandlerV0) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != MCPHumanDirectorWorkReviewPlanHTTPPathV0 {
		writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, http.StatusNotFound, newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
			r,
			MCPHumanDirectorWorkReviewPlanToolInputV0{},
			"path",
			MCPPublicErrPathUnsupportedV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, http.StatusMethodNotAllowed, newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
			r,
			MCPHumanDirectorWorkReviewPlanToolInputV0{},
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	var input MCPHumanDirectorWorkReviewPlanToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, http.StatusBadRequest, newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
			r,
			input,
			"body",
			code,
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, http.StatusInternalServerError, newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
			r,
			input,
			"executor",
			"human_director_work_review_plan_error",
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPHumanDirectorWorkReviewPlanEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, status, result)
}

func newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
	r *http.Request,
	input MCPHumanDirectorWorkReviewPlanToolInputV0,
	field string,
	message string,
) MCPHumanDirectorWorkReviewPlanToolResultV0 {
	requestID := firstNonEmptyMCPV0(input.RequestID, input.WorkIntake.RequestRef)
	return MCPHumanDirectorWorkReviewPlanToolResultV0{
		Estado:        MCPHumanDirectorWorkReviewPlanEstadoErrorV0,
		RequestID:     strings.TrimSpace(requestID),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.WorkIntake.CorrelationID, requestID),
		Accepted:      false,
		Errores: []MCPValidationIssueV0{{
			Code:    "human_director_work_review_plan_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func writeMCPHumanDirectorWorkReviewPlanHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPHumanDirectorWorkReviewPlanToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
