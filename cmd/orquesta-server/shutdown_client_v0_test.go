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

func TestRequestServerShutdownV0ReadyNoSaltaActiveWorkRefsSinContadorV0(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
			Estado:         "ok",
			Status:         "ready",
			ShutdownReady:  true,
			ActiveWorkRefs: []string{"shutdown-active-work-goal-backend-goal-ref-ready-sin-contador"},
		})
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err == nil ||
		!strings.Contains(err.Error(), "shutdown_not_ready status=ready") ||
		!strings.Contains(err.Error(), "active_work=1") ||
		!strings.Contains(err.Error(), "shutdown-active-work-goal-backend-goal-ref-ready-sin-contador") {
		t.Fatalf("ready con active_work_refs sin contador no debe cerrar signal: %v", err)
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

func TestRequestServerShutdownV0ReadyNoSaltaGoalActionsSinActiveWorkV0(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
			Estado:        "ok",
			Status:        "ready",
			ShutdownReady: true,
			GoalActions: []orquestaserver.ShutdownGoalActionV0{{
				Kind:        "goal_backend",
				WorkRef:     "goal-ref-action-ready-001",
				ActionTaken: "cleanup_required",
			}},
		})
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{})

	if err == nil ||
		!strings.Contains(err.Error(), "shutdown_not_ready status=ready") ||
		!strings.Contains(err.Error(), "active_work=1") ||
		!strings.Contains(err.Error(), "shutdown-goal-action-goal-backend-goal-ref-action-ready-001") {
		t.Fatalf("ready con goal_actions sin active_work no debe cerrar signal: %v", err)
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

func TestRequestServerShutdownV0ErrorTransporteConsultaStatusAccionableV0(t *testing.T) {
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Fatalf("response writer sin hijacker")
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownReady:           false,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-post-error"},
			}))
		default:
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{Forced: true, Reason: "test force con POST sin cuerpo"})

	if shutdownCalls != 1 || statusCalls != 1 {
		t.Fatalf("calls shutdown=%d status=%d", shutdownCalls, statusCalls)
	}
	if err == nil ||
		!strings.Contains(err.Error(), "shutdown_not_ready status=backend_still_running") ||
		!strings.Contains(err.Error(), "active_work=1") ||
		!strings.Contains(err.Error(), "shutdown-active-work-goal-backend-goal-ref-post-error") {
		t.Fatalf("err=%v", err)
	}
	if shutdownRequestErrorAllowsSignalV0(err, orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe permitir signal si el error ya conserva active_work")
	}
}

func TestRequestServerShutdownV0PostColgadoConsultaStatusAccionableV0(t *testing.T) {
	withShutdownClientRequestTimeoutForTestV0(t, 20*time.Millisecond)
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			time.Sleep(60 * time.Millisecond)
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownReady:           false,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-post-hang"},
			}))
		default:
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	started := time.Now()
	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{Forced: true, Reason: "test force con POST colgado"})

	if time.Since(started) > time.Second {
		t.Fatalf("POST colgado no debe esperar timeout global")
	}
	if shutdownCalls != 1 || statusCalls != 1 {
		t.Fatalf("calls shutdown=%d status=%d", shutdownCalls, statusCalls)
	}
	if err == nil ||
		!strings.Contains(err.Error(), "shutdown_not_ready status=backend_still_running") ||
		!strings.Contains(err.Error(), "active_work=1") ||
		!strings.Contains(err.Error(), "shutdown-active-work-goal-backend-goal-ref-post-hang") {
		t.Fatalf("err=%v", err)
	}
	if shutdownRequestErrorAllowsSignalV0(err, orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe permitir signal si el error ya conserva active_work")
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

func TestRequestServerShutdownV0ReintentaCleanupBackendStillRunningHastaReady(t *testing.T) {
	shutdownCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("request json=%v", err)
			}
			if request["cleanup_goal_backends"] != true {
				t.Fatalf("cleanup_goal_backends=%v", request["cleanup_goal_backends"])
			}
			if shutdownCalls == 1 {
				_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
					Estado:          "ok",
					Status:          "backend_still_running",
					ShutdownReady:   false,
					RunsRequested:   0,
					RunsStopped:     0,
					ActiveWorkCount: 1,
					ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-cleanup-retry"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:        "ok",
				Status:        "ready",
				ShutdownReady: true,
				RunsRequested: 0,
				RunsStopped:   0,
			})
		case orquestaserver.ServerStatusEndpointV0:
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownReady:           false,
				ShutdownRunsRequested:   0,
				ShutdownRunsStopped:     0,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-cleanup-retry"},
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
		t.Fatalf("shutdownCalls=%d, esperaba rePOST para cleanup backend", shutdownCalls)
	}
}

