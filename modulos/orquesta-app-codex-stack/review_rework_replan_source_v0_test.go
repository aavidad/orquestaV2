package orquestaappcodexstack

import (
	"context"
	"strings"
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

func TestReviewReworkReplanSourceV0FiltraEvidenciaAmbientalCodex(t *testing.T) {
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target"),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.EvidenceRefs = []string{
		"evidence-ref-neutral-review-rework",
		"evidence-ref-codex-supervisor-stack-drain",
		"evidence-ref-runtime-drain",
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%d %+v", len(plans), plans)
	}
	evidence := plans[0].EvidenceRefs
	if reviewReworkPlanHasEvidenceForTestV0(evidence, "evidence-ref-codex-supervisor-stack-drain") ||
		reviewReworkPlanHasEvidenceForTestV0(evidence, "evidence-ref-runtime-drain") {
		t.Fatalf("evidence_refs filtran detalles de runtime: %v", evidence)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(evidence, "evidence-ref-neutral-review-rework") {
		t.Fatalf("evidence_refs perdio evidencia neutral: %v", evidence)
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

func TestReviewReworkReplanSourceV0UsaAgentRefCortoYEstableParaReworkReal(t *testing.T) {
	longDeliveryRef := "ack-ref-app-stack-" + strings.Repeat("delivery-ref-real-review-rework-", 8)
	longTaskRef := "task-ref-app-change-" + strings.Repeat("review-rework-task-", 6)
	descriptor := reviewReworkDescriptorForTestV0(longDeliveryRef, longTaskRef)
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.Tasks = []string{longTaskRef}
	request.Run.ReworkRequests = []string{
		"rework-request-ref-" + strings.Repeat("review-result-real-damaged-delivery-", 6) +
			"#review_result:review-result-ref-long#review_request:review-request-ref-long#delivery:" +
			longDeliveryRef,
	}
	request.Run.ReviewResults = []string{
		"review-result-ref-long#review_result:changes_requested" +
			"#review_request:review-request-ref-long#delivery:" + longDeliveryRef,
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	if plans[0].TaskRef != longTaskRef {
		t.Fatalf("task_ref=%q", plans[0].TaskRef)
	}
	if len(plans[0].AgentRequestID) > 80 {
		t.Fatalf("agent_request_id demasiado largo: %s", plans[0].AgentRequestID)
	}
	if !strings.HasPrefix(plans[0].AgentRequestID, "agent-ref-task-ref-app-change-") {
		t.Fatalf("agent_request_id no conserva tarea: %s", plans[0].AgentRequestID)
	}

	request.Run.Agents = []string{plans[0].AgentRequestID}
	again, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0 replay: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("replay no debe duplicar agente de rework: %+v", again)
	}
}

func TestReviewReworkReplanSourceV0CortaBucleTrasRetriesAmpliosPorTarea(t *testing.T) {
	taskRef := "task-ref-target"
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			reviewReworkDescriptorForTestV0("delivery-ref-target", taskRef),
		),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.StartedAgents = make([]string, 0, reviewReworkReplanMaxRetryAgentsPerTaskV0)
	for idx := 0; idx < reviewReworkReplanMaxRetryAgentsPerTaskV0; idx++ {
		rework := reviewReworkProjectionV0{
			ReworkRequestRef: "rework-request-ref-target-" + string(rune('a'+idx)),
			ReviewResultRef:  "review-result-ref-target-" + string(rune('a'+idx)),
			ReviewRequestID:  "review-request-ref-target-" + string(rune('a'+idx)),
			DeliveryRef:      "delivery-ref-target-" + string(rune('a'+idx)),
		}
		request.Run.StartedAgents = append(
			request.Run.StartedAgents,
			"agent-ref-"+reviewReworkReplanAgentSuffixV0(rework, taskRef),
		)
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 0 {
		t.Fatalf("no debe encadenar rework indefinido tras limite amplio: %+v", plans)
	}
}

func TestReviewReworkReplanSourceV0DescribeWriteSetFaltanteParaElAgente(t *testing.T) {
	projectDir := t.TempDir()
	writeStackReviewGateFileForTestV0(t, projectDir, "internal/api/handler.go", "package api\n")
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.ProjectWorkDir = projectDir
	descriptor.Spec.AgentPacket.Task.WriteSet = []string{"internal/api", "web"}
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}

	plans, err := source.BuildReviewReworkReplanPlansV0(
		context.Background(),
		reviewReworkPlanRequestForTestV0(false),
	)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	if !strings.Contains(plans[0].Summary, "completar faltantes: web") {
		t.Fatalf("summary=%q", plans[0].Summary)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(plans[0].EvidenceRefs, "review-rework-missing-web") {
		t.Fatalf("evidence_refs=%v", plans[0].EvidenceRefs)
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
