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

func TestAgentPacketV0DirectorNormalUsaXHigh(t *testing.T) {
	packet := agentPacketV0(
		"agent-ref-director-001",
		"corr-spec-packet-director-001",
		"brainstorming_arquitectura",
		"director",
		orquestaruntime.AgentStartTaskV0{
			TaskRef:   "task-director-001",
			Objective: "Modo de ejecucion: normal.",
		},
	)

	if packet.CapacityLevel != "xhigh" {
		t.Fatalf("capacity=%q", packet.CapacityLevel)
	}
}

func TestAgentPacketV0PlanTemarioUsaXHigh(t *testing.T) {
	packet := agentPacketV0(
		"agent-ref-plan-temario-001",
		"corr-spec-packet-plan-temario-001",
		"programacion",
		"programacion",
		orquestaruntime.AgentStartTaskV0{
			TaskRef:  "task-plan-temario-001",
			Title:    "Planificar temario",
			WriteSet: []string{"external/opes/plan_temario/job-ref-plan-operadores-001"},
		},
	)

	if packet.CapacityLevel != "xhigh" {
		t.Fatalf("capacity=%q", packet.CapacityLevel)
	}
}
