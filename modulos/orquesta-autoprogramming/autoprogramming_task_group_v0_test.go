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

	want := []AutoprogrammingTaskGroupV0{
		{Area: "task-groups", TaskRefs: []string{"task-ref-a", "task-ref-b"}},
		{Area: "review-gate", TaskRefs: []string{"task-ref-c"}},
	}
	if !reflect.DeepEqual(groups, want) {
		t.Fatalf("groups=%+v want %+v", groups, want)
	}
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
