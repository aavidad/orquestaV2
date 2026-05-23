package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPAutoprogrammingSelfImprovementHTTPPathV0 = "/api/v0/autoprogramming/self-improvement"

func NewMCPAutoprogrammingSelfImprovementHTTPHandlerV0() http.Handler {
	return mcpAutoprogrammingSelfImprovementHTTPHandlerV0{
		executor: MCPAutoprogrammingSelfImprovementToolExecutorV0{},
	}
}

type mcpAutoprogrammingSelfImprovementHTTPHandlerV0 struct {
	executor MCPAutoprogrammingSelfImprovementToolExecutorV0
}

func (handler mcpAutoprogrammingSelfImprovementHTTPHandlerV0) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != MCPAutoprogrammingSelfImprovementHTTPPathV0 {
		writeMCPAutoprogrammingSelfImprovementHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingSelfImprovementHTTPErrorV0(
			r,
			MCPAutoprogrammingSelfImprovementToolInputV0{},
			"path",
			"ruta_no_soportada",
		))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPAutoprogrammingSelfImprovementHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingSelfImprovementHTTPErrorV0(
			r,
			MCPAutoprogrammingSelfImprovementToolInputV0{},
			"method",
			"metodo_no_permitido",
		))
		return
	}
	var input MCPAutoprogrammingSelfImprovementToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPAutoprogrammingSelfImprovementHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingSelfImprovementHTTPErrorV0(
			r,
			input,
			"body",
			"request_body_invalido",
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPAutoprogrammingSelfImprovementHTTPV0(w, http.StatusInternalServerError, newMCPAutoprogrammingSelfImprovementHTTPErrorV0(
			r,
			input,
			"executor",
			"autoprogramming_self_improvement_error",
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingSelfImprovementEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingSelfImprovementHTTPV0(w, status, result)
}

func newMCPAutoprogrammingSelfImprovementHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingSelfImprovementToolInputV0,
	field string,
	message string,
) MCPAutoprogrammingSelfImprovementToolResultV0 {
	requestID := firstNonEmptyMCPV0(input.RequestID, input.Proposal.RequestRef)
	return MCPAutoprogrammingSelfImprovementToolResultV0{
		Estado:        MCPAutoprogrammingSelfImprovementEstadoErrorV0,
		RequestID:     strings.TrimSpace(requestID),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, requestID),
		Accepted:      false,
		Background:    true,
		Errores: []MCPValidationIssueV0{{
			Code:    "autoprogramming_self_improvement_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func writeMCPAutoprogrammingSelfImprovementHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingSelfImprovementToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
