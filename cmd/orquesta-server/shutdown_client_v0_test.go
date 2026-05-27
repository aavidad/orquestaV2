package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestRequestServerShutdownV0DefaultCooperativo(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("request json=%v", err)
		}
		if request["forced"] != false {
			t.Fatalf("forced default=%v", request["forced"])
		}
		refs, ok := request["evidence_refs"].([]any)
		if !ok || !shutdownClientBodyRefsContainForTestV0(refs, orquestaruncontrol.RunControlEvidenceAutoResumeAllowedV0) {
			t.Fatalf("evidence_refs sin auto-resume: %#v", request["evidence_refs"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
			Estado:        "ok",
			Status:        "ready",
			ShutdownReady: true,
		})
	}))
	defer server.Close()

	if err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{}); err != nil {
		t.Fatalf("requestServerShutdownV0: %v", err)
	}
}

func TestParseStopOptionsV0ExigeRazonParaForce(t *testing.T) {
	_, err := parseStopOptionsV0([]string{"--force"}, bytes.NewBuffer(nil))

	if err == nil || err.Error() != "forced_reason_required" {
		t.Fatalf("error=%v", err)
	}
}

func TestRequestServerShutdownV0NoPropagaBodyCrudoEnError(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.Error(w, "secret_token=abc local_path=/tmp/private", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err == nil {
		t.Fatalf("error esperado")
	}
	if got := err.Error(); got != "shutdown_http_500" {
		t.Fatalf("error=%q", got)
	}
	if strings.Contains(err.Error(), "secret_token") || strings.Contains(err.Error(), "/tmp/private") {
		t.Fatalf("error filtra body crudo: %v", err)
	}
}

func TestRequestServerShutdownV0RechazaJSONConTrailingData(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"estado":"ok","shutdown_ready":true} {"extra":true}`))
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err == nil || err.Error() != "shutdown_response_trailing_data" {
		t.Fatalf("error=%v", err)
	}
}

func TestRequestServerShutdownV0EsperaWaitingDrainHastaEstadoDrenado(t *testing.T) {
	statusCalls := 0
	shutdownCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:         "ok",
				Status:         "waiting_drain",
				ShutdownReady:  false,
				RunsRequested:  1,
				RunsStopped:    0,
				AgentsInFlight: 1,
			})
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                     "running",
				ShutdownInProgress:         true,
				ShutdownStatus:             "waiting_drain",
				ShutdownReady:              true,
				ShutdownRunsRequested:      1,
				ShutdownRunsStopped:        1,
				ShutdownAgentsInFlight:     0,
				ShutdownCheckpointsPending: 0,
				ShutdownAsyncWorkActive:    0,
			})
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err != nil {
		t.Fatalf("requestServerShutdownV0: %v", err)
	}
	if statusCalls == 0 {
		t.Fatalf("no consulto estado para completar waiting_drain")
	}
	if shutdownCalls == 0 {
		t.Fatalf("no llamo shutdown")
	}
}

func shutdownClientBodyRefsContainForTestV0(values []any, want string) bool {
	for _, value := range values {
		if got, ok := value.(string); ok && got == want {
			return true
		}
	}
	return false
}

func TestRequestServerShutdownV0ReintentaPOSTParaIngerirCheckpointTardio(t *testing.T) {
	shutdownCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			if shutdownCalls == 1 {
				_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
					Estado:                  "ok",
					Status:                  "waiting_checkpoint",
					ShutdownReady:           false,
					RunsRequested:           1,
					RunsStopped:             0,
					CheckpointsPending:      1,
					CheckpointAgentsPending: 1,
				})
				return
			}
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:                  "ok",
				Status:                  "ready",
				ShutdownReady:           true,
				RunsRequested:           1,
				RunsStopped:             1,
				CheckpointsPending:      0,
				CheckpointAgentsPending: 0,
			})
		case orquestaserver.ServerStatusEndpointV0:
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                     "running",
				ShutdownInProgress:         true,
				ShutdownStatus:             "waiting_checkpoint",
				ShutdownReady:              false,
				ShutdownRunsRequested:      1,
				ShutdownRunsStopped:        0,
				ShutdownAgentsInFlight:     1,
				ShutdownCheckpointsPending: 1,
				ShutdownAsyncWorkActive:    0,
			})
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err != nil {
		t.Fatalf("requestServerShutdownV0: %v", err)
	}
	if shutdownCalls < 2 {
		t.Fatalf("shutdownCalls=%d, esperaba rePOST para ACK tardio", shutdownCalls)
	}
}

func TestWaitServerShutdownReadyV0NoEsperaEstadosNoRecuperables(t *testing.T) {
	started := time.Now()
	err := waitServerShutdownReadyV0(
		"127.0.0.1:1",
		serverShutdownClientOptionsV0{},
		serverShutdownClientResultV0{Status: "blocked", RunsRequested: 1},
		serverShutdownClientRequestIdentityV0{},
		time.Minute,
		time.Millisecond,
	)

	if err == nil || !strings.Contains(err.Error(), "shutdown_not_ready status=blocked") {
		t.Fatalf("err=%v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatalf("estado no recuperable no debe esperar")
	}
}

func TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado(t *testing.T) {
	status := orquestaserver.ServerPublicStatusV0{
		ShutdownInProgress:         true,
		ShutdownAgentsInFlight:     0,
		ShutdownCheckpointsPending: 0,
		ShutdownAsyncWorkActive:    0,
	}

	if !shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout con shutdown drenado debe permitir signal cooperativa")
	}
	status.ShutdownAgentsInFlight = 1
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout con agentes en vuelo no debe permitir signal")
	}
	status.ShutdownAgentsInFlight = 0
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, false) {
		t.Fatalf("request_failed no debe permitir signal sin force")
	}
	if !shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, true) {
		t.Fatalf("force debe permitir signal tras error de shutdown")
	}
}

func assertShutdownClientErrorV0(message string) error {
	return shutdownClientTestErrorV0(message)
}

type shutdownClientTestErrorV0 string

func (err shutdownClientTestErrorV0) Error() string {
	return string(err)
}
