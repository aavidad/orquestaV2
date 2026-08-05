package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type agentLaunchRecoveryFixture struct {
	repository       *memoryRepository
	accessRepository *memoryAccessRepository
	orchestrator     *Orchestrator
	clock            *mutableClock
	record           GoalRecord
	claim            ActionClaim
	request          ports.AgentLaunchRequest
	attempt          EffectAttempt
}

func newAgentLaunchRecoveryFixture(t *testing.T) agentLaunchRecoveryFixture {
	return newAgentLaunchRecoveryFixtureWithPreservation(t, false)
}

func newAgentLaunchRecoveryFixtureWithPreservation(
	t *testing.T,
	requiresPreservation bool,
) agentLaunchRecoveryFixture {
	return newAgentLaunchRecoveryFixtureConfigured(t, requiresPreservation, EgressPolicyAuthority{})
}

func newAgentLaunchRecoveryFixtureWithEgress(
	t *testing.T,
	policy EgressPolicyAuthority,
) agentLaunchRecoveryFixture {
	return newAgentLaunchRecoveryFixtureConfigured(t, false, policy)
}

func newAgentLaunchRecoveryFixtureConfigured(
	t *testing.T,
	requiresPreservation bool,
	policy EgressPolicyAuthority,
) agentLaunchRecoveryFixture {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	actor, project := testScope(t)
	principalRef, err := identity.NewPrincipalRef(actor.String())
	if err != nil {
		t.Fatal(err)
	}
	accessRepository := newMemoryAccessRepository()
	accessRepository.setRole(principalRef, project, identity.RoleProjectOwner)
	orchestrator, _ := newTestOrchestratorWithAccess(
		t, repository, accessRepository, clock, &scriptedAgent{now: clock.Now},
	)
	orchestrator.agentCapabilities.RequierePreservacionEntorno = requiresPreservation
	submitRequest := SubmitRequest{
		RequestRef: "request:agent-launch-recovery", Statement: "recover exact launch", Confirm: true,
	}
	if policy != (EgressPolicyAuthority{}) {
		orchestrator.egressPolicies = &egressPolicyResolverStub{authority: policy}
		submitRequest.Plan = &PlanSpec{
			Phases: []PhaseSpec{{Ref: "phase-instance:recovery-egress", Key: goal.DefaultPhaseKey().String(),
				TemplateRef: "phase-template:recovery-egress"}},
			WorkItems: []WorkItemSpec{{Key: "work", Objective: "recover exact governed launch",
				Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
				OutputContract: goal.OutputContractEvidenceBundle, EgressPolicyRef: policy.PolicyRef.String()}},
		}
	}
	submitted, err := orchestrator.Submit(context.Background(), accessForScope(t, actor, project), submitRequest)
	if err != nil {
		t.Fatal(err)
	}
	claim, found, err := orchestrator.ClaimNextAction(
		context.Background(), "worker:launch-before-restart", ActionClaimSelection{},
	)
	if err != nil || !found {
		t.Fatalf("claim found=%t err=%v", found, err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	phase, phaseFound := phaseForWorkItem(record.Goal, item)
	if !found || !executionFound || !phaseFound {
		t.Fatal("launch causal records missing")
	}
	execution.BudgetReservationRef = claim.BudgetReservationRef
	execution.EffectIntentRef = claim.Action.EffectIntent.Ref
	record, item, execution, proceed, err := orchestrator.prepareLaunchDispatch(
		context.Background(), claim, record, item, execution,
	)
	if err != nil || !proceed {
		t.Fatalf("prepare proceed=%t err=%v", proceed, err)
	}
	request := agentLaunchRequest(record.Goal, item, execution, phase)
	if err := bindDurableAgentLaunchEgressAuthority(record, &request); err != nil {
		t.Fatal(err)
	}
	request.ReferenciaColocacion = claim.ReferenciaColocacion
	request.RequierePreservacionEntorno = execution.RequierePreservacionEntorno
	attempt := EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: claim.Action.EffectIntent.Ref, IntentDigest: claim.Action.EffectIntent.Digest,
		ApprovalRef: claim.EffectApproval.Ref, Subject: claim.Action.EffectIntent.Subject,
		ActionRef: claim.Action.Ref, ActionFence: claim.Fence, WorkerRef: claim.WorkerRef,
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		StartedAt:      clock.Now().UTC(), ClaimLeaseUntil: claim.LeaseUntil,
	}
	record.EffectAttempts = append(record.EffectAttempts, attempt)
	claim.Token = "claim:after-restart"
	claim.WorkerRef = "worker:launch-after-restart"
	claim.DeliveryAttempt++
	claim.Fence++
	claim.Disposition = ActionClaimDispositionRecoverEffect
	claim.RecoveryEffectAttemptRef = attempt.Ref
	claim.LeaseUntil = attempt.ClaimLeaseUntil.Add(time.Minute)
	return agentLaunchRecoveryFixture{
		repository: repository, accessRepository: accessRepository,
		orchestrator: orchestrator, clock: clock,
		record: record, claim: claim, request: request, attempt: attempt,
	}
}

func TestProcessClaimRecoveryRebuildsDurableEgressWithoutCatalog(t *testing.T) {
	policy := testEgressPolicyAuthority(t, "egress-policy:recovery", `{"destinations":["example.org"]}`)
	fixture := newAgentLaunchRecoveryFixtureWithEgress(t, policy)
	launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
	fixture.orchestrator.launcher = launcher
	fixture.orchestrator.egressPolicies = &egressPolicyResolverStub{err: errors.New("catalog unavailable after admission")}
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	launcher.reconcileAcceptedAt = fixture.attempt.StartedAt.Add(time.Second)
	fixture.persistRecoveryClaim()

	if _, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim); err != nil {
		t.Fatalf("recover exact durable egress: %v", err)
	}
	resolver := fixture.orchestrator.egressPolicies.(*egressPolicyResolverStub)
	want := expectedAgentLaunchEgressAuthority(policy)
	if len(resolver.refs) != 0 || len(launcher.reconcileRequests) != 1 ||
		!ports.EqualAgentLaunchEgressAuthority(launcher.reconcileRequests[0].EgressAuthority, want) {
		t.Fatalf("catalog refs=%v reconcile=%+v want=%+v", resolver.refs, launcher.reconcileRequests, want)
	}
}

