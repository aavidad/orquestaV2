package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDrainRunV0RegistraEntregasDeProgramacionTrasDecisionDirector(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	codexStackWriteDelayedBatchDirectorDecisionsWithMicrotaskPhaseForTestV0(
		t,
		stack,
		director.RunRef,
		string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
	)

	drain, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-programming-delivery-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          24,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     4,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v drain=%+v", err, drain)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	descriptors := codexStackProgrammingDescriptorsForRunV0(t, stack, director.RunRef)
	if len(descriptors) != len(codexStackBatchDecisionTaskRefsForTestV0()) {
		t.Fatalf("programming descriptors=%d want=%d", len(descriptors), len(codexStackBatchDecisionTaskRefsForTestV0()))
	}
	if !codexStackRealSmokeAllProgrammingDeliveriesRegisteredV0(run.Deliveries, descriptors) {
		t.Fatalf("deliveries=%v descriptors=%v", run.Deliveries, codexStackRealSmokeProgrammingDescriptorsV0(descriptors))
	}
	if drain.Final.PendingOutboxCount != 0 {
		t.Fatalf("pending_outbox=%d refs=%v", drain.Final.PendingOutboxCount, drain.Final.PendingOutboxRefs)
	}
}

func codexStackProgrammingDescriptorsForRunV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	store, ok := stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0)
	if !ok {
		t.Fatalf("receipt store inesperado: %T", stack.Stores.ReceiptStore)
	}
	descriptors, err := store.ListCodexReceiptDescriptorsV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	return codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors)
}
