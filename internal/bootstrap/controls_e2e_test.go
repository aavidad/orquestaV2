package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"orquesta/internal/adapters/auth/localtoken"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestRealCodexCooperativeStopLeavesResidentSchedulerLive(t *testing.T) {
	root := t.TempDir()
	readyPath := filepath.Join(root, "ignore-term.ready")
	helperPath := filepath.Join(root, "codex-control-helper.sh")
	helper := "#!/bin/sh\n" +
		"output=\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--output-last-message\" ]; then shift; output=$1; fi\n" +
		"  shift\n" +
		"done\n" +
		"prompt=$(/bin/cat)\n" +
		"case \"$prompt\" in\n" +
		"  *bootstrap-helper:ignore-term*)\n" +
		"    trap '' TERM\n" +
		"    printf ready > " + strconv.Quote(readyPath) + "\n" +
		"    while :; do /bin/sleep 1; done\n" +
		"    ;;\n" +
		"  *)\n" +
		"    printf '%s\\n' '{\"artifact\":\"artifact:resident-scheduler-progress\"}' > \"$output\"\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(helperPath, []byte(helper), 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := writeTestConfig(t, root)
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\ncommand = "+strconv.Quote(helperPath)+"\ntimeout = \"5s\"",
	)
	configurarPerfilCuotaCodexPrueba(t, root, configPath, helperPath, 2)
	configureTestCodexRuntimeCgroup(t, configPath)
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "v14-cooperative-scheduler-liveness",
	})
	if err != nil {
		t.Fatalf("build production composition: %v", err)
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
	blocked, err := runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:v14-cooperative-blocked", Statement: "bootstrap-helper:ignore-term", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit TERM-resistant Goal: %v", err)
	}
	running := waitForRunningExecution(t, runtime, access, blocked.Record.Goal.Ref())
	waitForControlHelperReady(t, readyPath)
	item := running.Goal.WorkItems()[0]
	execution := running.Executions[0]
	cooperative, err := runtime.Orchestrator().Control(context.Background(), access, application.ControlRequest{
		RequestRef: "request:v14-cooperative-pending", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "yield scheduler while TERM is ignored",
	})
	if err != nil || !cooperative.Created {
		t.Fatalf("persist cooperative control: result=%+v err=%v", cooperative, err)
	}

	progress, err := runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:v14-scheduler-progress", Statement: "bootstrap-helper:progress", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit unrelated Goal: %v", err)
	}
	progressed := waitTerminalGoal(t, runtime, progress.Record.Goal.Ref())
	if progressed.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("unrelated Goal did not progress through sole scheduler: %s", progressed.Goal.State())
	}

	latest, err := runtime.Orchestrator().GetGoal(context.Background(), access, running.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item = latest.Goal.WorkItems()[0]
	execution = latest.Executions[0]
	cooperativePending := false
	for _, control := range latest.Controls {
		if control.Ref == cooperative.Control.Ref && control.Status == application.ControlRequested {
			cooperativePending = true
		}
	}
	if execution.State != application.ExecutionRunning || !cooperativePending {
		t.Fatalf("cooperative stop did not remain pending while unrelated Goal progressed: execution=%s controls=%+v",
			execution.State, latest.Controls)
	}
	forced, err := runtime.Orchestrator().Control(context.Background(), access, application.ControlRequest{
		RequestRef: "request:v14-forced-after-cooperative", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: latest.Goal.Ref(),
		ExpectedGoalRevision: latest.Goal.Revision(), ExpectedPlanGeneration: latest.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: latest.Goal.AppSpec().Generation(), ExpectedSpecHash: latest.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopForced, Reason: "escalate exact pending cooperative stop",
	})
	if err != nil || !forced.Created || forced.Control.SupersedesControlRef != cooperative.Control.Ref {
		t.Fatalf("persist forced escalation: result=%+v err=%v", forced, err)
	}
	approveRuntimeForcedStop(t, runtime, access, forced.Control, execution.Ref,
		"request:v15-approve-forced-after-cooperative")
	closed := waitForStoppedExecution(t, runtime, access, latest.Goal.Ref(), forced.Control.Ref)
	cooperativeFound := false
	for _, control := range closed.Controls {
		if control.Ref == cooperative.Control.Ref {
			cooperativeFound = true
			if control.Status != application.ControlSuperseded {
				t.Fatalf("cooperative control after forced convergence = %+v", control)
			}
		}
	}
	if !cooperativeFound {
		t.Fatalf("cooperative control disappeared after forced convergence: %+v", closed.Controls)
	}
}

func waitForControlHelperReady(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("TERM-resistant control helper never became ready: %s", path)
}

