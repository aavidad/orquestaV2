package main

import (
	"fmt"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func shutdownClientResultCanWaitV0(result serverShutdownClientResultV0) bool {
	switch strings.TrimSpace(result.Status) {
	case "waiting_drain", "waiting_checkpoint":
		return true
	default:
		return false
	}
}

func shutdownClientResultReadyForSignalV0(result serverShutdownClientResultV0) bool {
	noActiveWork := result.ActiveWorkCount == 0 &&
		len(compactStringsV0(result.ActiveWorkRefs)) == 0
	return (result.ShutdownReady && noActiveWork) ||
		(result.AgentsInFlight == 0 &&
			result.CheckpointsPending == 0 &&
			result.CheckpointAgentsPending == 0 &&
			noActiveWork &&
			result.RunsRequested <= result.RunsStopped)
}

func shutdownPublicStatusReadyForSignalV0(status orquestaserver.ServerPublicStatusV0) bool {
	noActiveWork := status.ShutdownActiveWorkCount == 0 &&
		len(compactStringsV0(status.ShutdownActiveWorkRefs)) == 0
	return (status.ShutdownReady && noActiveWork) ||
		(status.ShutdownInProgress &&
			status.ShutdownAgentsInFlight == 0 &&
			status.ShutdownCheckpointsPending == 0 &&
			status.ShutdownCheckpointAgentsPending == 0 &&
			status.ShutdownAsyncWorkActive == 0 &&
			noActiveWork &&
			status.ShutdownRunsRequested <= status.ShutdownRunsStopped)
}

func serverShutdownClientResultFromStatusV0(status orquestaserver.ServerPublicStatusV0) serverShutdownClientResultV0 {
	return serverShutdownClientResultV0{
		Estado:                  "ok",
		Status:                  strings.TrimSpace(status.ShutdownStatus),
		ShutdownReady:           status.ShutdownReady,
		RunsRequested:           status.ShutdownRunsRequested,
		RunsStopped:             status.ShutdownRunsStopped,
		AgentsInFlight:          status.ShutdownAgentsInFlight,
		CheckpointsPending:      status.ShutdownCheckpointsPending,
		CheckpointAgentsPending: status.ShutdownCheckpointAgentsPending,
		ActiveWorkCount:         status.ShutdownActiveWorkCount,
		ActiveWorkRefs:          compactStringsV0(status.ShutdownActiveWorkRefs),
	}
}

func shutdownClientNotReadyErrorV0(result serverShutdownClientResultV0) error {
	activeWorkRefs := shutdownClientNotReadyActiveWorkRefsV0(result)
	activeWorkRefsPart := ""
	if len(activeWorkRefs) > 0 {
		activeWorkRefsPart = " active_work_refs=" + strings.Join(activeWorkRefs, ",")
	}
	return fmt.Errorf(
		"shutdown_not_ready status=%s runs=%d/%d agents_in_flight=%d checkpoints=%d checkpoint_agents=%d active_work=%d%s",
		result.Status,
		result.RunsStopped,
		result.RunsRequested,
		result.AgentsInFlight,
		result.CheckpointsPending,
		result.CheckpointAgentsPending,
		result.ActiveWorkCount,
		activeWorkRefsPart,
	)
}

func shutdownClientNotReadyActiveWorkRefsV0(result serverShutdownClientResultV0) []string {
	refs := compactStringsV0(result.ActiveWorkRefs)
	out := make([]string, 0, 2)
	for _, ref := range refs {
		safe := safeShutdownClientRefPartV0(ref)
		if safe == "" || safe != strings.TrimSpace(ref) {
			continue
		}
		out = append(out, safe)
		if len(out) >= 2 {
			break
		}
	}
	return out
}

func shutdownRequestErrorMayStillNeedSignalV0(err error) bool {
	if err == nil {
		return false
	}
	switch strings.TrimSpace(err.Error()) {
	case "shutdown_timeout", "shutdown_cancelled", "shutdown_request_failed":
		return true
	default:
		return false
	}
}

func shutdownRequestErrorAllowsSignalV0(
	err error,
	status orquestaserver.ServerPublicStatusV0,
	forced bool,
) bool {
	if err == nil {
		return true
	}
	if forced {
		return !shutdownRequestErrorIsLiveWorkConflictV0(err) &&
			!shutdownPublicStatusHasBlockingWorkV0(status)
	}
	if strings.TrimSpace(err.Error()) != "shutdown_timeout" {
		return false
	}
	return status.ShutdownInProgress && !shutdownPublicStatusHasBlockingWorkV0(status)
}

func shutdownPublicStatusHasBlockingWorkV0(status orquestaserver.ServerPublicStatusV0) bool {
	return status.ShutdownAgentsInFlight > 0 ||
		status.ShutdownCheckpointsPending > 0 ||
		status.ShutdownCheckpointAgentsPending > 0 ||
		status.ShutdownAsyncWorkActive > 0 ||
		status.ShutdownActiveWorkCount > 0 ||
		len(compactStringsV0(status.ShutdownActiveWorkRefs)) > 0
}

func shutdownRequestErrorIsLiveWorkConflictV0(err error) bool {
	if err == nil {
		return false
	}
	message := strings.TrimSpace(err.Error())
	if strings.HasPrefix(message, "shutdown_http_409") {
		return true
	}
	return strings.Contains(message, "backend_still_running") ||
		strings.Contains(message, "active_goals_present") ||
		shutdownNotReadyErrorHasBlockingWorkV0(message)
}

func shutdownNotReadyErrorHasBlockingWorkV0(message string) bool {
	if !strings.HasPrefix(strings.TrimSpace(message), "shutdown_not_ready ") {
		return false
	}
	fields := strings.Fields(message)
	for _, field := range fields {
		switch {
		case strings.HasPrefix(field, "active_work_refs=") &&
			strings.TrimSpace(strings.TrimPrefix(field, "active_work_refs=")) != "":
			return true
		case strings.HasPrefix(field, "active_work="):
			value := strings.TrimSpace(strings.TrimPrefix(field, "active_work="))
			if value != "" && value != "0" {
				return true
			}
		case strings.HasPrefix(field, "agents_in_flight="):
			if shutdownNotReadyNumericFieldIsPositiveV0(field, "agents_in_flight=") {
				return true
			}
		case strings.HasPrefix(field, "checkpoints="):
			if shutdownNotReadyNumericFieldIsPositiveV0(field, "checkpoints=") {
				return true
			}
		case strings.HasPrefix(field, "checkpoint_agents="):
			if shutdownNotReadyNumericFieldIsPositiveV0(field, "checkpoint_agents=") {
				return true
			}
		case strings.HasPrefix(field, "runs="):
			if shutdownNotReadyRunsFieldIsPendingV0(field) {
				return true
			}
		}
	}
	return false
}

func shutdownNotReadyNumericFieldIsPositiveV0(field string, prefix string) bool {
	value := strings.TrimSpace(strings.TrimPrefix(field, prefix))
	return value != "" && value != "0"
}

func shutdownNotReadyRunsFieldIsPendingV0(field string) bool {
	value := strings.TrimSpace(strings.TrimPrefix(field, "runs="))
	stopped, requested, ok := strings.Cut(value, "/")
	return ok &&
		strings.TrimSpace(requested) != "" &&
		strings.TrimSpace(stopped) != strings.TrimSpace(requested)
}
