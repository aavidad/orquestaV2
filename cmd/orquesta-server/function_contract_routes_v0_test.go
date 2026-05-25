package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestacore "orquesta/modulos/orquesta-core"
)

func TestWithFunctionContractRoutesV0ExponeIndiceReadOnly(t *testing.T) {
	index := &serverFunctionContractIndexFakeV0{
		ListResult: orquestacore.ListFunctionContractsResultV0{
			Items: []orquestacore.FunctionContractSummaryV0{{
				FunctionContractRef: "contract:function:server:v0",
				Estado:              orquestacore.FunctionContractEstadoEvidenciaInsuficienteV0,
			}},
			Warnings: []string{"function_contract_payload_no_materializado"},
		},
	}
	handler := withFunctionContractRoutesV0(http.NotFoundHandler(), index)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/core/function-contracts/list", strings.NewReader(`{"page":{"limit":1}}`))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || index.ListCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", rec.Code, index.ListCalls, rec.Body.String())
	}
	var result orquestacore.ListFunctionContractsResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].FunctionContractRef != "contract:function:server:v0" {
		t.Fatalf("result=%+v", result)
	}
}

type serverFunctionContractIndexFakeV0 struct {
	ListResult orquestacore.ListFunctionContractsResultV0
	ListCalls  int
}

func (index *serverFunctionContractIndexFakeV0) ListFunctionContractsV0(
	_ context.Context,
	_ orquestacore.ListFunctionContractsRequestV0,
) (orquestacore.ListFunctionContractsResultV0, error) {
	index.ListCalls++
	return index.ListResult, nil
}

func (index *serverFunctionContractIndexFakeV0) ViewFunctionContractV0(
	_ context.Context,
	_ orquestacore.ViewFunctionContractRequestV0,
) (orquestacore.ViewFunctionContractResultV0, error) {
	return orquestacore.ViewFunctionContractResultV0{}, nil
}
