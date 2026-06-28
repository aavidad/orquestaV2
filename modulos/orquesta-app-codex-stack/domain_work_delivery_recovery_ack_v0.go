package orquestaappcodexstack

import (
	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func domainWorkSubmissionAckV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	record orquestaappchange.AppChangeRecordV0,
) (orquestaruntimecodex.CodexAgentAckV0, bool) {
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(
		descriptor.AckPath,
		descriptor.Spec,
	)
	if len(issues) == 0 {
		return ack, true
	}
	return recoverDomainWorkAckV0(descriptor, task, record)
}

func recoverDomainWorkAckV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	record orquestaappchange.AppChangeRecordV0,
) (orquestaruntimecodex.CodexAgentAckV0, bool) {
	if !workflowTaskHasDomainWorkContractV0(task) ||
		record.Request.ExternalWork == nil {
		return orquestaruntimecodex.CodexAgentAckV0{}, false
	}
	files, ok := domainWorkRecoveryArtifactFilesV0(
		descriptor.ProjectWorkDir,
		descriptor.Spec.AgentPacket.Task.WriteSet,
	)
	if !ok {
		return orquestaruntimecodex.CodexAgentAckV0{}, false
	}
	packet := descriptor.Spec.AgentPacket
	ack := orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     descriptor.Spec.RequestID,
		CorrelationID: descriptor.Spec.CorrelationID,
		AckRef:        packet.DeliveryRefs.AckRef,
		TargetModule:  packet.TargetModule,
		TaskRef:       packet.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0(files),
		Tests:         orquestaruntimecodex.EvidenceListV0(packet.Task.RequiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(packet.Task.RequiredTests),
		Notes: orquestaruntimecodex.EvidenceListV0{
			"domain_work_ack_recovered_from_valid_artifact",
			"contexto_truncado_resuelto: artefacto domain_work recuperado",
			"contexto_ref_only_resuelto: artefacto domain_work recuperado",
		},
	}
	if issues := orquestaruntimecodex.ValidateCodexAgentAckForSpecV0(
		ack,
		descriptor.Spec,
	); len(issues) > 0 {
		return orquestaruntimecodex.CodexAgentAckV0{}, false
	}
	return ack, true
}
