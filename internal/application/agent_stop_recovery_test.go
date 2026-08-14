package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type agentStopRecoveryFixture struct {
	record    GoalRecord
	claim     ActionClaim
	attempt   EffectAttempt
	control   ControlRecord
	execution ExecutionRecord
	system    *controlTestSystem
}

func newAgentStopRecoveryFixture(t *testing.T) agentStopRecoveryFixture {
	t.Helper()
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	if _, err := system.orchestrator.Control(
		context.Background(), system.access,
		system.request(t, "control:stop-recovery", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref),
	); err != nil {
		t.Fatal(err)
	}
	claim := system.claim(t, "claim:stop-before-restart")
	if claim.Action.Kind != ActionStopAgent {
		t.Fatalf("claimed action=%s want=%s", claim.Action.Kind, ActionStopAgent)
	}
	attempt, created, err := system.orchestrator.beginEffectAttempt(context.Background(), claim, system.clock.Now())
	if err != nil || !created {
		t.Fatalf("record Stop attempt: created=%t err=%v", created, err)
	}
	record = system.record(t)
	execution, found := executionForAction(record, claim.Action)
	control, controlFound := controlByRef(record.Controls, claim.Action.ControlRef)
	if !found || !controlFound {
		t.Fatal("Stop recovery causal records missing")
	}
	claim.Token = "claim:stop-after-restart"
	claim.WorkerRef = "worker:stop-after-restart"
	claim.DeliveryAttempt++
	claim.Fence++
	claim.Disposition = ActionClaimDispositionRecoverEffect
	claim.RecoveryEffectAttemptRef = attempt.Ref
	claim.LeaseUntil = attempt.ClaimLeaseUntil.Add(time.Minute)
	system.repository.mu.Lock()
	action := system.repository.actions[claim.Action.Ref]
	action.token, action.workerRef = claim.Token, claim.WorkerRef
	action.deliveryAttempt, action.fence, action.lease = claim.DeliveryAttempt, claim.Fence, claim.LeaseUntil
	system.repository.actions[claim.Action.Ref] = action
	system.repository.mu.Unlock()
	return agentStopRecoveryFixture{
		record: record, claim: claim, attempt: attempt, control: control, execution: execution, system: system,
	}
}

func TestBuildAgentStopRecoveryRequestUsesExactDurableAuthority(t *testing.T) {
	fixture := newAgentStopRecoveryFixture(t)
	request, attempt, err := BuildAgentStopRecoveryRequest(fixture.record, fixture.claim)
	if err != nil {
		t.Fatal(err)
	}
	want, err := agentStopRequest(fixture.record, fixture.control, fixture.execution)
	if err != nil {
		t.Fatal(err)
	}
	want.StopEffectAttemptRef, want.StopActionFence = fixture.attempt.Ref, fixture.attempt.ActionFence
	if attempt != fixture.attempt || !reflect.DeepEqual(request, want) ||
		request.StopActionFence >= fixture.claim.Fence || ports.ValidateAgentStopRequest(request) != nil {
		t.Fatalf("request=%+v want=%+v attempt=%+v", request, want, attempt)
	}
	selected, err := PreflightAgentStopRecoveryAttempt(fixture.record, fixture.claim.Action)
	if err != nil || selected != fixture.attempt {
		t.Fatalf("preflight attempt=%+v want=%+v err=%v", selected, fixture.attempt, err)
	}
}

