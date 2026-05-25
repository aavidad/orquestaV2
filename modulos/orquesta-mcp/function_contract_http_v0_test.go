package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestacore "orquesta/modulos/orquesta-core"
)

func TestMCPFunctionContractListHTTPHandlerV0DelegatesReadOnlyPort(t *testing.T) {
	port := &fakeFunctionContractIndexPortV0{
		list: orquestacore.ListFunctionContractsResultV0{
			Items: []orquestacore.FunctionContractSummaryV0{{
				FunctionContractRef: "contract:function:mcp:v0",
				Estado:              orquestacore.FunctionContractEstadoEvidenciaInsuficienteV0,
			}},
			Warnings: []string{"function_contract_payload_no_materializado"},
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPFunctionContractListHTTPPathV0, strings.NewReader(`{"page":{"limit":1}}`))

	NewMCPFunctionContractListHTTPHandlerV0(port).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || port.listCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", rec.Code, port.listCalls, rec.Body.String())
	}
	var result orquestacore.ListFunctionContractsResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].FunctionContractRef != "contract:function:mcp:v0" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPFunctionContractViewHTTPHandlerV0DevuelveErrorPublico(t *testing.T) {
	port := &fakeFunctionContractIndexPortV0{
		viewErr: orquestacore.NewFunctionContractQueryErrorV0(
			orquestacore.FunctionContractQueryErrEvidenciaInsuficienteV0,
			"function_contract_ref",
			[]string{"event-ref-function-contract"},
		),
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		MCPFunctionContractViewHTTPPathV0,
		strings.NewReader(`{"function_contract_ref":"contract:function:mcp:v0"}`),
	)

	NewMCPFunctionContractViewHTTPHandlerV0(port).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result mcpFunctionContractHTTPErrorV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestacore.FunctionContractQueryErrEvidenciaInsuficienteV0 ||
		len(result.Errores[0].Evidence) != 1 {
		t.Fatalf("errores=%+v", result.Errores)
	}
}

func TestMCPFunctionContractResourceV0BloqueaRegistro(t *testing.T) {
	resource := NewMCPFunctionContractResourceV0()
	if resource.ListEndpoint != MCPFunctionContractListHTTPPathV0 ||
		resource.ViewEndpoint != MCPFunctionContractViewHTTPPathV0 {
		t.Fatalf("resource=%+v", resource)
	}
	if !containsStringMCPTestV0(resource.BlockedOperations, "registrar") {
		t.Fatalf("blocked=%+v", resource.BlockedOperations)
	}
}

type fakeFunctionContractIndexPortV0 struct {
	list      orquestacore.ListFunctionContractsResultV0
	listCalls int
	viewErr   error
}

func (port *fakeFunctionContractIndexPortV0) ListFunctionContractsV0(
	_ context.Context,
	_ orquestacore.ListFunctionContractsRequestV0,
) (orquestacore.ListFunctionContractsResultV0, error) {
	port.listCalls++
	return port.list, nil
}

func (port *fakeFunctionContractIndexPortV0) ViewFunctionContractV0(
	_ context.Context,
	_ orquestacore.ViewFunctionContractRequestV0,
) (orquestacore.ViewFunctionContractResultV0, error) {
	return orquestacore.ViewFunctionContractResultV0{}, port.viewErr
}
