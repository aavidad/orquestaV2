package orquestamcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	orquestacore "orquesta/modulos/orquesta-core"
)

const (
	MCPFunctionContractListHTTPPathV0 = "/api/v0/core/function-contracts/list"
	MCPFunctionContractViewHTTPPathV0 = "/api/v0/core/function-contracts/view"
)

func NewMCPFunctionContractListHTTPHandlerV0(port orquestacore.FunctionContractReadIndexPortV0) http.Handler {
	return mcpFunctionContractListHTTPHandlerV0{port: port}
}

func NewMCPFunctionContractViewHTTPHandlerV0(port orquestacore.FunctionContractReadIndexPortV0) http.Handler {
	return mcpFunctionContractViewHTTPHandlerV0{port: port}
}

type mcpFunctionContractListHTTPHandlerV0 struct {
	port orquestacore.FunctionContractReadIndexPortV0
}

type mcpFunctionContractViewHTTPHandlerV0 struct {
	port orquestacore.FunctionContractReadIndexPortV0
}

type mcpFunctionContractHTTPErrorV0 struct {
	Errores []orquestacore.FunctionContractErrorV0 `json:"errores"`
}

func (handler mcpFunctionContractListHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !validateFunctionContractHTTPRouteV0(w, r, MCPFunctionContractListHTTPPathV0) {
		return
	}
	if handler.port == nil {
		writeFunctionContractHTTPErrorV0(w, http.StatusServiceUnavailable, "function_contract_index_no_configurado", "executor", nil)
		return
	}
	var input orquestacore.ListFunctionContractsRequestV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeFunctionContractHTTPErrorV0(w, http.StatusBadRequest, code, "body", nil)
		return
	}
	result, err := handler.port.ListFunctionContractsV0(r.Context(), input)
	if err != nil {
		writeFunctionContractQueryErrorV0(w, err)
		return
	}
	writeFunctionContractHTTPOKV0(w, result)
}

func (handler mcpFunctionContractViewHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !validateFunctionContractHTTPRouteV0(w, r, MCPFunctionContractViewHTTPPathV0) {
		return
	}
	if handler.port == nil {
		writeFunctionContractHTTPErrorV0(w, http.StatusServiceUnavailable, "function_contract_index_no_configurado", "executor", nil)
		return
	}
	var input orquestacore.ViewFunctionContractRequestV0
	if code := decodeMCPPublicHTTPJSONV0(w, r, &input); code != "" {
		writeFunctionContractHTTPErrorV0(w, http.StatusBadRequest, code, "body", nil)
		return
	}
	result, err := handler.port.ViewFunctionContractV0(r.Context(), input)
	if err != nil {
		writeFunctionContractQueryErrorV0(w, err)
		return
	}
	writeFunctionContractHTTPOKV0(w, result)
}

func validateFunctionContractHTTPRouteV0(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.URL.Path != path {
		writeFunctionContractHTTPErrorV0(w, http.StatusNotFound, "ruta_no_soportada", "path", nil)
		return false
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeFunctionContractHTTPErrorV0(w, http.StatusMethodNotAllowed, "metodo_no_permitido", "method", nil)
		return false
	}
	return true
}

func writeFunctionContractQueryErrorV0(w http.ResponseWriter, err error) {
	var queryErr interface {
		FunctionContractErrorsV0() []orquestacore.FunctionContractErrorV0
	}
	if errors.As(err, &queryErr) {
		writeFunctionContractHTTPErrorsV0(w, http.StatusBadRequest, queryErr.FunctionContractErrorsV0())
		return
	}
	writeFunctionContractHTTPErrorV0(
		w,
		http.StatusInternalServerError,
		orquestacore.FunctionContractQueryErrConsultaNoDisponibleV0,
		"index",
		nil,
	)
}

func writeFunctionContractHTTPErrorV0(
	w http.ResponseWriter,
	status int,
	code string,
	field string,
	evidence []string,
) {
	writeFunctionContractHTTPErrorsV0(w, status, []orquestacore.FunctionContractErrorV0{{
		Code:     strings.TrimSpace(code),
		Message:  strings.TrimSpace(code),
		Field:    strings.TrimSpace(field),
		Evidence: append([]string(nil), evidence...),
	}})
}

func writeFunctionContractHTTPErrorsV0(
	w http.ResponseWriter,
	status int,
	errs []orquestacore.FunctionContractErrorV0,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mcpFunctionContractHTTPErrorV0{Errores: errs})
}

func writeFunctionContractHTTPOKV0(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
