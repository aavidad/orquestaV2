package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexStackSelfImprovementAPIV0AutoPrepareRunEncola(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID:             "request-ref-self-improvement-stack-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoPrepareRun:        true,
		Proposal: orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
			ProjectRef:        "project-ref-autoprogramming-bridge-001",
			WorktreeRef:       "worktree-ref-autoprogramming-bridge-001",
			WorktreeIsolated:  true,
			BranchRef:         "branch-ref-autoprogramming-bridge-001",
			ObservedBy:        "director",
			FailureSummary:    "Mejora reutilizable detectada por el stack.",
			SuggestedArea:     "app-codex-stack",
			SuggestedWriteSet: []string{"modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go"},
			RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0"},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/self-improvement", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingSelfImprovementToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.PreparedRun == nil || !result.PreparedRun.Accepted || result.PreparedRun.RunRef == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestPrepareAutoprogrammingRunV0PersisteWorkflowTasksYRunContinuable(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		OperationalPlanStateStore:  planStore,
		OperationalPlanStateWriter: planStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			autoprogrammingBridgeCapacityDispatcherForTestV0(runStore, eventSink, ledger),
			autoprogrammingBridgeAgentDispatcherForTestV0(runStore, eventSink, ledger),
		},
	}

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:                 autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:              "2026-05-22T10:00:00Z",
		CorrelationID:           "corr-autoprogramming-bridge-001",
		RequestedBy:             "orquesta-test",
		AllowLegacyDirectorLoop: true,
		MaxBursts:               4,
		MaxCommands:             8,
	}, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if !bridged.Accepted || len(bridged.Tasks) != 1 || len(bridged.WaitAgentRefs) != 1 {
		t.Fatalf("bridged=%+v", bridged)
	}
	storedTasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), bridged.Run.RunID, bridged.Run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 ||
		storedTasks[0].TaskID != bridged.Tasks[0].TaskID ||
		storedTasks[0].RunID != bridged.Run.RunID ||
		storedTasks[0].RequiredTests[0] != "go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0" {
		t.Fatalf("stored_tasks=%+v bridged=%+v", storedTasks, bridged.Tasks)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(storedTasks[0].ContextRefs, autoprogrammingBridgeOperationalTaskSourceRefV0) {
		t.Fatalf("context_refs=%v", storedTasks[0].ContextRefs)
	}
	run, err := runStore.LoadRunV0(context.Background(), bridged.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(run.Tasks) != 1 ||
		run.Tasks[0] != bridged.Tasks[0].TaskID {
		t.Fatalf("run no continuable: %+v", run)
	}
	agentTask, err := (CodexLaunchSpecResolverV0{TaskStore: taskStore}).agentTaskV0(
		context.Background(),
		orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:   bridged.Run.RunID,
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef: bridged.Tasks[0].TaskID,
		},
		"programacion",
	)
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if agentTask.TaskRef != bridged.Tasks[0].TaskID ||
		len(agentTask.WriteSet) != 1 ||
		agentTask.WriteSet[0] != "modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go" {
		t.Fatalf("agent_task=%+v", agentTask)
	}
	if bridged.Continue.RunRef != bridged.Run.RunID ||
		len(bridged.Continue.WaitAgentRefs) != 1 ||
		bridged.Continue.WaitAgentRefs[0] != bridged.WaitAgentRefs[0] {
		t.Fatalf("continue=%+v wait=%v", bridged.Continue, bridged.WaitAgentRefs)
	}

	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		bridged.Continue,
		ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(continued.StartedAgents, bridged.WaitAgentRefs[0]) {
		t.Fatalf("continued=%+v wait_agent=%s", continued, bridged.WaitAgentRefs[0])
	}
	planRef := "operational-director-plan-director-decisions-" + bridged.Run.RunID
	planState, err := planStore.LoadOperationalDirectorPlanStateV0(context.Background(), bridged.Run.RunID, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := codexStackPlanStateStepForTestV0(t, planState, "step-wait-subagents")
	if planState.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(planState.RequiredTestRefs, storedTasks[0].RequiredTests[0]) {
		t.Fatalf("plan_state=%+v wait_step=%+v", planState, waitStep)
	}
}

func TestPrepareAutoprogrammingRunV0BloqueaLegacySinOptInV0(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:       request,
		OccurredAt:    "2026-06-27T10:00:00Z",
		CorrelationID: "corr-autoprogramming-legacy-optin-required-001",
		RequestedBy:   "orquesta-test",
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted ||
		len(bridged.Issues) != 1 ||
		bridged.Issues[0].Code != "autoprogramming_legacy_director_loop_opt_in_required" ||
		bridged.Issues[0].Field != "goal_migration" ||
		len(bridged.Tasks) != 0 ||
		len(bridged.WaitAgentRefs) != 0 ||
		bridged.Run.RunID != "" ||
		bridged.Continue.RunRef != "" {
		t.Fatalf("bridged=%+v", bridged)
	}
	if _, err := runStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run legacy no debe persistirse sin opt-in, err=%v", err)
	}
}

