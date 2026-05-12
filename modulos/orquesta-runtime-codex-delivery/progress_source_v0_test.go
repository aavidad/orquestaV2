package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressObservationSourceV0ReportaEstancamientoSinACKNiProgreso(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	request := codexProgressRequestForTestV0(spec)

	first, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("first BuildAgentProgressObservationsV0: %v", err)
	}
	if len(first) != 0 {
		t.Fatalf("first observations=%+v", first)
	}

	second, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("second BuildAgentProgressObservationsV0: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("second observations=%+v", second)
	}
	got := second[0]
	if got.Report.Status != orquestaruntime.AgentStalledV0 ||
		got.Report.AgentRequestID != spec.RequestID ||
		got.TaskRef != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("observacion no correlada: %+v", got)
	}
	if codexProgressObservationLeaksPathV0(got, ackPath) {
		t.Fatalf("observacion filtra path: %+v", got)
	}
}

func TestCodexProgressObservationSourceV0NoEscalaABucleSoloPorACKSinCambios(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.Policy.LoopAfterRepeatedActions = 3
	request := codexProgressRequestForTestV0(spec)

	if got, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil || len(got) != 0 {
		t.Fatalf("first observations=%+v err=%v", got, err)
	}
	if got, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil || len(got) != 1 {
		t.Fatalf("second observations=%+v err=%v", got, err)
	} else {
		request.Run.AgentAssessments = append(request.Run.AgentAssessments, codexProgressAssessmentProjectionForTestV0(got[0]))
	}
	third, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("third BuildAgentProgressObservationsV0: %v", err)
	}
	if len(third) != 0 {
		t.Fatalf("third observations=%+v", third)
	}
}

func TestCodexProgressObservationSourceV0ReemiteSiElRunNoHaMaterializadoDecision(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	request := codexProgressRequestForTestV0(spec)

	if _, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil {
		t.Fatalf("first BuildAgentProgressObservationsV0: %v", err)
	}
	firstDecision, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil || len(firstDecision) != 1 {
		t.Fatalf("first decision observations=%+v err=%v", firstDecision, err)
	}
	secondDecision, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("second decision BuildAgentProgressObservationsV0: %v", err)
	}
	if len(secondDecision) != 1 {
		t.Fatalf("second decision observations=%+v", secondDecision)
	}
}

func TestCodexProgressObservationSourceV0SilencioSostenidoReportaEstancamientoSinBucle(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.Policy.LoopAfterRepeatedActions = 3
	request := codexProgressRequestForTestV0(spec)

	for i := 0; i < 3; i++ {
		if _, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil {
			t.Fatalf("BuildAgentProgressObservationsV0 %d: %v", i+1, err)
		}
	}
	got, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("fourth BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 || got[0].Report.Status != orquestaruntime.AgentStalledV0 {
		t.Fatalf("stalled observation=%+v", got)
	}
	if got[0].Report.RepeatedActionCount != 0 || got[0].Report.NoProgressTicks == 0 {
		t.Fatalf("progress counters=%+v", got[0].Report)
	}
}