func TestProcessClaimRecoveryRejectsDurableEgressSwapBeforeReconciler(t *testing.T) {
	policy := testEgressPolicyAuthority(t, "egress-policy:recovery-original", `{"destinations":["example.org"]}`)
	fixture := newAgentLaunchRecoveryFixtureWithEgress(t, policy)
	fixture.record.WorkItemAuthorities[0].EgressPolicy = testEgressPolicyAuthority(
		t, "egress-policy:recovery-swapped", `{"destinations":["other.example"]}`,
	)
	launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
	fixture.orchestrator.launcher = launcher
	fixture.orchestrator.egressPolicies = nil
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	fixture.persistRecoveryClaim()

	_, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
	if err == nil || err.Error() != effectUnknownAppliedCode || len(launcher.reconcileRequests) != 0 {
		t.Fatalf("swapped egress crossed reconciler: reconciles=%d err=%v", len(launcher.reconcileRequests), err)
	}
}

func TestProcessClaimRecoversExactHistoricalAgentLaunch(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
	fixture.orchestrator.launcher = launcher
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	recoveryAcceptedAt := fixture.attempt.StartedAt.Add(time.Second)
	launcher.reconcileAcceptedAt = recoveryAcceptedAt
	fixture.persistRecoveryClaim()

	result, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
	if err != nil || !result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("ProcessClaim result=%+v err=%v", result, err)
	}
	if got := len(launcher.reconcileRequests); got != 1 {
		t.Fatalf("ReconcileLaunch calls=%d want=1", got)
	}
	launcher.scriptedAgent.mu.Lock()
	launchCalls := len(launcher.scriptedAgent.launchRequests)
	launcher.scriptedAgent.mu.Unlock()
	if launchCalls != 0 {
		t.Fatalf("Launch calls=%d want=0", launchCalls)
	}
	record, err := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	execution, found := executionForAction(record, fixture.claim.Action)
	if !found || execution.State != ExecutionRunning ||
		!execution.ProviderAcceptedAt.Equal(recoveryAcceptedAt) {
		t.Fatalf("execution=%+v found=%t", execution, found)
	}
	if len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 1 ||
		len(record.ConsumptionReceipts) != 1 {
		t.Fatalf("attempts=%d effect_receipts=%d consumption_receipts=%d",
			len(record.EffectAttempts), len(record.EffectReceipts), len(record.ConsumptionReceipts))
	}
	effectReceipt := record.EffectReceipts[0]
	consumption := record.ConsumptionReceipts[0]
	if effectReceipt.AttemptRef != fixture.attempt.Ref ||
		effectReceipt.ActionFence != fixture.attempt.ActionFence ||
		!effectReceipt.ConfirmedAt.Equal(recoveryAcceptedAt) ||
		consumption.Fence != fixture.claim.Fence {
		t.Fatalf("effect_receipt=%+v consumption=%+v", effectReceipt, consumption)
	}
}

func TestProcessClaimRecoveryReplaysExactExecutionSession(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	execution, found := executionForAction(fixture.record, fixture.claim.Action)
	if !found {
		t.Fatal("execution missing")
	}
	sessionRequest := ExecutionSessionRequest(fixture.record.Goal, execution)
	authority, err := DeriveExecutionSessionAuthority(sessionRequest, "execution_token")
	if err != nil {
		t.Fatal(err)
	}
	execution.ExecutionSessionRef = authority.SessionRef
	fixture.record.Executions = replaceExecution(fixture.record.Executions, execution)
	broker := &testExecutionSessionBroker{at: fixture.attempt.StartedAt}
	fixture.orchestrator.executionSessions = broker
	launcher := &recoveryCapableLauncher{
		scriptedAgent:       &scriptedAgent{now: fixture.clock.Now},
		reconcileAcceptedAt: fixture.attempt.StartedAt.Add(time.Second),
	}
	fixture.orchestrator.launcher = launcher
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	fixture.persistRecoveryClaim()

	if _, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim); err != nil {
		record, _ := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
		t.Fatalf("ProcessClaim err=%v reconciles=%d receipts=%d consumption=%+v",
			err, len(launcher.reconcileRequests), len(record.EffectReceipts), record.ConsumptionReceipts)
	}
	if len(broker.requests) != 1 || broker.requests[0] != sessionRequest ||
		len(launcher.reconcileRequests) != 1 {
		t.Fatalf("ensure=%+v reconcile=%d", broker.requests, len(launcher.reconcileRequests))
	}
	request := launcher.reconcileRequests[0]
	if request.SessionRef != authority.SessionRef ||
		request.AccessAuthority.ArtifactAccessRef != authority.ArtifactAccessRef ||
		request.AccessAuthority.MCPAccessRef != authority.MCPAccessRef ||
		request.AccessAuthority.MailboxEndpointRef != authority.MailboxEndpointRef {
		t.Fatalf("recovery request authority=%+v want=%+v", request, authority)
	}
}

func TestProcessClaimRecoveryKeepsDurableEnvironmentRequirementAcrossRestart(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixtureWithPreservation(t, true)
	execution, found := executionForAction(fixture.record, fixture.claim.Action)
	if !found || execution.State != ExecutionDispatching || !execution.RequierePreservacionEntorno {
		t.Fatalf("prepared execution lost durable preservation: execution=%+v found=%t", execution, found)
	}
	launcher := &recoveryCapableLauncher{
		scriptedAgent:       &scriptedAgent{now: fixture.clock.Now},
		reconcileAcceptedAt: fixture.attempt.StartedAt.Add(time.Second),
	}
	fixture.orchestrator.launcher = launcher
	fixture.orchestrator.agentCapabilities.RequierePreservacionEntorno = false
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	fixture.persistRecoveryClaim()

	if _, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim); err != nil {
		record, _ := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
		t.Fatalf("ProcessClaim err=%v reconciles=%d receipts=%d consumption=%+v",
			err, len(launcher.reconcileRequests), len(record.EffectReceipts), record.ConsumptionReceipts)
	}
	if len(launcher.reconcileRequests) != 1 ||
		!launcher.reconcileRequests[0].RequierePreservacionEntorno {
		t.Fatalf("reconcile request used current capabilities: %+v", launcher.reconcileRequests)
	}
	launcher.scriptedAgent.mu.Lock()
	launches := len(launcher.scriptedAgent.launchRequests)
	launcher.scriptedAgent.mu.Unlock()
	record, err := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	if err != nil || len(record.EffectAttempts) != 1 {
		t.Fatalf("attempts=%d err=%v", len(record.EffectAttempts), err)
	}
	execution, found = executionForAction(record, fixture.claim.Action)
	if launches != 0 || !found || !execution.RequierePreservacionEntorno {
		t.Fatalf("Launch=%d execution=%+v found=%t", launches, execution, found)
	}
}

