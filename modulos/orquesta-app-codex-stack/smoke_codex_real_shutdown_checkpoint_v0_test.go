package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackRealShutdownCheckpointOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_STACK_SHUTDOWN_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_STACK_SHUTDOWN_SMOKE=1 para probar shutdown cooperativo con Codex real")
	}
	cfg := codexStackRealSmokeConfigForTestV0(t)
	codexStackRealShutdownAppendAgentInstructionsV0(t, cfg.ProjectWorkDir)
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	stores := newCodexStackRealSmokeStoresV0()
	stack := codexStackRealSmokeBuildStackV0(t, cfg, stores, processRuntime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	descriptor := codexStackRealShutdownStartAgentV0(t, ctx, stack, stores, cfg.RuntimeWorkDir)
	defer codexStackRealSmokeStopAllProcessesV0(t, processRuntime, stores.ProcessRegistry, stores.ReceiptStore)
	codexStackRealShutdownAssertRunningV0(t, processRuntime, stores, descriptor.RunID, descriptor.AgentRef)

	first := codexStackRealShutdownPostV0(t, ctx, stack, "corr-app-stack-real-shutdown-001")
	codexStackRealShutdownAssertPendingV0(t, first, descriptor.AgentRef, cfg.RuntimeWorkDir)
	request := codexStackRealShutdownReadRequestV0(t, descriptor)
	ack := codexStackRealShutdownWaitForAckV0(t, ctx, descriptor, request)
	if strings.TrimSpace(ack.CheckpointRef) != strings.TrimSpace(request.CheckpointRef) {
		t.Fatalf("shutdown ack descorrelado ack=%+v request=%+v", ack, request)
	}

	second := codexStackRealShutdownPostV0(t, ctx, stack, "corr-app-stack-real-shutdown-002")
	if !codexStackRealShutdownCheckpointRecordedV0(second) {
		t.Fatalf("checkpoint no registrado result=%+v ack=%+v\n%s", second, ack, codexStackRealShutdownDiagnosticsV0(cfg.RuntimeWorkDir))
	}
}

func codexStackRealShutdownAppendAgentInstructionsV0(
	t *testing.T,
	projectDir string,
) {
	t.Helper()
	path := filepath.Join(projectDir, "AGENTS.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open AGENTS.md shutdown smoke: %v", err)
	}
	defer f.Close()
	_, err = f.WriteString(strings.Join([]string{
		"",
		"## Smoke shutdown cooperativo",
		"- Antes de escribir agent_ack.json espera hasta 90 segundos comprobando cada 2 segundos si aparece orquesta_shutdown_request.json.",
		"- Si aparece, lee la request y escribe agent_shutdown_checkpoint_ack.json correlado con checkpoint_ready; despues termina sin ampliar alcance.",
		"",
	}, "\n"))
	if err != nil {
		t.Fatalf("write AGENTS.md shutdown smoke: %v", err)
	}
}

func codexStackRealShutdownStartAgentV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	stores codexStackRealSmokeStoresV0,
	runtimeDir string,
) orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 {
	t.Helper()
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
		t.Fatalf("web status=%d body=%s\n%s", rec.Code, rec.Body.String(), codexStackRealShutdownDiagnosticsV0(runtimeDir))
	}
	descriptors := codexStackRealSmokeDescriptorsV0(t, stores.ReceiptStore)
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%d %+v\n%s", len(descriptors), descriptors, codexStackRealShutdownDiagnosticsV0(runtimeDir))
	}
	return descriptors[0]
}

