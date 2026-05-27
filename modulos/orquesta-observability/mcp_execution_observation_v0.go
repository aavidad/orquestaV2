package orquestaobservability

import "strings"

const (
	MCPExecutionObservationSchemaVersionV0 = "mcp_execution_observation.v0"
	MCPExecutionReasonOKV0                 = "ok"
	MCPExecutionReasonTimeoutV0            = "mcp_tool_timeout"
	MCPExecutionReasonCancelledV0          = "mcp_tool_cancelled"
	MCPExecutionReasonHandlerErrorV0       = "mcp_tool_handler_error"
)

type MCPExecutionObservationV0 struct {
	SchemaVersion  string `json:"schema_version"`
	Method         string `json:"method"`
	Profile        string `json:"profile"`
	ReasonCode     string `json:"reason_code"`
	DurationBucket string `json:"duration_bucket"`
	CorrelationID  string `json:"correlation_id,omitempty"`
}

func NormalizeMCPExecutionObservationV0(
	observation MCPExecutionObservationV0,
) MCPExecutionObservationV0 {
	observation.SchemaVersion = MCPExecutionObservationSchemaVersionV0
	observation.Method = compactMCPObservationTokenV0(observation.Method)
	observation.Profile = compactMCPObservationTokenV0(observation.Profile)
	observation.ReasonCode = compactMCPObservationTokenV0(observation.ReasonCode)
	observation.DurationBucket = compactMCPObservationTokenV0(observation.DurationBucket)
	observation.CorrelationID = compactMCPObservationTokenV0(observation.CorrelationID)
	if observation.Method == "" {
		observation.Method = "mcp_unknown"
	}
	if observation.Profile == "" {
		observation.Profile = "unknown"
	}
	if observation.ReasonCode == "" {
		observation.ReasonCode = MCPExecutionReasonOKV0
	}
	if observation.DurationBucket == "" {
		observation.DurationBucket = "unknown"
	}
	return observation
}

func compactMCPObservationTokenV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 96 || strings.ContainsAny(value, "\r\n\t /\\") {
		return ""
	}
	lower := strings.ToLower(value)
	for _, forbidden := range []string{
		"token", "secret", "password", "oauth", "bearer", "sk-", "home",
		"prompt", "transcript",
	} {
		if strings.Contains(lower, forbidden) {
			return ""
		}
	}
	return value
}
