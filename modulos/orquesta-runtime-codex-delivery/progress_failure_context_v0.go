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

type codexProgressFailureClassV0 string

const (
	codexProgressFailureNoACKV0           codexProgressFailureClassV0 = "no_ack"
	codexProgressFailureInterruptedV0     codexProgressFailureClassV0 = "interrupted"
	codexProgressFailureCapacityWarningV0 codexProgressFailureClassV0 = "capacity_warning"
)

func codexProgressReportWithProcessFailureContextV0(
	descriptor CodexReceiptDescriptorV0,
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	materialized := codexProgressWriteSetMaterializedV0(descriptor)
	if report.Status != orquestaruntime.AgentStoppedV0 {
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
	case codexProgressFailureCapacityWarningV0:
		report.Summary = "Proceso detenido por aviso de capacidad externa; requiere relevo."
		report.BudgetStatus = orquestaruntime.AgentProgressBudgetCapacityLimitedV0
		report.BudgetReason = "Aviso de capacidad externa antes de ACK."
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-capacity-warning",
			"evidence-ref-capacity-limited",
		))
	case codexProgressFailureInterruptedV0:
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
	case codexProgressFailureNoACKV0:
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
	logs := codexProgressFailureLogsFromDescriptorV0(descriptor)
	for _, log := range logs {
		if codexProgressTextHasCapacitySignalV0(log) {
			return codexProgressFailureCapacityWarningV0
		}
	}
	for _, log := range logs {
		if codexProgressTextHasInterruptedNoACKSignalV0(log) {
			return codexProgressFailureInterruptedV0
		}
	}
	return codexProgressFailureNoACKV0
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

func codexProgressTextHasCapacitySignalV0(text string) bool {
	normalized := strings.ToLower(text)
	return strings.Contains(normalized, "usage limit") ||
		strings.Contains(normalized, "quota") ||
		strings.Contains(normalized, "at capacity") ||
		strings.Contains(normalized, "capacity") ||
		strings.Contains(normalized, "credits")
}

func codexProgressTextHasInterruptedNoACKSignalV0(text string) bool {
	normalized := strings.ToLower(text)
	return strings.Contains(normalized, "turn interrupted") ||
		strings.Contains(normalized, "turn was interrupted") ||
		strings.Contains(normalized, "tokens used")
}

func codexProgressReadFailureLogV0(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return "", false
	}
	if len(data) > maxCodexProgressFailureLogBytesV0 {
		data = data[len(data)-maxCodexProgressFailureLogBytesV0:]
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
