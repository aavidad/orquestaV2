package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestagoal "orquesta/modulos/orquesta-goal"
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

func TestAutoprogrammingPromotionGoalRequiredTestEvidenceV0UsaAtestacionIndependienteV0(t *testing.T) {
	state := orquestagoal.GoalWorkStateV0{
		GoalRef: "goal-ref-promotion-attested-001",
		Spec: orquestagoal.GoalWorkSpecV0{RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "test-ref-promotion-attested-001",
			Command: "go test -count=1 ./modulos/orquesta-native-smoke-tool",
		}}},
		LastResult: &orquestagoal.GoalWorkResultV0{RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef: "test-ref-promotion-attested-001",
			Status:  orquestagoal.GoalRequiredTestAttestationStatusPassedV0,
		}}},
		LastClosure: &orquestagoal.GoalClosureValidationV0{
			Accepted: true,
			AttestationVerifications: []orquestagoal.GoalRequiredTestIdentityVerificationV0{{
				AttestationRef: "attestation-ref-promotion-independent-001",
				TestRef:        "test-ref-promotion-attested-001",
				Verified:       true,
				Independent:    true,
			}},
		},
	}

	evidence := autoprogrammingPromotionGoalRequiredTestEvidenceV0(state)
	if len(evidence) != 1 ||
		evidence[0].EvidenceRef != "attestation-ref-promotion-independent-001" ||
		evidence[0].TaskRef != state.GoalRef ||
		evidence[0].TestCommand != "go test -count=1 ./modulos/orquesta-native-smoke-tool" ||
		evidence[0].Status != orquestagoal.GoalRequiredTestAttestationStatusPassedV0 {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestAutoprogrammingPromotionGoalRequiredTestEvidenceV0RechazaAtestacionNoAutoritativaV0(t *testing.T) {
	base := orquestagoal.GoalWorkStateV0{
		GoalRef: "goal-ref-promotion-untrusted-001",
		Spec: orquestagoal.GoalWorkSpecV0{RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "test-ref-promotion-untrusted-001",
			Command: "go test ./modulos/orquesta-native-smoke-tool",
		}}},
		LastClosure: &orquestagoal.GoalClosureValidationV0{Accepted: true},
	}
	tests := []orquestagoal.GoalRequiredTestIdentityVerificationV0{
		{AttestationRef: "attestation-ref-unverified", TestRef: "test-ref-promotion-untrusted-001", Independent: true},
		{AttestationRef: "attestation-ref-dependent", TestRef: "test-ref-promotion-untrusted-001", Verified: true},
		{AttestationRef: "attestation-ref-unknown-test", TestRef: "test-ref-unknown", Verified: true, Independent: true},
		{TestRef: "test-ref-promotion-untrusted-001", Verified: true, Independent: true},
	}
	for _, verification := range tests {
		state := base
		closure := *base.LastClosure
		closure.AttestationVerifications = []orquestagoal.GoalRequiredTestIdentityVerificationV0{verification}
		state.LastClosure = &closure
		if evidence := autoprogrammingPromotionGoalRequiredTestEvidenceV0(state); len(evidence) != 0 {
			t.Fatalf("verification=%+v evidence=%+v", verification, evidence)
		}
	}
	base.LastClosure.Accepted = false
	base.LastClosure.AttestationVerifications = []orquestagoal.GoalRequiredTestIdentityVerificationV0{{
		AttestationRef: "attestation-ref-accepted-required",
		TestRef:        "test-ref-promotion-untrusted-001", Verified: true, Independent: true,
	}}
	if evidence := autoprogrammingPromotionGoalRequiredTestEvidenceV0(base); len(evidence) != 0 {
		t.Fatalf("cierre no aceptado produjo evidence=%+v", evidence)
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

func TestCodexStackAutoprogrammingPromotionV0QuedaPendientePorSolapeGoalFirstVivoV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalStateStore = goalStates
	stack.Stores.AppGoalStateStore = goalStates
	port := &fakeAutoprogrammingPromotionPortV0{}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, "run-autoprogramming-promotion-goal-overlap-001", []string{"feature.md"})
	seedAutoprogrammingPromotionLiveGoalFirstRunV0(t, ctx, stack, goalStates, "run-autoprogramming-live-goal-overlap-001", []string{"feature.md"})

	run := mustLoadCodexStackRunForTestV0(t, stack, "run-autoprogramming-promotion-goal-overlap-001")
	complete, refs, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, run)
	if err != nil {
		t.Fatalf("maybePromoteClosedAutoprogrammingRunV0: %v", err)
	}
	if complete || port.promotions != 0 || port.archives != 0 || len(refs) == 0 {
		t.Fatalf("complete=%v refs=%v port=%+v", complete, refs, port)
	}
}

