package orquestaautoprogramming

import (
	"errors"
	"reflect"
	"testing"
)

func TestGroupAutoprogrammingTasksByAreaV0GroupsByNormalizedArea(t *testing.T) {
	groups, err := GroupAutoprogrammingTasksByAreaV0([]AutoprogrammingTaskGroupCandidateV0{
		{TaskRef: " task-ref-a ", Area: " Task Groups "},
		{TaskRef: "task-ref-b", Area: "task_groups"},
		{TaskRef: "task-ref-a", Area: "task-groups"},
		{TaskRef: "task-ref-c", Area: "Review Gate"},
	})
	if err != nil {
		t.Fatalf("GroupAutoprogrammingTasksByAreaV0: %v", err)
	}

	gotRefs := []AutoprogrammingTaskGroupV0{
		{Area: groups[0].Area, TaskRefs: groups[0].TaskRefs},
		{Area: groups[1].Area, TaskRefs: groups[1].TaskRefs},
	}
	wantRefs := []AutoprogrammingTaskGroupV0{
		{Area: "task-groups", TaskRefs: []string{"task-ref-a", "task-ref-b"}},
		{Area: "review-gate", TaskRefs: []string{"task-ref-c"}},
	}
	if !reflect.DeepEqual(gotRefs, wantRefs) ||
		len(groups[0].Tasks) != 2 ||
		groups[0].Tasks[0].TaskRef != "task-ref-a" ||
		groups[0].Tasks[0].Area != "task-groups" {
		t.Fatalf("groups=%+v want %+v", groups, wantRefs)
	}
}

func TestGroupAutoprogrammingTasksByAreaV0PreservaContratoExplicito(t *testing.T) {
	groups, err := GroupAutoprogrammingTasksByAreaV0([]AutoprogrammingTaskGroupCandidateV0{{
		TaskRef:            " task-ref-a ",
		Area:               " App Stack ",
		Title:              " Cambio acotado ",
		Objective:          " Implementar contrato explicito. ",
		Context:            []string{" contexto durable ", ""},
		ContextRefs:        []string{"doc-ref:autoprog-t03"},
		AcceptanceCriteria: []string{" criterio verificable "},
		AcceptanceChecks: []AutoprogrammingAcceptanceCheckV0{{
			CriterionRef: " criterion-ref-001 ", Description: " comprobacion tipada ", Command: " go test ./modulos/orquesta-autoprogramming ",
		}},
		RequiredTests: []string{" go test ./modulos/orquesta-autoprogramming "},
		CompactRules:  []string{" conservar refs opacas "},
	}})
	if err != nil {
		t.Fatalf("GroupAutoprogrammingTasksByAreaV0: %v", err)
	}
	task := groups[0].Tasks[0]
	if task.Title != "Cambio acotado" ||
		task.Objective != "Implementar contrato explicito." ||
		task.Context[0] != "contexto durable" ||
		task.ContextRefs[0] != "doc-ref:autoprog-t03" ||
		task.AcceptanceCriteria[0] != "criterio verificable" ||
		task.AcceptanceChecks[0] != (AutoprogrammingAcceptanceCheckV0{CriterionRef: "criterion-ref-001", Description: "comprobacion tipada", Command: "go test ./modulos/orquesta-autoprogramming"}) ||
		task.RequiredTests[0] != "go test ./modulos/orquesta-autoprogramming" ||
		task.CompactRules[0] != "conservar refs opacas" {
		t.Fatalf("task=%+v", task)
	}
}

func TestGroupAutoprogrammingTasksByAreaV0RejectsDuplicateAcceptanceCheckRefInGroup(t *testing.T) {
	_, err := GroupAutoprogrammingTasksByAreaV0([]AutoprogrammingTaskGroupCandidateV0{
		{TaskRef: "task-ref-a", Area: "task-groups", AcceptanceChecks: []AutoprogrammingAcceptanceCheckV0{{CriterionRef: "criterion-ref-001", Command: "go test ./a"}}},
		{TaskRef: "task-ref-b", Area: "task-groups", AcceptanceChecks: []AutoprogrammingAcceptanceCheckV0{{CriterionRef: "criterion-ref-001", Command: "go test ./b"}}},
	})

	assertAutoprogrammingErrorV0(t, err, ErrAutoprogrammingInvalidoV0, "acceptance_checks.criterion_ref")
}

func TestGroupAutoprogrammingTasksByAreaV0RejectsMissingTaskRef(t *testing.T) {
	_, err := GroupAutoprogrammingTasksByAreaV0([]AutoprogrammingTaskGroupCandidateV0{
		{Area: "task-groups"},
	})

	assertAutoprogrammingErrorV0(t, err, ErrAutoprogrammingInvalidoV0, "task_ref")
}

func TestGroupAutoprogrammingTasksByAreaV0RejectsMissingArea(t *testing.T) {
	_, err := GroupAutoprogrammingTasksByAreaV0([]AutoprogrammingTaskGroupCandidateV0{
		{TaskRef: "task-ref-a", Area: " - _ "},
	})

	assertAutoprogrammingErrorV0(t, err, ErrAutoprogrammingInvalidoV0, "area")
}

func assertAutoprogrammingErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr ErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("err=%T, want ErrorV0", err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("err=%+v, want code=%q field=%q", publicErr, code, field)
	}
}
