package orquestaservershutdown

import "strings"

func ServerShutdownRequesterAuthorizedV0(requestedBy string) bool {
	value := strings.ToLower(strings.TrimSpace(requestedBy))
	if value == "" {
		return false
	}
	if strings.Contains(value, "agent") || strings.Contains(value, "agente") ||
		strings.Contains(value, "worker") || strings.Contains(value, "subagent") ||
		strings.Contains(value, "subagente") {
		return false
	}
	return strings.Contains(value, "director")
}

func serverShutdownRequesterAuthorizedV0(requestedBy string) bool {
	return ServerShutdownRequesterAuthorizedV0(requestedBy)
}

func missingServerShutdownDepV0(status string) ServerShutdownResultV0 {
	return withServerShutdownRecommendedActionV0(ServerShutdownResultV0{
		SchemaVersion: ServerShutdownSchemaVersionV0,
		Status:        status,
	})
}
