package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestRESTRunControlClientV0SerializaAccionesSoportadas(t *testing.T) {
	for _, action := range []string{
		WebRunControlActionPauseV0,
		WebRunControlActionResumeV0,
		WebRunControlActionStopV0,
		WebRunControlActionCancelV0,
	} {
		t.Run(action, func(t *testing.T) {
			var got orquestamcp.MCPRunControlToolInputV0
			server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != WebRunControlInboundEndpointV0 {
					t.Fatalf("path=%s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Fatalf("method=%s", r.Method)
				}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatalf("decode: %v", err)
				}
				_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunControlToolResultV0{
					Estado: orquestamcp.MCPRunControlEstadoOKV0,
					Action: got.Action,
					RunRef: got.RunRef,
					Status: got.Action + "_requested",
					Forced: got.Forced,
				})
			}))
			defer server.Close()

			client := NewRESTRunControlClientV0(server.URL, time.Second)
			vm, err := client.EnviarRunControl(context.TODO(), WebRunControlCommandV0{
				Action:       " " + action + " ",
				RunRef:       " run-web-control-001 ",
				Forced:       true,
				RequestedBy:  " operador ",
				EvidenceRefs: []string{" ev-001 ", ""},
			})

			if err != nil {
				t.Fatalf("EnviarRunControl: %v", err)
			}
			if got.Action != action || got.RunRef != "run-web-control-001" ||
				!got.Forced || got.RequestedBy != "operador" ||
				len(got.EvidenceRefs) != 1 || vm.Status != action+"_requested" {
				t.Fatalf("got=%+v vm=%+v", got, vm)
			}
		})
	}
}

func TestRESTRunControlClientV0ConservaErrorPublico400(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunControlToolResultV0{
			Estado: orquestamcp.MCPRunControlEstadoErrorV0,
			Errores: []orquestamcp.MCPValidationIssueV0{{
				Code:  "run_ref_requerido",
				Field: "run_ref",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTRunControlClientV0(server.URL, time.Second)
	vm, err := client.EnviarRunControl(context.TODO(), WebRunControlCommandV0{Action: "pause"})

	if err != nil {
		t.Fatalf("EnviarRunControl: %v", err)
	}
	if vm.Estado != WebRunControlEstadoErrorV0 ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "run_ref_requerido" {
		t.Fatalf("vm=%+v", vm)
	}
}
