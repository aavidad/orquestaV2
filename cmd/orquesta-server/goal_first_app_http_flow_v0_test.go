package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestServerAppHTTPGoalFirstLanzaObservaYCierraV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")

	backend := &goalFirstHTTPBackendForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	if !config.GoalObserverEnabled ||
		config.GoalObserverEnabledConfigured ||
		stack.Stores.AppGoalStateStore == nil ||
		stack.Ports.GoalStateStore == nil ||
		stack.Ports.GoalFirstRunMarkerStore == nil {
		t.Fatalf("wiring goal-first por defecto incompleto: config=%+v stores=%+v ports=%+v", config, stack.Stores, stack.Ports)
	}
	if _, ok := stack.Stores.AppGoalStateStore.(orquestagoal.GoalWorkRunMarkerListPortV0); !ok {
		t.Fatalf("AppGoalStateStore debe listar markers goal-first activos")
	}
	if stack.AllowLegacyAutoprogrammingRun ||
		stack.AllowLegacyExternalWorkRun ||
		stack.MCPTransportBindings.AllowLegacyAutoprogrammingSupervisorActions {
		t.Fatalf("legacy loop no debe quedar habilitado por defecto: stack=%+v bindings=%+v", stack, stack.MCPTransportBindings)
	}
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}

	started := postGoalFirstStartForTestV0(t, handler)
	if started.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 ||
		started.RunRef == "" ||
		started.GoalRef == "" ||
		started.ExternalGoalRef != "thread-ref-http-goal-first-001" ||
		started.GoalStatus != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("started=%+v", started)
	}
	if backend.packet.GoalRef != started.GoalRef || backend.packet.Objective == "" || len(backend.packet.ArtifactContracts) == 0 {
		t.Fatalf("packet no capturado: %+v started=%+v", backend.packet, started)
	}

	observed := postGoalFirstObserveForTestV0(t, handler, started.RunRef)
	if observed.Estado != orquestamcp.MCPObserveAppDirectorGoalEstadoOKV0 ||
		observed.GoalRef != started.GoalRef ||
		observed.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		observed.RunStatus != "cerrada" ||
		observed.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		!observed.ClosureAccepted {
		t.Fatalf("observed=%+v", observed)
	}
}

func TestServerAutoprogrammingHTTPGoalFirstPreparaSupervisaObservaYCierraV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")

	backend := &goalFirstHTTPBackendForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}

	prepared := postAutoprogrammingGoalFirstPrepareForTestV0(t, handler)
	if prepared.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0 ||
		!prepared.Accepted ||
		prepared.RunRef != "run-http-autoprogramming-goal-first-001" ||
		prepared.Goal == nil ||
		prepared.Goal.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		prepared.Goal.ExternalGoalRef != "thread-ref-http-goal-first-001" ||
		len(prepared.GoalSpecs) != 1 ||
		len(prepared.WorkflowTaskRefs) != 0 ||
		len(prepared.WaitAgentRefs) != 0 ||
		prepared.Continue != nil {
		t.Fatalf("prepared=%+v", prepared)
	}
	if backend.packet.RequestRef != prepared.RunRef ||
		backend.packet.GoalRef != prepared.Goal.GoalRef ||
		len(backend.packet.RequiredTests) == 0 {
		t.Fatalf("packet=%+v prepared=%+v", backend.packet, prepared)
	}

	status := postAutoprogrammingGoalFirstStatusForTestV0(t, handler, prepared.RunRef)
	if status.Estado != orquestamcp.MCPAutoprogrammingStatusEstadoOKV0 ||
		status.RunRef != prepared.RunRef ||
		!goalFirstHTTPDiagnosticsContainCodeForTestV0(status.Diagnostics, "autoprogramming_goal_first_observe_required") {
		t.Fatalf("status=%+v", status)
	}

	supervisor := postAutoprogrammingGoalFirstSuperviseForTestV0(t, handler, prepared.RunRef)
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != prepared.RunRef ||
		supervisor.StopReason != "goal_first_observe_required" ||
		supervisor.Last.Status != "running_live" ||
		!goalFirstHTTPStringInSetForTestV0(supervisor.NextActions, "observe_goal") ||
		!goalFirstHTTPDiagnosticsContainCodeForTestV0(supervisor.Diagnostics, "run_supervisor_goal_first_not_legacy") {
		t.Fatalf("supervisor=%+v", supervisor)
	}

	observed := postAutoprogrammingGoalFirstObserveForTestV0(t, handler, prepared.RunRef)
	if observed.Estado != orquestamcp.MCPAutoprogrammingObserveGoalEstadoOKV0 ||
		observed.GoalRef != prepared.Goal.GoalRef ||
		observed.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		observed.RunStatus != "cerrada" ||
		observed.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		!observed.ClosureAccepted {
		t.Fatalf("observed=%+v", observed)
	}
}

