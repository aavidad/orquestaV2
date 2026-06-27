package orquestaruntimecodexdelivery

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const maxCodexProgressFailureLogBytesV0 = 64 * 1024

const codexProgressWarningStreamFDEvidenceRefV0 = "evidence-ref-warning-stream-fd"

type codexProgressFailureClassV0 string

const (
	CodexProgressFailureNoACKV0                  codexProgressFailureClassV0 = "no_ack"
	CodexProgressFailureInterruptedV0            codexProgressFailureClassV0 = "interrupted"
	CodexProgressFailureCapacityWarningV0        codexProgressFailureClassV0 = "capacity_warning"
	CodexProgressFailureAuthInvalidV0            codexProgressFailureClassV0 = "auth_invalid"
	CodexProgressFailureProviderQuotaExhaustedV0 codexProgressFailureClassV0 = "provider_quota_exhausted"

	codexProgressFailureNoACKV0                  = CodexProgressFailureNoACKV0
	codexProgressFailureInterruptedV0            = CodexProgressFailureInterruptedV0
	codexProgressFailureCapacityWarningV0        = CodexProgressFailureCapacityWarningV0
	codexProgressFailureAuthInvalidV0            = CodexProgressFailureAuthInvalidV0
	codexProgressFailureProviderQuotaExhaustedV0 = CodexProgressFailureProviderQuotaExhaustedV0
)

func codexProgressReportWithProcessFailureContextV0(
	descriptor CodexReceiptDescriptorV0,
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	return CodexProgressReportWithProcessFailureContextV0(descriptor, report)
}

func CodexProgressReportWithProcessFailureContextV0(
	descriptor CodexReceiptDescriptorV0,
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	materialized := codexProgressWriteSetMaterializedV0(descriptor)
	streamFDWarning := codexProgressDescriptorHasStreamFDWarningV0(descriptor)
	if report.Status != orquestaruntime.AgentStoppedV0 {
		if streamFDWarning {
			report = codexProgressReportWithStreamFDWarningV0(report)
		}
		if materialized && codexProgressStatusNeedsMaterializedArtifactReviewV0(report.Status) {
			report.DecisionRequired = true
			report.Summary = "Proceso sin ACK pero con artefacto en write-set; requiere validacion antes de replanificar."
			report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
				report.EvidenceRefs,
				"evidence-ref-no-ack",
				"evidence-ref-artifact-without-ack",
			))
		}
		return report
	}
	failure := codexProgressFailureClassFromDescriptorV0(descriptor)
	switch failure {
	case CodexProgressFailureAuthInvalidV0:
		report.Summary = "Autenticacion externa invalida; requiere reautorizacion del operador."
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-auth-config-blocker",
		))
	case CodexProgressFailureCapacityWarningV0:
		report.Summary = "Proceso detenido por aviso de capacidad externa; requiere relevo."
		report.BudgetStatus = orquestaruntime.AgentProgressBudgetCapacityLimitedV0
		report.BudgetReason = "Aviso de capacidad externa antes de ACK."
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-capacity-warning",
			"evidence-ref-capacity-limited",
		))
	case CodexProgressFailureProviderQuotaExhaustedV0:
		report.Summary = "Cuota externa agotada antes de ACK; cerrar agente y replanificar automaticamente cuando haya capacidad."
		report.BudgetStatus = orquestaruntime.AgentProgressBudgetCapacityLimitedV0
		report.BudgetReason = "Cuota externa agotada antes de ACK."
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-provider-quota-exhausted",
			"evidence-ref-capacity-limited",
		))
	case CodexProgressFailureInterruptedV0:
		if materialized {
			report.Summary = "Proceso detenido sin ACK tras interrupcion compacta pero con artefacto en write-set; requiere validacion."
		} else {
			report.Summary = "Proceso detenido sin ACK tras interrupcion compacta."
		}
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-no-ack",
			"evidence-ref-no-ack-interrupted",
		))
	case CodexProgressFailureNoACKV0:
		if materialized {
			report.Summary = "Proceso detenido sin ACK pero con artefacto en write-set; requiere validacion."
		} else {
			report.Summary = "Proceso detenido sin ACK."
		}
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-no-ack",
		))
	}
	if materialized {
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-artifact-without-ack",
		))
	}
	if streamFDWarning {
		report = codexProgressReportWithStreamFDWarningV0(report)
	}
	return report
}

