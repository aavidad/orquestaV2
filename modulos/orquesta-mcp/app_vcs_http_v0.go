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
			MCPAppVCSToolInputV0{CorrelationID: r.Header.Get(MCPPublicCorrelationHeaderV0)},
			MCPPublicErrPathUnsupportedV0,
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
		writeMCPAppVCSHTTPV0(w, http.StatusMethodNotAllowed, NewMCPAppVCSErrorResultV0(
			MCPAppVCSToolInputV0{CorrelationID: r.Header.Get(MCPPublicCorrelationHeaderV0)},
			MCPPublicErrMethodNotAllowedV0,
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	var input MCPAppVCSToolInputV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeMCPAppVCSHTTPV0(w, http.StatusBadRequest, NewMCPAppVCSErrorResultV0(
			input,
			code,
			"body",
			code,
		))
		return
	}
	input, issues := normalizeMCPAppVCSIdentityV0(
		input,
		r.Header.Get(MCPPublicCorrelationHeaderV0),
		MCPPublicMutationHeaderIdempotencyKeyV0(r.Header.Get),
	)
	if len(issues) > 0 {
		writeMCPAppVCSHTTPV0(w, http.StatusBadRequest, NewMCPAppVCSErrorResultV0(
			input,
			issues[0].Code,
			issues[0].Field,
			issues[0].Message,
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
		w.Header().Set(MCPPublicCorrelationHeaderV0, result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
