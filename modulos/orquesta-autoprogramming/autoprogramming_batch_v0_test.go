package orquestaautoprogramming

import (
	"reflect"
	"strings"
	"testing"
)

func TestAutoprogrammingBatchV0CompletesFrozenBatchGateAndPromotion(t *testing.T) {
	batch := mustNewAutoprogrammingBatchV0(t)
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 1, "launch-a", "task-a"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 2, "launch-b", "task-b"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchFocalCloseV0(batch, 3, "close-a", "task-a"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchFocalCloseV0(batch, 4, "close-b", "task-b"))
	batch = integrateAutoprogrammingBatchMemberV0(t, batch, "a", "task-a", "source-a", "base-revision-001", "revision-a")
	batch = integrateAutoprogrammingBatchMemberV0(t, batch, "b", "task-b", "source-b", "revision-a", "revision-001")
	if batch.Status != AutoprogrammingBatchStatusPendingBatchGateV0 {
		t.Fatalf("status=%q", batch.Status)
	}

	for index, test := range batch.FrozenTests {
		hash := AutoprogrammingBatchTestHashV0(test)
		claimRef := "claim-ref-" + string(rune('a'+index))
		batch = mustBatchTransitionV0(t, ClaimAutoprogrammingBatchTestV0(batch, batch.StoreVersion, "claim-"+claimRef, "revision-001", hash, claimRef))
		batch = mustBatchTransitionV0(t, RecordAutoprogrammingBatchTestReceiptV0(batch, batch.StoreVersion, "receipt-"+claimRef, "revision-001", hash, claimRef, "receipt-ref-"+claimRef, AutoprogrammingBatchTestReceiptPassedV0))
	}
	if batch.Status != AutoprogrammingBatchStatusBatchGatePassedV0 {
		t.Fatalf("status=%q", batch.Status)
	}
	if got := CloseAutoprogrammingBatchV0(batch, batch.StoreVersion, "close-too-early"); got.Accepted {
		t.Fatal("close accepted without promotion")
	}
	batch = mustBatchTransitionV0(t, ClaimAutoprogrammingBatchPromotionV0(batch, batch.StoreVersion, "claim-promote", "promotion-claim-001", "revision-001"))
	if got := CloseAutoprogrammingBatchV0(batch, batch.StoreVersion, "close-with-orphan-promotion"); got.Accepted {
		t.Fatal("close accepted with promotion claim without receipt")
	}
	promotionExpected := batch.StoreVersion
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchPromotionV0(batch, promotionExpected, "promote", "promotion-claim-001", "revision-001", "promotion-receipt-001"))
	replay := RegisterAutoprogrammingBatchPromotionV0(batch, promotionExpected, "promote", "promotion-claim-001", "revision-001", "promotion-receipt-001")
	if !replay.Accepted || !replay.Replay || !reflect.DeepEqual(replay.Batch, batch) {
		t.Fatalf("promotion replay=%+v", replay)
	}
	batch = mustBatchTransitionV0(t, CloseAutoprogrammingBatchV0(batch, batch.StoreVersion, "close"))
	if batch.Status != AutoprogrammingBatchStatusClosedV0 || batch.StoreVersion != 16 {
		t.Fatalf("batch=%+v", batch)
	}
}

func TestAutoprogrammingBatchV0CASReplayAndFrozenPlan(t *testing.T) {
	batch := mustNewAutoprogrammingBatchV0(t)
	launched := mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 1, "launch-a", "task-a"))

	replay := RegisterAutoprogrammingBatchLaunchV0(launched, 1, "launch-a", "task-a")
	if !replay.Accepted || !replay.Replay || replay.Batch.StoreVersion != launched.StoreVersion {
		t.Fatalf("replay=%+v", replay)
	}
	if got := RegisterAutoprogrammingBatchLaunchV0(launched, 1, "launch-b", "task-b"); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_cas_conflict") {
		t.Fatalf("cas=%+v", got)
	}
	if got := RegisterAutoprogrammingBatchFocalCloseV0(launched, launched.StoreVersion, "launch-a", "task-a"); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_idempotency_key_reused") {
		t.Fatalf("reused=%+v", got)
	}

	launched.Members[0].WriteSet = []string{"other.go"}
	if got := ValidateAutoprogrammingBatchV0(launched); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_plan_hash_invalid") {
		t.Fatalf("plan mutation=%+v", got)
	}
}

