package orquestadirectorcycle

import (
	"context"
	"testing"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
)

const cycleStepRunRefV0 = "run-cycle-step-001"

type cycleStepWorkflowPortV0 struct {
	run orquestacoreworkflow.OrchestrationRunV0
}

func newCycleStepWorkflowV0(t *testing.T) *cycleStepWorkflowPortV0 {
	t.Helper()
	port := &cycleStepWorkflowPortV0{}
	port.mustApplyCommandV0(t, cycleStepStartCommandV0(t))
	port.mustApplyCommandV0(t, cycleStepOpenProgramacionCommandV0(t))
	return port
}

func (port *cycleStepWorkflowPortV0) HandleWorkflowCommandV0(
	ctx context.Context,
	command orquestacoreworkflow.OrchestrationCommandV0,
) (orquestacoreworkflow.OrchestrationCommandResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacoreworkflow.OrchestrationCommandResultV0{}, err
	}
	result, err := orquestacoreworkflow.HandleCommandV0(port.run, command)
	if err != nil {
		return result, err
	}
	for _, event := range result.Events {
		port.run, err = orquestacoreworkflow.ApplyEventV0(port.run, event)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (port *cycleStepWorkflowPortV0) mustApplyCommandV0(
	t *testing.T,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	if _, err := port.HandleWorkflowCommandV0(context.Background(), command); err != nil {
		t.Fatalf("apply command %s: %v", command.CommandType, err)
	}
}

type cycleStepLedgerAdapterV0 struct {
	ledger *orquestapersistence.InMemoryOutboxLedgerV0
}

func newCycleStepLedgerAdapterV0() *cycleStepLedgerAdapterV0 {
	return &cycleStepLedgerAdapterV0{ledger: orquestapersistence.NewInMemoryOutboxLedgerV0()}
}

func (adapter *cycleStepLedgerAdapterV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	saved, issues := adapter.ledger.SavePending(ctx, messages)
	return saved, cycleStepLedgerIssuesV0(issues)
}

func (adapter *cycleStepLedgerAdapterV0) ListPending(
	ctx context.Context,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	pending, issues := adapter.ledger.ListPending(ctx, orquestapersistence.OutboxPendingFilterV0{
		RunID:      filter.RunRef,
		TargetPort: filter.TargetPort,
	})
	return pending, cycleStepLedgerIssuesV0(issues)
}

func validCycleStepInputV0(
	workflow *cycleStepWorkflowPortV0,
	ledger *cycleStepLedgerAdapterV0,
	cycleRef string,
	tickRef string,
) DirectorCycleStepInputV0 {
	return DirectorCycleStepInputV0{
		Scheduler:      orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:       workflow,
		OutboxLedger:   ledger,
		CycleRef:       cycleRef,
		TickRef:        tickRef,
		RunRef:         workflow.run.RunID,
		OccurredAt:     "2026-05-06T13:00:00Z",
		Run:            workflow.run,
		WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{cycleStepWorkCandidateV0(workflow.run.RunID)},
		EvidenceRefs:   []string{"evidence-ref-cycle-step-001"},
	}
}

func cycleStepWorkCandidateV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	claimRef := "claim-ref-cycle-step-001"
	return orquestadirectorscheduler.SchedulableWorkCandidateV0{
		CandidateRef:      "candidate-ref-cycle-step-001",
		SubjectClaimRefs:  []string{claimRef},
		Claims:            []orquestacoreconcurrency.WorksetClaimV0{cycleStepClaimV0(runRef, claimRef)},
		CapacityCandidate: cycleStepCapacityCandidateV0(),
		GateCommandMeta:   cycleStepCommandMetaV0("cmd-cycle-step-gate", "gate"),
		EvidenceRefs:      []string{"evidence-ref-cycle-step-candidate-001"},
	}
}

func cycleStepClaimV0(runRef string, claimRef string) orquestacoreconcurrency.WorksetClaimV0 {
	return orquestacoreconcurrency.WorksetClaimV0{
		SchemaVersion:  orquestacoreconcurrency.WorksetClaimSchemaVersionV0,
		ClaimRef:       claimRef,
		RunRef:         runRef,
		TaskRef:        "task-ref-cycle-step-001",
		AgentRequestID: "agent-ref-cycle-step-001",
		WriteSet:       []orquestacoreconcurrency.ScopeRefV0{{Ref: "modulos/app/main.go"}},
		EvidenceRefs:   []string{"evidence-ref-cycle-step-claim-001"},
	}
}
