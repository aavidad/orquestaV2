package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestBuildStackFromEnvV0RequiredTestRunnerEjecutaGoTestYPersisteEvidencia(t *testing.T) {
	ctx := context.Background()
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go no encontrado: %v", err)
	}
	projectDir := serverRequiredTestTinyGoModuleV0(t)
	stateDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "required-test-output")
	runRef := "run-server-required-test-runner-001"
	planRef := "plan-server-required-test-runner-001"
	taskRef := "task-ref-autoprogramming-server-required-test-runner-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-server-required-test-runner-001"
	reviewRequestRef := "review-request-server-required-test-runner-001"
	reviewResultRef := "review-result-server-required-test-runner-001"
	acceptedReviewRef := "accepted-review-server-required-test-runner-001"

	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", projectDir)
	t.Setenv("ORQUESTA_SERVER_STATE_DIR", stateDir)
	t.Setenv("ORQUESTA_CODEX_RUNTIME_WORKDIR", filepath.Join(t.TempDir(), "runtime"))
	t.Setenv("ORQUESTA_CODEX_COMMAND", filepath.Join(projectDir, "codex-bin"))
	t.Setenv("ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED", "1")
	t.Setenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND", goCommand)
	t.Setenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", outputDir)
	t.Setenv("ORQUESTA_REQUIRED_TEST_ENV", "CGO_ENABLED=0")
	t.Setenv("ORQUESTA_OPES_BASE_URL", "")
	t.Setenv("OPES_BASE_URL", "")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}

	run := serverRequiredTestRunV0(
		runRef,
		taskRef,
		agentRef,
		deliveryRef,
		reviewRequestRef,
		reviewResultRef,
		acceptedReviewRef,
	)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, runRef, serverRequiredTestEventsV0(
		t,
		runRef,
		taskRef,
		agentRef,
		deliveryRef,
		reviewRequestRef,
		reviewResultRef,
		acceptedReviewRef,
	)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, serverRequiredTestWorkflowTaskV0(runRef, taskRef)); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(
		ctx,
		serverRequiredTestPlanStateV0(runRef, planRef, taskRef, agentRef, deliveryRef, reviewResultRef, acceptedReviewRef),
	); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-22T16:00:00Z",
		CorrelationID:              "corr-server-required-test-runner-001",
		MaxBursts:                  4,
		MaxStepsPerBurst:           4,
		MaxDispatchesPerWait:       1,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: status=%s loop=%s closure_issues=%+v state_status=%s active_step=%s steps=%+v closed=%v validations=%v closures=%v",
			result.Run.Status,
			result.LoopStatus,
			result.OperationalClosureIssues,
			state.Status,
			state.ActiveStepID,
			state.Steps,
			result.Run.ClosedTasks,
			result.Run.Validations,
			result.Run.Closures,
		)
	}
	replanStep := serverRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("plan state sin cierre/evidencia: state=%+v replan=%+v", state, replanStep)
	}
	evidence, err := stack.Stores.RequiredTestEvidenceStore.LoadRequiredTestEvidenceV0(ctx, runRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TestCommand != "go test ./..." ||
		evidence[0].TaskRef != taskRef ||
		evidence[0].DeliveryRef != deliveryRef ||
		evidence[0].ReviewRequestID != reviewRequestRef ||
		evidence[0].ReviewResultRef != reviewResultRef ||
		evidence[0].AcceptedReviewRef != acceptedReviewRef {
		t.Fatalf("evidence invalida: %+v", evidence)
	}
	outputRef := serverRequiredTestOutputRefV0(evidence[0].EvidenceRefs)
	if outputRef == "" {
		t.Fatalf("evidence sin artefacto de salida: %+v", evidence[0])
	}
	content, err := os.ReadFile(filepath.Join(outputDir, strings.TrimPrefix(outputRef, "required-test-output-v0/")))
	if err != nil {
		t.Fatalf("leer output required test: %v", err)
	}
	if !strings.Contains(string(content), "status=passed") ||
		!strings.Contains(string(content), "test_command=go test ./...") ||
		strings.Contains(string(content), outputDir) ||
		strings.Contains(string(content), projectDir) {
		t.Fatalf("output required test invalido: %q", string(content))
	}
}

