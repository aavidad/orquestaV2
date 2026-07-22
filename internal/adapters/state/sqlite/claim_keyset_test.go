package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestClaimKeysetFindsEligibleAfterIncompatibleWindows(t *testing.T) {
	repository, _ := openTestRepository(t)
	total := claimCandidateWindowSize*2 + 1
	seedKeysetClaimCandidates(t, repository, total, total-1)

	stats := &claimWindowQueryStats{}
	ctx := claimWindowStatsContext(stats)
	claim, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:keyset", Token: "claim:keyset", LeaseDuration: time.Minute,
		Capabilities: keysetCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found {
		t.Fatalf("claim after incompatible windows = found:%v err:%v", found, err)
	}
	if claim.Action.Ref != fmt.Sprintf("action:launch:keyset:%03d", total-1) {
		t.Fatalf("claimed %q, want final eligible action", claim.Action.Ref)
	}
	stats.assertBounded(t, 3, total)
	var fences int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM work_item_fences`).Scan(&fences); err != nil {
		t.Fatal(err)
	}
	if fences != 1 || claim.Fence != 1 {
		t.Fatalf("fence mutation = rows:%d claim:%d", fences, claim.Fence)
	}
}

func TestClaimKeysetNoEligibleCandidateUsesBoundedBatchQueries(t *testing.T) {
	repository, _ := openTestRepository(t)
	total := claimCandidateWindowSize*2 + 3
	seedKeysetClaimCandidates(t, repository, total, -1)

	stats := &claimWindowQueryStats{}
	ctx := claimWindowStatsContext(stats)
	_, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:keyset-negative", Token: "claim:keyset-negative", LeaseDuration: time.Minute,
		Capabilities: keysetCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || found {
		t.Fatalf("incompatible backlog claim = found:%v err:%v", found, err)
	}
	stats.assertBounded(t, 3, total)
	var claimed, fences int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE claim_token IS NOT NULL`).Scan(&claimed); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM work_item_fences`).Scan(&fences); err != nil {
		t.Fatal(err)
	}
	if claimed != 0 || fences != 0 {
		t.Fatalf("negative scan mutated claims=%d fences=%d", claimed, fences)
	}
}

type claimWindowQueryStats struct {
	candidateQueries, candidateResults, requirementQueries int
	resultItems, requirementItems, largestResult           int
}

func claimWindowStatsContext(stats *claimWindowQueryStats) context.Context {
	return context.WithValue(context.Background(), claimQueryObserverContextKey{}, func(event claimQueryObservation) {
		switch event.kind {
		case "candidate_window":
			stats.candidateQueries++
			if event.itemCount != claimCandidateWindowSize {
				panic("candidate query lost its fixed limit")
			}
		case "candidate_result":
			stats.candidateResults++
			stats.resultItems += event.itemCount
			if event.itemCount > stats.largestResult {
				stats.largestResult = event.itemCount
			}
		case "requirements_batch":
			stats.requirementQueries++
			stats.requirementItems += event.itemCount
		}
	})
}

func (stats claimWindowQueryStats) assertBounded(t *testing.T, windows, items int) {
	t.Helper()
	if stats.candidateQueries != windows || stats.candidateResults != windows ||
		stats.requirementQueries != windows || stats.resultItems != items ||
		stats.requirementItems != items || stats.largestResult > claimCandidateWindowSize {
		t.Fatalf("unbounded or N+1 claim scan: %+v windows=%d items=%d", stats, windows, items)
	}
}

func seedKeysetClaimCandidates(t *testing.T, repository *Repository, total, eligible int) {
	t.Helper()
	for index := 0; index < total; index++ {
		suffix := fmt.Sprintf("keyset-%03d", index)
		state := newCreateFixture(t, suffix, "request:"+suffix, "fingerprint:"+suffix, "actor:keyset", "project:keyset")
		state.Actions[0].Ref = fmt.Sprintf("action:launch:keyset:%03d", index)
		if _, _, err := createLegacyGoal(t, repository, state); err != nil {
			t.Fatalf("create candidate %d: %v", index, err)
		}
		skill := fmt.Sprintf("skill:blocked-%03d", index)
		if index == eligible {
			skill = "skill:eligible"
		}
		if _, err := repository.db.Exec(`
INSERT INTO work_item_requirement_refs(goal_ref, work_item_ref, kind, value, position)
VALUES (?, ?, 'skill', ?, 0)`, state.Goal.Ref().String(), state.Executions[0].WorkItemRef.String(), skill); err != nil {
			t.Fatalf("insert requirement %d: %v", index, err)
		}
	}
}

func keysetCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:keyset", ModelRef: "model:keyset", AgentRef: "agent:keyset",
		RoleKeys: []string{goal.DefaultRoleKey().String()}, SkillRefs: []string{"skill:eligible"},
	}
}
