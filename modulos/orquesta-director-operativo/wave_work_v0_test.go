package orquestadirectoroperativo

import "testing"

func TestBuildOperationalDirectorWaveWorkV0ProgrammingPreservesOperationalContract(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(nil))
	work := BuildOperationalDirectorWaveWorkV0(result.Plan)

	if len(work.Issues) > 0 {
		t.Fatalf("issues=%+v", work.Issues)
	}
	if !work.ReadyToLaunch ||
		work.PlanRef != result.Plan.PlanRef ||
		work.MaxParallelItems != result.Plan.MaxParallelAgents {
		t.Fatalf("work=%+v plan=%+v", work, result.Plan)
	}
	launchWave := waveByItemKindV0(work, OperationalDirectorStepLaunchSubagentsV0)
	if len(launchWave.Items) != result.Plan.MaxParallelAgents {
		t.Fatalf("launch wave=%+v plan=%+v", launchWave, result.Plan)
	}
	launchItem := workItemByKindV0(work, OperationalDirectorStepLaunchSubagentsV0)
	if launchItem.ItemID != "work-item-step-launch-subagents" ||
		launchItem.WorkProfileKind != "implementation" ||
		!stringInSetV0(launchItem.DependsOn, "work-item-step-split-work") ||
		len(launchItem.WriteSet) != 1 ||
		!stringInSetV0(launchItem.RequiredTests, "go test -count=1 ./modulos/orquesta-director-operativo") ||
		!containsFragmentInSetV0(launchItem.AcceptanceCriteria, "objetivo actual") {
		t.Fatalf("launch item=%+v", launchItem)
	}
	testItem := workItemByKindV0(work, OperationalDirectorStepRunRequiredTestsV0)
	if testItem.WorkProfileKind != "required_tests" ||
		!stringInSetV0(testItem.DependsOn, "work-item-step-review-deliveries") ||
		!stringInSetV0(testItem.RequiredTests, "go test -count=1 ./modulos/orquesta-director-operativo") {
		t.Fatalf("test item=%+v", testItem)
	}
}

func TestBuildOperationalDirectorWaveWorkV0DomainNeedsContextDoesNotExposeLaunchItem(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.ContextStatus = OperationalDirectorContextInsufficientV0
		request.MissingContext = []string{"topic_outline"}
		request.DomainRefs = nil
	}))
	work := BuildOperationalDirectorWaveWorkV0(result.Plan)

	if len(work.Issues) > 0 {
		t.Fatalf("issues=%+v", work.Issues)
	}
	if work.ReadyToLaunch || work.Status != OperationalDirectorPlanNeedsContextV0 {
		t.Fatalf("work=%+v", work)
	}
	if workHasItemKindV0(work, OperationalDirectorStepLaunchSubagentsV0) {
		t.Fatalf("launch item should not exist in blocked context work: %+v", work.Waves)
	}
	requestContextItem := workItemByKindV0(work, OperationalDirectorStepRequestDomainContextV0)
	if !stringInSetV0(requestContextItem.EvidenceRefs, "topic_outline") {
		t.Fatalf("request context item=%+v", requestContextItem)
	}
}

