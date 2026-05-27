package orquestaserver

import (
	"strings"
	"time"
)

const auditWriteFailureCodeV0 = "audit_write_failed"

func (tracker *StatusTrackerV0) MarkAuditWriteFailedV0(
	event string,
	severity string,
	now time.Time,
) StateV0 {
	event = compactAuditEventNameV0(event)
	severity = compactAuditSeverityV0(severity)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.AuditStatus = "degraded"
		state.AuditFailures++
		state.AuditLastFailedAt = formatTimeV0(now)
		state.AuditLastCode = auditWriteFailureCodeV0
		state.AuditLastEvent = event
		state.AuditLastSeverity = severity
		appendRecentServerErrorV0(
			state,
			now,
			auditWriteFailureCodeV0,
			"audit",
			auditWriteFailureCodeV0,
			nil,
		)
	})
}

func (tracker *StatusTrackerV0) MarkAuditWriteConfirmedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.AuditStatus = "ok"
		state.AuditLastConfirmed = formatTimeV0(now)
	})
}

func auditWriteFailureSeverityV0(event string) string {
	switch compactAuditEventNameV0(event) {
	case "healthz", "http_request":
		return "warning"
	case "":
		return "warning"
	default:
		return "warning"
	}
}

func compactAuditSeverityV0(severity string) string {
	switch strings.TrimSpace(severity) {
	case "info", "warning", "critical":
		return strings.TrimSpace(severity)
	default:
		return "warning"
	}
}

func compactAuditEventNameV0(event string) string {
	event = strings.TrimSpace(event)
	if event == "" {
		return "unknown"
	}
	var builder strings.Builder
	for _, r := range event {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			builder.WriteRune(r)
		} else if builder.Len() > 0 {
			break
		}
		if builder.Len() >= 80 {
			break
		}
	}
	out := builder.String()
	if out == "" {
		return "unknown"
	}
	return out
}
