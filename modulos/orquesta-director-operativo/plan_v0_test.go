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
	launchStep := planStepByKindV0(result.Plan, OperationalDirectorStepLaunchSubagentsV0)
	if launchStep.Status != OperationalDirectorStepPendingV0 ||
		len(launchStep.WriteSet) != 2 ||
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
	if !stringInSetV0(requestContextStep.EvidenceRefs, "topic_outline") ||
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

func TestBuildOperationalDirectorPlanV0RecursiveDelegationIsDirectorGoverned(t *testing.T) {
	result := BuildOperationalDirectorPlanV0(validDomainWorkRequestV0(func(request *OperationalDirectorRequestV0) {
		request.ProjectRef = "opes"
		request.Objective = "Crear temario completo con agentes por tema y subagentes por bloque."
		request.AllowRecursiveDelegation = true
		request.MaxDelegationDepth = 99
		request.MaxSubagentsPerAgent = 99
	}))

	if !result.Accepted || !result.ReadyToLaunch || result.Blocked {
		t.Fatalf("result=%+v", result)
	}
	if !result.Plan.RecursiveDelegation ||
		result.Plan.MaxDelegationDepth != MaxOperationalDirectorMaxDelegationDepthV0 ||
		result.Plan.MaxSubagentsPerAgent != MaxOperationalDirectorMaxSubagentsPerAgentV0 ||
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
		!containsFragmentInSetV0(governStep.AcceptanceCriteria, "parent_ref") {
		t.Fatalf("govern step=%+v", governStep)
	}
	reviewStep := planStepByKindV0(result.Plan, OperationalDirectorStepReviewDeliveriesV0)
	if !stringInSetV0(reviewStep.DependsOn, "step-govern-delegation") {
		t.Fatalf("review step=%+v", reviewStep)
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
		len(result.Plan.WriteSet) != 2 {
		t.Fatalf("plan=%+v", result.Plan)
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
