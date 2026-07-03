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
		if request["cleanup_goal_backends"] != true {
			t.Fatalf("cleanup_goal_backends default=%v", request["cleanup_goal_backends"])
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

func TestRequestServerShutdownV0ReadyNoSaltaActiveWorkPersistido(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
			Estado:          "ok",
			Status:          "ready",
			ShutdownReady:   true,
			ActiveWorkCount: 1,
			ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-ready"},
		})
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err == nil || !strings.Contains(err.Error(), "shutdown_not_ready status=ready") {
		t.Fatalf("ready con active_work persistido no debe cerrar signal: %v", err)
	}
}

func TestRequestServerShutdownV0ReadyNoSaltaActiveWorksEstructuradosV0(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
			Estado:        "ok",
			Status:        "ready",
			ShutdownReady: true,
			ActiveWorks: []serverShutdownClientActiveWorkV0{{
				Kind:            "goal_backend",
				RunRef:          "run-ref-ready-structured",
				WorkRef:         "goal-ref-ready-structured",
				ExternalWorkRef: "thread-ref-ready-structured",
				Status:          "backend_still_running",
			}},
		})
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err == nil ||
		!strings.Contains(err.Error(), "shutdown_not_ready status=ready") ||
		!strings.Contains(err.Error(), "active_work=1") {
		t.Fatalf("ready con active_works estructurado no debe cerrar signal: %v", err)
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
		ShutdownInProgress:              true,
		ShutdownAgentsInFlight:          0,
		ShutdownCheckpointsPending:      0,
		ShutdownCheckpointAgentsPending: 0,
		ShutdownAsyncWorkActive:         0,
	}

	if !shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout con shutdown drenado debe permitir signal cooperativa")
	}
	status.ShutdownAgentsInFlight = 1
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout con agentes en vuelo no debe permitir signal")
	}
	status.ShutdownAgentsInFlight = 0
	status.ShutdownCheckpointAgentsPending = 1
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout con agentes checkpoint pendientes no debe permitir signal")
	}
	status.ShutdownCheckpointAgentsPending = 0
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, false) {
		t.Fatalf("request_failed no debe permitir signal sin force")
	}
	if !shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, true) {
		t.Fatalf("force debe permitir signal tras error de transporte de shutdown")
	}
	status.ShutdownActiveWorkCount = 1
	status.ShutdownActiveWorkRefs = []string{"shutdown-active-work-goal-backend-goal-ref-force"}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, true) {
		t.Fatalf("force con active_work persistido no debe permitir signal")
	}
	status.ShutdownActiveWorkCount = 0
	status.ShutdownActiveWorkRefs = nil
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_http_409"), status, true) {
		t.Fatalf("force no debe saltar conflicto HTTP de trabajo vivo")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=backend_still_running runs=0/0 agents_in_flight=0 checkpoints=0 checkpoint_agents=0"), status, true) {
		t.Fatalf("force no debe saltar backend vivo")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=ready runs=1/1 agents_in_flight=0 checkpoints=0 checkpoint_agents=0 active_work=1 active_work_refs=shutdown-active-work-goal-backend-goal-ref-force"), status, true) {
		t.Fatalf("force no debe saltar active_work conservado en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_status_active_goals_present"), status, true) {
		t.Fatalf("force no debe saltar goals activos")
	}
}

func TestShutdownClientReadyV0BloqueaActiveWorkPersistido(t *testing.T) {
	status := orquestaserver.ServerPublicStatusV0{
		ShutdownInProgress:              true,
		ShutdownReady:                   true,
		ShutdownRunsRequested:           1,
		ShutdownRunsStopped:             1,
		ShutdownAsyncWorkActive:         0,
		ShutdownActiveWorkCount:         1,
		ShutdownActiveWorkRefs:          []string{"shutdown-active-work-goal-backend-goal-ref-test"},
		ShutdownCheckpointAgentsPending: 1,
	}

	if shutdownPublicStatusReadyForSignalV0(status) {
		t.Fatalf("status con active_work persistido no debe permitir signal")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout sin cuerpo no debe permitir signal si status conserva active_work")
	}
	result := serverShutdownClientResultFromStatusV0(status)
	if shutdownClientResultReadyForSignalV0(result) ||
		result.ActiveWorkCount != 1 ||
		result.CheckpointAgentsPending != 1 ||
		!shutdownClientStringRefsContainForTestV0(result.ActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-test") {
		t.Fatalf("result active_work no conservado/bloqueado: %+v", result)
	}
}

func TestShutdownClientNotReadyErrorV0IncluyeRefsActiveWorkCompactas(t *testing.T) {
	err := shutdownClientNotReadyErrorV0(serverShutdownClientResultV0{
		Status:          "ready",
		ShutdownReady:   true,
		ActiveWorkCount: 2,
		ActiveWorkRefs: []string{
			"shutdown-active-work-goal-backend-goal-ref-001",
			"shutdown-active-work-goal-backend-thread-ref-001",
			"shutdown-active-work-goal-backend-extra-ref-ignored",
			"/tmp/no-publicar",
		},
	})

	if err == nil ||
		!strings.Contains(err.Error(), "active_work=2") ||
		!strings.Contains(err.Error(), "active_work_refs=shutdown-active-work-goal-backend-goal-ref-001,shutdown-active-work-goal-backend-thread-ref-001") ||
		strings.Contains(err.Error(), "/tmp/no-publicar") ||
		strings.Contains(err.Error(), "extra-ref-ignored") {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeServerShutdownClientResultV0ConvierteActiveWorksEnRefs(t *testing.T) {
	result := normalizeServerShutdownClientResultV0(serverShutdownClientResultV0{
		ActiveWorks: []serverShutdownClientActiveWorkV0{{
			Kind:            "goal_backend",
			RunRef:          "run-ref-client-active-001",
			WorkRef:         "goal-ref-client-active-001",
			ExternalWorkRef: "thread-ref-client-active-001",
			Status:          "backend_still_running",
		}},
	})

	if result.ActiveWorkCount != 1 ||
		!shutdownClientStringRefsContainForTestV0(result.ActiveWorkRefs, "shutdown-active-work-goal-backend-run-ref-client-active-001") ||
		!shutdownClientStringRefsContainForTestV0(result.ActiveWorkRefs, "shutdown-active-work-goal-backend-goal-ref-client-active-001") ||
		!shutdownClientStringRefsContainForTestV0(result.ActiveWorkRefs, "shutdown-active-work-goal-backend-thread-ref-client-active-001") ||
		!shutdownClientStringRefsContainForTestV0(result.ActiveWorkRefs, "shutdown-active-work-goal-backend-backend-still-running") {
		t.Fatalf("active_works no normalizado: %+v", result)
	}
}

func assertShutdownClientErrorV0(message string) error {
	return shutdownClientTestErrorV0(message)
}

type shutdownClientTestErrorV0 string

func (err shutdownClientTestErrorV0) Error() string {
	return string(err)
}

func shutdownClientStringRefsContainForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
