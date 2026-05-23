package orquestamcp

import (
	"encoding/json"
	"net/http"
)

func NewMCPAppVCSHTTPHandlerV0(executor MCPAppVCSExecutorPortV0) http.Handler {
	return mcpAppVCSHTTPHandlerV0{executor: NewMCPAppVCSToolExecutorV0(executor)}
}

type mcpAppVCSHTTPHandlerV0 struct {
	executor MCPAppVCSToolExecutorV0
}

func (handler mcpAppVCSHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPAppVCSHTTPPathV0 {
		writeMCPAppVCSHTTPV0(w, http.StatusNotFound, NewMCPAppVCSErrorResultV0(
			MCPAppVCSToolInputV0{CorrelationID: r.Header.Get("X-Correlation-ID")},
			"ruta_no_soportada",
			"path",
			"ruta_no_soportada",
		))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeMCPAppVCSHTTPV0(w, http.StatusMethodNotAllowed, NewMCPAppVCSErrorResultV0(
			MCPAppVCSToolInputV0{CorrelationID: r.Header.Get("X-Correlation-ID")},
			"metodo_no_permitido",
			"method",
			"metodo_no_permitido",
		))
		return
	}
	var input MCPAppVCSToolInputV0
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeMCPAppVCSHTTPV0(w, http.StatusBadRequest, NewMCPAppVCSErrorResultV0(
			input,
			"request_body_invalido",
			"body",
			"request_body_invalido",
		))
		return
	}
	result, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPAppVCSHTTPV0(w, http.StatusInternalServerError, NewMCPAppVCSErrorResultV0(
			input,
			"app_vcs_error",
			"executor",
			err.Error(),
		))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPAppVCSEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPAppVCSHTTPV0(w, status, result)
}

func writeMCPAppVCSHTTPV0(w http.ResponseWriter, status int, result MCPAppVCSToolResultV0) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
