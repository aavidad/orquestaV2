package orquestaappcodexstack

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackDeliveryReviewClosureCausalComposicionV0(t *testing.T) {
	ctx := context.Background()
	cfg := codexStackRequiredTestLocalConfigV0(t)
	writeCodexStackRequiredTestTinyGoModuleV0(t, cfg.ProjectWorkDir)
	runtime := newFakeCodexStackRuntimeV0().withDeliveryBodyForTargetV0(
		"README.md",
		"# H0c\n\nEntrega observable para review y cierre causal.\n",
	)
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack := codexStackRealRequiredTestRunnerStackV0(
		t,
		cfg,
		runtime,
		evidenceStore,
		codexStackRequiredTestGoCommandV0(t),
		filepath.Join(t.TempDir(), "required-test-output"),
	)

	if _, ok := stack.Ports.DeliverySource.(orquestaruntimecodexdelivery.CodexDeliveryObservationSourceV0); !ok {
		t.Fatalf("delivery source no es la fuente real Codex: %T", stack.Ports.DeliverySource)
	}
	reviewSource, ok := stack.Ports.ReviewGateSource.(codexStackReviewGateRepairSourceV0)
	if !ok {
		t.Fatalf("review source no usa composicion real Codex: %T", stack.Ports.ReviewGateSource)
	}
	if _, ok := reviewSource.Inner.(orquestaruntimecodexdelivery.CodexReviewGateObservationSourceV0); !ok {
		t.Fatalf("review inner no es CodexReviewGateObservationSourceV0: %T", reviewSource.Inner)
	}
	if stack.Ports.OperationalClosureSource == nil || stack.Ports.ProgressSource == nil {
		t.Fatalf("fuentes causales incompletas: closure=%T progress=%T", stack.Ports.OperationalClosureSource, stack.Ports.ProgressSource)
	}

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, legacyAutoprogrammingPrepareRunInputForStackTestV0(orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-h0c-delivery-review-closure-001",
		CorrelationID:          "corr-h0c-delivery-review-closure-001",
		OccurredAt:             "2026-07-12T10:30:00Z",
		RequestedBy:            "h0c-composition-smoke",
		AutoprogrammingRequest: h0cAutoprogrammingRequestForTestV0(),
		MaxBursts:              8,
		MaxStepsPerBurst:       8,
		MaxDispatchesPerWait:   4,
		MaxCommands:            16,
		MaxOutboxPerCycle:      8,
	}))
	if !prepared.Accepted || prepared.RunRef == "" || prepared.Continue == nil || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepare no materializo run/agente H0c: %+v", prepared)
	}

	for cycle := 1; cycle <= 16; cycle++ {
		result := postRunSupervisorStackV0(t, stack, legacyRunSupervisorInputForStackTestV0(orquestamcp.MCPRunSupervisorToolInputV0{
			RequestID:                  fmt.Sprintf("request-h0c-supervisor-%03d", cycle),
			CorrelationID:              fmt.Sprintf("corr-h0c-supervisor-%03d", cycle),
			RunRef:                     prepared.RunRef,
			OperationalDirectorPlanRef: prepared.Continue.OperationalDirectorPlanRef,
			MaxTicks:                   1,
			MaxRunsPerTick:             1,
			MaxExecutions:              1,
			MaxBursts:                  12,
			MaxStepsPerBurst:           12,
			MaxDispatchesPerWait:       6,
			MaxCommands:                32,
			MaxOutboxPerCycle:          16,
			MaxDecisionCycles:          8,
			MaxExternalWaits:           1,
			AllowRepeatedRuns:          true,
		}))
		if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
			t.Fatalf("supervisor H0c cycle=%d result=%+v", cycle, result)
		}
		if mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef).Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
			break
		}
	}

	run := mustLoadCodexStackRunForTestV0(t, stack, prepared.RunRef)
	eventSink, ok := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	if !ok {
		t.Fatalf("event sink no permite evidencia durable: %T", stack.Stores.EventSink)
	}
	events, err := eventSink.LoadRunEventsV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunEventsV0: %v", err)
	}
	if err := verifyH0cDeliveryReviewClosureChainV0(run, events); err != nil {
		t.Fatal(err)
	}

	brokenACK := run
	brokenACK.Deliveries = nil
	if err := verifyH0cDeliveryReviewClosureChainV0(brokenACK, events); err == nil || !strings.Contains(err.Error(), "delivery_ack_missing") {
		t.Fatalf("traza sin ACK/delivery no quedo roja: %v", err)
	}
	brokenReview := run
	brokenReview.AcceptedReviews = nil
	if err := verifyH0cDeliveryReviewClosureChainV0(brokenReview, events); err == nil || !strings.Contains(err.Error(), "review_gate_not_consulted") {
		t.Fatalf("traza sin review gate no quedo roja: %v", err)
	}

	receiptStore, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("receipt store inesperado: %T", stack.Stores.ReceiptStore)
	}
	descriptors, err := receiptStore.ListCodexReceiptDescriptorsV0(ctx, orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: prepared.RunRef})
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("receipt durable ausente: descriptors=%+v err=%v", descriptors, err)
	}
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexDeliveryAckFileV0(descriptors[0].AckPath, descriptors[0].Spec)
	if len(issues) > 0 || ack.Status != "completed" || ack.AckRef == "" || len(ack.Files) == 0 {
		t.Fatalf("ACK no observable/revalidable: ack=%+v issues=%+v", ack, issues)
	}

	state := autoprogrammingClosurePlanStateForTestV0(t, stack, prepared.RunRef, prepared.Continue.OperationalDirectorPlanRef)
	closureStep := codexStackRequiredTestPlanStepV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		len(closureStep.DeliveryRefs) == 0 ||
		len(closureStep.ReviewResultRefs) == 0 ||
		len(closureStep.AcceptedReviewRefs) == 0 ||
		len(closureStep.RequiredTestEvidenceRefs) == 0 {
		t.Fatalf("plan sin refs causales durables: state=%+v closure=%+v", state, closureStep)
	}
}

