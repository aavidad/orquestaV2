package orquestaappcodexstack

import (
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackRealSmokeReworkDescriptorV0(t *testing.T) {
	const (
		taskRef        = "task-ref-rework-unit-001"
		originalAckRef = "ack-ref-original-unit-001"
		reworkAckRef   = "ack-ref-rework-unit-001"
		reworkAgentRef = "agent-ref-retry-unit-001"
	)
	if strings.Contains(reworkAgentRef, "agent-ref-rework-request-ref-") {
		t.Fatalf("fixture no debe depender del prefijo legacy: %s", reworkAgentRef)
	}

	run := orquestacoreworkflow.OrchestrationRunV0{
		ReworkRequests: []string{
			"rework-request-ref-unit-001#review_result:review-result-ref-unit-001" +
				"#review_request:review-request-ref-unit-001#delivery:" + originalAckRef,
		},
		ReplanDecisions: []string{
			"replan-ref-unit-001#source:rework-request-ref-unit-001#task:" + taskRef +
				"#action:retry_task#followups:capacity-ref-unit-001+" + reworkAgentRef,
		},
		ReviewResults: []string{
			"review-result-ref-unit-001#review_result:changes_requested" +
				"#review_request:review-request-ref-unit-001#delivery:" + originalAckRef,
		},
		StartedAgents: []string{reworkAgentRef},
	}
	original := codexStackRealSmokeDescriptorForUnitV0("agent-ref-original-unit-001", taskRef, originalAckRef)
	rework := codexStackRealSmokeDescriptorForUnitV0(reworkAgentRef, taskRef, reworkAckRef)
	descriptors := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		codexStackRealSmokeDescriptorForUnitV0("agent-ref-other-task-unit-001", "task-ref-other-unit-001", "ack-ref-other-unit-001"),
		codexStackRealSmokeDescriptorForUnitV0("agent-ref-same-ack-unit-001", taskRef, originalAckRef),
		rework,
	}

	got := codexStackRealSmokeReworkDescriptorV0(t, run, descriptors, original)
	if got.AgentRef != reworkAgentRef ||
		got.Spec.AgentPacket.Task.TaskRef != taskRef ||
		got.Spec.AgentPacket.DeliveryRefs.AckRef != reworkAckRef {
		t.Fatalf("descriptor=%+v", got)
	}
}

func codexStackRealSmokeDescriptorForUnitV0(
	agentRef string,
	taskRef string,
	ackRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		AgentRef: agentRef,
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID: agentRef,
			AgentPacket: orquestaruntime.AgentStartPacketV0{
				Phase: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Task: orquestaruntime.AgentStartTaskV0{
					TaskRef: taskRef,
				},
				DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
					AckRef: ackRef,
				},
			},
		},
	}
}
