package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

func TestRealCodexBudgetsAndEffectsThroughProductionComposition(t *testing.T) {
	root := t.TempDir()
	invocationsPath := filepath.Join(root, "codex-v15-invocations")
	helperPath := filepath.Join(root, "codex-v15-helper.sh")
	helper := "#!/bin/sh\n" +
		"output=\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--output-last-message\" ]; then shift; output=$1; fi\n" +
		"  shift\n" +
		"done\n" +
		"prompt=$(/bin/cat)\n" +
		"test -n \"$prompt\" && test -n \"$output\" || exit 73\n" +
		"printf 'launch\\n' >> " + strconv.Quote(invocationsPath) + "\n" +
		"printf '%s\\n' '{\"artifact\":\"artifact:v15-production-composition\"}' > \"$output\"\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\ncommand = "+strconv.Quote(helperPath)+"\ntimeout = \"5s\"",
	)
	replaceTestConfigValue(t, configPath, "max_concurrent_executions = 4\n", "")

	runtime, err := Build(context.Background(), Options{ConfigPath: configPath, Version: "v15-budget-effect-e2e"})
	if err != nil {
		t.Fatalf("build production composition: %v cause=%v", err, errors.Unwrap(err))
	}
	if runtime.config.RuntimeCodexMaxConcurrentExecutions() != 70 ||
		runtime.config.SchedulerMaxChildrenPerParent() != 6 {
		t.Fatalf("production defaults = launches %d fanout %d",
			runtime.config.RuntimeCodexMaxConcurrentExecutions(), runtime.config.SchedulerMaxChildrenPerParent())
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start production composition: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})

	access := testRuntimeAccess(t, runtime)
	created, err := runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:v15-production-budget-effect",
		Statement:  "produce V15 budget and effect evidence",
		Confirm:    true,
	})
	if err != nil {
		t.Fatalf("submit governed Goal: %v", err)
	}
	closed := waitTerminalGoal(t, runtime, created.Record.Goal.Ref())
	if closed.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("governed Goal state = %s", closed.Goal.State())
	}

	invocations, err := os.ReadFile(invocationsPath)
	if err != nil || strings.Count(string(invocations), "launch\n") != 1 {
		t.Fatalf("physical Codex launches = %q err=%v", invocations, err)
	}
	assertProductionBudgetLedger(t, runtime, closed)
	assertProductionLaunchEffectLedger(t, closed)
}

func assertProductionBudgetLedger(t *testing.T, runtime *Runtime, record application.GoalRecord) {
	t.Helper()
	expected, err := buildBudgetPolicy(runtime.config, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(record.BudgetEnvelopes) != 3 || len(record.BudgetReservations) != 1 ||
		len(record.BudgetSettlements) != 1 {
		t.Fatalf("budget ledger sizes = envelopes %d reservations %d settlements %d",
			len(record.BudgetEnvelopes), len(record.BudgetReservations), len(record.BudgetSettlements))
	}
	wantSubjects := map[governance.BudgetScope]string{
		governance.BudgetScopeDeployment: "deployment:local",
		governance.BudgetScopeProject:    record.Goal.Project().String(),
		governance.BudgetScopeGoal:       record.Goal.Ref().String(),
	}
	seen := make(map[governance.BudgetScope]bool, 3)
	for _, envelope := range record.BudgetEnvelopes {
		if seen[envelope.Scope] || envelope.SubjectRef != wantSubjects[envelope.Scope] ||
			envelope.Limit != expected.DeploymentEnvelope.Limit || envelope.PolicyHash != expected.PolicyHash {
			t.Fatalf("invalid materialized budget envelope: %+v", envelope)
		}
		seen[envelope.Scope] = true
	}
	item := record.Goal.WorkItems()[0]
	reservation := record.BudgetReservations[0]
	if item.BudgetDemand().Resources != expected.DefaultWorkItemDemand ||
		reservation.Resources != expected.DefaultWorkItemDemand || reservation.DemandRef != item.BudgetDemand().Ref ||
		reservation.PolicyHash != expected.PolicyHash || record.Executions[0].BudgetReservationRef != reservation.Ref {
		t.Fatalf("budget demand/reservation mismatch: item=%+v reservation=%+v", item.BudgetDemand(), reservation)
	}
	settlement := record.BudgetSettlements[0]
	localDimensions := governance.ResourceActiveTime | governance.ResourceProcessSlots | governance.ResourceDisk
	if settlement.ReservationRef != reservation.Ref || settlement.Reserved != reservation.Resources ||
		settlement.Observed.Known&localDimensions != localDimensions ||
		settlement.Observed.Known&(governance.ResourceTokens|governance.ResourceMoney) != 0 ||
		settlement.Charged.Tokens != reservation.Resources.Tokens ||
		settlement.Charged.MoneyMicros != reservation.Resources.MoneyMicros ||
		settlement.Released.ProcessSlots != 1 {
		t.Fatalf("conservative budget settlement mismatch: %+v", settlement)
	}
}

func assertProductionLaunchEffectLedger(t *testing.T, record application.GoalRecord) {
	t.Helper()
	if len(record.EffectIntents) != 1 || len(record.EffectApprovals) != 1 ||
		len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 1 {
		t.Fatalf("effect ledger sizes = intents %d approvals %d attempts %d receipts %d",
			len(record.EffectIntents), len(record.EffectApprovals), len(record.EffectAttempts), len(record.EffectReceipts))
	}
	intent, approval := record.EffectIntents[0], record.EffectApprovals[0]
	attempt, receipt := record.EffectAttempts[0], record.EffectReceipts[0]
	if intent.Kind != application.EffectKindAgentLaunch || approval.IntentRef != intent.Ref ||
		approval.Decision != application.EffectApproved || approval.Source != application.EffectApprovalSourceGoalConfirmation ||
		attempt.IntentRef != intent.Ref || attempt.ApprovalRef != approval.Ref || receipt.AttemptRef != attempt.Ref ||
		receipt.IntentRef != intent.Ref || receipt.ApprovalRef != approval.Ref || receipt.ExternalRef == "" ||
		receipt.Usage.Quality != governance.UsageQualityUnknown {
		t.Fatalf("effect causal chain mismatch: intent=%+v approval=%+v attempt=%+v receipt=%+v",
			intent, approval, attempt, receipt)
	}
	linked := 0
	for _, consumption := range record.ConsumptionReceipts {
		if consumption.EffectReceiptRef != "" {
			linked++
			if consumption.EffectReceiptRef != receipt.Ref {
				t.Fatalf("outbox receipt points to %q want %q", consumption.EffectReceiptRef, receipt.Ref)
			}
		}
	}
	if linked != 1 {
		t.Fatalf("external effect receipt links = %d", linked)
	}
}
