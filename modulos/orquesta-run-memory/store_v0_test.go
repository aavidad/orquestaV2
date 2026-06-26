package orquestarunmemory

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestRunMemoryStoreSatisfiesPortsV0(t *testing.T) {
	var _ orquestaruncontrol.RunControlPortV0 = NewRunMemoryStoreV0()
	var _ orquestarunqueue.RunQueueReaderPortV0 = NewRunMemoryStoreV0()
	var _ orquestarunqueue.RunQueuePriorityWriterPortV0 = NewRunMemoryStoreV0()
	var _ orquestarunqueue.RunQueuePortV0 = NewRunMemoryStoreV0()
}

func TestRunMemoryStorePauseResumeV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()

	paused, err := store.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef:       " run-1 ",
		RequestedBy:  " director ",
		Reason:       " wait ",
		EvidenceRefs: []string{" ev-1 ", "ev-1", ""},
	})
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if paused.Status != orquestaruncontrol.RunControlStatusPausedV0 {
		t.Fatalf("paused status=%q", paused.Status)
	}
	if paused.RunRef != "run-1" || paused.Meta.RequestedBy != "director" {
		t.Fatalf("paused state=%+v", paused)
	}
	if !reflect.DeepEqual(paused.EvidenceRefs, []string{"ev-1"}) {
		t.Fatalf("evidence refs=%#v", paused.EvidenceRefs)
	}

	resumed, err := store.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
		RunRef: "run-1",
	})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Status != orquestaruncontrol.RunControlStatusRunningV0 {
		t.Fatalf("resumed status=%q", resumed.Status)
	}
}

func TestRunMemoryStoreStopCancelV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()

	stopped, err := store.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef: "run-2",
		Forced: true,
	})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped.Status != orquestaruncontrol.RunControlStatusStopRequestedV0 || !stopped.Forced {
		t.Fatalf("stopped state=%+v", stopped)
	}

	canceled, err := store.CancelRunV0(ctx, orquestaruncontrol.CancelRunCommandV0{
		RunRef: "run-2",
	})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if canceled.Status != orquestaruncontrol.RunControlStatusCancelRequestedV0 || canceled.Forced {
		t.Fatalf("canceled state=%+v", canceled)
	}
}

func TestRunMemoryStoreCompleteRunControlV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()

	_, err := store.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef: " run-2 ",
		Forced: true,
	})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	completed, err := store.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         " run-2 ",
		TargetStatus:   " STOPPED ",
		RequestedBy:    " director ",
		Reason:         " drained ",
		IdempotencyKey: " idem-1 ",
		EvidenceRefs:   []string{" ev-1 ", "ev-1"},
	})
	if err != nil {
		t.Fatalf("complete stopped: %v", err)
	}
	if completed.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		completed.Forced ||
		completed.RunRef != "run-2" ||
		completed.Meta.RequestedBy != "director" ||
		!reflect.DeepEqual(completed.EvidenceRefs, []string{"ev-1"}) {
		t.Fatalf("completed=%+v", completed)
	}

	completed, err = store.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-2",
		TargetStatus: orquestaruncontrol.RunControlStatusCanceledV0,
	})
	if err != nil {
		t.Fatalf("complete canceled: %v", err)
	}
	if completed.Status != orquestaruncontrol.RunControlStatusCanceledV0 {
		t.Fatalf("completed=%+v", completed)
	}
}

func TestRunMemoryStoreCompleteRunControlRejectsNonFinalStatusV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	_, err := store.CompleteRunControlV0(context.Background(), orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-2",
		TargetStatus: orquestaruncontrol.RunControlStatusRunningV0,
	})
	var typed orquestaruncontrol.RunControlCompletionTargetErrorV0
	if err == nil || !errors.As(err, &typed) {
		t.Fatalf("err=%v typed=%+v", err, typed)
	}
}

func TestRunMemoryStoreReadReturnsSnapshotV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()

	state, err := store.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef:       "run-3",
		EvidenceRefs: []string{"ev-1"},
	})
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	state.EvidenceRefs[0] = "changed"

	read, err := store.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-3",
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.EvidenceRefs[0] != "ev-1" {
		t.Fatalf("state leaked mutable evidence refs: %+v", read)
	}
}

