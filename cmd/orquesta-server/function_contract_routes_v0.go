package main

import (
	"net/http"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func withFunctionContractRoutesV0(
	next http.Handler,
	index orquestacore.FunctionContractReadIndexPortV0,
) http.Handler {
	if index == nil {
		return next
	}
	mux := http.NewServeMux()
	mux.Handle(orquestamcp.MCPFunctionContractListHTTPPathV0, orquestamcp.NewMCPFunctionContractListHTTPHandlerV0(index))
	mux.Handle(orquestamcp.MCPFunctionContractViewHTTPPathV0, orquestamcp.NewMCPFunctionContractViewHTTPHandlerV0(index))
	mux.Handle("/", next)
	return mux
}
