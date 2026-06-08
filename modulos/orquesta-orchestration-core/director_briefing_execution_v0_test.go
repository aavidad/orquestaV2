package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestExecuteDirectorBriefingActionV0RunStepEjecutaUnaRafaga(t *testing.T) {
	runRef := "run-briefing-step-001"
	ledger := NewInMemoryOutboxLedgerV0()
	service := ServiceV0{
		RunStore: newMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateV0(runRef),
			},
		}},
		OutboxLedger: ledger,
	}

	result, err := service.ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing:     briefingWithActionV0(runRef, briefingActionV0(runRef, "action-step", orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0)),
		OccurredAt:   "2026-06-08T09:00:00Z",
		EvidenceRefs: []string{"evidence-request"},
	})
	if err != nil {
		t.Fatalf("execute briefing action: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusRunStepV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if result.Burst == nil || result.Burst.ExecutedSteps != 1 {
		t.Fatalf("burst=%+v", result.Burst)
	}
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestExecuteDirectorBriefingActionV0DispatchOutbox(t *testing.T) {
	runRef := "run-briefing-dispatch-001"
	ledger := NewInMemoryOutboxLedgerV0()
	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		mustCapacityOutboxMessageV0(t, runRef, "outbox-ref-briefing-dispatch-001"),
	})
	if len(issues) > 0 {
		t.Fatalf("seed outbox issues=%+v", issues)
	}
	service := ServiceV0{}

	result, err := service.ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing: briefingWithActionV0(runRef, briefingActionV0(runRef, "action-dispatch", orquestadirectorsupervisor.DirectorSupervisorActionKindDispatchOutboxV0)),
		Dispatchers: []OutboxDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
			Reader:     ledger,
			Claimer:    ledger,
			Executor:   fakeDispatchExecutorV0{},
			Acker:      ledger,
		}},
	})
	if err != nil {
		t.Fatalf("execute dispatch action: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusOutboxDispatchedV0 || len(result.Dispatches) != 1 {
		t.Fatalf("result=%+v", result)
	}
	pending, pendingIssues := ledger.ListPending(context.Background(),
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef},
	)
	if len(pendingIssues) > 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, pendingIssues)
	}
}

func TestExecuteDirectorBriefingActionV0DispatchOutboxRespetaTargetRefs(t *testing.T) {
	runRef := "run-briefing-target-001"
	ledger := NewInMemoryOutboxLedgerV0()
	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{
		mustCapacityOutboxMessageV0(t, runRef, "outbox-ref-ignored-001"),
		mustCapacityOutboxMessageV0(t, runRef, "outbox-ref-target-001"),
	})
	if len(issues) > 0 {
		t.Fatalf("seed outbox issues=%+v", issues)
	}
	action := briefingActionV0(runRef, "action-dispatch-target", orquestadirectorsupervisor.DirectorSupervisorActionKindDispatchOutboxV0)
	action.TargetRefs = []string{"outbox-ref-target-001"}

	result, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing: briefingWithActionV0(runRef, action),
		Dispatchers: []OutboxDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
			Reader:     ledger,
			Claimer:    ledger,
			Executor:   fakeDispatchExecutorV0{},
			Acker:      ledger,
		}},
	})
	if err != nil {
		t.Fatalf("execute targeted dispatch action: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusOutboxDispatchedV0 ||
		len(result.Dispatches) != 1 ||
		result.Dispatches[0].MessageID != "outbox-ref-target-001" {
		t.Fatalf("result=%+v", result)
	}
	pending, pendingIssues := ledger.ListPending(context.Background(),
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef},
	)
	if len(pendingIssues) > 0 || len(pending) != 1 || pending[0].MessageID != "outbox-ref-ignored-001" {
		t.Fatalf("pending=%+v issues=%+v", pending, pendingIssues)
	}
}

func TestExecuteDirectorBriefingActionV0ExternalPendingSinHandler(t *testing.T) {
	runRef := "run-briefing-external-001"
	result, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing: briefingWithActionV0(runRef, briefingActionV0(runRef, "action-wait", orquestadirectorsupervisor.DirectorSupervisorActionKindWaitSignalV0)),
	})
	if err != nil {
		t.Fatalf("execute external action: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusExternalPendingV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestExecuteDirectorBriefingActionV0DelegatesExternalHandler(t *testing.T) {
	runRef := "run-briefing-handler-001"
	handler := &recordingBriefingExternalHandlerV0{}
	result, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing:              briefingWithActionV0(runRef, briefingActionV0(runRef, "action-ask", orquestadirectorsupervisor.DirectorSupervisorActionKindAskDirectorV0)),
		ExternalActionHandler: handler,
	})
	if err != nil {
		t.Fatalf("execute handler action: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusExternalAppliedV0 || handler.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, handler.calls)
	}
	if result.External == nil || result.External.ActionRef != "action-ask" {
		t.Fatalf("external=%+v", result.External)
	}
}

