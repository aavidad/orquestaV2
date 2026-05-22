package orquestaappcodexstack

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestCodexStackRealRequiredTestRunnerOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_SMOKE=1")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_CONFIRM")) != "1" ||
		strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_REAL_REQUIRED_TEST_RUNNER_NO_CODEX_EXECUTION_CONFIRMED")) != "1" {
		t.Fatal("doble confirmacion requerida para smoke Codex real sin ejecucion")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_OPES_BASE_URL")) != "" ||
		strings.TrimSpace(os.Getenv("OPES_BASE_URL")) != "" {
		t.Fatal("este smoke no debe cablear OPES")
	}

	ctx := context.Background()
	cfg := codexStackRealSmokeConfigForTestV0(t)
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	goCommand := codexStackRequiredTestGoCommandV0(t)
	outputDir := codexStackRealSmokeEnsureDirV0(t, "ORQUESTA_REQUIRED_TEST_OUTPUT_DIR", codexStackRequiredTestOutputDirV0(t))

	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(t, cfg, processRuntime, evidenceStore, goCommand, outputDir)
	if stack.Ports.RequiredTestRunner == nil {
		t.Fatal("RequiredTestRunner no cableado")
	}

	refs := (codexStackRequiredTestRefsV0{}).withDefaultsV0()
	run := refs.runV0()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, refs.RunRef, refs.eventsV0(t)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, refs.workflowTaskV0()); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, refs.planStateV0()); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	result, err := orquestaappdirectorservice.ContinueAppDirectorV0(ctx, orquestaappdirectorservice.ContinueAppDirectorRequestV0{
		RunRef:                     refs.RunRef,
		OperationalDirectorPlanRef: refs.PlanRef,
		OccurredAt:                 "2026-05-22T17:00:00Z",
		CorrelationID:              "corr-rt-runner-continue",
		MaxBursts:                  1,
		MaxStepsPerBurst:           1,
		MaxDispatchesPerWait:       1,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}, stack.Ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: status=%s closed=%v validations=%v closures=%v", result.Run.Status, result.Run.ClosedTasks, result.Run.Validations, result.Run.Closures)
	}

	state, err := stack.Stores.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("plan state sin cierre/evidencia: state=%+v replan=%+v", state, replanStep)
	}
	evidence, err := evidenceStore.LoadRequiredTestEvidenceV0(ctx, refs.RunRef, replanStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TestCommand != "go test ./..." ||
		evidence[0].TaskRef != refs.TaskRef ||
		evidence[0].DeliveryRef != refs.DeliveryRef ||
		evidence[0].ReviewRequestID != refs.ReviewRequestRef ||
		evidence[0].ReviewResultRef != refs.ReviewResultRef ||
		evidence[0].AcceptedReviewRef != refs.AcceptedReviewRef {
		t.Fatalf("evidence invalida: %+v", evidence)
	}
	outputRef := codexStackRequiredTestOutputRefV0(evidence[0].EvidenceRefs)
	if outputRef == "" {
		t.Fatalf("evidence sin output: %+v", evidence[0])
	}
	content, err := os.ReadFile(filepath.Join(outputDir, strings.TrimPrefix(outputRef, "required-test-output-v0/")))
	if err != nil {
		t.Fatalf("leer output required test: %v", err)
	}
	if !strings.Contains(string(content), "status=passed") ||
		!strings.Contains(string(content), "test_command=go test ./...") ||
		strings.Contains(string(content), cfg.ProjectWorkDir) ||
		strings.Contains(string(content), outputDir) {
		t.Fatalf("output required test invalido: %q", string(content))
	}
	if entries := codexStackRequiredTestRuntimeEntriesV0(t, cfg.RuntimeWorkDir); len(entries) != 0 {
		t.Fatalf("Runtime Codex no debia ejecutarse; runtime_entries=%+v", entries)
	}
}

type codexStackRequiredTestRefsV0 struct {
	RunRef            string
	PlanRef           string
	TaskRef           string
	AgentRef          string
	DeliveryRef       string
	ReviewRequestRef  string
	ReviewResultRef   string
	AcceptedReviewRef string
}

