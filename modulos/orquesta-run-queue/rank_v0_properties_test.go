package orquestarunqueue

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestRankRunCandidatesV0PropDeterministaBajoPermutacionV0(t *testing.T) {
	// Unique updated_at values make the contract's final stable tie-break total.
	rapid.Check(t, func(rt *rapid.T) {
		now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
		candidates := rankCandidatesRapidV0(now).Draw(rt, "candidates")
		policy := DefaultRunQueueRankingPolicyV0(now)

		baseline := RankRunCandidatesV0(candidates, policy)
		permuted := permuteRunCandidatesRapidV0(rt, candidates)
		got := RankRunCandidatesV0(permuted, policy)

		if !reflect.DeepEqual(baseline, got) {
			rt.Fatalf("ranking changed under permutation: baseline=%+v permuted=%+v", baseline, got)
		}
	})
}

func TestRankRunCandidatesV0PropNoPierdeNiDuplicaEjecutablesV0(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
		candidates := rankCandidatesRapidV0(now).Draw(rt, "candidates")
		ranked := RankRunCandidatesV0(candidates, DefaultRunQueueRankingPolicyV0(now))

		want := executableRunRefsSetV0(candidates)
		got := make(map[string]struct{}, len(ranked))
		for index, candidate := range ranked {
			if candidate.Rank != index+1 {
				rt.Fatalf("rank is not contiguous: ranked=%+v", ranked)
			}
			if _, duplicate := got[candidate.RunRef]; duplicate {
				rt.Fatalf("duplicate executable run %q in ranked=%+v", candidate.RunRef, ranked)
			}
			got[candidate.RunRef] = struct{}{}
		}
		if !reflect.DeepEqual(got, want) {
			rt.Fatalf("executable candidates were lost or added: want=%v got=%v candidates=%+v", want, got, candidates)
		}
	})
}

func TestRankRunCandidatesV0PropPrioridadEsPrimariaV0(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
		candidates := rankExecutableCandidatesRapidV0(now).Draw(rt, "candidates")
		policy := DefaultRunQueueRankingPolicyV0(now)
		policy.FairnessWindowSeconds = 600
		policy.MaxRunsPerFairnessGroup = 1
		policy.FairnessBoostAfterSeconds = 900
		policy.FairnessBoostPerWindow = rapid.IntRange(1, 5).Draw(rt, "fairness_boost")
		policy.FairnessGroupLastSelectedAt = map[string]time.Time{
			"group-hot":     now.Add(-time.Minute),
			"group-starved": now.Add(-2 * time.Hour),
		}
		policy.FairnessGroupRunCounts = map[string]int{"group-hot": 1}

		ranked := RankRunCandidatesV0(candidates, policy)
		for index := 1; index < len(ranked); index++ {
			if ranked[index-1].PriorityScore < ranked[index].PriorityScore {
				rt.Fatalf("lower priority ranked before higher priority: ranked=%+v", ranked)
			}
		}
	})
}

func TestRankRunCandidatesV0PropFairnessOrdenaAntesQueAgingV0(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
		priority := rapid.IntRange(-5, 50).Draw(rt, "priority")
		boost := rapid.IntRange(1, 5).Draw(rt, "boost")
		policy := DefaultRunQueueRankingPolicyV0(now)
		policy.FairnessWindowSeconds = 600
		policy.MaxRunsPerFairnessGroup = 1
		policy.FairnessBoostAfterSeconds = 900
		policy.FairnessBoostPerWindow = boost
		policy.FairnessGroupLastSelectedAt = map[string]time.Time{
			"hot":     now.Add(-time.Minute),
			"starved": now.Add(-2 * time.Hour),
			"fresh":   now.Add(-time.Minute),
		}
		policy.FairnessGroupRunCounts = map[string]int{"hot": 1}
		candidates := []RunSchedulingCandidateV0{
			candidateWithGroupV0("hot", "app-hot", priority, now.Add(-24*time.Hour), "hot"),
			candidateWithGroupV0("starved", "app-starved", priority, now.Add(-time.Minute), "starved"),
			candidateWithGroupV0("fresh", "app-fresh", priority, now.Add(-12*time.Hour), "fresh"),
		}

		ranked := RankRunCandidatesV0(candidates, policy)
		if got, want := runRefsV0(ranked), []string{"starved", "fresh", "hot"}; !reflect.DeepEqual(got, want) {
			rt.Fatalf("fairness did not order before aging: got=%v ranked=%+v", got, ranked)
		}
		if !ranked[2].FairnessPaused || ranked[0].FairnessBoost != boost || ranked[1].FairnessBoost != 0 {
			rt.Fatalf("unexpected fairness evaluation: ranked=%+v", ranked)
		}
	})
}

