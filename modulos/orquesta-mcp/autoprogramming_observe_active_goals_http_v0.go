package orquestamcp

import (
	"encoding/json"
	"net/http"
)

func NewMCPAutoprogrammingObserveActiveGoalsHTTPHandlerV0(
	executor MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0,
) http.Handler {
	return mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0{executor: executor}
}

type mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0 struct {
	executor MCPTransportAutoprogrammingObserveActiveGoalsExecutorV0
}

func (handler mcpAutoprogrammingObserveActiveGoalsHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingObserveActiveGoalsHTTPPathV0 {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
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
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
			r,
			MCPAutoprogrammingObserveActiveGoalsToolInputV0{},
			"executor",
			"autoprogramming_observe_active_goals_no_configurado",
		))
		return
	}
	var input MCPAutoprogrammingObserveActiveGoalsToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(r, input, "body", code))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, http.StatusInternalServerError, mcpAutoprogrammingObserveActiveGoalsErrorV0(
			input,
			"autoprogramming_observe_active_goals_execute_error",
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_observe_active_goals_execute_error", err),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingObserveActiveGoalsEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(w, status, result)
}

func newMCPAutoprogrammingObserveActiveGoalsHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
	field string,
	message string,
) MCPAutoprogrammingObserveActiveGoalsToolResultV0 {
	result := mcpAutoprogrammingObserveActiveGoalsErrorV0(
		input,
		"autoprogramming_observe_active_goals_http_error",
		field,
		message,
	)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	return result
}

func writeMCPAutoprogrammingObserveActiveGoalsHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingObserveActiveGoalsToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