func (refs codexStackRequiredTestRefsV0) withDefaultsV0() codexStackRequiredTestRefsV0 {
	if refs.RunRef != "" {
		return refs
	}
	taskRef := "task-rt-runner-001"
	return codexStackRequiredTestRefsV0{
		RunRef:            "run-rt-runner-001",
		PlanRef:           "plan-rt-runner-001",
		TaskRef:           taskRef,
		AgentRef:          orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef),
		DeliveryRef:       "delivery-rt-runner-001",
		ReviewRequestRef:  "review-request-rt-runner-001",
		ReviewResultRef:   "review-result-rt-runner-001",
		AcceptedReviewRef: "accepted-review-rt-runner-001",
	}
}

func (refs codexStackRequiredTestRefsV0) runV0() orquestacoreworkflow.OrchestrationRunV0 {
	refs = refs.withDefaultsV0()
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           refs.RunRef,
		ProjectRef:      "project-rt-runner-001",
		AppSpecRef:      "app-spec-rt-runner-001",
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases:          codexStackRequiredTestPhasesV0(),
		Tasks:           []string{refs.TaskRef},
		Agents:          []string{refs.AgentRef},
		StartedAgents:   []string{refs.AgentRef},
		DeliveredAgents: []string{refs.AgentRef},
		Deliveries:      []string{refs.DeliveryRef},
		DeliveredTasks:  []string{refs.TaskRef},
		Reviews:         []string{refs.ReviewRequestRef},
		ReviewResults: []string{
			refs.ReviewResultRef + "#review_result:accepted#review_request:" + refs.ReviewRequestRef + "#delivery:" + refs.DeliveryRef,
		},
		AcceptedReviews: []string{refs.AcceptedReviewRef},
		LastEventID:     "evt-rt-runner-review-accepted",
		LastSequence:    4,
	}
}

func (refs codexStackRequiredTestRefsV0) eventsV0(t *testing.T) []orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	refs = refs.withDefaultsV0()
	delivery, err := orquestacoreworkflow.NewDeliveryRegisteredEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 1, "delivery"),
		orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  refs.DeliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       refs.TaskRef,
			AgentRef:     refs.AgentRef,
			Summary:      "Entrega de programacion lista para revision.",
			EvidenceRefs: []string{"artifact-ref-rt-delivery"},
		},
	)
	if err != nil {
		t.Fatalf("NewDeliveryRegisteredEventV0: %v", err)
	}
	requested, err := orquestacoreworkflow.NewReviewRequestedEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 2, "review-request"),
		orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: refs.ReviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     refs.DeliveryRef,
			Summary:         "Revision requerida para entrega de programacion.",
			EvidenceRefs:    []string{"artifact-ref-rt-review-request"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewRequestedEventV0: %v", err)
	}
	recorded, err := orquestacoreworkflow.NewReviewResultRecordedEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 3, "review-result"),
		orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: refs.ReviewResultRef,
			ReviewRequestID: refs.ReviewRequestRef,
			DeliveryRef:     refs.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Revision aceptada; ejecutar tests requeridos.",
			EvidenceRefs:    []string{"artifact-ref-rt-review-result"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewResultRecordedEventV0: %v", err)
	}
	accepted, err := orquestacoreworkflow.NewReviewAcceptedEventV0(
		codexStackRequiredTestEventMetaV0(refs.RunRef, 4, "review-accepted"),
		orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: refs.AcceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   refs.ReviewRequestRef,
			DeliveryRef:       refs.DeliveryRef,
			Summary:           "Revision aceptada con cadena causal completa.",
			EvidenceRefs:      []string{"artifact-ref-rt-review-accepted"},
		},
	)
	if err != nil {
		t.Fatalf("NewReviewAcceptedEventV0: %v", err)
	}
	return []orquestacoreworkflow.OrchestrationEventV0{delivery, requested, recorded, accepted}
}

