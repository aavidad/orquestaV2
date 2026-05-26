package orquestaautoprogramming

import (
	"fmt"
	"testing"
)

func TestValidateAutoprogrammingRequestV0AcceptsSmallIsolatedRequest(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(nil))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Groups) != 1 ||
		result.Groups[0].Area != "orchestration-core" ||
		len(result.Groups[0].TaskRefs) != 2 {
		t.Fatalf("groups=%+v", result.Groups)
	}
	if len(result.WriteSet) != 2 || len(result.RequiredTests) != 1 {
		t.Fatalf("write_set=%v required_tests=%v", result.WriteSet, result.RequiredTests)
	}
}

func TestValidateAutoprogrammingRequestV0RejectsMissingIsolationOrBranchRef(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.WorktreeIsolated = false
		request.BranchRef = " "
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "worktree_not_isolated")
	assertAutoprogrammingRequestIssueV0(t, result, "branch_ref_missing")
}

func TestValidateAutoprogrammingRequestV0RejectsBroadScope(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= AutoprogrammingRequestDefaultMaxTaskRefsV0+1; i++ {
			area := fmt.Sprintf("area-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: fmt.Sprintf("task-ref-broad-%02d", i),
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				fmt.Sprintf("modulos/orquesta-autoprogramming/%s/file.go", area),
			)
		}
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "scope_tasks_too_large")
	assertAutoprogrammingRequestIssueV0(t, result, "scope_areas_too_large")
	assertAutoprogrammingRequestIssueV0(t, result, "scope_write_set_too_large")
}

func TestValidateAutoprogrammingRequestV0AcceptsTenParentWaveByDefault(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= 10; i++ {
			taskRef := fmt.Sprintf("task-ref-parent-wave-%02d", i)
			area := fmt.Sprintf("parent-wave-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: taskRef,
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				"modulos/orquesta-autoprogramming/"+area+"/contrato.go",
			)
		}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Groups) != 10 || len(result.WriteSet) != 10 {
		t.Fatalf("groups=%d write_set=%d result=%+v", len(result.Groups), len(result.WriteSet), result)
	}
}

func TestValidateAutoprogrammingRequestV0RejectsMissingTestsAndWriteSet(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequiredTests = nil
		request.WriteSet = []string{" ", ""}
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "required_tests_missing")
	assertAutoprogrammingRequestIssueV0(t, result, "write_set_missing")
}

func TestValidateAutoprogrammingRequestV0RejectsUnsafeWriteSetPath(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.WriteSet = []string{
			"modulos/orquesta-orchestration-core/../fuera.go",
			".",
		}
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "write_set_path_invalid")
}

func TestValidateAutoprogrammingRequestV0RejectsDelegationBudgetFueraDeRango(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.MaxDelegationDepth = AutoprogrammingRequestMaxDelegationDepthV0 + 1
		request.MaxSubagentsPerAgent = AutoprogrammingRequestMaxSubagentsPerAgentV0 + 1
		request.MaxRecursiveAgents = AutoprogrammingRequestMaxRecursiveAgentsV0 + 1
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "delegation_depth_budget_invalid")
	assertAutoprogrammingRequestIssueV0(t, result, "subagents_per_agent_budget_invalid")
	assertAutoprogrammingRequestIssueV0(t, result, "recursive_agents_budget_invalid")
}

func validAutoprogrammingRequestV0(
	mutate func(*AutoprogrammingRequestV0),
) AutoprogrammingRequestV0 {
	request := AutoprogrammingRequestV0{
		RequestRef:       "autoprogramming-request-ref-orquesta-001",
		ProjectRef:       "project-ref-orquesta",
		WorktreeRef:      "worktree-ref-isolated-orquesta-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-autoprogramming-orquesta-001",
		Tasks: []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-autoprogramming-a", Area: "Orchestration Core"},
			{TaskRef: "task-ref-autoprogramming-b", Area: "orchestration_core"},
		},
		WriteSet: []string{
			"modulos/orquesta-orchestration-core/autoprogramming_request_v0.go",
			"modulos/orquesta-orchestration-core/autoprogramming_request_v0_test.go",
		},
		RequiredTests: []string{
			"go test -count=1 ./modulos/orquesta-orchestration-core",
		},
	}
	if mutate != nil {
		mutate(&request)
	}
	return request
}

func assertAutoprogrammingRequestIssueV0(
	t *testing.T,
	result AutoprogrammingRequestValidationResultV0,
	code string,
) {
	t.Helper()
	for _, issue := range result.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrado: %+v", code, result.Issues)
}
