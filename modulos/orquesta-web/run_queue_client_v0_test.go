package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestRESTRunQueueClientV0EnviaRankYProyectaVista(t *testing.T) {
	var received orquestamcp.MCPRunQueuePriorityToolInputV0
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != WebRunQueueInboundEndpointV0 {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunQueuePriorityToolResultV0{
			Estado:   orquestamcp.MCPRunQueuePriorityEstadoOKV0,
			Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
			QueueRef: "global",
			Count:    1,
			Ranked: []orquestamcp.MCPRunQueueRankedCandidateCompactV0{{
				Rank:          1,
				RunRef:        "run-ref-web-queue-001",
				AppRef:        "app-ref-web-queue-001",
				Status:        "ready",
				PriorityScore: 80,
			}},
		})
	}))
	defer server.Close()

	client := NewRESTRunQueueClientV0(server.URL, time.Second)
	vm, err := client.ConsultarRunQueue(context.Background(), WebRunQueueQueryV0{
		RequestID:     "request-ref-web-queue-001",
		CorrelationID: "corr-web-queue-001",
		Locale:        "es",
		Action:        "rank",
		QueueRef:      "global",
		Status:        " ready ",
		Limit:         5,
	})

	if err != nil {
		t.Fatalf("ConsultarRunQueue: %v", err)
	}
	if received.Action != "rank" ||
		received.QueueRef != "global" ||
		received.Status != "ready" ||
		received.Limit != 5 ||
		vm.Estado != WebRunQueueEstadoOKV0 ||
		len(vm.Ranked) != 1 ||
		vm.Ranked[0].PriorityScore != 80 {
		t.Fatalf("received=%+v vm=%+v", received, vm)
	}
}

func TestRESTRunQueueClientV0SetPriorityErroresPublicosNoSonTransporte(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunQueuePriorityToolResultV0{
			Estado: orquestamcp.MCPRunQueuePriorityEstadoErrorV0,
			Errores: []orquestamcp.MCPValidationIssueV0{{
				Code:  "run_ref_requerido",
				Field: "run_ref",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTRunQueueClientV0(server.URL, time.Second)
	vm, err := client.ConsultarRunQueue(context.Background(), WebRunQueueQueryV0{
		Action: WebRunQueueActionSetV0,
	})

	if err != nil {
		t.Fatalf("error publico no debe ser transporte: %v", err)
	}
	if vm.Estado != WebRunQueueEstadoErrorV0 ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "run_ref_requerido" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTRunQueueClientV0ConservaErroresPublicosEnTimeout(t *testing.T) {
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_ = json.NewEncoder(w).Encode(orquestamcp.MCPRunQueuePriorityToolResultV0{
			Estado: orquestamcp.MCPRunQueuePriorityEstadoErrorV0,
			Action: orquestamcp.MCPRunQueuePriorityActionSetV0,
			Errores: []orquestamcp.MCPValidationIssueV0{{
				Code:    "run_queue_priority_timeout",
				Field:   "executor",
				Message: "consulta de cola excedio la ventana HTTP acotada",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTRunQueueClientV0(server.URL, time.Second)
	vm, err := client.ConsultarRunQueue(context.Background(), WebRunQueueQueryV0{
		Action: WebRunQueueActionSetV0,
	})

	if err != nil {
		t.Fatalf("timeout publico no debe perderse como transporte: %v", err)
	}
	if vm.Estado != WebRunQueueEstadoErrorV0 ||
		vm.Action != WebRunQueueActionSetV0 ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "run_queue_priority_timeout" {
		t.Fatalf("vm=%+v", vm)
	}
}
