package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestCodexStackAutoprogrammingExecutorV0UsaPuertosDelStackYDevuelveContinue(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackAutoprogrammingExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:               autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:            "2026-05-22T11:00:00Z",
		CorrelationID:         "corr-autoprogramming-stack-executor-001",
		RequestedBy:           "orquesta-stack-executor-test",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		MaxBursts:             3,
		MaxCommands:           5,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Accepted || len(result.Tasks) != 1 || len(result.WaitAgentRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Continue.RunRef != result.Run.RunID ||
		result.Continue.CorrelationID != "corr-autoprogramming-stack-executor-001" ||
		result.Continue.WaitAgentRefs[0] != result.WaitAgentRefs[0] {
		t.Fatalf("continue=%+v wait=%v", result.Continue, result.WaitAgentRefs)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), result.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		run.Tasks[0] != result.Tasks[0].TaskID {
		t.Fatalf("run=%+v tasks=%+v", run, result.Tasks)
	}
	storedTasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(
		context.Background(),
		result.Run.RunID,
		result.Run.Tasks,
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 || storedTasks[0].TaskID != result.Tasks[0].TaskID {
		t.Fatalf("stored_tasks=%+v result_tasks=%+v", storedTasks, result.Tasks)
	}
}

func TestCodexStackAutoprogrammingExecutorV0RequiereStack(t *testing.T) {
	result, err := NewCodexStackAutoprogrammingExecutorV0(nil).Execute(
		context.Background(),
		AutoprogrammingBridgeRequestV0{Request: autoprogrammingBridgeRequestForTestV0()},
	)
	if err == nil {
		t.Fatalf("err nil result=%+v", result)
	}
}

func TestPrepareAutoprogrammingRunFromStackV0PropagaValidacionDePuertos(t *testing.T) {
	_, err := PrepareAutoprogrammingRunFromStackV0(
		context.Background(),
		StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(),
		}},
		AutoprogrammingBridgeRequestV0{Request: autoprogrammingBridgeRequestForTestV0()},
	)
	if err == nil {
		t.Fatalf("err nil")
	}
}

func TestPrepareAutoprogrammingRunFromStackV0GoalFirstFailsBeforeBackendWithoutIntentManifestV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	stack.Stores.AutoprogrammingIntentManifestStore = nil
	request := autoprogrammingBridgeRequestForTestV0()
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	result, err := PrepareAutoprogrammingRunFromStackV0(context.Background(), stack, AutoprogrammingBridgeRequestV0{
		Request: request, DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
	})
	if err != nil || result.Accepted || len(result.Issues) != 1 || result.Issues[0].Code != "intent_manifest_store_missing" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if runtime.launchCountV0() != 0 {
		t.Fatalf("backend invoked without manifest: %d", runtime.launchCountV0())
	}
}

func TestPrepareAutoprogrammingRunFromStackV0LegacyDoesNotRequireIntentManifestV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	stack.Stores.AutoprogrammingIntentManifestStore = nil
	stack.AllowLegacyAutoprogrammingRun = true
	request := autoprogrammingBridgeRequestForTestV0()
	// Legacy accepted this typed identity before manifests existed; the
	// goal-first manifest regex must not retroactively reject that path.
	request.RequestRef = "AUTOPROGRAMMING-REQUEST-REF-LEGACY-UPPERCASE"
	result, err := PrepareAutoprogrammingRunFromStackV0(context.Background(), stack, AutoprogrammingBridgeRequestV0{
		Request:                 request,
		DirectorExecutionMode:   orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AllowLegacyDirectorLoop: true,
	})
	if err != nil || !result.Accepted {
		t.Fatalf("legacy result=%+v err=%v", result, err)
	}
}

func TestPrepareAutoprogrammingRunFromStackV0GoalFirstRejectsManifestIdentityBeforeLaunchV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "AUTOPROGRAMMING-REQUEST-REF-GOAL-FIRST-UPPERCASE"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	result, err := PrepareAutoprogrammingRunFromStackV0(context.Background(), stack, AutoprogrammingBridgeRequestV0{
		Request: request, DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
	})
	foundIdentityIssue := false
	for _, issue := range result.Issues {
		foundIdentityIssue = foundIdentityIssue || issue.Code == "intent_manifest_request_ref_invalid"
	}
	if err != nil || result.Accepted || !foundIdentityIssue {
		t.Fatalf("goal-first invalid identity result=%+v err=%v", result, err)
	}
	if runtime.launchCountV0() != 0 {
		t.Fatalf("backend invoked with invalid manifest identity: %d", runtime.launchCountV0())
	}
}

func TestPrepareAutoprogrammingRunFromStackV0RejectsIntentManifestStoreSubstitutionBeforeLaunchV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	replacementRequest := autoprogrammingBridgeRequestForTestV0()
	replacementRequest.RequestRef = "request-ref-intent-manifest-substitution"
	replacement, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestV0(replacementRequest)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	stack.Stores.AutoprogrammingIntentManifestStore = substitutingAutoprogrammingIntentManifestStoreV0{Replacement: replacement}
	request := autoprogrammingBridgeRequestForTestV0()
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	result, err := PrepareAutoprogrammingRunFromStackV0(context.Background(), stack, AutoprogrammingBridgeRequestV0{
		Request: request, DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
	})
	if err != nil || result.Accepted || len(result.Issues) != 1 || result.Issues[0].Code != "intent_manifest_store_substitution" {
		t.Fatalf("substitution result=%+v err=%v", result, err)
	}
	if runtime.launchCountV0() != 0 {
		t.Fatalf("backend invoked after store substitution: %d", runtime.launchCountV0())
	}
}

type substitutingAutoprogrammingIntentManifestStoreV0 struct {
	Replacement orquestaautoprogramming.AutoprogrammingIntentManifestV0
}

func (store substitutingAutoprogrammingIntentManifestStoreV0) CreateAutoprogrammingIntentManifestIfAbsentV0(context.Context, orquestaautoprogramming.AutoprogrammingIntentManifestV0) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, error) {
	return store.Replacement, nil
}

