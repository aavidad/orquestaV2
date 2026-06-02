package orquestaoperatormcphermes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	operator "orquesta/modulos/orquesta-operator-mcp"
	operatorclient "orquesta/modulos/orquesta-operator-mcp-client"
)

func TestHermesOperatorMCPConnectorV0LlamaMCPPorHTTP(t *testing.T) {
	var gotTool string
	var gotConnectorRef string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Fatalf("path=%s want /mcp", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-ref-hermes-test" {
			t.Fatalf("authorization header inesperado")
		}
		var request struct {
			Method string `json:"method"`
			Params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		gotTool = request.Params.Name
		var input operator.OperatorStatusQueryV0
		if err := json.Unmarshal(request.Params.Arguments, &input); err != nil {
			t.Fatalf("decode arguments: %v", err)
		}
		gotConnectorRef = input.StatusConnectorRef
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc":"2.0",
			"id":"hermes-test",
			"result":{
				"content":[{
					"type":"text",
					"mimeType":"application/json",
					"text":"{\"estado\":\"ok\",\"status\":{\"status\":\"healthy\",\"evidence_refs\":[\"evidence-ref-hermes-api\"]}}"
				}]
			}
		}`))
	}))
	defer server.Close()

	connector, err := NewHermesOperatorMCPConnectorV0(HermesOperatorMCPConfigV0{
		BaseURL: server.URL,
		APIKey:  "token-ref-hermes-test",
		ToolNames: operatorclient.OperatorMCPClientToolNamesV0{
			Status: "hermes.status",
		},
		ConnectorRefs: operatorclient.OperatorMCPClientConnectorRefsV0{
			Status: "hermes-status-ref",
		},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewHermesOperatorMCPConnectorV0: %v", err)
	}
	result, err := connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{
		RequestRef:         "request-ref-hermes-test",
		SubjectRef:         "run-ref-hermes-test",
		StatusConnectorRef: "ignored-status-ref",
	})
	if err != nil {
		t.Fatalf("QueryOperatorStatusV0: %v", err)
	}
	if result.Status != "healthy" || gotTool != "hermes.status" || gotConnectorRef != "hermes-status-ref" {
		t.Fatalf("result=%+v gotTool=%s gotConnectorRef=%s", result, gotTool, gotConnectorRef)
	}
}

func TestHermesMCPJSONRPCClientV0MapeaErroresPublicos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jsonrpc":"2.0",
			"id":"hermes-test",
			"error":{"code":-32000,"message":"remote","data":{"error_code":"operator_mcp_connector_unavailable"}}
		}`))
	}))
	defer server.Close()
	connector, err := NewHermesOperatorMCPConnectorV0(HermesOperatorMCPConfigV0{
		BaseURL: server.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewHermesOperatorMCPConnectorV0: %v", err)
	}
	_, err = connector.QueryOperatorStatusV0(operator.OperatorStatusQueryV0{
		RequestRef:         "request-ref-hermes-test",
		SubjectRef:         "run-ref-hermes-test",
		StatusConnectorRef: "status-ref-hermes-test",
	})
	assertHermesPublicCodeV0(t, err, operator.ErrOperatorMCPConnectorUnavailableV0)
}

func TestHermesMCPJSONRPCClientV0ValidaEndpoint(t *testing.T) {
	for _, baseURL := range []string{
		"",
		"file:///tmp/hermes",
		"http://user:pass@127.0.0.1:8787/mcp",
		"http://127.0.0.1:8787/api/mcp",
	} {
		_, err := NewHermesMCPJSONRPCClientV0(HermesMCPJSONRPCClientConfigV0{BaseURL: baseURL})
		assertHermesPublicCodeV0(t, err, operator.ErrOperatorMCPConnectorUnavailableV0)
	}
}

func assertHermesPublicCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	got, ok := operator.PublicOperatorMCPErrorCodeV0(err)
	if !ok || got != code {
		t.Fatalf("error publico inesperado: got=%q ok=%v err=%v want=%q", got, ok, err, code)
	}
}
