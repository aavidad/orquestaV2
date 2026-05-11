package orquestacli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
)

func TestFunctionContractCliClientV0TimeoutDevuelveErrorTransporte(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeFunctionContractListSuccessV0(t, w)
	}))
	defer server.Close()

	client := mustNewFunctionContractCliClientV0(t, server.URL, 2*time.Millisecond)
	env := client.ListarFunctionContracts(context.Background(), functionContractInvocationV0(server.URL), ListarFunctionContractsRequestV0{})

	if env.OK || len(env.Errores) != 1 {
		t.Fatalf("timeout envelope inesperado: %+v", env)
	}
	if env.Errores[0].Codigo != CliErrErrorTransporteV0 || env.Errores[0].Detalle != "timeout" || !env.Meta.Retryable {
		t.Fatalf("timeout error inesperado: errores=%+v meta=%+v", env.Errores, env.Meta)
	}
}

func TestFunctionContractCliClientV0RechazaServerURLConCredenciales(t *testing.T) {
	_, err := NewFunctionContractCliClientV0("https://usuario:secreto@example.test", time.Second)
	if !IsCliClientErrorCodeV0(err, CliErrConfiguracionInvalidaV0) {
		t.Fatalf("error constructor=%v", err)
	}
	var cliErr CliClientErrorV0
	if !errors.As(err, &cliErr) || cliErr.Field != "server_url" || cliErr.Detail != "server_url_no_debe_contener_credenciales" {
		t.Fatalf("error publico inesperado: %+v", cliErr)
	}
	if strings.Contains(cliErr.Detail, "usuario") || strings.Contains(cliErr.Detail, "secreto") {
		t.Fatalf("detalle filtra credenciales: %+v", cliErr)
	}
}

func TestFunctionContractCliClientV0RegistrarBloqueado(t *testing.T) {
	client := &FunctionContractCliClientV0{}
	env := client.RegistrarFunctionContract(context.Background(), CliInvocationContextV0{
		RequestID:     "req-fc-reg",
		CorrelationID: "corr-fc-reg",
		OutputFormat:  CliOutputFormatJSONV0,
	}, validFunctionContractForCliV0())

	if env.OK || env.Data != nil || env.Meta.StatusCode != 0 || env.Meta.Retryable {
		t.Fatalf("envelope bloqueo inesperado: %+v", env)
	}
	if len(env.Errores) != 1 {
		t.Fatalf("errores=%+v", env.Errores)
	}
	if err := env.Errores[0]; err.Codigo != FunctionContractCliErrRegistrarBloqueadoV0 ||
		err.Campo != "registrar" ||
		err.MensajeI18N != CliDefaultErrorMessageNamespaceV0+FunctionContractCliErrRegistrarBloqueadoV0 {
		t.Fatalf("error bloqueo inesperado: %+v", err)
	}
}

func writeFunctionContractListSuccessV0(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	result := ListarFunctionContractsResultV0{
		Items: []FunctionContractResumenV0{{
			FunctionContractRef: "function_contract_ref_001",
			Titulo:              "Cerrar CLI-004",
			Modulo:              "orquesta-cli",
			ArchivoObjetivo:     "function_contract_client_v0.go",
			SimboloObjetivo:     "FunctionContractCliClientV0",
			Estado:              orquestacore.FunctionContractEstadoActivaV0,
			Version:             1,
		}},
		Warnings: []string{},
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		t.Fatalf("encode list response: %v", err)
	}
}

func writeFunctionContractViewSuccessV0(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	result := VerFunctionContractResultV0{
		FunctionContract: validFunctionContractForCliV0(),
		Warnings:         []string{},
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		t.Fatalf("encode view response: %v", err)
	}
}

func validFunctionContractForCliV0() orquestacore.FunctionContractV0 {
	return orquestacore.FunctionContractV0{
		Titulo:            "Cerrar CLI-004",
		Objetivo:          "Implementar cliente CLI read-only para FunctionContract v0.",
		ArchivoObjetivo:   "function_contract_client_v0.go",
		SimboloObjetivo:   "FunctionContractCliClientV0",
		WriteSet:          []string{"function_contract_client_v0.go", "function_contract_client_helpers_v0.go"},
		TestsObligatorios: []string{"go test -count=1 ./modulos/orquesta-cli"},
		FormatoEntrega:    orquestacore.FunctionContractFormatoPatchEvidenciaV0,
		CriterioCierre:    []string{"listar/ver usan REST publico", "registrar queda bloqueado"},
		Estado:            orquestacore.FunctionContractEstadoActivaV0,
		Version:           1,
	}
}
