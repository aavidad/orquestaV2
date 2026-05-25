package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDomainWorkPendingArtifactsV0RespetaWaitAgentRefs(t *testing.T) {
	ctx := context.Background()
	runtime := &noAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	result := postExternalWorkRunStackV0(t, stack)
	drainDomainWorkRunToLaunchForWaitAgentRefsTestV0(t, stack, result.RunRef)
	descriptor := domainWorkDescriptorForRunWaitAgentRefsTestV0(t, stack, result.RunRef)
	writeDomainWorkAckForWaitAgentRefsTestV0(t, runtime.fakeCodexStackRuntimeV0, descriptor)
	markDomainWorkDeliveryRegisteredForWaitAgentRefsTestV0(t, stack, result.RunRef, descriptor)
	domainWork.inputs = nil

	run := mustLoadCodexStackRunForTestV0(t, stack, result.RunRef)
	if err := stack.submitPendingDomainWorkArtifactsV0(ctx, DrainRunRequestV0{
		RunRef:        result.RunRef,
		OccurredAt:    "2026-05-17T12:00:00Z",
		CorrelationID: "corr-domain-work-wait-agent-refs-pending",
		WaitAgentRefs: []string{"agent-ref-out-of-scope"},
	}, run); err != nil {
		t.Fatalf("submitPendingDomainWorkArtifactsV0 fuera de scope: %v", err)
	}
	if submit, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); ok {
		t.Fatalf("submit_artifact fuera de scope: %+v", submit.ArtifactSubmission)
	}

	if err := stack.submitPendingDomainWorkArtifactsV0(ctx, DrainRunRequestV0{
		RunRef:        result.RunRef,
		OccurredAt:    "2026-05-17T12:01:00Z",
		CorrelationID: "corr-domain-work-wait-agent-refs-pending-in-scope",
		WaitAgentRefs: []string{descriptor.AgentRef},
	}, run); err != nil {
		t.Fatalf("submitPendingDomainWorkArtifactsV0 en scope: %v", err)
	}
	if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); !ok {
		t.Fatalf("submit_artifact en scope no invocado: inputs=%+v", domainWork.inputs)
	}
}

func TestDomainWorkRecoveryV0RespetaWaitAgentRefs(t *testing.T) {
	ctx := context.Background()
	runtime := newDomainWorkRunningMissingACKRuntimeV0()
	domainWork := &fakeCodexStackDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	result := postExternalWorkRunStackV0(t, stack)
	drainDomainWorkRunToLaunchForWaitAgentRefsTestV0(t, stack, result.RunRef)
	descriptor := domainWorkDescriptorForRunWaitAgentRefsTestV0(t, stack, result.RunRef)
	writeDomainWorkAckFailureLastMessageForTestV0(t, stack, result.RunRef)
	domainWork.inputs = nil

	run := mustLoadCodexStackRunForTestV0(t, stack, result.RunRef)
	observations, err := stack.recoverableDomainWorkDeliveryObservationsV0(ctx, DrainRunRequestV0{
		RunRef:        result.RunRef,
		OccurredAt:    "2026-05-17T12:02:00Z",
		CorrelationID: "corr-domain-work-wait-agent-refs-recovery",
		WaitAgentRefs: []string{"agent-ref-out-of-scope"},
	}, run)
	if err != nil {
		t.Fatalf("recoverableDomainWorkDeliveryObservationsV0 fuera de scope: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations fuera de scope=%+v", observations)
	}

	observations, err = stack.recoverableDomainWorkDeliveryObservationsV0(ctx, DrainRunRequestV0{
		RunRef:        result.RunRef,
		OccurredAt:    "2026-05-17T12:03:00Z",
		CorrelationID: "corr-domain-work-wait-agent-refs-recovery-in-scope",
		WaitAgentRefs: []string{descriptor.AgentRef},
	}, run)
	if err != nil {
		t.Fatalf("recoverableDomainWorkDeliveryObservationsV0 en scope: %v", err)
	}
	if len(observations) != 1 || observations[0].AgentRef != descriptor.AgentRef {
		t.Fatalf("observations en scope=%+v descriptor=%+v", observations, descriptor)
	}

	markDomainWorkRunStoppedForTestV0(t, stack, result.RunRef)
	run = mustLoadCodexStackRunForTestV0(t, stack, result.RunRef)
	if err := stack.submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0(ctx, DrainRunRequestV0{
		RunRef:        result.RunRef,
		OccurredAt:    "2026-05-17T12:04:00Z",
		CorrelationID: "corr-domain-work-wait-agent-refs-direct",
		WaitAgentRefs: []string{"agent-ref-out-of-scope"},
	}, run); err != nil {
		t.Fatalf("submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0 fuera de scope: %v", err)
	}
	if submit, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); ok {
		t.Fatalf("submit_artifact directo fuera de scope: %+v", submit.ArtifactSubmission)
	}

	if err := stack.submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0(ctx, DrainRunRequestV0{
		RunRef:        result.RunRef,
		OccurredAt:    "2026-05-17T12:05:00Z",
		CorrelationID: "corr-domain-work-wait-agent-refs-direct-in-scope",
		WaitAgentRefs: []string{descriptor.AgentRef},
	}, run); err != nil {
		t.Fatalf("submitRecoverableDomainWorkArtifactsWithoutCoreDeliveryV0 en scope: %v", err)
	}
	if _, ok := findDomainWorkSubmitForTestV0(domainWork.inputs); !ok {
		t.Fatalf("submit_artifact directo en scope no invocado: inputs=%+v", domainWork.inputs)
	}
}

