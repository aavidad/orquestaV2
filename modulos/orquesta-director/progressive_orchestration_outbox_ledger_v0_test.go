package orquestadirector

import (
	"context"
	"reflect"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestapersistence "orquesta/modulos/orquesta-persistence"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgressiveOrquestaV0OutboxLedgerAntesDeDispatchRuntimeFake(t *testing.T) {
	const (
		taskRef       = "task-ref-ledger-001"
		contractRef   = "contract-ref-ledger-001"
		capacityRef   = "capacity-request-ref-ledger-001"
		agentRef      = "agent-request-ref-ledger-001"
		assessmentRef = "assessment-ref-ledger-stop-001"
	)
	ctx := context.Background()
	ledger := orquestapersistence.NewInMemoryOutboxLedgerV0()
	dispatcher := newProgressiveRuntimeFakeOutboxDispatcherV0(t)
	h := newProgressiveHarnessV0(t)

	launch := h.prepareLaunchOutboxForLedger(taskRef, contractRef, capacityRef, agentRef)
	question := h.sendDirectorQuestionOutboxForLedger(taskRef, agentRef)
	stop := h.handle(h.assessAgentLoop(assessmentRef, agentRef, taskRef, "agent-progress-report-ref-ledger-loop-001"))
	if len(stop.Events) != 2 ||
		stop.Events[0].EventType != orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0 ||
		stop.Events[1].EventType != orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0 ||
		len(stop.Outbox) != 1 {
		t.Fatalf("stop workflow result inesperado: events=%+v outbox=%+v", stop.Events, stop.Outbox)
	}

	accepted, issues := ledger.GuardarPendientes(ctx, []orquestacoreworkflow.OutboxMessageV0{launch.Outbox[0], stop.Outbox[0]})
	requireProgressiveLedgerNoIssuesV0(t, "guardar launch/stop", issues)
	if got := progressiveOutboxMessageIDsV0(accepted); !reflect.DeepEqual(got, []string{launch.Outbox[0].MessageID, stop.Outbox[0].MessageID}) {
		t.Fatalf("accepted ids=%v", got)
	}

	agentPending := progressiveLedgerPendingV0(t, ctx, ledger, h.run.RunID, orquestacoreworkflow.OutboxTargetAgentLauncherV0)
	if got := progressiveOutboxTypesV0(agentPending); !reflect.DeepEqual(got, []string{
		orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
	}) {
		t.Fatalf("agent pending types=%v", got)
	}

	launched := progressiveDispatchAndAckRuntimePendingV0(t, ctx, ledger, dispatcher, agentPending[0], "dispatch-ref-ledger-launch-001")
	if launched.Status != orquestaruntime.RuntimeFakeLifecycleLaunchedV0 || launched.AgentRequestID != agentRef {
		t.Fatalf("launch snapshot inesperado: %+v", launched)
	}
	agentPending = progressiveLedgerPendingV0(t, ctx, ledger, h.run.RunID, orquestacoreworkflow.OutboxTargetAgentLauncherV0)
	if got := progressiveOutboxMessageIDsV0(agentPending); !reflect.DeepEqual(got, []string{stop.Outbox[0].MessageID}) {
		t.Fatalf("pending tras ack launch=%v", got)
	}

	stopped := progressiveDispatchAndAckRuntimePendingV0(t, ctx, ledger, dispatcher, agentPending[0], "dispatch-ref-ledger-stop-001")
	if stopped.Status != orquestaruntime.RuntimeFakeLifecycleStoppedV0 || stopped.AgentRequestID != agentRef {
		t.Fatalf("stop snapshot inesperado: %+v", stopped)
	}
	agentPending = progressiveLedgerPendingV0(t, ctx, ledger, h.run.RunID, orquestacoreworkflow.OutboxTargetAgentLauncherV0)
	if len(agentPending) != 0 {
		t.Fatalf("agent pending tras stop ack=%v", progressiveOutboxMessageIDsV0(agentPending))
	}

	_, issues = ledger.GuardarPendientes(ctx, question.Outbox)
	requireProgressiveLedgerNoIssuesV0(t, "guardar question", issues)
	directorPending := progressiveLedgerPendingV0(t, ctx, ledger, h.run.RunID, orquestacoreworkflow.OutboxTargetDirectorV0)
	if got := progressiveOutboxTypesV0(directorPending); !reflect.DeepEqual(got, []string{orquestacoreworkflow.OutboxMessageSendDirectorQuestionV0}) {
		t.Fatalf("director pending types=%v", got)
	}
	progressiveAckLedgerMessageV0(t, ctx, ledger, directorPending[0], "dispatch-ref-ledger-question-001")
	directorPending = progressiveLedgerPendingV0(t, ctx, ledger, h.run.RunID, orquestacoreworkflow.OutboxTargetDirectorV0)
	if len(directorPending) != 0 {
		t.Fatalf("director pending tras ack=%v", progressiveOutboxMessageIDsV0(directorPending))
	}
}

func (h *progressiveHarnessV0) prepareLaunchOutboxForLedger(
	taskRef string,
	contractRef string,
	capacityRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationCommandResultV0 {
	h.t.Helper()
	const (
		brainstormRef = "brainstorm-ref-ledger-001"
		voteRef       = "vote-ref-ledger-001"
		decisionRef   = "decision-ref-ledger-001"
	)
	h.openPhase(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	h.handle(h.requestBrainstorm(brainstormRef))
	h.openPhase(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0)
	h.handle(h.requestVote(voteRef, brainstormRef))
	h.handle(h.acceptDecision(decisionRef, voteRef))
	h.openPhase(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0)
	h.handle(h.publishFunctionContract(contractRef, decisionRef))
	h.handle(h.createMicrotask(taskRef, contractRef))
	h.openPhase(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	h.handle(h.requestCapacity(capacityRef, taskRef, "high"))
	h.handle(h.registerCapacityDecision(capacityRef, "high"))
	agent := h.handle(h.requestAgent(agentRef, taskRef, capacityRef, "implementador"))
	assertProgressiveWorkflowResultV0(h.t, agent, orquestacoreworkflow.OrchestrationEventAgentRequestedV0, 1)
	return agent
}

func (h *progressiveHarnessV0) sendDirectorQuestionOutboxForLedger(
	taskRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationCommandResultV0 {
	h.t.Helper()
	report := validSupervisionProgressReportV0(h.run.RunID, agentRef, orquestaruntime.AgentStalledV0)
	report.ReportID = "agent-progress-report-ref-ledger-stalled-001"
	report.NoProgressTicks = 6
	result, err := BuildAgentProgressSupervisionV0(AgentProgressSupervisionInputV0{
		CommandMeta:   h.meta("ledger-stalled-question"),
		Report:        report,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:       taskRef,
		AssessmentRef: "assessment-ref-ledger-stalled-001",
		QuestionID:    "question-ref-ledger-stalled-001",
	})
	if err != nil {
		h.t.Fatalf("BuildAgentProgressSupervisionV0 stalled: %v", err)
	}
	h.handle(result.AssessCommand)
	if result.AskDirectorCommand == nil {
		h.t.Fatalf("AskDirectorCommand requerido")
	}
	asked := h.handle(*result.AskDirectorCommand)
	if len(asked.Outbox) != 1 || asked.Outbox[0].MessageType != orquestacoreworkflow.OutboxMessageSendDirectorQuestionV0 {
		h.t.Fatalf("SendDirectorQuestion outbox inesperado: %+v", asked.Outbox)
	}
	return asked
}

func progressiveDispatchAndAckRuntimePendingV0(
	t *testing.T,
	ctx context.Context,
	ledger *orquestapersistence.InMemoryOutboxLedgerV0,
	dispatcher *progressiveRuntimeFakeOutboxDispatcherV0,
	message orquestacoreworkflow.OutboxMessageV0,
	dispatchRef string,
) orquestaruntime.RuntimeFakeLifecycleSnapshotV0 {
	t.Helper()
	snapshot, err := dispatcher.dispatch(message)
	if err != nil {
		t.Fatalf("dispatch pending %s: %v", message.MessageID, err)
	}
	progressiveAckLedgerMessageV0(t, ctx, ledger, message, dispatchRef)
	return snapshot
}

func progressiveAckLedgerMessageV0(
	t *testing.T,
	ctx context.Context,
	ledger *orquestapersistence.InMemoryOutboxLedgerV0,
	message orquestacoreworkflow.OutboxMessageV0,
	dispatchRef string,
) {
	t.Helper()
	snapshot, issues := ledger.RegistrarAck(ctx, orquestapersistence.OutboxDispatchAckV0{
		MessageID:    message.MessageID,
		RunID:        message.RunID,
		TargetPort:   message.TargetPort,
		Status:       orquestapersistence.OutboxDispatchStatusDispatchedV0,
		DispatchRef:  dispatchRef,
		DispatchedAt: "2026-05-05T11:00:00Z",
		EvidenceRefs: []string{"evidence-ref-ledger-dispatch-001"},
	})
	requireProgressiveLedgerNoIssuesV0(t, "ack "+message.MessageID, issues)
	if snapshot.MessageID != message.MessageID || snapshot.Status != orquestapersistence.OutboxDispatchStatusDispatchedV0 {
		t.Fatalf("ack snapshot inesperado: %+v", snapshot)
	}
}

func progressiveLedgerPendingV0(
	t *testing.T,
	ctx context.Context,
	ledger *orquestapersistence.InMemoryOutboxLedgerV0,
	runID string,
	targetPort string,
) []orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	pending, issues := ledger.ListarPendientes(ctx, orquestapersistence.OutboxPendingFilterV0{
		RunID:      runID,
		TargetPort: targetPort,
	})
	requireProgressiveLedgerNoIssuesV0(t, "listar "+targetPort, issues)
	return pending
}

func requireProgressiveLedgerNoIssuesV0(t *testing.T, step string, issues []orquestapersistence.OutboxLedgerIssueV0) {
	t.Helper()
	if len(issues) != 0 {
		t.Fatalf("%s issues=%+v", step, issues)
	}
}

func progressiveOutboxMessageIDsV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.MessageID)
	}
	return ids
}

func progressiveOutboxTypesV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	types := make([]string, 0, len(messages))
	for _, message := range messages {
		types = append(types, message.MessageType)
	}
	return types
}
