package orquestaruntimecodexgoal

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestCodexGoalObserverV0ProyectaUsageObservationNormalizadaV0(t *testing.T) {
	observer := CodexGoalObserverV0{Observer: &recordingCodexGoalObserverV0{
		receipt: CodexGoalObservationReceiptV0{
			Status:  orquestagoal.GoalStatusRunningV0,
			GoalRef: "goal-ref-usage-observation-001",
			UsageObservation: orquestagoal.GoalUsageObservationV0{
				TokensAccumulated: 42,
				RuntimeSeconds:    7,
				ObservedAt:        " 2026-07-11T12:34:56+02:00 ",
				EvidenceRefs:      []string{" evidence-ref-usage-observed-001 ", "evidence-ref-usage-observed-001"},
				SourceRef:         " codex-app-server-goal:thread-usage-observation-001 ",
			},
		},
	}}

	result, err := observer.ObserveGoalWorkV0(context.Background(), orquestagoal.GoalObservationRequestV0{
		GoalRef: "goal-ref-usage-observation-001",
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	usage := result.UsageObservation
	if usage.TokensAccumulated != 42 || usage.RuntimeSeconds != 7 ||
		usage.ObservedAt != "2026-07-11T10:34:56Z" ||
		usage.SourceRef != "codex-app-server-goal:thread-usage-observation-001" ||
		len(usage.EvidenceRefs) != 1 || usage.EvidenceRefs[0] != "evidence-ref-usage-observed-001" {
		t.Fatalf("usage=%+v", usage)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded orquestagoal.GoalWorkResultV0
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(decoded.UsageObservation, usage) {
		t.Fatalf("decoded usage=%+v want=%+v", decoded.UsageObservation, usage)
	}
}
