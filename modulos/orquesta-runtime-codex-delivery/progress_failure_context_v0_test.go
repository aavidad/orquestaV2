package orquestaruntimecodexdelivery

import (
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressReportWithProcessFailureContextV0ClasificaCuotaSinFiltrarProveedor(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("ERROR: You've hit your usage limit. Try again later."),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	report := validCodexProgressFailureReportForTestV0()

	got := codexProgressReportWithProcessFailureContextV0(
		CodexReceiptDescriptorV0{
			AckPath: filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
		report,
	)

	if !got.DecisionRequired ||
		got.BudgetReason != "Cuota externa agotada antes de entregar ACK." ||
		got.Summary != "Proceso detenido sin ACK por cuota externa agotada." {
		t.Fatalf("report=%+v", got)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(got); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
}

func validCodexProgressFailureReportForTestV0() orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:        "agent-progress-report-ref-quota-001",
		RunID:           "run-ref-quota-001",
		AgentRequestID:  "agent-ref-quota-001",
		Status:          orquestaruntime.AgentStoppedV0,
		Summary:         "Proceso observado como parado.",
		EvidenceRefs:    []string{"evidence-ref-quota-001"},
		NoProgressTicks: 1,
	}
}
