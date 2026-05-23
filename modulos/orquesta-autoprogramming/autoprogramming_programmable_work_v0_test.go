package orquestaautoprogramming

import (
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildAutoprogrammingProgrammableWorkV0BuildsWorkflowTaskForSingleGroup(t *testing.T) {
	request := validAutoprogrammingRequestV0(nil)

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Groups) != 1 || len(result.Work.Profiles) != 1 || len(result.Work.Tasks) != 1 {
		t.Fatalf("work=%+v", result.Work)
	}
	task := result.Work.Tasks[0]
	if task.WorkProfileKind != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("work_profile_kind=%q", task.WorkProfileKind)
	}
	if !reflect.DeepEqual(task.WriteSet, result.Work.Groups[0].WriteSet) ||
		!reflect.DeepEqual(task.WriteSet, []string{
			"modulos/orquesta-orchestration-core/autoprogramming_request_v0.go",
			"modulos/orquesta-orchestration-core/autoprogramming_request_v0_test.go",
		}) {
		t.Fatalf("write_set=%v", task.WriteSet)
	}
	if !reflect.DeepEqual(task.RequiredTests, request.RequiredTests) {
		t.Fatalf("required_tests=%v want %v", task.RequiredTests, request.RequiredTests)
	}
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "worktree_ref:"+request.WorktreeRef)
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "branch_ref:"+request.BranchRef)
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-autoprogramming-a")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-autoprogramming-b")
}

func TestBuildAutoprogrammingProgrammableWorkV0TransportaContratoExplicitoDeTarea(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            "task-ref-explicita-001",
			Area:               "Autoprogramming",
			Title:              "Cambio acotado 01",
			Objective:          "Transportar objetivo y contexto sin inferir por task_ref.",
			Context:            []string{"Paquete de agente ya trae criterios y reglas compactas."},
			ContextRefs:        []string{"doc-ref:autoprog-t03"},
			AcceptanceCriteria: []string{"objetivo, contexto y criterios llegan al worker"},
			RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-autoprogramming -run TestBuildAutoprogramming"},
			CompactRules:       []string{"tratar worktree_ref y branch_ref como refs opacas"},
		}}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/autoprogramming_programmable_work_v0.go",
		}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	if task.Title != "Cambio acotado 01" ||
		!stringsContainForAutoprogrammingTestV0(task.Summary, "Transportar objetivo") ||
		!stringsContainForAutoprogrammingTestV0(task.Summary, "Autoprogramacion acotada") {
		t.Fatalf("task summary/title=%+v", task)
	}
	for _, want := range []string{
		"Objetivo task-ref-explicita-001: Transportar objetivo",
		"Contexto task-ref-explicita-001: Paquete de agente",
		"Regla compacta task-ref-explicita-001: tratar worktree_ref",
	} {
		if !stringsContainForAutoprogrammingTestV0(task.Summary+"\n"+result.Work.Profiles[0].Objective, want) {
			t.Fatalf("objective no contiene %q: %+v", want, result.Work.Profiles[0])
		}
	}
	for _, want := range []string{
		"Criterio task-ref-explicita-001: objetivo, contexto y criterios llegan al worker",
		"Regla compacta task-ref-explicita-001: tratar worktree_ref",
	} {
		if !stringsSliceContainsSubstringForAutoprogrammingTestV0(task.AcceptanceCriteria, want) {
			t.Fatalf("criteria no contiene %q: %v", want, task.AcceptanceCriteria)
		}
	}
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "source_task_ref:task-ref-explicita-001")
	assertAutoprogrammingContextRefV0(t, task.ContextRefs, "doc-ref:autoprog-t03")
	if !stringsSliceContainsForAutoprogrammingTestV0(
		task.RequiredTests,
		"go test -count=1 ./modulos/orquesta-autoprogramming -run TestBuildAutoprogramming",
	) {
		t.Fatalf("required_tests=%v", task.RequiredTests)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0PartitionsWriteSetByGroupArea(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-groups", Area: "Task Group"},
			{TaskRef: "task-ref-review", Area: "Review Gate"},
		}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/autoprogramming_task_group_v0.go",
			"modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go",
		}
		request.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-autoprogramming"}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Tasks) != 2 {
		t.Fatalf("tasks=%d", len(result.Work.Tasks))
	}
	gotByArea := map[string][]string{}
	for _, group := range result.Work.Groups {
		gotByArea[group.Area] = group.Task.WriteSet
	}
	assertStringsEqualV0(t, gotByArea["task-group"], []string{
		"modulos/orquesta-autoprogramming/autoprogramming_task_group_v0.go",
	})
	assertStringsEqualV0(t, gotByArea["review-gate"], []string{
		"modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go",
	})
	assertNoWriteSetOverlapV0(t, result.Work.Tasks)
}

func TestBuildAutoprogrammingProgrammableWorkV0RejectsUnpartitionableWriteSet(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-group-a", Area: "Task Group"},
			{TaskRef: "task-ref-review-a", Area: "Review Gate"},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/README.md"}
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, AutoprogrammingRequestValidationResultV0{
		Issues: result.Issues,
	}, "write_set_unassigned")
}

func TestBuildAutoprogrammingProgrammableWorkV0RejectsWriteSetOverlap(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-api", Area: "API"},
			{TaskRef: "task-ref-api-client", Area: "API Client"},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/api_client_handler.go"}
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, AutoprogrammingRequestValidationResultV0{
		Issues: result.Issues,
	}, "write_set_overlap")
}

func TestBuildAutoprogrammingProgrammableWorkV0NoConfundeAreaComoSubcadena(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-api", Area: "API"},
			{TaskRef: "task-ref-capital", Area: "Capital"},
		}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/api/client.go",
			"modulos/orquesta-autoprogramming/capital/report.go",
		}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	gotByArea := map[string][]string{}
	for _, group := range result.Work.Groups {
		gotByArea[group.Area] = group.Task.WriteSet
	}
	assertStringsEqualV0(t, gotByArea["api"], []string{
		"modulos/orquesta-autoprogramming/api/client.go",
	})
	assertStringsEqualV0(t, gotByArea["capital"], []string{
		"modulos/orquesta-autoprogramming/capital/report.go",
	})
}

func assertAutoprogrammingContextRefV0(t *testing.T, refs []string, want string) {
	t.Helper()
	for _, ref := range refs {
		if ref == want {
			return
		}
	}
	t.Fatalf("context ref %q no encontrado en %v", want, refs)
}

func assertStringsEqualV0(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func assertNoWriteSetOverlapV0(
	t *testing.T,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) {
	t.Helper()
	seen := map[string]string{}
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			if owner := seen[path]; owner != "" {
				t.Fatalf("write_set path %q reused by %s and %s", path, owner, task.TaskID)
			}
			seen[path] = task.TaskID
		}
	}
}

func stringsContainForAutoprogrammingTestV0(value string, want string) bool {
	return strings.Contains(value, want)
}

func stringsSliceContainsForAutoprogrammingTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stringsSliceContainsSubstringForAutoprogrammingTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
