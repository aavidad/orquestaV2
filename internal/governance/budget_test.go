package governance_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"orquesta/internal/governance"
)

func TestBudgetContractUsesOneCanonicalEnvelopeAcrossLayers(t *testing.T) {
	currency, err := governance.NewCurrency("eur")
	if err != nil || currency != "EUR" {
		t.Fatalf("canonical currency = %q, %v", currency, err)
	}
	envelope := governance.BudgetEnvelope{
		Ref: "budget:deployment", SubjectRef: "deployment:local", Scope: governance.BudgetScopeDeployment,
		Limit:    governance.ResourceVector{Tokens: 100, MoneyMicros: 500, Currency: currency, ActiveTimeNS: 1_000, ProcessSlots: 2, DiskBytes: 2_000},
		Revision: 1, PolicyHash: strings.Repeat("a", 64), CreatedAt: time.Unix(1, 0).UTC(),
	}
	if err := governance.ValidateBudgetEnvelope(envelope); err != nil {
		t.Fatal(err)
	}
	reserved := governance.ResourceVector{Tokens: 25, MoneyMicros: 100, Currency: currency, ActiveTimeNS: 100, ProcessSlots: 1, DiskBytes: 250}
	demand := governance.BudgetDemand{Ref: "demand:work", Resources: governance.ResourceVector{
		Tokens: 50, MoneyMicros: 200, Currency: currency, ActiveTimeNS: 500, ProcessSlots: 1, DiskBytes: 500,
	}}
	if err := governance.ValidateBudgetDemand(demand); err != nil {
		t.Fatal(err)
	}
	requested, err := governance.Add(reserved, demand.Resources)
	if err != nil {
		t.Fatal(err)
	}
	if fits, err := governance.Fits(envelope.Limit, requested); err != nil || !fits {
		t.Fatalf("fits = %v, %v", fits, err)
	}
	if _, err := governance.Add(governance.ResourceVector{Tokens: math.MaxInt64}, governance.ResourceVector{Tokens: 1}); governance.ErrorCodeOf(err) != governance.ErrorOverflow {
		t.Fatalf("overflow code = %q, err=%v", governance.ErrorCodeOf(err), err)
	}

	reservation := validReservation("reservation:work", demand.Ref, demand.Resources)
	usage := governance.ResourceUsage{
		Resources: governance.ResourceVector{ActiveTimeNS: 350, ProcessSlots: 0},
		Known:     governance.ResourceActiveTime | governance.ResourceProcessSlots, Quality: governance.UsageQualityMeasured,
	}
	settlement, err := governance.Reconcile(reservation, usage)
	if err != nil {
		t.Fatal(err)
	}
	if settlement.Charged.Tokens != demand.Resources.Tokens || settlement.Charged.MoneyMicros != demand.Resources.MoneyMicros ||
		settlement.Charged.DiskBytes != demand.Resources.DiskBytes {
		t.Fatalf("unknown dimensions were not conservatively charged: %+v", settlement.Charged)
	}
	if settlement.Charged.ActiveTimeNS != 350 || settlement.Released.ActiveTimeNS != 150 ||
		settlement.Charged.ProcessSlots != 0 || settlement.Released.ProcessSlots != 1 {
		t.Fatalf("known usage did not release reservation: %+v", settlement)
	}
	if err := governance.ValidateBudgetSettlement(settlement); err != nil {
		t.Fatal(err)
	}
}

func TestResourceContractRejectsNegativesCurrencyConflictAndUnknownValues(t *testing.T) {
	eur, _ := governance.NewCurrency("EUR")
	usd, _ := governance.NewCurrency("USD")
	if err := governance.ValidateResourceVector(governance.ResourceVector{Tokens: -1}); governance.ErrorCodeOf(err) != governance.ErrorInvalidArgument {
		t.Fatalf("negative code = %q", governance.ErrorCodeOf(err))
	}
	if _, err := governance.Add(
		governance.ResourceVector{MoneyMicros: 1, Currency: eur},
		governance.ResourceVector{MoneyMicros: 1, Currency: usd},
	); governance.ErrorCodeOf(err) != governance.ErrorCurrencyConflict {
		t.Fatalf("currency conflict code = %q", governance.ErrorCodeOf(err))
	}
	usage := governance.ResourceUsage{Resources: governance.ResourceVector{Tokens: 1}, Quality: governance.UsageQualityUnknown}
	if err := governance.ValidateResourceUsage(usage); governance.ErrorCodeOf(err) != governance.ErrorInvalidArgument {
		t.Fatalf("unknown value code = %q", governance.ErrorCodeOf(err))
	}
}

func TestReconcileRecordsKnownOverrunWithoutMintingRelease(t *testing.T) {
	reservation := validReservation("reservation:1", "demand:1",
		governance.ResourceVector{Tokens: 10, ActiveTimeNS: 20, ProcessSlots: 1})
	usage := governance.ResourceUsage{
		Resources: governance.ResourceVector{Tokens: 15, ActiveTimeNS: 5, ProcessSlots: 0},
		Known:     governance.ResourceTokens | governance.ResourceActiveTime | governance.ResourceProcessSlots,
		Quality:   governance.UsageQualityExact,
	}
	settlement, err := governance.Reconcile(reservation, usage)
	if err != nil {
		t.Fatal(err)
	}
	if settlement.Charged.Tokens != 15 || settlement.Overrun.Tokens != 5 || settlement.Released.Tokens != 0 {
		t.Fatalf("token reconciliation = %+v", settlement)
	}
	if settlement.Released.ActiveTimeNS != 15 || settlement.Overrun.ActiveTimeNS != 0 || settlement.Released.ProcessSlots != 1 {
		t.Fatalf("release reconciliation = %+v", settlement)
	}
	settlement.Charged.Tokens--
	if err := governance.ValidateBudgetSettlement(settlement); governance.ErrorCodeOf(err) != governance.ErrorInvalidArgument {
		t.Fatalf("tampered settlement code = %q", governance.ErrorCodeOf(err))
	}
}

func validReservation(ref, demandRef string, resources governance.ResourceVector) governance.BudgetReservation {
	return governance.BudgetReservation{
		Ref: ref, DemandRef: demandRef, ActionRef: "action:1", EffectIntentRef: "effect-intent:1",
		ProjectRef: "project:1", GoalRef: "goal:1", WorkItemRef: "work-item:1", ExecutionRef: "execution:1",
		PlanGeneration: 1, AppSpecGeneration: 1, WorkItemGeneration: 1, Fence: 1,
		SpecHash: strings.Repeat("b", 64), PolicyHash: strings.Repeat("c", 64),
		Resources: resources, ReservedAt: time.Unix(2, 0).UTC(),
	}
}
