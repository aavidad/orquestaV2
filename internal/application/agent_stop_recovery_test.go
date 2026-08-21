package application

import (
	"context"
	"errors"
	"orquesta/internal/ports"
	"testing"
	"time"
)

func TestProcessStopReconcilesOnlyOneExactAmbiguousAttemptWithoutResend(t *testing.T) {
	tests := []struct {
		name                                    string
		mutate                                  func(*GoalRecord)
		crossReceipt, definite, expire, stopped bool
		wantReconcile                           int
	}{
		{name: "lost acknowledgement", stopped: true, wantReconcile: 1},
		{name: "crossed physical identity", mutate: func(record *GoalRecord) {
			execution := &record.Executions[0]
			execution.ExternalRef, execution.ProviderRef, execution.ModelRef, execution.AgentRef = "external:crossed", "provider:crossed", "model:crossed", "agent:crossed"
		}},
		{name: "action-only peer", mutate: func(record *GoalRecord) { addStopPeer(record, "action") }},
		{name: "intent-only peer", mutate: func(record *GoalRecord) { addStopPeer(record, "intent") }},
		{name: "second attempt", mutate: func(record *GoalRecord) { addStopPeer(record, "both") }},
		{name: "crossed attempt key", mutate: func(record *GoalRecord) {
			record.EffectAttempts[len(record.EffectAttempts)-1].IdempotencyKey += ":crossed"
		}},
		{name: "crossed attempt fence", mutate: func(record *GoalRecord) { record.EffectAttempts[len(record.EffectAttempts)-1].ActionFence++ }},
		{name: "crossed attempt lease", mutate: func(record *GoalRecord) {
			record.EffectAttempts[len(record.EffectAttempts)-1].ClaimLeaseUntil = record.EffectAttempts[len(record.EffectAttempts)-1].StartedAt
		}},
		{name: "related receipt", mutate: func(record *GoalRecord) {
			record.EffectReceipts = append(record.EffectReceipts, EffectReceipt{AttemptRef: record.EffectAttempts[len(record.EffectAttempts)-1].Ref})
		}},
		{name: "crossed reconciled receipt", crossReceipt: true, wantReconcile: 1},
		{name: "definitely unapplied", definite: true},
		{name: "expired claim lease", expire: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := newControlTestSystem(t, nil)
			system.launch(t)
			running := system.record(t)
			item := running.Goal.WorkItems()[0]
			execution := mustBoundExecution(t, running, item.Ref())
			probe := &stopRecoveryProbe{now: system.clock.Now, crossReceipt: test.crossReceipt, definite: test.definite}
			probe.afterStop = func() {
				if test.expire {
					system.clock.Advance(2 * time.Minute)
				}
				if test.mutate != nil {
					system.repository.mu.Lock()
					defer system.repository.mu.Unlock()
					record := system.repository.records[system.goalRef]
					test.mutate(&record)
					system.repository.records[system.goalRef] = record
				}
			}
			system.orchestrator.controller = probe
			created, err := system.orchestrator.Control(context.Background(), system.access, system.request(t, "control:stop-recovery:"+test.name, ControlStop, ControlTargetExecution, item.Ref(), execution.Ref))
			if err != nil {
				t.Fatal(err)
			}
			processed, processErr := system.orchestrator.ProcessNext(context.Background(), "worker:stop-recovery")
			record := system.record(t)
			current, _ := executionByRef(record.Executions, execution.Ref)
			if !processed.Processed || processed.Action != ActionStopAgent || (processErr == nil) != (test.stopped || test.definite) ||
				probe.stops != 1 || probe.reconciles != test.wantReconcile ||
				(current.State == ExecutionStopped) != test.stopped {
				t.Fatalf("processed=%+v err=%v stop/reconcile=%d/%d state=%s", processed, processErr, probe.stops, probe.reconciles, current.State)
			}
			if test.definite {
				outcome := record.EffectAttemptOutcomes[len(record.EffectAttemptOutcomes)-1]
				if len(record.EffectAttempts) != len(running.EffectAttempts)+1 ||
					len(record.EffectReceipts) != len(running.EffectReceipts) ||
					len(record.ConsumptionReceipts) != len(running.ConsumptionReceipts) ||
					outcome.ActionRef != "action:stop:"+created.Control.Ref+":"+execution.Ref.String() ||
					outcome.Outcome != EffectAttemptDefinitelyNotApplied {
					t.Fatalf("durable stop outcome attempts=%d receipts=%d outcome=%+v", len(record.EffectAttempts), len(record.EffectReceipts), outcome)
				}
			}
		})
	}
}

func addStopPeer(record *GoalRecord, relation string) {
	peer := record.EffectAttempts[len(record.EffectAttempts)-1]
	peer.Ref += ":peer"
	if relation == "action" {
		peer.IntentRef += ":crossed"
	}
	if relation == "intent" {
		peer.ActionRef += ":crossed"
	}
	record.EffectAttempts = append(record.EffectAttempts, peer)
}

type stopRecoveryProbe struct {
	now                    func() time.Time
	afterStop              func()
	crossReceipt, definite bool
	stops, reconciles      int
}

func (*stopRecoveryProbe) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}
func (probe *stopRecoveryProbe) Stop(context.Context, ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	probe.stops++
	probe.afterStop()
	if probe.definite {
		return ports.AgentStopReceipt{}, definiteStopFailure{}
	}
	return ports.AgentStopReceipt{}, errors.New("ack lost")
}
func (probe *stopRecoveryProbe) ReconcileStop(ctx context.Context, request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	probe.reconciles++
	receipt, _ := (unsupportedAgentController{}).Stop(ctx, request)
	receipt.Status, receipt.ReceiptRef, receipt.ConfirmedAt = ports.AgentStopped, "receipt:stop-reconciled:"+request.ExecutionRef.String(), probe.now().UTC()
	if probe.crossReceipt {
		receipt.IdempotencyKey += ":crossed"
	}
	return receipt, nil
}

type definiteStopFailure struct{}

func (definiteStopFailure) Error() string              { return "rejected before stop" }
func (definiteStopFailure) DefinitelyNotApplied() bool { return true }
