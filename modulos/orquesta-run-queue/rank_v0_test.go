package orquestarunqueue

import (
	"reflect"
	"testing"
	"time"
)

func TestRankRunCandidatesFiltersNonExecutableStatusesV0(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	candidates := []RunSchedulingCandidateV0{
		candidateV0("paused-run", "app-1", " paused ", 50, now),
		candidateV0("delivered-run", "app-1", " DELIVERED ", 50, now),
		candidateV0("canceled-run", "app-1", "CANCELED", 50, now),
		candidateV0("stopped-run", "app-1", RunStatusStoppedV0, 50, now),
		candidateV0("closed-run", "app-1", RunStatusClosedV0, 50, now),
		candidateV0("completed-run", "app-1", " completed ", 50, now),
		candidateV0("complete-run", "app-1", "COMPLETE", 50, now),
		candidateV0("done-run", "app-1", "done", 50, now),
		candidateV0("ready-run", "app-2", RunStatusReadyV0, 10, now),
		candidateV0("running-run", "app-2", RunStatusRunningV0, 9, now),
	}

	ranked := RankRunCandidatesV0(candidates, DefaultRunQueueRankingPolicyV0(now))

	got := runRefsV0(ranked)
	want := []string{"ready-run", "running-run"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected executable runs only, got %#v", got)
	}
}

func TestRankRunCandidatesOrdersPriorityBeforeAgingV0(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	policy := RunQueueRankingPolicyV0{
		Now:               now,
		AgingAfterSeconds: 1800,
		AgingStepSeconds:  1800,
		AgingBoostPerStep: 1,
		MaxAgingBoost:     9,
	}
	candidates := []RunSchedulingCandidateV0{
		candidateV0("same-priority-newer", "app-1", RunStatusReadyV0, 10, now.Add(-10*time.Minute)),
		candidateV0("high-priority-newer", "app-2", RunStatusReadyV0, 20, now.Add(-10*time.Minute)),
		candidateV0("same-priority-aged", "app-3", RunStatusReadyV0, 10, now.Add(-2*time.Hour)),
	}

	ranked := RankRunCandidatesV0(candidates, policy)

	got := runRefsV0(ranked)
	want := []string{"high-priority-newer", "same-priority-aged", "same-priority-newer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected order: got %#v want %#v", got, want)
	}
	if ranked[0].AgingBoost >= ranked[1].AgingBoost {
		t.Fatalf("test setup did not prove priority before aging: %#v", ranked)
	}
}

func TestRankRunCandidatesOrdersUpdatedAtStableAfterPriorityAndAgingV0(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	policy := RunQueueRankingPolicyV0{Now: now, MissingUpdatedAtLast: true}
	shared := now.Add(-5 * time.Minute)
	candidates := []RunSchedulingCandidateV0{
		candidateV0("newer-stable-a", "app-1", RunStatusReadyV0, 7, shared),
		candidateV0("older-first", "app-2", RunStatusReadyV0, 7, now.Add(-10*time.Minute)),
		candidateV0("newer-stable-b", "app-3", RunStatusReadyV0, 7, shared),
	}

	ranked := RankRunCandidatesV0(candidates, policy)

	got := runRefsV0(ranked)
	want := []string{"older-first", "newer-stable-a", "newer-stable-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected updated_at stable order: got %#v want %#v", got, want)
	}
}

func TestRankRunCandidatesDoesNotMutateInputsV0(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	candidates := []RunSchedulingCandidateV0{
		candidateV0("run-1", "app-1", RunStatusReadyV0, 1, now),
	}
	candidates[0].EvidenceRefs = []string{"evidence-1"}

	ranked := RankRunCandidatesV0(candidates, DefaultRunQueueRankingPolicyV0(now))
	ranked[0].EvidenceRefs[0] = "changed"

	if candidates[0].EvidenceRefs[0] != "evidence-1" {
		t.Fatalf("ranking leaked mutable evidence refs into input")
	}
}

func TestRankRunCandidatesFairnessPausaGrupoSoloDentroDePrioridadV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 9, 0, 0, 0, time.UTC)
	policy := DefaultRunQueueRankingPolicyV0(now)
	policy.FairnessWindowSeconds = 600
	policy.MaxRunsPerFairnessGroup = 1
	policy.FairnessGroupLastSelectedAt = map[string]time.Time{"group-hot": now.Add(-time.Minute)}
	policy.FairnessGroupRunCounts = map[string]int{"group-hot": 1}
	candidates := []RunSchedulingCandidateV0{
		candidateWithGroupV0("paused-high", "app-1", 20, now, "group-hot"),
		candidateWithGroupV0("fresh-same-priority", "app-2", 20, now, "group-fresh"),
		candidateWithGroupV0("lower-priority", "app-3", 10, now, "group-fresh"),
	}

	ranked := RankRunCandidatesV0(candidates, policy)

	got := runRefsV0(ranked)
	want := []string{"fresh-same-priority", "paused-high", "lower-priority"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fairness order: got %#v want %#v", got, want)
	}
	if !ranked[1].FairnessPaused ||
		!reflect.DeepEqual(ranked[1].FairnessReasonCodes, []string{RunQueueFairnessGroupPausedV0}) {
		t.Fatalf("fairness pause not exposed: %+v", ranked[1])
	}
}

func TestRankRunCandidatesFairnessDerivaGrupoYBoostDeterministaV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	policy := DefaultRunQueueRankingPolicyV0(now)
	policy.FairnessBoostAfterSeconds = 3600
	policy.FairnessBoostPerWindow = 2
	policy.FairnessGroupLastSelectedAt = map[string]time.Time{
		"app:app-starved": now.Add(-2 * time.Hour),
	}
	candidates := []RunSchedulingCandidateV0{
		candidateV0("run-normal", "app-normal", RunStatusReadyV0, 5, now.Add(-10*time.Minute)),
		candidateV0("run-starved", "app-starved", RunStatusReadyV0, 5, now.Add(-10*time.Minute)),
	}

	ranked := RankRunCandidatesV0(candidates, policy)

	if got := runRefsV0(ranked); !reflect.DeepEqual(got, []string{"run-starved", "run-normal"}) {
		t.Fatalf("unexpected boosted order: %#v", got)
	}
	if ranked[0].FairnessGroupRef != "app:app-starved" ||
		ranked[0].FairnessBoost != 2 ||
		!reflect.DeepEqual(ranked[0].FairnessReasonCodes, []string{
			RunQueueFairnessGroupMissingV0,
			RunQueueFairnessGroupBoostedV0,
		}) {
		t.Fatalf("boosted fairness not exposed: %+v", ranked[0])
	}
}

func candidateV0(runRef string, appRef string, status string, priority int, updatedAt time.Time) RunSchedulingCandidateV0 {
	return RunSchedulingCandidateV0{
		RunRef:        runRef,
		AppRef:        appRef,
		Status:        status,
		PriorityScore: priority,
		UpdatedAt:     updatedAt,
	}
}

func candidateWithGroupV0(runRef string, appRef string, priority int, updatedAt time.Time, groupRef string) RunSchedulingCandidateV0 {
	candidate := candidateV0(runRef, appRef, RunStatusReadyV0, priority, updatedAt)
	candidate.FairnessGroupRef = groupRef
	return candidate
}

func runRefsV0(ranked []RankedRunCandidateV0) []string {
	refs := make([]string, 0, len(ranked))
	for _, candidate := range ranked {
		refs = append(refs, candidate.RunRef)
	}
	return refs
}
