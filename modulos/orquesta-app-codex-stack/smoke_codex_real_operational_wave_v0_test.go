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
