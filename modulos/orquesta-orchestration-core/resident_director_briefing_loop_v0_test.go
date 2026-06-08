package orquestacionnucleoapp

import (
	"context"
	"strconv"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestRunResidentDirectorBriefingLoopV0EjecutaStepDespachaOutboxYCierra(t *testing.T) {
	runRef := "run-resident-briefing-loop-001"
	ledger := NewInMemoryOutboxLedgerV0()
	source := &sequenceResidentBriefingSourceV0{actions: []orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{
		briefingActionV0(runRef, "action-step", orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0),
		briefingActionV0(runRef, "action-dispatch", orquestadirectorsupervisor.DirectorSupervisorActionKindDispatchOutboxV0),
		briefingActionV0(runRef, "action-close", orquestadirectorsupervisor.DirectorSupervisorActionKindCloseOrIdleV0),
	}}
	service := ServiceV0{
		RunStore: newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateV0(runRef),
			},
		}},
		OutboxLedger: ledger,
	}

	result, err := service.RunResidentDirectorBriefingLoopV0(context.Background(), ResidentDirectorBriefingLoopRequestV0{
		RunRef:         runRef,
		OccurredAt:     "2026-06-08T10:00:00Z",
		BriefingSource: source,
		MaxActions:     4,
		Dispatchers: []OutboxDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
			Reader:     ledger,
			Claimer:    ledger,
			Executor:   fakeDispatchExecutorV0{},
			Acker:      ledger,
		}},
		EvidenceRefs: []string{"evidence-request"},
	})
	if err != nil {
		t.Fatalf("resident briefing loop: %v", err)
	}
	if result.Status != ResidentDirectorBriefingLoopStatusCompletedV0 ||
		result.ExecutedActions != 3 ||
		len(result.Steps) != 3 ||
		source.calls != 3 {
		t.Fatalf("result=%+v source_calls=%d", result, source.calls)
	}
	if result.Steps[0].Execution == nil ||
		result.Steps[0].Execution.Status != DirectorBriefingExecutionStatusRunStepV0 ||
		result.Steps[1].Execution == nil ||
		result.Steps[1].Execution.Status != DirectorBriefingExecutionStatusOutboxDispatchedV0 ||
		result.Steps[2].Execution == nil ||
		result.Steps[2].Execution.Status != DirectorBriefingExecutionStatusCloseOrIdlePendingV0 {
		t.Fatalf("steps=%+v", result.Steps)
	}
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(issues) > 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestRunResidentDirectorBriefingLoopV0NoAutoaplicaAskDirector(t *testing.T) {
	runRef := "run-resident-briefing-loop-director-001"
	action := briefingActionV0(runRef, "action-ask", orquestadirectorsupervisor.DirectorSupervisorActionKindAskDirectorV0)
	action.RequiresDirector = true
	action.SafeToApply = false
	source := &sequenceResidentBriefingSourceV0{actions: []orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{action}}

	result, err := (ServiceV0{}).RunResidentDirectorBriefingLoopV0(context.Background(), ResidentDirectorBriefingLoopRequestV0{
		RunRef:         runRef,
		BriefingSource: source,
		MaxActions:     3,
	})
	if err != nil {
		t.Fatalf("resident briefing loop: %v", err)
	}
	if result.Status != ResidentDirectorBriefingLoopStatusNeedsDirectorV0 ||
		result.ExecutedActions != 0 ||
		len(result.Steps) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunResidentDirectorBriefingLoopV0ParaEnExternalPendingSinHandler(t *testing.T) {
	runRef := "run-resident-briefing-loop-external-001"
	source := &sequenceResidentBriefingSourceV0{actions: []orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{
		briefingActionV0(runRef, "action-wait", orquestadirectorsupervisor.DirectorSupervisorActionKindWaitSignalV0),
	}}

	result, err := (ServiceV0{}).RunResidentDirectorBriefingLoopV0(context.Background(), ResidentDirectorBriefingLoopRequestV0{
		RunRef:         runRef,
		BriefingSource: source,
		MaxActions:     3,
	})
	if err != nil {
		t.Fatalf("resident briefing loop: %v", err)
	}
	if result.Status != ResidentDirectorBriefingLoopStatusExternalPendingV0 ||
		result.ExecutedActions != 1 ||
		len(result.Steps) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunResidentDirectorBriefingLoopV0RespetaMaxActions(t *testing.T) {
	runRef := "run-resident-briefing-loop-budget-001"
	source := &repeatingResidentBriefingSourceV0{
		action: briefingActionV0(runRef, "action-external", orquestadirectorsupervisor.DirectorSupervisorActionKindWaitSignalV0),
	}
	handler := &recordingBriefingExternalHandlerV0{}

	result, err := (ServiceV0{}).RunResidentDirectorBriefingLoopV0(context.Background(), ResidentDirectorBriefingLoopRequestV0{
		RunRef:                runRef,
		BriefingSource:        source,
		MaxActions:            2,
		ExternalActionHandler: handler,
	})
	if err != nil {
		t.Fatalf("resident briefing loop: %v", err)
	}
	if result.Status != ResidentDirectorBriefingLoopStatusBudgetExhaustedV0 ||
		result.ExecutedActions != 2 ||
		handler.calls != 2 ||
		source.calls != 2 {
		t.Fatalf("result=%+v handler_calls=%d source_calls=%d", result, handler.calls, source.calls)
	}
}

type sequenceResidentBriefingSourceV0 struct {
	actions []orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0
	calls   int
}

func (source *sequenceResidentBriefingSourceV0) BuildResidentDirectorBriefingV0(
	ctx context.Context,
	request ResidentDirectorBriefingBuildRequestV0,
) (orquestadirectorsupervisor.DirectorSupervisorBriefingV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{}, err
	}
	source.calls++
	if len(source.actions) < request.StepNumber {
		return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{
			SchemaVersion: orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0,
			RunRef:        request.RunRef,
			EvidenceRefs:  request.EvidenceRefs,
		}, nil
	}
	action := source.actions[request.StepNumber-1]
	return briefingWithActionV0(request.RunRef, action), nil
}

type repeatingResidentBriefingSourceV0 struct {
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0
	calls  int
}

func (source *repeatingResidentBriefingSourceV0) BuildResidentDirectorBriefingV0(
	ctx context.Context,
	request ResidentDirectorBriefingBuildRequestV0,
) (orquestadirectorsupervisor.DirectorSupervisorBriefingV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{}, err
	}
	source.calls++
	action := source.action
	action.ActionRef = source.action.ActionRef + "-" + strconv.Itoa(source.calls)
	return briefingWithActionV0(request.RunRef, action), nil
}
