package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackOperationalWaveFakeRuntimeV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	cfg.MaxBatchReady = 3
	cfg.MaxConcurrency = 3
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")

	runtime := &codexStackRequiredTestPendingRuntimeV0{}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)
	stack.Ports.ProgressSource = nil
	fixture := newCodexStackOperationalWaveFixtureV0("fake")

	codexStackOperationalWaveRunToCloseV0(t, ctx, stack, evidenceStore, cfg, fixture, true)
	if runtime.launchCountV0() != len(fixture.Refs) {
		t.Fatalf("launches=%d want=%d", runtime.launchCountV0(), len(fixture.Refs))
	}
}

func TestCodexStackRecursiveTreeFakeRuntimeV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	cfg.MaxBatchReady = 2
	cfg.MaxConcurrency = 2
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")

	runtime := &codexStackRequiredTestPendingRuntimeV0{}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)
	stack.Ports.ProgressSource = nil
	fixture := newCodexStackRecursiveTreeFixtureV0()

	codexStackRecursiveTreeRunToCloseV0(t, ctx, stack, evidenceStore, cfg, fixture, runtime, true)
	if runtime.launchCountV0() != len(fixture.Tasks) {
		t.Fatalf("launches=%d want=%d", runtime.launchCountV0(), len(fixture.Tasks))
	}
}

func TestCodexStackRecursiveTreeFakeRuntimeNoCierraPadreAntesDeHijosV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	cfg.MaxBatchReady = 2
	cfg.MaxConcurrency = 2
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")

	runtime := &codexStackRequiredTestPendingRuntimeV0{}
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)
	stack.Ports.DeliverySource = nil
	stack.Ports.ReviewGateSource = nil
	stack.Ports.ReviewReworkReplanSource = nil
	stack.Ports.AssessmentReplanSource = nil
	stack.Ports.ProgressSource = nil
	stack.Ports.DirectorDecisionSource = nil
	fixture := newCodexStackRecursiveTreeFixtureWithSuffixV0("no-close-parent-open-children")
	codexStackRecursiveTreeSeedV0(t, ctx, stack, fixture)

	parent := fixture.taskV0("task-recursive-parent")
	parentAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parent.TaskID)
	deliveryRef := "delivery-ref-" + parent.TaskID
	events := stackOperationalClosureRecursiveEventsForTestV0(t, fixture.RunRef, []orquestacoreworkflow.WorkflowTaskV0{parent})
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, fixture.RunRef, events); err != nil {
		t.Fatalf("AppendRunEventsV0 parent: %v", err)
	}
	if err := evidenceStore.SaveRequiredTestEvidenceV0(
		ctx,
		stackOperationalClosureRequiredTestEvidenceForTestV0(fixture.RunRef, parent.TaskID, deliveryRef),
	); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0 parent: %v", err)
	}

	run := mustLoadCodexStackRunForTestV0(t, stack, fixture.RunRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Phases = codexStackRequiredTestPhasesV0()
	run.Agents = []string{parentAgent}
	run.StartedAgents = []string{parentAgent}
	run.DeliveredAgents = []string{parentAgent}
	run.DeliveredTasks = []string{parent.TaskID}
	run.Deliveries = []string{deliveryRef}
	run.Reviews = []string{"review-request-ref-" + deliveryRef}
	run.ReviewResults = []string{
		"review-result-ref-" + deliveryRef + "#review_result:accepted#review_request:review-request-ref-" + deliveryRef + "#delivery:" + deliveryRef,
	}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	run.LastSequence = int64(len(events))
	run.LastEventID = events[len(events)-1].EventID
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0 parent delivery: %v", err)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(
		ctx,
		codexStackRecursiveTreeParentOnlyReadyToCloseStateV0(fixture, parent, deliveryRef),
	); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0 parent ready: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}

	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
		OccurredAt:                 "2026-05-22T20:41:00Z",
		CorrelationID:              "corr-recursive-tree-no-close-parent",
		WaitAgentRefs:              []string{parentAgent},
		MaxBursts:                  2,
		MaxStepsPerBurst:           4,
		MaxDispatchesPerWait:       2,
		MaxCommands:                16,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 parent ready: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	if codexStackStringInSetForTestV0(result.Run.ClosedTasks, parent.TaskID) ||
		len(result.Run.ClosedTasks) != 0 {
		t.Fatalf("no debe cerrar padre ni hijos con child_task_refs abiertos: closed=%v run=%+v", result.Run.ClosedTasks, result.Run)
	}
	if result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar run con hijos abiertos: run=%+v", result.Run)
	}
	state := codexStackOperationalWaveLoadPlanStateV0(t, ctx, stack, codexStackOperationalWaveFixtureV0{
		RunRef:  fixture.RunRef,
		PlanRef: fixture.PlanRef,
	})
	if state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		t.Fatalf("no debe cerrar plan con hijos abiertos: state=%+v", state)
	}
	if runtime.launchCountV0() != 0 {
		t.Fatalf("no debe lanzar agentes durante cierre acotado al padre: launches=%d", runtime.launchCountV0())
	}
}

func TestCodexSupervisorStackLifecycleV0AvanzaArbolRecursivoFakeSinManualPorNivelV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	cfg.MaxBatchReady = 2
	cfg.MaxConcurrency = 2
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := filepath.Join(t.TempDir(), "required-test-output")

	runtime := newFakeCodexStackRuntimeV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, runtime, evidenceStore, goCommand, outputDir)
	stack.Ports.ProgressSource = nil
	fixture := newCodexStackRecursiveTreeFixtureWithSuffixV0("supervisor-fake")
	codexStackRecursiveTreeSeedV0(t, ctx, stack, fixture)

	launchResult := codexStackRecursiveTreeSuperviseWithStackLifecycleV0(t, ctx, stack, fixture, "", 8)
	if launchResult.Last.SessionRef != fixture.RunRef ||
		launchResult.Last.Status == CodexSupervisorRuntimeFailedV0 ||
		launchResult.StopReason != CodexSupervisorStopDoneV0 {
		t.Fatalf("launchResult=%+v", launchResult)
	}
	if runtime.launchCountV0() != len(fixture.Tasks) {
		t.Fatalf("launches=%d want=%d", runtime.launchCountV0(), len(fixture.Tasks))
	}
	allAgents := codexStackRecursiveTreeAgentRefsV0(fixture)
	codexStackRecursiveTreeAssertDeliveredV0(t, stack, fixture.RunRef, allAgents)
	codexStackRecursiveTreeAssertLineageV0(t, ctx, stack, fixture)
	codexStackRecursiveTreeAssertSupervisorWaitScopesV0(t, ctx, stack, fixture)

	openCodexStackPhaseForTestV0(t, stack, fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Supervisor fake arbol recursivo: revisar entregas.")
	causals := codexStackOperationalWaveSeedAcceptedReviewsV0(
		t,
		ctx,
		stack,
		fixture.RunRef,
		allAgents,
		codexStackRecursiveTreeDeliveryByAgentV0(t, stack, fixture.RunRef, allAgents),
	)
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, codexStackRecursiveTreePlanStateV0(fixture, causals)); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	stack.Ports.DeliverySource = nil
	stack.Ports.ReviewGateSource = nil
	stack.Ports.ReviewReworkReplanSource = nil
	stack.Ports.AssessmentReplanSource = nil

	var closeResult CodexSupervisorResultV0
	var run orquestacoreworkflow.OrchestrationRunV0
	var state orquestacionnucleoapp.OperationalDirectorPlanStateV0
	for cycle := 1; cycle <= len(fixture.Tasks)+2; cycle++ {
		closeResult = codexStackRecursiveTreeSuperviseWithStackLifecycleV0(t, ctx, stack, fixture, fixture.PlanRef, 12)
		run = mustLoadCodexStackRunForTestV0(t, stack, fixture.RunRef)
		state = codexStackOperationalWaveLoadPlanStateV0(t, ctx, stack, codexStackOperationalWaveFixtureV0{
			RunRef:  fixture.RunRef,
			PlanRef: fixture.PlanRef,
		})
		if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 &&
			state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
			break
		}
	}
	if closeResult.StopReason != CodexSupervisorStopDoneV0 ||
		run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		t.Fatalf("closeResult=%+v run=%+v state=%+v", closeResult, run, state)
	}
	if len(run.ClosedTasks) != len(fixture.Tasks) {
		t.Fatalf("closed_tasks=%v want=%d", run.ClosedTasks, len(fixture.Tasks))
	}
	if runtime.launchCountV0() != len(fixture.Tasks) {
		t.Fatalf("supervisor relanzo agentes durante cierre: launches=%d want=%d", runtime.launchCountV0(), len(fixture.Tasks))
	}
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, fixture.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	codexStackRecursiveTreeAssertEvidenceV0(t, fixture, causals, evidence)
}

func TestCodexStackRealRecursiveTreeOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_RECURSIVE_TREE_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_REAL_RECURSIVE_TREE_SMOKE=1")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_RECURSIVE_TREE_CONFIRM")) != "1" ||
		strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_RECURSIVE_TREE_CODEX_EXECUTION_CONFIRMED")) != "1" {
		t.Fatal("doble confirmacion requerida para ejecutar arbol recursivo Codex real")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "" ||
		strings.TrimSpace(os.Getenv("OPES_BASE_URL")) != "" {
		t.Fatal("este smoke no debe cablear OPES")
	}

	cfg := codexStackRealSmokeConfigForTestV0(t)
	if cfg.Timeout < 1500*time.Second {
		cfg.Timeout = 1500 * time.Second
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REASONING_EFFORT")) == "" {
		cfg.ReasoningEffort = "high"
	}
	if strings.TrimSpace(cfg.ReasoningEffort) == "xhigh" {
		t.Fatal("este smoke recursivo no usa xhigh por defecto")
	}
	cfg.MaxBatchReady = 2
	cfg.MaxConcurrency = 2
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	codexStackRealSmokeWriteProjectContextV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", codexStackRequiredTestOutputDirV0(t))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, processRuntime, evidenceStore, goCommand, outputDir)
	stack.Ports.ProgressSource = nil
	defer codexStackRealSmokeStopAllProcessesV0(
		t,
		processRuntime,
		stack.Stores.ProcessRegistry.(*orquestaagentprocessregistrymemory.InMemoryAgentProcessRegistryV0),
		stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0),
	)

	codexStackRecursiveTreeRunToCloseV0(
		t,
		ctx,
		stack,
		evidenceStore,
		cfg,
		newCodexStackRecursiveTreeFixtureWithSuffixV0("real"),
		nil,
		false,
	)
}

type codexStackLaunchCounterForTestV0 interface {
	launchCountV0() int
}

func codexStackRecursiveTreeRunToCloseV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	evidenceStore orquestacionnucleoapp.RequiredTestEvidenceStorePortV0,
	cfg codexStackRealSmokeConfigV0,
	fixture codexStackRecursiveTreeFixtureV0,
	launchCounter codexStackLaunchCounterForTestV0,
	manualAck bool,
) {
	t.Helper()
	codexStackRecursiveTreeSeedV0(t, ctx, stack, fixture)

	parent := fixture.taskV0("task-recursive-parent")
	childA := fixture.taskV0("task-recursive-child-a")
	childB := fixture.taskV0("task-recursive-child-b")
	grandA := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-a1"),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-a2"),
	}
	grandB := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-b1"),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-b2"),
	}
	rootAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parent.TaskID)
	childAgents := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskID),
	}

	codexStackRecursiveTreeContinueLaunchV0(t, ctx, stack, launchCounter, fixture.RunRef, appDirectorWaitFilterForRecursiveTreeV0{
		WaveRef:   parent.WaveRef,
		CohortRef: parent.CohortRef,
	}, []string{rootAgent})
	codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, rootAgent, cfg, manualAck)

	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, parent.TaskID, "", childAgents, childAgents)
	codexStackRecursiveTreeContinueLaunchV0(t, ctx, stack, launchCounter, fixture.RunRef, appDirectorWaitFilterForRecursiveTreeV0{
		ParentTaskRef: parent.TaskID,
	}, childAgents)
	codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, childAgents[0], cfg, manualAck)

	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, childA.TaskID, childA.WaveRef, grandA, grandA)
	codexStackRecursiveTreeContinueLaunchV0(t, ctx, stack, launchCounter, fixture.RunRef, appDirectorWaitFilterForRecursiveTreeV0{
		ParentTaskRef: childA.TaskID,
		WaveRef:       childA.WaveRef,
	}, grandA)
	for _, agentRef := range grandA {
		codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, agentRef, cfg, manualAck)
	}

	codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, childAgents[1], cfg, manualAck)
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, childB.TaskID, childB.WaveRef, grandB, grandB)
	codexStackRecursiveTreeContinueLaunchV0(t, ctx, stack, launchCounter, fixture.RunRef, appDirectorWaitFilterForRecursiveTreeV0{
		ParentTaskRef: childB.TaskID,
		WaveRef:       childB.WaveRef,
	}, grandB)
	for _, agentRef := range grandB {
		codexStackRecursiveTreeDrainAgentV0(t, ctx, stack, fixture.RunRef, agentRef, cfg, manualAck)
	}

	allAgents := codexStackRecursiveTreeAgentRefsV0(fixture)
	codexStackRecursiveTreeAssertDeliveredV0(t, stack, fixture.RunRef, allAgents)
	stack.Ports.DeliverySource = nil
	openCodexStackPhaseForTestV0(t, stack, fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Smoke arbol recursivo: revisar entregas.")
	causals := codexStackOperationalWaveSeedAcceptedReviewsV0(
		t,
		ctx,
		stack,
		fixture.RunRef,
		allAgents,
		codexStackRecursiveTreeDeliveryByAgentV0(t, stack, fixture.RunRef, allAgents),
	)
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, codexStackRecursiveTreePlanStateV0(fixture, causals)); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	stack.Ports.ReviewGateSource = nil
	stack.Ports.ReviewReworkReplanSource = nil
	stack.Ports.AssessmentReplanSource = nil

	run, state := codexStackRecursiveTreeCloseV0(t, ctx, stack, fixture, cfg)
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		t.Fatalf("arbol no cerro: run_status=%s state=%+v run=%+v", run.Status, state, run)
	}
	if len(run.ClosedTasks) != len(fixture.Tasks) {
		t.Fatalf("closed_tasks=%v want=%d", run.ClosedTasks, len(fixture.Tasks))
	}
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, fixture.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	codexStackRecursiveTreeAssertEvidenceV0(t, fixture, causals, evidence)
}

func TestCodexStackRealOperationalWaveOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_SMOKE=1")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CONFIRM")) != "1" ||
		strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_OPERATIONAL_WAVE_CODEX_EXECUTION_CONFIRMED")) != "1" {
		t.Fatal("doble confirmacion requerida para ejecutar ola Codex real")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "" ||
		strings.TrimSpace(os.Getenv("OPES_BASE_URL")) != "" {
		t.Fatal("este smoke no debe cablear OPES")
	}

	cfg := codexStackRealSmokeConfigForTestV0(t)
	if cfg.Timeout < 720*time.Second {
		cfg.Timeout = 720 * time.Second
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REASONING_EFFORT")) == "" {
		cfg.ReasoningEffort = "high"
	}
	if strings.TrimSpace(cfg.ReasoningEffort) == "xhigh" {
		t.Fatal("este smoke de ola/cohorte no usa xhigh")
	}
	cfg.MaxBatchReady = 3
	cfg.MaxConcurrency = 3
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	codexStackRealSmokeWriteProjectContextV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", codexStackRequiredTestOutputDirV0(t))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, processRuntime, evidenceStore, goCommand, outputDir)
	stack.Ports.ProgressSource = nil
	defer codexStackRealSmokeStopAllProcessesV0(
		t,
		processRuntime,
		stack.Stores.ProcessRegistry.(*orquestaagentprocessregistrymemory.InMemoryAgentProcessRegistryV0),
		stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0),
	)

	codexStackOperationalWaveRunToCloseV0(
		t,
		ctx,
		stack,
		evidenceStore,
		cfg,
		newCodexStackOperationalWaveFixtureV0("real"),
		false,
	)
}

type codexStackOperationalWaveFixtureV0 struct {
	RunRef    string
	PlanRef   string
	WaveRef   string
	CohortRef string
	Refs      []codexStackRequiredTestRefsV0
}

func newCodexStackOperationalWaveFixtureV0(suffix string) codexStackOperationalWaveFixtureV0 {
	short := "f"
	if suffix == "real" {
		short = "r"
	}
	runRef := "run-wave-" + short + "-001"
	planRef := "plan-wave-" + short + "-001"
	refs := make([]codexStackRequiredTestRefsV0, 0, 3)
	for index := 1; index <= 3; index++ {
		taskRef := fmt.Sprintf("task-wave-%s-%d", short, index)
		refs = append(refs, (codexStackRequiredTestRefsV0{
			RunRef:  runRef,
			PlanRef: planRef,
			TaskRef: taskRef,
		}).withDefaultsV0())
	}
	return codexStackOperationalWaveFixtureV0{
		RunRef:    runRef,
		PlanRef:   planRef,
		WaveRef:   "wave-wide-001",
		CohortRef: "cohort-wide-001",
		Refs:      refs,
	}
}

func codexStackOperationalWaveRunToCloseV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	evidenceStore orquestacionnucleoapp.RequiredTestEvidenceStorePortV0,
	cfg codexStackRealSmokeConfigV0,
	fixture codexStackOperationalWaveFixtureV0,
	manualAck bool,
) {
	t.Helper()
	codexStackOperationalWaveSeedV0(t, ctx, stack, fixture)
	agentRefs := codexStackOperationalWaveAgentRefsV0(fixture)
	codexStackOperationalWaveAssertCohortDerivesWaitRefsV0(t, ctx, stack, fixture, agentRefs)

	started, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:               fixture.RunRef,
		OccurredAt:           "2026-05-22T19:00:00Z",
		CorrelationID:        "corr-operational-wave-start",
		WaitWaveRef:          fixture.WaveRef,
		WaitCohortRef:        fixture.CohortRef,
		MaxBursts:            6,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		MaxCommands:          16,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 start: %v %s\n%s", err, codexStackRequiredTestErrorDetailsV0(err), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	for _, agentRef := range agentRefs {
		if !codexStackStringInSetForTestV0(started.StartedAgents, agentRef) {
			t.Fatalf("agente de cohorte no arrancado: missing=%s started=%v", agentRef, started.StartedAgents)
		}
	}
	if manualAck {
		for _, descriptor := range codexStackOperationalWaveDescriptorsForAgentsV0(t, stack, agentRefs) {
			if err := writeCodexStackAckForDescriptorV0(t, descriptor); err != nil {
				t.Fatalf("write ack agent=%s: %v", descriptor.AgentRef, err)
			}
		}
	}

	causals := codexStackOperationalWaveDrainReviewV0(t, ctx, stack, fixture.RunRef, agentRefs, cfg.RuntimeWorkDir, cfg.Timeout)
	if len(causals) != len(fixture.Refs) {
		t.Fatalf("reviews causales=%d want=%d causals=%+v", len(causals), len(fixture.Refs), causals)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, codexStackOperationalWavePlanStateV0(fixture, causals)); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v %s", err, codexStackRequiredTestErrorDetailsV0(err))
	}
	stack.Ports.ReviewGateSource = nil
	stack.Ports.ReviewReworkReplanSource = nil
	stack.Ports.AssessmentReplanSource = nil

	var run orquestacoreworkflow.OrchestrationRunV0
	var state orquestacionnucleoapp.OperationalDirectorPlanStateV0
	for cycle := 1; cycle <= 12; cycle++ {
		result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OperationalDirectorPlanRef: fixture.PlanRef,
			OccurredAt:                 fmt.Sprintf("2026-05-22T19:%02d:00Z", cycle),
			CorrelationID:              fmt.Sprintf("corr-operational-wave-close-%03d", cycle),
			MaxBursts:                  2,
			MaxStepsPerBurst:           4,
			MaxDispatchesPerWait:       2,
			MaxCommands:                24,
			MaxOutboxPerCycle:          12,
			MaxExternalWaits:           1,
		}, stack.Ports)
		if err != nil {
			t.Fatalf("ContinueAppDirectorV0 close ciclo=%d: %v %s\n%s", cycle, err, codexStackRequiredTestErrorDetailsV0(err), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
		}
		run = result.Run
		state = codexStackOperationalWaveLoadPlanStateV0(t, ctx, stack, fixture)
		if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 &&
			state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
			break
		}
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		t.Fatalf("ola no cerro: run_status=%s state=%+v run=%+v", run.Status, state, run)
	}
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if len(replanStep.RequiredTestEvidenceRefs) < len(fixture.Refs) {
		t.Fatalf("evidencias insuficientes: refs=%v state=%+v", replanStep.RequiredTestEvidenceRefs, state)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, fixture.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	codexStackOperationalWaveAssertEvidenceV0(t, fixture, causals, evidence)
}

func codexStackOperationalWaveAssertCohortDerivesWaitRefsV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackOperationalWaveFixtureV0,
	agentRefs []string,
) {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, fixture.RunRef)
	derived, err := orquestacionnucleoapp.WorkflowTaskWaitAgentRefsV0(
		ctx,
		stack.Stores.TaskStore,
		run,
		orquestacionnucleoapp.WorkflowTaskWaitFilterV0{
			WaveRef:   fixture.WaveRef,
			CohortRef: fixture.CohortRef,
		},
	)
	if err != nil {
		t.Fatalf("WorkflowTaskWaitAgentRefsV0: %v", err)
	}
	if len(derived) != len(agentRefs) {
		t.Fatalf("wait refs derivadas=%v want=%v", derived, agentRefs)
	}
	for _, agentRef := range agentRefs {
		if !codexStackStringInSetForTestV0(derived, agentRef) {
			t.Fatalf("wait refs derivadas no contienen %s: %v", agentRef, derived)
		}
	}
	_, err = (orquestacionnucleoapp.WorkflowTaskCandidateProviderV0{
		TaskStore:       stack.Stores.TaskStore,
		RequestedBy:     "orquesta-app-stack-operational-wave-test",
		DefaultCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}).BuildSchedulerCandidatesV0(ctx, orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           run,
		StepNumber:    1,
		MaxSteps:      4,
		OccurredAt:    "2026-05-22T19:00:00Z",
		CorrelationID: "corr-operational-wave-candidate-debug",
	})
	if err != nil {
		var orchErr orquestacionnucleoapp.ErrorV0
		if errors.As(err, &orchErr) {
			t.Fatalf("WorkflowTaskCandidateProviderV0: code=%s field=%s message=%s", orchErr.Code, orchErr.Field, orchErr.Message)
		}
		t.Fatalf("WorkflowTaskCandidateProviderV0: %v", err)
	}
}