func TestAutoprogrammingBatchV0RejectsGateBeforeIntegrationAndDuplicateClaims(t *testing.T) {
	batch := mustNewAutoprogrammingBatchV0(t)
	batch.Status = AutoprogrammingBatchStatusPendingBatchGateV0
	if got := ValidateAutoprogrammingBatchV0(batch); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_gate_before_integration") {
		t.Fatalf("gate=%+v", got)
	}

	batch = readyAutoprogrammingBatchGateV0(t)
	hash := AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
	batch = mustBatchTransitionV0(t, ClaimAutoprogrammingBatchTestV0(batch, batch.StoreVersion, "claim-a", "revision-001", hash, "claim-ref-a"))
	got := ClaimAutoprogrammingBatchTestV0(batch, batch.StoreVersion, "claim-a-again", "revision-001", hash, "claim-ref-b")
	if got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_test_claim_duplicate") {
		t.Fatalf("duplicate=%+v", got)
	}
}

func TestAutoprogrammingBatchV0AcceptsSuccessiveIntegrationHeadsAndExactReplay(t *testing.T) {
	batch := mustNewAutoprogrammingBatchV0(t)
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 1, "launch-a", "task-a"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 2, "launch-b", "task-b"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchFocalCloseV0(batch, 3, "close-a", "task-a"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchFocalCloseV0(batch, 4, "close-b", "task-b"))
	batch = mustBatchTransitionV0(t, ClaimAutoprogrammingBatchIntegrationV0(batch, 5, "claim-integrate-a", "integration-claim-a", "task-a", "base-revision-001"))
	if batch.IntegrationClaims[0].SourceRevision != "" {
		t.Fatalf("pre-effect claim contains source revision: %+v", batch.IntegrationClaims[0])
	}
	claimReplay := ClaimAutoprogrammingBatchIntegrationV0(batch, 5, "claim-integrate-a", "integration-claim-a", "task-a", "base-revision-001")
	if !claimReplay.Accepted || !claimReplay.Replay || !reflect.DeepEqual(claimReplay.Batch, batch) {
		t.Fatalf("claim replay=%+v", claimReplay)
	}
	for _, divergent := range []struct{ claimRef, taskRef, parentRevision string }{
		{"integration-claim-other", "task-a", "base-revision-001"},
		{"integration-claim-a", "task-b", "base-revision-001"},
		{"integration-claim-a", "task-a", "parent-other"},
	} {
		got := ClaimAutoprogrammingBatchIntegrationV0(batch, 5, "claim-integrate-a", divergent.claimRef, divergent.taskRef, divergent.parentRevision)
		if got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_idempotency_key_reused") {
			t.Fatalf("claim divergence=%+v", got)
		}
	}
	if got := ClaimAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "claim-integrate-b-early", "integration-claim-b", "task-b", "base-revision-001"); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_integration_claim_orphaned") {
		t.Fatalf("orphan claim=%+v", got)
	}
	receiptExpected := batch.StoreVersion
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchIntegrationV0(batch, receiptExpected, "integrate-a", "integration-claim-a", "task-a", "source-a", "base-revision-001", "revision-a", "integration-receipt-a"))
	if batch.IntegratedRevision != "" || batch.Members[0].SourceRevision != "source-a" || batch.Members[0].IntegrationRevision != "revision-a" || batch.Members[0].IntegrationReceiptRef != "integration-receipt-a" {
		t.Fatalf("first integration=%+v", batch)
	}

	replay := RegisterAutoprogrammingBatchIntegrationV0(batch, receiptExpected, "integrate-a", "integration-claim-a", "task-a", "source-a", "base-revision-001", "revision-a", "integration-receipt-a")
	if !replay.Accepted || !replay.Replay || !reflect.DeepEqual(replay.Batch, batch) {
		t.Fatalf("replay=%+v batch=%+v", replay, batch)
	}
	if got := RegisterAutoprogrammingBatchIntegrationV0(batch, receiptExpected, "integrate-a", "integration-claim-a", "task-a", "source-a", "base-revision-001", "revision-a", "integration-receipt-other"); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_idempotency_key_reused") {
		t.Fatalf("divergence=%+v", got)
	}
	if got := RegisterAutoprogrammingBatchIntegrationV0(batch, receiptExpected, "integrate-a", "integration-claim-a", "task-a", "source-other", "base-revision-001", "revision-a", "integration-receipt-a"); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_idempotency_key_reused") {
		t.Fatalf("source divergence=%+v", got)
	}

	if got := ClaimAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "claim-b-stale", "integration-claim-b-stale", "task-b", "base-revision-001"); got.Accepted || !hasAutoprogrammingBatchIssueV0(got.Issues, "batch_integration_parent_revision_invalid") {
		t.Fatalf("stale head=%+v", got)
	}
	batch = integrateAutoprogrammingBatchMemberV0(t, batch, "b", "task-b", "source-b", "revision-a", "revision-b")
	if batch.Status != AutoprogrammingBatchStatusPendingBatchGateV0 || batch.IntegratedRevision != "revision-b" {
		t.Fatalf("batch=%+v", batch)
	}
	if batch.Members[0].SourceRevision != "source-a" || batch.Members[0].IntegrationRevision != "revision-a" || batch.Members[0].IntegrationReceiptRef != "integration-receipt-a" || batch.Members[1].SourceRevision != "source-b" || batch.Members[1].IntegrationRevision != "revision-b" || batch.Members[1].IntegrationReceiptRef != "integration-receipt-b" {
		t.Fatalf("members=%+v", batch.Members)
	}
}

