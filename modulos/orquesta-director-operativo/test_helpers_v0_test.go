package orquestadirectoroperativo

import (
	"strings"
	"testing"
)

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
