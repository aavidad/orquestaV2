package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexStackV0ReviewChangesRequestedReplanificaYArrancaAgente(t *testing.T) {
	ctx := context.Background()
	runtime := newDecisionWritingFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-review-rework-program-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 programacion: %v", err)
	}
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-review-rework-delivery-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("DrainRunV0 entrega: %v", err)
	}

	descriptor := mustCodexStackDescriptorByTaskRefV0(t, stack, "task-ref-stack-agenda-001")
	deliveryRef := descriptor.Spec.AgentPacket.DeliveryRefs.AckRef
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackHasRefV0(run.Deliveries, deliveryRef) {
		t.Fatalf("delivery inicial no registrada: deliveries=%v delivery_ref=%s", run.Deliveries, deliveryRef)
	}
	writeStackReviewGateFileForTestV0(t, descriptor.ProjectWorkDir, "go.mod", strings.Repeat("linea\n", 301))
	openCodexStackPhaseForTestV0(
		t,
		stack,
		director.RunRef,
		orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		"Revisar entrega de programacion.",
	)
	assertCodexStackReviewObservationForReworkV0(t, stack, run, deliveryRef)

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-review-rework-replan-001",
		MaxBursts:            24,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     2,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 review/rework: %v issues=%+v status=%s final=%+v",
			err,
			codexStackBurstIssuesForErrorV0(err),
			drain.Status,
			drain.Final,
		)
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackRefsContainPartV0(run.Reviews, deliveryRef) ||
		!codexStackRefsContainPartV0(run.ReviewResults, "#review_result:changes_requested") ||
		!codexStackRefsContainPartV0(run.ReworkRequests, deliveryRef) ||
		!codexStackRefsContainPartV0(run.ReplanDecisions, "#action:retry_task") ||
		!codexStackRefsContainPartV0(run.CapacityDecisions, "#capacity_decision:") ||
		!codexStackRefsContainPartV0(run.StartedAgents, "agent-ref-rework-request-ref-") {
		t.Fatalf("run sin retrabajo completo delivery=%s drain=%s phase=%s reviews=%v results=%v reworks=%v replan=%v capacity=%v started=%v",
			deliveryRef,
			drain.Status,
			run.CurrentPhase,
			run.Reviews,
			run.ReviewResults,
			run.ReworkRequests,
			run.ReplanDecisions,
			run.CapacityDecisions,
			run.StartedAgents,
		)
	}
	if codexStackRefsContainPartV0(run.AcceptedReviews, deliveryRef) {
		t.Fatalf("entrega con cambios solicitados no debe aceptarse: accepted_reviews=%v", run.AcceptedReviews)
	}
}

func assertCodexStackReviewObservationForReworkV0(
	t *testing.T,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
) {
	t.Helper()
	observations, err := stack.Ports.ReviewGateSource.BuildReviewGateObservationsV0(
		context.Background(),
		orquestacionnucleoapp.ReviewGateObservationRequestV0{
			Run:           run,
			CorrelationID: "corr-stack-review-rework-source-001",
			EvidenceRefs:  []string{"evidence-ref-stack-review-rework-source"},
		},
	)
	if err != nil {
		t.Fatalf("BuildReviewGateObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].Status != orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 {
		t.Fatalf("observations de revision invalidas delivery=%s run=%+v observations=%+v",
			deliveryRef,
			run,
			observations,
		)
	}
}

func openCodexStackPhaseForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
) {
	t.Helper()
	command, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-stack-open-phase-" + string(phase),
			RunID:          runRef,
			IdempotencyKey: "idem-stack-open-phase-" + string(phase),
			CorrelationID:  "corr-stack-open-phase-" + string(phase),
			RequestedBy:    "orquesta-app-codex-stack-test",
			OccurredAt:     "2026-05-10T12:10:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(phase),
			Reason:  reason,
		},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(
		context.Background(),
		stack.Stores.RunStore,
		stack.Ports.EventSink,
		command,
	); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 open phase: %v", err)
	}
}
