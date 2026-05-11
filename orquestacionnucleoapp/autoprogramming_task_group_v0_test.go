package orquestacionnucleoapp

import (
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

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "task_ref")
}

func TestGroupAutoprogrammingTasksByAreaV0RejectsMissingArea(t *testing.T) {
	_, err := GroupAutoprogrammingTasksByAreaV0([]AutoprogrammingTaskGroupCandidateV0{
		{TaskRef: "task-ref-a", Area: " - _ "},
	})

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "area")
}
