package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgressiveOrquestaV0FlujoCompletoAppSimpleHastaCierre(t *testing.T) {
	const (
		taskRef           = "task-ref-full-flow-001"
		contractRef       = "contract-ref-full-flow-001"
		capacityRef       = "capacity-request-ref-full-flow-001"
		agentRef          = "agent-request-ref-full-flow-001"
		deliveryRef       = "delivery-ref-full-flow-001"
		reviewRef         = "review-request-ref-full-flow-001"
		reviewResultRef   = "review-result-ref-full-flow-001"
		acceptedReviewRef = "accepted-review-ref-full-flow-001"
		validationRef     = "validation-ref-full-flow-001"
		closureRef        = "closure-ref-full-flow-001"
	)

	h := newProgressiveHarnessV0(t)
	runtimeFake := newProgressiveRuntimeFakeOutboxDispatcherV0(t)
	launched := h.prepareFullFlowTask(taskRef, contractRef, capacityRef, agentRef, runtimeFake)
	assertProgressiveRuntimeLaunchV0(t, launched, agentRef)
	started := h.handle(h.registerAgentStarted(agentRef, launched.LaunchRef))
	assertProgressiveWorkflowResultV0(t, started, orquestacoreworkflow.OrchestrationEventAgentStartedV0, 0)

	delivery := h.handle(h.registerDelivery(deliveryRef, taskRef, agentRef))
	assertProgressiveWorkflowResultV0(t, delivery, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, 0)

	h.openPhase(orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	review := h.handle(h.requestReview(reviewRef, deliveryRef))
	assertProgressiveWorkflowResultV0(t, review, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, 0)

	reviewResult := h.handle(h.recordAcceptedReviewResult(reviewResultRef, reviewRef, deliveryRef))
	assertProgressiveWorkflowResultV0(t, reviewResult, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, 0)

	accepted := h.handle(h.acceptReview(acceptedReviewRef, reviewRef, deliveryRef))
	assertProgressiveWorkflowResultV0(t, accepted, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, 0)

	closedTask := h.handle(h.closeTask(taskRef, deliveryRef, acceptedReviewRef))
	assertProgressiveWorkflowResultV0(t, closedTask, orquestacoreworkflow.OrchestrationEventTaskClosedV0, 0)

	h.openPhase(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0)
	validation := h.handle(h.registerFinalValidation(validationRef, taskRef))
	assertProgressiveWorkflowResultV0(t, validation, orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0, 0)

	h.openPhase(orquestacoreworkflow.OrchestrationPhaseCierreV0)
	closure := h.handle(h.closeRun(closureRef, validationRef))
	assertProgressiveWorkflowResultV0(t, closure, orquestacoreworkflow.OrchestrationEventRunClosedV0, 0)

	assertProgressiveFullFlowClosedV0(t, h.run, taskRef, deliveryRef, reviewRef, acceptedReviewRef, validationRef, closureRef)
	t.Logf(
		"flujo completo: status=%s fase=%s eventos=%d entrega=%s revision=%s validacion=%s cierre=%s",
		h.run.Status,
		h.run.CurrentPhase,
		len(h.events),
		deliveryRef,
		acceptedReviewRef,
		validationRef,
		closureRef,
	)
}

func (h *progressiveHarnessV0) prepareFullFlowTask(
	taskRef string,
	contractRef string,
	capacityRef string,
	agentRef string,
	runtimeFake *progressiveRuntimeFakeOutboxDispatcherV0,
) orquestaruntime.RuntimeFakeLifecycleSnapshotV0 {
	h.t.Helper()
	const (
		brainstormRef = "brainstorm-ref-full-flow-001"
		voteRef       = "vote-ref-full-flow-001"
		decisionRef   = "decision-ref-full-flow-001"
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
	capacity := h.handle(h.requestCapacity(capacityRef, taskRef, "high"))
	assertProgressiveWorkflowResultV0(h.t, capacity, orquestacoreworkflow.OrchestrationEventCapacityRequestedV0, 1)
	decision := h.handle(h.registerCapacityDecision(capacityRef, "high"))
	assertProgressiveWorkflowResultV0(h.t, decision, orquestacoreworkflow.OrchestrationEventCapacityDecidedV0, 0)
	agent := h.handle(h.requestAgent(agentRef, taskRef, capacityRef, "implementador"))
	assertProgressiveWorkflowResultV0(h.t, agent, orquestacoreworkflow.OrchestrationEventAgentRequestedV0, 1)
	return runtimeFake.mustDispatchOnly(agent.Outbox)
}

func assertProgressiveWorkflowResultV0(
	t *testing.T,
	result orquestacoreworkflow.OrchestrationCommandResultV0,
	eventType string,
	outboxLen int,
) {
	t.Helper()
	if len(result.Events) != 1 || result.Events[0].EventType != eventType {
		t.Fatalf("evento=%+v, want one %s", result.Events, eventType)
	}
	if len(result.Outbox) != outboxLen {
		t.Fatalf("outbox=%d, want %d", len(result.Outbox), outboxLen)
	}
}
