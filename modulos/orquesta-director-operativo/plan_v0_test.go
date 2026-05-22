package orquestadirectoroperativo

import (
	"strings"
	"testing"
)

func TestBuildOperationalDirectorPlanV0ProgrammingReady(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(nil))

	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		t.Fatalf("result=%+v", result)
	}
	if result.Plan.Mode != OperationalDirectorModeProgrammingV0 ||
		result.Plan.Status != OperationalDirectorPlanReadyV0 ||
		result.Plan.LoopBudget != DefaultOperationalDirectorMaxLoopsV0 ||
		result.Plan.MaxParallelAgents != DefaultOperationalDirectorMaxParallelAgentsV0 {
		t.Fatalf("plan=%+v", result.Plan)
	}
	for _, kind := range []OperationalDirectorStepKindV0{
		OperationalDirectorStepLaunchSubagentsV0,
		OperationalDirectorStepWaitSubagentsV0,
		OperationalDirectorStepReviewDeliveriesV0,
		OperationalDirectorStepRunRequiredTestsV0,
		OperationalDirectorStepReplanOrCloseV0,
	} {
		if !planHasStepKindV0(result.Plan, kind) {
			t.Fatalf("missing step %s in %+v", kind, result.Plan.Steps)
		}
	}
	if count := planStepKindCountV0(result.Plan, OperationalDirectorStepLaunchSubagentsV0); count != DefaultOperationalDirectorMaxParallelAgentsV0 {
		t.Fatalf("launch steps=%d plan=%+v", count, result.Plan.Steps)
	}
	launchStep := planStepByKindV0(result.Plan, OperationalDirectorStepLaunchSubagentsV0)
	if launchStep.Status != OperationalDirectorStepPendingV0 ||
		launchStep.WorkProfileKind != "implementation" ||
		len(launchStep.WriteSet) != 1 ||
		!stringInSetV0(launchStep.RequiredTests, "go test -count=1 ./modulos/orquesta-director-operativo") ||
		!containsFragmentInSetV0(launchStep.AcceptanceCriteria, "objetivo actual") {
		t.Fatalf("launch step=%+v", launchStep)
	}
}

func TestBuildOperationalDirectorPlanV0ProgrammingRejectsUnboundedWork(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(func(request *OperationalDirectorRequestV0) {
		request.WorktreeIsolated = false
		request.BranchRef = ""
		request.WriteSet = nil
		request.RequiredTests = nil
	}))

	if result.Accepted {
		t.Fatalf("accepted=true result=%+v", result)
	}
	assertIssueV0(t, result.Issues, "worktree_not_isolated")
	assertIssueV0(t, result.Issues, "branch_ref_missing")
	assertIssueV0(t, result.Issues, "write_set_missing")
	assertIssueV0(t, result.Issues, "required_tests_missing")
}

func TestBuildOperationalDirectorPlanV0DomainInsufficientContextBlocksLaunch(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.ContextStatus = OperationalDirectorContextInsufficientV0
		request.MissingContext = []string{"topic_outline", "source_refs"}
		request.DomainRefs = nil
	}))

	if !result.Accepted || result.ReadyToLaunch || !result.Blocked {
		t.Fatalf("result=%+v", result)
	}
	if result.Plan.Status != OperationalDirectorPlanNeedsContextV0 ||
		!planHasStepKindV0(result.Plan, OperationalDirectorStepRequestDomainContextV0) ||
		planHasStepKindV0(result.Plan, OperationalDirectorStepLaunchSubagentsV0) {
		t.Fatalf("plan=%+v", result.Plan)
	}
	requestContextStep := planStepByKindV0(result.Plan, OperationalDirectorStepRequestDomainContextV0)
	if requestContextStep.WorkProfileKind != "code_study" ||
		!stringInSetV0(requestContextStep.EvidenceRefs, "topic_outline") ||
		!stringInSetV0(requestContextStep.EvidenceRefs, "source_refs") {
		t.Fatalf("request context step=%+v", requestContextStep)
	}
}

func TestBuildOperationalDirectorPlanV0DomainReadyUsesSameOperationalLoop(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(nil))

	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		t.Fatalf("result=%+v", result)
	}
	for _, kind := range []OperationalDirectorStepKindV0{
		OperationalDirectorStepSplitWorkV0,
		OperationalDirectorStepLaunchSubagentsV0,
		OperationalDirectorStepWaitSubagentsV0,
		OperationalDirectorStepReviewDeliveriesV0,
		OperationalDirectorStepReplanOrCloseV0,
	} {
		if !planHasStepKindV0(result.Plan, kind) {
			t.Fatalf("missing step %s in %+v", kind, result.Plan.Steps)
		}
	}
	if planHasStepKindV0(result.Plan, OperationalDirectorStepRunRequiredTestsV0) {
		t.Fatalf("domain plan should not force programming tests: %+v", result.Plan.Steps)
	}
}

