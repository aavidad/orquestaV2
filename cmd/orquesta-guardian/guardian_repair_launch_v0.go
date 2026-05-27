package main

import (
	"strings"
)

func buildGuardianRepairLaunchPacketV0(
	config guardianConfigV0,
	result guardianResultV0,
	repairPacketPath string,
	inspection guardianRepairPacketInspectionV0,
) guardianRepairLaunchPacketV0 {
	budget := inspection.Packet.RepairBudget
	if budget.MaxAgents <= 0 {
		budget = guardianRepairBudgetV0{MaxAgents: 1, MaxAttempts: config.RepairMaxAttempts}
	}
	return guardianRepairLaunchPacketV0{
		SchemaVersion:      guardianRepairLaunchPacketSchemaVersionV0,
		Reason:             "break_glass",
		FailurePhase:       result.Phase,
		Objective:          guardianRepairObjectiveV0(result.Phase),
		WriteSet:           append([]string(nil), config.RepairCodexWriteSet...),
		RequiredTests:      append([]string(nil), config.RepairCodexRequiredTests...),
		AcceptanceCriteria: guardianRepairAcceptanceCriteriaV0(),
		ManifestRef:        guardianManifestRefV0(config, result),
		RepairPacketRef:    guardianRepairPacketRefV0(config, guardianResultV0{Phase: result.Phase, RepairPacketPath: repairPacketPath}),
		RepairAttemptRef:   strings.TrimSpace(inspection.Packet.RepairAttemptRef),
		FailurePacketHash:  strings.TrimSpace(inspection.Packet.FailurePacketHash),
		RunRef:             strings.TrimSpace(config.RepairCodexRunRef),
		PromotionRef:       strings.TrimSpace(config.RepairCodexPromotionRef),
		Budget:             budget,
		ACKContract: guardianRepairACKContractV0{
			SchemaVersion: "codex_agent_ack.v0",
			Required:      true,
			TerminalFile:  "agent_ack.json",
		},
		SandboxPolicy: guardianRepairSandboxV0{
			Sandbox:                 strings.TrimSpace(config.RepairCodexSandbox),
			BroadSandboxAllowed:     config.RepairCodexAllowBroadSandbox,
			BroadSandboxEvidenceRef: strings.TrimSpace(config.RepairCodexSandboxEvidenceRef),
		},
		EvidenceRefs: guardianRepairLaunchEvidenceRefsV0(config),
		CreatedAt:    config.OccurredAt.Format("2006-01-02T15:04:05Z"),
	}
}

func guardianRepairObjectiveV0(phase string) string {
	return strings.Join([]string{
		"Reparar candidato de Orquesta fallido en fase " + strings.TrimSpace(phase) + ".",
		"Trabajar solo dentro del write-set cerrado del packet de lanzamiento.",
		"Entregar agent_ack.json estructurado con pruebas requeridas y evidencia compacta.",
		"Si falta alcance, permisos o contexto, cerrar ACK failed con CONSULTA AL DIRECTOR.",
	}, " ")
}

func guardianRepairAcceptanceCriteriaV0() []string {
	return []string{
		"write-set cerrado respetado",
		"agent_ack.json estructurado escrito con status terminal",
		"pruebas requeridas ejecutadas o bloqueo explicado con CONSULTA AL DIRECTOR",
		"sin stdout, prompt o exit code como evidencia unica de cierre",
	}
}

func guardianRepairLaunchEvidenceRefsV0(config guardianConfigV0) []string {
	refs := []string{
		"evidence-ref-guardian-repair-break-glass",
		"evidence-ref-guardian-repair-write-set-closed",
		"evidence-ref-guardian-repair-ack-required",
	}
	if ref := strings.TrimSpace(config.RepairCodexSandboxEvidenceRef); ref != "" {
		refs = append(refs, ref)
	}
	return compactStringsV0(refs)
}

func guardianRepairCodexSandboxBroadV0(value string) bool {
	switch strings.TrimSpace(value) {
	case "danger-full-access", "full-access", "no-sandbox":
		return true
	default:
		return false
	}
}
