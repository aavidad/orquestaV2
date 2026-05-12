package orquestaruntimecodexdelivery

import (
	"context"
	"testing"
)

func TestCodexDeliveryObservationSourceV0OmiteACKFailedSinBloquearOtros(t *testing.T) {
	failedSpec := codexDeliverySpecForTestV0()
	failedSpec.RequestID = "agent-ref-failed-001"
	failedSpec.AgentPacket.RequestID = failedSpec.RequestID
	failedSpec.AgentPacket.Task.TaskRef = "task-ref-failed-001"
	failedSpec.AgentPacket.DeliveryRefs.AckRef = "ack-ref-failed-001"
	completedSpec := codexDeliverySpecForTestV0()
	completedSpec.RequestID = "agent-ref-completed-001"
	completedSpec.AgentPacket.RequestID = completedSpec.RequestID
	completedSpec.AgentPacket.Task.TaskRef = "task-ref-completed-001"
	completedSpec.AgentPacket.DeliveryRefs.AckRef = "ack-ref-completed-001"
	failedAck := codexDeliveryAckForTestV0(failedSpec)
	failedAck.Status = "failed"
	failedAck.Tests = nil
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{
			{
				DescriptorRef: "receipt-ref-failed-001",
				Spec:          failedSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, failedSpec, failedAck),
			},
			{
				DescriptorRef: "receipt-ref-completed-001",
				Spec:          completedSpec,
				AckPath:       writeCodexDeliveryAckForTestV0(t, completedSpec, codexDeliveryAckForTestV0(completedSpec)),
			},
		},
	}
	request := codexDeliveryRequestForTestV0(completedSpec, nil)
	request.Run.Agents = []string{failedSpec.RequestID, completedSpec.RequestID}
	request.Run.StartedAgents = []string{failedSpec.RequestID, completedSpec.RequestID}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 ||
		observations[0].AgentRef != completedSpec.RequestID ||
		observations[0].DeliveryRef != completedSpec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
}
