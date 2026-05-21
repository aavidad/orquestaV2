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
		work.MaxParallelItems != result.Plan.MaxParallelAgents ||
		len(work.Waves) != len(result.Plan.Steps) {
		t.Fatalf("work=%+v plan=%+v", work, result.Plan)
	}
	launchItem := workItemByKindV0(work, OperationalDirectorStepLaunchSubagentsV0)
	if launchItem.ItemID != "work-item-step-launch-subagents" ||
		launchItem.WorkProfileKind != "implementation" ||
		!stringInSetV0(launchItem.DependsOn, "work-item-step-split-work") ||
		len(launchItem.WriteSet) != 2 ||
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
