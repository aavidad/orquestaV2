package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestReviewReworkReplanSourceV0UsaTaskRefDelReceiptDelRework(t *testing.T) {
	store := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
		reviewReworkDescriptorForTestV0("delivery-ref-first", "task-ref-first"),
		reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
	)
	source := ReviewReworkReplanSourceV0{
		Store: store,
		Capacity: CapacityConfigV0{
			Tier: orquestacoreworkflow.OrchestrationCapacityXHighV0,
		},
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%d %+v", len(plans), plans)
	}
	plan := plans[0]
	if plan.TaskRef != "task-ref-target" {
		t.Fatalf("task_ref=%q, want task-ref-target", plan.TaskRef)
	}
	if plan.TaskRef == "task-ref-first" {
		t.Fatalf("uso la primera tarea como preferente: %+v", plan)
	}
	if plan.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("capacity=%q", plan.MinimumRecommendedCapacity)
	}
	if plan.ReplanRef == "" || plan.CapacityRequestRef == "" ||
		plan.AgentRequestID == "" || plan.ReasonRef == "" {
		t.Fatalf("refs incompletas: %+v", plan)
	}
}

func TestReviewReworkReplanSourceV0FallbackPrimeraTareaSoloSinReceipt(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 || plans[0].TaskRef != "task-ref-first" {
		t.Fatalf("plans=%+v", plans)
	}
	if plans[0].MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("capacity fallback=%q", plans[0].MinimumRecommendedCapacity)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(plans[0].EvidenceRefs, "evidence-ref-review-rework-task-fallback") {
		t.Fatalf("fallback sin evidencia: %v", plans[0].EvidenceRefs)
	}
}

func TestReviewReworkReplanSourceV0MantienePlanTrasReplanHastaAgente(t *testing.T) {
	store := orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
		reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
	)
	source := ReviewReworkReplanSourceV0{Store: store}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(true),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 || plans[0].TaskRef != "task-ref-target" {
		t.Fatalf("plan tras replan debe seguir disponible hasta agente: %+v", plans)
	}

	request := reviewReworkPlanRequestForTestV0(true)
	request.Run.Agents = []string{plans[0].AgentRequestID}
	plans, err = source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0 con agente: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("plan debe parar tras agente solicitado: %+v", plans)
	}
}

func TestBuildStackV0CableaReviewReworkReplanSource(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if stack.Ports.ReviewReworkReplanSource == nil {
		t.Fatalf("ReviewReworkReplanSource no cableado")
	}
}

func reviewReworkPlanRequestForTestV0(
	alreadyReplanned bool,
) orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0 {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:        "run-ref-review-rework-stack-001",
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Tasks:        []string{"task-ref-first", "task-ref-target"},
		ReworkRequests: []string{
			"rework-request-ref-target#review_result:review-result-ref-target" +
				"#review_request:review-request-ref-target#delivery:delivery-ref-target",
		},
		ReviewResults: []string{
			"review-result-ref-target#review_result:changes_requested" +
				"#review_request:review-request-ref-target#delivery:delivery-ref-target",
		},
	}
	if alreadyReplanned {
		run.ReplanDecisions = []string{
			"replan-ref-target#source:rework-request-ref-target#task:task-ref-target" +
				"#action:retry_task#followups:capacity-ref-target+agent-ref-target",
		}
	}
	return orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-10T10:00:00Z",
		CorrelationID: "corr-review-rework-stack-001",
		EvidenceRefs:  []string{"evidence-ref-request-review-rework"},
	}
}

func reviewReworkDescriptorForTestV0(
	deliveryRef string,
	taskRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-" + deliveryRef,
		RunID:         "run-ref-review-rework-stack-001",
		AgentRef:      "agent-ref-" + taskRef,
		AckPath:       "ack-ref-test.json",
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID: "agent-ref-" + taskRef,
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: deliveryRef,
				},
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef: taskRef,
				},
			},
		},
	}
}

func reviewReworkPlanHasEvidenceForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