func TestCodexStackAutoprogrammingPromotionV0NoCierraGoalFirstSinEstadoDurableV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	stack.Ports.GoalStateStore = goalStates
	stack.Ports.GoalFirstRunMarkerStore = goalStates
	stack.Stores.AppGoalStateStore = goalStates
	port := &fakeAutoprogrammingPromotionPortV0{}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	runRef := "run-autoprogramming-goal-first-missing-state-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-" + runRef,
		AppSpecRef:    "app-spec-ref-autoprogramming-missing-state",
		Status:        orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseCierreV0,
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	saveCodexStackGoalFirstRunMarkerForTestV0(t, ctx, goalStates, runRef)

	status, refs, err := stack.stackDrainQueueStatusAndEvidenceForCoordinatorV0(ctx, promotionE2ELoopResultV0(run))
	if err != nil {
		t.Fatalf("stackDrainQueueStatusAndEvidenceForCoordinatorV0: %v", err)
	}
	if status != "" {
		t.Fatalf("queue_status=%q want pending por GoalWorkState ausente", status)
	}
	if port.promotions != 0 || port.archives != 0 {
		t.Fatalf("port=%+v", port)
	}
	for _, want := range []string{
		"evidence-ref-codex-stack-autoprogramming-goal-first-state-missing",
		"evidence-ref-codex-stack-autoprogramming-goal-first-marker",
		"evidence-ref-goal-first-marker-" + runRef,
	} {
		if !codexStackStringInSetForTestV0(refs, want) {
			t.Fatalf("evidence_refs=%v missing %s", refs, want)
		}
	}
}

func TestCodexStackAutoprogrammingPromotionV0BloqueaRefsStagingInconsistentesV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	port := &fakeAutoprogrammingPromotionPortV0{}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	runRef := "run-autoprogramming-promotion-conflicting-refs-001"
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, runRef, nil)
	seedAutoprogrammingPromotionConflictingTaskV0(t, ctx, stack, runRef)

	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	complete, refs, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, run)
	if err != nil {
		t.Fatalf("maybePromoteClosedAutoprogrammingRunV0: %v", err)
	}
	if complete || port.promotions != 0 || port.archives != 0 || len(refs) == 0 {
		t.Fatalf("complete=%v refs=%v port=%+v", complete, refs, port)
	}
}

func TestCodexStackAutoprogrammingPromotionV0PendingPushExigeReciboIntegracionV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	port := &statusAutoprogrammingPromotionPortForTestV0{
		promoteStatus: orquestaautoprogramming.AutoprogrammingStagingEffectPendingPushV0,
	}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	runRef := "run-autoprogramming-pending-integration-001"
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, runRef, []string{"modulos/orquesta-app-codex-stack"})

	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	complete, refs, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, run)
	if err != nil {
		t.Fatalf("maybePromoteClosedAutoprogrammingRunV0: %v", err)
	}
	if complete || port.promotions != 1 || port.archives != 0 ||
		!codexStackStringInSetForTestV0(refs, "evidence-ref-autoprogramming-pending-integration") {
		t.Fatalf("complete=%v refs=%v port=%+v", complete, refs, port)
	}
}

