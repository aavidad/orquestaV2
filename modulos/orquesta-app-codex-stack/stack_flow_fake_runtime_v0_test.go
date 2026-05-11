package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type fakeCodexStackRuntimeV0 struct {
	mu        sync.Mutex
	next      int
	snapshots map[string]orquestaruntime.ProcessRuntimeSnapshotV0
	stops     []string
}

func newFakeCodexStackRuntimeV0() *fakeCodexStackRuntimeV0 {
	return &fakeCodexStackRuntimeV0{snapshots: map[string]orquestaruntime.ProcessRuntimeSnapshotV0{}}
}

func (runtime *fakeCodexStackRuntimeV0) LaunchV0(
	_ context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	if err := runtime.writeAckV0(req); err != nil {
		return orquestaruntime.ProcessRuntimeSnapshotV0{}, err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.next++
	ref := strconv.Itoa(runtime.next)
	snapshot := orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-app-stack-" + ref,
		SessionRef:    "session-ref-app-stack-" + ref,
		LaunchRef:     "launch-ref-app-stack-" + ref,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	return snapshot, nil
}

func (runtime *fakeCodexStackRuntimeV0) StopV0(
	_ context.Context,
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	snapshot := runtime.snapshots[processRef]
	snapshot.Status = orquestaruntime.ProcessRuntimeStoppedV0
	snapshot.StopRef = "stop-ref-" + processRef
	runtime.snapshots[processRef] = snapshot
	runtime.stops = append(runtime.stops, processRef)
	return snapshot, nil
}

func (runtime *fakeCodexStackRuntimeV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.snapshots[processRef], nil
}

func (runtime *fakeCodexStackRuntimeV0) launchCountV0() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.next
}

func (runtime *fakeCodexStackRuntimeV0) stopCountV0() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return len(runtime.stops)
}

func (runtime *fakeCodexStackRuntimeV0) writeAckV0(
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) error {
	runtimeDir := filepath.Dir(req.CommandPath)
	packetPath := filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentPacketFileNameV0)
	data, err := os.ReadFile(packetPath)
	if err != nil {
		return err
	}
	var packet orquestaruntime.AgentStartPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return err
	}
	files, err := runtime.writeDeliveryFilesV0(req.WorkingDir, packet.Task.WriteSet)
	if err != nil {
		return err
	}
	ack := map[string]any{
		"schema_version": "codex_agent_ack.v0",
		"request_id":     packet.RequestID,
		"correlation_id": packet.CorrelationID,
		"ack_ref":        packet.DeliveryRefs.AckRef,
		"target_module":  packet.TargetModule,
		"task_ref":       packet.Task.TaskRef,
		"status":         "completed",
		"files":          files,
		"tests":          packet.Task.RequiredTests,
		"notes":          []string{"fake runtime ack"},
	}
	ackData, err := json.Marshal(ack)
	if err != nil {
		return err
	}
	return os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		ackData,
		0o600,
	)
}

func (runtime *fakeCodexStackRuntimeV0) writeDeliveryFilesV0(
	projectDir string,
	writeSet []string,
) ([]string, error) {
	files := make([]string, 0, len(writeSet))
	for _, target := range writeSet {
		file := codexStackFakeDeliveryFileV0(target)
		if file == "" {
			continue
		}
		path := filepath.Join(projectDir, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte("entrega fake para revision\n"), 0o600); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func codexStackFakeDeliveryFileV0(target string) string {
	target = filepath.ToSlash(filepath.Clean(target))
	if target == "." || target == "" {
		return "entrega.md"
	}
	if filepath.Ext(target) == "" {
		return target + "/entrega.md"
	}
	return target
}