func serverRequiredTestTinyGoModuleV0(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/orquesta-required-test-integration\n\ngo 1.22\n",
		"calc.go": strings.Join([]string{
			"package calc",
			"",
			"func Add(a int, b int) int {",
			"\treturn a + b",
			"}",
			"",
		}, "\n"),
		"calc_test.go": strings.Join([]string{
			"package calc",
			"",
			"import \"testing\"",
			"",
			"func TestAdd(t *testing.T) {",
			"\tif Add(2, 3) != 5 {",
			"\t\tt.Fatalf(\"Add fallo\")",
			"\t}",
			"}",
			"",
		}, "\n"),
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func serverRequiredTestRunV0(
	runRef string,
	taskRef string,
	agentRef string,
	deliveryRef string,
	reviewRequestRef string,
	reviewResultRef string,
	acceptedReviewRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "project-server-required-test-runner-001",
		AppSpecRef:      "app-spec-server-required-test-runner-001",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases:          serverRequiredTestPhasesV0(),
		Tasks:           []string{taskRef},
		Agents:          []string{agentRef},
		StartedAgents:   []string{agentRef},
		DeliveredAgents: []string{agentRef},
		Deliveries:      []string{deliveryRef},
		DeliveredTasks:  []string{taskRef},
		Reviews:         []string{reviewRequestRef},
		ReviewResults: []string{
			reviewResultRef + "#review_result:accepted#review_request:" + reviewRequestRef + "#delivery:" + deliveryRef,
		},
		AcceptedReviews: []string{acceptedReviewRef},
		LastEventID:     "evt-server-required-test-runner-review-accepted",
		LastSequence:    4,
	}
}