func (store substitutingAutoprogrammingIntentManifestStoreV0) LoadAutoprogrammingIntentManifestV0(context.Context, string) (orquestaautoprogramming.AutoprogrammingIntentManifestV0, error) {
	return store.Replacement, nil
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0PreparaRunYSupervisorArranca(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepareInput := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-api-001",
		CorrelationID:          "corr-autoprogramming-api-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		OccurredAt:             "2026-05-22T11:15:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, prepareInput)
	if !prepared.Accepted ||
		prepared.RunRef == "" ||
		len(prepared.WorkflowTaskRefs) != 1 ||
		len(prepared.WaitAgentRefs) != 1 ||
		prepared.Continue == nil ||
		prepared.Continue.RunRef != prepared.RunRef {
		t.Fatalf("prepared=%+v", prepared)
	}

	supervisor := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:             "request-autoprogramming-supervisor-001",
		CorrelationID:         "corr-autoprogramming-api-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		RunRef:                prepared.RunRef,
		MaxTicks:              1,
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxBursts:             3,
		MaxStepsPerBurst:      3,
		MaxDispatchesPerWait:  3,
		MaxCommands:           5,
		MaxOutboxPerCycle:     5,
		AllowRepeatedRuns:     true,
	})
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != prepared.RunRef ||
		runtime.launchCountV0() != 1 {
		t.Fatalf("supervisor=%+v launches=%d", supervisor, runtime.launchCountV0())
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("started_agents=%v wait=%v", run.StartedAgents, prepared.WaitAgentRefs)
	}

	repeated := postAutoprogrammingPrepareRunStackV0(t, stack, prepareInput)
	run, err = stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 repeated: %v", err)
	}
	if repeated.RunRef != prepared.RunRef ||
		len(repeated.WaitAgentRefs) != 0 ||
		runtime.launchCountV0() != 1 ||
		!autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("repeated=%+v run=%+v launches=%d", repeated, run, runtime.launchCountV0())
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0AcceptedLegacyVisibleEnStatusV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "request-ref-remoto-telegram-nollm-runtime-20260705-001"
	request.Tasks[0].TaskRef = "task-autoprogramming-c20a585ff5c3-g01"

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, legacyAutoprogrammingPrepareRunInputForStackTestV0(orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-telegram-nollm-runtime-20260705-001",
		OccurredAt:             "2026-07-05T21:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	}))
	if !prepared.Accepted ||
		prepared.RunRef != request.RequestRef ||
		len(prepared.WorkflowTaskRefs) != 1 ||
		len(prepared.WaitAgentRefs) != 1 ||
		prepared.Continue == nil ||
		prepared.Goal != nil {
		t.Fatalf("prepared=%+v", prepared)
	}

	status := postAutoprogrammingStatusStackV0(t, stack, orquestamcp.MCPAutoprogrammingStatusToolInputV0{
		RequestID:  "request-ref-telegram-nollm-status-after-prepare-001",
		OccurredAt: "2026-07-05T21:00:05Z",
	})
	if status.Estado != orquestamcp.MCPAutoprogrammingStatusEstadoOKV0 ||
		status.Queue == nil ||
		!autoprogrammingStatusRankedRunForTestV0(status.Queue.Ranked, prepared.RunRef, "autoprogramming_prepare_run") ||
		status.Operator == nil ||
		!autoprogrammingStatusActiveRunForTestV0(status.Operator.ActiveRuns, prepared.RunRef) ||
		status.QueueHealth == nil ||
		status.QueueHealth.QueuedNotDispatched != 1 ||
		!autoprogrammingStatusDiagnosticForTestV0(status.Diagnostics, "autoprogramming_prepare_run_pending_dispatch", prepared.RunRef) {
		t.Fatalf("status=%+v", status)
	}

	otroRun := postDirectorAPIWithNameV0(t, stack, "app-prioritaria-d1", "App Prioritaria D1")
	setStackRunPriorityForTestV0(t, stack, otroRun.RunRef, otroRun.AppSpec.Slug, 90)
	explicit := postAutoprogrammingStatusStackV0(t, stack, orquestamcp.MCPAutoprogrammingStatusToolInputV0{
		RequestID:  "request-ref-telegram-nollm-status-explicit-001",
		RunRef:     prepared.RunRef,
		QueueLimit: 1,
		OccurredAt: "2026-07-05T21:00:06Z",
	})
	if explicit.Run == nil ||
		explicit.Run.Stats == nil ||
		explicit.Run.Stats.RunRef != prepared.RunRef ||
		explicit.Queue == nil ||
		!autoprogrammingStatusRankedRunForTestV0(explicit.Queue.Ranked, prepared.RunRef, "autoprogramming_prepare_run") ||
		len(explicit.Tasks) != 1 ||
		explicit.Tasks[0].TaskRef != prepared.WorkflowTaskRefs[0] {
		t.Fatalf("explicit=%+v prepared=%+v", explicit, prepared)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoAceptaSiColaNoProyectaRunV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "request-ref-autoprogramming-invisible-queue-001"

	result, err := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-07-06T12:00:00Z",
		"orquesta-stack-api-test",
		droppingRunQueueWriterForTestV0{},
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	).Execute(context.Background(), legacyAutoprogrammingPrepareRunInputForStackTestV0(orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-autoprogramming-invisible-queue-001",
		OccurredAt:             "2026-07-06T12:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_prepare_run_queue_error" ||
		!strings.Contains(result.Errores[0].Message, "autoprogramming_prepare_run_visibility_error") {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0BloqueaLegacySinOptInV0(t *testing.T) {
	config := codexStackBaseConfigForTestV0(t, newFakeCodexStackRuntimeV0(), nil, nil)
	config.AllowLegacyAutoprogrammingRun = false
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	request := autoprogrammingBridgeRequestForTestV0()
	result := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-api-legacy-optin-required-001",
		CorrelationID:          "corr-autoprogramming-api-legacy-optin-required-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: request,
	})
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		result.RunRef != "" ||
		len(result.WorkflowTaskRefs) != 0 ||
		len(result.WaitAgentRefs) != 0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "autoprogramming_legacy_director_loop_opt_in_required" {
		t.Fatalf("result=%+v", result)
	}
	if _, err := stack.Ports.RunStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run legacy no debe persistirse sin opt-in, err=%v", err)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0BloqueaLegacySinModoExplicitoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	request := autoprogrammingBridgeRequestForTestV0()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-api-legacy-mode-required-001",
		CorrelationID:          "corr-autoprogramming-api-legacy-mode-required-001",
		AutoprogrammingRequest: request,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/prepare-run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		result.RunRef != "" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestamcp.MCPAutoprogrammingPrepareRunLegacyDirectorModeRequiredV0 ||
		result.Errores[0].Field != "director_execution_mode" {
		t.Fatalf("result=%+v", result)
	}
	if _, err := stack.Ports.RunStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run legacy no debe persistirse sin modo explicito, err=%v", err)
	}
}

func TestCodexStackRunSupervisorV0BloqueaSupervisorGlobalLegacySinOptInV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	config := codexStackBaseConfigForTestV0(t, runtime, nil, nil)
	config.AllowLegacyAutoprogrammingRun = false
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	executor := NewCodexStackRunSupervisorExecutorV0(&stack)
	for _, input := range []orquestamcp.MCPRunSupervisorToolInputV0{
		{
			RequestID:     "request-autoprogramming-supervisor-legacy-resident-optin-required-001",
			CorrelationID: "corr-autoprogramming-supervisor-legacy-optin-required-001",
			ResidentMode:  true,
			MaxTicks:      1,
		},
		{
			RequestID:     "request-autoprogramming-supervisor-legacy-queue-optin-required-001",
			CorrelationID: "corr-autoprogramming-supervisor-legacy-optin-required-001",
			QueueRef:      DefaultRunQueueRefV0,
			MaxTicks:      1,
		},
		{
			RequestID:     "request-autoprogramming-supervisor-legacy-empty-optin-required-001",
			CorrelationID: "corr-autoprogramming-supervisor-legacy-optin-required-001",
			MaxTicks:      1,
		},
	} {
		result, err := executor.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("Execute %s: %v", input.RequestID, err)
		}
		if result.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
			result.StopReason != "legacy_supervise_requires_director_execution_mode" ||
			len(result.Errores) != 1 ||
			result.Errores[0].Code != "legacy_supervise_requires_director_execution_mode" ||
			!autoprogrammingBridgeStringInSetForTestV0(result.NextActions, "use_goal_first_prepare_run_and_observe_goal") {
			t.Fatalf("result %s=%+v", input.RequestID, result)
		}
	}
	optInResult, err := executor.Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:             "request-autoprogramming-supervisor-legacy-mode-without-optin-001",
		CorrelationID:         "corr-autoprogramming-supervisor-legacy-optin-required-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		ResidentMode:          true,
		MaxTicks:              1,
	})
	if err != nil {
		t.Fatalf("Execute legacy mode without opt-in: %v", err)
	}
	if optInResult.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		optInResult.StopReason != "legacy_supervise_requires_explicit_opt_in" ||
		len(optInResult.Errores) != 1 ||
		optInResult.Errores[0].Code != "legacy_supervise_requires_explicit_opt_in" {
		t.Fatalf("optInResult=%+v", optInResult)
	}
	if runtime.launchCountV0() != 0 {
		t.Fatalf("runtime no debe lanzarse sin opt-in, launches=%d", runtime.launchCountV0())
	}
}

