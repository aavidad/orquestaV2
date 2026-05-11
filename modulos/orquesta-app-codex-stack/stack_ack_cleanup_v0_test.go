package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexStackV0CleanupRuntimeTrasACKRegistrado(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if runtime.stopCountV0() != 4 {
		t.Fatalf("stops=%d want=4", runtime.stopCountV0())
	}
	if len(run.StoppedAgents) != 0 || len(run.ConfirmedStoppedAgents) != 0 {
		t.Fatalf("cleanup no debe contaminar paradas: stopped=%v confirmed=%v", run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	sink := stack.Stores.EventSink.(*orquestacionnucleoapp.InMemoryEventSinkV0)
	if codexStackRealSmokeEventCountV0(
		sink.EventsV0(),
		orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0,
	) != 0 {
		t.Fatalf("cleanup no debe emitir AgentStopConfirmed: %+v", sink.EventsV0())
	}
}

func TestCodexStackV0NoCleanupSinACK(t *testing.T) {
	runtime := &noAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
	stack := mustBuildCodexStackForTestV0(t, runtime)

	_ = postDirectorAPIV0(t, stack)
	if runtime.stopCountV0() != 0 {
		t.Fatalf("stops sin ACK=%d", runtime.stopCountV0())
	}
}

func TestCodexStackV0NoCleanupConACKInvalido(t *testing.T) {
	runtime := &invalidAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
	stack := mustBuildCodexStackForTestV0(t, runtime)

	postDirectorAllowingErrorV0(t, stack)
	if runtime.stopCountV0() != 0 {
		t.Fatalf("stops con ACK invalido=%d", runtime.stopCountV0())
	}
}

func postDirectorAllowingErrorV0(t *testing.T, stack StackV0) {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:            "request-ref-app-stack-invalid-ack-001",
		CorrelationID:        "corr-app-stack-invalid-ack-001",
		AppSpecRequest:       codexStackAppSpecRequestV0(),
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
}

type noAckCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func (runtime *noAckCodexStackRuntimeV0) LaunchV0(
	_ context.Context,
	_ orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return runtime.launchSnapshotWithoutACKV0("process-ref-app-stack-noack-")
}

type invalidAckCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func (runtime *invalidAckCodexStackRuntimeV0) LaunchV0(
	_ context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtimeDir := filepath.Dir(req.CommandPath)
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		[]byte(`{"schema_version":"codex_agent_ack.v0"}`),
		0o600,
	); err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	return runtime.launchSnapshotWithoutACKV0("process-ref-app-stack-invalidack-")
}

func (runtime *fakeCodexStackRuntimeV0) launchSnapshotWithoutACKV0(
	prefix string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.next++
	ref := strconv.Itoa(runtime.next)
	snapshot := orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    prefix + ref,
		SessionRef:    "session-ref-app-stack-noack-" + ref,
		LaunchRef:     "launch-ref-app-stack-noack-" + ref,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	return snapshot, nil
}
