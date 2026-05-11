package orquestaruntimecodexdelivery

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func codexDeliveryLoopSpecForTestV0(
	agentRef string,
	taskRef string,
) orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		SchemaVersion: orquestaruntime.ExternalAgentLaunchSpecSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-receipt-agent-001",
		ProfileRef:    "profile-ref-receipt-001",
		ConnectorRef:  "connector-ref-receipt-001",
		RuntimeKind:   "cli",
		LaunchMode:    orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Command: orquestaruntime.ExternalAgentCommandV0{
			CommandRef:    "command-ref-receipt-001",
			ExecutableRef: "exec-ref-receipt-001",
			WorkingDirRef: "workdir-ref-receipt-001",
		},
		AgentPacket: codexDeliveryLoopPacketForTestV0(agentRef, taskRef),
		Security: orquestaruntime.ExternalAgentSecurityPolicyV0{
			OptIn:                 true,
			ShellPolicy:           orquestaruntime.ExternalAgentShellForbiddenV0,
			PathInheritancePolicy: orquestaruntime.ExternalAgentPathInheritanceForbiddenV0,
			EnvironmentPolicy:     orquestaruntime.ExternalAgentEnvExplicitRefsOnlyV0,
			SecretsPolicy:         orquestaruntime.ExternalAgentSecretsReferencesOnlyV0,
			HomePathsPolicy:       orquestaruntime.ExternalAgentHomeOpaqueRefsOnlyV0,
			NetworkPolicy:         orquestaruntime.ExternalAgentNetworkClosedV0,
			TranscriptsPolicy:     orquestaruntime.ExternalAgentTranscriptsForbiddenV0,
		},
	}
}

func codexDeliveryLoopPacketForTestV0(
	agentRef string,
	taskRef string,
) orquestaruntime.AgentStartPacketV0 {
	return orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     agentRef,
		CorrelationID: "corr-receipt-agent-001",
		WorkOrderRef:  taskRef,
		TargetModule:  "agenda-app",
		Phase:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		CapacityLevel: "medium",
		Locale:        "es-ES",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef:       taskRef,
			Priority:      "normal",
			Title:         "Crear agenda",
			Objective:     "Crear agenda pequena.",
			TargetSymbol:  "Agenda",
			WriteSet:      []string{"README.md"},
			RequiredTests: []string{"go test ./..."},
			DoneCriteria:  []string{"Recibo compacto escrito."},
		},
		Context: orquestacontext.ContextMaterializedBundleV0{
			SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
			BundleRef:     "bundle-ref-receipt-001",
			WorkOrderRef:  taskRef,
			TargetModule:  "agenda-app",
			Entries: []orquestacontext.ContextMaterializedEntryV0{{
				EntryRef:  "entry-ref-receipt-001",
				Layer:     orquestacontext.ContextLayerTaskContextV0,
				Kind:      orquestacontext.ContextEntryDocRefV0,
				SourceRef: "source-ref-receipt-001",
				Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
				Required:  true,
			}},
		},
		DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
			MailboxRef:   "mailbox-ref-receipt-001",
			AckRef:       "ack-ref-receipt-001",
			ReadinessRef: "readiness-ref-receipt-001",
		},
		Policies: []string{"write_set_closed", "ack_required"},
	}
}
