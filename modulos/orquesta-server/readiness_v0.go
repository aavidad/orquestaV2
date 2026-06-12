package orquestaserver

import "strings"

const (
	ServerReadinessEndpointV0      = "/api/v0/server/readiness"
	ServerReadinessSchemaVersionV0 = "orquesta_server_readiness.v0"
)

type ServerReadinessV0 struct {
	SchemaVersion           string                   `json:"schema_version"`
	Ready                   bool                     `json:"ready"`
	Status                  string                   `json:"status"`
	LivenessStatus          string                   `json:"liveness_status"`
	StartupReady            bool                     `json:"startup_ready"`
	StartupStatus           string                   `json:"startup_status,omitempty"`
	StartupMessage          string                   `json:"startup_message,omitempty"`
	StartupRevision         StartupRevisionSummaryV0 `json:"startup_revision,omitempty"`
	LastHeartbeatAt         string                   `json:"last_heartbeat_at,omitempty"`
	LastStartupCheckAt      string                   `json:"last_startup_check_at,omitempty"`
	ExternalBridgeStatus    string                   `json:"external_bridge_status,omitempty"`
	ExternalBridgeReady     bool                     `json:"external_bridge_ready,omitempty"`
	ExternalBridgeLastError string                   `json:"external_bridge_last_error_code,omitempty"`
	EvidenceRefs            []string                 `json:"evidence_refs,omitempty"`
}

func NewServerReadinessV0(state StateV0) ServerReadinessV0 {
	status := strings.TrimSpace(state.Status)
	startupStatus := strings.TrimSpace(state.StartupStatus)
	ready := state.StartupReady && status == "running"
	if strings.TrimSpace(state.AuditStatus) == "degraded" &&
		strings.TrimSpace(state.AuditLastSeverity) == "warning" {
		ready = false
	}
	if state.SelfWatchdogShutdownRequested {
		ready = false
	}
	if status == "" {
		status = "unknown"
	}
	if startupStatus == "" && !state.StartupReady {
		startupStatus = "startup_unknown"
	}
	return ServerReadinessV0{
		SchemaVersion:           ServerReadinessSchemaVersionV0,
		Ready:                   ready,
		Status:                  status,
		LivenessStatus:          "ok",
		StartupReady:            state.StartupReady,
		StartupStatus:           startupStatus,
		StartupMessage:          publicReadinessMessageV0(state.StartupMessage),
		StartupRevision:         normalizeStartupRevisionSummaryV0(state.StartupRevision),
		LastHeartbeatAt:         strings.TrimSpace(state.LastHeartbeatAt),
		LastStartupCheckAt:      strings.TrimSpace(state.LastStartupCheckAt),
		ExternalBridgeStatus:    strings.TrimSpace(state.ExternalBridgeStatus),
		ExternalBridgeReady:     externalBridgeReadinessOKV0(state.ExternalBridgeStatus),
		ExternalBridgeLastError: strings.TrimSpace(state.ExternalBridgeLastError),
		EvidenceRefs:            compactServerStringsV0(append(append([]string(nil), state.StartupEvidenceRefs...), state.ExternalBridgeEvidenceRefs...)),
	}
}

func externalBridgeReadinessOKV0(status string) bool {
	switch strings.TrimSpace(status) {
	case "", "disabled", "waiting_initial_delay", "running", "idle", "stopping", "stopped":
		return true
	default:
		return false
	}
}

func publicReadinessMessageV0(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	lower := strings.ToLower(message)
	for _, forbidden := range []string{"/", "\\", "home", "token", "secret", "prompt", "transcript", "runtime"} {
		if strings.Contains(lower, forbidden) {
			return "startup_message_redacted"
		}
	}
	return message
}
