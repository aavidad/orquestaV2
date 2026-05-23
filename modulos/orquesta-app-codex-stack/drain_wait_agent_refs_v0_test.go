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

func TestDrainRunV0ConACKCompletoReentraDirectorConWaitAgentRefsV0(t *testing.T) {
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
	waitRefs := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		waitRefs = append(waitRefs, descriptor.AgentRef)
	}
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
	for _, descriptor := range descriptors[1:] {
		if err := writeCodexStackAckForDescriptorV0(t, descriptor); err != nil {
			t.Fatalf("write remaining ack %s: %v", descriptor.AgentRef, err)
		}
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
	if finalDrain.Status == orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("drain final scoped quedo quiescent sin reentrar: drain=%+v calls=%d", finalDrain, source.Calls)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	for _, descriptor := range descriptors {
		observation := drainObservationFromDescriptorForTestV0(descriptor)
		if !drainObservationAlreadyRegisteredV0(run, observation) {
			t.Fatalf("observacion final no registrada: artifacts=%v deliveries=%v wait=%v observation=%+v", run.PhaseArtifacts, run.Deliveries, waitRefs, observation)
		}
	}
	if len(run.PhaseArtifacts) < len(waitRefs) {
		t.Fatalf("observaciones finales invalidas: artifacts=%v deliveries=%v wait=%v", run.PhaseArtifacts, run.Deliveries, waitRefs)
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