func TestCodexStackRunSupervisorV0BloqueaRunRefLegacySinModoExplicitoV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	config := codexStackBaseConfigForTestV0(t, runtime, nil, nil)
	config.AllowLegacyAutoprogrammingRun = false
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	request := autoprogrammingBridgeRequestForTestV0()
	prepared, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:                 request,
		AllowLegacyDirectorLoop: true,
		MaxBursts:               1,
		MaxStepsPerBurst:        1,
		MaxDispatchesPerWait:    1,
		MaxCommands:             1,
		MaxOutboxPerCycle:       1,
	}, stack.Ports)
	if err != nil || !prepared.Accepted || strings.TrimSpace(prepared.Run.RunID) == "" {
		t.Fatalf("prepared=%+v err=%v", prepared, err)
	}
	runRef := strings.TrimSpace(prepared.Run.RunID)

	result, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:     "request-autoprogramming-supervisor-runref-mode-required-001",
			CorrelationID: "corr-autoprogramming-supervisor-runref-mode-required-001",
			RunRef:        runRef,
			MaxTicks:      1,
		},
	)
	if err != nil {
		t.Fatalf("Execute sin modo: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		result.StopReason != "legacy_run_supervise_requires_director_execution_mode" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "legacy_run_supervise_requires_director_execution_mode" {
		t.Fatalf("result=%+v", result)
	}
	if runtime.launchCountV0() != 0 {
		t.Fatalf("runtime no debe lanzarse sin marca legacy, launches=%d", runtime.launchCountV0())
	}

	optInResult, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:             "request-autoprogramming-supervisor-runref-legacy-without-optin-001",
			CorrelationID:         "corr-autoprogramming-supervisor-runref-mode-required-001",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
			RunRef:                runRef,
			MaxTicks:              1,
		},
	)
	if err != nil {
		t.Fatalf("Execute legacy without opt-in: %v", err)
	}
	if optInResult.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		optInResult.StopReason != "legacy_run_supervise_requires_explicit_opt_in" ||
		len(optInResult.Errores) != 1 ||
		optInResult.Errores[0].Code != "legacy_run_supervise_requires_explicit_opt_in" {
		t.Fatalf("optInResult=%+v", optInResult)
	}

	stack.AllowLegacyAutoprogrammingRun = true
	allowed, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(
		context.Background(),
		orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:             "request-autoprogramming-supervisor-runref-mode-legacy-001",
			CorrelationID:         "corr-autoprogramming-supervisor-runref-mode-required-001",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
			RunRef:                runRef,
			MaxTicks:              1,
		},
	)
	if err != nil {
		t.Fatalf("Execute con opt-in y modo legacy: %v", err)
	}
	if allowed.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		allowed.RunRef != runRef ||
		allowed.StopReason == "legacy_run_supervise_requires_director_execution_mode" ||
		allowed.StopReason == "legacy_run_supervise_requires_explicit_opt_in" {
		t.Fatalf("allowed=%+v launches=%d", allowed, runtime.launchCountV0())
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0DevuelveGoalSpecsCuandoGoalReady(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-ready-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-ready-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-goal-ready-api-001",
		CorrelationID:          "corr-autoprogramming-goal-ready-api-001",
		OccurredAt:             "2026-06-25T12:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})

	if !prepared.Accepted || prepared.RunRef != "" || len(prepared.WorkflowTaskRefs) != 0 ||
		len(prepared.WaitAgentRefs) != 0 ||
		prepared.Continue != nil ||
		prepared.PhaseID != "" ||
		len(prepared.GoalSpecs) != 0 ||
		len(prepared.GoalSpecSummaries) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}
	summary := prepared.GoalSpecSummaries[0]
	if summary.RunRef != "" ||
		summary.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		summary.SpecHash == "" ||
		summary.WriteSetCount != 1 ||
		summary.RequiredTestCount != 1 {
		t.Fatalf("goal spec summary inesperado=%+v request=%+v", summary, request)
	}
	if run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), request.RequestRef); err == nil {
		t.Fatalf("run legacy materializada en goal_ready: %+v", run)
	} else if !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("LoadRunV0 goal_ready: %v", err)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("goal_ready no debe encolar loop legacy: %+v", ranking)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0BackendGoalCompletoMarcaGoalFirstPorDefecto(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	projectDir := t.TempDir()
	writePath := filepath.Join(projectDir, "modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go")
	if err := os.MkdirAll(filepath.Dir(writePath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(writePath, []byte("package orquestaappcodexstack\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	stack.Codex.ProjectWorkDir = projectDir
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{
		Enabled:                false,
		GoalFirstSnapshotStore: snapshotStore,
	}
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalLauncher = launcher
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalObserver = &goalFirstQueueObserverForTestV0{}
	stack.Ports.GoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	stack.Ports.GoalStateStore = goalStates
	enableIndependentAttestationForStackTestV0(&stack)
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-default-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-default-001"
	request.Tasks[0].ContextRefs = nil

	prepared, err := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-06-27T12:00:00Z",
		"orquesta-stack-api-test",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	).Execute(context.Background(), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-goal-default-api-001",
		CorrelationID:          "corr-autoprogramming-goal-default-api-001",
		OccurredAt:             "2026-06-27T12:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
	})
	if err != nil {
		t.Fatalf("prepare.Execute: %v", err)
	}

	if !prepared.Accepted ||
		prepared.RunRef != request.RequestRef ||
		prepared.Goal == nil ||
		prepared.Goal.RunRef != prepared.RunRef ||
		prepared.Goal.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		len(prepared.GoalSpecs) != 1 ||
		len(prepared.WorkflowTaskRefs) != 0 ||
		len(prepared.WaitAgentRefs) != 0 ||
		prepared.Continue != nil ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("prepared=%+v runtime_launches=%d", prepared, runtime.launchCountV0())
	}
	if len(launcher.specs) != 1 || launcher.specs[0].RunRef != prepared.RunRef {
		t.Fatalf("launcher specs=%+v prepared=%+v", launcher.specs, prepared)
	}
	baselineRef := goalMaterialProgressContextRefV0(launcher.specs[0].ContextRefs, "worktree_baseline")
	if baselineRef == "" {
		t.Fatalf("goal sin baseline con promotion disabled: %+v", launcher.specs[0].ContextRefs)
	}
	if _, err := snapshotStore.LoadWorktreeSnapshotV0(context.Background(), baselineRef); err != nil {
		t.Fatalf("baseline no persistido con promotion disabled: %v", err)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
		t.Fatalf("goal-first por defecto materializo loop legacy: %+v", run)
	}
	if _, err := goalStates.LoadGoalWorkStateV0(context.Background(), prepared.RunRef); err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("goal-first por defecto no debe encolar legacy: %+v", ranking)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0GoalReadyLanzaGoalFirstSinColaLegacy(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	promotionPort := &fakeAutoprogrammingPromotionPortV0{}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{
		Enabled: true, Port: promotionPort,
		GoalFirstSnapshotStore: orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(),
	}
	launcher := &goalFirstQueueLauncherForTestV0{}
	observer := &goalFirstQueueObserverForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalLauncher = launcher
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalObserver = observer
	stack.Ports.GoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	stack.Ports.GoalStateStore = goalStates
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-first-launch-001"
	enableIndependentAttestationForStackTestV0(&stack)
	request.WriteSet = []string{"generated-apps"}
	request = withAutoprogrammingAttestationForTestV0(request)
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-first-launch-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}

	prepareExecutor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-06-25T12:30:00Z",
		"orquesta-stack-api-test",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	)
	prepared, err := prepareExecutor.Execute(context.Background(), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-goal-first-launch-api-001",
		CorrelationID:          "corr-autoprogramming-goal-first-launch-api-001",
		OccurredAt:             "2026-06-25T12:30:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if err != nil {
		t.Fatalf("prepare.Execute: %v", err)
	}

	if !prepared.Accepted ||
		prepared.RunRef != request.RequestRef ||
		prepared.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) ||
		prepared.Goal == nil ||
		prepared.Goal.RunRef != prepared.RunRef ||
		prepared.Goal.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		len(prepared.GoalSpecs) != 1 ||
		prepared.GoalSpecs[0].RunRef != prepared.RunRef ||
		len(prepared.WorkflowTaskRefs) != 0 ||
		len(prepared.WaitAgentRefs) != 0 ||
		prepared.Continue != nil {
		t.Fatalf("prepared=%+v", prepared)
	}
	if len(launcher.specs) != 1 || launcher.specs[0].RunRef != prepared.RunRef {
		t.Fatalf("launcher specs=%+v prepared=%+v", launcher.specs, prepared)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
		t.Fatalf("run goal-first materializo loop legacy: %+v", run)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.GoalRef != prepared.Goal.GoalRef ||
		state.Spec.RunRef != prepared.RunRef {
		t.Fatalf("state=%+v prepared=%+v", state, prepared)
	}
	generatedDir := filepath.Join(stack.Codex.ProjectWorkDir, "generated-apps")
	if err := os.MkdirAll(generatedDir, 0o700); err != nil {
		t.Fatalf("mkdir generated-apps: %v", err)
	}
	if err := os.WriteFile(filepath.Join(generatedDir, "bug088_second_artifact.txt"), []byte("second artifact\n"), 0o600); err != nil {
		t.Fatalf("write materialized artifact: %v", err)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("goal-first no debe encolar legacy: %+v", ranking)
	}
	supervisor, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-autoprogramming-goal-first-legacy-supervisor-001",
		CorrelationID: "corr-autoprogramming-goal-first-launch-api-001",
		RunRef:        prepared.RunRef,
		MaxTicks:      1,
	})
	if err != nil {
		t.Fatalf("supervisor goal-first: %v", err)
	}
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != prepared.RunRef ||
		supervisor.StopReason != "goal_first_observe_required" ||
		supervisor.Last.Status != string(CodexSupervisorRuntimeRunningLiveV0) ||
		!codexStackStringInSetForTestV0(supervisor.NextActions, "observe_goal") ||
		!codexStackDiagnosticsContainCodeForTestV0(supervisor.Diagnostics, "run_supervisor_goal_first_not_legacy") ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("supervisor=%+v launches=%d", supervisor, runtime.launchCountV0())
	}
	run, err = stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 tras supervisor legacy: %v", err)
	}
	if len(run.StartedAgents) != 0 {
		t.Fatalf("supervisor legacy arranco agentes en goal-first: %+v", run.StartedAgents)
	}

	observer.result = orquestagoal.GoalWorkResultV0{
		SchemaVersion:       orquestagoal.GoalWorkResultSchemaV0,
		Status:              orquestagoal.GoalStatusCompleteV0,
		GoalRef:             state.GoalRef,
		ExternalGoalRef:     state.ExternalGoalRef,
		Summary:             "autoprogramming goal-first completado",
		RequiredTestResults: autoprogrammingGoalRequiredTestResultsForTestV0(state.Spec, "evidence-ref-autoprogramming-goal-first-test-passed"),
		EvidenceRefs:        state.Spec.ClosurePolicy.RequiredEvidenceRefs,
	}
	observeExecutor := NewCodexStackAutoprogrammingObserveGoalExecutorV0(&stack)
	observed, err := observeExecutor.Execute(context.Background(), orquestamcp.MCPAutoprogrammingObserveGoalToolInputV0{
		RequestID:     "request-autoprogramming-goal-first-observe-api-001",
		CorrelationID: "corr-autoprogramming-goal-first-launch-api-001",
		RunRef:        prepared.RunRef,
	})
	if err != nil {
		t.Fatalf("observe.Execute: %v", err)
	}
	if observed.Estado != orquestamcp.MCPAutoprogrammingObserveGoalEstadoOKV0 ||
		observed.RunRef != prepared.RunRef ||
		observed.GoalStatus != orquestagoal.GoalStatusCompleteV0 ||
		!observed.ClosureAccepted ||
		observed.RunStatus != string(orquestacoreworkflow.OrchestrationRunStatusClosedV0) ||
		!codexStackStringHasPrefixForTestV0(observed.ArtifactRefs, "artifact-ref-materialized:") ||
		!codexStackStringInSetForTestV0(observed.EvidenceRefs, "evidence-ref-goal-materialized-partial-artifacts-written") {
		t.Fatalf("observed=%+v", observed)
	}
	closed, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 closed: %v", err)
	}
	if closed.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("closed run=%+v", closed)
	}
	terminalQueue := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:               orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef:             DefaultRunQueueRefV0,
		IncludeNonExecutable: true,
		Limit:                1,
	})
	if len(terminalQueue.Terminal) != 1 ||
		terminalQueue.Terminal[0].RunRef != prepared.RunRef ||
		terminalQueue.Terminal[0].Status != "closed" {
		t.Fatalf("terminalQueue=%+v", terminalQueue)
	}
	if promotionPort.promotions != 1 || promotionPort.archives != 1 {
		t.Fatalf("promotion=%+v", promotionPort)
	}
}

