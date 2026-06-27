package orquestaappcodexstack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestNuevaAppWebCodexStackRealMultiagentOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_STACK_MULTIAGENT_SMOKE=1 para lanzar varios Codex reales desde /nueva-app")
	}
	cfg := codexStackRealSmokeConfigForTestV0(t)
	cfg.MaxBatchReady = 4
	cfg.MaxConcurrency = 4
	maxExternalWaits := codexStackRealSmokeMaxExternalWaitsV0(cfg.Timeout, 2*time.Second)
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
		strings.NewReader(codexStackRealSmokeMultiagentFormValuesV0().Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Director arrancado") {
		t.Fatalf("web status=%d body=%s\n%s", rec.Code, rec.Body.String(), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	descriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
	if len(descriptors) < 4 {
		t.Fatalf("descriptors=%d want>=4 %+v\n%s", len(descriptors), descriptors, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	defer codexStackRealSmokeStopAllProcessesV0(t, processRuntime, stores.ProcessRegistry, stores.ReceiptStore)
	startedRun, err := stores.RunStore.LoadRunV0(ctx, descriptors[0].RunID)
	if err != nil {
		t.Fatalf("LoadRunV0 tras arranque: %v", err)
	}
	if !codexStackRealSmokeAllDescriptorsRequestedAndStartedV0(startedRun, descriptors) {
		t.Fatalf(
			"agents=%v started_agents=%v failed=%v stopped=%v descriptors=%v phase=%s\n%s",
			startedRun.Agents,
			startedRun.StartedAgents,
			startedRun.FailedAgents,
			startedRun.StoppedAgents,
			codexStackRealSmokeDescriptorAgentsV0(descriptors),
			startedRun.CurrentPhase,
			codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir),
		)
	}
	programmingStart := codexStackRealSmokeDrainUntilProgrammingStartedV0(
		t,
		ctx,
		stack,
		stores,
		descriptors[0].RunID,
		cfg.RuntimeWorkDir,
		maxExternalWaits,
	)
	run := programmingStart.Run
	programmingDrain := codexStackRealSmokeDrainUntilProgrammingDeliveredV0(
		t,
		ctx,
		stack,
		stores,
		descriptors[0].RunID,
		cfg.RuntimeWorkDir,
		maxExternalWaits,
	)
	run = programmingDrain.Run
	allDescriptors := programmingDrain.Descriptors
	programmingDescriptors := codexStackRealSmokeProgrammingReceiptDescriptorsV0(allDescriptors)
	if !codexStackRealSmokeAllProgrammingDeliveriesRegisteredV0(run.Deliveries, programmingDescriptors) {
		t.Fatalf(
			"deliveries=%v programming=%v drain_status=%s attempts=%d waits=%d\n%s",
			run.Deliveries,
			codexStackRealSmokeProgrammingDescriptorsV0(allDescriptors),
			programmingDrain.Drain.Status,
			programmingDrain.Drain.Attempts,
			programmingDrain.Drain.Waits,
			codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir),
		)
	}
	if len(run.PhaseArtifacts) < 4 {
		observations := codexStackRealSmokeDeliveryObservationsV0(t, ctx, stack, run)
		t.Fatalf(
			"phase_artifacts=%v drain_status=%s attempts=%d waits=%d observations=%d phase=%s started=%v agents=%v descriptors=%v",
			run.PhaseArtifacts,
			programmingDrain.Drain.Status,
			programmingDrain.Drain.Attempts,
			programmingDrain.Drain.Waits,
			len(observations),
			run.CurrentPhase,
			run.StartedAgents,
			run.Agents,
			codexStackRealSmokeDescriptorAgentsV0(descriptors),
		)
	}
	if got := codexStackRealSmokeEventCountV0(stores.EventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0); got < 4 {
		t.Fatalf("phase_artifacts_registrados=%d artifacts=%v", got, run.PhaseArtifacts)
	}
	allDescriptors = codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
	codexStackRealSmokeAssertProgrammingParallelWaveV0(
		t,
		stores.EventSink.EventsV0(),
		allDescriptors,
		1,
	)
	run, allDescriptors = codexStackRealSmokeRepairMissingDescriptorWriteSetsV0(
		t,
		ctx,
		stack,
		stores,
		descriptors[0].RunID,
		cfg.ProjectWorkDir,
		cfg.RuntimeWorkDir,
		maxExternalWaits,
		allDescriptors,
	)
	codexStackRealSmokeVerifyDescriptorWriteSetsV0(t, cfg.ProjectWorkDir, allDescriptors)
	codexStackRealSmokeVerifyGoFileSizesV0(t, cfg.ProjectWorkDir)
	codexStackRealSmokeVerifyGoAppCompilesV0(t, cfg.ProjectWorkDir)
}
