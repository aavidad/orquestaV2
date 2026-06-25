package orquestaserver

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const (
	serverOperationalMessageMaxRunesV0          = 192
	serverOperationalMessageRedactedV0          = "operational_message_redacted"
	serverStartupOperationalMessageRedactedV0   = "startup_message_redacted"
	serverOperationalMessageTruncatedSuffixV0   = "...[truncated]"
	serverOperationalMessageSensitiveTokenRefV0 = "[redacted]"
	serverOperationalMessageSchemaV0            = "orquesta_server_operational_message.v0"
)

type ServerOperationalMessageV0 struct {
	SchemaVersion string         `json:"schema_version,omitempty"`
	Scope         string         `json:"scope,omitempty"`
	ReasonCode    string         `json:"reason_code,omitempty"`
	Status        string         `json:"status,omitempty"`
	Message       string         `json:"message,omitempty"`
	RunRefs       []string       `json:"run_refs,omitempty"`
	GoalRefs      []string       `json:"goal_refs,omitempty"`
	RequestRefs   []string       `json:"request_refs,omitempty"`
	EvidenceRefs  []string       `json:"evidence_refs,omitempty"`
	Counters      map[string]int `json:"counters,omitempty"`
}

type serverOperationalMessageInputV0 struct {
	Scope        string
	ReasonCode   string
	Status       string
	Message      string
	RunRefs      []string
	GoalRefs     []string
	RequestRefs  []string
	EvidenceRefs []string
	Counters     map[string]int
}

func projectServerStartupMessageV0(message string) string {
	return projectServerOperationalMessageV0("startup", message)
}

func projectServerOperationalMessageV0(scope string, message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if orquestarails.SecurityModeProgrammingEnabledV0() {
		return truncateServerOperationalMessageV0(message)
	}
	if serverOperationalMessageLooksSensitiveV0(message) {
		if strings.TrimSpace(scope) == "startup" {
			return serverStartupOperationalMessageRedactedV0
		}
		return serverOperationalMessageRedactedV0
	}
	return truncateServerOperationalMessageV0(message)
}

func projectServerOperationalMessageRecordV0(
	input serverOperationalMessageInputV0,
) *ServerOperationalMessageV0 {
	message := projectServerOperationalMessageV0(input.Scope, input.Message)
	record := ServerOperationalMessageV0{
		SchemaVersion: serverOperationalMessageSchemaV0,
		Scope:         compactServerOperationalTokenV0(input.Scope),
		ReasonCode:    compactServerOperationalTokenV0(input.ReasonCode),
		Status:        compactServerOperationalTokenV0(input.Status),
		Message:       message,
		RunRefs:       compactServerOperationalRefsV0(input.RunRefs),
		GoalRefs:      compactServerOperationalRefsV0(input.GoalRefs),
		RequestRefs:   compactServerOperationalRefsV0(input.RequestRefs),
		EvidenceRefs:  compactServerOperationalRefsV0(input.EvidenceRefs),
		Counters:      compactServerOperationalCountersV0(input.Counters),
	}
	if record.Scope == "" &&
		record.ReasonCode == "" &&
		record.Status == "" &&
		record.Message == "" &&
		len(record.RunRefs) == 0 &&
		len(record.GoalRefs) == 0 &&
		len(record.RequestRefs) == 0 &&
		len(record.EvidenceRefs) == 0 &&
		len(record.Counters) == 0 {
		return nil
	}
	return &record
}

func copyServerOperationalMessageV0(
	message *ServerOperationalMessageV0,
) *ServerOperationalMessageV0 {
	if message == nil {
		return nil
	}
	out := *message
	out.RunRefs = append([]string(nil), message.RunRefs...)
	out.GoalRefs = append([]string(nil), message.GoalRefs...)
	out.RequestRefs = append([]string(nil), message.RequestRefs...)
	out.EvidenceRefs = append([]string(nil), message.EvidenceRefs...)
	out.Counters = compactServerOperationalCountersV0(message.Counters)
	return &out
}

func projectServerOperationalReasonFieldV0(key string, value string) string {
	key = strings.TrimSpace(key)
	value = projectServerOperationalMessageV0("", value)
	if key == "" || value == "" {
		return ""
	}
	return key + "=" + value
}

func serverOperationalMessageLooksSensitiveV0(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	for _, fragment := range []string{
		"/",
		"\\",
		"home",
		"token",
		"secret",
		"authorization",
		"bearer ",
		"password",
		"prompt",
		"transcript",
		"runtime",
		".codex",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func truncateServerOperationalMessageV0(message string) string {
	runes := []rune(strings.TrimSpace(message))
	if len(runes) <= serverOperationalMessageMaxRunesV0 {
		return string(runes)
	}
	suffix := []rune(serverOperationalMessageTruncatedSuffixV0)
	limit := serverOperationalMessageMaxRunesV0 - len(suffix)
	if limit < 1 {
		return serverOperationalMessageSensitiveTokenRefV0
	}
	return string(runes[:limit]) + serverOperationalMessageTruncatedSuffixV0
}

func compactServerOperationalTokenV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
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
	return builder.String()
}

func compactServerOperationalRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = compactServerOperationalRefV0(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) >= 16 {
			break
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func compactServerOperationalRefV0(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 128 {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == ':' || r == '.' {
			continue
		}
		return ""
	}
	return value
}

func compactServerOperationalCountersV0(values map[string]int) map[string]int {
	out := map[string]int{}
	for key, value := range values {
		key = compactServerOperationalTokenV0(key)
		if key == "" || value < 0 {
			continue
		}
		out[key] = value
		if len(out) >= 16 {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func boolToServerCounterV0(value bool) int {
	if value {
		return 1
	}
	return 0
}

func idleSelfImprovementReasonCodeV0(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "unknown"
	}
	if strings.HasPrefix(reason, "error:") {
		return "idle_self_improvement_error"
	}
	if index := strings.Index(reason, ";"); index > 0 {
		reason = reason[:index]
	}
	if index := strings.Index(reason, ":"); index > 0 {
		reason = reason[:index]
	}
	if reason == "" {
		return "unknown"
	}
	return reason
}