func verifyH0cDeliveryReviewClosureChainV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if len(run.Deliveries) == 0 {
		return fmt.Errorf("h0c_delivery_ack_missing")
	}
	if len(run.ReviewResults) == 0 || len(run.AcceptedReviews) == 0 {
		return fmt.Errorf("h0c_review_gate_not_consulted")
	}
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 || len(run.Closures) == 0 {
		return fmt.Errorf("h0c_causal_closure_missing")
	}
	required := []string{
		orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0,
		orquestacoreworkflow.OrchestrationEventReviewRequestedV0,
		orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0,
		orquestacoreworkflow.OrchestrationEventReviewAcceptedV0,
		orquestacoreworkflow.OrchestrationEventTaskClosedV0,
		orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0,
		orquestacoreworkflow.OrchestrationEventRunClosedV0,
	}
	next := 0
	for _, event := range events {
		if next < len(required) && event.EventType == required[next] {
			next++
		}
	}
	if next != len(required) {
		return fmt.Errorf("h0c_event_chain_incomplete: next=%s", required[next])
	}
	return nil
}

func h0cAutoprogrammingRequestForTestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	return orquestaautoprogramming.AutoprogrammingRequestV0{
		RequestRef:       "run-h0c-delivery-review-closure-001",
		ProjectRef:       "project-ref-h0c-delivery-review-closure-001",
		WorktreeRef:      "worktree-ref-h0c-delivery-review-closure-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-h0c-delivery-review-closure-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            "source-task-ref-h0c-delivery-review-closure-001",
			Area:               "docs",
			Objective:          "Materializar una entrega pequena y recorrer review y cierre causal.",
			AcceptanceCriteria: []string{"ACK observado, review aceptada y test independiente pasado"},
		}},
		WriteSet:      []string{"README.md"},
		RequiredTests: []string{"go test ./..."},
	}
}