func TestRealCodexControlsThroughProductionComposition(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	replaceTestConfigValue(t, configPath, "[runtime.codex]\ntimeout = \"1s\"",
		"[runtime.codex]\ncommand = "+strconv.Quote(executable)+"\ntimeout = \"5s\"",
	)
	configurarPerfilCuotaCodexPrueba(t, root, configPath, executable, 1)
	// Race instrumentation can extend cgroup/process-tree stop settlement beyond
	// the shared one-second fixture lease. Match this real-runtime case to the
	// existing ten-second observation window so its fenced receipt can settle.
	replaceTestConfigValue(t, configPath, "claim_lease = \"1s\"", "claim_lease = \"10s\"")
	configureTestCodexRuntimeCgroup(t, configPath)
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configPath, Version: "v14-real-codex-controls",
	})
	if err != nil {
		t.Fatalf("build production composition: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start production composition: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})

	controller, ok := runtime.agent.(application.AgentController)
	if !ok {
		t.Fatalf("production Codex composition omitted AgentController: %T", runtime.agent)
	}
	capabilities, err := controller.ControlCapabilities(context.Background())
	if err != nil || !capabilities.ForcedStop {
		t.Fatalf("production Codex control capability unavailable: %+v err=%v", capabilities, err)
	}

	access := testRuntimeAccess(t, runtime)
	created, err := runtime.Orchestrator().Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:v14-production-control", Statement: "bootstrap-helper:block", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit controlled Goal: %v", err)
	}
	running := waitForRunningExecution(t, runtime, access, created.Record.Goal.Ref())
	item := running.Goal.WorkItems()[0]
	execution := running.Executions[0]
	controlled, err := runtime.Orchestrator().Control(context.Background(), access, application.ControlRequest{
		RequestRef: "request:v14-production-stop", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopForced, Reason: "verify exact selective production control",
	})
	if err != nil {
		t.Fatalf("persist exact stop control: %v", err)
	}
	if !controlled.Created || controlled.Control.Status != application.ControlRequested {
		t.Fatalf("stop was not durably requested: %+v", controlled)
	}
	approveRuntimeForcedStop(t, runtime, access, controlled.Control, execution.Ref,
		"request:v15-approve-production-stop")
	closed := waitForStoppedExecution(t, runtime, access, created.Record.Goal.Ref(), controlled.Control.Ref)
	stoppedItem := closed.Goal.WorkItems()[0]
	if closed.Goal.IsTerminal() || closed.Goal.State() != goal.GoalStateRunning ||
		stoppedItem.State() != goal.WorkItemStateInterrupted ||
		closed.Executions[0].State != application.ExecutionStopped {
		t.Fatalf("stop did not preserve V14 lifecycle: goal=%s item=%s execution=%s",
			closed.Goal.State(), stoppedItem.State(), closed.Executions[0].State)
	}
	assertProductionStopEffectLedger(t, closed, controlled.Control, execution.Ref)
}

func assertProductionStopEffectLedger(
	t *testing.T,
	record application.GoalRecord,
	control application.ControlRecord,
	executionRef goal.ExecutionRef,
) {
	t.Helper()
	actionRef := "action:stop:" + control.Ref + ":" + executionRef.String()
	var intents, approvals, attempts, receipts, links int
	var intentRef, approvalRef, attemptRef, receiptRef string
	for _, intent := range record.EffectIntents {
		if intent.ActionRef == actionRef {
			intents++
			intentRef = intent.Ref
			if intent.Kind != application.EffectKindAgentStop {
				t.Fatalf("production stop intent kind = %q", intent.Kind)
			}
		}
	}
	for _, approval := range record.EffectApprovals {
		if approval.IntentRef == intentRef {
			approvals++
			approvalRef = approval.Ref
			if approval.Decision != application.EffectApproved ||
				approval.Source != application.EffectApprovalSourceExplicitDecision ||
				approval.DecidedBy == approval.ProposedBy {
				t.Fatalf("production stop approval = %+v", approval)
			}
		}
	}
	for _, attempt := range record.EffectAttempts {
		if attempt.IntentRef == intentRef {
			attempts++
			attemptRef = attempt.Ref
			if attempt.ApprovalRef != approvalRef || attempt.ActionRef != actionRef {
				t.Fatalf("production stop attempt = %+v", attempt)
			}
		}
	}
	for _, receipt := range record.EffectReceipts {
		if receipt.IntentRef == intentRef {
			receipts++
			receiptRef = receipt.Ref
			if receipt.ApprovalRef != approvalRef || receipt.AttemptRef != attemptRef || receipt.ActionRef != actionRef ||
				(receipt.Status != application.EffectStatusStopped &&
					receipt.Status != application.EffectStatusAlreadyStopped) {
				t.Fatalf("production stop effect receipt = %+v", receipt)
			}
		}
	}
	for _, consumption := range record.ConsumptionReceipts {
		if consumption.ActionRef == actionRef && consumption.EffectReceiptRef == receiptRef {
			links++
		}
	}
	if intents != 1 || approvals != 1 || attempts != 1 || receipts != 1 || links != 1 {
		t.Fatalf("production stop effect chain counts = intent:%d approval:%d attempt:%d receipt:%d link:%d",
			intents, approvals, attempts, receipts, links)
	}
}