func TestCodexStackRunSupervisorV0NoDrenaLegacySiGoalFirstNoTieneState(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	stack.Ports.GoalStateStore = newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-missing-state-supervisor-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-missing-state-supervisor-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request)
	if !work.Accepted || len(work.Work.GoalSpecs) != 1 {
		t.Fatalf("work=%+v", work)
	}
	run := autoprogrammingBridgeGoalRunV0(
		AutoprogrammingBridgeRequestV0{Request: request},
		work.Work,
	)
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	supervisor, err := NewCodexStackRunSupervisorExecutorV0(&stack).Execute(context.Background(), orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-autoprogramming-goal-missing-state-supervisor-001",
		CorrelationID: "corr-autoprogramming-goal-missing-state-supervisor-001",
		RunRef:        run.RunID,
		MaxTicks:      1,
	})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		supervisor.RunRef != run.RunID ||
		supervisor.StopReason != "goal_first_state_missing" ||
		supervisor.Last.Status != string(CodexSupervisorRuntimeStoppedV0) ||
		!codexStackDiagnosticsContainCodeForTestV0(supervisor.Diagnostics, "run_supervisor_goal_first_state_missing") ||
		!codexStackStringInSetForTestV0(supervisor.NextActions, "do_not_supervise_goal_first_with_legacy_loop") ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("supervisor=%+v launches=%d", supervisor, runtime.launchCountV0())
	}
	directDrain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:     run.RunID,
		OccurredAt: "2026-06-27T12:45:00Z",
		MaxBursts:  1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 directo goal-first sin state: %v", err)
	}
	if directDrain.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		directDrain.Final.Run.RunID != run.RunID ||
		len(directDrain.Attempts) != 1 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("directDrain=%+v launches=%d", directDrain, runtime.launchCountV0())
	}
}

