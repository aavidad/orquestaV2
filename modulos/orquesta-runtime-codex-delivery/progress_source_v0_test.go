package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexProgressObservationSourceV0NoReportaEstancamientoSinSenalVisibleSiSigueRunning(t *testing.T) {
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
	if len(second) != 0 {
		t.Fatalf("second observations=%+v", second)
	}
}

func TestCodexProgressObservationSourceV0NoEscalaABucleSoloPorACKSinCambios(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.Policy.LoopAfterRepeatedActions = 3
	source.EmitProgressing = true
	request := codexProgressRequestForTestV0(spec)

	if got, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil || len(got) != 1 {
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
	if len(third) != 1 ||
		third[0].Report.Status != orquestaruntime.AgentProgressingV0 ||
		third[0].DecisionRequired ||
		third[0].Report.DecisionRequired {
		t.Fatalf("third observations=%+v", third)
	}
}

func TestCodexProgressObservationSourceV0ReemiteDecisionSiElRunNoLaMaterializa(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.SnapshotSource = stoppedSnapshotSourceForProgressTestV0{}
	request := codexProgressRequestForTestV0(spec)

	firstDecision, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil || len(firstDecision) != 1 {
		t.Fatalf("first decision observations=%+v err=%v", firstDecision, err)
	}
	secondDecision, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("second decision BuildAgentProgressObservationsV0: %v", err)
	}
	if len(secondDecision) != 1 ||
		secondDecision[0].Report.Status != orquestaruntime.AgentStoppedV0 ||
		!secondDecision[0].DecisionRequired {
		t.Fatalf("second decision observations=%+v", secondDecision)
	}
}

func TestCodexProgressObservationSourceV0ACKFailedCierraEsperaExternaV0(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	ack := codexDeliveryAckForTestV0(spec)
	ack.Status = "failed"
	ack.Tests = nil
	ack.TestReceipts = nil
	writeCodexDeliveryAckAtPathForTestV0(t, ackPath, ack)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	request := codexProgressRequestForTestV0(spec)
	request.Run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0

	got, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("observations=%+v", got)
	}
	observation := got[0]
	if observation.Report.Status != orquestaruntime.AgentStoppedV0 ||
		!observation.DecisionRequired ||
		!observation.Report.DecisionRequired {
		t.Fatalf("observation=%+v", observation)
	}
	if observation.Report.AgentRequestID != spec.RequestID ||
		observation.TaskRef != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("refs observation=%+v", observation)
	}
	if observation.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("phase_id=%s", observation.PhaseID)
	}
	if !stringInCodexDeliverySetV0(observation.Report.EvidenceRefs, "evidence-ref-codex-ack-failed") {
		t.Fatalf("evidence refs=%+v", observation.Report.EvidenceRefs)
	}
	if codexProgressObservationLeaksPathV0(observation, ackPath) {
		t.Fatalf("observation filtra path local: %+v", observation)
	}
}

func TestCodexProgressObservationSourceV0SilencioSostenidoReportaEstancamientoSinBucle(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.Policy.LoopAfterRepeatedActions = 3
	source.EmitProgressing = true
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
	if len(got) != 1 ||
		got[0].Report.Status != orquestaruntime.AgentProgressingV0 ||
		got[0].DecisionRequired ||
		got[0].Report.DecisionRequired {
		t.Fatalf("running no signal observation=%+v", got)
	}
	if got[0].Report.RepeatedActionCount != 0 || got[0].Report.NoProgressTicks == 0 {
		t.Fatalf("progress counters=%+v", got[0].Report)
	}
	if !stringInCodexDeliverySetV0(got[0].Report.EvidenceRefs, "evidence-ref-running-no-visible-signal") {
		t.Fatalf("evidence refs sin running no visible signal: %+v", got[0].Report.EvidenceRefs)
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

func TestCodexProgressObservationSourceV0DetectaBuclePorDiffRepetidoConLogCreciendo(t *testing.T) {
	spec, ackPath := codexProgressSpecAndAckPathForTestV0(t)
	source := codexProgressSourceForTestV0(t, spec, ackPath)
	source.Policy.StalledAfterNoProgressTicks = 99
	source.Policy.LoopAfterRepeatedActions = 2
	request := codexProgressRequestForTestV0(spec)
	stderrPath := filepath.Join(filepath.Dir(ackPath), orquestaruntimecodex.CodexStderrFileNameV0)
	diffBlock := strings.Join([]string{
		"diff --git a/tmp/run/project/docs/manual_usuario.md b/tmp/run/project/docs/manual_usuario.md",
		"new file mode 100644",
		"@@ -0,0 +1,2 @@",
		"+# Manual",
		"+Contenido",
		"",
	}, "\n")

	if err := os.WriteFile(stderrPath, []byte(diffBlock), 0o600); err != nil {
		t.Fatalf("write stderr first: %v", err)
	}
	if got, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil || len(got) != 0 {
		t.Fatalf("first observations=%+v err=%v", got, err)
	}
	if err := appendCodexProgressTestLogV0(stderrPath, diffBlock); err != nil {
		t.Fatalf("append stderr second: %v", err)
	}
	if got, err := source.BuildAgentProgressObservationsV0(context.Background(), request); err != nil || len(got) != 0 {
		t.Fatalf("second observations=%+v err=%v", got, err)
	}
	if err := appendCodexProgressTestLogV0(stderrPath, diffBlock); err != nil {
		t.Fatalf("append stderr third: %v", err)
	}
	got, err := source.BuildAgentProgressObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("third BuildAgentProgressObservationsV0: %v", err)
	}
	if len(got) != 1 ||
		got[0].Report.Status != orquestaruntime.AgentLoopDetectedV0 ||
		got[0].Report.RepeatedActionCount != 2 ||
		!got[0].DecisionRequired {
		t.Fatalf("loop observation=%+v", got)
	}
	if !stringInCodexDeliverySetV0(got[0].Report.EvidenceRefs, "evidence-ref-repeated-action") {
		t.Fatalf("evidence refs sin repeated action: %+v", got[0].Report.EvidenceRefs)
	}
}