func TestProcessClaimRecoveryNeverFallsBackToLaunch(t *testing.T) {
	t.Run("reconciler unavailable", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		execution, found := executionForAction(fixture.record, fixture.claim.Action)
		if !found {
			t.Fatal("execution missing")
		}
		sessionRequest := ExecutionSessionRequest(fixture.record.Goal, execution)
		authority, deriveErr := DeriveExecutionSessionAuthority(sessionRequest, "execution_token")
		if deriveErr != nil {
			t.Fatal(deriveErr)
		}
		execution.ExecutionSessionRef = authority.SessionRef
		fixture.record.Executions = replaceExecution(fixture.record.Executions, execution)
		broker := &testExecutionSessionBroker{at: fixture.attempt.StartedAt}
		fixture.orchestrator.executionSessions = broker
		fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
		fixture.persistRecoveryClaim()

		_, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
		if err == nil || err.Error() != effectUnknownAppliedCode {
			t.Fatalf("error=%v", err)
		}
		fixture.orchestrator.launcher.(*scriptedAgent).mu.Lock()
		launches := len(fixture.orchestrator.launcher.(*scriptedAgent).launchRequests)
		fixture.orchestrator.launcher.(*scriptedAgent).mu.Unlock()
		if launches != 0 || len(broker.requests) != 0 {
			t.Fatalf("Launch calls=%d Ensure calls=%d want=0", launches, len(broker.requests))
		}
		record, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
		if getErr != nil || len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 0 ||
			len(record.ConsumptionReceipts) != 1 ||
			record.ConsumptionReceipts[0].Outcome != ActionConsumedQuarantined ||
			record.ConsumptionReceipts[0].Fence != fixture.claim.Fence {
			t.Fatalf("record=%+v get_err=%v", record, getErr)
		}
		execution, found = executionForAction(record, fixture.claim.Action)
		if !found || execution.BudgetReservationRef != fixture.claim.BudgetReservationRef ||
			execution.EffectIntentRef != fixture.claim.Action.EffectIntentRef {
			t.Fatalf("unknown-applied binding lost: execution=%+v found=%t", execution, found)
		}
	})

	t.Run("invalid receipt", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
		launcher := &recoveryCapableLauncher{
			scriptedAgent: &scriptedAgent{now: fixture.clock.Now},
			reconcileOverride: func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
				receipt := launchReceiptForRequest(request, fixture.attempt.StartedAt.Add(time.Second))
				receipt.ExternalRef = ""
				return receipt, nil
			},
		}
		fixture.orchestrator.launcher = launcher
		fixture.persistRecoveryClaim()

		_, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
		if err == nil || err.Error() != effectUnknownAppliedCode || len(launcher.reconcileRequests) != 1 {
			t.Fatalf("reconciles=%d err=%v", len(launcher.reconcileRequests), err)
		}
		launcher.scriptedAgent.mu.Lock()
		launches := len(launcher.scriptedAgent.launchRequests)
		launcher.scriptedAgent.mu.Unlock()
		if launches != 0 {
			t.Fatalf("Launch calls=%d want=0", launches)
		}
	})

	t.Run("plain pending text is not temporary authority", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
		launcher := &recoveryCapableLauncher{
			scriptedAgent: &scriptedAgent{now: fixture.clock.Now},
			reconcileErr:  errors.New("provider reconciliation pending"),
		}
		fixture.orchestrator.launcher = launcher
		fixture.persistRecoveryClaim()

		_, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
		if err == nil || err.Error() != effectUnknownAppliedCode || len(launcher.reconcileRequests) != 1 {
			t.Fatalf("reconciles=%d err=%v", len(launcher.reconcileRequests), err)
		}
		record, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
		if getErr != nil || len(record.EffectAttempts) != 1 || len(record.EffectReceipts) != 0 ||
			len(record.ConsumptionReceipts) != 1 ||
			record.ConsumptionReceipts[0].Outcome != ActionConsumedQuarantined {
			t.Fatalf("record=%+v get_err=%v", record, getErr)
		}
	})
}

func TestProcessClaimRecoveryTemporaryReconciliationRequeuesExactPhysicalAttempt(t *testing.T) {
	tests := map[string]error{
		"direct":  temporaryAgentTestError{},
		"wrapped": fmt.Errorf("provider wrapper: %w", temporaryAgentTestError{}),
	}
	for name, reconcileErr := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
			launcher := &recoveryCapableLauncher{
				scriptedAgent: &scriptedAgent{now: fixture.clock.Now}, reconcileErr: reconcileErr,
			}
			fixture.orchestrator.launcher = launcher
			state := &agentLaunchRecoveryRequeueCapture{StateRepository: fixture.repository}
			fixture.orchestrator.state = state
			fixture.persistRecoveryClaim()
			beforeExecution, found := executionForAction(fixture.record, fixture.claim.Action)
			if !found {
				t.Fatal("execution missing")
			}
			operationAt := fixture.clock.Now()

			result, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
			if err != nil || !result.Processed || len(launcher.reconcileRequests) != 1 || len(state.requeues) != 1 {
				t.Fatalf("result=%+v reconciles=%d requeues=%d err=%v",
					result, len(launcher.reconcileRequests), len(state.requeues), err)
			}
			requeued := state.requeues[0]
			if !reflect.DeepEqual(requeued.Claim, fixture.claim) ||
				!reflect.DeepEqual(requeued.Execution, beforeExecution) ||
				requeued.ErrorCode != agentLaunchReconciliationPendingCode ||
				requeued.ClearEffectBinding || requeued.BudgetSettlement != nil ||
				!requeued.OperationAt.Equal(operationAt) ||
				!requeued.AvailableAt.Equal(operationAt.Add(fixture.claim.Action.EffectIntent.QuotaRetryDelay)) {
				t.Fatalf("requeue=%+v", requeued)
			}
			launcher.scriptedAgent.mu.Lock()
			launches := len(launcher.scriptedAgent.launchRequests)
			launcher.scriptedAgent.mu.Unlock()
			record, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
			execution, found := executionForAction(record, fixture.claim.Action)
			fixture.repository.mu.Lock()
			capacity := fixture.repository.reservasCapacidad[fixture.claim.Action.Ref]
			placement := fixture.repository.colocaciones[fixture.claim.Action.Ref]
			action := fixture.repository.actions[fixture.claim.Action.Ref]
			fixture.repository.mu.Unlock()
			if launches != 0 || getErr != nil || !found || !reflect.DeepEqual(execution, beforeExecution) ||
				len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != fixture.attempt ||
				len(record.EffectReceipts) != 0 || len(record.ConsumptionReceipts) != 0 ||
				len(record.BudgetSettlements) != 0 ||
				execution.BudgetReservationRef != fixture.claim.BudgetReservationRef ||
				execution.EffectIntentRef != fixture.claim.Action.EffectIntentRef ||
				capacity != fixture.claim.CapacityReservation || placement != fixture.claim.ReferenciaColocacion ||
				action.token != "" || action.workerRef != "" || !action.lease.IsZero() ||
				!action.record.AvailableAt.Equal(requeued.AvailableAt) {
				t.Fatalf("Launch=%d execution=%+v capacity=%+v placement=%s action=%+v record=%+v err=%v",
					launches, execution, capacity, placement, action, record, getErr)
			}
		})
	}
}

