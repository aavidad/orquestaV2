package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackAutoprogrammingPromotionV0PromocionaTrasCierreCausalV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	port := &fakeAutoprogrammingPromotionPortV0{}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, "run-autoprogramming-promotion-001", nil)

	run := mustLoadCodexStackRunForTestV0(t, stack, "run-autoprogramming-promotion-001")
	complete, refs, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, run)
	if err != nil {
		t.Fatalf("maybePromoteClosedAutoprogrammingRunV0: %v", err)
	}
	if !complete || port.promotions != 1 || port.archives != 1 || len(refs) == 0 {
		t.Fatalf("complete=%v refs=%v port=%+v", complete, refs, port)
	}
	if port.lastPromotion.WorktreeRef != "worktree-ref-autoprogramming-promotion-001" ||
		port.lastPromotion.BranchRef != "branch-ref-autoprogramming-promotion-001" {
		t.Fatalf("refs no preservadas: %+v", port.lastPromotion)
	}
}

func TestCodexStackAutoprogrammingPromotionV0QuedaPendientePorSolapeVivoV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	port := &fakeAutoprogrammingPromotionPortV0{}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, "run-autoprogramming-promotion-002", nil)
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, "run-autoprogramming-live-002", []string{"modulos/orquesta-app-codex-stack"})
	_, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-autoprogramming-live-002",
		QueueRef:      normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		Status:        "ready",
		PriorityScore: 50,
	})
	if err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	run := mustLoadCodexStackRunForTestV0(t, stack, "run-autoprogramming-promotion-002")
	complete, refs, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, run)
	if err != nil {
		t.Fatalf("maybePromoteClosedAutoprogrammingRunV0: %v", err)
	}
	if complete || port.promotions != 0 || port.archives != 0 || len(refs) == 0 {
		t.Fatalf("complete=%v refs=%v port=%+v", complete, refs, port)
	}
}

func seedAutoprogrammingPromotionRunV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	writeSetOverride []string,
) {
	t.Helper()
	writeSet := writeSetOverride
	if len(writeSet) == 0 {
		writeSet = []string{"modulos/orquesta-app-codex-stack"}
	}
	taskRef := runRef + "-task"
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Autoprogramming promotion test",
		WriteSet:           writeSet,
		AcceptanceCriteria: []string{"cierre causal y promocion controlada"},
		RequiredTests:      []string{"go test ./modulos/orquesta-app-codex-stack"},
		ContextRefs: []string{
			autoprogrammingBridgeOperationalTaskSourceRefV0,
			"worktree_ref:worktree-ref-autoprogramming-promotion-001",
			"branch_ref:branch-ref-autoprogramming-promotion-001",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{FunctionName: "BuildAutoprogrammingProgrammableWorkV0"}},
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:   orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:           runRef,
		ProjectRef:      "project-ref-autoprogramming-promotion-001",
		AppSpecRef:      "app-spec-ref-autoprogramming-promotion-001",
		Status:          orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseCierreV0,
		Tasks:           []string{taskRef},
		ClosedTasks:     []string{taskRef},
		AcceptedReviews: []string{"accepted-review-ref-" + runRef},
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	seedAutoprogrammingPromotionEvidenceV0(t, ctx, stack, runRef, taskRef)
}

func seedAutoprogrammingPromotionEvidenceV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
	taskRef string,
) {
	t.Helper()
	evidenceRef := "test-evidence-ref-" + runRef
	evidence := orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       evidenceRef,
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommand:       "go test ./modulos/orquesta-app-codex-stack",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       "delivery-ref-" + runRef,
		ReviewRequestID:   "review-request-ref-" + runRef,
		ReviewResultRef:   "review-result-ref-" + runRef,
		AcceptedReviewRef: "accepted-review-ref-" + runRef,
		OccurredAt:        "2026-05-24T10:00:00Z",
		EvidenceRefs:      []string{"evidence-ref-required-test-" + runRef},
	}
	if stack.Stores.RequiredTestEvidenceStore == nil {
		t.Fatalf("RequiredTestEvidenceStore requerido")
	}
	if err := stack.Stores.RequiredTestEvidenceStore.SaveRequiredTestEvidenceV0(ctx, evidence); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	if stack.Stores.OperationalPlanStateStore == nil {
		t.Fatalf("OperationalPlanStateStore requerido")
	}
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion: orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:      "state-ref-" + runRef,
		PlanRef:       autoprogrammingPromotionPlanRefV0(runRef),
		RunRef:        runRef,
		Mode:          orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:        orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0,
		ObservedAt:    "2026-05-24T10:00:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
			StepID:                   "step-run-required-tests",
			Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
			Status:                   orquestadirectoroperativo.OperationalDirectorStepClosedV0,
			RequiredTestEvidenceRefs: []string{evidenceRef},
		}},
	}
	if err := stack.Stores.OperationalPlanStateWriter.SaveOperationalDirectorPlanStateV0(ctx, state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
}

func withAutoprogrammingPromotionStoresForTestV0(stack StackV0) StackV0 {
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack.Stores.OperationalPlanStateStore = planStore
	stack.Stores.OperationalPlanStateWriter = planStore
	stack.Stores.RequiredTestEvidenceStore = evidenceStore
	stack.Ports.OperationalPlanStateStore = planStore
	stack.Ports.OperationalPlanStateWriter = planStore
	stack.Ports.RequiredTestEvidenceStore = evidenceStore
	return stack
}

type fakeAutoprogrammingPromotionPortV0 struct {
	promotions    int
	archives      int
	lastPromotion orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0
}

func (fake *fakeAutoprogrammingPromotionPortV0) PromoteAutoprogrammingStagingV0(
	_ context.Context,
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	fake.promotions++
	fake.lastPromotion = command
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
		SchemaVersion: orquestaautoprogramming.AutoprogrammingStagingPromotionSchemaVersionV0,
		Status:        orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
		EvidenceRefs:  []string{"evidence-ref-fake-promotion"},
	}, nil
}

func (fake *fakeAutoprogrammingPromotionPortV0) ArchiveAutoprogrammingStagingV0(
	_ context.Context,
	command orquestaautoprogramming.AutoprogrammingStagingCleanupCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	fake.archives++
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
		SchemaVersion: orquestaautoprogramming.AutoprogrammingStagingPromotionSchemaVersionV0,
		Status:        orquestaautoprogramming.AutoprogrammingStagingEffectArchivedV0,
		ArchiveRef:    command.ArchiveRef,
		EvidenceRefs:  []string{"evidence-ref-fake-archive"},
	}, nil
}
