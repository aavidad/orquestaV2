package orquestaappcodexstack

import (
	"context"
	"strconv"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type delayedAckCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
	delay time.Duration
}

func newDelayedAckCodexStackRuntimeV0(delay time.Duration) *delayedAckCodexStackRuntimeV0 {
	return &delayedAckCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
		delay:                   delay,
	}
}

func (runtime *delayedAckCodexStackRuntimeV0) LaunchV0(
	_ context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	runtime.next++
	ref := strconv.Itoa(runtime.next)
	snapshot := orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    "process-ref-app-stack-delayed-" + ref,
		SessionRef:    "session-ref-app-stack-delayed-" + ref,
		LaunchRef:     "launch-ref-app-stack-delayed-" + ref,
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}
	runtime.snapshots[snapshot.ProcessRef] = snapshot
	runtime.mu.Unlock()

	go func() {
		time.Sleep(runtime.delay)
		_ = runtime.writeAckV0(req)
	}()
	return snapshot, nil
}
