package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunOrquestaCLIV0HelpEspanolSinRed(t *testing.T) {
	var stdout bytes.Buffer
	code := RunOrquestaCLIV0(context.Background(), []string{"--help"}, OrquestaCLIRunnerV0{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Uso: orquesta-cli")) ||
		!bytes.Contains(stdout.Bytes(), []byte("app spec solicitar")) {
		t.Fatalf("help inesperada: %s", stdout.String())
	}
}

func TestRunOrquestaCLIV0HelpInglesPorLocaleSinRed(t *testing.T) {
	var stdout bytes.Buffer
	code := RunOrquestaCLIV0(context.Background(), []string{"--help", "--locale", "en"}, OrquestaCLIRunnerV0{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: orquesta-cli")) ||
		!bytes.Contains(stdout.Bytes(), []byte("gobernanza catalogo ver")) {
		t.Fatalf("help inesperada: %s", stdout.String())
	}
}

func TestRunOrquestaCLIV0AppSpecSolicitarUsaClienteREST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != SolicitarNuevaAppCliEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(CliCorrelationHeaderV0); got != "corr-cmd-1" {
			t.Fatalf("correlation=%q", got)
		}
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody["source"] != SolicitarNuevaAppCliSourceV0 || reqBody["request_id"] != "req-cmd-1" {
			t.Fatalf("request tecnico inesperado: %+v", reqBody)
		}
		writeSolicitarNuevaAppSuccessV0(t, w, minimalAppSpecRequestForCliV0(), false)
	}))
	defer server.Close()

	body, _ := json.Marshal(minimalAppSpecRequestForCliV0())
	env, code := runCLIAndDecodeEnvelopeV0(t, []string{
		"app", "spec", "solicitar",
		"--server-url", server.URL,
		"--input", "-",
		"--request-id", "req-cmd-1",
		"--correlation-id", "corr-cmd-1",
		"--json",
	}, body)

	if code != 0 || !env.OK || env.Contract != CliContractSolicitarNuevaAppV0 {
		t.Fatalf("env/code inesperados: code=%d env=%+v", code, env)
	}
}

func TestRunOrquestaCLIV0FunctionContractListarReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != FunctionContractCliListEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var req ListarFunctionContractsRequestV0
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Filtros.Modulo != "orquesta-cli" || req.Page.Limit != 7 {
			t.Fatalf("request inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(ListarFunctionContractsResultV0{
			Items: []FunctionContractResumenV0{{
				FunctionContractRef: "function_contract_ref_1",
				Modulo:              "orquesta-cli",
				Estado:              "activa",
			}},
			Warnings: []string{},
		})
	}))
	defer server.Close()

	env, code := runCLIAndDecodeEnvelopeV0(t, []string{
		"contratos", "funcion", "listar",
		"--server-url", server.URL,
		"--module", "orquesta-cli",
		"--limit", "7",
		"--json",
	}, nil)

	if code != 0 || !env.OK || env.Contract != FunctionContractCliContractV0 {
		t.Fatalf("env/code inesperados: code=%d env=%+v", code, env)
	}
}

func TestRunOrquestaCLIV0FunctionContractRegistrarBloqueado(t *testing.T) {
	env, code := runCLIAndDecodeEnvelopeV0(t, []string{"contratos", "funcion", "registrar", "--json"}, nil)
	if code == 0 || env.OK || len(env.Errores) != 1 {
		t.Fatalf("env/code inesperados: code=%d env=%+v", code, env)
	}
	if env.Errores[0].Codigo != FunctionContractCliErrRegistrarBloqueadoV0 {
		t.Fatalf("error inesperado: %+v", env.Errores)
	}
}

func TestRunOrquestaCLIV0RechazaArgumentosExtra(t *testing.T) {
	env, code := runCLIAndDecodeEnvelopeV0(t, []string{"contratos", "funcion", "registrar", "basura", "--json"}, nil)
	if code == 0 || env.OK || len(env.Errores) != 1 {
		t.Fatalf("env/code inesperados: code=%d env=%+v", code, env)
	}
	if env.Errores[0].Codigo != CliErrOpcionInvalidaV0 ||
		env.Errores[0].Campo != "args" ||
		env.Errores[0].Detalle != "argumentos_posicionales_no_soportados" {
		t.Fatalf("error inesperado: %+v", env.Errores)
	}
}

func TestRunOrquestaCLIV0GovernanceCatalogoVerReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != GovernanceCatalogCliEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var req governanceCatalogQueryHTTPRequestV0
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Module != "orquesta-cli" || req.Role != "director" {
			t.Fatalf("request inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(testGovernanceCatalogResultV0())
	}))
	defer server.Close()

	env, code := runCLIAndDecodeEnvelopeV0(t, []string{
		"gobernanza", "catalogo", "ver",
		"--server-url", server.URL,
		"--module", "orquesta-cli",
		"--role", "director",
		"--json",
	}, nil)

	if code != 0 || !env.OK || env.Contract != CliContractGovernanceCatalogV0 {
		t.Fatalf("env/code inesperados: code=%d env=%+v", code, env)
	}
}

func runCLIAndDecodeEnvelopeV0(t *testing.T, args []string, stdin []byte) (CliOutputEnvelopeV0, int) {
	t.Helper()
	var stdout bytes.Buffer
	code := RunOrquestaCLIV0(context.Background(), args, OrquestaCLIRunnerV0{
		Stdin:  bytes.NewReader(stdin),
		Stdout: &stdout,
	})
	var env CliOutputEnvelopeV0
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	return env, code
}
