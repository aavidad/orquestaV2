package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

const MCPServerShutdownHTTPPathV0 = "/api/v0/server/shutdown"

func NewMCPServerShutdownHTTPHandlerV0(
	executor MCPTransportServerShutdownExecutorV0,
) http.Handler {
	return mcpServerShutdownHTTPHandlerV0{executor: executor}
}

type mcpServerShutdownHTTPHandlerV0 struct {
	executor MCPTransportServerShutdownExecutorV0
}

func (handler mcpServerShutdownHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPServerShutdownHTTPPathV0 {
		writeMCPServerShutdownHTTPV0(w, http.StatusNotFound, newMCPServerShutdownHTTPErrorV0(r, MCPServerShutdownToolInputV0{}, "path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPServerShutdownHTTPV0(w, http.StatusMethodNotAllowed, newMCPServerShutdownHTTPErrorV0(r, MCPServerShutdownToolInputV0{}, "method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	if handler.executor == nil {
		writeMCPServerShutdownHTTPV0(w, http.StatusServiceUnavailable, newMCPServerShutdownHTTPErrorV0(r, MCPServerShutdownToolInputV0{}, "executor", "server_shutdown_no_configurado"))
		return
	}
	var input MCPServerShutdownToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPServerShutdownHTTPV0(w, http.StatusBadRequest, newMCPServerShutdownHTTPErrorV0(r, input, "body", code))
		return
	}
	input, issues := normalizeMCPServerShutdownIdentityV0(
		input,
		r.Header.Get(MCPPublicCorrelationHeaderV0),
		MCPPublicMutationHeaderIdempotencyKeyV0(r.Header.Get),
	)
	if len(issues) > 0 {
		writeMCPServerShutdownHTTPV0(w, http.StatusBadRequest, newMCPServerShutdownErrorV0(input, issues[0].Code, issues[0].Field, issues[0].Code))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		if result.Estado == MCPServerShutdownEstadoErrorV0 && len(result.Errores) > 0 {
			writeMCPServerShutdownHTTPV0(w, http.StatusInternalServerError, result)
			return
		}
		writeMCPServerShutdownHTTPV0(
			w,
			http.StatusInternalServerError,
			newMCPServerShutdownHTTPErrorV0(
				r,
				input,
				"executor",
				publicMCPExecutorErrorMessageFromErrorV0("server_shutdown_executor_error", err),
			),
		)
		return
	}
	status := http.StatusOK
	if result.Estado == MCPServerShutdownEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPServerShutdownHTTPV0(w, status, result)
}

func newMCPServerShutdownHTTPErrorV0(
	r *http.Request,
	input MCPServerShutdownToolInputV0,
	field string,
	message string,
) MCPServerShutdownToolResultV0 {
	result := newMCPServerShutdownErrorV0(input, "server_shutdown_http_error", field, message)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get(MCPPublicCorrelationHeaderV0), input.CorrelationID, input.RequestID)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPServerShutdownHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPServerShutdownToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set(MCPPublicCorrelationHeaderV0, result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
