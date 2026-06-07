package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const MCPRuntimeModelsHTTPPathV0 = "/api/v0/runtime/models"

func NewMCPRuntimeModelsHTTPHandlerV0(
	port orquestaruntime.RuntimeModelManagerPortV0,
) http.Handler {
	return mcpRuntimeModelsHTTPHandlerV0{port: port}
}

type mcpRuntimeModelsHTTPHandlerV0 struct {
	port orquestaruntime.RuntimeModelManagerPortV0
}

func (handler mcpRuntimeModelsHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPRuntimeModelsHTTPPathV0 {
		writeMCPRuntimeModelsHTTPV0(w, http.StatusNotFound, newMCPRuntimeModelsHTTPErrorV0("path", MCPPublicErrPathUnsupportedV0))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPRuntimeModelsHTTPV0(w, http.StatusMethodNotAllowed, newMCPRuntimeModelsHTTPErrorV0("method", MCPPublicErrMethodNotAllowedV0))
		return
	}
	if handler.port == nil {
		writeMCPRuntimeModelsHTTPV0(w, http.StatusServiceUnavailable, newMCPRuntimeModelsHTTPErrorV0("port", "runtime_models_no_configurado"))
		return
	}
	var input MCPRuntimeModelsToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPRuntimeModelsHTTPV0(w, http.StatusBadRequest, newMCPRuntimeModelsHTTPErrorV0("body", code))
		return
	}
	result, err := (MCPRuntimeModelsToolExecutorV0{Port: handler.port}).Execute(r.Context(), input)
	if err != nil {
		writeMCPRuntimeModelsHTTPV0(w, http.StatusInternalServerError, newMCPRuntimeModelsHTTPErrorV0("port", "runtime_models_error"))
		return
	}
	status := http.StatusOK
	if result.Estado == MCPRuntimeModelsEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPRuntimeModelsHTTPV0(w, status, result)
}

func newMCPRuntimeModelsHTTPErrorV0(field string, message string) MCPRuntimeModelsToolResultV0 {
	result := newMCPRuntimeModelsErrorV0(MCPRuntimeModelsToolInputV0{}, "runtime_models_http_error", field)
	if len(result.Errores) > 0 {
		result.Errores[0].Message = strings.TrimSpace(message)
	}
	return result
}

func writeMCPRuntimeModelsHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPRuntimeModelsToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
