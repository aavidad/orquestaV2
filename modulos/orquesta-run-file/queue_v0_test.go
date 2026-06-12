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
		RunRef:           " run-created ",
		QueueRef:         " global ",
		AppRef:           " app-created ",
		FairnessGroupRef: " group-created ",
		PriorityScore:    50,
		UpdatedAt:        now,
		EvidenceRefs:     []string{" ev-1 ", "ev-1"},
	})
	if err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	if updated.Status != orquestarunqueue.RunStatusReadyV0 ||
		updated.RunRef != "run-created" ||
		updated.AppRef != "app-created" ||
		updated.FairnessGroupRef != "group-created" ||
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
	if len(listed) != 1 ||
		listed[0].PriorityScore != 50 ||
		listed[0].FairnessGroupRef != "group-created" ||
		!listed[0].UpdatedAt.Equal(now) {
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

func TestRunFileStoreQueuePersisteMetadataDeRescateV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-rescue",
		QueueRef:      "global",
		AppRef:        "app",
		PriorityScore: 50,
		UpdatedAt:     now,
		AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{
			GroupRef:     "attempt-topic-001",
			ConsumerRef:  "consumer",
			ObjectiveRef: "objective",
			WorkItemRef:  "topic-001",
			WriteSetRefs: []string{"topic/001"},
		},
		ParentRunRef:     "run-original",
		SupersedesRunRef: "run-original",
		RescueReason:     "estado_incierto",
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	listed, err := reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "global"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 1 ||
		listed[0].AttemptGroup.GroupRef != "attempt-topic-001" ||
		listed[0].AttemptGroup.WorkItemRef != "topic-001" ||
		listed[0].ParentRunRef != "run-original" ||
		listed[0].SupersedesRunRef != "run-original" ||
		listed[0].RescueReason != "estado_incierto" {
		t.Fatalf("listed=%+v", listed)
	}
}

func TestRunFileStoreQueueUpsertPersistsAfterRecreateV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)

	if _, err := store.UpsertRunSchedulingCandidateV0(context.Background(), "main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        " run-seeded ",
		AppRef:        "app-seeded",
		Status:        orquestarunqueue.RunStatusReadyV0,
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

func TestRunFileStoreListSchedulingCandidatesFiltraTerminalesAntesDeLimitV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC)

	seeds := []orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-a-stopped", AppRef: "app", Status: "stopped", PriorityScore: 70, UpdatedAt: now},
		{RunRef: "run-b-stopped", AppRef: "app", Status: "stopped", PriorityScore: 70, UpdatedAt: now},
		{RunRef: "run-c-ready", AppRef: "app", Status: orquestarunqueue.RunStatusReadyV0, PriorityScore: 50, UpdatedAt: now},
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

func TestRunFileStorePriorityWriterPuedePersistirEstadoTerminalV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	now := time.Date(2026, 5, 22, 9, 0, 0, 0, time.UTC)

	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-terminal",
		QueueRef:      "global",
		AppRef:        "app-terminal",
		PriorityScore: 50,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	updated, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-terminal",
		QueueRef:      "global",
		AppRef:        "app-terminal",
		Status:        orquestarunqueue.RunStatusClosedV0,
		PriorityScore: 50,
		UpdatedAt:     now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("SetRunPriorityV0 closed: %v", err)
	}
	if updated.Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("updated=%+v", updated)
	}
	reopened := mustNewRunFileStoreV0(t, dir)
	listed, err := reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "global",
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("listed=%+v", listed)
	}
	all, err := reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             "global",
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 include terminal: %v", err)
	}
	if len(all) != 1 || all[0].RunRef != "run-terminal" || all[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("all=%+v", all)
	}
}

func TestRunFileStoreQueuePersisteWorksetClaimsV0(t *testing.T) {
	dir := t.TempDir()
	store := mustNewRunFileStoreV0(t, dir)
	ctx := context.Background()
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)

	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-workset",
		QueueRef:      "global",
		AppRef:        "app",
		PriorityScore: 50,
		UpdatedAt:     now,
		WorksetClaims: []orquestarunqueue.WorksetClaimV0{{
			SchemaVersion: orquestarunqueue.WorksetClaimSchemaVersionV0,
			ClaimRef:      "claim-workset",
			RunRef:        "run-workset",
			TaskRef:       "task-workset",
			WriteSet:      []orquestarunqueue.ScopeRefV0{{Ref: "modulos/orquesta-run-file"}},
		}},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	reopened := mustNewRunFileStoreV0(t, dir)
	listed, err := reopened.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "global"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 1 || listed[0].WorksetClaims[0].WriteSet[0].Ref != "modulos/orquesta-run-file" {
		t.Fatalf("listed=%+v", listed)
	}
}
