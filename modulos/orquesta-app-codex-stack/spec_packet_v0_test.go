package orquestaappcodexstack

import (
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestAgentPacketV0UsaDeliveryRefsUnicasPorAgente(t *testing.T) {
	first := agentPacketV0(
		"agent-ref-task-agenda-api-001",
		"corr-spec-packet-001",
		"programacion",
		"programacion",
		orquestaruntime.AgentStartTaskV0{TaskRef: "task-agenda-api-001"},
	)
	second := agentPacketV0(
		"agent-ref-task-agenda-web-001",
		"corr-spec-packet-001",
		"programacion",
		"programacion",
		orquestaruntime.AgentStartTaskV0{TaskRef: "task-agenda-web-001"},
	)
	if first.DeliveryRefs.AckRef == second.DeliveryRefs.AckRef {
		t.Fatalf("ack refs duplicadas: %s", first.DeliveryRefs.AckRef)
	}
	if first.DeliveryRefs.MailboxRef == second.DeliveryRefs.MailboxRef {
		t.Fatalf("mailbox refs duplicadas: %s", first.DeliveryRefs.MailboxRef)
	}
	if first.Context.BundleRef == second.Context.BundleRef {
		t.Fatalf("context bundle refs duplicadas: %s", first.Context.BundleRef)
	}
}
