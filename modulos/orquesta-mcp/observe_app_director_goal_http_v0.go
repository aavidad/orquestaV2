package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPObserveAppDirectorGoalHTTPPathV0 = "/api/v0/apps/director/goal/observe"

func NewMCPObserveAppDirectorGoalHTTPHandlerV0(
	executor MCPTransportObserveAppDirectorGoalExecutorV0,
) http.Handler {
	return mcpObserveAppDirectorGoalHTTPHandlerV0{executor: executor}
}

type mcpObserveAppDirectorGoalHTTPHandlerV0 struct {
	executor MCPTransportObserveAppDirectorGoalExecutorV0
}

func (handler mcpObserveAppDirectorGoalHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPObserveAppDirectorGoalHTTPPathV0 {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusNotFound, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			MCPObserveAppDirectorGoalToolInputV0{},
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
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusMethodNotAllowed, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			MCPObserveAppDirectorGoalToolInputV0{},
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusServiceUnavailable, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			MCPObserveAppDirectorGoalToolInputV0{},
			"executor",
			"observe_app_director_goal_no_configurado",
		))
		return
	}
	var input MCPObserveAppDirectorGoalToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, newMCPObserveAppDirectorGoalHTTPErrorV0(r, input, "body", code))
		return
	}
	if strings.TrimSpace(input.RunRef) == "" {
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, newMCPObserveAppDirectorGoalHTTPErrorV0(r, input, "run_ref", "run_ref_requerido"))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		if result.Estado == MCPObserveAppDirectorGoalEstadoErrorV0 && len(result.Errores) > 0 {
			result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
			writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, result)
			return
		}
		if publicResult, ok := NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(input, err); ok {
			publicResult.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), publicResult.CorrelationID, input.CorrelationID, input.RequestID)
			writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusBadRequest, publicResult)
			return
		}
		writeMCPObserveAppDirectorGoalHTTPV0(w, http.StatusInternalServerError, newMCPObserveAppDirectorGoalHTTPErrorV0(
			r,
			input,
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("observe_app_director_goal_error", err),
		))
		return
	}
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID, input.CorrelationID, input.RequestID)
	status := http.StatusOK
	if result.Estado == MCPObserveAppDirectorGoalEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPObserveAppDirectorGoalHTTPV0(w, status, result)
}

func newMCPObserveAppDirectorGoalHTTPErrorV0(
	r *http.Request,
	input MCPObserveAppDirectorGoalToolInputV0,
	field string,
	message string,
) MCPObserveAppDirectorGoalToolResultV0 {
	result := NewMCPObserveAppDirectorGoalErrorResultV0(input, "observe_app_director_goal_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPObserveAppDirectorGoalHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPObserveAppDirectorGoalToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