func TestProcessClaimRecoverySessionRetryRequiresTypedTemporaryError(t *testing.T) {
	tests := map[string]struct {
		ensureErr   error
		wantRequeue bool
	}{
		"wrapped temporary": {
			ensureErr: fmt.Errorf("session transport: %w", temporaryAgentTestError{}), wantRequeue: true,
		},
		"plain pending text": {ensureErr: errors.New("session reconciliation pending")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			execution, found := executionForAction(fixture.record, fixture.claim.Action)
			if !found {
				t.Fatal("execution missing")
			}
			authority, err := DeriveExecutionSessionAuthority(
				ExecutionSessionRequest(fixture.record.Goal, execution), "execution_token",
			)
			if err != nil {
				t.Fatal(err)
			}
			execution.ExecutionSessionRef = authority.SessionRef
			fixture.record.Executions = replaceExecution(fixture.record.Executions, execution)
			broker := &agentLaunchRecoverySessionErrorBroker{err: test.ensureErr}
			fixture.orchestrator.executionSessions = broker
			launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
			fixture.orchestrator.launcher = launcher
			state := &agentLaunchRecoveryRequeueCapture{StateRepository: fixture.repository}
			fixture.orchestrator.state = state
			fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
			fixture.persistRecoveryClaim()

			_, err = fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
			launcher.scriptedAgent.mu.Lock()
			launches := len(launcher.scriptedAgent.launchRequests)
			launcher.scriptedAgent.mu.Unlock()
			if len(broker.requests) != 1 || len(launcher.reconcileRequests) != 0 || launches != 0 {
				t.Fatalf("Ensure=%d Reconcile=%d Launch=%d err=%v",
					len(broker.requests), len(launcher.reconcileRequests), launches, err)
			}
			record, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
			if getErr != nil || len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != fixture.attempt ||
				len(record.EffectReceipts) != 0 || len(record.BudgetSettlements) != 0 {
				t.Fatalf("record=%+v err=%v", record, getErr)
			}
			if test.wantRequeue {
				if err != nil || len(state.requeues) != 1 ||
					state.requeues[0].ErrorCode != agentLaunchReconciliationPendingCode ||
					state.requeues[0].BudgetSettlement != nil || state.requeues[0].ClearEffectBinding ||
					len(record.ConsumptionReceipts) != 0 {
					t.Fatalf("requeues=%+v consumptions=%+v err=%v",
						state.requeues, record.ConsumptionReceipts, err)
				}
				return
			}
			if err == nil || err.Error() != effectUnknownAppliedCode || len(state.requeues) != 0 ||
				len(record.ConsumptionReceipts) != 1 ||
				record.ConsumptionReceipts[0].Outcome != ActionConsumedQuarantined {
				t.Fatalf("requeues=%+v consumptions=%+v err=%v",
					state.requeues, record.ConsumptionReceipts, err)
			}
		})
	}
}

type agentLaunchRecoverySessionErrorBroker struct {
	requests []ports.ExecutionSessionEnsureRequest
	err      error
}

func (broker *agentLaunchRecoverySessionErrorBroker) Ensure(
	_ context.Context, request ports.ExecutionSessionEnsureRequest,
) (ports.ExecutionSessionReceipt, error) {
	broker.requests = append(broker.requests, request)
	return ports.ExecutionSessionReceipt{}, broker.err
}

func (broker *agentLaunchRecoverySessionErrorBroker) Revoke(
	context.Context, ports.ExecutionSessionEnsureRequest,
) error {
	return errors.New("test.not_used")
}

func TestProcessClaimRecoveryRequeueCASConflictDoesNotQuarantine(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	launcher := &recoveryCapableLauncher{
		scriptedAgent: &scriptedAgent{now: fixture.clock.Now}, reconcileErr: temporaryAgentTestError{},
	}
	fixture.orchestrator.launcher = launcher
	state := &agentLaunchRecoveryRequeueConflict{StateRepository: fixture.repository}
	fixture.orchestrator.state = state
	fixture.persistRecoveryClaim()
	before, err := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
	if !IsStateError(err, StateConflict) || state.requeues != 1 || state.quarantines != 0 {
		t.Fatalf("requeues=%d quarantines=%d err=%v", state.requeues, state.quarantines, err)
	}
	after, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	if getErr != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("state changed=%t err=%v", !reflect.DeepEqual(after, before), getErr)
	}
}

type agentLaunchRecoveryRequeueConflict struct {
	StateRepository
	requeues, quarantines int
}

func (state *agentLaunchRecoveryRequeueConflict) RequeueAction(
	context.Context, ActionRequeuedState,
) error {
	state.requeues++
	return &StateError{Code: StateConflict}
}