func approveRuntimeForcedStop(
	t *testing.T,
	runtime *Runtime,
	ownerAccess application.Access,
	control application.ControlRecord,
	executionRef goal.ExecutionRef,
	requestRef string,
) {
	t.Helper()
	owner, hierarchy, err := localIdentityComposition(runtime.config)
	if err != nil {
		t.Fatalf("compose local owner: %v", err)
	}
	actorRef, err := goal.NewActorRef("actor:critical-effect-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	principalRef, err := identity.NewPrincipalRef("principal:critical-effect-reviewer")
	if err != nil {
		t.Fatal(err)
	}
	reviewer, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, localtoken.AuthenticationMethod,
	)
	if err != nil {
		t.Fatalf("create independent effect reviewer: %v", err)
	}
	requestedAt := time.Now().UTC()
	grant, err := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: "membership-" + requestRef, Actor: owner, TargetRef: reviewer.Ref,
		ProjectRef: hierarchy.ProjectRef(), Role: identity.RoleReviewer, RequestedAt: requestedAt,
	})
	if err != nil {
		t.Fatalf("create reviewer membership grant: %v", err)
	}
	if _, _, created, grantErr := runtime.Orchestrator().GrantMembership(
		context.Background(), ownerAccess, grant, reviewer,
	); grantErr != nil || !created {
		t.Fatalf("grant independent reviewer membership: created=%v err=%v", created, grantErr)
	}
	reviewerAccess, err := application.NewAccess(reviewer, hierarchy.ProjectRef())
	if err != nil {
		t.Fatalf("create reviewer access: %v", err)
	}
	record, err := runtime.Orchestrator().GetGoal(context.Background(), ownerAccess, control.GoalRef)
	if err != nil {
		t.Fatalf("load forced stop intent: %v", err)
	}
	wantActionRef := "action:stop:" + control.Ref + ":" + executionRef.String()
	var intent application.EffectIntent
	for _, candidate := range record.EffectIntents {
		if candidate.ActionRef == wantActionRef {
			intent = candidate
			break
		}
	}
	if intent.Ref == "" {
		t.Fatalf("forced stop intent for %s missing: %+v", wantActionRef, record.EffectIntents)
	}
	decision, err := runtime.Orchestrator().DecideEffect(context.Background(), reviewerAccess,
		application.DecideEffectRequest{
			RequestRef: requestRef, GoalRef: control.GoalRef, IntentRef: intent.Ref,
			ExpectedIntentDigest: intent.Digest, Decision: application.EffectApproved,
			Reason: "independent approval for critical forced stop",
		})
	if err != nil || !decision.Created || decision.Approval.DecidedBy == intent.ProposedBy {
		t.Fatalf("approve forced stop: result=%+v err=%v", decision, err)
	}
}

func waitForRunningExecution(
	t *testing.T,
	runtime *Runtime,
	access application.Access,
	goalRef goal.GoalRef,
) application.GoalRecord {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		record, err := runtime.Orchestrator().GetGoal(context.Background(), access, goalRef)
		if err == nil && len(record.Executions) == 1 && record.Executions[0].State == application.ExecutionRunning &&
			record.Executions[0].ExternalRef != "" {
			return record
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Goal %s never reached a controllable running execution", goalRef)
	return application.GoalRecord{}
}

func waitForStoppedExecution(
	t *testing.T,
	runtime *Runtime,
	access application.Access,
	goalRef goal.GoalRef,
	controlRef string,
) application.GoalRecord {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var last application.GoalRecord
	var lastErr error
	for time.Now().Before(deadline) {
		record, err := runtime.Orchestrator().GetGoal(context.Background(), access, goalRef)
		last, lastErr = record, err
		if err == nil && len(record.Executions) == 1 && record.Executions[0].State == application.ExecutionStopped {
			for _, control := range record.Controls {
				if control.Ref == controlRef && control.Status == application.ControlConfirmed &&
					control.ReceiptRef != "" && !control.ConfirmedAt.IsZero() {
					return record
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Goal %s never persisted its exact stopped receipt: err=%v executions=%+v controls=%+v",
		goalRef, lastErr, last.Executions, last.Controls)
	return application.GoalRecord{}
}