func codexStackOperationalWaveSeedV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackOperationalWaveFixtureV0,
) {
	t.Helper()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, codexStackOperationalWavePendingRunV0(fixture)); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	for index, refs := range fixture.Refs {
		task := codexStackOperationalWaveTaskV0(fixture, refs, index+1)
		if err := taskWriter.SaveWorkflowTaskV0(ctx, task); err != nil {
			t.Fatalf("SaveWorkflowTaskV0 %s: %v", refs.TaskRef, err)
		}
	}
}

func codexStackOperationalWavePendingRunV0(
	fixture codexStackOperationalWaveFixtureV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	run := fixture.Refs[0].pendingRunV0()
	run.RunID = fixture.RunRef
	run.Tasks = codexStackOperationalWaveTaskRefsV0(fixture)
	run.Agents = nil
	run.StartedAgents = nil
	run.DeliveredAgents = nil
	run.Deliveries = nil
	run.DeliveredTasks = nil
	run.Reviews = nil
	run.ReviewResults = nil
	run.AcceptedReviews = nil
	return run
}

func codexStackOperationalWaveTaskV0(
	fixture codexStackOperationalWaveFixtureV0,
	refs codexStackRequiredTestRefsV0,
	index int,
) orquestacoreworkflow.WorkflowTaskV0 {
	task := refs.workflowTaskV0()
	task.WorkProfileKind = orquestacoreworkflow.WorkProfileDocumentationV0
	task.Title = fmt.Sprintf("Documento operacional de cohorte %02d", index)
	task.Summary = "Microtarea de ola amplia con cierre causal y tests requeridos."
	task.WriteSet = []string{fmt.Sprintf("docs/w%d.md", index)}
	task.AcceptanceCriteria = []string{
		"ACK de agente registrado dentro de WaitAgentRefs",
		"review aceptada causal",
		"operational_director.required_tests passed",
	}
	task.RequiredTests = []string{"go test ./..."}
	task.WaveRef = fixture.WaveRef
	task.CohortRef = fixture.CohortRef
	return task
}

func codexStackOperationalWaveDrainReviewV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	agentRefs []string,
	runtimeWorkDir string,
	timeout time.Duration,
) map[string]codexStackRequiredTestCausalReviewRefsV0 {
	t.Helper()
	deliveryByAgent := codexStackOperationalWaveDrainUntilDeliveriesV0(t, ctx, stack, runRef, agentRefs, runtimeWorkDir, timeout)
	stack.Ports.DeliverySource = nil
	openCodexStackPhaseForTestV0(t, stack, runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0, "Smoke ola/cohorte: revisar entregas.")
	return codexStackOperationalWaveSeedAcceptedReviewsV0(t, ctx, stack, runRef, agentRefs, deliveryByAgent)
}

func codexStackOperationalWaveDrainUntilDeliveriesV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	agentRefs []string,
	runtimeWorkDir string,
	timeout time.Duration,
) map[string]string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	deliveries := map[string]string{}
	for cycle := 1; cycle <= 36 && time.Now().Before(deadline); cycle++ {
		if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
			RunRef:               runRef,
			CorrelationID:        fmt.Sprintf("corr-operational-wave-delivery-%03d", cycle),
			WaitAgentRefs:        agentRefs,
			MaxBursts:            6,
			MaxStepsPerBurst:     10,
			MaxDispatchesPerWait: 4,
			MaxCommands:          24,
			MaxOutboxPerCycle:    12,
			MaxExternalWaits:     1,
		}); err != nil {
			t.Fatalf("DrainRunV0 delivery ciclo=%d: %v\n%s", cycle, err, codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
		}
		run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
		for _, descriptor := range codexStackOperationalWaveDescriptorsForAgentsV0(t, stack, agentRefs) {
			ack, ok := codexStackRealSmokeCompletedAckV0(descriptor)
			if ok && codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, ack.AckRef) {
				deliveries[descriptor.AgentRef] = ack.AckRef
			}
		}
		if len(deliveries) == len(agentRefs) {
			return deliveries
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("sin entregas completas: deliveries=%v descriptors=%v run=%+v\n%s",
		deliveries,
		codexStackRealSmokeDescriptorAgentsV0(codexStackOperationalWaveDescriptorsForAgentsV0(t, stack, agentRefs)),
		mustLoadCodexStackRunForTestV0(t, stack, runRef),
		codexStackRealSmokeDiagnosticsV0(runtimeWorkDir),
	)
	return nil
}

func codexStackOperationalWaveSeedAcceptedReviewsV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	agentRefs []string,
	deliveryByAgent map[string]string,
) map[string]codexStackRequiredTestCausalReviewRefsV0 {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	events := []orquestacoreworkflow.OrchestrationEventV0{}
	causals := map[string]codexStackRequiredTestCausalReviewRefsV0{}
	nextSequence := run.LastSequence + 1
	for index, agentRef := range agentRefs {
		deliveryRef := strings.TrimSpace(deliveryByAgent[agentRef])
		if deliveryRef == "" {
			t.Fatalf("sin delivery para review agent=%s deliveries=%v", agentRef, deliveryByAgent)
		}
		reviewRequestRef := fmt.Sprintf("review-request-wave-%03d", index+1)
		reviewResultRef := fmt.Sprintf("review-result-wave-%03d", index+1)
		acceptedReviewRef := fmt.Sprintf("accepted-review-wave-%03d", index+1)
		requested, err := orquestacoreworkflow.NewReviewRequestedEventV0(
			codexStackOperationalWaveEventMetaV0(runRef, nextSequence, fmt.Sprintf("review-request-%03d", index+1)),
			orquestacoreworkflow.ReviewRequestedPayloadV0{
				ReviewRequestID: reviewRequestRef,
				PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				DeliveryRef:     deliveryRef,
				Summary:         "Revision requerida para entrega de cohorte.",
				EvidenceRefs:    []string{fmt.Sprintf("artifact-ref-wave-review-request-%03d", index+1)},
			},
		)
		if err != nil {
			t.Fatalf("NewReviewRequestedEventV0 agent=%s: %v", agentRef, err)
		}
		nextSequence++
		recorded, err := orquestacoreworkflow.NewReviewResultRecordedEventV0(
			codexStackOperationalWaveEventMetaV0(runRef, nextSequence, fmt.Sprintf("review-result-%03d", index+1)),
			orquestacoreworkflow.ReviewResultV0{
				ReviewResultRef: reviewResultRef,
				ReviewRequestID: reviewRequestRef,
				DeliveryRef:     deliveryRef,
				Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
				Summary:         "Revision aceptada; ejecutar tests requeridos.",
				EvidenceRefs:    []string{fmt.Sprintf("artifact-ref-wave-review-result-%03d", index+1)},
			},
		)
		if err != nil {
			t.Fatalf("NewReviewResultRecordedEventV0 agent=%s: %v", agentRef, err)
		}
		nextSequence++
		accepted, err := orquestacoreworkflow.NewReviewAcceptedEventV0(
			codexStackOperationalWaveEventMetaV0(runRef, nextSequence, fmt.Sprintf("review-accepted-%03d", index+1)),
			orquestacoreworkflow.ReviewAcceptedPayloadV0{
				AcceptedReviewRef: acceptedReviewRef,
				PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				ReviewRequestID:   reviewRequestRef,
				DeliveryRef:       deliveryRef,
				Summary:           "Revision aceptada con cadena causal completa.",
				EvidenceRefs:      []string{fmt.Sprintf("artifact-ref-wave-review-accepted-%03d", index+1)},
			},
		)
		if err != nil {
			t.Fatalf("NewReviewAcceptedEventV0 agent=%s: %v", agentRef, err)
		}
		nextSequence++
		events = append(events, requested, recorded, accepted)
		causals[agentRef] = codexStackRequiredTestCausalReviewRefsV0{
			DeliveryRef:       deliveryRef,
			ReviewRequestRef:  reviewRequestRef,
			ReviewResultRef:   reviewResultRef,
			AcceptedReviewRef: acceptedReviewRef,
		}
		run.Reviews = append(run.Reviews, reviewRequestRef)
		run.ReviewResults = append(run.ReviewResults, reviewResultRef+"#review_result:accepted#review_request:"+reviewRequestRef+"#delivery:"+deliveryRef)
		run.AcceptedReviews = append(run.AcceptedReviews, acceptedReviewRef)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, runRef, events); err != nil {
		t.Fatalf("AppendRunEventsV0 reviews: %v", err)
	}
	run.LastSequence = nextSequence - 1
	run.LastEventID = events[len(events)-1].EventID
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0 reviews: %v", err)
	}
	return causals
}

