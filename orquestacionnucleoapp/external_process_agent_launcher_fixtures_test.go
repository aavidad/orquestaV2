package orquestacionnucleoapp

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func externalProcessLauncherInboundV0(
	runRef string,
	agentRequestID string,
) orquestaruntime.AgentLauncherInboundV0 {
	payload := orquestaruntime.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     agentRequestID,
		RunID:              runRef,
		PhaseID:            "programacion",
		TaskRef:            "task-ref-process-001",
		CapacityRequestRef: "capacity-ref-process-001",
		Role:               "implementacion",
		Summary:            "Orden compacta para validar lanzamiento de proceso.",
		EvidenceRefs:       []string{"evidence-ref-process-launch-001"},
	}
	return orquestaruntime.AgentLauncherInboundV0{
		TargetPort:     orquestaruntime.AgentLauncherTargetPortV0,
		MessageType:    orquestaruntime.AgentLauncherMessageTypeV0,
		CorrelationID:  "corr-process-launch-001",
		IdempotencyKey: "idem-process-launch-001",
		Payload:        &payload,
	}
}

func runtimeLaunchRequestForExternalProcessTestV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
	inbound orquestaruntime.AgentLauncherInboundV0,
) orquestaruntime.RuntimeLaunchRequestV0 {
	return orquestaruntime.RuntimeLaunchRequestV0{
		SchemaVersion:  orquestaruntime.RuntimeLaunchRequestSchemaVersionV0,
		RequestID:      payload.AgentRequestID,
		CorrelationID:  inbound.CorrelationID,
		IdempotencyKey: inbound.IdempotencyKey,
		RequestedAt:    "2026-05-08T11:01:00Z",
		Source: &orquestaruntime.RuntimeLaunchSourceV0{
			Module:     orquestaruntime.RuntimeLaunchSourceModuleCoreV0,
			AdapterRef: "nucleo-external-process-agent-launcher-v0",
		},
		Locale:           "es-ES",
		LaunchMode:       orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Task:             runtimeLaunchTaskForExternalProcessTestV0(payload),
		FunctionContract: runtimeFunctionContractForExternalProcessTestV0(),
		CapacityDecision: runtimeCapacityDecisionForExternalProcessTestV0(),
		RuntimeBinding:   runtimeBindingForExternalProcessTestV0(),
		EvidenceRefs:     runtimeEvidenceRefsForExternalProcessTestV0(payload),
		ContextBundle:    contextBundleForExternalProcessTestV0(),
		Delivery: &orquestaruntime.RuntimeDeliveryV0{
			MailboxProtocol:         orquestaruntime.DeliveryMailboxProtocolV0,
			AckRequired:             true,
			ReadinessTimeoutSeconds: 30,
			MaxStartupSeconds:       30,
		},
		Safety: &orquestaruntime.RuntimeSafetyV0{
			SecretsPolicy:   orquestaruntime.SafetyPolicyReferencesOnlyV0,
			HomePathsPolicy: orquestaruntime.SafetyPolicyOpaqueRefsOnlyV0,
			ProviderPolicy:  orquestaruntime.SafetyPolicyOpaqueRefsOnlyV0,
			WriteSetPolicy:  orquestaruntime.SafetyPolicyWriteSetClosedV0,
		},
	}
}

func runtimeLaunchTaskForExternalProcessTestV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) *orquestaruntime.RuntimeLaunchTaskV0 {
	return &orquestaruntime.RuntimeLaunchTaskV0{
		TaskRef:  payload.TaskRef,
		PhaseRef: payload.PhaseID,
		Priority: "normal",
	}
}

func runtimeFunctionContractForExternalProcessTestV0() *orquestaruntime.RuntimeFunctionContractV0 {
	return &orquestaruntime.RuntimeFunctionContractV0{
		SourceContract:  orquestaruntime.FunctionContractSourceV0,
		ContractRef:     "function-contract-process-001",
		ContractVersion: orquestaruntime.FunctionContractVersionV0,
		State:           orquestaruntime.FunctionContractStateActiveV0,
		Titulo:          "Validar lanzador externo de proceso",
		Objetivo:        "Arrancar un proceso controlado desde el nucleo.",
		ArchivoObjetivo: "orquestacionnucleoapp/external_process_agent_launcher_test.go",
		SimboloObjetivo: "ExternalProcessAgentLauncherV0",
		WriteSet: []string{
			"orquestacionnucleoapp/external_process_agent_launcher_test.go",
		},
		TestsObligatorios: []string{
			"go test -count=1 ./orquestacionnucleoapp",
		},
		CriterioCierre: []string{
			"LaunchAgentV0 arranca un proceso real por puerto inyectado.",
			"AgentLauncherExecutorV0 registra AgentStarted.",
		},
	}
}