func TestBuildOperationalDirectorWaveWorkV0DomainLaunchItemsKeepShardedWriteSet(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.MaxParallelAgents = 3
		request.WriteSet = []string{
			"external/domain-work/job-001/block-a",
			"external/domain-work/job-001/block-b",
			"external/domain-work/job-001/block-c",
			"external/domain-work/job-001/block-d",
		}
	}))
	work := BuildOperationalDirectorWaveWorkV0(result.Plan)

	if len(work.Issues) > 0 {
		t.Fatalf("issues=%+v", work.Issues)
	}
	launchWave := waveByItemKindV0(work, OperationalDirectorStepLaunchSubagentsV0)
	if len(launchWave.Items) != 3 {
		t.Fatalf("launch wave=%+v", launchWave)
	}
	assertStringSetEqualsV0(t, launchWave.Items[0].WriteSet,
		"external/domain-work/job-001/block-a",
		"external/domain-work/job-001/block-d",
	)
	assertStringSetEqualsV0(t, launchWave.Items[1].WriteSet, "external/domain-work/job-001/block-b")
	assertStringSetEqualsV0(t, launchWave.Items[2].WriteSet, "external/domain-work/job-001/block-c")
	for _, item := range launchWave.Items {
		if item.WorkProfileKind != "domain_work" ||
			!stringInSetV0(item.DomainRefs, "domain-job-ref-001") ||
			!containsFragmentInSetV0(item.AcceptanceCriteria, "write-set asignado es ownership inicial") {
			t.Fatalf("launch item=%+v", item)
		}
	}
}

func TestBuildOperationalDirectorWaveWorkV0GroupsIndependentItemsInSameWave(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(nil))
	plan := result.Plan
	plan.Steps = append(plan.Steps, OperationalDirectorStepV0{
		StepID:        "step-review-contract",
		Kind:          OperationalDirectorStepReviewDeliveriesV0,
		Status:        OperationalDirectorStepPendingV0,
		Title:         "Revisar contrato externo en paralelo",
		DependsOn:     []string{"step-wait-subagents"},
		DomainRefs:    []string{"domain-job-ref-001"},
		EvidenceRefs:  []string{"artifact-ref-001"},
		RequiredTests: []string{"validator-ref-001"},
	})

	work := BuildOperationalDirectorWaveWorkV0(plan)

	if len(work.Issues) > 0 {
		t.Fatalf("issues=%+v", work.Issues)
	}
	reviewWave := waveByItemKindV0(work, OperationalDirectorStepReviewDeliveriesV0)
	if reviewWave.WaveID == "" || len(reviewWave.Items) != 2 {
		t.Fatalf("review wave=%+v work=%+v", reviewWave, work)
	}
	for _, item := range reviewWave.Items {
		if !stringInSetV0(item.DependsOn, "work-item-step-wait-subagents") {
			t.Fatalf("item=%+v", item)
		}
	}
}

func TestBuildOperationalDirectorWaveWorkV0ReportsUnresolvableDependencies(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(nil))
	plan := result.Plan
	plan.Steps[0].DependsOn = []string{"step-missing"}

	work := BuildOperationalDirectorWaveWorkV0(plan)

	assertIssueV0(t, work.Issues, "step_dependency_cycle")
}

func TestBuildOperationalDirectorWaveWorkV0RejectsInvalidDelegationRefsAndBudgets(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.AllowRecursiveDelegation = true
		request.MaxDelegationDepth = 2
		request.MaxSubagentsPerAgent = 2
		request.MaxRecursiveAgents = 12
	}))
	plan := result.Plan
	governIndex := planStepIndexByKindV0(plan, OperationalDirectorStepGovernDelegationV0)
	plan.Steps[governIndex].ParentStepID = "step-missing-parent"
	plan.Steps[governIndex].ChildStepIDs = []string{"step-missing-child"}
	plan.Steps[governIndex].DelegationDepth = plan.MaxDelegationDepth + 1
	plan.Steps[governIndex].MaxChildAgents = plan.MaxSubagentsPerAgent + 1

	work := BuildOperationalDirectorWaveWorkV0(plan)

	if work.ReadyToLaunch {
		t.Fatalf("invalid delegation plan should not be ready: %+v", work)
	}
	assertIssueV0(t, work.Issues, "parent_step_missing")
	assertIssueV0(t, work.Issues, "child_step_missing")
	assertIssueV0(t, work.Issues, "delegation_depth_exceeds_budget")
	assertIssueV0(t, work.Issues, "max_child_agents_exceeds_budget")
	if work.MaxRecursiveAgents != 12 {
		t.Fatalf("max_recursive_agents no preservado: work=%+v", work)
	}
}