func (state *agentLaunchRecoveryRequeueConflict) QuarantineAction(
	ctx context.Context, input ActionQuarantinedState,
) error {
	state.quarantines++
	return state.StateRepository.QuarantineAction(ctx, input)
}

func TestProcessClaimRecoveryRejectsCrossedDisposition(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	fixture.persistRecoveryClaim()

	tests := map[string]ActionClaim{
		"recovery on observe": func() ActionClaim {
			claim := fixture.claim
			claim.Action.Kind = ActionObserveAgent
			return claim
		}(),
		"recovery ref on normal": func() ActionClaim {
			claim := fixture.claim
			claim.Disposition = ActionClaimDispositionNormal
			return claim
		}(),
	}
	for name, claim := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := fixture.orchestrator.ProcessClaim(context.Background(), claim)
			if !IsStateError(err, StateConflict) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	record, err := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	if err != nil || len(record.ConsumptionReceipts) != 0 || len(record.EffectAttempts) != 1 {
		t.Fatalf("record mutated: receipts=%d attempts=%d err=%v",
			len(record.ConsumptionReceipts), len(record.EffectAttempts), err)
	}
}

func TestProcessClaimRecoveryCancellationDoesNotMutateState(t *testing.T) {
	for name, beforeCall := range map[string]bool{"before reconcile": true, "during reconcile": false} {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
			launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
			fixture.orchestrator.launcher = launcher
			fixture.persistRecoveryClaim()
			ctx, cancel := context.WithCancel(context.Background())
			if beforeCall {
				cancel()
			} else {
				launcher.reconcileErr = context.Canceled
			}
			defer cancel()

			_, err := fixture.orchestrator.ProcessClaim(ctx, fixture.claim)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error=%v", err)
			}
			record, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
			if getErr != nil || len(record.ConsumptionReceipts) != 0 || len(record.EffectAttempts) != 1 {
				t.Fatalf("record mutated: receipts=%d attempts=%d get_err=%v",
					len(record.ConsumptionReceipts), len(record.EffectAttempts), getErr)
			}
			if beforeCall && len(launcher.reconcileRequests) != 0 ||
				!beforeCall && len(launcher.reconcileRequests) != 1 {
				t.Fatalf("reconcile calls=%d before=%t", len(launcher.reconcileRequests), beforeCall)
			}
		})
	}
}

func TestProcessClaimRecoveryValidatesLiveClaimBeforeAnyEffect(t *testing.T) {
	tests := map[string]func(*testing.T, *agentLaunchRecoveryFixture){
		"expired": func(_ *testing.T, fixture *agentLaunchRecoveryFixture) {
			fixture.clock.now = fixture.claim.LeaseUntil
		},
		"reclaimed token and fence": func(_ *testing.T, fixture *agentLaunchRecoveryFixture) {
			fixture.repository.mu.Lock()
			action := fixture.repository.actions[fixture.claim.Action.Ref]
			action.token, action.workerRef = "claim:later", "worker:later"
			action.deliveryAttempt, action.fence = fixture.claim.DeliveryAttempt+1, fixture.claim.Fence+1
			action.lease = fixture.claim.LeaseUntil.Add(time.Minute)
			fixture.repository.actions[fixture.claim.Action.Ref] = action
			fixture.repository.mu.Unlock()
		},
		"crossed recovery ref": func(_ *testing.T, fixture *agentLaunchRecoveryFixture) {
			fixture.claim.RecoveryEffectAttemptRef = "effect-attempt:crossed"
		},
		"crossed placement": func(t *testing.T, fixture *agentLaunchRecoveryFixture) {
			var err error
			fixture.claim.ReferenciaColocacion, err = ports.NewAgentPlacementRef("placement:crossed")
			if err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, invalidate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			execution, found := executionForAction(fixture.record, fixture.claim.Action)
			if !found {
				t.Fatal("execution missing")
			}
			sessionRequest := ExecutionSessionRequest(fixture.record.Goal, execution)
			authority, err := DeriveExecutionSessionAuthority(sessionRequest, "execution_token")
			if err != nil {
				t.Fatal(err)
			}
			execution.ExecutionSessionRef = authority.SessionRef
			fixture.record.Executions = replaceExecution(fixture.record.Executions, execution)
			broker := &testExecutionSessionBroker{at: fixture.attempt.StartedAt}
			fixture.orchestrator.executionSessions = broker
			launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
			fixture.orchestrator.launcher = launcher
			fixture.persistRecoveryClaim()
			invalidate(t, &fixture)
			before, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
			if getErr != nil {
				t.Fatal(getErr)
			}

			_, err = fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
			if !IsStateError(err, StateConflict) {
				t.Fatalf("error=%v", err)
			}
			launcher.scriptedAgent.mu.Lock()
			launches := len(launcher.scriptedAgent.launchRequests)
			launcher.scriptedAgent.mu.Unlock()
			after, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
			if getErr != nil || len(broker.requests) != 0 || len(launcher.reconcileRequests) != 0 || launches != 0 ||
				!reflect.DeepEqual(after, before) {
				t.Fatalf("Ensure=%d Reconcile=%d Launch=%d state_changed=%t err=%v",
					len(broker.requests), len(launcher.reconcileRequests), launches,
					!reflect.DeepEqual(after, before), getErr)
			}
		})
	}
}

