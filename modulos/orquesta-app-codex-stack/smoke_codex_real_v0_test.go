package orquestaappcodexstack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestNuevaAppWebCodexStackRealOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_STACK_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_STACK_SMOKE=1 para lanzar Codex real desde /nueva-app")
	}
	cfg := codexStackRealSmokeConfigForTestV0(t)
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	stores := newCodexStackRealSmokeStoresV0()
	stack := codexStackRealSmokeBuildStackV0(t, cfg, stores, processRuntime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"/nueva-app",
		strings.NewReader(codexStackRealSmokeFormValuesV0().Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Director arrancado") {
		t.Fatalf("web status=%d body=%s\n%s", rec.Code, rec.Body.String(), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	descriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%d %+v\n%s", len(descriptors), descriptors, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	descriptor := descriptors[0]
	defer codexStackRealSmokeStopProcessV0(t, processRuntime, stores.ProcessRegistry, descriptor.RunID, descriptor.AgentRef)

	if err := codexStackRealSmokeWaitForAckPathV0(ctx, descriptor.AckPath, descriptor.Spec); err != nil {
		t.Fatalf("ack no validado: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:        descriptor.RunID,
		CorrelationID: "corr-app-stack-real-drain",
	}); err != nil {
		t.Fatalf("DrainRunV0: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	run, err := stores.RunStore.LoadRunV0(ctx, descriptor.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	ackRef := descriptor.Spec.AgentPacket.DeliveryRefs.AckRef
	if !codexStackRealSmokeContainsProjectionPartV0(run.PhaseArtifacts, ackRef) {
		t.Fatalf("phase_artifacts=%v missing=%s\n%s", run.PhaseArtifacts, ackRef, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if !codexStackRealSmokeHasEventV0(stores.EventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0) {
		t.Fatalf("sink no contiene PhaseArtifactRegistered")
	}
	codexStackRealSmokeVerifyDocsV0(t, cfg.ProjectWorkDir)
}