func TestServerAppHTTPGoalFirstReanudaTrasRestartSinLegacyV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")

	startBackend := &goalFirstHTTPBackendForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	startStack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  startBackend,
		Observer: startBackend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 start: %v", err)
	}
	startHandler, err := buildServerAppHandlerV0(startStack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0 start: %v", err)
	}
	started := postGoalFirstStartForTestV0(t, startHandler)
	if startBackend.startCalls != 1 || started.RunRef == "" || started.GoalRef == "" {
		t.Fatalf("start incompleto backend=%+v started=%+v", startBackend, started)
	}
	if _, err := startStack.Stores.AppGoalStateStore.LoadGoalWorkStateV0(context.Background(), started.RunRef); err != nil {
		t.Fatalf("GoalWorkStateV0 no persistido: %v", err)
	}
	if _, err := startStack.Ports.GoalFirstRunMarkerStore.LoadGoalWorkRunMarkerV0(context.Background(), started.RunRef); err != nil {
		t.Fatalf("GoalWorkRunMarkerV0 no persistido: %v", err)
	}

	restartBackend := &goalFirstHTTPBackendForTestV0{packet: startBackend.packet}
	restartedStack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  restartBackend,
		Observer: restartBackend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 restart: %v", err)
	}
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-http-goal-first-restart-continue-001",
			RequestedBy:   "orquesta-server-goal-first-restart-test",
		},
		restartedStack.Ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 restart: %v", err)
	}
	if continued.Status != orquestaappdirectorservice.StartAppDirectorStatusPendingV0 ||
		continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		len(continued.StartedAgents) != 0 ||
		!goalFirstHTTPClosureIssuesContainCodeForTestV0(continued.OperationalClosureIssues, "app_director_goal_first_observe_required") {
		t.Fatalf("continued=%+v", continued)
	}
	restartHandler, err := buildServerAppHandlerV0(restartedStack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0 restart: %v", err)
	}
	supervisor := postRunSupervisorGoalFirstForTestV0(t, restartHandler, started.RunRef)
	if supervisor.StopReason != "goal_first_observe_required" ||
		!goalFirstHTTPStringInSetForTestV0(supervisor.NextActions, "observe_goal") ||
		!goalFirstHTTPDiagnosticsContainCodeForTestV0(supervisor.Diagnostics, "run_supervisor_goal_first_not_legacy") {
		t.Fatalf("supervisor=%+v", supervisor)
	}
	observed := postGoalFirstObserveForTestV0(t, restartHandler, started.RunRef)
	if observed.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		observed.RunStatus != "cerrada" ||
		!observed.ClosureAccepted ||
		restartBackend.startCalls != 0 ||
		restartBackend.observeCalls != 1 {
		t.Fatalf("observed=%+v restartBackend=%+v", observed, restartBackend)
	}
	persistedRun, err := restartedStack.Stores.RunStore.LoadRunV0(context.Background(), started.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 restart: %v", err)
	}
	if persistedRun.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(persistedRun.Tasks) != 0 ||
		len(persistedRun.Agents) != 0 ||
		len(persistedRun.StartedAgents) != 0 {
		t.Fatalf("loop legacy activado tras restart: run=%+v", persistedRun)
	}
}