func TestCodexStackSupervisorGlobalNoDrenaLegacySiGoalFirstEnCola(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalLauncher = launcher
	stack.Ports.GoalObserver = &goalFirstQueueObserverForTestV0{}
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	stack.Ports.GoalStateStore = goalStates
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-first-global-supervisor-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-first-global-supervisor-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	prepared, err := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-06-27T12:30:00Z",
		"orquesta-stack-api-test",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	).Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-goal-first-global-supervisor-001",
		CorrelationID:          "corr-autoprogramming-goal-first-global-supervisor-001",
		OccurredAt:             "2026-06-27T12:30:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
	})
	if err != nil {
		t.Fatalf("prepare.Execute: %v", err)
	}
	if !prepared.Accepted || prepared.Goal == nil || len(prepared.WorkflowTaskRefs) != 0 {
		t.Fatalf("prepared=%+v", prepared)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        prepared.RunRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        prepared.ProjectRef,
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: DefaultRunQueuePriorityScoreV0,
		UpdatedAt:     stackNowV0(stack.Clock),
		RequestedBy:   "orquesta-goal-first-global-supervisor-test",
		Reason:        "candidato accidental goal-first para probar guard",
		EvidenceRefs:  []string{"evidence-ref-goal-first-global-supervisor-queued"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	supervised, err := stack.RunGlobalSupervisorV0(ctx, orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          DefaultRunQueueRefV0,
		MaxTicks:          1,
		MaxRunsPerTick:    1,
		MaxExecutions:     1,
		AllowRepeatedRuns: true,
		CorrelationID:     "corr-autoprogramming-goal-first-global-supervisor-001",
	})
	if err != nil {
		t.Fatalf("RunGlobalSupervisorV0: %v", err)
	}
	if supervised.TotalExecutions != 1 ||
		len(supervised.Ticks) != 1 ||
		len(supervised.Ticks[0].Result.Executions) != 1 {
		t.Fatalf("supervised=%+v", supervised)
	}
	execution := supervised.Ticks[0].Result.Executions[0]
	if execution.RunRef != prepared.RunRef ||
		execution.Outcome != codexStackGoalFirstObserveRequiredOutcomeV0 ||
		execution.QueueStatus != orquestarunqueue.RunStatusDeliveredV0 ||
		!codexStackDrainDiagnosticsContainKindStatusForTestV0(execution.Diagnostics, "goal_first", "observe_required") ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("execution=%+v launches=%d", execution, runtime.launchCountV0())
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.StartedAgents) != 0 {
		t.Fatalf("guard global materializo legacy: %+v", run)
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             DefaultRunQueueRefV0,
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 ||
		candidates[0].RunRef != prepared.RunRef ||
		candidates[0].Status != orquestarunqueue.RunStatusDeliveredV0 {
		t.Fatalf("candidates=%+v", candidates)
	}
	snapshot, err := (CodexSupervisorStackLifecycleV0{
		Stack:  stack,
		RunRef: prepared.RunRef,
	}).LaunchV0(ctx)
	if err != nil {
		t.Fatalf("LaunchV0 lifecycle directo: %v", err)
	}
	if snapshot.Status != CodexSupervisorRuntimeRunningLiveV0 ||
		snapshot.SessionRef != prepared.RunRef ||
		!codexStackDrainDiagnosticsContainKindStatusForTestV0(snapshot.Diagnostics, "goal_first", "observe_required") ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("snapshot=%+v launches=%d", snapshot, runtime.launchCountV0())
	}
	directDrain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:     prepared.RunRef,
		OccurredAt: "2026-06-27T12:30:00Z",
		MaxBursts:  1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 directo goal-first: %v", err)
	}
	if directDrain.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		directDrain.Final.Run.RunID != prepared.RunRef ||
		len(directDrain.Attempts) != 1 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("directDrain=%+v launches=%d", directDrain, runtime.launchCountV0())
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0GoalReadyLanzaBatchGoalsSinColaLegacy(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalLauncher = launcher
	stack.Ports.GoalObserver = &goalFirstQueueObserverForTestV0{}
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalClosureValidator = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	stack.Ports.GoalStateStore = goalStates
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-first-batch-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-first-batch-api-001"
	request.Tasks[0].Area = "api"
	request.Tasks[0].RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestGoalBatchAPI"}
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	second := request.Tasks[0]
	second.TaskRef = "source-task-ref-autoprogramming-goal-first-batch-web-001"
	second.Area = "web"
	second.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestGoalBatchWeb"}
	request.Tasks = append(request.Tasks, second)
	request.WriteSet = []string{
		"modulos/orquesta-app-codex-stack/api/batch_goal.go",
		"modulos/orquesta-app-codex-stack/web/batch_goal.go",
	}
	request.MaxTaskRefs = 2
	request.MaxAreas = 2
	request.MaxWriteSetEntries = 2
	request = withAutoprogrammingAttestationForTestV0(request)
	blocked, err := PrepareAutoprogrammingRunFromStackV0(context.Background(), stack, AutoprogrammingBridgeRequestV0{Request: request})
	if err != nil || blocked.Accepted || len(blocked.Issues) != 1 || blocked.Issues[0].Code != "physical_goal_workspace_required" {
		t.Fatalf("multi-goal shared workspace must fail before launch: blocked=%+v err=%v", blocked, err)
	}
	stack.AutoprogrammingPromotion.GoalWorkspaceProvisioner = &fakeGoalWorkspaceProvisionerForStackTestV0{root: t.TempDir()}
	stack.AutoprogrammingPromotion.GoalWorkspaceRoot = t.TempDir()
	stack.Stores.AutoprogrammingBatchStore = newAutoprogrammingBatchStoreForTestV0()
	// A request-only manifest from the failed preflight is immutable. The
	// versioned prepare envelope therefore retries under a fresh request ref.
	request.RequestRef = "run-autoprogramming-goal-first-batch-retry-001"

	prepared, err := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		&stack,
		"2026-06-27T10:30:00Z",
		"orquesta-stack-api-test",
		stack.Stores.RunQueue,
		stack.RunQueue,
		stack.Clock,
		stack.Codex.RuntimeWorkDir,
	).Execute(context.Background(), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-goal-first-batch-api-001",
		CorrelationID:          "corr-autoprogramming-goal-first-batch-api-001",
		OccurredAt:             "2026-06-27T10:30:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
	})
	if err != nil {
		t.Fatalf("prepare.Execute: %v", err)
	}

	wantRunRefs := []string{
		"run-autoprogramming-goal-first-batch-retry-001-goal-01",
		"run-autoprogramming-goal-first-batch-retry-001-goal-02",
	}
	if !prepared.Accepted ||
		prepared.RunRef != wantRunRefs[0] ||
		prepared.Goal != nil ||
		len(prepared.Goals) != 2 ||
		len(prepared.GoalSpecs) != 2 ||
		len(prepared.WorkflowTaskRefs) != 0 ||
		len(prepared.WaitAgentRefs) != 0 ||
		prepared.Continue != nil {
		t.Fatalf("prepared=%+v", prepared)
	}
	for i, runRef := range wantRunRefs {
		if prepared.Goals[i].RunRef != runRef ||
			prepared.GoalSpecs[i].RunRef != runRef ||
			prepared.Goals[i].GoalStatus != orquestagoal.GoalStatusRunningV0 {
			t.Fatalf("goal %d prepared=%+v specs=%+v want_run=%s", i, prepared.Goals, prepared.GoalSpecs, runRef)
		}
		run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), runRef)
		if err != nil {
			t.Fatalf("LoadRunV0 %s: %v", runRef, err)
		}
		if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
			t.Fatalf("batch goal materializo loop legacy: %+v", run)
		}
		if _, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef); err != nil {
			t.Fatalf("LoadGoalWorkStateV0 %s: %v", runRef, err)
		}
	}
	if len(launcher.specs) != 2 ||
		launcher.specs[0].RunRef != wantRunRefs[0] ||
		launcher.specs[1].RunRef != wantRunRefs[1] {
		t.Fatalf("launcher specs=%+v", launcher.specs)
	}
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("batch goal-first no debe encolar legacy: %+v", ranking)
	}
}

