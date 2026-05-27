package orquestarunqueue

import (
	"reflect"
	"testing"
	"time"
)

func TestRankRunCandidatesSerializaWorksetSolapadoV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	candidates := []RunSchedulingCandidateV0{
		candidateWithWorksetV0("run-a", "claim-a", "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md", 50, now),
		candidateWithWorksetV0("run-b", "claim-b", "docs", 40, now),
		candidateWithWorksetV0("run-c", "claim-c", "modulos/orquesta-run-queue", 30, now),
	}

	ranked := RankRunCandidatesV0(candidates, DefaultRunQueueRankingPolicyV0(now))

	got := runRefsV0(ranked)
	want := []string{"run-a", "run-c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked=%v want %v", got, want)
	}
}

func TestEvaluateRunQueueWorksetConcurrencyV0ExponeBloqueoCompacto(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	candidates := []RunSchedulingCandidateV0{
		candidateWithWorksetV0("run-a", "claim-a", "modulos/orquesta-run-queue", 50, now),
		candidateWithWorksetV0("run-b", "claim-b", "modulos/orquesta-run-queue/rank_v0.go", 40, now),
	}

	evaluation := EvaluateRunQueueWorksetConcurrencyV0(candidates)

	if !reflect.DeepEqual(evaluation.AllowedRunRefs, []string{"run-a"}) {
		t.Fatalf("allowed=%v", evaluation.AllowedRunRefs)
	}
	if len(evaluation.Blocks) != 1 ||
		evaluation.Blocks[0].RunRef != "run-b" ||
		evaluation.Blocks[0].BlockedByRunRef != "run-a" ||
		evaluation.Blocks[0].Reason != RunQueueWorksetReasonSerializedV0 ||
		len(evaluation.Blocks[0].ConflictRefs) == 0 {
		t.Fatalf("blocks=%+v", evaluation.Blocks)
	}
}

func TestEvaluateRunQueueWorksetConcurrencyV0BloqueaClaimAusenteOptIn(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	candidates := []RunSchedulingCandidateV0{
		candidateV0("run-no-claim", "app", RunStatusReadyV0, 50, now),
	}

	evaluation := EvaluateRunQueueWorksetConcurrencyWithPolicyV0(candidates, RunQueueWorksetPolicyV0{
		RequireClaims: true,
	})

	if len(evaluation.Blocks) != 1 ||
		evaluation.Blocks[0].RunRef != "run-no-claim" ||
		evaluation.Blocks[0].Reason != RunQueueWorksetReasonMissingClaimV0 {
		t.Fatalf("blocks=%+v", evaluation.Blocks)
	}
}

func TestRankRunCandidatesBloqueaWorksetInvalidoV0(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	candidate := candidateWithWorksetV0("run-bad", "claim-bad", "$HOME/private", 50, now)

	ranked := RankRunCandidatesV0([]RunSchedulingCandidateV0{candidate}, DefaultRunQueueRankingPolicyV0(now))

	if len(ranked) != 0 {
		t.Fatalf("ranked=%+v", ranked)
	}
}

func candidateWithWorksetV0(
	runRef string,
	claimRef string,
	writeRef string,
	priority int,
	updatedAt time.Time,
) RunSchedulingCandidateV0 {
	candidate := candidateV0(runRef, "app-autoprogramming", RunStatusReadyV0, priority, updatedAt)
	candidate.WorksetClaims = []WorksetClaimV0{{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      claimRef,
		RunRef:        runRef,
		TaskRef:       "task-" + runRef,
		WriteSet:      []ScopeRefV0{{Ref: writeRef}},
	}}
	return candidate
}