func (refs codexStackRequiredTestRefsV0) workflowTaskV0() orquestacoreworkflow.WorkflowTaskV0 {
	refs = refs.withDefaultsV0()
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             refs.TaskRef,
		RunID:              refs.RunRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Implementar modulo Go con tests",
		Summary:            "Microtarea de programacion con cierre causal.",
		WriteSet:           []string{"go.mod", "calc.go", "calc_test.go"},
		AcceptanceCriteria: []string{"operational_director.required_tests passed", "codigo Go probado"},
		RequiredTests:      []string{"go test ./..."},
		WaveRef:            "wave-rt-runner-001",
		CohortRef:          "cohort-rt-runner-001",
	}
}

func (refs codexStackRequiredTestRefsV0) planStateV0() orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	refs = refs.withDefaultsV0()
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:    orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:         "state-rt-runner-001",
		PlanRef:          refs.PlanRef,
		RequestRef:       "request-rt-runner-001",
		RunRef:           refs.RunRef,
		ProjectRef:       "project-rt-runner-001",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:           orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:     "step-run-required-tests",
		ActiveWaveRef:    "wave-rt-runner-001",
		ActiveCohortRef:  "cohort-rt-runner-001",
		RequiredTestRefs: []string{"go test ./..."},
		ObservedAt:       "2026-05-22T16:59:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-rt-runner-001",
				CohortRef: "cohort-rt-runner-001",
				TaskRefs:  []string{refs.TaskRef},
				AgentRefs: []string{refs.AgentRef},
			},
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:            "wave-rt-runner-001",
				CohortRef:          "cohort-rt-runner-001",
				TaskRefs:           []string{refs.TaskRef},
				AgentRefs:          []string{refs.AgentRef},
				DeliveryRefs:       []string{refs.DeliveryRef},
				ReviewResultRefs:   []string{refs.ReviewResultRef},
				AcceptedReviewRefs: []string{refs.AcceptedReviewRef},
			},
			{
				StepID:           "step-run-required-tests",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          "wave-rt-runner-001",
				CohortRef:        "cohort-rt-runner-001",
				TaskRefs:         []string{refs.TaskRef},
				AgentRefs:        []string{refs.AgentRef},
				DeliveryRefs:     []string{refs.DeliveryRef},
				ReviewResultRefs: []string{refs.ReviewResultRef},
			},
			{
				StepID:    "step-replan-or-close",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:   "wave-rt-runner-001",
				CohortRef: "cohort-rt-runner-001",
			},
		},
	}
}