func TestRunMemoryStoreReadMissingReturnsTypedNotFoundV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	_, err := store.ReadRunControlStateV0(context.Background(), orquestaruncontrol.RunControlReadRequestV0{
		RunRef: "run-missing",
	})
	var typed orquestaruncontrol.RunControlStateNotFoundErrorV0
	if err == nil || !errors.As(err, &typed) || typed.RunRef != "run-missing" {
		t.Fatalf("err=%v typed=%+v", err, typed)
	}
}

func TestRunMemoryStorePriorityWriterAndReaderV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	_, err := store.UpsertRunSchedulingCandidateV0(ctx, "main", candidateForTestV0(
		"run-1", "app-1", "ready", 1, now,
	))
	if err != nil {
		t.Fatalf("seed run-1: %v", err)
	}
	updated, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        " run-1 ",
		QueueRef:      "main",
		PriorityScore: 9,
		UpdatedAt:     now.Add(time.Minute),
		EvidenceRefs:  []string{" ev-2 ", "ev-2"},
	})
	if err != nil {
		t.Fatalf("set priority: %v", err)
	}
	if updated.PriorityScore != 9 ||
		!updated.UpdatedAt.Equal(now.Add(time.Minute)) ||
		updated.EvidenceRefs[0] != "ev-2" {
		t.Fatalf("updated candidate=%+v", updated)
	}

	listed, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: "main",
		AppRefs:  []string{"app-1"},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].PriorityScore != 9 {
		t.Fatalf("listed=%+v", listed)
	}
	listed[0].EvidenceRefs[0] = "changed"

	again, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{})
	if err != nil {
		t.Fatalf("list again: %v", err)
	}
	if again[0].EvidenceRefs[0] != "ev-2" {
		t.Fatalf("candidate leaked mutable evidence refs: %+v", again[0])
	}
}

func TestRunMemoryStoreListSchedulingCandidatesIncluyeNoEjecutablesOptInV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()
	now := time.Date(2026, 5, 25, 16, 0, 0, 0, time.UTC)
	if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "main", candidateForTestV0(
		"run-completed", "app-1", "completed", 9, now,
	)); err != nil {
		t.Fatalf("seed completed: %v", err)
	}

	visible, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "main"})
	if err != nil {
		t.Fatalf("list visible: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("visible=%+v", visible)
	}
	all, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{
		QueueRef:             "main",
		IncludeNonExecutable: true,
	})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 1 || all[0].RunRef != "run-completed" {
		t.Fatalf("all=%+v", all)
	}
}

func TestRunMemoryStoreRankingUsesQueuePolicyV0(t *testing.T) {
	store := NewRunMemoryStoreV0()
	ctx := context.Background()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	seeds := []orquestarunqueue.RunSchedulingCandidateV0{
		candidateForTestV0("run-low-old", "app-1", "ready", 1, now.Add(-2*time.Hour)),
		candidateForTestV0("run-high", "app-2", "ready", 5, now.Add(-5*time.Minute)),
		candidateForTestV0("run-paused", "app-3", "paused", 99, now.Add(-3*time.Hour)),
	}
	for _, seed := range seeds {
		if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "main", seed); err != nil {
			t.Fatalf("seed %s: %v", seed.RunRef, err)
		}
	}

	candidates, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	ranked := orquestarunqueue.RankRunCandidatesV0(
		candidates,
		orquestarunqueue.DefaultRunQueueRankingPolicyV0(now),
	)
	got := rankedRunRefsForTestV0(ranked)
	want := []string{"run-high", "run-low-old"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked refs got %#v want %#v", got, want)
	}
}

func candidateForTestV0(
	runRef string,
	appRef string,
	status string,
	priority int,
	updatedAt time.Time,
) orquestarunqueue.RunSchedulingCandidateV0 {
	return orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        runRef,
		AppRef:        appRef,
		Status:        status,
		PriorityScore: priority,
		UpdatedAt:     updatedAt,
	}
}

func rankedRunRefsForTestV0(values []orquestarunqueue.RankedRunCandidateV0) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.RunRef)
	}
	return out
}
