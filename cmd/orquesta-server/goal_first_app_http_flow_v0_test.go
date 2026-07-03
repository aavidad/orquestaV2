package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerAppHTTPGoalFirstLanzaObservaYCierraV0(t *testing.T) {
	disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t)
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

func TestRuntimeGoalFirstResidenteCierraAppSinObserveManualV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t)
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
	config.Addr = "127.0.0.1:0"
	config.AuditDisabled = true
	config.TickInterval = time.Hour
	config.GoalObserverInterval = time.Hour
	config.ResidentDirectorEnabled = false
	config.ShutdownGracePeriod = 500 * time.Millisecond
	supervisorWakeup := &serverSupervisorWakeupRelayV0{}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	}, supervisorWakeup)
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	appHandler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stack:          &stack,
		projectWorkDir: config.IdleSelfImprovementProjectWorkDir,
		runtimeWorkDir: config.RuntimeWorkDir,
		stateDir:       config.StateDir,
	}
	runtime, err := orquestaserver.NewRuntimeV0(config, orquestaserver.RuntimeDepsV0{
		AppHandler:     appHandler,
		Supervisor:     supervisor,
		GoalStateStore: stack.Stores.AppGoalStateStore,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	supervisorWakeup.bindRuntimeV0(runtime)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()

	started := postGoalFirstStartURLForTestV0(t, "http://"+waitRuntimeAddrForGoalFirstHTTPTestV0(t, runtime))
	serverState := waitGoalFirstObserverTerminalForTestV0(t, runtime)
	state := waitGoalFirstClosedForTestV0(t, stack.Stores.AppGoalStateStore, started.RunRef)
	cancel()
	if err := waitRuntimeDoneForGoalFirstHTTPTestV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusCompleteV0 ||
		state.LastClosure == nil ||
		!state.LastClosure.Accepted ||
		backend.observeCalls == 0 {
		t.Fatalf("goal no cerro autonomamente: state=%+v backend=%+v", state, backend)
	}
	if serverState.GoalObserverTicks == 0 || serverState.GoalObserverTerminal == 0 {
		t.Fatalf(
			"goal observer residente no dejo evidencia terminal: status=%s ticks=%d observed=%d terminal=%d issues=%d message=%+v",
			serverState.GoalObserverStatus,
			serverState.GoalObserverTicks,
			serverState.GoalObserverObserved,
			serverState.GoalObserverTerminal,
			serverState.GoalObserverIssues,
			serverState.GoalObserverOperationalMessage,
		)
	}
}

func TestRuntimeExternalWorkGoalFirstResidenteCierraSinObserveManualV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t)
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "runtime")
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envOPESProjectWorkDirV0, projectDir)
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv("OPES_BASE_URL", "")
	t.Setenv(envDomainWorkFileEnabledV0, "1")
	t.Setenv(envDomainWorkFileDirV0, filepath.Join(stateDir, "domain-work-jobs"))

	backend := &goalFirstHTTPBackendForTestV0{projectDir: projectDir}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	config.Addr = "127.0.0.1:0"
	config.AuditDisabled = true
	config.TickInterval = time.Hour
	config.GoalObserverInterval = time.Hour
	config.ResidentDirectorEnabled = false
	config.ShutdownGracePeriod = 500 * time.Millisecond
	supervisorWakeup := &serverSupervisorWakeupRelayV0{}
	stack, err := buildStackFromEnvWithGoalBackendV0(config, serverCodexGoalBackendV0{
		Starter:  backend,
		Observer: backend,
	}, supervisorWakeup)
	if err != nil {
		t.Fatalf("buildStackFromEnvWithGoalBackendV0: %v", err)
	}
	if stack.DomainWork == nil || !stack.DomainDelivery.Enabled || stack.DomainDelivery.Ledger == nil {
		t.Fatalf("domain delivery aislado no cableado: domain=%T delivery=%+v", stack.DomainWork, stack.DomainDelivery)
	}
	appHandler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stack:          &stack,
		projectWorkDir: config.IdleSelfImprovementProjectWorkDir,
		runtimeWorkDir: config.RuntimeWorkDir,
		stateDir:       config.StateDir,
	}
	runtime, err := orquestaserver.NewRuntimeV0(config, orquestaserver.RuntimeDepsV0{
		AppHandler:     appHandler,
		Supervisor:     supervisor,
		GoalStateStore: stack.Stores.AppGoalStateStore,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	supervisorWakeup.bindRuntimeV0(runtime)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(ctx) }()

	started := postExternalWorkGoalFirstRunURLForTestV0(t, "http://"+waitRuntimeAddrForGoalFirstHTTPTestV0(t, runtime))
	if started.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 ||
		started.RoutePolicy != orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0 ||
		started.DirectorExecutionMode != orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0 ||
		started.GoalRef == "" ||
		!goalFirstHTTPStringInSetForTestV0(started.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveActiveGoalsV0) ||
		goalFirstHTTPStringInSetForTestV0(started.NextActions, orquestamcp.MCPExternalWorkRunNextActionObserveGoalV0) {
		t.Fatalf("external-work goal-first no arranco en modo residente: %+v", started)
	}
	serverState := waitGoalFirstObserverTerminalForTestV0(t, runtime)
	state := waitGoalFirstClosedForTestV0(t, stack.Stores.AppGoalStateStore, started.RunRef)
	run := waitExternalWorkRunClosedForTestV0(t, stack.Stores.RunStore, started.RunRef)
	cancel()
	if err := waitRuntimeDoneForGoalFirstHTTPTestV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusCompleteV0 ||
		state.LastClosure == nil ||
		!state.LastClosure.Accepted ||
		run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		backend.observeCalls == 0 {
		t.Fatalf("external-work no cerro autonomamente: state=%+v run=%+v backend=%+v", state, run, backend)
	}
	if serverState.GoalObserverTerminal == 0 {
		t.Fatalf("goal observer residente no publico terminal external-work: %+v", serverState)
	}
	if !pathExistsForTestV0(filepath.Join(stateDir, "domain-work-jobs", "domain_work_artifacts_v0.json")) {
		t.Fatalf("domain-work file aislado no recibio artifact submit")
	}
}

