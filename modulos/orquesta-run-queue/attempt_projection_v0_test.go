package orquestarunqueue

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestProjectRunQueueAttemptsV0EnlazaRescatesYActivoV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	group := RunQueueAttemptGroupV0{
		ConsumerRef:  "consumer-a",
		ObjectiveRef: "objective-a",
		WorkItemRef:  "topic-001",
		WriteSetRefs: []string{" tema/001 ", "tema/001"},
	}
	candidates := []RunSchedulingCandidateV0{
		{
			RunRef:       "run-original",
			AppRef:       "app-a",
			Status:       RunStatusRunningV0,
			UpdatedAt:    now.Add(-20 * time.Minute),
			AttemptGroup: group,
		},
		{
			RunRef:           "run-rescue-1",
			AppRef:           "app-a",
			Status:           RunStatusReadyV0,
			UpdatedAt:        now.Add(-5 * time.Minute),
			AttemptGroup:     group,
			ParentRunRef:     "run-original",
			SupersedesRunRef: "run-original",
			RescueReason:     "estado_incierto",
			EvidenceRefs:     []string{" evidence-1 ", "evidence-1"},
		},
	}

	projections := ProjectRunQueueAttemptsV0(candidates)

	if len(projections) != 1 {
		t.Fatalf("projections=%+v", projections)
	}
	got := projections[0]
	if got.OriginalRunRef != "run-original" ||
		got.ActiveAttemptRef != "run-rescue-1" ||
		got.ParentRunRef != "run-original" ||
		got.SupersedesRunRef != "run-original" ||
		got.RescueReason != "estado_incierto" ||
		got.Status != RunStatusReadyV0 {
		t.Fatalf("projection=%+v", got)
	}
	if !reflect.DeepEqual(got.RunRefs, []string{"run-original", "run-rescue-1"}) ||
		!reflect.DeepEqual(got.RescueRunRefs, []string{"run-rescue-1"}) ||
		!reflect.DeepEqual(got.EvidenceRefs, []string{"evidence-1"}) {
		t.Fatalf("refs projection=%+v", got)
	}
	if got.AttemptGroup.WriteSetRefs[0] != "tema/001" {
		t.Fatalf("attempt group no normalizado: %+v", got.AttemptGroup)
	}
}

func TestProjectRunQueueAttemptsV0NoConfundeGruposSinMetadataV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	projections := ProjectRunQueueAttemptsV0([]RunSchedulingCandidateV0{
		{RunRef: "run-b", Status: RunStatusReadyV0, UpdatedAt: now},
		{RunRef: "run-a", Status: RunStatusReadyV0, UpdatedAt: now},
	})

	if len(projections) != 2 ||
		projections[0].GroupRef != "run:run-a" ||
		projections[1].GroupRef != "run:run-b" {
		t.Fatalf("projections=%+v", projections)
	}
}

func TestProjectRunQueueAttemptsV0FakeSetentaYDosItemsConRescatesV0(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	candidates := make([]RunSchedulingCandidateV0, 0, 143)
	for item := 1; item <= 72; item++ {
		group := RunQueueAttemptGroupV0{
			ConsumerRef:  "consumer-fake",
			ObjectiveRef: "objective-fake",
			WorkItemRef:  "item-" + threeDigitsTestV0(item),
			WriteSetRefs: []string{"work/item-" + threeDigitsTestV0(item)},
		}
		original := "run-item-" + threeDigitsTestV0(item) + "-original"
		candidates = append(candidates, RunSchedulingCandidateV0{
			RunRef:        original,
			AppRef:        "app-fake",
			Status:        RunStatusRunningV0,
			PriorityScore: 10,
			UpdatedAt:     now.Add(-time.Duration(item) * time.Minute),
			AttemptGroup:  group,
		})
		if item == 72 {
			continue
		}
		candidates = append(candidates, RunSchedulingCandidateV0{
			RunRef:           "run-item-" + threeDigitsTestV0(item) + "-rescue",
			AppRef:           "app-fake",
			Status:           RunStatusReadyV0,
			PriorityScore:    20,
			UpdatedAt:        now.Add(time.Duration(item) * time.Second),
			AttemptGroup:     group,
			ParentRunRef:     original,
			SupersedesRunRef: original,
			RescueReason:     "estado_incierto",
		})
	}

	projections := ProjectRunQueueAttemptsV0(candidates)

	rescued := 0
	for _, projection := range projections {
		if len(projection.RescueRunRefs) > 0 {
			rescued++
			if projection.ActiveAttemptRef == projection.OriginalRunRef ||
				projection.ParentRunRef == "" ||
				projection.SupersedesRunRef == "" {
				t.Fatalf("rescue projection incompleta: %+v", projection)
			}
		}
	}
	if len(projections) != 72 || rescued != 71 {
		t.Fatalf("projections=%d rescued=%d", len(projections), rescued)
	}
}

func threeDigitsTestV0(value int) string {
	return fmt.Sprintf("%03d", value)
}