func TestCodexProgressObservationSourceV0NoReportaSiHayAvanceDeLogs(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	request := codexProgressRequestForTestV0(spec)
	logPath := filepath.Join(filepath.Dir(ackPath), orquestaruntimecodex.CodexStdoutFileNameV0)

	if _, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil {
		t.Fatalf("first BuildAgentProgressObservationsV0: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("avance compacto"), 0o600); err != nil {
		t.Fatalf("write progress log: %v", err)
	}
	got, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("second BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("observations=%+v", got)
	}
}

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

func TestCodexProgressObservationSourceV0ClasificaCapacidadLimitadaSinACK(t *testing.T) {
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
		report.BudgetStatus != orquestaruntime.AgentProgressBudgetCapacityLimitedV0 ||
		!report.DecisionRequired {
		t.Fatalf("report capacity inesperado: %+v", report)
	}
	if !stringInCodexDeliverySetV0(report.EvidenceRefs, "evidence-ref-capacity-warning") {
		t.Fatalf("evidence refs sin capacity_warning compacto: %+v", report.EvidenceRefs)
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

func TestCodexProgressObservationSourceV0OmiteSiACKYaEstaListo(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	if err := os.WriteFile(ackPath, codexProgressACKBytesForTestV0(t, spec), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}

	got, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("observations=%+v", got)
	}
}

func TestCodexProgressObservationSourceV0ExigeRegistroDeProceso(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	store := NewInMemoryCodexReceiptDescriptorStoreV0(codexProgressDescriptorForTestV0(spec, ackPath))
	source := CodexProgressObservationSourceV0{
		Store:           store,
		ProcessRegistry: orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0(),
		State:           NewInMemoryCodexProgressStateStoreV0(),
		Policy:          orquestaruntime.AgentProgressHeartbeatPolicyV0{StalledAfterNoProgressTicks: 1},
	}

	_, err := source.BuildAgentProgressObservationsV0(
		context.Background(),
		codexProgressRequestForTestV0(spec),
	)
	if err == nil {
		t.Fatalf("esperaba error por process registry vacio")
	}
}

func TestCodexProgressObservationSourceV0AlimentaCandidateProvider(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	request := orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           codexProgressRunForTestV0(spec),
		OccurredAt:    "2026-05-09T12:30:00Z",
		CorrelationID: "corr-progress-provider-001",
	}
	provider := orquestacionnucleoapp.ProgressSupervisionCandidateProviderV0{
		ProgressSource: source,
		RequestedBy:    "orquesta",
	}

	if _, err := provider.BuildSchedulerCandidatesV0(context.Background(), request); err != nil {
		t.Fatalf("first BuildSchedulerCandidatesV0: %v", err)
	}
	got, err := provider.BuildSchedulerCandidatesV0(context.Background(), request)
	if err != nil {
		t.Fatalf("second BuildSchedulerCandidatesV0: %v", err)
	}
	if len(got.ProgressSupervisionCandidates) != 1 {
		t.Fatalf("progress candidates=%+v", got.ProgressSupervisionCandidates)
	}
	candidate := got.ProgressSupervisionCandidates[0]
	if candidate.SupervisionInput.Report.Status != orquestaruntime.AgentStalledV0 ||
		candidate.SupervisionInput.QuestionID == "" {
		t.Fatalf("candidate=%+v", candidate)
	}
}

func codexProgressSpecAndAckPathForTestV0(
	t *testing.T,
) (orquestaruntime.ExternalAgentLaunchSpecV0, string) {
	t.Helper()
	spec := codexDeliverySpecForTestV0()
	ackPath := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	return spec, ackPath
}

func codexProgressSourceForTestV0(
	t *testing.T,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ackPath string,
) CodexProgressObservationSourceV0 {
	t.Helper()
	registry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), codexProgressProcessRecordForTestV0(spec)); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	return CodexProgressObservationSourceV0{
		Store:           NewInMemoryCodexReceiptDescriptorStoreV0(codexProgressDescriptorForTestV0(spec, ackPath)),
		ProcessRegistry: registry,
		State:           NewInMemoryCodexProgressStateStoreV0(),
		Policy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 1,
			LoopAfterRepeatedActions:    4,
		},
	}
}

func codexProgressRequestForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacionnucleoapp.AgentProgressObservationRequestV0 {
	return orquestacionnucleoapp.AgentProgressObservationRequestV0{
		Run:           codexProgressRunForTestV0(spec),
		CorrelationID: "corr-progress-001",
	}
}

func codexProgressRunForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}
}

func codexProgressDescriptorForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ackPath string,
) CodexReceiptDescriptorV0 {
	return CodexReceiptDescriptorV0{
		DescriptorRef: "receipt-ref-progress-001",
		RunID:         "run-ref-001",
		AgentRef:      spec.RequestID,
		Spec:          spec,
		AckPath:       ackPath,
	}
}

func codexProgressProcessRecordForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacionnucleoapp.AgentProcessRegistryRecordV0 {
	return orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          "run-ref-001",
		AgentRequestID: spec.RequestID,
		ProcessRef:     "process-ref-progress-001",
		SessionRef:     "session-ref-progress-001",
		LaunchRef:      "launch-ref-progress-001",
		ReadinessRef:   "readiness-ref-progress-001",
		EvidenceRefs:   []string{"process-ref-progress-001", "session-ref-progress-001"},
	}
}

func codexProgressACKBytesForTestV0(
	t *testing.T,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []byte {
	t.Helper()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	return data
}

func codexProgressObservationLeaksPathV0(
	observation orquestacionnucleoapp.AgentProgressObservationV0,
	path string,
) bool {
	values := append([]string{
		observation.CandidateRef,
		observation.PhaseID,
		observation.TaskRef,
		observation.DeliveryRef,
		observation.AssessmentRef,
		observation.QuestionID,
		observation.Report.ReportID,
		observation.Report.Summary,
	}, observation.EvidenceRefs...)
	values = append(values, observation.Report.EvidenceRefs...)
	for _, value := range values {
		if strings.Contains(value, path) {
			return true
		}
	}
	return false
}

func codexProgressAssessmentProjectionForTestV0(
	observation orquestacionnucleoapp.AgentProgressObservationV0,
) string {
	return orquestacoreworkflow.AgentAssessmentProjectionRefV0(
		orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-" + observation.Report.ReportID,
			PhaseID:        observation.PhaseID,
			AgentRequestID: observation.Report.AgentRequestID,
			TaskRef:        observation.TaskRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityMediumV0,
			Summary:        "Assessment fake ya materializado.",
			EvidenceRefs:   observation.EvidenceRefs,
		},
	)
}
