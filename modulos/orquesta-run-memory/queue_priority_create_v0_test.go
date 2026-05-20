package orquestarunmemory

import (
	"context"
	"testing"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestRunMemoryStorePriorityWriterCreatesCandidateInQueueV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()
	now := time.Date(2026, 5, 11, 13, 0, 0, 0, time.UTC)

	updated, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-created",
		QueueRef:      "global",
		AppRef:        "app-created",
		PriorityScore: 50,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("set priority: %v", err)
	}
	if updated.Status != "ready" || !updated.UpdatedAt.Equal(now) {
		t.Fatalf("updated=%+v", updated)
	}

	listed, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "global",
		AppRefs:  []string{"app-created"},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].RunRef != "run-created" {
		t.Fatalf("listed=%+v", listed)
	}
}

func TestRunMemoryStoreListSchedulingCandidatesFiltraTerminalesAntesDeLimitV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC)

	seeds := []orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-a-stopped", AppRef: "app", Status: "stopped", PriorityScore: 70, UpdatedAt: now},
		{RunRef: "run-b-stopped", AppRef: "app", Status: "stopped", PriorityScore: 70, UpdatedAt: now},
		{RunRef: "run-c-ready", AppRef: "app", Status: "ready", PriorityScore: 50, UpdatedAt: now},
	}
	for _, seed := range seeds {
		if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "global", seed); err != nil {
			t.Fatalf("seed %s: %v", seed.RunRef, err)
		}
	}

	listed, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "global",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 1 || listed[0].RunRef != "run-c-ready" {
		t.Fatalf("listed=%+v", listed)
	}
}
