package orquestacoreleases

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateAgentLeasePolicyV0(policy AgentLeasePolicyV0) []AgentLeaseIssueV0 {
	v := agentLeaseValidatorV0{}
	v.requireOpaque("lease_policy_ref", policy.LeasePolicyRef)
	v.requirePositiveSeconds("launch_timeout_seconds", policy.LaunchTimeoutSeconds)
	v.requirePositiveSeconds("heartbeat_timeout_seconds", policy.HeartbeatTimeoutSeconds)
	v.requirePositiveSeconds("total_timeout_seconds", policy.TotalTimeoutSeconds)
	v.requireTimeoutAction("timeout_action", policy.TimeoutAction)
	if policy.MaxRetries < 0 || policy.MaxRetries > 10 {
		v.add(ErrAgentLeaseInvalidoV0, "max_retries")
	}
	v.validateEvidenceRefs("evidence_refs", policy.EvidenceRefs)
	if policy.TotalTimeoutSeconds > 0 {
		if policy.LaunchTimeoutSeconds > policy.TotalTimeoutSeconds {
			v.add(ErrAgentLeaseTiempoInvalidoV0, "launch_timeout_seconds")
		}
		if policy.HeartbeatTimeoutSeconds > policy.TotalTimeoutSeconds {
			v.add(ErrAgentLeaseTiempoInvalidoV0, "heartbeat_timeout_seconds")
		}
	}
	return v.issues
}

func ValidateAgentHeartbeatReportV0(report AgentHeartbeatReportV0) []AgentLeaseIssueV0 {
	v := agentLeaseValidatorV0{}
	v.requireOpaque("heartbeat_ref", report.HeartbeatRef)
	v.requireOpaque("run_ref", report.RunRef)
	v.requireOpaque("agent_request_id", report.AgentRequestID)
	v.requireOpaque("lease_ref", report.LeaseRef)
	v.requireObservedAt("observed_at", report.ObservedAt)
	v.requireHeartbeatStatus("status", report.Status)
	v.optionalOpaque("progress_report_ref", report.ProgressReportRef)
	v.validateEvidenceRefs("evidence_refs", report.EvidenceRefs)
	return v.issues
}

type agentLeaseValidatorV0 struct {
	issues []AgentLeaseIssueV0
}

func (v *agentLeaseValidatorV0) requireOpaque(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(ErrAgentLeaseReferenciaNoOpacaV0, field)
		return
	}
	v.optionalOpaque(field, value)
}

func (v *agentLeaseValidatorV0) optionalOpaque(field, value string) {
	if value == "" {
		return
	}
	value = strings.TrimSpace(value)
	switch {
	case containsForbiddenAgentLeaseDetailV0(value):
		v.add(ErrAgentLeaseDetalleProhibidoV0, field)
	case !agentLeaseOpaqueRefPatternV0.MatchString(value):
		v.add(ErrAgentLeaseReferenciaNoOpacaV0, field)
	}
}

func (v *agentLeaseValidatorV0) requirePositiveSeconds(field string, value int) {
	if value < 1 || value > 86400 {
		v.add(ErrAgentLeaseTiempoInvalidoV0, field)
	}
}

func (v *agentLeaseValidatorV0) requireTimeoutAction(field string, action AgentLeaseTimeoutActionV0) {
	if !isOneOfAgentLeaseV0(string(action),
		string(AgentLeaseTimeoutRetryV0),
		string(AgentLeaseTimeoutStopAgentV0),
		string(AgentLeaseTimeoutAskDirectorV0),
		string(AgentLeaseTimeoutReplanTaskV0),
		string(AgentLeaseTimeoutAlertOnlyV0),
	) {
		v.add(ErrAgentLeaseAccionInvalidaV0, field)
	}
}

func (v *agentLeaseValidatorV0) requireHeartbeatStatus(field string, status AgentHeartbeatStatusV0) {
	if !isOneOfAgentLeaseV0(string(status),
		string(AgentHeartbeatAliveV0),
		string(AgentHeartbeatProgressingV0),
		string(AgentHeartbeatStalledV0),
		string(AgentHeartbeatStoppedV0),
		string(AgentHeartbeatFailedV0),
	) {
		v.add(ErrAgentHeartbeatStatusInvalidoV0, field)
	}
}

func (v *agentLeaseValidatorV0) requireObservedAt(field, value string) {
	if !validAgentLeaseUTCInstantV0(value) {
		v.add(ErrAgentHeartbeatObservedAtV0, field)
	}
}

func (v *agentLeaseValidatorV0) validateEvidenceRefs(field string, refs []string) {
	if len(refs) > 16 {
		v.add(ErrAgentLeaseInvalidoV0, field)
	}
	seen := map[string]bool{}
	for i, ref := range refs {
		itemField := fmt.Sprintf("%s[%d]", field, i)
		v.requireOpaque(itemField, ref)
		if seen[ref] {
			v.add(ErrAgentLeaseInvalidoV0, itemField)
		}
		seen[ref] = true
	}
}

func (v *agentLeaseValidatorV0) add(code AgentLeaseIssueCodeV0, field string) {
	v.issues = append(v.issues, AgentLeaseIssueV0{Code: code, Field: field})
}

func detectForbiddenAgentLeaseJSONDetailsV0(data []byte) []AgentLeaseIssueV0 {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return detectSensitiveAgentLeaseJSONDetailsV0(raw, "")
}

func detectSensitiveAgentLeaseJSONDetailsV0(value any, field string) []AgentLeaseIssueV0 {
	var issues []AgentLeaseIssueV0
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			nestedField := joinAgentLeaseJSONFieldV0(field, key)
			if containsSensitiveAgentLeaseJSONKeyValueV0(key, nested) {
				issues = append(issues, AgentLeaseIssueV0{Code: ErrAgentLeaseDetalleProhibidoV0, Field: nestedField})
				continue
			}
			issues = append(issues, detectSensitiveAgentLeaseJSONDetailsV0(nested, nestedField)...)
		}
	case []any:
		for i, nested := range typed {
			issues = append(issues, detectSensitiveAgentLeaseJSONDetailsV0(nested, fmt.Sprintf("%s[%d]", field, i))...)
		}
	case string:
		if containsForbiddenAgentLeaseDetailV0(typed) {
			issues = append(issues, AgentLeaseIssueV0{Code: ErrAgentLeaseDetalleProhibidoV0, Field: field})
		}
	}
	return issues
}

func containsSensitiveAgentLeaseJSONKeyValueV0(key string, value any) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if !sensitiveAgentLeaseJSONValueKeysV0[key] {
		return false
	}
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	default:
		return true
	}
}

func joinAgentLeaseJSONFieldV0(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