type fakeGoalWorkspaceProvisionerForStackTestV0 struct {
	root        string
	lastRequest orquestaruntimeworktree.GoalWorkspaceRequestV0
}

func (fake *fakeGoalWorkspaceProvisionerForStackTestV0) PrepareGoalWorkspaceV0(
	_ context.Context,
	request orquestaruntimeworktree.GoalWorkspaceRequestV0,
) (orquestaruntimeworktree.GoalWorkspaceV0, []orquestaruntimeworktree.WorktreeIssueV0) {
	fake.lastRequest = request
	dir := filepath.Join(fake.root, request.GoalRef)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return orquestaruntimeworktree.GoalWorkspaceV0{}, []orquestaruntimeworktree.WorktreeIssueV0{{Code: orquestaruntimeworktree.WorktreeIssueFilesystemV0}}
	}
	return orquestaruntimeworktree.GoalWorkspaceV0{
		SchemaVersion: orquestaruntimeworktree.GoalWorkspaceSchemaVersionV0,
		RunRef:        request.RunRef, GoalRef: request.GoalRef, ProjectRef: request.ProjectRef,
		WorktreeRef: request.WorktreeRef, WorkspaceID: "workspace-" + request.GoalRef,
		ProjectWorkDir: dir, BaseRevision: "base-revision-test",
	}, nil
}

func (fake *fakeGoalWorkspaceProvisionerForStackTestV0) ResolveGoalWorkspaceV0(
	ctx context.Context,
	request orquestaruntimeworktree.GoalWorkspaceRequestV0,
) (orquestaruntimeworktree.GoalWorkspaceV0, []orquestaruntimeworktree.WorktreeIssueV0) {
	return fake.PrepareGoalWorkspaceV0(ctx, request)
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0EncolaYSupervisorGlobalArranca(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepareInput := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-queue-api-001",
		CorrelationID:          "corr-autoprogramming-queue-api-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		OccurredAt:             "2026-05-22T11:25:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, prepareInput)
	if !prepared.Accepted || prepared.RunRef == "" || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != prepared.RunRef ||
		ranking.Ranked[0].AppRef != prepared.ProjectRef ||
		ranking.Ranked[0].PriorityScore != DefaultRunQueuePriorityScoreV0 {
		t.Fatalf("ranking=%+v prepared=%+v", ranking, prepared)
	}

	supervisor := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:             "request-autoprogramming-global-supervisor-001",
		CorrelationID:         "corr-autoprogramming-queue-api-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		QueueRef:              DefaultRunQueueRefV0,
		MaxTicks:              1,
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxBursts:             3,
		MaxStepsPerBurst:      3,
		MaxDispatchesPerWait:  3,
		MaxCommands:           5,
		MaxOutboxPerCycle:     5,
		AllowRepeatedRuns:     true,
	})
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != prepared.RunRef ||
		supervisor.Last.SessionRef != prepared.RunRef ||
		runtime.launchCountV0() != 1 {
		t.Fatalf("supervisor=%+v launches=%d", supervisor, runtime.launchCountV0())
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("started_agents=%v wait=%v", run.StartedAgents, prepared.WaitAgentRefs)
	}
}