func codexStackOperationalWaveEventMetaV0(
	runRef string,
	sequence int64,
	suffix string,
) orquestacoreworkflow.OrchestrationEventMetaV0 {
	return orquestacoreworkflow.OrchestrationEventMetaV0{
		EventID:       "evt-operational-wave-" + suffix,
		RunID:         runRef,
		Sequence:      sequence,
		CorrelationID: "corr-operational-wave-seed-review",
		OccurredAt:    "2026-05-22T19:08:00Z",
	}
}

func codexStackOperationalWavePlanStateV0(
	fixture codexStackOperationalWaveFixtureV0,
	causals map[string]codexStackRequiredTestCausalReviewRefsV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	taskRefs := codexStackOperationalWaveTaskRefsV0(fixture)
	agentRefs := codexStackOperationalWaveAgentRefsV0(fixture)
	deliveryRefs := make([]string, 0, len(agentRefs))
	reviewResultRefs := make([]string, 0, len(agentRefs))
	acceptedReviewRefs := make([]string, 0, len(agentRefs))
	for _, agentRef := range agentRefs {
		causal := causals[agentRef]
		deliveryRefs = append(deliveryRefs, causal.DeliveryRef)
		reviewResultRefs = append(reviewResultRefs, causal.ReviewResultRef)
		acceptedReviewRefs = append(acceptedReviewRefs, causal.AcceptedReviewRef)
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:    orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "state-" + fixture.PlanRef,
		PlanRef:          fixture.PlanRef,
		RequestRef:       "request-" + fixture.PlanRef,
		RunRef:           fixture.RunRef,
		ProjectRef:       "project-wave-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-run-required-tests",
		ActiveWaveRef:    fixture.WaveRef,
		ActiveCohortRef:  fixture.CohortRef,
		RequiredTestRefs: []string{"go test ./..."},
		ObservedAt:       "2026-05-22T19:10:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   fixture.WaveRef,
				CohortRef: fixture.CohortRef,
				TaskRefs:  taskRefs,
				AgentRefs: agentRefs,
			},
			{
				StepID:    "step-wait-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   fixture.WaveRef,
				CohortRef: fixture.CohortRef,
				TaskRefs:  taskRefs,
				AgentRefs: agentRefs,
			},
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:            fixture.WaveRef,
				CohortRef:          fixture.CohortRef,
				TaskRefs:           taskRefs,
				AgentRefs:          agentRefs,
				DeliveryRefs:       deliveryRefs,
				ReviewResultRefs:   reviewResultRefs,
				AcceptedReviewRefs: acceptedReviewRefs,
			},
			{
				StepID:           "step-run-required-tests",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          fixture.WaveRef,
				CohortRef:        fixture.CohortRef,
				TaskRefs:         taskRefs,
				AgentRefs:        agentRefs,
				DeliveryRefs:     deliveryRefs,
				ReviewResultRefs: reviewResultRefs,
			},
			{
				StepID:    "step-replan-or-close",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   fixture.WaveRef,
				CohortRef: fixture.CohortRef,
			},
		},
	}
}

func codexStackOperationalWaveAssertEvidenceV0(
	t *testing.T,
	fixture codexStackOperationalWaveFixtureV0,
	causals map[string]codexStackRequiredTestCausalReviewRefsV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) {
	t.Helper()
	byTask := map[string]orquestacionnucleoapp.RequiredTestEvidenceV0{}
	for _, item := range evidence {
		if item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
			byTask[item.TaskRef] = item
		}
	}
	for _, refs := range fixture.Refs {
		item, ok := byTask[refs.TaskRef]
		if !ok {
			t.Fatalf("sin evidencia passed task=%s evidence=%+v", refs.TaskRef, evidence)
		}
		causal := causals[refs.AgentRef]
		if item.TestCommand != "go test ./..." ||
			item.DeliveryRef != causal.DeliveryRef ||
			item.ReviewRequestID != causal.ReviewRequestRef ||
			item.ReviewResultRef != causal.ReviewResultRef ||
			item.AcceptedReviewRef != causal.AcceptedReviewRef {
			t.Fatalf("evidencia no causal task=%s evidence=%+v causal=%+v", refs.TaskRef, item, causal)
		}
	}
}

func codexStackOperationalWaveLoadPlanStateV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackOperationalWaveFixtureV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	t.Helper()
	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	return state
}

func codexStackOperationalWaveDescriptorsForAgentsV0(
	t *testing.T,
	stack StackV0,
	agentRefs []string,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	allowed := map[string]bool{}
	for _, ref := range agentRefs {
		allowed[strings.TrimSpace(ref)] = true
	}
	out := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
	for _, descriptor := range codexStackRequiredTestReceiptDescriptorsV0(t, stack) {
		if allowed[strings.TrimSpace(descriptor.AgentRef)] {
			out = append(out, descriptor)
		}
	}
	if len(out) != len(agentRefs) {
		t.Fatalf("descriptors=%d want=%d agents=%v got=%v", len(out), len(agentRefs), agentRefs, codexStackRealSmokeDescriptorAgentsV0(out))
	}
	return out
}

func codexStackOperationalWaveTaskRefsV0(fixture codexStackOperationalWaveFixtureV0) []string {
	out := make([]string, 0, len(fixture.Refs))
	for _, refs := range fixture.Refs {
		out = append(out, refs.TaskRef)
	}
	return out
}

