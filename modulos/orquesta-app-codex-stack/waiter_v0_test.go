package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func TestDescriptorsHaveReadyAckV0AceptaACKFallidoComoTerminal(t *testing.T) {
	descriptor := waiterDescriptorForTestV0(t)
	writeWaiterAckWithStatusForTestV0(t, descriptor, "failed")

	ready, err := descriptorsHaveReadyAckV0(
		[]string{descriptor.AgentRef},
		[]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{descriptor},
	)
	if err != nil || !ready {
		t.Fatalf("failed ack terminal debe desbloquear wait: ready=%v err=%v", ready, err)
	}
}

func TestWaiterStartedAgentRefsForWaitV0FiltraCohorte(t *testing.T) {
	got := waiterStartedAgentRefsForWaitV0(
		[]string{"agent-old", "agent-new", "agent-new"},
		[]string{"agent-new"},
	)
	if len(got) != 1 || got[0] != "agent-new" {
		t.Fatalf("got=%v", got)
	}
}

func TestAckAwareWaiterV0WaitAgentRefsNoEsperaAgenteViejoVivo(t *testing.T) {
	newDescriptor := waiterDescriptorForAgentForTestV0(t, "agent-new")
	writeWaiterAckForTestV0(t, newDescriptor)
	waiter := AckAwareWaiterV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			newDescriptor,
		),
		Interval: time.Millisecond,
	}

	ready, err := waiter.allStartedAgentsHaveAckV0(
		context.Background(),
		orquestacionnucleoapp.ExternalProgressWaitRequestV0{
			RunRef:        newDescriptor.RunID,
			WaitAgentRefs: []string{"agent-new"},
			LastResult: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Run: orquestacoreworkflow.OrchestrationRunV0{
					StartedAgents: []string{"agent-old", "agent-new"},
				},
			},
		},
	)
	if err != nil || !ready {
		t.Fatalf("cohorte nueva con ACK no debe quedar bloqueada por agente viejo: ready=%v err=%v", ready, err)
	}
}

func TestAckAwareWaiterV0WaitAgentRefsVacioMantieneEsperaLegacy(t *testing.T) {
	oldDescriptor := waiterDescriptorForAgentForTestV0(t, "agent-old")
	newDescriptor := waiterDescriptorForAgentForTestV0(t, "agent-new")
	writeWaiterAckForTestV0(t, newDescriptor)
	waiter := AckAwareWaiterV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(
			oldDescriptor,
			newDescriptor,
		),
		Interval: time.Millisecond,
	}
	request := orquestacionnucleoapp.ExternalProgressWaitRequestV0{
		RunRef: newDescriptor.RunID,
		LastResult: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				StartedAgents: []string{"agent-old", "agent-new"},
			},
		},
	}

	ready, err := waiter.allStartedAgentsHaveAckV0(context.Background(), request)
	if err != nil || ready {
		t.Fatalf("WaitAgentRefs vacio debe esperar todos los agentes arrancados: ready=%v err=%v", ready, err)
	}
	writeWaiterAckForTestV0(t, oldDescriptor)
	ready, err = waiter.allStartedAgentsHaveAckV0(context.Background(), request)
	if err != nil || !ready {
		t.Fatalf("WaitAgentRefs vacio debe continuar cuando todos tienen ACK: ready=%v err=%v", ready, err)
	}
}

func waiterDescriptorForTestV0(t *testing.T) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	return waiterDescriptorForAgentForTestV0(t, "agent-ref-waiter-001")
}

func waiterDescriptorForAgentForTestV0(
	t *testing.T,
	agentRef string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
	spec := waiterSpecForTestV0()
	spec.RequestID = agentRef
	spec.AgentPacket.RequestID = agentRef
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef:  "descriptor-ref-waiter-" + agentRef,
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
	writeWaiterAckWithStatusForTestV0(t, descriptor, "completed")
}

func writeWaiterAckWithStatusForTestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	status string,
) {
	t.Helper()
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     descriptor.Spec.RequestID,
		CorrelationID: descriptor.Spec.CorrelationID,
		AckRef:        descriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  descriptor.Spec.AgentPacket.TargetModule,
		TaskRef:       descriptor.Spec.AgentPacket.Task.TaskRef,
		Status:        status,
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