func TestProcessAgentStopRecoveryRequeuesOnlyReadOnlyPendingOutcomes(t *testing.T) {
	for _, test := range []struct {
		name   string
		status ports.AgentStopStatus
		err    error
	}{
		{"pending", ports.AgentStopPending, nil},
		{"unsupported", ports.AgentStopUnsupported, nil},
		{"temporary failure", "", temporaryAgentTestError{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentStopRecoveryFixture(t)
			controller := &agentStopRecoveryProcessorController{status: test.status, err: test.err,
				confirmedAt: fixture.system.clock.Now()}
			fixture.system.orchestrator.controller = controller
			wantRequest, _, err := BuildAgentStopRecoveryRequest(fixture.record, fixture.claim)
			if err != nil {
				t.Fatal(err)
			}
			result, err := fixture.system.orchestrator.ProcessClaim(context.Background(), fixture.claim)
			if err != nil || !result.Processed || result.Action != ActionStopAgent ||
				len(controller.reconciles) != 1 || !reflect.DeepEqual(controller.reconciles[0], wantRequest) ||
				controller.stops != 0 || controller.capabilityCalls != 0 {
				t.Fatalf("result=%+v err=%v reconciles=%+v stops=%d capabilities=%d",
					result, err, controller.reconciles, controller.stops, controller.capabilityCalls)
			}
			fixture.system.repository.mu.Lock()
			action := fixture.system.repository.actions[fixture.claim.Action.Ref]
			record := fixture.system.repository.records[fixture.claim.Action.GoalRef]
			fixture.system.repository.mu.Unlock()
			if action.token != "" || action.workerRef != "" || !action.lease.IsZero() ||
				len(record.EffectAttempts) != len(fixture.record.EffectAttempts) ||
				len(record.EffectReceipts) != 1 || len(record.ConsumptionReceipts) != 1 {
				t.Fatalf("released action=%+v attempts=%+v receipts=%+v consumptions=%+v",
					action, record.EffectAttempts, record.EffectReceipts, record.ConsumptionReceipts)
			}
			if err := fixture.system.repository.ValidateAgentStopRecoveryClaim(
				context.Background(), fixture.claim,
			); !IsStateError(err, StateConflict) {
				t.Fatalf("released claim remained current: %v", err)
			}
		})
	}
}

func TestProcessAgentStopRecoveryLeavesTerminalAndInvalidResultsUnconsumed(t *testing.T) {
	for _, test := range []struct {
		name   string
		status ports.AgentStopStatus
		err    error
		mutate func(*ports.AgentStopReceipt)
	}{
		{"stopped", ports.AgentStopped, nil, nil},
		{"already stopped", ports.AgentStopAlreadyStopped, nil, nil},
		{"already completed", ports.AgentStopAlreadyCompleted, nil, nil},
		{"already failed", ports.AgentStopAlreadyFailed, nil, nil},
		{"permanent failure", "", errors.New("provider response invalid"), nil},
		{"crossed receipt", ports.AgentStopPending, nil, func(receipt *ports.AgentStopReceipt) {
			receipt.StopActionFence++
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAgentStopRecoveryFixture(t)
			controller := &agentStopRecoveryProcessorController{status: test.status, err: test.err, mutate: test.mutate,
				confirmedAt: fixture.system.clock.Now()}
			fixture.system.orchestrator.controller = controller
			before := fixture.system.effects(t)
			result, err := fixture.system.orchestrator.ProcessClaim(context.Background(), fixture.claim)
			if !result.Processed || result.Action != ActionStopAgent || !IsStateError(err, StateConflict) ||
				len(controller.reconciles) != 1 || controller.stops != 0 || controller.capabilityCalls != 0 {
				t.Fatalf("result=%+v err=%v reconciles=%d stops=%d capabilities=%d",
					result, err, len(controller.reconciles), controller.stops, controller.capabilityCalls)
			}
			fixture.system.assertEffects(t, before)
		})
	}
}