func codexStackOperationalWaveAgentRefsV0(fixture codexStackOperationalWaveFixtureV0) []string {
	out := make([]string, 0, len(fixture.Refs))
	for _, refs := range fixture.Refs {
		out = append(out, refs.AgentRef)
	}
	return out
}

type codexStackRecursiveTreeFixtureV0 struct {
	RunRef  string
	PlanRef string
	Tasks   []orquestacoreworkflow.WorkflowTaskV0
}

type appDirectorWaitFilterForRecursiveTreeV0 struct {
	CohortRef     string
	WaveRef       string
	ParentTaskRef string
}

func newCodexStackRecursiveTreeFixtureV0() codexStackRecursiveTreeFixtureV0 {
	return newCodexStackRecursiveTreeFixtureWithSuffixV0("fake")
}

func newCodexStackRecursiveTreeFixtureWithSuffixV0(suffix string) codexStackRecursiveTreeFixtureV0 {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = "smoke"
	}
	runRef := "run-stack-recursive-tree-" + suffix + "-001"
	parent := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-parent", "", "wave-recursive-root", "cohort-recursive-root", 0, 2)
	childA := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-child-a", parent.TaskID, "wave-recursive-child-a", "cohort-recursive-children", 1, 2)
	childB := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-child-b", parent.TaskID, "wave-recursive-child-b", "cohort-recursive-children", 1, 2)
	grandA1 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-grand-a1", childA.TaskID, "wave-recursive-child-a", "cohort-recursive-grandchildren-a", 2, 0)
	grandA2 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-grand-a2", childA.TaskID, "wave-recursive-child-a", "cohort-recursive-grandchildren-a", 2, 0)
	grandB1 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-grand-b1", childB.TaskID, "wave-recursive-child-b", "cohort-recursive-grandchildren-b", 2, 0)
	grandB2 := stackOperationalClosureRecursiveTaskForTestV0(runRef, "task-recursive-grand-b2", childB.TaskID, "wave-recursive-child-b", "cohort-recursive-grandchildren-b", 2, 0)
	parent.ChildTaskRefs = []string{childA.TaskID, childB.TaskID}
	childA.ChildTaskRefs = []string{grandA1.TaskID, grandA2.TaskID}
	childB.ChildTaskRefs = []string{grandB1.TaskID, grandB2.TaskID}
	childA.DependsOn = []string{parent.TaskID}
	childB.DependsOn = []string{parent.TaskID}
	grandA1.DependsOn = []string{childA.TaskID}
	grandA2.DependsOn = []string{childA.TaskID}
	grandB1.DependsOn = []string{childB.TaskID}
	grandB2.DependsOn = []string{childB.TaskID}
	tasks := []orquestacoreworkflow.WorkflowTaskV0{parent, childA, childB, grandA1, grandA2, grandB1, grandB2}
	return codexStackRecursiveTreeFixtureV0{
		RunRef:  runRef,
		PlanRef: "plan-stack-recursive-tree-" + suffix + "-001",
		Tasks:   tasks,
	}
}

func (fixture codexStackRecursiveTreeFixtureV0) taskV0(
	taskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	for _, task := range fixture.Tasks {
		if task.TaskID == taskRef {
			return task
		}
	}
	return orquestacoreworkflow.WorkflowTaskV0{}
}

func codexStackRecursiveTreeSeedV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackRecursiveTreeFixtureV0,
) {
	t.Helper()
	stackOperationalClosureAssertRecursiveLimitsForTestV0(t, fixture.Tasks, 2)
	run := (codexStackRequiredTestRefsV0{
		RunRef:  fixture.RunRef,
		PlanRef: fixture.PlanRef,
		TaskRef: fixture.Tasks[0].TaskID,
	}).withDefaultsV0().pendingRunV0()
	run.Tasks = codexStackRecursiveTreeTaskRefsV0(fixture)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	for _, task := range fixture.Tasks {
		if err := taskWriter.SaveWorkflowTaskV0(ctx, task); err != nil {
			t.Fatalf("SaveWorkflowTaskV0 %s: %v", task.TaskID, err)
		}
	}
}

func codexStackRecursiveTreeSuperviseWithStackLifecycleV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackRecursiveTreeFixtureV0,
	planRef string,
	maxTicks int,
) CodexSupervisorResultV0 {
	t.Helper()
	result, err := SuperviseCodexV0(ctx, CodexSupervisorDepsV0{AgentLifecycle: CodexSupervisorStackLifecycleV0{
		Stack:  stack,
		RunRef: fixture.RunRef,
		DrainRequest: DrainRunRequestV0{
			RunRef:                     fixture.RunRef,
			OperationalDirectorPlanRef: planRef,
			OccurredAt:                 "2026-05-22T20:30:00Z",
			CorrelationID:              "corr-recursive-tree-supervisor-" + strings.TrimSpace(planRef),
			MaxBursts:                  8,
			MaxStepsPerBurst:           10,
			MaxDispatchesPerWait:       4,
			MaxCommands:                40,
			MaxOutboxPerCycle:          20,
			MaxDecisionCycles:          4,
			MaxExternalWaits:           1,
		},
	}}, CodexSupervisorCommandV0{
		MaxTicks:        maxTicks,
		ContinueMessage: DefaultCodexSupervisorContinueMessageV0,
	})
	if err != nil {
		t.Fatalf("SuperviseCodexV0 plan=%s: %v result=%+v", planRef, err, result)
	}
	if len(result.History) == 0 || result.History[0].Action != "launch" {
		t.Fatalf("history sin launch inicial: %+v", result.History)
	}
	if len(result.History) > 1 && result.History[1].Action != "continue" {
		t.Fatalf("history sin sigue/continue: %+v", result.History)
	}
	return result
}

func codexStackRecursiveTreeAssertLineageV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackRecursiveTreeFixtureV0,
) {
	t.Helper()
	tasks, err := stack.Stores.TaskStore.LoadWorkflowTasksV0(ctx, fixture.RunRef, codexStackRecursiveTreeTaskRefsV0(fixture))
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	stackOperationalClosureAssertRecursiveLimitsForTestV0(t, tasks, 2)
	for _, want := range fixture.Tasks {
		var got orquestacoreworkflow.WorkflowTaskV0
		for _, task := range tasks {
			if task.TaskID == want.TaskID {
				got = task
				break
			}
		}
		if got.TaskID == "" {
			t.Fatalf("task no encontrada: %s tasks=%+v", want.TaskID, tasks)
		}
		if got.ParentTaskRef != want.ParentTaskRef ||
			got.DelegationDepth != want.DelegationDepth ||
			got.MaxChildAgents != want.MaxChildAgents ||
			got.WaveRef != want.WaveRef ||
			got.CohortRef != want.CohortRef {
			t.Fatalf("lineage task=%s got=%+v want=%+v", want.TaskID, got, want)
		}
		stackOperationalClosureAssertRefsForTestV0(t, "child refs "+want.TaskID, got.ChildTaskRefs, want.ChildTaskRefs)
	}
}

func codexStackRecursiveTreeAssertSupervisorWaitScopesV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackRecursiveTreeFixtureV0,
) {
	t.Helper()
	parent := fixture.taskV0("task-recursive-parent")
	childA := fixture.taskV0("task-recursive-child-a")
	childB := fixture.taskV0("task-recursive-child-b")
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, parent.TaskID, "", []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskID),
	}, []string{})
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, childA.TaskID, childA.WaveRef, []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-a1"),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-a2"),
	}, []string{})
	codexStackRecursiveTreeAssertWaitSnapshotV0(t, ctx, stack, fixture.RunRef, childB.TaskID, childB.WaveRef, []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-b1"),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-recursive-grand-b2"),
	}, []string{})
}

