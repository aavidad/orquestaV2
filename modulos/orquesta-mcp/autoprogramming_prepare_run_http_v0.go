package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	MCPAutoprogrammingPrepareRunHTTPPathV0                  = "/api/v0/autoprogramming/prepare-run"
	MCPAutoprogrammingPrepareRunHTTPErrorCodeV0             = "autoprogramming_prepare_run_http_error"
	MCPAutoprogrammingPrepareRunHTTPNotConfiguredCodeV0     = "autoprogramming_prepare_run_no_configurado"
	MCPAutoprogrammingPrepareRunHTTPExecutorErrorCodeV0     = "autoprogramming_prepare_run_error"
	MCPAutoprogrammingPrepareRunHTTPInvalidBodyCodeV0       = "request_body_invalido"
	MCPAutoprogrammingPrepareRunHTTPUnsupportedPathCodeV0   = "ruta_no_soportada"
	MCPAutoprogrammingPrepareRunHTTPUnsupportedMethodCodeV0 = "metodo_no_permitido"
)

func NewMCPAutoprogrammingPrepareRunHTTPHandlerV0(
	executor MCPTransportAutoprogrammingPrepareRunExecutorV0,
) http.Handler {
	return mcpAutoprogrammingPrepareRunHTTPHandlerV0{executor: executor}
}

type mcpAutoprogrammingPrepareRunHTTPHandlerV0 struct {
	executor MCPTransportAutoprogrammingPrepareRunExecutorV0
}

func (handler mcpAutoprogrammingPrepareRunHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAutoprogrammingPrepareRunHTTPPathV0 {
		writeMCPAutoprogrammingPrepareRunHTTPV0(w, http.StatusNotFound, newMCPAutoprogrammingPrepareRunHTTPErrorV0(
			r,
			MCPAutoprogrammingPrepareRunToolInputV0{},
			"path",
			MCPAutoprogrammingPrepareRunHTTPUnsupportedPathCodeV0,
		))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPAutoprogrammingPrepareRunHTTPV0(w, http.StatusMethodNotAllowed, newMCPAutoprogrammingPrepareRunHTTPErrorV0(
			r,
			MCPAutoprogrammingPrepareRunToolInputV0{},
			"method",
			MCPAutoprogrammingPrepareRunHTTPUnsupportedMethodCodeV0,
		))
		return
	}
	if handler.executor == nil {
		writeMCPAutoprogrammingPrepareRunHTTPV0(w, http.StatusServiceUnavailable, newMCPAutoprogrammingPrepareRunHTTPErrorV0(
			r,
			MCPAutoprogrammingPrepareRunToolInputV0{},
			"executor",
			MCPAutoprogrammingPrepareRunHTTPNotConfiguredCodeV0,
		))
		return
	}
	var input MCPAutoprogrammingPrepareRunToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPAutoprogrammingPrepareRunHTTPV0(w, http.StatusBadRequest, newMCPAutoprogrammingPrepareRunHTTPErrorV0(
			r,
			input,
			"body",
			MCPAutoprogrammingPrepareRunHTTPInvalidBodyCodeV0,
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPAutoprogrammingPrepareRunHTTPV0(w, http.StatusInternalServerError, newMCPAutoprogrammingPrepareRunHTTPErrorV0(
			r,
			input,
			"executor",
			MCPAutoprogrammingPrepareRunHTTPExecutorErrorCodeV0,
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingPrepareRunEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAutoprogrammingPrepareRunHTTPV0(w, status, result)
}

func newMCPAutoprogrammingPrepareRunHTTPErrorV0(
	r *http.Request,
	input MCPAutoprogrammingPrepareRunToolInputV0,
	field string,
	code string,
) MCPAutoprogrammingPrepareRunToolResultV0 {
	result := NewMCPAutoprogrammingPrepareRunErrorResultV0(input, MCPAutoprogrammingPrepareRunHTTPErrorCodeV0, field, code)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Code = strings.TrimSpace(code)
		result.Errores[0].Message = strings.TrimSpace(code)
	}
	return result
}

func writeMCPAutoprogrammingPrepareRunHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPAutoprogrammingPrepareRunToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
