package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPHumanDirectorWorkReviewPlanHTTPPathV0 = "/api/v0/director/human-work/review-plan"

func NewMCPHumanDirectorWorkReviewPlanHTTPHandlerV0() http.Handler {
	return mcpHumanDirectorWorkReviewPlanHTTPHandlerV0{
		executor: MCPHumanDirectorWorkReviewPlanToolExecutorV0{},
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
			"ruta_no_soportada",
		))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, http.StatusMethodNotAllowed, newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
			r,
			MCPHumanDirectorWorkReviewPlanToolInputV0{},
			"method",
			"metodo_no_permitido",
		))
		return
	}
	var input MCPHumanDirectorWorkReviewPlanToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPHumanDirectorWorkReviewPlanHTTPV0(w, http.StatusBadRequest, newMCPHumanDirectorWorkReviewPlanHTTPErrorV0(
			r,
			input,
			"body",
			"request_body_invalido",
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