func TestProcessAgentStopRecoveryFencesClaimProviderAndCancellationBeforeMutation(t *testing.T) {
	t.Run("pre-canceled", func(t *testing.T) {
		fixture := newAgentStopRecoveryFixture(t)
		controller := &agentStopRecoveryProcessorController{status: ports.AgentStopPending}
		fixture.system.orchestrator.controller = controller
		before := fixture.system.effects(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := fixture.system.orchestrator.ProcessClaim(ctx, fixture.claim)
		if !errors.Is(err, context.Canceled) || len(controller.reconciles) != 0 || controller.stops != 0 {
			t.Fatalf("pre-cancel err=%v reconciles=%d stops=%d", err, len(controller.reconciles), controller.stops)
		}
		fixture.system.assertEffects(t, before)
	})
	t.Run("crossed durable claim", func(t *testing.T) {
		fixture := newAgentStopRecoveryFixture(t)
		controller := &agentStopRecoveryProcessorController{status: ports.AgentStopPending}
		fixture.system.orchestrator.controller = controller
		before := fixture.system.effects(t)
		crossed := fixture.claim
		crossed.Fence++
		_, err := fixture.system.orchestrator.ProcessClaim(context.Background(), crossed)
		if !IsStateError(err, StateConflict) || len(controller.reconciles) != 0 || controller.stops != 0 {
			t.Fatalf("crossed err=%v reconciles=%d stops=%d", err, len(controller.reconciles), controller.stops)
		}
		fixture.system.assertEffects(t, before)
	})
	t.Run("crossed provider composition", func(t *testing.T) {
		fixture := newAgentStopRecoveryFixture(t)
		controller := &agentStopRecoveryProcessorController{status: ports.AgentStopPending}
		fixture.system.orchestrator.controller = controller
		fixture.system.orchestrator.agentCapabilities.ProviderRef = "provider:crossed"
		before := fixture.system.effects(t)
		_, err := fixture.system.orchestrator.ProcessClaim(context.Background(), fixture.claim)
		if !IsStateError(err, StateConflict) || len(controller.reconciles) != 0 || controller.stops != 0 {
			t.Fatalf("provider err=%v reconciles=%d stops=%d", err, len(controller.reconciles), controller.stops)
		}
		fixture.system.assertEffects(t, before)
	})
	t.Run("claim expires during preparation", func(t *testing.T) {
		fixture := newAgentStopRecoveryFixture(t)
		controller := &agentStopRecoveryProcessorController{status: ports.AgentStopPending}
		fixture.system.orchestrator.controller = controller
		fixture.system.orchestrator.state = &agentStopRecoveryGetGoalHookState{
			StateRepository: fixture.system.repository,
			hook: func() {
				fixture.system.clock.Advance(fixture.claim.LeaseUntil.Sub(fixture.system.clock.Now()))
			},
		}
		before := fixture.system.effects(t)
		_, err := fixture.system.orchestrator.ProcessClaim(context.Background(), fixture.claim)
		if !IsStateError(err, StateConflict) || len(controller.reconciles) != 0 || controller.stops != 0 {
			t.Fatalf("expired err=%v reconciles=%d stops=%d", err, len(controller.reconciles), controller.stops)
		}
		fixture.system.assertEffects(t, before)
	})
	t.Run("authority revoked after claim", func(t *testing.T) {
		fixture := newAgentStopRecoveryFixture(t)
		controller := &agentStopRecoveryProcessorController{status: ports.AgentStopPending}
		fixture.system.orchestrator.controller = controller
		intent := fixture.claim.Action.EffectIntent
		fixture.system.accessStore.setRole(intent.ProposedBy, intent.Subject.ProjectRef, "")
		result, err := fixture.system.orchestrator.ProcessClaim(context.Background(), fixture.claim)
		if err != nil || !result.Processed || len(controller.reconciles) != 0 || controller.stops != 0 {
			t.Fatalf("revoked result=%+v err=%v reconciles=%d stops=%d",
				result, err, len(controller.reconciles), controller.stops)
		}
		fixture.system.repository.mu.Lock()
		action := fixture.system.repository.actions[fixture.claim.Action.Ref]
		fixture.system.repository.mu.Unlock()
		if action.token != "" || action.workerRef != "" || !action.lease.IsZero() {
			t.Fatalf("revoked claim was not requeued: %+v", action)
		}
	})
	t.Run("cancel during reconcile", func(t *testing.T) {
		fixture := newAgentStopRecoveryFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		controller := &agentStopRecoveryProcessorController{status: ports.AgentStopPending, hook: cancel}
		fixture.system.orchestrator.controller = controller
		before := fixture.system.effects(t)
		_, err := fixture.system.orchestrator.ProcessClaim(ctx, fixture.claim)
		if !errors.Is(err, context.Canceled) || len(controller.reconciles) != 1 || controller.stops != 0 {
			t.Fatalf("cancel err=%v reconciles=%d stops=%d", err, len(controller.reconciles), controller.stops)
		}
		fixture.system.assertEffects(t, before)
	})
}

type agentStopRecoveryGetGoalHookState struct {
	StateRepository
	hook func()
}

func (state *agentStopRecoveryGetGoalHookState) GetGoal(
	ctx context.Context,
	ref goal.GoalRef,
) (GoalRecord, error) {
	record, err := state.StateRepository.GetGoal(ctx, ref)
	hook := state.hook
	state.hook = nil
	if hook != nil {
		hook()
	}
	return record, err
}

type agentStopRecoveryProcessorController struct {
	status          ports.AgentStopStatus
	err             error
	confirmedAt     time.Time
	hook            func()
	mutate          func(*ports.AgentStopReceipt)
	capabilityCalls int
	stops           int
	reconciles      []ports.AgentStopRequest
}

func (controller *agentStopRecoveryProcessorController) ControlCapabilities(
	context.Context,
) (ports.AgentControlCapabilities, error) {
	controller.capabilityCalls++
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (controller *agentStopRecoveryProcessorController) Stop(
	context.Context,
	ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	controller.stops++
	return ports.AgentStopReceipt{}, errors.New("test.second_stop_forbidden")
}

func (controller *agentStopRecoveryProcessorController) ReconcileStop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	controller.reconciles = append(controller.reconciles, request)
	if controller.hook != nil {
		controller.hook()
	}
	if controller.err != nil {
		return ports.AgentStopReceipt{}, controller.err
	}
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, StopEffectAttemptRef: request.StopEffectAttemptRef,
		StopActionFence: request.StopActionFence, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: controller.status,
	}
	if controller.status == ports.AgentStopped || controller.status == ports.AgentStopAlreadyStopped ||
		controller.status == ports.AgentStopAlreadyCompleted || controller.status == ports.AgentStopAlreadyFailed {
		receipt.ReceiptRef, receipt.ConfirmedAt = "receipt:stop-recovery:"+request.ExecutionRef.String(), controller.confirmedAt
	}
	if controller.mutate != nil {
		controller.mutate(&receipt)
	}
	return receipt, nil
}