func codexProgressStatusNeedsMaterializedArtifactReviewV0(
	status orquestaruntime.AgentProgressStatusV0,
) bool {
	return status == orquestaruntime.AgentStalledV0 ||
		status == orquestaruntime.AgentLoopDetectedV0
}

func codexProgressFailureClassFromDescriptorV0(
	descriptor CodexReceiptDescriptorV0,
) codexProgressFailureClassV0 {
	return CodexProgressFailureClassFromDescriptorV0(descriptor)
}

func CodexProgressFailureClassFromDescriptorV0(
	descriptor CodexReceiptDescriptorV0,
) codexProgressFailureClassV0 {
	if codexProgressDescriptorHasProviderQuotaExhaustedV0(descriptor) {
		return CodexProgressFailureProviderQuotaExhaustedV0
	}
	logs := codexProgressFailureLogsFromDescriptorV0(descriptor)
	for _, log := range logs {
		if codexProgressTextHasInterruptedNoACKSignalV0(log) {
			return CodexProgressFailureInterruptedV0
		}
	}
	return CodexProgressFailureNoACKV0
}

func codexProgressFailureLogsFromDescriptorV0(
	descriptor CodexReceiptDescriptorV0,
) []string {
	ackPath := strings.TrimSpace(descriptor.AckPath)
	if ackPath == "" {
		return nil
	}
	dir := filepath.Dir(ackPath)
	names := []string{
		orquestaruntimecodex.CodexStderrFileNameV0,
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
	}
	logs := make([]string, 0, len(names))
	for _, name := range names {
		log, ok := codexProgressReadFailureLogV0(filepath.Join(dir, name))
		if ok {
			logs = append(logs, log)
		}
	}
	return logs
}

func codexProgressDescriptorHasProviderQuotaExhaustedV0(
	descriptor CodexReceiptDescriptorV0,
) bool {
	ackPath := strings.TrimSpace(descriptor.AckPath)
	if ackPath == "" {
		return false
	}
	path := filepath.Join(
		filepath.Dir(ackPath),
		orquestaruntimecodex.CodexUsageAccountingFileNameV0,
	)
	data, ok := codexProgressReadTailV0(path, maxCodexProgressFailureLogBytesV0)
	if !ok {
		return false
	}
	snapshot := orquestaruntimecodex.BuildCodexUsageAccountingSnapshotV0([]string{string(data)})
	return snapshot.Observed &&
		snapshot.QuotaStatus == orquestaruntimecodex.CodexUsageQuotaExhaustedV0
}

func codexProgressDescriptorHasStreamFDWarningV0(
	descriptor CodexReceiptDescriptorV0,
) bool {
	for _, log := range codexProgressFailureLogsFromDescriptorV0(descriptor) {
		if codexProgressTextHasStreamFDWarningSignalV0(log) {
			return true
		}
	}
	return false
}

func codexProgressTextHasStreamFDWarningSignalV0(text string) bool {
	normalized := strings.ToLower(text)
	return strings.Contains(normalized, "failed to create stream fd")
}

func codexProgressReportWithStreamFDWarningV0(
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
		report.EvidenceRefs,
		codexProgressWarningStreamFDEvidenceRefV0,
	))
	return report
}

func codexProgressTextHasInterruptedNoACKSignalV0(text string) bool {
	normalized := strings.ToLower(text)
	return strings.Contains(normalized, "turn interrupted") ||
		strings.Contains(normalized, "turn was interrupted") ||
		strings.Contains(normalized, "tokens used")
}

func codexProgressReadFailureLogV0(path string) (string, bool) {
	data, ok := codexProgressReadTailV0(path, maxCodexProgressFailureLogBytesV0)
	if !ok {
		return "", false
	}
	return string(data), true
}

func codexProgressWriteSetMaterializedV0(
	descriptor CodexReceiptDescriptorV0,
) bool {
	projectDir := strings.TrimSpace(descriptor.ProjectWorkDir)
	if projectDir == "" {
		return false
	}
	for _, raw := range descriptor.Spec.AgentPacket.Task.WriteSet {
		rel, ok := cleanCodexReceiptAckFilePathV0(raw)
		if !ok {
			continue
		}
		if codexProgressPathMaterializedV0(filepath.Join(projectDir, filepath.FromSlash(rel))) {
			return true
		}
	}
	return false
}

func codexProgressPathMaterializedV0(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return info.Size() > 0
	}
	found := false
	visited := 0
	_ = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		visited++
		if visited > 512 {
			return filepath.SkipAll
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err == nil && info.Size() > 0 {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