func TestRequestServerShutdownV0ReintentaCleanupBackendStillRunningHTTP409HastaReady(t *testing.T) {
	shutdownCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			if shutdownCalls == 1 {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
					Estado:          "ok",
					Status:          "backend_still_running",
					ShutdownReady:   false,
					RunsRequested:   0,
					RunsStopped:     0,
					ActiveWorkCount: 1,
					ActiveWorks: []serverShutdownClientActiveWorkV0{{
						Kind:    "goal_backend",
						WorkRef: "goal-ref-cleanup-http-409",
						Status:  "backend_still_running",
					}},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:        "ok",
				Status:        "ready",
				ShutdownReady: true,
				RunsRequested: 0,
				RunsStopped:   0,
			})
		case orquestaserver.ServerStatusEndpointV0:
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownReady:           false,
				ShutdownRunsRequested:   0,
				ShutdownRunsStopped:     0,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs: []string{
					"shutdown-active-work-goal-backend-goal-ref-cleanup-http-409",
				},
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
		t.Fatalf("shutdownCalls=%d, esperaba rePOST tras HTTP 409 backend_still_running", shutdownCalls)
	}
}

func TestRequestServerShutdownV0NoReintentaHTTP409ActiveGoalsPresent(t *testing.T) {
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:         "ok",
				Status:         "active_goals_present",
				ShutdownReady:  false,
				RunsRequested:  1,
				RunsStopped:    0,
				AgentsInFlight: 1,
			})
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                 "running",
				ShutdownInProgress:     true,
				ShutdownStatus:         "active_goals_present",
				ShutdownRunsRequested:  1,
				ShutdownRunsStopped:    0,
				ShutdownAgentsInFlight: 1,
			})
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{Forced: true})

	if err == nil || !strings.Contains(err.Error(), "shutdown_not_ready status=active_goals_present") {
		t.Fatalf("error active_goals_present esperado: %v", err)
	}
	if shutdownCalls != 1 {
		t.Fatalf("shutdownCalls=%d, no debe reintentar active_goals_present", shutdownCalls)
	}
	if statusCalls != 0 {
		t.Fatalf("statusCalls=%d, active_goals_present no debe entrar en espera", statusCalls)
	}
}

func TestWaitServerShutdownReadyV0CortaTrasRepostNoRecuperableV0(t *testing.T) {
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:         "ok",
				Status:         "active_goals_present",
				ShutdownReady:  false,
				RunsRequested:  1,
				RunsStopped:    0,
				AgentsInFlight: 1,
			})
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			http.Error(w, "status unavailable", http.StatusServiceUnavailable)
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	started := time.Now()
	err := waitServerShutdownReadyV0(
		strings.TrimPrefix(server.URL, "http://"),
		serverShutdownClientOptionsV0{Forced: true},
		serverShutdownClientResultV0{
			Estado:          "ok",
			Status:          "backend_still_running",
			ShutdownReady:   false,
			ActiveWorkCount: 1,
			ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-repost"},
		},
		serverShutdownClientRequestIdentityV0{
			requestID:      "request-test-repost-nonrecoverable",
			correlationID:  "corr-test-repost-nonrecoverable",
			idempotencyKey: "idem-test-repost-nonrecoverable",
			reason:         "test repost nonrecoverable",
		},
		time.Minute,
		time.Millisecond,
	)

	if err == nil || !strings.Contains(err.Error(), "shutdown_not_ready status=active_goals_present") {
		t.Fatalf("error active_goals_present esperado: %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatalf("estado no recuperable del rePOST no debe esperar hasta timeout")
	}
	if shutdownCalls != 1 {
		t.Fatalf("shutdownCalls=%d, debe cortar tras primer rePOST no recuperable", shutdownCalls)
	}
	if statusCalls != 0 {
		t.Fatalf("statusCalls=%d, no debe consultar status tras cuerpo shutdown no recuperable", statusCalls)
	}
}