func TestPreflightAgentStopRecoveryRejectsSettledCrossedAndNonUniqueHistory(t *testing.T) {
	fixture := newAgentStopRecoveryFixture(t)
	for name, mutate := range map[string]func(*GoalRecord){
		"attempt absent": func(candidate *GoalRecord) {
			candidate.EffectAttempts = nil
		},
		"receipt related by action": func(candidate *GoalRecord) {
			candidate.EffectReceipts = append(candidate.EffectReceipts, EffectReceipt{ActionRef: fixture.claim.Action.Ref})
		},
		"receipt related by intent": func(candidate *GoalRecord) {
			candidate.EffectReceipts = append(candidate.EffectReceipts, EffectReceipt{IntentRef: fixture.claim.Action.EffectIntentRef})
		},
		"receipt related by attempt": func(candidate *GoalRecord) {
			candidate.EffectReceipts = append(candidate.EffectReceipts, EffectReceipt{AttemptRef: fixture.attempt.Ref})
		},
		"second attempt": func(candidate *GoalRecord) {
			peer := fixture.attempt
			peer.Ref, peer.WorkerRef, peer.ActionFence = fixture.attempt.Ref+":peer", "worker:stop-peer", fixture.attempt.ActionFence+1
			candidate.EffectAttempts = append(candidate.EffectAttempts, peer)
		},
		"duplicate attempt": func(candidate *GoalRecord) {
			candidate.EffectAttempts = append(candidate.EffectAttempts, fixture.attempt)
		},
		"crossed action for same intent": func(candidate *GoalRecord) {
			peer := fixture.attempt
			peer.Ref, peer.ActionRef = fixture.attempt.Ref+":crossed", "action:stop:crossed"
			candidate.EffectAttempts = append(candidate.EffectAttempts, peer)
		},
		"crossed intent for same action": func(candidate *GoalRecord) {
			peer := fixture.attempt
			peer.Ref, peer.IntentRef = fixture.attempt.Ref+":crossed", "effect-intent:crossed"
			candidate.EffectAttempts = append(candidate.EffectAttempts, peer)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := fixture.record
			candidate.EffectAttempts = append([]EffectAttempt(nil), fixture.record.EffectAttempts...)
			candidate.EffectReceipts = append([]EffectReceipt(nil), fixture.record.EffectReceipts...)
			mutate(&candidate)
			if _, err := PreflightAgentStopRecoveryAttempt(candidate, fixture.claim.Action); !errors.Is(err, ErrAgentStopRecoveryInvalid) {
				t.Fatalf("invalid history accepted: %+v err=%v", candidate.EffectAttempts, err)
			}
		})
	}
}

