package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestFailedObservationSettlementPreservesReportedUsage(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 18, 17, 30, 0, 0, time.UTC)}
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentFailed, ErrorCode: "agent.test_failed", Usage: governance.ResourceUsage{
			Resources: governance.ResourceVector{Tokens: 7},
			Known:     governance.ResourceTokens, Quality: governance.UsageQualityExact,
		},
	}}}
	repository := newMemoryRepository()
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:failed-observation-usage", Statement: "preserve observed usage", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:usage-launch"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(context.Background(), "worker:usage-observe"); err != nil {
		t.Fatal(err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || len(record.BudgetReservations) != 1 || len(record.BudgetSettlements) != 1 {
		t.Fatalf("failed usage ledger missing: record=%+v err=%v", record, err)
	}
	reservation, settlement := record.BudgetReservations[0], record.BudgetSettlements[0]
	if settlement.Observed.Resources.Tokens != 7 || settlement.Charged.Tokens != 7 ||
		settlement.Charged.MoneyMicros != reservation.Resources.MoneyMicros ||
		settlement.Observed.Known&governance.ResourceTokens == 0 ||
		settlement.Observed.Known&governance.ResourceMoney != 0 ||
		settlement.Observed.Quality != governance.UsageQualityMeasured ||
		settlement.Observed.Resources.DiskBytes != 0 {
		t.Fatalf("reported failure usage was lost: reservation=%+v settlement=%+v", reservation, settlement)
	}
}