func TestExecuteDirectorBriefingActionV0PropagaExternalPendingDelHandler(t *testing.T) {
	runRef := "run-briefing-handler-pending-001"
	handler := &recordingBriefingExternalHandlerV0{
		status: DirectorBriefingExecutionStatusExternalPendingV0,
	}
	result, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing:              briefingWithActionV0(runRef, briefingActionV0(runRef, "action-pending", orquestadirectorsupervisor.DirectorSupervisorActionKindAskDirectorV0)),
		ExternalActionHandler: handler,
	})
	if err != nil {
		t.Fatalf("execute handler action: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusExternalPendingV0 ||
		result.External == nil ||
		result.External.Status != DirectorBriefingExecutionStatusExternalPendingV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestExecuteDirectorBriefingActionV0RechazaRunRefInconsistente(t *testing.T) {
	runRef := "run-briefing-invalid-001"
	_, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing: briefingWithActionV0(runRef, briefingActionV0("run-otro", "action-invalid", orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0)),
	})
	assertNucleoErrorV0(t, err, ErrDirectorBriefingExecutionInvalidV0, "action.run_ref")
}

func TestExecuteDirectorBriefingActionV0DevuelveErrorDelHandler(t *testing.T) {
	runRef := "run-briefing-handler-error-001"
	handler := &recordingBriefingExternalHandlerV0{err: errors.New("handler temporalmente no disponible")}
	result, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing:              briefingWithActionV0(runRef, briefingActionV0(runRef, "action-error", orquestadirectorsupervisor.DirectorSupervisorActionKindInspectErrorV0)),
		ExternalActionHandler: handler,
	})
	if err == nil {
		t.Fatal("expected handler error")
	}
	if result.Status != DirectorBriefingExecutionStatusExternalFailedV0 || result.External == nil {
		t.Fatalf("result=%+v", result)
	}
}

func TestExecuteDirectorBriefingActionV0PermiteKindNuevoPorHandler(t *testing.T) {
	runRef := "run-briefing-kind-nuevo-001"
	handler := &recordingBriefingExternalHandlerV0{}
	result, err := (ServiceV0{}).ExecuteDirectorBriefingActionV0(context.Background(), DirectorBriefingExecutionRequestV0{
		Briefing:              briefingWithActionV0(runRef, briefingActionV0(runRef, "action-kind-new", "accion_futura_por_puerto")),
		ExternalActionHandler: handler,
	})
	if err != nil {
		t.Fatalf("execute future kind: %v", err)
	}
	if result.Status != DirectorBriefingExecutionStatusExternalAppliedV0 || handler.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, handler.calls)
	}
}

type recordingBriefingExternalHandlerV0 struct {
	calls  int
	status string
	err    error
}

func (handler *recordingBriefingExternalHandlerV0) ExecuteDirectorBriefingExternalActionV0(
	ctx context.Context,
	request DirectorBriefingExternalActionRequestV0,
) (DirectorBriefingExternalActionResultV0, error) {
	if err := ctx.Err(); err != nil {
		return DirectorBriefingExternalActionResultV0{}, err
	}
	handler.calls++
	return DirectorBriefingExternalActionResultV0{
		Status:       handler.status,
		RunRef:       request.Briefing.RunRef,
		ActionRef:    request.Action.ActionRef,
		EvidenceRefs: []string{"evidence-external-handler"},
	}, handler.err
}

func briefingWithActionV0(
	runRef string,
	action orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0,
) orquestadirectorsupervisor.DirectorSupervisorBriefingV0 {
	return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{
		SchemaVersion: orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0,
		RunRef:        runRef,
		NextAction:    &action,
		ActionQueue:   []orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{action},
		EvidenceRefs:  []string{"evidence-briefing"},
	}
}

func briefingActionV0(
	runRef string,
	actionRef string,
	kind string,
) orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0 {
	return orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{
		ActionRef:   actionRef,
		Kind:        kind,
		RunRef:      runRef,
		ReasonCode:  "test_briefing",
		SafeToApply: true,
	}
}