func TestWaitServerShutdownReadyV0RepostColgadoRespetaDeadlineYDevuelveStatusAccionableV0(t *testing.T) {
	withShutdownClientRequestTimeoutForTestV0(t, 20*time.Millisecond)
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			time.Sleep(60 * time.Millisecond)
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownReady:           false,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-repost-hang"},
			})
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	started := time.Now()
	err := waitServerShutdownReadyV0(
		strings.TrimPrefix(server.URL, "http://"),
		serverShutdownClientOptionsV0{Forced: true},
		serverShutdownClientResultV0{
			Estado:          "ok",
			Status:          "backend_still_running",
			ShutdownReady:   false,
			ActiveWorkCount: 1,
			ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-repost-hang"},
		},
		serverShutdownClientRequestIdentityV0{
			requestID:      "request-test-repost-hang",
			correlationID:  "corr-test-repost-hang",
			idempotencyKey: "idem-test-repost-hang",
			reason:         "test repost hang",
		},
		70*time.Millisecond,
		5*time.Millisecond,
	)

	if time.Since(started) > time.Second {
		t.Fatalf("rePOST colgado no debe esperar timeout global")
	}
	if err == nil ||
		!strings.Contains(err.Error(), "shutdown_not_ready status=backend_still_running") ||
		!strings.Contains(err.Error(), "active_work=1") ||
		!strings.Contains(err.Error(), "shutdown-active-work-goal-backend-goal-ref-repost-hang") {
		t.Fatalf("err=%v", err)
	}
	if shutdownCalls == 0 || statusCalls == 0 {
		t.Fatalf("calls shutdown=%d status=%d", shutdownCalls, statusCalls)
	}
}

func TestRequestServerShutdownV0NoReintentaBackendStillRunningConRunsPendientes(t *testing.T) {
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:          "ok",
				Status:          "backend_still_running",
				ShutdownReady:   false,
				RunsRequested:   1,
				RunsStopped:     0,
				ActiveWorkCount: 1,
				ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-run-pending"},
			})
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownRunsRequested:   1,
				ShutdownRunsStopped:     0,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-run-pending"},
			})
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"), serverShutdownClientOptionsV0{Forced: true})

	if err == nil || !strings.Contains(err.Error(), "shutdown_not_ready status=backend_still_running") {
		t.Fatalf("error backend_still_running con runs pendientes esperado: %v", err)
	}
	if shutdownCalls != 1 {
		t.Fatalf("shutdownCalls=%d, no debe reintentar backend_still_running con runs pendientes", shutdownCalls)
	}
	if statusCalls != 0 {
		t.Fatalf("statusCalls=%d, backend con runs pendientes no debe entrar en espera de cleanup", statusCalls)
	}
}

