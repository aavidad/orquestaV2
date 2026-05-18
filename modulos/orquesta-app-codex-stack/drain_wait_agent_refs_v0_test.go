package orquestaappcodexstack

import (
	"context"
	"testing"

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
		MaxExternalWaits:     0,
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
