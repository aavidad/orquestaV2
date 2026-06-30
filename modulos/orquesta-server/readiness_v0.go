package orquestaserver

import "strings"

const (
	ServerReadinessEndpointV0      = "/api/v0/server/readiness"
	ServerReadinessSchemaVersionV0 = "orquesta_server_readiness.v0"

	serverReadinessExternalWorkGoalBackendRequiredCodeV0 = "external_work_goal_backend_required"
	serverReadinessCodexGoalBackendDegradedCodeV0        = "codex_goal_backend_degraded"
)

type ServerReadinessV0 struct {
	SchemaVersion                          string                        `json:"schema_version"`
	Ready                                  bool                          `json:"ready"`
	Status                                 string                        `json:"status"`
	LivenessStatus                         string                        `json:"liveness_status"`
	StartupReady                           bool                          `json:"startup_ready"`
	StartupStatus                          string                        `json:"startup_status,omitempty"`
	StartupMessage                         string                        `json:"startup_message,omitempty"`
	StartupRevision                        StartupRevisionSummaryV0      `json:"startup_revision,omitempty"`
	RuntimeIdentity                        ServerPublicRuntimeIdentityV0 `json:"runtime_identity,omitempty"`
	StartupBlockers                        []StartupBlockerV0            `json:"startup_blockers,omitempty"`
	LastHeartbeatAt                        string                        `json:"last_heartbeat_at,omitempty"`
	LastStartupCheckAt                     string                        `json:"last_startup_check_at,omitempty"`
	ExternalBridgeStatus                   string                        `json:"external_bridge_status,omitempty"`
	ExternalBridgeReady                    bool                          `json:"external_bridge_ready,omitempty"`
	ExternalBridgeLastError                string                        `json:"external_bridge_last_error_code,omitempty"`
	IdleSelfImprovementGoalActive          bool                          `json:"idle_self_improvement_goal_active,omitempty"`
	IdleSelfImprovementGoalRef             string                        `json:"idle_self_improvement_goal_ref,omitempty"`
	IdleSelfImprovementGoalStatus          string                        `json:"idle_self_improvement_goal_status,omitempty"`
	IdleSelfImprovementGoalReasonCode      string                        `json:"idle_self_improvement_goal_reason_code,omitempty"`
	IdleSelfImprovementGoalClosureAccepted bool                          `json:"idle_self_improvement_goal_closure_accepted,omitempty"`
	Diagnostics                            []ServerDiagnosticV0          `json:"diagnostics,omitempty"`
	EvidenceRefs                           []string                      `json:"evidence_refs,omitempty"`
}

func NewServerReadinessV0(state StateV0) ServerReadinessV0 {
	status := strings.TrimSpace(state.Status)
	startupStatus := strings.TrimSpace(state.StartupStatus)
	ready := state.StartupReady && status == "running"
	diagnostics := serverReadinessBlockingDiagnosticsV0(state.EffectiveConfig.Diagnostics)
	if strings.TrimSpace(state.AuditStatus) == "degraded" &&
		strings.TrimSpace(state.AuditLastSeverity) == "warning" {
		ready = false
	}
	if state.SelfWatchdogShutdownRequested {
		ready = false
	}
	if len(diagnostics) > 0 {
		ready = false
	}
	if status == "" {
		status = "unknown"
	}
	if startupStatus == "" && !state.StartupReady {
		startupStatus = "startup_unknown"
	}
	if len(diagnostics) > 0 {
		startupStatus = "startup_degraded_" + strings.TrimSpace(diagnostics[0].Code)
	}
	startupMessage := publicReadinessMessageV0(state.StartupMessage)
	if startupMessage == "" && len(diagnostics) > 0 {
		startupMessage = publicReadinessMessageV0(diagnostics[0].Message)
	}
	goal := NewServerPublicIdleSelfImprovementGoalStateV0(state)
	return ServerReadinessV0{
		SchemaVersion:                          ServerReadinessSchemaVersionV0,
		Ready:                                  ready,
		Status:                                 status,
		LivenessStatus:                         "ok",
		StartupReady:                           state.StartupReady,
		StartupStatus:                          startupStatus,
		StartupMessage:                         startupMessage,
		StartupRevision:                        normalizeStartupRevisionSummaryV0(state.StartupRevision),
		RuntimeIdentity:                        NewServerPublicRuntimeIdentityV0(state),
		StartupBlockers:                        normalizeStartupBlockersV0(state.StartupBlockers),
		LastHeartbeatAt:                        strings.TrimSpace(state.LastHeartbeatAt),
		LastStartupCheckAt:                     strings.TrimSpace(state.LastStartupCheckAt),
		ExternalBridgeStatus:                   strings.TrimSpace(state.ExternalBridgeStatus),
		ExternalBridgeReady:                    externalBridgeReadinessOKV0(state.ExternalBridgeStatus),
		ExternalBridgeLastError:                strings.TrimSpace(state.ExternalBridgeLastError),
		IdleSelfImprovementGoalActive:          serverReadinessGoalActiveV0(goal),
		IdleSelfImprovementGoalRef:             serverReadinessGoalRefV0(goal),
		IdleSelfImprovementGoalStatus:          serverReadinessGoalStatusV0(goal),
		IdleSelfImprovementGoalReasonCode:      serverReadinessGoalReasonCodeV0(goal),
		IdleSelfImprovementGoalClosureAccepted: serverReadinessGoalClosureAcceptedV0(goal),
		Diagnostics:                            diagnostics,
		EvidenceRefs: compactServerStringsV0(append(
			append(append([]string(nil), state.StartupEvidenceRefs...), state.ExternalBridgeEvidenceRefs...),
			serverReadinessDiagnosticEvidenceRefsV0(diagnostics)...,
		)),
	}
}

func serverReadinessBlockingDiagnosticsV0(diagnostics []ServerDiagnosticV0) []ServerDiagnosticV0 {
	normalized := normalizeServerEffectiveConfigDiagnosticsV0(diagnostics)
	out := make([]ServerDiagnosticV0, 0, len(normalized))
	for _, diagnostic := range normalized {
		switch diagnostic.Code {
		case serverReadinessExternalWorkGoalBackendRequiredCodeV0,
			serverReadinessCodexGoalBackendDegradedCodeV0:
			out = append(out, diagnostic)
		}
	}
	return out
}

func serverReadinessDiagnosticEvidenceRefsV0(diagnostics []ServerDiagnosticV0) []string {
	refs := []string{}
	for _, diagnostic := range diagnostics {
		refs = append(refs, diagnostic.EvidenceRefs...)
	}
	return compactServerStringsV0(refs)
}

func serverReadinessGoalActiveV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) bool {
	return goal != nil && goal.Active
}

func serverReadinessGoalRefV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) string {
	if goal == nil {
		return ""
	}
	return strings.TrimSpace(goal.GoalRef)
}

func serverReadinessGoalStatusV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) string {
	if goal == nil {
		return ""
	}
	return strings.TrimSpace(firstNonEmptyServerDiagnosticV0(
		goal.OperationalStatus,
		goal.ResultStatus,
		goal.ReceiptStatus,
	))
}

func serverReadinessGoalReasonCodeV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) string {
	if goal == nil {
		return ""
	}
	return strings.TrimSpace(goal.OperationalReasonCode)
}

func serverReadinessGoalClosureAcceptedV0(goal *ServerPublicIdleSelfImprovementGoalStateV0) bool {
	return goal != nil && goal.ClosureAccepted
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