func drainDomainWorkRunToLaunchForWaitAgentRefsTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	for attempt := 1; attempt <= 6; attempt++ {
		descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
			context.Background(),
			orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
		)
		if err != nil {
			t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
		}
		if len(descriptors) > 0 {
			return
		}
		if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               runRef,
			OccurredAt:           "2026-05-17T11:59:00Z",
			CorrelationID:        "corr-domain-work-wait-agent-refs-launch",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxDecisionCycles:    4,
			MaxExternalWaits:     0,
		}); err != nil {
			t.Fatalf("DrainRunV0 launch intento %d: %v", attempt, err)
		}
	}
	t.Fatalf("descriptor domain_work no creado para run=%s", runRef)
}

func writeDomainWorkAckForWaitAgentRefsTestV0(
	t *testing.T,
	runtime *fakeCodexStackRuntimeV0,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) {
	t.Helper()
	files, err := runtime.writeDeliveryFilesV0(
		descriptor.ProjectWorkDir,
		descriptor.Spec.AgentPacket.Task.WriteSet,
	)
	if err != nil {
		t.Fatalf("write delivery files: %v", err)
	}
	packet := descriptor.Spec.AgentPacket
	data, err := json.Marshal(map[string]any{
		"schema_version": "codex_agent_ack.v0",
		"request_id":     packet.RequestID,
		"correlation_id": packet.CorrelationID,
		"ack_ref":        packet.DeliveryRefs.AckRef,
		"target_module":  packet.TargetModule,
		"task_ref":       packet.Task.TaskRef,
		"status":         "completed",
		"files":          files,
		"tests":          packet.Task.RequiredTests,
		"test_receipts":  codexStackRequiredTestReceiptsV0(packet.Task.RequiredTests),
		"notes":          codexStackFakeAckNotesV0(packet, "ack de prueba para wait_agent_refs domain_work"),
	})
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(descriptor.AckPath, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}

func domainWorkDescriptorForRunWaitAgentRefsTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	descriptors, err := stack.Stores.ReceiptStore.ListCodexReceiptDescriptorsV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: runRef},
	)
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v", descriptors)
	}
	return descriptors[0]
}

func markDomainWorkDeliveryRegisteredForWaitAgentRefsTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) {
	t.Helper()
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	run.Deliveries = compactStringsV0(append(
		run.Deliveries,
		descriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
	))
	run.DeliveredAgents = compactStringsV0(append(run.DeliveredAgents, descriptor.AgentRef))
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
}