func TestServerAppHTTPGoalFirstRestartMarkerSinStateNoDrenaLegacyV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")

	startBackend := &goalFirstHTTPBackendForTestV0{}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	startStack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  startBackend,
		Observer: startBackend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 start: %v", err)
	}
	startHandler, err := buildServerAppHandlerV0(startStack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0 start: %v", err)
	}
	started := postGoalFirstStartForTestV0(t, startHandler)
	if _, err := startStack.Ports.GoalFirstRunMarkerStore.LoadGoalWorkRunMarkerV0(context.Background(), started.RunRef); err != nil {
		t.Fatalf("GoalWorkRunMarkerV0 no persistido: %v", err)
	}
	removeGoalFirstHTTPStateFilesForTestV0(t, stateDir)

	restartBackend := &goalFirstHTTPBackendForTestV0{packet: startBackend.packet}
	restartedStack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  restartBackend,
		Observer: restartBackend,
	})
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0 restart: %v", err)
	}
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:        started.RunRef,
			CorrelationID: "corr-http-goal-first-marker-missing-state-001",
			RequestedBy:   "orquesta-server-goal-first-restart-test",
		},
		restartedStack.Ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 marker sin state: %v", err)
	}
	if continued.Status != orquestaappdirectorservice.StartAppDirectorStatusPendingV0 ||
		continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		len(continued.StartedAgents) != 0 ||
		!goalFirstHTTPClosureIssuesContainCodeForTestV0(continued.OperationalClosureIssues, "app_director_goal_first_state_missing") {
		t.Fatalf("continued=%+v", continued)
	}
	restartHandler, err := buildServerAppHandlerV0(restartedStack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0 restart: %v", err)
	}
	supervisor := postRunSupervisorGoalFirstForTestV0(t, restartHandler, started.RunRef)
	if supervisor.StopReason != "goal_first_state_missing" ||
		!goalFirstHTTPStringInSetForTestV0(supervisor.NextActions, "repair_goal_state_from_launcher_receipt_or_mark_blocked") ||
		!goalFirstHTTPDiagnosticsContainCodeForTestV0(supervisor.Diagnostics, "run_supervisor_goal_first_state_missing") ||
		restartBackend.startCalls != 0 ||
		restartBackend.observeCalls != 0 {
		t.Fatalf("supervisor=%+v restartBackend=%+v", supervisor, restartBackend)
	}
	persistedRun, err := restartedStack.Stores.RunStore.LoadRunV0(context.Background(), started.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 restart: %v", err)
	}
	if len(persistedRun.Tasks) != 0 ||
		len(persistedRun.Agents) != 0 ||
		len(persistedRun.StartedAgents) != 0 {
		t.Fatalf("loop legacy activado con marker sin state: run=%+v", persistedRun)
	}
}

