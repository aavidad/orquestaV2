package orquestacoreworkflow

import "strings"

type AgentWorkAssessmentProjectionV0 struct {
	AssessmentRef  string
	PhaseID        string
	AgentRequestID string
	TaskRef        string
	DeliveryRef    string
	Verdict        string
	Action         string
	Severity       string
}

func AgentAssessmentProjectionRefV0(
	payload AgentWorkAssessedPayloadV0,
) string {
	payload = normalizeAgentWorkAssessedPayloadV0(payload)
	parts := []string{payload.AssessmentRef}
	parts = appendProjectionPartV0(parts, "phase", payload.PhaseID)
	parts = appendProjectionPartV0(parts, "agent", payload.AgentRequestID)
	parts = appendProjectionPartV0(parts, "task", payload.TaskRef)
	parts = appendProjectionPartV0(parts, "delivery", payload.DeliveryRef)
	parts = appendProjectionPartV0(parts, "verdict", payload.Verdict)
	parts = appendProjectionPartV0(parts, "action", payload.Action)
	parts = appendProjectionPartV0(parts, "severity", payload.Severity)
	return strings.Join(parts, "#")
}

func ParseAgentAssessmentProjectionV0(
	value string,
) (AgentWorkAssessmentProjectionV0, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return AgentWorkAssessmentProjectionV0{}, false
	}
	head, tail, _ := strings.Cut(value, "#")
	projection := AgentWorkAssessmentProjectionV0{
		AssessmentRef: strings.TrimSpace(head),
	}
	for tail != "" {
		var part string
		part, tail, _ = strings.Cut(tail, "#")
		key, val, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		applyAssessmentProjectionPartV0(&projection, key, val)
	}
	return projection, projection.AssessmentRef != ""
}

func AgentAssessmentProjectionIDV0(value string) string {
	projection, ok := ParseAgentAssessmentProjectionV0(value)
	if !ok {
		return ""
	}
	return projection.AssessmentRef
}

func appendProjectionPartV0(parts []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return parts
	}
	return append(parts, strings.TrimSpace(key)+":"+value)
}

func applyAssessmentProjectionPartV0(
	projection *AgentWorkAssessmentProjectionV0,
	key string,
	value string,
) {
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(key) {
	case "phase":
		projection.PhaseID = value
	case "agent":
		projection.AgentRequestID = value
	case "task":
		projection.TaskRef = value
	case "delivery":
		projection.DeliveryRef = value
	case "verdict":
		projection.Verdict = value
	case "action":
		projection.Action = value
	case "severity":
		projection.Severity = value
	}
}
