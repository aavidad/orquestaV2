package orquestacli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestRunOrquestaCLIV0AutoprogramacionRunControlarUsaAPI(t *testing.T) {
	var input orquestamcp.MCPRunControlToolInputV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != AutoprogrammingRunControlCliEndpointV0 {
			t.Fatalf("request inesperada %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunControlToolResultV0{
			Estado: orquestamcp.MCPRunControlEstadoOKV0,
			Action: "pause",
			RunRef: "run-ref-cli-control-001",
			Status: "paused",
		})
	}))
	defer server.Close()

	env, code := runCLIAndDecodeEnvelopeV0(t, []string{
		"autoprogramacion", "run", "controlar",
		"--server-url", server.URL,
		"--action", "pause",
		"--run-ref", "run-ref-cli-control-001",
		"--requested-by", "operator-ref-cli",
		"--reason", "supervision",
		"--idempotency-key", "idem-cli-control-001",
		"--evidence-refs", "evidence-ref-cli-control-001",
		"--json",
	}, nil)

	if code != 0 || !env.OK || env.Contract != CliContractAutoprogrammingV0 {
		t.Fatalf("env/code inesperados: code=%d env=%+v", code, env)
	}
	if input.Action != "pause" ||
		input.RunRef != "run-ref-cli-control-001" ||
		input.RequestedBy != "operator-ref-cli" ||
		input.IdempotencyKey != "idem-cli-control-001" ||
		len(input.EvidenceRefs) != 1 {
		t.Fatalf("input=%+v", input)
	}
}