func serverRequiredTestPhasesV0() []orquestacoreworkflow.OrchestrationPhaseV0 {
	capacity := orquestacoreworkflow.OrchestrationCapacityHighV0
	return []orquestacoreworkflow.OrchestrationPhaseV0{
		{ID: orquestacoreworkflow.OrchestrationPhaseProgramacionV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseRevisionV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusActiveV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseCierreV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
	}
}

func serverRequiredTestEventsV0(
	t *testing.T,
	runRef string,
	taskRef string,
	agentRef string,
	deliveryRef string,
	reviewRequestRef string,
	reviewResultRef string,
	acceptedReviewRef string,
) []orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	delivery, err := orquestacoreworkflow.NewDeliveryRegisteredEventV0(
		serverRequiredTestEventMetaV0(runRef, 1, "delivery"),
		orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  deliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       taskRef,
			AgentRef:     agentRef,
			Summary:      "Entrega de programacion lista para revision.",
			EvidenceRefs: []string{"artifact-ref-server-required-test-delivery"},
		},
	)
	if err != nil {
		t.Fatalf("NewDeliveryRegisteredEventV0: %v", err)
	}
	requested, err := orquestacoreworkflow.NewReviewRequestedEventV0(
		serverRequiredTestEventMetaV0(runRef, 2, "review-request"),
		orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: reviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Revision requerida para entrega de programacion.",
			EvidenceRefs:    []string{"artifact-ref-server-required-test-review-request"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewRequestedEventV0: %v", err)
	}
	recorded, err := orquestacoreworkflow.NewReviewResultRecordedEventV0(
		serverRequiredTestEventMetaV0(runRef, 3, "review-result"),
		orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestRef,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Revision aceptada; ejecutar tests requeridos.",
			EvidenceRefs:    []string{"artifact-ref-server-required-test-review-result"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewResultRecordedEventV0: %v", err)
	}
	accepted, err := orquestacoreworkflow.NewReviewAcceptedEventV0(
		serverRequiredTestEventMetaV0(runRef, 4, "review-accepted"),
		orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: acceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   reviewRequestRef,
			DeliveryRef:       deliveryRef,
			Summary:           "Revision aceptada con cadena causal completa.",
			EvidenceRefs:      []string{"artifact-ref-server-required-test-review-accepted"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewAcceptedEventV0: %v", err)
	}
	return []orquestacoreworkflow.OrchestrationEventV0{delivery, requested, recorded, accepted}
}

func serverRequiredTestEventMetaV0(
	runRef string,
	sequence int64,
	suffix string,
) orquestacoreworkflow.OrchestrationEventMetaV0 {
	return orquestacoreworkflow.OrchestrationEventMetaV0{
		EventID:       "evt-server-required-test-runner-" + suffix,
		RunID:         runRef,
		Sequence:      sequence,
		CorrelationID: "corr-server-required-test-runner-seed",
		OccurredAt:    "2026-05-22T15:59:00Z",
	}
}

func serverRequiredTestWorkflowTaskV0(
	runRef string,
	taskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Implementar modulo Go con tests",
		Summary:            "Microtarea de programacion con cierre causal.",
		WriteSet:           []string{"go.mod", "calc.go", "calc_test.go"},
		AcceptanceCriteria: []string{"operational_director.required_tests passed", "codigo Go probado"},
		RequiredTests:      []string{"go test ./..."},
		WaveRef:            "wave-server-required-test-runner-001",
		CohortRef:          "cohort-server-required-test-runner-001",
	}
}

func serverRequiredTestPlanStateV0(
	runRef string,
	planRef string,
	taskRef string,
	agentRef string,
	deliveryRef string,
	reviewResultRef string,
	acceptedReviewRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:   orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "state-server-required-test-runner-001",
		PlanRef:         planRef,
		RequestRef:      "request-server-required-test-runner-001",
		RunRef:          runRef,
		ProjectRef:      "project-server-required-test-runner-001",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-run-required-tests",
		ActiveWaveRef:   "wave-server-required-test-runner-001",
		ActiveCohortRef: "cohort-server-required-test-runner-001",
		RequiredTestRefs: []string{
			"go test ./...",
		},
		ObservedAt: "2026-05-22T15:58:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-server-required-test-runner-001",
				CohortRef: "cohort-server-required-test-runner-001",
				TaskRefs:  []string{taskRef},
				AgentRefs: []string{agentRef},
			},
			{
				StepID:       "step-review-deliveries",
				Kind:         orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:       orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:      "wave-server-required-test-runner-001",
				CohortRef:    "cohort-server-required-test-runner-001",
				TaskRefs:     []string{taskRef},
				AgentRefs:    []string{agentRef},
				DeliveryRefs: []string{deliveryRef},
				ReviewResultRefs: []string{
					reviewResultRef,
				},
				AcceptedReviewRefs: []string{acceptedReviewRef},
			},
			{
				StepID:           "step-run-required-tests",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          "wave-server-required-test-runner-001",
				CohortRef:        "cohort-server-required-test-runner-001",
				TaskRefs:         []string{taskRef},
				AgentRefs:        []string{agentRef},
				DeliveryRefs:     []string{deliveryRef},
				ReviewResultRefs: []string{reviewResultRef},
			},
			{
				StepID:    "step-replan-or-close",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   "wave-server-required-test-runner-001",
				CohortRef: "cohort-server-required-test-runner-001",
			},
		},
	}
}

func serverRequiredTestPlanStepV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step no encontrado: %s en %+v", stepID, state.Steps)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func serverRequiredTestOutputRefV0(refs []string) string {
	for _, ref := range refs {
		if strings.HasPrefix(strings.TrimSpace(ref), "required-test-output-v0/") {
			return strings.TrimSpace(ref)
		}
	}
	return ""
}