func codexStackRealRequiredTestRunnerStackV0(
	t *testing.T,
	cfg codexStackRealSmokeConfigV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	evidenceStore orquestacionnucleoapp.RequiredTestEvidenceStorePortV0,
	goCommand string,
	outputDir string,
) StackV0 {
	t.Helper()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	runMemory := orquestarunmemory.NewRunMemoryStoreV0()
	stack, err := BuildStackV0(ConfigV0{
		Enabled: true,
		Timeout: cfg.Timeout,
		DirectorLimits: orquestaweb.WebArrancarDirectorAppLimitsV0{
			MaxBursts:            4,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 2,
			MaxCommands:          8,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     1,
		},
		Stores: StoresV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			OutboxLedger:               orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
			TaskStore:                  taskStore,
			WaitStateStore:             waitStore,
			OperationalPlanStateWriter: planStore,
			OperationalPlanStateStore:  planStore,
			RequiredTestEvidenceStore:  evidenceStore,
			AppChangeStore:             orquestaappchange.NewInMemoryAppChangeStoreV0(),
			ReceiptStore:               orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			ProgressState:              orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
			ProcessRegistry:            orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0(),
			RunControl:                 runMemory,
			RunQueue:                   runMemory,
		},
		Codex: CodexRuntimeConfigV0{
			CommandPath:     cfg.CommandPath,
			ProjectWorkDir:  cfg.ProjectWorkDir,
			RuntimeWorkDir:  cfg.RuntimeWorkDir,
			CodeHomeDir:     cfg.CodeHomeDir,
			HomeDir:         cfg.HomeDir,
			PathEnv:         cfg.PathEnv,
			Model:           cfg.Model,
			ReasoningEffort: cfg.ReasoningEffort,
			Profile:         cfg.Profile,
			Sandbox:         cfg.Sandbox,
			ApprovalPolicy:  cfg.ApprovalPolicy,
			ExtraArgs:       cfg.ExtraArgs,
			PromptHints: []string{
				"Smoke opt-in: validar cableado RequiredTestRunner sin lanzar Codex real.",
			},
			Runtime:        processRuntime,
			ProcessStopper: processRuntime,
			SnapshotSource: processRuntime,
			MaxBatchReady:  1,
			MaxConcurrency: 1,
			WaitInterval:   time.Second,
		},
		Capacity: CapacityConfigV0{
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			OccurredAt:      "2026-05-22T17:00:00Z",
			RequestedBy:     "orquesta-app-stack-required-test-smoke",
			Summary:         "Capacidad real opt-in para required tests sin lanzar Codex.",
			EvidenceRefs:    []string{"evidence-ref-rt-runner"},
		},
		ReviewGate: ReviewGateConfigV0{
			FileEvidence: orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
		},
		RequiredTests: orquestacionnucleoapp.RequiredTestRunnerV0{
			Executor: orquestaruntimerequiredtest.LocalCommandExecutorV0{
				ProjectWorkDir: cfg.ProjectWorkDir,
				OutputDir:      outputDir,
				AllowedCommands: map[string]string{
					"go": goCommand,
				},
				Env:            []string{"CGO_ENABLED=0", "GOCACHE=" + filepath.Join(outputDir, "go-cache")},
				MaxOutputBytes: 1024 * 1024,
			},
			EvidenceWriter: evidenceStore,
		},
	})
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack
}

func writeCodexStackRequiredTestTinyGoModuleV0(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"go.mod": "module example.com/orquesta-rt-runner\n\ngo 1.22\n",
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
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func codexStackRequiredTestGoCommandV0(t *testing.T) string {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_GO_COMMAND"))
	if raw == "" {
		raw = "go"
	}
	if filepath.IsAbs(raw) {
		return raw
	}
	path, err := exec.LookPath(raw)
	if err != nil {
		t.Fatalf("go command no encontrado: %s", raw)
	}
	return path
}

func codexStackRequiredTestOutputDirV0(t *testing.T) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv("ORQUESTA_REQUIRED_TEST_OUTPUT_DIR"))
	if value == "" {
		return filepath.Join(t.TempDir(), "required-test-output")
	}
	return value
}

func codexStackRequiredTestPhasesV0() []orquestacoreworkflow.OrchestrationPhaseV0 {
	capacity := orquestacoreworkflow.OrchestrationCapacityHighV0
	return []orquestacoreworkflow.OrchestrationPhaseV0{
		{ID: orquestacoreworkflow.OrchestrationPhaseProgramacionV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseRevisionV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusActiveV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
		{ID: orquestacoreworkflow.OrchestrationPhaseCierreV0, Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0, RecommendedCapacity: capacity},
	}
}

func codexStackRequiredTestEventMetaV0(
	runRef string,
	sequence int64,
	suffix string,
) orquestacoreworkflow.OrchestrationEventMetaV0 {
	return orquestacoreworkflow.OrchestrationEventMetaV0{
		EventID:       "evt-rt-runner-" + suffix,
		RunID:         runRef,
		Sequence:      sequence,
		CorrelationID: "corr-rt-runner-seed",
		OccurredAt:    "2026-05-22T16:59:00Z",
	}
}

func codexStackRequiredTestPlanStepV0(
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

func codexStackRequiredTestOutputRefV0(refs []string) string {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, "required-test-output-v0/") {
			return ref
		}
	}
	return ""
}

func codexStackRequiredTestRuntimeEntriesV0(t *testing.T, runtimeDir string) []string {
	t.Helper()
	entries := []string{}
	err := filepath.WalkDir(runtimeDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == runtimeDir {
			return nil
		}
		rel, err := filepath.Rel(runtimeDir, path)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk runtime dir: %v", err)
	}
	return entries
}