func TestCodexStackAutoprogrammingSupervisorGlobalReemplazaAskDirectorPerdido(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-global-stopped-001",
		CorrelationID:          "corr-autoprogramming-global-stopped-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		OccurredAt:             "2026-05-22T11:35:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}
	first := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:             "request-autoprogramming-global-stopped-supervisor-001",
		CorrelationID:         "corr-autoprogramming-global-stopped-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		QueueRef:              DefaultRunQueueRefV0,
		MaxTicks:              1,
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxBursts:             3,
		MaxStepsPerBurst:      3,
		MaxDispatchesPerWait:  3,
		MaxCommands:           5,
		MaxOutboxPerCycle:     5,
		AllowRepeatedRuns:     true,
	})
	if first.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 || runtime.launchCountV0() != 1 {
		t.Fatalf("first=%+v launches=%d", first, runtime.launchCountV0())
	}
	agentRef := prepared.WaitAgentRefs[0]
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, prepared.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	runtime.markStoppedForTestV0(record.ProcessRef)

	second := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:             "request-autoprogramming-global-stopped-supervisor-002",
		CorrelationID:         "corr-autoprogramming-global-stopped-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		QueueRef:              DefaultRunQueueRefV0,
		MaxTicks:              1,
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxBursts:             8,
		MaxStepsPerBurst:      6,
		MaxDispatchesPerWait:  8,
		MaxCommands:           20,
		MaxOutboxPerCycle:     8,
		AllowRepeatedRuns:     true,
	})
	if second.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		t.Fatalf("second=%+v", second)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.LostAgents, agentRef) ||
		autoprogrammingBridgeStringInSetForTestV0(run.StoppedAgents, agentRef) ||
		autoprogrammingBridgeStringInSetForTestV0(run.ConfirmedStoppedAgents, agentRef) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("supervisor global no activo revision recuperable: lost=%v stopped=%v confirmed=%v assessments=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents, run.AgentAssessments)
	}

	third := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:             "request-autoprogramming-global-stopped-supervisor-003",
		CorrelationID:         "corr-autoprogramming-global-stopped-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		QueueRef:              DefaultRunQueueRefV0,
		MaxTicks:              1,
		MaxRunsPerTick:        1,
		MaxExecutions:         1,
		MaxBursts:             8,
		MaxStepsPerBurst:      6,
		MaxDispatchesPerWait:  8,
		MaxCommands:           20,
		MaxOutboxPerCycle:     8,
		AllowRepeatedRuns:     true,
	})
	if third.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		t.Fatalf("third=%+v", third)
	}
	run, err = stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 third: %v", err)
	}
	if len(run.StartedAgents) != 2 || runtime.launchCountV0() != 2 ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) ||
		!codexStackRefsContainPartV0(run.ReplanDecisions, "#action:"+string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)) {
		t.Fatalf("supervisor no reemplazo ask_director perdido: started=%v launches=%d assessments=%v replans=%v run=%+v",
			run.StartedAgents,
			runtime.launchCountV0(),
			run.AgentAssessments,
			run.ReplanDecisions,
			run,
		)
	}
}

func TestCodexStackAutoprogrammingSupervisorResidenteReemplazaAskDirectorPerdido(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-resident-lost-001",
		CorrelationID:          "corr-autoprogramming-resident-lost-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		OccurredAt:             "2026-05-22T11:45:00Z",
		RequestedBy:            "orquesta-stack-resident-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}
	command := orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          DefaultRunQueueRefV0,
		MaxTicks:          1,
		MaxRunsPerTick:    1,
		MaxExecutions:     1,
		AllowRepeatedRuns: true,
		AllowLegacyDrain:  true,
		CorrelationID:     "corr-autoprogramming-resident-lost-001",
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            8,
			MaxStepsPerBurst:     6,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxDecisionCycles:    1,
		},
	}
	if result, err := stack.RunGlobalSupervisorV0(ctx, command); err != nil || runtime.launchCountV0() != 1 {
		t.Fatalf("first result=%+v err=%v launches=%d", result, err, runtime.launchCountV0())
	}
	agentRef := prepared.WaitAgentRefs[0]
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, prepared.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	runtime.markStoppedForTestV0(record.ProcessRef)

	if result, err := stack.RunGlobalSupervisorV0(ctx, command); err != nil {
		t.Fatalf("second result=%+v err=%v", result, err)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 second: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.LostAgents, agentRef) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("second no dejo assessment recuperable: lost=%v assessments=%v run=%+v", run.LostAgents, run.AgentAssessments, run)
	}

	if result, err := stack.RunGlobalSupervisorV0(ctx, command); err != nil {
		t.Fatalf("third result=%+v err=%v", result, err)
	}
	run, err = stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 third: %v", err)
	}
	if len(run.StartedAgents) != 2 || runtime.launchCountV0() != 2 ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) ||
		!codexStackRefsContainPartV0(run.ReplanDecisions, "#action:"+string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)) {
		t.Fatalf("supervisor residente no reemplazo ask_director perdido: started=%v launches=%d assessments=%v replans=%v run=%+v",
			run.StartedAgents,
			runtime.launchCountV0(),
			run.AgentAssessments,
			run.ReplanDecisions,
			run,
		)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0RespetaPrioridadSolicitada(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-low-priority-001",
		CorrelationID:          "corr-autoprogramming-low-priority-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		PriorityScore:          10,
	})
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != prepared.RunRef ||
		ranking.Ranked[0].PriorityScore != 10 {
		t.Fatalf("ranking=%+v prepared=%+v", ranking, prepared)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0ReabreCandidatoStoppedComoReadyV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-reopen-stopped-001",
		CorrelationID:          "corr-autoprogramming-reopen-stopped-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	stopped := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		RequestID:     "request-autoprogramming-reopen-stopped-stop-001",
		CorrelationID: "corr-autoprogramming-reopen-stopped-001",
		Action:        orquestamcp.MCPRunQueuePriorityActionSetV0,
		QueueRef:      DefaultRunQueueRefV0,
		RunRef:        prepared.RunRef,
		AppRef:        prepared.ProjectRef,
		Status:        "stopped",
		PriorityScore: 0,
	})
	if stopped.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
		t.Fatalf("stopped=%+v", stopped)
	}
	empty := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(empty.Ranked) != 0 {
		t.Fatalf("stopped candidate visible=%+v", empty)
	}

	reopened := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if !reopened.Accepted ||
		reopened.RunRef != prepared.RunRef ||
		len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != prepared.RunRef ||
		ranking.Ranked[0].Status != "ready" {
		t.Fatalf("reopened=%+v ranking=%+v", reopened, ranking)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoReencolaRunCerradaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-closed-idempotent-001",
		CorrelationID:          "corr-autoprogramming-closed-idempotent-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	run.Status = orquestacoreworkflow.OrchestrationRunStatusClosedV0
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 closed: %v", err)
	}

	reprepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})

	if !reprepared.Accepted ||
		reprepared.RunRef != prepared.RunRef ||
		len(ranking.Ranked) != 0 {
		t.Fatalf("reprepared=%+v ranking=%+v", reprepared, ranking)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoReencolaRunCanceladaPorControlV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-cancel-control-001",
		CorrelationID:          "corr-autoprogramming-cancel-control-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	cancelled := postRunControlStackV0(t, stack, orquestamcp.MCPRunControlToolInputV0{
		Action:      "cancel",
		RunRef:      prepared.RunRef,
		RequestedBy: "director",
		Reason:      "run apartada por humano",
	})
	if cancelled.Status != string(orquestaruncontrol.RunControlStatusCancelRequestedV0) {
		t.Fatalf("cancelled=%+v", cancelled)
	}
	reprepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})

	if !reprepared.Accepted ||
		reprepared.RunRef != prepared.RunRef ||
		len(ranking.Ranked) != 0 {
		t.Fatalf("reprepared=%+v ranking=%+v", reprepared, ranking)
	}
}