func TestProcessClaimRecoveryRevokedAuthorityStopsBeforeSessionAndProvider(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	execution, found := executionForAction(fixture.record, fixture.claim.Action)
	if !found {
		t.Fatal("execution missing")
	}
	sessionRequest := ExecutionSessionRequest(fixture.record.Goal, execution)
	authority, err := DeriveExecutionSessionAuthority(sessionRequest, "execution_token")
	if err != nil {
		t.Fatal(err)
	}
	execution.ExecutionSessionRef = authority.SessionRef
	fixture.record.Executions = replaceExecution(fixture.record.Executions, execution)
	broker := &testExecutionSessionBroker{at: fixture.attempt.StartedAt}
	fixture.orchestrator.executionSessions = broker
	launcher := &recoveryCapableLauncher{scriptedAgent: &scriptedAgent{now: fixture.clock.Now}}
	fixture.orchestrator.launcher = launcher
	state := &agentLaunchRecoveryRequeueCapture{StateRepository: fixture.repository}
	fixture.orchestrator.state = state
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	fixture.persistRecoveryClaim()
	revokeAgentLaunchRecoveryAuthority(t, fixture)

	_, err = fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim)
	if err != nil || len(broker.requests) != 0 || len(launcher.reconcileRequests) != 0 ||
		len(state.requeues) != 1 || state.requeues[0].ErrorCode != agentLaunchRecoveryApprovalRequiredCode ||
		state.requeues[0].ClearEffectBinding || state.requeues[0].BudgetSettlement != nil {
		t.Fatalf("Ensure=%d Reconcile=%d err=%v",
			len(broker.requests), len(launcher.reconcileRequests), err)
	}
	launcher.scriptedAgent.mu.Lock()
	launches := len(launcher.scriptedAgent.launchRequests)
	launcher.scriptedAgent.mu.Unlock()
	record, getErr := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	execution, found = executionForAction(record, fixture.claim.Action)
	fixture.repository.mu.Lock()
	capacity := fixture.repository.reservasCapacidad[fixture.claim.Action.Ref]
	action, actionFound := fixture.repository.actions[fixture.claim.Action.Ref]
	fixture.repository.mu.Unlock()
	if launches != 0 || getErr != nil || len(record.EffectAttempts) != 1 ||
		len(record.EffectReceipts) != 0 || len(record.ConsumptionReceipts) != 0 ||
		len(record.BudgetSettlements) != 0 || !actionFound || action.token != "" ||
		action.workerRef != "" || !action.lease.IsZero() ||
		!action.record.AvailableAt.Equal(fixture.clock.Now().Add(fixture.claim.Action.EffectIntent.QuotaRetryDelay)) ||
		!found || execution.BudgetReservationRef != fixture.claim.BudgetReservationRef ||
		execution.EffectIntentRef != fixture.claim.Action.EffectIntentRef ||
		capacity.State != AgentCapacityReserved {
		t.Fatalf("Launch=%d execution=%+v capacity=%+v attempts=%d receipts=%d settlements=%d err=%v",
			launches, execution, capacity, len(record.EffectAttempts), len(record.EffectReceipts),
			len(record.BudgetSettlements), getErr)
	}
}

type agentLaunchRecoveryRequeueCapture struct {
	StateRepository
	requeues []ActionRequeuedState
}

func (state *agentLaunchRecoveryRequeueCapture) RequeueAction(
	ctx context.Context,
	requeue ActionRequeuedState,
) error {
	state.requeues = append(state.requeues, requeue)
	return state.StateRepository.RequeueAction(ctx, requeue)
}

func TestProcessClaimRecoveryKeepsReceiptWhenAuthorityChangesAfterReconcileStarts(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	fixture.clock.now = fixture.attempt.ClaimLeaseUntil.Add(time.Second)
	launcher := &recoveryCapableLauncher{
		scriptedAgent:       &scriptedAgent{now: fixture.clock.Now},
		reconcileAcceptedAt: fixture.attempt.StartedAt.Add(time.Second),
		reconcileHook: func() {
			revokeAgentLaunchRecoveryAuthority(t, fixture)
		},
	}
	fixture.orchestrator.launcher = launcher
	fixture.persistRecoveryClaim()

	if _, err := fixture.orchestrator.ProcessClaim(context.Background(), fixture.claim); err != nil {
		t.Fatal(err)
	}
	record, err := fixture.repository.GetGoal(context.Background(), fixture.record.Goal.Ref())
	if err != nil || len(launcher.reconcileRequests) != 1 || len(record.EffectAttempts) != 1 ||
		len(record.EffectReceipts) != 1 || len(record.ConsumptionReceipts) != 1 ||
		record.ConsumptionReceipts[0].Outcome != ActionConsumedCompleted {
		t.Fatalf("reconciles=%d attempts=%d receipts=%d consumption=%+v err=%v",
			len(launcher.reconcileRequests), len(record.EffectAttempts), len(record.EffectReceipts),
			record.ConsumptionReceipts, err)
	}
}

func (fixture agentLaunchRecoveryFixture) persistRecoveryClaim() {
	fixture.repository.mu.Lock()
	defer fixture.repository.mu.Unlock()
	fixture.repository.records[fixture.record.Goal.Ref()] = fixture.record
	action := fixture.repository.actions[fixture.claim.Action.Ref]
	action.token, action.workerRef = fixture.claim.Token, fixture.claim.WorkerRef
	action.deliveryAttempt, action.fence, action.lease =
		fixture.claim.DeliveryAttempt, fixture.claim.Fence, fixture.claim.LeaseUntil
	fixture.repository.actions[fixture.claim.Action.Ref] = action
}

