package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexProgressReportWithRunningNoVisibleSignalV0(
	descriptor CodexReceiptDescriptorV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
	report orquestaruntime.AgentProgressReportV0,
) orquestaruntime.AgentProgressReportV0 {
	if snapshot.Status != orquestaruntime.ProcessRuntimeRunningV0 ||
		codexProgressDescriptorHasVisibleSignalV0(descriptor) {
		return report
	}
	if report.Status == orquestaruntime.AgentStalledV0 {
		report.Status = orquestaruntime.AgentProgressingV0
		report.Summary = "Proceso en ejecucion sin senal compacta observable."
	}
	switch report.BudgetStatus {
	case orquestaruntime.AgentProgressBudgetStalledV0:
		report.BudgetStatus = orquestaruntime.AgentProgressBudgetWorkingV0
		report.BudgetReason = "within_budget_process_running_no_visible_signal"
		report.DecisionRequired = false
	case orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0:
		report.BudgetStatus = orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0
		report.BudgetReason = "over_budget_process_running_no_visible_signal"
	}
	if report.BudgetStatus != orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0 {
		report.DecisionRequired = false
	}
	report.EvidenceRefs = compactCodexDeliveryRefsV0(append(
		report.EvidenceRefs,
		"evidence-ref-running-no-visible-signal",
	))
	return report
}

func codexProgressDescriptorHasVisibleSignalV0(
	descriptor CodexReceiptDescriptorV0,
) bool {
	ackPath := strings.TrimSpace(descriptor.AckPath)
	if ackPath == "" {
		return false
	}
	dir := filepath.Dir(ackPath)
	for _, name := range []string{
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexStderrFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		orquestaruntimecodex.CodexAgentAckFileNameV0,
	} {
		if codexProgressFileHasVisibleSignalV0(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

func codexProgressFileHasVisibleSignalV0(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