func TestAutoprogrammingBatchV0FailedReceiptRequiresReworkAndInvalidatesIntegration(t *testing.T) {
	batch := readyAutoprogrammingBatchGateV0(t)
	previousGeneration := batch.GateGeneration
	hash := AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
	batch = mustBatchTransitionV0(t, ClaimAutoprogrammingBatchTestV0(batch, batch.StoreVersion, "claim-a", "revision-001", hash, "claim-ref-a"))
	batch = mustBatchTransitionV0(t, RecordAutoprogrammingBatchTestReceiptV0(batch, batch.StoreVersion, "failed-a", "revision-001", hash, "claim-ref-a", "receipt-ref-a", AutoprogrammingBatchTestReceiptFailedV0))
	if batch.Status != AutoprogrammingBatchStatusReworkPendingV0 {
		t.Fatalf("status=%q", batch.Status)
	}
	batch = mustBatchTransitionV0(t, RequestAutoprogrammingBatchReworkV0(batch, batch.StoreVersion, "rework-a", "task-a"))
	if batch.Status != AutoprogrammingBatchStatusGoalsRunningV0 || batch.GateGeneration != previousGeneration+1 || batch.IntegratedRevision != "" || batch.Members[0].FocalStatus != AutoprogrammingBatchFocalRunningV0 {
		t.Fatalf("batch=%+v", batch)
	}
	if autoprogrammingBatchAllTestsPassedV0(batch) || batch.TestClaims[0].GateGeneration != previousGeneration || batch.TestReceipts[0].GateGeneration != previousGeneration {
		t.Fatalf("old generation evidence reused: %+v", batch)
	}
	for _, member := range batch.Members {
		if member.IntegrationStatus != AutoprogrammingBatchIntegrationPendingV0 {
			t.Fatalf("member=%+v", member)
		}
	}
}

