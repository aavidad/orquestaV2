package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPAutoprogrammingValidateRequestHTTPPathV0 = "/api/v0/autoprogramming/validate-request"

func NewMCPAutoprogrammingValidateRequestHTTPHandlerV0() http.Handler {
	return mcpAutoprogrammingValidateRequestHTTPHandlerV0{
		executor: MCPAutoprogrammingValidateRequestToolExecutorV0{},
	}
}

type mcpAutoprogrammingValidateRequestHTTPHandlerV0 struct {
	executor MCPAutoprogrammingValidateRequestToolExecutorV0
}

func (handler mcpAutoprogrammingValidateRequestHTTPHandlerV0) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != MCPAutoprogrammingValidateRequestHTTPPathV0 {
		writeMCPAutoprogrammingValidateRequestHTTPV0(
			w,
			http.StatusNotFound,
			newMCPAutoprogrammingValidateRequestHTTPErrorV0(r, MCPAutoprogrammingValidateRequestToolInputV0{}, "path", "ruta_no_soportada"),
		)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPAutoprogrammingValidateRequestHTTPV0(
			w,
			http.StatusMethodNotAllowed,
			newMCPAutoprogrammingValidateRequestHTTPErrorV0(r, MCPAutoprogrammingValidateRequestToolInputV0{}, "method", "metodo_no_permitido"),
		)
		return
	}
	var input MCPAutoprogrammingValidateRequestToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPAutoprogrammingValidateRequestHTTPV0(
			w,
			http.StatusBadRequest,
			newMCPAutoprogrammingValidateRequestHTTPErrorV0(r, input, "body", "request_body_invalido"),
		)
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPAutoprogrammingValidateRequestHTTPV0(
			w,
			http.StatusInternalServerError,
			newMCPAutoprogrammingValidateRequestHTTPErrorV0(r, input, "executor", "autoprogramming_validate_error"),
		)
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingValidateRequestEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingValidateRequestHTTPV0(w, status, result)
}

func newMCPAutoprogrammingValidateRequestHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingValidateRequestToolInputV0,
	field string,
	message string,
) MCPAutoprogrammingValidateRequestToolResultV0 {
	return MCPAutoprogrammingValidateRequestToolResultV0{
		Estado:        MCPAutoprogrammingValidateRequestEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
		Accepted:      false,
		Errores: []MCPValidationIssueV0{{
			Code:    "autoprogramming_validate_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func writeMCPAutoprogrammingValidateRequestHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingValidateRequestToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
