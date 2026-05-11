package orquestacionnucleoapp

import (
	"context"
	"strings"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const externalProcessCleanupTimeoutV0 = 2 * time.Second

func cleanupStartedAgentProcessV0(
	stopper ProcessRuntimeStopPortV0,
	snapshot orquestaruntime.ProcessRuntimeSnapshotV0,
) {
	processRef := strings.TrimSpace(snapshot.ProcessRef)
	if stopper == nil || processRef == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), externalProcessCleanupTimeoutV0)
	defer cancel()
	_, _ = stopper.StopV0(ctx, processRef)
}
