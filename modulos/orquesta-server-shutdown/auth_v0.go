package orquestaservershutdown

import "strings"

func serverShutdownRequesterAuthorizedV0(requestedBy string) bool {
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

func missingServerShutdownDepV0(status string) ServerShutdownResultV0 {
	return ServerShutdownResultV0{
		SchemaVersion: ServerShutdownSchemaVersionV0,
		Status:        status,
	}
}
