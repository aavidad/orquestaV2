package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDrainRunV0WaitAgentRefsNoIngiereACKFueraDeScope(t *testing.T) {
	runtime := &noAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
	stack := mustBuildCodexStackForTestV0(t, runtime)
	ctx := context.Background()

	director := postDirectorAPIV0(t, stack)
	receiptStore := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	descriptors := codexStackRealSmokeDescriptorsV0(t, receiptStore)
	if len(descriptors) < 2 {
		t.Fatalf("descriptors=%d want>=2", len(descriptors))
	}
	scoped := descriptors[0]
	outOfScope := descriptors[1]
	if err := writeCodexStackAckForDescriptorV0(t, scoped); err != nil {
		t.Fatalf("write scoped ack: %v", err)
	}
	if err := writeCodexStackAckForDescriptorV0(t, outOfScope); err != nil {
		t.Fatalf("write out-of-scope ack: %v", err)
	}

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-drain-wait-agent-refs-001",
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxDispatchesPerWait: 1,
		MaxCommands:          4,
		MaxOutboxPerCycle:    1,
		MaxDecisionCycles:    1,
		MaxExternalWaits:     1,
		WaitAgentRefs:        []string{scoped.AgentRef},
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}

	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	scopedObservation := drainObservationFromDescriptorForTestV0(scoped)
	outOfScopeObservation := drainObservationFromDescriptorForTestV0(outOfScope)
	if !drainObservationAlreadyRegisteredV0(run, scopedObservation) {
		t.Fatalf("scoped observation no registrada: deliveries=%v artifacts=%v observation=%+v", run.Deliveries, run.PhaseArtifacts, scopedObservation)
	}
	if drainObservationAlreadyRegisteredV0(run, outOfScopeObservation) {
		t.Fatalf("out-of-scope observation registrada: deliveries=%v artifacts=%v observation=%+v", run.Deliveries, run.PhaseArtifacts, outOfScopeObservation)
	}
	if codexStackStringInSetForTestV0(run.DeliveredAgents, outOfScope.AgentRef) {
		t.Fatalf("delivered_agents=%v contains out-of-scope agent=%s", run.DeliveredAgents, outOfScope.AgentRef)
	}
}

func TestDrainRunV0ConACKParcialNoReentraDirectorHastaCerrarWaitAgentRefs(t *testing.T) {
	runtime := &noAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
	stack := mustBuildCodexStackForTestV0(t, runtime)
	ctx := context.Background()
	director := postDirectorAPIV0(t, stack)
	receiptStore := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	descriptors := codexStackRealSmokeDescriptorsV0(t, receiptStore)
	if len(descriptors) < 2 {
		t.Fatalf("descriptors=%d want>=2", len(descriptors))
	}
	first := descriptors[0]
	second := descriptors[1]
	waitRefs := []string{first.AgentRef, second.AgentRef}
	source := &countingCodexStackDirectorDecisionSourceV0{}
	stack.Ports.DirectorDecisionSource = source
	if err := writeCodexStackAckForDescriptorV0(t, first); err != nil {
		t.Fatalf("write first ack: %v", err)
	}

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-drain-partial-ack-wait-refs-001",
		MaxBursts:            4,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		MaxCommands:          8,
		MaxOutboxPerCycle:    4,
		MaxDecisionCycles:    2,
		MaxExternalWaits:     1,
		WaitAgentRefs:        waitRefs,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 first: %v", err)
	}
	if drain.Final.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		drain.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		source.Calls != 0 {
		t.Fatalf("drain parcial reentro/cambio run: drain=%+v calls=%d", drain, source.Calls)
	}
	if err := writeCodexStackAckForDescriptorV0(t, second); err != nil {
		t.Fatalf("write second ack: %v", err)
	}
	finalDrain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-drain-partial-ack-wait-refs-002",
		MaxBursts:            4,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		MaxCommands:          8,
		MaxOutboxPerCycle:    4,
		MaxDecisionCycles:    2,
		MaxExternalWaits:     1,
		WaitAgentRefs:        waitRefs,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 second: %v", err)
	}
	if finalDrain.Final.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		t.Fatalf("drain final scoped cambio estado del run: drain=%+v", finalDrain)
	}
	if source.Calls != 0 {
		t.Fatalf("drain final scoped reentro: drain=%+v calls=%d", finalDrain, source.Calls)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	firstObservation := drainObservationFromDescriptorForTestV0(first)
	secondObservation := drainObservationFromDescriptorForTestV0(second)
	if !drainObservationAlreadyRegisteredV0(run, firstObservation) ||
		!drainObservationAlreadyRegisteredV0(run, secondObservation) {
		t.Fatalf("observaciones finales invalidas: artifacts=%v deliveries=%v wait=%v", run.PhaseArtifacts, run.Deliveries, waitRefs)
	}
	if len(run.Reviews) != 0 || len(run.AcceptedReviews) != 0 || len(run.ClosedTasks) != 0 {
		t.Fatalf("drain scoped no debe revisar/cerrar en el mismo intento: run=%+v", run)
	}
}

type countingCodexStackDirectorDecisionSourceV0 struct {
	Calls int
}

func (source *countingCodexStackDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	_ context.Context,
	_ orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	source.Calls++
	return nil, nil
}
