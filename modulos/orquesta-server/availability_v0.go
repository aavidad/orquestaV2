package orquestaserver

import "strings"

type ServerAvailabilityV0 struct {
	Status       string   `json:"availability_status,omitempty"`
	Reason       string   `json:"availability_reason,omitempty"`
	NextActions  []string `json:"availability_next_actions,omitempty"`
	EvidenceRefs []string `json:"availability_evidence_refs,omitempty"`
}

func NewServerAvailabilityV0(state StateV0) ServerAvailabilityV0 {
	status := strings.TrimSpace(state.Status)
	startupStatus := strings.TrimSpace(state.StartupStatus)
	switch {
	case status == "stale" && startupStatus == ServerProcessStaleReasonCodeV0:
		reason := "server_process_stale"
		if strings.TrimSpace(state.LastHeartbeatAt) != "" ||
			serverAvailabilityHasRefV0(state.StartupEvidenceRefs, "evidence-ref-startup-ready") {
			reason = "server_crashed_after_readiness"
		}
		return ServerAvailabilityV0{
			Status: "crashed",
			Reason: reason,
			NextActions: []string{
				"inspect_orquesta_server_status_or_statefile",
				"cleanup_owned_goal_backend_if_owner_matches",
				"restart_orquesta_server_goal_first_app_server_tmux",
			},
			EvidenceRefs: []string{"evidence-ref-server-statefile-process-not-alive"},
		}
	case status == "stopped":
		return ServerAvailabilityV0{
			Status: "stopped",
			Reason: "server_stopped",
			NextActions: []string{
				"start_orquesta_server_goal_first_app_server_tmux",
				"check_orquesta_readiness",
			},
			EvidenceRefs: []string{"evidence-ref-server-state-stopped"},
		}
	case status == "running" && state.StartupReady:
		return ServerAvailabilityV0{
			Status: "running",
			Reason: "server_ready",
		}
	case status == "":
		return ServerAvailabilityV0{
			Status: "unknown",
			Reason: "server_state_unknown",
			NextActions: []string{
				"inspect_orquesta_server_status_or_statefile",
				"check_orquesta_readiness",
			},
		}
	default:
		return ServerAvailabilityV0{
			Status: status,
			Reason: firstNonEmptyServerDiagnosticV0(startupStatus, "server_not_ready"),
		}
	}
}

func serverAvailabilityHasRefV0(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}
