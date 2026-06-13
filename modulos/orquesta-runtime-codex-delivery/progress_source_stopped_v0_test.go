package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressObservationSourceV0PuedeEmitirProgresoParaEstadisticas(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.EmitProgressing = true

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
	if got[0].Report.Status != orquestaruntime.AgentProgressingV0 ||
		got[0].DecisionRequired ||
		got[0].Report.DecisionRequired {
		t.Fatalf("progress report inesperado: %+v", got[0])
	}
}

func TestCodexProgressObservationSourceV0ReportaProcesoParadoSinACKEnPrimerTick(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 || got[0].Report.Status != orquestaruntime.AgentStoppedV0 {
		t.Fatalf("observations=%+v", got)
	}
	if !stringInCodexDeliverySetV0(got[0].Report.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("evidence refs sin no_ack compacto: %+v", got[0].Report.EvidenceRefs)
	}
}

func TestCodexProgressObservationSourceV0ReportaArtefactoSinACKComoRevision(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# entrega\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	descriptor := codexProgressDescriptorForTestV0(spec, ackPath)
	descriptor.ProjectWorkDir = projectDir
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.Store = NewInMemoryCodexReceiptDescriptorStoreV0(descriptor)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 || got[0].Report.Status != orquestaruntime.AgentStoppedV0 {
		t.Fatalf("observations=%+v", got)
	}
	if !stringInCodexDeliverySetV0(got[0].Report.EvidenceRefs, "evidence-ref-artifact-without-ack") {
		t.Fatalf("evidence refs sin artifact_without_ack: %+v", got[0].Report.EvidenceRefs)
	}
	if codexProgressObservationLeaksPathV0(got[0], projectDir) {
		t.Fatalf("observacion filtra path: %+v", got[0])
	}
}

func TestCodexProgressObservationSourceV0SnapshotPerdidoEsProcesoParado(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = missingSnapshotSourceForProgressTestV0{}

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 || got[0].Report.Status != orquestaruntime.AgentStoppedV0 {
		t.Fatalf("observations=%+v", got)
	}
	if !got[0].DecisionRequired || !got[0].Report.DecisionRequired {
		t.Fatalf("proceso perdido debe requerir decision: %+v", got[0])
	}
}

type missingSnapshotSourceForProgressTestV0 struct{}

func (missingSnapshotSourceForProgressTestV0) SnapshotV0(
	string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return orquestaruntime.ProcessRuntimeSnapshotV0{}, orquestaruntime.ProcessRuntimeErrorV0{
		Code:       orquestaruntime.ProcessRuntimeNoEncontradoV0,
		MessageKey: "process_runtime.no_encontrado",
		Field:      "process_ref",
		Retryable:  true,
	}
}

func TestCodexProgressObservationSourceV0NoClasificaCapacidadPorTextoSinACK(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	stderrPath := filepath.Join(filepath.Dir(ackPath), orquestaruntimecodex.CodexStderrFileNameV0)
	if err := os.WriteFile(
		stderrPath,
		[]byte("ERROR: Selected model is at capacity. Please try a different model."),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

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
		report.BudgetStatus != "" ||
		!report.DecisionRequired ||
		!stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("report no_ack inesperado: %+v", report)
	}
	if stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-capacity-warning") {
		t.Fatalf("texto de capacidad no debe generar capacity_warning: %+v", report.EvidenceRefs)
	}
	if codexProgressObservationLeaksPathV0(got[0], ackPath) {
		t.Fatalf("observacion filtra path: %+v", got[0])
	}
}

func TestCodexProgressObservationSourceV0ClasificaCuotaEstructuradaSinACK(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	if err := os.WriteFile(
		filepath.Join(filepath.Dir(ackPath), orquestaruntimecodex.CodexUsageAccountingFileNameV0),
		[]byte(`{"usage":{},"quota":{"status":"exhausted"}}`),
		0o600,
	); err != nil {
		t.Fatalf("write usage accounting: %v", err)
	}

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
		report.BudgetStatus != orquestaruntime.AgentProgressBudgetCapacityLimitedV0 ||
		!report.DecisionRequired ||
		!stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-provider-quota-exhausted") ||
		!stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-capacity-limited") {
		t.Fatalf("report cuota estructurada inesperado: %+v", report)
	}
	if codexProgressObservationLeaksPathV0(got[0], ackPath) {
		t.Fatalf("observacion filtra path: %+v", got[0])
	}
}

func TestCodexProgressObservationSourceV0NoClasificaAuthPorTextoLibreSinACK(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	stderrPath := filepath.Join(filepath.Dir(ackPath), orquestaruntimecodex.CodexStderrFileNameV0)
	if err := os.WriteFile(
		stderrPath,
		[]byte("ERROR 401 token_invalidated refresh_token_reused"),
		0o600,
	); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

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
		!report.DecisionRequired ||
		!stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-no-ack") {
		t.Fatalf("report no_ack inesperado: %+v", report)
	}
	if report.BudgetStatus != "" ||
		stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-auth-config-blocker") ||
		stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-capacity-warning") {
		t.Fatalf("texto libre no debe generar auth/capacity: %+v", report)
	}
	if codexProgressObservationLeaksPathV0(got[0], ackPath) {
		t.Fatalf("observacion filtra path: %+v", got[0])
	}
}

type stoppedSnapshotSourceForProgressTestV0 struct{}

func (stoppedSnapshotSourceForProgressTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    "session-ref-progress-001",
		LaunchRef:     "launch-ref-progress-001",
		Status:        orquestaruntime.ProcessRuntimeStoppedV0,
	}, nil
}
