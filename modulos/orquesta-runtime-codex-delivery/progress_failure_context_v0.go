package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const maxCodexProgressFailureLogBytesV0 = 64 * 1024

func codexProgressReportWithProcessFailureContextV0(
	descriptor CodexReceiptDescriptorV0,
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	if report.Status != orquestaruntime.AgentStoppedV0 {
		return report
	}
	failure := codexProgressFailureClassFromDescriptorV0(descriptor)
	switch failure {
	case "quota_exhausted":
		report.Summary = "Proceso detenido sin ACK por cuota externa agotada."
		report.BudgetReason = "Cuota externa agotada antes de entregar ACK."
		report.DecisionRequired = true
		report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
			report.EvidenceRefs,
			"evidence-ref-codex-quota-exhausted",
		))
	default:
		if strings.TrimSpace(report.Summary) == "" {
			report.Summary = "Proceso detenido sin ACK."
		}
	}
	return report
}

func codexProgressFailureClassFromDescriptorV0(
	descriptor CodexReceiptDescriptorV0,
) string {
	stderr, ok := codexProgressReadFailureLogV0(
		filepath.Join(
			filepath.Dir(strings.TrimSpace(descriptor.AckPath)),
			orquestaruntimecodex.CodexStderrFileNameV0,
		),
	)
	if !ok {
		return ""
	}
	normalized := strings.ToLower(stderr)
	if strings.Contains(normalized, "usage limit") ||
		strings.Contains(normalized, "quota") ||
		strings.Contains(normalized, "credits") {
		return "quota_exhausted"
	}
	return ""
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
