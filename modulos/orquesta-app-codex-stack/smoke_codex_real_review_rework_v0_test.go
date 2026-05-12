package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestNuevaAppWebCodexStackRealReviewReworkOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_STACK_REVIEW_REWORK_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_STACK_REVIEW_REWORK_SMOKE=1 para probar review/rework con Codex real")
	}
	cfg := codexStackRealSmokeConfigForTestV0(t)
	cfg.MaxBatchReady = 4
	cfg.MaxConcurrency = 4
	if cfg.Timeout < 900*time.Second {
		cfg.Timeout = 900 * time.Second
	}
	maxExternalWaits := codexStackRealSmokeMaxExternalWaitsV0(cfg.Timeout, 2*time.Second)
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	stores := newCodexStackRealSmokeStoresV0()
	stack := codexStackRealSmokeBuildStackV0(t, cfg, stores, processRuntime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	defer codexStackRealSmokeStopAllProcessesV0(t, processRuntime, stores.ProcessRegistry, stores.ReceiptStore)

	runRef := codexStackRealSmokeStartMultiagentAppV0(t, ctx, stack, cfg.RuntimeWorkDir)
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-app-stack-real-review-initial-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     maxExternalWaits,
	}); err != nil {
		t.Fatalf("DrainRunV0 inicial: %v issues=%+v\n%s",
			err,
			codexStackBurstIssuesForErrorV0(err),
			codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir),
		)
	}
	programmingDelivery := codexStackRealSmokeDrainUntilFirstProgrammingDeliveryV0(
		t,
		ctx,
		stack,
		stores,
		runRef,
		cfg.RuntimeWorkDir,
		maxExternalWaits,
	)
	descriptor := programmingDelivery.Descriptor
	deliveryRef, damagedPath := codexStackRealSmokeAppendOversizedEvidenceV0(t, cfg.ProjectWorkDir, descriptor)
	openCodexStackPhaseForTestV0(
		t,
		stack,
		runRef,
		orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		"Smoke real: revisar entrega de programacion.",
	)

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-app-stack-real-review-rework-drain",
		MaxBursts:            24,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          24,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     maxExternalWaits,
	}); err != nil {
		t.Fatalf("DrainRunV0 review/rework: %v issues=%+v damaged=%s\n%s",
			err,
			codexStackBurstIssuesForErrorV0(err),
			damagedPath,
			codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir),
		)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	reworkDescriptor := codexStackRealSmokeReworkDescriptorV0(
		t,
		run,
		codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore),
		descriptor,
	)
	codexStackRealSmokeAssertReworkProjectedV0(
		t,
		run,
		deliveryRef,
		codexStackRealSmokeDescriptorAgentRefV0(reworkDescriptor),
	)
	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		CorrelationID:        "corr-app-stack-real-review-rework-ack-drain",
		MaxBursts:            16,
		MaxStepsPerBurst:     12,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     maxExternalWaits,
	}); err != nil {
		t.Fatalf("DrainRunV0 rework ACK: %v\n%s", err, codexStackRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	run = mustLoadCodexStackRunForTestV0(t, stack, runRef)
	if !codexStackRealSmokeContainsProjectionPartV0(
		run.Deliveries,
		reworkDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
	) {
		t.Fatalf("delivery rework no registrada: deliveries=%v rework=%s",
			run.Deliveries,
			reworkDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
		)
	}
	codexStackRealSmokeAssertReviewGateAcceptedV0(
		t,
		ctx,
		stack,
		run,
		reworkDescriptor.Spec.AgentPacket.DeliveryRefs.AckRef,
	)
}

func codexStackRealSmokeAppendOversizedEvidenceV0(
	t *testing.T,
	projectDir string,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (string, string) {
	t.Helper()
	ack, issues := orquestaruntimecodex.ReadAndValidateCodexAgentAckFileV0(descriptor.AckPath, descriptor.Spec)
	if len(issues) > 0 {
		t.Fatalf("ACK no usable para review: %+v", issues)
	}
	for _, raw := range ack.Files {
		rel, ok := codexStackRealSmokeReviewWritablePathV0(raw)
		if !ok {
			continue
		}
		fullPath := filepath.Join(projectDir, filepath.FromSlash(rel))
		if !codexStackRealSmokeReviewCanAppendV0(fullPath) {
			continue
		}
		codexStackRealSmokeAppendReviewLinesV0(t, fullPath, rel)
		return ack.AckRef, rel
	}
	t.Fatalf("ACK sin fichero textual modificable para review: files=%v", ack.Files)
	return "", ""
}

func codexStackRealSmokeReviewWritablePathV0(raw string) (string, bool) {
	rel := strings.TrimSpace(raw)
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "://") {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func codexStackRealSmokeReviewCanAppendV0(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".mod", ".md", ".html", ".css", ".js", ".txt":
		return true
	default:
		return false
	}
}

func codexStackRealSmokeAppendReviewLinesV0(t *testing.T, fullPath string, rel string) {
	t.Helper()
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	lineCount := strings.Count(string(data), "\n")
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lineCount++
	}
	needed := 301 - lineCount
	if needed < 1 {
		needed = 1
	}
	line := codexStackRealSmokeReviewFillerLineV0(rel)
	addition := "\n" + strings.Repeat(line+"\n", needed)
	if err := os.WriteFile(fullPath, append(data, []byte(addition)...), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func codexStackRealSmokeReviewFillerLineV0(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".mod", ".js", ".css":
		return "// orquesta-review-smoke-line"
	case ".html":
		return "<!-- orquesta-review-smoke-line -->"
	default:
		return "orquesta-review-smoke-line"
	}
}