func seedAutoprogrammingPromotionLiveGoalFirstRunV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	goalStates orquestagoal.GoalWorkStateStorePortV0,
	runRef string,
	writeSet []string,
) {
	t.Helper()
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-" + runRef,
		AppSpecRef:    "app-spec-ref-" + runRef,
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0 goal-first live: %v", err)
	}
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec: orquestagoal.GoalWorkSpecV0{
			GoalRef:      "goal-ref-" + runRef,
			RunRef:       runRef,
			RequestRef:   runRef,
			ProjectRef:   "project-ref-" + runRef,
			WorkKind:     orquestaautoprogramming.AutoprogrammingGoalWorkKindV0,
			Objective:    "Autoprogramacion viva con write-set solapado",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			ContextRefs: []orquestagoal.GoalContextRefV0{
				{Kind: "worktree", Ref: "worktree-ref-" + runRef, Required: true},
				{Kind: "branch", Ref: "branch-ref-" + runRef, Required: true},
			},
			WriteSet: autoprogrammingPromotionGoalWriteScopesForTestV0(writeSet),
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			Status:  orquestagoal.GoalStatusRunningV0,
			GoalRef: "goal-ref-" + runRef,
		},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	_, err = stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      normalizeRunQueueConfigV0(stack.RunQueue).QueueRef,
		Status:        orquestarunqueue.RunStatusRunningV0,
		PriorityScore: 50,
	})
	if err != nil {
		t.Fatalf("SetRunPriorityV0 goal-first live: %v", err)
	}
}

func autoprogrammingPromotionGoalWriteScopesForTestV0(paths []string) []orquestagoal.GoalWriteScopeV0 {
	out := make([]orquestagoal.GoalWriteScopeV0, 0, len(paths))
	for _, path := range paths {
		out = append(out, orquestagoal.GoalWriteScopeV0{Path: path})
	}
	return out
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

func seedAutoprogrammingPromotionConflictingTaskV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	taskRef := runRef + "-task-conflicting"
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Autoprogramming promotion conflicting staging refs",
		WriteSet:           []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{"no promocionar refs de staging inconsistentes"},
		RequiredTests:      []string{"go test ./modulos/orquesta-app-codex-stack"},
		ContextRefs: []string{
			autoprogrammingBridgeOperationalTaskSourceRefV0,
			"worktree_ref:worktree-ref-autoprogramming-promotion-conflicting",
			"branch_ref:branch-ref-autoprogramming-promotion-conflicting",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{FunctionName: "BuildAutoprogrammingProgrammableWorkV0"}},
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 conflicting: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	run.Tasks = append(run.Tasks, taskRef)
	run.ClosedTasks = append(run.ClosedTasks, taskRef)
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0 conflicting: %v", err)
	}
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
		SchemaVersion:         orquestaautoprogramming.AutoprogrammingStagingPromotionSchemaVersionV0,
		Status:                orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
		IntegrationStatus:     orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0,
		IntegrationReceiptRef: "integration-receipt-ref-fake-promotion",
		EvidenceRefs:          []string{"evidence-ref-fake-promotion"},
	}, nil
}

func TestAutoprogrammingPromotionEffectWithIntegrationStatusV0DoesNotInferIntegrationV0(t *testing.T) {
	result := autoprogrammingPromotionEffectWithIntegrationStatusV0(orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
		Status: orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
	})
	if result.IntegrationStatus != orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusPendingIntegrationV0 ||
		autoprogrammingPromotionEffectCompleteV0(result) {
		t.Fatalf("result=%+v", result)
	}
}

func TestAutoprogrammingPromotionGoalContextRefV0PrefijoVacioNoCapturaOtroKindV0(t *testing.T) {
	refs := []orquestagoal.GoalContextRefV0{
		{Kind: "request", Ref: "request-ref-no-es-workspace"},
		{Kind: "worktree", Ref: "worktree_ref:worktree-ref-legacy"},
	}
	if got := autoprogrammingPromotionGoalContextRefV0(refs, "goal_workspace", ""); got != "" {
		t.Fatalf("prefijo vacio capturo ref ajena: %q", got)
	}
	if got := autoprogrammingPromotionGoalContextRefV0(refs, "worktree_missing", "worktree_ref:"); got != "worktree-ref-legacy" {
		t.Fatalf("fallback legacy perdido: %q", got)
	}
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