func codexStackRecursiveTreeContinueLaunchV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	launchCounter codexStackLaunchCounterForTestV0,
	runRef string,
	filter appDirectorWaitFilterForRecursiveTreeV0,
	expectedAgentRefs []string,
) {
	t.Helper()
	before := 0
	if launchCounter != nil {
		before = launchCounter.launchCountV0()
	}
	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-22T20:00:00Z",
		CorrelationID:        "corr-recursive-tree-launch-" + strings.ReplaceAll(strings.Join(compactStringsV0(expectedAgentRefs), "-"), "agent-ref-", ""),
		WaitWaveRef:          filter.WaveRef,
		WaitCohortRef:        filter.CohortRef,
		WaitParentTaskRef:    filter.ParentTaskRef,
		MaxBursts:            6,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		MaxCommands:          24,
		MaxOutboxPerCycle:    12,
		MaxExternalWaits:     1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 launch filter=%+v: %v %s", filter, err, codexStackRequiredTestErrorDetailsV0(err))
	}
	if launchCounter != nil {
		if got := launchCounter.launchCountV0() - before; got != len(expectedAgentRefs) {
			t.Fatalf("launches nuevos=%d want=%d filter=%+v started=%v run=%+v", got, len(expectedAgentRefs), filter, result.StartedAgents, result.Run)
		}
	}
	for _, agentRef := range expectedAgentRefs {
		if !codexStackStringInSetForTestV0(result.StartedAgents, agentRef) {
			t.Fatalf("agent no arrancado: agent=%s filter=%+v started=%v", agentRef, filter, result.StartedAgents)
		}
	}
	codexStackOperationalWaveDescriptorsForAgentsV0(t, stack, expectedAgentRefs)
}

func codexStackRecursiveTreeDrainAgentV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	agentRef string,
	cfg codexStackRealSmokeConfigV0,
	manualAck bool,
) string {
	t.Helper()
	if manualAck {
		return codexStackRecursiveTreeAckAndDrainV0(t, ctx, stack, runRef, agentRef, cfg)
	}
	delivered := codexStackRequiredTestDrainUntilDeliveryV0(t, ctx, stack, runRef, agentRef, cfg.RuntimeWorkDir, cfg.Timeout)
	deliveryRef := strings.TrimSpace(delivered.Spec.AgentPacket.DeliveryRefs.AckRef)
	if deliveryRef == "" {
		t.Fatalf("delivery ref vacia agent=%s descriptor=%+v", agentRef, delivered)
	}
	return deliveryRef
}

func codexStackRecursiveTreeAckAndDrainV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	agentRef string,
	cfg codexStackRealSmokeConfigV0,
) string {
	t.Helper()
	descriptor := codexStackRequiredTestDescriptorForAgentV0(t, stack, agentRef)
	if err := writeCodexStackAckForDescriptorV0(t, descriptor); err != nil {
		t.Fatalf("write ack agent=%s: %v", agentRef, err)
	}
	request := DrainRunRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-22T20:05:00Z",
		CorrelationID: "corr-recursive-tree-delivery-" + agentRef,
		WaitAgentRefs: []string{agentRef},
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	observations, err := stack.drainAvailableObservationsV0(ctx, request, run)
	if err != nil {
		t.Fatalf("drainAvailableObservationsV0 agent=%s: %v\n%s", agentRef, err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	observations = drainObservationsForWaitAgentRefsV0(observations, []string{agentRef})
	if len(observations) != 1 {
		t.Fatalf("observations=%+v agent=%s\n%s", observations, agentRef, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	run, err = stack.applyDrainObservationsV0(ctx, request, observations)
	if err != nil {
		t.Fatalf("applyDrainObservationsV0 agent=%s: %v\n%s", agentRef, err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	deliveryRef := strings.TrimSpace(observations[0].DeliveryRef)
	if deliveryRef == "" ||
		!codexStackStringInSetForTestV0(run.DeliveredAgents, agentRef) ||
		!codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, deliveryRef) {
		t.Fatalf("delivery no registrada: agent=%s delivery=%s run=%+v", agentRef, deliveryRef, run)
	}
	return deliveryRef
}

func codexStackRecursiveTreeAssertWaitSnapshotV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	parentTaskRef string,
	waveRef string,
	wantAgentRefs []string,
	wantPendingAgentRefs []string,
) {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	snapshot, err := orquestacionnucleoapp.BuildWorkflowTaskWaitSnapshotV0(ctx, stack.Stores.TaskStore, run, orquestacionnucleoapp.WorkflowTaskWaitFilterV0{
		ParentTaskRef: parentTaskRef,
		WaveRef:       waveRef,
	})
	if err != nil {
		t.Fatalf("BuildWorkflowTaskWaitSnapshotV0 parent=%s wave=%s: %v", parentTaskRef, waveRef, err)
	}
	stackOperationalClosureAssertRefsForTestV0(t, "agent refs "+parentTaskRef, snapshot.AgentRefs, wantAgentRefs)
	stackOperationalClosureAssertRefsForTestV0(t, "pending agent refs "+parentTaskRef, snapshot.PendingAgentRefs, wantPendingAgentRefs)
}

func codexStackRecursiveTreeAssertDeliveredV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	agentRefs []string,
) {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	for _, agentRef := range agentRefs {
		if !codexStackStringInSetForTestV0(run.DeliveredAgents, agentRef) {
			t.Fatalf("agent no entregado: agent=%s delivered=%v run=%+v", agentRef, run.DeliveredAgents, run)
		}
	}
}

func codexStackRecursiveTreeDeliveryByAgentV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	agentRefs []string,
) map[string]string {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	out := map[string]string{}
	for _, descriptor := range codexStackOperationalWaveDescriptorsForAgentsV0(t, stack, agentRefs) {
		ack, ok := codexStackRealSmokeCompletedAckV0(descriptor)
		if ok && codexStackRealSmokeContainsProjectionPartV0(run.Deliveries, ack.AckRef) {
			out[descriptor.AgentRef] = ack.AckRef
		}
	}
	if len(out) != len(agentRefs) {
		t.Fatalf("deliveries por agente incompletas: got=%v want=%v run=%+v", out, agentRefs, run)
	}
	return out
}

func codexStackRecursiveTreePlanStateV0(
	fixture codexStackRecursiveTreeFixtureV0,
	causals map[string]codexStackRequiredTestCausalReviewRefsV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	agentRefs := codexStackRecursiveTreeAgentRefsV0(fixture)
	taskRefs := codexStackRecursiveTreeTaskRefsV0(fixture)
	deliveryRefs := make([]string, 0, len(agentRefs))
	reviewResultRefs := make([]string, 0, len(agentRefs))
	acceptedReviewRefs := make([]string, 0, len(agentRefs))
	for _, agentRef := range agentRefs {
		causal := causals[agentRef]
		deliveryRefs = append(deliveryRefs, causal.DeliveryRef)
		reviewResultRefs = append(reviewResultRefs, causal.ReviewResultRef)
		acceptedReviewRefs = append(acceptedReviewRefs, causal.AcceptedReviewRef)
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:    orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "state-" + fixture.PlanRef,
		PlanRef:          fixture.PlanRef,
		RequestRef:       "request-" + fixture.PlanRef,
		RunRef:           fixture.RunRef,
		ProjectRef:       "project-recursive-tree-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-run-required-tests",
		ActiveWaveRef:    "wave-recursive-tree",
		ActiveCohortRef:  "cohort-recursive-tree",
		RequiredTestRefs: []string{"go test ./..."},
		ObservedAt:       "2026-05-22T20:10:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-recursive-tree",
				CohortRef: "cohort-recursive-tree",
				TaskRefs:  taskRefs,
				AgentRefs: agentRefs,
			},
			{
				StepID:    "step-wait-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-recursive-tree",
				CohortRef: "cohort-recursive-tree",
				TaskRefs:  taskRefs,
				AgentRefs: agentRefs,
			},
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:            "wave-recursive-tree",
				CohortRef:          "cohort-recursive-tree",
				TaskRefs:           taskRefs,
				AgentRefs:          agentRefs,
				DeliveryRefs:       deliveryRefs,
				ReviewResultRefs:   reviewResultRefs,
				AcceptedReviewRefs: acceptedReviewRefs,
			},
			{
				StepID:           "step-run-required-tests",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          "wave-recursive-tree",
				CohortRef:        "cohort-recursive-tree",
				TaskRefs:         taskRefs,
				AgentRefs:        agentRefs,
				DeliveryRefs:     deliveryRefs,
				ReviewResultRefs: reviewResultRefs,
			},
			{
				StepID:    "step-replan-or-close",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   "wave-recursive-tree",
				CohortRef: "cohort-recursive-tree",
			},
		},
	}
}