func TestServerAutoprogrammingHTTPGoalFirstPreparaSupervisaObservaYCierraV0(t *testing.T) {
	disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t)
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
		len(prepared.GoalSpecs) != 0 ||
		len(prepared.GoalSpecSummaries) != 1 ||
		prepared.GoalSpecSummaries[0].SpecHash == "" ||
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
	disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t)
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
	disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t)
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
		!goalFirstHTTPClosureIssuesContainCodeForTestV0(continued.OperationalClosureIssues, "app_director_goal_first_observe_required") ||
		!goalFirstHTTPStringInSetForTestV0(continued.EvidenceRefs, "evidence-ref-app-director-goal-state-repaired-from-marker-v0") {
		t.Fatalf("continued=%+v", continued)
	}
	repairedState, err := restartedStack.Ports.GoalStateStore.LoadGoalWorkStateV0(context.Background(), started.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 repaired: %v", err)
	}
	if repairedState.GoalRef != started.GoalRef ||
		repairedState.ExternalGoalRef == "" ||
		!goalFirstHTTPStringInSetForTestV0(repairedState.EvidenceRefs, "evidence-ref-app-director-goal-state-repaired-from-marker-v0") {
		t.Fatalf("repairedState=%+v started=%+v", repairedState, started)
	}
	restartHandler, err := buildServerAppHandlerV0(restartedStack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0 restart: %v", err)
	}
	supervisor := postRunSupervisorGoalFirstForTestV0(t, restartHandler, started.RunRef)
	if supervisor.StopReason != "goal_first_observe_required" ||
		!goalFirstHTTPStringInSetForTestV0(supervisor.NextActions, "observe_goal") ||
		!goalFirstHTTPDiagnosticsContainCodeForTestV0(supervisor.Diagnostics, "run_supervisor_goal_first_not_legacy") ||
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

