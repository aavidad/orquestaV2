package orquestaappcodexstack

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func agentPacketV0(
	agentRef string,
	correlationID string,
	phase string,
	area string,
	task orquestaruntime.AgentStartTaskV0,
) orquestaruntime.AgentStartPacketV0 {
	phase = firstPacketValueV0(phase, "brainstorming_arquitectura")
	deliverySuffix := packetRefSuffixV0(agentRef, area)
	capacityPolicy := packetCapacityPolicyV0(area, task)
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: correlationID,
		WorkOrderRef:  task.TaskRef,
		TargetModule:  "orquesta-app-stack-" + area,
		Phase:         phase,
		CapacityLevel: capacityPolicy.CapacityLevel,
		Locale:        "es-ES",
		Task:          task,
		Context:       contextBundleV0(area, task.TaskRef),
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-app-stack-" + deliverySuffix,
			AckRef:       "ack-ref-app-stack-" + deliverySuffix,
			ReadinessRef: "readiness-ref-app-stack-" + deliverySuffix,
		},
		Policies: packetPoliciesV0(capacityPolicy),
	}
}

func packetCapacityLevelV0(area string, task orquestaruntime.AgentStartTaskV0) string {
	return packetCapacityPolicyV0(area, task).CapacityLevel
}

func packetCapacityPolicyV0(
	area string,
	task orquestaruntime.AgentStartTaskV0,
) orquestaautoprogramming.CapacityReasoningPolicyDecisionV0 {
	return orquestaautoprogramming.DecideCapacityReasoningPolicyV0(
		orquestaautoprogramming.CapacityReasoningPolicyInputV0{
			DomainRefs: []string{area},
			TaskRef:    task.TaskRef,
			Title:      task.Title,
			Objective:  task.Objective,
			WriteSet:   task.WriteSet,
		},
	)
}

func packetPoliciesV0(
	capacityPolicy orquestaautoprogramming.CapacityReasoningPolicyDecisionV0,
) []string {
	policies := []string{
		"write_set_closed",
		"ack_required",
		"context_small_by_refs",
		"capacity_policy_ref:" + capacityPolicy.PolicyRef,
	}
	for _, ref := range capacityPolicy.EvidenceRefs {
		policies = append(policies, "capacity_policy_evidence_ref:"+ref)
	}
	return policies
}

func contextBundleV0(area string, taskRef string) orquestacontext.ContextMaterializedBundleV0 {
	contextSuffix := packetRefSuffixV0(taskRef, area)
	return orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-app-stack-" + contextSuffix,
		WorkOrderRef:  taskRef,
		TargetModule:  "orquesta-app-stack-" + area,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:          "entry-ref-app-stack-" + contextSuffix,
			Layer:             orquestacontext.ContextLayerTaskContextV0,
			Kind:              orquestacontext.ContextEntryDocRefV0,
			SourceRef:         "source-ref-app-stack-" + contextSuffix,
			Mode:              orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:          true,
			RefOnlyReason:     orquestacontext.ContextRefOnlyReasonMaterializationMissingV0,
			RequiredRefAction: orquestacontext.ContextRequiredRefActionAckEvidenceV0,
		}},
	}
}

func firstPacketValueV0(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func packetRefSuffixV0(values ...string) string {
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), "-")
		if value != "" {
			return value
		}
	}
	return "sin-ref"
}