func TestBuildOperationalDirectorPlanV0DomainReadyShardsWriteSetByMaxParallelAgents(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.MaxParallelAgents = 3
		request.WriteSet = []string{
			"external/domain-work/job-001/topic-01",
			"external/domain-work/job-001/topic-02",
			"external/domain-work/job-001/topic-03",
			"external/domain-work/job-001/topic-04",
			"external/domain-work/job-001/topic-05",
		}
	}))

	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		t.Fatalf("result=%+v", result)
	}
	launchSteps := planStepsByKindV0(result.Plan, OperationalDirectorStepLaunchSubagentsV0)
	if len(launchSteps) != 3 {
		t.Fatalf("launch steps=%+v", launchSteps)
	}
	assertStringSetEqualsV0(t, launchSteps[0].WriteSet,
		"external/domain-work/job-001/topic-01",
		"external/domain-work/job-001/topic-04",
	)
	assertStringSetEqualsV0(t, launchSteps[1].WriteSet,
		"external/domain-work/job-001/topic-02",
		"external/domain-work/job-001/topic-05",
	)
	assertStringSetEqualsV0(t, launchSteps[2].WriteSet,
		"external/domain-work/job-001/topic-03",
	)
	for _, step := range launchSteps {
		if step.WorkProfileKind != "domain_work" ||
			!stringInSetV0(step.DomainRefs, "domain-job-ref-001") ||
			!containsFragmentInSetV0(step.AcceptanceCriteria, "write-set asignado es ownership inicial") {
			t.Fatalf("launch step=%+v", step)
		}
	}
}

func TestBuildOperationalDirectorPlanV0DomainReadyRejectsMissingWriteSet(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.WriteSet = nil
	}))

	if result.Accepted {
		t.Fatalf("accepted=true result=%+v", result)
	}
	assertIssueV0(t, result.Issues, "write_set_missing")
}

func TestBuildOperationalDirectorPlanV0RecursiveDelegationIsDirectorGoverned(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.ProjectRef = "opes"
		request.Objective = "Crear temario completo con agentes por tema y subagentes por bloque."
		request.AllowRecursiveDelegation = true
		request.MaxDelegationDepth = 99
		request.MaxSubagentsPerAgent = 99
		request.MaxRecursiveAgents = MaxOperationalDirectorMaxRecursiveAgentsV0 + 1
	}))

	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		t.Fatalf("result=%+v", result)
	}
	if !result.Plan.RecursiveDelegation ||
		result.Plan.MaxDelegationDepth != MaxOperationalDirectorMaxDelegationDepthV0 ||
		result.Plan.MaxSubagentsPerAgent != MaxOperationalDirectorMaxSubagentsPerAgentV0 ||
		result.Plan.MaxRecursiveAgents != MaxOperationalDirectorMaxRecursiveAgentsV0 ||
		!planHasStepKindV0(result.Plan, OperationalDirectorStepGovernDelegationV0) {
		t.Fatalf("plan=%+v", result.Plan)
	}
	governStep := planStepByKindV0(result.Plan, OperationalDirectorStepGovernDelegationV0)
	if !stringInSetV0(governStep.DependsOn, "step-wait-subagents") {
		t.Fatalf("govern step=%+v", governStep)
	}
	if governStep.ParentStepID != "step-launch-subagents" ||
		governStep.DelegationDepth != 1 ||
		governStep.MaxChildAgents != MaxOperationalDirectorMaxSubagentsPerAgentV0 ||
		!containsFragmentInSetV0(governStep.AcceptanceCriteria, "parent_ref") ||
		!containsFragmentInSetV0(governStep.AcceptanceCriteria, "MaxRecursiveAgents") {
		t.Fatalf("govern step=%+v", governStep)
	}
	reviewStep := planStepByKindV0(result.Plan, OperationalDirectorStepReviewDeliveriesV0)
	if !stringInSetV0(reviewStep.DependsOn, "step-govern-delegation") {
		t.Fatalf("review step=%+v", reviewStep)
	}
}