func runtimeCapacityDecisionForExternalProcessTestV0() *orquestaruntime.RuntimeCapacityDecisionV0 {
	return &orquestaruntime.RuntimeCapacityDecisionV0{
		DecisionRef:     "capacity-decision-process-001",
		ContractVersion: orquestaruntime.CapacityDecisionVersionV0,
		NivelCapacidad:  "medium",
		ReasoningEffort: "medium",
		PoolRef:         "pool-ref-process-001",
		ModelRef:        "modelo-ref-process-001",
		QuotaRef:        "quota-ref-process-001",
	}
}

func runtimeBindingForExternalProcessTestV0() *orquestaruntime.RuntimeBindingV0 {
	return &orquestaruntime.RuntimeBindingV0{
		LogicalAgentRef: "logical-agent-process-001",
		RuntimeKind:     "cli",
		ConnectorRef:    "connector-ref-process-001",
		ProviderRef:     "supplier-ref-process-001",
		ModelRef:        "modelo-ref-process-001",
		HomeRef:         "agenthome-ref-process-001",
		CredentialKind:  "none",
		CredentialRef:   "credref-process-001",
	}
}

func runtimeEvidenceRefsForExternalProcessTestV0(
	payload orquestaruntime.LaunchRuntimeAgentRequestV0,
) *orquestaruntime.RuntimeEvidenceRefsV0 {
	return &orquestaruntime.RuntimeEvidenceRefsV0{
		MailboxRef:    "mailbox-ref-" + payload.AgentRequestID,
		AckRef:        "ack-ref-" + payload.AgentRequestID,
		ReadinessRef:  "readiness-ref-" + payload.AgentRequestID,
		CheckpointRef: "checkpoint-ref-" + payload.AgentRequestID,
	}
}

func contextBundleForExternalProcessTestV0() *orquestacontext.ContextBundleV0 {
	bundle := orquestacontext.BuildContextBundleV0(orquestacontext.ContextBundleRequestV0{
		SchemaVersion: orquestacontext.ContextBundleRequestSchemaVersionV0,
		BundleRef:     "context-bundle-process-001",
		WorkOrderRef:  "work-order-process-001",
		TargetModule:  "orquesta-runtime",
		Phase:         "programacion",
		TaskKind:      "microtarea_codigo",
		Objective:     "Validar lanzamiento controlado con contexto pequeno.",
		CapacityLevel: "medium",
		ReadSet:       []string{"agent_launcher_inbound_v0.go"},
		WriteSet:      []string{"orquestacionnucleoapp/external_process_agent_launcher_test.go"},
		ContractRefs:  []string{"ExternalAgentLaunchSpecV0", "AgentStartPacketV0"},
	})
	return &bundle
}

func materializedContextForExternalProcessTestV0(
	bundle orquestacontext.ContextBundleV0,
) orquestacontext.ContextMaterializedBundleV0 {
	entries := make([]orquestacontext.ContextMaterializedEntryV0, 0, len(bundle.Entries))
	for _, entry := range bundle.Entries {
		entries = append(entries, orquestacontext.ContextMaterializedEntryV0{
			EntryRef:  entry.EntryRef,
			Layer:     entry.Layer,
			Kind:      entry.Kind,
			SourceRef: entry.SourceRef,
			Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:  entry.Required,
		})
	}
	return orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     bundle.BundleRef,
		WorkOrderRef:  bundle.WorkOrderRef,
		TargetModule:  bundle.TargetModule,
		Entries:       entries,
	}
}

func externalAgentConnectorProfileForProcessTestV0() orquestaruntime.ExternalAgentConnectorProfileV0 {
	return orquestaruntime.BuildClosedExternalAgentConnectorProfileV0(orquestaruntime.ExternalAgentConnectorProfileRefsV0{
		ProfileRef:    "ext-prof-process-001",
		ConnectorRef:  "ext-conn-process-001",
		RuntimeKind:   "cli",
		CommandRef:    "cmdref-process-001",
		ExecutableRef: "execref-process-001",
		ArgRefs:       []string{"argref-process-mode-001"},
		EnvRefs:       []string{"envref-process-child-001"},
		WorkingDirRef: "workdirref-process-001",
	})
}

func hasEventTypeV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