func TestRequestServerShutdownV0CortaEsperaSiStatusBackendTieneRunsPendientes(t *testing.T) {
	shutdownCalls := 0
	statusCalls := 0
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v0/server/shutdown":
			shutdownCalls++
			_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
				Estado:          "ok",
				Status:          "backend_still_running",
				ShutdownReady:   false,
				RunsRequested:   0,
				RunsStopped:     0,
				ActiveWorkCount: 1,
				ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-status-pending"},
			})
		case orquestaserver.ServerStatusEndpointV0:
			statusCalls++
			_ = json.NewEncoder(w).Encode(orquestaserver.ServerPublicStatusV0{
				Status:                  "running",
				ShutdownInProgress:      true,
				ShutdownStatus:          "backend_still_running",
				ShutdownRunsRequested:   1,
				ShutdownRunsStopped:     0,
				ShutdownActiveWorkCount: 1,
				ShutdownActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-status-pending"},
			})
		default:
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := waitServerShutdownReadyV0(
		strings.TrimPrefix(server.URL, "http://"),
		serverShutdownClientOptionsV0{},
		serverShutdownClientResultV0{
			Estado:          "ok",
			Status:          "backend_still_running",
			ShutdownReady:   false,
			ActiveWorkCount: 1,
			ActiveWorkRefs:  []string{"shutdown-active-work-goal-backend-goal-ref-status-pending"},
		},
		serverShutdownClientRequestIdentityV0{
			requestID:      "request-test-status-pending",
			correlationID:  "corr-test-status-pending",
			idempotencyKey: "idem-test-status-pending",
			reason:         "test status pending",
		},
		time.Minute,
		time.Millisecond,
	)

	if err == nil || !strings.Contains(err.Error(), "shutdown_not_ready status=backend_still_running") {
		t.Fatalf("error status backend_still_running con runs pendientes esperado: %v", err)
	}
	if shutdownCalls != 1 {
		t.Fatalf("shutdownCalls=%d, no debe reintentar tras status no recuperable", shutdownCalls)
	}
	if statusCalls != 1 {
		t.Fatalf("statusCalls=%d, debe cortar tras primer status no recuperable", statusCalls)
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
	status.ShutdownStatus = "backend_still_running"
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, true) {
		t.Fatalf("force no debe saltar backend vivo declarado en status aunque no haya contadores")
	}
	status.ShutdownStatus = "active_goals_present"
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout no debe saltar goals activos declarados en status aunque no haya contadores")
	}
	status.ShutdownStatus = "waiting_drain"
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, true) {
		t.Fatalf("force no debe saltar espera de drenaje declarada en status aunque no haya contadores")
	}
	status.ShutdownStatus = "waiting_checkpoint"
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_timeout"), status, false) {
		t.Fatalf("timeout no debe saltar espera de checkpoint declarada en status aunque no haya contadores")
	}
	status.ShutdownStatus = "stop_pending"
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_request_failed"), status, true) {
		t.Fatalf("force no debe saltar stop_pending declarado en status aunque no haya contadores")
	}
	status.ShutdownStatus = ""
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
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=waiting_drain runs=0/0 agents_in_flight=0 checkpoints=0 checkpoint_agents=0 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar waiting_drain conservado solo en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=waiting_checkpoint runs=0/0 agents_in_flight=0 checkpoints=0 checkpoint_agents=0 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar waiting_checkpoint conservado solo en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=stop_pending runs=0/0 agents_in_flight=0 checkpoints=0 checkpoint_agents=0 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar stop_pending conservado solo en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=ready runs=1/1 agents_in_flight=0 checkpoints=0 checkpoint_agents=0 active_work=1 active_work_refs=shutdown-active-work-goal-backend-goal-ref-force"), status, true) {
		t.Fatalf("force no debe saltar active_work conservado en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=waiting_checkpoint runs=1/1 agents_in_flight=0 checkpoints=0 checkpoint_agents=1 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar checkpoint_agents conservado solo en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=waiting_drain runs=1/1 agents_in_flight=1 checkpoints=0 checkpoint_agents=0 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar agents_in_flight conservado solo en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=waiting_checkpoint runs=1/1 agents_in_flight=0 checkpoints=1 checkpoint_agents=0 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar checkpoints conservados solo en error shutdown_not_ready")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0("shutdown_not_ready status=waiting_drain runs=0/1 agents_in_flight=0 checkpoints=0 checkpoint_agents=0 active_work=0"), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar runs pendientes conservados solo en error shutdown_not_ready")
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

