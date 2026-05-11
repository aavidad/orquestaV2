package orquestacli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
)

func TestFunctionContractCliClientV0ListarOK(t *testing.T) {
	var received ListarFunctionContractsRequestV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != FunctionContractCliListEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(FunctionContractCliCorrelationHeaderV0); got != "corr-fc-1" {
			t.Fatalf("correlation=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if received.RequestID != "req-fc-1" || received.CorrelationID != "corr-fc-1" {
			t.Fatalf("ids no propagados: %+v", received)
		}
		writeFunctionContractListSuccessV0(t, w)
	}))
	defer server.Close()

	client := mustNewFunctionContractCliClientV0(t, server.URL, time.Second)
	env := client.ListarFunctionContracts(context.Background(), functionContractInvocationV0(server.URL), ListarFunctionContractsRequestV0{
		Filtros: FunctionContractFiltrosV0{Modulo: "orquesta-cli", Estado: orquestacore.FunctionContractEstadoActivaV0},
		Page:    FunctionContractPageRequestV0{Limit: 10},
	})

	if !env.OK || env.Contract != FunctionContractCliContractV0 || env.Version != FunctionContractCliContractVersionV0 {
		t.Fatalf("envelope inesperado: %+v", env)
	}
	data, ok := env.Data.(ListarFunctionContractsResultV0)
	if !ok {
		t.Fatalf("data type=%T", env.Data)
	}
	if len(data.Items) != 1 || data.Items[0].FunctionContractRef != "function_contract_ref_001" {
		t.Fatalf("items inesperados: %+v", data.Items)
	}
}

func TestFunctionContractCliClientV0VerOK(t *testing.T) {
	var received VerFunctionContractRequestV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != FunctionContractCliViewEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if received.FunctionContractRef != "function_contract_ref_001" ||
			received.RequestID != "req-fc-1" ||
			received.CorrelationID != "corr-fc-1" {
			t.Fatalf("request inesperado: %+v", received)
		}
		writeFunctionContractViewSuccessV0(t, w)
	}))
	defer server.Close()

	client := mustNewFunctionContractCliClientV0(t, server.URL, time.Second)
	env := client.VerFunctionContract(context.Background(), functionContractInvocationV0(server.URL), VerFunctionContractRequestV0{
		FunctionContractRef: " function_contract_ref_001 ",
		Version:             1,
	})

	if !env.OK || env.Meta.StatusCode != http.StatusOK {
		t.Fatalf("envelope inesperado: %+v", env)
	}
	data, ok := env.Data.(VerFunctionContractResultV0)
	if !ok {
		t.Fatalf("data type=%T", env.Data)
	}
	if data.FunctionContract.Titulo != "Cerrar CLI-004" ||
		data.FunctionContract.Estado != orquestacore.FunctionContractEstadoActivaV0 {
		t.Fatalf("function_contract inesperado: %+v", data.FunctionContract)
	}
}

func TestFunctionContractCliClientV0Respuesta400DevuelveErroresPublicos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(functionContractHTTPErrorV0{
			Errores: []orquestacore.FunctionContractErrorV0{{
				Code:    "filtro_no_soportado",
				Field:   "filtros.estado",
				Message: "estado no soportado",
			}},
		})
	}))
	defer server.Close()

	client := mustNewFunctionContractCliClientV0(t, server.URL, time.Second)
	env := client.ListarFunctionContracts(context.Background(), functionContractInvocationV0(server.URL), ListarFunctionContractsRequestV0{})

	if env.OK || env.Data != nil || env.Meta.StatusCode != http.StatusBadRequest || env.Meta.Retryable {
		t.Fatalf("envelope 400 inesperado: %+v", env)
	}
	if len(env.Errores) != 1 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	if err := env.Errores[0]; err.Codigo != "filtro_no_soportado" ||
		err.Campo != "filtros.estado" ||
		err.MensajeI18N != "orquesta_core.function_contract.errores.filtro_no_soportado" ||
		err.Detalle != "" {
		t.Fatalf("error publico inesperado: %+v", err)
	}
}

func TestFunctionContractCliClientV0RespuestaInvalida(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"function_contract":{"titulo":"incompleto"}}`))
	}))
	defer server.Close()

	client := mustNewFunctionContractCliClientV0(t, server.URL, time.Second)
	env := client.VerFunctionContract(context.Background(), functionContractInvocationV0(server.URL), VerFunctionContractRequestV0{
		FunctionContractRef: "function_contract_ref_001",
	})

	if env.OK || len(env.Errores) != 1 || env.Errores[0].Codigo != CliErrRespuestaInvalidaV0 {
		t.Fatalf("respuesta invalida no detectada: %+v", env)
	}
}

func mustNewFunctionContractCliClientV0(t *testing.T, serverURL string, timeout time.Duration) *FunctionContractCliClientV0 {
	t.Helper()
	client, err := NewFunctionContractCliClientV0(serverURL, timeout)
	if err != nil {
		t.Fatalf("NewFunctionContractCliClientV0: %v", err)
	}
	return client
}

func functionContractInvocationV0(serverURL string) CliInvocationContextV0 {
	return CliInvocationContextV0{
		RequestID:     "req-fc-1",
		CorrelationID: "corr-fc-1",
		ServerURL:     serverURL,
		Timeout:       time.Second,
		OutputFormat:  CliOutputFormatJSONV0,
	}
}
