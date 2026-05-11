package orquestaruntimecodexdelivery

import (
	"context"
	"testing"
	"time"
)

func TestCodexProgressStateV0NoInflarTicksSinVentanaReal(t *testing.T) {
	store := NewInMemoryCodexProgressStateStoreV0()
	observedAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	sample := codexProgressRateLimitSampleV0(observedAt)

	first, err := store.ObserveCodexProgressV0(context.Background(), sample)
	if err != nil {
		t.Fatalf("first observe: %v", err)
	}
	if !first.SampleAccepted || first.Current.TickCounter != 1 {
		t.Fatalf("first=%+v", first)
	}

	second := sample
	second.ObservedAt = observedAt.Add(time.Second)
	secondState, err := store.ObserveCodexProgressV0(context.Background(), second)
	if err != nil {
		t.Fatalf("second observe: %v", err)
	}
	if secondState.SampleAccepted || secondState.Current.TickCounter != 1 {
		t.Fatalf("second=%+v", secondState)
	}

	third := sample
	third.ObservedAt = observedAt.Add(6 * time.Second)
	thirdState, err := store.ObserveCodexProgressV0(context.Background(), third)
	if err != nil {
		t.Fatalf("third observe: %v", err)
	}
	if !thirdState.SampleAccepted || thirdState.Current.TickCounter != 2 {
		t.Fatalf("third=%+v", thirdState)
	}
}

func codexProgressRateLimitSampleV0(observedAt time.Time) CodexProgressSampleV0 {
	return CodexProgressSampleV0{
		RunID:                "run-ref-progress-rate-001",
		AgentRequestID:       "agent-ref-progress-rate-001",
		ProcessRef:           "process-ref-progress-rate-001",
		Signature:            "stdout:0;stderr:0;last_message:0;ack:missing;",
		EvidenceRefs:         []string{"evidence-ref-progress-rate-001"},
		ObservedAt:           observedAt,
		MinUnchangedInterval: 5 * time.Second,
	}
}
