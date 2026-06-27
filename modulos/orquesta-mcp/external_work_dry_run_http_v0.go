package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strings"

	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

const (
	MCPExternalWorkDryRunHTTPPathV0                  = "/api/v0/external-work/dry-run"
	MCPExternalWorkDryRunHTTPInvalidBodyCodeV0       = MCPPublicErrBodyInvalidV0
	MCPExternalWorkDryRunHTTPUnsupportedPathCodeV0   = MCPPublicErrPathUnsupportedV0
	MCPExternalWorkDryRunHTTPUnsupportedMethodCodeV0 = MCPPublicErrMethodNotAllowedV0
)

func NewMCPExternalWorkDryRunHTTPHandlerV0(
	executor MCPTransportExternalWorkDryRunExecutorV0,
) http.Handler {
	return mcpExternalWorkDryRunHTTPHandlerV0{executor: executor}
}

func NewMCPExternalWorkDryRunHTTPHandlerWithConfigV0(
	config orquestaexternalworkrun.StartExternalWorkRunConfigV0,
) http.Handler {
	return NewMCPExternalWorkDryRunHTTPHandlerV0(
		MCPExternalWorkDryRunToolExecutorV0{Config: config},
	)
}

type mcpExternalWorkDryRunHTTPHandlerV0 struct {
	executor MCPTransportExternalWorkDryRunExecutorV0
}

func (handler mcpExternalWorkDryRunHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPExternalWorkDryRunHTTPPathV0 {
		writeMCPExternalWorkDryRunHTTPV0(w, http.StatusNotFound, newMCPExternalWorkDryRunHTTPErrorV0(
			r,
			MCPExternalWorkDryRunToolInputV0{},
			"path",
			MCPExternalWorkDryRunHTTPUnsupportedPathCodeV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodPost) {
		return
	}
	if r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodPost)
		writeMCPExternalWorkDryRunHTTPV0(w, http.StatusMethodNotAllowed, newMCPExternalWorkDryRunHTTPErrorV0(
			r,
			MCPExternalWorkDryRunToolInputV0{},
			"method",
			MCPExternalWorkDryRunHTTPUnsupportedMethodCodeV0,
		))
		return
	}
	var input MCPExternalWorkDryRunToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileDomainWorkV0); code != "" {
		writeMCPExternalWorkDryRunHTTPV0(w, http.StatusBadRequest, newMCPExternalWorkDryRunHTTPErrorV0(
			r,
			input,
			"body",
			code,
		))
		return
	}
	executor := handler.executor
	if executor == nil {
		executor = MCPExternalWorkDryRunToolExecutorV0{}
	}
	result, err := executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPExternalWorkDryRunHTTPV0(w, http.StatusInternalServerError, MCPExternalWorkDryRunToolResultV0{
			Estado:                MCPExternalWorkRunEstadoErrorV0,
			RoutePolicy:           MCPExternalWorkRunRoutePolicyGoalFirstV0,
			DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
			CorrelationID:         firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			Errores: []MCPExternalWorkRunIssueV0{{
				Code:  MCPExternalWorkRunHTTPExecutorErrorCodeV0,
				Field: "executor",
			}},
		})
		return
	}
	status := http.StatusOK
	if result.Estado == MCPExternalWorkRunEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPExternalWorkDryRunHTTPV0(w, status, result)
}

func newMCPExternalWorkDryRunHTTPErrorV0(
	r *http.Request,
	input MCPExternalWorkDryRunToolInputV0,
	field string,
	code string,
) MCPExternalWorkDryRunToolResultV0 {
	return MCPExternalWorkDryRunToolResultV0{
		Estado:                MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		CorrelationID:         firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
		Errores: []MCPExternalWorkRunIssueV0{{
			Code:  strings.TrimSpace(code),
			Field: strings.TrimSpace(field),
		}},
	}
}

func writeMCPExternalWorkDryRunHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPExternalWorkDryRunToolResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
