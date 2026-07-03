package orquestaservershutdown

import "strings"

const (
	ServerShutdownRecommendedActionObserveActiveGoalsV0       = "observe_active_goals_before_shutdown"
	ServerShutdownRecommendedActionRetryCleanupGoalBackendsV0 = "retry_shutdown_with_cleanup_goal_backends"
	ServerShutdownRecommendedActionWaitOrReconcileBackendV0   = "wait_or_reconcile_goal_backend_cleanup"
	ServerShutdownRecommendedActionWaitCheckpointV0           = "wait_shutdown_checkpoint"
	ServerShutdownRecommendedActionWaitDrainV0                = "wait_shutdown_drain"
	ServerShutdownRecommendedActionConfigureShutdownPortsV0   = "configure_server_shutdown_ports"
	ServerShutdownRecommendedActionReviewRequesterAuthorityV0 = "review_shutdown_requester_authority"
)

func withServerShutdownRecommendedActionV0(
	result ServerShutdownResultV0,
) ServerShutdownResultV0 {
	result.RecommendedAction = recommendedActionForServerShutdownResultV0(result)
	return result
}

func recommendedActionForServerShutdownResultV0(
	result ServerShutdownResultV0,
) string {
	switch strings.TrimSpace(result.Status) {
	case ServerShutdownStatusActiveGoalsPresentV0:
		return ServerShutdownRecommendedActionObserveActiveGoalsV0
	case ServerShutdownStatusBackendStillRunningV0:
		if serverShutdownEvidenceContainsV0(
			result.EvidenceRefs,
			"evidence-ref-shutdown-goal-backend-cleanup-requested",
		) {
			return ServerShutdownRecommendedActionWaitOrReconcileBackendV0
		}
		return ServerShutdownRecommendedActionRetryCleanupGoalBackendsV0
	case ServerShutdownStatusWaitingCheckpointV0:
		return ServerShutdownRecommendedActionWaitCheckpointV0
	case ServerShutdownStatusWaitingDrainV0, "stop_pending":
		return ServerShutdownRecommendedActionWaitDrainV0
	case ServerShutdownStatusNoQueueReaderV0,
		ServerShutdownStatusNoRunControlReaderV0,
		ServerShutdownStatusNoRunControlWriterV0:
		return ServerShutdownRecommendedActionConfigureShutdownPortsV0
	case ServerShutdownStatusRequesterDeniedV0:
		return ServerShutdownRecommendedActionReviewRequesterAuthorityV0
	default:
		return ""
	}
}

func serverShutdownEvidenceContainsV0(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}