func TestRankRunCandidatesV0PropAgingEsDesempateAcotadoV0(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
		after := rapid.IntRange(1, 300).Draw(rt, "after_seconds")
		step := rapid.IntRange(1, 300).Draw(rt, "step_seconds")
		perStep := rapid.IntRange(1, 10).Draw(rt, "boost_per_step")
		maxBoost := rapid.IntRange(1, 30).Draw(rt, "max_boost")
		ageSeconds := after + step*rapid.IntRange(1, 20).Draw(rt, "age_steps")
		policy := RunQueueRankingPolicyV0{
			Now:               now,
			AgingAfterSeconds: after,
			AgingStepSeconds:  step,
			AgingBoostPerStep: perStep,
			MaxAgingBoost:     maxBoost,
		}
		candidates := []RunSchedulingCandidateV0{
			candidateV0("old", "app-old", RunStatusReadyV0, 7, now.Add(-time.Duration(ageSeconds)*time.Second)),
			candidateV0("new", "app-new", RunStatusReadyV0, 7, now),
		}

		ranked := RankRunCandidatesV0(candidates, policy)
		expected := (ageSeconds / step) * perStep
		if expected > maxBoost {
			expected = maxBoost
		}
		if got, want := runRefsV0(ranked), []string{"old", "new"}; !reflect.DeepEqual(got, want) || ranked[0].AgingBoost != expected || ranked[1].AgingBoost != 0 {
			rt.Fatalf("aging contract violated: expected_boost=%d ranked=%+v", expected, ranked)
		}
	})
}

func rankCandidatesRapidV0(now time.Time) *rapid.Generator[[]RunSchedulingCandidateV0] {
	return rapid.Custom(func(rt *rapid.T) []RunSchedulingCandidateV0 {
		count := rapid.IntRange(1, 16).Draw(rt, "count")
		statuses := []string{RunStatusReadyV0, RunStatusRunningV0, RunStatusPausedV0, RunStatusDeliveredV0, RunStatusCanceledV0, RunStatusStoppedV0, RunStatusClosedV0, "completed", "done"}
		candidates := make([]RunSchedulingCandidateV0, count)
		for index := range candidates {
			candidates[index] = candidateV0(
				fmt.Sprintf("run-%02d", index),
				fmt.Sprintf("app-%d", index%4),
				rapid.SampledFrom(statuses).Draw(rt, fmt.Sprintf("status_%d", index)),
				rapid.IntRange(-10, 50).Draw(rt, fmt.Sprintf("priority_%d", index)),
				now.Add(-time.Duration(index+1)*time.Minute),
			)
		}
		return candidates
	})
}

func rankExecutableCandidatesRapidV0(now time.Time) *rapid.Generator[[]RunSchedulingCandidateV0] {
	return rapid.Custom(func(rt *rapid.T) []RunSchedulingCandidateV0 {
		candidates := rankCandidatesRapidV0(now).Draw(rt, "base")
		for index := range candidates {
			candidates[index].Status = RunStatusReadyV0
			candidates[index].FairnessGroupRef = []string{"group-hot", "group-starved", "group-fresh"}[index%3]
		}
		return candidates
	})
}

func permuteRunCandidatesRapidV0(rt *rapid.T, candidates []RunSchedulingCandidateV0) []RunSchedulingCandidateV0 {
	permuted := append([]RunSchedulingCandidateV0(nil), candidates...)
	for last := len(permuted) - 1; last > 0; last-- {
		swap := rapid.IntRange(0, last).Draw(rt, fmt.Sprintf("swap_%d", last))
		permuted[last], permuted[swap] = permuted[swap], permuted[last]
	}
	return permuted
}

func executableRunRefsSetV0(candidates []RunSchedulingCandidateV0) map[string]struct{} {
	refs := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if IsExecutableRunStatusV0(candidate.Status) {
			refs[candidate.RunRef] = struct{}{}
		}
	}
	return refs
}
