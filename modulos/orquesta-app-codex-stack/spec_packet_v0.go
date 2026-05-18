package orquestaappcodexstack

import (
	"strings"

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
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: correlationID,
		WorkOrderRef:  task.TaskRef,
		TargetModule:  "orquesta-app-stack-" + area,
		Phase:         phase,
		CapacityLevel: packetCapacityLevelV0(area, task),
		Locale:        "es-ES",
		Task:          task,
		Context:       contextBundleV0(area, task.TaskRef),
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-app-stack-" + deliverySuffix,
			AckRef:       "ack-ref-app-stack-" + deliverySuffix,
			ReadinessRef: "readiness-ref-app-stack-" + deliverySuffix,
		},
		Policies: []string{"write_set_closed", "ack_required", "context_small_by_refs"},
	}
}

func packetCapacityLevelV0(area string, task orquestaruntime.AgentStartTaskV0) string {
	if area == "director" && strings.Contains(task.Objective, "Modo de ejecucion: normal.") {
		return "xhigh"
	}
	if taskRequiresXHighPacketCapacityV0(task) {
		return "xhigh"
	}
	return "high"
}

func taskRequiresXHighPacketCapacityV0(task orquestaruntime.AgentStartTaskV0) bool {
	value := strings.ToLower(strings.Join(append(
		[]string{task.TaskRef, task.Title, task.Objective},
		task.WriteSet...,
	), "\n"))
	for _, marker := range []string{
		"plan_temario",
		"plan_tema",
		"plan_documento",
		"document_plan",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func contextBundleV0(area string, taskRef string) orquestacontext.ContextMaterializedBundleV0 {
	contextSuffix := packetRefSuffixV0(taskRef, area)
	return orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-app-stack-" + contextSuffix,
		WorkOrderRef:  taskRef,
		TargetModule:  "orquesta-app-stack-" + area,
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:  "entry-ref-app-stack-" + contextSuffix,
			Layer:     orquestacontext.ContextLayerTaskContextV0,
			Kind:      orquestacontext.ContextEntryDocRefV0,
			SourceRef: "source-ref-app-stack-" + contextSuffix,
			Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:  true,
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