func TestBuildOperationalDirectorPlanV0RecursiveAgentBudgetZeroMeansUnbounded(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(func(request *OperationalDirectorRequestV0) {
		request.AllowRecursiveDelegation = true
		request.MaxRecursiveAgents = 0
	}))

	if !result.Accepted {
		t.Fatalf("result=%+v", result)
	}
	if result.Plan.MaxRecursiveAgents != 0 {
		t.Fatalf("max_recursive_agents=%d", result.Plan.MaxRecursiveAgents)
	}
}

func TestBuildOperationalDirectorPlanV0ClampsBudgetsAndDeduplicates(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validProgrammingRequestV0(func(request *OperationalDirectorRequestV0) {
		request.MaxLoops = 99
		request.MaxParallelAgents = 99
		request.WriteSet = append(request.WriteSet, request.WriteSet[0], " ")
	}))

	if !result.Accepted {
		t.Fatalf("result=%+v", result)
	}
	if result.Plan.LoopBudget != MaxOperationalDirectorMaxLoopsV0 ||
		result.Plan.MaxParallelAgents != MaxOperationalDirectorMaxParallelAgentsV0 ||
		len(result.Plan.WriteSet) != 2 ||
		planStepKindCountV0(result.Plan, OperationalDirectorStepLaunchSubagentsV0) != MaxOperationalDirectorMaxParallelAgentsV0 {
		t.Fatalf("plan=%+v", result.Plan)
	}
	waitStep := planStepByKindV0(result.Plan, OperationalDirectorStepWaitSubagentsV0)
	if len(waitStep.DependsOn) != MaxOperationalDirectorMaxParallelAgentsV0 ||
		!stringInSetV0(waitStep.DependsOn, "step-launch-subagents-06") {
		t.Fatalf("wait step=%+v", waitStep)
	}
}

func validProgrammingRequestV0(
	mutate func(*OperationalDirectorRequestV0),
) OperationalDirectorRequestV0 {
	request := OperationalDirectorRequestV0{
		RequestRef:       "req-director-operativo-001",
		RunRef:           "run-director-operativo-001",
		ProjectRef:       "orquesta",
		Objective:        "Implementar un corte pequeno del director operativo.",
		Mode:             OperationalDirectorModeProgrammingV0,
		WorktreeRef:      "worktree-ref-orquesta-director-operativo",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-director-operativo",
		WriteSet: []string{
			"modulos/orquesta-director-operativo/plan_v0.go",
			"modulos/orquesta-director-operativo/plan_v0_test.go",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-director-operativo"},
	}
	if mutate != nil {
		mutate(&request)
	}
	return request
}

func validDomainWorkRequestV0(
	mutate func(*OperationalDirectorRequestV0),
) OperationalDirectorRequestV0 {
	request := OperationalDirectorRequestV0{
		RequestRef:    "req-director-dominio-001",
		RunRef:        "run-director-dominio-001",
		ProjectRef:    "external-domain",
		Objective:     "Resolver trabajo documental externo con artefacto validable.",
		Mode:          OperationalDirectorModeDomainWorkV0,
		ContextStatus: OperationalDirectorContextSufficientV0,
		DomainRefs:    []string{"domain-job-ref-001", "domain-topic-ref-001"},
		WriteSet:      []string{"external/domain-work/domain-job-ref-001"},
	}
	if mutate != nil {
		mutate(&request)
	}
	return request
}

func planHasStepKindV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) bool {
	for _, step := range plan.Steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}

func planStepByKindV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) OperationalDirectorStepV0 {
	for _, step := range plan.Steps {
		if step.Kind == kind {
			return step
		}
	}
	return OperationalDirectorStepV0{}
}

func planStepsByKindV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) []OperationalDirectorStepV0 {
	var steps []OperationalDirectorStepV0
	for _, step := range plan.Steps {
		if step.Kind == kind {
			steps = append(steps, step)
		}
	}
	return steps
}

func planStepKindCountV0(
	plan OperationalDirectorPlanV0,
	kind OperationalDirectorStepKindV0,
) int {
	count := 0
	for _, step := range plan.Steps {
		if step.Kind == kind {
			count++
		}
	}
	return count
}

func stringInSetV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsFragmentInSetV0(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func assertIssueV0(
	t *testing.T,
	issues []OperationalDirectorIssueV0,
	code string,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q not found in %+v", code, issues)
}

func assertStringSetEqualsV0(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
	for _, value := range want {
		if !stringInSetV0(got, value) {
			t.Fatalf("got=%+v want=%+v", got, want)
		}
	}
}
