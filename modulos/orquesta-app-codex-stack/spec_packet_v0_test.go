package orquestaappcodexstack

import (
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
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

func TestAgentPacketV0DirectorNormalUsaMediumConPoliticaAuditable(t *testing.T) {
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

	if packet.CapacityLevel != "medium" ||
		!agentPacketHasPolicyForTestV0(packet, "capacity_policy_ref:capacity-policy-ref-background-medium-v0") ||
		!agentPacketHasPolicyForTestV0(packet, "capacity_policy_evidence_ref:evidence-ref-capacity-policy-ref-background-medium-v0") {
		t.Fatalf("capacity=%q", packet.CapacityLevel)
	}
}

func TestAgentPacketV0ProgramacionUsaHighConPoliticaAuditable(t *testing.T) {
	packet := agentPacketV0(
		"agent-ref-programacion-001",
		"corr-spec-packet-programacion-001",
		"programacion",
		"programacion",
		orquestaruntime.AgentStartTaskV0{
			TaskRef:   "task-programacion-001",
			Objective: "crear_app_completa server-first",
			WriteSet:  []string{"go.mod", "internal/modules/**"},
		},
	)

	if packet.CapacityLevel != "high" ||
		!agentPacketHasPolicyForTestV0(packet, "capacity_policy_ref:capacity-policy-ref-high-risk-v0") ||
		!agentPacketHasPolicyForTestV0(packet, "capacity_policy_evidence_ref:evidence-ref-capacity-policy-ref-high-risk-v0") {
		t.Fatalf("capacity=%q policies=%v", packet.CapacityLevel, packet.Policies)
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
	if !agentPacketHasPolicyForTestV0(packet, "capacity_policy_ref:capacity-policy-ref-opes-document-v0") {
		t.Fatalf("policies=%v", packet.Policies)
	}
}

func TestAgentPacketV0MarcaContextoRequiredRefOnly(t *testing.T) {
	packet := agentPacketV0(
		"agent-ref-context-ref-only-001",
		"corr-spec-packet-context-ref-only-001",
		"programacion",
		"programacion",
		orquestaruntime.AgentStartTaskV0{TaskRef: "task-context-ref-only-001"},
	)
	entry := packet.Context.Entries[0]

	if entry.Mode != orquestacontext.ContextMaterializationModeRefOnlyV0 ||
		entry.RefOnlyReason != orquestacontext.ContextRefOnlyReasonMaterializationMissingV0 ||
		entry.RequiredRefAction != orquestacontext.ContextRequiredRefActionAckEvidenceV0 {
		t.Fatalf("context required ref_only sin guard: %+v", entry)
	}
}

func agentPacketHasPolicyForTestV0(packet orquestaruntime.AgentStartPacketV0, policy string) bool {
	for _, candidate := range packet.Policies {
		if candidate == policy {
			return true
		}
	}
	return false
}