func revokeAgentLaunchRecoveryAuthority(t *testing.T, fixture agentLaunchRecoveryFixture) {
	t.Helper()
	principalRef := fixture.claim.Action.EffectIntent.ProposedBy
	projectRef := fixture.claim.Action.EffectIntent.Subject.ProjectRef
	active, err := fixture.accessRepository.Membership(context.Background(), principalRef, projectRef)
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := identity.NewMembership(identity.MembershipInput{
		PrincipalRef: principalRef, ProjectRef: projectRef, Role: active.Role(),
		Revision: active.Revision() + 1, Status: identity.MembershipRevoked,
		GrantedBy: active.GrantedBy(), GrantedAt: active.GrantedAt(),
		RevokedBy: principalRef, RevokedAt: fixture.clock.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.accessRepository.seedMembership(revoked)
}

func TestAgentLaunchRecoveryBuildsHistoricalAuthorityUnderNewFence(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	request, attempt, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request)
	if err != nil || attempt != fixture.attempt {
		t.Fatalf("attempt=%+v err=%v", attempt, err)
	}
	want := ports.AgentLaunchEffectAuthority{
		AuthorizationReceiptRef: fixture.claim.Action.EffectIntent.Authority.Ref(),
		EffectApprovalRef:       fixture.attempt.ApprovalRef,
		EffectAttemptRef:        fixture.attempt.Ref,
		ActionFence:             fixture.attempt.ActionFence,
		StartedAt:               fixture.attempt.StartedAt,
		ClaimLeaseUntil:         fixture.attempt.ClaimLeaseUntil,
		ApprovalExpiresAt:       fixture.claim.EffectApproval.ExpiresAt,
	}
	if fixture.claim.Fence <= request.EffectAuthority.ActionFence || request.EffectAuthority != want {
		t.Fatalf("claim fence=%d authority=%+v want=%+v", fixture.claim.Fence, request.EffectAuthority, want)
	}
}

func TestAgentLaunchRecoverySelectorRejectsReceiptsAndAmbiguity(t *testing.T) {
	tests := map[string]func(*agentLaunchRecoveryFixture){
		"receipt plus ambiguous": func(fixture *agentLaunchRecoveryFixture) {
			peer := fixture.attempt
			peer.Ref += ":peer"
			peer.ActionFence++
			fixture.record.EffectAttempts = append(fixture.record.EffectAttempts, peer)
			fixture.claim.RecoveryEffectAttemptRef, fixture.claim.Fence = peer.Ref, peer.ActionFence+1
			fixture.record.EffectReceipts = append(fixture.record.EffectReceipts, recoveryEffectReceipt(fixture.attempt))
		},
		"stale crossed receipt": func(fixture *agentLaunchRecoveryFixture) {
			receipt := recoveryEffectReceipt(fixture.attempt)
			receipt.IntentRef = "effect-intent:crossed"
			fixture.record.EffectReceipts = append(fixture.record.EffectReceipts, receipt)
		},
		"two ambiguous": func(fixture *agentLaunchRecoveryFixture) {
			peer := fixture.attempt
			peer.Ref += ":peer"
			peer.ActionFence++
			fixture.record.EffectAttempts = append(fixture.record.EffectAttempts, peer)
			fixture.claim.Fence = peer.ActionFence + 1
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			mutate(&fixture)
			if _, err := SelectAgentLaunchRecoveryAttempt(fixture.record, fixture.claim); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestAgentLaunchRecoverySelectorSkipsOnlyValidZeroRelease(t *testing.T) {
	t.Run("previous exact release", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		fixture.record.EffectAttempts[0].ActionFence++
		fixture.attempt = fixture.record.EffectAttempts[0]
		fixture.claim.Fence = fixture.attempt.ActionFence + 1
		previous := fixture.attempt
		previous.Ref += ":released"
		previous.ActionFence--
		previous.WorkerRef = "worker:historical-released"
		fixture.record.EffectAttempts = append([]EffectAttempt{previous}, fixture.record.EffectAttempts...)
		fixture.record.BudgetSettlements = append(fixture.record.BudgetSettlements,
			exactZeroReleaseForRecovery(fixture.claim.BudgetReservation, previous))
		selected, err := SelectAgentLaunchRecoveryAttempt(fixture.record, fixture.claim)
		if err != nil || selected != fixture.attempt {
			t.Fatalf("selected=%+v err=%v released=%t reservation_fence=%d attempt_fence=%d",
				selected, err, effectAttemptDefinitelyUnapplied(fixture.record, previous),
				fixture.claim.BudgetReservation.Fence, previous.ActionFence)
		}
	})
	t.Run("malformed released attempt", func(t *testing.T) {
		fixture := newAgentLaunchRecoveryFixture(t)
		fixture.record.EffectAttempts[0].ActionFence++
		fixture.attempt = fixture.record.EffectAttempts[0]
		fixture.claim.Fence = fixture.attempt.ActionFence + 1
		previous := fixture.attempt
		previous.Ref += ":released"
		previous.ActionFence--
		previous.ClaimLeaseUntil = time.Time{}
		fixture.record.EffectAttempts = append([]EffectAttempt{previous}, fixture.record.EffectAttempts...)
		fixture.record.BudgetSettlements = append(fixture.record.BudgetSettlements,
			exactZeroReleaseForRecovery(fixture.claim.BudgetReservation, previous))
		if _, err := SelectAgentLaunchRecoveryAttempt(fixture.record, fixture.claim); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestAgentLaunchRecoveryPreflightIgnoresReleasedPeerApproval(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	fixture.record.EffectAttempts[0].ActionFence++
	fixture.attempt = fixture.record.EffectAttempts[0]
	fixture.claim.Fence = fixture.attempt.ActionFence + 1
	previous := fixture.attempt
	previous.Ref += ":released-revoked-approver"
	previous.ApprovalRef = "effect-approval:released-revoked-approver"
	previous.ActionFence--
	previous.WorkerRef = "worker:released-revoked-approver"
	fixture.record.EffectAttempts = append([]EffectAttempt{previous}, fixture.record.EffectAttempts...)
	fixture.record.BudgetSettlements = append(fixture.record.BudgetSettlements,
		exactZeroReleaseForRecovery(fixture.claim.BudgetReservation, previous))

	selected, err := PreflightAgentLaunchRecoveryAttempt(fixture.record, fixture.claim.Action)
	if err != nil || selected != fixture.attempt {
		t.Fatalf("selected=%+v err=%v ambiguous=%+v", selected, err, fixture.attempt)
	}
}

func TestAgentLaunchBlockingAttemptAdmissionUsesExactZeroRelease(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	if !AgentLaunchHasBlockingEffectAttempt(fixture.record, fixture.claim.Action) {
		t.Fatal("ambiguous launch attempt did not block normal admission")
	}
	fixture.record.BudgetSettlements = append(fixture.record.BudgetSettlements,
		exactZeroReleaseForRecovery(fixture.claim.BudgetReservation, fixture.attempt))
	if AgentLaunchHasBlockingEffectAttempt(fixture.record, fixture.claim.Action) {
		t.Fatal("exact causal zero-release kept blocking normal admission")
	}
	fixture.record.EffectAttempts[0].ClaimLeaseUntil = time.Time{}
	if !AgentLaunchHasBlockingEffectAttempt(fixture.record, fixture.claim.Action) {
		t.Fatal("malformed historical authority reopened normal admission")
	}
}

func TestAgentLaunchRecoveryRejectsLeaseFenceAndCrossBindings(t *testing.T) {
	tests := map[string]func(*agentLaunchRecoveryFixture){
		"lease missing": func(f *agentLaunchRecoveryFixture) { f.record.EffectAttempts[0].ClaimLeaseUntil = time.Time{} },
		"lease before start": func(f *agentLaunchRecoveryFixture) {
			f.record.EffectAttempts[0].ClaimLeaseUntil = f.record.EffectAttempts[0].StartedAt
		},
		"fence not superior": func(f *agentLaunchRecoveryFixture) { f.claim.Fence = f.attempt.ActionFence },
		"subject crossed": func(f *agentLaunchRecoveryFixture) {
			f.record.EffectAttempts[0].Subject.ExecutionRef = mustExecutionRefRecovery(t, "execution:crossed")
		},
		"session crossed": func(f *agentLaunchRecoveryFixture) {
			f.request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:crossed")
		},
		"placement crossed": func(f *agentLaunchRecoveryFixture) {
			f.request.ReferenciaColocacion, _ = ports.NewAgentPlacementRef("placement:crossed")
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newAgentLaunchRecoveryFixture(t)
			mutate(&fixture)
			if _, _, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestAgentLaunchRecoveryUsesApprovalValidityAtAttemptStart(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixture(t)
	intent := fixture.claim.Action.EffectIntent
	approver := testPrincipal(t, "principal:recovery-approver", "actor:recovery-approver", identity.PrincipalKindHuman)
	approval := fixture.claim.EffectApproval
	approval.RequestRef = "request:recovery-explicit-approval"
	approval.RequestFingerprint = strings.Repeat("e", 64)
	approval.ProposedBy = intent.ProposedBy
	approval.DecidedBy = approver.Ref
	approval.Source = EffectApprovalSourceExplicitDecision
	approval.Reason = "recover historical launch"
	approval.DecidedAt = fixture.attempt.StartedAt
	approval.ExpiresAt = approval.DecidedAt.Add(intent.ApprovalTTL)
	approval.AuthorizationReceipt = effectTestAuthorization(
		t, approver, intent.Subject.ProjectRef, identity.PermissionEffectsApprove,
		effectApprovalResourceRef(intent.Subject.GoalRef, intent.Ref, intent.Digest), approval.DecidedAt,
	)
	if err := ValidateEffectApproval(intent, approval); err != nil {
		t.Fatalf("explicit approval fixture invalid: %v", err)
	}
	fixture.claim.EffectApproval = approval
	fixture.record.EffectApprovals[0] = approval
	fixture.record.EffectAttempts[0].ApprovalRef = approval.Ref
	fixture.attempt = fixture.record.EffectAttempts[0]
	fixture.claim.LeaseUntil = approval.ExpiresAt.Add(time.Hour)
	if _, _, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request); err != nil {
		t.Fatalf("approval expired now rejected: %v", err)
	}
	fixture.record.EffectAttempts[0].StartedAt = approval.ExpiresAt
	fixture.record.EffectAttempts[0].ClaimLeaseUntil = approval.ExpiresAt.Add(time.Minute)
	fixture.claim.RecoveryEffectAttemptRef = fixture.record.EffectAttempts[0].Ref
	if _, _, err := BuildAgentLaunchRecoveryRequest(fixture.record, fixture.claim, fixture.request); !errors.Is(err, ErrAgentLaunchRecoveryInvalid) {
		t.Fatalf("approval invalid at StartedAt error=%v", err)
	}
}

func TestAgentLaunchRecoveryRequiresReconcilerCapability(t *testing.T) {
	launcher := &scriptedAgent{}
	if reconciler, err := AgentLaunchReconcilerFrom(launcher); reconciler != nil ||
		!errors.Is(err, ErrAgentLaunchRecoveryUnsupported) {
		t.Fatalf("reconciler=%T err=%v", reconciler, err)
	}
	capable := &recoveryCapableLauncher{scriptedAgent: launcher}
	if reconciler, err := AgentLaunchReconcilerFrom(capable); err != nil || reconciler != capable {
		t.Fatalf("reconciler=%T err=%v", reconciler, err)
	}
}

type recoveryCapableLauncher struct {
	scriptedAgent       *scriptedAgent
	reconcileRequests   []ports.AgentLaunchRequest
	reconcileAcceptedAt time.Time
	reconcileErr        error
	reconcileOverride   func(ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error)
	reconcileHook       func()
}

func (launcher *recoveryCapableLauncher) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	return launcher.scriptedAgent.Capabilities(ctx)
}
func (launcher *recoveryCapableLauncher) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	return launcher.scriptedAgent.Launch(ctx, request)
}
func (launcher *recoveryCapableLauncher) ReconcileLaunch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	launcher.reconcileRequests = append(launcher.reconcileRequests, request)
	if launcher.reconcileErr != nil {
		return ports.AgentLaunchReceipt{}, launcher.reconcileErr
	}
	if launcher.reconcileHook != nil {
		launcher.reconcileHook()
	}
	if launcher.reconcileOverride != nil {
		return launcher.reconcileOverride(request)
	}
	at := launcher.reconcileAcceptedAt
	if at.IsZero() {
		at = request.EffectAuthority.StartedAt
	}
	receipt := launchReceiptForRequest(request, at)
	receipt.RequierePreservacionEntorno = request.RequierePreservacionEntorno
	return receipt, nil
}

func recoveryEffectReceipt(attempt EffectAttempt) EffectReceipt {
	return EffectReceipt{
		Ref: "effect-receipt:" + attempt.Ref, IntentRef: attempt.IntentRef, IntentDigest: attempt.IntentDigest,
		ApprovalRef: attempt.ApprovalRef, AttemptRef: attempt.Ref, Subject: attempt.Subject,
		ActionRef: attempt.ActionRef, ActionFence: attempt.ActionFence, IdempotencyKey: attempt.IdempotencyKey,
		ExternalRef: "receipt:external:" + attempt.Ref, Status: EffectStatusAccepted,
		Usage: unknownUsage(), ConfirmedAt: attempt.StartedAt,
	}
}

func exactZeroReleaseForRecovery(
	reservation governance.BudgetReservation,
	attempt EffectAttempt,
) governance.BudgetSettlement {
	zero := governance.ResourceVector{Currency: reservation.Resources.Currency}
	return governance.BudgetSettlement{
		ReservationRef: reservation.Ref, CausalAttemptRef: attempt.Ref, Reserved: reservation.Resources,
		Observed: governance.ResourceUsage{
			Resources: zero, Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
		},
		Charged: zero, Released: reservation.Resources, Overrun: zero, SettledAt: attempt.StartedAt,
	}
}

func mustExecutionRefRecovery(t *testing.T, value string) goal.ExecutionRef {
	t.Helper()
	parsed, err := goal.NewExecutionRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