func TestPreflightAgentStopRecoveryRejectsEveryHistoricalAttemptMismatch(t *testing.T) {
	fixture := newAgentStopRecoveryFixture(t)
	for name, mutate := range map[string]func(*EffectAttempt){
		"ref":              func(candidate *EffectAttempt) { candidate.Ref = "" },
		"intent ref":       func(candidate *EffectAttempt) { candidate.IntentRef = "effect-intent:crossed" },
		"intent digest":    func(candidate *EffectAttempt) { candidate.IntentDigest = "sha256:crossed" },
		"subject":          func(candidate *EffectAttempt) { candidate.Subject = EffectSubject{} },
		"action ref":       func(candidate *EffectAttempt) { candidate.ActionRef = "action:stop:crossed" },
		"fence":            func(candidate *EffectAttempt) { candidate.ActionFence = 0 },
		"worker":           func(candidate *EffectAttempt) { candidate.WorkerRef = "" },
		"idempotency":      func(candidate *EffectAttempt) { candidate.IdempotencyKey = "stop:crossed" },
		"started":          func(candidate *EffectAttempt) { candidate.StartedAt = time.Time{} },
		"lease absent":     func(candidate *EffectAttempt) { candidate.ClaimLeaseUntil = time.Time{} },
		"lease not future": func(candidate *EffectAttempt) { candidate.ClaimLeaseUntil = candidate.StartedAt },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := fixture.record
			candidate.EffectAttempts = append([]EffectAttempt(nil), fixture.record.EffectAttempts...)
			mutate(&candidate.EffectAttempts[len(candidate.EffectAttempts)-1])
			if _, err := PreflightAgentStopRecoveryAttempt(candidate, fixture.claim.Action); !errors.Is(err, ErrAgentStopRecoveryInvalid) {
				t.Fatalf("mismatch %s accepted: %+v err=%v", name, candidate.EffectAttempts, err)
			}
		})
	}
}

func TestSelectAgentStopRecoveryRejectsEveryClaimAuthorityMismatch(t *testing.T) {
	fixture := newAgentStopRecoveryFixture(t)
	resourceFixture := newAgentLaunchRecoveryFixture(t)
	budgetBinding := resourceFixture.claim.BudgetReservation
	capacityBinding := resourceFixture.claim.CapacityReservation
	if governance.ValidateBudgetReservation(budgetBinding) != nil ||
		ValidateAgentCapacityReservation(capacityBinding) != nil {
		t.Fatal("launch fixture did not provide valid resource bindings")
	}
	crossedPlacement, err := ports.NewAgentPlacementRef("placement:crossed")
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*ActionClaim){
		"normal disposition": func(candidate *ActionClaim) { candidate.Disposition = ActionClaimDispositionNormal },
		"attempt ref absent": func(candidate *ActionClaim) { candidate.RecoveryEffectAttemptRef = "" },
		"attempt ref crossed": func(candidate *ActionClaim) {
			candidate.RecoveryEffectAttemptRef = "effect-attempt:crossed"
		},
		"fence not superior": func(candidate *ActionClaim) { candidate.Fence = fixture.attempt.ActionFence },
		"token absent":       func(candidate *ActionClaim) { candidate.Token = "" },
		"worker absent":      func(candidate *ActionClaim) { candidate.WorkerRef = "" },
		"retry projection": func(candidate *ActionClaim) {
			candidate.RetryBudgetExhaustion.FrontierDigest = "sha256:crossed"
		},
		"budget binding": func(candidate *ActionClaim) { candidate.BudgetReservationRef = "budget-reservation:crossed" },
		"budget reservation": func(candidate *ActionClaim) {
			candidate.BudgetReservation = budgetBinding
		},
		"capacity reservation": func(candidate *ActionClaim) {
			candidate.CapacityReservation = capacityBinding
		},
		"placement binding": func(candidate *ActionClaim) {
			candidate.ReferenciaColocacion = crossedPlacement
		},
		"approval crossed": func(candidate *ActionClaim) { candidate.EffectApproval.Ref = "effect-approval:crossed" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := fixture.claim
			mutate(&candidate)
			if _, err := SelectAgentStopRecoveryAttempt(fixture.record, candidate); !errors.Is(err, ErrAgentStopRecoveryInvalid) {
				t.Fatalf("mismatched claim accepted: %+v err=%v", candidate, err)
			}
		})
	}
}

