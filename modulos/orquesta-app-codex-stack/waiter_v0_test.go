package orquestaappcodexstack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestDescriptorsHaveReadyAckV0ValidaContenidoNoSoloPath(t *testing.T) {
	descriptor := waiterDescriptorForTestV0(t)
	started := []string{descriptor.AgentRef}

	ready, err := descriptorsHaveReadyAckV0(started, []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{descriptor})
	if err != nil || ready {
		t.Fatalf("missing ack ready=%v err=%v", ready, err)
	}
	if err := os.WriteFile(descriptor.AckPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid ack: %v", err)
	}
	ready, err = descriptorsHaveReadyAckV0(started, []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{descriptor})
	if err == nil || ready {
		t.Fatalf("invalid ack ready=%v err=%v", ready, err)
	}
	writeWaiterAckForTestV0(t, descriptor)
	ready, err = descriptorsHaveReadyAckV0(started, []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{descriptor})
	if err != nil || !ready {
		t.Fatalf("valid ack ready=%v err=%v", ready, err)
	}
}

func waiterDescriptorForTestV0(t *testing.T) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	spec := waiterSpecForTestV0()
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  "descriptor-ref-waiter-001",
		RunID:          "run-ref-waiter-001",
		AgentRef:       spec.RequestID,
		Spec:           spec,
		AckPath:        filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0),
		ProjectWorkDir: t.TempDir(),
	}
}

func waiterSpecForTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-waiter-001",
		CorrelationID: "corr-waiter-001",
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			RequestID:     "agent-ref-waiter-001",
			CorrelationID: "corr-waiter-001",
			TargetModule:  "orquesta-app-stack-waiter",
			Phase:         "brainstorming_arquitectura",
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:  "task-ref-waiter-001",
				WriteSet: []string{"docs/waiter.md"},
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				AckRef:       "ack-ref-waiter-001",
				MailboxRef:   "mailbox-ref-waiter-001",
				ReadinessRef: "readiness-ref-waiter-001",
			},
		},
	}
}

func writeWaiterAckForTestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) {
	t.Helper()
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     descriptor.Spec.RequestID,
		CorrelationID: descriptor.Spec.CorrelationID,
		AckRef:        descriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  descriptor.Spec.AgentPacket.TargetModule,
		TaskRef:       descriptor.Spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		Files:         descriptor.Spec.AgentPacket.Task.WriteSet,
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(descriptor.AckPath, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}