func TestAutoprogrammingBatchPlanHashV0IsDeterministic(t *testing.T) {
	plan := autoprogrammingBatchPlanFixtureV0()
	first := mustNewAutoprogrammingBatchV0(t)
	plan.Members[0], plan.Members[1] = plan.Members[1], plan.Members[0]
	plan.FrozenTests[0], plan.FrozenTests[1] = plan.FrozenTests[1], plan.FrozenTests[0]
	second := NewAutoprogrammingBatchV0(plan)
	if !second.Accepted || first.PlanHash != second.Batch.PlanHash {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	changed := autoprogrammingBatchPlanFixtureV0()
	changed.FrozenTests[0].Command = "go test -count=1 ./modulos/other"
	third := NewAutoprogrammingBatchV0(changed)
	if !third.Accepted || first.PlanHash == third.Batch.PlanHash {
		t.Fatalf("first=%q third=%q", first.PlanHash, third.Batch.PlanHash)
	}
}

func mustNewAutoprogrammingBatchV0(t *testing.T) AutoprogrammingBatchV0 {
	t.Helper()
	result := NewAutoprogrammingBatchV0(autoprogrammingBatchPlanFixtureV0())
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	for _, member := range result.Batch.Members {
		if member.SourceRevision != "" || member.IntegrationRevision != "" || member.IntegrationReceiptRef != "" {
			t.Fatalf("new member contains integration state: %+v", member)
		}
	}
	return result.Batch
}

func readyAutoprogrammingBatchGateV0(t *testing.T) AutoprogrammingBatchV0 {
	t.Helper()
	batch := mustNewAutoprogrammingBatchV0(t)
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 1, "launch-a", "task-a"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchLaunchV0(batch, 2, "launch-b", "task-b"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchFocalCloseV0(batch, 3, "close-a", "task-a"))
	batch = mustBatchTransitionV0(t, RegisterAutoprogrammingBatchFocalCloseV0(batch, 4, "close-b", "task-b"))
	batch = integrateAutoprogrammingBatchMemberV0(t, batch, "a", "task-a", "source-a", "base-revision-001", "revision-a")
	return integrateAutoprogrammingBatchMemberV0(t, batch, "b", "task-b", "source-b", "revision-a", "revision-001")
}

func integrateAutoprogrammingBatchMemberV0(t *testing.T, batch AutoprogrammingBatchV0, suffix, taskRef, sourceRevision, parentRevision, integrationRevision string) AutoprogrammingBatchV0 {
	t.Helper()
	claimRef := "integration-claim-" + suffix
	batch = mustBatchTransitionV0(t, ClaimAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "claim-integration-"+suffix, claimRef, taskRef, parentRevision))
	return mustBatchTransitionV0(t, RegisterAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "receipt-integration-"+suffix, claimRef, taskRef, sourceRevision, parentRevision, integrationRevision, "integration-receipt-"+suffix))
}

func mustBatchTransitionV0(t *testing.T, result AutoprogrammingBatchTransitionResultV0) AutoprogrammingBatchV0 {
	t.Helper()
	if !result.Accepted {
		t.Fatalf("issues=%+v batch=%+v", result.Issues, result.Batch)
	}
	return result.Batch
}

func hasAutoprogrammingBatchIssueV0(issues []AutoprogrammingRequestIssueV0, want string) bool {
	for _, issue := range issues {
		if issue.Code == want {
			return true
		}
	}
	return false
}

func autoprogrammingBatchPlanFixtureV0() AutoprogrammingBatchPlanV0 {
	return AutoprogrammingBatchPlanV0{
		BatchRef: "batch-ref-001", RequestRef: "request-ref-001", ProjectRef: "project-ref-001", BaseRevision: "base-revision-001",
		Members: []AutoprogrammingBatchMemberV0{
			{TaskRef: "task-a", GoalRef: "goal-a", RunRef: "run-a", WorkspaceRef: "workspace-a", WriteSet: []string{"modulos/orquesta-autoprogramming/a.go"}},
			{TaskRef: "task-b", GoalRef: "goal-b", RunRef: "run-b", WorkspaceRef: "workspace-b", WriteSet: []string{"modulos/orquesta-autoprogramming/b.go"}},
		},
		FrozenTests: []AutoprogrammingBatchTestV0{
			{Command: "go test -count=1 ./modulos/orquesta-autoprogramming", SHA256: strings.Repeat("a", 64)},
			{Command: "go test -count=1 ./modulos/orquesta-goal", SHA256: strings.Repeat("b", 64)},
		},
	}
}
