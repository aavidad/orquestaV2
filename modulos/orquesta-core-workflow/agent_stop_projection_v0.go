package orquestacoreworkflow

import "strings"

type AgentStopRequestProjectionV0 struct {
	AgentRequestID string
	ReasonCode     string
}

func AgentStopRequestProjectionRefV0(
	payload AgentStopRequestedPayloadV0,
) string {
	payload = normalizeAgentStopRequestedPayloadV0(payload)
	parts := []string{payload.AgentRequestID}
	parts = appendProjectionPartV0(parts, "reason", payload.ReasonCode)
	return strings.Join(parts, "#")
}

func ParseAgentStopRequestProjectionV0(
	value string,
) (AgentStopRequestProjectionV0, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return AgentStopRequestProjectionV0{}, false
	}
	head, tail, _ := strings.Cut(value, "#")
	projection := AgentStopRequestProjectionV0{
		AgentRequestID: strings.TrimSpace(head),
	}
	for tail != "" {
		var part string
		part, tail, _ = strings.Cut(tail, "#")
		key, val, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		applyAgentStopProjectionPartV0(&projection, key, val)
	}
	return projection, projection.AgentRequestID != "" && projection.ReasonCode != ""
}

func AgentStopRequestProjectionAgentIDV0(value string) string {
	projection, ok := ParseAgentStopRequestProjectionV0(value)
	if !ok {
		return ""
	}
	return projection.AgentRequestID
}

func agentStopProjectionFieldUnsafeV0(
	payload StopAgentCommandPayloadV0,
) bool {
	return agentStopProjectionContainsSeparatorV0(payload.AgentRequestID) ||
		agentStopProjectionContainsSeparatorV0(payload.ReasonCode)
}

func agentStopProjectionContainsSeparatorV0(value string) bool {
	return strings.Contains(strings.TrimSpace(value), "#")
}

func applyAgentStopProjectionPartV0(
	projection *AgentStopRequestProjectionV0,
	key string,
	value string,
) {
	if projection == nil {
		return
	}
	switch strings.TrimSpace(key) {
	case "reason":
		projection.ReasonCode = strings.TrimSpace(value)
	}
}
