package orquestarunqueue

import (
	"fmt"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestProjectRunQueueAttemptsV0PropSupersedesNoDependeDePermutacionV0(t *testing.T) {
	// A superseded attempt cannot become active merely because it was read first.
	rapid.Check(t, func(rt *rapid.T) {
		now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
		groupRef := fmt.Sprintf("attempt-group-%d", rapid.IntRange(0, 9999).Draw(rt, "group"))
		candidates := []RunSchedulingCandidateV0{
			{
				RunRef:        "run-original",
				Status:        RunStatusReadyV0,
				PriorityScore: 100,
				UpdatedAt:     now,
				AttemptGroup:  RunQueueAttemptGroupV0{GroupRef: groupRef},
			},
			{
				RunRef:           "run-rescue",
				Status:           RunStatusReadyV0,
				PriorityScore:    1,
				UpdatedAt:        now.Add(-time.Hour),
				AttemptGroup:     RunQueueAttemptGroupV0{GroupRef: groupRef},
				ParentRunRef:     "run-original",
				SupersedesRunRef: "run-original",
			},
		}

		for _, permuted := range [][]RunSchedulingCandidateV0{candidates, {candidates[1], candidates[0]}} {
			projections := ProjectRunQueueAttemptsV0(permuted)
			if len(projections) != 1 || projections[0].ActiveAttemptRef != "run-rescue" {
				rt.Fatalf("superseded original remained active: projections=%+v", projections)
			}
		}
	})
}
