package orquestaappcodexstack

import (
	"path/filepath"
	"strings"

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
		record.Request.ExternalWork == nil ||
		!domainWorkRecoveryLastMessageConfirmsAckFailureV0(descriptor) {
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

func domainWorkRecoveryLastMessageConfirmsAckFailureV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) bool {
	ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	if ackRef == "" {
		return false
	}
	runtimeDir := filepath.Dir(strings.TrimSpace(descriptor.AckPath))
	if domainWorkRecoveryFileContainsV0(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexLastMessageFileNameV0),
		"ACK "+ackRef+" failed",
	) {
		return true
	}
	if domainWorkRecoveryFileContainsAnyV0(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]string{"turn interrupted", "turn was interrupted", "tokens used"},
	) {
		return true
	}
	return domainWorkRecoveryFileContainsV0(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		"patch rejected: writing outside of the project",
	)
}

func domainWorkRecoveryFileContainsV0(path string, needle string) bool {
	data, ok := codexStackReadTailFileV0(path, codexStackRuntimeLogTailMaxBytesV0)
	if !ok {
		return false
	}
	return strings.Contains(string(data), needle)
}

func domainWorkRecoveryFileContainsAnyV0(path string, needles []string) bool {
	data, ok := codexStackReadTailFileV0(path, codexStackRuntimeLogTailMaxBytesV0)
	if !ok {
		return false
	}
	normalized := strings.ToLower(string(data))
	for _, needle := range needles {
		if strings.Contains(normalized, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}