func TestShutdownClientReadyV0NoSaltaTrabajoPendienteAunqueReadyV0(t *testing.T) {
	result := serverShutdownClientResultV0{
		ShutdownReady:           true,
		RunsRequested:           1,
		RunsStopped:             1,
		AgentsInFlight:          0,
		CheckpointsPending:      0,
		CheckpointAgentsPending: 1,
	}
	if shutdownClientResultReadyForSignalV0(result) {
		t.Fatalf("resultado shutdown_ready no debe permitir signal con checkpoint_agents pendientes")
	}

	result.CheckpointAgentsPending = 0
	result.CheckpointsPending = 1
	if shutdownClientResultReadyForSignalV0(result) {
		t.Fatalf("resultado shutdown_ready no debe permitir signal con checkpoints pendientes")
	}

	result.CheckpointsPending = 0
	result.AgentsInFlight = 1
	if shutdownClientResultReadyForSignalV0(result) {
		t.Fatalf("resultado shutdown_ready no debe permitir signal con agentes en vuelo")
	}

	result.AgentsInFlight = 0
	result.AsyncWorkActive = 1
	if shutdownClientResultReadyForSignalV0(result) {
		t.Fatalf("resultado shutdown_ready no debe permitir signal con async work pendiente")
	}
	if shutdownRequestErrorAllowsSignalV0(assertShutdownClientErrorV0(shutdownClientNotReadyErrorV0(result).Error()), orquestaserver.ServerPublicStatusV0{}, true) {
		t.Fatalf("force no debe saltar async work conservado solo en error shutdown_not_ready")
	}

	status := orquestaserver.ServerPublicStatusV0{
		ShutdownInProgress:              true,
		ShutdownReady:                   true,
		ShutdownRunsRequested:           1,
		ShutdownRunsStopped:             1,
		ShutdownCheckpointAgentsPending: 1,
	}
	if shutdownPublicStatusReadyForSignalV0(status) {
		t.Fatalf("status shutdown_ready no debe permitir signal con checkpoint_agents pendientes")
	}

	status.ShutdownCheckpointAgentsPending = 0
	status.ShutdownAsyncWorkActive = 1
	if shutdownPublicStatusReadyForSignalV0(status) {
		t.Fatalf("status shutdown_ready no debe permitir signal con async work pendiente")
	}
	statusResult := serverShutdownClientResultFromStatusV0(status)
	if statusResult.AsyncWorkActive != 1 || shutdownClientResultReadyForSignalV0(statusResult) {
		t.Fatalf("status result no conserva/bloquea async work: %+v", statusResult)
	}
}

func TestShutdownClientReadyV0NoSaltaStatusConflictoSinContadoresV0(t *testing.T) {
	for _, status := range []string{"backend_still_running", "active_goals_present", "waiting_drain", "waiting_checkpoint", "stop_pending"} {
		result := serverShutdownClientResultV0{
			Status:        status,
			RunsRequested: 0,
			RunsStopped:   0,
		}
		if shutdownClientResultReadyForSignalV0(result) {
			t.Fatalf("result status=%s no debe permitir signal por runs 0/0", status)
		}

		publicStatus := orquestaserver.ServerPublicStatusV0{
			ShutdownInProgress:    true,
			ShutdownStatus:        status,
			ShutdownRunsRequested: 0,
			ShutdownRunsStopped:   0,
		}
		if shutdownPublicStatusReadyForSignalV0(publicStatus) {
			t.Fatalf("public status=%s no debe permitir signal por runs 0/0", status)
		}
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

func TestShutdownClientNotReadyErrorV0IncluyeRecommendedActionSegura(t *testing.T) {
	err := shutdownClientNotReadyErrorV0(serverShutdownClientResultV0{
		Status:            "backend_still_running",
		RecommendedAction: "wait_or_reconcile_goal_backend_cleanup",
		ActiveWorkCount:   1,
	})

	if err == nil ||
		!strings.Contains(err.Error(), "recommended_action=wait_or_reconcile_goal_backend_cleanup") {
		t.Fatalf("err=%v", err)
	}

	err = shutdownClientNotReadyErrorV0(serverShutdownClientResultV0{
		Status:            "backend_still_running",
		RecommendedAction: "/tmp/private/action",
		ActiveWorkCount:   1,
	})
	if err == nil || strings.Contains(err.Error(), "recommended_action=") || strings.Contains(err.Error(), "/tmp") {
		t.Fatalf("err con accion no segura=%v", err)
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

func withShutdownClientRequestTimeoutForTestV0(t *testing.T, timeout time.Duration) {
	t.Helper()
	previous := shutdownClientRequestTimeoutV0
	shutdownClientRequestTimeoutV0 = timeout
	t.Cleanup(func() {
		shutdownClientRequestTimeoutV0 = previous
	})
}