func TestPrepareAutoprogrammingRunV0NoPersisteRequestInvalida(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequiredTests = nil

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted || len(bridged.Issues) == 0 {
		t.Fatalf("bridged=%+v", bridged)
	}
	if _, err := runStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe persistirse, err=%v", err)
	}
}

func TestPrepareAutoprogrammingRunV0BackendGoalCompletoActivaGoalFirstPorComposicion(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-default-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-default-001"
	request.Tasks[0].ContextRefs = nil

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
		GoalFirstRunMarkerStore:    goalStates,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if !bridged.Accepted ||
		bridged.Work.GoalMigration.Status != orquestaautoprogramming.AutoprogrammingGoalMigrationGoalReadyV0 ||
		len(bridged.Work.GoalSpecs) != 1 ||
		len(bridged.Tasks) != 0 ||
		len(bridged.WaitAgentRefs) != 0 ||
		strings.TrimSpace(bridged.Continue.RunRef) != "" ||
		len(bridged.GoalStates) != 1 ||
		len(bridged.GoalReceipts) != 1 ||
		len(launcher.specs) != 1 {
		t.Fatalf("bridged=%+v launcher=%+v", bridged, launcher.specs)
	}
	run, err := runStore.LoadRunV0(context.Background(), request.RequestRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
		t.Fatalf("run goal-first contiene loop legacy: %+v", run)
	}
	marker, err := goalStates.LoadGoalWorkRunMarkerV0(context.Background(), request.RequestRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if marker.RunRef != request.RequestRef ||
		marker.GoalRef != bridged.GoalState.GoalRef ||
		marker.Status != orquestagoal.GoalStatusRunningV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(marker.EvidenceRefs, "evidence-ref-autoprogramming-goal-first-run-marker-v0") {
		t.Fatalf("marker=%+v state=%+v", marker, bridged.GoalState)
	}
}

func TestAutoprogrammingBridgeRequestWithGoalFirstBackendMarkersV0RespetaLegacyRequired(t *testing.T) {
	request := AutoprogrammingBridgeRequestV0{Request: autoprogrammingBridgeRequestForTestV0()}
	request.Request.Tasks[0].ContextRefs = []string{"goal_migration:legacy-required"}
	got := autoprogrammingBridgeRequestWithGoalFirstBackendMarkersV0(
		request,
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			GoalLauncher:               &goalFirstQueueLauncherForTestV0{},
			GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
			GoalObserver:               &goalFirstQueueObserverForTestV0{},
			GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:             newGoalFirstQueueStateStoreForTestV0(),
		},
	)
	refs := got.Request.Tasks[0].ContextRefs
	if autoprogrammingBridgeStringInSetForTestV0(refs, "goal_migration:goal-first") ||
		!autoprogrammingBridgeStringInSetForTestV0(refs, "goal_migration:legacy-required") {
		t.Fatalf("context_refs=%v", refs)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyConBackendParcialNoLanzaNiCaeALegacy(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-backend-parcial-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-backend-parcial-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalStateStore:             goalStates,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted ||
		len(bridged.Issues) != 1 ||
		bridged.Issues[0].Code != "autoprogramming_goal_backend_incomplete" ||
		bridged.Issues[0].Field != "ports.goal_observer" {
		t.Fatalf("bridged=%+v", bridged)
	}
	if len(launcher.specs) != 0 || len(goalStates.states) != 0 {
		t.Fatalf("backend parcial no debe lanzar goal: specs=%+v states=%+v", launcher.specs, goalStates.states)
	}
	if _, err := runStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe persistirse, err=%v", err)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyMultiGoalLanzaBatchSinLegacy(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-multi-001"
	request.Tasks = []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
		{
			TaskRef:       "source-task-ref-autoprogramming-goal-multi-api-001",
			Area:          "api",
			RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestGoalMultiAPI"},
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
			},
		},
		{
			TaskRef:       "source-task-ref-autoprogramming-goal-multi-web-001",
			Area:          "web",
			RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestGoalMultiWeb"},
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
			},
		},
	}
	request.WriteSet = []string{
		"modulos/orquesta-app-codex-stack/api/goal_multi.go",
		"modulos/orquesta-app-codex-stack/web/goal_multi.go",
	}
	request.MaxTaskRefs = 2
	request.MaxAreas = 2
	request.MaxWriteSetEntries = 2
	request = withAutoprogrammingAttestationForTestV0(request)

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
		GoalFirstRunMarkerStore:    goalStates,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if !bridged.Accepted ||
		len(bridged.Issues) != 0 ||
		len(bridged.Work.GoalSpecs) != 2 ||
		bridged.Run.RunID != "run-autoprogramming-goal-multi-001-goal-01" ||
		len(bridged.Tasks) != 0 ||
		len(bridged.WaitAgentRefs) != 0 ||
		strings.TrimSpace(bridged.Continue.RunRef) != "" ||
		len(bridged.GoalStates) != 2 ||
		len(bridged.GoalReceipts) != 2 ||
		bridged.GoalReceipt != nil ||
		strings.TrimSpace(bridged.GoalState.GoalRef) != "" {
		t.Fatalf("bridged=%+v", bridged)
	}
	wantRunRefs := []string{
		"run-autoprogramming-goal-multi-001-goal-01",
		"run-autoprogramming-goal-multi-001-goal-02",
	}
	for i, runRef := range wantRunRefs {
		if bridged.Work.GoalSpecs[i].RunRef != runRef ||
			bridged.GoalStates[i].RunRef != runRef {
			t.Fatalf("goal %d spec/state run_ref=%s/%s want %s", i, bridged.Work.GoalSpecs[i].RunRef, bridged.GoalStates[i].RunRef, runRef)
		}
		run, err := runStore.LoadRunV0(context.Background(), runRef)
		if err != nil {
			t.Fatalf("LoadRunV0 %s: %v", runRef, err)
		}
		if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
			t.Fatalf("run goal-first no debe materializar loop legacy: %+v", run)
		}
		if _, ok := goalStates.states[runRef]; !ok {
			t.Fatalf("goal state %s no persistido: %+v", runRef, goalStates.states)
		}
		marker, err := goalStates.LoadGoalWorkRunMarkerV0(context.Background(), runRef)
		if err != nil {
			t.Fatalf("LoadGoalWorkRunMarkerV0 %s: %v", runRef, err)
		}
		if marker.RunRef != runRef ||
			marker.GoalRef != bridged.GoalStates[i].GoalRef ||
			marker.Status != orquestagoal.GoalStatusRunningV0 {
			t.Fatalf("marker %d=%+v state=%+v", i, marker, bridged.GoalStates[i])
		}
	}
	if len(launcher.specs) != 2 ||
		launcher.specs[0].RunRef != wantRunRefs[0] ||
		launcher.specs[1].RunRef != wantRunRefs[1] ||
		len(goalStates.states) != 2 ||
		len(goalStates.markers) != 2 {
		t.Fatalf("multi-goal no lanzo batch correcto: specs=%+v states=%+v markers=%+v", launcher.specs, goalStates.states, goalStates.markers)
	}
	if _, err := runStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("multi-goal no debe materializar run agregada legacy, err=%v", err)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyMultiGoalStateStoreFallaSinRelanzar(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	baseGoalStates := newGoalFirstQueueStateStoreForTestV0()
	goalStates := &autoprogrammingBridgeFailingGoalStateStoreForTestV0{
		delegate: baseGoalStates,
		failOn:   2,
	}
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-state-fails-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-state-api-001"
	request.Tasks[0].Area = "api"
	request.Tasks[0].RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestGoalStateAPI"}
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	second := request.Tasks[0]
	second.TaskRef = "source-task-ref-autoprogramming-goal-state-web-001"
	second.Area = "web"
	second.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestGoalStateWeb"}
	request.Tasks = append(request.Tasks, second)
	request.WriteSet = []string{
		"modulos/orquesta-app-codex-stack/api/state_fails.go",
		"modulos/orquesta-app-codex-stack/web/state_fails.go",
	}
	request.MaxTaskRefs = 2
	request.MaxAreas = 2
	request.MaxWriteSetEntries = 2
	request = withAutoprogrammingAttestationForTestV0(request)
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
	}

	bridged, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	wantRunRefs := []string{
		"run-autoprogramming-goal-state-fails-001-goal-01",
		"run-autoprogramming-goal-state-fails-001-goal-02",
	}
	if bridged.Accepted ||
		len(bridged.Issues) != 1 ||
		bridged.Issues[0].Code != "autoprogramming_goal_state_save_failed" ||
		len(bridged.GoalStates) != 1 ||
		bridged.GoalStates[0].RunRef != wantRunRefs[0] ||
		len(launcher.specs) != 2 ||
		len(baseGoalStates.states) != 1 {
		t.Fatalf("bridged=%+v specs=%+v states=%+v", bridged, launcher.specs, baseGoalStates.states)
	}
	for _, runRef := range wantRunRefs {
		run, err := runStore.LoadRunV0(ctx, runRef)
		if err != nil {
			t.Fatalf("LoadRunV0 %s: %v", runRef, err)
		}
		if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
			t.Fatalf("state store failure no debe materializar legacy: %+v", run)
		}
	}

	retry, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, ports)
	if err != nil {
		t.Fatalf("retry PrepareAutoprogrammingRunV0: %v", err)
	}
	if retry.Accepted ||
		len(retry.Issues) != 1 ||
		retry.Issues[0].Code != "autoprogramming_goal_state_unavailable_for_existing_run" ||
		len(launcher.specs) != 2 {
		t.Fatalf("retry=%+v specs=%+v", retry, launcher.specs)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyLaunchFailedPersisteStateBloqueadoV0(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{
		receipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			EvidenceRefs:  []string{"evidence-ref-goal-launch-socket-missing"},
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: "codex_app_server_control_socket_missing",
			}},
		},
		err: errors.New("failed to connect to socket at /home/user/.codex/app-server-control/app-server-control.sock"),
	}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-launch-fails-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-launch-fails-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
		GoalFirstRunMarkerStore:    goalStates,
	}

	bridged, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted ||
		bridged.Run.RunID != request.RequestRef ||
		len(bridged.Issues) != 1 ||
		bridged.Issues[0].Code != "autoprogramming_goal_launch_failed" ||
		bridged.Issues[0].Field != "goal_launcher" ||
		!strings.Contains(bridged.Issues[0].Message, "codex_app_server_control_socket_missing") ||
		strings.Contains(bridged.Issues[0].Message, "/home/user") ||
		len(bridged.GoalStates) != 1 ||
		len(launcher.specs) != 1 {
		t.Fatalf("bridged=%+v specs=%+v", bridged, launcher.specs)
	}
	run, err := runStore.LoadRunV0(ctx, request.RequestRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.Tasks) != 0 || len(run.FunctionContracts) != 0 {
		t.Fatalf("launch failed no debe materializar legacy: %+v", run)
	}
	state, err := goalStates.LoadGoalWorkStateV0(ctx, request.RequestRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusInvalidV0 ||
		state.RunRef != request.RequestRef ||
		state.LaunchReceipt.Issues[0].Code != "codex_app_server_control_socket_missing" ||
		!autoprogrammingBridgeStringInSetForTestV0(state.EvidenceRefs, "evidence-ref-autoprogramming-goal-first-launch-failed-state-v0") ||
		!autoprogrammingBridgeStringInSetForTestV0(state.EvidenceRefs, "evidence-ref-goal-launch-socket-missing") {
		t.Fatalf("state reparable inesperado=%+v", state)
	}
	marker, err := goalStates.LoadGoalWorkRunMarkerV0(ctx, request.RequestRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if marker.Status != orquestagoal.GoalStatusInvalidV0 ||
		marker.RunRef != request.RequestRef ||
		!autoprogrammingBridgeStringInSetForTestV0(marker.EvidenceRefs, "evidence-ref-autoprogramming-goal-first-run-marker-v0") ||
		!autoprogrammingBridgeStringInSetForTestV0(marker.EvidenceRefs, "evidence-ref-goal-launch-socket-missing") {
		t.Fatalf("marker reparable inesperado=%+v", marker)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyRunExistenteConSpecDistintoNoReutiliza(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-spec-mismatch-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-spec-mismatch-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
	}

	first, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, ports)
	if err != nil {
		t.Fatalf("first PrepareAutoprogrammingRunV0: %v", err)
	}
	if !first.Accepted || len(first.GoalStates) != 1 || len(launcher.specs) != 1 {
		t.Fatalf("first=%+v specs=%+v", first, launcher.specs)
	}

	changed := request
	changed.WriteSet = []string{"modulos/orquesta-app-codex-stack/changed/spec_mismatch.go"}
	changed = withAutoprogrammingAttestationForTestV0(changed)
	second, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: changed,
	}, ports)
	if err != nil {
		t.Fatalf("second PrepareAutoprogrammingRunV0: %v", err)
	}
	if second.Accepted ||
		len(second.Issues) != 1 ||
		second.Issues[0].Code != "autoprogramming_goal_state_spec_mismatch" ||
		second.Issues[0].Field != "goal_state.spec" ||
		len(launcher.specs) != 1 {
		t.Fatalf("second=%+v specs=%+v", second, launcher.specs)
	}
	stored, err := goalStates.LoadGoalWorkStateV0(ctx, first.Run.RunID)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if stored.Spec.WriteSet[0].Path == changed.WriteSet[0] {
		t.Fatalf("state existente no debe mutar: %+v", stored.Spec.WriteSet)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyRunExistenteSinGoalStateNoRelanzaGoal(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-existing-missing-state-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-existing-missing-state-001"
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
	existing := autoprogrammingBridgeGoalRunV0(
		AutoprogrammingBridgeRequestV0{Request: request},
		work.Work,
	)
	if err := runStore.SaveRunV0(ctx, existing); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	bridged, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted ||
		bridged.Run.RunID != existing.RunID ||
		len(bridged.Issues) != 1 ||
		bridged.Issues[0].Code != "autoprogramming_goal_state_unavailable_for_existing_run" ||
		bridged.Issues[0].Field != "goal_state" {
		t.Fatalf("bridged=%+v", bridged)
	}
	if len(launcher.specs) != 0 || len(goalStates.states) != 0 {
		t.Fatalf("run existente sin goal_state no debe relanzar: specs=%+v states=%+v", launcher.specs, goalStates.states)
	}
	stored, err := runStore.LoadRunV0(ctx, existing.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(stored.Tasks) != 0 || len(stored.FunctionContracts) != 0 {
		t.Fatalf("no debe materializar loop legacy: %+v", stored)
	}
}

func TestPrepareAutoprogrammingRunV0GoalReadyRunExistenteConStateSinMarkerReparaMarkerSinRelanzarV0(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-goal-existing-state-no-marker-001"
	request.Tasks[0].TaskRef = "source-task-ref-autoprogramming-goal-existing-state-no-marker-001"
	request.Tasks[0].ContextRefs = []string{
		"goal_migration:goal-first",
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	basePorts := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		DirectorTaskStore:          taskStore,
		GoalLauncher:               launcher,
		GoalRequiredTestSpecBinder: independentSpecBinderForStackTestV0{},
		GoalObserver:               &goalFirstQueueObserverForTestV0{},
		GoalClosureValidator:       orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:             goalStates,
	}
	first, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, basePorts)
	if err != nil {
		t.Fatalf("first PrepareAutoprogrammingRunV0: %v", err)
	}
	if !first.Accepted ||
		len(first.GoalStates) != 1 ||
		len(goalStates.states) != 1 ||
		len(goalStates.markers) != 0 ||
		len(launcher.specs) != 1 {
		t.Fatalf("first=%+v states=%+v markers=%+v specs=%+v", first, goalStates.states, goalStates.markers, launcher.specs)
	}

	retryPorts := basePorts
	retryPorts.GoalFirstRunMarkerStore = goalStates
	retry, err := PrepareAutoprogrammingRunV0(ctx, AutoprogrammingBridgeRequestV0{
		Request: request,
	}, retryPorts)
	if err != nil {
		t.Fatalf("retry PrepareAutoprogrammingRunV0: %v", err)
	}
	if !retry.Accepted ||
		len(retry.Issues) != 0 ||
		len(retry.GoalStates) != 1 ||
		len(launcher.specs) != 1 {
		t.Fatalf("retry=%+v specs=%+v", retry, launcher.specs)
	}
	marker, err := goalStates.LoadGoalWorkRunMarkerV0(ctx, request.RequestRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if marker.RunRef != request.RequestRef ||
		marker.GoalRef != first.GoalState.GoalRef ||
		marker.Status != orquestagoal.GoalStatusRunningV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(marker.EvidenceRefs, "evidence-ref-autoprogramming-goal-first-run-marker-v0") {
		t.Fatalf("marker=%+v state=%+v", marker, first.GoalState)
	}
}

type autoprogrammingBridgeFailingGoalStateStoreForTestV0 struct {
	delegate *goalFirstQueueStateStoreForTestV0
	failOn   int
	saves    int
}

func (store *autoprogrammingBridgeFailingGoalStateStoreForTestV0) SaveGoalWorkStateV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	store.saves++
	if store.failOn > 0 && store.saves == store.failOn {
		return errors.New("goal state store unavailable")
	}
	return store.delegate.SaveGoalWorkStateV0(ctx, state)
}

func (store *autoprogrammingBridgeFailingGoalStateStoreForTestV0) LoadGoalWorkStateV0(
	ctx context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	return store.delegate.LoadGoalWorkStateV0(ctx, runRef)
}

func TestPrepareAutoprogrammingRunV0EsIdempotenteYNoSobrescribeRunViva(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      ledger,
		DirectorTaskStore: taskStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			autoprogrammingBridgeCapacityDispatcherForTestV0(runStore, eventSink, ledger),
			autoprogrammingBridgeAgentDispatcherForTestV0(runStore, eventSink, ledger),
		},
	}
	request := AutoprogrammingBridgeRequestV0{
		Request:                 autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:              "2026-05-22T10:10:00Z",
		CorrelationID:           "corr-autoprogramming-idempotent-001",
		RequestedBy:             "orquesta-test",
		AllowLegacyDirectorLoop: true,
		MaxBursts:               4,
		MaxCommands:             8,
	}

	first, err := PrepareAutoprogrammingRunV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 first: %v", err)
	}
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		first.Continue,
		ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(continued.StartedAgents, first.WaitAgentRefs[0]) {
		t.Fatalf("continued=%+v wait_agent=%s", continued, first.WaitAgentRefs[0])
	}

	second, err := PrepareAutoprogrammingRunV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 second: %v", err)
	}
	run, err := runStore.LoadRunV0(context.Background(), first.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, first.WaitAgentRefs[0]) {
		t.Fatalf("prepare-run repetido sobrescribio started_agents: run=%+v", run)
	}
	if second.Run.RunID != first.Run.RunID ||
		second.Continue.RunRef != first.Run.RunID ||
		second.WaitAgentRefs[0] != first.WaitAgentRefs[0] {
		t.Fatalf("second=%+v first=%+v", second, first)
	}
	changedRequest := request
	changedRequest.Request.Tasks = append([]orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0(nil), request.Request.Tasks...)
	changedRequest.Request.Tasks[0].AcceptanceCriteria = append(
		append([]string(nil), request.Request.Tasks[0].AcceptanceCriteria...),
		"criterio nuevo reparable sin bloquear reentrada",
	)
	third, err := PrepareAutoprogrammingRunV0(context.Background(), changedRequest, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 tolerante: %v", err)
	}
	if third.Run.RunID != first.Run.RunID ||
		third.Tasks[0].TaskID != first.Tasks[0].TaskID ||
		autoprogrammingBridgeStringInSetForTestV0(third.Tasks[0].AcceptanceCriteria, "criterio nuevo reparable sin bloquear reentrada") {
		t.Fatalf("third no reutiliza tarea viva: third=%+v first=%+v", third, first)
	}
}

func TestPrepareAutoprogrammingRunV0AceptaAppSpecLegacyCompatibleV0(t *testing.T) {
	existing := orquestacoreworkflow.OrchestrationRunV0{
		RunID:             "request-ref-autoprogramming-backlog-t36-codex-ack-strict-terminal-validation-e71900c9",
		ProjectRef:        "project-ref-orquesta-server",
		AppSpecRef:        "app-spec-ref-autoprogramming-e44e32f96f05",
		Tasks:             []string{"task-autoprogramming-001"},
		FunctionContracts: []string{"function-contract-ref-autoprogramming-001"},
	}
	expected := existing
	expected.AppSpecRef = "app-spec-ref-autoprogramming-e05c9dcabfdfd015a9bbc205b6ca9644a55b8629cf423a2605fa2701338c6c51"

	if err := autoprogrammingBridgeValidateExistingRunV0(existing, expected); err != nil {
		t.Fatalf("legacy compatible rechazado: %v", err)
	}

	expected.AppSpecRef = "app-spec-ref-other-product-001"
	if err := autoprogrammingBridgeValidateExistingRunV0(existing, expected); err == nil {
		t.Fatalf("app spec no autoprogramming incompatible aceptado")
	}
}
