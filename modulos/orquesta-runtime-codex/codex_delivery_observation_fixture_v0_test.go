package orquestaruntimecodex

import orquestaruntime "orquesta/modulos/orquesta-runtime"

func codexNeutralSpecForDeliveryObservationTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	spec := codexSpecForTestV0()
	spec.RequestID = "agent-ref-001"
	spec.CorrelationID = "corr-agent-001"
	spec.AgentPacket.RequestID = spec.RequestID
	spec.AgentPacket.CorrelationID = spec.CorrelationID
	spec.AgentPacket.DeliveryRefs.AckRef = "ack-ref-001"
	spec.AgentPacket.DeliveryRefs.MailboxRef = "mailbox-ref-001"
	spec.AgentPacket.DeliveryRefs.ReadinessRef = "readiness-ref-001"
	spec.AgentPacket.DeliveryRefs.CheckpointRef = ""
	return spec
}

func codexNeutralAckForDeliveryObservationTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) CodexAgentAckV0 {
	return CodexAgentAckV0{
		SchemaVersion: CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        codexAgentAckStatusCompletedV0,
		Files:         EvidenceListV0{"README.md"},
		Tests:         EvidenceListV0{"go test ./..."},
		TestReceipts: []CodexRequiredTestReceiptV0{{
			SchemaVersion:  CodexRequiredTestReceiptSchemaVersionV0,
			Command:        "go test ./...",
			Status:         "passed",
			ExitCode:       codexDeliveryObservationIntPtrForTestV0(0),
			EvidenceRefs:   []string{"required-test-receipt-ref-001"},
			OccurredAt:     "2026-05-24T10:00:00Z",
			Sequence:       1,
			OutputRedacted: codexDeliveryObservationBoolPtrForTestV0(true),
		}},
	}
}

func codexDeliveryObservationIntPtrForTestV0(value int) *int {
	return &value
}

func codexDeliveryObservationBoolPtrForTestV0(value bool) *bool {
	return &value
}