func disableSelfProgrammingOnlyForGoalFirstHTTPTestV0(t *testing.T) {
	t.Helper()
	t.Setenv(envServerSelfProgrammingOnlyV0, "false")
	t.Setenv(envServerSelfProgrammingRootV0, "")
	t.Setenv(envServerGoalObserverEnabledV0, "")
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

func postGoalFirstStartURLForTestV0(
	t *testing.T,
	baseURL string,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	body := []byte(`{
		"request_id":"req-http-goal-first-resident-001",
		"correlation_id":"corr-http-goal-first-resident-001",
		"app_spec_request":{
			"schema_version":"app_spec_request.v0",
			"request_id":"request-ref-http-goal-first-resident-001",
			"source":"orquesta-web",
			"locale":"es-ES",
			"nombre":"Agenda",
			"objetivo":"Gestionar contactos y citas desde una API y una web.",
			"tipo_app":"mixed",
			"preferencias_tecnicas":{"lenguaje":"go","arquitectura":"hexagonal"},
			"calidad":{"pruebas":"media","accesibilidad":"basica","observabilidad":true}
		}
	}`)
	req, err := http.NewRequest(http.MethodPost, baseURL+orquestamcp.MCPArrancarDirectorAppHTTPPathV0, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-goal-first-resident-001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("start http: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status=%d", resp.StatusCode)
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	return result
}

func postExternalWorkGoalFirstRunURLForTestV0(
	t *testing.T,
	baseURL string,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	body := []byte(`{
		"request_id":"req-http-external-work-goal-first-resident-001",
		"correlation_id":"corr-http-external-work-goal-first-resident-001",
		"director_execution_mode":"goal_first",
		"app_change_request":{
			"change_ref":"change-ref-http-external-work-goal-first-resident-001",
			"app_ref":"opes",
			"user_intent":"Resolver trabajo externo aislado de prueba.",
			"allowed_write_set":["external/opes/draft_content_block/job-ref-http-resident-001"],
			"acceptance_criteria":["artefacto domain-work aceptado por conector file aislado"],
			"external_work":{
				"project_ref":"opes",
				"job_ref":"job-ref-http-resident-001",
				"work_kind":"draft_content_block",
				"input_fields":[{"name":"topic_id","value":"topic-ref-http-resident-001"}]
			}
		}
	}`)
	req, err := http.NewRequest(http.MethodPost, baseURL+orquestamcp.MCPExternalWorkRunHTTPPathV0, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-http-external-work-goal-first-resident-001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("external-work run http: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read external-work run body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("external-work run status=%d body=%s", resp.StatusCode, string(data))
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode external-work run: %v", err)
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

func waitRuntimeAddrForGoalFirstHTTPTestV0(
	t *testing.T,
	runtime *orquestaserver.RuntimeV0,
) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := runtime.StateV0()
		if state.Status == "running" && state.Addr != "" {
			return state.Addr
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime no publico addr: %+v", runtime.StateV0())
	return ""
}

func waitGoalFirstClosedForTestV0(
	t *testing.T,
	store orquestagoal.GoalWorkStateStorePortV0,
	runRef string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last orquestagoal.GoalWorkStateV0
	for time.Now().Before(deadline) {
		state, err := store.LoadGoalWorkStateV0(context.Background(), runRef)
		if err == nil {
			last = state
			if state.Status == orquestagoal.GoalStatusCompleteV0 &&
				state.LastClosure != nil &&
				state.LastClosure.Accepted {
				return state
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goal no cerro autonomamente: last=%+v", last)
	return orquestagoal.GoalWorkStateV0{}
}

func waitExternalWorkRunClosedForTestV0(
	t *testing.T,
	store orquestacionnucleoapp.RunStorePortV0,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last orquestacoreworkflow.OrchestrationRunV0
	for time.Now().Before(deadline) {
		run, err := store.LoadRunV0(context.Background(), runRef)
		if err == nil {
			last = run
			if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
				return run
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("external-work run no cerro autonomamente: last=%+v", last)
	return orquestacoreworkflow.OrchestrationRunV0{}
}

func waitGoalFirstObserverTerminalForTestV0(
	t *testing.T,
	runtime *orquestaserver.RuntimeV0,
) orquestaserver.StateV0 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last orquestaserver.StateV0
	for time.Now().Before(deadline) {
		last = runtime.StateV0()
		if last.GoalObserverTerminal > 0 && !last.GoalObserverTickActive {
			return last
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goal observer residente no publico terminal: %+v", last.GoalObserverOperationalMessage)
	return orquestaserver.StateV0{}
}

func waitRuntimeDoneForGoalFirstHTTPTestV0(
	t *testing.T,
	done <-chan error,
) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatalf("runtime no paro")
		return nil
	}
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
	projectDir   string
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
	if backend.packet.ClosurePolicy.RequireDomainReceipt && strings.TrimSpace(backend.projectDir) != "" {
		writeGoalFirstHTTPDomainArtifactsForTestV0(backend.projectDir, backend.packet)
	}
	return orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Summary:         "goal-first fake completo con app Go net/http",
		ArtifactRefs:    goalFirstHTTPRequiredArtifactRefsForTestV0(backend.packet),
		ArtifactPaths:   goalFirstHTTPArtifactPathsForTestV0(backend.packet),
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

func goalFirstHTTPArtifactPathsForTestV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) []string {
	if strings.TrimSpace(packet.WorkKind) != "new_app" {
		return nil
	}
	if len(packet.WriteSet) == 0 {
		return nil
	}
	scopePath := filepath.ToSlash(strings.Trim(filepath.Clean(strings.TrimSpace(packet.WriteSet[0].Path)), "/"))
	if scopePath == "" || scopePath == "." || strings.HasPrefix(scopePath, "../") {
		return nil
	}
	return []string{
		scopePath + "/go.mod",
		scopePath + "/main.go",
		scopePath + "/docs/technical_stack.md",
	}
}

func writeGoalFirstHTTPDomainArtifactsForTestV0(
	projectDir string,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) {
	if len(packet.WriteSet) == 0 {
		return
	}
	scopePath := filepath.ToSlash(strings.Trim(filepath.Clean(strings.TrimSpace(packet.WriteSet[0].Path)), "/"))
	if scopePath == "" || scopePath == "." || strings.HasPrefix(scopePath, "../") {
		return
	}
	dir := filepath.Join(projectDir, filepath.FromSlash(scopePath))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	for _, contract := range packet.ArtifactContracts {
		artifactType := strings.TrimSpace(contract.ArtifactType)
		if !contract.Required || artifactType == "" {
			continue
		}
		body := `{"artifact_type":"` + artifactType + `","payload_json":{"body":"Contenido de prueba external-work goal-first residente.","topic_id":"topic-ref-http-resident-001"}}`
		_ = os.WriteFile(filepath.Join(dir, artifactType+".json"), []byte(body), 0o600)
	}
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
