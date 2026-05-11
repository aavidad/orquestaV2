package orquestaappcodexstack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestNuevaAppWebCodexStackRealCambioMitadOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_STACK_CHANGE_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_STACK_CHANGE_SMOKE=1 para probar cambio real a mitad")
	}
	cfg := codexStackRealSmokeConfigForTestV0(t)
	cfg.MaxBatchReady = 4
	cfg.MaxConcurrency = 4
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	stores := newCodexStackRealSmokeStoresV0()
	stack := codexStackRealSmokeBuildStackV0(t, cfg, stores, processRuntime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	defer codexStackRealSmokeStopAllProcessesV0(t, processRuntime, stores.ProcessRegistry, stores.ReceiptStore)

	runRef := codexStackRealSmokeStartMultiagentAppV0(t, ctx, stack, cfg.RuntimeWorkDir)
	initialDescriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
	if len(initialDescriptors) < 4 {
		t.Fatalf("descriptors iniciales=%d\n%s", len(initialDescriptors), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if err := codexStackRealSmokeWaitForDescriptorsAckV0(ctx, initialDescriptors); err != nil {
		t.Fatalf("ACK inicial no validado: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-app-stack-real-change-initial-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 4,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("DrainRunV0 inicial: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	programmingBeforeChange := codexStackRealSmokeProgrammingReceiptDescriptorsV0(
		codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore),
	)
	if len(programmingBeforeChange) == 0 {
		t.Fatalf("no hay agentes de programacion antes del cambio\n%s", codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}

	codexStackRealSmokePostMidRunChangeV0(t, ctx, stack, runRef)
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-app-stack-real-change-mid-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("DrainRunV0 cambio: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}

	allDescriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
	changeDescriptor, ok := codexStackRealSmokeDescriptorByWriteSetV0(
		allDescriptors,
		"docs/change-request-midrun.md",
	)
	if !ok {
		t.Fatalf("descriptor de cambio no encontrado descriptors=%v\n%s", codexStackRealSmokeDescriptorAgentsV0(allDescriptors), codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if err := codexStackRealSmokeWaitForAckPathV0(ctx, changeDescriptor.AckPath, changeDescriptor.Spec); err != nil {
		t.Fatalf("ACK cambio no validado: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-app-stack-real-change-final-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("DrainRunV0 final: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	codexStackRealSmokeVerifyProjectFileV0(t, cfg.ProjectWorkDir, "docs/change-request-midrun.md")
}

func codexStackRealSmokeStartMultiagentAppV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runtimeWorkDir string,
) string {
	t.Helper()
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
		t.Fatalf("web status=%d body=%s\n%s", rec.Code, rec.Body.String(), codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
	}
	descriptors := codexStackRealSmokeDescriptorsV0(t, stack.Stores.ReceiptStore.(*orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0))
	if len(descriptors) == 0 {
		t.Fatalf("sin descriptores tras /nueva-app\n%s", codexStackRealSmokeDiagnosticsV0(runtimeWorkDir))
	}
	return descriptors[0].RunID
}

func codexStackRealSmokePostMidRunChangeV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	values := url.Values{}
	values.Set("request_id", "request-ref-app-stack-real-change-midrun-001")
	values.Set("correlation_id", "corr-app-stack-real-change-midrun-001")
	values.Set("locale", "es-ES")
	values.Set("run_ref", runRef)
	values.Set("app_ref", "app-ref-agenda-api-web")
	values.Set("change_ref", "change-ref-real-midrun-001")
	values.Set("user_intent", "Anadir una nota de cambio de vista mensual mientras la programacion sigue en curso.")
	values.Set("target_area", "docs")
	values.Set("acceptance_criteria", "documentar vista mensual de agenda en docs/change-request-midrun.md")
	values.Set("allowed_write_set", "docs/change-request-midrun.md")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/app-change", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Cambio enviado al director") {
		t.Fatalf("app-change status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func codexStackRealSmokeDescriptorByWriteSetV0(
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	path string,
) (orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, bool) {
	for _, descriptor := range descriptors {
		for _, entry := range descriptor.Spec.AgentPacket.Task.WriteSet {
			if strings.TrimSpace(entry) == path {
				return descriptor, true
			}
		}
	}
	return orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}, false
}
