package orquestarunfile

import (
	"context"
	"reflect"
	"testing"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestRunFileStoreQueuePersistsAfterRecreateV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	now := time.Date(2026, 5, 12, 9, 0, 0, 0, time.UTC)

	updated, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        " run-created ",
		QueueRef:      " global ",
		AppRef:        " app-created ",
		PriorityScore: 50,
		UpdatedAt:     now,
		EvidenceRefs:  []string{" ev-1 ", "ev-1"},
	})
	if err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	if updated.Status != "ready" ||
		updated.RunRef != "run-created" ||
		updated.AppRef != "app-created" ||
		!reflect.DeepEqual(updated.EvidenceRefs, []string{"ev-1"}) {
		t.Fatalf("updated=%+v", updated)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	listed, err := reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "global",
		AppRefs:  []string{"app-created"},
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 1 || listed[0].PriorityScore != 50 || !listed[0].UpdatedAt.Equal(now) {
		t.Fatalf("listed=%+v", listed)
	}

	listed[0].EvidenceRefs[0] = "mutated"
	again, err := reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 again: %v", err)
	}
	if again[0].EvidenceRefs[0] != "ev-1" {
		t.Fatalf("candidate leaked mutable evidence refs: %+v", again[0])
	}

	if _, err := reopened.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-created",
		PriorityScore: 70,
		UpdatedAt:     now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 second: %v", err)
	}
	reopened = mustNewRunFileStoreV0(t, dir)
	listed, err = reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "global",
		AppRefs:  []string{"app-created"},
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 after update: %v", err)
	}
	if len(listed) != 1 ||
		listed[0].PriorityScore != 70 ||
		listed[0].AppRef != "app-created" ||
		!listed[0].UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("listed after update=%+v", listed)
	}
}

func TestRunFileStoreQueueUpsertPersistsAfterRecreateV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)

	if _, err := store.UpsertRunSchedulingCandidateV0(context.Background(), "main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        " run-seeded ",
		AppRef:        "app-seeded",
		Status:        "ready",
		PriorityScore: 3,
		UpdatedAt:     now,
		EvidenceRefs:  []string{"seed"},
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	listed, err := reopened.ListRunSchedulingCandidatesV0(context.Background(), orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "main",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 1 || listed[0].RunRef != "run-seeded" {
		t.Fatalf("listed=%+v", listed)
	}
}
