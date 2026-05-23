package orquestaautoprogramming

import (
	"encoding/json"
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

func TestBuildAutoprogrammingProgrammableWorkV0CompactaTextoLargoParaWorkflowTask(t *testing.T) {
	longObjective := strings.Repeat("iterar pensar tareas agentes pruebas revision cierre ", 20)
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:   "task-ref-texto-largo-001",
			Area:      "Autoprogramming",
			Title:     strings.Repeat("Ciclo residente ", 20),
			Objective: longObjective,
			Context: []string{
				strings.Repeat("contexto operativo con runtime provider db sql oauth docker tmux token budget ", 10),
				strings.Repeat("hexagonal puro y refs opacas sin producto dentro del nucleo ", 10),
			},
			ContextRefs: []string{"doc-ref:autoprogramacion-pendientes"},
			AcceptanceCriteria: []string{
				strings.Repeat("criterio de cierre con pruebas reales evidencia durable y cola residente ", 10),
				strings.Repeat("criterio de reparacion con followups y self repair sin tirar todo ", 10),
			},
			CompactRules: []string{
				strings.Repeat("regla compacta caveman y reutilizar codigo existente ", 10),
			},
		}}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/autoprogramming_task_contract_v0.go"}
		request.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-autoprogramming"}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	task := result.Work.Tasks[0]
	if len([]rune(task.Title)) > 160 || len([]rune(task.Summary)) > 600 {
		t.Fatalf("task text no compactado title=%d summary=%d task=%+v", len([]rune(task.Title)), len([]rune(task.Summary)), task)
	}
	for _, criterion := range task.AcceptanceCriteria {
		if len([]rune(criterion)) > 600 {
			t.Fatalf("criterion demasiado largo len=%d value=%q", len([]rune(criterion)), criterion)
		}
	}
	payload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	if len(payload) > 4096 {
		t.Fatalf("payload demasiado grande: %d %s", len(payload), payload)
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

func TestBuildAutoprogrammingProgrammableWorkV0SequencesRepairableWriteSetOverlap(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-api", Area: "API"},
			{TaskRef: "task-ref-api-client", Area: "API Client"},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/api_client_handler.go"}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Partition.Repairs) != 1 ||
		result.Work.Partition.Repairs[0].Code != "write_set_overlap_sequenced" {
		t.Fatalf("repairs=%+v", result.Work.Partition.Repairs)
	}
	if len(result.Work.Groups) != 2 || len(result.Work.Groups[1].Task.DependsOn) != 1 {
		t.Fatalf("groups=%+v", result.Work.Groups)
	}
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