func TestBuildAgentStopRecoveryRequestFailsClosedBeforeProviderForCrossedSnapshot(t *testing.T) {
	fixture := newAgentStopRecoveryFixture(t)
	for name, mutate := range map[string]func(*GoalRecord){
		"external ref absent": func(candidate *GoalRecord) {
			candidate.Executions[0].ExternalRef = ""
		},
		"execution queued": func(candidate *GoalRecord) {
			candidate.Executions[0].State = ExecutionQueued
		},
		"execution dispatching": func(candidate *GoalRecord) {
			candidate.Executions[0].State = ExecutionDispatching
		},
		"execution awaiting": func(candidate *GoalRecord) {
			candidate.Executions[0].State = ExecutionAwaitingCommit
		},
		"execution terminal": func(candidate *GoalRecord) {
			candidate.Executions[0].State = ExecutionStopped
		},
		"control settled": func(candidate *GoalRecord) {
			candidate.Controls[0].Status = ControlConfirmed
		},
		"control mode crossed": func(candidate *GoalRecord) {
			candidate.Controls[0].Mode = ports.AgentStopForced
		},
		"launch receipt absent": func(candidate *GoalRecord) {
			candidate.EffectReceipts = nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := fixture.record
			candidate.Executions = append([]ExecutionRecord(nil), fixture.record.Executions...)
			candidate.Controls = append([]ControlRecord(nil), fixture.record.Controls...)
			candidate.EffectReceipts = append([]EffectReceipt(nil), fixture.record.EffectReceipts...)
			mutate(&candidate)
			if _, _, err := BuildAgentStopRecoveryRequest(candidate, fixture.claim); !errors.Is(err, ErrAgentStopRecoveryInvalid) {
				t.Fatalf("crossed snapshot accepted: err=%v", err)
			}
		})
	}
}

func TestSelectAgentStopRecoveryRequiresOneExactHistoricalApproval(t *testing.T) {
	fixture := newAgentStopRecoveryFixture(t)
	for name, mutate := range map[string]func(*GoalRecord){
		"approval absent": func(candidate *GoalRecord) {
			candidate.EffectApprovals = nil
		},
		"approval duplicate": func(candidate *GoalRecord) {
			candidate.EffectApprovals = append(candidate.EffectApprovals, fixture.claim.EffectApproval)
		},
		"attempt approval crossed": func(candidate *GoalRecord) {
			candidate.EffectAttempts[len(candidate.EffectAttempts)-1].ApprovalRef = "effect-approval:crossed"
		},
		"attempt predates approval": func(candidate *GoalRecord) {
			candidate.EffectAttempts[len(candidate.EffectAttempts)-1].StartedAt =
				fixture.claim.EffectApproval.DecidedAt.Add(-time.Nanosecond)
		},
		"attempt predates intent": func(candidate *GoalRecord) {
			candidate.EffectAttempts[len(candidate.EffectAttempts)-1].StartedAt =
				fixture.claim.Action.EffectIntent.CreatedAt.Add(-time.Nanosecond)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := fixture.record
			candidate.EffectApprovals = append([]EffectApproval(nil), fixture.record.EffectApprovals...)
			candidate.EffectAttempts = append([]EffectAttempt(nil), fixture.record.EffectAttempts...)
			mutate(&candidate)
			if _, err := SelectAgentStopRecoveryAttempt(candidate, fixture.claim); !errors.Is(err, ErrAgentStopRecoveryInvalid) {
				t.Fatalf("historical approval mismatch accepted: err=%v", err)
			}
		})
	}
}
