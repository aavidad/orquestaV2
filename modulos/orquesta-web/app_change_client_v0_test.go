package orquestaweb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestRESTAppChangeClientV0EnviaContratoYDecodificaResultado(t *testing.T) {
	var got orquestamcp.MCPRequestAppChangeToolInputV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != AppChangeEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPRequestAppChangeToolResultV0{
			Estado:              orquestamcp.MCPRequestAppChangeEstadoOKV0,
			RunRef:              got.AppChangeRequest.RunRef,
			ChangeRef:           got.AppChangeRequest.ChangeRef,
			DirectorQuestionRef: "question-ref-client-change-001",
		})
	}))
	defer server.Close()
	client := NewRESTAppChangeClientV0(server.URL, 0)

	vm, err := client.RequestAppChange(nil, WebAppChangeFormV0{
		RunRef:     "run-ref-client-change-001",
		ChangeRef:  "change-ref-client-001",
		UserIntent: "Cambiar la web.",
	})
	if err != nil {
		t.Fatalf("RequestAppChange: %v", err)
	}
	if vm.Estado != WebAppChangeEstadoAceptadoV0 ||
		vm.DirectorQuestionRef != "question-ref-client-change-001" ||
		got.AppChangeRequest.RunRef != "run-ref-client-change-001" {
		t.Fatalf("vm=%+v got=%+v", vm, got)
	}
}