func codexStackRecursiveTreeParentOnlyReadyToCloseStateV0(
	fixture codexStackRecursiveTreeFixtureV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
	deliveryRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	parentAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(parent.TaskID)
	reviewResultRef := "review-result-ref-" + deliveryRef
	acceptedReviewRef := "accepted-review-ref-" + deliveryRef
	requiredTestEvidenceRefs := []string{"test-evidence-ref-" + deliveryRef}
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:    orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "state-" + fixture.PlanRef + "-parent-only",
		PlanRef:          fixture.PlanRef,
		RequestRef:       "request-" + fixture.PlanRef + "-parent-only",
		RunRef:           fixture.RunRef,
		ProjectRef:       "project-recursive-tree-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-replan-or-close",
		ActiveWaveRef:    parent.WaveRef,
		ActiveCohortRef:  parent.CohortRef,
		RequiredTestRefs: []string{"go test ./..."},
		ObservedAt:       "2026-05-22T20:40:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   parent.WaveRef,
				CohortRef: parent.CohortRef,
				TaskRefs:  []string{parent.TaskID},
				AgentRefs: []string{parentAgent},
			},
			{
				StepID:    "step-wait-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   parent.WaveRef,
				CohortRef: parent.CohortRef,
				TaskRefs:  []string{parent.TaskID},
				AgentRefs: []string{parentAgent},
			},
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:            parent.WaveRef,
				CohortRef:          parent.CohortRef,
				TaskRefs:           []string{parent.TaskID},
				AgentRefs:          []string{parentAgent},
				DeliveryRefs:       []string{deliveryRef},
				ReviewResultRefs:   []string{reviewResultRef},
				AcceptedReviewRefs: []string{acceptedReviewRef},
			},
			{
				StepID:                   "step-run-required-tests",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:                  parent.WaveRef,
				CohortRef:                parent.CohortRef,
				TaskRefs:                 []string{parent.TaskID},
				AgentRefs:                []string{parentAgent},
				DeliveryRefs:             []string{deliveryRef},
				ReviewResultRefs:         []string{reviewResultRef},
				RequiredTestEvidenceRefs: requiredTestEvidenceRefs,
				Reason:                   "required-tests-passed",
			},
			{
				StepID:                   "step-replan-or-close",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:                  parent.WaveRef,
				CohortRef:                parent.CohortRef,
				TaskRefs:                 []string{parent.TaskID},
				AgentRefs:                []string{parentAgent},
				DeliveryRefs:             []string{deliveryRef},
				ReviewResultRefs:         []string{reviewResultRef},
				RequiredTestEvidenceRefs: requiredTestEvidenceRefs,
				Reason:                   "required-tests-passed",
			},
		},
	}
}

func codexStackRecursiveTreeCloseV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	fixture codexStackRecursiveTreeFixtureV0,
	cfg codexStackRealSmokeConfigV0,
) (orquestacoreworkflow.OrchestrationRunV0, orquestacionnucleoapp.OperationalDirectorPlanStateV0) {
	t.Helper()
	var run orquestacoreworkflow.OrchestrationRunV0
	var state orquestacionnucleoapp.OperationalDirectorPlanStateV0
	for cycle := 1; cycle <= 20; cycle++ {
		result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OperationalDirectorPlanRef: fixture.PlanRef,
			OccurredAt:                 fmt.Sprintf("2026-05-22T20:%02d:00Z", 10+cycle),
			CorrelationID:              fmt.Sprintf("corr-recursive-tree-close-%03d", cycle),
			MaxBursts:                  3,
			MaxStepsPerBurst:           6,
			MaxDispatchesPerWait:       2,
			MaxCommands:                32,
			MaxOutboxPerCycle:          16,
			MaxExternalWaits:           1,
		}, stack.Ports)
		if err != nil {
			t.Fatalf("ContinueAppDirectorV0 close ciclo=%d: %v %s\n%s", cycle, err, codexStackRequiredTestErrorDetailsV0(err), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
		}
		run = result.Run
		var loadErr error
		state, loadErr = stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
		if loadErr != nil {
			t.Fatalf("LoadOperationalDirectorPlanStateV0 close ciclo=%d: %v", cycle, loadErr)
		}
		if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 &&
			state.Status == orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
			return run, state
		}
	}
	return run, state
}

func codexStackRecursiveTreeAssertEvidenceV0(
	t *testing.T,
	fixture codexStackRecursiveTreeFixtureV0,
	causals map[string]codexStackRequiredTestCausalReviewRefsV0,
	evidence []orquestacionnucleoapp.RequiredTestEvidenceV0,
) {
	t.Helper()
	byTask := map[string]orquestacionnucleoapp.RequiredTestEvidenceV0{}
	for _, item := range evidence {
		if item.Status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
			byTask[item.TaskRef] = item
		}
	}
	for _, task := range fixture.Tasks {
		item, ok := byTask[task.TaskID]
		if !ok {
			t.Fatalf("sin evidencia passed task=%s evidence=%+v", task.TaskID, evidence)
		}
		causal := causals[orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID)]
		if item.TestCommand != "go test ./..." ||
			item.DeliveryRef != causal.DeliveryRef ||
			item.ReviewRequestID != causal.ReviewRequestRef ||
			item.ReviewResultRef != causal.ReviewResultRef ||
			item.AcceptedReviewRef != causal.AcceptedReviewRef {
			t.Fatalf("evidencia no causal task=%s evidence=%+v causal=%+v", task.TaskID, item, causal)
		}
	}
}

func codexStackRecursiveTreeTaskRefsV0(
	fixture codexStackRecursiveTreeFixtureV0,
) []string {
	out := make([]string, 0, len(fixture.Tasks))
	for _, task := range fixture.Tasks {
		out = append(out, task.TaskID)
	}
	return out
}

func codexStackRecursiveTreeAgentRefsV0(
	fixture codexStackRecursiveTreeFixtureV0,
) []string {
	out := make([]string, 0, len(fixture.Tasks))
	for _, task := range fixture.Tasks {
		out = append(out, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return out
}