func TestCodexStackServerShutdownV0CierraRunPreparadaSinEntrarAlDirector(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-shutdown-001",
		CorrelationID:          "corr-autoprogramming-shutdown-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		OccurredAt:             "2026-05-23T12:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	shutdown := postServerShutdownStackV0(t, stack, orquestamcp.MCPServerShutdownToolInputV0{
		RequestID:      "request-server-shutdown-prepared-001",
		CorrelationID:  "corr-autoprogramming-shutdown-001",
		Forced:         true,
		MaxTicks:       1,
		MaxRunsPerTick: 1,
		MaxExecutions:  1,
		RequestedBy:    "orquesta-director",
		Reason:         "shutdown de run preparada sin agentes vivos",
	})
	if shutdown.Estado != orquestamcp.MCPServerShutdownEstadoOKV0 ||
		!shutdown.ShutdownReady ||
		shutdown.Status != orquestaservershutdown.ServerShutdownStatusReadyV0 ||
		shutdown.RunsRequested != 1 ||
		shutdown.RunsStopped != 1 ||
		shutdown.AgentsInFlight != 0 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("shutdown=%+v launches=%d", shutdown, runtime.launchCountV0())
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0DevuelveErroresPublicos(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-autoprogramming-api-invalid-001",
	})

	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0BloqueaConfigProjectionMismatchV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	config := codexStackBaseConfigForTestV0(t, runtime, nil, nil)
	config.ConfigProjectionSettings = []orquestamcp.MCPConfigProjectionSettingV0{{
		Key:   "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS",
		Value: "450000",
	}}
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-ref-autoprogramming-config-projection-mismatch-001"

	result := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-config-projection-mismatch-001",
		CorrelationID:          "corr-autoprogramming-config-projection-mismatch-001",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: request,
		RequiredSettings: []orquestamcp.MCPRequiredSettingV0{{
			Key:   "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS",
			Value: "999999",
		}},
	})

	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		result.RunRef != "" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != orquestamcp.MCPConfigProjectionMismatchV0 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("result=%+v launches=%d", result, runtime.launchCountV0())
	}
	if _, err := stack.Ports.RunStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("prepare-run con mismatch no debe persistir run, err=%v", err)
	}
}

func postAutoprogrammingPrepareRunStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode prepare run: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/prepare-run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("prepare status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode prepare run: %v", err)
	}
	return result
}

func postAutoprogrammingStatusStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPAutoprogrammingStatusToolInputV0,
) orquestamcp.MCPAutoprogrammingStatusToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode autoprogramming status: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/status", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("autoprogramming status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingStatusToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode autoprogramming status: %v", err)
	}
	return result
}

func autoprogrammingStatusRankedRunForTestV0(
	candidates []orquestamcp.MCPRunQueueRankedCandidateCompactV0,
	runRef string,
	reason string,
) bool {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RunRef) == runRef &&
			strings.TrimSpace(candidate.Reason) == reason {
			return true
		}
	}
	return false
}

func autoprogrammingStatusActiveRunForTestV0(
	runs []orquestamcp.MCPAutoprogrammingActiveRunV0,
	runRef string,
) bool {
	for _, run := range runs {
		if strings.TrimSpace(run.RunRef) == runRef {
			return true
		}
	}
	return false
}

func autoprogrammingStatusDiagnosticForTestV0(
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
	code string,
	runRef string,
) bool {
	wantScope := "run:" + strings.TrimSpace(runRef)
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == code &&
			strings.TrimSpace(diagnostic.Scope) == wantScope {
			return true
		}
	}
	return false
}

type droppingRunQueueWriterForTestV0 struct{}

func (droppingRunQueueWriterForTestV0) SetRunPriorityV0(
	context.Context,
	orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	return orquestarunqueue.RunSchedulingCandidateV0{}, nil
}

func legacyAutoprogrammingPrepareRunInputForStackTestV0(
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
	if strings.TrimSpace(input.DirectorExecutionMode) == "" {
		input.DirectorExecutionMode = orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0
	}
	return input
}

func legacyRunSupervisorInputForStackTestV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
) orquestamcp.MCPRunSupervisorToolInputV0 {
	if strings.TrimSpace(input.DirectorExecutionMode) == "" {
		input.DirectorExecutionMode = orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0
	}
	return input
}

func autoprogrammingGoalRequiredTestResultsForTestV0(
	spec orquestagoal.GoalWorkSpecV0,
	evidenceRef string,
) []orquestagoal.GoalRequiredTestResultV0 {
	results := make([]orquestagoal.GoalRequiredTestResultV0, 0, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		var evidenceRefs []string
		if strings.TrimSpace(evidenceRef) != "" {
			evidenceRefs = []string{evidenceRef}
		}
		results = append(results, orquestagoal.GoalRequiredTestResultV0{
			TestRef:      test.TestRef,
			Status:       "passed",
			EvidenceRefs: evidenceRefs,
		})
	}
	return results
}

func codexStackDrainDiagnosticsContainKindStatusForTestV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
	kind string,
	status string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind == kind && diagnostic.Status == status {
			return true
		}
	}
	return false
}

func codexStackStringHasPrefixForTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func postServerShutdownStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPServerShutdownToolInputV0,
) orquestamcp.MCPServerShutdownToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode shutdown: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/server/shutdown", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("shutdown status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPServerShutdownToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode shutdown: %v", err)
	}
	return result
}
