package orquestaweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRESTArrancarDirectorAppClientV0EnviaPOSTJSONYProyectaDirector(t *testing.T) {
	var received arrancarDirectorAppRequestEnvelopeV0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != ArrancarDirectorAppEndpointV0 {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get(ArrancarDirectorAppCorrelationV0); got != "req-director-client-001" {
			t.Fatalf("correlation=%q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeDirectorSuccessV0(t, w)
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	client.Limits = WebArrancarDirectorAppLimitsV0{
		MaxBursts:            4,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 2,
		MaxCommands:          9,
		MaxOutboxPerCycle:    5,
		MaxExternalWaits:     1,
	}

	vm, err := client.ArrancarDirectorApp(
		context.Background(),
		minimalFormForClientV0("req-director-client-001"),
	)
	if err != nil {
		t.Fatalf("ArrancarDirectorApp: %v", err)
	}
	if received.AppSpecRequest.RequestID != "req-director-client-001" ||
		received.AppSpecRequest.Source != WebNuevaAppSourceV0 ||
		received.MaxBursts != 4 ||
		received.MaxStepsPerBurst != 3 ||
		received.MaxDispatchesPerWait != 2 ||
		received.MaxCommands != 9 ||
		received.MaxOutboxPerCycle != 5 ||
		received.MaxExternalWaits != 1 {
		t.Fatalf("payload inesperado: %+v", received)
	}
	if vm.Estado != WebNuevaAppEstadoDirector ||
		vm.Director == nil ||
		vm.Director.RunRef != "run-ref-director-client-001" ||
		len(vm.Director.DirectorTasks) != 1 ||
		vm.Director.DirectorTasks[0].Capacity != "high" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTArrancarDirectorAppClientV0ErroresPublicosNoSonTransporte(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(WebArrancarDirectorAppResultV0{
			Estado:    ArrancarDirectorAppEstadoErrorV0,
			RequestID: "req-director-invalid",
			Errores: []WebNuevaAppIssueV0{{
				Code:    "app_spec_invalida",
				Field:   "nombre",
				Message: "nombre requerido",
			}},
		})
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	vm, err := client.ArrancarDirectorApp(
		context.Background(),
		minimalFormForClientV0("req-director-invalid"),
	)
	if err != nil {
		t.Fatalf("error publico no debe ser transporte: %v", err)
	}
	if vm.Estado != WebNuevaAppEstadoInvalida ||
		len(vm.ErroresPublicos) != 1 ||
		vm.ErroresPublicos[0].Code != "app_spec_invalida" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestRESTArrancarDirectorAppClientV0Status500DevuelveErrorPublico(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "stack privado", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRESTArrancarDirectorAppClientV0(server.URL, time.Second)
	_, err := client.ArrancarDirectorApp(
		context.Background(),
		minimalFormForClientV0("req-director-500"),
	)
	var clientErr WebNuevaAppClientErrorV0
	if !errors.As(err, &clientErr) ||
		clientErr.Code != WebNuevaAppErrTransporteV0 ||
		clientErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error inesperado: %#v", err)
	}
}

func writeDirectorSuccessV0(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(WebArrancarDirectorAppResultV0{
		Estado:     ArrancarDirectorAppEstadoOKV0,
		RequestID:  "req-director-client-001",
		RunRef:     "run-ref-director-client-001",
		PhaseID:    "brainstorming_arquitectura",
		LoopStatus: "quiescent",
		DirectorTasks: []WebNuevaAppDirectorTaskV0{{
			TaskRef:        "task-ref-director-client-001",
			AgentRequestID: "agent-ref-director-client-001",
			Capacity:       "high",
		}},
		StartedAgents: []string{"agent-ref-director-client-001"},
		EvidenceRefs:  []string{"evidence-ref-director-client-001"},
	}); err != nil {
		t.Fatalf("encode success: %v", err)
	}
}
