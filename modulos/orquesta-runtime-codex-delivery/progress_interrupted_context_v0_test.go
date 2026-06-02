package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressObservationSourceV0ClasificaInterrupcionSinACKComoStopped(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	runtimeDir := filepath.Dir(ackPath)
	localPath := filepath.Join(runtimeDir, "home", "project")
	writeCodexProgressRuntimeFileForTestV0(
		t,
		ackPath,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		"turn interrupted\n42 tokens used\nHOME="+localPath+"\nmodel=external\nprovider=external\n",
	)

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("observations=%+v", got)
	}
	report := got[0].Report
	if report.Status != orquestaruntime.AgentStoppedV0 ||
		report.BudgetStatus == orquestaruntime.AgentProgressBudgetCapacityLimitedV0 ||
		!got[0].DecisionRequired ||
		!report.DecisionRequired {
		t.Fatalf("report interrupcion inesperado: %+v observation=%+v", report, got[0])
	}
	if !stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-no-ack-interrupted") {
		t.Fatalf("evidence refs sin no_ack compacto: %+v", report.EvidenceRefs)
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(report); len(issues) > 0 {
		t.Fatalf("progress report invalido: %+v", issues)
	}
	assertCodexProgressObservationNoLeaksV0(
		t,
		got[0],
		ackPath,
		runtimeDir,
		localPath,
		"HOME=",
		"model=",
		"provider=",
		"turn interrupted",
		"tokens used",
	)
}

func TestCodexProgressObservationSourceV0NoConvierteTextoCapacidadEnCapacityLimited(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	writeCodexProgressRuntimeFileForTestV0(
		t,
		ackPath,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		"turn interrupted\n42 tokens used\nselected service is at capacity\n",
	)

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("observations=%+v", got)
	}
	report := got[0].Report
	if report.Status != orquestaruntime.AgentStoppedV0 ||
		report.BudgetStatus == orquestaruntime.AgentProgressBudgetCapacityLimitedV0 ||
		!report.DecisionRequired {
		t.Fatalf("report interrupcion inesperado: %+v", report)
	}
	if stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-capacity-limited") {
		t.Fatalf("texto capacity no debe generar capacity_limited: %+v", report.EvidenceRefs)
	}
	if !stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-no-ack-interrupted") {
		t.Fatalf("interrupcion debe conservar evidencia no_ack: %+v", report.EvidenceRefs)
	}
}

func writeCodexProgressRuntimeFileForTestV0(
	t *testing.T,
	ackPath string,
	name string,
	content string,
) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(filepath.Dir(ackPath), name), []byte(content), 0o600); err != nil {
		t.Fatalf("write runtime file %s: %v", name, err)
	}
}

func assertCodexProgressObservationNoLeaksV0(
	t *testing.T,
	observation orquestacionnucleoapp.AgentProgressObservationV0,
	forbidden ...string,
) {
	t.Helper()
	values := []string{
		observation.CandidateRef,
		observation.PhaseID,
		observation.TaskRef,
		observation.DeliveryRef,
		observation.AssessmentRef,
		observation.QuestionID,
		observation.Report.ReportID,
		observation.Report.Summary,
		observation.Report.BudgetReason,
	}
	values = append(values, observation.EvidenceRefs...)
	values = append(values, observation.Report.EvidenceRefs...)
	for _, value := range values {
		for _, needle := range forbidden {
			if needle != "" && strings.Contains(value, needle) {
				t.Fatalf("observacion filtra %q en %q: %+v", needle, value, observation)
			}
		}
	}
}