func postGoalFirstStartForTestV0(
	t *testing.T,
	handler http.Handler,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	body := []byte(`{
		"request_id":"req-http-goal-first-001",
		"correlation_id":"corr-http-goal-first-001",
		"app_spec_request":{
			"schema_version":"app_spec_request.v0",
			"request_id":"request-ref-http-goal-first-001",
			"source":"orquesta-web",
			"locale":"es-ES",
			"nombre":"Agenda",
			"objetivo":"Gestionar contactos y citas desde una API y una web.",
			"tipo_app":"mixed",
			"preferencias_tecnicas":{"lenguaje":"go","arquitectura":"hexagonal"},
			"calidad":{"pruebas":"media","accesibilidad":"basica","observabilidad":true}
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPArrancarDirectorAppHTTPPathV0, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-goal-first-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	return result
}

func postGoalFirstObserveForTestV0(
	t *testing.T,
	handler http.Handler,
	runRef string,
) orquestamcp.MCPObserveAppDirectorGoalToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
		RequestID:   "req-http-goal-first-observe-001",
		RunRef:      runRef,
		RequestedBy: "orquesta-server-test",
	})
	if err != nil {
		t.Fatalf("marshal observe: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPObserveAppDirectorGoalHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-goal-first-observe-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("observe status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode observe: %v", err)
	}
	return result
}

func postAutoprogrammingGoalFirstPrepareForTestV0(
	t *testing.T,
	handler http.Handler,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:     "request-http-autoprogramming-goal-first-001",
		CorrelationID: "corr-http-autoprogramming-goal-first-001",
		OccurredAt:    "2026-06-26T10:00:00Z",
		RequestedBy:   "orquesta-server-test",
		AutoprogrammingRequest: orquestaautoprogramming.AutoprogrammingRequestV0{
			RequestRef:       "run-http-autoprogramming-goal-first-001",
			ProjectRef:       "project-ref-http-autoprogramming-goal-first-001",
			WorktreeRef:      "worktree-ref-http-autoprogramming-goal-first-001",
			WorktreeIsolated: true,
			BranchRef:        "branch-ref-http-autoprogramming-goal-first-001",
			Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
				TaskRef:            "source-task-ref-http-autoprogramming-goal-first-001",
				Area:               "cmd-orquesta-server",
				Title:              "Cubrir autoprogramacion goal-first por HTTP",
				Objective:          "Verificar que prepare-run lanza goal, supervise no usa loop legacy y observe cierra por evidencias.",
				ContextRefs:        goalFirstHTTPAutoprogrammingCapabilityRefsForTestV0(),
				AcceptanceCriteria: []string{"prepare-run devuelve goal running", "supervise redirige a observe_goal", "observe cierra aceptado"},
				WriteSet:           []string{"cmd/orquesta-server/goal_first_app_http_flow_v0_test.go"},
				RequiredTests:      []string{"go test -count=1 ./cmd/orquesta-server -run TestServerAutoprogrammingHTTPGoalFirstPreparaSupervisaObservaYCierraV0"},
			}},
			WriteSet: []string{
				"cmd/orquesta-server/goal_first_app_http_flow_v0_test.go",
			},
			RequiredTests: []string{
				"go test -count=1 ./cmd/orquesta-server -run TestServerAutoprogrammingHTTPGoalFirstPreparaSupervisaObservaYCierraV0",
			},
		},
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 3,
		MaxCommands:          5,
		MaxOutboxPerCycle:    5,
		PriorityScore:        90,
	})
	if err != nil {
		t.Fatalf("marshal prepare: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAutoprogrammingPrepareRunHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-autoprogramming-goal-first-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("prepare status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode prepare: %v", err)
	}
	return result
}

func postAutoprogrammingGoalFirstStatusForTestV0(
	t *testing.T,
	handler http.Handler,
	runRef string,
) orquestamcp.MCPAutoprogrammingStatusToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPAutoprogrammingStatusToolInputV0{
		RequestID:     "request-http-autoprogramming-goal-first-status-001",
		CorrelationID: "corr-http-autoprogramming-goal-first-001",
		RunRef:        runRef,
	})
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAutoprogrammingStatusHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-autoprogramming-goal-first-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	return result
}

func postAutoprogrammingGoalFirstSuperviseForTestV0(
	t *testing.T,
	handler http.Handler,
	runRef string,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-http-autoprogramming-goal-first-supervise-001",
		CorrelationID: "corr-http-autoprogramming-goal-first-001",
		RunRef:        runRef,
		MaxTicks:      1,
	})
	if err != nil {
		t.Fatalf("marshal supervise: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAutoprogrammingSuperviseHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-autoprogramming-goal-first-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("supervise code=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode supervise: %v", err)
	}
	return result
}

func postAutoprogrammingGoalFirstObserveForTestV0(
	t *testing.T,
	handler http.Handler,
	runRef string,
) orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID:     "request-http-autoprogramming-goal-first-observe-001",
		CorrelationID: "corr-http-autoprogramming-goal-first-001",
		RunRef:        runRef,
		RequestedBy:   "orquesta-server-test",
	})
	if err != nil {
		t.Fatalf("marshal observe autoprogramming: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPAutoprogrammingObserveGoalHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-autoprogramming-goal-first-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("observe autoprogramming status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingObserveGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode observe autoprogramming: %v", err)
	}
	return result
}

func postRunSupervisorGoalFirstForTestV0(
	t *testing.T,
	handler http.Handler,
	runRef string,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	t.Helper()
	payload, err := json.Marshal(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-http-goal-first-run-supervisor-001",
		CorrelationID: "corr-http-goal-first-run-supervisor-001",
		RunRef:        runRef,
		MaxTicks:      1,
	})
	if err != nil {
		t.Fatalf("marshal run supervise: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPRunSupervisorHTTPPathV0, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-goal-first-run-supervisor-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("run supervise code=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode run supervise: %v", err)
	}
	return result
}

type goalFirstHTTPBackendForTestV0 struct {
	packet       orquestaruntimecodexgoal.CodexGoalStartPacketV0
	startCalls   int
	observeCalls int
}

func (backend *goalFirstHTTPBackendForTestV0) StartCodexGoalV0(
	_ context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	backend.startCalls++
	backend.packet = packet
	return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: "thread-ref-http-goal-first-001",
		EvidenceRefs:    []string{"evidence-ref-http-goal-first-launch"},
	}, nil
}

func (backend *goalFirstHTTPBackendForTestV0) ObserveCodexGoalV0(
	_ context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	backend.observeCalls++
	return orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Summary:         "goal-first fake completo",
		ArtifactRefs:    goalFirstHTTPRequiredArtifactRefsForTestV0(backend.packet),
		RequiredTestResults: goalFirstHTTPRequiredTestResultsForTestV0(
			backend.packet,
			"evidence-ref-http-goal-first-required-test",
		),
		EvidenceRefs: append(
			[]string{"evidence-ref-http-goal-first-observed"},
			backend.packet.ClosurePolicy.RequiredEvidenceRefs...,
		),
	}, nil
}

func goalFirstHTTPRequiredArtifactRefsForTestV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) []string {
	refs := make([]string, 0, len(packet.ArtifactContracts))
	for _, contract := range packet.ArtifactContracts {
		if contract.Required && contract.ArtifactRef != "" {
			refs = append(refs, contract.ArtifactRef)
		}
	}
	return refs
}

func goalFirstHTTPRequiredTestResultsForTestV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
	evidenceRefs ...string,
) []orquestagoal.GoalRequiredTestResultV0 {
	results := make([]orquestagoal.GoalRequiredTestResultV0, 0, len(packet.RequiredTests))
	for _, test := range packet.RequiredTests {
		results = append(results, orquestagoal.GoalRequiredTestResultV0{
			TestRef:      test.TestRef,
			Status:       "passed",
			EvidenceRefs: append([]string(nil), evidenceRefs...),
		})
	}
	return results
}

func goalFirstHTTPAutoprogrammingCapabilityRefsForTestV0() []string {
	return []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
}

func goalFirstHTTPStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func goalFirstHTTPDiagnosticsContainCodeForTestV0(
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
	want string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == want {
			return true
		}
	}
	return false
}

func goalFirstHTTPClosureIssuesContainCodeForTestV0(
	issues []orquestacionnucleoapp.ErrorV0,
	want string,
) bool {
	for _, issue := range issues {
		if issue.Code == want {
			return true
		}
	}
	return false
}

func removeGoalFirstHTTPStateFilesForTestV0(t *testing.T, stateDir string) {
	t.Helper()
	dir := filepath.Join(stateDir, "orchestration-state", "app_director_goal_states")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir goal states: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			t.Fatalf("Remove goal state %s: %v", entry.Name(), err)
		}
	}
}