func codexStackRealShutdownPostV0(
	t *testing.T,
	ctx context.Context,
	stack StackV0,
	correlationID string,
) orquestamcp.MCPServerShutdownToolResultV0 {
	t.Helper()
	body, err := json.Marshal(orquestamcp.MCPServerShutdownToolInputV0{
		RequestID:      "request-ref-app-stack-real-shutdown",
		CorrelationID:  correlationID,
		Forced:         false,
		MaxTicks:       1,
		MaxRunsPerTick: 1,
		MaxExecutions:  1,
		RequestedBy:    "orquesta-app-stack-shutdown-smoke",
		Reason:         "shutdown cooperativo real opt-in",
		EvidenceRefs:   []string{"evidence-ref-app-stack-real-shutdown"},
	})
	if err != nil {
		t.Fatalf("marshal shutdown: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v0/server/shutdown", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("shutdown status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPServerShutdownToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode shutdown: %v body=%s", err, rec.Body.String())
	}
	return result
}

func codexStackRealShutdownAssertRunningV0(
	t *testing.T,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	stores codexStackRealSmokeStoresV0,
	runRef string,
	agentRef string,
) {
	t.Helper()
	record, err := stores.ProcessRegistry.ResolveAgentProcessV0(context.Background(), runRef, agentRef)
	if err != nil {
		t.Fatalf("process registry missing run=%s agent=%s: %v", runRef, agentRef, err)
	}
	snapshot, err := processRuntime.SnapshotV0(record.ProcessRef)
	if err != nil {
		t.Fatalf("process snapshot: %v", err)
	}
	if snapshot.Status != orquestaruntime.ProcessRuntimeRunningV0 {
		t.Fatalf("process status=%s want=running", snapshot.Status)
	}
}

func codexStackRealShutdownAssertPendingV0(
	t *testing.T,
	result orquestamcp.MCPServerShutdownToolResultV0,
	agentRef string,
	runtimeDir string,
) {
	t.Helper()
	if result.Status != "waiting_checkpoint" ||
		result.ShutdownReady ||
		result.CheckpointAgentsPending == 0 ||
		!codexStackRealShutdownResultHasPendingAgentV0(result, agentRef) {
		t.Fatalf("shutdown pending result=%+v\n%s", result, codexStackRealShutdownDiagnosticsV0(runtimeDir))
	}
}

func codexStackRealShutdownResultHasPendingAgentV0(
	result orquestamcp.MCPServerShutdownToolResultV0,
	agentRef string,
) bool {
	for _, run := range result.Runs {
		for _, pending := range run.PendingCheckpointAgentRefs {
			if strings.TrimSpace(pending) == strings.TrimSpace(agentRef) {
				return true
			}
		}
	}
	return false
}

func codexStackRealShutdownReadRequestV0(
	t *testing.T,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestaruntimecodex.CodexShutdownRequestV0 {
	t.Helper()
	path := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexShutdownRequestFileNameV0)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("shutdown request no escrita: %v", err)
	}
	var request orquestaruntimecodex.CodexShutdownRequestV0
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatalf("decode shutdown request: %v", err)
	}
	if issues := orquestaruntimecodex.ValidateCodexShutdownRequestV0(request); len(issues) > 0 {
		t.Fatalf("shutdown request invalida: %+v", issues)
	}
	if request.RunRef != descriptor.RunID || request.AgentRef != descriptor.AgentRef {
		t.Fatalf("shutdown request descorrelada request=%+v descriptor=%+v", request, descriptor)
	}
	return request
}

func codexStackRealShutdownWaitForAckV0(
	t *testing.T,
	ctx context.Context,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	request orquestaruntimecodex.CodexShutdownRequestV0,
) orquestaruntimecodex.CodexShutdownCheckpointAckV0 {
	t.Helper()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	ackPath := filepath.Join(filepath.Dir(descriptor.AckPath), orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0)
	for {
		ack, issues := orquestaruntimecodex.ReadCodexShutdownCheckpointAckFileV0(ackPath, request)
		if len(issues) == 0 {
			return ack
		}
		if !codexStackRealShutdownAckPendingV0(issues) {
			t.Fatalf("shutdown ack invalido issues=%+v\n%s", issues, codexStackRealShutdownDiagnosticsV0(filepath.Dir(ackPath)))
		}
		select {
		case <-ctx.Done():
			t.Fatalf("shutdown ack timeout: %v\n%s", ctx.Err(), codexStackRealShutdownDiagnosticsV0(filepath.Dir(ackPath)))
		case <-ticker.C:
		}
	}
}

func codexStackRealShutdownAckPendingV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	if len(issues) != 1 {
		return false
	}
	issue := issues[0]
	return issue.Retryable &&
		strings.TrimSpace(issue.Field) == orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0 &&
		codexStackRealSmokeContainsProjectionPartV0(issue.Evidence, "checkpoint_ack_not_ready")
}

func codexStackRealShutdownCheckpointRecordedV0(
	result orquestamcp.MCPServerShutdownToolResultV0,
) bool {
	if result.CheckpointAgentsPending != 0 {
		return false
	}
	for _, run := range result.Runs {
		if strings.TrimSpace(run.CheckpointRef) != "" &&
			len(run.PendingCheckpointAgentRefs) == 0 {
			return true
		}
	}
	return false
}

func codexStackRealShutdownDiagnosticsV0(runtimeDir string) string {
	parts := []string{codexStackRealSmokeDiagnosticsV0(runtimeDir)}
	_ = filepath.WalkDir(runtimeDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		switch entry.Name() {
		case orquestaruntimecodex.CodexShutdownRequestFileNameV0,
			orquestaruntimecodex.CodexShutdownCheckpointAckFileNameV0:
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				parts = append(parts, path+": "+codexStackRealSmokeTruncateV0(string(data)))
			}
		}
		return nil
	})
	return strings.Join(compactStringsV0(parts), "\n")
}