func TestBuildOperationalDirectorWaveWorkV0RejectsReadyPlanWithoutOperationalCycle(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(nil))
	plan := result.Plan
	var trimmed []OperationalDirectorStepV0
	for _, step := range plan.Steps {
		if step.Kind == OperationalDirectorStepWaitSubagentsV0 ||
			step.Kind == OperationalDirectorStepReviewDeliveriesV0 ||
			step.Kind == OperationalDirectorStepRunRequiredTestsV0 {
			continue
		}
		trimmed = append(trimmed, step)
	}
	plan.Steps = trimmed

	work := BuildOperationalDirectorWaveWorkV0(plan)

	if work.ReadyToLaunch {
		t.Fatalf("incomplete operational cycle should not be ready: %+v", work)
	}
	assertIssueV0(t, work.Issues, "ready_step_missing")
	assertIssueV0(t, work.Issues, "run_required_tests_missing")
}

func TestBuildOperationalDirectorWaveWorkV0RejectsDomainReadyPlanWithoutOpaqueScope(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(nil))
	plan := result.Plan
	plan.DomainRefs = nil
	plan.WriteSet = nil

	work := BuildOperationalDirectorWaveWorkV0(plan)

	if work.ReadyToLaunch {
		t.Fatalf("domain work without refs/write-set should not be ready: %+v", work)
	}
	assertIssueV0(t, work.Issues, "domain_refs_missing")
	assertIssueV0(t, work.Issues, "write_set_missing")
}

func TestBuildOperationalDirectorWaveWorkV0RejectsDomainReadyPlanWithUnsafeWriteSet(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(nil))
	plan := result.Plan
	plan.WriteSet = []string{"../opes-interno"}

	work := BuildOperationalDirectorWaveWorkV0(plan)

	if work.ReadyToLaunch {
		t.Fatalf("domain work with unsafe write-set should not be ready: %+v", work)
	}
	assertIssueV0(t, work.Issues, "write_set_path_invalid")
}

func TestBuildOperationalDirectorWaveWorkV0RejectsNeedsContextPlanWithLaunch(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.ContextStatus = OperationalDirectorContextInsufficientV0
		request.MissingContext = []string{"source_refs"}
		request.DomainRefs = nil
	}))
	plan := result.Plan
	plan.Steps = append(plan.Steps, OperationalDirectorStepV0{
		StepID: "step-launch-illegal",
		Kind:   OperationalDirectorStepLaunchSubagentsV0,
		Status: OperationalDirectorStepPendingV0,
		Title:  "Lanzamiento no permitido sin contexto",
	})

	work := BuildOperationalDirectorWaveWorkV0(plan)

	if work.ReadyToLaunch {
		t.Fatalf("needs_context plan should not be ready: %+v", work)
	}
	assertIssueV0(t, work.Issues, "needs_context_launch_forbidden")
}

func workItemByKindV0(
	work OperationalDirectorWaveWorkV0,
	kind OperationalDirectorStepKindV0,
) OperationalDirectorWorkItemV0 {
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.Kind == kind {
				return item
			}
		}
	}
	return OperationalDirectorWorkItemV0{}
}

func workHasItemKindV0(
	work OperationalDirectorWaveWorkV0,
	kind OperationalDirectorStepKindV0,
) bool {
	return workItemByKindV0(work, kind).ItemID != ""
}

func waveByItemKindV0(
	work OperationalDirectorWaveWorkV0,
	kind OperationalDirectorStepKindV0,
) OperationalDirectorWaveV0 {
	for _, wave := range work.Waves {
		for _, item := range wave.Items {
			if item.Kind == kind {
				return wave
			}
		}
	}
	return OperationalDirectorWaveV0{}
}

func planStepIndexByKindV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) int {
	for index, step := range plan.Steps {
		if step.Kind == kind {
			return index
		}
	}
	return -1
}
